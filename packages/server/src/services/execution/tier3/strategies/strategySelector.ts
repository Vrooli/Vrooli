import { type Logger } from "winston";
import {
    type ExecutionStrategy,
    type ExecutionContext,
    type StrategyExecutionResult,
    StrategyType,
} from "@vrooli/shared";
import { StrategyFactory } from "./strategyFactory.js";

/**
 * StrategySelector - High-level interface for strategy selection and execution
 * 
 * This selector provides:
 * - Automatic strategy selection based on context
 * - Fallback mechanisms for failed strategies
 * - Performance-based strategy optimization
 * - A/B testing capabilities for strategy improvement
 */
export class StrategySelector {
    private readonly logger: Logger;
    private readonly factory: StrategyFactory;
    private readonly fallbackOrder: StrategyType[] = [
        StrategyType.DETERMINISTIC,
        StrategyType.CONVERSATIONAL,
        StrategyType.REASONING,
    ];

    constructor(logger: Logger) {
        this.logger = logger;
        this.factory = new StrategyFactory(logger);
    }

    /**
     * Execute a step with automatic strategy selection
     */
    async execute(context: ExecutionContext): Promise<StrategyExecutionResult> {
        const startTime = Date.now();
        
        this.logger.info("[StrategySelector] Starting execution", {
            stepId: context.stepId,
            stepType: context.stepType,
        });
        
        // Select primary strategy
        const primaryStrategy = this.factory.selectStrategy(context);
        
        if (!primaryStrategy) {
            return this.createErrorResult(
                "No suitable strategy found",
                context,
                Date.now() - startTime,
            );
        }
        
        // Try primary strategy
        try {
            const result = await this.executeWithStrategy(primaryStrategy, context);
            
            if (result.success) {
                this.logger.info("[StrategySelector] Execution successful", {
                    stepId: context.stepId,
                    strategy: primaryStrategy.type,
                    executionTime: result.metadata.executionTime,
                });
                
                // Success - learning now happens through event emission
                
                return result;
            }
            
            // Strategy executed but failed
            this.logger.warn("[StrategySelector] Primary strategy failed", {
                stepId: context.stepId,
                strategy: primaryStrategy.type,
                error: result.error,
            });
            
            // Failure - learning now happens through event emission
            
        } catch (error) {
            this.logger.error("[StrategySelector] Primary strategy threw error", {
                stepId: context.stepId,
                strategy: primaryStrategy.type,
                error: error instanceof Error ? error.message : String(error),
            });
        }
        
        // Try fallback strategies
        return this.executeFallbacks(context, primaryStrategy.type, startTime);
    }

