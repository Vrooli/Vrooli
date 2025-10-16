import axios, { AxiosInstance } from 'axios'
import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: unknown
  }
}

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '16000'

const API_BASE_URL = resolveApiBase({
  explicitUrl: import.meta.env.VITE_API_URL as string | undefined,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: true,
})

export interface StatsSnapshot {
  totalGraphs: number
  conversionsToday: number
  activeUsers: number
}

export interface PluginSummary {
  id: string
  name: string
  description: string
  category?: string
  enabled?: boolean
  formats: string[]
  metadata?: Record<string, any>
  version?: string
  owner?: string
}

export interface PluginCollection {
  data: PluginSummary[]
  total?: number
}

export interface GraphSummary {
  id: string
  name: string
  description?: string
  type: string
  updated_at: string
  created_at: string
  tags?: string[]
  version?: string
  summary?: string
  status?: string
}

export interface GraphCollection {
  data: GraphSummary[]
  total?: number
}

class ApiClient {
  private client: AxiosInstance
  
  constructor() {
    // Use relative URLs - the vite proxy will handle routing to the correct backend
    this.client = axios.create({
      baseURL: API_BASE_URL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    const previewTokenFromEnv = (import.meta.env.VITE_PREVIEW_ACCESS_TOKEN || '').trim()
    const previewToken = previewTokenFromEnv || (import.meta.env.PROD ? 'graph-studio-preview-token' : '')

    if (previewToken) {
      this.client.defaults.headers.common['X-Preview-Token'] = previewToken
    }

    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        // Add auth token if available
        const token = localStorage.getItem('auth_token')
        if (token) {
          config.headers.Authorization = `Bearer ${token}`
        }
        
        // Add user ID if available
        const userId = localStorage.getItem('user_id')
        if (userId) {
          config.headers['X-User-ID'] = userId
        }
        
        return config
      },
      (error) => Promise.reject(error)
    )
    
    // Response interceptor
    this.client.interceptors.response.use(
      (response) => response.data,
      (error) => {
        if (error.response?.status === 401) {
          // Handle unauthorized
          localStorage.removeItem('auth_token')
          window.location.href = '/login'
        }
        return Promise.reject(error)
      }
    )
  }
  
  // Health check
  async healthCheck() {
    return this.client.get('/health')
  }
  
  // Stats
  async getStats(): Promise<StatsSnapshot> {
    return this.client.get<StatsSnapshot>('/api/v1/stats') as unknown as Promise<StatsSnapshot>
  }
  
  // Plugins
  async getPlugins(params?: { category?: string }): Promise<PluginCollection> {
    return this.client.get<PluginCollection>('/api/v1/plugins', { params }) as unknown as Promise<PluginCollection>
  }
  
  // Graphs
  async getGraphs(params?: { 
    type?: string
    tag?: string
    limit?: number
    offset?: number 
  }): Promise<GraphCollection> {
    return this.client.get<GraphCollection>('/api/v1/graphs', { params }) as unknown as Promise<GraphCollection>
  }
  
  async getGraph(id: string): Promise<GraphSummary> {
    return this.client.get<GraphSummary>(`/api/v1/graphs/${id}`) as unknown as Promise<GraphSummary>
  }
  
  async createGraph(data: {
    name: string
    type: string
    description?: string
    data?: any
    metadata?: Record<string, any>
    tags?: string[]
  }) {
    return this.client.post('/api/v1/graphs', data)
  }
  
  async updateGraph(id: string, data: {
    name?: string
    description?: string
    data?: any
    metadata?: Record<string, any>
    tags?: string[]
  }) {
    return this.client.put(`/api/v1/graphs/${id}`, data)
  }
  
  async deleteGraph(id: string) {
    return this.client.delete(`/api/v1/graphs/${id}`)
  }
  
  // Graph operations
  async validateGraph(id: string) {
    return this.client.post(`/api/v1/graphs/${id}/validate`)
  }
  
  async convertGraph(id: string, targetFormat: string, options?: any) {
    return this.client.post(`/api/v1/graphs/${id}/convert`, {
      target_format: targetFormat,
      options,
    })
  }
  
  async renderGraph(id: string, format: 'svg' | 'png' | 'html' = 'svg', options?: any) {
    return this.client.post(`/api/v1/graphs/${id}/render`, {
      format,
      options,
    })
  }
}

export const api = new ApiClient()
