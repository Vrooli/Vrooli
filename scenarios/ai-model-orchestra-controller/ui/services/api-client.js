/**
 * AI Model Orchestra Controller API Client
 * Centralized API communication service
 */

class OrchestraAPIClient {
    constructor() {
        this.baseUrl = null;
        this.initialized = false;
        this.retryCount = 3;
        this.retryDelay = 1000;
    }

    /**
     * Initialize the API client with configuration
     */
    async initialize() {
        try {
            const response = await fetch('/api/config');
            const config = await response.json();
            this.baseUrl = config.apiUrl;
            this.initialized = true;
            return config;
        } catch (error) {
            console.error('Failed to initialize API client:', error);
            // Fallback to environment variable or default
            this.baseUrl = `http://localhost:${window.API_PORT || '15000'}`;
            this.initialized = true;
            return { apiUrl: this.baseUrl };
        }
    }

    /**
     * Make API request with retry logic
     */
    async request(endpoint, options = {}) {
        if (!this.initialized) {
            await this.initialize();
        }

        const url = `${this.baseUrl}${endpoint}`;
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        };

        const finalOptions = { ...defaultOptions, ...options };
        
        for (let attempt = 0; attempt <= this.retryCount; attempt++) {
            try {
                const response = await fetch(url, finalOptions);
                
                if (!response.ok && attempt < this.retryCount) {
                    await this.delay(this.retryDelay * Math.pow(2, attempt));
                    continue;
                }
                
                const data = await response.json();
                return { success: response.ok, data, status: response.status };
            } catch (error) {
                if (attempt === this.retryCount) {
                    console.error(`API request failed after ${this.retryCount} retries:`, error);
                    return { success: false, error: error.message, status: 0 };
                }
                await this.delay(this.retryDelay * Math.pow(2, attempt));
            }
        }
    }

    /**
     * Delay helper for retry logic
     */
    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    /**
     * Health check
     */
    async checkHealth() {
        return this.request('/api/v1/health');
    }

    /**
     * Select optimal model for task
     */
    async selectModel(taskType, requirements = {}) {
        return this.request('/api/v1/ai/select-model', {
            method: 'POST',
            body: JSON.stringify({ taskType, requirements })
        });
    }

    /**
     * Route AI request through orchestrator
     */
    async routeRequest(taskType, prompt, requirements = {}) {
        return this.request('/api/v1/ai/route-request', {
            method: 'POST',
            body: JSON.stringify({ taskType, prompt, requirements })
        });
    }

    /**
     * Get model status and metrics
     */
    async getModelsStatus(includeMetrics = true) {
        const params = includeMetrics ? '?includeMetrics=true' : '';
        return this.request(`/api/v1/ai/models/status${params}`);
    }

    /**
     * Get resource metrics
     */
    async getResourceMetrics(hours = 1, includeHistory = true) {
        const params = `?hours=${hours}&includeHistory=${includeHistory}`;
        return this.request(`/api/v1/ai/resources/metrics${params}`);
    }

    /**
     * Subscribe to real-time updates (placeholder for WebSocket)
     */
    subscribe(event, callback) {
        // TODO: Implement WebSocket subscription
        console.log(`Subscription to ${event} registered (WebSocket not yet implemented)`);
    }
}

// Export as singleton
const apiClient = new OrchestraAPIClient();
export default apiClient;