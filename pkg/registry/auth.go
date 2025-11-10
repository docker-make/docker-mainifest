package registry

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

// tokenResponse 表示认证服务器返回的 token 响应
type tokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// getAuthToken 获取用于访问 registry 的 bearer token
func (c *Client) getAuthToken(image string, registryKey string) (string, error) {
	// 规范化镜像名称
	normalizedImage := NormalizeImageName(image, registryKey)

	// 构建 scope
	scopes := []string{fmt.Sprintf("repository:%s:pull", normalizedImage)}

	return c.GetAuthTokenWithScopes(scopes, registryKey)
}

// GetAuthTokenForImages 获取可以访问多个镜像的 bearer token
// images: 镜像列表，如 []string{"nginx", "redis", "postgres"}
// 返回的 token 可以访问所有指定的镜像
//
// 限制：
//   - 建议镜像数量不超过 50 个（受 URL 长度和服务器限制）
//   - 如果需要访问更多镜像，建议分批获取 token 或使用缓存机制
//   - 镜像名称越长，支持的数量越少
func (c *Client) GetAuthTokenForImages(images []string, registryKey string) (string, error) {
	if len(images) == 0 {
		return "", fmt.Errorf("镜像列表不能为空")
	}

	// 建议的最大数量（保守估计）
	const maxRecommendedImages = 50
	if len(images) > maxRecommendedImages {
		c.logger.Warn("请求的镜像数量超过建议值",
			zap.Int("count", len(images)),
			zap.Int("maxRecommended", maxRecommendedImages),
			zap.String("message", "可能会遇到 URL 长度限制或服务器拒绝"))
	}

	// 为每个镜像构建 scope
	scopes := make([]string, 0, len(images))
	for _, image := range images {
		normalizedImage := NormalizeImageName(image, registryKey)
		scope := fmt.Sprintf("repository:%s:pull", normalizedImage)
		scopes = append(scopes, scope)
	}

	return c.GetAuthTokenWithScopes(scopes, registryKey)
}

// getAuthTokenWithScopes 使用指定的 scopes 获取认证 token
func (c *Client) GetAuthTokenWithScopes(scopes []string, registryKey string) (string, error) {
	// 获取 registry 配置
	config, ok := GetRegistry(registryKey)
	if !ok {
		return "", fmt.Errorf("未找到 registry 配置: %s", registryKey)
	}

	// 构建认证 URL
	authURL, err := c.BuildAuthURLWithScopes(config, scopes)
	if err != nil {
		return "", fmt.Errorf("构建认证 URL 失败: %w", err)
	}

	// 创建请求
	req, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建认证请求失败: %w", err)
	}

	// 如果有凭据，添加 Basic Auth
	// 根据 registry key 查找对应的凭据
	if cred, ok := c.GetCredential(registryKey); ok && cred.Username != "" && cred.Token != "" {
		auth := cred.Username + ":" + cred.Token
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encodedAuth)
	}

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("认证请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("认证失败 (状态码: %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取认证响应失败: %w", err)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析认证响应失败: %w", err)
	}

	// 返回 token（优先使用 token 字段，如果没有则使用 access_token）
	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("认证响应中没有找到 token")
}

// buildAuthURLWithScopes 构建认证服务的 URL（支持多个 scope）
func (c *Client) BuildAuthURLWithScopes(config *RegistryConfig, scopes []string) (string, error) {
	var finalURL string

	switch config.Key {
	case DockerHubKey:
		// Docker Hub 使用独立的认证服务
		authURL := config.AuthURL + "/token"
		params := url.Values{}
		params.Set("service", config.Service)
		// 添加多个 scope 参数
		for _, scope := range scopes {
			params.Add("scope", scope)
		}
		finalURL = authURL + "?" + params.Encode()

	case GHCRKey:
		// GitHub Container Registry 使用 OAuth2 token endpoint
		authURL := config.AuthURL + "/token"
		params := url.Values{}
		// 添加多个 scope 参数
		for _, scope := range scopes {
			params.Add("scope", scope)
		}
		// GHCR 不需要 service 参数，但需要正确的 scope 格式
		finalURL = authURL + "?" + params.Encode()

	default:
		// 自定义 registry，使用标准认证流程
		authURL := config.AuthURL + "/token"
		params := url.Values{}
		if config.Service != "" {
			params.Set("service", config.Service)
		}
		for _, scope := range scopes {
			params.Add("scope", scope)
		}
		finalURL = authURL + "?" + params.Encode()
	}

	// 检查 URL 长度（保守的限制是 2048 字符）
	const maxURLLength = 2048
	if len(finalURL) > maxURLLength {
		return "", fmt.Errorf("生成的 URL 太长 (%d 字符 > %d)，请减少镜像数量或使用分批处理",
			len(finalURL), maxURLLength)
	}

	return finalURL, nil
}

