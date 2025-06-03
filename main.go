package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"web_tool_server",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add get_screenshot tool
	s.AddTool(mcp.NewTool("get_screenshot",
		mcp.WithDescription("Returns a base64 encoded screenshot of the current page."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL of the page to screenshot."),
		),
	), getScreenshotHandler)

	// Add get_html tool
	s.AddTool(mcp.NewTool("get_html",
		mcp.WithDescription("Returns the HTML content of the current page."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL of the page to get HTML from."),
		),
	), getHTMLHandler)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// getScreenshotHandler reads the screenshot.png.b64 file and returns its content
func getScreenshotHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := request.RequireString("url")
	if err != nil {
		return nil, fmt.Errorf("missing or invalid 'url' argument: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}
	defer func() {
		if err := pw.Stop(); err != nil {
			log.Printf("could not stop playwright: %v", err)
		}
	}()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			log.Printf("could not close browser: %v", err)
		}
	}()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000), // 30 seconds timeout
	}); err != nil {
		return nil, fmt.Errorf("could not goto URL: %w", err)
	}

	// Add a small delay to ensure all dynamic content is loaded
	time.Sleep(2 * time.Second)

	screenshotBytes, err := page.Screenshot(playwright.PageScreenshotOptions{
		FullPage: playwright.Bool(true),
		Type:     playwright.ScreenshotTypePng,
	})
	if err != nil {
		return nil, fmt.Errorf("could not take screenshot: %w", err)
	}

	encodedScreenshot := base64.StdEncoding.EncodeToString(screenshotBytes)
	return mcp.NewToolResultText(encodedScreenshot), nil
}

// getHTMLHandler gets the HTML content of a page using Playwright
func getHTMLHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := request.RequireString("url")
	if err != nil {
		return nil, fmt.Errorf("missing or invalid 'url' argument: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}
	defer func() {
		if err := pw.Stop(); err != nil {
			log.Printf("could not stop playwright: %v", err)
		}
	}()

	browser, err := pw.Chromium.Launch()
	if err != nil {
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}
	defer func() {
		if err := browser.Close(); err != nil {
			log.Printf("could not close browser: %v", err)
		}
	}()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	if _, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000), // 30 seconds timeout
	}); err != nil {
		return nil, fmt.Errorf("could not goto URL: %w", err)
	}

	// Add a small delay to ensure all dynamic content is loaded
	time.Sleep(2 * time.Second)

	htmlContent, err := page.Content()
	if err != nil {
		return nil, fmt.Errorf("could not get HTML content: %w", err)
	}

	return mcp.NewToolResultText(htmlContent), nil
}
