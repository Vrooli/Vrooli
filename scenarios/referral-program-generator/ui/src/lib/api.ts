import axios from 'axios';

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: unknown;
  }
}

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '8080';
const LOOPBACK_DEFAULT = `http://127.0.0.1:${DEFAULT_API_PORT}`;

const isLocalHostname = (hostname?: string | null) => {
  if (!hostname) return false;
  const normalized = hostname.toLowerCase();
  return normalized === 'localhost' || normalized === '127.0.0.1' || normalized === '0.0.0.0' || normalized === '::1' || normalized === '[::1]';
};

const isLikelyProxiedPath = (pathname?: string | null) => {
  if (!pathname) return false;
  return pathname.includes('/apps/') && pathname.includes('/proxy/');
};

const normalizeBase = (value: string) => value.replace(/\/+$/, '');

const resolveApiBaseUrl = (): string => {
  const explicit = (import.meta.env.VITE_API_URL as string | undefined)?.trim();
  if (explicit) {
    const normalized = normalizeBase(explicit);

    if (typeof window !== 'undefined') {
      const { hostname, origin, pathname } = window.location;
      const hasProxyBootstrap = typeof window.__APP_MONITOR_PROXY_INFO__ !== 'undefined';
      const proxiedPath = isLikelyProxiedPath(pathname);
      const isRemote = !isLocalHostname(hostname);

      if (isRemote && !hasProxyBootstrap && !proxiedPath && /localhost|127\.0\.0\.1/i.test(normalized)) {
        return normalizeBase(origin);
      }

      if (hasProxyBootstrap || proxiedPath) {
        return normalized;
      }
    }

    return normalized;
  }

  if (typeof window !== 'undefined') {
    const { hostname, origin, pathname } = window.location;
    const hasProxyBootstrap = typeof window.__APP_MONITOR_PROXY_INFO__ !== 'undefined';
    const proxiedPath = isLikelyProxiedPath(pathname);

    if (!hasProxyBootstrap && !proxiedPath && !isLocalHostname(hostname)) {
      return normalizeBase(origin);
    }

    if (hasProxyBootstrap || proxiedPath) {
      return LOOPBACK_DEFAULT;
    }
  }

  return LOOPBACK_DEFAULT;
};

const API_BASE_URL = resolveApiBaseUrl();

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
