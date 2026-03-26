/**
 * Types for the prompt-manager UI.
 *
 * API types are re-exported from @/lib/schemas (single source of truth).
 * UI-only types are defined here.
 */

// Re-export API types from schemas (these include runtime validation)
export type {
  FolderType,
  Skill,
  CreateSkillRequest,
  UpdateSkillRequest,
  Tag,
  SkillTestRequest,
  SkillTestResult,
  SyncResponse,
  HealthResponse,
  UsageResponse,
  RatingResponse,
  DisplayFormat,
  DisplayResponse,
} from '@/lib/schemas'

/**
 * Folder metadata for UI display.
 * This is a UI-only type (not from API - computed client-side).
 */
export interface Folder {
  id: 'core' | 'local' | 'drafts'
  name: string
  description: string
  icon: string
  skillCount: number
}

/**
 * SearchFilters for filtering skills.
 * This is a UI-only type used for local filtering.
 */
export interface SearchFilters {
  tag?: string
  folder?: 'core' | 'local' | 'drafts'
  modes?: string[]
}

/**
 * Sidebar search mode.
 */
export type SkillSearchMode = 'quick' | 'content' | 'ai'

/**
 * Content search options (UI state).
 */
export interface ContentSearchOptions {
  caseSensitive: boolean
  wholeWord: boolean
  regex: boolean
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

// Re-export entity editor store types
export type {
  EntitySnapshot,
  EntityEditState,
  PersistedEntityState,
  ValidationResult,
  SaveResult,
} from './entityEditorStore'
export {
  MAX_HISTORY_SIZE,
  PERSISTENCE_DEBOUNCE_MS,
  STALENESS_THRESHOLD_MS,
  ENTITY_PERSISTENCE_VERSION,
  ENTITY_STORAGE_KEYS,
} from './entityEditorStore'
