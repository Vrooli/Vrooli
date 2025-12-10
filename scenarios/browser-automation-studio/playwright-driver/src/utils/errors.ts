import type { FailureKind } from '../types';
import { ZodError } from 'zod';

/**
 * Error codes for playwright driver.
 *
 * These codes are STABLE and can be relied upon for automation and error handling.
 * Each code maps to a specific error class and has clear semantics:
 *
 * Error Kind Semantics:
 * - 'engine': Error in the browser engine or driver - often retryable
 * - 'orchestration': Invalid request from the caller - fix the input
 * - 'infra': Resource or infrastructure issue - check system limits
 * - 'timeout': Operation timed out - may succeed with longer timeout or retry
 * - 'user': User-initiated cancellation
 * - 'cancelled': Operation was cancelled
 *
 * Error Code Reference:
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
 */

/**
 * Base error for all playwright driver errors
 */
export class PlaywrightDriverError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly kind: FailureKind = 'engine',
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
    super(`Session not found: ${sessionId}`, 'SESSION_NOT_FOUND', 'engine', false, {
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
    super(message, 'INVALID_INSTRUCTION', 'orchestration', false, details);
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
      'orchestration',
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
      'engine',
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
    super(message, 'TIMEOUT', 'timeout', true, { timeout });
    this.name = 'TimeoutError';
  }
}

/**
 * Navigation error
 */
export class NavigationError extends PlaywrightDriverError {
  constructor(message: string, url?: string) {
    super(message, 'NAVIGATION_ERROR', 'engine', true, { url });
    this.name = 'NavigationError';
  }
}

/**
 * Resource limit error
 */
export class ResourceLimitError extends PlaywrightDriverError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 'RESOURCE_LIMIT', 'infra', false, details);
    this.name = 'ResourceLimitError';
  }
}

/**
 * Configuration error
 */
export class ConfigurationError extends PlaywrightDriverError {
  constructor(message: string, details?: Record<string, unknown>) {
    super(message, 'CONFIGURATION_ERROR', 'orchestration', false, details);
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
      'engine',
      true,
      { selector, frameId, frameUrl }
    );
    this.name = 'FrameNotFoundError';
  }
}

/**
 * Format Zod validation errors into a human-readable message.
 * Groups errors by path for cleaner output.
 */
function formatZodError(error: ZodError): string {
  const issues = error.issues;
  if (issues.length === 0) {
    return 'Validation failed';
  }

  if (issues.length === 1) {
    const issue = issues[0];
    const path = issue.path.length > 0 ? issue.path.join('.') : 'value';
    return `${path}: ${issue.message}`;
  }

  // Multiple issues - format as list
  const formatted = issues.map((issue) => {
    const path = issue.path.length > 0 ? issue.path.join('.') : 'value';
    return `${path}: ${issue.message}`;
  });

  return `Validation errors: ${formatted.join('; ')}`;
}

/**
 * Convert Playwright error to driver error
 *
 * Handles various error types:
 * - PlaywrightDriverError: Pass through as-is
 * - ZodError: Convert to InvalidInstructionError with formatted message
 * - Playwright-specific errors: Map based on message content
 * - Generic errors: Wrap as PLAYWRIGHT_ERROR
 * - Unknown values: Wrap as UNKNOWN_ERROR
 */
export function normalizeError(error: unknown): PlaywrightDriverError {
  if (error instanceof PlaywrightDriverError) {
    return error;
  }

  // Handle Zod validation errors specifically
  if (error instanceof ZodError) {
    const message = formatZodError(error);
    return new InvalidInstructionError(message, {
      zodIssues: error.issues.map((issue) => ({
        path: issue.path,
        message: issue.message,
        code: issue.code,
      })),
    });
  }

  if (error instanceof Error) {
    const message = error.message.toLowerCase();

    // Timeout errors
    if (message.includes('timeout') || message.includes('exceeded')) {
      return new TimeoutError(error.message, 30000);
    }

    // Selector errors
    if (
      message.includes('selector') ||
      message.includes('element') ||
      message.includes('locator')
    ) {
      return new SelectorNotFoundError('unknown', undefined);
    }

    // Navigation errors
    if (message.includes('navigation') || message.includes('navigate')) {
      return new NavigationError(error.message);
    }

    // Frame errors
    if (message.includes('frame')) {
      return new FrameNotFoundError();
    }

    // Generic error
    return new PlaywrightDriverError(error.message, 'PLAYWRIGHT_ERROR', 'engine', false);
  }

  // Unknown error
  return new PlaywrightDriverError(
    'Unknown error occurred',
    'UNKNOWN_ERROR',
    'engine',
    false,
    { error }
  );
}
