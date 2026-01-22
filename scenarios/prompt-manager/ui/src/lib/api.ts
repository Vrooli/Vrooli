/**
 * API client for prompt-manager Go API.
 *
 * Endpoints aligned with api/prompts/handlers.go:
 * - GET /api/v1/prompts - list prompts (with optional filters)
 * - GET /api/v1/prompts/sync - sync prompts with content
 * - POST /api/v1/prompts - create prompt
 * - GET /api/v1/prompts/{id} - get single prompt
 * - PUT /api/v1/prompts/{id} - update prompt
 * - DELETE /api/v1/prompts/{id} - delete prompt
 * - POST /api/v1/prompts/{id}/use - record usage
 * - PUT /api/v1/prompts/{id}/rating - set rating
 * - GET /api/v1/tags - list tags
 * - POST /api/v1/tags - create tag
 * - POST /api/v1/prompts/{id}/test - test with Ollama
 * - GET /api/v1/prompts/{id}/test-history - get test history
 */

import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import type {
  Prompt,
  CreatePromptRequest,
  UpdatePromptRequest,
  Tag,
  PromptTestRequest,
  PromptTestResult,
  SearchFilters,
  HealthResponse,
  Folder,
  FolderType,
} from '@/types'

// Use @vrooli/api-base for automatic API resolution across all deployment contexts
const API_BASE = resolveApiBase({ appendSuffix: true })
console.log('[prompt-manager api] API_BASE resolved to:', API_BASE)

/**
 * Static folder definitions.
 * The API uses folder-based organization, not campaigns.
 */
export const FOLDERS: Folder[] = [
  {
    id: 'core',
    name: 'Core',
    description: 'System prompts and steering templates (read-only)',
    icon: 'shield',
    readonly: true,
    promptCount: 0,  // Updated dynamically
  },
  {
    id: 'local',
    name: 'Local',
    description: 'Your saved prompts',
    icon: 'folder',
    readonly: false,
    promptCount: 0,
  },
  {
    id: 'drafts',
    name: 'Drafts',
    description: 'Work in progress prompts',
    icon: 'edit',
    readonly: false,
    promptCount: 0,
  },
]

/**
 * API response type for usage recording
 */
interface UsageResponse {
  status: string
  usageCount: number
  lastUsed: string
}

/**
 * API response type for rating
 */
interface RatingResponse {
  status: string
  rating: number
}

class ApiClient {
  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const url = buildApiUrl(endpoint, { baseUrl: API_BASE })
    console.log('[prompt-manager api] Fetching:', url)
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (options?.headers) {
        const optHeaders = options.headers as Record<string, string>
        Object.assign(headers, optHeaders)
      }
      const response = await fetch(url, {
        ...options,
        headers,
      })

      console.log('[prompt-manager api] Response status:', response.status, response.statusText)

      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Unknown error')
        throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`)
      }

      // Handle 204 No Content responses
      if (response.status === 204) {
        return {} as T
      }

      return await response.json() as T
    } catch (error) {
      console.error('[prompt-manager api] Fetch error:', error)
      throw error
    }
  }

  // Folder methods (computed from prompts, not a separate API)
  async getFolders(): Promise<Folder[]> {
    // Get all prompts and compute folder counts
    const prompts = await this.getPrompts()
    const counts: Record<FolderType, number> = { core: 0, local: 0, drafts: 0 }

    for (const prompt of prompts) {
      if (prompt.folder in counts) {
        counts[prompt.folder]++
      }
    }

    return FOLDERS.map(folder => ({
      ...folder,
      promptCount: counts[folder.id],
    }))
  }

  // Prompt methods - aligned with api/prompts/handlers.go
  async getPrompts(filters?: SearchFilters): Promise<Prompt[]> {
    const params = new URLSearchParams()
    if (filters?.tag) params.append('tag', filters.tag)
    if (filters?.folder) params.append('folder', filters.folder)
    if (filters?.modes) {
      for (const mode of filters.modes) {
        params.append('modes', mode)
      }
    }

    const queryString = params.toString()
    return this.request<Prompt[]>(`/prompts${queryString ? `?${queryString}` : ''}`)
  }

  async getPromptsByFolder(folder: FolderType): Promise<Prompt[]> {
    return this.getPrompts({ folder })
  }

  async getPrompt(id: string): Promise<Prompt> {
    return this.request<Prompt>(`/prompts/${encodeURIComponent(id)}`)
  }

  async createPrompt(prompt: CreatePromptRequest): Promise<Prompt> {
    return this.request<Prompt>('/prompts', {
      method: 'POST',
      body: JSON.stringify(prompt),
    })
  }

  async updatePrompt(id: string, updates: UpdatePromptRequest): Promise<Prompt> {
    return this.request<Prompt>(`/prompts/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    })
  }

  async deletePrompt(id: string): Promise<void> {
    await this.request<Record<string, never>>(`/prompts/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }

  // Usage tracking
  async recordUsage(id: string): Promise<UsageResponse> {
    return this.request<UsageResponse>(`/prompts/${encodeURIComponent(id)}/use`, {
      method: 'POST',
    })
  }

  async setRating(id: string, rating: number, notes?: string): Promise<RatingResponse> {
    return this.request<RatingResponse>(`/prompts/${encodeURIComponent(id)}/rating`, {
      method: 'PUT',
      body: JSON.stringify({ rating, notes }),
    })
  }

  // Tags
  async getTags(): Promise<Tag[]> {
    return this.request<Tag[]>('/tags')
  }

  async createTag(tag: Omit<Tag, 'id'>): Promise<Tag> {
    return this.request<Tag>('/tags', {
      method: 'POST',
      body: JSON.stringify(tag),
    })
  }

  // Testing (requires Ollama)
  async testPrompt(id: string, request: PromptTestRequest): Promise<PromptTestResult> {
    return this.request<PromptTestResult>(`/prompts/${encodeURIComponent(id)}/test`, {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async getTestHistory(id: string, limit?: number): Promise<PromptTestResult[]> {
    const params = limit ? `?limit=${limit}` : ''
    return this.request<PromptTestResult[]>(`/prompts/${encodeURIComponent(id)}/test-history${params}`)
  }

  // Search - client-side filtering since no dedicated search endpoint
  async searchPrompts(query: string, filters?: SearchFilters): Promise<Prompt[]> {
    const allPrompts = await this.getPrompts(filters)
    const lowerQuery = query.toLowerCase()

    return allPrompts.filter(prompt =>
      prompt.name.toLowerCase().includes(lowerQuery) ||
      prompt.description.toLowerCase().includes(lowerQuery) ||
      prompt.content.toLowerCase().includes(lowerQuery) ||
      prompt.tags.some(tag => tag.toLowerCase().includes(lowerQuery))
    )
  }

  // Health check
  async healthCheck(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health')
  }
}

export const api = new ApiClient()
