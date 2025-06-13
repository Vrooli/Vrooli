import { type Logger } from "winston";
import {
    type ExecutionContext,
    type ExecutionStrategy,
    type StrategyType,
    type StrategyFactoryConfig,
    StrategyType as StrategyTypeEnum,
} from "@vrooli/shared";
import { ConversationalStrategy } from "../strategies/conversationalStrategy.js";
import { ReasoningStrategy } from "../strategies/reasoningStrategy.js";
import { DeterministicStrategy } from "../strategies/deterministicStrategy.js";
import { ToolOrchestrator } from "./toolOrchestrator.js";
import { ValidationEngine } from "./validationEngine.js";

/**
 * Usage hints provided by Tier 2 for strategy selection
 */
export interface UsageHints {
    historicalSuccessRate?: number;
    executionFrequency?: number;
    averageComplexity?: number;
    userPreference?: StrategyType;
    domainRestrictions?: string[];
}

/**
 * StrategySelector - Intelligent strategy selection for step execution
 * 
 * This component implements the two-layer rule set for strategy resolution:
 * 1. Declarative default - From routine manifest
 * 2. Adaptive override - Based on runtime context and policies
 * 
 * Strategy selection follows the evolution pipeline:
 * - New tasks start with Conversational (maximum flexibility)
 * - Proven patterns graduate to Reasoning (structured analysis)
 * - High-volume stable processes achieve Deterministic (optimal automation)
 */
export class StrategySelector {
    private readonly strategies: Map<StrategyType, ExecutionStrategy>;
    private readonly config: StrategyFactoryConfig;
    private readonly logger: Logger;
    private readonly toolOrchestrator?: ToolOrchestrator;
    private readonly validationEngine?: ValidationEngine;

    constructor(
        config: StrategyFactoryConfig, 
        logger: Logger,
        toolOrchestrator?: ToolOrchestrator,
        validationEngine?: ValidationEngine,
    ) {
        this.config = config;
        this.logger = logger;
        this.toolOrchestrator = toolOrchestrator;
        this.validationEngine = validationEngine;
        this.strategies = new Map();

        // Initialize available strategies
        this.initializeStrategies();
    }

    /**
     * Selects the optimal execution strategy for a step
     * 
     * Resolution layers:
     * 1. Check manifest declaration (author's intent)
     * 2. Apply policy overrides (security, compliance)
     * 3. Consider performance hints (historical data)
     * 4. Apply fallback chain if needed
     */
    async selectStrategy(
        context: ExecutionContext,
        usageHints?: UsageHints,
    ): Promise<ExecutionStrategy> {
        const stepId = context.stepId;

        // 1. Get declared strategy from manifest
        const declaredStrategy = this.getDeclaredStrategy(context);
        this.logger.debug(`[StrategySelector] Declared strategy: ${declaredStrategy}`, { stepId });

        // 2. Check for policy violations
        const policyOverride = await this.checkPolicyOverride(
            declaredStrategy,
            context,
            usageHints,
        );

        if (policyOverride) {
            this.logger.info("[StrategySelector] Policy override applied", {
                stepId,
                declared: declaredStrategy,
                override: policyOverride,
            });
            return this.getOrCreateStrategy(policyOverride);
        }

        // 3. Check for adaptive optimization
        if (this.config.adaptationEnabled) {
            const optimizedStrategy = await this.checkAdaptiveOptimization(
                declaredStrategy,
                context,
                usageHints,
            );

            if (optimizedStrategy !== declaredStrategy) {
                this.logger.info("[StrategySelector] Adaptive optimization applied", {
                    stepId,
                    declared: declaredStrategy,
                    optimized: optimizedStrategy,
                });
                return this.getOrCreateStrategy(optimizedStrategy);
            }
        }

        // 4. Use declared strategy
        return this.getOrCreateStrategy(declaredStrategy);
    }

    /**
     * Initializes available strategies
     */
    private initializeStrategies(): void {
        // Strategies are created on-demand to reduce memory usage
        this.logger.debug("[StrategySelector] Strategy factory initialized");
    }

    /**
     * Gets or creates a strategy instance
     */
    private getOrCreateStrategy(type: StrategyType): ExecutionStrategy {
        let strategy = this.strategies.get(type);
        
        if (!strategy) {
            strategy = this.createStrategy(type);
            this.strategies.set(type, strategy);
        }

        return strategy;
    }

    /**
     * Creates a strategy instance
     */
    private createStrategy(type: StrategyType): ExecutionStrategy {
        switch (type) {
            case StrategyTypeEnum.CONVERSATIONAL:
                return new ConversationalStrategy(this.logger, this.toolOrchestrator, this.validationEngine);
            
            case StrategyTypeEnum.REASONING:
                return new ReasoningStrategy(this.logger);
            
            case StrategyTypeEnum.DETERMINISTIC:
                return new DeterministicStrategy(this.logger, this.toolOrchestrator, this.validationEngine);
            
            default:
                this.logger.warn(`[StrategySelector] Unknown strategy type: ${type}, falling back to conversational`);
                return new ConversationalStrategy(this.logger, this.toolOrchestrator, this.validationEngine);
        }
    }

    /**
     * Gets declared strategy from context
     */
    private getDeclaredStrategy(context: ExecutionContext): StrategyType {
        // Check explicit strategy in config
        const configStrategy = context.config.strategy as string;
        if (configStrategy && this.isValidStrategy(configStrategy)) {
            return configStrategy as StrategyType;
        }

        // Check step type hints
        const stepType = context.stepType.toLowerCase();
        if (stepType.includes("deterministic") || stepType.includes("automated")) {
            return StrategyTypeEnum.DETERMINISTIC;
        }
        if (stepType.includes("reasoning") || stepType.includes("analysis")) {
            return StrategyTypeEnum.REASONING;
        }

        // Default to configured default
        return this.config.defaultStrategy;
    }

