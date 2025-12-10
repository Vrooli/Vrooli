/**
 * Validation Utilities
 *
 * Hardening utilities for validating assumptions about data formats,
 * bounds, and system state. These guards help catch edge cases that
 * might otherwise cause silent failures or unexpected behavior.
 *
 * @module utils/validation
 */

import type { Page } from 'playwright';
import { logger } from './logger';

// UUID v4 regex pattern
const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

// Timeout bounds (ms)
export const MIN_TIMEOUT_MS = 100;
export const MAX_TIMEOUT_MS = 600_000; // 10 minutes

// Recording buffer limits
// NOTE: This is the default. For runtime configuration, use config.recording.maxBufferSize
// See config.ts for the RECORDING_MAX_BUFFER_SIZE environment variable
export const MAX_RECORDING_BUFFER_SIZE = 10_000;

/**
 * Validate that a string is a valid UUID v4 format.
 * Returns the validated UUID or throws if invalid.
 */
export function validateUUID(value: string, context: string): string {
  if (!value || typeof value !== 'string') {
    throw new Error(`${context}: UUID is required but got ${typeof value}`);
  }

  const trimmed = value.trim();
  if (!UUID_PATTERN.test(trimmed)) {
    throw new Error(`${context}: Invalid UUID format: ${trimmed.slice(0, 50)}`);
  }

  return trimmed;
}

/**
 * Check if a string looks like a valid UUID without throwing.
 */
export function isValidUUID(value: unknown): value is string {
  return typeof value === 'string' && UUID_PATTERN.test(value.trim());
}

/**
 * Validate and clamp a timeout value to safe bounds.
 * Logs a warning if clamping was necessary.
 */
export function validateTimeout(
  value: number | undefined,
  defaultValue: number,
  context: string
): number {
  if (value === undefined || value === null) {
    return defaultValue;
  }

  if (typeof value !== 'number' || isNaN(value)) {
    logger.warn('Invalid timeout value, using default', {
      context,
      value,
      default: defaultValue,
    });
    return defaultValue;
  }

  if (value < MIN_TIMEOUT_MS) {
    logger.warn('Timeout below minimum, clamping', {
      context,
      value,
      min: MIN_TIMEOUT_MS,
    });
    return MIN_TIMEOUT_MS;
  }

  if (value > MAX_TIMEOUT_MS) {
    logger.warn('Timeout above maximum, clamping', {
      context,
      value,
      max: MAX_TIMEOUT_MS,
    });
    return MAX_TIMEOUT_MS;
  }

  return Math.floor(value); // Ensure integer
}

/**
 * Validate step index is a non-negative integer.
 */
export function validateStepIndex(value: number | undefined, context: string): number {
  if (value === undefined || value === null) {
    return 0;
  }

  if (typeof value !== 'number' || isNaN(value) || !Number.isInteger(value) || value < 0) {
    logger.warn('Invalid step index, defaulting to 0', {
      context,
      value,
    });
    return 0;
  }

  return value;
}

/**
 * Calculate duration with safety checks.
 * Returns 0 if duration would be negative (clock skew).
 */
export function safeDuration(startedAt: Date, completedAt: Date): number {
  const duration = completedAt.getTime() - startedAt.getTime();

  if (duration < 0) {
    logger.warn('Negative duration detected (clock skew?)', {
      startedAt: startedAt.toISOString(),
      completedAt: completedAt.toISOString(),
      rawDuration: duration,
    });
    return 0;
  }

  return duration;
}

/**
 * Safely serialize a value to JSON, handling non-serializable types.
 * Returns the original value if serializable, or a sanitized version otherwise.
 */
export function safeSerializable<T>(value: T, context: string): T {
  if (value === undefined || value === null) {
    return value;
  }

  try {
    // Test serialization
    JSON.stringify(value);
    return value;
  } catch (error) {
    logger.warn('Non-serializable value detected, sanitizing', {
      context,
      error: error instanceof Error ? error.message : String(error),
      valueType: typeof value,
    });

    // Attempt to create a serializable version
    if (typeof value === 'object') {
      return JSON.parse(JSON.stringify(value, (_, v) => {
        if (typeof v === 'bigint') return v.toString();
        if (typeof v === 'function') return '[Function]';
        if (typeof v === 'symbol') return v.toString();
        return v;
      })) as T;
    }

    return String(value) as unknown as T;
  }
}

