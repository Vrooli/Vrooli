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
import { ConversationalStrategy as LegacyConversationalStrategy } from "../../../../../legacy-execution/backup-2025-06-05-160205/execution-attempt-old/strategies/conversationalStrategy.js";
import { type ExecutionStrategyDependencies } from "../../../../../legacy-execution/backup-2025-06-05-160205/execution-attempt-old/strategies/executionStrategy.js";

/**
 * Adapter to bridge legacy ConversationalStrategy to new ExecutionStrategy interface
 * This allows gradual migration while maintaining compatibility
 */
export class ConversationalStrategyAdapter {
    readonly type = StrategyType.CONVERSATIONAL;
    readonly name = "ConversationalStrategy";
    readonly version = "1.0.0-adapter";

    private readonly logger: Logger;
    private readonly legacyStrategy: LegacyConversationalStrategy;

    constructor(logger: Logger) {
        this.logger = logger;
        this.legacyStrategy = new LegacyConversationalStrategy();
    }

    /**
     * Checks if this strategy can handle the given step type
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        return this.legacyStrategy.canHandle(stepType, config);
    }

    /**
     * Executes the strategy using legacy implementation
     */
    async execute(context: StrategyExecutionContext): Promise<StrategyExecutionResult> {
        const startTime = Date.now();
        
        this.logger.info("[ConversationalStrategyAdapter] Executing with legacy strategy", {
            stepId: context.stepId,
            stepType: context.stepType,
        });

        try {
            // Convert new context to legacy format
            const legacyContext = this.adaptContext(context);
            
            // Execute legacy strategy
            const legacyResult = await this.legacyStrategy.execute(legacyContext);
            
            // Convert legacy result to new format
            return this.adaptResult(legacyResult, Date.now() - startTime);

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
                    executionTime: Date.now() - startTime,
                    resourceUsage: { computeTime: Date.now() - startTime },
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
        // Based on legacy ConversationalStrategy patterns
        const inputComplexity = Object.keys(context.inputs).length;
        const baseTokens = 1000;
        const contextMultiplier = inputComplexity * 200;
        
        return {
            tokens: baseTokens + contextMultiplier,
            apiCalls: 1,
            computeTime: 15000, // 15 seconds for conversational tasks
            cost: 0.03, // Estimated cost based on token usage
        };
    }


    /**
     * Returns performance metrics
     */
    getPerformanceMetrics(): StrategyPerformance {
        // Return placeholder metrics for adapter
        return {
            totalExecutions: 0,
            successCount: 0,
            failureCount: 0,
            averageExecutionTime: 0,
            averageResourceUsage: {},
            averageConfidence: 0.7, // Default for conversational
            evolutionScore: 0,
        };
    }

    /**
     * Converts new ExecutionContext to legacy format
     */
    private adaptContext(context: StrategyExecutionContext): ExecutionStrategyDependencies {
        // Build legacy context structure
        const legacyContext = {
            subroutineInstanceId: context.stepId,
            routine: {
                // Map basic routine data
                id: context.stepId,
                name: context.config.name as string || "Conversational Task",
                description: context.config.description as string || "Conversational execution",
                instructions: context.config.instructions as string || "",
            },
            ioMapping: {
                inputs: this.adaptInputs(context.inputs),
                outputs: this.adaptOutputs(context.config.expectedOutputs as Record<string, unknown> || {}),
            },
            limits: {
                maxCredits: BigInt(context.resources?.credits || 10000),
                maxToolCalls: 10,
                maxTimeMs: context.constraints?.maxTime || 300000, // 5 minutes
            },
            userData: {
                id: context.metadata?.userId || "unknown",
                languages: ["en"],
            },
        };

        // Create mock dependencies for legacy strategy
        const reasoningEngine = this.createMockReasoningEngine();
        const messageStore = this.createMockMessageStore();
        
        return {
            context: legacyContext,
            reasoningEngine,
            messageStore,
            logger: this.logger,
            availableTools: context.resources?.tools || [],
            botParticipant: this.createMockBotParticipant(),
            abortSignal: new AbortController().signal,
        };
    }

    /**
     * Adapts inputs to legacy format
     */
    private adaptInputs(inputs: Record<string, unknown>): Record<string, { value: unknown; name?: string }> {
        const adapted: Record<string, { value: unknown; name?: string }> = {};
        
        for (const [key, value] of Object.entries(inputs)) {
            adapted[key] = {
                value,
                name: key,
            };
        }
        
        return adapted;
    }

    /**
     * Adapts outputs to legacy format
     */
    private adaptOutputs(outputs: Record<string, unknown>): Record<string, { value?: unknown; name?: string; description?: string }> {
        const adapted: Record<string, { value?: unknown; name?: string; description?: string }> = {};
        
        for (const [key, value] of Object.entries(outputs)) {
            adapted[key] = {
                name: key,
                description: typeof value === "string" ? value : undefined,
            };
        }
        
        return adapted;
    }

    /**
     * Converts legacy result to new format
     */
    private adaptResult(legacyResult: any, executionTime: number): StrategyExecutionResult {
        const resourceUsage: ResourceUsage = {
            tokens: legacyResult.toolCallsCount || 0,
            apiCalls: 1,
            computeTime: executionTime,
            cost: Number(legacyResult.creditsUsed || 0) * 0.00001, // Convert credits to cost
        };

        if (legacyResult.success) {
            return {
                success: true,
                result: this.extractOutputs(legacyResult),
                metadata: {
                    strategyType: this.type,
                    executionTime,
                    resourceUsage,
                    confidence: 0.8, // Default confidence for legacy
                    fallbackUsed: false,
                },
                feedback: {
                    outcome: "success",
                    performanceScore: this.calculatePerformanceScore(legacyResult),
                    improvements: this.extractImprovements(legacyResult),
                },
            };
        } else {
            return {
                success: false,
                error: legacyResult.error?.message || "Legacy execution failed",
                metadata: {
                    strategyType: this.type,
                    executionTime,
                    resourceUsage,
                    confidence: 0,
                    fallbackUsed: false,
                },
                feedback: {
                    outcome: "failure",
                    performanceScore: 0,
                    issues: [legacyResult.error?.message || "Unknown error"],
                },
            };
        }
    }

    /**
     * Extracts outputs from legacy result
     */
    private extractOutputs(legacyResult: any): Record<string, unknown> {
        const outputs: Record<string, unknown> = {};
        
        if (legacyResult.ioMapping?.outputs) {
            for (const [key, output] of Object.entries(legacyResult.ioMapping.outputs as Record<string, any>)) {
                outputs[key] = output.value;
            }
        }
        
        // Include conversation messages if available
        if (legacyResult.messages) {
            outputs.conversationHistory = legacyResult.messages;
        }
        
        // Fallback to metadata if no specific outputs
        if (Object.keys(outputs).length === 0 && legacyResult.metadata) {
            outputs.result = legacyResult.metadata;
        }
        
        return outputs;
    }

    /**
     * Calculates performance score from legacy result
     */
    private calculatePerformanceScore(legacyResult: any): number {
        let score = 0.7; // Base score for completion
        
        // Bonus for efficiency (fewer tool calls)
        if (legacyResult.toolCallsCount < 5) {
            score += 0.1;
        }
        
        // Bonus for speed (under metrics average)
        if (legacyResult.metadata?.metrics?.averageResponseTime < 10000) {
            score += 0.1;
        }
        
        // Bonus for conversation turns (indicates engagement)
        if (legacyResult.metadata?.metrics?.conversationTurns > 1) {
            score += 0.1;
        }
        
        return Math.min(1, score);
    }

    /**
     * Extracts improvement suggestions from legacy result
     */
    private extractImprovements(legacyResult: any): string[] {
        const improvements: string[] = [];
        
        if (legacyResult.timeElapsed > 30000) {
            improvements.push("Consider optimizing for faster response times");
        }
        
        if (legacyResult.toolCallsCount > 10) {
            improvements.push("High tool usage - consider more efficient strategies");
        }
        
        if (legacyResult.metadata?.metrics?.conversationTurns === 1) {
            improvements.push("Single turn conversation - consider multi-turn engagement");
        }
        
        return improvements;
    }

    /**
     * Creates mock reasoning engine for legacy compatibility
     */
    private createMockReasoningEngine(): any {
        return {
            async runLoop(...args: any[]): Promise<any> {
                // Simplified mock - returns basic message structure
                return {
                    finalMessage: {
                        id: "mock-message",
                        text: "This is a mock response from the adapter.",
                        config: { role: "assistant" },
                    },
                    responseStats: {
                        creditsUsed: BigInt(100),
                        toolCalls: 0,
                    },
                };
            },
        };
    }

    /**
     * Creates mock message store for legacy compatibility
     */
    private createMockMessageStore(): any {
        return {
            async addMessage(conversationId: string, message: any): Promise<void> {
                this.logger.debug("[MockMessageStore] Adding message", {
                    conversationId,
                    messageId: message.id,
                });
            },
        };
    }

    /**
     * Creates mock bot participant for legacy compatibility
     */
    private createMockBotParticipant(): any {
        return {
            config: {
                model: "gpt-4o-mini",
                temperature: 0.7,
            },
        };
    }
}