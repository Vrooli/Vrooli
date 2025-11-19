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
import {
  clearStoredAuth,
  dispatchAuthRequiredEvent,
  extractUserIdentifiers,
  getStoredAuthToken,
  getStoredAuthUser,
  resolveAuthenticatorLoginUrl
} from '@/utils/auth'

const explicitApiUrlEnv = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim()
const explicitApiUrl = explicitApiUrlEnv ? explicitApiUrlEnv : undefined

const API_BASE_URL = resolveApiBase({
  appendSuffix: true,
  explicitUrl: explicitApiUrl,
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
      const token = getStoredAuthToken()
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }

      const storedUser = getStoredAuthUser()
      if (storedUser) {
        const { userId, email } = extractUserIdentifiers(storedUser)
        if (userId) {
          config.headers['X-User-ID'] = userId
          config.headers['X-Auth-User-ID'] = userId
        }
        if (email) {
          config.headers['X-User-Email'] = email
        }
      }
      return config
    })

    // Handle auth errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          clearStoredAuth()
          const loginUrl = resolveAuthenticatorLoginUrl()
          dispatchAuthRequiredEvent(loginUrl)
        }
        return Promise.reject(error)
      }
    )
  }

  // Auth
  validateToken = async (): Promise<User> => {
    const response = await this.client.get('/auth/validate')
    return response.data
  }

  login = async (credentials: { email: string; password: string }): Promise<{
    success: boolean
    token: string
    refresh_token?: string
    refreshToken?: string
    user?: User
    message?: string
  }> => {
    const response = await this.client.post('/auth/login', credentials)
    return response.data
  }

  // Events
  createEvent = async (data: CreateEventRequest): Promise<Event> => {
    const response = await this.client.post('/events', data)
    return response.data
  }

  getEvents = async (params?: {
    startDate?: string
    endDate?: string
    eventType?: string
    status?: string
    limit?: number
  }): Promise<{ events: Event[]; totalCount: number; hasMore: boolean }> => {
    const response = await this.client.get('/events', { params })
    return response.data
  }

  getEvent = async (id: string): Promise<Event> => {
    const response = await this.client.get(`/events/${id}`)
    return response.data
  }

  updateEvent = async (id: string, data: Partial<CreateEventRequest>): Promise<Event> => {
    const response = await this.client.put(`/events/${id}`, data)
    return response.data
  }

  deleteEvent = async (id: string): Promise<void> => {
    await this.client.delete(`/events/${id}`)
  }

  // Chat & NLP
  sendChatMessage = async (data: ChatRequest): Promise<ChatResponse> => {
    const response = await this.client.post('/schedule/chat', data)
    return response.data
  }

  // Schedule Optimization
  optimizeSchedule = async (data: OptimizationRequest): Promise<OptimizationResponse> => {
    const response = await this.client.post('/schedule/optimize', data)
    return response.data
  }

  // Reminders
  processReminders = async (): Promise<{ status: string; message: string }> => {
    const response = await this.client.post('/reminders/process')
    return response.data
  }

  // Health check
  healthCheck = async (): Promise<{
    status: string
    timestamp: string
    version: string
    services: Record<string, string>
  }> => {
    const response = await this.client.get('/health')
    return response.data
  }
}

export const api = new CalendarAPI()
