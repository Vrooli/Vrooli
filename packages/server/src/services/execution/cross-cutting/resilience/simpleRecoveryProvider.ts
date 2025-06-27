/**
 * Simple Recovery Provider
 * 
 * Provides basic recovery strategy lookup that emits events for resilience
 * agents to analyze and optimize. Replaces the complex RecoverySelector.
 */

import { type Logger } from "winston";
import { type EventBus } from "../events/eventBus.js";
import {
    type ErrorClassification,
    type ErrorContext,
    type RecoveryStrategyConfig,
    RecoveryStrategy,
    ErrorSeverity,
    ErrorCategory,
    BackoffType,
} from "@vrooli/shared";

/**
 * Simple Recovery Provider - Basic strategy lookup + event emission
 * 
 * Provides simple recovery strategy selection based on basic rules.
 * Complex optimization and learning happens in resilience agents.
 */
export class SimpleRecoveryProvider {
    private readonly basicStrategies: Map<string, RecoveryStrategyConfig>;
    private readonly eventBus: EventBus;
    private readonly logger: Logger;

    constructor(eventBus: EventBus, logger: Logger) {
        this.eventBus = eventBus;
        this.logger = logger;
        this.basicStrategies = new Map();
        this.initializeBasicStrategies();
    }

    /**
     * Gets recovery strategy using basic lookup + emits events for agent optimization
     */
    async getStrategy(
        classification: ErrorClassification,
        context: ErrorContext,
    ): Promise<RecoveryStrategyConfig> {
        // 1. Basic strategy lookup (configurable via data)
        const strategyKey = this.buildStrategyKey(classification);
        let strategy = this.basicStrategies.get(strategyKey) || this.getDefaultStrategy(classification);

        // 2. Basic context adaptation (minimal logic)
        strategy = this.adaptBasic(strategy, classification, context);

        // 3. Emit event for resilience agents to analyze and optimize
        await this.eventBus.publish("recovery/strategy_selected", {
            classification: {
                severity: classification.severity,
                category: classification.category,
                recoverability: classification.recoverability,
            },
            context: {
                tier: context.tier,
                component: context.component,
                attemptCount: context.attemptCount,
            },
            selectedStrategy: strategy.strategyType,
            strategyKey,
            selectionReason: "basic_lookup",
            timestamp: new Date(),
        });

        this.logger.debug("[SimpleRecoveryProvider] Strategy selected", {
            strategy: strategy.strategyType,
            severity: classification.severity,
            category: classification.category,
        });

        return strategy;
    }

    /**
     * Records strategy outcome for agent learning
     */
    async recordOutcome(
        strategyType: RecoveryStrategy,
        classification: ErrorClassification,
        context: ErrorContext,
        success: boolean,
        duration: number,
        resourceCost: number,
    ): Promise<void> {
        // Emit outcome event for learning agents
        await this.eventBus.publish("recovery/outcome", {
            strategyType,
            classification: {
                severity: classification.severity,
                category: classification.category,
                recoverability: classification.recoverability,
            },
            context: {
                tier: context.tier,
                component: context.component,
                attemptCount: context.attemptCount,
            },
            success,
            duration,
            resourceCost,
            timestamp: new Date(),
        });
    }

    /**
     * Basic strategy key generation
     */
    private buildStrategyKey(classification: ErrorClassification): string {
        return `${classification.severity}:${classification.category}`;
    }

    /**
     * Basic context adaptation (minimal intelligence)
     */
    private adaptBasic(
        strategy: RecoveryStrategyConfig,
        classification: ErrorClassification,
        context: ErrorContext,
    ): RecoveryStrategyConfig {
        const adapted = { ...strategy };

        // Only essential adaptations - let agents handle complex logic
        if (classification.severity === ErrorSeverity.CRITICAL) {
            adapted.maxAttempts = Math.min(adapted.maxAttempts, 2);
        }

        if (context.attemptCount > 2) {
            adapted.maxAttempts = Math.max(1, adapted.maxAttempts - context.attemptCount);
        }

        return adapted;
    }

