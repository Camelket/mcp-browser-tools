class NetworkRequestCapture {
  constructor() {
    this.requests = [];
  }

  async setupInterception(page) {
    const requests = this.requests;
    
    await page.route('**/*', async (route, request) => {
      const requestData = {
        url: request.url(),
        method: request.method(),
        headers: request.headers(),
        request_body: request.postData() || ''
      };
      
      try {
        const response = await route.fetch();
        const responseBody = await this.safelyGetResponseBody(response);
        
        requestData.response_status = response.status();
        requestData.response_headers = response.headers();
        requestData.response_body = responseBody;
        
        requests.push(requestData);
        await route.fulfill({
          response,
          body: responseBody
        });
      } catch (error) {
        requests.push({
          ...requestData,
          error: error.message
        });
        await route.continue();
      }
    });
  }

  async safelyGetResponseBody(response) {
    try {
      // Only attempt to get body for text-based content types
      const contentType = response.headers()['content-type'] || '';
      if (contentType.includes('text/') || 
          contentType.includes('application/json') || 
          contentType.includes('application/xml')) {
        return await response.text();
      }
      return '';
    } catch (error) {
      return `[Error getting response body: ${error.message}]`;
    }
  }

  getRequests() {
    return this.requests;
  }
}

module.exports = NetworkRequestCapture;