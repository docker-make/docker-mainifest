package main

import (
	"fmt"

	"github.com/docker-make/docker-mainifest/pkg/registry"
)

func MultiRegistryTest() {
	fmt.Println("=== 多 Registry 凭据管理测试 ===")

	// 创建客户端
	client := registry.NewClient()

	// 测试 1: 添加 Docker Hub 凭据
	fmt.Println("1. 添加 Docker Hub 凭据")
	client.AddCredential(registry.DockerHubKey, "dockerhub_user", "dockerhub_token")
	if cred, ok := client.GetCredential(registry.DockerHubKey); ok {
		fmt.Printf("   ✓ 成功: username=%s, token=%s\n\n", cred.Username, cred.Token)
	} else {
		fmt.Println("   ✗ 失败")
	}

	// 测试 2: 添加 GHCR 凭据
	fmt.Println("2. 添加 GitHub Container Registry 凭据")
	client.AddCredential(registry.GHCRKey, "github_user", "ghp_token")
	if cred, ok := client.GetCredential(registry.GHCRKey); ok {
		fmt.Printf("   ✓ 成功: username=%s, token=%s\n\n", cred.Username, cred.Token)
	} else {
		fmt.Println("   ✗ 失败")
	}

	// 测试 3: 注册自定义 registry
	fmt.Println("3. 注册自定义 registry")
	customConfig := registry.RegistryConfig{
		Name:        "My Registry",
		RegistryURL: "https://my-registry.example.com",
		AuthURL:     "https://my-registry.example.com/auth",
		Service:     "my-registry.example.com",
	}
	if err := registry.RegisterRegistry("my-registry", customConfig); err != nil {
		fmt.Printf("   ✗ 失败: %v\n\n", err)
	} else {
		fmt.Println("   ✓ 成功注册自定义 registry")
	}

	// 测试 4: 为自定义 registry 添加凭据
	fmt.Println("4. 为自定义 registry 添加凭据")
	client.AddCredential("my-registry", "custom_user", "custom_token")
	if cred, ok := client.GetCredential("my-registry"); ok {
		fmt.Printf("   ✓ 成功: username=%s, token=%s\n\n", cred.Username, cred.Token)
	} else {
		fmt.Println("   ✗ 失败")
	}

	// 测试 5: 列出所有 registry
	fmt.Println("5. 列出所有可用的 registry")
	registries := registry.ListRegistries()
	for key, config := range registries {
		fmt.Printf("   - %s: %s (%s)\n", key, config.Name, config.RegistryURL)
	}
	fmt.Println()

	// 测试 6: 更新凭据
	fmt.Println("6. 更新 Docker Hub 凭据")
	client.AddCredential(registry.DockerHubKey, "dockerhub_user_updated", "dockerhub_token_updated")
	if cred, ok := client.GetCredential(registry.DockerHubKey); ok {
		fmt.Printf("   ✓ 成功: username=%s, token=%s\n\n", cred.Username, cred.Token)
	} else {
		fmt.Println("   ✗ 失败")
	}

	// 测试 7: 删除凭据
	fmt.Println("7. 删除 GHCR 凭据")
	client.RemoveCredential(registry.GHCRKey)
	if _, ok := client.GetCredential(registry.GHCRKey); !ok {
		fmt.Println("   ✓ 成功: 凭据已删除")
	} else {
		fmt.Println("   ✗ 失败: 凭据仍然存在")
	}

	// 测试 8: 使用 NewClientWithCredentials 创建客户端
	fmt.Println("8. 使用 NewClientWithCredentials 创建客户端")
	credentials := map[string]*registry.RegistryCredential{
		registry.DockerHubKey: {Username: "user1", Token: "token1"},
		registry.GHCRKey:      {Username: "user2", Token: "token2"},
	}
	client2 := registry.NewClientWithCredentials(credentials)

	fmt.Println("   凭据列表:")
	for key := range credentials {
		if cred, ok := client2.GetCredential(key); ok {
			fmt.Printf("   - %s: username=%s, token=%s\n", key, cred.Username, cred.Token)
		}
	}
	fmt.Println()

	fmt.Println("=== 所有测试完成 ===")
}
