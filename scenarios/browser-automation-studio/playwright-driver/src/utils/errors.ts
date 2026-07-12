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
  createError: (
    message: string,
    extractedData: Record<string, unknown>,
    context?: NormalizeContext
  ) => PlaywrightDriverError;
  /** Optional function to extract data from error message */
  extractData?: (message: string) => Record<string, unknown>;
}

/**
 * Optional context a caller can supply to improve classification.
 *
 * The most useful field is `selector`: instruction handlers already know which
 * selector they were operating on, so passing it through lets us produce a
 * precise SelectorNotFoundError even when the underlying Playwright message
 * uses an unexpected shape.
 */
export interface NormalizeContext {
  /** The selector the failing operation was targeting, if known. */
  selector?: string;
}

/** Max length of an original engine message we embed in a wrapped error. */
const MAX_DETAIL_LENGTH = 300;

/** Truncate an engine message so wrapped errors stay readable. */
function truncateDetail(detail: string): string {
  const trimmed = detail.trim();
  return trimmed.length > MAX_DETAIL_LENGTH
    ? `${trimmed.slice(0, MAX_DETAIL_LENGTH - 1)}…`
    : trimmed;
}

/**
 * Best-effort extraction of the selector from a raw Playwright error message.
 *
 * Playwright phrases selector failures several ways; we try the common shapes
 * and fall back to `undefined` when none match (callers then use context or a
 * placeholder). Extraction never throws.
 */
export function extractSelectorFromMessage(message: string): string | undefined {
  const patterns: RegExp[] = [
    // `... while parsing css selector "@selector/foo"`
    /parsing css selector\s+"([^"]+)"/i,
    // `waiting for locator('#foo')` / `waiting for locator("#foo")`
    /waiting for locator\(\s*['"]([^'"]+)['"]\s*\)/i,
    // `page.waitForSelector: ... selector "#foo"` / `selector: #foo`
    /selector[:\s]+["']([^"']+)["']/i,
    // `waiting for selector "#foo"`
    /waiting for selector\s+["']([^"']+)["']/i,
    // `waiting for #foo to be visible`
    /waiting for\s+(\S+)\s+to be (?:visible|hidden|attached|detached|enabled|editable)/i,
  ];

  for (const re of patterns) {
    const match = message.match(re);
    if (match?.[1]) {
      return match[1];
    }
  }

  return undefined;
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
    extractData: (message): Record<string, unknown> => {
      const match = message.match(/(\d+)\s*ms/i);
      const timeout = match?.[1];
      return { timeout: timeout ? parseInt(timeout, 10) : undefined };
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
    extractData: (message): Record<string, unknown> => ({
      selector: extractSelectorFromMessage(message),
    }),
    createError: (message, data, context) => {
      const selector = context?.selector ?? (data.selector as string | undefined) ?? 'unknown';
      return new SelectorNotFoundError(selector, undefined, message);
    },
  },
];

/**
 * Classify an error message using the pattern registry.
 * Returns the first matching error class or null if no match.
 */
function classifyErrorByPattern(
  message: string,
  context?: NormalizeContext
): PlaywrightDriverError | null {
  const lowerMessage = message.toLowerCase();

  for (const pattern of ERROR_PATTERNS) {
    const matches = pattern.patterns.some((p) => lowerMessage.includes(p));
    if (matches) {
      const extractedData = pattern.extractData?.(message) ?? {};
      return pattern.createError(message, extractedData, context);
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
  constructor(selector: string, timeout?: number, detail?: string) {
    const trimmedDetail = detail?.trim();
    super(
      `Selector not found: ${selector}${timeout ? ` (timeout: ${timeout}ms)` : ''}${
        trimmedDetail ? ` — ${truncateDetail(trimmedDetail)}` : ''
      }`,
      'SELECTOR_NOT_FOUND',
      FailureKind.ENGINE,
      true,
      { selector, timeout, detail: trimmedDetail || undefined }
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
export function normalizeError(error: unknown, context?: NormalizeContext): PlaywrightDriverError {
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
    const classified = classifyErrorByPattern(error.message, context);
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

  const firstIssue = issues[0];
  const message =
    issues.length === 1 && firstIssue
      ? `Validation error: ${firstIssue.path.join('.') || 'value'} - ${firstIssue.message}`
      : `Validation errors: ${issues.map((issue) => issue.path.join('.') || 'value').join(', ')}`;

  return new InvalidInstructionError(message, { zodIssues: issues });
}
