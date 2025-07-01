import {
    type ExecutionContext,
    type ExecutionStrategy,
    StrategyType,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import { type ToolOrchestrator } from "../engine/toolOrchestrator.js";
import { type ValidationEngine } from "../engine/validationEngine.js";
import { ConversationalStrategy } from "./conversationalStrategy.js";
import { DeterministicStrategy } from "./deterministicStrategy.js";
import { ReasoningStrategy } from "./reasoningStrategy.js";

/**
 * StrategyFactory - Creates and manages execution strategies
 * 
 * This factory provides:
 * - Strategy instantiation based on type
 * - Strategy selection based on context
 * - Strategy lifecycle management
 * 
 * Performance tracking emerges from monitoring agents observing execution events
 */
export class StrategyFactory {
    private readonly strategies: Map<StrategyType, ExecutionStrategy>;
    private toolOrchestrator?: ToolOrchestrator;
    private validationEngine?: ValidationEngine;

    constructor(toolOrchestrator?: ToolOrchestrator, validationEngine?: ValidationEngine) {
        this.strategies = new Map();
        this.toolOrchestrator = toolOrchestrator;
        this.validationEngine = validationEngine;

        // Initialize all available strategies
        this.initializeStrategies();
    }

    /**
     * Set the shared services for dependency injection
     */
    setSharedServices(toolOrchestrator: ToolOrchestrator, validationEngine: ValidationEngine): void {
        this.toolOrchestrator = toolOrchestrator;
        this.validationEngine = validationEngine;

        // Update existing strategies with shared services
        for (const strategy of this.strategies.values()) {
            if ("setToolOrchestrator" in strategy && typeof strategy.setToolOrchestrator === "function") {
                strategy.setToolOrchestrator(toolOrchestrator);
            }
            if ("setValidationEngine" in strategy && typeof strategy.setValidationEngine === "function") {
                strategy.setValidationEngine(validationEngine);
            }
        }
    }

    /**
     * Initialize all available strategies
     */
    private initializeStrategies(): void {
        logger.info("[StrategyFactory] Initializing execution strategies");

        // Create strategy instances with optional shared services
        const conversationalStrategy = new ConversationalStrategy(this.toolOrchestrator, this.validationEngine);
        const deterministicStrategy = new DeterministicStrategy(this.toolOrchestrator, this.validationEngine);
        const reasoningStrategy = new ReasoningStrategy(this.toolOrchestrator, this.validationEngine);

        this.registerStrategy(conversationalStrategy);
        this.registerStrategy(deterministicStrategy);
        this.registerStrategy(reasoningStrategy);

        logger.info("[StrategyFactory] Strategies initialized", {
            available: Array.from(this.strategies.keys()),
        });
    }

    /**
     * Register a strategy
     */
    private registerStrategy(strategy: ExecutionStrategy): void {
        this.strategies.set(strategy.type, strategy);

        logger.debug("[StrategyFactory] Strategy registered", {
            type: strategy.type,
            name: strategy.name,
            version: strategy.version,
        });
    }

    /**
     * Get a strategy by type
     */
    getStrategy(type: StrategyType): ExecutionStrategy | undefined {
        const strategy = this.strategies.get(type);

        return strategy;
    }

    /**
     * Select the best strategy for a given context
     */
    selectStrategy(context: ExecutionContext): ExecutionStrategy | null {
        logger.debug("[StrategyFactory] Selecting strategy", {
            stepId: context.stepId,
            stepType: context.stepType,
        });

        // Check if context specifies a strategy
        const requestedStrategy = context.config.strategy as string;
        if (requestedStrategy) {
            const strategyType = this.parseStrategyType(requestedStrategy);
            if (strategyType) {
                const strategy = this.getStrategy(strategyType);
                if (strategy) {
                    logger.debug("[StrategyFactory] Using requested strategy", {
                        type: strategyType,
                    });
                    return strategy;
                }
            }
        }

        // Check constraints for allowed strategies
        if (context.constraints?.allowedStrategies) {
            const allowed = context.constraints.allowedStrategies;
            const availableStrategies = Array.from(this.strategies.values())
                .filter(s => allowed.includes(s.type));

            // Find first strategy that can handle the step
            for (const strategy of availableStrategies) {
                if (strategy.canHandle(context.stepType, context.config)) {
                    logger.debug("[StrategyFactory] Selected from allowed strategies", {
                        type: strategy.type,
                    });
                    return strategy;
                }
            }
        }

        // Score each strategy for the context
        const scores = this.scoreStrategies(context);

        // Select highest scoring strategy
        const bestStrategy = scores.reduce((best, current) =>
            current.score > best.score ? current : best,
        );

        if (bestStrategy.score > 0) {
            const strategy = this.getStrategy(bestStrategy.type);

            logger.info("[StrategyFactory] Selected strategy", {
                type: bestStrategy.type,
                score: bestStrategy.score,
                stepType: context.stepType,
            });

            return strategy || null;
        }

        logger.warn("[StrategyFactory] No suitable strategy found", {
            stepType: context.stepType,
        });

        return null;
    }

    /**
     * Score strategies for a given context
     */
    private scoreStrategies(context: ExecutionContext): Array<{
        type: StrategyType;
        score: number;
    }> {
        const scores: Array<{ type: StrategyType; score: number }> = [];

        for (const [type, strategy] of this.strategies) {
            let score = 0;

            // Base score: can the strategy handle this step?
            if (strategy.canHandle(context.stepType, context.config)) {
                score += 1.0;
            } else {
                // Skip strategies that can't handle the step
                continue;
            }

            // Resource efficiency score
            const estimatedResources = strategy.estimateResources(context);
            if (context.constraints?.maxCost && estimatedResources.cost) {
                const costEfficiency = 1 - (estimatedResources.cost / context.constraints.maxCost);
                score += Math.max(0, costEfficiency * 0.3);
            }

            if (context.constraints?.maxTime && estimatedResources.computeTime) {
                const timeEfficiency = 1 - (estimatedResources.computeTime / context.constraints.maxTime);
                score += Math.max(0, timeEfficiency * 0.3);
            }

            // Performance history score
            const performance = strategy.getPerformanceMetrics();
            if (performance.totalExecutions > 0) {
                const successRate = performance.successCount / performance.totalExecutions;
                score += successRate * 0.2;

                // Evolution score bonus
                score += performance.evolutionScore * 0.1;
            }

            // Strategy-specific bonuses
            score += this.getStrategySpecificScore(type, context);

            scores.push({ type, score });
        }

        // Sort by score descending
        scores.sort((a, b) => b.score - a.score);

        return scores;
    }

    /**
     * Get strategy-specific scoring adjustments
     */
    private getStrategySpecificScore(type: StrategyType, context: ExecutionContext): number {
        let score = 0;

        switch (type) {
            case StrategyType.DETERMINISTIC:
                // Bonus for well-defined, predictable tasks
                if (context.config.expectedOutputs && Object.keys(context.config.expectedOutputs).length > 0) {
                    score += 0.1;
                }
                // Bonus for no LLM requirement
                if (!context.resources?.models || context.resources.models.length === 0) {
                    score += 0.1;
                }
                break;

            case StrategyType.CONVERSATIONAL:
                // Bonus for interactive or chat-like tasks
                if (context.inputs.message || context.inputs.prompt) {
                    score += 0.1;
                }
                // Bonus for available LLM models
                if (context.resources?.models && context.resources.models.length > 0) {
                    score += 0.1;
                }
                break;

            case StrategyType.REASONING:
                // Bonus for complex, analytical tasks
                const inputComplexity = Object.keys(context.inputs).length;
                if (inputComplexity > 3) {
                    score += 0.1;
                }
                // Bonus for high confidence requirement
                if (context.constraints?.requiredConfidence && context.constraints.requiredConfidence > 0.8) {
                    score += 0.1;
                }
                break;
        }

        return score;
    }

    /**
     * Parse strategy type from string
     */
    private parseStrategyType(strategy: string): StrategyType | null {
        const normalized = strategy.toUpperCase();

        if (normalized in StrategyType) {
            return StrategyType[normalized as keyof typeof StrategyType];
        }

        // Handle common variations
        switch (normalized) {
            case "CHAT":
            case "CONVERSATION":
                return StrategyType.CONVERSATIONAL;
            case "REASON":
            case "ANALYZE":
                return StrategyType.REASONING;
            case "PROCESS":
            case "TRANSFORM":
                return StrategyType.DETERMINISTIC;
            default:
                return null;
        }
    }

    /**
     * Get factory statistics - minimal implementation
     * Real statistics emerge from monitoring agents
     */
    getStatistics(): {
        availableStrategies: string[];
    } {
        const availableStrategies = Array.from(this.strategies.keys());

        return {
            availableStrategies,
        };
    }

    /**
     * Reset factory state
     */
    reset(): void {
        // No state to reset in minimal implementation
        logger.info("[StrategyFactory] Factory reset completed");
    }
}
