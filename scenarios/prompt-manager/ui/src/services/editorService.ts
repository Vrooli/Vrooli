/**
 * Editor Service - Pure functions for form state management and validation.
 *
 * Handles:
 * - Converting between Prompt and PromptFormState
 * - Validating form fields
 * - Detecting dirty state
 */

import type { Prompt, UpdatePromptRequest } from '@/types'
import type { PromptFormState, ValidationResult } from '@/types/editor'

/**
 * Convert a Prompt to editable form state.
 *
 * @param prompt - The prompt to convert
 * @returns Form state ready for editing
 */
export function promptToFormState(prompt: Prompt): PromptFormState {
  return {
    name: prompt.name,
    description: prompt.description,
    content: prompt.content,
    modes: [...prompt.modes],
    tags: prompt.tags.join(', '),
    icon: prompt.icon ?? '',
    targetToolId: prompt.targetToolId ?? '',
    draft: prompt.draft,
  }
}

/**
 * Convert form state to API update request.
 *
 * @param formState - The form state to convert
 * @returns Update request for the API
 */
export function formStateToUpdateRequest(formState: PromptFormState): UpdatePromptRequest {
  return {
    name: formState.name,
    description: formState.description,
    content: formState.content,
    modes: formState.modes,
    tags: parseTags(formState.tags),
    icon: formState.icon || undefined,
    targetToolId: formState.targetToolId || undefined,
    draft: formState.draft,
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
export function validateFormState(formState: PromptFormState): ValidationResult {
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
  // We don't enforce this as an error since prompts can exist without modes

  return {
    valid: Object.keys(errors).length === 0,
    errors,
  }
}

/**
 * Check if form state differs from original prompt.
 *
 * @param original - Original prompt
 * @param current - Current form state
 * @returns True if there are unsaved changes
 */
export function isDirty(original: Prompt, current: PromptFormState): boolean {
  // Compare each field
  if (original.name !== current.name) return true
  if (original.description !== current.description) return true
  if (original.content !== current.content) return true
  if (original.draft !== current.draft) return true
  if ((original.icon ?? '') !== current.icon) return true
  if ((original.targetToolId ?? '') !== current.targetToolId) return true

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
 * Create an empty form state for a new prompt.
 *
 * @returns Default form state for creating a new prompt
 */
export function createEmptyFormState(): PromptFormState {
  return {
    name: '',
    description: '',
    content: '',
    modes: [],
    tags: '',
    icon: '',
    targetToolId: '',
    draft: true,
  }
}

/**
 * Get a summary of changes for display.
 *
 * @param original - Original prompt
 * @param current - Current form state
 * @returns Human-readable list of changed fields
 */
export function getChangeSummary(original: Prompt, current: PromptFormState): string[] {
  const changes: string[] = []

  if (original.name !== current.name) changes.push('name')
  if (original.description !== current.description) changes.push('description')
  if (original.content !== current.content) changes.push('content')
  if (original.draft !== current.draft) changes.push('draft status')
  if ((original.icon ?? '') !== current.icon) changes.push('icon')
  if ((original.targetToolId ?? '') !== current.targetToolId) changes.push('target tool')

  const currentTags = parseTags(current.tags)
  if (!arraysEqual(original.tags, currentTags)) changes.push('tags')

  if (!arraysEqual(original.modes, current.modes)) changes.push('modes')

  return changes
}
