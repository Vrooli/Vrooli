/**
 * Centralized timestamp utilities for converting proto Timestamps to JavaScript Dates.
 *
 * Proto Timestamp uses:
 * - seconds: bigint (seconds since Unix epoch)
 * - nanos: number (nanoseconds within the second)
 *
 * These utilities provide consistent conversion across the UI codebase.
 */

import type { Timestamp } from '@bufbuild/protobuf/wkt';

/**
 * Proto Timestamp shape - supports both Timestamp WKT and raw proto objects.
 * The bufbuild/protobuf library uses bigint for seconds.
 */
export interface ProtoTimestamp {
  seconds?: bigint | number;
  nanos?: number;
}

/**
 * Convert a proto Timestamp to a JavaScript Date.
 *
 * Handles both:
 * - Timestamp WKT from @bufbuild/protobuf (seconds as bigint)
 * - Raw proto objects (seconds as number)
 *
 * @param value - Proto Timestamp or undefined/null
 * @returns JavaScript Date or undefined if value is falsy or invalid
 *
 * @example
 * ```typescript
 * const date = protoTimestampToDate(timeline.startedAt);
 * if (date) {
 *   console.log(date.toISOString());
 * }
 * ```
 */
export function protoTimestampToDate(
  value?: ProtoTimestamp | Timestamp | null
): Date | undefined {
  if (!value) return undefined;

  // Handle both bigint and number for seconds
  const seconds = typeof value.seconds === 'bigint'
    ? Number(value.seconds)
    : Number(value.seconds ?? 0);

  const millis = Math.floor(Number(value.nanos ?? 0) / 1_000_000);
  const result = new Date(seconds * 1000 + millis);

  // Return undefined for invalid dates
  return Number.isNaN(result.valueOf()) ? undefined : result;
}

/**
 * Convert a proto Timestamp to an ISO string.
 *
 * Useful for serialization and logging.
 *
 * @param value - Proto Timestamp or undefined/null
 * @returns ISO date string or undefined if value is falsy or invalid
 *
 * @example
 * ```typescript
 * const iso = protoTimestampToISOString(entry.timestamp);
 * // "2024-01-15T12:30:45.123Z"
 * ```
 */
export function protoTimestampToISOString(
  value?: ProtoTimestamp | Timestamp | null
): string | undefined {
  const date = protoTimestampToDate(value);
  return date?.toISOString();
}

/**
 * Parse various timestamp formats to a Date.
 *
 * Handles:
 * - Date objects (passed through)
 * - ISO strings
 * - Unix timestamps (number)
 * - Proto Timestamps
 *
 * @param value - Timestamp in various formats
 * @returns JavaScript Date or undefined if parsing fails
 *
 * @example
 * ```typescript
 * const date = parseTimestamp(event.timestamp);
 * ```
 */
export function parseTimestamp(
  value: Date | string | number | ProtoTimestamp | Timestamp | undefined | null
): Date | undefined {
  if (!value) return undefined;

  // Already a Date
  if (value instanceof Date) {
    return Number.isNaN(value.valueOf()) ? undefined : value;
  }

  // Proto Timestamp (has seconds field)
  if (typeof value === 'object' && 'seconds' in value) {
    return protoTimestampToDate(value);
  }

  // String or number - try parsing
  const parsed = new Date(value as string | number);
  return Number.isNaN(parsed.valueOf()) ? undefined : parsed;
}

// Re-export for backwards compatibility with existing imports
// These are aliases to the new function names
export { protoTimestampToDate as timestampToDate };
