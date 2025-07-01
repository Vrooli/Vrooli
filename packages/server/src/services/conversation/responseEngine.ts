/* eslint-disable func-style */
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { ChatConfig, DEFAULT_LANGUAGE, MINUTES_5_MS, McpSwarmToolName, McpToolName, MessageConfig, PendingToolCallStatus, SECONDS_1_MS, TeamConfig, generatePK, nanoid, type ChatConfigObject, type PendingToolCallEntry, type SessionUser, type TeamConfigObject, type ToolFunctionCall } from "@vrooli/shared";
import * as fs from "fs/promises";
import type OpenAI from "openai";
import * as path from "path";
import { fileURLToPath } from "url";
import { readOneHelper } from "../../actions/reads.js";
import type { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import { Notify } from "../../notify/notify.js";
import { SocketService } from "../../sockets/io.js"; // Still needed for roomHasOpenConnections check
import { type LLMCompletionTask } from "../../tasks/taskTypes.js";
import { BusService } from "../bus.js";
import { EventTypes, EventUtils, getUnifiedEventSystem, type IEventBus } from "../events/index.js";
import { CompositeGraph, type AgentGraph } from "../execution/tier1/agentGraph.js";
import { ToolRegistry } from "../mcp/registry.js";
import { SwarmTools } from "../mcp/tools.js";
import { type Tool } from "../mcp/types.js";
import { CachedConversationStateStore, PrismaChatStore, type ConversationStateStore } from "./chatStore.js";
import { RedisContextBuilder, type ContextBuilder } from "./contextBuilder.js";
import { RedisMessageStore, type MessageStore } from "./messageStore.js";
import type { FunctionCallStreamEvent } from "./router.js";
import { FallbackRouter, type LlmRouter } from "./router.js";
import { CompositeToolRunner, McpToolRunner, type ToolRunner } from "./toolRunner.js";
import { type BotParticipant, type ConversationState, type MessageState, type ResponseStats, type SwarmEvent, type SwarmStartedEvent } from "./types.js";

/**
 * Standardized error codes for response engine failures
 */
enum ResponseErrorCode {
    /** Generic cancellation (usually user-initiated) */
    CANCELLED = "CANCELLED",
    /** Operation timed out */
    TIMED_OUT = "TIMED_OUT",
    /** Tool execution failed */
    TOOL_EXECUTION_ERROR = "TOOL_EXECUTION_ERROR",
    /** Rate limited by external service */
    RATE_LIMITED = "RATE_LIMITED",
    /** Timeout during tool execution */
    TIMEOUT = "TIMEOUT",
    /** Network connectivity issues */
    NETWORK_ERROR = "NETWORK_ERROR",
    /** External service unavailable */
    SERVICE_UNAVAILABLE = "SERVICE_UNAVAILABLE",
    /** Temporary failure that may resolve */
    TEMPORARY_FAILURE = "TEMPORARY_FAILURE",
}

//TODO handle message updates (branching conversation)
// TODO: Fully implement backend execution for SCHEDULED_FOR_EXECUTION tool calls (e.g., via a job queue). Also, evaluate if general "resource mutations" beyond tool calls need a similar scheduling mechanism.
// TODO swarm world model should suggest starting a metacognition loop to best complete the goal, and adding to to the chat context. Can search for existing prompts.
//TODO conversation context should be able to store recommended routines to call for certain scenarios, pulled from the team config if available.
//TODO make sure all SwarmEvents are being used

const DEFAULT_GOAL = "Process current event.";

const VROOLI_WELCOME_MESSAGE = "Welcome to Vrooli, a polymorphic, collaborative, and self-improving automation platform that helps you stay organized and achieve your goals.\\n\\n";

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
# Vrooli ‣ ResponseEngine   

## PURPOSE  
Central execution engine for AI response generation – one instance per Node.js process.  

This class is a low-level execution engine meant to supplement the following high-level engines:  
  • ConversationService - Chatting with one or more bots directly ChatGPT-style (stateless)  
  • SwarmStateMachine - Autonomous agent swarm (group of bots) working together to achieve a common goal (stateful)  
  • RoutineStateMachine - BPMN-like workflow orchestration for completing a specific task (stateful)  

The high level engines are responsible for:  
  • Managing state (e.g. where we are in a routine)  
  • Building the conversation context (available bots and tools, chat history, routine context, etc.)  
  • Deciding when to call the ResponseEngine  
  • Handling the ResponseEngine's output  
  • Data persistence
  • Limiting the number of concurrent conversations (typically through a semaphore)

Why this setup? Think of each event process like a mini transaction:
  • INPUT  –  An event (user message, webhook etc.) is sent by the EventBus to an available worker 
              (high-level engine), which directly calls the ResponseEngine.
  • WORK   –  Picks responders, streams LLM responses, executes tool calls,
              enqueue any follow‑up actions **inside the same event loop** 
              until the bot emits a done event or a limit is reached.
  • OUTPUT –  Sends results to higher-level services, which may create outgoing bus events or emit websocket events to clients.

This structure gives Vrooli strict control over: per-event credit/tool budgets, 
play‑nice horizontal scaling, and—critically— a clear *handoff point* to a higher-level reasoning engine.

» MAJOR COLLABORATORS  
  ────────────────────  
  • **ContextBuilder**         – prompt window builder (combines messages + world‑model + tool schemas).
  • **LlmRouter**              – smart multi‑provider stream wrapper (OpenAI, Anthropic…).
  • **ToolRunner**             – executes MCP and AI-service tool calls, returns cost + output.

» EVENT SYSTEM  
  ────────────────────    
  Events are how services are triggered. 
  We use an event bus because it allows us to decouple services from each other, 
  easily plug in different event buses, and scale horizontally.  

  Outbound events:  
  • **credit:cost_incurred**       – Stores the cost incurred for response generation/tool use during event processing.  
*/
export class ReasoningEngine {
    private readonly unifiedEventBus: IEventBus | null;

    /**
     * @param contextBuilder     Collects all chat history that can fit into the context window of a bot turn.
     * @param llmRouter          Streams Responses‑API calls to chosen LLM provider (e.g. OpenAI, Anthropic).
     * @param toolRunner         Executes MCP tool calls (performs the actual work of a tool).
     */
    constructor(
        private readonly contextBuilder: ContextBuilder,
        private readonly llmRouter: LlmRouter,
        public readonly toolRunner: ToolRunner,
    ) {
        this.unifiedEventBus = getUnifiedEventSystem();
    }

    /**
     * Handles abort or timeout scenarios by emitting appropriate events and returning an error.
     */
    private _handleAbortOrTimeout(
        signalReason: string | Error | undefined | null,
        context: {
            botId: string;
            chatId?: string;
            stage: string; // e.g., "before-loop", "in-loop", "before-llm", "before-tool-call"
            toolName?: string; // Relevant for tool call stage
        },
    ): Error { // Returns the error to be thrown
        let errorMessage = "Response cancelled";
        let errorCode = ResponseErrorCode.CANCELLED;
        let logMessage = `Response cancelled by signal at stage '${context.stage}' for bot ${context.botId}`;
        if (context.chatId) logMessage += ` in chat ${context.chatId}`;
        if (context.toolName) logMessage += ` for tool ${context.toolName}`;

        let errorToThrow: Error;

        if (signalReason === "TurnTimeout") {
            errorMessage = context.toolName ? `Tool call ${context.toolName} aborted due to turn timeout` : "Bot turn timed out";
            errorCode = ResponseErrorCode.TIMED_OUT;
            logMessage = `${context.toolName ? `Tool call ${context.toolName}` : "Bot turn"} timed out at stage '${context.stage}' for bot ${context.botId}`;
            if (context.chatId) logMessage += ` in chat ${context.chatId}`;
            errorToThrow = new Error("ResponseTimeoutError");
        } else if (signalReason instanceof Error) {
            errorMessage = signalReason.message;
            errorToThrow = signalReason;
        } else if (typeof signalReason === "string" && signalReason.length > 0) {
            errorMessage = signalReason;
            errorToThrow = new Error(signalReason);
        } else {
            errorToThrow = new Error("ResponseCancelledError");
        }

        logger.info(logMessage + (signalReason ? `. Reason: ${String(signalReason)}` : ""));

        // Fire-and-forget error events - don't await to avoid blocking
        this._emitErrorEvents(context.chatId, context.botId, context.stage, errorMessage, errorCode, context.toolName).catch(err => {
            logger.error("Failed to emit error events", { error: err });
        });
        return errorToThrow;
    }

    /**
     * Emits appropriate error events based on the stage and context.
     */
    private async _emitErrorEvents(
        chatId: string | undefined,
        botId: string,
        stage: string,
        errorMessage: string,
        errorCode: ResponseErrorCode,
        toolName?: string,
    ): Promise<void> {
        if (!chatId || !this.unifiedEventBus) return;

        // Determine if the error is retryable based on error code and context
        const isRetryable = this._isErrorRetryable(errorCode, stage, toolName);

        const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
        const metadata = EventUtils.createEventMetadata("fire-and-forget", "high", {
            conversationId: chatId,
        });

        if (stage === "before-tool-call" || stage === "tool-call-processing") {
            // Specific for tool failures
            await this.unifiedEventBus.publish(
                EventUtils.createBaseEvent(
                    EventTypes.BOT_STATUS_UPDATED,
                    {
                        chatId,
                        botId,
                        status: "tool_failed",
                        toolInfo: { callId: "unknown", name: toolName || "unknown_tool", error: errorMessage },
                        error: { message: errorMessage, code: errorCode, retryable: isRetryable },
                    },
                    eventSource,
                    metadata,
                ),
            );
        } else {
            // General response stream error
            await this.unifiedEventBus.publish(
                EventUtils.createBaseEvent(
                    EventTypes.BOT_RESPONSE_STREAM,
                    {
                        __type: "error",
                        chatId,
                        botId,
                        error: { message: errorMessage, code: errorCode, retryable: isRetryable },
                    },
                    eventSource,
                    metadata,
                ),
            );

            // Bot status error
            await this.unifiedEventBus.publish(
                EventUtils.createBaseEvent(
                    EventTypes.BOT_STATUS_UPDATED,
                    {
                        chatId,
                        botId,
                        status: "error_internal",
                        error: { message: errorMessage, code: errorCode, retryable: isRetryable },
                    },
                    eventSource,
                    metadata,
                ),
            );
        }
    }

    /**
     * Determines if an error is retryable based on the error code, stage, and context.
     */
    private _isErrorRetryable(errorCode: ResponseErrorCode, stage: string, _toolName?: string): boolean {
        // Timeout errors are generally retryable as they indicate transient issues
        if (errorCode === ResponseErrorCode.TIMED_OUT) {
            return true;
        }

        // Generic cancellation is typically not retryable (user-initiated)
        if (errorCode === ResponseErrorCode.CANCELLED) {
            return false;
        }

        // For tool-related stages, some errors might be retryable
        if (stage === "before-tool-call" || stage === "tool-call-processing") {
            // Tool-specific retry logic could be added here in the future
            // For now, assume tool errors are not retryable unless it's a timeout
            return false;
        }

        // Internal errors during LLM calls might be retryable in some cases
        // This is conservative - only timeout-related issues are marked as retryable
        return false;
    }

    /**
     * Prepares tools for LLM by transforming them to OpenAI format.
     */
    private _prepareToolsForLlm(availableTools: Tool[]): OpenAI.Responses.Tool[] {
        // Create sets for efficient lookup of Vrooli custom tool names
        const mcpToolNames = new Set(Object.values(McpToolName));
        const mcpSwarmToolNames = new Set(Object.values(McpSwarmToolName));

        return availableTools.reduce((acc, ts) => {
            if (mcpToolNames.has(ts.name as McpToolName) || mcpSwarmToolNames.has(ts.name as McpSwarmToolName)) {
                acc.push({
                    type: "function",
                    name: ts.name,
                    description: ts.description || "",
                    parameters: ts.inputSchema as Record<string, unknown>,
                    strict: true,
                });
            } else if (ts.name === "web_search") {
                acc.push({ type: "web_search_preview" });
            } else if (ts.name === "file_search") {
                acc.push({ type: "file_search", vector_store_ids: [] });
            } else {
                logger.warn(`Unknown tool type encountered in toolsForLlm mapping: ${ts.name}. Skipping this tool.`);
                // Do not throw an error, just skip adding the tool.
            }
            return acc;
        }, [] as OpenAI.Responses.Tool[]);
    }

    /**
     * Prepares input messages for the reasoning loop.
     */
    private async _prepareInputMessages(
        startMessage: { id: string } | { text: string },
        chatId: string | undefined,
        bot: BotParticipant,
        model: string,
        systemMessageContent: string,
        toolsForLlm: OpenAI.Responses.Tool[],
    ): Promise<MessageState[]> {
        if (chatId && "id" in startMessage) {
            // Build context, reserving tokens for tools (none for now) and world-model
            const { messages } = await this.contextBuilder.build(
                chatId,
                bot,
                model,
                startMessage.id,
                { tools: toolsForLlm, systemMessage: systemMessageContent },
            );
            return messages;
        } else if ("text" in startMessage) {
            // Standalone prompt
            const tmp: MessageState = {
                id: generatePK().toString(),
                createdAt: new Date(),
                config: MessageConfig.default().export(),
                language: DEFAULT_LANGUAGE,
                text: startMessage.text,
                parent: null,
                user: { id: bot.id },
            } as MessageState;
            return [tmp];
        } else {
            throw new Error("Invalid startMessage for runLoop");
        }
    }

    /**
     * Checks if the response limits have been exceeded.
     */
    private _hasExceededLimits(
        responseStats: ResponseStats,
        convoAllocatedToolCalls: number,
        convoAllocatedCredits: bigint,
        responseLimits: Required<NonNullable<ChatConfigObject["limits"]>>,
    ): boolean {
        // Check against overall allocation for this runLoop instance
        const overallBudgetExceeded =
            (convoAllocatedToolCalls <= 0 && responseStats.toolCalls > 0) || // No tool calls allowed if allocation is zero or less
            (responseStats.toolCalls >= convoAllocatedToolCalls && convoAllocatedToolCalls > 0) || // Exceeded allocated tool calls
            (convoAllocatedCredits <= BigInt(0) && responseStats.creditsUsed > BigInt(0)) || // No credits allowed if allocation is zero or less
            (responseStats.creditsUsed >= convoAllocatedCredits && convoAllocatedCredits > BigInt(0)); // Exceeded allocated credits

        // Check if the CUMULATIVE stats for this bot's entire runLoop execution
        // have breached the per-bot-response limits.
        const perBotResponseLimitsBreached =
            responseStats.toolCalls >= responseLimits.maxToolCallsPerBotResponse ||
            responseStats.creditsUsed >= BigInt(responseLimits.maxCreditsPerBotResponse);

        return overallBudgetExceeded || perBotResponseLimitsBreached;
    }

    /**
     * Calculates the effective max credits for an LLM call.
     */
    private _calculateEffectiveMaxCredits(
        responseStats: ResponseStats,
        convoAllocatedCredits: bigint,
        responseLimits: Required<NonNullable<ChatConfigObject["limits"]>>,
    ): bigint {
        const remainingAllocatedCreditsForRun = convoAllocatedCredits - responseStats.creditsUsed;
        const capFromResponseLimits = BigInt(responseLimits.maxCreditsPerBotResponse);

        if (remainingAllocatedCreditsForRun <= BigInt(0)) {
            return BigInt(0); // No budget left in allocation
        } else {
            // Use the smaller of the two positive budget values
            return capFromResponseLimits < remainingAllocatedCreditsForRun ? capFromResponseLimits : remainingAllocatedCreditsForRun;
        }
    }

    /**
     * Processes a stream event and updates the loop state accordingly.
     */
    private async _processStreamEvent(
        ev: any,
        chatId: string | undefined,
        bot: BotParticipant,
        availableTools: Tool[],
        draftMessage: string,
        toolCalls: ToolFunctionCall[],
        responseStats: ResponseStats,
        abortSignal: AbortSignal | undefined,
        config: ChatConfigObject,
        userData: SessionUser,
    ): Promise<{ draftMessage: string; nextInputs: MessageState[]; responseStats: ResponseStats }> {
        const nextInputs: MessageState[] = [];
        let updatedDraftMessage = draftMessage;

        switch (ev.type) {
            case "message":
                // Append to draft message
                updatedDraftMessage = ev.final ? ev.content : draftMessage + ev.content;
                // Emit to client
                if (chatId && this.unifiedEventBus) {
                    const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                    const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                        conversationId: chatId,
                    });

                    this.unifiedEventBus.publish(
                        EventUtils.createBaseEvent(
                            EventTypes.BOT_RESPONSE_STREAM,
                            {
                                __type: ev.final ? "end" : "stream",
                                chatId,
                                botId: bot.id,
                                chunk: ev.content,
                            },
                            eventSource,
                            metadata,
                        ),
                    ).catch(err => logger.error("Failed to emit response stream event", { error: err }));
                }
                break;

            case "reasoning":
                // Emit to client
                if (chatId && this.unifiedEventBus) {
                    const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                    const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                        conversationId: chatId,
                    });

                    this.unifiedEventBus.publish(
                        EventUtils.createBaseEvent(
                            EventTypes.BOT_MODEL_REASONING_STREAM,
                            {
                                __type: "stream",
                                chatId,
                                botId: bot.id,
                                chunk: ev.content,
                            },
                            eventSource,
                            metadata,
                        ),
                    ).catch(err => logger.error("Failed to emit reasoning stream event", { error: err }));
                }
                break;

            case "function_call": {
                // Check for cancellation before tool call
                if (abortSignal?.aborted) {
                    throw this._handleAbortOrTimeout(abortSignal.reason, {
                        botId: bot.id, chatId, stage: "before-tool-call", toolName: ev.name,
                    });
                }

                // Execute the tool call (sync or async) and record result
                const { toolResult, functionCallEntry, cost } = await this.processFunctionCall(
                    chatId,
                    bot,
                    ev as FunctionCallStreamEvent,
                    availableTools, // Pass full Tool objects down
                    abortSignal,
                    config,
                    userData,
                );

                // Update cumulative stats for the entire runLoop after tool call
                responseStats.toolCalls++;
                responseStats.creditsUsed += cost;

                // Format toolResult (the raw output) into a MessageState for the LLM
                const toolResponseMessage: MessageState = {
                    id: generatePK().toString(), // New ID for this tool response message
                    createdAt: new Date(),
                    config: {
                        role: "tool",
                        toolCallId: ev.callId, // Link to the original function call request ID
                        __version: "1.0.0", // Example, ensure this aligns with MessageConfig
                    },
                    text: JSON.stringify(toolResult), // Content is the stringified output of the tool
                    language: DEFAULT_LANGUAGE, // Or derive from context
                    parent: ev.responseId ? { id: ev.responseId } : null,
                    user: { id: bot.id }, // Or a generic system/tool user ID
                } as MessageState; // Cast to MessageState

                nextInputs.push(toolResponseMessage);
                toolCalls.push(functionCallEntry);
                break;
            }

            case "done": {
                const cost = BigInt(ev.cost);
                responseStats.creditsUsed += cost; // Add LLM generation cost to cumulative response stats
                // Emit to client
                if (chatId && this.unifiedEventBus) {
                    const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                    const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                        conversationId: chatId,
                    });

                    this.unifiedEventBus.publish(
                        EventUtils.createBaseEvent(
                            EventTypes.BOT_RESPONSE_STREAM,
                            {
                                __type: "end",
                                chatId,
                                botId: bot.id,
                                finalMessage: updatedDraftMessage,
                            },
                            eventSource,
                            metadata,
                        ),
                    ).catch(err => logger.error("Failed to emit response end event", { error: err }));
                }
                break;
            }
        }

        return { draftMessage: updatedDraftMessage, nextInputs, responseStats };
    }

    /**
     * Finalizes the response by building the final message state and updating stats.
     */
    private _finalizeResponse(
        draftMessage: string,
        toolCalls: ToolFunctionCall[],
        inputs: MessageState[],
        previousResponseId: string | undefined,
        config: ChatConfigObject,
    ): MessageState {
        // Build and return the final MessageState for the bot response.
        // Generate message-level configuration with default values and set role to 'assistant'.
        const msgConfig = MessageConfig.default();
        msgConfig.setRole("assistant");
        // Attach the tool calls recorded during execution
        msgConfig.setToolCalls(toolCalls);
        const configObj = msgConfig.export();
        const language = inputs.length > 0 ? inputs[inputs.length - 1].language : DEFAULT_LANGUAGE;

        // Construct the MessageState
        const responseMessage: MessageState = {
            id: generatePK().toString(),
            createdAt: new Date(),
            config: configObj,
            language,
            text: draftMessage,
            parent: previousResponseId ? { id: previousResponseId } : null,
            user: { id: "bot" }, // This will be overridden by the caller
        };

        // Mark turn completion time
        if (!config.stats) {
            config.stats = ChatConfig.defaultStats();
        }
        config.stats.lastProcessingCycleEndedAt = Date.now();

        return responseMessage;
    }

    /**
     * Executes a single reasoning loop for a bot.
     * @param startMessage - Either an object with {id} to start from existing history, or {text} for standalone prompts
     * @param systemMessageContent - The system message content to use for the context window
     * @param availableTools - The tools available to the bot
     * @param bot - Information about the bot that's running the loop
     * @param creditAccountId - The credit account to charge for the response (tied to a user or team)
     * @param config - The conversation configuration
     * @param convoAllocatedToolCalls - Max conversation-level tool calls allocated to this run
     * @param convoAllocatedCredits - Max conversation-level credits allocated to this run
     * @param userData - User session data, including credits.
     * @param chatId - The chat this loop is for, if applicable
     * @param model - The model to use for the LLM call
     * @param abortSignal - Optional AbortSignal to check for cancellation
     * @returns A MessageState object representing the final message from the bot and the stats for this response generation.
     */
    async runLoop(
        startMessage: { id: string } | { text: string },
        systemMessageContent: string,
        availableTools: Tool[],
        bot: BotParticipant,
        creditAccountId: string,
        config: ChatConfigObject,
        convoAllocatedToolCalls: number,
        convoAllocatedCredits: bigint,
        userData: SessionUser,
        chatId?: string,
        model?: string,
        abortSignal?: AbortSignal,
    ): Promise<{ finalMessage: MessageState; responseStats: ResponseStats }> {
        // Signal typing start
        if (chatId && this.unifiedEventBus) {
            const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
            const metadata = EventUtils.createEventMetadata("fire-and-forget", "low", {
                conversationId: chatId,
            });

            this.unifiedEventBus.publish(
                EventUtils.createBaseEvent(
                    EventTypes.BOT_TYPING_UPDATED,
                    {
                        chatId,
                        starting: [bot.id],
                    },
                    eventSource,
                    metadata,
                ),
            ).catch(err => logger.error("Failed to emit typing start event", { error: err }));
        }

        // Transform availableTools to OpenAI.Responses.Tool[] format for ContextBuilder and LlmRouter
        const toolsForLlm = this._prepareToolsForLlm(availableTools);

        // Check for cancellation at the very beginning
        if (abortSignal?.aborted) {
            throw this._handleAbortOrTimeout(abortSignal.reason, {
                botId: bot.id, chatId, stage: "before-loop",
            });
        }

        // Build context if we have a chatId + message ID, else wrap a text prompt
        let inputs = await this._prepareInputMessages(
            startMessage,
            chatId,
            bot,
            model ?? DEFAULT_LANGUAGE,
            systemMessageContent,
            toolsForLlm,
        );

        let draftMessage = "";
        // Collect all tool calls made during this loop
        const toolCalls: ToolFunctionCall[] = [];
        let previousResponseId: string | undefined;
        const responseStats: ResponseStats = { toolCalls: 0, creditsUsed: BigInt(0) };

        // Compute per-bot-response limits from the overall conversation config.
        const responseLimits = new ChatConfig({ config }).getEffectiveLimits();

        // Prepare the final system message for the LLM call
        const finalSystemMessageForLlm = systemMessageContent;

        try {
            // Emit thinking status before starting the main loop
            if (chatId && this.unifiedEventBus) {
                const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                    conversationId: chatId,
                });

                this.unifiedEventBus.publish(
                    EventUtils.createBaseEvent(
                        EventTypes.BOT_STATUS_UPDATED,
                        {
                            chatId,
                            botId: bot.id,
                            status: "thinking",
                            message: "Processing request...",
                        },
                        eventSource,
                        metadata,
                    ),
                ).catch(err => logger.error("Failed to emit thinking status event", { error: err }));
            }

            while (inputs.length && !this._hasExceededLimits(responseStats, convoAllocatedToolCalls, convoAllocatedCredits, responseLimits)) {
                // Check for cancellation at the beginning of each iteration
                if (abortSignal?.aborted) {
                    throw this._handleAbortOrTimeout(abortSignal.reason, {
                        botId: bot.id, chatId, stage: "in-loop",
                    });
                }

                const chosenModel = model ?? config.preferredModel ?? bot.config?.model ?? "";

                // Calculate effective max credits for the upcoming LLM call
                const effectiveMaxCreditsForLlm = this._calculateEffectiveMaxCredits(
                    responseStats,
                    convoAllocatedCredits,
                    responseLimits,
                );

                // Check for cancellation before LLM call
                if (abortSignal?.aborted) {
                    throw this._handleAbortOrTimeout(abortSignal.reason, {
                        botId: bot.id, chatId, stage: "before-llm-call",
                    });
                }

                const stream = this.llmRouter.stream({
                    model: chosenModel,
                    previous_response_id: previousResponseId,
                    input: inputs,
                    tools: toolsForLlm,
                    parallel_tool_calls: true,
                    systemMessage: finalSystemMessageForLlm,
                    userData,
                    maxCredits: effectiveMaxCreditsForLlm,
                    signal: abortSignal,
                });

                const nextInputs: MessageState[] = [];
                for await (const ev of stream) {
                    const result = await this._processStreamEvent(
                        ev, chatId, bot, availableTools, draftMessage, toolCalls,
                        responseStats, abortSignal, config, userData,
                    );

                    draftMessage = result.draftMessage;
                    nextInputs.push(...result.nextInputs);
                    Object.assign(responseStats, result.responseStats);
                    previousResponseId = ev.responseId;
                }

                // Prepare for next iteration
                inputs = nextInputs;
            }

            // Emit processing_complete status if the loop finished without exhausting inputs
            if (chatId && inputs.length === 0 && this.unifiedEventBus) {
                const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                    conversationId: chatId,
                });

                this.unifiedEventBus.publish(
                    EventUtils.createBaseEvent(
                        EventTypes.BOT_STATUS_UPDATED,
                        {
                            chatId,
                            botId: bot.id,
                            status: "processing_complete",
                            message: "Finished processing turn.",
                        },
                        eventSource,
                        metadata,
                    ),
                ).catch(err => logger.error("Failed to emit processing complete event", { error: err }));
            }

        } finally {
            // Send cost incurred event 
            // Calculate which source of credits will be consumed
            const { calculateFreeCreditsBalance, getConsumedCreditSource } = await import("../billing/creditBalanceService.js");
            const freeBalance = await calculateFreeCreditsBalance(BigInt(creditAccountId));
            const consumedSource = getConsumedCreditSource(freeBalance, BigInt(responseStats.creditsUsed));

            await BusService.get().getBus().publish({
                type: "billing:event",
                id: `${bot.id}-${chatId}-${responseStats.creditsUsed}`,
                accountId: creditAccountId,
                delta: (-responseStats.creditsUsed).toString(), // MAKE SURE THIS IS NEGATIVE
                entryType: CreditEntryType.Spend,
                source: CreditSourceSystem.InternalAgent,
                meta: { // Additional metadata that might be useful to filter events later
                    chatId,
                    botId: bot.id,
                    consumedCreditSource: consumedSource,
                    freeCreditsBeforeSpend: freeBalance.toString(),
                },
            });

            // Signal typing end
            if (chatId && this.unifiedEventBus) {
                const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                const metadata = EventUtils.createEventMetadata("fire-and-forget", "low", {
                    conversationId: chatId,
                });

                this.unifiedEventBus.publish(
                    EventUtils.createBaseEvent(
                        EventTypes.BOT_TYPING_UPDATED,
                        {
                            chatId,
                            stopping: [bot.id],
                        },
                        eventSource,
                        metadata,
                    ),
                ).catch(err => logger.error("Failed to emit typing stop event", { error: err }));
            }
        }

        const finalMessage = this._finalizeResponse(draftMessage, toolCalls, inputs, previousResponseId, config);
        finalMessage.user = { id: bot.id }; // Set the correct bot ID

        return { finalMessage, responseStats };
    }

    // --------------------------------------------------------------------------
    // Helper: process a function_call event, handling sync vs async tools
    private async processFunctionCall(
        conversationId: string | undefined,
        callerBot: BotParticipant,
        ev: FunctionCallStreamEvent,
        availableTools: Tool[],
        abortSignal?: AbortSignal,
        config?: ChatConfigObject,
        userData?: SessionUser,
    ): Promise<{ toolResult: unknown; functionCallEntry: ToolFunctionCall; cost: bigint }> {
        // Check for cancellation at the start of tool processing
        if (abortSignal?.aborted) {
            throw this._handleAbortOrTimeout(abortSignal.reason, {
                botId: callerBot.id,
                chatId: conversationId,
                stage: "tool-call-processing",
                toolName: ev.name,
            });
        }

        const toolName = ev.name;
        const schedulingDecision = this._determineToolScheduling(toolName, config);

        if (schedulingDecision.requiresApproval || schedulingDecision.requiresSchedulingOnly) {
            return await this._handleDeferredToolCall(
                ev, callerBot, conversationId, schedulingDecision,
                config, userData, availableTools,
            );
        }

        // If not deferred (or fell through scheduling logic), proceed with immediate execution
        return await this._executeToolImmediately(
            ev, callerBot, conversationId, availableTools, abortSignal,
        );
    }

    /**
     * Determines if a tool call requires approval or scheduling.
     */
    private _determineToolScheduling(
        toolName: string,
        config?: ChatConfigObject,
    ): {
        requiresApproval: boolean;
        requiresSchedulingOnly: boolean;
        specificDelayMs?: number;
        schedulingRules?: any;
    } {
        // Use getEffectiveScheduling to ensure validated and default-applied rules are used.
        const effectiveScheduling = config ? new ChatConfig({ config }).getEffectiveScheduling() : ChatConfig.defaultScheduling();
        const schedulingRules = effectiveScheduling;

        let requiresApproval = false;
        let requiresSchedulingOnly = false;
        let specificDelayMs: number | undefined;

        if (schedulingRules) {
            if (Array.isArray(schedulingRules.requiresApprovalTools) && schedulingRules.requiresApprovalTools.includes(toolName)) {
                requiresApproval = true;
            } else if (schedulingRules.requiresApprovalTools === "all") {
                requiresApproval = true;
            }

            // A tool requiring approval is inherently "scheduled" (pending approval first)
            // If not requiring approval, it might still be scheduled with a simple delay.
            if (!requiresApproval) {
                specificDelayMs = schedulingRules.toolSpecificDelays?.[toolName] ?? schedulingRules.defaultDelayMs;
                if (specificDelayMs && specificDelayMs > 0) {
                    requiresSchedulingOnly = true;
                }
            }
        } else {
            if (config) {
                logger.warn(`Scheduling rules are undefined for tool call ${toolName}, but ChatConfigObject was provided. Tool will execute immediately if not requiring approval otherwise.`, { toolName });
            }
        }

        return {
            requiresApproval,
            requiresSchedulingOnly,
            specificDelayMs,
            schedulingRules,
        };
    }

    /**
     * Handles deferred tool calls that require approval or scheduling.
     */
    private async _handleDeferredToolCall(
        ev: FunctionCallStreamEvent,
        callerBot: BotParticipant,
        conversationId: string | undefined,
        schedulingDecision: {
            requiresApproval: boolean;
            requiresSchedulingOnly: boolean;
            specificDelayMs?: number;
            schedulingRules?: any;
        },
        config?: ChatConfigObject,
        userData?: SessionUser,
        availableTools?: Tool[],
    ): Promise<{ toolResult: unknown; functionCallEntry: ToolFunctionCall; cost: bigint }> {
        const pendingId = nanoid();
        const now = Date.now();
        let currentStatus: PendingToolCallStatus | null = null;
        let scheduledExecutionTime: number | undefined;
        let approvalTimeoutTimestamp: number | undefined;

        if (schedulingDecision.requiresApproval) {
            currentStatus = PendingToolCallStatus.PENDING_APPROVAL;
            approvalTimeoutTimestamp = now + (schedulingDecision.schedulingRules?.approvalTimeoutMs || MINUTES_5_MS);
        } else if (schedulingDecision.requiresSchedulingOnly && schedulingDecision.specificDelayMs) {
            currentStatus = PendingToolCallStatus.SCHEDULED_FOR_EXECUTION;
            scheduledExecutionTime = now + schedulingDecision.specificDelayMs;
            // TODO-SCHEDULED-TOOL: Implement job queue integration (e.g., BullMQ) for SCHEDULED_FOR_EXECUTION.
        } else {
            logger.warn(`Tool call ${ev.name} for bot ${callerBot.id} entered scheduling block without clear status. Defaulting to immediate execution.`, { ev });
        }

        if (currentStatus === PendingToolCallStatus.PENDING_APPROVAL || currentStatus === PendingToolCallStatus.SCHEDULED_FOR_EXECUTION) {
            // Create a PendingToolCallEntry object
            const pendingEntryForConfig: PendingToolCallEntry = {
                pendingId,
                toolCallId: ev.callId,
                toolName: ev.name,
                toolArguments: JSON.stringify(ev.arguments),
                callerBotId: callerBot.id,
                conversationId: conversationId || "unknown_conversation",
                requestedAt: now,
                status: currentStatus,
                scheduledExecutionTime,
                approvalTimeoutAt: approvalTimeoutTimestamp,
                userIdToApprove: userData?.id,
                executionAttempts: 0,
            };

            // Store the PendingToolCallEntry in ChatConfigObject
            if (config) {
                config.pendingToolCalls = config.pendingToolCalls || [];
                config.pendingToolCalls.push(pendingEntryForConfig);
                logger.info(`Tool call ${ev.name} (${ev.callId}) for bot ${callerBot.id} deferred and added to ChatConfig. Status: ${pendingEntryForConfig.status}`, { pendingEntry: pendingEntryForConfig });
            } else {
                logger.error(`Cannot defer tool call ${ev.name} (${ev.callId}) for bot ${callerBot.id}: ChatConfigObject is undefined.`);
            }

            // Emit socket events if approval is needed
            if (currentStatus === PendingToolCallStatus.PENDING_APPROVAL && conversationId && userData) {
                await this._emitToolApprovalEvents(
                    conversationId, callerBot, ev, pendingId,
                    approvalTimeoutTimestamp, userData, availableTools,
                    schedulingDecision.schedulingRules,
                );
            }

            // Return a placeholder result to the LLM indicating deferral
            const deferredToolResult = {
                __vrooli_tool_deferred: true,
                pendingId,
                status: currentStatus,
                message: currentStatus === PendingToolCallStatus.PENDING_APPROVAL
                    ? `Tool call ${ev.name} is awaiting user approval. You will be notified of the outcome.`
                    : `Tool call ${ev.name} has been scheduled for later execution. You will be notified upon completion.`,
            };

            const functionCallEntryDeferred: ToolFunctionCall = {
                id: ev.callId,
                function: { name: ev.name, arguments: JSON.stringify(ev.arguments) },
                result: { success: true, output: JSON.stringify(deferredToolResult) },
            };

            return { toolResult: deferredToolResult, functionCallEntry: functionCallEntryDeferred, cost: BigInt(0) };
        }

        // Fallback to immediate execution if scheduling logic fails
        return await this._executeToolImmediately(ev, callerBot, conversationId, availableTools);
    }

    /**
     * Emits socket events for tool approval notifications.
     */
    private async _emitToolApprovalEvents(
        conversationId: string,
        callerBot: BotParticipant,
        ev: FunctionCallStreamEvent,
        pendingId: string,
        approvalTimeoutTimestamp: number | undefined,
        userData: SessionUser,
        availableTools?: Tool[],
        schedulingRules?: any,
    ): Promise<void> {
        if (!this.unifiedEventBus) return;

        const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
        const metadata = EventUtils.createEventMetadata("reliable", "high", {
            conversationId,
            userId: userData.id,
        });

        // Emit tool approval required event
        await this.unifiedEventBus.publish(
            EventUtils.createBaseEvent(
                EventTypes.TOOL_APPROVAL_REQUIRED,
                {
                    pendingId,
                    toolCallId: ev.callId,
                    toolName: ev.name,
                    toolArguments: ev.arguments as Record<string, any>,
                    callerBotId: callerBot.id,
                    callerBotName: callerBot.name,
                    approvalTimeoutAt: approvalTimeoutTimestamp,
                    estimatedCost: availableTools?.find(t => t.name === ev.name)?.estimatedCost,
                    conversationId,
                },
                eventSource,
                metadata,
            ),
        );

        // Check if we need to send push notification
        const hasActiveConnection = SocketService.get().roomHasOpenConnections(conversationId);
        if (!hasActiveConnection) {
            Notify(userData.languages)
                .pushToolApprovalRequired(
                    conversationId,
                    pendingId,
                    ev.name,
                    callerBot.name,
                    "your Vrooli task", // Generic fallback for conversation name
                    approvalTimeoutTimestamp,
                    availableTools?.find(t => t.name === ev.name)?.estimatedCost,
                    schedulingRules?.autoRejectOnTimeout,
                )
                .toUser(userData.id);
            logger.info(`Sent tool approval push notification for ${pendingId} to user ${userData.id} for conversation ${conversationId}.`);
        }

        // Emit bot status update
        await this.unifiedEventBus.publish(
            EventUtils.createBaseEvent(
                EventTypes.BOT_STATUS_UPDATED,
                {
                    chatId: conversationId,
                    botId: callerBot.id,
                    status: "tool_pending_approval",
                    toolInfo: { callId: ev.callId, name: ev.name, pendingId },
                    message: `Tool ${ev.name} is awaiting user approval.`,
                },
                eventSource,
                metadata,
            ),
        );
    }

    /**
     * Executes a tool call immediately (synchronously or asynchronously).
     */
    private async _executeToolImmediately(
        ev: FunctionCallStreamEvent,
        callerBot: BotParticipant,
        conversationId: string | undefined,
        availableTools?: Tool[],
        abortSignal?: AbortSignal,
    ): Promise<{ toolResult: unknown; functionCallEntry: ToolFunctionCall; cost: bigint }> {
        const args = ev.arguments as Record<string, any>;
        const isAsync = args.isAsync === true;
        const fnEntryBase = {
            id: ev.callId,
            function: { name: ev.name, arguments: JSON.stringify(ev.arguments) },
        } as Omit<ToolFunctionCall, "result">;
        const _toolCost = BigInt(0); // Unused but kept for potential future use

        // Emit tool_calling status BEFORE executing
        if (conversationId && this.unifiedEventBus) {
            const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
            const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                conversationId,
            });

            this.unifiedEventBus.publish(
                EventUtils.createBaseEvent(
                    EventTypes.BOT_STATUS_UPDATED,
                    {
                        chatId: conversationId,
                        botId: callerBot.id,
                        status: "tool_calling",
                        toolInfo: { callId: ev.callId, name: ev.name, args: JSON.stringify(ev.arguments) },
                        message: `Using tool: ${ev.name}`,
                    },
                    eventSource,
                    metadata,
                ),
            ).catch(err => logger.error("Failed to emit tool calling event", { error: err }));
        }

        const toolCallResponse = await this.toolRunner.run(ev.name, ev.arguments, {
            conversationId: conversationId || "",
            callerBotId: callerBot.id,
            signal: abortSignal,
        });

        return this._formatToolResponse(
            toolCallResponse, fnEntryBase, ev, callerBot,
            conversationId, isAsync,
        );
    }

    /**
     * Formats the tool response into the expected return format.
     */
    private _formatToolResponse(
        toolCallResponse: any,
        fnEntryBase: Omit<ToolFunctionCall, "result">,
        ev: FunctionCallStreamEvent,
        callerBot: BotParticipant,
        conversationId: string | undefined,
        _isAsync: boolean, // Unused but kept for potential future differentiation
    ): { toolResult: unknown; functionCallEntry: ToolFunctionCall; cost: bigint } {
        let toolCost = BigInt(0);
        let output: unknown = null;
        let entry: ToolFunctionCall;

        if (toolCallResponse.ok) {
            output = toolCallResponse.data.output;
            toolCost = BigInt(toolCallResponse.data.creditsUsed);
            entry = { ...fnEntryBase, result: { success: true, output } };

            if (conversationId && this.unifiedEventBus) {
                const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                    conversationId,
                });

                this.unifiedEventBus.publish(
                    EventUtils.createBaseEvent(
                        EventTypes.BOT_STATUS_UPDATED,
                        {
                            chatId: conversationId,
                            botId: callerBot.id,
                            status: "tool_completed",
                            toolInfo: { callId: ev.callId, name: ev.name, result: JSON.stringify(output) },
                        },
                        eventSource,
                        metadata,
                    ),
                ).catch(err => logger.error("Failed to emit tool completed event", { error: err }));
            }
        } else {
            // Tool call failed
            output = toolCallResponse.error;
            toolCost = toolCallResponse.error.creditsUsed ? BigInt(toolCallResponse.error.creditsUsed) : BigInt(0);
            entry = { ...fnEntryBase, result: { success: false, error: toolCallResponse.error } };

            // Determine if this tool error is retryable
            const isToolErrorRetryable = this._isToolErrorRetryable(toolCallResponse.error);

            if (conversationId && this.unifiedEventBus) {
                const eventSource = EventUtils.createEventSource("cross-cutting", "ReasoningEngine");
                const metadata = EventUtils.createEventMetadata("fire-and-forget", "medium", {
                    conversationId,
                });

                this.unifiedEventBus.publish(
                    EventUtils.createBaseEvent(
                        EventTypes.BOT_STATUS_UPDATED,
                        {
                            chatId: conversationId,
                            botId: callerBot.id,
                            status: "tool_failed",
                            toolInfo: { callId: ev.callId, name: ev.name, error: JSON.stringify(toolCallResponse.error) },
                            error: {
                                message: toolCallResponse.error.message || "Tool execution failed",
                                code: toolCallResponse.error.code || ResponseErrorCode.TOOL_EXECUTION_ERROR,
                                retryable: isToolErrorRetryable,
                            },
                        },
                        eventSource,
                        metadata,
                    ),
                ).catch(err => logger.error("Failed to emit tool failed event", { error: err }));
            }
        }

        return { toolResult: output, functionCallEntry: entry, cost: toolCost };
    }

    /**
     * Determines if a tool execution error is retryable based on the error details.
     */
    private _isToolErrorRetryable(error: any): boolean {
        if (!error) return false;

        const errorCode = error.code;
        const errorMessage = error.message || "";

        // Specific retryable error codes
        const retryableErrorCodes = [
            ResponseErrorCode.RATE_LIMITED,
            ResponseErrorCode.TIMEOUT,
            ResponseErrorCode.NETWORK_ERROR,
            ResponseErrorCode.SERVICE_UNAVAILABLE,
            ResponseErrorCode.TEMPORARY_FAILURE,
        ];

        if (retryableErrorCodes.includes(errorCode)) {
            return true;
        }

        // Check for retryable patterns in error messages (case-insensitive)
        const retryablePatterns = [
            /rate.?limit/i,
            /timeout/i,
            /temporary/i,
            /try.?again/i,
            /service.?unavailable/i,
            /502|503|504/i, // HTTP error codes
        ];

        return retryablePatterns.some(pattern => pattern.test(errorMessage));
    }
}

