package registry

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RegistryCredential 表示 registry 的认证凭据
type RegistryCredential struct {
	Username string
	Token    string
}

// Client 表示一个 Docker Registry 客户端
// 支持多个 registry 的独立凭据管理
type Client struct {
	httpClient  *http.Client
	credentials map[string]*RegistryCredential // registry key -> 凭据
	mu          sync.RWMutex                   // 保护 credentials 的并发访问
	logger      *zap.Logger                    // 日志记录器
}

// NewClient 创建一个空的 registry 客户端
// 自动从环境变量读取代理设置 (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
// 默认使用 nop logger（不输出日志）
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		},
		credentials: make(map[string]*RegistryCredential),
		logger:      zap.NewNop(),
	}
}

// NewClientWithCredentials 创建一个带初始凭据的 registry 客户端
// credentials: registry key -> 凭据的映射
// 默认使用 nop logger（不输出日志）
func NewClientWithCredentials(credentials map[string]*RegistryCredential) *Client {
	client := NewClient()
	for key, cred := range credentials {
		client.credentials[key] = cred
	}
	return client
}

// NewClientWithProxy 创建一个带有自定义代理的 registry 客户端
// proxyURL: 代理服务器地址，例如 "http://proxy.example.com:8080"
func NewClientWithProxy(proxyURL string) (*Client, error) {
	var transport *http.Transport

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	} else {
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		credentials: make(map[string]*RegistryCredential),
		logger:      zap.NewNop(),
	}, nil
}

// AddCredential 添加或更新指定 registry 的凭据
func (c *Client) AddCredential(registryKey, username, token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.credentials[registryKey] = &RegistryCredential{
		Username: username,
		Token:    token,
	}
}

// RemoveCredential 删除指定 registry 的凭据
func (c *Client) RemoveCredential(registryKey string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.credentials, registryKey)
}

// GetCredential 获取指定 registry 的凭据
func (c *Client) GetCredential(registryKey string) (*RegistryCredential, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cred, ok := c.credentials[registryKey]
	return cred, ok
}

// NewClientWithLogger 创建一个带自定义 logger 的 registry 客户端
// logger: 自定义的 zap.Logger 实例
func NewClientWithLogger(logger *zap.Logger) *Client {
	client := NewClient()
	if logger != nil {
		client.logger = logger
	}
	return client
}

// WithLogger 为已存在的 Client 设置 logger
// 返回 Client 本身以支持链式调用
func (c *Client) WithLogger(logger *zap.Logger) *Client {
	if logger != nil {
		c.logger = logger
	}
	return c
}
