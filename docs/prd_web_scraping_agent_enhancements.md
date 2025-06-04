## Product Requirements Document: Web Scraping Agent Enhancements

### Overview

This document outlines enhancements to our existing web scraping agent, focusing on improving efficiency, data richness, and user experience. The web scraping agent is designed to automate the extraction of data from websites, serving a diverse user base including data analysts, researchers, and developers who require structured information from the web.

The primary problem these enhancements address is the current overhead associated with repeated browser launches for each scraping task and the limited scope of data captured beyond basic HTML. By introducing a "Summary Tool" and a "Persistent Reusable Playwright Instance," we aim to significantly reduce operational costs, accelerate data acquisition, and provide a more comprehensive dataset for analysis. These features are valuable as they directly translate to faster, more cost-effective, and insightful web data extraction, empowering users to achieve their data-driven objectives with greater ease and depth.

### Core Features

#### 1. Summary Tool

*   **What it does**: The Summary Tool provides a comprehensive snapshot of a web page after it has fully loaded. It encapsulates various critical data points into a single, well-formed object, offering a holistic view of the page's state and interactions. The output includes:
    *   `full_html`: The complete HTML content of the page after all dynamic elements have rendered.
    *   `links`: A structured list of all hyperlinks present on the page, including their text and href attributes.
    *   `network_requests`: Detailed information about all network requests made during the page load, encompassing request and response headers, and body content where applicable.
    *   `screenshot`: A base64 encoded PNG image representing a full-page screenshot, capturing the visual state of the page.

*   **Why it's important (benefits)**: This tool is crucial for several reasons:
    *   **Rich Data Context**: Provides a much richer context than just HTML, enabling deeper analysis of page behavior, linked resources, and visual layout.
    *   **Debugging and Validation**: Facilitates easier debugging of scraping logic by offering insights into network activity and visual rendering.
    *   **Efficiency**: Consolidates multiple data points into a single API call, reducing the need for separate requests for different types of page information.
    *   **Comprehensive Auditing**: Allows for a complete audit trail of how a page loaded and what resources it interacted with.

*   **How it works at a high level**: Upon activation, the Summary Tool leverages the underlying Playwright instance to navigate to the specified URL. It then intercepts network requests, captures the fully rendered HTML, extracts all link elements, and takes a full-page screenshot. All this collected data is then aggregated into a predefined JSON object structure and returned to the user.

#### 2. Persistent Reusable Playwright Instance

*   **What it does**: This feature maintains a single, long-lived Playwright browser instance that can be reused across multiple scraping tasks or tool calls. Instead of launching a new browser for each operation, the agent will connect to an already running instance. It will also incorporate an inactivity timeout (e.g., 1 minute) to automatically close the instance if it remains unused for a specified period, balancing persistence with resource management.

*   **Why it's important (benefits)**: The benefits of a persistent instance are significant:
    *   **Reduced Overhead**: Eliminates the substantial startup time and resource consumption associated with launching a new browser instance for every scraping job.
    *   **Increased Performance**: Leads to faster execution of scraping tasks, especially for scenarios involving multiple interactions with the same or related domains.
    *   **Resource Optimization**: More efficient use of system resources by avoiding redundant browser processes.
    *   **Improved Stability**: A stable, long-running instance can potentially handle complex navigation and interactions more reliably.

*   **How it works at a high level**: A dedicated service or module will manage the lifecycle of the Playwright browser instance. When a scraping task is initiated, it will first check for an active, available instance. If one exists, it will be reused; otherwise, a new instance will be launched and made available. An internal timer will track inactivity, triggering the closure of the instance if the timeout is reached. This manager will expose an API for tools (like the Summary Tool) to request and release the browser context.

### User Experience

The introduction of these features will significantly enhance the user journey for data extraction.

*   **User Persona: Data Analyst Sarah**: Sarah frequently needs to extract product information from e-commerce sites. With the persistent Playwright instance, her scripts will run significantly faster, allowing her to iterate on her data models more rapidly. The Summary Tool will provide her with not just product details (HTML), but also related product links and an understanding of how the page loads (network requests), enabling her to discover hidden relationships and debug issues more effectively. She can quickly verify the visual state of the page with the screenshot, ensuring the data she's extracting is from the correct visual context.

*   **User Persona: Developer David**: David is building an automated testing suite for a web application. The persistent Playwright instance means his test runs are quicker, reducing CI/CD pipeline times. The Summary Tool's detailed network request information will be invaluable for debugging API calls and understanding front-end performance issues, while the full HTML and screenshot provide a complete picture of the application's state at any given point.

