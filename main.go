package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	v1pb "github.com/usememos/memos/proto/gen/api/v1"
	"github.com/usememos/memos/proto/gen/api/v1/apiv1connect"
	fieldmaskpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

// authTransport injects a Bearer token into every outgoing HTTP request.
type authTransport struct {
	token     string
	transport http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.token != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.transport.RoundTrip(req)
}

// MemosClient wraps the connectrpc-generated MemoServiceClient.
type MemosClient struct {
	MemoService apiv1connect.MemoServiceClient
}

// NewMemosClient creates a MemosClient that talks to baseURL using the
// Connect protocol (HTTP/1.1 or HTTP/2, works with stock memos servers).
func NewMemosClient(baseURL, authToken string) *MemosClient {
	httpClient := &http.Client{
		Transport: &authTransport{
			token:     authToken,
			transport: http.DefaultTransport,
		},
	}
	memoSvc := apiv1connect.NewMemoServiceClient(httpClient, baseURL, connect.WithGRPCWeb())
	return &MemosClient{MemoService: memoSvc}
}

// memoToText formats a proto Memo as a human-readable string.
func memoToText(m *v1pb.Memo) string {
	createTime := ""
	if m.GetCreateTime() != nil {
		createTime = m.GetCreateTime().AsTime().Format("2006-01-02T15:04:05Z")
	}
	updateTime := ""
	if m.GetUpdateTime() != nil {
		updateTime = m.GetUpdateTime().AsTime().Format("2006-01-02T15:04:05Z")
	}
	return fmt.Sprintf("Name: %s\nVisibility: %s\nPinned: %v\nCreated: %s\nUpdated: %s\nContent:\n%s",
		m.GetName(),
		m.GetVisibility().String(),
		m.GetPinned(),
		createTime,
		updateTime,
		m.GetContent(),
	)
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

	client := NewMemosClient(serverURL, authToken)

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

		// Build a CEL filter: wrap plain keywords in content.contains().
		filter := query
		isCEL := strings.Contains(query, ".contains(") ||
			strings.Contains(query, " == ") ||
			strings.Contains(query, " != ") ||
			strings.Contains(query, " && ") ||
			strings.Contains(query, " || ")
		if !isCEL {
			filter = fmt.Sprintf("content.contains(%q)", query)
		}

		resp, err := client.MemoService.ListMemos(ctx, connect.NewRequest(&v1pb.ListMemosRequest{
			Filter:   filter,
			PageSize: int32(pageSize),
		}))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %s", err)), nil
		}

		memos := resp.Msg.GetMemos()
		if len(memos) == 0 {
			return mcp.NewToolResultText("No memos found matching the query."), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d memo(s):\n\n", len(memos)))
		for i, m := range memos {
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
		resp, err := client.MemoService.GetMemo(ctx, connect.NewRequest(&v1pb.GetMemoRequest{Name: name}))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(memoToText(resp.Msg)), nil
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
		visStr := req.GetString("visibility", "PRIVATE")

		vis := v1pb.Visibility(v1pb.Visibility_value["VISIBILITY_"+visStr])
		if vis == v1pb.Visibility_VISIBILITY_UNSPECIFIED {
			vis = v1pb.Visibility_PRIVATE
		}

		resp, err := client.MemoService.CreateMemo(ctx, connect.NewRequest(&v1pb.CreateMemoRequest{
			Memo: &v1pb.Memo{
				Content:    content,
				Visibility: vis,
			},
		}))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memo created successfully.\n\n%s", memoToText(resp.Msg))), nil
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
		memo := &v1pb.Memo{Name: name}
		paths := []string{}

		if content, ok := args["content"].(string); ok && content != "" {
			memo.Content = content
			paths = append(paths, "content")
		}
		if visStr, ok := args["visibility"].(string); ok && visStr != "" {
			vis := v1pb.Visibility(v1pb.Visibility_value["VISIBILITY_"+visStr])
			if vis == v1pb.Visibility_VISIBILITY_UNSPECIFIED {
				vis = v1pb.Visibility_PRIVATE
			}
			memo.Visibility = vis
			paths = append(paths, "visibility")
		}
		if _, exists := args["pinned"]; exists {
			memo.Pinned = req.GetBool("pinned", false)
			paths = append(paths, "pinned")
		}

		if len(paths) == 0 {
			return mcp.NewToolResultError("at least one field (content, visibility, or pinned) must be provided"), nil
		}

		resp, err := client.MemoService.UpdateMemo(ctx, connect.NewRequest(&v1pb.UpdateMemoRequest{
			Memo:       memo,
			UpdateMask: &fieldmaskpb.FieldMask{Paths: paths},
		}))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("update memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Memo updated successfully.\n\n%s", memoToText(resp.Msg))), nil
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
		_, err := client.MemoService.DeleteMemo(ctx, connect.NewRequest(&v1pb.DeleteMemoRequest{Name: name}))
		if err != nil {
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
		resp, err := client.MemoService.CreateMemoComment(ctx, connect.NewRequest(&v1pb.CreateMemoCommentRequest{
			Name:    name,
			Comment: &v1pb.Memo{Content: content},
		}))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("comment memo failed: %s", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Comment added successfully.\n\n%s", memoToText(resp.Msg))), nil
	})

	httpServer := server.NewStreamableHTTPServer(s, server.WithStateLess(true))
	addr := ":" + port
	log.Printf("memos-mcp HTTP server listening on %s", addr)
	if err := httpServer.Start(addr); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}

