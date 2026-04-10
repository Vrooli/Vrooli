/**
 * Utility functions for handling protobuf types in the UI layer.
 */

/**
 * Timestamp-like object with toJsonString method (from @bufbuild/protobuf).
 */
interface BufTimestamp {
  toJsonString(): string;
}

/**
 * Plain timestamp object with seconds and optional nanos.
 */
interface PlainTimestamp {
  seconds: number;
  nanos?: number;
}

/**
 * Normalize various timestamp formats to ISO string.
 * Handles:
 * - Buf Timestamp objects with toJsonString() method
 * - Plain objects with seconds/nanos (protobuf Timestamp)
 * - Date instances
 * - ISO date strings
 * - null/undefined (returns undefined)
 *
 * @param value - The timestamp value to normalize
 * @returns ISO date string or undefined if value is falsy
 */
export function normalizeTimestamp(value: unknown): string | undefined {
  if (!value) {
    return undefined;
  }

  // Buf Timestamp with toJsonString method
  if (typeof (value as BufTimestamp).toJsonString === 'function') {
    return (value as BufTimestamp).toJsonString();
  }

  // Already a string (ISO date)
  if (typeof value === 'string') {
    return value;
  }

  // Date instance
  if (value instanceof Date) {
    return value.toISOString();
  }

  // Plain object with seconds/nanos (protobuf Timestamp)
  const maybeTimestamp = value as PlainTimestamp;
  if (typeof maybeTimestamp.seconds === 'number') {
    const milliseconds =
      maybeTimestamp.seconds * 1000 + (maybeTimestamp.nanos ? maybeTimestamp.nanos / 1_000_000 : 0);
    return new Date(milliseconds).toISOString();
  }

  return undefined;
}

/**
 * Normalize a timestamp with a fallback value.
 *
 * @param value - The timestamp value to normalize
 * @param fallback - Fallback value if normalization returns undefined
 * @returns Normalized timestamp string or fallback
 */
export function normalizeTimestampOrFallback(value: unknown, fallback: string): string {
  return normalizeTimestamp(value) ?? fallback;
}

/**
 * Normalize a timestamp with current time as fallback.
 *
 * @param value - The timestamp value to normalize
 * @returns Normalized timestamp string or current time
 */
export function normalizeTimestampOrNow(value: unknown): string {
  return normalizeTimestamp(value) ?? new Date().toISOString();
}
