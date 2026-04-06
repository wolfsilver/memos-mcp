# memos-mcp

> **[English Documentation](README.md)**

基于 [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) 的 [Memos](https://github.com/usememos/memos) 服务端，通过官方 gRPC/Connect API 与 Memos 通信，并以 **HTTP**（Streamable HTTP 传输）方式对外暴露 MCP 接口。

## 功能特性

| 工具 | 描述 |
|------|------|
| `search_memos` | 通过关键词或 CEL 过滤表达式搜索 Memo |
| `get_memo` | 获取单条 Memo 的完整内容 |
| `create_memo` | 创建新 Memo |
| `update_memo` | 修改内容、可见性或置顶状态 |
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
| `MEMOS_AUTH_TOKEN` | 选填 | API 访问令牌，可在 **Memos → 设置 → 访问令牌** 中生成 |
| `PORT` | 选填 | MCP HTTP 服务监听端口（默认 `8080`）。 |

## 安装

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
      "url": "http://localhost:8080/mcp",
      "env": {
        "MEMOS_SERVER_URL": "http://localhost:5230",
        "MEMOS_AUTH_TOKEN": "your_access_token_here"
      }
    }
  }
}
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
