/**
 * ResponseService - Clean Individual Bot Response Generation
 * 
 * This service extracts the core response generation logic from ReasoningEngine.runLoop()
 * and provides a clean interface using unified types. It handles:
 * - LLM API calls
 * - Tool execution
 * - Resource tracking
 * - Error handling
 * 
 * Design Principles:
 * - Clean input/output types (ResponseContext → ResponseResult)
 * - No transformation logic (direct delegation to existing services)
 * - Reuse existing LlmRouter, ToolRunner, ContextBuilder
 * - Maintain compatibility with existing conversation system
 */

import type {
    BotId,
    ResponseContext,
    ResponseResult,
} from "@vrooli/shared";
import { generatePK, type ChatMessage, type ExecutionError, type ToolCall } from "@vrooli/shared";
import { logger } from "../../events/logger.js";
import { ContextBuilder } from "./contextBuilder.js";
import type { LlmRouter } from "./router.js";
import type { ToolRunner } from "./toolRunner.js";
import type {
    LlmCallResult,
    ResponseGenerationParams,
    ResponseGenerationState,
    ResponseServiceConfig,
    ResponseServiceDependencies,
    ToolExecutionResult,
} from "./types.js";

/**
 * Default configuration for ResponseService
 */
const DEFAULT_CONFIG: ResponseServiceConfig = {
    defaultTimeoutMs: 60000, // 60 seconds
    defaultMaxTokens: 4000,
    defaultTemperature: 0.7,
    enableDetailedLogging: false,
};

/**
 * Maximum number of tool call iterations to prevent infinite loops
 */
const MAX_TOOL_ITERATIONS = 10;

/**
 * ResponseService - Generates individual bot responses with clean interfaces
 * 
 * This service provides the core response generation functionality extracted
 * from ReasoningEngine.runLoop() but with unified types and clean interfaces.
 */
export class ResponseService {
    private readonly config: ResponseServiceConfig;

    constructor(
        private readonly llmRouter: LlmRouter,
        private readonly toolRunner: ToolRunner,
        private readonly contextBuilder: ContextBuilder,
        config?: Partial<ResponseServiceConfig>,
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };

