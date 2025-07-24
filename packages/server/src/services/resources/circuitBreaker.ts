/**
 * Circuit Breaker implementation for resource resilience
 * Prevents cascading failures by temporarily blocking calls to failing resources
 */

import { logger } from "../../events/logger.js";

export enum CircuitState {
    Closed = "closed",     // Normal operation
    Open = "open",         // Circuit is open, calls are blocked
    HalfOpen = "half-open" // Testing if service has recovered
}

export interface CircuitBreakerConfig {
    /** Number of failures before opening the circuit */
    failureThreshold: number;
    /** Time in ms to wait before attempting recovery */
    recoveryTimeout: number;
    /** Timeout in ms for calls when in half-open state */
    halfOpenTimeout: number;
    /** Name for logging purposes */
    name: string;
}

export interface CircuitBreakerStats {
    state: CircuitState;
    failureCount: number;
    successCount: number;
    lastFailureTime?: Date;
    lastSuccessTime?: Date;
    totalCalls: number;
    blockedCalls: number;
}

/**
 * Circuit breaker that protects against cascading failures
 */
export class CircuitBreaker {
    private state: CircuitState = CircuitState.Closed;
    private failureCount = 0;
    private successCount = 0;
    private lastFailureTime?: Date;
    private lastSuccessTime?: Date;
    private totalCalls = 0;
    private blockedCalls = 0;
    private nextAttemptTime?: Date;

    constructor(private config: CircuitBreakerConfig) {}

    /**
     * Execute a function with circuit breaker protection
     */
    async execute<T>(fn: () => Promise<T>): Promise<T> {
        this.totalCalls++;

        if (this.state === CircuitState.Open) {
            if (this.shouldAttemptReset()) {
                this.state = CircuitState.HalfOpen;
                logger.debug(`[CircuitBreaker:${this.config.name}] Attempting recovery (half-open)`);
            } else {
                this.blockedCalls++;
                throw new Error(`Circuit breaker is OPEN for ${this.config.name}. Next attempt at ${this.nextAttemptTime?.toISOString()}`);
            }
        }

        try {
            const result = await this.callWithTimeout(fn);
            this.onSuccess();
            return result;
        } catch (error) {
            this.onFailure();
            throw error;
        }
    }

    /**
     * Call function with timeout in half-open state
     */
    private async callWithTimeout<T>(fn: () => Promise<T>): Promise<T> {
        if (this.state === CircuitState.HalfOpen) {
            return Promise.race([
                fn(),
                new Promise<never>((_, reject) => {
                    setTimeout(() => reject(new Error("Half-open timeout")), this.config.halfOpenTimeout);
                }),
            ]);
        }
        return fn();
    }

    /**
     * Handle successful call
     */
    private onSuccess(): void {
        this.successCount++;
        this.lastSuccessTime = new Date();
        
        if (this.state === CircuitState.HalfOpen) {
            logger.info(`[CircuitBreaker:${this.config.name}] Recovery successful, closing circuit`);
            this.reset();
        } else if (this.state === CircuitState.Closed) {
            // Reset failure count on successful call
            this.failureCount = 0;
        }
    }

    /**
     * Handle failed call
     */
    private onFailure(): void {
        this.failureCount++;
        this.lastFailureTime = new Date();

        if (this.state === CircuitState.HalfOpen) {
            logger.warn(`[CircuitBreaker:${this.config.name}] Recovery failed, opening circuit again`);
            this.openCircuit();
        } else if (this.state === CircuitState.Closed && this.failureCount >= this.config.failureThreshold) {
            logger.warn(`[CircuitBreaker:${this.config.name}] Failure threshold reached (${this.failureCount}), opening circuit`);
            this.openCircuit();
        }
    }

    /**
     * Open the circuit (block calls)
     */
    private openCircuit(): void {
        this.state = CircuitState.Open;
        this.nextAttemptTime = new Date(Date.now() + this.config.recoveryTimeout);
        
        logger.warn(`[CircuitBreaker:${this.config.name}] Circuit opened, blocking calls until ${this.nextAttemptTime.toISOString()}`);
    }

    /**
     * Check if we should attempt to reset the circuit
     */
    private shouldAttemptReset(): boolean {
        return this.nextAttemptTime ? new Date() >= this.nextAttemptTime : false;
    }

    /**
     * Reset the circuit to closed state
     */
    private reset(): void {
        this.state = CircuitState.Closed;
        this.failureCount = 0;
        this.nextAttemptTime = undefined;
    }

    /**
     * Force reset the circuit (for testing or manual recovery)
     */
    forceReset(): void {
        logger.info(`[CircuitBreaker:${this.config.name}] Manually resetting circuit`);
        this.reset();
    }

    /**
     * Get current circuit breaker statistics
     */
    getStats(): CircuitBreakerStats {
        return {
            state: this.state,
            failureCount: this.failureCount,
            successCount: this.successCount,
            lastFailureTime: this.lastFailureTime,
            lastSuccessTime: this.lastSuccessTime,
            totalCalls: this.totalCalls,
            blockedCalls: this.blockedCalls,
        };
    }

    /**
     * Check if circuit is currently allowing calls
     */
    isCallAllowed(): boolean {
        if (this.state === CircuitState.Closed) {
            return true;
        }
        if (this.state === CircuitState.Open) {
            return this.shouldAttemptReset();
        }
        // Half-open state allows calls
        return true;
    }
}

/**
 * Factory for creating circuit breakers with sensible defaults
 */
export class CircuitBreakerFactory {
    /**
     * Create circuit breaker for resource discovery
     */
    static forResourceDiscovery(resourceId: string): CircuitBreaker {
        return new CircuitBreaker({
            name: `discovery-${resourceId}`,
            failureThreshold: 3,
            recoveryTimeout: 30000, // 30 seconds
            halfOpenTimeout: 5000,   // 5 seconds
        });
    }

    /**
     * Create circuit breaker for health checks
     */
    static forHealthCheck(resourceId: string): CircuitBreaker {
        return new CircuitBreaker({
            name: `health-${resourceId}`,
            failureThreshold: 5,
            recoveryTimeout: 60000, // 1 minute
            halfOpenTimeout: 10000, // 10 seconds
        });
    }

    /**
     * Create circuit breaker for resource operations
     */
    static forResourceOperation(resourceId: string): CircuitBreaker {
        return new CircuitBreaker({
            name: `operation-${resourceId}`,
            failureThreshold: 10,
            recoveryTimeout: 120000, // 2 minutes
            halfOpenTimeout: 15000,  // 15 seconds
        });
    }
}
