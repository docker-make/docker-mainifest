.PHONY: build clean run test install help

# 默认目标
.DEFAULT_GOAL := help

# 构建二进制文件
build: ## 构建 docker-auth 二进制文件
	@echo "正在构建..."
	@go build -o docker-auth ./cmd/docker-auth
	@echo "构建完成: ./docker-auth"

# 清理构建产物
clean: ## 清理构建产物
	@echo "正在清理..."
	@rm -f docker-auth
	@echo "清理完成"

# 运行示例
run: build ## 运行示例 (匿名访问 nginx)
	@echo "运行示例..."
	@./docker-auth -image nginx -tag latest -pretty

# 运行示例代码
example: ## 运行示例代码
	@echo "运行示例代码..."
	@go run ./example/main.go

# 安装到 GOPATH
install: ## 安装到 GOPATH/bin
	@echo "正在安装..."
	@go install ./cmd/docker-auth
	@echo "安装完成"

# 运行测试
test: ## 运行测试
	@echo "运行测试..."
	@go test ./...

# 格式化代码
fmt: ## 格式化代码
	@echo "格式化代码..."
	@go fmt ./...

# 检查代码
vet: ## 运行 go vet
	@echo "检查代码..."
	@go vet ./...

# 下载依赖
deps: ## 下载依赖
	@echo "下载依赖..."
	@go mod download
	@go mod tidy

# 帮助信息
help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

