const playwright = require('playwright');

class BrowserInstanceManager {
  constructor(options = {}) {
    this.browser = null;
    this.inactivityTimeout = options.timeout || 60000; // Default 1 minute
    this.inactivityTimer = null;
    this.isShuttingDown = false;
    this.logger = options.logger || console;
    this.browserContexts = new Set(); // To keep track of active contexts
  }

  static getInstance(options = {}) {
    if (!BrowserInstanceManager.instance) {
      BrowserInstanceManager.instance = new BrowserInstanceManager(options);
    }
    return BrowserInstanceManager.instance;
  }

  async getBrowserInstance() {
    if (!this.browser) {
      this.logger.info('Launching new browser instance...');
      this.browser = await playwright.chromium.launch({
        headless: true,
        // Additional Playwright options can be passed here
      });

      // Handle browser disconnection
      this.browser.on('disconnected', () => {
        this.logger.warn('Browser disconnected unexpectedly. Resetting instance.');
        this.browser = null;
        this.browserContexts.clear(); // Clear all contexts as they are no longer valid
        this.clearInactivityTimer(); // Stop the timer if browser crashes
      });
    }
    this.resetInactivityTimer();
    return this.browser;
  }

  async acquireBrowserContext() {
    const browser = await this.getBrowserInstance();
    const context = await browser.newContext();
    this.browserContexts.add(context);
    this.resetInactivityTimer();
    return context;
  }

  async releaseBrowserContext(context) {
    if (context) {
      await context.close();
      this.browserContexts.delete(context);
    }
    // Only reset timer if there are no active contexts
    if (this.browserContexts.size === 0) {
      this.resetInactivityTimer();
    }
  }

  resetInactivityTimer() {
    if (this.inactivityTimer) {
      clearTimeout(this.inactivityTimer);
    }
    
    this.inactivityTimer = setTimeout(() => {
      this.logger.info(`Browser instance inactive for ${this.inactivityTimeout}ms, shutting down`);
      this.shutdown();
    }, this.inactivityTimeout);
  }

  clearInactivityTimer() {
    if (this.inactivityTimer) {
      clearTimeout(this.inactivityTimer);
      this.inactivityTimer = null;
    }
  }

  async shutdown() {
    if (this.browser && !this.isShuttingDown) {
      this.isShuttingDown = true;
      this.logger.info('Shutting down browser instance');
      
      if (this.inactivityTimer) {
        clearTimeout(this.inactivityTimer);
        this.inactivityTimer = null;
      }
      
      // Close all active contexts before closing the browser
      for (const context of this.browserContexts) {
        try {
          await context.close();
        } catch (error) {
          this.logger.error('Error closing browser context during shutdown:', error);
        }
      }
      this.browserContexts.clear();
 
      await this.browser.close();
      this.browser = null;
      this.isShuttingDown = false;
      this.logger.info('Browser instance successfully shut down');
    }
  }

  setInactivityTimeout(timeoutMs) {
    if (typeof timeoutMs !== 'number' || timeoutMs < 0) {
      throw new Error('Inactivity timeout must be a non-negative number.');
    }
    this.logger.info(`Setting inactivity timeout to ${timeoutMs}ms`);
    this.inactivityTimeout = timeoutMs;
    // Reset timer immediately if there are no active contexts, otherwise it will be reset on next acquire/release
    if (this.browserContexts.size === 0) {
      this.resetInactivityTimer();
    }
  }

  keepAlive() {
    this.logger.debug('Keep-alive signal received');
    this.resetInactivityTimer();
  }
}

module.exports = BrowserInstanceManager;