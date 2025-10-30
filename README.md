# Docker Auth - Docker 镜像信息获取工具

一个用 Go 实现的 Docker 镜像 manifest 获取工具，支持 Docker Hub 和 GitHub Container Registry。

## 功能特性

- ✅ 支持从 Docker Hub 获取镜像 manifest
- ✅ 支持从 GitHub Container Registry (ghcr.io) 获取镜像 manifest
- ✅ 支持使用 Personal Access Token (PAT) 认证
- ✅ 支持匿名访问公开镜像
- ✅ **支持批量获取多个镜像信息（逗号分隔）**
- ✅ **支持批量获取 Manifest（顺序/并发）**
- ✅ **支持代理服务器（HTTP_PROXY、HTTPS_PROXY）**
- ✅ **支持批量认证 Token（一次认证访问多个镜像）**
- ✅ **并发控制和错误容错**
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
go get github.com/jianxiong-cao/docker-auth
```

## 使用方法

### 命令行工具

```bash
# 构建
go build -o docker-auth ./cmd/docker-auth

# Docker Hub - 单个镜像
./docker-auth -image nginx -tag latest -username myuser -token dckr_pat_owm...

# Docker Hub - 多个镜像（逗号分隔）
./docker-auth -image nginx,redis,postgres -username myuser -token dckr_pat_owm...

# 多个镜像，每个带不同标签
./docker-auth -image nginx:latest,redis:alpine,postgres:14 -username myuser -token dckr_pat_owm...

# 匿名访问公开镜像
./docker-auth -image nginx,redis,alpine

# GitHub Container Registry
./docker-auth -image ghcr.io/owner/repo -tag latest -username ghuser -token ghp_xxx...

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
    "github.com/jianxiong-cao/docker-auth/pkg/registry"
)

func main() {
    // 使用 Docker Hub PAT 认证
    client := registry.NewClient("myuser", "dckr_pat_owm...")
    manifest, err := client.GetManifest("library/nginx", "latest")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(manifest)

    // 匿名访问
    anonClient := registry.NewClient("", "")
    manifest, err = anonClient.GetManifest("library/nginx", "latest")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(manifest)

    // GitHub Container Registry
    ghcrClient := registry.NewClient("ghuser", "ghp_xxx...")
    manifest, err = ghcrClient.GetManifest("ghcr.io/owner/repo", "latest")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(manifest)
}
```

#### 批量获取多个镜像的 Token

```go
package main

import (
    "fmt"
    "log"
    "github.com/jianxiong-cao/docker-auth/pkg/registry"
)

func main() {
    client := registry.NewClient("myuser", "dckr_pat_owm...")
    
    // 获取可以访问多个镜像的 token
    images := []string{"library/nginx", "library/redis", "library/postgres"}
    token, err := client.GetAuthTokenForImages(images, registry.DockerHubConfig)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Token (可访问 %d 个镜像): %s\n", len(images), token)
    
    // 如果镜像很多，可以分批获取
    manyImages := make([]string, 100) // 假设有 100 个镜像
    tokens, err := client.GetAuthTokenForImagesBatch(manyImages, registry.DockerHubConfig, 20)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("获取了 %d 个 token，每个覆盖约 20 个镜像\n", len(tokens))
}
```

#### 批量获取多个镜像的 Manifest

```go
package main

import (
    "fmt"
    "log"
    "github.com/jianxiong-cao/docker-auth/pkg/registry"
)

func main() {
    client := registry.NewClient("myuser", "token")
    
    // 定义要获取的镜像列表
    imageSpecs := []registry.ImageSpec{
        {Image: "nginx", Tag: "latest"},
        {Image: "redis", Tag: "alpine"},
        {Image: "postgres", Tag: "14"},
    }
    
    // 方式1: 顺序获取，使用批量认证
    results := client.GetManifestsWithDigest(imageSpecs, 0, true)
    
    // 方式2: 并发获取（5 个并发），使用批量认证
    // results := client.GetManifestsWithDigest(imageSpecs, 5, true)
    
    // 处理结果
    for _, result := range results {
        if result.Error != nil {
            fmt.Printf("✗ %s:%s 失败: %v\n", result.Image, result.Tag, result.Error)
            continue
        }
        fmt.Printf("✓ %s:%s\n", result.Image, result.Tag)
        fmt.Printf("  Digest: %s\n", result.Digest)
        fmt.Printf("  Manifest 长度: %d 字节\n\n", len(result.Manifest))
    }
}
```

#### 使用代理服务器

```go
package main

