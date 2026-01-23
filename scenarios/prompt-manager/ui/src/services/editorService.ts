/**
 * Editor Service - Pure functions for form state management and validation.
 *
 * Handles:
 * - Converting between Skill and SkillFormState (legacy)
 * - Converting between Skill and NormalizedFormState (new)
 * - Validating form fields
 * - Detecting dirty state
 */

import type { Skill, UpdateSkillRequest } from '@/types'
import type { SkillFormState, ValidationResult } from '@/types/editor'
import type { NormalizedFormState } from '@/types/editorStore'

/**
 * Convert a Skill to editable form state.
 *
 * @param skill - The skill to convert
 * @returns Form state ready for editing
 */
export function skillToFormState(skill: Skill): SkillFormState {
  return {
    name: skill.name,
    description: skill.description,
    content: skill.content,
    modes: [...skill.modes],
    tags: skill.tags.join(', '),
    icon: skill.icon ?? '',
    draft: skill.draft,
    folder: skill.folder,
  }
}

/**
 * Convert form state to API update request.
 *
 * @param formState - The form state to convert
 * @returns Update request for the API
 */
export function formStateToUpdateRequest(formState: SkillFormState): UpdateSkillRequest {
  return {
    name: formState.name,
    description: formState.description,
    content: formState.content,
    modes: formState.modes,
    tags: parseTags(formState.tags),
    icon: formState.icon || undefined,
    draft: formState.draft,
    folder: formState.folder,
  }
}

/**
 * Parse comma-separated tags string into array.
 *
 * @param tagsString - Comma-separated tags
 * @returns Array of trimmed, non-empty tags
 */
export function parseTags(tagsString: string): string[] {
  return tagsString
    .split(',')
    .map((t) => t.trim())
    .filter((t) => t.length > 0)
}

/**
 * Format tags array as comma-separated string.
 *
 * @param tags - Array of tags
 * @returns Comma-separated string
 */
export function formatTags(tags: string[]): string {
  return tags.join(', ')
}

/**
 * Validate form state.
 *
 * @param formState - The form state to validate
 * @returns Validation result with errors by field
 */
export function validateFormState(formState: SkillFormState): ValidationResult {
  const errors: Record<string, string> = {}

  // Name is required
  if (!formState.name.trim()) {
    errors.name = 'Name is required'
  } else if (formState.name.length > 100) {
    errors.name = 'Name must be 100 characters or less'
  }

  // Description is optional but limited
  if (formState.description.length > 500) {
    errors.description = 'Description must be 500 characters or less'
  }

  // Content is required
  if (!formState.content.trim()) {
    errors.content = 'Content is required'
  }

  // At least one mode is recommended (warning, not error)
  // We don't enforce this as an error since skills can exist without modes

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}

/**
 * Check if form state differs from original skill.
 *
 * @param original - Original skill
 * @param current - Current form state
 * @returns True if there are unsaved changes
 */
export function isDirty(original: Skill, current: SkillFormState): boolean {
  // Compare each field
  if (original.name !== current.name) return true
  if (original.description !== current.description) return true
  if (original.content !== current.content) return true
  if (original.draft !== current.draft) return true
  if (original.folder !== current.folder) return true
  if ((original.icon ?? '') !== current.icon) return true

  // Compare tags (convert current to array for comparison)
  const currentTags = parseTags(current.tags)
  if (!arraysEqual(original.tags, currentTags)) return true

  // Compare modes
  if (!arraysEqual(original.modes, current.modes)) return true

  return false
}

/**
 * Check if two string arrays are equal.
 */
function arraysEqual(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false
  }
  return true
}

/**
 * Create an empty form state for a new skill.
 *
 * @returns Default form state for creating a new skill
 */
export function createEmptyFormState(): SkillFormState {
  return {
    name: '',
    description: '',
    content: '',
    modes: [],
    tags: '',
    icon: '',
    draft: true,
    folder: 'local',
  }
}

/**
 * Get a summary of changes for display.
 *
 * @param original - Original skill
 * @param current - Current form state
 * @returns Human-readable list of changed fields
 */
export function getChangeSummary(original: Skill, current: SkillFormState): string[] {
  const changes: string[] = []

  if (original.name !== current.name) changes.push('name')
  if (original.description !== current.description) changes.push('description')
  if (original.content !== current.content) changes.push('content')
  if (original.draft !== current.draft) changes.push('draft status')
  if (original.folder !== current.folder) changes.push('folder')
  if ((original.icon ?? '') !== current.icon) changes.push('icon')

  const currentTags = parseTags(current.tags)
  if (!arraysEqual(original.tags, currentTags)) changes.push('tags')

  if (!arraysEqual(original.modes, current.modes)) changes.push('modes')

  return changes
}

// ============================================================================
// Normalized Form State Functions (New API)
// ============================================================================

/**
 * Convert NormalizedFormState to API update request.
 * Tags are already an array, so no parsing needed.
 *
 * @param state - The normalized form state
 * @param originalFile - Original filename to detect renames
 * @returns Update request for the API
 */
export function normalizedStateToUpdateRequest(
  state: NormalizedFormState,
  originalFile?: string
): UpdateSkillRequest {
  const request: UpdateSkillRequest = {
    name: state.name,
    description: state.description,
    content: state.content,
    modes: state.modes,
    tags: state.tags, // Already an array
    icon: state.icon || undefined,
    draft: state.draft,
    folder: state.folder,
  }
  // Only include file if it changed (to trigger rename)
  if (originalFile && state.file !== originalFile) {
    request.file = state.file
  }
  return request
}

/**
 * Convert legacy SkillFormState to NormalizedFormState.
 * Useful during migration period.
 *
 * @param formState - Legacy form state with comma-separated tags
 * @param file - Optional filename (legacy state doesn't have this)
 * @returns Normalized form state with tags as array
 */
export function legacyToNormalizedState(
  formState: SkillFormState,
  file = ''
): NormalizedFormState {
  return {
    file,
    name: formState.name,
    description: formState.description,
    content: formState.content,
    modes: formState.modes,
    tags: parseTags(formState.tags), // Parse comma-separated string
    icon: formState.icon,
    draft: formState.draft,
    folder: formState.folder,
  }
}

/**
 * Convert NormalizedFormState to legacy SkillFormState.
 * Useful during migration period. Note: file field is lost.
 *
 * @param state - Normalized form state with tags as array
 * @returns Legacy form state with comma-separated tags
 */
export function normalizedToLegacyState(state: NormalizedFormState): SkillFormState {
  return {
    name: state.name,
    description: state.description,
    content: state.content,
    modes: state.modes,
    tags: formatTags(state.tags), // Convert array to comma-separated string
    icon: state.icon,
    draft: state.draft,
    folder: state.folder,
  }
}
