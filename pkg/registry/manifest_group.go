package registry

import (
	"fmt"
	"io"
	"net/http"
	"sync"

	"go.uber.org/zap"
)

// groupToken 表示一组镜像的批量认证 token
type groupToken struct {
	registryKey string
	token       string
}

// subGroup 子组结构：包含具体的镜像列表和索引
type subGroup struct {
	registryKey string
	specs       []ImageSpec
	indices     []int
	token       groupToken // 该子组的批量 token
}

// registryGroup registry 初步分组结构
type registryGroup struct {
	registryKey string
	specs       []ImageSpec
	indices     []int
}

// groupImagesByRegistry 按 registry key 和数量限制对镜像进行分组
func (c *Client) groupImagesByRegistry(imageSpecs []ImageSpec, size *int) []*subGroup {
	maxBatchSize := 30 // 每个批次的最大镜像数量
	if size != nil {
		maxBatchSize = *size
	}
	if maxBatchSize > 30 || maxBatchSize < 1 {
		maxBatchSize = 30
	}

	// 第一步：按 registry key 初步分组
	primaryGroups := make(map[string]*registryGroup)

	for i, spec := range imageSpecs {
		registryKey := DetectRegistry(spec.Image)

		if primaryGroups[registryKey] == nil {
			primaryGroups[registryKey] = &registryGroup{
				registryKey: registryKey,
				specs:       []ImageSpec{},
				indices:     []int{},
			}
		}
		primaryGroups[registryKey].specs = append(primaryGroups[registryKey].specs, spec)
		primaryGroups[registryKey].indices = append(primaryGroups[registryKey].indices, i)
	}

	// 第二步：对每个 registry 组按数量限制进行子分组
	var subGroups []*subGroup

	for registryKey, group := range primaryGroups {
		totalImages := len(group.specs)

		if totalImages > maxBatchSize {
			// 超过限制，分成多个子组
			numSubGroups := (totalImages + maxBatchSize - 1) / maxBatchSize
			c.logger.Warn("镜像数量超过单批限制",
				zap.String("registryKey", registryKey),
				zap.Int("totalImages", totalImages),
				zap.Int("maxBatchSize", maxBatchSize),
				zap.Int("numSubGroups", numSubGroups))

			for i := 0; i < totalImages; i += maxBatchSize {
				end := i + maxBatchSize
				if end > totalImages {
					end = totalImages
				}

				subGroups = append(subGroups, &subGroup{
					registryKey: group.registryKey,
					specs:       group.specs[i:end],
					indices:     group.indices[i:end],
				})
			}
		} else {
			// 不超过限制，直接作为一个子组
			subGroups = append(subGroups, &subGroup{
				registryKey: group.registryKey,
				specs:       group.specs,
				indices:     group.indices,
			})
		}
	}

	// 显示分组信息
	c.printGroupInfo(primaryGroups, subGroups)

	return subGroups
}

// printGroupInfo 打印分组信息
func (c *Client) printGroupInfo(primaryGroups map[string]*registryGroup, subGroups []*subGroup) {
	if len(primaryGroups) > 1 {
		c.logger.Info("检测到多个 registry",
			zap.Int("registryCount", len(primaryGroups)),
			zap.Int("totalBatches", len(subGroups)))
	} else if len(subGroups) > 1 {
		c.logger.Info("将分批处理",
			zap.Int("batches", len(subGroups)))
	}

	for i, sg := range subGroups {
		registryName := sg.registryKey
		if config, ok := GetRegistry(sg.registryKey); ok {
			registryName = config.Name
		}
		c.logger.Info("处理批次",
			zap.Int("batchNumber", i+1),
			zap.String("registry", registryName),
			zap.Int("imageCount", len(sg.specs)))
	}
}

// acquireBatchTokens 为每个子组获取批量认证 token
func (c *Client) acquireBatchTokens(subGroups []*subGroup) {
	for _, sg := range subGroups {
		if len(sg.specs) <= 1 {
			continue // 单个镜像不需要批量认证
		}

		// 提取该子组的所有镜像名称
		images := make([]string, len(sg.specs))
		for i, spec := range sg.specs {
			images[i] = spec.Image
		}

		// 获取批量 token
		token, err := c.GetAuthTokenForImages(images, sg.registryKey)
		if err == nil {
			sg.token = groupToken{
				registryKey: sg.registryKey,
				token:       token,
			}
			c.logger.Info("已获取批量认证 token",
				zap.Int("imageCount", len(sg.specs)))
		} else {
			c.logger.Warn("批量认证失败，将单独认证",
				zap.Int("imageCount", len(sg.specs)),
				zap.Error(err))
		}
	}
}

// fetchManifestsSequentially 顺序获取所有 manifest
func (c *Client) fetchManifestsSequentially(subGroups []*subGroup, results []ManifestResult) {
	for _, sg := range subGroups {
		for idx, spec := range sg.specs {
			originalIndex := sg.indices[idx]
			results[originalIndex] = c.fetchSingleManifest(spec, sg.token)
		}
	}
}

// fetchManifestsConcurrently 并发获取所有 manifest
func (c *Client) fetchManifestsConcurrently(subGroups []*subGroup, results []ManifestResult, concurrency int) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for _, sg := range subGroups {
		for idx, spec := range sg.specs {
			wg.Add(1)

			originalIndex := sg.indices[idx]
			token := sg.token

			go func(index int, imgSpec ImageSpec, tok groupToken) {
				defer wg.Done()
				semaphore <- struct{}{} // 获取信号量
				defer func() { <-semaphore }()

				results[index] = c.fetchSingleManifest(imgSpec, tok)
			}(originalIndex, spec, token)
		}
	}

	wg.Wait()
}

// fetchSingleManifest 获取单个镜像的 manifest
func (c *Client) fetchSingleManifest(spec ImageSpec, token groupToken) ManifestResult {
	// 检查是否有批量 token
	if token.token != "" {
		// 使用批量 token
		return c.getManifestWithBatchToken(spec, token.token, token.registryKey)
	}

	// 单独认证
	manifest, digest, err := c.GetManifestWithDigest(spec.Image, spec.Tag)
	return ManifestResult{
		Image:    spec.Image,
		Tag:      spec.Tag,
		Manifest: manifest,
		Digest:   digest,
		Error:    err,
	}
}

// getManifestWithBatchToken 使用已获取的批量 token 获取 manifest
func (c *Client) getManifestWithBatchToken(spec ImageSpec, token string, registryKey string) ManifestResult {
	result := ManifestResult{
		Image: spec.Image,
		Tag:   spec.Tag,
	}

	// 获取 registry 配置
	config, ok := GetRegistry(registryKey)
	if !ok {
		result.Error = fmt.Errorf("未找到 registry 配置: %s", registryKey)
		return result
	}

	// 规范化镜像名称
	normalizedImage := NormalizeImageName(spec.Image, registryKey)

	// 构建 manifest URL
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", config.RegistryURL, normalizedImage, spec.Tag)

	// 创建请求
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		result.Error = fmt.Errorf("创建请求失败: %w", err)
		return result
	}

	// 设置必要的 headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Add("Accept", "application/vnd.oci.image.index.v1+json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		result.Error = fmt.Errorf("请求失败: %w", err)
		return result
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		result.Error = fmt.Errorf("获取 manifest 失败 (状态码: %d): %s", resp.StatusCode, string(body))
		return result
	}

	// 获取 Docker-Content-Digest header
	result.Digest = resp.Header.Get("Docker-Content-Digest")

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Error = fmt.Errorf("读取响应失败: %w", err)
		return result
	}

	result.Manifest = string(body)
	return result
}
