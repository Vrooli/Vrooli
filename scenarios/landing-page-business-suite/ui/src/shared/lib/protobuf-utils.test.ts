import { afterEach, describe, expect, it, vi } from 'vitest';
import { normalizeTimestamp, normalizeTimestampOrFallback, normalizeTimestampOrNow } from './protobuf-utils';

describe('protobuf timestamp utilities', () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it('normalizes plain protobuf timestamps without nanos and rejects unknown values', () => {
    expect(normalizeTimestamp({ seconds: 1 })).toBe('1970-01-01T00:00:01.000Z');
    expect(normalizeTimestamp({ notATimestamp: true })).toBeUndefined();
  });

  it('uses explicit and clock-based fallbacks when a timestamp is absent', () => {
    expect(normalizeTimestampOrFallback(null, 'fallback')).toBe('fallback');

    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-30T12:00:00.000Z'));
    expect(normalizeTimestampOrNow(undefined)).toBe('2026-07-30T12:00:00.000Z');
  });
});
