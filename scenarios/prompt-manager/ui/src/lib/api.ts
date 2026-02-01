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
 *
 * All API responses are validated at runtime using Zod schemas to prevent
 * crashes from mismatched API responses.
 */

import { resolveApiBase, buildApiUrl } from '@vrooli/api-base'
import type { ZodType } from 'zod'
import {
  parseOrThrow,
  SkillSchema,
  SkillArraySchema,
  TagSchema,
  TagArraySchema,
  SkillTestResultSchema,
  SkillTestResultArraySchema,
  UsageResponseSchema,
  RatingResponseSchema,
  HealthResponseSchema,
  DisplayResponseSchema,
  AgentSchema,
  AgentArraySchema,
  EffectiveSkillsResponseSchema,
  AISearchResponseSchema,
  AISearchStatusSchema,
  AIReindexStatusSchema,
  LinkPreviewDataSchema,
  type Skill,
  type CreateSkillRequest,
  type UpdateSkillRequest,
  type Tag,
  type SkillTestRequest,
  type SkillTestResult,
  type UsageResponse,
  type RatingResponse,
  type HealthResponse,
  type DisplayResponse,
  type DisplayFormat,
  type Agent,
  type CreateAgentRequest,
  type UpdateAgentRequest,
  type EffectiveSkillsResponse,
  type AISearchResponse,
  type AISearchStatus,
  type AIReindexStatus,
  type LinkPreviewData,
  type FolderType,
} from '@/lib/schemas'
import type { SearchFilters, Folder } from '@/types'

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
 * AI search request parameters (used for request body, not validated)
 */
export interface AISearchRequest {
  query: string
  limit?: number
  output?: 'results' | 'combined' | 'both'
  format?: DisplayFormat
  renderLimit?: number
}

