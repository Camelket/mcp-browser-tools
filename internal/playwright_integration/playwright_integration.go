package playwright_integration

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Camelket/mcp-browser-tools/internal/browser"
	"github.com/playwright-community/playwright-go"
)

// PlaywrightRunFunc defines the type for the playwright.Run function.
type PlaywrightRunFunc func(options ...*playwright.RunOptions) (*playwright.Playwright, error)

// PlaywrightIntegration provides a high-level interface for Playwright interactions.
type PlaywrightIntegration struct {
	browserManager      *browser.BrowserInstanceManager
	logger              *slog.Logger
	capturedNetworkData []CapturedNetworkActivity
	pendingRequests     map[string]CapturedRequest // Map to store requests by URL until response is received
}

// PageScreenshotOptions provides options for capturing a screenshot.
type PageScreenshotOptions struct {
	FullPage bool
}

// CapturedRequest holds details of an intercepted network request.
type CapturedRequest struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// CapturedResponse holds details of an intercepted network response.
type CapturedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body,omitempty"`
}

// CapturedNetworkActivity holds details of a full request-response cycle.
type CapturedNetworkActivity struct {
	Request  CapturedRequest  `json:"request"`
	Response CapturedResponse `json:"response"`
}

// NewPlaywrightIntegration creates a new PlaywrightIntegration instance.
func NewPlaywrightIntegration(browserManager *browser.BrowserInstanceManager, logger *slog.Logger) (*PlaywrightIntegration, error) {
	if browserManager == nil {
		return nil, fmt.Errorf("browser instance manager cannot be nil")
	}
	return &PlaywrightIntegration{
		browserManager:      browserManager,
		logger:              logger,
		capturedNetworkData: []CapturedNetworkActivity{},
		pendingRequests:     make(map[string]CapturedRequest),
	}, nil
}

// Close stops the Playwright instance.
func (pi *PlaywrightIntegration) Close() {
	// The browser instance is managed by BrowserInstanceManager, so we don't stop Playwright here.
	// We just ensure any pending requests are cleared.
	pi.pendingRequests = make(map[string]CapturedRequest)
}

// NewPage creates a new browser page using the managed browser instance.
func (pi *PlaywrightIntegration) NewPage(ctx context.Context) (playwright.Page, error) {
	browser, err := pi.browserManager.GetBrowserInstance(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get browser instance: %w", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("could not create page: %w", err)
	}

	// Use a goroutine to close the page if the parent context is cancelled
	go func() {
		select {
		case <-ctx.Done():
			pi.logger.Debug("Context cancelled. Closing page.")
			if page != nil {
				page.Close()
			}
		default:
			// Do nothing if context is not done
		}
	}()

	return page, nil
}

// NavigateToURL navigates to a given URL with configurable options.
func (pi *PlaywrightIntegration) NavigateToURL(ctx context.Context, url string, options *playwright.PageGotoOptions) (playwright.Page, error) {
	pi.logger.Info("Navigating to URL", "url", url)

	page, err := pi.NewPage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new page: %w", err)
	}

	// Dereference options if it's not nil
	if options != nil {
		if _, err = page.Goto(url, *options); err != nil {
			page.Close() // Close page if navigation fails
			return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
		}
	} else {
		if _, err = page.Goto(url); err != nil {
			page.Close() // Close page if navigation fails
			return nil, fmt.Errorf("failed to navigate to %s: %w", url, err)
		}
	}

	pi.logger.Info("Successfully navigated to URL", "url", url)
	return page, nil
}

// ExecuteScript executes JavaScript code on a given playwright.Page and returns the result.
func (pi *PlaywrightIntegration) ExecuteScript(ctx context.Context, page playwright.Page, script string, args ...interface{}) (interface{}, error) {
	if page == nil {
		return nil, fmt.Errorf("playwright.Page cannot be nil")
	}
	pi.logger.Debug("Executing script on page.")

	result, err := page.Evaluate(script, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute script: %w", err)
	}
	pi.logger.Debug("Script executed successfully.")
	return result, nil
}

