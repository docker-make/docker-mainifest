# 更新日志

## v2.0.0 - 多 Registry 凭据支持 (2025-10-30)

### 重大变更（Breaking Changes）

#### API 变更
- `NewClient()` 现在不接受参数，返回空客户端
  - **旧**: `registry.NewClient(username, token string)`
  - **新**: `registry.NewClient()` + `client.AddCredential(registryKey, username, token)`

- 删除 `NewClientForRegistry(username, token, registryType)` 函数
  - 使用 `NewClient()` + `AddCredential()` 替代

#### 结构变更
- `Client` 结构重构，支持多 registry 凭据管理
  - 新增 `credentials map[string]*RegistryCredential` 字段
  - 新增 `registries map[string]*RegistryConfig` 字段
  - 删除 `config *RegistryConfig` 字段（不再需要向后兼容）

### 新增功能

#### 1. 多 Registry 凭据管理
- ✅ 支持为不同 registry 设置独立凭据
- ✅ 支持 Docker Hub、GitHub Container Registry
- ✅ 支持自定义 registry 注册
- ✅ 线程安全的凭据操作

#### 2. 新增 API

**Client 方法**
- `AddCredential(registryKey, username, token)` - 添加或更新凭据
- `RemoveCredential(registryKey)` - 删除凭据
- `GetCredential(registryKey)` - 获取凭据
- `RegisterCustomRegistry(key, config)` - 注册自定义 registry
- `GetRegistryConfig(key)` - 获取 registry 配置
- `ListRegistries()` - 列出所有可用 registry

**构造函数**
- `NewClient()` - 创建空客户端
- `NewClientWithCredentials(credentials)` - 使用初始凭据创建客户端
- `NewClientWithProxy(proxyURL)` - 创建带代理的客户端（已更新）

**全局函数**
- `RegisterCustomRegistry(key, config)` - 全局注册自定义 registry
- `GetRegistry(key)` - 获取 registry 配置
- `UnregisterCustomRegistry(key)` - 注销自定义 registry
- `ListRegistries()` - 列出所有全局 registry
- `GetRegistryKey(registryType)` - 获取 registry key

#### 3. 命令行增强

**新增参数**
- `-dockerhub-username` - Docker Hub 用户名
- `-dockerhub-token` - Docker Hub token
- `-ghcr-username` - GitHub 用户名
- `-ghcr-token` - GitHub token
- `-credentials` - 通用凭据格式（可重复使用）

**使用示例**
```bash
# 单个 registry
./docker-auth -image nginx -dockerhub-username user -dockerhub-token xxx

# 多个 registry
./docker-auth -image nginx,ghcr.io/owner/repo \
  -credentials dockerhub:user1:token1 \
  -credentials ghcr:user2:token2
```

#### 4. 自定义 Registry 支持

可以注册和使用自定义 registry：

```go
customConfig := registry.RegistryConfig{
    RegistryURL: "https://my-registry.example.com",
    AuthURL:     "https://my-registry.example.com/auth",
    Service:     "my-registry.example.com",
}
registry.RegisterCustomRegistry("my-registry", customConfig)
client.AddCredential("my-registry", "username", "token")
```

### 改进

#### 1. 线程安全
- 所有凭据管理操作使用 `sync.RWMutex` 保护
- 支持并发访问和修改

#### 2. 灵活性
- 可在运行时动态添加、更新、删除凭据
- 支持初始化时批量设置凭据
- 支持全局和客户端级别的 registry 注册

#### 3. 向后兼容性
- 保留了原有的 manifest 获取接口
- 批量获取功能继续正常工作
- 自动检测 registry 类型的逻辑不变

### 文档

新增文档：
- `MULTI_REGISTRY_GUIDE.md` - 多 registry 凭据管理完整指南
- `CHANGELOG.md` - 更新日志

更新示例：
- `example/multi_registry_demo.go` - 多 registry 凭据管理演示
- `example/batch_manifests.go` - 更新以使用新 API

### 测试

所有功能已测试：
- ✅ 基本凭据管理（添加、获取、删除、更新）
- ✅ 多 registry 凭据独立管理
- ✅ 自定义 registry 注册和使用
- ✅ 批量获取镜像（顺序和并发）
- ✅ 命令行参数解析
- ✅ 线程安全性

### 迁移指南

#### 从 v1.x 迁移到 v2.0

**步骤 1: 更新客户端创建**

```go
// 旧代码
client := registry.NewClient("username", "token")

// 新代码 - 单个 registry
client := registry.NewClient()
client.AddCredential(registry.DockerHubKey, "username", "token")

// 新代码 - 多个 registry
client := registry.NewClient()
client.AddCredential(registry.DockerHubKey, "user1", "token1")
client.AddCredential(registry.GHCRKey, "user2", "token2")
```

**步骤 2: 更新环境变量（可选）**

如果使用环境变量，建议更新为更明确的名称：

```bash
# 旧方式
export DOCKER_USERNAME="user"
export DOCKER_TOKEN="token"

# 新方式
export DOCKERHUB_USERNAME="user"
export DOCKERHUB_TOKEN="token"
export GHCR_USERNAME="ghuser"
export GHCR_TOKEN="ghp_token"
```

**步骤 3: 测试**

运行测试确保一切正常：
```bash
go test ./...
```

### 已知问题

无

### 致谢

感谢所有贡献者和用户的反馈！