/**
 CompletionService
 -----------------------------------------------------------------------------
 High-level facade for conversational workflows. Delegates single-bot &
 multi-bot interactions to ReasoningEngine. Handles context building, 
 storing messages, updating the conversation state (e.g. active bot), 
 and limiting the number of concurrent completions for a given conversation.

 This class has two responsibilities:
 1. Responding to user messages (should be behind a queue in case of high load)
 2. Acting as a low-level engine for generating responses for the swarm state machine. 
 This gives the class similar responsibilities as the ReasoningEngine, but with 
 conversation-specific logic included (e.g. AgentGraph for picking responders).

 » MAJOR COLLABORATORS  
  ────────────────────  
  • **ReasoningEngine**         – low-level engine for generating responses.
  • **AgentGraph**              – strategy for picking responding bots.
  • **ConversationStateStore**  – efficient storage and retrieval of conversation state.
  • **MessageStore**            – efficient storage and retrieval of message state.

 */
export class CompletionService {
    private readonly activeControllers = new Map<string, AbortController>();
    private readonly unifiedEventBus: IEventBus | null;

    /**
     * @param reasoningEngine    Low-level engine for generating responses.
     * @param agentGraph         Strategy for picking responding bots (e.g. direct mentions, OpenAI Swarm).
     * @param conversationStore  Efficient storage and retrieval of conversation state
     * @param messageStore       Efficient storage and retrieval of message state
     * @param toolRegistry       Tool registry for managing tools and their definitions
     */
    constructor(
        private readonly reasoningEngine: ReasoningEngine,
        private readonly agentGraph: AgentGraph,
        private readonly conversationStore: ConversationStateStore,
        private readonly messageStore: MessageStore,
        private readonly toolRegistry: ToolRegistry,
    ) {
        this.unifiedEventBus = getUnifiedEventSystem();
    }

