import { FailureKind } from '../proto';
import { ZodError } from 'zod';

/**
 * Error Classification Module
 *
 * This module provides a unified error handling system for the playwright driver.
 * All error classification decisions are made through explicit pattern matching,
 * making it easy to understand and extend.
 *
 * ERROR CODE REFERENCE (STABLE - can be relied upon for automation):
 * - SESSION_NOT_FOUND: Session ID doesn't exist (create a new session)
 * - INVALID_INSTRUCTION: Malformed instruction (fix the instruction)
 * - UNSUPPORTED_INSTRUCTION: Unknown instruction type (check supported types)
 * - SELECTOR_NOT_FOUND: Element not found (verify selector, consider wait)
 * - TIMEOUT: Operation timed out (retry or increase timeout)
 * - NAVIGATION_ERROR: Page navigation failed (check URL, network)
 * - RESOURCE_LIMIT: System limit reached (wait and retry later)
 * - CONFIGURATION_ERROR: Invalid configuration (fix config)
 * - FRAME_NOT_FOUND: Frame not found (verify frame exists)
 * - PLAYWRIGHT_ERROR: Generic Playwright error (check logs)
 * - UNKNOWN_ERROR: Unexpected error (check logs, report if persistent)
 *
 * ERROR KIND SEMANTICS:
 * - 'engine': Error in browser engine or driver - often retryable
 * - 'orchestration': Invalid request from caller - fix the input
 * - 'infra': Resource or infrastructure issue - check system limits
 * - 'timeout': Operation timed out - may succeed with longer timeout or retry
 * - 'user': User-initiated cancellation
 * - 'cancelled': Operation was cancelled
 */

// =============================================================================
// ERROR PATTERN REGISTRY
// =============================================================================

/**
 * Pattern definition for error classification.
 *
 * Each pattern defines:
 * - patterns: String fragments to match in error messages (case-insensitive)
 * - errorClass: Factory function to create the specific error
 * - extractData: Optional function to extract data from the error message
 */
interface ErrorPattern {
  /** String patterns to match in error message (matched case-insensitive, any match triggers) */
  patterns: string[];
  /** Factory to create the appropriate error */
  createError: (message: string, extractedData: Record<string, unknown>) => PlaywrightDriverError;
  /** Optional function to extract data from error message */
  extractData?: (message: string) => Record<string, unknown>;
}

/**
 * Pattern registry for error classification.
 *
 * Order matters: first matching pattern wins.
 * More specific patterns should come before generic ones.
 */
const ERROR_PATTERNS: ErrorPattern[] = [
  // Timeout errors (most common, check first)
  {
    patterns: ['timeout', 'exceeded'],
    createError: (message, data) => new TimeoutError(message, (data.timeout as number) ?? 0),
    extractData: (message) => {
      const match = message.match(/(\d+)\s*ms/i);
      return { timeout: match ? parseInt(match[1], 10) : undefined };
    },
  },

  // Frame errors (before generic selector errors)
  {
    patterns: ['frame'],
    createError: (message) => new FrameNotFoundError(undefined, undefined, message),
  },

  // Navigation errors
  {
    patterns: ['navigation', 'navigate'],
    createError: (message) => new NavigationError(message),
  },

  // Selector/element errors (broad category)
  {
    patterns: ['selector', 'element', 'locator', 'waiting for'],
    createError: (_message) => new SelectorNotFoundError('unknown'),
  },
];

/**
 * Classify an error message using the pattern registry.
 * Returns the first matching error class or null if no match.
 */
function classifyErrorByPattern(message: string): PlaywrightDriverError | null {
  const lowerMessage = message.toLowerCase();

  for (const pattern of ERROR_PATTERNS) {
    const matches = pattern.patterns.some((p) => lowerMessage.includes(p));
    if (matches) {
      const extractedData = pattern.extractData?.(message) ?? {};
      return pattern.createError(message, extractedData);
    }
  }

  return null;
}

/**
 * Get the current error pattern registry (read-only).
 * Useful for testing and introspection.
 */
export function getErrorPatterns(): readonly ErrorPattern[] {
  return ERROR_PATTERNS;
}

/**
 * Export the pattern type for custom pattern creation.
 */
export type { ErrorPattern };

// =============================================================================
// ERROR CLASSES
// =============================================================================

/**
 * Base error for all playwright driver errors
 */
export class PlaywrightDriverError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly kind: FailureKind = FailureKind.ENGINE,
    public readonly retryable: boolean = false,
    public readonly details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'PlaywrightDriverError';
  }
}

/**
 * Session not found
 */
export class SessionNotFoundError extends PlaywrightDriverError {
  constructor(sessionId: string) {
    super(`Session not found: ${sessionId}`, 'SESSION_NOT_FOUND', FailureKind.ENGINE, false, {
      sessionId,
    });
    this.name = 'SessionNotFoundError';
  }
}

/**
 * Invalid instruction
 */
