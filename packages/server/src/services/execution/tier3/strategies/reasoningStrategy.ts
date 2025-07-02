import {
    type ExecutionContext,
    type IExecutionStrategy,
    type ResourceUsage,
    type StrategyExecutionResult,
    generatePK,
    toBotId,
    toSwarmId,
    type ResponseContext,
    type SessionUser,
} from "@vrooli/shared";
import { logger } from "../../../../events/logger.js";
import type { ResponseService } from "../../../response/responseService.js";
import type { ToolOrchestrator } from "../engine/toolOrchestrator.js";
import type { ValidationEngine } from "../engine/validationEngine.js";

/**
 * REASONING STRATEGY (Single-Turn Response)
 * 
 * This strategy provides single-turn, single-bot reasoning execution using ResponseService.
 * It's designed for tasks where ONE bot should handle the entire reasoning process,
 * similar to OpenAI's o3 model - performing multiple reasoning steps, responses, and
 * tool calls within a single turn.
 * 
 * WHAT THIS DOES:
 * - Execute single-turn reasoning via ResponseService
 * - Support multi-step reasoning within one response
 * - Track resource usage for the complete reasoning turn
 * 
 * WHEN TO USE THIS:
 * - Single bot should handle the entire task
 * - Multi-step reasoning needed but within one response
 * - Tool calls and reasoning should be in one coherent flow
 * 
 * WHEN TO USE ConversationalStrategy INSTEAD:
 * - Multiple team members need to collaborate
 * - Task requires back-and-forth between different specialized bots
 * - Complex coordination between multiple AI agents
 */
export class ReasoningStrategy implements IExecutionStrategy {
    readonly type = "reasoning";
    readonly name = "ReasoningStrategy";
    readonly version = "4.0.0-response-service";

    constructor(
        private readonly responseService: ResponseService,
        private readonly toolOrchestrator?: ToolOrchestrator,
        private readonly validationEngine?: ValidationEngine,
    ) {
        logger.info("[ReasoningStrategy] Initialized with ResponseService for single-turn reasoning");
    }

    /**
     * Main execution method for IExecutionStrategy interface
     */
    async execute(context: ExecutionContext): Promise<StrategyExecutionResult> {
        return this.executeStrategy(context, context);
    }

    /**
     * Strategy-specific methods required by IExecutionStrategy
     */
    canHandle(stepType: string, config?: Record<string, unknown>): boolean {
        return this.isApplicable({ stepType } as ExecutionContext);
    }

    estimateResources(context: ExecutionContext): ResourceUsage {
        // Basic estimation - agents can improve this
        return {
            credits: 2000, // Reasoning typically uses more tokens
            tokens: 2000,
            duration: 10000, // 10 seconds
        };
    }

    getPerformanceMetrics(): any {
        // Basic metrics - agents can enhance this
        return {
            totalExecutions: 0,
            successCount: 0,
            failureCount: 0,
            averageExecutionTime: 0,
            averageResourceUsage: { credits: 0, tokens: 0, duration: 0 },
            averageConfidence: 0.7,
            evolutionScore: 0.5,
        };
    }

    /**
     * SINGLE-TURN REASONING EXECUTION
     * 
     * Uses ResponseService to perform reasoning in a single turn.
     * The bot can perform multiple reasoning steps, tool calls, and refinements
     * within this single response - similar to how o3 works.
     */
    protected async executeStrategy(
        context: ExecutionContext,
    ): Promise<StrategyExecutionResult> {
        const startTime = Date.now();
        const stepId = context.stepId || generatePK().toString();

        logger.info("[ReasoningStrategy] Starting single-turn reasoning execution", {
            stepId,
            stepType: context.stepType,
        });

        try {
            // Build ResponseContext for ResponseService
            const responseContext = this.buildResponseContext(context);

            // Execute reasoning via ResponseService
            const response = await this.responseService.generateResponse({
                context: responseContext,
                abortSignal: undefined, // Could add timeout support here
            });

            const executionTime = Date.now() - startTime;

            if (response.success) {
                // Extract outputs from the reasoning response
                const outputs = this.parseReasoningOutputs(response.message, context);

                const result: StrategyExecutionResult = {
                    success: true,
                    outputs,
                    resourceUsage: {
                        credits: parseInt(response.resourcesUsed.creditsUsed),
                        tokens: parseInt(response.resourcesUsed.creditsUsed), // Approximate
                        duration: executionTime,
                    },
                    metadata: {
                        strategy: this.type,
                        executionTime,
                        reasoning: response.message.config?.text || "",
                        toolCallsExecuted: response.toolCalls?.length || 0,
                        confidence: response.confidence,
                    },
                };

                logger.info("[ReasoningStrategy] Reasoning execution completed", {
                    stepId,
                    success: true,
                    executionTime,
                    creditsUsed: response.resourcesUsed.creditsUsed,
                    toolCalls: response.toolCalls?.length || 0,
                });

                return result;
            } else {
                throw new Error(response.error?.message || "Reasoning failed");
            }

        } catch (error) {
            const executionTime = Date.now() - startTime;
            const errorMessage = error instanceof Error ? error.message : String(error);

            logger.error("[ReasoningStrategy] Reasoning execution failed", {
                stepId,
                error: errorMessage,
                executionTime,
            });

            return {
                success: false,
                error: errorMessage,
                resourceUsage: {
                    credits: 0,
                    tokens: 0,
                    duration: executionTime,
                },
                metadata: {
                    strategy: this.type,
                    executionTime,
                },
            };
        }
    }

