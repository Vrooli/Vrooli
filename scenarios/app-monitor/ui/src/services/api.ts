import axios, { AxiosError } from 'axios';
import type { App, Resource, LogEntry, ApiResponse, AppLogsResponse } from '@/types';
import { logger } from '@/services/logger';

// Create axios instance with default config
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 35000, // 35s to accommodate 30s backend timeout for resource checks
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for debugging
api.interceptors.request.use(
  (config) => {
    logger.logAPICall(config.method || 'GET', config.url || '', config.data);
    return config;
  },
  (error) => {
    logger.error('API Request error', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    logger.logAPIResponse(response.config.url || '', response.status, response.data);
    return response;
  },
  (error: AxiosError<ApiResponse>) => {
    const message = error.response?.data?.error || error.message || 'Network error';
    logger.error('API Response error', { message, status: error.response?.status });
    
    // Enhanced error object
    const enhancedError = {
      ...error,
      message,
      status: error.response?.status,
      originalError: error,
    };
    
    return Promise.reject(enhancedError);
  }
);

// App Management
export const appService = {
  // Get all apps
  async getApps(): Promise<App[]> {
    try {
      const { data } = await api.get<App[]>('/apps');
      return data || [];
    } catch (error) {
      logger.error('Failed to fetch apps', error);
      return [];
    }
  },

  // Get single app
  async getApp(id: string): Promise<App | null> {
    try {
      const { data } = await api.get<ApiResponse<App>>(`/apps/${id}`);
      return data.data || null;
    } catch (error) {
      logger.error(`Failed to fetch app ${id}`, error);
      return null;
    }
  },

  // Control app (start/stop/restart)
  async controlApp(id: string, action: 'start' | 'stop' | 'restart'): Promise<boolean> {
    try {
      const { data } = await api.post<ApiResponse>(`/apps/${id}/${action}`);
      return data.success !== false;
    } catch (error) {
      logger.error(`Failed to ${action} app ${id}`, error);
      return false;
    }
  },

  // Get app logs
  async getAppLogs(appName: string, logType: 'lifecycle' | 'background' | 'both' = 'both'): Promise<AppLogsResponse> {
    try {
      const params = new URLSearchParams({ type: logType });
      const { data } = await api.get<AppLogsResponse>(`/logs/${appName}?${params}`);
      return data;
    } catch (error) {
      logger.error(`Failed to fetch logs for ${appName}`, error);
      return { logs: [], error: 'Failed to fetch logs' };
    }
  },

  // Control all apps
  async controlAllApps(action: 'start' | 'stop' | 'restart'): Promise<boolean> {
    try {
      const { data } = await api.post<ApiResponse>(`/apps/all/${action}`);
      return data.success !== false;
    } catch (error) {
      logger.error(`Failed to ${action} all apps`, error);
      return false;
    }
  },
};

// Resource Management
export const resourceService = {
  // Get all resources
  async getResources(): Promise<Resource[]> {
    try {
      const { data } = await api.get<Resource[]>('/resources');
      return data || [];
    } catch (error) {
      logger.error('Failed to fetch resources', error);
      return [];
    }
  },

  // Get resource status
  async getResourceStatus(id: string): Promise<Resource | null> {
    try {
      const { data } = await api.get<ApiResponse<Resource>>(`/resources/${id}/status`);
      return data.data || null;
    } catch (error) {
      logger.error(`Failed to fetch resource ${id}`, error);
      return null;
    }
  },
};

// Metrics are now handled by system-monitor iframe

// System Logs
export const logService = {
  // Get system logs
  async getSystemLogs(level?: string): Promise<LogEntry[]> {
    try {
      const params = level ? `?level=${level}` : '';
      const { data } = await api.get<ApiResponse<LogEntry[]>>(`/logs${params}`);
      return data.data || [];
    } catch (error) {
      logger.error('Failed to fetch system logs', error);
      return [];
    }
  },

  // Stream logs (returns EventSource)
  streamLogs(onMessage: (log: LogEntry) => void): EventSource {
    const eventSource = new EventSource('/api/v1/logs/stream');
    
    eventSource.onmessage = (event) => {
      try {
        const log = JSON.parse(event.data) as LogEntry;
        onMessage(log);
      } catch (error) {
        logger.error('Failed to parse log stream message', error);
      }
    };
    
    eventSource.onerror = (error) => {
      logger.error('Log stream error', error);
    };
    
    return eventSource;
  },
};

// System Information
export const systemService = {
  async getSystemInfo(): Promise<any> {
    try {
      const { data } = await api.get('/system/info');
      return data;
    } catch (error) {
      logger.error('Failed to fetch system info', error);
      return null;
    }
  },
};

// Health Check
export const healthService = {
  async checkHealth(): Promise<boolean> {
    try {
      const { data } = await api.get('/health');
      return data.status === 'healthy';
    } catch (error) {
      logger.error('Health check failed', error);
      return false;
    }
  },

  async performHealthCheck(): Promise<ApiResponse> {
    try {
      const { data } = await api.post<ApiResponse>('/health/check');
      return data;
    } catch (error) {
      logger.error('Health check action failed', error);
      return { success: false, error: 'Health check failed' };
    }
  },
};

// Terminal service removed - functionality not needed

export default api;