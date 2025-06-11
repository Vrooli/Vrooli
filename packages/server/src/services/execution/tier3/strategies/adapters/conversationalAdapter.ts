import { type Logger } from "winston";
import {
    type ExecutionContext as Tier3ExecutionContext,
    type ExecutionContext as StrategyExecutionContext,
    type StrategyExecutionResult,
    type StrategyFeedback,
    type StrategyPerformance,
    type ResourceUsage,
    StrategyType,
} from "@vrooli/shared";
import { ConversationalStrategy } from "../conversationalStrategy.js";

/**
 * Adapter to bridge ConversationalStrategy to ExecutionStrategy interface
 * Now uses the modern ConversationalStrategy implementation
 */
export class ConversationalStrategyAdapter {
    readonly type = StrategyType.CONVERSATIONAL;
    readonly name = "ConversationalStrategy";
    readonly version = "2.0.0-adapter";

    private readonly logger: Logger;
    private readonly strategy: ConversationalStrategy;

    constructor(logger: Logger) {
        this.logger = logger;
        this.strategy = new ConversationalStrategy(logger);
    }

    /**
     * Checks if this strategy can handle the given step type
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        return this.strategy.canHandle(stepType, config);
    }

    /**
     * Executes the strategy using modern implementation
     */
    async execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult> {
        this.logger.info("[ConversationalStrategyAdapter] Executing with modern strategy", {
            stepId: context.stepId,
            stepType: context.stepType,
        });

        try {
            // Execute modern strategy directly - it already returns StrategyExecutionResult
            return await this.strategy.execute(context);

        } catch (error) {
            this.logger.error("[ConversationalStrategyAdapter] Execution failed", {
                stepId: context.stepId,
                error: error instanceof Error ? error.message : String(error),
            });

            return {
                success: false,
                error: error instanceof Error ? error.message : "Unknown error",
                metadata: {
                    strategyType: this.type,
                    executionTime: Date.now(),
                    resourceUsage: { computeTime: Date.now() },
                    confidence: 0,
                    fallbackUsed: false,
                },
                feedback: {
                    outcome: "failure",
                    performanceScore: 0,
                    issues: [error instanceof Error ? error.message : "Unknown error"],
                },
            };
        }
    }

    /**
     * Estimates resource requirements
     */
    estimateResources(context: StrategyExecutionContext): ResourceUsage {
        return this.strategy.estimateResources(context);
    }


    /**
     * Returns performance metrics
     */
    getPerformanceMetrics(): StrategyPerformance {
        return this.strategy.getPerformanceMetrics();
    }

}