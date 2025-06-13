/**
 * Minimal Resilience Infrastructure
 * 
 * Provides ONLY basic resilience components.
 * All intelligence emerges from resilience agents.
 * 
 * PRINCIPLE: This module follows the minimal architecture principle.
 * Complex error classification, recovery strategy optimization, and 
 * pattern detection should be handled by resilience agents through
 * event subscriptions, not hardcoded here.
 */

// New minimal components
export { MinimalCircuitBreaker } from "./minimalCircuitBreaker.js";
export { MinimalCircuitBreakerManager } from "./minimalCircuitBreakerManager.js";
export type { MinimalCircuitBreakerConfig } from "./minimalCircuitBreaker.js";

// Event publishing (already minimal)
export { ResilienceEventPublisher } from "./resilienceEventPublisher.js";

// Basic error classification (will be simplified further)
export { ErrorClassifier } from "./errorClassifier.js";
export { RecoverySelector } from "./recoverySelector.js";

// DEPRECATED - Complex adaptive components that should be agent-driven
export { 
    AdaptiveCircuitBreaker, 
    CircuitBreakerOpenError, 
    CircuitBreakerFactory,
} from "./circuitBreaker.js";
export { CircuitBreakerManager } from "./circuitBreakerManager.js";
export { FallbackStrategyEngine, FallbackStrategy } from "./fallbackStrategies.js";

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
 * Example resilience agent that would use these minimal components:
 * 
 * ```typescript
 * // Deploy resilience management agent as a routine
 * class ResilienceManagementAgent {
 *     constructor(private eventBus: EventBus) {
 *         // Subscribe to circuit breaker events
 *         eventBus.subscribe("execution.metrics.type.safety", this.analyzeResilience);
 *     }
 *     
 *     async analyzeResilience(event: MetricEvent) {
 *         if (event.name.startsWith("circuit_breaker.")) {
 *             // Track failures and successes
 *             // Decide when to open/close circuits
 *             // Learn optimal thresholds
 *             // Detect error patterns
 *             // All intelligence emerges here, not in infrastructure
 *         }
 *     }
 * }
 * ```
 */

/**
 * Minimal Usage - Agents provide the intelligence:
 * 
 * ```typescript
 * // Infrastructure provides events, agents provide intelligence
 * const publisher = new ResilienceEventPublisher(eventBus);
 * const circuitBreaker = new MinimalCircuitBreaker(config);
 * 
 * // Agent decides recovery strategy based on events
 * eventBus.subscribe("execution.events.error", async (event) => {
 *   const strategy = await resilienceAgent.analyzeAndRecommend(event);
 *   await resilienceAgent.executeStrategy(strategy);
 * });
 * ```
 */