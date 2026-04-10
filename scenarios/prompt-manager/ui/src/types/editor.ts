/**
 * Editor-related types for the prompt manager UI.
 * Separated from API types to maintain clean boundaries.
 */

import type { Skill, FolderType } from './index'

/**
 * Tree node structure for displaying skills in a hierarchical view.
 * Adapted from agent-inbox ItemTreeSidebar.
 */
export interface TreeNode {
  id: string           // Unique node ID (path-based for categories, "item-{id}" for leaves)
  label: string        // Display name
  isCategory: boolean  // true for folders, false for items
  children: TreeNode[]
  itemId?: string      // Only for leaf nodes - the actual skill ID
  depth: number
}

/**
 * Form state for editing a skill.
 * Separated from Skill to allow independent tracking of edits.
 */
export interface SkillFormState {
  name: string
  description: string
  content: string
  modes: string[]
  tags: string         // Comma-separated for input simplicity
  icon: string
  draft: boolean
  folder: FolderType   // Folder determines git behavior
}

/**
 * Tracks pending changes for a single skill.
 * Enables multi-item editing with save/discard functionality.
 */
export interface PendingChange {
  original: Skill
  current: SkillFormState
  isDirty: boolean
}

/**
 * Validation result for form fields.
 */
export interface ValidationResult {
  valid: boolean
  errors: Record<string, string>
}

/**
 * Mode suggestion for category path editor.
 */
export interface ModeSuggestion {
  value: string
  isNew: boolean // True if this is a new mode not used by existing skills
}
