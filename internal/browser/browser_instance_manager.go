package browser

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// BrowserInstanceManager manages a single, persistent Playwright browser instance.
type BrowserInstanceManager struct {
	browser           playwright.Browser
	mu                sync.Mutex
	logger            *slog.Logger
	inactivityTimer   *time.Timer
	inactivityTimeout time.Duration // Configurable inactivity timeout
	cancelTimeout     context.CancelFunc
}

// NewBrowserInstanceManager creates and returns a new BrowserInstanceManager.
func NewBrowserInstanceManager(logger *slog.Logger) *BrowserInstanceManager {
	return &BrowserInstanceManager{
		logger:            logger,
		inactivityTimeout: 1 * time.Minute, // Default to 1 minute
	}
}

// SetInactivityTimeout sets the inactivity timeout duration.
func (bim *BrowserInstanceManager) SetInactivityTimeout(timeout time.Duration) {
	bim.mu.Lock()
	defer bim.mu.Unlock()
	bim.inactivityTimeout = timeout
	bim.logger.Debug("Inactivity timeout set", slog.Duration("timeout", timeout))
}

// GetBrowserInstance returns the single, persistent Playwright browser instance.
// If the instance does not exist or is closed, it launches a new one.
// This method is thread-safe.
func (bim *BrowserInstanceManager) GetBrowserInstance(ctx context.Context) (playwright.Browser, error) {
	bim.logger.Debug("GetBrowserInstance called.")
	bim.mu.Lock()
	defer bim.mu.Unlock()

	// Check if the browser instance is valid and not closed.
	if bim.browser != nil {
		bim.logger.Debug("Returning existing browser instance.")
		bim.ResetInactivityTimer() // Reset timer on use
		return bim.browser, nil
	}

	bim.logger.Info("Launching new browser instance.")
	bim.logger.Debug("Calling playwright.Run()...")
	pw, err := playwright.Run()
	if err != nil {
		bim.logger.Error("Failed to launch Playwright", slog.Any("error", err))
		return nil, err
	}
	bim.logger.Debug("playwright.Run() successful. Launching Chromium...")

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if err != nil {
		bim.logger.Error("Failed to launch browser", slog.Any("error", err))
		return nil, err
	}
	bim.logger.Debug("Chromium launched.")

	bim.browser = browser
	bim.logger.Info("Browser instance launched successfully.")
	bim.ResetInactivityTimer() // Start timer after launch
	return bim.browser, nil
}

// CloseBrowserInstance closes the Playwright browser instance if it's open.
func (bim *BrowserInstanceManager) CloseBrowserInstance() error {
	bim.mu.Lock()
	defer bim.mu.Unlock()

	if bim.inactivityTimer != nil {
		bim.inactivityTimer.Stop()
		bim.inactivityTimer = nil
	}
	if bim.cancelTimeout != nil {
		bim.cancelTimeout()
		bim.cancelTimeout = nil
	}

	if bim.browser != nil {
		bim.logger.Info("Closing browser instance.")
		if err := bim.browser.Close(); err != nil {
			bim.logger.Error("Failed to close browser", slog.Any("error", err))
			return err
		}
		bim.browser = nil
		bim.logger.Info("Browser instance closed.")
	} else {
		bim.logger.Debug("No active browser instance to close.")
	}
	return nil
}

// ResetInactivityTimer resets an inactivity timer for the browser instance.
func (bim *BrowserInstanceManager) ResetInactivityTimer() {
	if bim.inactivityTimer != nil {
		bim.inactivityTimer.Stop()
	}
	if bim.cancelTimeout != nil {
		bim.cancelTimeout()
	}

	var ctx context.Context
	ctx, bim.cancelTimeout = context.WithCancel(context.Background())

	bim.inactivityTimer = time.AfterFunc(bim.inactivityTimeout, func() {
		select {
		case <-ctx.Done():
			// Timer was cancelled, do nothing
			bim.logger.Debug("Inactivity timer cancelled.")
			return
		default:
			bim.logger.Info("Browser inactivity timeout reached, closing instance.")
			if err := bim.CloseBrowserInstance(); err != nil {
				bim.logger.Error("Failed to close browser on inactivity timeout", slog.Any("error", err))
			}
		}
	})
	bim.logger.Debug("Inactivity timer reset.", slog.Duration("timeout", bim.inactivityTimeout))
}

// KeepAlive resets the inactivity timer without requiring a browser instance.
func (bim *BrowserInstanceManager) KeepAlive() {
	bim.ResetInactivityTimer()
	bim.logger.Debug("Browser instance keep-alive signal received, timer reset.")
}