    public async getConversationState(conversationId: string): Promise<ConversationState | null> {
        return this.conversationStore.get(conversationId);
    }

    public updateConversationConfig(conversationId: string, config: ChatConfigObject): void {
        this.conversationStore.updateConfig(conversationId, config);
    }

    /**
     * Gets the reasoning engine for direct integration
     */
    public getReasoningEngine(): ReasoningEngine {
        return this.reasoningEngine;
    }

    /**
     * Gets the tool registry for tool resolution
     */
    public getToolRegistry(): ToolRegistry {
        return this.toolRegistry;
    }

    /**
     * Fetches and attaches team config to the conversation state if a teamId is present.
     * This method handles authorization by ensuring the user has access to the team.
     */
    public async attachTeamConfig(
        conversationState: ConversationState,
        initiatingUser: SessionUser,
    ): Promise<ConversationState> {
        if (!conversationState.config.teamId) {
            // No team assigned, return as-is
            return conversationState;
        }

        try {
            // Create a mock request object for the readOneHelper
            const mockReq: Parameters<typeof RequestService.assertRequestFrom>[0] = {
                session: {
                    fromSafeOrigin: true,
                    isLoggedIn: true,
                    languages: initiatingUser.languages ?? [DEFAULT_LANGUAGE],
                    userId: initiatingUser.id,
                    users: [initiatingUser],
                },
            };

            // Fetch the team using readOneHelper to ensure proper authorization
            const team = await readOneHelper({
                info: { id: true, config: true } as any, // Basic info we need
                input: { id: conversationState.config.teamId },
                objectType: "Team",
                req: mockReq,
            });

            if (team && team.config) {
                // Parse the team config
                const teamConfig = TeamConfig.parse({ config: team.config }, logger, { useFallbacks: true }).export();

                // Return updated conversation state with team config attached
                return {
                    ...conversationState,
                    teamConfig,
                };
            } else {
                logger.warn(`Team ${conversationState.config.teamId} not found or user ${initiatingUser.id} lacks access. Proceeding without team config.`);
                return conversationState;
            }
        } catch (error) {
            logger.error(`Failed to fetch team config for teamId ${conversationState.config.teamId}:`, error);
            // Return original state without team config rather than failing the entire operation
            return conversationState;
        }
    }

