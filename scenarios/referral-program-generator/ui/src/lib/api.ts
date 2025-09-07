import axios from 'axios';

// Get API base URL from environment or default to localhost
const API_BASE_URL = import.meta.env.VITE_API_URL || `http://localhost:${import.meta.env.VITE_API_PORT || '8080'}`;

// Create axios instance with default configuration
export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000, // 30 seconds timeout for analysis operations
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for adding auth tokens (if needed)
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for handling common errors
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized access
      localStorage.removeItem('auth_token');
      // Redirect to login if needed
    }
    
    return Promise.reject(error);
  }
);

// API endpoint functions
export const referralAPI = {
  // Health check
  health: () => api.get('/health'),
  
  // Scenario analysis
  analyzeScenario: (data: { scenario_path?: string; url?: string; mode: 'local' | 'deployed' }) =>
    api.post('/api/v1/referral/analyze', data),
  
  // Program generation
  generateProgram: (data: {
    analysis_data: any;
    commission_rate?: number;
    custom_branding?: any;
    output_directory?: string;
  }) => api.post('/api/v1/referral/generate', data),
  
  // Program implementation
  implementProgram: (data: {
    program_id: string;
    scenario_path: string;
    auto_mode: boolean;
  }) => api.post('/api/v1/referral/implement', data),
  
  // List programs
  listPrograms: (scenario?: string) => 
    api.get('/api/v1/referral/programs', { params: scenario ? { scenario } : {} }),
  
  // Dashboard stats (mock endpoint for now)
  getDashboardStats: () => 
    api.get('/api/v1/referral/dashboard/stats').catch(() => ({
      data: {
        totalPrograms: 12,
        totalRevenue: 47892,
        activeLinks: 156,
        conversionRate: 0.087,
        recentPrograms: [
          {
            id: '1',
            scenario_name: 'my-saas-app',
            commission_rate: 0.25,
            created_at: new Date().toISOString(),
          },
          {
            id: '2',
            scenario_name: 'e-commerce-platform',
            commission_rate: 0.15,
            created_at: new Date(Date.now() - 86400000).toISOString(),
          },
        ],
      }
    })),
};

export default api;