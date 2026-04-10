import { z } from 'zod';

/**
 * Result type for successful parsing.
 */
export interface ParseSuccess<T> {
  success: true;
  data: T;
}

/**
 * Result type for failed parsing.
 */
export interface ParseFailure {
  success: false;
  error: string;
  details: z.ZodError;
  raw: unknown;
}

/**
 * Discriminated union result type for safe parsing.
 */
export type ParseResult<T> = ParseSuccess<T> | ParseFailure;

/**
 * Safely parse data against a Zod schema with detailed error logging.
 *
 * @param schema - The Zod schema to validate against
 * @param data - The unknown data to validate
 * @param context - A descriptive context string for error logging (e.g., "WorkflowDefinition")
 * @returns A discriminated union result with either the validated data or error details
 *
 * @example
 * const result = safeParse(WorkflowSchema, apiResponse, 'WorkflowDefinition');
 * if (result.success) {
 *   // result.data is fully typed
 *   console.log(result.data.nodes);
 * } else {
 *   // Handle validation failure
 *   console.error(result.error);
 * }
 */
export function safeParse<T>(
  schema: z.ZodSchema<T>,
  data: unknown,
  context: string
): ParseResult<T> {
  const result = schema.safeParse(data);

  if (result.success) {
    return { success: true, data: result.data };
  }

  // Format error for logging
  const formattedError = result.error.format();
  console.error(`[API Validation] ${context}:`, formattedError);

  // Create a human-readable error summary
  const errorMessages = result.error.errors.map((e: z.ZodIssue) => {
    const path = e.path.length > 0 ? e.path.join('.') : 'root';
    return `${path}: ${e.message}`;
  });

  return {
    success: false,
    error: `Invalid ${context} response: ${errorMessages.join('; ')}`,
    details: result.error,
    raw: data,
  };
}

/**
 * Parse data with a schema, throwing a detailed error on failure.
 * Use this when you want to fail fast rather than handle errors gracefully.
 *
 * @param schema - The Zod schema to validate against
 * @param data - The unknown data to validate
 * @param context - A descriptive context string for error messages
 * @returns The validated data
 * @throws Error with detailed validation failure message
 *
 * @example
 * try {
 *   const workflow = parseOrThrow(WorkflowSchema, apiResponse, 'WorkflowDefinition');
 *   // workflow is fully typed
 * } catch (error) {
 *   // error.message contains validation details
 * }
 */
export function parseOrThrow<T>(
  schema: z.ZodSchema<T>,
  data: unknown,
  context: string
): T {
  const result = safeParse(schema, data, context);

  if (result.success) {
    return result.data;
  }

  throw new Error(result.error);
}

/**
 * Parse data with a schema, returning null on failure.
 * Logs validation errors to console for debugging.
 *
 * @param schema - The Zod schema to validate against
 * @param data - The unknown data to validate
 * @param context - A descriptive context string for error logging
 * @returns The validated data or null if validation fails
 *
 * @example
 * const workflow = parseOrNull(WorkflowSchema, rawWorkflow, 'WorkflowDefinition');
 * if (workflow) {
 *   // workflow is fully typed
 * } else {
 *   // Show fallback UI
 * }
 */
export function parseOrNull<T>(
  schema: z.ZodSchema<T>,
  data: unknown,
  context: string
): T | null {
  const result = safeParse(schema, data, context);
  return result.success ? result.data : null;
}

/**
 * Parse an array of items, filtering out invalid entries.
 * Returns only the items that pass validation.
 *
 * @param schema - The Zod schema for individual items
 * @param data - Array of unknown data to validate
 * @param context - A descriptive context string for error logging
 * @returns Array of validated items (invalid items are filtered out)
 *
 * @example
 * const workflows = parseArrayFiltered(WorkflowSchema, rawWorkflows, 'Workflow');
 * // workflows contains only valid Workflow objects
 */
export function parseArrayFiltered<T>(
  schema: z.ZodSchema<T>,
  data: unknown[],
  context: string
): T[] {
  return data
    .map((item, index) => {
      const result = safeParse(schema, item, `${context}[${index}]`);
      return result.success ? result.data : null;
    })
    .filter((item): item is T => item !== null);
}

/**
 * Validation error class for use in components.
 * Extends Error with additional context for UI display.
 */
export class ValidationError extends Error {
  readonly context: string;
  readonly details?: z.ZodError;
  readonly raw?: unknown;

  constructor(message: string, context: string, details?: z.ZodError, raw?: unknown) {
    super(message);
    this.name = 'ValidationError';
    this.context = context;
    this.details = details;
    this.raw = raw;
  }
}

/**
 * Create a ValidationError from a ParseFailure result.
 */
export function toValidationError(failure: ParseFailure, context: string): ValidationError {
  return new ValidationError(failure.error, context, failure.details, failure.raw);
}
