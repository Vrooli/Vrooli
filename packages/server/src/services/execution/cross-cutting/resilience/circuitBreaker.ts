/**
 * MINIMAL CIRCUIT BREAKER
 * 
 * Provides ONLY the basic circuit breaking state management infrastructure.
 * All intelligence, optimization, and learning comes from emergent resilience agents.
 * 
 * WHAT THIS DOES:
 * - Track basic circuit states (CLOSED → OPEN → HALF_OPEN)
 * - Emit resilience events for agents to analyze
 * - Simple failure counting and timeout tracking
 * 
 * WHAT THIS DOES NOT DO (EMERGENT CAPABILITIES):
 * - Adaptive threshold adjustment (resilience agents optimize this)
 * - Complex fallback strategies (agents develop these)
 * - Performance-based state transitions (monitoring agents handle this)
 * - Error classification and recovery strategies (agents learn these)
 * - Health check algorithms (agents create sophisticated monitoring)
 */

import { type EventBus } from "../events/eventBus.js";
import {
    CircuitState,
    type CircuitBreakerConfig,
    type CircuitBreakerState,
    ResilienceEventType,
} from "@vrooli/shared";

/**
 * Simple circuit breaker configuration
 */
const DEFAULT_CONFIG: CircuitBreakerConfig = {
    failureThreshold: 5,
    timeoutMs: 10000,
    resetTimeoutMs: 30000,
    successThreshold: 2,
};

/**
 * Basic circuit breaker metrics
 */
interface CircuitBreakerMetrics {
    requestCount: number;
    successCount: number;
    failureCount: number;
    lastFailureTime?: Date;
    lastSuccessTime?: Date;
}

/**
 * Circuit breaker execution context
 */
interface ExecutionContext {
    operationId: string;
    timeout: number;
    startTime: Date;
    metadata?: Record<string, unknown>;
}

/**
 * Circuit breaker error
 */
export class CircuitBreakerOpenError extends Error {
    constructor(
        message: string,
        public readonly state: CircuitState,
        public readonly stateInfo: CircuitBreakerState,
    ) {
        super(message);
        this.name = "CircuitBreakerOpenError";
    }
}

/**
 * MINIMAL CIRCUIT BREAKER
 * 
 * Simple state management with event emission for agents to enhance.
 */
export class CircuitBreaker {
    private readonly service: string;
    private readonly operation: string;
    private readonly config: CircuitBreakerConfig;
    private readonly eventBus: EventBus;
    
    // Basic state tracking
    private state: CircuitState = CircuitState.CLOSED;
    private metrics: CircuitBreakerMetrics;
    private stateChangeTime: Date = new Date();
    private halfOpenSuccessCount = 0;
    
    // Simple timer for reset attempts
    private resetTimer?: NodeJS.Timeout;

    constructor(
        service: string,
        operation: string,
        config: Partial<CircuitBreakerConfig>,
        eventBus: EventBus,
    ) {
        this.service = service;
        this.operation = operation;
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.eventBus = eventBus;
        
        this.metrics = {
            requestCount: 0,
            successCount: 0,
            failureCount: 0,
        };
    }

    /**
     * Execute operation with basic circuit breaker protection
     */
    async execute<T>(operation: () => Promise<T>): Promise<T> {
        const executionContext: ExecutionContext = {
            operationId: `${this.service}:${this.operation}:${Date.now()}`,
            timeout: this.config.timeoutMs,
            startTime: new Date(),
        };

        // Basic state check
        if (this.state === CircuitState.OPEN) {
            await this.emitRejectionEvent(executionContext);
            throw new CircuitBreakerOpenError(
                `Circuit breaker OPEN for ${this.service}:${this.operation}`,
                this.state,
                this.getState(),
            );
        }

        return this.executeWithBasicMonitoring(operation, executionContext);
    }

    /**
     * Basic execution with simple success/failure tracking
     */
    private async executeWithBasicMonitoring<T>(
        operation: () => Promise<T>,
        context: ExecutionContext,
    ): Promise<T> {
        this.metrics.requestCount++;

        try {
            // Basic timeout protection
            const timeoutPromise = new Promise<never>((_, reject) => {
                setTimeout(() => reject(new Error("Operation timeout")), context.timeout);
            });

            const result = await Promise.race([operation(), timeoutPromise]);
            
            await this.handleSuccess(context);
            return result;

        } catch (error) {
            await this.handleFailure(error, context);
            throw error;
        }
    }

