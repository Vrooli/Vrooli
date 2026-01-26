/**
 * API client for prompt-manager Go API.
 *
 * Endpoints aligned with api/skills/handlers.go:
 * - GET /api/v1/skills - list skills (with optional filters)
 * - GET /api/v1/skills/sync - sync skills with content
 * - POST /api/v1/skills - create skill
 * - GET /api/v1/skills/{id} - get single skill
 * - PUT /api/v1/skills/{id} - update skill
 * - DELETE /api/v1/skills/{id} - delete skill
 * - POST /api/v1/skills/{id}/use - record usage
 * - PUT /api/v1/skills/{id}/rating - set rating
 * - GET /api/v1/tags - list tags
 * - POST /api/v1/tags - create tag
 * - POST /api/v1/skills/{id}/test - test with Ollama
 * - GET /api/v1/skills/{id}/test-history - get test history
 */

import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import type {
  Skill,
  CreateSkillRequest,
  UpdateSkillRequest,
  Tag,
  SkillTestRequest,
  SkillTestResult,
  SearchFilters,
  HealthResponse,
  Folder,
  FolderType,
} from '@/types'
import type { Member, CreateMemberRequest, UpdateMemberRequest } from '@/types/member'
import type { DisplayFormat, DisplayResponse } from '@/types/world'

// Use @vrooli/api-base for automatic API resolution across all deployment contexts
const API_BASE = resolveApiBase({ appendSuffix: true })
console.log('[prompt-manager api] API_BASE resolved to:', API_BASE)

/**
 * Static folder definitions.
 * The API uses folder-based organization.
 * Folders determine git behavior, not editability:
 * - core: Important skills (git-tracked)
 * - local: Personal skills (gitignored)
 * - drafts: Work in progress skills
 */
