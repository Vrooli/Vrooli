/* eslint-disable func-style */
import { type BotConfigObject, ChatConfig, type ChatConfigObject, DEFAULT_LANGUAGE, generatePK, McpSwarmToolName, McpToolName, MessageConfig, MINUTES_5_MS, nanoid, type PendingToolCallEntry, PendingToolCallStatus, SECONDS_1_MS, type SessionUser, type ToolFunctionCall } from "@local/shared";
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import * as fs from "fs/promises";
import type OpenAI from "openai";
import * as path from "path";
import { fileURLToPath } from "url"; // For __dirname in ESM
import { logger } from "../../events/logger.js";
import { Notify } from "../../notify/notify.js"; // Added import for Notify
import { SocketService } from "../../sockets/io.js";
import type { ManagedTaskStateMachine } from "../../tasks/activeTaskRegistry.js";
import { type LLMCompletionTask } from "../../tasks/taskTypes.js";
import { BusService } from "../bus.js";
import { ToolRegistry } from "../mcp/registry.js";
import { SwarmTools } from "../mcp/tools.js";
import { type Tool } from "../mcp/types.js";
import { type AgentGraph, CompositeGraph } from "./agentGraph.js";
import { CachedConversationStateStore, type ConversationStateStore, PrismaChatStore } from "./chatStore.js";
import { type ContextBuilder, RedisContextBuilder } from "./contextBuilder.js";
import { type MessageStore, RedisMessageStore } from "./messageStore.js";
import type { FunctionCallStreamEvent } from "./router.js";
import { FallbackRouter, type LlmRouter } from "./router.js";
import { CompositeToolRunner, McpToolRunner, type ToolRunner } from "./toolRunner.js";
import { type BotParticipant, type ConversationState, type MessageState, type ResponseStats, type SwarmEvent, type SwarmStartedEvent } from "./types.js";

// TODO Tthe failure cases could be handled better. E.g. emitting event to show retry button
//TODO handle message updates (branching conversation)
// TODO make sure preferred model is stored in the conversation state
// TODO need turn timeouts
// TODO add mechanism for swarm and routine state machines to have a delay set by config that schedules any runs and resource mutations instead of running them immediately. This would be clearly defined in the context, and a list of schedules tool calls would be included in the context too. This enables the swarm to plan around the schedule, which is needed for users to approve/reject scheduled tool calls for safety reasons.
// TODO swarm world model should suggest starting a metacognition loop to best complete the goal, and adding to to the chat context. Can search for existing prompts.
//TODO conversation context should be able to store recommended routines to call for certain scenarios, pulled from the team config if available.
// TODO have built-in events for swarm that bots can be assigned to. For example, you could have a bot that simplly checks if the current subtask is complete. If so, the swarm closes. If not, we nudge the swarm to continue.
//TODO swarm needs to see builtInToolDefinitions and swarmToolDefinitions from the MCP registry
//TODO swarm should be able to share data with specific bots - not just the overall swarm shared state (that goes to all bots).
//TODO should be using limits.delayBetweenProcessingCyclesMs for when you want the swarm to be slowed down
//TODO make sure all SwarmEvents are being used

