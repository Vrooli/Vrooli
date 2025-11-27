import type { FailureKind } from '../types';

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
 * Convert Playwright error to driver error
 */
export function normalizeError(error: unknown): PlaywrightDriverError {
  if (error instanceof PlaywrightDriverError) {
    return error;
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
