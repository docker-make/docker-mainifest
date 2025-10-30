# Docker Auth - Docker 镜像信息获取工具

一个用 Go 实现的 Docker 镜像 manifest 获取工具，支持 Docker Hub 和 GitHub Container Registry。

## 主要特点

- 🚀 **高效批量获取**：支持并发获取多个镜像，自动批量认证减少请求次数
- 🔒 **多 Registry 管理**：统一管理 Docker Hub、GHCR 等多个 registry 的凭据
- 🎯 **智能分组**：自动按 registry 类型和数量限制分组，优化认证性能
- 📊 **结构化日志**：基于 zap 的结构化日志，便于监控和调试
- 🔌 **可扩展**：支持注册自定义 registry，适配私有镜像仓库
- 🌐 **代理支持**：完整的 HTTP/HTTPS 代理支持
- 💪 **类型安全**：完整的 Go 类型系统，避免运行时错误

## 功能特性

- ✅ 支持从 Docker Hub 获取镜像 manifest
- ✅ 支持从 GitHub Container Registry (ghcr.io) 获取镜像 manifest
- ✅ 支持使用 Personal Access Token (PAT) 认证
- ✅ 支持匿名访问公开镜像
- ✅ **支持批量获取多个镜像信息（逗号分隔）**
- ✅ **支持批量获取 Manifest（顺序/并发，默认并发数=5）**
- ✅ **支持代理服务器（HTTP_PROXY、HTTPS_PROXY）**
- ✅ **支持批量认证 Token（一次认证访问多个镜像）**
- ✅ **多 Registry 凭据管理（同时支持 Docker Hub、GHCR 等）**
- ✅ **智能分组机制（自动按 registry 分组，每批最多 30 个镜像）**
- ✅ **并发控制和错误容错**
- ✅ **结构化日志支持（zap）**
- ✅ 可作为命令行工具或 Go 库使用

## 支持的 Registry

### Docker Hub
- Registry: `docker.io` / `registry-1.docker.io`
- Token: Docker Hub PAT (`dckr_pat_xxx...`)
- 镜像格式: `nginx`, `library/nginx`, `username/image`

### GitHub Container Registry
- Registry: `ghcr.io`
- Token: GitHub PAT (`ghp_xxx...` 或 `github_pat_xxx...`)
- 镜像格式: `ghcr.io/owner/repo`

## 安装

```bash
# 作为库使用
go get github.com/docker-make/docker-mainifest

# 或克隆仓库并构建
git clone https://github.com/docker-make/docker-mainifest.git
cd docker-auth
make build
```

## 快速开始

```bash
# 1. 获取单个公开镜像（无需认证）
./docker-auth -image nginx

# 2. 获取多个镜像（带认证）
./docker-auth -image nginx,redis,postgres \
  -dockerhub-username myuser \
  -dockerhub-token dckr_pat_xxx

# 3. 查看格式化的 manifest 和 digest
./docker-auth -image nginx -pretty -digest
```

## 使用方法

### 命令行工具

```bash
# 构建
go build -o docker-auth ./cmd/docker-auth
# 或使用 Makefile
make build

# Docker Hub - 单个镜像（匿名访问公开镜像）
./docker-auth -image nginx -tag latest

# Docker Hub - 单个镜像带认证
./docker-auth -image nginx -tag latest -dockerhub-username myuser -dockerhub-token dckr_pat_owm...

# Docker Hub - 多个镜像（逗号分隔）
./docker-auth -image nginx,redis,postgres -dockerhub-username myuser -dockerhub-token dckr_pat_owm...

# 多个镜像，每个带不同标签
./docker-auth -image nginx:latest,redis:alpine,postgres:14 -dockerhub-username myuser -dockerhub-token dckr_pat_owm...

# 匿名访问公开镜像
./docker-auth -image nginx,redis,alpine

# GitHub Container Registry 带认证
./docker-auth -image ghcr.io/owner/repo -tag latest -ghcr-username ghuser -ghcr-token ghp_xxx...

# 同时访问多个 registry（使用通用凭据格式）
./docker-auth -image nginx,ghcr.io/owner/repo \
  -credentials dockerhub:myuser:dckr_pat_xxx \
  -credentials ghcr:ghuser:ghp_xxx

# 格式化输出并显示 digest
./docker-auth -image nginx -pretty -digest

# 使用代理
export HTTP_PROXY=http://proxy.example.com:8080
export HTTPS_PROXY=http://proxy.example.com:8080
./docker-auth -image nginx
```

