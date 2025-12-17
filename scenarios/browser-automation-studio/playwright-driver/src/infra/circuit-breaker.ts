/**
 * Circuit Breaker Pattern Implementation
 *
 * Prevents cascading failures by tracking consecutive failures and
 * temporarily disabling operations when a threshold is reached.
 *
 * State machine:
 * - CLOSED: Normal operation, requests pass through
 * - OPEN: Failures exceeded threshold, requests blocked
 * - HALF_OPEN: Testing if service recovered, single request allowed
 *
 * CLOSED --[failures >= threshold]--> OPEN
 * OPEN --[reset timeout elapsed]--> HALF_OPEN
 * HALF_OPEN --[success]--> CLOSED
 * HALF_OPEN --[failure]--> OPEN
 *
 * @module infra/circuit-breaker
 */

import { logger, scopedLog, LogContext, metrics } from '../utils';

// =============================================================================
// Types
// =============================================================================

/**
 * Circuit breaker configuration.
 */
export interface CircuitBreakerConfig {
  /** Number of consecutive failures before opening circuit */
  maxFailures: number;
  /** Time in ms before attempting half-open state */
  resetTimeoutMs: number;
  /** Optional name for logging */
  name?: string;
}

/**
 * Internal state for a single circuit.
 */
export interface CircuitState {
  consecutiveFailures: number;
  isOpen: boolean;
  lastFailureTime: number | null;
  totalFailures: number;
  totalSuccesses: number;
  /** Prevents multiple concurrent half-open attempts */
  halfOpenInProgress: boolean;
}

/**
 * Statistics for a circuit.
 */
export interface CircuitStats {
  isOpen: boolean;
  consecutiveFailures: number;
  totalFailures: number;
  totalSuccesses: number;
  halfOpenInProgress: boolean;
  lastFailureTime: number | null;
}

/**
 * Circuit breaker interface.
 */
export interface CircuitBreaker<K = string> {
  /** Record a successful operation - resets consecutive failure count */
  recordSuccess(key: K): void;
  /** Record a failed operation - may trip the circuit. Returns true if circuit is now open */
  recordFailure(key: K): boolean;
  /** Check if circuit is open (should skip operations) */
  isOpen(key: K): boolean;
  /** Attempt to enter half-open state. Returns true if this caller should attempt the test */
  tryEnterHalfOpen(key: K): boolean;
  /** Clean up circuit state for a key */
  cleanup(key: K): void;
  /** Get statistics for a circuit */
  getStats(key: K): CircuitStats | null;
  /** Get all active circuit keys */
  getActiveKeys(): K[];
}

// =============================================================================
// Default Configuration
// =============================================================================

export const DEFAULT_CIRCUIT_BREAKER_CONFIG: CircuitBreakerConfig = {
  maxFailures: 5,
  resetTimeoutMs: 30_000, // 30 seconds
  name: 'circuit-breaker',
};

// =============================================================================
// Implementation
// =============================================================================

/**
 * Create a new circuit breaker instance.
 *
 * Each instance manages multiple circuits keyed by a generic key type.
 * This allows a single circuit breaker to track multiple endpoints/services.
 *
 * @example
 * ```typescript
 * // Per-session callback circuit breaker
 * const callbackBreaker = createCircuitBreaker<string>({
 *   maxFailures: 5,
 *   resetTimeoutMs: 30_000,
 *   name: 'callback',
 * });
 *
 * // Usage
 * if (callbackBreaker.isOpen(sessionId) && !callbackBreaker.tryEnterHalfOpen(sessionId)) {
 *   return; // Skip callback, circuit is open
 * }
 *
 * try {
 *   await sendCallback();
 *   callbackBreaker.recordSuccess(sessionId);
 * } catch (err) {
 *   callbackBreaker.recordFailure(sessionId);
 * }
 * ```
 */
export function createCircuitBreaker<K = string>(
  config: Partial<CircuitBreakerConfig> = {}
): CircuitBreaker<K> {
  const cfg: CircuitBreakerConfig = { ...DEFAULT_CIRCUIT_BREAKER_CONFIG, ...config };
  const circuits = new Map<K, CircuitState>();

  /**
   * Get or create circuit state for a key.
   */
  function getCircuit(key: K): CircuitState {
    let state = circuits.get(key);
    if (!state) {
      state = {
        consecutiveFailures: 0,
        isOpen: false,
        lastFailureTime: null,
        totalFailures: 0,
        totalSuccesses: 0,
        halfOpenInProgress: false,
      };
      circuits.set(key, state);
    }
    return state;
  }

  return {
    recordSuccess(key: K): void {
      const state = getCircuit(key);
      state.consecutiveFailures = 0;
      state.isOpen = false;
      state.halfOpenInProgress = false;
      state.totalSuccesses++;
    },

    recordFailure(key: K): boolean {
      const state = getCircuit(key);
      state.consecutiveFailures++;
      state.totalFailures++;
      state.lastFailureTime = Date.now();

      // If we were in half-open state and failed, re-open the circuit
      if (state.halfOpenInProgress) {
        state.halfOpenInProgress = false;
        state.isOpen = true;
        logger.info(scopedLog(LogContext.RECORDING, `${cfg.name}: half-open failed, re-opening`), {
          key: String(key),
          totalFailures: state.totalFailures,
        });
        return true;
      }

      // Check if we should open the circuit
      if (state.consecutiveFailures >= cfg.maxFailures && !state.isOpen) {
        state.isOpen = true;
        logger.warn(scopedLog(LogContext.RECORDING, `${cfg.name}: circuit opened`), {
          key: String(key),
          consecutiveFailures: state.consecutiveFailures,
          totalFailures: state.totalFailures,
          hint: `Operations disabled after ${cfg.maxFailures} consecutive failures. Will retry in ${cfg.resetTimeoutMs / 1000}s.`,
        });
        // Track circuit breaker state change in metrics
        metrics.circuitBreakerStateChanges.inc({ session_id: String(key), state: 'opened' });
      }

      return state.isOpen;
    },

    isOpen(key: K): boolean {
      const state = circuits.get(key);
      return state?.isOpen ?? false;
    },

    tryEnterHalfOpen(key: K): boolean {
      const state = getCircuit(key);

      // Not open, no need for half-open
      if (!state.isOpen) {
        return false;
      }

      // Already attempting half-open
      if (state.halfOpenInProgress) {
        return false;
      }

      // Check if enough time has passed
      if (!state.lastFailureTime) {
        return false;
      }

      const elapsed = Date.now() - state.lastFailureTime;
      if (elapsed < cfg.resetTimeoutMs) {
        return false;
      }

      // Atomically claim the half-open attempt
      state.halfOpenInProgress = true;
      logger.info(scopedLog(LogContext.RECORDING, `${cfg.name}: entering half-open state`), {
        key: String(key),
        lastFailureAge: elapsed,
      });

      return true;
    },

    cleanup(key: K): void {
      circuits.delete(key);
    },

    getStats(key: K): CircuitStats | null {
      const state = circuits.get(key);
      if (!state) return null;

      return {
        isOpen: state.isOpen,
        consecutiveFailures: state.consecutiveFailures,
        totalFailures: state.totalFailures,
        totalSuccesses: state.totalSuccesses,
        halfOpenInProgress: state.halfOpenInProgress,
        lastFailureTime: state.lastFailureTime,
      };
    },

    getActiveKeys(): K[] {
      return Array.from(circuits.keys());
    },
  };
}
