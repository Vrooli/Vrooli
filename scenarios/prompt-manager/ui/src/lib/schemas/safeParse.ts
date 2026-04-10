/**
 * Safe parsing utilities for Zod schemas.
 *
 * Provides a consistent interface for runtime validation at API boundaries.
 * These utilities ensure that invalid data is caught early and handled gracefully.
 */

import type { ZodType } from 'zod'
import { z } from 'zod'

/**
 * Result type for safe parsing operations.
 * Uses a discriminated union to make success/failure explicit.
 */
export type ParseResult<T> =
  | { success: true; data: T }
  | { success: false; error: ValidationError }

/**
 * Custom error class for validation failures.
 * Provides structured error information for logging and debugging.
 */
export class ValidationError extends Error {
  readonly context: string
  readonly issues: string[]
  readonly originalError: z.ZodError

  constructor(zodError: z.ZodError, context: string) {
    const issues = zodError.issues.map((issue) => {
      const path = issue.path.join('.')
      return path ? `${path}: ${issue.message}` : issue.message
    })

    const message = `Validation failed at ${context}: ${issues.join('; ')}`
    super(message)

    this.name = 'ValidationError'
    this.context = context
    this.issues = issues
    this.originalError = zodError

    // Ensure proper prototype chain for instanceof checks
    Object.setPrototypeOf(this, ValidationError.prototype)
  }
}

/**
 * Safely parse data with a Zod schema.
 *
 * @param schema - Zod schema to validate against
 * @param data - Data to validate (should be unknown from API response)
 * @param context - Context string for error messages (e.g., endpoint name)
 * @returns ParseResult with either validated data or ValidationError
 *
 * @example
 * const result = safeParse(SkillSchema, apiResponse, '/skills/123')
 * if (result.success) {
 *   console.log(result.data.name)
 * } else {
 *   console.error(result.error.message)
 * }
 */
export function safeParse<T>(
  schema: ZodType<T>,
  data: unknown,
  context: string
): ParseResult<T> {
  const result = schema.safeParse(data)

  if (result.success) {
    return { success: true, data: result.data }
  }

  return { success: false, error: new ValidationError(result.error, context) }
}

/**
 * Parse data with a Zod schema, throwing on failure.
 *
 * Use this when validation failure should be treated as an exceptional condition.
 * The thrown ValidationError can be caught by error boundaries or try/catch.
 *
 * @param schema - Zod schema to validate against
 * @param data - Data to validate
 * @param context - Context string for error messages
 * @returns Validated data of type T
 * @throws ValidationError if validation fails
 *
 * @example
 * try {
 *   const skill = parseOrThrow(SkillSchema, apiResponse, '/skills/123')
 *   // skill is now guaranteed to be valid
 * } catch (error) {
 *   if (error instanceof ValidationError) {
 *     console.error('API returned invalid data:', error.issues)
 *   }
 * }
 */
export function parseOrThrow<T>(
  schema: ZodType<T>,
  data: unknown,
  context: string
): T {
  const result = safeParse(schema, data, context)

  if (result.success) {
    return result.data
  }

  throw result.error
}

/**
 * Parse data with a Zod schema, returning null on failure.
 *
 * Use this for optional data where null is an acceptable fallback.
 * Logs a warning on validation failure for debugging.
 *
 * @param schema - Zod schema to validate against
 * @param data - Data to validate
 * @param context - Context string for error messages
 * @returns Validated data or null if validation fails
 *
 * @example
 * const skill = parseOrNull(SkillSchema, cachedData, 'cached-skill')
 * if (skill) {
 *   // Use cached data
 * } else {
 *   // Fetch fresh data
 * }
 */
export function parseOrNull<T>(
  schema: ZodType<T>,
  data: unknown,
  context: string
): T | null {
  const result = safeParse(schema, data, context)

  if (result.success) {
    return result.data
  }

  console.warn(`[parseOrNull] ${result.error.message}`)
  return null
}

/**
 * Parse an array of items, filtering out invalid entries.
 *
 * Use this when you want to gracefully handle partially invalid arrays.
 * Invalid items are logged and excluded from the result.
 *
 * @param schema - Zod schema for individual items
 * @param data - Array of items to validate
 * @param context - Context string for error messages
 * @returns Array of valid items (invalid items filtered out)
 *
 * @example
 * // API returns 10 skills, 2 are malformed
 * const validSkills = parseArrayFiltered(SkillSchema, apiResponse, '/skills')
 * // validSkills contains 8 valid skills, 2 were logged and filtered out
 */
export function parseArrayFiltered<T>(
  schema: ZodType<T>,
  data: unknown[],
  context: string
): T[] {
  const validItems: T[] = []
  let invalidCount = 0

  for (let i = 0; i < data.length; i++) {
    const item = data[i]
    const result = schema.safeParse(item)

    if (result.success) {
      validItems.push(result.data)
    } else {
      invalidCount++
      console.warn(
        `[parseArrayFiltered] Invalid item at index ${i} in ${context}:`,
        result.error.issues.map((issue) => issue.message).join('; ')
      )
    }
  }

  if (invalidCount > 0) {
    console.warn(
      `[parseArrayFiltered] Filtered out ${invalidCount}/${data.length} invalid items from ${context}`
    )
  }

  return validItems
}