export const FOLDERS: Folder[] = [
  {
    id: 'core',
    name: 'Core',
    description: 'Important skills (git-tracked)',
    icon: 'shield',
    skillCount: 0,  // Updated dynamically
  },
  {
    id: 'local',
    name: 'Local',
    description: 'Personal skills (gitignored)',
    icon: 'folder',
    skillCount: 0,
  },
  {
    id: 'drafts',
    name: 'Drafts',
    description: 'Work in progress skills',
    icon: 'edit',
    skillCount: 0,
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

/**
 * AI search types
 */
export interface AISearchResult {
  id: string
  name: string
  description: string
  folder: string
  tags: string[]
  modes: string[]
  score: number
  scorePercent: number
}

export interface AISearchResponse {
  results: AISearchResult[]
  total: number
  query: string
  method: 'ai' | 'text'
}

export interface AISearchRequest {
  query: string
  limit?: number
}

export interface AISearchStatus {
  available: boolean
  ollama: boolean
  qdrant: boolean
  indexedCount: number
  message?: string
}

export interface AIReindexStatus {
  running: boolean
  startedAt?: string
  finishedAt?: string
  indexed: number
  skipped: number
  errors: number
  total: number
  message?: string
  canceled?: boolean
  error?: string
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

  // Folder methods (computed from skills, not a separate API)
  async getFolders(): Promise<Folder[]> {
    // Get all skills and compute folder counts
    const skills = await this.getSkills()
    const counts: Record<FolderType, number> = { core: 0, local: 0, drafts: 0 }

    for (const skill of skills) {
      if (skill.folder in counts) {
        counts[skill.folder]++
      }
    }

    return FOLDERS.map(folder => ({
      ...folder,
      skillCount: counts[folder.id],
    }))
  }

  // Skill methods - aligned with api/skills/handlers.go
  async getSkills(filters?: SearchFilters): Promise<Skill[]> {
    const params = new URLSearchParams()
    if (filters?.tag) params.append('tag', filters.tag)
    if (filters?.folder) params.append('folder', filters.folder)
    if (filters?.modes) {
      for (const mode of filters.modes) {
        params.append('modes', mode)
      }
    }

    const queryString = params.toString()
    return this.request<Skill[]>(`/skills${queryString ? `?${queryString}` : ''}`)
  }

  async getSkillsByFolder(folder: FolderType): Promise<Skill[]> {
    return this.getSkills({ folder })
  }

  async getSkill(id: string): Promise<Skill> {
    return this.request<Skill>(`/skills/${encodeURIComponent(id)}`)
  }

  async createSkill(skill: CreateSkillRequest): Promise<Skill> {
    return this.request<Skill>('/skills', {
      method: 'POST',
      body: JSON.stringify(skill),
    })
  }

  async updateSkill(id: string, updates: UpdateSkillRequest): Promise<Skill> {
    return this.request<Skill>(`/skills/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    })
  }

  async deleteSkill(id: string): Promise<void> {
    await this.request<Record<string, never>>(`/skills/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }

  // Usage tracking
  async recordUsage(id: string): Promise<UsageResponse> {
    return this.request<UsageResponse>(`/skills/${encodeURIComponent(id)}/use`, {
      method: 'POST',
    })
  }

  async setRating(id: string, rating: number, notes?: string): Promise<RatingResponse> {
    return this.request<RatingResponse>(`/skills/${encodeURIComponent(id)}/rating`, {
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
  async testSkill(id: string, request: SkillTestRequest): Promise<SkillTestResult> {
    return this.request<SkillTestResult>(`/skills/${encodeURIComponent(id)}/test`, {
      method: 'POST',
      body: JSON.stringify(request),
    })
  }

  async getTestHistory(id: string, limit?: number): Promise<SkillTestResult[]> {
    const params = limit ? `?limit=${limit}` : ''
    return this.request<SkillTestResult[]>(`/skills/${encodeURIComponent(id)}/test-history${params}`)
  }

  // Search - client-side filtering since no dedicated search endpoint
  async searchSkills(query: string, filters?: SearchFilters): Promise<Skill[]> {
    const allSkills = await this.getSkills(filters)
    const lowerQuery = query.toLowerCase()

    return allSkills.filter(skill =>
      skill.name.toLowerCase().includes(lowerQuery) ||
      skill.description.toLowerCase().includes(lowerQuery) ||
      skill.content.toLowerCase().includes(lowerQuery) ||
      skill.tags.some(tag => tag.toLowerCase().includes(lowerQuery))
    )
  }

  // Display skills
  async displaySkills(identifiers: string[], format: DisplayFormat = 'xml'): Promise<DisplayResponse> {
    return this.request<DisplayResponse>('/skills/display', {
      method: 'POST',
      body: JSON.stringify({ identifiers, format }),
    })
  }

  // Health check
  async healthCheck(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health')
  }

  // AI Search - semantic search using Ollama embeddings and Qdrant
  async aiSearch(query: string, limit = 5): Promise<AISearchResponse> {
    return this.request<AISearchResponse>('/search/ai', {
      method: 'POST',
      body: JSON.stringify({ query, limit }),
    })
  }

  async getAISearchStatus(): Promise<AISearchStatus> {
    return this.request<AISearchStatus>('/search/ai/status')
  }

  async reindexAISearch(): Promise<AIReindexStatus> {
    return this.request<AIReindexStatus>('/search/ai/reindex', {
      method: 'POST',
    })
  }

  async getAISearchReindexStatus(): Promise<AIReindexStatus> {
    return this.request<AIReindexStatus>('/search/ai/reindex/status')
  }

  async cancelAISearchReindex(): Promise<AIReindexStatus> {
    return this.request<AIReindexStatus>('/search/ai/reindex/cancel', {
      method: 'POST',
    })
  }

  // Member methods - aligned with api/members/handlers.go
  async getMembers(): Promise<Member[]> {
    return this.request<Member[]>('/members')
  }

  async getMember(id: string): Promise<Member> {
    return this.request<Member>(`/members/${encodeURIComponent(id)}`)
  }

  async createMember(member: CreateMemberRequest): Promise<Member> {
    return this.request<Member>('/members', {
      method: 'POST',
      body: JSON.stringify(member),
    })
  }

  async updateMember(id: string, updates: UpdateMemberRequest): Promise<Member> {
    return this.request<Member>(`/members/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    })
  }

  async deleteMember(id: string): Promise<void> {
    await this.request<Record<string, never>>(`/members/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }
}

export const api = new ApiClient()
