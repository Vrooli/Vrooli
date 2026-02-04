/**
 * Types for the editor store - centralized form state management.
 *
 * Key design decisions:
 * - NormalizedFormState uses string[] for tags (not comma-separated)
 * - History stacks enable undo/redo per prompt
 * - Persisted state includes versioning for migrations
 */

import type { FolderType } from './index'

/**
 * Normalized form state - all fields have consistent types.
 * This eliminates type conversion issues that cause false-positive dirty detection.
 */
export interface NormalizedFormState {
  file: string // Filename (e.g., "my-skill.md")
  name: string
  description: string
  content: string
  modes: string[] // Always array
  tags: string[] // Always array (not comma-separated)
  icon: string // Always string (never null/undefined)
  defaultScope: string // Default scope skill ID (empty string if none)
  draft: boolean
  folder: FolderType
}

/**
 * History snapshot for undo/redo.
 * Captures the full form state at a point in time.
 */
export interface FormSnapshot {
  state: NormalizedFormState
  timestamp: number
}

/**
 * Edit state for a single prompt with history.
 * Tracks original state (from API), current edits, and undo/redo stacks.
 */
export interface PromptEditState {
  /** Original state from API, normalized */
  original: NormalizedFormState
  /** Current edits */
  current: NormalizedFormState
  /** Undo stack (max 50 entries) */
  undoStack: FormSnapshot[]
  /** Redo stack */
  redoStack: FormSnapshot[]
  /** When editing started on this prompt */
  editStartedAt: number
  /** When the prompt was last modified */
  lastModifiedAt: number
}

/**
 * Persisted state structure for localStorage.
 * Includes versioning for future migrations.
 */
export interface PersistedEditorState {
  /** Version number for migration support */
  version: number
  /** Map of prompt ID to edit state */
  prompts: Record<string, PromptEditState>
  /** When the state was last saved */
  lastSavedAt: number
}

/**
 * Validation result for form fields.
 * Re-exported here for convenience.
 */
export interface ValidationResult {
  valid: boolean
  errors: Record<string, string>
}

/** Maximum number of undo/redo entries per prompt */
export const MAX_HISTORY_SIZE = 50

/** Debounce delay for localStorage persistence (ms) */
export const PERSISTENCE_DEBOUNCE_MS = 1000

/** Staleness threshold - prompts older than this are cleaned up (24 hours) */
export const STALENESS_THRESHOLD_MS = 24 * 60 * 60 * 1000

/** Current persistence version */
export const PERSISTENCE_VERSION = 1

/** localStorage key for persisted state */
export const STORAGE_KEY = 'prompt-manager-editor-state'
