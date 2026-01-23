// Types aligned with the Go API (api/skills/models.go)
// These match the exact response shapes from the prompt-manager API

/**
 * Folder represents the organizational structure for skills.
 * - "core": Important skills (git-tracked)
 * - "local": Personal skills (gitignored)
 * - "drafts": Work-in-progress skills
 */
export type FolderType = 'core' | 'local' | 'drafts'

/**
 * Folder metadata for UI display
 */
export interface Folder {
  id: FolderType
  name: string
  description: string
  icon: string
  skillCount: number
}

/**
 * Skill matches the API's Response type from api/skills/models.go
 */
export interface Skill {
  id: string
  name: string
  description: string
  content: string
  modes: string[]
  tags: string[]
  icon?: string
  targetToolId?: string | null
  draft: boolean
  folder: FolderType
  createdAt: string
  updatedAt: string
  usageCount: number
  lastUsed?: string | null
  effectivenessRating?: number | null
}

/**
 * CreateSkillRequest matches the API's CreateRequest type
 */
export interface CreateSkillRequest {
  id?: string
  name: string
  description: string
  content: string
  modes?: string[]
  tags?: string[]
  icon?: string
  targetToolId?: string | null
  draft?: boolean
  folder: FolderType  // Can create in any folder
}

/**
 * UpdateSkillRequest matches the API's UpdateRequest type
 */
export interface UpdateSkillRequest {
  name?: string
  description?: string
  content?: string
  modes?: string[]
  tags?: string[]
  icon?: string
  targetToolId?: string | null
  draft?: boolean
  folder?: FolderType
}

/**
 * Tag from the tags domain
 */
export interface Tag {
  id: string
  name: string
  color?: string
  description?: string
}

/**
 * SkillTestRequest for testing skills with Ollama
 */
export interface SkillTestRequest {
  model: string
  inputVariables?: Record<string, string>
  maxTokens?: number
  temperature?: number
}

/**
 * SkillTestResult from the testing domain
 */
export interface SkillTestResult {
  id: string
  skillId: string
  model: string
  inputVariables?: Record<string, string>
  response: string
  responseTime: number
  tokenCount?: number
  rating?: number
  notes?: string
  testedAt: string
}

/**
 * SearchFilters for filtering skills
 */
export interface SearchFilters {
  tag?: string
  folder?: FolderType
  modes?: string[]
}

/**
 * SyncResponse matches the API's SyncResponse type
 */
export interface SyncResponse {
  skills: Skill[]
  lastUpdated: string
  hash: string
}

/**
 * HealthResponse from the health endpoint
 */
export interface HealthResponse {
  status: string
  service: string
  version: string
  readiness: boolean
  timestamp: string
  dependencies?: Record<string, string>
}

// Theme types for UI settings
export type Theme = 'light' | 'dark' | 'system'

export interface AppSettings {
  theme: Theme
  sidebarCollapsed: boolean
  editorSettings: {
    fontSize: number
    tabSize: number
    wordWrap: boolean
    minimap: boolean
  }
}

// Local-only state (not in API)
export interface LocalSkillState {
  favorites: Set<string>  // Skill IDs marked as favorite locally
}
