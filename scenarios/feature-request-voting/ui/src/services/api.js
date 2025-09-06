import axios from 'axios';

const API_BASE_URL = '/api/v1';

class ApiService {
  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth token to requests if available
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('authToken');
      if (token) {
        config.headers['Authorization'] = `Bearer ${token}`;
      }
      
      const userId = localStorage.getItem('userId');
      if (userId) {
        config.headers['X-User-ID'] = userId;
      }
      
      const sessionId = localStorage.getItem('sessionId');
      if (sessionId) {
        config.headers['X-Session-ID'] = sessionId;
      }
      
      return config;
    });

    // Generate session ID if not exists
    if (!localStorage.getItem('sessionId')) {
      localStorage.setItem('sessionId', this.generateSessionId());
    }
  }

  generateSessionId() {
    return 'session_' + Math.random().toString(36).substr(2, 9) + Date.now().toString(36);
  }

  setAuthToken(token) {
    localStorage.setItem('authToken', token);
  }

  clearAuthToken() {
    localStorage.removeItem('authToken');
    localStorage.removeItem('userId');
  }

  // Scenarios
  async getScenarios() {
    const response = await this.client.get('/scenarios');
    return response.data;
  }

  async getScenario(id) {
    const response = await this.client.get(`/scenarios/${id}`);
    return response.data;
  }

  // Feature Requests
  async getFeatureRequests(scenarioId, params = {}) {
    const response = await this.client.get(`/scenarios/${scenarioId}/feature-requests`, { params });
    return response.data.requests || [];
  }

  async getFeatureRequest(id) {
    const response = await this.client.get(`/feature-requests/${id}`);
    return response.data;
  }

  async createFeatureRequest(data) {
    const response = await this.client.post('/feature-requests', data);
    return response.data;
  }

  async updateFeatureRequest(id, updates) {
    const response = await this.client.put(`/feature-requests/${id}`, updates);
    return response.data;
  }

  async deleteFeatureRequest(id) {
    const response = await this.client.delete(`/feature-requests/${id}`);
    return response.data;
  }

  async voteOnRequest(requestId, value) {
    const response = await this.client.post(`/feature-requests/${requestId}/vote`, { value });
    return response.data;
  }

  // Authentication (integrate with scenario-authenticator)
  async authenticate(credentials) {
    // This would integrate with scenario-authenticator
    // For now, return mock data
    return {
      id: 'user_' + Math.random().toString(36).substr(2, 9),
      email: credentials.email,
      display_name: credentials.email.split('@')[0],
      token: 'mock_token_' + Date.now(),
    };
  }

  async register(userData) {
    // This would integrate with scenario-authenticator
    // For now, return mock data
    return {
      id: 'user_' + Math.random().toString(36).substr(2, 9),
      email: userData.email,
      display_name: userData.name || userData.email.split('@')[0],
      token: 'mock_token_' + Date.now(),
    };
  }
}

export default new ApiService();