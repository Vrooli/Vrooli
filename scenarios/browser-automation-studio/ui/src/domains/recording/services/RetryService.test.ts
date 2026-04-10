import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  createInitialRetryState,
  calculateRetryDelay,
  getNextRetryState,
  canRetry,
  getRemainingCooldown,
  createSuccessState,
  createManualRetryState,
  DEFAULT_RETRY_CONFIG,
  type RetryConfig,
  type RetryState,
} from './RetryService';

describe('RetryService', () => {
  describe('createInitialRetryState', () => {
    it('returns state with zero attempts', () => {
      const state = createInitialRetryState();
      expect(state.attempts).toBe(0);
    });

    it('returns state not in cooldown', () => {
      const state = createInitialRetryState();
      expect(state.inCooldown).toBe(false);
      expect(state.nextRetryAt).toBeNull();
    });

    it('returns state with maxRetriesExceeded false', () => {
      const state = createInitialRetryState();
      expect(state.maxRetriesExceeded).toBe(false);
    });
  });

  describe('calculateRetryDelay', () => {
    const config: RetryConfig = {
      maxRetries: 5,
      baseDelayMs: 1000,
      maxDelayMs: 30000,
    };

    it('returns 0 for attempt <= 0', () => {
      expect(calculateRetryDelay(0, config)).toBe(0);
      expect(calculateRetryDelay(-1, config)).toBe(0);
      expect(calculateRetryDelay(-100, config)).toBe(0);
    });

    it('returns baseDelayMs for attempt 1', () => {
      expect(calculateRetryDelay(1, config)).toBe(1000);
    });

    it('returns baseDelayMs * 2 for attempt 2', () => {
      expect(calculateRetryDelay(2, config)).toBe(2000);
    });

    it('returns baseDelayMs * 4 for attempt 3', () => {
      expect(calculateRetryDelay(3, config)).toBe(4000);
    });

    it('follows exponential backoff pattern', () => {
      // Delay formula: baseDelay * 2^(attempt-1)
      expect(calculateRetryDelay(1, config)).toBe(1000); // 1000 * 2^0 = 1000
      expect(calculateRetryDelay(2, config)).toBe(2000); // 1000 * 2^1 = 2000
      expect(calculateRetryDelay(3, config)).toBe(4000); // 1000 * 2^2 = 4000
      expect(calculateRetryDelay(4, config)).toBe(8000); // 1000 * 2^3 = 8000
      expect(calculateRetryDelay(5, config)).toBe(16000); // 1000 * 2^4 = 16000
    });

    it('caps delay at maxDelayMs', () => {
      // attempt 6 would be 1000 * 2^5 = 32000, but capped at 30000
      expect(calculateRetryDelay(6, config)).toBe(30000);
      // Higher attempts should also be capped
      expect(calculateRetryDelay(10, config)).toBe(30000);
      expect(calculateRetryDelay(100, config)).toBe(30000);
    });

    it('works with different base delay values', () => {
      const fastConfig: RetryConfig = {
        maxRetries: 3,
        baseDelayMs: 500,
        maxDelayMs: 5000,
      };
      expect(calculateRetryDelay(1, fastConfig)).toBe(500);
      expect(calculateRetryDelay(2, fastConfig)).toBe(1000);
      expect(calculateRetryDelay(3, fastConfig)).toBe(2000);
      expect(calculateRetryDelay(4, fastConfig)).toBe(4000);
      expect(calculateRetryDelay(5, fastConfig)).toBe(5000); // capped
    });
  });

  describe('getNextRetryState', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-01-20T10:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('increments attempts', () => {
      const state = getNextRetryState(0, DEFAULT_RETRY_CONFIG);
      expect(state.attempts).toBe(1);

      const state2 = getNextRetryState(2, DEFAULT_RETRY_CONFIG);
      expect(state2.attempts).toBe(3);
    });

    it('sets inCooldown true when under max', () => {
      const state = getNextRetryState(0, DEFAULT_RETRY_CONFIG);
      expect(state.inCooldown).toBe(true);
    });

    it('calculates nextRetryAt correctly', () => {
      const now = Date.now();
      const state = getNextRetryState(0, DEFAULT_RETRY_CONFIG);
      // First retry (attempt 1): baseDelay * 2^0 = 1000ms
      expect(state.nextRetryAt).toBe(now + 1000);

      const state2 = getNextRetryState(1, DEFAULT_RETRY_CONFIG);
      // Second retry (attempt 2): baseDelay * 2^1 = 2000ms
      expect(state2.nextRetryAt).toBe(now + 2000);
    });

    it('sets maxRetriesExceeded at max retries', () => {
      // DEFAULT_RETRY_CONFIG.maxRetries is 3, so at attempt 3 we hit the max
      const state = getNextRetryState(2, DEFAULT_RETRY_CONFIG);
      expect(state.attempts).toBe(3);
      expect(state.maxRetriesExceeded).toBe(true);
    });

    it('does not set cooldown when max exceeded', () => {
      const state = getNextRetryState(2, DEFAULT_RETRY_CONFIG);
      expect(state.maxRetriesExceeded).toBe(true);
      expect(state.inCooldown).toBe(false);
      expect(state.nextRetryAt).toBeNull();
    });

    it('returns correct state for each attempt sequence', () => {
      // Simulate a sequence of failures
      const state1 = getNextRetryState(0, DEFAULT_RETRY_CONFIG);
      expect(state1).toEqual({
        attempts: 1,
        inCooldown: true,
        nextRetryAt: Date.now() + 1000,
        maxRetriesExceeded: false,
      });

      const state2 = getNextRetryState(1, DEFAULT_RETRY_CONFIG);
      expect(state2).toEqual({
        attempts: 2,
        inCooldown: true,
        nextRetryAt: Date.now() + 2000,
        maxRetriesExceeded: false,
      });

      const state3 = getNextRetryState(2, DEFAULT_RETRY_CONFIG);
      expect(state3).toEqual({
        attempts: 3,
        inCooldown: false,
        nextRetryAt: null,
        maxRetriesExceeded: true,
      });
    });

    it('works with custom config', () => {
      const customConfig: RetryConfig = {
        maxRetries: 2,
        baseDelayMs: 500,
        maxDelayMs: 5000,
      };

      const state1 = getNextRetryState(0, customConfig);
      expect(state1.attempts).toBe(1);
      expect(state1.inCooldown).toBe(true);
      expect(state1.nextRetryAt).toBe(Date.now() + 500);
      expect(state1.maxRetriesExceeded).toBe(false);

      const state2 = getNextRetryState(1, customConfig);
      expect(state2.attempts).toBe(2);
      expect(state2.maxRetriesExceeded).toBe(true);
      expect(state2.inCooldown).toBe(false);
    });
  });

  describe('canRetry', () => {
    it('returns true when not in cooldown and max not exceeded', () => {
      const state: RetryState = {
        attempts: 1,
        inCooldown: false,
        nextRetryAt: null,
        maxRetriesExceeded: false,
      };
      expect(canRetry(state)).toBe(true);
    });

    it('returns false when in cooldown', () => {
      const state: RetryState = {
        attempts: 1,
        inCooldown: true,
        nextRetryAt: Date.now() + 1000,
        maxRetriesExceeded: false,
      };
      expect(canRetry(state)).toBe(false);
    });

    it('returns false when max exceeded', () => {
      const state: RetryState = {
        attempts: 3,
        inCooldown: false,
        nextRetryAt: null,
        maxRetriesExceeded: true,
      };
      expect(canRetry(state)).toBe(false);
    });

    it('returns false when both in cooldown and max exceeded', () => {
      // Edge case: shouldn't happen normally, but test the logic
      const state: RetryState = {
        attempts: 3,
        inCooldown: true,
        nextRetryAt: Date.now() + 1000,
        maxRetriesExceeded: true,
      };
      expect(canRetry(state)).toBe(false);
    });

    it('returns true for initial state', () => {
      const state = createInitialRetryState();
      expect(canRetry(state)).toBe(true);
    });
  });

  describe('getRemainingCooldown', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-01-20T10:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns 0 when not in cooldown', () => {
      const state: RetryState = {
        attempts: 1,
        inCooldown: false,
        nextRetryAt: null,
        maxRetriesExceeded: false,
      };
      expect(getRemainingCooldown(state)).toBe(0);
    });

    it('returns 0 when nextRetryAt is null', () => {
      const state: RetryState = {
        attempts: 1,
        inCooldown: true, // inconsistent state, but test the null check
        nextRetryAt: null,
        maxRetriesExceeded: false,
      };
      expect(getRemainingCooldown(state)).toBe(0);
    });

    it('returns remaining time when in cooldown', () => {
      const now = Date.now();
      const state: RetryState = {
        attempts: 1,
        inCooldown: true,
        nextRetryAt: now + 5000,
        maxRetriesExceeded: false,
      };
      expect(getRemainingCooldown(state)).toBe(5000);
    });

    it('returns 0 when nextRetryAt in past', () => {
      const now = Date.now();
      const state: RetryState = {
        attempts: 1,
        inCooldown: true,
        nextRetryAt: now - 1000, // 1 second in the past
        maxRetriesExceeded: false,
      };
      expect(getRemainingCooldown(state)).toBe(0);
    });

    it('calculates remaining time correctly as time passes', () => {
      const state: RetryState = {
        attempts: 1,
        inCooldown: true,
        nextRetryAt: Date.now() + 5000,
        maxRetriesExceeded: false,
      };

      expect(getRemainingCooldown(state)).toBe(5000);

      // Advance time by 2 seconds
      vi.advanceTimersByTime(2000);
      expect(getRemainingCooldown(state)).toBe(3000);

      // Advance time past the cooldown
      vi.advanceTimersByTime(4000);
      expect(getRemainingCooldown(state)).toBe(0);
    });
  });

  describe('createSuccessState', () => {
    it('returns initial state', () => {
      const state = createSuccessState();
      expect(state).toEqual(createInitialRetryState());
    });

    it('returns fresh state object', () => {
      const state1 = createSuccessState();
      const state2 = createSuccessState();
      expect(state1).not.toBe(state2); // Different references
      expect(state1).toEqual(state2); // Same values
    });
  });

  describe('createManualRetryState', () => {
    it('returns initial state', () => {
      const state = createManualRetryState();
      expect(state).toEqual(createInitialRetryState());
    });

    it('returns fresh state object', () => {
      const state1 = createManualRetryState();
      const state2 = createManualRetryState();
      expect(state1).not.toBe(state2); // Different references
      expect(state1).toEqual(state2); // Same values
    });
  });

  describe('DEFAULT_RETRY_CONFIG', () => {
    it('has expected default values', () => {
      expect(DEFAULT_RETRY_CONFIG.maxRetries).toBe(3);
      expect(DEFAULT_RETRY_CONFIG.baseDelayMs).toBe(1000);
      expect(DEFAULT_RETRY_CONFIG.maxDelayMs).toBe(30000);
    });
  });

  describe('integration scenarios', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-01-20T10:00:00Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('simulates complete retry cycle with success before max', () => {
      // Initial state - can retry
      let state = createInitialRetryState();
      expect(canRetry(state)).toBe(true);

      // First failure
      state = getNextRetryState(state.attempts, DEFAULT_RETRY_CONFIG);
      expect(state.attempts).toBe(1);
      expect(state.inCooldown).toBe(true);
      expect(canRetry(state)).toBe(false);

      // Wait for cooldown to expire
      vi.advanceTimersByTime(1000);
      // Manually clear cooldown (as the hook would do)
      state = { ...state, inCooldown: false, nextRetryAt: null };
      expect(canRetry(state)).toBe(true);

      // Success! Reset state
      state = createSuccessState();
      expect(state.attempts).toBe(0);
      expect(canRetry(state)).toBe(true);
    });

    it('simulates complete retry cycle reaching max retries', () => {
      let state = createInitialRetryState();
      let attempts = 0;

      // Fail 3 times (DEFAULT_RETRY_CONFIG.maxRetries)
      for (let i = 0; i < 3; i++) {
        state = getNextRetryState(attempts, DEFAULT_RETRY_CONFIG);
        attempts = state.attempts;
      }

      expect(state.attempts).toBe(3);
      expect(state.maxRetriesExceeded).toBe(true);
      expect(canRetry(state)).toBe(false);
    });

    it('simulates manual retry after max exceeded', () => {
      // Get to max retries
      let state = getNextRetryState(2, DEFAULT_RETRY_CONFIG);
      expect(state.maxRetriesExceeded).toBe(true);
      expect(canRetry(state)).toBe(false);

      // User triggers manual retry
      state = createManualRetryState();
      expect(state.attempts).toBe(0);
      expect(state.maxRetriesExceeded).toBe(false);
      expect(canRetry(state)).toBe(true);
    });
  });
});