/**
 * Check if a Playwright page is still connected and usable.
 */
export async function isPageUsable(page: Page): Promise<boolean> {
  try {
    // Check if page is closed
    if (page.isClosed()) {
      return false;
    }

    // Try a lightweight operation to verify connection
    await page.evaluate(() => true);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validate that a page is usable before performing operations.
 * Throws a descriptive error if the page is not usable.
 */
export async function requireUsablePage(page: Page, context: string): Promise<void> {
  if (page.isClosed()) {
    throw new Error(`${context}: Page is closed`);
  }

  try {
    await page.evaluate(() => true);
  } catch (error) {
    throw new Error(
      `${context}: Page is not usable - ${error instanceof Error ? error.message : String(error)}`
    );
  }
}

/**
 * Validate params object exists and is an object.
 */
export function validateParams(params: unknown, context: string): Record<string, unknown> {
  if (params === undefined || params === null) {
    logger.debug('Params undefined, using empty object', { context });
    return {};
  }

  if (typeof params !== 'object' || Array.isArray(params)) {
    throw new Error(`${context}: Params must be an object, got ${typeof params}`);
  }

  return params as Record<string, unknown>;
}

/**
 * Normalize instruction type to lowercase for consistent lookup.
 */
export function normalizeInstructionType(type: string): string {
  if (!type || typeof type !== 'string') {
    throw new Error(`Instruction type is required but got: ${type}`);
  }
  return type.toLowerCase().trim();
}

/**
 * Type guard for valid console log types.
 */
export function isValidConsoleLogType(
  type: string
): type is 'log' | 'warn' | 'error' | 'info' | 'debug' {
  return ['log', 'warn', 'error', 'info', 'debug'].includes(type);
}

/**
 * Map Playwright console type to our supported types.
 */
export function normalizeConsoleLogType(msgType: string): 'log' | 'warn' | 'error' | 'info' | 'debug' {
  if (msgType === 'warning') return 'warn';
  if (isValidConsoleLogType(msgType)) return msgType;
  return 'log';
}

// ============================================================================
// Timestamp Validation for Idempotency & Replay Safety
// ============================================================================

/**
 * Maximum allowed clock drift between client and server (ms).
 * Requests with timestamps outside this window are considered stale or from the future.
 */
export const MAX_CLOCK_DRIFT_MS = 60_000; // 1 minute

/**
 * Maximum age for idempotency keys before they're considered expired (ms).
 * After this time, duplicate requests will be treated as new requests.
 */
export const IDEMPOTENCY_KEY_MAX_AGE_MS = 3600_000; // 1 hour

/**
 * Timestamp validation result.
 */
export interface TimestampValidation {
  valid: boolean;
  reason?: 'future' | 'stale' | 'invalid_format';
  adjustedTimestamp?: Date;
  driftMs?: number;
}

/**
 * Validate a client-provided timestamp against server time.
 *
 * Temporal hardening:
 * - Rejects timestamps too far in the future (potential replay attack)
 * - Accepts timestamps within clock drift tolerance
 * - Logs clock skew for monitoring
 *
 * @param timestamp - Client-provided timestamp (ISO 8601 string or Date)
 * @param maxAgeMs - Maximum age allowed for the timestamp (default: IDEMPOTENCY_KEY_MAX_AGE_MS)
 * @returns Validation result with adjusted timestamp if valid
 */
export function validateTimestamp(
  timestamp: string | Date | number,
  maxAgeMs: number = IDEMPOTENCY_KEY_MAX_AGE_MS
): TimestampValidation {
  const now = Date.now();

  // Parse timestamp
  let clientTime: number;
  try {
    if (timestamp instanceof Date) {
      clientTime = timestamp.getTime();
    } else if (typeof timestamp === 'number') {
      clientTime = timestamp;
    } else if (typeof timestamp === 'string') {
      clientTime = new Date(timestamp).getTime();
    } else {
      return { valid: false, reason: 'invalid_format' };
    }

    if (isNaN(clientTime)) {
      return { valid: false, reason: 'invalid_format' };
    }
  } catch {
    return { valid: false, reason: 'invalid_format' };
  }

  const drift = clientTime - now;

  // Reject timestamps too far in the future (with clock drift tolerance)
  if (drift > MAX_CLOCK_DRIFT_MS) {
    logger.warn('Timestamp from the future rejected', {
      clientTimestamp: new Date(clientTime).toISOString(),
      serverTime: new Date(now).toISOString(),
      driftMs: drift,
      maxDriftMs: MAX_CLOCK_DRIFT_MS,
    });
    return { valid: false, reason: 'future', driftMs: drift };
  }

  // Reject timestamps that are too old
  const age = now - clientTime;
  if (age > maxAgeMs) {
    logger.debug('Stale timestamp rejected', {
      clientTimestamp: new Date(clientTime).toISOString(),
      serverTime: new Date(now).toISOString(),
      ageMs: age,
      maxAgeMs,
    });
    return { valid: false, reason: 'stale', driftMs: drift };
  }

  // Log significant clock drift for monitoring
  if (Math.abs(drift) > 5000) {
    logger.info('Significant clock drift detected', {
      clientTimestamp: new Date(clientTime).toISOString(),
      serverTime: new Date(now).toISOString(),
      driftMs: drift,
      hint: 'Consider NTP synchronization on clients',
    });
  }

  return {
    valid: true,
    adjustedTimestamp: new Date(clientTime),
    driftMs: drift,
  };
}

/**
 * Generate a monotonic timestamp that accounts for clock skew.
 *
 * Uses a high-resolution timer combined with the wall clock to ensure
 * timestamps are always increasing within a process, even if the system
 * clock is adjusted backwards.
 */
let lastTimestamp = 0;
let lastHrTime = process.hrtime.bigint();

export function getMonotonicTimestamp(): Date {
  const now = Date.now();
  const hrDiff = Number(process.hrtime.bigint() - lastHrTime) / 1_000_000; // Convert to ms

  // If wall clock went backwards but hrtime advanced, use the calculated time
  if (now < lastTimestamp && hrDiff > 0) {
    const monotonic = Math.max(lastTimestamp + 1, Math.floor(lastTimestamp + hrDiff));
    lastTimestamp = monotonic;
    logger.debug('Clock went backwards, using monotonic time', {
      wallClock: now,
      monotonic,
      hrDiffMs: hrDiff,
    });
    return new Date(monotonic);
  }

  // Normal case: wall clock is advancing
  lastTimestamp = now;
  lastHrTime = process.hrtime.bigint();
  return new Date(now);
}

/**
 * Check if a timestamp is recent enough to be from an active request.
 * Used to detect retries vs new requests.
 *
 * @param timestamp - Timestamp to check
 * @param maxAgeMs - Maximum age to consider "recent"
 * @returns true if timestamp is recent
 */
export function isRecentTimestamp(timestamp: Date | string | number, maxAgeMs: number = 30_000): boolean {
  const validation = validateTimestamp(timestamp, maxAgeMs);
  return validation.valid;
}

/**
 * Idempotency key metadata for tracking.
 */
export interface IdempotencyKeyMetadata {
  key: string;
  createdAt: number;
  expiresAt: number;
  result?: unknown;
}

/**
 * Create an idempotency key with expiration tracking.
 *
 * @param key - The idempotency key
 * @param ttlMs - Time to live in milliseconds
 * @returns Metadata for the idempotency key
 */
export function createIdempotencyKey(
  key: string,
  ttlMs: number = IDEMPOTENCY_KEY_MAX_AGE_MS
): IdempotencyKeyMetadata {
  const now = Date.now();
  return {
    key,
    createdAt: now,
    expiresAt: now + ttlMs,
  };
}

/**
 * Check if an idempotency key is still valid (not expired).
 */
export function isIdempotencyKeyValid(metadata: IdempotencyKeyMetadata): boolean {
  return Date.now() < metadata.expiresAt;
}
