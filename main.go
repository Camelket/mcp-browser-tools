package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/Camelket/mcp-browser-tools/internal/browser"
	"github.com/Camelket/mcp-browser-tools/internal/playwright_integration"
	"github.com/Camelket/mcp-browser-tools/internal/summary_tool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	browserManager := browser.NewBrowserInstanceManager(logger.With("component", "BrowserInstanceManager"))
	defer browserManager.CloseBrowserInstance()

	pwIntegration, err := playwright_integration.NewPlaywrightIntegration(browserManager, logger.With("component", "PlaywrightIntegration"))
	if err != nil {
		logger.Error("Failed to initialize PlaywrightIntegration", "error", err)
		os.Exit(1)
	}
	// No need to defer pwIntegration.Close() here, as browserManager handles the lifecycle.

	summaryTool := summary_tool.NewSummaryTool(pwIntegration, logger)

	// Create a new MCP server
	s := server.NewMCPServer(
		"web_tool_server",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add get_page_summary tool
	s.AddTool(mcp.NewTool("get_page_summary",
		mcp.WithDescription("Returns the HTML content and a base64 encoded screenshot of the current page."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL of the page to get summary from."),
		),
	), GetPageSummaryHandler(summaryTool))

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}

// GetPageSummaryHandler handles the get_page_summary MCP tool call.
func GetPageSummaryHandler(st *summary_tool.SummaryTool) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		url, err := request.RequireString("url")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid 'url' argument: %w", err)
		}

		pageSummary, err := st.CapturePageSummary(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("failed to capture page summary: %w", err)
		}

		encodedScreenshot := base64.StdEncoding.EncodeToString(pageSummary.Screenshot)

		// Use mcp.NewToolResultText or a similar function
		return mcp.NewToolResultText(fmt.Sprintf("URL: %s\nHTML: %s\nScreenshot: %s\nLinks: %v", pageSummary.URL, pageSummary.HTML, encodedScreenshot, pageSummary.Links)), nil
	}
}
