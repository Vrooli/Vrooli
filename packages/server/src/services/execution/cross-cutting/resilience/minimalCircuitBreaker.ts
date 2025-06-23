/**
 * Minimal Circuit Breaker - Event-driven resilience
 * 
 * This circuit breaker provides ONLY basic state management and event emission.
 * All intelligence (failure tracking, threshold adjustment, pattern detection)
 * emerges from resilience agents that subscribe to events.
 * 
 * IMPORTANT: This component does NOT:
 * - Track failure history
 * - Calculate failure rates
 * - Adjust thresholds dynamically
 * - Detect patterns or anomalies
 * - Make complex decisions
 * 
 * It ONLY manages state transitions and emits events.
 */

import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";
import { ExecutionEventEmitter, type ComponentEventEmitter } from "../monitoring/ExecutionEventEmitter.js";
import { ErrorHandler, type ComponentErrorHandler } from "../../shared/ErrorHandler.js";
import { CircuitState, DegradationMode } from "@vrooli/shared";

/**
 * Minimal circuit breaker configuration
 */
export interface MinimalCircuitBreakerConfig {
    service: string;
    operation: string;
    timeoutMs?: number;
    resetTimeoutMs?: number;
}

/**
 * Minimal Circuit Breaker
 * 
 * Provides basic circuit breaking with event emission.
 * Resilience agents subscribe to events to provide intelligent
 * failure tracking, threshold adjustment, and pattern detection.
 */
export class MinimalCircuitBreaker {
    private state: CircuitState = CircuitState.CLOSED;
    private resetTimer?: NodeJS.Timeout;
    private readonly eventEmitter: ComponentEventEmitter;
    private readonly errorHandler: ComponentErrorHandler;
    
    constructor(
        private readonly config: MinimalCircuitBreakerConfig,
        private readonly logger: Logger,
        eventBus: EventBus,
    ) {
        const executionEmitter = new ExecutionEventEmitter(logger, eventBus);
        this.eventEmitter = executionEmitter.createComponentEmitter(
            "cross-cutting", 
            `circuit-breaker:${config.service}:${config.operation}`,
        );
        
        const errorHandler = new ErrorHandler(logger, executionEmitter.eventPublisher);
        this.errorHandler = errorHandler.createComponentHandler(
            `CircuitBreaker:${config.service}:${config.operation}`,
        );
    }
    
    /**
     * Execute operation with circuit breaker protection
     * Emits events for agents to analyze
     */
    async execute<T>(
        operation: () => Promise<T>,
        metadata?: Record<string, unknown>,
    ): Promise<T> {
        const executionId = `${this.config.service}:${this.config.operation}:${Date.now()}`;
        
        // Check if circuit allows execution
        if (this.state === CircuitState.OPEN) {
            await this.emitExecutionEvent(executionId, "rejected", { 
                state: this.state,
                ...metadata, 
            });
            
            throw new Error(`Circuit breaker OPEN for ${this.config.service}:${this.config.operation}`);
        }
        
        // Execute with timeout and emit results
        const startTime = Date.now();
        
        try {
            const timeoutMs = this.config.timeoutMs || 10000;
            const result = await this.executeWithTimeout(operation, timeoutMs);
            
            const duration = Date.now() - startTime;
            
            // Emit success event
            await this.emitExecutionEvent(executionId, "success", {
                state: this.state,
                duration,
                ...metadata,
            });
            
            return result;
            
        } catch (error) {
            const duration = Date.now() - startTime;
            
            // Emit failure event
            await this.emitExecutionEvent(executionId, "failure", {
                state: this.state,
                duration,
                error: error instanceof Error ? error.message : String(error),
                errorType: error instanceof Error ? error.constructor.name : "UnknownError",
                ...metadata,
            });
            
            throw error;
        }
    }
    