const DEFAULT_GOAL = "Process current event.";

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
    /**
     * @param contextBuilder     Collects all chat history that can fit into the context window of a bot turn.
     * @param llmRouter          Streams Responses‑API calls to chosen LLM provider (e.g. OpenAI, Anthropic).
     * @param toolRunner         Executes MCP tool calls (performs the actual work of a tool).
     */
    constructor(
        private readonly contextBuilder: ContextBuilder,
        private readonly llmRouter: LlmRouter,
        public readonly toolRunner: ToolRunner,
    ) { }

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
        if (chatId) {
            SocketService.get().emitSocketEvent("typing", chatId, { starting: [bot.id] });
        }

        // Create sets for efficient lookup of Vrooli custom tool names
        const mcpToolNames = new Set(Object.values(McpToolName));
        const mcpSwarmToolNames = new Set(Object.values(McpSwarmToolName));

        // Transform availableTools to OpenAI.Responses.Tool[] format for ContextBuilder and LlmRouter
        const toolsForLlm: OpenAI.Responses.Tool[] = availableTools.reduce((acc, ts) => {
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

        // Check for cancellation at the very beginning
        if (abortSignal?.aborted) {
            const reason = abortSignal.reason;
            let errorMessage = "Response cancelled";
            let errorCode = "CANCELLED";
            let logMessage = `Response cancelled by signal before starting for bot ${bot.id} in chat ${chatId}`;
            let errorToThrow = new Error("ResponseCancelledError");

            if (reason === "TurnTimeout") {
                errorMessage = "Bot turn timed out";
                errorCode = "TIMED_OUT";
                logMessage = `Bot turn timed out before starting for bot ${bot.id} in chat ${chatId}`;
                errorToThrow = new Error("ResponseTimeoutError");
            } else if (reason instanceof Error) {
                errorMessage = reason.message;
            } else if (typeof reason === "string" && reason.length > 0) {
                errorMessage = reason;
            }

            logger.info(logMessage + (abortSignal.reason ? `. Reason: ${String(abortSignal.reason)}` : ""));
            if (chatId) {
                SocketService.get().emitSocketEvent("responseStream", chatId, {
                    __type: "error",
                    botId: bot.id,
                    error: { message: errorMessage, code: errorCode },
                });
                SocketService.get().emitSocketEvent("botStatusUpdate", chatId, {
                    chatId,
                    botId: bot.id,
                    status: "error_internal",
                    error: { message: errorMessage, code: errorCode },
                });
            }
            throw errorToThrow;
        }

        // Build context if we have a chatId + message ID, else wrap a text prompt
        let inputs: MessageState[];
        if (chatId && "id" in startMessage) {
            // Build context, reserving tokens for tools (none for now) and world-model
            const { messages } = await this.contextBuilder.build(
                chatId,
                bot,
                model ?? DEFAULT_LANGUAGE,
                startMessage.id,
                { tools: toolsForLlm, systemMessage: systemMessageContent },
            );
            inputs = messages;
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
            inputs = [tmp];
        } else {
            throw new Error("Invalid startMessage for runLoop");
        }
        let draftMessage = "";
        // Collect all tool calls made during this loop
        const toolCalls: ToolFunctionCall[] = [];
        let previousResponseId: string | undefined;
        const responseStats: ResponseStats = { toolCalls: 0, creditsUsed: BigInt(0) };

        // Compute per-bot-response limits from the overall conversation config.
        // These 'responseLimits' apply to the cumulative actions and resource usage of a single bot
        // within one full execution of this runLoop.
        const responseLimits = new ChatConfig({ config }).getEffectiveLimits();

        // Prepare the final system message for the LLM call
        // The systemMessageContent is now the complete, bot-specific system message.
        const finalSystemMessageForLlm = systemMessageContent;
        // if (bot.meta?.systemPrompt) { // This logic is now handled upstream in CompletionService._buildSystemMessage
        //     finalSystemMessageForLlm = `${finalSystemMessageForLlm}\n${bot.meta.systemPrompt}`.trim();
        // }

        // exceeded() checks if the *cumulative* stats for this runLoop (responseStats)
        // have breached EITHER the total allocation passed into this runLoop (convoAllocated...)
        // OR the specific per-bot-response limits defined in the conversation config (responseLimits).
        // A bot's execution stops if it hits any of these caps.
        // Note: convoAllocatedToolCalls and convoAllocatedCredits can be 0 if the conversation budget is already exhausted.
        const exceeded = () => {
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
        };

        try {
            // Emit thinking status before starting the main loop
            if (chatId) {
                SocketService.get().emitSocketEvent("botStatusUpdate", chatId, { chatId, botId: bot.id, status: "thinking", message: "Processing request..." });
            }
            while (inputs.length && !exceeded()) {
                // Check for cancellation at the beginning of each iteration
                if (abortSignal?.aborted) {
                    const reason = abortSignal.reason;
                    let errorMessage = "Response cancelled";
                    let errorCode = "CANCELLED";
                    let logMessage = `Response cancelled by signal during loop for bot ${bot.id} in chat ${chatId}`;
                    let errorToThrow = new Error("ResponseCancelledError");

                    if (reason === "TurnTimeout") {
                        errorMessage = "Bot turn timed out";
                        errorCode = "TIMED_OUT";
                        logMessage = `Bot turn timed out during loop for bot ${bot.id} in chat ${chatId}`;
                        errorToThrow = new Error("ResponseTimeoutError");
                    } else if (reason instanceof Error) {
                        errorMessage = reason.message;
                    } else if (typeof reason === "string" && reason.length > 0) {
                        errorMessage = reason;
                    }
                    logger.info(logMessage + (abortSignal.reason ? `. Reason: ${String(abortSignal.reason)}` : ""));
                    if (chatId) {
                        SocketService.get().emitSocketEvent("responseStream", chatId, {
                            __type: "error",
                            botId: bot.id,
                            error: { message: errorMessage, code: errorCode },
                        });
                        SocketService.get().emitSocketEvent("botStatusUpdate", chatId, {
                            chatId,
                            botId: bot.id,
                            status: "error_internal",
                            error: { message: errorMessage, code: errorCode },
                        });
                    }
                    throw errorToThrow;
                }

                const chosenModel = model ?? bot.config?.model ?? "";

                // Calculate effective max credits for the upcoming LLM call
                // It should be the lesser of the per-response cap and the bot's remaining allocated credits for this runLoop
                const remainingAllocatedCreditsForRun = convoAllocatedCredits - responseStats.creditsUsed;
                const capFromResponseLimits = BigInt(responseLimits.maxCreditsPerBotResponse);

                let effectiveMaxCreditsForLlm: bigint;
                if (remainingAllocatedCreditsForRun <= BigInt(0)) {
                    effectiveMaxCreditsForLlm = BigInt(0); // No budget left in allocation
                } else {
                    // Use the smaller of the two positive budget values
                    if (capFromResponseLimits < remainingAllocatedCreditsForRun) {
                        effectiveMaxCreditsForLlm = capFromResponseLimits;
                    } else {
                        effectiveMaxCreditsForLlm = remainingAllocatedCreditsForRun;
                    }
                }

                // Check for cancellation before LLM call
                if (abortSignal?.aborted) {
                    const reason = abortSignal.reason;
                    let errorMessage = "Response cancelled";
                    let errorCode = "CANCELLED";
                    let logMessage = `Response cancelled by signal before LLM call for bot ${bot.id} in chat ${chatId}`;
                    let errorToThrow = new Error("ResponseCancelledError");

                    if (reason === "TurnTimeout") {
                        errorMessage = "Bot turn timed out";
                        errorCode = "TIMED_OUT";
                        logMessage = `Bot turn timed out before LLM call for bot ${bot.id} in chat ${chatId}`;
                        errorToThrow = new Error("ResponseTimeoutError");
                    } else if (reason instanceof Error) {
                        errorMessage = reason.message;
                    } else if (typeof reason === "string" && reason.length > 0) {
                        errorMessage = reason;
                    }
                    logger.info(logMessage + (abortSignal.reason ? `. Reason: ${String(abortSignal.reason)}` : ""));
                    if (chatId) {
                        SocketService.get().emitSocketEvent("responseStream", chatId, {
                            __type: "error",
                            botId: bot.id,
                            error: { message: errorMessage, code: errorCode },
                        });
                        // No separate botStatusUpdate here as the stream error implies overall failure for this attempt.
                    }
                    throw errorToThrow;
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

                const nextInputs: any[] = [];
                for await (const ev of stream) {
                    switch (ev.type) {
                        case "message":
                            // Append to draft message
                            draftMessage = ev.final ? ev.content : draftMessage + ev.content;
                            // Emit to client
                            if (chatId) {
                                SocketService.get().emitSocketEvent("responseStream", chatId, {
                                    __type: ev.final ? "end" : "stream",
                                    botId: bot.id,
                                    chunk: ev.content,
                                });
                            }
                            break;
                        case "reasoning":
                            // Emit to client
                            if (chatId) {
                                SocketService.get().emitSocketEvent("modelReasoningStream", chatId, {
                                    __type: "stream",
                                    botId: bot.id,
                                    chunk: ev.content,
                                });
                            }
                            break;
                        case "function_call": {
                            // Check for cancellation before tool call
                            if (abortSignal?.aborted) {
                                const reason = abortSignal.reason;
                                let errorMessage = "Tool call cancelled";
                                let errorCode = "CANCELLED";
                                let logMessage = `Tool call ${ev.name} cancelled by signal for bot ${bot.id} in chat ${chatId}`;
                                let errorToThrow = new Error("ResponseCancelledError"); // Or a specific ToolCallCancelledError

                                if (reason === "TurnTimeout") {
                                    errorMessage = `Tool call ${ev.name} aborted due to turn timeout`;
                                    errorCode = "TIMED_OUT";
                                    logMessage = `Tool call ${ev.name} aborted due to turn timeout for bot ${bot.id} in chat ${chatId}`;
                                    errorToThrow = new Error("ResponseTimeoutError");
                                } else if (reason instanceof Error) {
                                    errorMessage = reason.message;
                                } else if (typeof reason === "string" && reason.length > 0) {
                                    errorMessage = reason;
                                }
                                logger.info(logMessage + (abortSignal.reason ? `. Reason: ${String(abortSignal.reason)}` : ""));

                                if (chatId) {
                                    SocketService.get().emitSocketEvent("botStatusUpdate", chatId, {
                                        chatId,
                                        botId: bot.id,
                                        status: "tool_failed",
                                        toolInfo: { callId: ev.callId, name: ev.name, error: errorMessage },
                                        error: { message: errorMessage, code: errorCode },
                                    });
                                }
                                throw errorToThrow;
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
                                    // Ensure __version and other MessageConfig defaults are included if necessary
                                    // For simplicity, assuming MessageConfig.default() handles this or it's added later.
                                    // Let's use a minimal config for now and refine if MessageState creation fails.
                                    __version: "1.0.0", // Example, ensure this aligns with MessageConfig
                                },
                                text: JSON.stringify(toolResult), // Content is the stringified output of the tool
                                language: DEFAULT_LANGUAGE, // Or derive from context
                                parent: previousResponseId ? { id: previousResponseId } : null, // Should this link to the assistant's msg that made the call?
                                user: { id: bot.id }, // Or a generic system/tool user ID
                                // Ensure all other potentially required fields of MessageState are populated
                            } as MessageState; // Cast to MessageState

                            nextInputs.push(toolResponseMessage);
                            toolCalls.push(functionCallEntry);
                            break;
                        }

                        case "done": {
                            const cost = BigInt(ev.cost);
                            responseStats.creditsUsed += cost; // Add LLM generation cost to cumulative response stats
                            // Emit to client
                            if (chatId) {
                                SocketService.get().emitSocketEvent("responseStream", chatId, {
                                    __type: "end",
                                    botId: bot.id,
                                    finalMessage: draftMessage,
                                });
                            }
                            break;
                        }
                        // Example of how to handle a conceptual error event from the stream
                        // This assumes the llmRouter.stream() can yield an event like { type: "error", errorDetails: { message: string, code?: string } }
                        /*
                        case "error": { // Hypothetical error event from stream
                            responseStats.creditsUsed += BigInt(ev.cost || 0); // Add any cost associated with the error event
                            if (chatId) {
                                SocketService.get().emitSocketEvent("responseStream", chatId, {
                                    __type: "error",
                                    botId: bot.id,
                                    error: {
                                        message: ev.errorDetails?.message || "Unknown stream error",
                                        code: ev.errorDetails?.code || "LLM_STREAM_ERROR",
                                        details: JSON.stringify(ev.errorDetails)
                                    }
                                });
                                // Also consider for modelReasoningStream if applicable
                                SocketService.get().emitSocketEvent("modelReasoningStream", chatId, {
                                    __type: "error",
                                    botId: bot.id,
                                    error: {
                                        message: ev.errorDetails?.message || "Unknown reasoning stream error",
                                        code: ev.errorDetails?.code || "REASONING_STREAM_ERROR",
                                        details: JSON.stringify(ev.errorDetails)
                                    }
                                });
                            }
                            // Potentially throw an error here to stop the runLoop or set a flag
                            // For now, just emitting and breaking
                            inputs = []; // Stop further processing in the loop
                            break;
                        }
                        */
                    }
                    previousResponseId = ev.responseId;
                }

                // Prepare for next iteration
                inputs = nextInputs;
            }

            // Emit processing_complete status if the loop finished without exhausting inputs (meaning bot did its work for this turn)
            if (chatId && inputs.length === 0) { // Check if inputs are now empty, meaning the bot processed its turn
                SocketService.get().emitSocketEvent("botStatusUpdate", chatId, {
                    chatId,
                    botId: bot.id,
                    status: "processing_complete",
                    message: "Finished processing turn.",
                });
            }

        } finally {
            // Send cost incurred event 
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
                },
            });

            // Signal typing end
            if (chatId) {
                SocketService.get().emitSocketEvent("typing", chatId, { stopping: [bot.id] });
            }
        }

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
            user: { id: bot.id },
        };

        // Mark turn completion time
        if (!config.stats) {
            config.stats = {
                totalToolCalls: 0,
                totalCredits: "0",
                startedAt: Date.now(),
                lastProcessingCycleEndedAt: null,
            } as ChatConfigObject["stats"];
        }
        config.stats.lastProcessingCycleEndedAt = Date.now();

        return { finalMessage: responseMessage, responseStats };
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
            const reason = abortSignal.reason;
            let errorMessage = `Tool call ${ev.name} cancelled`;
            let errorCode = "CANCELLED";
            let logMessage = `Tool call ${ev.name} cancelled by signal before execution for bot ${callerBot.id}`;
            let errorToThrow = new Error("ToolCallCancelledBySignal");

            if (reason === "TurnTimeout") {
                errorMessage = `Tool call ${ev.name} aborted due to turn timeout`;
                errorCode = "TIMED_OUT";
                logMessage = `Tool call ${ev.name} aborted due to turn timeout before execution for bot ${callerBot.id}`;
                errorToThrow = new Error("ResponseTimeoutError");
            } else if (reason instanceof Error) {
                errorMessage = reason.message;
            } else if (typeof reason === "string" && reason.length > 0) {
                errorMessage = reason;
            }

            logger.info(logMessage + (abortSignal.reason ? `. Reason: ${String(abortSignal.reason)}` : ""));

            if (conversationId) {
                SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                    chatId: conversationId,
                    botId: callerBot.id,
                    status: "tool_failed",
                    toolInfo: { callId: ev.callId, name: ev.name, error: errorMessage },
                    error: { message: errorMessage, code: errorCode },
                });
            }
            throw errorToThrow;
        }

        const toolName = ev.name;
        // Use getEffectiveScheduling to ensure validated and default-applied rules are used.
        const effectiveScheduling = config ? new ChatConfig({ config }).getEffectiveScheduling() : ChatConfig.defaultScheduling();
        const schedulingRules = effectiveScheduling;

        let requiresApproval = false;
        let requiresSchedulingOnly = false; // True if delayed but no explicit approval needed
        let specificDelayMs: number | undefined;

        // Determine if the tool call requires approval or is subject to a configured delay.
        // - If `schedulingRules.requiresApprovalTools` includes the tool (or is "all"),
        //   the tool call is marked as PENDING_APPROVAL.
        // - Otherwise, if `schedulingRules.toolSpecificDelays` or `defaultDelayMs`
        //   specifies a delay, the tool call is marked as SCHEDULED_FOR_EXECUTION.
        // - If neither approval nor a specific delay is configured, the tool executes immediately.
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
            if (config) { // Only log if config itself is present, as schedulingRules is part of config
                logger.warn(`Scheduling rules are undefined for tool call ${toolName} in conversation ${conversationId}, but ChatConfigObject was provided. Tool will execute immediately if not requiring approval otherwise.`, { toolName, conversationId, callerBotId: callerBot.id });
            }
        }

        if (requiresApproval || requiresSchedulingOnly) {
            const pendingId = nanoid();
            const now = Date.now();
            let currentStatus: PendingToolCallStatus | null = null;
            let scheduledExecutionTime: number | undefined;
            let approvalTimeoutTimestamp: number | undefined;

            if (requiresApproval) {
                currentStatus = PendingToolCallStatus.PENDING_APPROVAL;
                approvalTimeoutTimestamp = now + (schedulingRules?.approvalTimeoutMs || MINUTES_5_MS);
            } else if (requiresSchedulingOnly && specificDelayMs) {
                currentStatus = PendingToolCallStatus.SCHEDULED_FOR_EXECUTION;
                scheduledExecutionTime = now + specificDelayMs;
                // TODO-SCHEDULED-TOOL: Implement job queue integration (e.g., BullMQ) for SCHEDULED_FOR_EXECUTION.
                // This involves: 
                // 1. Defining a new job type for the BullMQ.
                // 2. Enqueueing a job here with `pendingId`, `scheduledExecutionTime`, and necessary context.
                // 3. Creating a worker process to pick up these jobs from the queue.
                // 4. The worker would then likely trigger an internal event (e.g., "ScheduledToolExecutionDue") 
                //    or directly call a method in CompletionService to execute the tool and re-engage the bot.
                // 5. Ensure robust error handling, retries, and logging for the queue and worker.
            } else {
                logger.warn(`Tool call ${toolName} for bot ${callerBot.id} entered scheduling block without clear status. Defaulting to immediate execution.`, { ev });
            }

            if (currentStatus === PendingToolCallStatus.PENDING_APPROVAL || currentStatus === PendingToolCallStatus.SCHEDULED_FOR_EXECUTION) {
                // 3. Create a PendingToolCallEntry object
                const pendingEntryForConfig: PendingToolCallEntry = {
                    pendingId,
                    toolCallId: ev.callId,
                    toolName,
                    toolArguments: JSON.stringify(ev.arguments),
                    callerBotId: callerBot.id,
                    conversationId: conversationId || "unknown_conversation",
                    requestedAt: now,
                    status: currentStatus,
                    scheduledExecutionTime,
                    approvalTimeoutAt: approvalTimeoutTimestamp,
                    userIdToApprove: userData?.id,
                    executionAttempts: 0,
                    // Note: statusReason, approvedOrRejectedByUserId, and decisionTime are typically set
                    // by the separate process that handles the approval/rejection or timeout event for this pending call,
                    // not at the time of initial deferral.
                };

                // 4. Store the PendingToolCallEntry in ChatConfigObject.
                if (config) {
                    config.pendingToolCalls = config.pendingToolCalls || [];
                    config.pendingToolCalls.push(pendingEntryForConfig); // Use the correctly typed object
                    logger.info(`Tool call ${toolName} (${ev.callId}) for bot ${callerBot.id} deferred and added to ChatConfig. Status: ${pendingEntryForConfig.status}`, { pendingEntry: pendingEntryForConfig });
                } else {
                    logger.error(`Cannot defer tool call ${toolName} (${ev.callId}) for bot ${callerBot.id}: ChatConfigObject is undefined.`);
                }

                // 5. Emit a socket event to the client if approval is needed.
                if (currentStatus === PendingToolCallStatus.PENDING_APPROVAL && conversationId && userData) {
                    const hasActiveConnection = SocketService.get().roomHasOpenConnections(conversationId);

                    if (hasActiveConnection) {
                        SocketService.get().emitSocketEvent("tool_approval_required", conversationId, {
                            pendingId,
                            toolCallId: ev.callId,
                            toolName,
                            toolArguments: ev.arguments as Record<string, any>,
                            callerBotId: callerBot.id,
                            callerBotName: callerBot.name, // Added callerBotName
                            approvalTimeoutAt: approvalTimeoutTimestamp,
                            estimatedCost: availableTools.find(t => t.name === toolName)?.estimatedCost, // Use the retrieved estimatedCost
                        });
                    } else {
                        Notify(userData.languages) // Pass user's languages for i18n
                            .pushToolApprovalRequired(
                                conversationId,
                                pendingId,
                                toolName,
                                callerBot.name, // Changed from callerBot.id to callerBot.name
                                config?.goal || "your Vrooli task", // conversationName: Use chat goal or a generic fallback
                                approvalTimeoutTimestamp,
                                availableTools.find(t => t.name === toolName)?.estimatedCost, // Use the retrieved estimatedCost
                                schedulingRules?.autoRejectOnTimeout,
                            )
                            .toUser(userData.id);
                        logger.info(`Sent tool approval push notification for ${pendingId} to user ${userData.id} for conversation ${conversationId}.`);
                    }

                    SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                        chatId: conversationId,
                        botId: callerBot.id,
                        status: "tool_pending_approval",
                        toolInfo: { callId: ev.callId, name: toolName, pendingId },
                        message: `Tool ${toolName} is awaiting user approval.`,
                    });
                }

                // 6. Return a placeholder result to the LLM indicating deferral.
                const deferredToolResult = {
                    __vrooli_tool_deferred: true,
                    pendingId,
                    status: currentStatus,
                    message: currentStatus === PendingToolCallStatus.PENDING_APPROVAL
                        ? `Tool call ${toolName} is awaiting user approval. You will be notified of the outcome.`
                        : `Tool call ${toolName} has been scheduled for later execution. You will be notified upon completion.`,
                };
                const functionCallEntryDeferred: ToolFunctionCall = {
                    id: ev.callId,
                    function: { name: toolName, arguments: JSON.stringify(ev.arguments) },
                    // Indicate success to LLM for this step (deferral was successful), but output shows it's deferred.
                    result: { success: true, output: JSON.stringify(deferredToolResult) },
                };
                // Cost for deferring a tool call is currently BigInt(0).
                // The actual cost of the tool will be incurred upon its execution.
                // This could be revisited to add a nominal scheduling fee if needed in the future (not likely, but you never know).
                return { toolResult: deferredToolResult, functionCallEntry: functionCallEntryDeferred, cost: BigInt(0) };
            }
        }

        // If not deferred (or fell through scheduling logic), proceed with immediate execution
        const args = ev.arguments as Record<string, any>;
        const isAsync = args.isAsync === true;
        const fnEntryBase = {
            id: ev.callId,
            function: { name: ev.name, arguments: JSON.stringify(ev.arguments) },
        } as Omit<ToolFunctionCall, "result">;
        let toolCost = BigInt(0);

        // Emit tool_calling status BEFORE deciding to defer or execute directly
        if (conversationId) {
            SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                chatId: conversationId,
                botId: callerBot.id,
                status: "tool_calling",
                toolInfo: { callId: ev.callId, name: ev.name, args: JSON.stringify(ev.arguments) },
                message: `Using tool: ${ev.name}`,
            });
        }

        let toolCallResponse: any;
        let entry: ToolFunctionCall;
        let output: unknown = null;

        if (isAsync) {
            // For async tool initiation, the McpToolRunner should ideally return the initiation cost.
            // This part is more conceptual as the actual async handling might be more complex
            // and involve the registry providing this initiation cost.
            toolCallResponse = await this.toolRunner.run(ev.name, ev.arguments, { conversationId: conversationId || "", callerBotId: callerBot.id, signal: abortSignal });

            if (toolCallResponse.ok) {
                output = toolCallResponse.data.output; // Should be an ack like { status: "started", callId: ev.callId }
                toolCost = BigInt(toolCallResponse.data.creditsUsed); // Initiation cost from ToolRunner
                entry = { ...fnEntryBase, result: { success: true, output } };
                if (conversationId) {
                    SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                        chatId: conversationId,
                        botId: callerBot.id,
                        status: "tool_completed", // Assuming async start is a form of completion for this event
                        toolInfo: { callId: ev.callId, name: ev.name, result: JSON.stringify(output) },
                    });
                }
            } else {
                // Async initiation failed
                output = toolCallResponse.error;
                toolCost = toolCallResponse.error.creditsUsed ? BigInt(toolCallResponse.error.creditsUsed) : BigInt(0); // Cost of failed initiation
                entry = { ...fnEntryBase, result: { success: false, error: toolCallResponse.error } };
                if (conversationId) {
                    SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                        chatId: conversationId,
                        botId: callerBot.id,
                        status: "tool_failed",
                        toolInfo: { callId: ev.callId, name: ev.name, error: JSON.stringify(toolCallResponse.error) },
                    });
                }
            }
            return { toolResult: output, functionCallEntry: entry, cost: toolCost };
        } else {
            // Synchronous execution
            toolCallResponse = await this.toolRunner.run(ev.name, ev.arguments, { conversationId: conversationId || "", callerBotId: callerBot.id, signal: abortSignal });

            if (toolCallResponse.ok) {
                output = toolCallResponse.data.output;
                toolCost = BigInt(toolCallResponse.data.creditsUsed);
                entry = { ...fnEntryBase, result: { success: true, output: toolCallResponse.data.output } };
                if (conversationId) {
                    SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                        chatId: conversationId,
                        botId: callerBot.id,
                        status: "tool_completed",
                        toolInfo: { callId: ev.callId, name: ev.name, result: JSON.stringify(output) },
                    });
                }
            } else {
                // Synchronous tool call failed
                output = toolCallResponse.error; // Store the error object as the output for the LLM to see
                toolCost = toolCallResponse.error.creditsUsed ? BigInt(toolCallResponse.error.creditsUsed) : BigInt(0); // Use cost from error if available
                entry = { ...fnEntryBase, result: { success: false, error: toolCallResponse.error } };
                if (conversationId) {
                    SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                        chatId: conversationId,
                        botId: callerBot.id,
                        status: "tool_failed",
                        toolInfo: { callId: ev.callId, name: ev.name, error: JSON.stringify(toolCallResponse.error) },
                    });
                }
            }
            return { toolResult: output, functionCallEntry: entry, cost: toolCost };
        }
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
    ) { }

    public async getConversationState(conversationId: string): Promise<ConversationState | null> {
        return this.conversationStore.get(conversationId);
    }

    public updateConversationConfig(conversationId: string, config: ChatConfigObject): void {
        this.conversationStore.updateConfig(conversationId, config);
    }

    /**
     * Builds the system message for a bot based on its role and the current context.
     */
    private async _buildSystemMessage(
        goal: string,
        bot: BotParticipant,
        convoConfig: ChatConfigObject,
    ): Promise<string> {
        const appName = "Vrooli";
        const appDescription = "a polymorphic, collaborative, and self-improving automation platform that helps you stay organized and achieve your goals.";
        let baseSystemMessage = `Welcome to ${appName}, ${appDescription}.\\n\\n`;

        const botRole = bot.meta?.role || "leader"; // Default to leader if no role
        const botId = bot.id;

        const __filename = fileURLToPath(import.meta.url);
        const __dirname = path.dirname(__filename);
        const promptPath = path.join(__dirname, "prompt1.txt"); // Changed to prompt1.txt
        let roleSpecificTemplate = "";

        // Define the Recruitment Rule as a constant
        const recruitmentRule = `## Recruitment rule:
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

        let roleSpecificInstructions = "Perform tasks according to your role and the overall goal."; // Default
        if (botRole === "leader" || botRole === "coordinator" || botRole === "delegator") {
            roleSpecificInstructions = recruitmentRule;
        }

        // TODO: Eventually, load prompts from a dedicated /prompts/{role}.txt directory or database.
        // The logic for loading different files based on role is removed for now, as prompt1.txt is the base.
        // We will handle role-specific content through placeholders like {{ROLE_SPECIFIC_INSTRUCTIONS}}.

        try {
            logger.info(`Loading base prompt template from: ${promptPath}`);
            roleSpecificTemplate = await fs.readFile(promptPath, "utf-8");
        } catch (error) {
            logger.error(`Failed to load prompt template from ${promptPath}:`, { error });
            roleSpecificTemplate = "Your primary goal is: {{GOAL}}. Please act according to your role: {{ROLE}}. Critical: Prompt template file not found.";
        }

        let memberCountLabel = "1 member";
        if (convoConfig.teamId) {
            memberCountLabel = "team-based swarm";
        }

        const builtInToolSchemas = this.toolRegistry.getBuiltInDefinitions();
        const swarmToolSchemas = this.toolRegistry.getSwarmToolDefinitions();
        const allToolSchemas = [...builtInToolSchemas, ...swarmToolSchemas];

        let processedPrompt = roleSpecificTemplate;
        processedPrompt = processedPrompt.replace(/{{GOAL}}/g, goal);
        processedPrompt = processedPrompt.replace(/{{MEMBER_COUNT_LABEL}}/g, memberCountLabel);
        processedPrompt = processedPrompt.replace(/{{ISO_EPOCH_SECONDS}}/g, Math.floor(Date.now() / SECONDS_1_MS).toString());
        processedPrompt = processedPrompt.replace(/{{DISPLAY_DATE}}/g, new Date().toLocaleString());
        processedPrompt = processedPrompt.replace(/{{ROLE\s*\|\s*upper}}/g, botRole.toUpperCase());
        processedPrompt = processedPrompt.replace(/{{ROLE}}/g, botRole);
        processedPrompt = processedPrompt.replace(/{{BOT_ID}}/g, botId);
        processedPrompt = processedPrompt.replace(/{{ROLE_SPECIFIC_INSTRUCTIONS}}/g, roleSpecificInstructions);

        const swarmStateString = this._buildSwarmStateString(convoConfig);
        processedPrompt = processedPrompt.replace(/{{SWARM_STATE}}/g, swarmStateString);

        const toolSchemasString = allToolSchemas.length > 0 ? JSON.stringify(allToolSchemas, null, 2) : "No tools available for this role.";
        processedPrompt = processedPrompt.replace(/{{TOOL_SCHEMAS}}/g, toolSchemasString);

        baseSystemMessage += processedPrompt;
        return baseSystemMessage.trim();
    }

    /**
     * Builds a string representation of the current swarm state for inclusion in the system prompt.
     * @param convoConfig The conversation configuration object containing swarm state.
     * @returns A formatted string of the swarm state, or a default message if unavailable.
     */
    private _buildSwarmStateString(convoConfig: ChatConfigObject | undefined): string {
        /** How long each swarm state section can be before we truncate it. */
        const MAX_STRING_PREVIEW_LENGTH = 2_000;
        let swarmStateOutput = "SWARM STATE DETAILS: Not available or error formatting state.";

        if (!convoConfig) {
            return "SWARM STATE DETAILS: Configuration not available.";
        }

        try {
            // Ensure these properties exist on convoConfig, even if they are empty arrays.
            const teamId = convoConfig.teamId || "No team"; //TODO should store team information, since it contains things like MOISE+ hierarchy, etc.
            const swarmLeader = convoConfig.swarmLeader || "No leader";
            const subtasks = convoConfig.subtasks || [];
            const subtaskLeaders = convoConfig.subtaskLeaders || {};
            const eventSubscriptions = convoConfig.eventSubscriptions || {};
            const blackboard = convoConfig.blackboard || [];
            const resources = convoConfig.resources || [];
            const records = convoConfig.records || [];
            const stats = convoConfig.stats || {};
            const limits = convoConfig.limits || {};

            const formattedSwarmStateParts: string[] = [];

            // Handle teamId
            formattedSwarmStateParts.push(`- Team ID:\n${teamId}\n`);

            // Handle swarmLeader
            formattedSwarmStateParts.push(`- Swarm Leader:\n${swarmLeader}\n`);

            // Handle subtasks
            const activeSubtasksCount = subtasks.filter(st => typeof st === "object" && st !== null && (st.status === "todo" || st.status === "in_progress")).length;
            const completedSubtasksCount = subtasks.filter(st => typeof st === "object" && st !== null && st.status === "done").length;
            const subtasksString = JSON.stringify(subtasks, null, 2);
            const truncatedSubtasksString = subtasksString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (subtasksString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Subtasks (active: ${activeSubtasksCount}, completed: ${completedSubtasksCount}):\n${truncatedSubtasksString}\n`);

            // Handle subtaskLeaders
            const subtaskLeadersString = JSON.stringify(subtaskLeaders, null, 2);
            const truncatedSubtaskLeadersString = subtaskLeadersString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (subtaskLeadersString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Subtask Leaders:\n${truncatedSubtaskLeadersString}\n`);

            // Handle eventSubscriptions
            const eventSubscriptionsString = JSON.stringify(eventSubscriptions, null, 2);
            const truncatedEventSubscriptionsString = eventSubscriptionsString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (eventSubscriptionsString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Event Subscriptions:\n${truncatedEventSubscriptionsString}\n`);

            // Handle blackboard
            const blackboardString = JSON.stringify(blackboard, null, 2);
            const truncatedBlackboardString = blackboardString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (blackboardString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Blackboard:\n${truncatedBlackboardString}\n`);

            // Handle resources
            const resourcesString = JSON.stringify(resources, null, 2);
            const truncatedResourcesString = resourcesString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (resourcesString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Resources:\n${truncatedResourcesString}\n`);

            // Handle records
            const recordsString = JSON.stringify(records, null, 2);
            const truncatedRecordsString = recordsString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (recordsString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Records:\n${truncatedRecordsString}\n`);

            // Handle stats
            const statsString = JSON.stringify(stats, null, 2);
            const truncatedStatsString = statsString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (statsString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Stats:\n${truncatedStatsString}\n`);

            // Handle limits
            const limitsString = JSON.stringify(limits, null, 2);
            const truncatedLimitsString = limitsString.substring(0, MAX_STRING_PREVIEW_LENGTH) + (limitsString.length > MAX_STRING_PREVIEW_LENGTH ? "..." : "");
            formattedSwarmStateParts.push(`- Limits:\n${truncatedLimitsString}\n`);

            swarmStateOutput = `\nSWARM STATE DETAILS:\n${formattedSwarmStateParts.join("\n\n")}`;
        } catch (e) {
            logger.error("Error formatting swarm state for prompt:", e);
            // Fallback to the default message initialized if an error occurs
        }
        return swarmStateOutput;
    }

    /**
     * Public wrapper for generating a system message for a specific bot.
     */
    public async generateSystemMessageForBot(goal: string, bot: BotParticipant, convoConfig: ChatConfigObject): Promise<string> {
        return this._buildSystemMessage(goal, bot, convoConfig);
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
                const botSystemMessage = await this._buildSystemMessage(goal, responder, conversationState.config);

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
        currentConfigStats: ChatConfigObject["stats"],
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
                        const botSystemMessage = await this._buildSystemMessage(specificGoal, responder, conversationState.config);
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
                        const botSystemMessage = await this._buildSystemMessage(goalForEvent, responder, conversationState.config);
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
                            const botSystemMessage = await this._buildSystemMessage(goalForEvent, callerBot, conversationState.config);
                            const botResponse = await this.reasoningEngine.runLoop(
                                { id: toolResponseMessage.id }, botSystemMessage, fullAvailableToolsForEvent, callerBot, creditAccountId,
                                conversationState.config, reEngageAllocatedToolCalls, reEngageAllocatedCredits, event.sessionUser,
                                conversationId, undefined, controller.signal,
                            );
                            results.push(botResponse.finalMessage);
                            currentResponseStats.push(botResponse.responseStats);
                        } catch (reEngageError: any) {
                            logger.error(`Error re-engaging bot ${callerBot.id} after approved tool execution`, { error: reEngageError });
                            if (conversationId) {
                                SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                                    chatId: conversationId, botId: callerBot.id, status: "error_internal",
                                    error: { message: reEngageError.message || "Failed to process tool result", code: "REENGAGE_FAILURE" },
                                });
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
                            const botSystemMessage = await this._buildSystemMessage(goalForEvent, callerBot, conversationState.config);
                            const botResponse = await this.reasoningEngine.runLoop(
                                { id: toolRejectionMessage.id }, botSystemMessage, fullAvailableToolsForEvent, callerBot, creditAccountId,
                                conversationState.config, reEngageAllocatedToolCalls, reEngageAllocatedCredits, event.sessionUser,
                                conversationId, undefined, controller.signal,
                            );
                            results.push(botResponse.finalMessage);
                            currentResponseStats.push(botResponse.responseStats);
                        } catch (reEngageError: any) {
                            logger.error(`Error re-engaging bot ${callerBot.id} after tool rejection`, { error: reEngageError });
                            if (conversationId) {
                                SocketService.get().emitSocketEvent("botStatusUpdate", conversationId, {
                                    chatId: conversationId, botId: callerBot.id, status: "error_internal",
                                    error: { message: reEngageError.message || "Failed to process tool rejection", code: "REENGAGE_FAILURE" },
                                });
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

type State = (typeof SwarmStateMachine.State)[keyof typeof SwarmStateMachine.State];

/**
 * SwarmStateMachine
 * -----------------------------------------------------------------------------
 * Coordinates a virtual "swarm" of agents collaborating via a shared conversation.
 * It manages the swarm's lifecycle states and dispatches agent actions (e.g., sub‑goals,
 * tool invocations) while recording conversation‑level context (sub‑goals, outcomes).
 *
 * This class is "permissive by design": bots negotiate *how* to share memory
 * (e.g. scratchpad messages, JSON blobs) and *how* to parcel work among
 * themselves.  The world‑model seeded at initialization explains the available
 * tools (including **routines**) and encourages agents to externalize complex
 * coordination by invoking or creating reusable routines.
 *
 * Anti‑framework philosophy – keep the swarm loop mostly sequential and
 * trivial to reason about.  Whenever agents hit a task that needs branching,
 * retries, or heavy parallelism, they should call a routine (synchronously or
 * asynchronously) and resume once results are ready.
 *
 * Lifecycle States:
 *   1. **UNINITIALIZED** – no swarm context created yet
 *   2. **STARTING**      – preparing to start
 *   3. **RUNNING**       – processing events / generating sub‑goals
 *   4. **PAUSED**        – paused by user / system
 *   5. **STOPPED**       – stopped by user
 *   6. **FAILED**        – unrecoverable error
 *   7. **TERMINATED**    – shut down; resources released
 *
 * Transitions:
 *   • UNINITIALIZED → STARTING  on start()
 *   • STARTING → RUNNING        on start()
 *   • RUNNING ↔ PAUSED          on pause/resume
 *   • RUNNING → STOPPED         on stop()
 *   • * → FAILED                on fatal error
 *   • * → TERMINATED            on shutdown()
 *
 * Responsibilities:
 *   • Maintain shared conversation context (sub‑goals, outcomes)
 *   • Dispatch agents via ConversationService
 *   • Queue asynchronous tool events until relevant agent is ready
 *   • Publish pause / resume / shutdown signals
 *
 * Routine orchestration lives entirely in **RoutineStateMachine**.
 */
export class SwarmStateMachine implements ManagedTaskStateMachine {
    private disposed = false;
    private processingLock = false;
    static State = {
        UNINITIALIZED: "UNINITIALIZED",
        STARTING: "STARTING",
        RUNNING: "RUNNING",
        IDLE: "IDLE",
        PAUSED: "PAUSED",
        STOPPED: "STOPPED",
        FAILED: "FAILED",
        TERMINATED: "TERMINATED",
    } as const;
    private state: State = SwarmStateMachine.State.UNINITIALIZED;
    private readonly eventQueue: SwarmEvent[] = [];
    private conversationId: string | null = null; // To store convoId for getTaskId
    private initiatingUser: SessionUser | null = null; // To store initiatingUser for getAssociatedUserId

    constructor(
        private readonly completion: CompletionService,
    ) { }

    // Implementation of ManagedTaskStateMachine methods
    public getTaskId(): string {
        if (!this.conversationId) {
            // This should ideally not happen if 'start' is called correctly
            logger.error("SwarmStateMachine: getTaskId called before conversationId was set.");
            return "undefined_swarm_task_id";
        }
        return this.conversationId;
    }

    public getCurrentSagaStatus(): string {
        // Map internal state to standardized status
        // Example mapping, adjust as needed for exact semantics
        switch (this.state) {
            case SwarmStateMachine.State.UNINITIALIZED:
                return "UNINITIALIZED"; // Or perhaps a more specific pre-start status
            case SwarmStateMachine.State.STARTING:
                return "STARTING";
            case SwarmStateMachine.State.RUNNING:
                return "RUNNING";
            case SwarmStateMachine.State.IDLE:
                return "IDLE";
            case SwarmStateMachine.State.PAUSED:
                return "PAUSED";
            case SwarmStateMachine.State.STOPPED:
                return "STOPPED"; // Or map to COMPLETED if that's the intent
            case SwarmStateMachine.State.FAILED:
                return "FAILED";
            case SwarmStateMachine.State.TERMINATED:
                return "TERMINATED"; // Or COMPLETED
            default:
                logger.warn(`SwarmStateMachine: Unknown state encountered in getCurrentSagaStatus: ${this.state}`);
                return "UNKNOWN";
        }
    }

    public async requestPause(): Promise<boolean> {
        if (this.state === SwarmStateMachine.State.RUNNING || this.state === SwarmStateMachine.State.IDLE) {
            await this.pause(); // Assuming pause is async or has async implications
            return true;
        }
        logger.warn(`SwarmStateMachine: requestPause called in non-pausable state: ${this.state}`);
        return false;
    }

    public async requestStop(reason: string): Promise<boolean> {
        logger.info(`SwarmStateMachine: requestStop called for ${this.conversationId}. Reason: ${reason}`);
        // shutdown seems to be the closest equivalent. Consider if a different state like 'STOPPED' is needed before TERMINATED.
        await this.shutdown();
        return true; // Assuming shutdown always attempts to stop and terminate
    }

    public getAssociatedUserId(): string | undefined {
        return this.initiatingUser?.id;
    }
    // End of ManagedTaskStateMachine methods

    async start(convoId: string, goal: string, initiatingUser: SessionUser): Promise<void> {
        if (this.state !== SwarmStateMachine.State.UNINITIALIZED) {
            logger.warn(`SwarmStateMachine for ${convoId} already started. Current state: ${this.state}`);
            return;
        }
        this.conversationId = convoId; // Store convoId
        this.initiatingUser = initiatingUser; // Store initiatingUser
        this.state = SwarmStateMachine.State.STARTING;
        logger.info(`Starting SwarmStateMachine for ${convoId} with goal: "${goal}"`);

        const convoState = await this.completion.getConversationState(convoId);
        if (!convoState) {
            logger.error(`Failed to load ConversationState for ${convoId} in SwarmStateMachine.start`);
            this.state = SwarmStateMachine.State.FAILED;
            return;
        }

        let leaderBotParticipant: BotParticipant | undefined;
        const configuredLeaderId = convoState.config.swarmLeader;

        if (configuredLeaderId) {
            leaderBotParticipant = convoState.participants.find(p => p.id === configuredLeaderId);
            if (!leaderBotParticipant) {
                logger.warn(`Configured swarmLeader ID '${configuredLeaderId}' not found in participants for convo ${convoId}. Creating placeholder for initial system message.`);
                const placeholderBotConfig: BotConfigObject = { __version: "1.0", model: "default" };
                leaderBotParticipant = {
                    id: configuredLeaderId, // Use the configured ID
                    name: "Leader (Placeholder)",
                    config: placeholderBotConfig,
                    meta: { role: "leader" },
                };
            }
        } else if (convoState.participants.length > 0) {
            leaderBotParticipant = { ...convoState.participants[0] }; // Shallow copy to avoid modifying original participant's meta directly
            // Ensure meta exists before assigning role
            leaderBotParticipant.meta = { ...leaderBotParticipant.meta, role: "leader" };
            logger.warn(`No swarmLeader configured for convo ${convoId}. Using first participant '${leaderBotParticipant.id}' as fallback leader for initial system message.`);
        } else {
            logger.error(`CRITICAL: No swarmLeader configured and no participants found for convo ${convoId}. Swarm may not function correctly. Creating a temporary placeholder leader for system message generation.`);
            const placeholderBotConfig: BotConfigObject = { __version: "1.0", model: "default" };
            leaderBotParticipant = {
                id: generatePK().toString(), // Last resort: generate a temporary ID
                name: "Leader (Emergency Placeholder)",
                config: placeholderBotConfig,
                meta: { role: "leader" },
            };
        }

        const systemMessageString = await this.completion.generateSystemMessageForBot(
            goal,
            leaderBotParticipant,
            convoState.config,
        );

        convoState.initialLeaderSystemMessage = systemMessageString;

        const configToUpdate = convoState.config;
        configToUpdate.goal = goal;
        configToUpdate.subtasks = configToUpdate.subtasks ?? [];
        configToUpdate.blackboard = configToUpdate.blackboard ?? [];
        configToUpdate.resources = configToUpdate.resources ?? [];
        configToUpdate.stats = configToUpdate.stats ?? ChatConfig.defaultStats();

        this.completion.updateConversationConfig(convoId, configToUpdate);
        logger.info(`Updated and persisted ChatConfigObject for swarm ${convoId} with initial goal, system message, and ensured swarm fields.`);

        const startEvent: SwarmStartedEvent = {
            type: "swarm_started",
            conversationId: convoId,
            goal,
            sessionUser: initiatingUser,
        };
        await this.handleEvent(startEvent);
        this.state = SwarmStateMachine.State.IDLE;
    }

    async handleEvent(ev: SwarmEvent): Promise<void> {
        if (this.state === SwarmStateMachine.State.TERMINATED) return;
        this.eventQueue.push(ev);
        if (this.state === SwarmStateMachine.State.IDLE && ev.conversationId) {
            this.drain(ev.conversationId).catch(err => logger.error("Error draining swarm queue from handleEvent", { error: err, event: ev }));
        }
    }

    private async drain(convoId: string) {
        if (this.state === SwarmStateMachine.State.PAUSED || this.state === SwarmStateMachine.State.TERMINATED || this.state === SwarmStateMachine.State.FAILED) {
            logger.info(`Drain called in state ${this.state}, not processing queue for ${convoId}.`);
            return;
        }
        if (this.processingLock) {
            logger.info(`Drain already in progress for ${convoId}, skipping.`);
            return;
        }

        this.processingLock = true;
        this.state = SwarmStateMachine.State.RUNNING;

        if (this.eventQueue.length === 0) {
            this.state = SwarmStateMachine.State.IDLE;
            return;
        }
        const ev = this.eventQueue.shift()!;
        if (ev.conversationId !== convoId) {
            logger.warn("Swarm drain called with convoId mismatch", { expected: convoId, actual: ev.conversationId });
            this.eventQueue.unshift(ev);
            this.state = SwarmStateMachine.State.IDLE;
            return;
        }
        try {
            await this.completion.handleInternalEvent(ev);
        } catch (error) {
            logger.error("Error processing event in SwarmStateMachine drain", { error, event: ev });
        }

        if (this.eventQueue.length === 0) {
            this.state = SwarmStateMachine.State.IDLE;
        } else {
            setImmediate(() => this.drain(convoId).catch(err => logger.error("Error in scheduled subsequent drain", { error: err, conversationId: convoId })));
        }
    }

    async pause(): Promise<void> {
        if (this.state === SwarmStateMachine.State.RUNNING || this.state === SwarmStateMachine.State.IDLE) {
            this.state = SwarmStateMachine.State.PAUSED;
            logger.info(`SwarmStateMachine for ${this.conversationId} paused.`);
        }
    }

    async resume(convoId: string): Promise<void> {
        if (this.state === SwarmStateMachine.State.PAUSED) {
            this.state = SwarmStateMachine.State.IDLE;
            logger.info(`SwarmStateMachine for ${convoId} resumed. Draining event queue.`);
            await this.drain(convoId);
        }
    }

    async shutdown(): Promise<void> {
        this.disposed = true;
        this.eventQueue.length = 0;
        this.state = SwarmStateMachine.State.TERMINATED;
        logger.info("SwarmStateMachine shutdown complete.");
    }

    private fail(err: unknown) {
        logger.error(`Swarm failure for ${this.conversationId}`, { error: err });
        this.state = SwarmStateMachine.State.FAILED;
    }

    /**
     * Handles the scenario where a tool call, previously deferred for user approval,
     * has now been approved.
     * This method will queue an internal event to trigger the execution of the tool.
     * @param approvedToolCall - The PendingToolCallEntry that was approved.
     */
    public async handleToolApproval(approvedToolCall: PendingToolCallEntry): Promise<void> {
        if (!this.conversationId) {
            logger.error("SwarmStateMachine: handleToolApproval called before conversationId was set.", { approvedToolCall });
            return;
        }
        if (this.state === SwarmStateMachine.State.TERMINATED || this.state === SwarmStateMachine.State.FAILED) {
            logger.warn(`SwarmStateMachine for ${this.conversationId} is in a terminal state (${this.state}). Cannot process approved tool call ${approvedToolCall.pendingId}.`);
            return;
        }
        if (!this.initiatingUser) {
            logger.error(`SwarmStateMachine for ${this.conversationId}: initiatingUser is null. Cannot process approved tool call ${approvedToolCall.pendingId} without user context.`);
            return;
        }

        logger.info(`SwarmStateMachine for ${this.conversationId}: Queuing event for approved tool call ${approvedToolCall.pendingId} (Tool: ${approvedToolCall.toolName}).`);

        const toolExecutionEvent: SwarmEvent = {
            type: "ApprovedToolExecutionRequest",
            conversationId: this.conversationId,
            sessionUser: this.initiatingUser, // Use the stored initiating user
            payload: {
                pendingToolCall: approvedToolCall,
            },
        };

        // TODO web socket for tool approval here?

        SocketService.get().emitSocketEvent("botStatusUpdate", this.conversationId, {
            chatId: this.conversationId,
            botId: approvedToolCall.callerBotId,
            status: "tool_calling",
            toolInfo: {
                callId: approvedToolCall.toolCallId,
                name: approvedToolCall.toolName,
                pendingId: approvedToolCall.pendingId,
            },
            message: `Tool ${approvedToolCall.toolName} (ID: ${approvedToolCall.toolCallId}) was approved by user. Bot ${approvedToolCall.callerBotId} is now preparing for execution.`,
        });

        await this.handleEvent(toolExecutionEvent);
    }

    /**
     * Handles the scenario where a tool call, previously deferred for user approval,
     * has now been rejected.
     * This method will queue an internal event to notify the swarm.
     * @param rejectedToolCall - The PendingToolCallEntry that was rejected.
     * @param reason - Optional reason for rejection.
     */
    public async handleToolRejection(rejectedToolCall: PendingToolCallEntry, reason?: string): Promise<void> {
        if (!this.conversationId) {
            logger.error("SwarmStateMachine: handleToolRejection called before conversationId was set.", { rejectedToolCall });
            return;
        }
        if (this.state === SwarmStateMachine.State.TERMINATED || this.state === SwarmStateMachine.State.FAILED) {
            logger.warn(`SwarmStateMachine for ${this.conversationId} is in a terminal state (${this.state}). Cannot process rejected tool call ${rejectedToolCall.pendingId}.`);
            return;
        }
        if (!this.initiatingUser) {
            logger.error(`SwarmStateMachine for ${this.conversationId}: initiatingUser is null. Cannot process rejected tool call ${rejectedToolCall.pendingId} without user context.`);
            return;
        }

        logger.info(`SwarmStateMachine for ${this.conversationId}: Queuing event for rejected tool call ${rejectedToolCall.pendingId} (Tool: ${rejectedToolCall.toolName}). Reason: ${reason || "No reason provided"}`);

        const toolRejectionEvent: SwarmEvent = {
            type: "RejectedToolExecutionRequest",
            conversationId: this.conversationId,
            sessionUser: this.initiatingUser, // Use the stored initiating user
            payload: {
                pendingToolCall: rejectedToolCall,
                reason,
            },
        };
        // Emit a socket event to inform the UI/client about the rejection.
        // This is useful for providing immediate feedback to the user.
        SocketService.get().emitSocketEvent("tool_approval_rejected", this.conversationId, {
            pendingId: rejectedToolCall.pendingId,
            toolCallId: rejectedToolCall.toolCallId,
            toolName: rejectedToolCall.toolName,
            reason: reason || "No reason provided",
            callerBotId: rejectedToolCall.callerBotId, // Ensure callerBotId is on PendingToolCallEntry
        });

        SocketService.get().emitSocketEvent("botStatusUpdate", this.conversationId, {
            chatId: this.conversationId,
            botId: rejectedToolCall.callerBotId, // Assuming callerBotId is on PendingToolCallEntry
            status: "tool_rejected_by_user",
            toolInfo: {
                callId: rejectedToolCall.toolCallId,
                name: rejectedToolCall.toolName,
                pendingId: rejectedToolCall.pendingId,
                reason: reason || "No reason provided",
            },
            message: `Tool ${rejectedToolCall.toolName} was rejected by the user. Reason: ${reason || "Not specified"}`,
        });

        await this.handleEvent(toolRejectionEvent);
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

// SwarmStateMachine class definition must come before this instantiation
export const swarmStateMachine = new SwarmStateMachine(
    completionService,
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
