/**
 * Resilience Infrastructure Index
 * Exports core resilience components for emergent intelligence
 * 
 * This module provides the foundation for AI swarms to learn resilience patterns
 * through systematic error classification, intelligent recovery strategy selection,
 * circuit breaking with adaptive thresholds, intelligent fallback strategies,
 * and rich event publishing for pattern detection and learning.
 */

export { ErrorClassifier } from "./errorClassifier.js";
export { RecoverySelector } from "./recoverySelector.js";
export { ResilienceEventPublisher } from "./resilienceEventPublisher.js";

// Circuit Breaker Components
export { 
    AdaptiveCircuitBreaker, 
    CircuitBreakerOpenError, 
    CircuitBreakerFactory,
} from "./circuitBreaker.js";
export { CircuitBreakerManager } from "./circuitBreakerManager.js";

// Fallback Strategy Components
export { 
    FallbackStrategyEngine, 
    FallbackStrategy,
} from "./fallbackStrategies.js";

/**
 * Re-export key types for convenience
 */
export type {
    ErrorClassification,
    ErrorContext,
    RecoveryStrategyConfig,
    ResilienceEvent,
    ResilienceEventType,
    ResilienceOutcome,
    ResilienceLearningData,
    ErrorPattern,
    RecoveryStrategy,
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
    UserImpactLevel,
} from "@vrooli/shared";

/**
 * Resilience Infrastructure Factory
 * Creates coordinated resilience components with proper dependency injection
 */
export class ResilienceInfrastructure {
    private errorClassifier: ErrorClassifier;
    private recoverySelector: RecoverySelector;
    private eventPublisher: ResilienceEventPublisher;
    private circuitBreakerManager: CircuitBreakerManager;
    private fallbackEngine: FallbackStrategyEngine;

    constructor(
        telemetryShim: any, // TelemetryShim type
        eventBus: any, // RedisEventBus type
    ) {
        // Initialize components
        this.errorClassifier = new ErrorClassifier();
        this.recoverySelector = new RecoverySelector();
        this.eventPublisher = new ResilienceEventPublisher(telemetryShim, eventBus);
        this.circuitBreakerManager = new CircuitBreakerManager(
            telemetryShim,
            eventBus,
            this.errorClassifier,
        );
        this.fallbackEngine = new FallbackStrategyEngine(telemetryShim, eventBus);
    }

    /**
     * Get error classifier instance
     */
    getErrorClassifier(): ErrorClassifier {
        return this.errorClassifier;
    }

    /**
     * Get recovery selector instance
     */
    getRecoverySelector(): RecoverySelector {
        return this.recoverySelector;
    }

    /**
     * Get event publisher instance
     */
    getEventPublisher(): ResilienceEventPublisher {
        return this.eventPublisher;
    }

    /**
     * Get circuit breaker manager instance
     */
    getCircuitBreakerManager(): CircuitBreakerManager {
        return this.circuitBreakerManager;
    }

    /**
     * Get fallback strategy engine instance
     */
    getFallbackEngine(): FallbackStrategyEngine {
        return this.fallbackEngine;
    }

    /**
     * Coordinated error handling workflow
     * Provides a high-level interface for the complete resilience flow
     */
    async handleError(
        error: Error,
        context: any, // ErrorContext type
        source: any, // ResilienceEventSource type
    ): Promise<{
        classification: any; // ErrorClassification type
        strategy: any; // RecoveryStrategyConfig type
        eventId: string;
    }> {
        // Step 1: Classify the error
        const classification = await this.errorClassifier.classify(error, context);
        
        // Step 2: Publish error detection event
        await this.eventPublisher.publishErrorDetected(error, classification, context, source);
        
        // Step 3: Publish classification event
        await this.eventPublisher.publishErrorClassified(
            classification,
            context,
            source,
            performance.now(),
        );
        
        // Step 4: Select recovery strategy
        const strategy = await this.recoverySelector.selectStrategy(classification, context);
        
        // Step 5: Publish recovery initiation
        await this.eventPublisher.publishRecoveryInitiated(
            classification,
            context,
            strategy,
            source,
        );
        
        return {
            classification,
            strategy,
            eventId: source.requestId,
        };
    }

    /**
     * Record recovery outcome for learning
     */
    async recordRecoveryOutcome(
        classification: any, // ErrorClassification type
        context: any, // ErrorContext type
        strategy: any, // RecoveryStrategyConfig type
        outcome: any, // ResilienceOutcome type
        source: any, // ResilienceEventSource type
        error?: Error,
    ): Promise<void> {
        // Record in recovery selector for learning
        this.recoverySelector.recordOutcome(
            strategy.strategyType,
            classification,
            context,
            outcome.success,
            outcome.duration,
            this.calculateResourceCost(outcome.resourceUsage),
        );
        
        // Publish appropriate event
        if (outcome.success) {
            await this.eventPublisher.publishRecoveryCompleted(
                classification,
                context,
                strategy,
                outcome,
                source,
            );
        } else {
            await this.eventPublisher.publishRecoveryFailed(
                classification,
                context,
                strategy,
                outcome,
                source,
                error,
            );
        }
    }

