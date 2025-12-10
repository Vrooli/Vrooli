/**
 * Validation Utilities Tests
 *
 * Tests for hardening utilities that validate assumptions about
 * data formats, bounds, and system state.
 */
import {
  validateUUID,
  isValidUUID,
  validateTimeout,
  validateStepIndex,
  safeDuration,
  safeSerializable,
  validateParams,
  normalizeInstructionType,
  isValidConsoleLogType,
  normalizeConsoleLogType,
  validateTimestamp,
  getMonotonicTimestamp,
  isRecentTimestamp,
  createIdempotencyKey,
  isIdempotencyKeyValid,
  MIN_TIMEOUT_MS,
  MAX_TIMEOUT_MS,
  MAX_CLOCK_DRIFT_MS,
  IDEMPOTENCY_KEY_MAX_AGE_MS,
} from '../../../src/utils/validation';

// Mock the logger to prevent console output during tests
jest.mock('../../../src/utils/logger', () => ({
  logger: {
    warn: jest.fn(),
    debug: jest.fn(),
    info: jest.fn(),
    error: jest.fn(),
  },
}));

describe('validateUUID', () => {
  it('should accept valid UUID v4', () => {
    const uuid = '550e8400-e29b-41d4-a716-446655440000';
    expect(validateUUID(uuid, 'test')).toBe(uuid);
  });

  it('should accept UUID with uppercase letters', () => {
    const uuid = '550E8400-E29B-41D4-A716-446655440000';
    expect(validateUUID(uuid, 'test')).toBe(uuid);
  });

  it('should trim whitespace from UUID', () => {
    const uuid = '  550e8400-e29b-41d4-a716-446655440000  ';
    expect(validateUUID(uuid, 'test')).toBe(uuid.trim());
  });

  it('should reject empty string', () => {
    expect(() => validateUUID('', 'test')).toThrow('UUID is required');
  });

  it('should reject null/undefined', () => {
    expect(() => validateUUID(null as unknown as string, 'test')).toThrow('UUID is required');
    expect(() => validateUUID(undefined as unknown as string, 'test')).toThrow('UUID is required');
  });

  it('should reject invalid format', () => {
    expect(() => validateUUID('not-a-uuid', 'test')).toThrow('Invalid UUID format');
    expect(() => validateUUID('550e8400-e29b-31d4-a716-446655440000', 'test')).toThrow('Invalid UUID format'); // v3
  });
});

describe('isValidUUID', () => {
  it('should return true for valid UUIDs', () => {
    expect(isValidUUID('550e8400-e29b-41d4-a716-446655440000')).toBe(true);
  });

  it('should return false for invalid values', () => {
    expect(isValidUUID('not-a-uuid')).toBe(false);
    expect(isValidUUID('')).toBe(false);
    expect(isValidUUID(null)).toBe(false);
    expect(isValidUUID(undefined)).toBe(false);
    expect(isValidUUID(123)).toBe(false);
  });
});

describe('validateTimeout', () => {
  it('should return default for undefined', () => {
    expect(validateTimeout(undefined, 30000, 'test')).toBe(30000);
  });

  it('should return default for NaN', () => {
    expect(validateTimeout(NaN, 30000, 'test')).toBe(30000);
  });

  it('should clamp values below minimum', () => {
    expect(validateTimeout(50, 30000, 'test')).toBe(MIN_TIMEOUT_MS);
  });

  it('should clamp values above maximum', () => {
    expect(validateTimeout(999999999, 30000, 'test')).toBe(MAX_TIMEOUT_MS);
  });

  it('should accept valid timeout values', () => {
    expect(validateTimeout(5000, 30000, 'test')).toBe(5000);
  });

  it('should floor decimal values', () => {
    expect(validateTimeout(5000.7, 30000, 'test')).toBe(5000);
  });
});

describe('validateStepIndex', () => {
  it('should return 0 for undefined', () => {
    expect(validateStepIndex(undefined, 'test')).toBe(0);
  });

  it('should return 0 for negative values', () => {
    expect(validateStepIndex(-1, 'test')).toBe(0);
  });

  it('should return 0 for non-integers', () => {
    expect(validateStepIndex(1.5, 'test')).toBe(0);
  });

  it('should return 0 for NaN', () => {
    expect(validateStepIndex(NaN, 'test')).toBe(0);
  });

  it('should accept valid step indices', () => {
    expect(validateStepIndex(0, 'test')).toBe(0);
    expect(validateStepIndex(5, 'test')).toBe(5);
    expect(validateStepIndex(100, 'test')).toBe(100);
  });
});

describe('safeDuration', () => {
  it('should calculate positive duration', () => {
    const start = new Date('2024-01-01T00:00:00Z');
    const end = new Date('2024-01-01T00:00:05Z');
    expect(safeDuration(start, end)).toBe(5000);
  });

  it('should return 0 for negative duration (clock skew)', () => {
    const start = new Date('2024-01-01T00:00:05Z');
    const end = new Date('2024-01-01T00:00:00Z');
    expect(safeDuration(start, end)).toBe(0);
  });

  it('should return 0 for zero duration', () => {
    const time = new Date('2024-01-01T00:00:00Z');
    expect(safeDuration(time, time)).toBe(0);
  });
});