    /**
     * Handle successful execution
     */
    private async handleSuccess(context: ExecutionContext): Promise<void> {
        this.metrics.successCount++;
        this.metrics.lastSuccessTime = new Date();

        if (this.state === CircuitState.HALF_OPEN) {
            this.halfOpenSuccessCount++;
            
            // Simple success threshold check
            if (this.halfOpenSuccessCount >= this.config.successThreshold) {
                await this.changeState(CircuitState.CLOSED, "Success threshold met");
            }
        }

        await this.emitSuccessEvent(context);
    }

    /**
     * Handle failed execution
     */
    private async handleFailure(error: unknown, context: ExecutionContext): Promise<void> {
        this.metrics.failureCount++;
        this.metrics.lastFailureTime = new Date();

        // Simple failure threshold check
        if (this.state === CircuitState.CLOSED && 
            this.metrics.failureCount >= this.config.failureThreshold) {
            await this.changeState(CircuitState.OPEN, "Failure threshold exceeded");
        } else if (this.state === CircuitState.HALF_OPEN) {
            await this.changeState(CircuitState.OPEN, "Failure in half-open state");
        }

        await this.emitFailureEvent(error, context);
    }

    /**
     * Change circuit state
     */
    private async changeState(newState: CircuitState, reason: string): Promise<void> {
        const oldState = this.state;
        this.state = newState;
        this.stateChangeTime = new Date();

        // Reset counters for half-open state
        if (newState === CircuitState.HALF_OPEN) {
            this.halfOpenSuccessCount = 0;
        }

        // Set simple reset timer for open state
        if (newState === CircuitState.OPEN) {
            this.scheduleReset();
        }

        await this.emitStateChangeEvent(oldState, newState, reason);
    }

    /**
     * Schedule simple reset attempt
     */
    private scheduleReset(): void {
        if (this.resetTimer) {
            clearTimeout(this.resetTimer);
        }

        this.resetTimer = setTimeout(async () => {
            if (this.state === CircuitState.OPEN) {
                await this.changeState(CircuitState.HALF_OPEN, "Reset timeout reached");
            }
        }, this.config.resetTimeoutMs);
    }

    /**
     * Get current state
     */
    getState(): CircuitBreakerState {
        return {
            state: this.state,
            failureCount: this.metrics.failureCount,
            successCount: this.metrics.successCount,
            lastFailureTime: this.metrics.lastFailureTime,
            lastSuccessTime: this.metrics.lastSuccessTime,
            stateChangeTime: this.stateChangeTime,
            metadata: {
                service: this.service,
                operation: this.operation,
            },
        };
    }

    /**
     * Cleanup resources
     */
    destroy(): void {
        if (this.resetTimer) {
            clearTimeout(this.resetTimer);
        }
    }

    /**
     * Emit success event for agents to analyze
     */
    private async emitSuccessEvent(context: ExecutionContext): Promise<void> {
        await this.eventBus.emit({
            type: ResilienceEventType.CIRCUIT_BREAKER_SUCCESS,
            data: {
                service: this.service,
                operation: this.operation,
                state: this.state,
                metrics: this.metrics,
                context,
            },
            timestamp: new Date(),
        });
    }

    /**
     * Emit failure event for agents to analyze
     */
    private async emitFailureEvent(error: unknown, context: ExecutionContext): Promise<void> {
        await this.eventBus.emit({
            type: ResilienceEventType.CIRCUIT_BREAKER_FAILURE,
            data: {
                service: this.service,
                operation: this.operation,
                state: this.state,
                metrics: this.metrics,
                error: error instanceof Error ? error.message : String(error),
                context,
            },
            timestamp: new Date(),
        });
    }

    /**
     * Emit state change event for agents to analyze
     */
    private async emitStateChangeEvent(
        oldState: CircuitState,
        newState: CircuitState,
        reason: string,
    ): Promise<void> {
        await this.eventBus.emit({
            type: ResilienceEventType.CIRCUIT_BREAKER_STATE_CHANGE,
            data: {
                service: this.service,
                operation: this.operation,
                oldState,
                newState,
                reason,
                metrics: this.metrics,
                timestamp: this.stateChangeTime,
            },
            timestamp: new Date(),
        });
    }

    /**
     * Emit rejection event for agents to analyze
     */
    private async emitRejectionEvent(context: ExecutionContext): Promise<void> {
        await this.eventBus.emit({
            type: ResilienceEventType.CIRCUIT_BREAKER_REJECTION,
            data: {
                service: this.service,
                operation: this.operation,
                state: this.state,
                metrics: this.metrics,
                context,
            },
            timestamp: new Date(),
        });
    }
}