    private _truncateStringForPrompt(value: any, maxLength: number): string {
        if (value === undefined || value === null) {
            return "Not set";
        }
        const stringifiedValue = typeof value === "string" ? value : JSON.stringify(value, null, 2);
        if (stringifiedValue.length <= maxLength) {
            return stringifiedValue;
        }
        return stringifiedValue.substring(0, maxLength) + "...";
    }

    /**
     * Loads the role-specific prompt template from disk.
     */
    private async _loadPromptTemplate(): Promise<string> {
        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        const promptPath = path.join(__dirname, "prompt.txt");

        try {
            logger.info(`Loading base prompt template from: ${promptPath}`);
            return await fs.readFile(promptPath, "utf-8");
        } catch (error) {
            logger.error(`Failed to load prompt template from ${promptPath}:`, { error });
            return "Your primary goal is: {{GOAL}}. Please act according to your role: {{ROLE}}. Critical: Prompt template file not found.";
        }
    }

    /**
     * Processes template variables in the prompt.
     */
    private _processPromptTemplate(
        template: string,
        goal: string,
        bot: BotParticipant,
        convoConfig: ChatConfigObject,
        teamConfig?: TeamConfigObject,
    ): string {
        const botRole = bot.meta?.role || "leader"; // Default to leader if no role
        const botId = bot.id;

        let memberCountLabel = "1 member";
        if (convoConfig.teamId) {
            memberCountLabel = "team-based swarm";
        }

        const builtInToolSchemas = this.toolRegistry.getBuiltInDefinitions();
        const swarmToolSchemas = this.toolRegistry.getSwarmToolDefinitions();
        const allToolSchemas = [...builtInToolSchemas, ...swarmToolSchemas];

        let roleSpecificInstructions = "Perform tasks according to your role and the overall goal.";
        if (botRole === "leader" || botRole === "coordinator" || botRole === "delegator") {
            roleSpecificInstructions = RECRUITMENT_RULE_PROMPT;
        }

        const swarmStateString = this._buildSwarmStateString(convoConfig, teamConfig);
        const toolSchemasString = allToolSchemas.length > 0 ? JSON.stringify(allToolSchemas, null, 2) : "No tools available for this role.";

        let processedPrompt = template;
        processedPrompt = processedPrompt.replace(/{{GOAL}}/g, goal);
        processedPrompt = processedPrompt.replace(/{{MEMBER_COUNT_LABEL}}/g, memberCountLabel);
        processedPrompt = processedPrompt.replace(/{{ISO_EPOCH_SECONDS}}/g, Math.floor(Date.now() / SECONDS_1_MS).toString());
        processedPrompt = processedPrompt.replace(/{{DISPLAY_DATE}}/g, new Date().toLocaleString());
        processedPrompt = processedPrompt.replace(/{{ROLE\s*\|\s*upper}}/g, botRole.toUpperCase());
        processedPrompt = processedPrompt.replace(/{{ROLE}}/g, botRole);
        processedPrompt = processedPrompt.replace(/{{BOT_ID}}/g, botId);
        processedPrompt = processedPrompt.replace(/{{ROLE_SPECIFIC_INSTRUCTIONS}}/g, roleSpecificInstructions);
        processedPrompt = processedPrompt.replace(/{{SWARM_STATE}}/g, swarmStateString);
        processedPrompt = processedPrompt.replace(/{{TOOL_SCHEMAS}}/g, toolSchemasString);

        return processedPrompt;
    }

