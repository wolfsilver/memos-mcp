# memos-mcp

> **[中文文档](README_zh.md)**

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for [Memos](https://github.com/usememos/memos) — the open-source, self-hosted note-taking service. It communicates with Memos via the official HTTP REST API (v1) and exposes the MCP endpoint over **HTTP** (Streamable HTTP transport).

## Features

| Tool | Description |
|------|-------------|
| `search_memos` | Search memos by keyword or CEL filter expression |
| `get_memo` | Retrieve the full content of a single memo |
| `create_memo` | Create a new memo |
| `update_memo` | Update content, visibility, or pinned state |
| `delete_memo` | Permanently delete a memo |
| `comment_memo` | Add a comment to a memo |

## Requirements

- A running [Memos](https://github.com/usememos/memos) instance (v0.22+)
- Go 1.21+ (only needed to build from source)

## Configuration

The server is configured via environment variables:

| Variable | Required | Description |
|----------|----------|-------------|
| `MEMOS_SERVER_URL` | **Yes** | Base URL of your Memos instance, e.g. `http://localhost:5230` |
| `MEMOS_AUTH_TOKEN` | No | API access token for authentication. Generate one in **Memos → Settings → Access Tokens**. |
| `PORT` | No | Port the MCP HTTP server listens on (default: `8080`). |

## Installation

### Docker (recommended)

```bash
docker run -d \
  -e MEMOS_SERVER_URL=http://your-memos-instance:5230 \
  -e MEMOS_AUTH_TOKEN=your_access_token_here \
  -p 8080:8080 \
  ghcr.io/wolfsilver/memos-mcp:latest
```

### Build from source

```bash
git clone https://github.com/wolfsilver/memos-mcp.git
cd memos-mcp
go build -o memos-mcp .
```

### Using `go install`

```bash
go install github.com/wolfsilver/memos-mcp@latest
```

## Usage

### With Claude Desktop

Add the server to your `claude_desktop_config.json`:

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

### With other MCP clients

Run the server binary directly. It listens on HTTP (default port `8080`) and exposes the MCP endpoint at `/mcp`:

```bash
export MEMOS_SERVER_URL=http://localhost:5230
export MEMOS_AUTH_TOKEN=your_access_token_here
export PORT=8080          # optional, defaults to 8080
./memos-mcp
# MCP endpoint: http://localhost:8080/mcp
```

## Tool Reference

### `search_memos`

Search memos by keyword or a [CEL](https://cel.dev/) filter expression.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Keyword or CEL expression, e.g. `"golang"` or `"content.contains('golang')"` |
| `page_size` | number | No | Max results to return (default: 20) |

### `get_memo`

Retrieve a single memo by its resource name or UID.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Resource name, e.g. `"memos/abc123"` or just the UID `"abc123"` |

### `create_memo`

Create a new memo.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `content` | string | Yes | Markdown content |
| `visibility` | string | No | `PRIVATE` (default), `PROTECTED`, or `PUBLIC` |

### `update_memo`

Update an existing memo. At least one optional field is required.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Resource name, e.g. `"memos/abc123"` |
| `content` | string | No | New markdown content |
| `visibility` | string | No | `PRIVATE`, `PROTECTED`, or `PUBLIC` |
| `pinned` | boolean | No | Pin or unpin the memo |

### `delete_memo`

Permanently delete a memo.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Resource name, e.g. `"memos/abc123"` |

### `comment_memo`

Add a comment to a memo.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Resource name of the memo to comment on |
| `content` | string | Yes | Comment text |

## License

[MIT](LICENSE)