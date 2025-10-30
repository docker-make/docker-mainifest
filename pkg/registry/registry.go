package registry

import (
	"fmt"
	"strings"
	"sync"
)

// RegistryConfig 存储 registry 的配置信息
type RegistryConfig struct {
	Key         string // registry 的唯一标识符
	Name        string // registry 的显示名称
	RegistryURL string // registry API 地址
	AuthURL     string // 认证服务地址
	Service     string // 服务名称
}

// Registry key 常量
const (
	DockerHubKey = "dockerhub"
	GHCRKey      = "ghcr"
)

var (
	// registries 存储所有已注册的 registry（包括内置和自定义）
	registries = map[string]*RegistryConfig{
		DockerHubKey: {
			Key:         DockerHubKey,
			Name:        "Docker Hub",
			RegistryURL: "https://registry-1.docker.io",
			AuthURL:     "https://auth.docker.io",
			Service:     "registry.docker.io",
		},
		GHCRKey: {
			Key:         GHCRKey,
			Name:        "GitHub Container Registry",
			RegistryURL: "https://ghcr.io",
			AuthURL:     "https://ghcr.io",
			Service:     "ghcr.io",
		},
	}
	registryMu sync.RWMutex
)

// DetectRegistry 根据镜像名称检测使用哪个 registry
// 返回 registry key
func DetectRegistry(image string) string {
	// 如果镜像以 ghcr.io/ 开头，使用 GitHub Container Registry
	if strings.HasPrefix(image, "ghcr.io/") {
		return GHCRKey
	}

	// 检查是否匹配其他自定义 registry
	registryMu.RLock()
	defer registryMu.RUnlock()

	for key, config := range registries {
		// 尝试从镜像名提取域名并匹配
		if strings.Contains(image, "/") {
			parts := strings.SplitN(image, "/", 2)
			domain := parts[0]
			if strings.Contains(domain, ".") {
				// 检查是否匹配 registry URL
				if strings.Contains(config.RegistryURL, domain) {
					return key
				}
			}
		}
	}

	// 默认使用 Docker Hub
	return DockerHubKey
}

// NormalizeImageName 规范化镜像名称
// 对于 Docker Hub，如果没有 / 则添加 library/ 前缀
// 对于 GHCR，移除 ghcr.io/ 前缀
// 对于自定义 registry，移除域名前缀
func NormalizeImageName(image, registryKey string) string {
	switch registryKey {
	case DockerHubKey:
		// 如果镜像名中没有 /，说明是官方镜像，添加 library/ 前缀
		if !strings.Contains(image, "/") {
			return "library/" + image
		}
		return image
	case GHCRKey:
		// 移除 ghcr.io/ 前缀
		return strings.TrimPrefix(image, "ghcr.io/")
	default:
		// 自定义 registry，尝试移除域名前缀
		parts := strings.SplitN(image, "/", 2)
		if len(parts) == 2 && strings.Contains(parts[0], ".") {
			return parts[1]
		}
		return image
	}
}

// RegisterRegistry 注册一个 registry
// key: registry 的唯一标识符
// config: registry 的配置信息
func RegisterRegistry(key string, config RegistryConfig) error {
	if key == "" {
		return fmt.Errorf("registry key 不能为空")
	}

	// 检查是否与内置 registry 冲突
	if key == DockerHubKey || key == GHCRKey {
		return fmt.Errorf("registry key '%s' 已被内置 registry 使用", key)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	// 检查是否已注册
	if _, exists := registries[key]; exists {
		return fmt.Errorf("registry key '%s' 已被注册", key)
	}

	// 设置 key
	config.Key = key
	if config.Name == "" {
		config.Name = key
	}

	registries[key] = &config
	return nil
}

// GetRegistry 获取指定 key 的 registry 配置
func GetRegistry(key string) (*RegistryConfig, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	config, ok := registries[key]
	return config, ok
}

// UnregisterRegistry 注销一个 registry
func UnregisterRegistry(key string) error {
	// 不能删除内置 registry
	if key == DockerHubKey || key == GHCRKey {
		return fmt.Errorf("不能删除内置 registry '%s'", key)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registries[key]; !exists {
		return fmt.Errorf("registry key '%s' 未注册", key)
	}

	delete(registries, key)
	return nil
}

// ListRegistries 列出所有已注册的 registry
func ListRegistries() map[string]*RegistryConfig {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[string]*RegistryConfig)
	for key, config := range registries {
		// 复制配置以避免外部修改
		configCopy := *config
		result[key] = &configCopy
	}

	return result
}