    /**
     * Builds the system message for a bot based on its role and the current context.
     */
    private async _buildSystemMessage(
        goal: string,
        bot: BotParticipant,
        convoConfig: ChatConfigObject,
        teamConfig?: TeamConfigObject,
    ): Promise<string> {
        const template = await this._loadPromptTemplate();
        const processedPrompt = this._processPromptTemplate(template, goal, bot, convoConfig, teamConfig);
        return VROOLI_WELCOME_MESSAGE + processedPrompt.trim();
    }

    /**
     * Builds a string representation of the current swarm state for inclusion in the system prompt.
     * @param convoConfig The conversation configuration object containing swarm state.
     * @param teamConfig Optional team configuration with organizational structure.
     * @returns A formatted string of the swarm state, or a default message if unavailable.
     */
    private _buildSwarmStateString(convoConfig: ChatConfigObject | undefined, teamConfig?: TeamConfigObject): string {
        /** How long each swarm state section can be before we truncate it. */
        const MAX_STRING_PREVIEW_LENGTH = 2_000;
        let swarmStateOutput = "SWARM STATE DETAILS: Not available or error formatting state.";

        if (!convoConfig) {
            return "SWARM STATE DETAILS: Configuration not available.";
        }

        try {
            const teamId = convoConfig.teamId || "No team assigned";
            const swarmLeader = convoConfig.swarmLeader || "No leader assigned";
            const subtasks = convoConfig.subtasks || [];
            const subtaskLeaders = convoConfig.subtaskLeaders || {};
            const eventSubscriptions = convoConfig.eventSubscriptions || {};
            const blackboard = convoConfig.blackboard || [];
            const resources = convoConfig.resources || [];
            const records = convoConfig.records || [];
            const stats = convoConfig.stats || {};
            const limits = convoConfig.limits || {};
            const pendingToolCalls = convoConfig.pendingToolCalls || [];

            const formattedSwarmStateParts: string[] = [];

            // Team information with organizational structure
            if (teamConfig) {
                const teamInfo = {
                    id: teamId,
                    structure: teamConfig.structure || { type: "Not specified", content: "No organizational structure defined" },
                };
                formattedSwarmStateParts.push(`- Team Configuration:\n${this._truncateStringForPrompt(teamInfo, MAX_STRING_PREVIEW_LENGTH)}\n`);
            } else {
                formattedSwarmStateParts.push(`- Team ID:\n${this._truncateStringForPrompt(teamId, MAX_STRING_PREVIEW_LENGTH)}\n`);
            }

            formattedSwarmStateParts.push(`- Swarm Leader:\n${this._truncateStringForPrompt(swarmLeader, MAX_STRING_PREVIEW_LENGTH)}\n`);

            const activeSubtasksCount = subtasks.filter(st => typeof st === "object" && st !== null && (st.status === "todo" || st.status === "in_progress")).length;
            const completedSubtasksCount = subtasks.filter(st => typeof st === "object" && st !== null && st.status === "done").length;
            formattedSwarmStateParts.push(`- Subtasks (active: ${activeSubtasksCount}, completed: ${completedSubtasksCount}):\n${this._truncateStringForPrompt(subtasks, MAX_STRING_PREVIEW_LENGTH)}\n`);

            formattedSwarmStateParts.push(`- Subtask Leaders:\n${this._truncateStringForPrompt(subtaskLeaders, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Event Subscriptions:\n${this._truncateStringForPrompt(eventSubscriptions, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Blackboard:\n${this._truncateStringForPrompt(blackboard, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Resources:\n${this._truncateStringForPrompt(resources, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Records:\n${this._truncateStringForPrompt(records, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Stats:\n${this._truncateStringForPrompt(stats, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Limits:\n${this._truncateStringForPrompt(limits, MAX_STRING_PREVIEW_LENGTH)}\n`);
            formattedSwarmStateParts.push(`- Pending Tool Calls:\n${this._truncateStringForPrompt(pendingToolCalls, MAX_STRING_PREVIEW_LENGTH)}\n`);

            swarmStateOutput = `\nSWARM STATE DETAILS:\n${formattedSwarmStateParts.join("\n\n")}`;
        } catch (e) {
            logger.error("Error formatting swarm state for prompt:", e);
        }
        return swarmStateOutput;
    }

