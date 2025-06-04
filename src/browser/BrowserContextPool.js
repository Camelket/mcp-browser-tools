class BrowserContextPool {
  constructor(browserManager, options = {}) {
    this.browserManager = browserManager;
    this.maxPoolSize = options.maxPoolSize || 5;
    this.idleTimeout = options.idleTimeout || 30000; // 30 seconds
    this.pool = [];
    this.inUse = new Map();
    this.logger = options.logger || console;
    this.cleanupIntervalId = null; // To store the interval ID for cleanup
  }

  async initialize(options = {}) {
    // Pre-create some contexts if needed
    if (options.preCreateContexts) {
      for (let i = 0; i < Math.min(options.preCreateContexts, this.maxPoolSize); i++) {
        const context = await this.createContext();
        this.pool.push({
          context,
          createdAt: Date.now(),
          lastUsed: Date.now()
        });
      }
    }
    this.startIdleCleanupInterval();
  }

  async createContext() {
    const browser = await this.browserManager.getBrowserInstance();
    return await browser.newContext();
  }

  async acquireContext() {
    // Check if there's an available context in the pool
    if (this.pool.length > 0) {
      const contextInfo = this.pool.shift();
      contextInfo.lastUsed = Date.now();
      this.inUse.set(contextInfo.context, contextInfo);
      return contextInfo.context;
    }
    
    // If pool is empty but we haven't reached max size, create a new context
    if (this.inUse.size < this.maxPoolSize) {
      const context = await this.createContext();
      const contextInfo = {
        context,
        createdAt: Date.now(),
        lastUsed: Date.now()
      };
      this.inUse.set(context, contextInfo);
      return context;
    }
    
    // If we've reached max size, wait for a context to become available
    return new Promise((resolve) => {
      const checkInterval = setInterval(async () => {
        if (this.pool.length > 0) {
          clearInterval(checkInterval);
          const contextInfo = this.pool.shift();
          contextInfo.lastUsed = Date.now();
          this.inUse.set(contextInfo.context, contextInfo);
          resolve(contextInfo.context);
        }
      }, 100);
    });
  }

  releaseContext(context) {
    if (this.inUse.has(context)) {
      const contextInfo = this.inUse.get(context);
      this.inUse.delete(context);
      contextInfo.lastUsed = Date.now();
      this.pool.push(contextInfo);
      
      // Reset the inactivity timer on the browser manager
      this.browserManager.resetInactivityTimer();
    }
  }

  async cleanupIdleContexts() {
    const now = Date.now();
    const contextsToRemove = [];
    
    // Find idle contexts in the pool
    for (let i = 0; i < this.pool.length; i++) {
      const contextInfo = this.pool[i];
      if (now - contextInfo.lastUsed > this.idleTimeout) {
        contextsToRemove.push(i);
      }
    }
    
    // Remove idle contexts in reverse order to avoid index issues
    for (let i = contextsToRemove.length - 1; i >= 0; i--) {
      const index = contextsToRemove[i];
      const contextInfo = this.pool.splice(index, 1)[0];
      await contextInfo.context.close();
      this.logger.debug(`Closed idle context (unused for ${now - contextInfo.lastUsed}ms)`);
    }
  }

  startIdleCleanupInterval() {
    // Run cleanup every half of the idle timeout period
    const interval = Math.max(this.idleTimeout / 2, 5000);
    this.cleanupIntervalId = setInterval(() => this.cleanupIdleContexts(), interval);
  }

  stopIdleCleanupInterval() {
    if (this.cleanupIntervalId) {
      clearInterval(this.cleanupIntervalId);
      this.cleanupIntervalId = null;
    }
  }

  async drain() {
    this.stopIdleCleanupInterval();
    for (const contextInfo of this.pool) {
      await contextInfo.context.close();
    }
    this.pool = [];
    for (const contextInfo of this.inUse.values()) {
      await contextInfo.context.close();
    }
    this.inUse.clear();
  }
}

module.exports = BrowserContextPool;