    /**
     * Build ResponseContext for ResponseService
     * Converts ExecutionContext to the format expected by ResponseService
     */
    private buildResponseContext(context: ExecutionContext): ResponseContext {
        // Create a reasoning-focused bot configuration
        const reasoningBotConfig = {
            id: "reasoning_bot",
            name: "Reasoning Specialist",
            model: context.config?.model || "gpt-4",
            temperature: context.config?.temperature || 0.7,
            maxTokens: context.config?.maxTokens || 2000,
            systemPrompt: this.buildReasoningSystemPrompt(context),
        };

        // Build conversation history with the reasoning task
        const conversationHistory = [
            {
                id: generatePK().toString(),
                createdAt: new Date(),
                config: {
                    role: "user" as const,
                    text: this.buildReasoningPrompt(context),
                },
            },
        ];

        return {
            swarmId: toSwarmId(context.swarmId || generatePK().toString()),
            userData: context.userData || { id: "system", name: "Reasoning System" } as SessionUser,
            timestamp: new Date(),
            botId: toBotId("reasoning_bot"),
            botConfig: reasoningBotConfig,
            conversationHistory,
            availableTools: context.resources?.tools || [],
            strategy: "reasoning",
            constraints: {
                maxTokens: context.config?.maxTokens || 2000,
                temperature: context.config?.temperature || 0.7,
                timeoutMs: context.config?.timeoutMs || 120000, // 2 minutes
            },
        };
    }

    /**
     * Build system prompt for reasoning tasks
     */
    private buildReasoningSystemPrompt(context: ExecutionContext): string {
        return `You are a reasoning specialist designed to work through complex problems step-by-step.
Your approach should be thorough, logical, and transparent in your reasoning process.
You can use available tools to gather information and validate your reasoning.
Always structure your response clearly and provide the requested outputs.`;
    }

    /**
     * Build reasoning prompt from execution context
     */
    private buildReasoningPrompt(context: ExecutionContext): string {
        const stepData = context.stepData || {};
        const inputs = stepData.inputs || {};
        const outputs = stepData.outputs || {};

        return `Please reason through this task step-by-step:

Task: ${context.stepType}
${context.config?.description ? `\nDescription: ${context.config.description}` : ""}

Inputs available:
${Object.entries(inputs).map(([key, value]) => `- ${key}: ${JSON.stringify(value)}`).join("\n")}

Expected outputs:
${Object.keys(outputs).map(key => `- ${key}`).join("\n")}

Please provide your reasoning and the required outputs in a clear, structured format.`;
    }

    /**
     * Parse reasoning outputs from ResponseService message
     */
    private parseReasoningOutputs(message: any, context: ExecutionContext): Record<string, unknown> {
        // Simple output extraction - agents can develop sophisticated parsing
        const outputs: Record<string, unknown> = {};
        const expectedOutputs = Object.keys(context.stepData?.outputs || {});
        const responseText = message.config?.text || "";

        // Basic attempt to extract outputs from response
        for (const outputKey of expectedOutputs) {
            // Simple pattern matching - agents can improve this
            const pattern = new RegExp(`${outputKey}[:\\s]*(.+?)(?=\\n|$)`, "i");
            const match = responseText.match(pattern);

            if (match && match[1]) {
                outputs[outputKey] = match[1].trim();
            } else {
                // Fallback: use the full response if no specific pattern found
                outputs[outputKey] = responseText;
            }
        }

        return outputs;
    }

    /**
     * Check if strategy is suitable for given context
     * Simple heuristic - agents can develop sophisticated selection logic
     */
    isApplicable(context: ExecutionContext): boolean {
        return context.stepType === "reasoning" ||
            context.stepType === "analysis" ||
            context.stepType === "decision";
    }
}
