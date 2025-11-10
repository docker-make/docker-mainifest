package main

import (
	"fmt"

	"github.com/docker-make/docker-mainifest/pkg/registry"
)

// TestCustomRegistry 测试未注册的自定义镜像源
func TestCustomRegistry() {
	fmt.Println("=== 测试自定义镜像源 ===")

	// 创建客户端
	client := registry.NewClient()

	// 测试 lscr.io 的镜像
	fmt.Println("\n测试 lscr.io/linuxserver/phpmyadmin:latest")
	fmt.Println("-----------------------------------")

	manifest, digest, err := client.GetManifestWithDigest("lscr.io/linuxserver/phpmyadmin", "latest")
	if err != nil {
		fmt.Printf("✗ 获取 manifest 失败: %v\n", err)
	} else {
		fmt.Printf("✓ 获取 manifest 成功\n")
		fmt.Printf("  Digest: %s\n", digest)
		fmt.Printf("  Manifest: %d 字节\n", len(manifest))
	}

	// 测试其他自定义源
	fmt.Println("\n测试 quay.io/prometheus/prometheus:latest")
	fmt.Println("-----------------------------------")

	manifest, digest, err = client.GetManifestWithDigest("quay.io/prometheus/prometheus", "latest")
	if err != nil {
		fmt.Printf("✗ 获取 manifest 失败: %v\n", err)
	} else {
		fmt.Printf("✓ 获取 manifest 成功\n")
		fmt.Printf("  Digest: %s\n", digest)
		fmt.Printf("  Manifest: %d 字节\n", len(manifest))
	}

	// 测试标准 Docker Hub 镜像
	fmt.Println("\n测试 nginx:latest (Docker Hub)")
	fmt.Println("-----------------------------------")

	manifest, digest, err = client.GetManifestWithDigest("nginx", "latest")
	if err != nil {
		fmt.Printf("✗ 获取 manifest 失败: %v\n", err)
	} else {
		fmt.Printf("✓ 获取 manifest 成功\n")
		fmt.Printf("  Digest: %s\n", digest)
		fmt.Printf("  Manifest: %d 字节\n", len(manifest))
	}
}

// 如果要单独运行此文件，请取消下面的注释
// func main() {
// 	TestCustomRegistry()
// }