import (
    "log"
    "github.com/jianxiong-cao/docker-auth/pkg/registry"
)

func main() {
    // 方法1: 自动从环境变量读取代理（HTTP_PROXY, HTTPS_PROXY）
    client := registry.NewClient("myuser", "token")
    
    // 方法2: 手动指定代理
    client, err := registry.NewClientWithProxy("myuser", "token", "http://proxy.example.com:8080")
    if err != nil {
        log.Fatal(err)
    }
    
    manifest, _ := client.GetManifest("nginx", "latest")
    println(manifest)
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

#### `registry.NewClient(username, token string) *Client`
创建一个新的 registry 客户端，自动从环境变量读取代理设置。

- `username`: 用户名（可选，匿名访问时传空字符串）
- `token`: PAT token（可选，匿名访问时传空字符串）

#### `registry.NewClientWithProxy(username, token, proxyURL string) (*Client, error)`
创建带有自定义代理的 registry 客户端。

- `proxyURL`: 代理服务器地址（如 `http://proxy.example.com:8080`）

### Manifest 获取

#### `client.GetManifest(image, tag string) (string, error)`
获取单个镜像的 manifest。

- `image`: 镜像名称
  - Docker Hub: `nginx`, `library/nginx`, `username/image`
  - GHCR: `ghcr.io/owner/repo`
- `tag`: 镜像标签（如 `latest`, `1.0.0`）

返回 manifest JSON 字符串。

#### `client.GetManifestWithDigest(image, tag string) (manifest, digest string, error)`
获取单个镜像的 manifest 并返回其 digest。

#### `client.GetManifestsWithDigest(imageSpecs []ImageSpec, concurrency int, batchAuth bool) []ManifestResult`
批量获取多个镜像的 manifest 和 digest。

- `imageSpecs`: 镜像规格列表 `[]ImageSpec{{"nginx", "latest"}, {"redis", "alpine"}}`
- `concurrency`: 并发数（0 表示顺序执行，> 0 表示并发执行）
- `batchAuth`: 是否使用批量认证（推荐，可减少认证请求）

**智能分组机制：**
- 自动按 registry 类型分组（Docker Hub、GHCR 等）
- 每个 registry 组自动限制最多 30 个镜像
- 超过 30 个自动分成多个子组
- 每个子组获取独立的批量认证 token

返回结果列表，每个结果包含：
- `Image`: 镜像名称
- `Tag`: 镜像标签
- `Manifest`: Manifest JSON 字符串
- `Digest`: Manifest digest
- `Error`: 错误信息（如果获取失败）

### 批量认证

#### `client.GetAuthTokenForImages(images []string, config RegistryConfig) (string, error)`
获取可以访问多个镜像的 bearer token。

- `images`: 镜像列表（如 `[]string{"nginx", "redis", "postgres"}`）
- `config`: Registry 配置（`registry.DockerHubConfig` 或 `registry.GHCRConfig`）
- **限制**: 建议不超过 50 个镜像

返回一个 token，可用于访问所有指定的镜像。

#### `client.GetAuthTokenForImagesBatch(images []string, config RegistryConfig, batchSize int) ([]string, error)`
分批获取多个镜像的 token。

- `batchSize`: 每批的镜像数量（建议 10-30）

返回 token 列表，每个 token 对应一批镜像。

#### `client.EstimateMaxImagesForBatch(sampleImages []string, config RegistryConfig) int`
估算在不超过 URL 长度限制的情况下，可以一次性获取多少个镜像的 token。

### 命令行参数

```
-image string
    镜像名称（必填）
    支持单个或多个镜像，多个镜像用逗号分隔
    示例: nginx 或 nginx,redis,postgres

-tag string
    镜像标签（默认: latest）
    注意: 如果镜像名中已包含标签，此参数将被忽略

-username string
    用户名（可选）

-token string
    认证 token（可选）

-pretty
    格式化输出 JSON

-digest
    显示 manifest digest
```

## 许可证

MIT License