    /**
     * Checks for policy-based strategy override
     */
    private async checkPolicyOverride(
        declaredStrategy: StrategyType,
        context: ExecutionContext,
        usageHints?: UsageHints,
    ): Promise<StrategyType | null> {
        // Security violations
        if (this.hasSecurityViolation(declaredStrategy, context)) {
            this.logger.warn("[StrategySelector] Security violation detected", {
                stepId: context.stepId,
                strategy: declaredStrategy,
            });
            return StrategyTypeEnum.DETERMINISTIC; // Force deterministic for security
        }

        // Domain restrictions
        if (usageHints?.domainRestrictions) {
            const violation = this.checkDomainRestrictions(
                declaredStrategy,
                usageHints.domainRestrictions,
            );
            if (violation) {
                return violation;
            }
        }

        // Resource constraints
        if (this.hasResourceConstraints(declaredStrategy, context)) {
            return this.selectResourceOptimalStrategy(declaredStrategy, context);
        }

        return null;
    }

    /**
     * Checks for adaptive optimization opportunities
     */
    private async checkAdaptiveOptimization(
        declaredStrategy: StrategyType,
        context: ExecutionContext,
        usageHints?: UsageHints,
    ): Promise<StrategyType> {
        if (!usageHints) {
            return declaredStrategy;
        }

        // Evolution pipeline: Conversational -> Reasoning -> Deterministic
        // Let optimization agents determine graduation through event analysis

        // Check for graduation from Conversational to Reasoning
        if (declaredStrategy === StrategyTypeEnum.CONVERSATIONAL) {
            this.shouldGraduateToReasoning(usageHints); // Emits event only
        }

        // Check for graduation from Reasoning to Deterministic
        if (declaredStrategy === StrategyTypeEnum.REASONING) {
            this.shouldGraduateToDeterministic(usageHints); // Emits event only
        }

        return declaredStrategy;
    }

    /**
     * Emit strategy graduation consideration event for optimization agents
     */
    private emitGraduationEvent(targetStrategy: string, hints: UsageHints): void {
        this.logger.debug("Strategy graduation consideration", {
            type: "execution.strategy.graduation_considered",
            targetStrategy,
            hints,
            timestamp: new Date(),
        });
    }

    /**
     * Basic strategy graduation checks
     * Note: Optimization agents should determine sophisticated thresholds through events
     */
    private shouldGraduateToReasoning(hints: UsageHints): boolean {
        // Always stay with declared strategy - optimization agents can override via events
        // Emit graduation consideration event for agent analysis
        this.emitGraduationEvent('reasoning', hints);
        return false; // Let agents decide graduation through events
    }

    /**
     * Basic strategy graduation checks  
     * Note: Optimization agents should determine sophisticated thresholds through events
     */
    private shouldGraduateToDeterministic(hints: UsageHints): boolean {
        // Always stay with declared strategy - optimization agents can override via events
        // Emit graduation consideration event for agent analysis
        this.emitGraduationEvent('deterministic', hints);
        return false; // Let agents decide graduation through events
    }

    /**
     * Basic security checks
     * Note: Complex security policies should be handled by security agents
     */
    private hasSecurityViolation(strategy: StrategyType, context: ExecutionContext): boolean {
        // Minimal check - let security agents handle complex policies
        const hasSensitiveData = context.config.sensitiveData === true;
        
        // Basic rule: no conversational for sensitive data
        return strategy === StrategyTypeEnum.CONVERSATIONAL && hasSensitiveData;
    }

    /**
     * Basic domain restriction checks
     * Note: Complex compliance policies should be handled by security agents
     */
    private checkDomainRestrictions(
        strategy: StrategyType,
        restrictions: string[],
    ): StrategyType | null {
        // Minimal compliance check - let security agents handle complex policies
        if (restrictions.length > 0 && strategy === StrategyTypeEnum.CONVERSATIONAL) {
            return StrategyTypeEnum.REASONING; // Basic safe fallback
        }

        return null;
    }

    /**
     * Basic resource constraint checks
     * Note: Complex resource optimization should be handled by optimization agents
     */
    private hasResourceConstraints(strategy: StrategyType, context: ExecutionContext): boolean {
        const constraints = context.constraints;
        
        // Minimal check - let optimization agents handle complex resource logic
        return (constraints.maxTime !== undefined && constraints.maxTime < 1000) ||
               (constraints.maxCost !== undefined && constraints.maxCost < 0.001);
    }

    /**
     * Basic resource-based strategy selection
     * Note: Complex resource optimization should be handled by optimization agents
     */
    private selectResourceOptimalStrategy(
        _currentStrategy: StrategyType,
        context: ExecutionContext,
    ): StrategyType {
        const constraints = context.constraints;

        // Minimal logic - let optimization agents handle sophisticated resource allocation
        if (constraints.maxTime && constraints.maxTime < 1000) {
            return StrategyTypeEnum.DETERMINISTIC; // Fastest option
        }

        return StrategyTypeEnum.REASONING; // Safe middle ground
    }

    /**
     * Validates strategy type
     */
    private isValidStrategy(strategy: string): boolean {
        return Object.values(StrategyTypeEnum).includes(strategy as StrategyType);
    }
}