*   **Key User Flows**:
    1.  **Rapid Data Extraction**: User initiates a scraping task. The agent reuses an existing Playwright instance, leading to near-instantaneous page loading and data extraction.
    2.  **Comprehensive Page Analysis**: User requests a page summary. The Summary Tool returns a rich object containing HTML, links, network requests, and a screenshot, all in one go, simplifying data collection and analysis.
    3.  **Debugging and Iteration**: User encounters an issue. They use the Summary Tool to get a complete picture of the problematic page, quickly identifying the root cause (e.g., a broken link, a failed network request, or an unexpected visual layout).

### Technical Architecture

#### System Components Involved

*   **Playwright Integration Layer**: A module responsible for abstracting Playwright interactions, managing browser contexts, and handling page navigation.
*   **Browser Instance Manager**: A new service or component dedicated to managing the lifecycle of the persistent Playwright instance, including creation, reuse, and inactivity-based termination.
*   **Summary Data Collector**: A component within the Playwright Integration Layer responsible for orchestrating the collection of HTML, links, network requests, and screenshots.
*   **API Gateway/Tool Interface**: The existing interface through which external tools or user requests interact with the web scraping agent.

#### Data Models for the Summary Tool's Output

The Summary Tool will output a JSON object conforming to the following structure:

```json
{
  "url": "string",
  "timestamp": "ISO 8601 string",
  "full_html": "string",
  "links": [
    {
      "text": "string",
      "href": "string"
    }
  ],
  "network_requests": [
    {
      "url": "string",
      "method": "string",
      "headers": {
        "key": "value"
      },
      "request_body": "string (optional)",
      "response_status": "number",
      "response_headers": {
        "key": "value"
      },
      "response_body": "string (optional)"
    }
  ],
  "screenshot": "base64 encoded PNG string"
}
```

#### APIs and Integrations

*   **Summary Tool Interaction**: The Summary Tool will expose a simple API endpoint (e.g., `/summarize_page`) that accepts a URL. Internally, it will call the Playwright Integration Layer to perform the data collection.
*   **Playwright Instance Management API**: The Browser Instance Manager will expose internal APIs for:
    *   `acquire_browser_context()`: Requests a browser context from the persistent instance.
    *   `release_browser_context()`: Releases the browser context back to the pool.
    *   `set_inactivity_timeout(duration_ms)`: Configures the timeout for the persistent instance.
*   **Network Interception**: Playwright's `page.route()` will be used to intercept and capture network requests and responses.
*   **Screenshot Capture**: Playwright's `page.screenshot()` will be used for full-page screenshot capture.

#### Infrastructure Requirements

*   **Playwright Dependencies**: The environment running the web scraping agent must have all necessary Playwright browser dependencies installed (e.g., Chromium, Firefox, WebKit). This might require Docker images with pre-installed browsers or specific system configurations.
*   **Memory and CPU**: Maintaining a persistent browser instance will require more dedicated memory and CPU resources compared to ephemeral instances, especially if multiple contexts are managed concurrently.
*   **Storage**: Increased storage might be needed for storing screenshots and potentially large HTML or network request bodies, though these would typically be transient.

### Development Roadmap

#### Phase 1: Minimum Viable Product (MVP)

*   **Goal**: Implement the core functionality of the persistent Playwright instance and a basic Summary Tool.
*   **Features**:
    *   **Persistent Playwright Instance**:
        *   Ability to launch a single Playwright browser instance and keep it alive.
        *   Basic mechanism to acquire and release page contexts from this instance.
        *   Manual closure of the instance.
    *   **Summary Tool (Basic)**:
        *   Capture `full_html` and `screenshot` for a given URL using the persistent instance.
        *   Return these two data points in a basic JSON object.
*   **Deliverables**: Functional persistent browser instance, basic Summary Tool API endpoint.

#### Phase 2: Enhanced Summary Tool & Instance Management

*   **Goal**: Expand the Summary Tool's capabilities and introduce automated instance management.
*   **Features**:
    *   **Summary Tool (Enhanced)**:
        *   Add `links` extraction to the Summary Tool output.
        *   Implement `network_requests` capture (request/response headers, URL, method).
    *   **Persistent Playwright Instance (Automated)**:
        *   Implement the inactivity timeout mechanism for automatic instance closure.
        *   Graceful shutdown of the persistent instance.
