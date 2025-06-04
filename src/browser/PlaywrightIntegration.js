const BrowserInstanceManager = require('./BrowserInstanceManager');

class PlaywrightIntegration {
  constructor(browserManager) {
    if (!(browserManager instanceof BrowserInstanceManager)) {
      throw new Error("PlaywrightIntegration requires an instance of BrowserInstanceManager.");
    }
    this.browserManager = browserManager;
  }

  async navigateToUrl(url, options = {}) {
    let context;
    let page;
    try {
      context = await this.browserManager.acquireBrowserContext();
      page = await context.newPage();
      
      // Configure navigation timeout
      page.setDefaultNavigationTimeout(options.timeout || 30000);
      
      // Navigate to the URL
      await page.goto(url, {
        waitUntil: options.waitUntil || 'networkidle'
      });
      
      return { page, context };
    } catch (error) {
      if (page) {
        await page.close();
      }
      if (context) {
        this.browserManager.releaseBrowserContext(context);
      }
      throw new Error(`Navigation failed: ${error.message}`);
    }
  }

  async closePage(page, context) {
    if (page) {
      await page.close();
    }
    if (context) {
      this.browserManager.releaseBrowserContext(context);
    }
  }

  async executeScript(page, script) {
    if (!page) {
      throw new Error("Page object is required to execute script.");
    }
    return await page.evaluate(script);
  }

  async captureScreenshot(page, options = {}) {
    if (!page) {
      throw new Error("Page object is required to capture screenshot.");
    }
    return await page.screenshot({
      fullPage: options.fullPage || true,
      type: 'png'
    });
  }

  async setupNetworkInterception(page, handlers) {
    if (!page) {
      throw new Error("Page object is required to setup network interception.");
    }
    await page.route('**/*', (route, request) => {
      if (handlers.onRequest) handlers.onRequest(route, request);
      route.continue();
    });
  }
}

module.exports = PlaywrightIntegration;