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
 * - GET /api/v1/search/skills/content - content search
 * - GET /api/v1/agent-file-templates - list agent file templates
 * - POST /api/v1/prompt-preview - preview constructed prompts
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
  GraphResponseSchema,
  NodeDetailResponseSchema,
  RegenerateResponseSchema,
  NodeListResponseSchema,
  PopularityResponseSchema,
  CircularRefResponseSchema,
  GraphHealthResponseSchema,
  AgentSchema,
  AgentArraySchema,
  SoulResponseSchema,
  AgentFileListResponseSchema,
  AgentFileContentResponseSchema,
  AgentFileTemplateListResponseSchema,
  PromptPreviewResponseSchema,
  AgentTeamsResponseSchema,
  AISearchResponseSchema,
  AISearchStatusSchema,
  AIReindexStatusSchema,
  ContentSearchResponseSchema,
  LinkPreviewDataSchema,
  TeamArraySchema,
  TeamDetailsSchema,
  TeamRoleSchema,
  TeamMemberSchema,
  TeamSharedFileListResponseSchema,
  TeamSharedFileContentResponseSchema,
  AvailableCCTeamSchema,
  ExportCCResponseSchema,
  ExclusiveMembersResponseSchema,
  WorldScaleConfigSchema,
  WorldSeatsConfigSchema,
  type WorldScaleConfig,
  type WorldSeatsConfig,
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
  type GraphResponse,
  type NodeDetailResponse,
  type RegenerateResponse,
  type NodeListResponse,
  type PopularityResponse,
  type CircularRefResponse,
  type GraphHealthResponse,
  type NodeHealthResponse,
  type Agent,
  type CreateAgentRequest,
  type UpdateAgentRequest,
  type SoulResponse,
  type AgentFileListResponse,
  type AgentFileContentResponse,
  type AgentFileCreateRequest,
  type AgentFileRenameRequest,
  type AgentFileTemplateListResponse,
  type PromptPreviewResponse,
  type AgentTeamsResponse,
  type AISearchResponse,
  type AISearchStatus,
  type AIReindexStatus,
  type ContentSearchResponse,
  type LinkPreviewData,
  type FolderType,
  type Team,
  type TeamDetails,
  type TeamRole,
  type TeamMember,
  type CreateTeamRequest,
  type UpdateTeamRequest,
  type AddMemberRequest,
  type UpdateMemberRequest,
  type TeamSharedFileListResponse,
  type TeamSharedFileContentResponse,
  type TeamSharedFileCreateRequest,
  type TeamSharedFileRenameRequest,
  type AvailableCCTeam,
  type ExportCCResponse,
  type ExclusiveMembersResponse,
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

/**
 * Content search request parameters.
 */
export interface ContentSearchRequest {
  query: string
  tags?: string[]
  folders?: string[]
  caseSensitive?: boolean
  wholeWord?: boolean
  regex?: boolean
  limit?: number
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