// CaptureScreenshot captures a screenshot of a given playwright.Page.
func (pi *PlaywrightIntegration) CaptureScreenshot(ctx context.Context, page playwright.Page, options PageScreenshotOptions) ([]byte, error) {
	if page == nil {
		return nil, fmt.Errorf("playwright.Page cannot be nil")
	}
	pi.logger.Debug("Capturing screenshot.")

	screenshot, err := page.Screenshot(playwright.PageScreenshotOptions{FullPage: playwright.Bool(options.FullPage)})
	if err != nil {
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}
	pi.logger.Debug("Screenshot captured successfully.")
	return screenshot, nil
}

// SetupNetworkInterception sets up network interception on a given playwright.Page.
func (pi *PlaywrightIntegration) SetupNetworkInterception(ctx context.Context, page playwright.Page) error {
	if page == nil {
		return fmt.Errorf("playwright.Page cannot be nil")
	}

	pi.logger.Debug("Setting up network interception.")

	// Clear previous network data for a new navigation
	pi.capturedNetworkData = []CapturedNetworkActivity{}
	pi.pendingRequests = make(map[string]CapturedRequest) // Clear pending requests for a new navigation

	// Set up request interception
	err := page.Route("**/*", func(route playwright.Route) {
		request := route.Request()

		// Capture request details
		reqHeaders := make(map[string]string)
		headersArray, err := request.HeadersArray()
		if err != nil {
			pi.logger.Warn("Failed to get request headers array", "error", err)
		} else {
			for _, header := range headersArray {
				reqHeaders[header.Name] = header.Value
			}
		}
		capturedReq := CapturedRequest{
			URL:     request.URL(),
			Method:  request.Method(),
			Headers: reqHeaders,
		}

		// Capture request body for POST requests
		if request.Method() == "POST" {
			postData, err := request.PostData()
			if err != nil {
				pi.logger.Warn("Failed to get request post data", "error", err)
			} else if postData != "" {
				capturedReq.Body = postData
			}
		}

		// Store the request in pendingRequests map
		pi.pendingRequests[request.URL()] = capturedReq

		// Continue the request
		route.Continue()
	})
	if err != nil {
		return fmt.Errorf("failed to set up request interception: %w", err)
	}

	// Set up response interception
	page.On("response", func(response playwright.Response) {
		// Retrieve the corresponding request from pendingRequests
		reqURL := response.Request().URL()
		capturedReq, ok := pi.pendingRequests[reqURL]
		if !ok {
			pi.logger.Debug("No matching pending request found for response", "url", reqURL)
			return
		}

		// Capture response details
		respHeaders := make(map[string]string)
		headersArray, err := response.HeadersArray()
		if err != nil {
			pi.logger.Warn("Failed to get response headers array", "error", err)
		} else {
			for _, header := range headersArray {
				respHeaders[header.Name] = header.Value
			}
		}
		capturedResp := CapturedResponse{
			Status:  response.Status(),
			Headers: respHeaders,
		}

		// Capture response body
		body, err := response.Body()
		if err != nil {
			pi.logger.Warn("Failed to get response body", "error", err)
		} else {
			capturedResp.Body = string(body)
		}

		// Store the captured activity
		pi.capturedNetworkData = append(pi.capturedNetworkData, CapturedNetworkActivity{
			Request:  capturedReq,
			Response: capturedResp,
		})

		// Remove from pending requests
		delete(pi.pendingRequests, reqURL)
	})

	pi.logger.Debug("Network interception set up successfully.")
	return nil
}

// GetCapturedNetworkData returns the captured network activity.
func (pi *PlaywrightIntegration) GetCapturedNetworkData() []CapturedNetworkActivity {
	return pi.capturedNetworkData
}
