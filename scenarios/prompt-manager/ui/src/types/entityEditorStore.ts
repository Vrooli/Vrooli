/**
 * Types for the unified entity editor store system.
 *
 * This module defines common types used by all entity editor stores
 * (skills, agents, teams) to provide consistent form state management
 * with undo/redo and localStorage persistence.
 *
 * Key design principles:
 * - Generic history snapshot system works for any entity type
 * - Per-entity undo/redo stacks
 * - Debounced localStorage persistence
 * - Staleness cleanup for old edits
 */

/**
 * Generic history snapshot for undo/redo.
 * Captures the full form state at a point in time.
 */
export interface EntitySnapshot<T> {
  state: T
  timestamp: number
}

/**
 * Generic edit state for a single entity with history.
 * Tracks original state (from API), current edits, and undo/redo stacks.
 */
export interface EntityEditState<T> {
  /** Original state from API */
  original: T
  /** Current edits */
  current: T
  /** Undo stack (max entries configurable) */
  undoStack: EntitySnapshot<T>[]
  /** Redo stack */
  redoStack: EntitySnapshot<T>[]
  /** When editing started on this entity */
  editStartedAt: number
  /** When the entity was last modified */
  lastModifiedAt: number
}

/**
 * Persisted state structure for localStorage.
 * Includes versioning for future migrations.
 */
export interface PersistedEntityState<T> {
  /** Version number for migration support */
  version: number
  /** Map of entity ID to edit state */
  entities: Record<string, EntityEditState<T>>
  /** When the state was last saved */
  lastSavedAt: number
}

/**
 * Validation result for form fields.
 */
export interface ValidationResult {
  valid: boolean
  errors: Record<string, string>
}

/**
 * Save result from batch save operations.
 */
export interface SaveResult {
  success: boolean
  savedCount: number
  failedCount: number
  errors: Array<{ id: string; name: string; message: string }>
}

/** Maximum number of undo/redo entries per entity */
export const MAX_HISTORY_SIZE = 50

/** Debounce delay for localStorage persistence (ms) */
export const PERSISTENCE_DEBOUNCE_MS = 1000

/** Staleness threshold - entities older than this are cleaned up (24 hours) */
export const STALENESS_THRESHOLD_MS = 24 * 60 * 60 * 1000

/** Current persistence version */
export const ENTITY_PERSISTENCE_VERSION = 1

/**
 * localStorage keys for different entity stores
 */
export const ENTITY_STORAGE_KEYS = {
  skills: 'prompt-manager-editor-state',
  agents: 'prompt-manager-agent-editor-state',
  teams: 'prompt-manager-team-editor-state',
} as const