export class InvalidInstructionError extends PlaywrightDriverError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 'INVALID_INSTRUCTION', FailureKind.ORCHESTRATION, false, details);
    this.name = 'InvalidInstructionError';
  }
}

/**
 * Unsupported instruction type
 */
export class UnsupportedInstructionError extends PlaywrightDriverError {
  constructor(type: string) {
    super(
      `Unsupported instruction type: ${type}`,
      'UNSUPPORTED_INSTRUCTION',
      FailureKind.ORCHESTRATION,
      false,
      { type }
    );
    this.name = 'UnsupportedInstructionError';
  }
}

/**
 * Selector not found
 */
export class SelectorNotFoundError extends PlaywrightDriverError {
  constructor(selector: string, timeout?: number) {
    super(
      `Selector not found: ${selector}${timeout ? ` (timeout: ${timeout}ms)` : ''}`,
      'SELECTOR_NOT_FOUND',
      FailureKind.ENGINE,
      true,
      { selector, timeout }
    );
    this.name = 'SelectorNotFoundError';
  }
}

/**
 * Timeout error
 */
export class TimeoutError extends PlaywrightDriverError {
  constructor(message: string, timeout: number) {
    super(message, 'TIMEOUT', FailureKind.TIMEOUT, true, { timeout });
    this.name = 'TimeoutError';
  }
}

/**
 * Navigation error
 */
export class NavigationError extends PlaywrightDriverError {
  constructor(message: string, url?: string) {
    super(message, 'NAVIGATION_ERROR', FailureKind.ENGINE, true, { url });
    this.name = 'NavigationError';
  }
}

/**
 * Resource limit error
 */
export class ResourceLimitError extends PlaywrightDriverError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 'RESOURCE_LIMIT', FailureKind.INFRA, false, details);
    this.name = 'ResourceLimitError';
  }
}

/**
 * Configuration error
 */
export class ConfigurationError extends PlaywrightDriverError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 'CONFIGURATION_ERROR', FailureKind.ORCHESTRATION, false, details);
    this.name = 'ConfigurationError';
  }
}

/**
 * Frame not found error
 */
export class FrameNotFoundError extends PlaywrightDriverError {
  constructor(selector?: string, frameId?: string, frameUrl?: string) {
    super(
      `Frame not found${selector ? `: selector=${selector}` : ''}${frameId ? `, frameId=${frameId}` : ''}${frameUrl ? `, url=${frameUrl}` : ''}`,
      'FRAME_NOT_FOUND',
      FailureKind.ENGINE,
      true,
      { selector, frameId, frameUrl }
    );
    this.name = 'FrameNotFoundError';
  }
}

// =============================================================================
// ERROR NORMALIZATION
// =============================================================================

/**
 * Convert any error to a PlaywrightDriverError.
 *
 * Uses the ERROR_PATTERNS registry for consistent, explicit classification.
 *
 * Classification order:
 * 1. PlaywrightDriverError - pass through as-is
 * 2. ZodError - convert to InvalidInstructionError with validation details
 * 3. Error with classifiable message - use ERROR_PATTERNS registry
 * 4. Error without match - wrap as PLAYWRIGHT_ERROR
 * 5. Non-Error values - wrap as UNKNOWN_ERROR
 */
export function normalizeError(error: unknown): PlaywrightDriverError {
  // 1. Already a PlaywrightDriverError - pass through
  if (error instanceof PlaywrightDriverError) {
    return error;
  }

  // 2. Zod validation errors - convert with details
  if (error instanceof ZodError) {
    return normalizeZodError(error);
  }

  // 3. Standard Error objects - classify by message
  if (error instanceof Error) {
    // Try pattern-based classification first
    const classified = classifyErrorByPattern(error.message);
    if (classified) {
      return classified;
    }

    // No pattern match - wrap as generic Playwright error
    return new PlaywrightDriverError(
      error.message,
      'PLAYWRIGHT_ERROR',
      FailureKind.ENGINE,
      false
    );
  }

  // 4. Non-Error values - wrap as unknown
  return new PlaywrightDriverError(
    'Unknown error occurred',
    'UNKNOWN_ERROR',
    FailureKind.ENGINE,
    false,
    { error }
  );
}

/**
 * Convert ZodError to InvalidInstructionError with structured details.
 */
function normalizeZodError(error: ZodError): InvalidInstructionError {
  if (error.issues.length === 0) {
    return new InvalidInstructionError('Validation failed', { zodIssues: [] });
  }

  const issues = error.issues.map((issue) => ({
    path: issue.path,
    message: issue.message,
    code: issue.code,
  }));

  const message =
    issues.length === 1
      ? `Validation error: ${issues[0].path.join('.') || 'value'} - ${issues[0].message}`
      : `Validation errors: ${issues.map((issue) => issue.path.join('.') || 'value').join(', ')}`;

  return new InvalidInstructionError(message, { zodIssues: issues });
}
