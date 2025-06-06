package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"

	"github.com/Camelket/mcp-browser-tools/internal/playwright_integration"
	"github.com/Camelket/mcp-browser-tools/internal/summary_tool"
)

var (
	pw      *playwright.Playwright
	browser playwright.Browser
	logger  *slog.Logger
)

func TestMain(m *testing.M) {
	var err error
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Ensure Playwright browsers are installed
	// This is typically done once during setup, but good to have for E2E tests
	err = playwright.Install()
	if err != nil {
		logger.Error("Failed to install Playwright browsers", "error", err)
		os.Exit(1)
	}

	pw, err = PlaywrightRunFunc()
	if err != nil {
		logger.Error("Could not start Playwright", "error", err)
		os.Exit(1)
	}

	browser, err = pw.Chromium.Launch()
	if err != nil {
		logger.Error("Could not launch Chromium", "error", err)
		os.Exit(1)
	}

	code := m.Run()

	if err := browser.Close(); err != nil {
		logger.Error("Could not close browser", "error", err)
	}
	if err := pw.Stop(); err != nil {
		logger.Error("Could not stop Playwright", "error", err)
	}

	os.Exit(code)
}

// setupTestServer creates a new HTTP test server with the given HTML content.
func setupTestServer(t *testing.T, htmlContent string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, htmlContent)
	}))
	t.Cleanup(func() {
		ts.Close()
	})
	return ts
}

func TestCapturePageSummary_Basic(t *testing.T) {
	htmlContent := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Test Page</title>
		</head>
		<body>
			<h1>Hello, Playwright!</h1>
			<a href="/link1">Link 1</a>
			<a href="https://example.com/link2">Link 2</a>
			<a href="link3.html">Link 3</a>
		</body>
		</html>
	`
	ts := setupTestServer(t, htmlContent)
	testURL := ts.URL

	pwIntegration, err := playwright_integration.NewPlaywrightIntegration(pw, logger)
	assert.NoError(t, err)
	defer pwIntegration.Close()

	st := summary_tool.NewSummaryTool(pwIntegration, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pageSummary, err := st.CapturePageSummary(ctx, testURL)
	assert.NoError(t, err)
	assert.NotNil(t, pageSummary)

	assert.Equal(t, testURL, pageSummary.URL)
	assert.True(t, strings.Contains(pageSummary.HTML, "<h1>Hello, Playwright!</h1>"))
	assert.NotEmpty(t, pageSummary.Screenshot)

	expectedLinks := []string{
		testURL + "/link1",
		"https://example.com/link2",
		testURL + "/link3.html",
	}
	assert.ElementsMatch(t, expectedLinks, pageSummary.Links)
}
