/**
 * Minimal Circuit Breaker Manager - Basic registry only
 * 
 * This manager provides ONLY basic circuit breaker registry functionality.
 * All intelligence (pattern detection, configuration optimization, health monitoring)
 * emerges from resilience management agents.
 * 
 * IMPORTANT: This component does NOT:
 * - Optimize configurations
 * - Detect patterns
 * - Monitor health
 * - Make coordination decisions
 * - Learn from usage
 * 
 * It ONLY manages circuit breaker instances and emits events.
 */

import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";
import { MinimalCircuitBreaker, type MinimalCircuitBreakerConfig } from "./minimalCircuitBreaker.js";
import { ExecutionEventEmitter, type ComponentEventEmitter } from "../monitoring/ExecutionEventEmitter.js";
import { ErrorHandler, type ComponentErrorHandler } from "../../shared/ErrorHandler.js";

/**
 * Minimal Circuit Breaker Manager
 * 
 * Provides basic registry for circuit breakers.
 * Management agents subscribe to events to provide intelligent
 * configuration, pattern detection, and coordination.
 */
export class MinimalCircuitBreakerManager {
    private readonly circuitBreakers = new Map<string, MinimalCircuitBreaker>();
    private readonly eventEmitter: ComponentEventEmitter;
    private readonly errorHandler: ComponentErrorHandler;
    
    constructor(
        private readonly logger: Logger,
        private readonly eventBus: EventBus,
    ) {
        const executionEmitter = new ExecutionEventEmitter(logger, eventBus);
        this.eventEmitter = executionEmitter.createComponentEmitter(
            "cross-cutting", 
            "circuit-breaker-manager",
        );
        
        const errorHandler = new ErrorHandler(logger, executionEmitter.eventPublisher);
        this.errorHandler = errorHandler.createComponentHandler("CircuitBreakerManager");
    }
    
    /**
     * Get or create a circuit breaker
     * Emits events for management agents to analyze
     */
    async getCircuitBreaker(
        service: string,
        operation: string,
        config?: Partial<MinimalCircuitBreakerConfig>,
    ): Promise<MinimalCircuitBreaker> {
        const key = `${service}:${operation}`;
        
        // Return existing if found
        const existing = this.circuitBreakers.get(key);
        if (existing) {
            await this.emitAccessEvent(service, operation, "existing");
            return existing;
        }
        
        // Create new circuit breaker
        const cbConfig: MinimalCircuitBreakerConfig = {
            service,
            operation,
            timeoutMs: config?.timeoutMs || 10000,
            resetTimeoutMs: config?.resetTimeoutMs || 30000,
        };
        
        const circuitBreaker = new MinimalCircuitBreaker(cbConfig, this.logger, this.eventBus);
        this.circuitBreakers.set(key, circuitBreaker);
        
        // Emit creation event for management agents
        await this.emitCreationEvent(service, operation, cbConfig);
        
        return circuitBreaker;
    }
    
    /**
     * Remove a circuit breaker
     * Called by management agents during cleanup
     */
    async removeCircuitBreaker(service: string, operation: string): Promise<boolean> {
        const key = `${service}:${operation}`;
        const circuitBreaker = this.circuitBreakers.get(key);
        
        if (!circuitBreaker) {
            return false;
        }
        
        circuitBreaker.destroy();
        this.circuitBreakers.delete(key);
        
        await this.emitRemovalEvent(service, operation);
        return true;
    }
    
    /**
     * List all circuit breakers
     * Management agents use this to analyze the system
     */
    listCircuitBreakers(): Array<{ service: string; operation: string; state: string }> {
        const list: Array<{ service: string; operation: string; state: string }> = [];
        
        for (const [key, cb] of this.circuitBreakers) {
            const [service, operation] = key.split(":");
            list.push({
                service,
                operation,
                state: cb.getState(),
            });
        }
        
        return list;
    }
    
    /**
     * Get a specific circuit breaker if it exists
     */
    getExistingCircuitBreaker(service: string, operation: string): MinimalCircuitBreaker | undefined {
        const key = `${service}:${operation}`;
        return this.circuitBreakers.get(key);
    }
    
    /**
     * Cleanup all circuit breakers
     */
    async destroy(): Promise<void> {
        for (const cb of this.circuitBreakers.values()) {
            cb.destroy();
        }
        this.circuitBreakers.clear();
        
        await this.emitShutdownEvent();
    }
    
    /**
     * Emit creation event
     */
    private async emitCreationEvent(
        service: string,
        operation: string,
        config: MinimalCircuitBreakerConfig,
    ): Promise<void> {
        await this.eventEmitter.emitMetric(
            "safety",
            "circuit_breaker_manager.created",
            1,
            "event",
            {
                service,
                operation,
                config: JSON.stringify(config),
            },
        );
    }
    
    /**
     * Emit access event
     */
    private async emitAccessEvent(
        service: string,
        operation: string,
        type: "existing" | "new",
    ): Promise<void> {
        await this.eventEmitter.emitMetric(
            "safety",
            "circuit_breaker_manager.accessed",
            1,
            "event",
            {
                service,
                operation,
                type,
            },
        );
    }
    
    /**
     * Emit removal event
     */
    private async emitRemovalEvent(
        service: string,
        operation: string,
    ): Promise<void> {
        await this.eventEmitter.emitMetric(
            "safety",
            "circuit_breaker_manager.removed",
            1,
            "event",
            {
                service,
                operation,
            },
        );
    }
    
    /**
     * Emit shutdown event
     */
    private async emitShutdownEvent(): Promise<void> {
        await this.eventEmitter.emitMetric(
            "safety",
            "circuit_breaker_manager.shutdown",
            1,
            "event",
            {
                totalCircuitBreakers: this.circuitBreakers.size,
            },
        );
    }
}

/**
 * Example circuit breaker management agent:
 * 
 * ```typescript
 * // This would be a routine deployed by teams, NOT hardcoded
 * class CircuitBreakerOptimizationAgent {
 *     async onCreationEvent(event: MetricEvent) {
 *         const { service, operation } = event.tags;
 *         
 *         // Analyze historical patterns for this service
 *         const patterns = await this.analyzeServicePatterns(service);
 *         
 *         // Recommend optimal configuration
 *         if (patterns.hasHighFailureRate) {
 *             await this.recommendConfiguration(service, operation, {
 *                 timeoutMs: 5000, // Shorter timeout for failing services
 *                 resetTimeoutMs: 60000, // Longer reset for problematic services
 *             });
 *         }
 *     }
 *     
 *     async onManagerShutdown(event: MetricEvent) {
 *         // Save learned patterns for next startup
 *         await this.persistLearnedPatterns();
 *     }
 * }
 * ```
 */