### 作为 Go 库使用

#### 基础用法

```go
package main

import (
    "fmt"
    "log"
    "github.com/docker-make/docker-mainifest/pkg/registry"
)

func main() {
    // 创建客户端
    client := registry.NewClient()
    
    // 为 Docker Hub 添加凭据
    client.AddCredential(registry.DockerHubKey, "myuser", "dckr_pat_owm...")
    
    // 获取单个镜像的 manifest 和 digest
    manifest, digest, err := client.GetManifestWithDigest("nginx", "latest")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Digest: %s\n", digest)
    fmt.Println(manifest)

    // 匿名访问公开镜像（无需添加凭据）
    anonClient := registry.NewClient()
    manifest, digest, err = anonClient.GetManifestWithDigest("library/nginx", "latest")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(manifest)

    // 同时支持多个 registry
    multiClient := registry.NewClient()
    multiClient.AddCredential(registry.DockerHubKey, "myuser", "dckr_pat_xxx...")
    multiClient.AddCredential(registry.GHCRKey, "ghuser", "ghp_xxx...")
    
    // 访问 Docker Hub 镜像
    manifest, _, _ = multiClient.GetManifestWithDigest("nginx", "latest")
    
    // 访问 GHCR 镜像
    manifest, _, _ = multiClient.GetManifestWithDigest("ghcr.io/owner/repo", "latest")
}
```

#### 批量获取多个镜像的 Token

```go
package main

import (
    "fmt"
    "log"
    "github.com/docker-make/docker-mainifest/pkg/registry"
)

func main() {
    // 创建客户端并添加凭据
    client := registry.NewClient()
    client.AddCredential(registry.DockerHubKey, "myuser", "dckr_pat_owm...")
    
    // 获取可以访问多个镜像的批量 token
    images := []string{"library/nginx", "library/redis", "library/postgres"}
    token, err := client.GetAuthTokenForImages(images, registry.DockerHubKey)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Token (可访问 %d 个镜像): %s\n", len(images), token)
    
    // 估算最大批处理大小
    maxImages := client.EstimateMaxImagesForBatch(images, registry.DockerHubKey)
    fmt.Printf("估算的最大批处理大小: %d 个镜像\n", maxImages)
}
```

#### 批量获取多个镜像的 Manifest

```go
package main

import (
    "fmt"
    "github.com/docker-make/docker-mainifest/pkg/registry"
)

func main() {
    // 创建客户端并配置凭据
    client := registry.NewClient()
    client.AddCredential(registry.DockerHubKey, "myuser", "dckr_pat_xxx...")
    
    // 定义要获取的镜像列表
    imageSpecs := []registry.ImageSpec{
        {Image: "nginx", Tag: "latest"},
        {Image: "redis", Tag: "alpine"},
        {Image: "postgres", Tag: "14"},
    }
    
    // 方式1: 并发获取（默认并发=5），使用批量认证
    results := client.GetManifestsWithDigest(imageSpecs, 5, true, nil)
    
    // 方式2: 顺序获取，使用批量认证
    // results := client.GetManifestsWithDigest(imageSpecs, 0, true, nil)
    
    // 方式3: 自定义批处理大小（每批最多 20 个镜像）
    // batchSize := 20
    // results := client.GetManifestsWithDigest(imageSpecs, 5, true, &batchSize)
    
    // 处理结果
    successCount := 0
    for _, result := range results {
        if result.Error != nil {
            fmt.Printf("✗ %s:%s 失败: %v\n", result.Image, result.Tag, result.Error)
            continue
        }
        successCount++
        fmt.Printf("✓ %s:%s\n", result.Image, result.Tag)
        fmt.Printf("  Digest: %s\n", result.Digest)
        fmt.Printf("  Manifest 长度: %d 字节\n\n", len(result.Manifest))
    }
    
    fmt.Printf("总计: %d 个镜像，成功: %d，失败: %d\n", 
        len(results), successCount, len(results)-successCount)
}
```

#### 使用代理服务器