    /**
     * Get default strategy for unmatched cases
     */
    private getDefaultStrategy(classification: ErrorClassification): RecoveryStrategyConfig {
        if (classification.severity === ErrorSeverity.FATAL || 
            classification.severity === ErrorSeverity.CRITICAL) {
            return this.basicStrategies.get("emergency") || this.createEmergencyStrategy();
        }

        if (classification.securityRisk) {
            return this.basicStrategies.get("escalate_human") || this.createEscalateHumanStrategy();
        }

        return this.basicStrategies.get("retry") || this.createRetryStrategy();
    }

    /**
     * Initialize basic strategy lookup table (data-driven)
     */
    private initializeBasicStrategies(): void {
        // Critical errors
        this.basicStrategies.set("CRITICAL:SECURITY", this.createEscalateHumanStrategy());
        this.basicStrategies.set("FATAL:SECURITY", this.createEmergencyStrategy());
        
        // Transient errors
        this.basicStrategies.set("WARNING:TRANSIENT", this.createRetryStrategy());
        this.basicStrategies.set("ERROR:TRANSIENT", this.createWaitRetryStrategy());
        
        // Resource errors  
        this.basicStrategies.set("WARNING:RESOURCE", this.createWaitRetryStrategy());
        this.basicStrategies.set("ERROR:RESOURCE", this.createReduceScopeStrategy());
        
        // Logic errors
        this.basicStrategies.set("ERROR:LOGIC", this.createFallbackStrategy());
        this.basicStrategies.set("WARNING:LOGIC", this.createRetryModifiedStrategy());

        // Fallbacks
        this.basicStrategies.set("emergency", this.createEmergencyStrategy());
        this.basicStrategies.set("escalate_human", this.createEscalateHumanStrategy());
        this.basicStrategies.set("retry", this.createRetryStrategy());

        this.logger.info("[SimpleRecoveryProvider] Basic strategies initialized", {
            strategyCount: this.basicStrategies.size,
        });
    }

    /**
     * Basic strategy factory methods
     */
    private createEmergencyStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.EMERGENCY_STOP,
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
            conditions: [],
            estimatedSuccessRate: 1.0,
            resourceRequirements: {},
        };
    }

    private createEscalateHumanStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.ESCALATE_TO_HUMAN,
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
            conditions: [],
            estimatedSuccessRate: 0.95,
            resourceRequirements: { human_attention: 1 },
        };
    }

    private createRetryStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.RETRY_SAME,
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
            conditions: [],
            estimatedSuccessRate: 0.7,
            resourceRequirements: { retry_budget: 1 },
        };
    }

    private createWaitRetryStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.WAIT_AND_RETRY,
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
            conditions: [],
            estimatedSuccessRate: 0.75,
            resourceRequirements: { wait_time: 30 },
        };
    }

    private createReduceScopeStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.REDUCE_SCOPE,
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
            priority: 4,
            timeoutMs: 15000,
            conditions: [],
            estimatedSuccessRate: 0.8,
            resourceRequirements: { scope_adjustment: 1 },
        };
    }

    private createFallbackStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.FALLBACK_STRATEGY,
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
            priority: 5,
            timeoutMs: 20000,
            conditions: [],
            estimatedSuccessRate: 0.6,
            resourceRequirements: { alternate_strategy: 1 },
        };
    }

    private createRetryModifiedStrategy(): RecoveryStrategyConfig {
        return {
            strategyType: RecoveryStrategy.RETRY_MODIFIED,
            maxAttempts: 2,
            backoffStrategy: {
                type: BackoffType.LINEAR,
                initialDelayMs: 500,
                maxDelayMs: 10000,
                multiplier: 1.5,
                jitterPercent: 0.1,
                adaptiveAdjustment: true,
            },
            fallbackActions: [],
            priority: 6,
            timeoutMs: 45000,
            conditions: [],
            estimatedSuccessRate: 0.65,
            resourceRequirements: { modification_effort: 1 },
        };
    }
}