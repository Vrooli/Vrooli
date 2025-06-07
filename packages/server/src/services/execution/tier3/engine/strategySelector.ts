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

        // Check for graduation from Conversational to Reasoning
        if (declaredStrategy === StrategyTypeEnum.CONVERSATIONAL) {
            if (this.shouldGraduateToReasoning(usageHints)) {
                return StrategyTypeEnum.REASONING;
            }
        }

        // Check for graduation from Reasoning to Deterministic
        if (declaredStrategy === StrategyTypeEnum.REASONING) {
            if (this.shouldGraduateToDeterministic(usageHints)) {
                return StrategyTypeEnum.DETERMINISTIC;
            }
        }

        return declaredStrategy;
    }

    /**
     * Checks if strategy should graduate to reasoning
     */
    private shouldGraduateToReasoning(hints: UsageHints): boolean {
        const successThreshold = 0.8;
        const frequencyThreshold = 10;

        return (
            (hints.historicalSuccessRate ?? 0) >= successThreshold &&
            (hints.executionFrequency ?? 0) >= frequencyThreshold
        );
    }

    /**
     * Checks if strategy should graduate to deterministic
     */
    private shouldGraduateToDeterministic(hints: UsageHints): boolean {
        const successThreshold = 0.95;
        const frequencyThreshold = 50;
        const complexityThreshold = 0.3; // Low complexity

        return (
            (hints.historicalSuccessRate ?? 0) >= successThreshold &&
            (hints.executionFrequency ?? 0) >= frequencyThreshold &&
            (hints.averageComplexity ?? 1) <= complexityThreshold
        );
    }

    /**
     * Checks for security violations
     */
    private hasSecurityViolation(strategy: StrategyType, context: ExecutionContext): boolean {
        // Check for sensitive data handling
        const hasSensitiveData = context.config.sensitiveData === true;
        const hasComplianceRequirement = context.config.compliance !== undefined;

        // Conversational strategy not allowed for sensitive operations
        if (strategy === StrategyTypeEnum.CONVERSATIONAL && (hasSensitiveData || hasComplianceRequirement)) {
            return true;
        }

        return false;
    }

    /**
     * Checks domain restrictions
     */
    private checkDomainRestrictions(
        strategy: StrategyType,
        restrictions: string[],
    ): StrategyType | null {
        // HIPAA compliance requires deterministic
        if (restrictions.includes("HIPAA") && strategy !== StrategyTypeEnum.DETERMINISTIC) {
            return StrategyTypeEnum.DETERMINISTIC;
        }

        // Financial regulations prefer reasoning or deterministic
        if (restrictions.includes("PCI-DSS") && strategy === StrategyTypeEnum.CONVERSATIONAL) {
            return StrategyTypeEnum.REASONING;
        }

        return null;
    }

    /**
     * Checks for resource constraints
     */
    private hasResourceConstraints(strategy: StrategyType, context: ExecutionContext): boolean {
        const constraints = context.constraints;

        // Very tight time constraints
        if (constraints.maxTime && constraints.maxTime < 5000) { // 5 seconds
            return strategy !== StrategyTypeEnum.DETERMINISTIC;
        }

        // Very tight cost constraints
        if (constraints.maxCost && constraints.maxCost < 0.01) { // 1 cent
            return strategy === StrategyTypeEnum.CONVERSATIONAL;
        }

        return false;
    }

    /**
     * Selects resource-optimal strategy
     */
    private selectResourceOptimalStrategy(
        _currentStrategy: StrategyType,
        context: ExecutionContext,
    ): StrategyType {
        const constraints = context.constraints;

        // For very tight constraints, use deterministic
        if (constraints.maxTime && constraints.maxTime < 5000) {
            return StrategyTypeEnum.DETERMINISTIC;
        }

        // For moderate constraints, use reasoning
        if (constraints.maxCost && constraints.maxCost < 0.1) {
            return StrategyTypeEnum.REASONING;
        }

        // Otherwise reasoning is a good middle ground
        return StrategyTypeEnum.REASONING;
    }

    /**
     * Validates strategy type
     */
    private isValidStrategy(strategy: string): boolean {
        return Object.values(StrategyTypeEnum).includes(strategy as StrategyType);
    }
}