*   **Deliverables**: Fully featured Summary Tool, automated Playwright instance lifecycle management.

#### Phase 3: Performance & Scalability

*   **Goal**: Optimize performance and prepare for potential scaling.
*   **Features**:
    *   **Connection Pooling**: Implement a connection pooling mechanism for Playwright contexts to handle concurrent requests more efficiently.
    *   **Error Handling & Resilience**: Robust error handling for Playwright interactions and instance failures.
    *   **Configuration**: Externalize configuration for inactivity timeout and other Playwright launch options.
*   **Deliverables**: Performance improvements, robust error handling, configurable agent.

### Logical Dependency Chain

1.  **Persistent Reusable Playwright Instance (MVP)**: This is the foundational component. Without a persistent instance, the benefits of the Summary Tool (reduced overhead) cannot be fully realized. This involves setting up the core Playwright management logic.
2.  **Summary Tool (Basic)**: Once the persistent instance is stable, the basic Summary Tool can be built on top of it. This allows for quick validation of the persistent instance's functionality and provides immediate value with HTML and screenshot capture.
3.  **Summary Tool (Enhanced)**: Building on the basic Summary Tool, adding link and network request capture enhances the data richness. This can be done incrementally, as it relies on the established Playwright interaction patterns.
4.  **Persistent Reusable Playwright Instance (Automated Management)**: Implementing the inactivity timeout and graceful shutdown mechanisms. This refines the resource management aspect and improves the overall stability and maintainability of the system.
5.  **Performance & Scalability Enhancements**: Once the core features are stable and functional, focus can shift to optimizing performance, adding connection pooling, and improving error handling for production readiness.

This sequence ensures that a usable and valuable product is delivered incrementally, with each phase building upon a solid foundation.

### Risks and Mitigations

*   **Risk: Performance Degradation due to Persistent Instance**:
    *   **Description**: A long-running Playwright instance might accumulate memory leaks or become unstable over time, leading to performance degradation or crashes.
    *   **Mitigation**:
        *   Implement regular health checks and monitoring for the Playwright instance (e.g., memory usage, CPU).
        *   Introduce a configurable maximum lifespan for the instance, forcing a restart after a certain period or number of operations.
        *   Utilize Playwright's `browser.newPage()` and `page.close()` effectively to manage individual page contexts without restarting the entire browser.
        *   Implement the inactivity timeout to ensure unused instances are closed, freeing up resources.

*   **Risk: Resource Exhaustion (Memory/CPU)**:
    *   **Description**: Maintaining a persistent browser instance, especially with multiple concurrent page contexts or heavy network interception, could lead to high memory and CPU consumption.
    *   **Mitigation**:
        *   Implement connection pooling with a configurable maximum number of active page contexts.
        *   Optimize network interception logic to only capture necessary data.
        *   Provide clear infrastructure requirements and recommendations for deployment environments.
        *   Consider headless mode by default to reduce rendering overhead.

*   **Risk: Data Volume for Summary Tool**:
    *   **Description**: The `full_html`, `network_requests`, and `screenshot` components of the Summary Tool can generate very large data payloads, impacting API response times and storage if persisted.
    *   **Mitigation**:
        *   Implement data compression for the `screenshot` (PNG optimization) and potentially for `full_html` if transferred over the wire.
        *   Offer options to selectively include/exclude certain data points in the Summary Tool's output based on user needs (e.g., `include_screenshot=false`).
        *   For `network_requests`, allow filtering by resource type or status code to reduce the volume.
        *   Implement streaming for large responses if feasible.

*   **Risk: Playwright Version Compatibility**:
    *   **Description**: Future updates to Playwright might introduce breaking changes, requiring updates to our integration layer.
    *   **Mitigation**:
        *   Maintain a dedicated Playwright integration layer that abstracts the underlying library, minimizing the impact of external changes.
        *   Implement automated tests for the Playwright integration to quickly identify breaking changes.
        *   Regularly review Playwright release notes and plan for timely updates.

*   **Risk: Inactivity Timeout Edge Cases**:
    *   **Description**: The inactivity timeout might close the browser instance prematurely if there are long-running but infrequent tasks, or if the timeout is misconfigured.
    *   **Mitigation**:
        *   Make the inactivity timeout configurable by the user.
        *   Implement robust logging to track instance activity and closure events.
        *   Provide clear documentation and best practices for configuring the timeout based on usage patterns.
        *   Consider a "keep-alive" mechanism for long-running tasks that periodically signal activity.