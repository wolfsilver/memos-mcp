package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Config holds the server configuration.
type Config struct {
	ServerURL string
	AuthToken string
}

// MemosClient is an HTTP client for the memos REST API.
type MemosClient struct {
	config     Config
	httpClient *http.Client
}

// Memo represents a memos object returned by the API.
type Memo struct {
	Name       string   `json:"name"`
	UID        string   `json:"uid"`
	Content    string   `json:"content"`
	Visibility string   `json:"visibility"`
	CreateTime string   `json:"createTime"`
	UpdateTime string   `json:"updateTime"`
	Pinned     bool     `json:"pinned"`
	Tags       []string `json:"tags"`
}

// ListMemosResponse is the response from the list/search memos API.
type ListMemosResponse struct {
	Memos         []Memo `json:"memos"`
	NextPageToken string `json:"nextPageToken"`
}

// CreateMemoRequest is the body for creating a memo.
type CreateMemoRequest struct {
	Content    string `json:"content"`
	Visibility string `json:"visibility,omitempty"`
}

// UpdateMemoRequest is the body for updating a memo.
type UpdateMemoRequest struct {
	Content    string `json:"content,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	Pinned     *bool  `json:"pinned,omitempty"`
	State      string `json:"state,omitempty"`
}

// contextKey is an unexported type for context keys in this package.
type contextKey int

const (
	// authTokenKey is the context key for a per-request auth token.
	authTokenKey contextKey = iota
)

// Memo state constants used with UpdateMemo.
const (
	MemoStateNormal   = "NORMAL"
	MemoStateArchived = "ARCHIVED"
)

// CreateCommentRequest is the body for creating a comment on a memo.
type CreateCommentRequest struct {
	Content string `json:"content"`
}

// NewMemosClient creates a new MemosClient from the given config.
func NewMemosClient(cfg Config) *MemosClient {
	return &MemosClient{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

func (c *MemosClient) doRequest(ctx context.Context, method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	fullURL := strings.TrimRight(c.config.ServerURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Per-request token (from MCP client Authorization header) takes precedence
	// over the server-level token configured via MEMOS_AUTH_TOKEN.
	token := c.config.AuthToken
	if t, ok := ctx.Value(authTokenKey).(string); ok && t != "" {
		token = t
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

// ListMemos fetches memos with an optional filter and page size.
func (c *MemosClient) ListMemos(ctx context.Context, filter string, pageSize int) (*ListMemosResponse, error) {
	q := url.Values{}
	if filter != "" {
		q.Set("filter", filter)
	}
	if pageSize > 0 {
		q.Set("pageSize", fmt.Sprintf("%d", pageSize))
	}
	path := "/api/v1/memos"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	data, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var result ListMemosResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse list memos response: %w", err)
	}
	return &result, nil
}

// GetMemo fetches a single memo by its name (e.g. "memos/abc123").
func (c *MemosClient) GetMemo(ctx context.Context, name string) (*Memo, error) {
	data, err := c.doRequest(ctx, http.MethodGet, "/api/v1/"+name, nil)
	if err != nil {
		return nil, err
	}
	var memo Memo
	if err := json.Unmarshal(data, &memo); err != nil {
		return nil, fmt.Errorf("parse memo response: %w", err)
	}
	return &memo, nil
}

// CreateMemo creates a new memo with the given content and visibility.
func (c *MemosClient) CreateMemo(ctx context.Context, content, visibility string) (*Memo, error) {
	reqBody := CreateMemoRequest{Content: content, Visibility: visibility}
	data, err := c.doRequest(ctx, http.MethodPost, "/api/v1/memos", reqBody)
	if err != nil {
		return nil, err
	}
	var memo Memo
	if err := json.Unmarshal(data, &memo); err != nil {
		return nil, fmt.Errorf("parse create memo response: %w", err)
	}
	return &memo, nil
}

// UpdateMemo updates an existing memo identified by name.
func (c *MemosClient) UpdateMemo(ctx context.Context, name string, req UpdateMemoRequest, updateMask []string) (*Memo, error) {
	q := url.Values{}
	if len(updateMask) > 0 {
		q.Set("updateMask", strings.Join(updateMask, ","))
	}
	path := "/api/v1/" + name
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	data, err := c.doRequest(ctx, http.MethodPatch, path, req)
	if err != nil {
		return nil, err
	}
	var memo Memo
	if err := json.Unmarshal(data, &memo); err != nil {
		return nil, fmt.Errorf("parse update memo response: %w", err)
	}
	return &memo, nil
}

// DeleteMemo deletes a memo by its name.
func (c *MemosClient) DeleteMemo(ctx context.Context, name string) error {
	_, err := c.doRequest(ctx, http.MethodDelete, "/api/v1/"+name, nil)
	return err
}

// CreateComment adds a comment to a memo.
func (c *MemosClient) CreateComment(ctx context.Context, memoName, content string) (*Memo, error) {
	reqBody := CreateCommentRequest{Content: content}
	data, err := c.doRequest(ctx, http.MethodPost, "/api/v1/"+memoName+"/comments", reqBody)
	if err != nil {
		return nil, err
	}
	var comment Memo
	if err := json.Unmarshal(data, &comment); err != nil {
		return nil, fmt.Errorf("parse comment response: %w", err)
	}
	return &comment, nil
}

// memoToText formats a Memo as a human-readable string.
func memoToText(m Memo) string {
	return fmt.Sprintf("Name: %s\nUID: %s\nVisibility: %s\nPinned: %v\nCreated: %s\nUpdated: %s\nContent:\n%s",
		m.Name, m.UID, m.Visibility, m.Pinned, m.CreateTime, m.UpdateTime, m.Content)
}

func main() {
	serverURL := os.Getenv("MEMOS_SERVER_URL")
	if serverURL == "" {
		log.Fatal("MEMOS_SERVER_URL environment variable is required")
	}
	// Normalise: add http:// scheme if not present.
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "http://" + serverURL
	}

	authToken := os.Getenv("MEMOS_AUTH_TOKEN")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cfg := Config{
		ServerURL: serverURL,
		AuthToken: authToken,
	}
	client := NewMemosClient(cfg)

	s := server.NewMCPServer(
		"memos-mcp",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// ── search_memos ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("search_memos",
		mcp.WithDescription("Search memos by keyword or filter expression. Returns a list of matching memos."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Keyword or CEL filter expression to search memos. Example: \"hello\" or \"content.contains('hello')\"")),
		mcp.WithNumber("page_size",
			mcp.Description("Maximum number of results to return (default 20)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := req.GetString("query", "")
		pageSize := req.GetInt("page_size", 20)

		// Build a filter: treat as a raw CEL expression only when it contains
		// known CEL operators/functions; otherwise wrap in content.contains().
		filter := query
		isCEL := strings.Contains(query, ".contains(") ||
			strings.Contains(query, " == ") ||
			strings.Contains(query, " != ") ||
			strings.Contains(query, " && ") ||
			strings.Contains(query, " || ")
		if !isCEL {
			filter = fmt.Sprintf("content.contains(%q)", query)
		}

		result, err := client.ListMemos(ctx, filter, pageSize)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %s", err)), nil
		}

		if len(result.Memos) == 0 {
			return mcp.NewToolResultText("No memos found matching the query."), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d memo(s):\n\n", len(result.Memos)))
		for i, m := range result.Memos {
			sb.WriteString(fmt.Sprintf("--- Memo %d ---\n%s\n\n", i+1, memoToText(m)))
		}
		return mcp.NewToolResultText(sb.String()), nil
	})

	// ── get_memo ──────────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("get_memo",
		mcp.WithDescription("Retrieve a single memo by its resource name or UID."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Resource name of the memo, e.g. \"memos/abc123\" or just the UID \"abc123\"")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		if !strings.HasPrefix(name, "memos/") {
			name = "memos/" + name
		}
		memo, err := client.GetMemo(ctx, name)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(memoToText(*memo)), nil
	})

	// ── create_memo ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("create_memo",
		mcp.WithDescription("Create a new memo with the given content."),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Markdown content of the new memo")),
		mcp.WithString("visibility",
			mcp.Description("Visibility of the memo: PRIVATE, PROTECTED, or PUBLIC (default: PRIVATE)")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		content := req.GetString("content", "")
		visibility := req.GetString("visibility", "PRIVATE")
		memo, err := client.CreateMemo(ctx, content, visibility)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memo created successfully.\n\n%s", memoToText(*memo))), nil
	})

	// ── update_memo ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("update_memo",
		mcp.WithDescription("Update the content, visibility, or pinned state of an existing memo."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Resource name of the memo, e.g. \"memos/abc123\"")),
		mcp.WithString("content",
			mcp.Description("New markdown content for the memo")),
		mcp.WithString("visibility",
			mcp.Description("New visibility: PRIVATE, PROTECTED, or PUBLIC")),
		mcp.WithBoolean("pinned",
			mcp.Description("Whether the memo should be pinned")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		if !strings.HasPrefix(name, "memos/") {
			name = "memos/" + name
		}

		args := req.GetArguments()
		updateReq := UpdateMemoRequest{}
		updateMask := []string{}

		if content, ok := args["content"].(string); ok && content != "" {
			updateReq.Content = content
			updateMask = append(updateMask, "content")
		}
		if visibility, ok := args["visibility"].(string); ok && visibility != "" {
			updateReq.Visibility = visibility
			updateMask = append(updateMask, "visibility")
		}
		if _, exists := args["pinned"]; exists {
			pinned := req.GetBool("pinned", false)
			updateReq.Pinned = &pinned
			updateMask = append(updateMask, "pinned")
		}

		if len(updateMask) == 0 {
			return mcp.NewToolResultError("at least one field (content, visibility, or pinned) must be provided"), nil
		}

		memo, err := client.UpdateMemo(ctx, name, updateReq, updateMask)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memo updated successfully.\n\n%s", memoToText(*memo))), nil
	})

	// ── delete_memo ───────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("delete_memo",
		mcp.WithDescription("Permanently delete a memo by its resource name."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Resource name of the memo, e.g. \"memos/abc123\"")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		if !strings.HasPrefix(name, "memos/") {
			name = "memos/" + name
		}
		if err := client.DeleteMemo(ctx, name); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("delete memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memo \"%s\" deleted successfully.", name)), nil
	})

	// ── comment_memo ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("comment_memo",
		mcp.WithDescription("Add a comment to an existing memo."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Resource name of the memo to comment on, e.g. \"memos/abc123\"")),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Content of the comment")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		content := req.GetString("content", "")
		if !strings.HasPrefix(name, "memos/") {
			name = "memos/" + name
		}
		comment, err := client.CreateComment(ctx, name, content)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("comment memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Comment added successfully.\n\n%s", memoToText(*comment))), nil
	})

	// ── archive_memo ──────────────────────────────────────────────────────────
	s.AddTool(mcp.NewTool("archive_memo",
		mcp.WithDescription("Archive a memo so it no longer appears in the default view."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Resource name of the memo, e.g. \"memos/abc123\"")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := req.GetString("name", "")
		if !strings.HasPrefix(name, "memos/") {
			name = "memos/" + name
		}
		archiveReq := UpdateMemoRequest{State: MemoStateArchived}
		memo, err := client.UpdateMemo(ctx, name, archiveReq, []string{"state"})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("archive memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memo archived successfully.\n\n%s", memoToText(*memo))), nil
	})

	httpServer := server.NewStreamableHTTPServer(s,
		server.WithStateLess(true),
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			if auth := r.Header.Get("Authorization"); auth != "" {
				const prefix = "Bearer "
				if strings.HasPrefix(auth, prefix) {
					ctx = context.WithValue(ctx, authTokenKey, strings.TrimPrefix(auth, prefix))
				}
			}
			return ctx
		}),
	)
	addr := ":" + port
	log.Printf("memos-mcp HTTP server listening on %s", addr)
	if err := httpServer.Start(addr); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