```go
package main

import (
    "log"
    "github.com/docker-make/docker-mainifest/pkg/registry"
)

func main() {
    // 方法1: 自动从环境变量读取代理（HTTP_PROXY, HTTPS_PROXY）
    client := registry.NewClient()
    client.AddCredential(registry.DockerHubKey, "myuser", "token")
    
    // 方法2: 手动指定代理
    client, err := registry.NewClientWithProxy("http://proxy.example.com:8080")
    if err != nil {
        log.Fatal(err)
    }
    client.AddCredential(registry.DockerHubKey, "myuser", "token")
    
    // 获取 manifest
    manifest, digest, _ := client.GetManifestWithDigest("nginx", "latest")
    println(digest)
    println(manifest)
}
```

#### 使用结构化日志

```go
package main

import (
    "github.com/docker-make/docker-mainifest/pkg/registry"
    "go.uber.org/zap"
)

func main() {
    // 创建 logger（生产环境配置）
    logger, _ := zap.NewProduction()
    defer logger.Sync()
    
    // 创建带日志的客户端
    client := registry.NewClientWithLogger(logger)
    client.AddCredential(registry.DockerHubKey, "myuser", "token")
    
    // 或者为已有客户端设置 logger
    // client := registry.NewClient()
    // client.WithLogger(logger)
    
    // 批量获取时会自动输出结构化日志
    imageSpecs := []registry.ImageSpec{
        {Image: "nginx", Tag: "latest"},
        {Image: "redis", Tag: "alpine"},
    }
    results := client.GetManifestsWithDigest(imageSpecs, 5, true, nil)
    
    // 日志会显示分组信息、进度等
    for _, result := range results {
        if result.Error != nil {
            logger.Error("获取失败", 
                zap.String("image", result.Image),
                zap.Error(result.Error))
        }
    }
}
```

## 认证说明

### Docker Hub PAT
从 Docker Hub 获取 Personal Access Token:
1. 登录 Docker Hub
2. 进入 Account Settings -> Security
3. 创建新的 Access Token
4. Token 格式: `dckr_pat_xxx...`

### GitHub PAT
从 GitHub 获取 Personal Access Token:
1. 登录 GitHub
2. 进入 Settings -> Developer settings -> Personal access tokens
3. 创建新的 token，需要 `read:packages` 权限
4. Token 格式: `ghp_xxx...` 或 `github_pat_xxx...`

## API 文档

### 客户端创建

#### `registry.NewClient() *Client`
创建一个新的空 registry 客户端，自动从环境变量读取代理设置（HTTP_PROXY, HTTPS_PROXY）。

返回一个空客户端，需要使用 `AddCredential()` 添加凭据。

#### `registry.NewClientWithCredentials(credentials map[string]*RegistryCredential) *Client`
创建一个带初始凭据的 registry 客户端。

- `credentials`: registry key 到凭据的映射

#### `registry.NewClientWithProxy(proxyURL string) (*Client, error)`
创建带有自定义代理的 registry 客户端。

- `proxyURL`: 代理服务器地址（如 `http://proxy.example.com:8080`）

#### `registry.NewClientWithLogger(logger *zap.Logger) *Client`
创建带自定义日志记录器的 registry 客户端。

- `logger`: zap.Logger 实例

### 凭据管理

#### `client.AddCredential(registryKey, username, token string)`
为指定 registry 添加或更新凭据。

- `registryKey`: registry 标识（如 `registry.DockerHubKey`, `registry.GHCRKey`）
- `username`: 用户名
- `token`: 认证 token

#### `client.RemoveCredential(registryKey string)`
删除指定 registry 的凭据。

#### `client.GetCredential(registryKey string) (*RegistryCredential, bool)`
获取指定 registry 的凭据。

#### `client.WithLogger(logger *zap.Logger) *Client`
为已存在的客户端设置 logger，支持链式调用。

### Manifest 获取

#### `client.GetManifestWithDigest(image, tag string) (manifest, digest string, err error)`
获取单个镜像的 manifest 并返回其 digest。

- `image`: 镜像名称
  - Docker Hub: `nginx`, `library/nginx`, `username/image`
  - GHCR: `ghcr.io/owner/repo`
- `tag`: 镜像标签（如 `latest`, `1.0.0`）

返回：
- `manifest`: Manifest JSON 字符串
- `digest`: Manifest digest（如 `sha256:xxx...`）
- `err`: 错误信息