// EstimateMaxImagesForBatch 估算在不超过 URL 长度限制的情况下，可以一次性获取多少个镜像的 token
// 这是一个辅助函数，帮助您确定合适的批处理大小
func (c *Client) EstimateMaxImagesForBatch(sampleImages []string, registryKey string) int {
	if len(sampleImages) == 0 {
		return 0
	}

	// 获取 registry 配置
	config, ok := GetRegistry(registryKey)
	if !ok {
		return 0
	}

	// 计算平均镜像名称长度（包含 URL 编码后）
	totalLength := 0
	for _, image := range sampleImages {
		normalizedImage := NormalizeImageName(image, registryKey)
		scope := fmt.Sprintf("repository:%s:pull", normalizedImage)
		encodedScope := url.QueryEscape(scope)
		totalLength += len(encodedScope) + len("&scope=")
	}

	avgScopeLength := totalLength / len(sampleImages)

	// 基础 URL 长度
	baseURL := config.AuthURL + "/token?service=" + config.Service
	baseLength := len(baseURL)

	// 保守的 URL 长度限制
	const maxURLLength = 2048

	// 计算可以容纳多少个 scope
	availableLength := maxURLLength - baseLength
	maxImages := availableLength / avgScopeLength

	// 保守估计，减少 10%
	maxImages = int(float64(maxImages) * 0.9)

	if maxImages < 1 {
		maxImages = 1
	}

	return maxImages
}

// parseWWWAuthenticate 解析 WWW-Authenticate header（如果需要动态获取认证参数）
// 这个函数可以在未来用于更灵活的认证流程
func ParseWWWAuthenticate(header string) (realm, service, scope string, err error) {
	// WWW-Authenticate: Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:library/nginx:pull"

	if !strings.HasPrefix(header, "Bearer ") {
		return "", "", "", fmt.Errorf("不支持的认证类型")
	}

	// 移除 "Bearer " 前缀
	params := strings.TrimPrefix(header, "Bearer ")

	// 解析参数
	parts := strings.Split(params, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")

		switch key {
		case "realm":
			realm = value
		case "service":
			service = value
		case "scope":
			scope = value
		}
	}

	if realm == "" {
		return "", "", "", fmt.Errorf("未找到 realm 参数")
	}

	return realm, service, scope, nil
}

// getAuthTokenViaWWWAuthenticate 通过 WWW-Authenticate 动态获取认证 token
// 用于未注册的自定义 registry
func (c *Client) getAuthTokenViaWWWAuthenticate(registryURL, image string) (string, error) {
	// 首先尝试访问 manifest 接口，不带认证
	manifestURL := fmt.Sprintf("%s/v2/%s/manifests/latest", registryURL, image)

	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建探测请求失败: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("探测请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 如果不是 401，说明不需要认证或有其他问题
	if resp.StatusCode != http.StatusUnauthorized {
		return "", fmt.Errorf("未预期的响应状态: %d", resp.StatusCode)
	}

	// 解析 WWW-Authenticate header
	wwwAuth := resp.Header.Get("Www-Authenticate")
	if wwwAuth == "" {
		return "", fmt.Errorf("未找到 Www-Authenticate header")
	}

	realm, service, scope, err := ParseWWWAuthenticate(wwwAuth)
	if err != nil {
		return "", fmt.Errorf("解析 WWW-Authenticate 失败: %w", err)
	}

	c.logger.Debug("从 WWW-Authenticate 获取认证参数",
		zap.String("realm", realm),
		zap.String("service", service),
		zap.String("scope", scope))

	// 构建认证 URL
	authURL := realm
	params := url.Values{}
	if service != "" {
		params.Set("service", service)
	}
	if scope != "" {
		params.Set("scope", scope)
	}

	if len(params) > 0 {
		authURL += "?" + params.Encode()
	}

	// 请求 token
	authReq, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建认证请求失败: %w", err)
	}

	// 尝试添加凭据（如果有的话）
	// 对于自定义源，尝试使用域名作为 key 查找凭据
	domain := extractDomain(registryURL)
	if cred, ok := c.GetCredential(domain); ok && cred.Username != "" && cred.Token != "" {
		auth := cred.Username + ":" + cred.Token
		encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
		authReq.Header.Set("Authorization", "Basic "+encodedAuth)
	}

	authResp, err := c.httpClient.Do(authReq)
	if err != nil {
		return "", fmt.Errorf("认证请求失败: %w", err)
	}
	defer authResp.Body.Close()

	if authResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(authResp.Body)
		return "", fmt.Errorf("认证失败 (状态码: %d): %s", authResp.StatusCode, string(body))
	}

	// 解析 token 响应
	body, err := io.ReadAll(authResp.Body)
	if err != nil {
		return "", fmt.Errorf("读取认证响应失败: %w", err)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析认证响应失败: %w", err)
	}

	// 返回 token
	if tokenResp.Token != "" {
		return tokenResp.Token, nil
	}
	if tokenResp.AccessToken != "" {
		return tokenResp.AccessToken, nil
	}

	return "", fmt.Errorf("认证响应中没有找到 token")
}

// extractDomain 从 URL 中提取域名
func extractDomain(urlStr string) string {
	// 移除 https:// 或 http:// 前缀
	urlStr = strings.TrimPrefix(urlStr, "https://")
	urlStr = strings.TrimPrefix(urlStr, "http://")

	// 获取域名部分
	parts := strings.Split(urlStr, "/")
	if len(parts) > 0 {
		return parts[0]
	}

	return urlStr
}