    /**
     * Public wrapper for generating a system message for a specific bot.
     */
    public async generateSystemMessageForBot(goal: string, bot: BotParticipant, convoConfig: ChatConfigObject, teamConfig?: TeamConfigObject): Promise<string> {
        return this._buildSystemMessage(goal, bot, convoConfig, teamConfig);
    }

    /**
     * Responds to a user or bot message event
     * Picks responders, runs the reasoning loop for each responder, 
     * stores response messages, and returns messages to the caller.
     * 
     * @param event - The message event to respond to
     * @returns - A list of response messages
     */
    async respond({
        chatId,
        messageId,
        model,
        userData,
    }: LLMCompletionTask): Promise<MessageState[]> {
        // Get conversation state
        const conversationState = await this.conversationStore.get(chatId);
        if (!conversationState) {
            throw new Error(`Conversation state not found for chatId: ${chatId}`);
        }

        // Update preferred model in config if a model was specified
        if (model && model.trim() !== "") {
            conversationState.config.preferredModel = model;
            this.conversationStore.updateConfig(chatId, conversationState.config);
        }

        // Load the triggering message
        const messageState = await this.messageStore.getMessage(messageId);
        if (!messageState) {
            throw new Error(`Message ${messageId} not found in messageStore`);
        }

        // Determine credit account (e.g. user who triggered the message)
        const creditAccountId = userData.creditAccountId;
        if (!creditAccountId) {
            // This is a critical error, as billing depends on it.
            // Log the error and the userData for debugging.
            const logMessage = `Critical: creditAccountId is missing from userData in CompletionService.respond. Billing will fail. UserID: ${userData.id}, ChatID: ${chatId}, MessageID: ${messageId}`;
            logger.error(logMessage);
            throw new Error("User credit account ID is missing. Cannot proceed with response generation.");
        }

        // Get available tools
        const availableToolsFromState = conversationState.availableTools;
        // Fetch full tool definitions, including estimatedCost
        const fullAvailableTools: Tool[] = availableToolsFromState
            .map(toolRef => {
                // Assuming toolRef is either a Tool object or just a name string
                const toolName = typeof toolRef === "string" ? toolRef : toolRef.name;
                const toolDef = this.toolRegistry.getToolDefinition(toolName);
                if (!toolDef) {
                    logger.warn(`Tool definition not found in registry for: ${toolName}. It will not be available.`);
                    return null;
                }
                // Ensure the definition conforms to the Tool interface, especially for estimatedCost
                return toolDef as Tool;
            })
            .filter(tool => tool !== null) as Tool[];

        // Determine responders
        const { responders } = await this.agentGraph.selectResponders(conversationState, messageState);

        // Initialize results
        const results: MessageState[] = [];
        const allResponseStats: ResponseStats[] = [];

        // Ensure stats object exists on conversation config BEFORE calculating remaining limits
        if (!conversationState.config.stats) {
            conversationState.config.stats = ChatConfig.defaultStats();
        }

        const effectiveLimits = new ChatConfig({ config: conversationState.config }).getEffectiveLimits();
        const numResponders = responders.length > 0 ? responders.length : 1;

        const { toolCallsPerBot: convoAllocatedToolCallsPerBot, creditsPerBot: convoAllocatedCreditsPerBot } = this._calculatePerBotAllocation(
            conversationState.config.stats,
            effectiveLimits,
            numResponders,
        );

        const maxTurnDurationMs = effectiveLimits.maxDurationMs;
        const controller = new AbortController();
        this.activeControllers.set(chatId, controller);
        let overallTurnTimeoutId: NodeJS.Timeout | undefined;

        if (maxTurnDurationMs > 0) {
            overallTurnTimeoutId = setTimeout(() => {
                logger.info(`Overall turn timeout of ${maxTurnDurationMs}ms reached for chat ${chatId}. Aborting all bot responses for this turn.`);
                controller.abort("TurnTimeout");
            }, maxTurnDurationMs);
        }

        try {
            await Promise.all(responders.map(async (responder) => {
                const goal = conversationState.config.goal || "Follow the user's instructions.";
                const botSystemMessage = await this._buildSystemMessage(goal, responder, conversationState.config, conversationState.teamConfig);

                const response = await this.reasoningEngine.runLoop(
                    { id: messageId },
                    botSystemMessage, // Pass bot-specific system message
                    fullAvailableTools, // Pass full Tool objects
                    responder,
                    creditAccountId,
                    conversationState.config,
                    convoAllocatedToolCallsPerBot,
                    convoAllocatedCreditsPerBot,
                    userData,
                    chatId,
                    model,
                    controller.signal,
                );
                results.push(response.finalMessage);
                allResponseStats.push(response.responseStats);
            }));
        } catch (error) {
            logger.error(`Error during respond for chat ${chatId}. One or more bot turns may have failed.`, { error });
            // Rethrow to indicate failure of the respond operation to the caller
            throw error;
        } finally {
            if (overallTurnTimeoutId) {
                clearTimeout(overallTurnTimeoutId);
            }
            this.activeControllers.delete(chatId);
        }

        // Store responses by adding each message to cache
        await Promise.all(results.map(m => this.messageStore.addMessage(chatId, m)));

        // Aggregate stats from all responses
        let totalToolCallsInTurn = 0;
        let totalCreditsInTurn = BigInt(0);
        for (const stats of allResponseStats) {
            totalToolCallsInTurn += stats.toolCalls;
            totalCreditsInTurn += stats.creditsUsed;
        }

        // Update conversation-level stats
        conversationState.config.stats.totalToolCalls += totalToolCallsInTurn;
        conversationState.config.stats.totalCredits = (BigInt(conversationState.config.stats.totalCredits) + totalCreditsInTurn).toString();
        conversationState.config.stats.lastProcessingCycleEndedAt = Date.now();

        // Persist updated stats back to the conversation store
        this.conversationStore.updateConfig(chatId, conversationState.config);
        return results;
    }

    /**
     * Calculates the per-bot allocation of tool calls and credits for the current turn/event.
     *
     * This method divides the *currently known remaining* conversation budget (tool calls and credits)
     * approximately evenly among the number of bots responding in the current parallel execution batch.
     * This is a conservative strategy designed to prevent exceeding total conversation limits when bots
     * operate concurrently. Each bot's runLoop is then responsible for respecting this allocated slice.
     *
     * @param currentConfigStats The current statistics from the conversation's config.
     * @param conversationLimits The effective, capped limits for the overall conversation.
     * @param numResponders The number of bots that will be responding concurrently.
     * @returns An object containing the calculated toolCallsPerBot and creditsPerBot.
     */
    private _calculatePerBotAllocation(
        currentConfigStats: NonNullable<ChatConfigObject["stats"]>,
        conversationLimits: Required<NonNullable<ChatConfigObject["limits"]>>,
        numResponders: number,
    ): { toolCallsPerBot: number; creditsPerBot: bigint } {
        const safeNumResponders = numResponders > 0 ? numResponders : 1; // Avoid division by zero

        const remainingConversationToolCalls = Math.max(0, Number(conversationLimits.maxToolCalls) - currentConfigStats.totalToolCalls);
        const remainingConversationCredits = BigInt(conversationLimits.maxCredits) - BigInt(currentConfigStats.totalCredits);

        const toolCallsPerBot = Math.floor(remainingConversationToolCalls / safeNumResponders);
        // Only allocate credits if there are positive remaining credits
        const creditsPerBot = remainingConversationCredits > BigInt(0) ? remainingConversationCredits / BigInt(safeNumResponders) : BigInt(0);

        // Note: BigInt division truncates (floor). Any minor rounding errors in credit distribution
        // are considered insignificant due to the large number of credits representing $1.

        return { toolCallsPerBot, creditsPerBot };
    }