        logger.info("[ResponseService] Initialized with clean response generation interface", {
            config: this.config,
        });
    }

    /**
     * 🎯 Main Entry Point: Generate a complete bot response
     * 
     * INPUT: ResponseContext (unified context type)
     * OUTPUT: ResponseResult (unified result type)
     * 
     * This is the main interface that replaces ReasoningEngine.runLoop()
     * with clean types and zero transformation logic.
     */
    async generateResponse(params: ResponseGenerationParams): Promise<ResponseResult> {
        const { context, abortSignal, overrides } = params;
        const effectiveConfig = { ...this.config, ...overrides };
        const startTime = Date.now();

        // Validate context
        const validation = this.contextBuilder.validateContext(context);
        if (!validation.valid) {
            return this.createErrorResult(
                context.botId,
                startTime,
                new Error(`Invalid context: ${validation.errors.join(", ")}`),
                "INVALID_CONTEXT",
            );
        }

        // Initialize response generation state
        const state: ResponseGenerationState = {
            messages: [],
            toolCalls: [],
            totalResourcesUsed: {
                creditsUsed: "0",
                durationMs: 0,
                toolCalls: 0,
                memoryUsedMB: 0,
                apiCalls: 0,
            },
            shouldContinue: true,
            iterationCount: 0,
            startTime,
        };

        logger.info("[ResponseService] Starting response generation", {
            botId: context.botId,
            swarmId: context.swarmId,
            strategy: context.strategy,
            availableTools: context.availableTools.length,
        });

        try {
            // Check for cancellation before starting
            this.checkAbortSignal(abortSignal, "before-response-generation");

            // Build conversation context using existing logic
            const conversationMessages = await this.contextBuilder.buildContext(context);

            // Build system message
            const systemMessage = await this.contextBuilder.buildSystemMessage(context);

            // Create full message context (system + conversation history)
            const fullMessages: ChatMessage[] = [
                {
                    id: generatePK().toString(),
                    createdAt: new Date(),
                    config: {
                        role: "system",
                        text: systemMessage,
                    },
                    user: { id: "system" },
                },
                ...conversationMessages,
            ];

            // Main response generation loop (handles tool calls)
            await this.runResponseLoop(context, fullMessages, state, abortSignal, effectiveConfig);

            // Return successful result
            const finalMessage = state.messages[state.messages.length - 1];
            const duration = Date.now() - startTime;

            return {
                success: true,
                message: finalMessage,
                toolCalls: state.toolCalls,
                confidence: this.calculateConfidence(state),
                continueConversation: this.shouldContinueConversation(state),
                botId: context.botId,
                resourcesUsed: {
                    ...state.totalResourcesUsed,
                    durationMs: duration,
                },
                duration,
                metadata: {
                    strategy: context.strategy,
                    model: context.botConfig.model,
                    iterationCount: state.iterationCount,
                    toolExecutionDetails: this.extractToolExecutionDetails(state),
                },
            };

        } catch (error) {
            logger.error("[ResponseService] Response generation failed", {
                botId: context.botId,
                error: error instanceof Error ? error.message : String(error),
                iterationCount: state.iterationCount,
            });

            return this.createErrorResult(
                context.botId,
                startTime,
                error instanceof Error ? error : new Error(String(error)),
                "RESPONSE_GENERATION_FAILED",
            );
        }
    }

    /**
     * Main response generation loop that handles LLM calls and tool execution
     * This replicates the core logic from ReasoningEngine.runLoop()
     */
    private async runResponseLoop(
        context: ResponseContext,
        messages: ChatMessage[],
        state: ResponseGenerationState,
        abortSignal?: AbortSignal,
        config: ResponseServiceConfig = this.config,
    ): Promise<void> {
        const constraints = this.contextBuilder.getEffectiveConstraints(context);

        while (state.shouldContinue && state.iterationCount < MAX_TOOL_ITERATIONS) {
            state.iterationCount++;

            // Check for cancellation
            this.checkAbortSignal(abortSignal, "before-llm-call");

            logger.debug("[ResponseService] Starting iteration", {
                iteration: state.iterationCount,
                botId: context.botId,
                messageCount: messages.length,
            });

            // Make LLM call
            const llmResult = await this.makeLlmCall(context, messages, constraints, config);

            // Update state with LLM result
            this.updateStateFromLlmCall(state, llmResult);

            if (!llmResult.success) {
                state.error = llmResult.error;
                state.shouldContinue = false;
                break;
            }

            // Add the LLM response message
            if (llmResult.message) {
                messages.push(llmResult.message);
                state.messages.push(llmResult.message);
            }

            // Handle tool calls if present
            if (llmResult.toolCalls && llmResult.toolCalls.length > 0) {
                const toolResults = await this.executeToolCalls(
                    llmResult.toolCalls,
                    context,
                    state,
                    abortSignal,
                );

                // Add tool result messages to conversation
                for (const toolResult of toolResults) {
                    if (toolResult.success) {
                        const toolMessage = this.createToolResultMessage(toolResult);
                        messages.push(toolMessage);
                    }
                }

                // Continue loop for potential follow-up responses
                state.shouldContinue = true;
            } else {
                // No tool calls, response is complete
                state.shouldContinue = false;
            }

            // Check resource limits
            if (this.hasExceededLimits(state, constraints)) {
                logger.warn("[ResponseService] Resource limits exceeded", {
                    botId: context.botId,
                    resourcesUsed: state.totalResourcesUsed,
                    constraints,
                });
                state.shouldContinue = false;
            }
        }

        // Check if we hit max iterations
        if (state.iterationCount >= MAX_TOOL_ITERATIONS) {
            logger.warn("[ResponseService] Max iterations reached", {
                botId: context.botId,
                iterations: state.iterationCount,
            });
        }
    }

    /**
     * Make a single LLM API call using existing LlmRouter
     * This delegates to existing logic with zero transformation
     */
    private async makeLlmCall(
        context: ResponseContext,
        messages: ChatMessage[],
        constraints: Required<NonNullable<ResponseContext["constraints"]>>,
        _config: ResponseServiceConfig,
    ): Promise<LlmCallResult> {
        const startTime = Date.now();

        try {
            // Convert to LlmRouter streaming parameters
            const streamOptions = {
                model: context.botConfig.model || "gpt-4",
                input: messages,
                tools: context.availableTools,
                userData: context.userData,
                maxCredits: BigInt(constraints.maxCredits),
                temperature: constraints.temperature,
                parallel_tool_calls: true,
                systemMessage: "", // Will be built by context builder
            };

            logger.debug("[ResponseService] Making LLM call", {
                model: streamOptions.model,
                messageCount: messages.length,
                toolCount: context.availableTools.length,
            });

            // Collect streaming events into a single response
            let messageContent = "";
            const toolCalls: ToolCall[] = [];
            let cost = 0;
            let responseId = "";

            for await (const event of this.llmRouter.stream(streamOptions)) {
                switch (event.type) {
                    case "message":
                        messageContent += event.content;
                        responseId = event.responseId;
                        break;
                    case "function_call":
                        toolCalls.push({
                            id: event.callId,
                            function: {
                                name: event.name,
                                arguments: event.arguments,
                            },
                        });
                        break;
                    case "reasoning":
                        // For now, append reasoning to message content
                        // This could be handled differently based on requirements
                        messageContent += `\n[Reasoning: ${event.content}]`;
                        break;
                    case "done":
                        cost = event.cost;
                        break;
                }
            }

            // Create the response message
            const message: ChatMessage = {
                id: responseId || `msg_${Date.now()}`,
                createdAt: new Date().toISOString(),
                config: {
                    role: "assistant",
                    content: messageContent,
                },
                user: { id: context.botId },
                text: messageContent,
                language: "en",
            };

            const duration = Date.now() - startTime;

            return {
                success: true,
                message,
                toolCalls,
                resourcesUsed: {
                    creditsUsed: cost.toString(),
                    durationMs: duration,
                    toolCalls: 0, // Will be counted separately
                    stepsExecuted: 1,
                    memoryUsedMB: 0,
                },
                duration,
                metadata: {
                    model: streamOptions.model,
                    tokens: { input: 0, output: 0, total: 0 }, // Token counts not available from stream
                    finishReason: toolCalls.length > 0 ? "tool_calls" : "stop",
                },
            };

        } catch (error) {
            const duration = Date.now() - startTime;

            return {
                success: false,
                error: this.createExecutionError(error, "LLM_CALL_FAILED"),
                resourcesUsed: {
                    creditsUsed: "0",
                    durationMs: duration,
                    toolCalls: 0,
                    apiCalls: 1,
                },
                duration,
            };
        }
    }

    /**
     * Execute multiple tool calls using existing ToolRunner
     * This delegates to existing logic with zero transformation
     */
    private async executeToolCalls(
        toolCalls: ToolCall[],
        context: ResponseContext,
        state: ResponseGenerationState,
        abortSignal?: AbortSignal,
    ): Promise<ToolExecutionResult[]> {
        const results: ToolExecutionResult[] = [];

        for (const toolCall of toolCalls) {
            // Check for cancellation before each tool call
            this.checkAbortSignal(abortSignal, "before-tool-call");

            const toolResult = await this.executeToolCall(toolCall, context);
            results.push(toolResult);

            // Update state
            this.updateStateFromToolCall(state, toolResult);

            // If a tool call fails, we might want to continue with other tools
            // This matches the existing behavior in ReasoningEngine
        }

        return results;
    }

    /**
     * Execute a single tool call using existing ToolRunner
     */
    private async executeToolCall(
        toolCall: ToolCall,
        context: ResponseContext,
    ): Promise<ToolExecutionResult> {
        const startTime = Date.now();

        try {
            // Convert to ToolRunner meta format (zero transformation)
            const toolMeta = {
                conversationId: context.swarmId, // Use swarmId as conversation identifier
                callerBotId: context.botId,
                sessionUser: context.userData,
            };

            logger.debug("[ResponseService] Executing tool call", {
                toolName: toolCall.function.name,
                botId: context.botId,
            });

            // Delegate to existing ToolRunner (no changes needed)
            const result = await this.toolRunner.run(
                toolCall.function.name,
                toolCall.function.arguments,
                toolMeta,
            );

            const duration = Date.now() - startTime;

            if (result.ok) {
                return {
                    success: true,
                    output: result.data.output,
                    resourcesUsed: {
                        creditsUsed: result.data.creditsUsed,
                        durationMs: duration,
                        toolCalls: 1,
                        apiCalls: 0,
                    },
                    duration,
                    originalCall: toolCall,
                };
            } else {
                return {
                    success: false,
                    error: this.createExecutionError(
                        new Error(result.error.message),
                        result.error.code,
                    ),
                    resourcesUsed: {
                        creditsUsed: result.error.creditsUsed || "0",
                        durationMs: duration,
                        toolCalls: 1,
                        apiCalls: 0,
                    },
                    duration,
                    originalCall: toolCall,
                };
            }

        } catch (error) {
            const duration = Date.now() - startTime;

            return {
                success: false,
                error: this.createExecutionError(error, "TOOL_EXECUTION_FAILED"),
                resourcesUsed: {
                    creditsUsed: "0",
                    durationMs: duration,
                    toolCalls: 1,
                    apiCalls: 0,
                },
                duration,
                originalCall: toolCall,
            };
        }
    }

    /**
     * Utility methods for state management and result creation
     */

    private updateStateFromLlmCall(state: ResponseGenerationState, result: LlmCallResult): void {
        // Add to accumulated resources
        const currentCredits = BigInt(state.totalResourcesUsed.creditsUsed);
        const newCredits = BigInt(result.resourcesUsed.creditsUsed);
        state.totalResourcesUsed.creditsUsed = (currentCredits + newCredits).toString();

        state.totalResourcesUsed.apiCalls = (state.totalResourcesUsed.apiCalls || 0) + 1;

        // Store tool calls for later reference
        if (result.toolCalls) {
            state.toolCalls.push(...result.toolCalls);
        }
    }

    private updateStateFromToolCall(state: ResponseGenerationState, result: ToolExecutionResult): void {
        // Add to accumulated resources
        const currentCredits = BigInt(state.totalResourcesUsed.creditsUsed);
        const newCredits = BigInt(result.resourcesUsed.creditsUsed);
        state.totalResourcesUsed.creditsUsed = (currentCredits + newCredits).toString();

        state.totalResourcesUsed.toolCalls = (state.totalResourcesUsed.toolCalls || 0) + 1;
    }

    private createToolResultMessage(toolResult: ToolExecutionResult): ChatMessage {
        return {
            id: generatePK().toString(),
            createdAt: new Date(),
            config: {
                role: "tool",
                text: typeof toolResult.output === "string"
                    ? toolResult.output
                    : JSON.stringify(toolResult.output),
                toolCallId: toolResult.originalCall.id,
            },
            user: { id: "system" },
        };
    }

    private createErrorResult(
        botId: BotId,
        startTime: number,
        error: Error,
        errorCode: string,
    ): ResponseResult {
        const duration = Date.now() - startTime;

        return {
            success: false,
            message: {
                id: generatePK().toString(),
                createdAt: new Date(),
                config: {
                    role: "assistant",
                    text: `I encountered an error: ${error.message}`,
                },
                user: { id: botId },
            },
            continueConversation: false,
            botId,
            error: this.createExecutionError(error, errorCode),
            resourcesUsed: {
                creditsUsed: "0",
                durationMs: duration,
                toolCalls: 0,
            },
            duration,
            metadata: {
                errorCode,
                errorMessage: error.message,
            },
        };
    }

    private createExecutionError(error: unknown, code: string): ExecutionError {
        const message = error instanceof Error ? error.message : String(error);

        return {
            code,
            message,
            tier: "response-service",
            type: error instanceof Error ? error.constructor.name : "UnknownError",
            context: {
                timestamp: new Date().toISOString(),
                service: "ResponseService",
            },
        };
    }

    private checkAbortSignal(signal?: AbortSignal, stage?: string): void {
        if (signal?.aborted) {
            throw new Error(`Response generation cancelled at ${stage || "unknown stage"}`);
        }
    }

    private calculateConfidence(state: ResponseGenerationState): number {
        // Simple confidence calculation based on success
        // This can be enhanced with more sophisticated logic
        const LOW_CONFIDENCE = 0.1;
        const HIGH_CONFIDENCE = 0.8;
        return state.error ? LOW_CONFIDENCE : HIGH_CONFIDENCE;
    }

    private shouldContinueConversation(state: ResponseGenerationState): boolean {
        // Continue if there were tool calls or if explicitly requested
        return state.toolCalls.length > 0 || state.shouldContinue;
    }

    private hasExceededLimits(
        state: ResponseGenerationState,
        constraints: Required<NonNullable<ResponseContext["constraints"]>>,
    ): boolean {
        const usedCredits = BigInt(state.totalResourcesUsed.creditsUsed);
        const maxCredits = BigInt(constraints.maxCredits);

        return usedCredits >= maxCredits;
    }

    private extractToolExecutionDetails(state: ResponseGenerationState): Array<{
        toolName: string;
        success: boolean;
        duration: number;
        creditsUsed: string;
    }> {
        // This would extract details from the state if we were tracking them
        // For now, return basic info
        return state.toolCalls.map(toolCall => ({
            toolName: toolCall.function.name,
            success: true, // We'd need to track this in state
            duration: 0, // We'd need to track this in state
            creditsUsed: "0", // We'd need to track this in state
        }));
    }

    /**
     * Factory method to create ResponseService with proper dependencies
     */
    static create(dependencies: ResponseServiceDependencies): ResponseService {
        const contextBuilder = new ContextBuilder(dependencies.contextBuilder);

        return new ResponseService(
            dependencies.llmRouter,
            dependencies.toolRunner,
            contextBuilder,
            dependencies.config,
        );
    }
}
