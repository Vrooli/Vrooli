import type {
  Campaign,
  Prompt,
  PromptTestRequest,
  PromptTestResponse,
  SearchFilters,
  ApiConfig
} from '@/types'

declare global {
  interface Window {
    __APP_MONITOR_PROXY_INFO__?: unknown
  }
}

const DEFAULT_API_PORT = (import.meta.env.VITE_API_PORT as string | undefined)?.trim() || '15000'
const LOOPBACK_DEFAULT = `http://127.0.0.1:${DEFAULT_API_PORT}`

const isLocalHostname = (hostname?: string | null) => {
  if (!hostname) return false
  const normalized = hostname.toLowerCase()
  return (
    normalized === 'localhost' ||
    normalized === '127.0.0.1' ||
    normalized === '0.0.0.0' ||
    normalized === '::1' ||
    normalized === '[::1]'
  )
}

const isLikelyProxiedPath = (pathname?: string | null) => {
  if (!pathname) return false
  return pathname.includes('/apps/') && pathname.includes('/proxy/')
}

const normalizeBaseUrl = (value: string) => value.replace(/\/+$/, '')

const resolveSmartBaseUrl = (candidate?: string | null): string => {
  const envUrl = (import.meta.env.VITE_API_URL as string | undefined)?.trim()
  const explicit = candidate?.trim() || envUrl

  if (explicit) {
    const normalizedExplicit = normalizeBaseUrl(explicit)

    if (typeof window !== 'undefined') {
      const { hostname, origin, pathname } = window.location
      const hasProxyBootstrap = typeof window.__APP_MONITOR_PROXY_INFO__ !== 'undefined'
      const proxiedPath = isLikelyProxiedPath(pathname)
      const isRemote = !isLocalHostname(hostname)

      if (isRemote && !hasProxyBootstrap && !proxiedPath && /localhost|127\.0\.0\.1/i.test(normalizedExplicit)) {
        return normalizeBaseUrl(origin)
      }

      if (hasProxyBootstrap || proxiedPath) {
        return normalizedExplicit
      }
    }

    return normalizedExplicit
  }

  if (typeof window !== 'undefined') {
    const { hostname, origin, pathname } = window.location
    const hasProxyBootstrap = typeof window.__APP_MONITOR_PROXY_INFO__ !== 'undefined'
    const proxiedPath = isLikelyProxiedPath(pathname)

    if (!hasProxyBootstrap && !proxiedPath && !isLocalHostname(hostname)) {
      return normalizeBaseUrl(origin)
    }

    if (hasProxyBootstrap || proxiedPath) {
      return LOOPBACK_DEFAULT
    }
  }

  return LOOPBACK_DEFAULT
}

class ApiClient {
  private baseUrl: string = ''
  
  constructor() {
    this.initializeConfig()
  }

  private async initializeConfig(): Promise<void> {
    try {
      const response = await fetch('/api/config')
      const config: ApiConfig = await response.json()
      this.baseUrl = resolveSmartBaseUrl(config.apiUrl)
    } catch (error) {
      console.error('Failed to fetch API config:', error)
      this.baseUrl = resolveSmartBaseUrl(null)
    }
  }

  private async request<T>(
    endpoint: string, 
    options?: RequestInit
  ): Promise<T> {
    if (!this.baseUrl) {
      await this.initializeConfig()
    }

    const url = `${this.baseUrl}${endpoint}`
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    })

    if (!response.ok) {
      throw new Error(`API error: ${response.status} ${response.statusText}`)
    }

    // Handle 204 No Content responses
    if (response.status === 204) {
      return {} as T
    }

    return response.json()
  }

  // Campaign methods
  async getCampaigns(): Promise<Campaign[]> {
    return this.request<Campaign[]>('/api/v1/campaigns')
  }

  async getCampaign(id: string): Promise<Campaign> {
    return this.request<Campaign>(`/api/v1/campaigns/${id}`)
  }

  async createCampaign(campaign: Omit<Campaign, 'id' | 'created_at' | 'updated_at' | 'prompt_count'>): Promise<Campaign> {
    return this.request<Campaign>('/api/v1/campaigns', {
      method: 'POST',
      body: JSON.stringify(campaign),
    })
  }

  async updateCampaign(id: string, campaign: Partial<Campaign>): Promise<Campaign> {
    return this.request<Campaign>(`/api/v1/campaigns/${id}`, {
      method: 'PUT',
      body: JSON.stringify(campaign),
    })
  }

  async deleteCampaign(id: string): Promise<void> {
    return this.request<void>(`/api/v1/campaigns/${id}`, {
      method: 'DELETE',
    })
  }

  // Prompt methods
  async getPrompts(filters?: SearchFilters): Promise<Prompt[]> {
    const params = new URLSearchParams()
    if (filters?.campaign_id) params.append('campaign_id', filters.campaign_id)
    if (filters?.query) params.append('q', filters.query)
    if (filters?.is_favorite !== undefined) params.append('is_favorite', String(filters.is_favorite))
    if (filters?.limit) params.append('limit', String(filters.limit))
    if (filters?.offset) params.append('offset', String(filters.offset))
    if (filters?.tags?.length) {
      filters.tags.forEach(tag => params.append('tags', tag))
    }
    
    const queryString = params.toString()
    return this.request<Prompt[]>(`/api/v1/prompts${queryString ? `?${queryString}` : ''}`)
  }

  async getCampaignPrompts(campaignId: string): Promise<Prompt[]> {
    return this.request<Prompt[]>(`/api/v1/campaigns/${campaignId}/prompts`)
  }

  async getPrompt(id: string): Promise<Prompt> {
    return this.request<Prompt>(`/api/v1/prompts/${id}`)
  }

  async createPrompt(prompt: Omit<Prompt, 'id' | 'created_at' | 'updated_at' | 'usage_count'>): Promise<Prompt> {
    return this.request<Prompt>('/api/v1/prompts', {
      method: 'POST',
      body: JSON.stringify(prompt),
    })
  }

  async updatePrompt(id: string, prompt: Partial<Prompt>): Promise<Prompt> {
    return this.request<Prompt>(`/api/v1/prompts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(prompt),
    })
  }

  async deletePrompt(id: string): Promise<void> {
    return this.request<void>(`/api/v1/prompts/${id}`, {
      method: 'DELETE',
    })
  }

  async testPrompt(id: string, request: PromptTestRequest): Promise<PromptTestResponse> {
    return this.request<PromptTestResponse>(`/api/v1/prompts/${id}/test`, {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async recordPromptUsage(id: string): Promise<void> {
    return this.request<void>(`/api/v1/prompts/${id}/use`, {
      method: 'POST',
    })
  }

  // Search methods
  async searchPrompts(query: string, filters?: SearchFilters): Promise<Prompt[]> {
    return this.request<Prompt[]>('/api/v1/search/prompts', {
      method: 'POST',
      body: JSON.stringify({ query, ...filters }),
    })
  }

  async searchTags(query: string): Promise<string[]> {
    return this.request<string[]>(`/api/v1/search/tags?q=${encodeURIComponent(query)}`)
  }

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    return this.request<{ status: string }>('/health')
  }
}

export const api = new ApiClient()
