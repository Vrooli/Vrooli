/**
 * API Configuration and Utilities
 * Handles dynamic API URL discovery for the AI Chatbot Manager
 */

class APIClient {
  constructor() {
    this.baseURL = null;
    this.baseURLPromise = null;
  }

  /**
   * Get the API base URL dynamically (memoized)
   * Priority: Environment variable -> Config endpoint -> Error
   */
  async getAPIBaseURL() {
    // Return cached value if already resolved
    if (this.baseURL) {
      return this.baseURL;
    }

    // Return existing promise if in progress
    if (this.baseURLPromise) {
      return this.baseURLPromise;
    }

    // Create new promise for URL resolution
    this.baseURLPromise = this._resolveAPIBaseURL();
    this.baseURL = await this.baseURLPromise;
    return this.baseURL;
  }

  /**
   * Internal method to resolve the API base URL
   */
  async _resolveAPIBaseURL() {
    // First try: Check if API_URL is provided via environment
    if (process.env.REACT_APP_API_URL) {
      return process.env.REACT_APP_API_URL;
    }

    // Second try: Construct from API_PORT environment variable
    if (process.env.REACT_APP_API_PORT) {
      return `http://localhost:${process.env.REACT_APP_API_PORT}`;
    }

    // Third try: Fetch from UI server config endpoint
    try {
      const response = await fetch('/config');
      if (response.ok) {
        const config = await response.json();
        return config.apiUrl;
      }
    } catch (error) {
      console.error('Failed to fetch config from UI server:', error);
    }

    // No fallback - proper configuration required
    throw new Error('API URL not configured. Ensure the scenario is running through Vrooli lifecycle system.');
  }

  /**
   * Make an API request with proper error handling
   */
  async request(endpoint, options = {}) {
    const baseURL = await this.getAPIBaseURL();
    const url = `${baseURL}${endpoint}`;
    
    const defaultOptions = {
      headers: {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      },
    };

    const config = { ...defaultOptions, ...options };

    try {
      const response = await fetch(url, config);
      return response;
    } catch (error) {
      console.error(`API request failed: ${url}`, error);
      throw error;
    }
  }

  /**
   * GET request helper
   */
  async get(endpoint) {
    return this.request(endpoint, { method: 'GET' });
  }

  /**
   * POST request helper
   */
  async post(endpoint, data) {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  /**
   * PUT request helper
   */
  async put(endpoint, data) {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  /**
   * DELETE request helper
   */
  async delete(endpoint) {
    return this.request(endpoint, { method: 'DELETE' });
  }
}

// Create and export singleton instance
const apiClient = new APIClient();
export default apiClient;

// Export the base URL for direct access if needed
export const getAPIBaseURL = () => apiClient.getAPIBaseURL();