class ApiClient {
  /**
   * Make an API request with schema validation.
   *
   * @param endpoint - API endpoint path
   * @param options - Fetch options
   * @param schema - Zod schema for response validation
   * @returns Validated response data
   * @throws ValidationError if schema validation fails
   */
  private async request<T>(
    endpoint: string,
    options: RequestInit | undefined,
    schema: ZodType<T>
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

      const data: unknown = await response.json()
      return parseOrThrow(schema, data, endpoint)
    } catch (error) {
      console.error('[prompt-manager api] Fetch error:', error)
      throw error
    }
  }

  /**
   * Make an API request that returns no content (204).
   */
  private async requestVoid(endpoint: string, options: RequestInit): Promise<void> {
    const url = buildApiUrl(endpoint, { baseUrl: API_BASE })
    console.log('[prompt-manager api] Fetching:', url)
    try {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json',
      }
      if (options.headers) {
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
    return this.request<Skill[]>(
      `/skills${queryString ? `?${queryString}` : ''}`,
      undefined,
      SkillArraySchema
    )
  }

  async getSkillsByFolder(folder: FolderType): Promise<Skill[]> {
    return this.getSkills({ folder })
  }

  async getSkill(id: string): Promise<Skill> {
    return this.request<Skill>(
      `/skills/${encodeURIComponent(id)}`,
      undefined,
      SkillSchema
    )
  }

  async createSkill(skill: CreateSkillRequest): Promise<Skill> {
    return this.request<Skill>(
      '/skills',
      {
        method: 'POST',
        body: JSON.stringify(skill),
      },
      SkillSchema
    )
  }

  async updateSkill(id: string, updates: UpdateSkillRequest): Promise<Skill> {
    return this.request<Skill>(
      `/skills/${encodeURIComponent(id)}`,
      {
        method: 'PUT',
        body: JSON.stringify(updates),
      },
      SkillSchema
    )
  }

  async deleteSkill(id: string): Promise<void> {
    await this.requestVoid(`/skills/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }

  // Usage tracking
  async recordUsage(id: string): Promise<UsageResponse> {
    return this.request<UsageResponse>(
      `/skills/${encodeURIComponent(id)}/use`,
      { method: 'POST' },
      UsageResponseSchema
    )
  }

  async setRating(id: string, rating: number, notes?: string): Promise<RatingResponse> {
    return this.request<RatingResponse>(
      `/skills/${encodeURIComponent(id)}/rating`,
      {
        method: 'PUT',
        body: JSON.stringify({ rating, notes }),
      },
      RatingResponseSchema
    )
  }

  // Tags
  async getTags(): Promise<Tag[]> {
    return this.request<Tag[]>('/tags', undefined, TagArraySchema)
  }

  async createTag(tag: Omit<Tag, 'id'>): Promise<Tag> {
    return this.request<Tag>(
      '/tags',
      {
        method: 'POST',
        body: JSON.stringify(tag),
      },
      TagSchema
    )
  }

  // Testing (requires Ollama)
  async testSkill(id: string, request: SkillTestRequest): Promise<SkillTestResult> {
    return this.request<SkillTestResult>(
      `/skills/${encodeURIComponent(id)}/test`,
      {
        method: 'POST',
        body: JSON.stringify(request),
      },
      SkillTestResultSchema
    )
  }

  async getTestHistory(id: string, limit?: number): Promise<SkillTestResult[]> {
    const params = limit ? `?limit=${limit}` : ''
    return this.request<SkillTestResult[]>(
      `/skills/${encodeURIComponent(id)}/test-history${params}`,
      undefined,
      SkillTestResultArraySchema
    )
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
    return this.request<DisplayResponse>(
      '/skills/read',
      {
        method: 'POST',
        body: JSON.stringify({ identifiers, output: 'combined', format }),
      },
      DisplayResponseSchema
    )
  }

  // Health check
  async healthCheck(): Promise<HealthResponse> {
    return this.request<HealthResponse>('/health', undefined, HealthResponseSchema)
  }

  // AI Search - semantic search using Ollama embeddings and Qdrant
  async aiSearch(query: string, limit = 5, options?: Omit<AISearchRequest, 'query' | 'limit'>): Promise<AISearchResponse> {
    return this.request<AISearchResponse>(
      '/search/ai',
      {
        method: 'POST',
        body: JSON.stringify({ query, limit, ...options }),
      },
      AISearchResponseSchema
    )
  }

  async getAISearchStatus(): Promise<AISearchStatus> {
    return this.request<AISearchStatus>('/search/ai/status', undefined, AISearchStatusSchema)
  }

  async reindexAISearch(): Promise<AIReindexStatus> {
    return this.request<AIReindexStatus>(
      '/search/ai/reindex',
      { method: 'POST' },
      AIReindexStatusSchema
    )
  }

  async getAISearchReindexStatus(): Promise<AIReindexStatus> {
    return this.request<AIReindexStatus>('/search/ai/reindex/status', undefined, AIReindexStatusSchema)
  }

  async cancelAISearchReindex(): Promise<AIReindexStatus> {
    return this.request<AIReindexStatus>(
      '/search/ai/reindex/cancel',
      { method: 'POST' },
      AIReindexStatusSchema
    )
  }

  // Agent methods - aligned with api/agents/handlers.go
  async getAgents(): Promise<Agent[]> {
    return this.request<Agent[]>('/agents', undefined, AgentArraySchema)
  }

  async getAgent(id: string): Promise<Agent> {
    return this.request<Agent>(
      `/agents/${encodeURIComponent(id)}`,
      undefined,
      AgentSchema
    )
  }

  async createAgent(agent: CreateAgentRequest): Promise<Agent> {
    return this.request<Agent>(
      '/agents',
      {
        method: 'POST',
        body: JSON.stringify(agent),
      },
      AgentSchema
    )
  }

  async updateAgent(id: string, updates: UpdateAgentRequest): Promise<Agent> {
    return this.request<Agent>(
      `/agents/${encodeURIComponent(id)}`,
      {
        method: 'PUT',
        body: JSON.stringify(updates),
      },
      AgentSchema
    )
  }

  async deleteAgent(id: string): Promise<void> {
    await this.requestVoid(`/agents/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }

  async getEffectiveSkills(agentId: string, teamId?: string): Promise<EffectiveSkillsResponse> {
    const params = teamId ? `?teamId=${encodeURIComponent(teamId)}` : ''
    return this.request<EffectiveSkillsResponse>(
      `/agents/${encodeURIComponent(agentId)}/effective-skills${params}`,
      undefined,
      EffectiveSkillsResponseSchema
    )
  }
}

export const api = new ApiClient()

// -----------------------------------------------------------------------------
// Link Preview API Functions
// -----------------------------------------------------------------------------

/**
 * Fetch OpenGraph metadata preview for a URL.
 * @param url - The URL to fetch preview for
 * @returns Preview data or null if unavailable
 */
export async function fetchLinkPreview(url: string): Promise<LinkPreviewData | null> {
  const apiUrl = buildApiUrl(`/og-metadata?url=${encodeURIComponent(url)}`, {
    baseUrl: API_BASE,
  })

  const res = await fetch(apiUrl, {
    headers: { 'Content-Type': 'application/json' },
    cache: 'no-store',
  })

  if (res.status === 204) {
    // No content - preview unavailable
    return null
  }

  if (!res.ok) {
    throw new Error(`Failed to fetch link preview: ${res.status}`)
  }

  const data: unknown = await res.json()

  // Handle siteName -> site_name mapping before validation
  const normalized = data as Record<string, unknown>
  if ('siteName' in normalized && !('site_name' in normalized)) {
    normalized.site_name = normalized.siteName
    delete normalized.siteName
  }

  return parseOrThrow(LinkPreviewDataSchema, normalized, '/og-metadata')
}