    async handleInternalEvent(event: SwarmEvent): Promise<void> {
        const { conversationId, type } = event;
        logger.info(`CompletionService handling internal event: ${type} for convo: ${conversationId}`);

        const conversationState = await this.conversationStore.get(conversationId);
        if (!conversationState) {
            logger.error(`Conversation state not found for ${conversationId} in handleInternalEvent`);
            throw new Error(`Conversation state not found for ${conversationId}`);
        }

        const fullAvailableToolsForEvent: Tool[] = (conversationState.availableTools || [])
            .map(toolRef => {
                const toolName = typeof toolRef === "string" ? toolRef : toolRef.name;
                const toolDef = this.toolRegistry.getToolDefinition(toolName);
                if (!toolDef) {
                    logger.warn(`Tool definition not found in registry for: ${toolName} during internal event. It will not be available.`);
                    return null;
                }
                return toolDef as Tool;
            })
            .filter(tool => tool !== null) as Tool[];

        const sessionUserToUse = event.sessionUser;
        if (!sessionUserToUse || typeof sessionUserToUse.creditAccountId !== "string" || sessionUserToUse.creditAccountId.trim() === "") {
            const error = `Critical: creditAccountId is missing, invalid, or empty in sessionUser for SwarmEvent. Billing will fail. UserID: ${sessionUserToUse?.id}, ChatID: ${conversationId}`;
            logger.error(error);
            throw new Error(error);
        }
        const creditAccountId: string = sessionUserToUse.creditAccountId;

        const results: MessageState[] = [];
        let currentResponseStats: ResponseStats[] = [];

        // Ensure stats object exists on conversation config BEFORE calculating remaining limits
        if (!conversationState.config.stats) {
            conversationState.config.stats = ChatConfig.defaultStats();
        }
        const effectiveLimitsEvent = new ChatConfig({ config: conversationState.config }).getEffectiveLimits();
        const currentConvoStats = conversationState.config.stats;

        const maxTurnDurationMs = effectiveLimitsEvent.maxDurationMs;
        const controller = new AbortController();
        this.activeControllers.set(conversationId, controller);
        let eventTimeoutId: NodeJS.Timeout | undefined;

        if (maxTurnDurationMs > 0) {
            eventTimeoutId = setTimeout(() => {
                logger.info(`Internal event processing timeout of ${maxTurnDurationMs}ms reached for conversation ${conversationId}, event type ${event.type}. Aborting.`);
                controller.abort("TurnTimeout");
            }, maxTurnDurationMs);
        }

        try {
            const goalForEvent = conversationState.config.goal || DEFAULT_GOAL;

            switch (event.type) {
                case "swarm_started": {
                    const allResponseStatsSwarmStarted: ResponseStats[] = [];
                    const swarmStartEvent = event as SwarmStartedEvent;
                    const specificGoal = swarmStartEvent.goal || goalForEvent;
                    const systemMessageText = `Swarm initiated. Goal: ${specificGoal}.`;
                    const startMessage = { text: systemMessageText };
                    const msgCfg = MessageConfig.default();
                    msgCfg.setRole("system");
                    const systemMessageConfigExport = msgCfg.export();
                    const syntheticMessageForSelection: MessageState = {
                        id: generatePK().toString(),
                        createdAt: new Date(),
                        config: systemMessageConfigExport,
                        text: systemMessageText,
                        language: DEFAULT_LANGUAGE,
                        parent: null,
                        user: { id: sessionUserToUse.id },
                    } as MessageState;

                    const { responders } = await this.agentGraph.selectResponders(conversationState, syntheticMessageForSelection);
                    if (!responders || responders.length === 0) {
                        logger.warn(`No responders found for swarm_started in convo ${conversationId}`);
                        return;
                    }
                    const numSwarmResponders = responders.length > 0 ? responders.length : 1;
                    const { toolCallsPerBot: convoAllocatedToolCallsPerBotEvent, creditsPerBot: convoAllocatedCreditsPerBotEvent } = this._calculatePerBotAllocation(
                        currentConvoStats,
                        effectiveLimitsEvent,
                        numSwarmResponders,
                    );

                    await Promise.all(responders.map(async (responder) => {
                        const botSystemMessage = await this._buildSystemMessage(specificGoal, responder, conversationState.config, conversationState.teamConfig);
                        const response = await this.reasoningEngine.runLoop(
                            startMessage,
                            botSystemMessage,
                            fullAvailableToolsForEvent,
                            responder,
                            creditAccountId,
                            conversationState.config,
                            convoAllocatedToolCallsPerBotEvent,
                            convoAllocatedCreditsPerBotEvent,
                            event.sessionUser,
                            conversationId,
                            undefined, // model for swarm_started can be default
                            controller.signal,
                        );
                        results.push(response.finalMessage);
                        allResponseStatsSwarmStarted.push(response.responseStats);
                    }));
                    currentResponseStats = allResponseStatsSwarmStarted;
                    break;
                }
                case "external_message_created": {
                    if (!event.payload || !event.payload.messageId) {
                        logger.error("external_message_created event missing messageId in payload", { event });
                        return;
                    }
                    const { messageId } = event.payload;
                    const messageState = await this.messageStore.getMessage(messageId);
                    if (!messageState) {
                        logger.error(`Message ${messageId} not found for external_message_created in ${conversationId}`);
                        return;
                    }
                    const allResponseStatsExternal: ResponseStats[] = [];
                    const { responders } = await this.agentGraph.selectResponders(conversationState, messageState);
                    if (!responders || responders.length === 0) {
                        logger.warn(`No responders found for external_message_created in convo ${conversationId}`, { messageId });
                        return;
                    }
                    const numMsgResponders = responders.length > 0 ? responders.length : 1;
                    const { toolCallsPerBot: convoAllocatedToolCallsPerBotEvent, creditsPerBot: convoAllocatedCreditsPerBotEvent } = this._calculatePerBotAllocation(
                        currentConvoStats,
                        effectiveLimitsEvent,
                        numMsgResponders,
                    );

                    await Promise.all(responders.map(async (responder) => {
                        const botSystemMessage = await this._buildSystemMessage(goalForEvent, responder, conversationState.config, conversationState.teamConfig);
                        const response = await this.reasoningEngine.runLoop(
                            { id: messageId },
                            botSystemMessage,
                            fullAvailableToolsForEvent,
                            responder,
                            creditAccountId,
                            conversationState.config,
                            convoAllocatedToolCallsPerBotEvent,
                            convoAllocatedCreditsPerBotEvent,
                            event.sessionUser,
                            conversationId,
                            undefined, // Should probably add way to pass in model override here
                            controller.signal,
                        );
                        results.push(response.finalMessage);
                        allResponseStatsExternal.push(response.responseStats);
                    }));
                    currentResponseStats = allResponseStatsExternal;
                    break;
                }
                case "ApprovedToolExecutionRequest": {
                    if (!event.payload || !event.payload.pendingToolCall) {
                        logger.error("ApprovedToolExecutionRequest event missing pendingToolCall in payload", { event });
                        return;
                    }
                    const { pendingToolCall } = event.payload as { pendingToolCall: PendingToolCallEntry };
                    logger.info(`Processing ApprovedToolExecutionRequest for pendingId: ${pendingToolCall.pendingId}, tool: ${pendingToolCall.toolName}`);

                    const pIndex = (conversationState.config.pendingToolCalls || []).findIndex(p => p.pendingId === pendingToolCall.pendingId);
                    if (pIndex !== -1) {
                        conversationState.config.pendingToolCalls![pIndex].status = PendingToolCallStatus.EXECUTING;
                        conversationState.config.pendingToolCalls![pIndex].lastAttemptTime = Date.now();
                        this.conversationStore.updateConfig(conversationId, conversationState.config); // Persist status update
                    } else {
                        logger.error(`PendingToolCallEntry not found for approved tool: ${pendingToolCall.pendingId}`, { conversationId });
                        // Decide if we should proceed or return early if the entry is critical for context/logging
                    }

                    let toolCallResponseData: Awaited<ReturnType<typeof this.reasoningEngine.toolRunner.run>>; // Corrected: reasoningEngine.toolRunner
                    let executedToolCost = BigInt(0);
                    try {
                        const toolArgs = JSON.parse(pendingToolCall.toolArguments);
                        // Corrected: Access toolRunner via reasoningEngine instance
                        toolCallResponseData = await this.reasoningEngine.toolRunner.run(
                            pendingToolCall.toolName,
                            toolArgs,
                            {
                                conversationId: event.conversationId,
                                callerBotId: pendingToolCall.callerBotId,
                                sessionUser: event.sessionUser,
                                signal: controller.signal,
                            },
                        );
                        executedToolCost = toolCallResponseData.ok ? BigInt(toolCallResponseData.data.creditsUsed) : (toolCallResponseData.error.creditsUsed ? BigInt(toolCallResponseData.error.creditsUsed) : BigInt(0));
                    } catch (execError: any) {
                        logger.error(`Error executing approved tool ${pendingToolCall.toolName} (pendingId: ${pendingToolCall.pendingId})`, { error: execError });
                        toolCallResponseData = { ok: false, error: { code: "TOOL_EXECUTION_ERROR", message: execError.message || "Unknown tool execution error" } };
                    }

                    if (pIndex !== -1) {
                        conversationState.config.pendingToolCalls![pIndex].status = toolCallResponseData.ok ? PendingToolCallStatus.COMPLETED_SUCCESS : PendingToolCallStatus.COMPLETED_FAILURE;
                        conversationState.config.pendingToolCalls![pIndex].result = toolCallResponseData.ok ? JSON.stringify(toolCallResponseData.data.output) : undefined;
                        conversationState.config.pendingToolCalls![pIndex].error = !toolCallResponseData.ok ? JSON.stringify(toolCallResponseData.error) : undefined;
                        conversationState.config.pendingToolCalls![pIndex].cost = executedToolCost.toString();
                    } else {
                        logger.warn(`Could not update PendingToolCallEntry ${pendingToolCall.pendingId} with final execution status as it was not found.`);
                    }

                    // TODO-SCHEDULED-TOOL: Refactor message config creation to MessageConfig.fromSystemToolCallResponse(toolCallId)
                    const toolResponseMessageCfg = MessageConfig.default();
                    toolResponseMessageCfg.setRole("tool");
                    // toolResponseMessageCfg.setToolCallId(pendingToolCall.toolCallId); // This method doesn't exist
                    // For a tool response, we need to provide a ToolFunctionCall object
                    const toolCallResultEntry: ToolFunctionCall = {
                        id: pendingToolCall.toolCallId, // This is the original tool_call_id from the LLM request
                        function: { // Mocking function details as they are not strictly needed for the response message
                            name: pendingToolCall.toolName,
                            arguments: pendingToolCall.toolArguments, // Original arguments
                        },
                        result: toolCallResponseData.ok
                            ? { success: true, output: JSON.stringify(toolCallResponseData.data.output) }
                            : { success: false, error: toolCallResponseData.error },
                    };
                    toolResponseMessageCfg.setToolCalls([toolCallResultEntry]);

                    const toolResponseMessage: MessageState = {
                        id: generatePK().toString(),
                        createdAt: new Date(),
                        config: toolResponseMessageCfg.export(),
                        text: toolCallResponseData.ok ? JSON.stringify(toolCallResponseData.data.output) : JSON.stringify(toolCallResponseData.error),
                        language: DEFAULT_LANGUAGE,
                        parent: null,
                        user: { id: pendingToolCall.callerBotId },
                    };
                    results.push(toolResponseMessage);
                    currentResponseStats.push({ toolCalls: 1, creditsUsed: executedToolCost });

                    const callerBot = conversationState.participants.find(p => p.id === pendingToolCall.callerBotId);
                    if (callerBot) {
                        logger.info(`Re-engaging bot ${callerBot.id} after approved tool execution: ${pendingToolCall.toolName}`);
                        const { toolCallsPerBot: reEngageAllocatedToolCalls, creditsPerBot: reEngageAllocatedCredits } = this._calculatePerBotAllocation(
                            currentConvoStats, effectiveLimitsEvent, 1,
                        );
                        try {
                            const botSystemMessage = await this._buildSystemMessage(goalForEvent, callerBot, conversationState.config, conversationState.teamConfig);
                            const botResponse = await this.reasoningEngine.runLoop(
                                { id: toolResponseMessage.id }, botSystemMessage, fullAvailableToolsForEvent, callerBot, creditAccountId,
                                conversationState.config, reEngageAllocatedToolCalls, reEngageAllocatedCredits, event.sessionUser,
                                conversationId, undefined, controller.signal,
                            );
                            results.push(botResponse.finalMessage);
                            currentResponseStats.push(botResponse.responseStats);
                        } catch (reEngageError: any) {
                            logger.error(`Error re-engaging bot ${callerBot.id} after approved tool execution`, { error: reEngageError });
                            if (conversationId && this.unifiedEventBus) {
                                const eventSource = EventUtils.createEventSource("cross-cutting", "CompletionService");
                                const metadata = EventUtils.createEventMetadata("fire-and-forget", "high", {
                                    conversationId,
                                });

                                this.unifiedEventBus.publish(
                                    EventUtils.createBaseEvent(
                                        EventTypes.BOT_STATUS_UPDATED,
                                        {
                                            chatId: conversationId,
                                            botId: callerBot.id,
                                            status: "error_internal",
                                            error: { message: reEngageError.message || "Failed to process tool result", code: "REENGAGE_FAILURE" },
                                        },
                                        eventSource,
                                        metadata,
                                    ),
                                ).catch(err => logger.error("Failed to emit re-engage error event", { error: err }));
                            }
                        }
                    } else {
                        logger.error(`Caller bot ${pendingToolCall.callerBotId} not found for re-engagement.`);
                    }
                    break;
                }
                case "RejectedToolExecutionRequest": {
                    if (!event.payload || !event.payload.pendingToolCall) {
                        logger.error("RejectedToolExecutionRequest event missing pendingToolCall in payload", { event });
                        return;
                    }
                    const { pendingToolCall, reason } = event.payload as { pendingToolCall: PendingToolCallEntry, reason?: string };
                    logger.info(`Processing RejectedToolExecutionRequest for pendingId: ${pendingToolCall.pendingId}, tool: ${pendingToolCall.toolName}, reason: ${reason}`);

                    const pIndex = (conversationState.config.pendingToolCalls || []).findIndex(p => p.pendingId === pendingToolCall.pendingId);
                    if (pIndex !== -1) {
                        conversationState.config.pendingToolCalls![pIndex].status = PendingToolCallStatus.REJECTED_BY_USER;
                        conversationState.config.pendingToolCalls![pIndex].statusReason = reason || "Rejected by user without specific reason.";
                        conversationState.config.pendingToolCalls![pIndex].decisionTime = Date.now();
                        if (pendingToolCall.approvedOrRejectedByUserId) {
                            conversationState.config.pendingToolCalls![pIndex].approvedOrRejectedByUserId = pendingToolCall.approvedOrRejectedByUserId;
                        }
                    } else {
                        logger.error(`PendingToolCallEntry not found for rejected tool: ${pendingToolCall.pendingId}`, { conversationId });
                    }

                    const conciseRejectionReason = `Tool call '${pendingToolCall.toolName}' (ID: ${pendingToolCall.toolCallId}) was rejected. Reason: ${reason || "User declined."}`;
                    const detailedRejectionPayloadForLlm = JSON.stringify({
                        __vrooli_tool_rejected: true,
                        toolName: pendingToolCall.toolName,
                        pendingId: pendingToolCall.pendingId,
                        reason: reason || "Tool call was rejected by the user.",
                        message: conciseRejectionReason, // LLM sees this concise message within the structured payload
                    });

                    const toolRejectionMessageCfg = MessageConfig.default();
                    toolRejectionMessageCfg.setRole("tool");
                    const toolCallRejectionEntry: ToolFunctionCall = {
                        id: pendingToolCall.toolCallId,
                        function: {
                            name: pendingToolCall.toolName,
                            arguments: pendingToolCall.toolArguments,
                        },
                        result: {
                            success: false,
                            error: {
                                code: "TOOL_REJECTED_BY_USER",
                                message: conciseRejectionReason, // Use the concise reason here for the error object
                            },
                        },
                    };
                    toolRejectionMessageCfg.setToolCalls([toolCallRejectionEntry]);

                    const toolRejectionMessage: MessageState = {
                        id: generatePK().toString(),
                        createdAt: new Date(),
                        config: toolRejectionMessageCfg.export(),
                        text: detailedRejectionPayloadForLlm, // The full JSON string for the LLM to parse in the message text
                        language: DEFAULT_LANGUAGE,
                        parent: null,
                        user: { id: pendingToolCall.callerBotId },
                    };
                    results.push(toolRejectionMessage);

                    const callerBot = conversationState.participants.find(p => p.id === pendingToolCall.callerBotId);
                    if (callerBot) {
                        logger.info(`Re-engaging bot ${callerBot.id} after tool rejection: ${pendingToolCall.toolName}`);
                        const { toolCallsPerBot: reEngageAllocatedToolCalls, creditsPerBot: reEngageAllocatedCredits } = this._calculatePerBotAllocation(
                            currentConvoStats, effectiveLimitsEvent, 1,
                        );
                        try {
                            const botSystemMessage = await this._buildSystemMessage(goalForEvent, callerBot, conversationState.config, conversationState.teamConfig);
                            const botResponse = await this.reasoningEngine.runLoop(
                                { id: toolRejectionMessage.id }, botSystemMessage, fullAvailableToolsForEvent, callerBot, creditAccountId,
                                conversationState.config, reEngageAllocatedToolCalls, reEngageAllocatedCredits, event.sessionUser,
                                conversationId, undefined, controller.signal,
                            );
                            results.push(botResponse.finalMessage);
                            currentResponseStats.push(botResponse.responseStats);
                        } catch (reEngageError: any) {
                            logger.error(`Error re-engaging bot ${callerBot.id} after tool rejection`, { error: reEngageError });
                            if (conversationId && this.unifiedEventBus) {
                                const eventSource = EventUtils.createEventSource("cross-cutting", "CompletionService");
                                const metadata = EventUtils.createEventMetadata("fire-and-forget", "high", {
                                    conversationId,
                                });

                                this.unifiedEventBus.publish(
                                    EventUtils.createBaseEvent(
                                        EventTypes.BOT_STATUS_UPDATED,
                                        {
                                            chatId: conversationId,
                                            botId: callerBot.id,
                                            status: "error_internal",
                                            error: { message: reEngageError.message || "Failed to process tool rejection", code: "REENGAGE_FAILURE" },
                                        },
                                        eventSource,
                                        metadata,
                                    ),
                                ).catch(err => logger.error("Failed to emit rejection error event", { error: err }));
                            }
                        }
                    } else {
                        logger.error(`Caller bot ${pendingToolCall.callerBotId} not found for re-engagement after rejection.`);
                    }
                    break;
                }
                default: {
                    logger.warn(`Unhandled SwarmEvent type: ${event.type} in convo ${conversationId}`);
                    // Clear timeout if we return early for an unhandled event type.
                    if (eventTimeoutId) {
                        clearTimeout(eventTimeoutId);
                    }
                    return;
                }
            }

            if (results.length > 0) {
                await Promise.all(results.map(m => this.messageStore.addMessage(conversationId, m)));

                // Aggregate stats from all responses for the handled event
                let totalToolCallsInEvent = 0;
                let totalCreditsInEvent = BigInt(0);
                for (const stats of currentResponseStats) {
                    totalToolCallsInEvent += stats.toolCalls;
                    totalCreditsInEvent += stats.creditsUsed;
                }

                // Update conversation-level stats
                conversationState.config.stats.totalToolCalls += totalToolCallsInEvent;
                conversationState.config.stats.totalCredits = (BigInt(conversationState.config.stats.totalCredits) + totalCreditsInEvent).toString();
                conversationState.config.stats.lastProcessingCycleEndedAt = Date.now();

                // Persist updated stats back to the conversation store
                this.conversationStore.updateConfig(conversationId, conversationState.config);
            }
        } catch (error) {
            logger.error(`Error during handleInternalEvent for convo ${conversationId}, event ${event.type}`, { error });
            // Rethrow to indicate failure of the handleInternalEvent operation to the caller
            throw error;
        } finally {
            if (eventTimeoutId) {
                clearTimeout(eventTimeoutId);
            }
            this.activeControllers.delete(conversationId); // Remove controller on completion or error
        }
    }

