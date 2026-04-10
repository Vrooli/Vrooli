/**
 * Validation utilities for UI forms and requests.
 *
 * Provides consistent validation across the application for:
 * - Hex colors
 * - Identifiers (kebab-case)
 * - Agent/Member/Skill create requests
 *
 * Uses Zod schemas internally for consistency with API validation.
 */

import { z } from 'zod'
import {
  HexColorSchema,
  KebabCaseIdSchema,
  FolderTypeSchema,
} from '@/lib/schemas'

/**
 * Validates a hex color string.
 * @param color - Color string to validate (e.g., "#FF5733")
 * @returns true if valid hex color format
 */
export function isValidHexColor(color: string): boolean {
  return HexColorSchema.safeParse(color).success
}

/**
 * Validates a kebab-case identifier.
 * @param id - Identifier to validate (e.g., "my-skill-id")
 * @returns true if valid kebab-case format
 */
export function isValidKebabCase(id: string): boolean {
  return KebabCaseIdSchema.safeParse(id).success
}

/**
 * Validation result type
 */
export interface ValidationResult {
  valid: boolean
  errors: string[]
}

/**
 * Convert Zod parse result to ValidationResult format.
 */
function zodToValidationResult<T>(
  schema: z.ZodType<T>,
  data: unknown
): ValidationResult {
  const result = schema.safeParse(data)
  if (result.success) {
    return { valid: true, errors: [] }
  }

  const errors = result.error.issues.map((issue) => {
    const path = issue.path.join('.')
    return path ? `${path}: ${issue.message}` : issue.message
  })
  return { valid: false, errors }
}

/**
 * Validates agent create request using Zod schema.
 */
export function validateAgentCreate(req: {
  displayName?: string
  appearance?: {
    body?: string
    head?: string
    accent?: string
  }
}): ValidationResult {
  // Create a partial schema that matches the input format
  const InputSchema = z.object({
    displayName: z.string().min(1, 'Display name is required').max(100, 'Display name must be 100 characters or less'),
    appearance: z.object({
      body: HexColorSchema.optional(),
      head: HexColorSchema.optional(),
      accent: HexColorSchema.optional(),
    }).optional(),
  })

  // Transform the input to match expected format
  const input = {
    displayName: req.displayName?.trim() ?? '',
    appearance: req.appearance,
  }

  return zodToValidationResult(InputSchema, input)
}

/**
 * Validates member create request (legacy format).
 * Uses manual validation to maintain exact error messages.
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
 * Validates skill create request using Zod schema.
 */
export function validateSkillCreate(req: {
  name?: string
  content?: string
  folder?: string
}): ValidationResult {
  const SkillCreateInputSchema = z.object({
    name: z.string().min(1, 'Name is required'),
    content: z.string().min(1, 'Content is required'),
    folder: FolderTypeSchema,
  })

  // Transform the input to handle empty strings
  const input = {
    name: req.name?.trim() ?? '',
    content: req.content?.trim() ?? '',
    folder: req.folder ?? '',
  }

  const result = SkillCreateInputSchema.safeParse(input)

  if (result.success) {
    return { valid: true, errors: [] }
  }

  // Transform Zod errors to user-friendly messages
  const errors = result.error.issues.map((issue) => {
    const isFolder = issue.path.includes('folder')
    if (isFolder && (issue.code === 'invalid_value' || issue.code === 'invalid_type')) {
      // In Zod v4, enum validation errors use 'invalid_value'
      return 'Folder must be local, drafts, or core'
    }
    return issue.message
  })

  return { valid: false, errors }
}
