/**
 * Safe parsing utilities for runtime validation with Zod.
 *
 * This module provides a standardized way to validate API responses and external data
 * at system boundaries. Components receive validated, typed data and can handle
 * validation failures gracefully with explicit error states.
 *
 * Key principle from ui-health.md:
 * - Validate once at the system boundary (in Services)
 * - Then trust the data as it flows through the application
 */

import { z, type ZodSchema, type ZodError } from "zod";

/**
 * Result type for safe parsing operations.
 * Uses a discriminated union so callers must handle both success and failure cases.
 */
export type ParseResult<T> =
  | { success: true; data: T }
  | { success: false; error: string; details?: ZodError };

/**
 * Safely parse data against a Zod schema.
 * Returns a discriminated union that makes validation failures explicit.
 *
 * @example
 * ```ts
 * const result = safeParse(PipelineStatusSchema, apiResponse);
 * if (!result.success) {
 *   // Handle validation error - show error UI, log, etc.
 *   console.error('Validation failed:', result.error);
 *   return;
 * }
 * // result.data is now typed and validated
 * processPipeline(result.data);
 * ```
 */
export function safeParse<T>(
  schema: ZodSchema<T>,
  data: unknown
): ParseResult<T> {
  const result = schema.safeParse(data);

  if (result.success) {
    return { success: true, data: result.data };
  }

  // Format error message for debugging
  const errorMessage = formatZodError(result.error);

  return {
    success: false,
    error: errorMessage,
    details: result.error,
  };
}

/**
 * Parse with fallback value for non-critical data.
 * Use when you have a sensible default and don't need to surface the error.
 *
 * @example
 * ```ts
 * const config = safeParseWithDefault(ConfigSchema, maybeConfig, defaultConfig);
 * ```
 */
export function safeParseWithDefault<T>(
  schema: ZodSchema<T>,
  data: unknown,
  defaultValue: T
): T {
  const result = schema.safeParse(data);
  if (result.success) {
    return result.data;
  }

  // Log warning in development
  if (import.meta.env.DEV) {
    console.warn("[safeParse] Using default value due to validation failure:", {
      error: formatZodError(result.error),
      data,
    });
  }

  return defaultValue;
}

/**
 * Parse and throw on failure - use only when failure is truly exceptional.
 * Prefer safeParse() for API responses where malformed data is possible.
 */
export function parseOrThrow<T>(schema: ZodSchema<T>, data: unknown): T {
  const result = schema.safeParse(data);
  if (!result.success) {
    throw new ValidationError(formatZodError(result.error), result.error);
  }
  return result.data;
}

/**
 * Custom error class for validation failures.
 * Can be caught specifically in error boundaries or try/catch blocks.
 */
export class ValidationError extends Error {
  public readonly zodError: ZodError;

  constructor(message: string, zodError: ZodError) {
    super(message);
    this.name = "ValidationError";
    this.zodError = zodError;
  }
}

/**
 * Format Zod validation error into a human-readable message.
 */
function formatZodError(error: ZodError): string {
  const issues = error.issues.map((issue: z.ZodIssue) => {
    const path = issue.path.length > 0 ? `${issue.path.join(".")}: ` : "";
    return `${path}${issue.message}`;
  });

  if (issues.length === 1 && issues[0]) {
    return issues[0];
  }

  return `Validation failed:\n- ${issues.join("\n- ")}`;
}

/**
 * Type guard to check if an error is a ValidationError.
 */
export function isValidationError(error: unknown): error is ValidationError {
  return error instanceof ValidationError;
}

// Re-export commonly used Zod utilities for convenience
export { z } from "zod";
export type { ZodSchema, ZodError } from "zod";
