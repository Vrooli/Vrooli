import type { 
  Campaign, 
  Prompt, 
  PromptTestRequest, 
  PromptTestResponse, 
  SearchFilters,
  ApiConfig 
} from '@/types'

class ApiClient {
  private baseUrl: string = ''
  
  constructor() {
    this.initializeConfig()
  }

  private async initializeConfig(): Promise<void> {
    try {
      const response = await fetch('/api/config')
      const config: ApiConfig = await response.json()
      this.baseUrl = config.apiUrl
    } catch (error) {
      console.error('Failed to fetch API config:', error)
      // Fallback to environment or default
      this.baseUrl = 'http://localhost:15000'
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