#### `client.GetManifestsWithDigest(imageSpecs []ImageSpec, concurrency int, batchAuth bool, maxBatchSize *int) []ManifestResult`
批量获取多个镜像的 manifest 和 digest。

参数：
- `imageSpecs`: 镜像规格列表 `[]ImageSpec{{Image: "nginx", Tag: "latest"}, {Image: "redis", Tag: "alpine"}}`
- `concurrency`: 并发数（0 表示顺序执行，> 0 表示并发执行，建议 5-10）
- `batchAuth`: 是否使用批量认证（推荐 true，可显著减少认证请求）
- `maxBatchSize`: 每批最大镜像数量（可选，默认 30，范围 1-30）

**智能分组机制：**
- 自动按 registry 类型分组（Docker Hub、GHCR 等）
- 每个 registry 组自动限制最多 30 个镜像（或自定义大小）
- 超过限制自动分成多个子组
- 每个子组获取独立的批量认证 token
- 支持混合多个 registry 的镜像

返回：`[]ManifestResult`，每个结果包含：
- `Image`: 镜像名称
- `Tag`: 镜像标签
- `Manifest`: Manifest JSON 字符串
- `Digest`: Manifest digest
- `Error`: 错误信息（如果获取失败）

### 批量认证

#### `client.GetAuthTokenForImages(images []string, registryKey string) (string, error)`
获取可以访问多个镜像的批量 bearer token。

参数：
- `images`: 镜像列表（如 `[]string{"nginx", "redis", "postgres"}`）
- `registryKey`: Registry 标识（如 `registry.DockerHubKey`, `registry.GHCRKey`）

限制：
- **建议不超过 50 个镜像**（受 URL 长度限制）
- 镜像名称越长，支持的数量越少
- 超过限制会返回错误

返回一个 token，可用于访问所有指定的镜像。

#### `client.GetAuthTokenWithScopes(scopes []string, registryKey string) (string, error)`
使用指定的 scopes 获取认证 token（低级 API）。

参数：
- `scopes`: scope 列表（如 `[]string{"repository:nginx:pull", "repository:redis:pull"}`）
- `registryKey`: Registry 标识

用于高级场景，通常不需要直接调用。

#### `client.BuildAuthURLWithScopes(config *RegistryConfig, scopes []string) (string, error)`
构建认证服务的 URL（支持多个 scope）。

会自动检查 URL 长度限制（2048 字符），超过会返回错误。

#### `client.EstimateMaxImagesForBatch(sampleImages []string, registryKey string) int`
估算在不超过 URL 长度限制的情况下，可以一次性获取多少个镜像的 token。

这是一个辅助函数，帮助确定合适的批处理大小。基于样本镜像名称计算平均长度，然后估算最大容量。

### 命令行参数

```
-image string
    镜像名称（必填）
    支持单个或多个镜像，多个镜像用逗号分隔
    示例: nginx 或 nginx,redis,postgres
    支持带标签: nginx:latest,redis:alpine

-tag string
    镜像标签（默认: latest）
    注意: 如果镜像名中已包含标签（如 nginx:1.19），此参数将被忽略

-dockerhub-username string
    Docker Hub 用户名（可选）
    需要配合 -dockerhub-token 使用

-dockerhub-token string
    Docker Hub token（可选）
    格式: dckr_pat_xxx...

-ghcr-username string
    GitHub 用户名（可选）
    需要配合 -ghcr-token 使用

-ghcr-token string
    GitHub token（可选）
    格式: ghp_xxx... 或 github_pat_xxx...

-credentials string
    通用凭据格式（可重复使用）
    格式: registry:username:token
    示例: -credentials dockerhub:user1:token1 -credentials ghcr:user2:token2
    支持同时配置多个 registry 的凭据

-pretty
    格式化输出 JSON（默认: false）

-digest
    显示 manifest digest（默认: false）
```

### 常量定义

```go
// Registry Key 常量
registry.DockerHubKey = "dockerhub"  // Docker Hub registry 标识
registry.GHCRKey = "ghcr"            // GitHub Container Registry 标识
```

### 自定义 Registry（高级功能）

#### `registry.RegisterRegistry(key string, config RegistryConfig) error`
注册一个自定义 registry。

参数：
- `key`: registry 的唯一标识符（不能与内置的冲突）
- `config`: registry 配置信息