describe('safeSerializable', () => {
  it('should pass through null/undefined', () => {
    expect(safeSerializable(null, 'test')).toBeNull();
    expect(safeSerializable(undefined, 'test')).toBeUndefined();
  });

  it('should pass through serializable objects', () => {
    const obj = { key: 'value', nested: { a: 1 } };
    expect(safeSerializable(obj, 'test')).toEqual(obj);
  });

  it('should handle BigInt by converting to string', () => {
    const obj = { big: BigInt(123) };
    const result = safeSerializable(obj, 'test');
    expect(result).toEqual({ big: '123' });
  });

  it('should pass through objects with functions (functions become undefined in JSON)', () => {
    // Note: Functions in objects don't cause JSON.stringify to throw,
    // they just become undefined. This test documents actual behavior.
    const obj = { fn: () => {}, value: 'test' };
    const result = safeSerializable(obj, 'test');
    // Function is preserved in-memory since stringify doesn't throw
    expect(result).toEqual(obj);
  });
});

describe('validateParams', () => {
  it('should return empty object for null/undefined', () => {
    expect(validateParams(null, 'test')).toEqual({});
    expect(validateParams(undefined, 'test')).toEqual({});
  });

  it('should throw for array', () => {
    expect(() => validateParams([], 'test')).toThrow('Params must be an object');
  });

  it('should throw for primitive types', () => {
    expect(() => validateParams('string', 'test')).toThrow('Params must be an object');
    expect(() => validateParams(123, 'test')).toThrow('Params must be an object');
  });

  it('should accept valid objects', () => {
    const params = { key: 'value' };
    expect(validateParams(params, 'test')).toEqual(params);
  });
});

describe('normalizeInstructionType', () => {
  it('should convert to lowercase', () => {
    expect(normalizeInstructionType('CLICK')).toBe('click');
    expect(normalizeInstructionType('Navigate')).toBe('navigate');
  });

  it('should trim whitespace', () => {
    expect(normalizeInstructionType('  click  ')).toBe('click');
  });

  it('should throw for empty string', () => {
    expect(() => normalizeInstructionType('')).toThrow('Instruction type is required');
  });

  it('should throw for null/undefined', () => {
    expect(() => normalizeInstructionType(null as unknown as string)).toThrow();
    expect(() => normalizeInstructionType(undefined as unknown as string)).toThrow();
  });
});

describe('isValidConsoleLogType', () => {
  it('should return true for valid types', () => {
    expect(isValidConsoleLogType('log')).toBe(true);
    expect(isValidConsoleLogType('warn')).toBe(true);
    expect(isValidConsoleLogType('error')).toBe(true);
    expect(isValidConsoleLogType('info')).toBe(true);
    expect(isValidConsoleLogType('debug')).toBe(true);
  });

  it('should return false for invalid types', () => {
    expect(isValidConsoleLogType('warning')).toBe(false);
    expect(isValidConsoleLogType('trace')).toBe(false);
    expect(isValidConsoleLogType('')).toBe(false);
  });
});

describe('normalizeConsoleLogType', () => {
  it('should convert warning to warn', () => {
    expect(normalizeConsoleLogType('warning')).toBe('warn');
  });

  it('should pass through valid types', () => {
    expect(normalizeConsoleLogType('log')).toBe('log');
    expect(normalizeConsoleLogType('error')).toBe('error');
  });

  it('should default unknown types to log', () => {
    expect(normalizeConsoleLogType('trace')).toBe('log');
    expect(normalizeConsoleLogType('unknown')).toBe('log');
  });
});

// ============================================================================
// Timestamp Validation Tests for Idempotency & Replay Safety
// ============================================================================

describe('validateTimestamp', () => {
  it('should accept timestamps within clock drift tolerance', () => {
    const now = Date.now();
    const result = validateTimestamp(now);

    expect(result.valid).toBe(true);
    expect(result.adjustedTimestamp).toBeDefined();
    expect(result.driftMs).toBeDefined();
    expect(Math.abs(result.driftMs!)).toBeLessThan(1000); // Within 1 second
  });

  it('should accept ISO 8601 string timestamps', () => {
    const now = new Date();
    const result = validateTimestamp(now.toISOString());

    expect(result.valid).toBe(true);
    expect(result.adjustedTimestamp).toBeInstanceOf(Date);
  });

  it('should accept Date objects', () => {
    const now = new Date();
    const result = validateTimestamp(now);

    expect(result.valid).toBe(true);
    expect(result.adjustedTimestamp).toBeInstanceOf(Date);
  });

  it('should reject timestamps too far in the future', () => {
    const futureTime = Date.now() + MAX_CLOCK_DRIFT_MS + 10000;
    const result = validateTimestamp(futureTime);

    expect(result.valid).toBe(false);
    expect(result.reason).toBe('future');
    expect(result.driftMs).toBeGreaterThan(MAX_CLOCK_DRIFT_MS);
  });

  it('should reject stale timestamps', () => {
    const staleTime = Date.now() - IDEMPOTENCY_KEY_MAX_AGE_MS - 10000;
    const result = validateTimestamp(staleTime);

    expect(result.valid).toBe(false);
    expect(result.reason).toBe('stale');
  });

  it('should respect custom maxAgeMs parameter', () => {
    const slightlyOld = Date.now() - 5000;

    // Should be valid with default maxAge
    expect(validateTimestamp(slightlyOld).valid).toBe(true);

    // Should be stale with 1 second maxAge
    expect(validateTimestamp(slightlyOld, 1000).valid).toBe(false);
  });

  it('should reject invalid format', () => {
    expect(validateTimestamp('not-a-date').valid).toBe(false);
    expect(validateTimestamp('not-a-date').reason).toBe('invalid_format');
  });

  it('should handle edge cases', () => {
    // @ts-expect-error - testing invalid input
    expect(validateTimestamp(null).valid).toBe(false);
    // @ts-expect-error - testing invalid input
    expect(validateTimestamp({}).valid).toBe(false);
  });
});

