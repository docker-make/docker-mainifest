package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/docker-make/docker-mainifest/pkg/registry"
)

// credentialsFlag 实现 flag.Value 接口，用于支持重复的 -credentials 参数
type credentialsFlag []string

func (c *credentialsFlag) String() string {
	return strings.Join(*c, ", ")
}

func (c *credentialsFlag) Set(value string) error {
	*c = append(*c, value)
	return nil
}

func main() {
	// 定义命令行参数
	image := flag.String("image", "", "镜像名称 (必填)\n"+
		"  支持单个或多个镜像，多个镜像用逗号分隔\n"+
		"  单个: nginx, library/nginx, ghcr.io/owner/repo\n"+
		"  多个: nginx,redis,postgres 或 nginx:latest,redis:alpine")
	tag := flag.String("tag", "latest", "镜像标签 (默认: latest)\n"+
		"  注意: 如果镜像名中已包含标签（如 nginx:1.19），此参数将被忽略")

	// 多种凭据配置方式
	dockerhubUsername := flag.String("dockerhub-username", "", "Docker Hub 用户名 (可选)")
	dockerhubToken := flag.String("dockerhub-token", "", "Docker Hub token (可选)\n"+
		"  格式: dckr_pat_xxx...")
	ghcrUsername := flag.String("ghcr-username", "", "GitHub 用户名 (可选)")
	ghcrToken := flag.String("ghcr-token", "", "GitHub token (可选)\n"+
		"  格式: ghp_xxx... 或 github_pat_xxx...")

	var credentialsList credentialsFlag
	flag.Var(&credentialsList, "credentials", "通用凭据格式 (可重复使用)\n"+
		"  格式: registry:username:token\n"+
		"  示例: -credentials dockerhub:user1:token1 -credentials ghcr:user2:token2")

	pretty := flag.Bool("pretty", false, "格式化输出 JSON (默认: false)")
	showDigest := flag.Bool("digest", false, "显示 manifest digest (默认: false)")

	// 自定义 Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Docker Auth - Docker 镜像信息获取工具\n\n")
		fmt.Fprintf(os.Stderr, "用法:\n")
		fmt.Fprintf(os.Stderr, "  %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  # Docker Hub - 单个镜像\n")
		fmt.Fprintf(os.Stderr, "  %s -image nginx -tag latest\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Docker Hub - 多个镜像\n")
		fmt.Fprintf(os.Stderr, "  %s -image nginx,redis,postgres\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 多个镜像带不同标签\n")
		fmt.Fprintf(os.Stderr, "  %s -image nginx:latest,redis:alpine,postgres:14\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Docker Hub 镜像带认证\n")
		fmt.Fprintf(os.Stderr, "  %s -image nginx -dockerhub-username user -dockerhub-token xxx\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # GitHub Container Registry 带认证\n")
		fmt.Fprintf(os.Stderr, "  %s -image ghcr.io/owner/repo -ghcr-username ghuser -ghcr-token ghp_xxx\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 同时访问多个 registry\n")
		fmt.Fprintf(os.Stderr, "  %s -image nginx,ghcr.io/owner/repo -credentials dockerhub:user1:token1 -credentials ghcr:user2:token2\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # 格式化输出并显示 digest\n")
		fmt.Fprintf(os.Stderr, "  %s -image nginx -pretty -digest\n\n", os.Args[0])
	}

	flag.Parse()

	// 检查必填参数
	if *image == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定镜像名称\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// 解析镜像列表（支持逗号分隔）
	imageList := strings.Split(*image, ",")

	// 清理空白字符
	var images []string
	for _, img := range imageList {
		img = strings.TrimSpace(img)
		if img != "" {
			images = append(images, img)
		}
	}

	if len(images) == 0 {
		fmt.Fprintf(os.Stderr, "错误: 没有有效的镜像名称\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// 创建客户端并配置凭据
	client := registry.NewClient()

	// 处理 Docker Hub 凭据
	if *dockerhubUsername != "" && *dockerhubToken != "" {
		client.AddCredential(registry.DockerHubKey, *dockerhubUsername, *dockerhubToken)
		fmt.Fprintf(os.Stderr, "已配置 Docker Hub 凭据\n")
	}

	// 处理 GHCR 凭据
	if *ghcrUsername != "" && *ghcrToken != "" {
		client.AddCredential(registry.GHCRKey, *ghcrUsername, *ghcrToken)
		fmt.Fprintf(os.Stderr, "已配置 GitHub Container Registry 凭据\n")
	}

	// 处理通用凭据格式
	for _, cred := range credentialsList {
		parts := strings.SplitN(cred, ":", 3)
		if len(parts) != 3 {
			fmt.Fprintf(os.Stderr, "警告: 凭据格式错误，应为 registry:username:token，跳过: %s\n", cred)
			continue
		}
		registryKey, username, token := parts[0], parts[1], parts[2]
		client.AddCredential(registryKey, username, token)
		fmt.Fprintf(os.Stderr, "已配置 %s 凭据\n", registryKey)
	}

	// 单个镜像：使用原有方式
	if len(images) == 1 {
		imageName, imageTag := parseImageAndTag(images[0], *tag)

		var manifestJSON string
		var err error
		var digest string
		manifestJSON, digest, err = client.GetManifestWithDigest(imageName, imageTag)

		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}

		if *showDigest && digest != "" {
			fmt.Fprintf(os.Stderr, "Digest: %s\n\n", digest)
		}

		printManifest(manifestJSON, *pretty)
		return
	}

	// 多个镜像：使用批量获取（更高效）
	fmt.Fprintf(os.Stderr, "准备批量获取 %d 个镜像...\n", len(images))

	// 构建 ImageSpec 列表
	imageSpecs := make([]registry.ImageSpec, len(images))
	for i, img := range images {
		imageName, imageTag := parseImageAndTag(img, *tag)
		imageSpecs[i] = registry.ImageSpec{
			Image: imageName,
			Tag:   imageTag,
		}
	}

	// 批量获取（并发=5，使用批量认证）
	results := client.GetManifestsWithDigest(imageSpecs, 5, true, nil)

	// 输出结果
	fmt.Fprintf(os.Stderr, "\n========================================\n")
	successCount := 0
	failCount := 0

	for i, result := range results {
		fmt.Fprintf(os.Stderr, "\n[%d/%d] 镜像: %s:%s\n", i+1, len(results), result.Image, result.Tag)
		fmt.Fprintf(os.Stderr, "----------------------------------------\n")

		if result.Error != nil {
			fmt.Fprintf(os.Stderr, "✗ 失败: %v\n", result.Error)
			failCount++
			continue
		}

		successCount++
		if *showDigest && result.Digest != "" {
			fmt.Fprintf(os.Stderr, "✓ Digest: %s\n", result.Digest)
		} else {
			fmt.Fprintf(os.Stderr, "✓ 成功\n")
		}

		// 输出 manifest
		printManifest(result.Manifest, *pretty)

		if i < len(results)-1 {
			fmt.Println()
		}
	}

	// 输出统计信息
	fmt.Fprintf(os.Stderr, "\n========================================\n")
	fmt.Fprintf(os.Stderr, "总计: %d 个镜像, 成功: %d, 失败: %d\n",
		len(results), successCount, failCount)

	if failCount > 0 {
		os.Exit(1)
	}
}

// printManifest 输出 manifest JSON
func printManifest(manifestJSON string, pretty bool) {
	if pretty {
		var jsonData interface{}
		if err := json.Unmarshal([]byte(manifestJSON), &jsonData); err != nil {
			fmt.Fprintf(os.Stderr, "警告: 无法解析 JSON，将输出原始数据\n")
			fmt.Println(manifestJSON)
		} else {
			prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
			if err != nil {
				fmt.Println(manifestJSON)
			} else {
				fmt.Println(string(prettyJSON))
			}
		}
	} else {
		fmt.Println(manifestJSON)
	}
}

// parseImageAndTag 解析镜像名称和标签
// 如果镜像名中包含标签（如 nginx:1.19），使用镜像中的标签
// 否则使用默认标签
func parseImageAndTag(image string, defaultTag string) (string, string) {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) == 2 {
		// 镜像名中包含标签
		return parts[0], parts[1]
	}
	// 使用默认标签
	return image, defaultTag
}