```go
config := registry.RegistryConfig{
    Name:        "My Private Registry",
    RegistryURL: "https://my-registry.example.com",
    AuthURL:     "https://my-registry.example.com/auth",
    Service:     "my-registry.example.com",
}
err := registry.RegisterRegistry("myregistry", config)
```

#### `registry.GetRegistry(key string) (*RegistryConfig, bool)`
获取指定 key 的 registry 配置。

#### `registry.UnregisterRegistry(key string) error`
注销一个自定义 registry（不能删除内置 registry）。

#### `registry.ListRegistries() map[string]*RegistryConfig`
列出所有已注册的 registry（包括内置和自定义）。

#### `registry.DetectRegistry(image string) string`
根据镜像名称自动检测应该使用哪个 registry。

#### `registry.NormalizeImageName(image, registryKey string) string`
规范化镜像名称：
- Docker Hub: 无 `/` 的镜像自动添加 `library/` 前缀
- GHCR: 移除 `ghcr.io/` 前缀
- 自定义: 移除域名前缀

## 依赖项

本项目使用以下第三方库：

```go
require (
    go.uber.org/zap v1.27.0      // 结构化日志库
    go.uber.org/multierr v1.10.0 // 多错误处理（zap 依赖）
)
```

## 性能优化建议

### 批量获取优化

1. **使用批量认证**：设置 `batchAuth=true` 可以减少 80% 以上的认证请求
   ```go
   results := client.GetManifestsWithDigest(imageSpecs, 5, true, nil)
   ```

2. **适当的并发数**：建议并发数设置为 5-10，过高的并发可能触发 rate limit
   ```go
   concurrency := 5  // 推荐值
   results := client.GetManifestsWithDigest(imageSpecs, concurrency, true, nil)
   ```

3. **批处理大小**：默认每批 30 个镜像，可以根据镜像名长度调整
   ```go
   batchSize := 20  // 镜像名较长时减小批处理大小
   results := client.GetManifestsWithDigest(imageSpecs, 5, true, &batchSize)
   ```

### 错误处理

批量获取时建议检查每个结果的错误：

```go
for _, result := range results {
    if result.Error != nil {
        log.Printf("获取 %s:%s 失败: %v", result.Image, result.Tag, result.Error)
        // 实现重试逻辑或记录错误
        continue
    }
    // 处理成功的结果
}
```

### 日志级别

生产环境建议使用 `zap.NewProduction()`，开发环境使用 `zap.NewDevelopment()`：

```go
// 生产环境：JSON 格式，只记录 Info 及以上级别
logger, _ := zap.NewProduction()

// 开发环境：友好格式，记录 Debug 及以上级别
logger, _ := zap.NewDevelopment()
```

## 常见问题

### Q: 批量获取时部分镜像失败怎么办？

A: 批量获取采用"尽力而为"的策略，即使部分镜像失败，其他镜像仍会继续获取。检查返回结果中的 `Error` 字段即可。

### Q: URL 长度限制错误如何解决？

A: 减少每批的镜像数量，或使用 `EstimateMaxImagesForBatch()` 估算合适的批处理大小：

```go
maxSize := client.EstimateMaxImagesForBatch(sampleImages, registry.DockerHubKey)
results := client.GetManifestsWithDigest(imageSpecs, 5, true, &maxSize)
```

### Q: 如何同时访问多个 Registry？

A: 使用通用凭据格式或分别添加凭据：

```bash
# 命令行
./docker-auth -image nginx,ghcr.io/owner/repo \
  -credentials dockerhub:user1:token1 \
  -credentials ghcr:user2:token2

# Go 代码
client := registry.NewClient()
client.AddCredential(registry.DockerHubKey, "user1", "token1")
client.AddCredential(registry.GHCRKey, "user2", "token2")
```

### Q: 私有镜像仓库如何使用？

A: 注册自定义 registry：

```go
config := registry.RegistryConfig{
    Name:        "My Private Registry",
    RegistryURL: "https://registry.example.com",
    AuthURL:     "https://registry.example.com/auth",
    Service:     "registry.example.com",
}
registry.RegisterRegistry("private", config)

client.AddCredential("private", "username", "token")
```

## 版本要求

- Go 1.21 或更高版本
- 支持的操作系统：Linux、macOS、Windows

## 许可证

MIT License