    /**
     * Attempts to cancel an ongoing response generation for a given chat ID.
     * @param chatId The ID of the chat for which to cancel the response.
     */
    public requestCancellation(chatId: string): void {
        const controller = this.activeControllers.get(chatId);
        if (controller) {
            logger.info(`Cancellation requested for chatId: ${chatId}. Aborting controller.`);
            controller.abort();
            // The AbortSignal being observed by ReasoningEngine.runLoop will handle the rest.
            // We can delete from activeControllers here, or let the finally block in runLoop invocation do it.
            // Deleting here ensures it's promptly removed if the loop is already finished or stuck before checking signal.
            this.activeControllers.delete(chatId);
        } else {
            logger.warn(`No active AbortController found for chatId: ${chatId} to cancel.`);
        }
    }
}

// Instantiate stores and services that are dependencies
const prismaChatStore = new PrismaChatStore();
// Wrap PrismaChatStore with CachedConversationStateStore
export const conversationStateStore = new CachedConversationStateStore(prismaChatStore);
export const messageStore = new RedisMessageStore(); // Export if used directly elsewhere

const contextBuilder = new RedisContextBuilder();

// Instantiate Tool Runners and their dependencies
const swarmTools = new SwarmTools(logger, conversationStateStore);
const mcpToolRunner = new McpToolRunner(logger, swarmTools);
const toolRunner = new CompositeToolRunner(mcpToolRunner);

const agentGraphInstance = new CompositeGraph();
const llmRouter = new FallbackRouter();

// Instantiate ToolRegistry
const toolRegistry = new ToolRegistry(logger);

// Instantiate core engines/services
const reasoningEngine = new ReasoningEngine(
    contextBuilder,
    llmRouter,
    toolRunner,
);

export const completionService = new CompletionService(
    reasoningEngine,
    agentGraphInstance,
    conversationStateStore,
    messageStore,
    toolRegistry,
);

/**
 * Represents a record of an active swarm in the system.
 */
export interface ActiveSwarmRecord {
    /** Whether the user associated with the swarm has premium status */
    hasPremium: boolean;
    /** The time the swarm was added to the registry (timestamp in milliseconds) */
    startTime: number;
    /** The unique ID of the conversation the swarm belongs to */
    conversationId: string;
    /** The unique ID of the user who initiated or owns the swarm */
    userId: string;
}

