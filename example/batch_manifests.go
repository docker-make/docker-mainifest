package main

import (
	"fmt"
	"os"

	"github.com/docker-make/docker-mainifest/pkg/registry"
)

func BatchManifests() {
	fmt.Println("=== 批量获取镜像 Manifest 示例 ===")

	// 创建客户端
	client, err := registry.NewClientWithProxy("http://127.0.0.1:8899")
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		return
	}

	// 从环境变量获取认证信息（可选）
	// Docker Hub 凭据
	dockerhubUsername := os.Getenv("DOCKERHUB_USERNAME")
	dockerhubToken := os.Getenv("DOCKERHUB_TOKEN")
	if dockerhubUsername != "" && dockerhubToken != "" {
		client.AddCredential(registry.DockerHubKey, dockerhubUsername, dockerhubToken)
		fmt.Println("✓ 已配置 Docker Hub 凭据")
	}

	// GitHub Container Registry 凭据
	ghcrUsername := os.Getenv("GHCR_USERNAME")
	ghcrToken := os.Getenv("GHCR_TOKEN")
	if ghcrUsername != "" && ghcrToken != "" {
		client.AddCredential(registry.GHCRKey, ghcrUsername, ghcrToken)
		fmt.Println("✓ 已配置 GitHub Container Registry 凭据")
	}

	// 定义要获取的镜像列表
	// 注意：如果超过 30 个镜像，会自动分成多个批次
	imageSpecs := []registry.ImageSpec{
		{Image: "nginx", Tag: "latest"},
		{Image: "redis", Tag: "alpine"},
		{Image: "postgres", Tag: "14"},
		{Image: "mysql", Tag: "8.0"},
		{Image: "alpine", Tag: "latest"},
		{Image: "ghcr.io/gethomepage/homepage", Tag: "latest"},
		{Image: "ghcr.io/ofkm/arcane", Tag: "latest"},
		{Image: "jianxcao/media302", Tag: "latest"},
		// 可以添加 GHCR 镜像，会自动按 registry 类型分组
		// {Image: "ghcr.io/owner/repo", Tag: "latest"},
	}

	fmt.Printf("准备获取 %d 个镜像的 manifest...\n\n", len(imageSpecs))
	var results []registry.ManifestResult
	// 方式1: 顺序获取 + 批量认证
	fmt.Println("方式1: 顺序获取 + 批量认证")
	fmt.Println("-----------------------------------")
	results = client.GetManifestsWithDigest(imageSpecs, 0, true, nil)
	printResults(results)

	// 方式2: 并发获取 + 批量认证（推荐）
	fmt.Println("\n方式2: 并发获取（3个并发）+ 批量认证")
	fmt.Println("-----------------------------------")
	results = client.GetManifestsWithDigest(imageSpecs, 3, true, nil)
	printResults(results)

	// 方式3: 并发获取 + 单独认证
	fmt.Println("\n方式3: 并发获取（3个并发）+ 单独认证")
	fmt.Println("-----------------------------------")
	results = client.GetManifestsWithDigest(imageSpecs, 3, false, nil)
	printResults(results)
}

func printResults(results []registry.ManifestResult) {
	successCount := 0
	failCount := 0

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("✗ %s:%s - 失败\n", result.Image, result.Tag)
			fmt.Printf("  错误: %v\n", result.Error)
			failCount++
		} else {
			fmt.Printf("✓ %s:%s - 成功\n", result.Image, result.Tag)
			fmt.Printf("  Digest: %s\n", result.Digest)
			fmt.Printf("  Manifest: %d 字节\n", len(result.Manifest))
			successCount++
		}
		fmt.Println()
	}

	fmt.Printf("总计: %d 个镜像, 成功: %d, 失败: %d\n",
		len(results), successCount, failCount)
}

func main() {
	BatchManifests()
	// AddCustomRegistry()
}
