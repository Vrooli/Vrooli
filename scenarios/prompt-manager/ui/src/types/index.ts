// Types aligned with the Go API (api/prompts/models.go)
// These match the exact response shapes from the prompt-manager API

/**
 * Folder represents the organizational structure for prompts.
 * - "core": Read-only system prompts (e.g., steering prompts)
 * - "local": User-created prompts that persist
 * - "drafts": Work-in-progress prompts
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
  readonly: boolean
  promptCount: number
}

/**
 * Prompt matches the API's Response type from api/prompts/models.go
 */
export interface Prompt {
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
 * CreatePromptRequest matches the API's CreateRequest type
 */
export interface CreatePromptRequest {
  id?: string
  name: string
  description: string
  content: string
  modes?: string[]
  tags?: string[]
  icon?: string
  targetToolId?: string | null
  draft?: boolean
  folder: 'local' | 'drafts'  // Can only create in writable folders
}

/**
 * UpdatePromptRequest matches the API's UpdateRequest type
 */
export interface UpdatePromptRequest {
  name?: string
  description?: string
  content?: string
  modes?: string[]
  tags?: string[]
  icon?: string
  targetToolId?: string | null
  draft?: boolean
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
 * PromptTestRequest for testing prompts with Ollama
 */
export interface PromptTestRequest {
  model: string
  inputVariables?: Record<string, string>
  maxTokens?: number
  temperature?: number
}

/**
 * PromptTestResult from the testing domain
 */
export interface PromptTestResult {
  id: string
  promptId: string
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
 * SearchFilters for filtering prompts
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
  prompts: Prompt[]
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
export interface LocalPromptState {
  favorites: Set<string>  // Prompt IDs marked as favorite locally
}
