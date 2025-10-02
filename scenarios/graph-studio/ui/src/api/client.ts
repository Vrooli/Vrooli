import axios, { AxiosInstance } from 'axios'

class ApiClient {
  private client: AxiosInstance
  
  constructor() {
    // Use relative URLs - the vite proxy will handle routing to the correct backend
    this.client = axios.create({
      baseURL: '', // Empty base URL means use relative paths
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })
    
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
  async getStats() {
    return this.client.get('/api/v1/stats')
  }
  
  // Plugins
  async getPlugins(params?: { category?: string }) {
    return this.client.get('/api/v1/plugins', { params })
  }
  
  // Graphs
  async getGraphs(params?: { 
    type?: string
    tag?: string
    limit?: number
    offset?: number 
  }) {
    return this.client.get('/api/v1/graphs', { params })
  }
  
  async getGraph(id: string) {
    return this.client.get(`/api/v1/graphs/${id}`)
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