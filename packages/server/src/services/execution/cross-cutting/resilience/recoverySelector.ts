/**
 * Recovery Strategy Selector
 * Implements context-aware strategy selection based on error classification
 * Supports adaptive strategy selection that learns from outcomes
 */

import type {
    ErrorClassification,
    ErrorContext,
    RecoveryStrategy,
    RecoveryStrategyConfig,
    BackoffStrategyConfig,
    FallbackAction,
    RecoveryCondition,
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
    BackoffType,
    FallbackType,
    ConditionOperator,
} from "@vrooli/shared";
import {
    RecoveryStrategy as Strategy,
    ErrorSeverity,
    ErrorCategory,
    ErrorRecoverability,
    BackoffType,
    FallbackType,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";

/**
 * Strategy effectiveness tracking
 */
interface StrategyEffectiveness {
    strategy: RecoveryStrategy;
    successRate: number;
    averageDuration: number;
    resourceCost: number;
    attempts: number;
    lastUsed: Date;
    contextFactors: string[];
}

/**
 * Context scoring weights for strategy selection
 */
const CONTEXT_WEIGHTS = {
    severity: 0.3,
    category: 0.25,
    recoverability: 0.2,
    attemptCount: 0.15,
    resourceState: 0.1,
} as const;

/**
 * Recovery strategy selector with adaptive learning
 */
export class RecoverySelector {
    private strategyDatabase = new Map<string, RecoveryStrategyConfig>();
    private effectivenessHistory = new Map<string, StrategyEffectiveness[]>();
    private readonly maxHistorySize = 500;

    constructor() {
        this.initializeDefaultStrategies();
    }

    /**
     * Select optimal recovery strategy based on error classification and context
     */
    async selectStrategy(
        classification: ErrorClassification,
        context: ErrorContext,
    ): Promise<RecoveryStrategyConfig> {
        const startTime = performance.now();

        try {
            // Get candidate strategies
            const candidates = this.getCandidateStrategies(classification, context);
            
            if (candidates.length === 0) {
                logger.warn("No candidate strategies found, using fallback", {
                    classification,
                    tier: context.tier,
                    component: context.component,
                });
                return this.createFallbackStrategy(classification, context);
            }

            // Score and rank strategies
            const scoredStrategies = await Promise.all(
                candidates.map(async strategy => ({
                    strategy,
                    score: await this.scoreStrategy(strategy, classification, context),
                })),
            );

            // Sort by score (highest first)
            scoredStrategies.sort((a, b) => b.score - a.score);

            const selectedStrategy = scoredStrategies[0].strategy;
            
            // Adapt strategy based on context
            const adaptedStrategy = this.adaptStrategy(selectedStrategy, classification, context);

            const duration = performance.now() - startTime;
            logger.debug("Strategy selected", {
                selectedStrategy: adaptedStrategy.strategyType,
                score: scoredStrategies[0].score,
                candidateCount: candidates.length,
                duration: `${duration.toFixed(2)}ms`,
                tier: context.tier,
                component: context.component,
            });

            return adaptedStrategy;
        } catch (error) {
            logger.error("Error during strategy selection", error);
            return this.createFallbackStrategy(classification, context);
        }
    }

    /**
     * Get candidate strategies based on classification
     */
    private getCandidateStrategies(
        classification: ErrorClassification,
        context: ErrorContext,
    ): RecoveryStrategyConfig[] {
        const candidates: RecoveryStrategyConfig[] = [];

        // Filter strategies based on classification
        for (const strategy of this.strategyDatabase.values()) {
            if (this.isStrategyApplicable(strategy, classification, context)) {
                candidates.push(strategy);
            }
        }

        // If no direct matches, get broader strategies
        if (candidates.length === 0) {
            for (const strategy of this.strategyDatabase.values()) {
                if (this.isStrategyCompatible(strategy, classification, context)) {
                    candidates.push(strategy);
                }
            }
        }

        return candidates;
    }

    /**
     * Check if strategy is directly applicable
     */
    private isStrategyApplicable(
        strategy: RecoveryStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): boolean {
        // Check if strategy conditions are met
        for (const condition of strategy.conditions) {
            if (!this.evaluateCondition(condition, classification, context)) {
                return false;
            }
        }

        // Check resource requirements
        if (!this.hasRequiredResources(strategy, context)) {
            return false;
        }

        // Check if strategy hasn't been used too recently
        if (context.previousStrategies.includes(strategy.strategyType) &&
            strategy.strategyType !== Strategy.WAIT_AND_RETRY) {
            return false;
        }

        return true;
    }

    /**
     * Check if strategy is broadly compatible
     */
    private isStrategyCompatible(
        strategy: RecoveryStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): boolean {
        // Always allow escalation strategies
        if (strategy.strategyType === Strategy.ESCALATE_TO_PARENT ||
            strategy.strategyType === Strategy.ESCALATE_TO_HUMAN) {
            return true;
        }

        // Check severity compatibility
        if (classification.severity === ErrorSeverity.FATAL &&
            strategy.strategyType !== Strategy.EMERGENCY_STOP &&
            strategy.strategyType !== Strategy.ESCALATE_TO_HUMAN) {
            return false;
        }

        // Check recoverability compatibility
        if (classification.recoverability === ErrorRecoverability.NONE &&
            strategy.strategyType !== Strategy.EMERGENCY_STOP &&
            strategy.strategyType !== Strategy.ESCALATE_TO_HUMAN) {
            return false;
        }

        return true;
    }

    /**
     * Score strategy based on effectiveness and context
     */
    private async scoreStrategy(
        strategy: RecoveryStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): Promise<number> {
        let score = strategy.estimatedSuccessRate; // Base score

        // Adjust based on historical effectiveness
        const effectiveness = this.getStrategyEffectiveness(
            strategy.strategyType,
            classification,
            context,
        );
        
        if (effectiveness) {
            // Weight historical success rate more heavily
            score = (score * 0.3) + (effectiveness.successRate * 0.7);
            
            // Penalize recent failures
            if (effectiveness.attempts > 0) {
                const recentSuccessRate = effectiveness.successRate;
                if (recentSuccessRate < 0.5) {
                    score *= 0.8; // 20% penalty for poor recent performance
                }
            }
        }

        // Context-based adjustments
        score += this.calculateContextBonus(strategy, classification, context);

        // Resource efficiency adjustment
        score += this.calculateResourceEfficiencyBonus(strategy, context);

        // Penalty for repeated use
        if (context.previousStrategies.includes(strategy.strategyType)) {
            score *= 0.7; // 30% penalty for repetition
        }

        // Priority adjustment
        score += strategy.priority * 0.1;

        return Math.max(0, Math.min(1, score)); // Clamp to [0, 1]
    }

    /**
     * Calculate context-based bonus
     */
    private calculateContextBonus(
        strategy: RecoveryStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): number {
        let bonus = 0;

        // Severity-based bonuses
        if (classification.severity === ErrorSeverity.CRITICAL) {
            if (strategy.strategyType === Strategy.ESCALATE_TO_HUMAN ||
                strategy.strategyType === Strategy.EMERGENCY_STOP) {
                bonus += 0.2;
            }
        }

        // Category-based bonuses
        if (classification.category === ErrorCategory.TRANSIENT) {
            if (strategy.strategyType === Strategy.RETRY_SAME ||
                strategy.strategyType === Strategy.WAIT_AND_RETRY) {
                bonus += 0.15;
            }
        } else if (classification.category === ErrorCategory.RESOURCE) {
            if (strategy.strategyType === Strategy.REDUCE_SCOPE ||
                strategy.strategyType === Strategy.WAIT_AND_RETRY) {
                bonus += 0.15;
            }
        }

        // Tier-specific bonuses
        if (context.tier === 1) {
            if (strategy.strategyType === Strategy.ESCALATE_TO_HUMAN ||
                strategy.strategyType === Strategy.FALLBACK_STRATEGY) {
                bonus += 0.1;
            }
        } else if (context.tier === 3) {
            if (strategy.strategyType === Strategy.ESCALATE_TO_PARENT ||
                strategy.strategyType === Strategy.FALLBACK_MODEL) {
                bonus += 0.1;
            }
        }

        return bonus;
    }

    /**
     * Calculate resource efficiency bonus
     */
    private calculateResourceEfficiencyBonus(
        strategy: RecoveryStrategyConfig,
        context: ErrorContext,
    ): number {
        let bonus = 0;

        // Check available resources
        const resourceState = context.resourceState as any;
        
        if (resourceState?.memoryUsage > 0.8) {
            // High memory usage - prefer lighter strategies
            if (strategy.strategyType === Strategy.REDUCE_SCOPE ||
                strategy.strategyType === Strategy.GRACEFUL_DEGRADATION) {
                bonus += 0.1;
            }
        }

        if (resourceState?.cpuUsage > 0.8) {
            // High CPU usage - prefer simpler strategies
            if (strategy.strategyType === Strategy.LOG_WARNING ||
                strategy.strategyType === Strategy.ESCALATE_TO_PARENT) {
                bonus += 0.1;
            }
        }

        // Cost efficiency
        const totalResourceCost = Object.values(strategy.resourceRequirements)
            .reduce((sum, cost) => sum + cost, 0);
        
        if (totalResourceCost < 0.1) {
            bonus += 0.05; // Small bonus for low-cost strategies
        }

        return bonus;
    }

    /**
     * Get strategy effectiveness from history
     */
    private getStrategyEffectiveness(
        strategyType: RecoveryStrategy,
        classification: ErrorClassification,
        context: ErrorContext,
    ): StrategyEffectiveness | undefined {
        const contextKey = this.buildContextKey(classification, context);
        const history = this.effectivenessHistory.get(contextKey) || [];
        
        return history.find(h => h.strategy === strategyType);
    }

    /**
     * Adapt strategy configuration based on context
     */
    private adaptStrategy(
        baseStrategy: RecoveryStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): RecoveryStrategyConfig {
        const adapted = { ...baseStrategy };

        // Adjust max attempts based on severity and previous attempts
        if (classification.severity === ErrorSeverity.CRITICAL) {
            adapted.maxAttempts = Math.min(adapted.maxAttempts, 2);
        } else if (context.attemptCount > 2) {
            adapted.maxAttempts = Math.max(1, adapted.maxAttempts - context.attemptCount);
        }

        // Adjust timeout based on context
        if (context.performanceMetrics.averageResponseTime) {
            const avgResponseTime = context.performanceMetrics.averageResponseTime;
            adapted.timeoutMs = Math.max(
                adapted.timeoutMs,
                avgResponseTime * 2, // At least 2x average response time
            );
        }

        // Adjust backoff strategy based on error category
        if (classification.category === ErrorCategory.RESOURCE ||
            classification.category === ErrorCategory.TRANSIENT) {
            adapted.backoffStrategy = this.adaptBackoffStrategy(
                adapted.backoffStrategy,
                classification,
                context,
            );
        }

        // Add context-specific fallback actions
        adapted.fallbackActions = [
            ...adapted.fallbackActions,
            ...this.generateContextualFallbacks(classification, context),
        ];

        return adapted;
    }

    /**
     * Adapt backoff strategy
     */
    private adaptBackoffStrategy(
        baseBackoff: BackoffStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): BackoffStrategyConfig {
        const adapted = { ...baseBackoff };

        // Increase delays for resource errors
        if (classification.category === ErrorCategory.RESOURCE) {
            adapted.initialDelayMs *= 2;
            adapted.maxDelayMs *= 1.5;
        }

        // Use exponential backoff with jitter for network errors
        if (classification.category === ErrorCategory.TRANSIENT) {
            adapted.type = BackoffType.EXPONENTIAL_JITTER;
            adapted.jitterPercent = 0.1;
        }

        // Adaptive adjustment for repeated failures
        if (context.attemptCount > 1) {
            adapted.adaptiveAdjustment = true;
            adapted.multiplier = Math.min(adapted.multiplier * 1.2, 3.0);
        }

        return adapted;
    }

    /**
     * Generate contextual fallback actions
     */
    private generateContextualFallbacks(
        classification: ErrorClassification,
        context: ErrorContext,
    ): FallbackAction[] {
        const fallbacks: FallbackAction[] = [];

        // Security-related fallbacks
        if (classification.securityRisk) {
            fallbacks.push({
                type: FallbackType.MANUAL_INTERVENTION,
                configuration: {
                    reason: "Security risk detected",
                    escalationLevel: "immediate",
                },
                conditions: [],
                priority: 10,
                estimatedSuccessRate: 0.9,
                qualityReduction: 0.0,
                resourceAdjustment: {},
            });
        }

        // Data risk fallbacks
        if (classification.dataRisk) {
            fallbacks.push({
                type: FallbackType.USE_CACHED_RESULT,
                configuration: {
                    maxAge: 300000, // 5 minutes
                    staleThreshold: 0.8,
                },
                conditions: [],
                priority: 8,
                estimatedSuccessRate: 0.7,
                qualityReduction: 0.2,
                resourceAdjustment: { cache_hits: 1 },
            });
        }

        // Tier-specific fallbacks
        if (context.tier === 3) {
            fallbacks.push({
                type: FallbackType.ALTERNATE_TOOL,
                configuration: {
                    preferredTools: ["backup_executor", "simple_executor"],
                },
                conditions: [],
                priority: 6,
                estimatedSuccessRate: 0.6,
                qualityReduction: 0.3,
                resourceAdjustment: { tool_switches: 1 },
            });
        }

        return fallbacks;
    }

    /**
     * Evaluate recovery condition
     */
    private evaluateCondition(
        condition: RecoveryCondition,
        classification: ErrorClassification,
        context: ErrorContext,
    ): boolean {
        const value = this.getConditionValue(condition.field, classification, context);
        return this.compareValues(value, condition.value, condition.operator);
    }

    /**
     * Get condition value from classification or context
     */
    private getConditionValue(
        field: string,
        classification: ErrorClassification,
        context: ErrorContext,
    ): unknown {
        // Check classification first
        if (field in classification) {
            return (classification as any)[field];
        }

        // Check context
        if (field in context) {
            return (context as any)[field];
        }

        // Check nested fields
        const dotIndex = field.indexOf(".");
        if (dotIndex > 0) {
            const obj = field.startsWith("classification.") ? classification : context;
            const nestedField = field.substring(dotIndex + 1);
            return (obj as any)[nestedField];
        }

        return undefined;
    }

    /**
     * Compare values using operator
     */
    private compareValues(
        actual: unknown,
        expected: unknown,
        operator: ConditionOperator,
    ): boolean {
        switch (operator) {
            case "EQUALS":
                return actual === expected;
            case "NOT_EQUALS":
                return actual !== expected;
            case "GREATER_THAN":
                return typeof actual === "number" && typeof expected === "number" &&
                    actual > expected;
            case "LESS_THAN":
                return typeof actual === "number" && typeof expected === "number" &&
                    actual < expected;
            case "CONTAINS":
                return typeof actual === "string" && typeof expected === "string" &&
                    actual.includes(expected);
            case "IN":
                return Array.isArray(expected) && expected.includes(actual);
            case "NOT_IN":
                return Array.isArray(expected) && !expected.includes(actual);
            case "REGEX_MATCH":
                return typeof actual === "string" && typeof expected === "string" &&
                    new RegExp(expected).test(actual);
            default:
                return false;
        }
    }

    /**
     * Check if required resources are available
     */
    private hasRequiredResources(
        strategy: RecoveryStrategyConfig,
        context: ErrorContext,
    ): boolean {
        const resourceState = context.resourceState as any;
        
        for (const [resource, required] of Object.entries(strategy.resourceRequirements)) {
            const available = resourceState[resource] || 0;
            if (available < required) {
                return false;
            }
        }
        
        return true;
    }

    /**
     * Create fallback strategy for unrecoverable situations
     */
    private createFallbackStrategy(
        classification: ErrorClassification,
        context: ErrorContext,
    ): RecoveryStrategyConfig {
        // Critical errors require immediate escalation
        if (classification.severity === ErrorSeverity.FATAL ||
            classification.severity === ErrorSeverity.CRITICAL) {
            return this.strategyDatabase.get("emergency_stop") ||
                this.createEmergencyStopStrategy();
        }

        // Security risks require human intervention
        if (classification.securityRisk) {
            return this.strategyDatabase.get("escalate_to_human") ||
                this.createEscalateToHumanStrategy();
        }

        // Data risks need careful handling
        if (classification.dataRisk) {
            return this.strategyDatabase.get("graceful_degradation") ||
                this.createGracefulDegradationStrategy();
        }

        // Default fallback is to escalate to parent tier
        return this.strategyDatabase.get("escalate_to_parent") ||
            this.createEscalateToParentStrategy();
    }

    /**
     * Record strategy outcome for learning
     */
    recordOutcome(
        strategyType: RecoveryStrategy,
        classification: ErrorClassification,
        context: ErrorContext,
        success: boolean,
        duration: number,
        resourceCost: number,
    ): void {
        const contextKey = this.buildContextKey(classification, context);
        const history = this.effectivenessHistory.get(contextKey) || [];
        
        // Find or create effectiveness record
        let effectiveness = history.find(h => h.strategy === strategyType);
        if (!effectiveness) {
            effectiveness = {
                strategy: strategyType,
                successRate: 0,
                averageDuration: 0,
                resourceCost: 0,
                attempts: 0,
                lastUsed: new Date(),
                contextFactors: this.extractContextFactors(classification, context),
            };
            history.push(effectiveness);
        }

        // Update effectiveness metrics
        const totalAttempts = effectiveness.attempts + 1;
        const successCount = (effectiveness.successRate * effectiveness.attempts) + (success ? 1 : 0);
        
        effectiveness.successRate = successCount / totalAttempts;
        effectiveness.averageDuration = (
            (effectiveness.averageDuration * effectiveness.attempts) + duration
        ) / totalAttempts;
        effectiveness.resourceCost = (
            (effectiveness.resourceCost * effectiveness.attempts) + resourceCost
        ) / totalAttempts;
        effectiveness.attempts = totalAttempts;
        effectiveness.lastUsed = new Date();

        // Limit history size
        if (history.length > this.maxHistorySize) {
            history.splice(0, history.length - this.maxHistorySize);
        }

        this.effectivenessHistory.set(contextKey, history);

        logger.debug("Strategy outcome recorded", {
            strategy: strategyType,
            success,
            duration,
            newSuccessRate: effectiveness.successRate,
            totalAttempts: effectiveness.attempts,
        });
    }

    /**
     * Build context key for effectiveness tracking
     */
    private buildContextKey(
        classification: ErrorClassification,
        context: ErrorContext,
    ): string {
        return [
            classification.severity,
            classification.category,
            classification.recoverability,
            context.tier,
            context.component,
        ].join("::");
    }

    /**
     * Extract context factors for effectiveness analysis
     */
    private extractContextFactors(
        classification: ErrorClassification,
        context: ErrorContext,
    ): string[] {
        const factors = [
            `severity:${classification.severity}`,
            `category:${classification.category}`,
            `tier:${context.tier}`,
            `attempts:${Math.min(context.attemptCount, 5)}`, // Cap at 5 for grouping
        ];

        if (classification.dataRisk) factors.push("data_risk");
        if (classification.securityRisk) factors.push("security_risk");
        if (classification.multipleComponentsAffected) factors.push("multiple_components");

        return factors;
    }

    /**
     * Initialize default recovery strategies
     */
    private initializeDefaultStrategies(): void {
        // Emergency stop strategy
        this.strategyDatabase.set("emergency_stop", this.createEmergencyStopStrategy());
        
        // Escalation strategies
        this.strategyDatabase.set("escalate_to_human", this.createEscalateToHumanStrategy());
        this.strategyDatabase.set("escalate_to_parent", this.createEscalateToParentStrategy());
        
        // Retry strategies
        this.strategyDatabase.set("retry_same", this.createRetrySameStrategy());
        this.strategyDatabase.set("wait_and_retry", this.createWaitAndRetryStrategy());
        this.strategyDatabase.set("retry_modified", this.createRetryModifiedStrategy());
        
        // Fallback strategies
        this.strategyDatabase.set("fallback_strategy", this.createFallbackStrategyConfig());
        this.strategyDatabase.set("fallback_model", this.createFallbackModelStrategy());
        this.strategyDatabase.set("reduce_scope", this.createReduceScopeStrategy());
        this.strategyDatabase.set("graceful_degradation", this.createGracefulDegradationStrategy());
        
        // Logging strategies
        this.strategyDatabase.set("log_warning", this.createLogWarningStrategy());
        this.strategyDatabase.set("log_info", this.createLogInfoStrategy());

        logger.info("Recovery strategy database initialized", {
            strategyCount: this.strategyDatabase.size,
        });
    }

    /**
     * Strategy creation methods
     */
    private createEmergencyStopStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.EMERGENCY_STOP,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 10,
            timeoutMs: 1000,
            conditions: [
                {
                    field: "severity",
                    operator: "IN",
                    value: [ErrorSeverity.FATAL, ErrorSeverity.CRITICAL],
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 1.0,
            resourceRequirements: {},
        };
    }

    private createEscalateToHumanStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.ESCALATE_TO_HUMAN,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 9,
            timeoutMs: 5000,
            conditions: [
                {
                    field: "securityRisk",
                    operator: "EQUALS",
                    value: true,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.95,
            resourceRequirements: { human_attention: 1 },
        };
    }

    private createEscalateToParentStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.ESCALATE_TO_PARENT,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 7,
            timeoutMs: 10000,
            conditions: [
                {
                    field: "tier",
                    operator: "GREATER_THAN",
                    value: 1,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.8,
            resourceRequirements: { escalation_capacity: 1 },
        };
    }

    private createRetrySameStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.RETRY_SAME,
            maxAttempts: 3,
            backoffStrategy: {
                type: BackoffType.EXPONENTIAL,
                initialDelayMs: 100,
                maxDelayMs: 5000,
                multiplier: 2,
                jitterPercent: 0.1,
                adaptiveAdjustment: true,
            },
            fallbackActions: [],
            priority: 5,
            timeoutMs: 30000,
            conditions: [
                {
                    field: "category",
                    operator: "EQUALS",
                    value: ErrorCategory.TRANSIENT,
                    weight: 1.0,
                },
                {
                    field: "attemptCount",
                    operator: "LESS_THAN",
                    value: 3,
                    weight: 0.8,
                },
            ],
            estimatedSuccessRate: 0.7,
            resourceRequirements: { retry_budget: 1 },
        };
    }

    private createWaitAndRetryStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.WAIT_AND_RETRY,
            maxAttempts: 3,
            backoffStrategy: {
                type: BackoffType.EXPONENTIAL_JITTER,
                initialDelayMs: 1000,
                maxDelayMs: 30000,
                multiplier: 2,
                jitterPercent: 0.2,
                adaptiveAdjustment: true,
            },
            fallbackActions: [],
            priority: 6,
            timeoutMs: 60000,
            conditions: [
                {
                    field: "category",
                    operator: "IN",
                    value: [ErrorCategory.RESOURCE, ErrorCategory.TRANSIENT],
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.75,
            resourceRequirements: { wait_time: 30 },
        };
    }

    private createRetryModifiedStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.RETRY_MODIFIED,
            maxAttempts: 2,
            backoffStrategy: {
                type: BackoffType.LINEAR,
                initialDelayMs: 500,
                maxDelayMs: 10000,
                multiplier: 1.5,
                jitterPercent: 0.1,
                adaptiveAdjustment: true,
            },
            fallbackActions: [
                {
                    type: FallbackType.REDUCE_SCOPE,
                    configuration: { reductionFactor: 0.5 },
                    conditions: [],
                    priority: 8,
                    estimatedSuccessRate: 0.8,
                    qualityReduction: 0.3,
                    resourceAdjustment: { scope_reduction: 1 },
                },
            ],
            priority: 6,
            timeoutMs: 45000,
            conditions: [
                {
                    field: "recoverability",
                    operator: "IN",
                    value: [ErrorRecoverability.AUTOMATIC, ErrorRecoverability.PARTIAL],
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.65,
            resourceRequirements: { modification_effort: 1 },
        };
    }

    private createFallbackStrategyConfig(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.FALLBACK_STRATEGY,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [
                {
                    type: FallbackType.ALTERNATE_STRATEGY,
                    configuration: { strategiesPool: ["simple", "conservative"] },
                    conditions: [],
                    priority: 7,
                    estimatedSuccessRate: 0.6,
                    qualityReduction: 0.2,
                    resourceAdjustment: { strategy_switch: 1 },
                },
            ],
            priority: 5,
            timeoutMs: 20000,
            conditions: [
                {
                    field: "category",
                    operator: "NOT_EQUALS",
                    value: ErrorCategory.SECURITY,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.6,
            resourceRequirements: { alternate_strategy: 1 },
        };
    }

    private createFallbackModelStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.FALLBACK_MODEL,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [
                {
                    type: FallbackType.ALTERNATE_MODEL,
                    configuration: { modelTier: "lower", preserveContext: true },
                    conditions: [],
                    priority: 6,
                    estimatedSuccessRate: 0.7,
                    qualityReduction: 0.3,
                    resourceAdjustment: { model_switch: 1 },
                },
            ],
            priority: 6,
            timeoutMs: 25000,
            conditions: [
                {
                    field: "tier",
                    operator: "EQUALS",
                    value: 3,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.7,
            resourceRequirements: { alternate_model: 1 },
        };
    }

    private createReduceScopeStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.REDUCE_SCOPE,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [
                {
                    type: FallbackType.REDUCE_SCOPE,
                    configuration: { reductionStrategy: "progressive", maxReduction: 0.7 },
                    conditions: [],
                    priority: 7,
                    estimatedSuccessRate: 0.8,
                    qualityReduction: 0.4,
                    resourceAdjustment: { scope_reduction: 2 },
                },
            ],
            priority: 4,
            timeoutMs: 15000,
            conditions: [
                {
                    field: "category",
                    operator: "IN",
                    value: [ErrorCategory.RESOURCE, ErrorCategory.LOGIC],
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.8,
            resourceRequirements: { scope_adjustment: 1 },
        };
    }

    private createGracefulDegradationStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.GRACEFUL_DEGRADATION,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [
                {
                    type: FallbackType.PARTIAL_SERVICE,
                    configuration: { serviceLevel: 0.6, maintainCore: true },
                    conditions: [],
                    priority: 6,
                    estimatedSuccessRate: 0.9,
                    qualityReduction: 0.4,
                    resourceAdjustment: { degradation_level: 1 },
                },
            ],
            priority: 4,
            timeoutMs: 10000,
            conditions: [
                {
                    field: "dataRisk",
                    operator: "EQUALS",
                    value: true,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 0.85,
            resourceRequirements: { degradation_capacity: 1 },
        };
    }

    private createLogWarningStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.LOG_WARNING,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 2,
            timeoutMs: 1000,
            conditions: [
                {
                    field: "severity",
                    operator: "EQUALS",
                    value: ErrorSeverity.WARNING,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 1.0,
            resourceRequirements: {},
        };
    }

    private createLogInfoStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: Strategy.LOG_INFO,
            maxAttempts: 1,
            backoffStrategy: {
                type: BackoffType.FIXED,
                initialDelayMs: 0,
                maxDelayMs: 0,
                multiplier: 1,
                jitterPercent: 0,
                adaptiveAdjustment: false,
            },
            fallbackActions: [],
            priority: 1,
            timeoutMs: 500,
            conditions: [
                {
                    field: "severity",
                    operator: "EQUALS",
                    value: ErrorSeverity.INFO,
                    weight: 1.0,
                },
            ],
            estimatedSuccessRate: 1.0,
            resourceRequirements: {},
        };
    }

    /**
     * Get strategy effectiveness statistics
     */
    getEffectivenessStatistics(): {
        strategiesTracked: number;
        totalOutcomes: number;
        averageSuccessRate: number;
        bestPerformingStrategies: Array<{
            strategy: RecoveryStrategy;
            successRate: number;
            attempts: number;
        }>;
    } {
        let totalOutcomes = 0;
        let totalSuccessRate = 0;
        const strategyPerformance = new Map<RecoveryStrategy, { successRate: number; attempts: number }>();

        for (const history of this.effectivenessHistory.values()) {
            for (const effectiveness of history) {
                totalOutcomes += effectiveness.attempts;
                totalSuccessRate += effectiveness.successRate * effectiveness.attempts;

                const existing = strategyPerformance.get(effectiveness.strategy);
                if (existing) {
                    const combinedAttempts = existing.attempts + effectiveness.attempts;
                    const combinedSuccessRate = (
                        (existing.successRate * existing.attempts) +
                        (effectiveness.successRate * effectiveness.attempts)
                    ) / combinedAttempts;

                    strategyPerformance.set(effectiveness.strategy, {
                        successRate: combinedSuccessRate,
                        attempts: combinedAttempts,
                    });
                } else {
                    strategyPerformance.set(effectiveness.strategy, {
                        successRate: effectiveness.successRate,
                        attempts: effectiveness.attempts,
                    });
                }
            }
        }

        const bestPerformingStrategies = Array.from(strategyPerformance.entries())
            .map(([strategy, perf]) => ({
                strategy,
                successRate: perf.successRate,
                attempts: perf.attempts,
            }))
            .filter(s => s.attempts >= 3) // Only strategies with sufficient data
            .sort((a, b) => b.successRate - a.successRate)
            .slice(0, 5); // Top 5

        return {
            strategiesTracked: strategyPerformance.size,
            totalOutcomes,
            averageSuccessRate: totalOutcomes > 0 ? totalSuccessRate / totalOutcomes : 0,
            bestPerformingStrategies,
        };
    }
}