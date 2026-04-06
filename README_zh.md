# memos-mcp

> **[English Documentation](README.md)**

基于 [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) 的 [Memos](https://github.com/usememos/memos) 服务端，通过官方 HTTP REST API (v1) 与 Memos 通信，并以 **HTTP**（Streamable HTTP 传输）方式对外暴露 MCP 接口。

## 功能特性

| 工具 | 描述 |
|------|------|
| `search_memos` | 通过关键词或 CEL 过滤表达式搜索 Memo |
| `get_memo` | 获取单条 Memo 的完整内容 |
| `create_memo` | 创建新 Memo |
| `update_memo` | 修改内容、可见性或置顶状态 |
| `archive_memo` | 归档 Memo（从默认视图中隐藏） |
| `delete_memo` | 永久删除 Memo |
| `comment_memo` | 为 Memo 添加评论 |

## 环境要求

- 运行中的 [Memos](https://github.com/usememos/memos) 实例（v0.22+）
- Go 1.21+（仅源码构建时需要）

## 配置

通过环境变量进行配置：

| 变量 | 是否必填 | 描述 |
|------|----------|------|
| `MEMOS_SERVER_URL` | **必填** | Memos 实例的基础 URL，例如 `http://localhost:5230` |
| `MEMOS_AUTH_TOKEN` | 选填 | API 访问令牌，可在 **Memos → 设置 → 访问令牌** 中生成。若未设置，也可由 MCP 客户端在请求时传入（见[客户端令牌认证](#客户端令牌认证)）。 |
| `PORT` | 选填 | MCP HTTP 服务监听端口（默认 `8080`）。 |

## 安装

### Docker（推荐）

```bash
docker run -d \
  -e MEMOS_SERVER_URL=http://your-memos-instance:5230 \
  -e MEMOS_AUTH_TOKEN=your_access_token_here \
  -p 8080:8080 \
  ghcr.io/wolfsilver/memos-mcp:latest
```

### 从源码构建

```bash
git clone https://github.com/wolfsilver/memos-mcp.git
cd memos-mcp
go build -o memos-mcp .
```

### 使用 `go install`

```bash
go install github.com/wolfsilver/memos-mcp@latest
```

## 使用方法

### 配合 Claude Desktop

在 `claude_desktop_config.json` 中添加服务端配置：

```json
{
  "mcpServers": {
    "memos": {
      "type": "streamable-http",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

### 配合 VS Code（GitHub Copilot）

在 VS Code 的 `settings.json`（用户级或工作区级）中添加：

```json
{
  "mcp": {
    "servers": {
      "memos": {
        "type": "http",
        "url": "http://localhost:8080/mcp",
        "headers": {
          "Authorization": "Bearer your_access_token_here"
        }
      }
    }
  }
}
```

或在工作区根目录创建 `.vscode/mcp.json`：

```json
{
  "servers": {
    "memos": {
      "type": "http",
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer your_access_token_here"
      }
    }
  }
}
```

### 配合 OpenAI Codex / Codex CLI

在启动 Codex 会话时传入 MCP 服务地址，或在 `~/.codex/config.toml` 中配置：

```toml
[[mcp_servers]]
name = "memos"
type = "http"
url  = "http://localhost:8080/mcp"

[mcp_servers.headers]
Authorization = "Bearer your_access_token_here"
```

### 配合其他 MCP 客户端

直接运行可执行文件，服务端默认监听 `8080` 端口，MCP 接口路径为 `/mcp`：

```bash
export MEMOS_SERVER_URL=http://localhost:5230
export MEMOS_AUTH_TOKEN=your_access_token_here
export PORT=8080          # 可选，默认 8080
./memos-mcp
# MCP 接口地址：http://localhost:8080/mcp
```

## 客户端令牌认证

除了（或代替）在服务端设置 `MEMOS_AUTH_TOKEN` 之外，每个 MCP 客户端都可以在请求时通过标准 HTTP `Authorization` 头传入**自己的** Memos 访问令牌：

```
Authorization: Bearer <your_access_token>
```

请求级别的令牌优先于服务端 `MEMOS_AUTH_TOKEN` 环境变量。这使您可以运行**单个 memos-mcp 实例**，同时为多个用户提供服务，每个用户使用自己的令牌进行认证。

## 工具参考

### `search_memos` — 搜索 Memo

通过关键词或 [CEL](https://cel.dev/) 过滤表达式搜索 Memo。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `query` | string | 是 | 关键词或 CEL 表达式，如 `"golang"` 或 `"content.contains('golang')"` |
| `page_size` | number | 否 | 最多返回条数（默认：20） |

### `get_memo` — 获取单条 Memo

通过资源名称或 UID 获取单条 Memo。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `name` | string | 是 | 资源名称，如 `"memos/abc123"` 或仅 UID `"abc123"` |

### `create_memo` — 创建 Memo

创建一条新 Memo。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `content` | string | 是 | Markdown 内容 |
| `visibility` | string | 否 | `PRIVATE`（默认）、`PROTECTED` 或 `PUBLIC` |

### `update_memo` — 修改 Memo

修改已有 Memo，至少需要提供一个可选字段。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `name` | string | 是 | 资源名称，如 `"memos/abc123"` |
| `content` | string | 否 | 新的 Markdown 内容 |
| `visibility` | string | 否 | `PRIVATE`、`PROTECTED` 或 `PUBLIC` |
| `pinned` | boolean | 否 | 置顶或取消置顶 |

### `archive_memo` — 归档 Memo

将指定 Memo 归档，使其不再出现在默认视图中。可通过 `update_memo` 将 `state` 设为 `NORMAL` 来恢复。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `name` | string | 是 | 资源名称，如 `"memos/abc123"` |

### `delete_memo` — 删除 Memo

永久删除一条 Memo。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `name` | string | 是 | 资源名称，如 `"memos/abc123"` |

### `comment_memo` — 评论 Memo

为指定 Memo 添加评论。

| 参数 | 类型 | 是否必填 | 描述 |
|------|------|----------|------|
| `name` | string | 是 | 要评论的 Memo 资源名称 |
| `content` | string | 是 | 评论内容 |

## 开源许可

[MIT](LICENSE)
