/**
 * API Configuration and Utilities
 * Handles dynamic API URL discovery for the AI Chatbot Manager
 */

class APIClient {
  constructor() {
    this.baseURL = this.getAPIBaseURL();
  }

  /**
   * Get the API base URL dynamically
   * Priority: Environment variable -> Dynamic discovery -> Fallback
   */
  getAPIBaseURL() {
    // First try: Check if API_URL is provided via environment
    if (process.env.REACT_APP_API_URL) {
      return process.env.REACT_APP_API_URL;
    }

    // Second try: Construct from API_PORT environment variable
    if (process.env.REACT_APP_API_PORT) {
      return `http://localhost:${process.env.REACT_APP_API_PORT}`;
    }

    // Fallback: Use default port range midpoint for development
    // In production, this should be configured properly via environment
    const fallbackPort = 17000; // Middle of range 15000-19999
    console.warn(`API URL not configured. Using fallback: http://localhost:${fallbackPort}`);
    console.warn(`Set REACT_APP_API_URL or REACT_APP_API_PORT environment variable for proper configuration`);
    
    return `http://localhost:${fallbackPort}`;
  }

  /**
   * Make an API request with proper error handling
   */
  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    
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
export const getAPIBaseURL = () => apiClient.baseURL;