    /**
     * Add learned error pattern
     */
    addErrorPattern(pattern: any /* ErrorPattern type */): void {
        this.errorClassifier.addPattern(pattern);
    }

    /**
     * Get comprehensive statistics
     */
    getStatistics(): {
        classification: {
            totalClassifications: number;
            uniqueErrors: number;
            patterns: number;
            averageConfidence: number;
        };
        recovery: {
            strategiesTracked: number;
            totalOutcomes: number;
            averageSuccessRate: number;
            bestPerformingStrategies: Array<{
                strategy: string;
                successRate: number;
                attempts: number;
            }>;
        };
        publishing: {
            publishedEvents: number;
            droppedEvents: number;
            bufferedEvents: number;
            averageOverheadMs: number;
            patternCacheSize: number;
        };
    } {
        return {
            classification: this.errorClassifier.getStatistics(),
            recovery: this.recoverySelector.getEffectivenessStatistics(),
            publishing: this.eventPublisher.getStatistics(),
        };
    }

    /**
     * Execute operation with comprehensive resilience protection
     */
    async executeWithProtection<T>(
        service: string,
        operation: string,
        operationFn: () => Promise<T>,
        config?: {
            circuitBreakerConfig?: any; // CircuitBreakerConfig type
            fallbackConfig?: any; // FallbackAction type
            timeoutMs?: number;
        },
    ): Promise<T> {
        return this.circuitBreakerManager.executeWithProtection(
            service,
            operation,
            operationFn,
            config?.circuitBreakerConfig,
        );
    }

    /**
     * Shutdown all components gracefully
     */
    async shutdown(): Promise<void> {
        await this.eventPublisher.stop();
        await this.circuitBreakerManager.shutdown();
        await this.fallbackEngine.shutdown();
    }

    /**
     * Helper method to calculate resource cost
     */
    private calculateResourceCost(resourceUsage: Record<string, number>): number {
        return Object.values(resourceUsage).reduce((sum, cost) => sum + (cost || 0), 0);
    }
}

/**
 * Default instance factory
 * Creates a pre-configured resilience infrastructure instance
 */
export function createResilienceInfrastructure(
    telemetryShim: any,
    eventBus: any,
    options?: {
        enablePatternDetection?: boolean;
        enableLearning?: boolean;
        maxOverheadMs?: number;
    },
): ResilienceInfrastructure {
    return new ResilienceInfrastructure(telemetryShim, eventBus);
}

/**
 * Usage Examples and Integration Patterns
 * 
 * Basic Error Handling:
 * ```typescript
 * const resilience = createResilienceInfrastructure(telemetryShim, eventBus);
 * 
 * try {
 *   // Some operation that might fail
 * } catch (error) {
 *   const { classification, strategy } = await resilience.handleError(
 *     error,
 *     context,
 *     source
 *   );
 *   
 *   // Execute recovery strategy
 *   const outcome = await executeStrategy(strategy);
 *   
 *   // Record outcome for learning
 *   await resilience.recordRecoveryOutcome(
 *     classification,
 *     context,
 *     strategy,
 *     outcome,
 *     source
 *   );
 * }
 * ```
 * 
 * Circuit Breaker Integration:
 * ```typescript
 * const resilience = createResilienceInfrastructure(telemetryShim, eventBus);
 * 
 * // Execute with circuit breaker protection
 * try {
 *   const result = await resilience.executeWithProtection(
 *     "llm-service", 
 *     "generateResponse",
 *     () => llmService.generateResponse(input),
 *     {
 *       circuitBreakerConfig: { failureThreshold: 3, timeoutMs: 30000 },
 *       timeoutMs: 45000
 *     }
 *   );
 * } catch (error) {
 *   // Circuit breaker or operation failed
 *   console.error("Operation failed:", error);
 * }
 * ```
 * 
 * Agent Learning Integration:
 * ```typescript
 * // Agents can subscribe to resilience events for learning
 * eventBus.subscribe({
 *   id: "resilience-learning-agent",
 *   filters: [{ field: "type", value: "resilience.*" }],
 *   handler: async (event) => {
 *     // Extract patterns and update agent knowledge
 *     await agent.learnFromResilienceEvent(event);
 *   }
 * });
 * ```
 * 
 * Pattern Detection:
 * ```typescript
 * // Add learned patterns to improve classification
 * resilience.addErrorPattern({
 *   id: "network-timeout-pattern",
 *   name: "Network Timeout Pattern",
 *   description: "Recurring network timeouts in tier 3",
 *   frequency: 15,
 *   severity: ErrorSeverity.ERROR,
 *   category: ErrorCategory.TRANSIENT,
 *   triggerConditions: [
 *     { field: "errorMessage", operator: "CONTAINS", value: "timeout", weight: 0.8 },
 *     { field: "tier", operator: "EQUALS", value: 3, weight: 0.6 }
 *   ],
 *   effectiveStrategies: [waitAndRetryStrategy],
 *   successRate: 0.85,
 *   averageResolutionTime: 5000,
 *   lastSeen: new Date(),
 *   confidence: 0.9
 * });
 * ```
 */