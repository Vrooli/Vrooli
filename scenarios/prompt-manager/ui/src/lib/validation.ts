/**
 * Validation utilities for UI forms and requests.
 *
 * Provides consistent validation across the application for:
 * - Hex colors
 * - Identifiers (kebab-case)
 * - Agent/Member/Skill create requests
 */

/**
 * Validates a hex color string.
 * @param color - Color string to validate (e.g., "#FF5733")
 * @returns true if valid hex color format
 */
export function isValidHexColor(color: string): boolean {
  return /^#[0-9A-Fa-f]{6}$/.test(color)
}

/**
 * Validates a kebab-case identifier.
 * @param id - Identifier to validate (e.g., "my-skill-id")
 * @returns true if valid kebab-case format
 */
export function isValidKebabCase(id: string): boolean {
  return /^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(id)
}

/**
 * Validation result type
 */
export interface ValidationResult {
  valid: boolean
  errors: string[]
}

/**
 * Validates agent create request
 */
export function validateAgentCreate(req: {
  displayName?: string
  appearance?: {
    body?: string
    head?: string
    accent?: string
  }
}): ValidationResult {
  const errors: string[] = []

  if (!req.displayName || req.displayName.trim() === '') {
    errors.push('Display name is required')
  } else if (req.displayName.length > 100) {
    errors.push('Display name must be 100 characters or less')
  }

  if (req.appearance) {
    if (req.appearance.body && !isValidHexColor(req.appearance.body)) {
      errors.push('Body color must be a valid hex color (e.g., #FF5733)')
    }
    if (req.appearance.head && !isValidHexColor(req.appearance.head)) {
      errors.push('Head color must be a valid hex color (e.g., #FF5733)')
    }
    if (req.appearance.accent && !isValidHexColor(req.appearance.accent)) {
      errors.push('Accent color must be a valid hex color (e.g., #FF5733)')
    }
  }

  return { valid: errors.length === 0, errors }
}

/**
 * Validates member create request (legacy format)
 */
export function validateMemberCreate(req: {
  name?: string
  bodyColor?: string
  headColor?: string
  accentColor?: string
}): ValidationResult {
  const errors: string[] = []

  if (!req.name || req.name.trim() === '') {
    errors.push('Name is required')
  }

  if (!req.bodyColor) {
    errors.push('Body color is required')
  } else if (!isValidHexColor(req.bodyColor)) {
    errors.push('Body color must be a valid hex color (e.g., #FF5733)')
  }

  if (!req.headColor) {
    errors.push('Head color is required')
  } else if (!isValidHexColor(req.headColor)) {
    errors.push('Head color must be a valid hex color (e.g., #FF5733)')
  }

  if (!req.accentColor) {
    errors.push('Accent color is required')
  } else if (!isValidHexColor(req.accentColor)) {
    errors.push('Accent color must be a valid hex color (e.g., #FF5733)')
  }

  return { valid: errors.length === 0, errors }
}

/**
 * Validates skill create request
 */
export function validateSkillCreate(req: {
  name?: string
  content?: string
  folder?: string
}): ValidationResult {
  const errors: string[] = []

  if (!req.name || req.name.trim() === '') {
    errors.push('Name is required')
  }

  if (!req.content || req.content.trim() === '') {
    errors.push('Content is required')
  }

  if (!req.folder) {
    errors.push('Folder is required')
  } else if (!['local', 'drafts', 'core'].includes(req.folder)) {
    errors.push('Folder must be local, drafts, or core')
  }

  return { valid: errors.length === 0, errors }
}
