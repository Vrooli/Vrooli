/**
 * API Configuration and Utilities
 * Handles dynamic API URL discovery for the AI Chatbot Manager
 */

class APIClient {
  private baseURL: string | null = null;
  private baseURLPromise: Promise<string> | null = null;

  async getAPIBaseURL(): Promise<string> {
    if (this.baseURL) {
      return this.baseURL;
    }

    if (this.baseURLPromise) {
      return this.baseURLPromise;
    }

    this.baseURLPromise = this.resolveAPIBaseURL();
    this.baseURL = await this.baseURLPromise;
    return this.baseURL;
  }

  private async resolveAPIBaseURL(): Promise<string> {
    const explicitUrl = import.meta.env.VITE_API_URL;
    if (explicitUrl) {
      return explicitUrl;
    }

    const explicitPort = import.meta.env.VITE_API_PORT;
    if (explicitPort) {
      return `http://localhost:${explicitPort}`;
    }

    try {
      const response = await fetch('/config');
      if (response.ok) {
        const config = (await response.json()) as { apiUrl?: string };
        if (config.apiUrl) {
          return config.apiUrl;
        }
      }
    } catch (error) {
      console.warn('Failed to fetch /config from UI server', error);
    }

    throw new Error('API URL not configured. Ensure the scenario is running through the Vrooli lifecycle system.');
  }

  async request(endpoint: string, options: RequestInit = {}): Promise<Response> {
    const baseURL = await this.getAPIBaseURL();
    const url = `${baseURL}${endpoint}`;

    const defaultOptions: RequestInit = {
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
    };

    const mergedOptions: RequestInit = {
      ...defaultOptions,
      ...options,
      headers: {
        ...(defaultOptions.headers ?? {}),
        ...(options.headers ?? {}),
      },
    };

    try {
      const response = await fetch(url, mergedOptions);
      return response;
    } catch (error) {
      console.error(`[API] Request failed for ${url}`, error);
      throw error;
    }
  }

  get(endpoint: string): Promise<Response> {
    return this.request(endpoint, { method: 'GET' });
  }

  post(endpoint: string, data: unknown): Promise<Response> {
    return this.request(endpoint, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  put(endpoint: string, data: unknown): Promise<Response> {
    return this.request(endpoint, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  delete(endpoint: string): Promise<Response> {
    return this.request(endpoint, { method: 'DELETE' });
  }
}

const apiClient = new APIClient();
export default apiClient;

export const getAPIBaseURL = () => apiClient.getAPIBaseURL();
