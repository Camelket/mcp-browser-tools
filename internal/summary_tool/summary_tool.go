package summary_tool

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/Camelket/mcp-browser-tools/internal/playwright_integration"
	"golang.org/x/net/html"
)

// SummaryTool represents a tool for capturing page summaries using Playwright.
type SummaryTool struct {
	playwright *playwright_integration.PlaywrightIntegration
	logger     *slog.Logger
}

// PageSummary holds the captured URL, HTML content, screenshot data, extracted links, and network activity.
type PageSummary struct {
	URL             string
	HTML            string
	Screenshot      []byte
	Links           []string
	NetworkActivity []playwright_integration.CapturedNetworkActivity
}

// NewSummaryTool creates and returns a new SummaryTool instance.
func NewSummaryTool(pw *playwright_integration.PlaywrightIntegration, logger *slog.Logger) *SummaryTool {
	return &SummaryTool{
		playwright: pw,
		logger:     logger,
	}
}

// CapturePageSummary navigates to a URL, captures its HTML content, a full-page screenshot, and network activity.
func (st *SummaryTool) CapturePageSummary(ctx context.Context, url string) (*PageSummary, error) {
	st.logger.Info("Capturing page summary", "url", url)

	page, err := st.playwright.NewPage(ctx)
	if err != nil {
		st.logger.Error("Failed to create new page", "error", err)
		return nil, fmt.Errorf("failed to create new page: %w", err)
	}
	defer func() {
		if err := page.Close(); err != nil {
			st.logger.Error("Failed to close page", "error", err)
		}
	}()

	// Setup network interception before navigation
	if err := st.playwright.SetupNetworkInterception(ctx, page); err != nil {
		st.logger.Error("Failed to set up network interception", "error", err)
		return nil, fmt.Errorf("failed to set up network interception: %w", err)
	}

	// Use the PlaywrightIntegration's NavigateToURL function to leverage its timeout and logging.
	// Temporarily setting a 60-second timeout for debugging.
	page, err = st.playwright.NavigateToURL(ctx, url, nil, 60.0) // 60 seconds timeout
	if err != nil {
		st.logger.Error("Failed to navigate to URL", "url", url, "error", err)
		return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
	}

	htmlContent, err := page.Content()
	if err != nil {
		st.logger.Error("Failed to get HTML content", "url", url, "error", err)
		return nil, fmt.Errorf("failed to get HTML content for %s: %w", url, err)
	}

	screenshot, err := st.playwright.CaptureScreenshot(ctx, page, playwright_integration.PageScreenshotOptions{FullPage: true})
	if err != nil {
		st.logger.Error("Failed to capture screenshot", "url", url, "error", err)
		return nil, fmt.Errorf("failed to capture screenshot for %s: %w", url, err)
	}

	// Get captured network data
	networkActivity := st.playwright.GetCapturedNetworkData()
	st.logger.Info("Captured network activity", "count", len(networkActivity), "url", url)

	st.logger.Info("Successfully captured page summary", "url", url)

	links, err := st.extractLinks(htmlContent, url)
	if err != nil {
		st.logger.Error("Failed to extract links", "url", url, "error", err)
		// Continue even if link extraction fails, as it's not critical for the summary itself
	} else {
		st.logger.Info("Extracted links", "count", len(links), "url", url)
	}

	return &PageSummary{
		URL:             url,
		HTML:            htmlContent,
		Screenshot:      screenshot,
		Links:           links,
		NetworkActivity: networkActivity,
	}, nil
}

// extractLinks parses the HTML content and extracts all unique, absolute URLs from <a> tags.
func (st *SummaryTool) extractLinks(htmlContent string, baseURL string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var links []string
	visited := make(map[string]bool)
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL %s: %w", baseURL, err)
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					parsedURL, err := url.Parse(a.Val)
					if err != nil {
						st.logger.Error("Failed to parse link URL", "url", a.Val, "error", err)
						continue
					}
					resolvedURL := base.ResolveReference(parsedURL).String()
					if _, ok := visited[resolvedURL]; !ok {
						links = append(links, resolvedURL)
						visited[resolvedURL] = true
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return links, nil
}
