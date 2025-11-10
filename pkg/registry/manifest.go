package registry

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// GetManifestWithDigest 获取 manifest 并返回其 digest
// digest 可以用于确保镜像的完整性
func (c *Client) GetManifestWithDigest(image, tag string) (manifest string, digest string, err error) {
	// 检测 registry key
	registryKey := DetectRegistry(image)

	// 检查是否为未注册的自定义 registry
	isCustomUnregistered := false
	var customDomain string
	if len(registryKey) > 7 && registryKey[:7] == "custom:" {
		isCustomUnregistered = true
		customDomain = registryKey[7:] // 提取域名
		c.logger.Debug("检测到未注册的自定义源", zap.String("domain", customDomain))
	}

	var config *RegistryConfig
	var normalizedImage string
	var token string

	if isCustomUnregistered {
		// 对于未注册的自定义源，使用 WWW-Authenticate 流程
		// 构建 registry URL
		registryURL := "https://" + customDomain

		// 规范化镜像名称（移除域名前缀）
		parts := strings.SplitN(image, "/", 2)
		if len(parts) == 2 {
			normalizedImage = parts[1]
		} else {
			normalizedImage = image
		}

		// 通过 WWW-Authenticate 获取 token
		token, err = c.getAuthTokenViaWWWAuthenticate(registryURL, normalizedImage)
		if err != nil {
			return "", "", fmt.Errorf("通过 WWW-Authenticate 获取认证 token 失败: %w", err)
		}

		// 使用临时配置
		config = &RegistryConfig{
			RegistryURL: registryURL,
		}
	} else {
		// 对于已注册的 registry，使用标准流程
		var ok bool
		config, ok = GetRegistry(registryKey)
		if !ok {
			return "", "", fmt.Errorf("未找到 registry 配置: %s", registryKey)
		}

		// 规范化镜像名称
		normalizedImage = NormalizeImageName(image, registryKey)

		// 获取认证 token
		token, err = c.getAuthToken(image, registryKey)
		if err != nil {
			return "", "", fmt.Errorf("获取认证 token 失败: %w", err)
		}
	}

	// 构建 manifest URL
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/%s", config.RegistryURL, normalizedImage, tag)

	c.logger.Debug("获取 manifest", zap.String("url", manifestURL))
	// 创建请求
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("创建请求失败: %w", err)
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
		return "", "", fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("获取 manifest 失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	// 获取 Docker-Content-Digest header
	digest = resp.Header.Get("Docker-Content-Digest")

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %w", err)
	}

	return string(body), digest, nil
}

// ManifestResult 表示单个镜像的 manifest 获取结果
type ManifestResult struct {
	Image    string // 镜像名称
	Tag      string // 镜像标签
	Manifest string // Manifest JSON 字符串
	Digest   string // Manifest digest
	Error    error  // 错误信息（如果获取失败）
}

// ImageSpec 表示镜像规格（名称+标签）
type ImageSpec struct {
	Image string
	Tag   string
}

// GetManifestsWithDigest 批量获取多个镜像的 manifest 和 digest
// imageSpecs: 镜像规格列表
// concurrency: 并发数（0 表示顺序执行，> 0 表示并发执行）
// batchAuth: 是否使用批量认证（推荐，可以减少认证请求）
//
// 自动处理不同 registry 的镜像：
//   - 自动检测每个镜像的 registry 类型（Docker Hub、GHCR 等）
//   - 按 registry 分组
//   - 每组限制最多 30 个镜像，超过则继续分组
//   - 每组使用独立的批量认证 token
func (c *Client) GetManifestsWithDigest(imageSpecs []ImageSpec, concurrency int, batchAuth bool, maxBatchSize *int) []ManifestResult {
	if len(imageSpecs) == 0 {
		return nil
	}

	// 第一步：分组
	subGroups := c.groupImagesByRegistry(imageSpecs, maxBatchSize)

	// 第二步：为每个子组获取批量 token
	if batchAuth {
		c.acquireBatchTokens(subGroups)
	}

	// 第三步：获取 manifest
	results := make([]ManifestResult, len(imageSpecs))
	if concurrency <= 0 {
		c.fetchManifestsSequentially(subGroups, results)
	} else {
		c.fetchManifestsConcurrently(subGroups, results, concurrency)
	}

	return results
}