  // Content Search - line-level content matches
  async searchSkillContent(request: ContentSearchRequest): Promise<ContentSearchResponse> {
    const params = new URLSearchParams()
    params.set('q', request.query)
    if (request.tags) {
      for (const tag of request.tags) {
        params.append('tag', tag)
      }
    }
    if (request.folders) {
      for (const folder of request.folders) {
        params.append('folder', folder)
      }
    }
    if (request.caseSensitive) params.set('caseSensitive', 'true')
    if (request.wholeWord) params.set('wholeWord', 'true')
    if (request.regex) params.set('regex', 'true')
    if (request.limit) params.set('limit', request.limit.toString())

    const queryString = params.toString()
    return this.request<ContentSearchResponse>(
      `/search/skills/content${queryString ? `?${queryString}` : ''}`,
      undefined,
      ContentSearchResponseSchema
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

  async getAgentSoul(id: string): Promise<SoulResponse> {
    return this.request<SoulResponse>(
      `/agents/${encodeURIComponent(id)}/soul`,
      undefined,
      SoulResponseSchema
    )
  }

  async setAgentSoul(id: string, content: string): Promise<SoulResponse> {
    return this.request<SoulResponse>(
      `/agents/${encodeURIComponent(id)}/soul`,
      {
        method: 'PUT',
        body: JSON.stringify({ content }),
      },
      SoulResponseSchema
    )
  }

  async listAgentFiles(id: string): Promise<AgentFileListResponse> {
    return this.request<AgentFileListResponse>(
      `/agents/${encodeURIComponent(id)}/files`,
      undefined,
      AgentFileListResponseSchema
    )
  }

  async getAgentFileTemplates(): Promise<AgentFileTemplateListResponse> {
    return this.request<AgentFileTemplateListResponse>(
      '/agent-file-templates',
      undefined,
      AgentFileTemplateListResponseSchema
    )
  }

  async getAgentFileContent(id: string, path: string): Promise<AgentFileContentResponse> {
    const params = `?path=${encodeURIComponent(path)}`
    return this.request<AgentFileContentResponse>(
      `/agents/${encodeURIComponent(id)}/files/content${params}`,
      undefined,
      AgentFileContentResponseSchema
    )
  }

  async setAgentFileContent(id: string, path: string, content: string): Promise<void> {
    const params = `?path=${encodeURIComponent(path)}`
    await this.requestVoid(
      `/agents/${encodeURIComponent(id)}/files/content${params}`,
      {
        method: 'PUT',
        body: JSON.stringify({ content }),
      }
    )
  }

  async createAgentFile(id: string, request: AgentFileCreateRequest): Promise<void> {
    await this.requestVoid(
      `/agents/${encodeURIComponent(id)}/files`,
      {
        method: 'POST',
        body: JSON.stringify(request),
      }
    )
  }

  async renameAgentFile(id: string, request: AgentFileRenameRequest): Promise<void> {
    await this.requestVoid(
      `/agents/${encodeURIComponent(id)}/files/rename`,
      {
        method: 'POST',
        body: JSON.stringify(request),
      }
    )
  }

  async deleteAgentFile(id: string, path: string): Promise<void> {
    const params = `?path=${encodeURIComponent(path)}`
    await this.requestVoid(
      `/agents/${encodeURIComponent(id)}/files${params}`,
      {
        method: 'DELETE',
      }
    )
  }

  async previewPrompt(agentId: string, teamId?: string): Promise<PromptPreviewResponse> {
    return this.request<PromptPreviewResponse>(
      '/prompt-preview',
      {
        method: 'POST',
        body: JSON.stringify({ agentId, teamId }),
      },
      PromptPreviewResponseSchema
    )
  }

  async deleteAgent(id: string): Promise<void> {
    await this.requestVoid(`/agents/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }

  async getAgentTeams(id: string): Promise<AgentTeamsResponse> {
    return this.request<AgentTeamsResponse>(
      `/agents/${encodeURIComponent(id)}/teams`,
      undefined,
      AgentTeamsResponseSchema
    )
  }

  // Team methods - aligned with api/teams/handlers.go
  async getTeams(): Promise<Team[]> {
    return this.request<Team[]>('/teams', undefined, TeamArraySchema)
  }

  async getTeam(id: string): Promise<TeamDetails> {
    return this.request<TeamDetails>(
      `/teams/${encodeURIComponent(id)}`,
      undefined,
      TeamDetailsSchema
    )
  }

  async createTeam(team: CreateTeamRequest): Promise<TeamDetails> {
    return this.request<TeamDetails>(
      '/teams',
      {
        method: 'POST',
        body: JSON.stringify(team),
      },
      TeamDetailsSchema
    )
  }

  async updateTeam(id: string, updates: UpdateTeamRequest): Promise<TeamDetails> {
    return this.request<TeamDetails>(
      `/teams/${encodeURIComponent(id)}`,
      {
        method: 'PUT',
        body: JSON.stringify(updates),
      },
      TeamDetailsSchema
    )
  }

  async deleteTeam(id: string): Promise<void> {
    await this.requestVoid(`/teams/${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
  }

  async getTeamExclusiveMembers(teamId: string): Promise<ExclusiveMembersResponse> {
    return this.request<ExclusiveMembersResponse>(
      `/teams/${encodeURIComponent(teamId)}/exclusive-members`,
      undefined,
      ExclusiveMembersResponseSchema
    )
  }

  async addTeamMember(teamId: string, request: AddMemberRequest): Promise<TeamMember> {
    return this.request<TeamMember>(
      `/teams/${encodeURIComponent(teamId)}/members`,
      {
        method: 'POST',
        body: JSON.stringify(request),
      },
      TeamMemberSchema
    )
  }

  async updateTeamMember(teamId: string, agentId: string, request: UpdateMemberRequest): Promise<TeamMember> {
    return this.request<TeamMember>(
      `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}`,
      {
        method: 'PUT',
        body: JSON.stringify(request),
      },
      TeamMemberSchema
    )
  }

  async removeTeamMember(teamId: string, agentId: string): Promise<void> {
    await this.requestVoid(
      `/teams/${encodeURIComponent(teamId)}/members/${encodeURIComponent(agentId)}`,
      { method: 'DELETE' }
    )
  }

  async getTeamRoles(teamId: string): Promise<TeamRole[]> {
    return this.request<TeamRole[]>(
      `/teams/${encodeURIComponent(teamId)}/roles`,
      undefined,
      TeamRoleSchema.array()
    )
  }

  async setTeamRoles(teamId: string, roles: TeamRole[]): Promise<TeamRole[]> {
    return this.request<TeamRole[]>(
      `/teams/${encodeURIComponent(teamId)}/roles`,
      {
        method: 'PUT',
        body: JSON.stringify({ roles }),
      },
      TeamRoleSchema.array()
    )
  }

  async listTeamSharedFiles(teamId: string): Promise<TeamSharedFileListResponse> {
    return this.request<TeamSharedFileListResponse>(
      `/teams/${encodeURIComponent(teamId)}/shared/files`,
      undefined,
      TeamSharedFileListResponseSchema
    )
  }

  async getTeamSharedFileContent(teamId: string, path: string): Promise<TeamSharedFileContentResponse> {
    const params = `?path=${encodeURIComponent(path)}`
    return this.request<TeamSharedFileContentResponse>(
      `/teams/${encodeURIComponent(teamId)}/shared/files/content${params}`,
      undefined,
      TeamSharedFileContentResponseSchema
    )
  }

  async setTeamSharedFileContent(teamId: string, path: string, content: string): Promise<void> {
    const params = `?path=${encodeURIComponent(path)}`
    await this.requestVoid(
      `/teams/${encodeURIComponent(teamId)}/shared/files/content${params}`,
      {
        method: 'PUT',
        body: JSON.stringify({ content }),
      }
    )
  }

  async createTeamSharedFile(teamId: string, request: TeamSharedFileCreateRequest): Promise<void> {
    await this.requestVoid(
      `/teams/${encodeURIComponent(teamId)}/shared/files`,
      {
        method: 'POST',
        body: JSON.stringify(request),
      }
    )
  }

  async renameTeamSharedFile(teamId: string, request: TeamSharedFileRenameRequest): Promise<void> {
    await this.requestVoid(
      `/teams/${encodeURIComponent(teamId)}/shared/files/rename`,
      {
        method: 'POST',
        body: JSON.stringify(request),
      }
    )
  }

  async deleteTeamSharedFile(teamId: string, path: string): Promise<void> {
    const params = `?path=${encodeURIComponent(path)}`
    await this.requestVoid(
      `/teams/${encodeURIComponent(teamId)}/shared/files${params}`,
      {
        method: 'DELETE',
      }
    )
  }

  // Claude Code interop methods
  async listAvailableCCTeams(): Promise<AvailableCCTeam[]> {
    return this.request<AvailableCCTeam[]>(
      '/teams/import/claude-code/available',
      undefined,
      AvailableCCTeamSchema.array()
    )
  }

  async importClaudeCodeTeam(teamName: string): Promise<TeamDetails> {
    return this.request<TeamDetails>(
      '/teams/import/claude-code',
      {
        method: 'POST',
        body: JSON.stringify({ teamName }),
      },
      TeamDetailsSchema
    )
  }

  async exportClaudeCodeTeam(teamId: string): Promise<ExportCCResponse> {
    return this.request<ExportCCResponse>(
      `/teams/${encodeURIComponent(teamId)}/export/claude-code`,
      undefined,
      ExportCCResponseSchema
    )
  }

  // World scale methods
  async getWorldScale(): Promise<WorldScaleConfig> {
    return this.request<WorldScaleConfig>('/world-scale', undefined, WorldScaleConfigSchema)
  }

  async setWorldScale(config: WorldScaleConfig): Promise<WorldScaleConfig> {
    return this.request<WorldScaleConfig>(
      '/world-scale',
      {
        method: 'PUT',
        body: JSON.stringify(config),
      },
      WorldScaleConfigSchema
    )
  }

  // World seats methods
  async getWorldSeats(): Promise<WorldSeatsConfig> {
    return this.request<WorldSeatsConfig>('/world-seats', undefined, WorldSeatsConfigSchema)
  }

  async setWorldSeats(config: WorldSeatsConfig): Promise<WorldSeatsConfig> {
    return this.request<WorldSeatsConfig>(
      '/world-seats',
      {
        method: 'PUT',
        body: JSON.stringify(config),
      },
      WorldSeatsConfigSchema
    )
  }

  // Graph methods - aligned with api/graph/handlers.go
  async getGraph(): Promise<GraphResponse> {
    return this.request<GraphResponse>('/graph', undefined, GraphResponseSchema)
  }

  async getGraphNode(id: string): Promise<NodeDetailResponse> {
    return this.request<NodeDetailResponse>(
      `/graph/nodes/${encodeURIComponent(id)}`,
      undefined,
      NodeDetailResponseSchema
    )
  }

  async regenerateGraph(): Promise<RegenerateResponse> {
    return this.request<RegenerateResponse>(
      '/graph/regenerate',
      { method: 'POST' },
      RegenerateResponseSchema
    )
  }

  async getOrphanedSkills(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/orphans',
      undefined,
      NodeListResponseSchema
    )
  }

  async getSkilllessAgents(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/skillless',
      undefined,
      NodeListResponseSchema
    )
  }

  async getEmptyTeams(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/empty-teams',
      undefined,
      NodeListResponseSchema
    )
  }

  async getUnaffiliatedAgents(): Promise<NodeListResponse> {
    return this.request<NodeListResponse>(
      '/graph/unaffiliated',
      undefined,
      NodeListResponseSchema
    )
  }

  async getCLIlessSkills(): Promise<NodeListResponse> {
    // No dedicated backend endpoint - compute from full graph client-side.
    // Fetch the full graph and filter skills that have no code-usage edges.
    const idx = await this.getGraph()
    const hasCLI = new Set<string>()
    for (const e of idx.graph.edges) {
      if (e.kind === 'code-usage') hasCLI.add(e.from)
    }
    return idx.graph.nodes.filter((n) => n.type === 'skill' && !hasCLI.has(n.id))
  }

  async getPopular(limit?: number): Promise<PopularityResponse> {
    const params = new URLSearchParams()
    if (limit) params.set('limit', limit.toString())
    const qs = params.toString()
    return this.request<PopularityResponse>(
      `/graph/popular${qs ? `?${qs}` : ''}`,
      undefined,
      PopularityResponseSchema
    )
  }

  async getCircularRefs(): Promise<CircularRefResponse> {
    return this.request<CircularRefResponse>(
      '/graph/cycles',
      undefined,
      CircularRefResponseSchema
    )
  }

  async getGraphHealth(): Promise<GraphHealthResponse> {
    return this.request<GraphHealthResponse>(
      '/graph/health',
      undefined,
      GraphHealthResponseSchema
    )
  }

  async getNodeHealth(id: string): Promise<NodeHealthResponse | null> {
    // No per-node health endpoint - filter from full health scores
    const scores = await this.getGraphHealth()
    return scores.find((s) => s.nodeId === id) ?? null
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
