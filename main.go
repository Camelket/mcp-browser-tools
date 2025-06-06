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

	// Add get_html tool
	s.AddTool(mcp.NewTool("get_html",
		mcp.WithDescription("Returns the HTML content of the specified URL."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL of the page to get HTML from."),
		),
	), GetHTMLHandler(pwIntegration))

	// Add get_screenshot tool
	s.AddTool(mcp.NewTool("get_screenshot",
		mcp.WithDescription("Returns a base64 encoded screenshot of the specified URL."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL of the page to get a screenshot from."),
		),
		mcp.WithBoolean("full_page",
			mcp.Description("Whether to take a full page screenshot. Defaults to false."),
		),
	), GetScreenshotHandler(pwIntegration))

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

// GetHTMLHandler handles the get_html MCP tool call.
func GetHTMLHandler(pi *playwright_integration.PlaywrightIntegration) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		url, err := request.RequireString("url")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid 'url' argument: %w", err)
		}

		page, err := pi.NavigateToURL(ctx, url, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to navigate to URL: %w", err)
		}
		defer page.Close()

		htmlContent, err := page.Content()
		if err != nil {
			return nil, fmt.Errorf("failed to get HTML content: %w", err)
		}

		return mcp.NewToolResultText(htmlContent), nil
	}
}

// GetScreenshotHandler handles the get_screenshot MCP tool call.
func GetScreenshotHandler(pi *playwright_integration.PlaywrightIntegration) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		url, err := request.RequireString("url")
		if err != nil {
			return nil, fmt.Errorf("missing or invalid 'url' argument: %w", err)
		}

		fullPage := false // Default value
		fullPageStr, err := request.RequireString("full_page")
		if err == nil { // Parameter was provided
			if fullPageStr == "true" {
				fullPage = true
			}
		}

		page, err := pi.NavigateToURL(ctx, url, nil, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to navigate to URL: %w", err)
		}
		defer page.Close()

		screenshotBytes, err := pi.CaptureScreenshot(ctx, page, playwright_integration.PageScreenshotOptions{FullPage: fullPage})
		if err != nil {
			return nil, fmt.Errorf("failed to capture screenshot: %w", err)
		}

		encodedScreenshot := base64.StdEncoding.EncodeToString(screenshotBytes)

		return mcp.NewToolResultText(encodedScreenshot), nil
	}
}
