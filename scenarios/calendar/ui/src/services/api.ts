import axios, { AxiosInstance } from 'axios'
import type {
  Event,
  CreateEventRequest,
  ChatRequest,
  ChatResponse,
  OptimizationRequest,
  OptimizationResponse,
  User
} from '@/types'
import { resolveApiBase } from '@vrooli/api-base'

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '18000'

const API_BASE_URL = resolveApiBase({
  explicitUrl: import.meta.env.VITE_API_BASE_URL as string | undefined,
  defaultPort: DEFAULT_API_PORT,
  appendSuffix: true,
})

class CalendarAPI {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json'
      }
    })

    // Add auth token to requests
    this.client.interceptors.request.use((config) => {
      const token = localStorage.getItem('auth_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    })

    // Handle auth errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('auth_token')
          window.location.href = '/login'
        }
        return Promise.reject(error)
      }
    )
  }

  // Auth
  async validateToken(): Promise<User> {
    const response = await this.client.get('/auth/validate')
    return response.data
  }

  // Events
  async createEvent(data: CreateEventRequest): Promise<Event> {
    const response = await this.client.post('/events', data)
    return response.data
  }

  async getEvents(params?: {
    startDate?: string
    endDate?: string
    eventType?: string
    status?: string
    limit?: number
  }): Promise<{ events: Event[]; totalCount: number; hasMore: boolean }> {
    const response = await this.client.get('/events', { params })
    return response.data
  }

  async getEvent(id: string): Promise<Event> {
    const response = await this.client.get(`/events/${id}`)
    return response.data
  }

  async updateEvent(id: string, data: Partial<CreateEventRequest>): Promise<Event> {
    const response = await this.client.put(`/events/${id}`, data)
    return response.data
  }

  async deleteEvent(id: string): Promise<void> {
    await this.client.delete(`/events/${id}`)
  }

  // Chat & NLP
  async sendChatMessage(data: ChatRequest): Promise<ChatResponse> {
    const response = await this.client.post('/schedule/chat', data)
    return response.data
  }

  // Schedule Optimization
  async optimizeSchedule(data: OptimizationRequest): Promise<OptimizationResponse> {
    const response = await this.client.post('/schedule/optimize', data)
    return response.data
  }

  // Reminders
  async processReminders(): Promise<{ status: string; message: string }> {
    const response = await this.client.post('/reminders/process')
    return response.data
  }

  // Health check
  async healthCheck(): Promise<{
    status: string
    timestamp: string
    version: string
    services: Record<string, string>
  }> {
    const response = await this.client.get('/health')
    return response.data
  }
}

export const api = new CalendarAPI()
