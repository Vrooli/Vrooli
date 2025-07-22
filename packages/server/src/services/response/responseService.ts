/**
 * ResponseService - Clean Individual Bot Response Generation with Integrated Prompt Support
 * 
 * This service extracts the core response generation logic from ReasoningEngine.runLoop()
 * and provides a clean interface using unified types. It handles:
 * - LLM API calls
 * - Tool execution
 * - Resource tracking
 * - Error handling
 * - Prompt generation with template support
 * - Swarm state formatting
 * 
 * Design Principles:
 * - Clean input/output types (ResponseContext → ResponseResult)
 * - No transformation logic (direct delegation to existing services)
 * - Reuse existing LlmRouter, ToolRunner, ContextBuilder
 * - Maintain compatibility with existing conversation system
 * - Integrated prompt generation (previously in PromptService)
 */

import type {
    BotId,
    MessageState,
    ResponseContext,
    ResponseResult,
    BotParticipant,
    ChatConfigObject,
    SessionUser,
    SwarmState,
    TriggerContext,
    ModelType,
    AgentSpec,
    DataSensitivityConfig,
    ResourceSubType,
} from "@vrooli/shared";
import { EventTypes, generatePK, SECONDS_1_MS, StandardVersionConfig, getTranslation, validatePK, valueFromDot, type BotStatus, type ErrorPayload, type ExecutionError, type SocketEventPayloads, type TeamConfigObject, type ToolCall, ModelStrategy } from "@vrooli/shared";
import * as fs from "fs/promises";
import * as path from "path";
import { fileURLToPath } from "url";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { getAuthenticatedData } from "../../utils/getAuthenticatedData.js";
import { permissionsCheck } from "../../validators/permissions.js";
import { SwarmStateAccessor } from "../execution/shared/SwarmStateAccessor.js";
import type { ToolRegistry } from "../mcp/registry.js";
import { EventPublisher } from "../events/publisher.js";
import { MessageHistoryBuilder } from "./messageHistoryBuilder.js";
import { MessageTypeAdapters } from "./typeAdapters.js";
import { FallbackRouter, type LlmRouter } from "./router.js";
import { CompositeToolRunner, type ToolRunner } from "./toolRunner.js";
import { ToolConverter } from "./converters/toolConverter.js";
import { NetworkMonitor } from "./NetworkMonitor.js";
import { ModelSelectionStrategyFactory } from "./ModelSelectionStrategy.js";
import { AIServiceRegistry } from "./registry.js";
import type {
    LlmCallResult,
    ResponseGenerationParams,
    ResponseGenerationState,
    ResponseServiceConfig,
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

// ===== PROMPT GENERATION CONSTANTS (Integrated from promptService) =====

/**
 * Time constants for prompt functionality
 */
const MINUTES_5 = 5;
const SECONDS_PER_MINUTE = 60;
const PROMPT_PREFIX_LENGTH = 7;


/**
 * Recruitment rule prompt for leader/coordinator/delegator roles
 */
const RECRUITMENT_RULE_PROMPT = `## Recruitment rule:
If setting a new goal that spans multiple knowledge domains OR is estimated to exceed 2 hours OR 500 reasoning steps, you MUST add *all* of the following subtasks to the swarm's subtasks via the \`update_swarm_shared_state\` tool BEFORE any domain work:

[
  { "id":"T1", "description":"Look for a suitable existing team",
    "status":"todo" },
  { "id":"T2", "description":"If a team is found, set it as the swarm's team",
    "status":"todo", "depends_on":["T1"] },
  { "id":"T3", "description":"If not, create a new team for the task",
    "status":"todo", "depends_on":["T1"] },
  { "id":"T4", "description":"{{GOAL}}",
    "status":"todo", "depends_on":["T2","T3"] }
]`;

/**
 * Maximum length for swarm state sections before truncation
 */
const MAX_STRING_PREVIEW_LENGTH = 2_000;

/**
 * Default fallback template when template file cannot be loaded
 */
const DEFAULT_TEMPLATE_FALLBACK = "Your primary goal is: {{GOAL}}. Please act according to your role: {{ROLE}}. Critical: Prompt template file not found.";

/**
 * Roles that should receive the recruitment rule instructions
 */
const LEADERSHIP_ROLES = ["leader", "coordinator", "delegator"] as const;

/**
 * Default member count label for single-member scenarios
 */
const DEFAULT_MEMBER_COUNT_LABEL = "1 member";

/**
 * Team-based member count label
 */
const TEAM_BASED_MEMBER_COUNT_LABEL = "team-based swarm";

// ===== PROMPT GENERATION INTERFACES =====

/**
 * Context information needed for prompt generation
 */
export interface PromptContext {
    goal: string;
    bot: BotParticipant;
    convoConfig: ChatConfigObject;
    team?: {
        id: string;
        name: string;
        config: TeamConfigObject;
    };
    toolRegistry?: ToolRegistry;
    /** User ID for security validation */
    userId?: string;
    /** Swarm ID for security context */
    swarmId?: string;
    /** Previous execution steps for variable resolution */
    previousSteps?: unknown;
    /** Input data for variable resolution */
    input?: unknown;
    /** Full SwarmState if available (needed for building TriggerContext) */
    swarmState?: SwarmState;
}

/**
 * Template variable replacements
 */
interface TemplateVariables {
    // Special computed variables that can't be handled by dot notation
    memberCountLabel: string;
    isoEpochSeconds: string;
    displayDate: string;
    roleSpecificInstructions: string;
    swarmState: string;
    toolSchemas: string;
}

/**
 * Configuration options for prompt generation
 */
interface PromptServiceConfig {
    templateDirectory?: string;
    defaultTemplate?: string;
    enableTemplateCache?: boolean;
    maxStringPreviewLength?: number;
    /** Cache TTL for user prompts in milliseconds */
    userPromptCacheTTL?: number;
}

/**
 * Cache entry for templates
 */
interface CacheEntry {
    content: string;
    metadata?: {
        name: string;
        description: string;
        version: string;
    };
    cachedAt: number;
}

/**
 * Processed prompt data
 */
interface ProcessedPrompt {
    template: string;
    inputSchema?: Record<string, unknown>;
    metadata?: {
        name: string;
        description: string;
        version: string;
    };
    processingRules?: Record<string, unknown>;
}

/**
 * Agent prompt resolution result
 */
interface AgentPromptResolution {
    content: string;
    mode: "supplement" | "replace";
    variables: Record<string, string>;
}

// Default prompt configuration
const DEFAULT_PROMPT_CONFIG: Required<PromptServiceConfig> = {
    templateDirectory: "",  // Will be set dynamically
    defaultTemplate: "prompt.txt",
    enableTemplateCache: true,
    maxStringPreviewLength: MAX_STRING_PREVIEW_LENGTH,
    userPromptCacheTTL: MINUTES_5 * SECONDS_PER_MINUTE * SECONDS_1_MS,
};

/**
 * ResponseService - Generates individual bot responses with clean interfaces
 * 
 * This service provides the core response generation functionality extracted
 * from ReasoningEngine.runLoop() but with unified types and clean interfaces.
 * Now includes integrated prompt generation functionality.
 */
export class ResponseService {
    private readonly config: ResponseServiceConfig;
    
    // Static template cache for prompt functionality
    private static readonly templateCache = new Map<string, string | CacheEntry>();

    constructor(
        config?: Partial<ResponseServiceConfig>,
        private readonly toolRunner: ToolRunner = new CompositeToolRunner(),
        private readonly llmRouter: LlmRouter = new FallbackRouter(),
    ) {
        this.config = { ...DEFAULT_CONFIG, ...config };

        logger.info("[ResponseService] Initialized with UnifiedPromptBuilder interface", {
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

        // Validate context (basic validation since UnifiedPromptBuilder handles most edge cases)
        if (!context.swarmId || !context.bot || !context.bot.config || !context.userData) {
            return this.createErrorResult(
                context.bot?.id || "unknown",
                startTime,
                new Error("Invalid context: missing required fields"),
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
                stepsExecuted: 0,
            },
            shouldContinue: true,
            iterationCount: 0,
            startTime,
        };

        logger.info("[ResponseService] Starting response generation", {
            botId: context.bot?.id,
            swarmId: context.swarmId,
            strategy: context.strategy,
            availableTools: context.availableTools.length,
        });

        try {
            // Emit typing start event
            await this.emitTypingStatus(context.bot.id, true);

            // Emit thinking status
            await this.emitBotStatus(context.bot.id, "thinking", "Processing request...");

            // Check for cancellation before starting
            this.checkAbortSignal(abortSignal, "before-response-generation");

            // Build complete message array using MessageHistoryBuilder
            // Returns MessageState[] directly - no conversion needed
            const fullMessages = await MessageHistoryBuilder.get().buildMessages(context);

            // Main response generation loop (handles tool calls)
            await this.runResponseLoop(context, fullMessages, state, abortSignal, effectiveConfig);

            // Emit processing complete status
            await this.emitBotStatus(context.bot.id, "processing_complete", "Finished processing turn.");

            // Return successful result
            const finalMessage = state.messages[state.messages.length - 1];
            const duration = Date.now() - startTime;

            return {
                success: true,
                message: finalMessage,
                toolCalls: state.toolCalls,
                confidence: this.calculateConfidence(state),
                continueConversation: this.shouldContinueConversation(state),
                botId: context.bot.id,
                resourcesUsed: {
                    ...state.totalResourcesUsed,
                    durationMs: duration,
                },
                duration,
                metadata: {
                    strategy: context.strategy,
                    model: context.bot.config.modelConfig?.preferredModel,
                    iterationCount: state.iterationCount,
                    toolExecutionDetails: this.extractToolExecutionDetails(state),
                },
            };

        } catch (error) {
            logger.error("[ResponseService] Response generation failed", {
                botId: context.bot.id,
                error: error instanceof Error ? error.message : String(error),
                iterationCount: state.iterationCount,
            });

            return this.createErrorResult(
                context.bot.id,
                startTime,
                error instanceof Error ? error : new Error(String(error)),
                "RESPONSE_GENERATION_FAILED",
            );
        } finally {
            // Always emit typing stop
            await this.emitTypingStatus(context.bot.id, false);
        }
    }

    /**
     * Main response generation loop that handles LLM calls and tool execution
     * This replicates the core logic from ReasoningEngine.runLoop()
     */
    private async runResponseLoop(
        context: ResponseContext,
        messages: MessageState[],
        state: ResponseGenerationState,
        abortSignal?: AbortSignal,
        config: ResponseServiceConfig = this.config,
    ): Promise<void> {
        const constraints = this.getEffectiveConstraints(context);

        while (state.shouldContinue && state.iterationCount < MAX_TOOL_ITERATIONS) {
            state.iterationCount++;

            // Check for cancellation
            this.checkAbortSignal(abortSignal, "before-llm-call");

            // Processing iteration

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
                    botId: context.bot.id,
                    resourcesUsed: state.totalResourcesUsed,
                    constraints,
                });
                state.shouldContinue = false;
            }
        }

        // Check if we hit max iterations
        if (state.iterationCount >= MAX_TOOL_ITERATIONS) {
            logger.warn("[ResponseService] Max iterations reached", {
                botId: context.bot.id,
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
        messages: MessageState[],
        constraints: Required<NonNullable<ResponseContext["constraints"]>>,
        _config: ResponseServiceConfig,
    ): Promise<LlmCallResult> {
        const startTime = Date.now();

        try {
            // Select model using proper hierarchy and strategy:
            // 1. Chat's modelConfig (highest priority)
            // 2. Bot's modelConfig
            // 3. Team's modelConfig
            // 4. System default
            const model = await this.selectModel(context);

            // Convert to LlmRouter streaming parameters
            const streamOptions = {
                model,
                input: messages,
                tools: ToolConverter.toOpenAIFormat(context.availableTools),
                userData: context.userData,
                maxCredits: BigInt(constraints.maxCredits),
                temperature: constraints.temperature,
                parallel_tool_calls: true,
                systemMessage: "", // Will be built by context builder
            };

            // Making LLM call

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

                        // Emit streaming event if enabled
                        if (this.config.enableStreaming && this.config.streamingChatId) {
                            await this.emitBotResponseStream({
                                __type: event.final ? "end" : "stream",
                                chatId: this.config.streamingChatId,
                                botId: context.bot.id,
                                chunk: event.content,
                            });
                        }
                        break;

                    case "function_call":
                        // Validate arguments type - should be either string (JSON) or object
                        const validatedArguments = (typeof event.arguments === "string" || 
                                                   (typeof event.arguments === "object" && event.arguments !== null))
                            ? event.arguments as string | Record<string, unknown>
                            : JSON.stringify(event.arguments); // Fallback: convert to string
                        
                        toolCalls.push({
                            id: event.callId,
                            function: {
                                name: event.name,
                                arguments: validatedArguments,
                            },
                        });
                        break;

                    case "reasoning":
                        // Emit reasoning stream event if enabled
                        if (this.config.enableStreaming && this.config.streamingChatId) {
                            await this.emitBotModelReasoningStream({
                                __type: "stream",
                                chatId: this.config.streamingChatId,
                                botId: context.bot.id,
                                chunk: event.content,
                            });
                        }
                        // For now, append reasoning to message content
                        // This could be handled differently based on requirements
                        messageContent += `\n[Reasoning: ${event.content}]`;
                        break;

                    case "done":
                        cost = event.cost;

                        // Emit final streaming event if enabled
                        if (this.config.enableStreaming && this.config.streamingChatId) {
                            await this.emitBotResponseStream({
                                __type: "end",
                                chatId: this.config.streamingChatId,
                                botId: context.bot.id,
                                finalMessage: messageContent,
                            });
                        }
                        break;
                }
            }

            // Create the response message using type adapters
            const message = MessageTypeAdapters.createMessageState({
                id: responseId || `msg_${Date.now()}`,
                text: messageContent,
                role: "assistant",
                userId: context.bot.id,
                language: "en",
            });

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
                    stepsExecuted: 0,
                    memoryUsedMB: 0,
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
                callerBotId: context.bot.id,
                sessionUser: context.userData,
            };

            // Executing tool call

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
                        stepsExecuted: 0,
                        memoryUsedMB: 0,
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
                        stepsExecuted: 0,
                        memoryUsedMB: 0,
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
                    stepsExecuted: 0,
                    memoryUsedMB: 0,
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
        state.totalResourcesUsed.stepsExecuted = (state.totalResourcesUsed.stepsExecuted || 0) + 1;

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


    private createToolResultMessage(toolResult: ToolExecutionResult): MessageState {
        return MessageTypeAdapters.createToolResultMessage({
            toolCallId: toolResult.originalCall.id,
            output: toolResult.output,
            success: toolResult.success,
            toolName: toolResult.originalCall.function.name,
        });
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
            message: MessageTypeAdapters.createMessageState({
                id: generatePK().toString(),
                text: `I encountered an error: ${error.message}`,
                role: "assistant",
                userId: botId,
                language: "en",
            }),
            continueConversation: false,
            botId,
            error: this.createExecutionError(error, errorCode),
            resourcesUsed: {
                creditsUsed: "0",
                durationMs: duration,
                toolCalls: 0,
                stepsExecuted: 0,
                memoryUsedMB: 0,
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
            tier: "tier3",
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

    /**
     * Select the appropriate model based on hierarchy and strategy:
     * 1. Chat's modelConfig (highest priority)
     * 2. Bot's modelConfig
     * 3. Team's modelConfig
     * 4. System default
     */
    private async selectModel(context: ResponseContext): Promise<string> {
        // Get effective model config from hierarchy
        const modelConfig = this.getEffectiveModelConfig(context);
        
        // Get network state and registry
        const networkMonitor = NetworkMonitor.getInstance();
        const networkState = await networkMonitor.getState();
        const registry = AIServiceRegistry.get();
        
        // Apply model selection strategy
        const strategy = ModelSelectionStrategyFactory.getStrategy(modelConfig.strategy);
        
        try {
            const selectedModel = await strategy.selectModel({
                modelConfig,
                networkState,
                registry,
                userCredits: context.userData?.credits ? Number(context.userData.credits) : undefined,
            });
            
            logger.debug("[ResponseService] Selected model", {
                model: selectedModel,
                strategy: modelConfig.strategy,
                source: this.getModelConfigSource(context),
            });
            
            return selectedModel;
        } catch (error) {
            logger.error("[ResponseService] Model selection failed", { error, modelConfig });
            // Fallback to system default
            return "gpt-4";
        }
    }

    /**
     * Get effective model config from hierarchy
     */
    private getEffectiveModelConfig(context: ResponseContext) {
        // Priority: Chat > Bot > Team > Default
        const chatConfig = context.chatConfig?.modelConfig;
        const botConfig = context.bot.config?.modelConfig;
        const teamConfig = context.teamConfig?.modelConfig;
        
        if (chatConfig) return chatConfig;
        if (botConfig) return botConfig;
        if (teamConfig) return teamConfig;
        
        // Default configuration
        return {
            strategy: ModelStrategy.FALLBACK,
            preferredModel: "gpt-4",
            offlineOnly: false,
        };
    }

    /**
     * Get the source of the model config for logging
     */
    private getModelConfigSource(context: ResponseContext): string {
        if (context.chatConfig?.modelConfig) return "chat";
        if (context.bot.config?.modelConfig) return "bot";
        if (context.teamConfig?.modelConfig) return "team";
        return "default";
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
     * Emit bot response stream event
     */
    private async emitBotResponseStream(data: {
        __type: "stream" | "end" | "error";
        chatId: string;
        botId?: string;
        chunk?: string;
        finalMessage?: string;
        error?: ErrorPayload;
    }): Promise<void> {
        await this.emitEventSafely(EventTypes.CHAT.BOT_RESPONSE_STREAM, data);
    }

    /**
     * Emit bot model reasoning stream event
     */
    private async emitBotModelReasoningStream(data: {
        __type: "stream" | "end" | "error";
        chatId: string;
        botId?: string;
        chunk?: string;
        error?: ErrorPayload;
    }): Promise<void> {
        await this.emitEventSafely(EventTypes.CHAT.BOT_MODEL_REASONING_STREAM, data);
    }

    /**
     * Emit bot typing updated event
     */
    private async emitBotTypingUpdated(data: {
        chatId: string;
        starting?: string[];
        stopping?: string[];
    }): Promise<void> {
        await this.emitEventSafely(EventTypes.CHAT.BOT_TYPING_UPDATED, data);
    }

    /**
     * Emit bot status updated event
     */
    private async emitBotStatusUpdated(data: {
        chatId: string;
        botId: string;
        status: BotStatus;
        message?: string;
        toolInfo?: {
            callId: string;
            name: string;
            args?: string;
            result?: string;
            error?: string;
            pendingId?: string;
            reason?: string;
        };
        error?: ErrorPayload;
    }): Promise<void> {
        await this.emitEventSafely(EventTypes.CHAT.BOT_STATUS_UPDATED, data);
    }

    /**
     * Safely emit an event to the event bus
     */
    private async emitEventSafely<T extends SocketEventPayloads[keyof SocketEventPayloads]>(
        eventType: string,
        data: T,
    ): Promise<void> {
        try {
            const { proceed, reason } = await EventPublisher.emit(eventType, data);

            if (!proceed) {
                // Streaming events being blocked is noteworthy but not critical
                logger.warn("[ResponseService] Event blocked", {
                    eventType,
                    reason,
                    chatId: this.config.streamingChatId,
                });
            }
        } catch (error) {
            // Don't let streaming errors break the main flow
            logger.error("[ResponseService] Failed to emit event", {
                error: error instanceof Error ? error.message : String(error),
                eventType,
            });
        }
    }

    /**
     * Emit typing status events
     */
    async emitTypingStatus(botId: string, starting: boolean): Promise<void> {
        if (!this.config.enableStreaming || !this.config.streamingChatId) {
            return;
        }

        await this.emitBotTypingUpdated({
            chatId: this.config.streamingChatId,
            [starting ? "starting" : "stopping"]: [botId],
        });
    }

    /**
     * Emit bot status events
     */
    async emitBotStatus(botId: string, status: string, message?: string): Promise<void> {
        if (!this.config.enableStreaming || !this.config.streamingChatId) {
            return;
        }

        await this.emitBotStatusUpdated({
            chatId: this.config.streamingChatId,
            botId,
            status: status as BotStatus,
            message,
        });
    }

    /**
     * Get effective constraints from context and defaults
     */
    private getEffectiveConstraints(context: ResponseContext): Required<NonNullable<ResponseContext["constraints"]>> {
        const defaults = {
            maxTokens: 4000,
            temperature: 0.7,
            timeoutMs: 60000, // 60 seconds
            maxCredits: "1000", // Default credit limit
        };

        return {
            maxTokens: context.constraints?.maxTokens ?? defaults.maxTokens,
            temperature: context.constraints?.temperature ?? defaults.temperature,
            timeoutMs: context.constraints?.timeoutMs ?? defaults.timeoutMs,
            maxCredits: context.constraints?.maxCredits ?? defaults.maxCredits,
        };
    }

    // ===== INTEGRATED PROMPT GENERATION METHODS =====

    /**
     * 🎯 Build sophisticated system message with template support
     * 
     * INPUT: PromptContext (bot, config, goal, etc.)
     * OUTPUT: Fully processed system message string
     * 
     * Enhanced to support agent-specific prompts that replace or supplement the base template
     * Now supports direct prompt content from bot configuration and SwarmStateAccessor integration
     */
    static async buildSystemMessage(
        context: PromptContext,
        options?: {
            templateIdentifier?: string;
            userData?: SessionUser | null;
            config?: PromptServiceConfig;
            directPromptContent?: string;
        },
    ): Promise<string> {
        const config = this.getPromptConfig(options?.config);

        // Log when enhanced SwarmState context is available
        logger.debug("[ResponseService] Building system message", {
            botId: context.bot.id,
            hasSwarmState: !!context.swarmState,
        });

        // 1. Check for direct prompt content first (highest priority)
        if (options?.directPromptContent) {
            logger.debug("[ResponseService] Using direct prompt content", {
                botId: context.bot.id,
                contentLength: options.directPromptContent.length,
            });

            // Prepare template variables with SwarmStateAccessor integration
            const variables = await this.prepareTemplateVariables(
                context,
                undefined,
                config,
            );

            // Process direct content with variables
            const processedPrompt = ResponseService.processTemplate(options.directPromptContent, variables, context);
            await this.validatePromptSafety(processedPrompt, context);
            return processedPrompt.trim();
        }

        // 2. Check for agent-specific prompt configuration
        const agentSpec = context.bot.config?.agentSpec as AgentSpec | undefined;
        if (agentSpec?.prompt?.source === "direct" && agentSpec.prompt.content) {
            logger.debug("[ResponseService] Using agent-specific direct prompt", {
                botId: context.bot.id,
                mode: agentSpec.prompt.mode,
                contentLength: agentSpec.prompt.content.length,
            });

            // Prepare template variables with custom mappings
            const variables = await this.prepareTemplateVariables(
                context,
                agentSpec.prompt.variables,
                config,
            );

            // Process agent prompt directly
            const processedPrompt = ResponseService.processTemplate(agentSpec.prompt.content, variables, context);
            await this.validatePromptSafety(processedPrompt, context);
            return processedPrompt.trim();
        }

        // 3. Fall back to template-based approach
        const templateName = options?.templateIdentifier || config.defaultTemplate;

        // Load base template
        const baseTemplate = await this.loadTemplate(templateName, options?.userData, config);

        // Resolve agent-specific prompt if available (resource-based or supplement mode)
        const agentPrompt = await this.resolveAgentPrompt(context, agentSpec);

        // Prepare template variables with safety checks
        const variables = await this.prepareTemplateVariables(
            context,
            agentPrompt?.variables,
            config,
        );

        // Process base template
        let systemMessage = ResponseService.processTemplate(baseTemplate, variables, context);

        // Apply agent prompt based on mode
        if (agentPrompt) {
            if (agentPrompt.mode === "replace") {
                systemMessage = ResponseService.processTemplate(
                    agentPrompt.content,
                    variables,
                    context,
                );
            } else {
                // Supplement mode
                systemMessage = ResponseService.composePrompts(
                    systemMessage,
                    agentPrompt.content,
                    variables,
                    context,
                );
            }
        }

        // Final safety validation
        await this.validatePromptSafety(systemMessage, context);

        return systemMessage.trim();
    }

    /**
     * 📁 Load Template from File System or Database
     */
    static async loadTemplate(
        identifier: string,
        userData?: SessionUser | null,
        config?: PromptServiceConfig,
    ): Promise<string> {
        // Check if this is a user prompt
        if (identifier.startsWith("prompt:")) {
            const promptId = identifier.substring(PROMPT_PREFIX_LENGTH);
            const processed = await this.loadUserPrompt(promptId, userData, config);
            return processed.template;
        }

        // Otherwise, load from file system
        return this.loadFileTemplate(identifier, config);
    }

    /**
     * Load template from file system
     */
    private static async loadFileTemplate(
        templateName: string,
        config?: PromptServiceConfig,
    ): Promise<string> {
        const cfg = this.getPromptConfig(config);
        const fileName = templateName || cfg.defaultTemplate;

        // Check cache first
        if (cfg.enableTemplateCache && this.templateCache.has(fileName)) {
            const cached = this.templateCache.get(fileName);
            if (typeof cached === "string") {
                return cached;
            }
        }

        const templatePath = path.join(cfg.templateDirectory, fileName);

        try {
            logger.debug(`[ResponseService] Loading template from: ${templatePath}`);
            const template = await fs.readFile(templatePath, "utf-8");

            // Cache the template
            if (cfg.enableTemplateCache) {
                this.templateCache.set(fileName, template);
            }

            return template;
        } catch (error) {
            logger.error(`[ResponseService] Failed to load template from ${templatePath}`, { error });
            return DEFAULT_TEMPLATE_FALLBACK;
        }
    }

    /**
     * Load user-generated prompt from database
     */
    private static async loadUserPrompt(
        promptVersionId: string,
        userData?: SessionUser | null,
        config?: PromptServiceConfig,
    ): Promise<ProcessedPrompt> {
        const cfg = this.getPromptConfig(config);

        // 1. Check cache first
        const cacheKey = `prompt:${promptVersionId}`;
        if (cfg.enableTemplateCache && this.templateCache.has(cacheKey)) {
            const cached = this.templateCache.get(cacheKey) as CacheEntry;
            if (cached && this.isUserPromptCacheValid(cached, cfg)) {
                return cached.content as unknown as ProcessedPrompt;
            }
        }

        // 2. Query database with correct identifier and relationship structure
        const isValidPK = validatePK(promptVersionId);
        const whereClause = isValidPK
            ? { id: BigInt(promptVersionId), resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false }
            : { publicId: promptVersionId, resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false };

        const promptVersion = await DbProvider.get().resource_version.findUnique({
            where: whereClause,
            select: {
                id: true,
                config: true,
                isDeleted: true,
                isPrivate: true,
                resourceSubType: true,
                translations: {
                    select: {
                        id: true,
                        language: true,
                        name: true,
                        description: true,
                    },
                },
                root: {
                    select: {
                        id: true,
                        isDeleted: true,
                        isPrivate: true,
                        ownedByUserId: true,
                        ownedByTeamId: true,
                        ownedByTeam: userData ? {
                            select: {
                                id: true,
                                members: {
                                    select: {
                                        userId: true,
                                        isAdmin: true,
                                        permissions: true,
                                    },
                                },
                            },
                        } : false,
                        ownedByUser: {
                            select: {
                                id: true,
                            },
                        },
                    },
                },
            },
        });

        if (!promptVersion) {
            throw new CustomError("0022", "NotFound", { objectType: "Prompt" });
        }

        // 3. Use existing permission infrastructure with actual database ID
        const actualId = promptVersion.id.toString();
        const authDataById = await getAuthenticatedData(
            { ResourceVersion: [actualId] } as { [key in ModelType]?: string[] },
            userData ?? null,
        );

        // Check read permissions using existing system
        await permissionsCheck(
            authDataById,
            { Read: [actualId] },
            {},
            userData ?? null,
        );

        // 4. Process and cache the prompt
        const processed = this.processUserPrompt(promptVersion as any, userData?.languages?.[0] || "en");

        // Cache with TTL
        if (cfg.enableTemplateCache) {
            this.templateCache.set(cacheKey, {
                content: processed as unknown as string,
                cachedAt: Date.now(),
            });
        }

        return processed;
    }

    /**
     * Check if user prompt cache is still valid
     */
    private static isUserPromptCacheValid(entry: CacheEntry, config: Required<PromptServiceConfig>): boolean {
        if (!config.userPromptCacheTTL) return true;
        return Date.now() - entry.cachedAt < config.userPromptCacheTTL;
    }

    /**
     * Process user prompt from database into usable format
     */
    private static processUserPrompt(
        promptVersion: {
            config?: Record<string, unknown>;
            translations?: Array<{ language: string; name?: string; description?: string }>;
            versionLabel?: string;
            [key: string]: unknown;
        },
        language: string,
    ): ProcessedPrompt {
        const config = (promptVersion.config || {}) as Record<string, unknown>;

        // Extract template from config
        const template = this.extractTemplateFromConfig(config);

        // Parse input schema if present
        const inputSchema = config.schema ?
            JSON.parse(config.schema as string) : null;

        // Get translated metadata
        const translation = getTranslation(promptVersion, [language]);

        return {
            template,
            inputSchema,
            metadata: {
                name: translation.name || "",
                description: translation.description || "",
                version: promptVersion.versionLabel || "",
            },
            processingRules: config.props as Record<string, unknown> | undefined,
        };
    }

    /**
     * Extract template string from config using proper StandardVersionConfig parser
     */
    private static extractTemplateFromConfig(config: Record<string, unknown>): string {
        try {
            // Use StandardVersionConfig constructor directly to avoid type issues
            const parsedConfig = new StandardVersionConfig({
                config: {
                    __version: (config.__version as string) || "1.0",
                    ...config,
                },
                resourceSubType: "StandardPrompt" as ResourceSubType,
            });

            // Check props.template first (most direct)
            if (parsedConfig.props?.template && typeof parsedConfig.props.template === "string") {
                return parsedConfig.props.template;
            }

            // Check props.content second
            if (parsedConfig.props?.content && typeof parsedConfig.props.content === "string") {
                return parsedConfig.props.content;
            }

            // Check schema field - could contain template as JSON or raw text
            if (parsedConfig.schema && typeof parsedConfig.schema === "string") {
                try {
                    const parsed = JSON.parse(parsedConfig.schema) as Record<string, unknown>;
                    if (parsed.template && typeof parsed.template === "string") {
                        return parsed.template;
                    }
                    if (parsed.content && typeof parsed.content === "string") {
                        return parsed.content;
                    }
                } catch {
                    // If schema is not JSON, treat it as the raw template
                    return parsedConfig.schema;
                }
            }

            logger.warn("[ResponseService] No template found in StandardPrompt config", {
                hasProps: !!parsedConfig.props,
                hasSchema: !!parsedConfig.schema,
                propsKeys: parsedConfig.props ? Object.keys(parsedConfig.props) : [],
            });

        } catch (error) {
            logger.error("[ResponseService] Failed to parse StandardPrompt config", { error });
        }

        return DEFAULT_TEMPLATE_FALLBACK;
    }

    /**
     * Resolve agent-specific prompt configuration
     */
    private static async resolveAgentPrompt(
        context: PromptContext,
        agentSpec?: AgentSpec,
    ): Promise<AgentPromptResolution | null> {
        if (!agentSpec?.prompt) return null;

        if (agentSpec.prompt.source === "direct") {
            if (!agentSpec.prompt.content) {
                throw new Error("Direct prompt source requires content field");
            }
            return {
                content: agentSpec.prompt.content,
                mode: agentSpec.prompt.mode,
                variables: agentSpec.prompt.variables || {},
            };
        }

        // Resource-based prompt
        if (!agentSpec.prompt.resourceId) {
            throw new Error("Resource prompt source requires resourceId field");
        }
        const resource = await this.loadPromptResource(
            agentSpec.prompt.resourceId,
            context,
        );

        return {
            content: resource.content,
            mode: agentSpec.prompt.mode,
            variables: agentSpec.prompt.variables || {},
        };
    }

    /**
     * Load prompt from resource
     */
    private static async loadPromptResource(
        resourceId: string,
        _context: PromptContext,
    ): Promise<{ content: string }> {
        try {
            // Load prompt resource from database using existing patterns
            const isValidPK = validatePK(resourceId);
            const whereClause = isValidPK
                ? { id: BigInt(resourceId), resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false }
                : { publicId: resourceId, resourceSubType: "StandardPrompt" as ResourceSubType, isDeleted: false };

            const promptVersion = await DbProvider.get().resource_version.findUnique({
                where: whereClause,
                select: {
                    config: true,
                },
            });

            if (!promptVersion) {
                throw new CustomError("0022", "NotFound", { objectType: "PromptResource" });
            }

            const config = (promptVersion.config || {}) as Record<string, unknown>;
            const content = this.extractTemplateFromConfig(config);

            return { content };
        } catch (error) {
            logger.error("[ResponseService] Failed to load prompt resource", { resourceId, error });
            return { content: DEFAULT_TEMPLATE_FALLBACK };
        }
    }

    /**
     * Prepare template variables with safety checks
     */
    private static async prepareTemplateVariables(
        context: PromptContext,
        customMappings?: Record<string, string>,
        config?: Required<PromptServiceConfig>,
    ): Promise<TemplateVariables> {
        const cfg = config || this.getPromptConfig();

        // Build standard variables
        const variables = this.buildTemplateVariables(context, cfg);

        // Resolve custom mappings with safety checks
        if (customMappings) {
            for (const [varName, path] of Object.entries(customMappings)) {
                try {
                    const value = await this.resolveVariableWithSafety(
                        path,
                        context,
                    );
                    (variables as any)[varName] = value;
                } catch (error) {
                    logger.warn(`[ResponseService] Failed to resolve variable ${varName} from path ${path}`, { error });
                    (variables as any)[varName] = "";
                }
            }
        }

        return variables;
    }

    /**
     * Resolve variable with safety checks using SwarmStateAccessor
     */
    private static async resolveVariableWithSafety(
        path: string,
        context: PromptContext,
    ): Promise<string> {
        const [scope, ...pathParts] = path.split(".");

        // Check if accessing sensitive data
        const sensitivity = this.checkDataSensitivity(path, context);
        if (sensitivity) {
            await this.handleSensitiveDataAccess(
                path,
                sensitivity,
                context,
            );
        }

        // Use SwarmStateAccessor for swarm-related data if available
        if ((scope === "swarm" || scope === "context" || scope === "config")) {
            try {
                let triggerContext: TriggerContext;

                if (context.swarmState) {
                    // Build TriggerContext from SwarmState
                    triggerContext = (new SwarmStateAccessor()).buildTriggerContext(
                        context.swarmState,
                        undefined,
                        context.bot,
                    );
                    logger.debug(`[ResponseService] Built TriggerContext from SwarmState for path ${path}`, {
                        agentId: context.bot.id,
                        swarmId: context.swarmId,
                        pathScope: scope,
                    });
                } else {
                    // Not enough context to use SwarmStateAccessor properly
                    throw new Error("Insufficient context: Need either triggerContext or swarmState");
                }

                const value = await (new SwarmStateAccessor()).accessData(
                    pathParts.join("."),
                    triggerContext,
                    context.swarmState,
                );

                logger.debug("[ResponseService] SwarmStateAccessor resolved variable successfully", {
                    path,
                    hasValue: value !== undefined,
                    agentId: context.bot.id,
                });

                return String(value || "");
            } catch (error) {
                logger.warn(`[ResponseService] SwarmStateAccessor failed for path ${path}`, {
                    error: error instanceof Error ? error.message : String(error),
                    agentId: context.bot.id,
                    swarmId: context.swarmId,
                });
                // Fall back to direct access with logging
            }
        }

        // Fall back to direct access for non-swarm data or when SwarmStateAccessor unavailable
        switch (scope) {
            case "input":
                return String(valueFromDot(context.input as any, pathParts.join(".")) || "");
            case "context":
                return String(valueFromDot(context as any, pathParts.join(".")) || "");
            case "config":
                return String(valueFromDot(context.convoConfig as any, pathParts.join(".")) || "");
            case "previous":
                return String(valueFromDot(context.previousSteps as any, pathParts.join(".")) || "");
            case "swarm":
                // If no SwarmStateAccessor, try direct access to convoConfig
                return String(valueFromDot(context.convoConfig as any, pathParts.join(".")) || "");
            default:
                throw new Error(`Unknown variable scope: ${scope}`);
        }
    }

    /**
     * Check if data path is sensitive
     */
    private static checkDataSensitivity(
        path: string,
        context: PromptContext,
    ): DataSensitivityConfig | null {
        const secrets = context.convoConfig?.secrets;
        if (!secrets) return null;

        // Check if path matches any secret pattern
        for (const [pattern, sensitivity] of Object.entries(secrets)) {
            if (this.pathMatchesPattern(path, pattern)) {
                return sensitivity;
            }
        }

        return null;
    }

    /**
     * Check if path matches pattern (simple glob-like matching)
     */
    private static pathMatchesPattern(path: string, pattern: string): boolean {
        // Convert glob pattern to regex
        const regexPattern = pattern
            .replace(/\*/g, ".*")
            .replace(/\?/g, ".")
            .replace(/\./g, "\\.");

        const regex = new RegExp(`^${regexPattern}$`);
        return regex.test(path);
    }

    /**
     * Handle sensitive data access
     */
    private static async handleSensitiveDataAccess(
        path: string,
        sensitivity: DataSensitivityConfig,
        context: PromptContext,
    ): Promise<void> {
        // For now, just log the access - in full implementation this would
        // emit safety events to the event bus
        if (sensitivity.accessLog) {
            logger.info("[ResponseService] Accessing sensitive data", {
                path,
                sensitivity: sensitivity.type,
                agentId: context.bot.id,
                userId: context.userId,
                swarmId: context.swarmId,
            });
        }

        // In the full implementation, this would check sensitivity.requireConfirmation
        // and emit pre-action safety events
    }

    /**
     * Compose prompts for supplement mode
     */
    private static composePrompts(
        basePrompt: string,
        agentPrompt: string,
        variables: TemplateVariables,
        context: PromptContext,
    ): string {
        const processedAgentPrompt = ResponseService.processTemplate(agentPrompt, variables, context);

        return `${basePrompt}\n\n## Agent-Specific Instructions\n${processedAgentPrompt}`;
    }

    /**
     * Validate prompt safety (placeholder for future implementation)
     */
    private static async validatePromptSafety(
        prompt: string,
        context: PromptContext,
    ): Promise<void> {
        // Placeholder for prompt injection detection and other safety checks
        logger.debug("[ResponseService] Validating prompt safety", {
            promptLength: prompt.length,
            agentId: context.bot.id,
        });
    }

    /**
     * 🔄 Process Template Variables
     */
    static processTemplate(template: string, variables: TemplateVariables, context: PromptContext): string {
        let processedPrompt = template;

        // Replace special computed variables that can't be handled by dot notation
        processedPrompt = processedPrompt.replace(/{{MEMBER_COUNT_LABEL}}/g, variables.memberCountLabel);
        processedPrompt = processedPrompt.replace(/{{ISO_EPOCH_SECONDS}}/g, variables.isoEpochSeconds);
        processedPrompt = processedPrompt.replace(/{{DISPLAY_DATE}}/g, variables.displayDate);
        processedPrompt = processedPrompt.replace(/{{ROLE_SPECIFIC_INSTRUCTIONS}}/g, variables.roleSpecificInstructions);
        processedPrompt = processedPrompt.replace(/{{SWARM_STATE}}/g, variables.swarmState);
        processedPrompt = processedPrompt.replace(/{{TOOL_SCHEMAS}}/g, variables.toolSchemas);

        // Handle dot notation variables
        processedPrompt = processedPrompt.replace(/{{GOAL}}/g, context.goal || "");
        processedPrompt = processedPrompt.replace(/{{BOT\.role}}/g, context.bot.role || "");
        processedPrompt = processedPrompt.replace(/{{BOT\.name}}/g, context.bot.name || "");
        processedPrompt = processedPrompt.replace(/{{BOT\.id}}/g, context.bot.id || "");
        processedPrompt = processedPrompt.replace(/{{TEAM_NAME}}/g, context.team?.name || "");
        processedPrompt = processedPrompt.replace(/{{TEAM\.name}}/g, context.team?.name || "");
        processedPrompt = processedPrompt.replace(/{{TEAM\.id}}/g, context.team?.id || "");
        processedPrompt = processedPrompt.replace(/{{TEAM\.goal}}/g, context.team?.config.goal || "");
        processedPrompt = processedPrompt.replace(/{{TEAM\.businessPrompt}}/g, context.team?.config.businessPrompt || "");

        return processedPrompt;
    }

    /**
     * 🏗️ Build Template Variables from Context
     */
    private static buildTemplateVariables(
        context: PromptContext,
        config: Required<PromptServiceConfig>,
    ): TemplateVariables {
        const { bot, convoConfig, team, toolRegistry } = context;
        const botRole = bot.role || "leader"; // Default to leader if no role

        // Determine member count label
        let memberCountLabel = DEFAULT_MEMBER_COUNT_LABEL;
        if (convoConfig.teamId) {
            memberCountLabel = TEAM_BASED_MEMBER_COUNT_LABEL;
        }

        // Get role-specific instructions
        const roleSpecificInstructions = this.getRoleSpecificInstructions(botRole);

        // Build swarm state string
        const swarmState = this.formatSwarmState(convoConfig, team?.config, config);

        // Build tool schemas string
        const toolSchemas = this.buildToolSchemasString(toolRegistry);

        return {
            memberCountLabel,
            isoEpochSeconds: Math.floor(Date.now() / SECONDS_1_MS).toString(),
            displayDate: new Date().toLocaleString(),
            roleSpecificInstructions,
            swarmState,
            toolSchemas,
        };
    }

    /**
     * 🎭 Get Role-Specific Instructions
     */
    private static getRoleSpecificInstructions(botRole: string): string {
        if (LEADERSHIP_ROLES.includes(botRole as typeof LEADERSHIP_ROLES[number])) {
            return RECRUITMENT_RULE_PROMPT;
        }
        return "Perform tasks according to your role and the overall goal.";
    }

    /**
     * 🌊 Format Swarm State Information
     */
    static formatSwarmState(
        convoConfig: ChatConfigObject | undefined,
        teamConfig?: TeamConfigObject,
        config?: PromptServiceConfig,
    ): string {
        const cfg = this.getPromptConfig(config);
        let swarmStateOutput = "SWARM STATE DETAILS: Not available or error formatting state.";

        if (!convoConfig) {
            return "SWARM STATE DETAILS: Configuration not available.";
        }

        try {
            const teamId = convoConfig.teamId || "No team assigned";
            const swarmLeader = convoConfig.swarmLeader || "No leader assigned";
            const subtasks = convoConfig.subtasks || [];
            const subtaskLeaders = convoConfig.subtaskLeaders || {};
            const blackboard = convoConfig.blackboard || [];
            const resources = convoConfig.resources || [];
            const records = convoConfig.records || [];
            const stats = convoConfig.stats || {};
            const limits = convoConfig.limits || {};
            const pendingToolCalls = convoConfig.pendingToolCalls || [];

            const formattedSwarmStateParts: string[] = [];

            // Team information
            if (teamConfig) {
                const teamInfo = {
                    id: teamId,
                    deploymentType: teamConfig.deploymentType,
                    goal: teamConfig.goal,
                    businessPrompt: teamConfig.businessPrompt,
                    resourceQuota: teamConfig.resourceQuota,
                };
                formattedSwarmStateParts.push(`- Team Configuration:\n${this.truncateForPrompt(teamInfo, cfg.maxStringPreviewLength)}\n`);
            } else {
                formattedSwarmStateParts.push(`- Team ID:\n${this.truncateForPrompt(teamId, cfg.maxStringPreviewLength)}\n`);
            }

            formattedSwarmStateParts.push(`- Swarm Leader:\n${this.truncateForPrompt(swarmLeader, cfg.maxStringPreviewLength)}\n`);

            // Subtasks with counts
            const activeSubtasksCount = subtasks.filter(st =>
                typeof st === "object" && st !== null && (st.status === "todo" || st.status === "in_progress"),
            ).length;
            const completedSubtasksCount = subtasks.filter(st =>
                typeof st === "object" && st !== null && st.status === "done",
            ).length;
            formattedSwarmStateParts.push(`- Subtasks (active: ${activeSubtasksCount}, completed: ${completedSubtasksCount}):\n${this.truncateForPrompt(subtasks, cfg.maxStringPreviewLength)}\n`);

            // Other swarm state components
            formattedSwarmStateParts.push(`- Subtask Leaders:\n${this.truncateForPrompt(subtaskLeaders, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Blackboard:\n${this.truncateForPrompt(blackboard, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Resources:\n${this.truncateForPrompt(resources, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Records:\n${this.truncateForPrompt(records, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Stats:\n${this.truncateForPrompt(stats, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Limits:\n${this.truncateForPrompt(limits, cfg.maxStringPreviewLength)}\n`);
            formattedSwarmStateParts.push(`- Pending Tool Calls:\n${this.truncateForPrompt(pendingToolCalls, cfg.maxStringPreviewLength)}\n`);

            swarmStateOutput = `\nSWARM STATE DETAILS:\n${formattedSwarmStateParts.join("\n\n")}`;
        } catch (e) {
            logger.error("[ResponseService] Error formatting swarm state for prompt", { error: e });
        }

        return swarmStateOutput;
    }

    /**
     * 🔧 Build Tool Schemas String
     */
    private static buildToolSchemasString(toolRegistry?: ToolRegistry): string {
        if (!toolRegistry) {
            return "No tools available for this role.";
        }

        try {
            const builtInToolSchemas = toolRegistry.getBuiltInDefinitions();
            const swarmToolSchemas = toolRegistry.getSwarmToolDefinitions();
            const allToolSchemas = [...builtInToolSchemas, ...swarmToolSchemas];

            return allToolSchemas.length > 0
                ? JSON.stringify(allToolSchemas, null, 2)
                : "No tools available for this role.";
        } catch (error) {
            logger.error("[ResponseService] Error building tool schemas string", { error });
            return "Error loading tool schemas.";
        }
    }

    /**
     * ✂️ Smart Content Truncation
     */
    private static truncateForPrompt(value: unknown, maxLength: number): string {
        const limit = maxLength;

        if (value === undefined || value === null) {
            return "Not set";
        }

        const stringifiedValue = typeof value === "string" ? value : JSON.stringify(value, null, 2);

        if (stringifiedValue.length <= limit) {
            return stringifiedValue;
        }

        return stringifiedValue.substring(0, limit) + "...";
    }

    /**
     * 📂 Get Default Template Directory
     */
    private static getDefaultTemplateDirectory(): string {
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        return path.join(__dirname);
    }

    /**
     * Get merged prompt configuration
     */
    private static getPromptConfig(config?: PromptServiceConfig): Required<PromptServiceConfig> {
        return {
            ...DEFAULT_PROMPT_CONFIG,
            templateDirectory: config?.templateDirectory || this.getDefaultTemplateDirectory(),
            ...config,
        };
    }

    /**
     * 🧹 Clear Template Cache
     */
    static clearTemplateCache(): void {
        this.templateCache.clear();
        logger.debug("[ResponseService] Template cache cleared");
    }

    /**
     * 📊 Get Cache Statistics
     */
    static getCacheStats(): { size: number; enabled: boolean } {
        return {
            size: this.templateCache.size,
            enabled: DEFAULT_PROMPT_CONFIG.enableTemplateCache,
        };
    }
}