    /**
     * Open the circuit
     * Called by resilience agents when they detect issues
     */
    async open(reason: string, metadata?: Record<string, unknown>): Promise<void> {
        if (this.state === CircuitState.OPEN) return;
        
        const oldState = this.state;
        this.state = CircuitState.OPEN;
        
        await this.emitStateChange(oldState, CircuitState.OPEN, { reason, ...metadata });
        
        // Set timer to half-open
        if (this.resetTimer) {
            clearTimeout(this.resetTimer);
        }
        
        const resetTimeoutMs = this.config.resetTimeoutMs || 30000;
        this.resetTimer = setTimeout(() => {
            this.halfOpen("timeout");
        }, resetTimeoutMs);
    }
    
    /**
     * Half-open the circuit for testing
     * Called automatically after timeout or by agents
     */
    async halfOpen(reason: string): Promise<void> {
        if (this.state !== CircuitState.OPEN) return;
        
        const oldState = this.state;
        this.state = CircuitState.HALF_OPEN;
        
        await this.emitStateChange(oldState, CircuitState.HALF_OPEN, { reason });
    }
    
    /**
     * Close the circuit
     * Called by resilience agents when they detect recovery
     */
    async close(reason: string, metadata?: Record<string, unknown>): Promise<void> {
        if (this.state === CircuitState.CLOSED) return;
        
        const oldState = this.state;
        this.state = CircuitState.CLOSED;
        
        if (this.resetTimer) {
            clearTimeout(this.resetTimer);
            this.resetTimer = undefined;
        }
        
        await this.emitStateChange(oldState, CircuitState.CLOSED, { reason, ...metadata });
    }
    
    /**
     * Get current state
     */
    getState(): CircuitState {
        return this.state;
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
     * Helper to execute with timeout
     */
    private async executeWithTimeout<T>(
        operation: () => Promise<T>,
        timeoutMs: number,
    ): Promise<T> {
        return Promise.race([
            operation(),
            new Promise<T>((_, reject) => 
                setTimeout(() => reject(new Error("Operation timeout")), timeoutMs),
            ),
        ]);
    }
    
    /**
     * Emit execution event
     */
    private async emitExecutionEvent(
        executionId: string,
        event: "success" | "failure" | "rejected",
        metadata: Record<string, unknown>,
    ): Promise<void> {
        await this.eventEmitter.emitMetric(
            "safety",
            `circuit_breaker.${event}`,
            1,
            "count",
            {
                service: this.config.service,
                operation: this.config.operation,
                executionId,
                ...metadata,
            },
        );
    }
    
    /**
     * Emit state change event
     */
    private async emitStateChange(
        oldState: CircuitState,
        newState: CircuitState,
        metadata: Record<string, unknown>,
    ): Promise<void> {
        await this.eventEmitter.emitMetric(
            "safety",
            "circuit_breaker.state_change",
            1,
            "event",
            {
                service: this.config.service,
                operation: this.config.operation,
                oldState,
                newState,
                ...metadata,
            },
        );
    }
}

/**
 * Example resilience agent that would manage circuit breakers:
 * 
 * ```typescript
 * // This would be a routine deployed by teams, NOT hardcoded
 * class CircuitBreakerManagementAgent {
 *     private failureWindows = new Map<string, number[]>();
 *     
 *     async onFailureEvent(event: MetricEvent) {
 *         const key = `${event.tags.service}:${event.tags.operation}`;
 *         
 *         // Track failures in time window
 *         this.trackFailure(key, event.timestamp);
 *         
 *         // Calculate failure rate
 *         const failureRate = this.calculateFailureRate(key);
 *         
 *         // Decide whether to open circuit
 *         if (failureRate > 0.5) { // Agent decides threshold
 *             await this.openCircuit(event.tags.service, event.tags.operation, 
 *                 `Failure rate ${failureRate} exceeds threshold`);
 *         }
 *         
 *         // Agent can evolve its decision logic over time
 *     }
 *     
 *     async onSuccessEvent(event: MetricEvent) {
 *         // Agent decides when to close circuit after recovery
 *     }
 * }
 * ```
 */
