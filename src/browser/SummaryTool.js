const PlaywrightIntegration = require('./PlaywrightIntegration');
const NetworkRequestCapture = require('./NetworkRequestCapture');

class SummaryTool {
  constructor(playwrightIntegration) {
    if (!(playwrightIntegration instanceof PlaywrightIntegration)) {
      throw new Error("SummaryTool requires an instance of PlaywrightIntegration.");
    }
    this.playwrightIntegration = playwrightIntegration;
  }

  async extractLinks(page) {
    return await page.evaluate(() => {
      const linkElements = document.querySelectorAll('a');
      return Array.from(linkElements).map(link => {
        return {
          text: link.innerText.trim() || link.textContent.trim() || '',
          href: link.href || ''
        };
      });
    });
  }

  async summarizePage(url, options = {}) {
    const networkCapture = new NetworkRequestCapture();
    let page;
    let context;
    try {
      ({ page, context } = await this.playwrightIntegration.navigateToUrl(url, {
        waitUntil: options.waitUntil || 'networkidle'
      }));
      
      // Set up network interception before navigation
      await networkCapture.setupInterception(page);

      // Navigate to the URL with interception enabled
      await page.goto(url, { waitUntil: 'networkidle' });
      
      // Capture the full HTML
      const fullHtml = await page.content();
      
      // Extract links
      const links = await this.extractLinks(page);
      
      // Get captured network requests
      const networkRequests = networkCapture.getRequests();

      // Capture screenshot
      const screenshotBuffer = await this.playwrightIntegration.captureScreenshot(page);
      const screenshot = screenshotBuffer.toString('base64');
      
      // Create the summary object
      const summary = {
        url,
        timestamp: new Date().toISOString(),
        full_html: fullHtml,
        links,
        network_requests: networkRequests,
        screenshot
      };
      
      return summary;
    } catch (error) {
      throw new Error(`Failed to summarize page: ${error.message}`);
    } finally {
      if (page && context) {
        await this.playwrightIntegration.closePage(page, context);
      }
    }
  }
}

/**
 * API endpoint implementation for summarizing a page.
 * This function assumes `playwrightIntegration` is available in its scope,
 * either passed as an argument or imported and instantiated globally.
 * @param {object} req - The request object.
 * @param {object} res - The response object.
 * @param {PlaywrightIntegration} playwrightIntegration - An instance of PlaywrightIntegration.
 */
async function handleSummarizePageRequest(req, res, playwrightIntegration) {
  const { url } = req.body;
  
  if (!url) {
    return res.status(400).json({ error: 'URL is required' });
  }
  
  try {
    // Ensure playwrightIntegration is provided
    if (!playwrightIntegration) {
      throw new Error("PlaywrightIntegration instance is not provided to handleSummarizePageRequest.");
    }
    const summaryTool = new SummaryTool(playwrightIntegration);
    const summary = await summaryTool.summarizePage(url);
    return res.json(summary);
  } catch (error) {
    console.error(`Error in handleSummarizePageRequest: ${error.message}`);
    return res.status(500).json({ error: error.message });
  }
}

module.exports = {
  SummaryTool,
  handleSummarizePageRequest
};