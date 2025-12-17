/**
 * Circuit Breaker Tests
 *
 * Tests the circuit breaker state machine:
 * - CLOSED → OPEN after maxFailures consecutive failures
 * - OPEN → HALF_OPEN after resetTimeoutMs
 * - HALF_OPEN → CLOSED on success
 * - HALF_OPEN → OPEN on failure
 */

// Mock the utils module to avoid proto dependency chain
jest.mock('../../../src/utils', () => ({
  logger: {
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn(),
    debug: jest.fn(),
  },
  scopedLog: jest.fn((context: string, message: string) => `[${context}] ${message}`),
  LogContext: {
    RECORDING: 'recording',
    SESSION: 'session',
    SERVER: 'server',
  },
  metrics: {
    circuitBreakerStateChanges: {
      inc: jest.fn(),
    },
  },
}));

import { createCircuitBreaker, type CircuitBreaker } from '../../../src/infra/circuit-breaker';

describe('CircuitBreaker', () => {
  let breaker: CircuitBreaker<string>;

  beforeEach(() => {
    jest.useFakeTimers();
    breaker = createCircuitBreaker({
      maxFailures: 3,
      resetTimeoutMs: 1000,
      name: 'test',
    });
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('initial state', () => {
    it('starts closed', () => {
      expect(breaker.isOpen('key')).toBe(false);
    });

    it('returns null stats for unknown key', () => {
      expect(breaker.getStats('unknown')).toBe(null);
    });

    it('has no active keys initially', () => {
      expect(breaker.getActiveKeys()).toEqual([]);
    });
  });

  describe('CLOSED state', () => {
    it('stays closed on success', () => {
      breaker.recordSuccess('key');
      expect(breaker.isOpen('key')).toBe(false);
    });

    it('stays closed after failures below threshold', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      expect(breaker.isOpen('key')).toBe(false);
    });

    it('tracks consecutive failures', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      const stats = breaker.getStats('key');
      expect(stats?.consecutiveFailures).toBe(2);
      expect(stats?.totalFailures).toBe(2);
    });

    it('resets consecutive failures on success', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordSuccess('key');
      const stats = breaker.getStats('key');
      expect(stats?.consecutiveFailures).toBe(0);
      expect(stats?.totalFailures).toBe(2); // Total still tracked
    });
  });

  describe('CLOSED → OPEN transition', () => {
    it('opens after maxFailures consecutive failures', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      expect(breaker.isOpen('key')).toBe(false);

      breaker.recordFailure('key'); // 3rd failure = maxFailures
      expect(breaker.isOpen('key')).toBe(true);
    });

    it('returns true from recordFailure when circuit opens', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      const opened = breaker.recordFailure('key');
      expect(opened).toBe(true);
    });

    it('does not open if failures are not consecutive', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordSuccess('key'); // Resets consecutive count
      breaker.recordFailure('key');
      expect(breaker.isOpen('key')).toBe(false);
    });
  });

  describe('OPEN state', () => {
    beforeEach(() => {
      // Open the circuit
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordFailure('key');
    });

    it('is open', () => {
      expect(breaker.isOpen('key')).toBe(true);
    });

    it('tryEnterHalfOpen returns false before timeout', () => {
      jest.advanceTimersByTime(500); // Half of resetTimeoutMs
      expect(breaker.tryEnterHalfOpen('key')).toBe(false);
    });

    it('continues to track failures while open', () => {
      breaker.recordFailure('key');
      const stats = breaker.getStats('key');
      expect(stats?.totalFailures).toBe(4);
    });
  });

  describe('OPEN → HALF_OPEN transition', () => {
    beforeEach(() => {
      // Open the circuit
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordFailure('key');
    });

    it('allows half-open after timeout', () => {
      jest.advanceTimersByTime(1000); // resetTimeoutMs
      expect(breaker.tryEnterHalfOpen('key')).toBe(true);
    });

    it('only allows one caller to enter half-open', () => {
      jest.advanceTimersByTime(1000);
      expect(breaker.tryEnterHalfOpen('key')).toBe(true);
      expect(breaker.tryEnterHalfOpen('key')).toBe(false); // Second caller blocked
    });

    it('sets halfOpenInProgress flag', () => {
      jest.advanceTimersByTime(1000);
      breaker.tryEnterHalfOpen('key');
      const stats = breaker.getStats('key');
      expect(stats?.halfOpenInProgress).toBe(true);
    });
  });

  describe('HALF_OPEN → CLOSED transition', () => {
    beforeEach(() => {
      // Open the circuit and enter half-open
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      jest.advanceTimersByTime(1000);
      breaker.tryEnterHalfOpen('key');
    });

    it('closes on success', () => {
      breaker.recordSuccess('key');
      expect(breaker.isOpen('key')).toBe(false);
    });

    it('clears halfOpenInProgress on success', () => {
      breaker.recordSuccess('key');
      const stats = breaker.getStats('key');
      expect(stats?.halfOpenInProgress).toBe(false);
    });

    it('resets consecutive failures on success', () => {
      breaker.recordSuccess('key');
      const stats = breaker.getStats('key');
      expect(stats?.consecutiveFailures).toBe(0);
    });
  });

  describe('HALF_OPEN → OPEN transition', () => {
    beforeEach(() => {
      // Open the circuit and enter half-open
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      jest.advanceTimersByTime(1000);
      breaker.tryEnterHalfOpen('key');
    });

    it('re-opens on failure', () => {
      breaker.recordFailure('key');
      expect(breaker.isOpen('key')).toBe(true);
    });

    it('clears halfOpenInProgress on failure', () => {
      breaker.recordFailure('key');
      const stats = breaker.getStats('key');
      expect(stats?.halfOpenInProgress).toBe(false);
    });

    it('returns true from recordFailure when re-opening', () => {
      const reopened = breaker.recordFailure('key');
      expect(reopened).toBe(true);
    });
  });

  describe('multiple keys', () => {
    it('tracks keys independently', () => {
      breaker.recordFailure('key1');
      breaker.recordFailure('key1');
      breaker.recordFailure('key1');

      breaker.recordFailure('key2');

      expect(breaker.isOpen('key1')).toBe(true);
      expect(breaker.isOpen('key2')).toBe(false);
    });

    it('lists active keys', () => {
      breaker.recordFailure('key1');
      breaker.recordSuccess('key2');

      const keys = breaker.getActiveKeys();
      expect(keys).toContain('key1');
      expect(keys).toContain('key2');
    });
  });

  describe('cleanup', () => {
    it('removes key state', () => {
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      breaker.recordFailure('key');
      expect(breaker.isOpen('key')).toBe(true);

      breaker.cleanup('key');

      expect(breaker.isOpen('key')).toBe(false);
      expect(breaker.getStats('key')).toBe(null);
    });

    it('removes from active keys', () => {
      breaker.recordFailure('key');
      breaker.cleanup('key');
      expect(breaker.getActiveKeys()).not.toContain('key');
    });
  });

  describe('stats', () => {
    it('tracks total successes and failures', () => {
      breaker.recordSuccess('key');
      breaker.recordSuccess('key');
      breaker.recordFailure('key');

      const stats = breaker.getStats('key');
      expect(stats?.totalSuccesses).toBe(2);
      expect(stats?.totalFailures).toBe(1);
    });

    it('tracks lastFailureTime', () => {
      const now = Date.now();
      jest.setSystemTime(now);

      breaker.recordFailure('key');

      const stats = breaker.getStats('key');
      expect(stats?.lastFailureTime).toBe(now);
    });
  });
});