describe('getMonotonicTimestamp', () => {
  it('should return a Date object', () => {
    const result = getMonotonicTimestamp();
    expect(result).toBeInstanceOf(Date);
  });

  it('should return timestamps that are close to current time', () => {
    const before = Date.now();
    const result = getMonotonicTimestamp();
    const after = Date.now();

    expect(result.getTime()).toBeGreaterThanOrEqual(before - 1);
    expect(result.getTime()).toBeLessThanOrEqual(after + 1);
  });

  it('should return increasing timestamps for consecutive calls', () => {
    const timestamps: number[] = [];

    for (let i = 0; i < 10; i++) {
      timestamps.push(getMonotonicTimestamp().getTime());
    }

    // Each timestamp should be >= the previous one
    for (let i = 1; i < timestamps.length; i++) {
      expect(timestamps[i]).toBeGreaterThanOrEqual(timestamps[i - 1]);
    }
  });
});

describe('isRecentTimestamp', () => {
  it('should return true for timestamps within default threshold', () => {
    const now = Date.now();
    expect(isRecentTimestamp(now)).toBe(true);
    expect(isRecentTimestamp(new Date())).toBe(true);
    expect(isRecentTimestamp(new Date().toISOString())).toBe(true);
  });

  it('should return false for old timestamps', () => {
    const oldTime = Date.now() - 60000; // 1 minute ago
    expect(isRecentTimestamp(oldTime, 30000)).toBe(false);
  });

  it('should respect custom maxAgeMs', () => {
    const fiveSecondsAgo = Date.now() - 5000;

    // 5 seconds ago is NOT recent with 1 second threshold
    expect(isRecentTimestamp(fiveSecondsAgo, 1000)).toBe(false);

    // 5 seconds ago IS recent with 10 second threshold
    expect(isRecentTimestamp(fiveSecondsAgo, 10000)).toBe(true);
  });
});

describe('createIdempotencyKey', () => {
  it('should create metadata with correct key', () => {
    const key = 'test-key-123';
    const metadata = createIdempotencyKey(key);

    expect(metadata.key).toBe(key);
    expect(metadata.createdAt).toBeDefined();
    expect(metadata.expiresAt).toBeDefined();
  });

  it('should set expiration based on TTL', () => {
    const key = 'test-key';
    const ttlMs = 60000; // 1 minute

    const before = Date.now();
    const metadata = createIdempotencyKey(key, ttlMs);
    const after = Date.now();

    // Expires should be TTL milliseconds after creation
    expect(metadata.expiresAt - metadata.createdAt).toBe(ttlMs);
    expect(metadata.createdAt).toBeGreaterThanOrEqual(before);
    expect(metadata.createdAt).toBeLessThanOrEqual(after);
  });

  it('should use default TTL when not specified', () => {
    const metadata = createIdempotencyKey('test');

    expect(metadata.expiresAt - metadata.createdAt).toBe(IDEMPOTENCY_KEY_MAX_AGE_MS);
  });
});

describe('isIdempotencyKeyValid', () => {
  it('should return true for non-expired keys', () => {
    const metadata = createIdempotencyKey('test', 60000);
    expect(isIdempotencyKeyValid(metadata)).toBe(true);
  });

  it('should return false for expired keys', () => {
    // Create a key that expired 1 second ago
    const metadata = {
      key: 'expired-test',
      createdAt: Date.now() - 10000,
      expiresAt: Date.now() - 1000,
    };

    expect(isIdempotencyKeyValid(metadata)).toBe(false);
  });

  it('should return false for keys expiring exactly now', () => {
    const now = Date.now();
    const metadata = {
      key: 'edge-case',
      createdAt: now - 1000,
      expiresAt: now, // Expires exactly now
    };

    // At the exact expiration moment, key is considered invalid
    expect(isIdempotencyKeyValid(metadata)).toBe(false);
  });

  it('should return true for keys about to expire', () => {
    const metadata = {
      key: 'almost-expired',
      createdAt: Date.now() - 1000,
      expiresAt: Date.now() + 1, // Expires in 1ms
    };

    expect(isIdempotencyKeyValid(metadata)).toBe(true);
  });
});
