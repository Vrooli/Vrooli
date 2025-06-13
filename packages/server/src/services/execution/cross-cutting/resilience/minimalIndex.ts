/**
 * Minimal Resilience Infrastructure
 * 
 * Provides ONLY basic resilience components.
 * All intelligence emerges from resilience agents.
 * 
 * DEPRECATED: The old resilience infrastructure with adaptive thresholds,
 * pattern detection, and learning will be removed. Use minimal components
 * that emit events for resilience agents to analyze.
 */

// New minimal components
export { MinimalCircuitBreaker } from "./minimalCircuitBreaker.js";
export { MinimalCircuitBreakerManager } from "./minimalCircuitBreakerManager.js";
export type { MinimalCircuitBreakerConfig } from "./minimalCircuitBreaker.js";

// Basic error classification (keep for now, simplify later)
export { ErrorClassifier } from "./errorClassifier.js";

// Event publishing (keep - this is already minimal)
export { ResilienceEventPublisher } from "./resilienceEventPublisher.js";

// DEPRECATED - to be removed
export { 
    AdaptiveCircuitBreaker, 
    CircuitBreakerOpenError, 
    CircuitBreakerFactory,
} from "./circuitBreaker.js";
export { CircuitBreakerManager } from "./circuitBreakerManager.js";
export { FallbackStrategyEngine, FallbackStrategy } from "./fallbackStrategies.js";
export { RecoverySelector } from "./recoverySelector.js";

/**
 * Re-export key types for convenience
 */
export type {
    ErrorClassification,
    ErrorContext,
    CircuitState,
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
} from "@vrooli/shared";

/**
 * Example resilience agent usage:
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
 *             // All intelligence emerges here
 *         }
 *     }
 * }
 * ```
 */