    /**
     * Execute with a specific strategy
     */
    private async executeWithStrategy(
        strategy: ExecutionStrategy,
        context: ExecutionContext,
    ): Promise<StrategyExecutionResult> {
        this.logger.debug("[StrategySelector] Executing with strategy", {
            stepId: context.stepId,
            strategy: strategy.type,
        });
        
        try {
            return await strategy.execute(context);
        } catch (error) {
            // Return error result instead of throwing
            return {
                success: false,
                error: error instanceof Error ? error.message : String(error),
                metadata: {
                    strategyType: strategy.type,
                    executionTime: 0,
                    resourceUsage: {},
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
     * Execute fallback strategies
     */
    private async executeFallbacks(
        context: ExecutionContext,
        failedStrategy: StrategyType,
        startTime: number,
    ): Promise<StrategyExecutionResult> {
        this.logger.info("[StrategySelector] Attempting fallback strategies", {
            stepId: context.stepId,
            failedStrategy,
        });
        
        // Filter out the failed strategy and any disallowed strategies
        const availableFallbacks = this.fallbackOrder.filter(type => {
            if (type === failedStrategy) return false;
            if (context.constraints?.allowedStrategies) {
                return context.constraints.allowedStrategies.includes(type);
            }
            return true;
        });
        
        for (const strategyType of availableFallbacks) {
            const strategy = this.factory.getStrategy(strategyType);
            
            if (!strategy) continue;
            
            // Check if strategy can handle this step
            if (!strategy.canHandle(context.stepType, context.config)) {
                continue;
            }
            
            this.logger.debug("[StrategySelector] Trying fallback strategy", {
                stepId: context.stepId,
                strategy: strategyType,
            });
            
            try {
                const result = await this.executeWithStrategy(strategy, context);
                
                if (result.success) {
                    // Update metadata to indicate fallback was used
                    result.metadata.fallbackUsed = true;
                    
                    // Fallback success - learning now happens through event emission
                    
                    this.logger.info("[StrategySelector] Fallback strategy successful", {
                        stepId: context.stepId,
                        strategy: strategyType,
                        executionTime: Date.now() - startTime,
                    });
                    
                    return result;
                }
                
                // Fallback failure - learning now happens through event emission
                
            } catch (error) {
                this.logger.error("[StrategySelector] Fallback strategy error", {
                    stepId: context.stepId,
                    strategy: strategyType,
                    error: error instanceof Error ? error.message : String(error),
                });
            }
        }
        
        // All strategies failed
        return this.createErrorResult(
            "All strategies failed",
            context,
            Date.now() - startTime,
        );
    }

    /**
     * Create an error result
     */
    private createErrorResult(
        error: string,
        context: ExecutionContext,
        executionTime: number,
    ): StrategyExecutionResult {
        return {
            success: false,
            error,
            metadata: {
                strategyType: StrategyType.DETERMINISTIC, // Default for errors
                executionTime,
                resourceUsage: {
                    computeTime: executionTime,
                },
                confidence: 0,
                fallbackUsed: true,
            },
            feedback: {
                outcome: "failure",
                performanceScore: 0,
                issues: [error],
                improvements: [
                    "Review step configuration",
                    "Check input data validity",
                    "Consider adding more specific strategy hints",
                ],
            },
        };
    }

    /**
     * Get selector statistics
     */
    getStatistics(): {
        factory: ReturnType<StrategyFactory["getStatistics"]>;
        fallbackOrder: StrategyType[];
    } {
        return {
            factory: this.factory.getStatistics(),
            fallbackOrder: this.fallbackOrder,
        };
    }

    /**
     * Suggest strategy for a given context (without executing)
     */
    suggestStrategy(context: ExecutionContext): {
        recommended: StrategyType | null;
        alternatives: StrategyType[];
        reasoning: string;
    } {
        const primaryStrategy = this.factory.selectStrategy(context);
        
        if (!primaryStrategy) {
            return {
                recommended: null,
                alternatives: [],
                reasoning: "No suitable strategy found for the given context",
            };
        }
        
        // Find alternatives
        const alternatives: StrategyType[] = [];
        for (const type of this.fallbackOrder) {
            if (type === primaryStrategy.type) continue;
            
            const strategy = this.factory.getStrategy(type);
            if (strategy && strategy.canHandle(context.stepType, context.config)) {
                alternatives.push(type);
            }
        }
        
        // Generate reasoning
        const reasoning = this.generateStrategyReasoning(primaryStrategy.type, context);
        
        return {
            recommended: primaryStrategy.type,
            alternatives,
            reasoning,
        };
    }

    /**
     * Generate reasoning for strategy selection
     */
    private generateStrategyReasoning(type: StrategyType, context: ExecutionContext): string {
        const reasons: string[] = [];
        
        switch (type) {
            case StrategyType.DETERMINISTIC:
                reasons.push("Selected for predictable, well-defined task execution");
                if (!context.resources?.models || context.resources.models.length === 0) {
                    reasons.push("No LLM resources required");
                }
                if (context.config.expectedOutputs) {
                    reasons.push("Clear output expectations defined");
                }
                break;
                
            case StrategyType.CONVERSATIONAL:
                reasons.push("Selected for interactive, conversation-based execution");
                if (context.inputs.message || context.inputs.prompt) {
                    reasons.push("Natural language input detected");
                }
                if (context.resources?.models && context.resources.models.length > 0) {
                    reasons.push("LLM resources available");
                }
                break;
                
            case StrategyType.REASONING:
                reasons.push("Selected for complex analytical reasoning");
                if (Object.keys(context.inputs).length > 3) {
                    reasons.push("Multiple inputs require analysis");
                }
                if (context.constraints?.requiredConfidence && context.constraints.requiredConfidence > 0.8) {
                    reasons.push("High confidence requirement");
                }
                break;
        }
        
        return reasons.join("; ");
    }

    /**
     * Reset selector state
     */
    reset(): void {
        this.factory.reset();
        this.logger.info("[StrategySelector] Selector reset completed");
    }
}