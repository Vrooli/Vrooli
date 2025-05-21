/* eslint-disable func-style */
import { ChatConfig, type ChatConfigObject, DEFAULT_LANGUAGE, generatePK, MessageConfig, type SessionUser, type SessionUserSession, type ToolFunctionCall } from "@local/shared";
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import type OpenAI from "openai";
import { logger } from "../../events/logger.js";
import { SocketService } from "../../sockets/io.js";
import { type LLMCompletionTask } from "../../tasks/taskTypes.js";
import { BusService, type EventBus } from "../bus.js";
import { type ToolInputSchema } from "../mcp/types.js";
import { type AgentGraph, CompositeGraph } from "./agentGraph.js";
import { CachedConversationStateStore, type ConversationStateStore, PrismaChatStore } from "./chatStore.js";
import { type ContextBuilder, RedisContextBuilder } from "./contextBuilder.js";
import { type MessageStore, RedisMessageStore } from "./messageStore.js";
import type { FunctionCallStreamEvent } from "./router.js";
import { FallbackRouter, type LlmRouter } from "./router.js";
import { CompositeToolRunner, type ToolRunner } from "./toolRunner.js";
import { type BotParticipant, type MessageState, type SwarmInternalEvent, type SwarmStartedEvent, type TurnStats } from "./types.js";
import { WorldModel } from "./worldModel.js";

// /** Maximum iterations for the drain loop, if something goes wrong with normal limit logic */
// const MAX_DRAIN_ITERATIONS = 1_000;

// /**
//  * Increment the tool call counter for the given turn and conversation config.
//  */
// function bumpToolStats(
//     turn: TurnStats,
//     convoCfg: ChatConfigObject,
//     amount = 1,
// ) {
//     turn.toolCalls += amount;      // whole-turn cap
//     turn.botToolCalls += amount;      // per-bot cap
//     convoCfg.stats.totalToolCalls += amount; // conversation-wide
// }

// /**
//  * Increment the total credits used for the given turn and conversation config.
//  */
// function bumpCreditStats(
//     turn: TurnStats,
//     convoCfg: ChatConfigObject,
//     amount: bigint,
// ) {
//     turn.creditsUsed += amount;
//     convoCfg.stats.totalCredits = (
//         BigInt(convoCfg.stats.totalCredits) + amount
//     ).toString();
// }

// TODO make sure that when messages/chats/particpants are added/updated/deleted (ModelLogic trigger), this class handles it correctly (including adding to context collector cache)
// TODO make sure that chat updates update or invalidate the conversation state cache
// TODO I don't think reduceUserCredits is being called in the right places (or anywhere). Need to carefully validate that the code reduces spent credits.
// TODO Tthe failure cases could be handled better. E.g. emitting event to show retry button
//TODO handle message updates (branching conversation)
// TODO make sure preferred model is stored in the conversation state
//TODO canceling responses
// TODO need turn timeouts
//TODO make sure all limiting logic is in place and functioning correctly
// TODO add mechanism for swarm and routine state machines to have a delay set by config that schedules any runs and resource mutations instead of running them immediately. This would be clearly defined in the context, and a list of schedules tool calls would be included in the context too. This enables the swarm to plan around the schedule, which is needed for users to approve/reject scheduled tool calls for safety reasons.
// TODO swarm world model should suggest starting a metacognition loop to best complete the goal, and adding to to the chat context. Can search for existing prompts.
//TODO conversation context should be able to store recommended routines to call for certain scenarios, pulled from the team config if available.


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
        private readonly toolRunner: ToolRunner,
    ) { }

    /**
     * Executes a single reasoning loop for a bot.
     * @param startMessage - Either an object with {id} to start from existing history, or {text} for standalone prompts
     * @param worldModel - The world model to use for the context window
     * @param availableTools - The tools available to the bot
     * @param bot - Information about the bot that's running the loop
     * @param creditAccountId - The credit account to charge for the response (tied to a user or team)
     * @param config - The conversation configuration
     * @param userData - Additional data related to the session
     * @param chatId - The chat this loop is for, if applicable
     * @param model - The model to use for the LLM call
     * @returns A MessageState object representing the final message from the bot
     */
    async runLoop(
        startMessage: { id: string } | { text: string },
        worldModel: WorldModel,
        availableTools: ToolInputSchema[],
        bot: BotParticipant,
        creditAccountId: string,
        config: ChatConfigObject,
        userData: SessionUser,
        chatId?: string,
        model?: string,
    ): Promise<MessageState> {
        // Signal typing start
        if (chatId) {
            SocketService.get().emitSocketEvent("typing", chatId, { starting: [bot.id] });
        }

        // Build context if we have a chatId + message ID, else wrap a text prompt
        let inputs: MessageState[];
        if (chatId && "id" in startMessage) {
            // Build context, reserving tokens for tools (none for now) and world-model
            const toolsForLLM: OpenAI.Responses.Tool[] = []; //TODO: add tools
            const { messages } = await this.contextBuilder.build(
                chatId,
                bot,
                model ?? DEFAULT_LANGUAGE,
                startMessage.id,
                { tools: toolsForLLM, world: worldModel },
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
        let tools: ToolInputSchema[] = availableTools;
        let draftMessage = "";
        // Collect all tool calls made during this loop
        const toolCalls: ToolFunctionCall[] = [];
        let previousResponseId: string | undefined;
        const turnStats: TurnStats = { toolCalls: 0, botToolCalls: 0, creditsUsed: 0n };

        // Compute limits
        const limits = new ChatConfig({ config }).getEffectiveLimits();
        // If the bot has a custom system prompt, merge it into the world model config
        if (bot.meta?.systemPrompt) {
            // Append bot-specific system prompt to existing system message
            const existing = worldModel.getConfig().systemMessage;
            worldModel.updateConfig({ systemMessage: `${existing}\n${bot.meta.systemPrompt}` });
        }
        const exceeded = () =>
            turnStats.toolCalls >= limits.maxToolCallsPerTurn ||
            turnStats.botToolCalls >= limits.maxToolCallsPerBotTurn ||
            turnStats.creditsUsed >= BigInt(limits.maxCreditsPerTurn);

        try {
            while (inputs.length && !exceeded()) {
                const chosenModel = model ?? bot.config?.model ?? "";
                // Convert our tool schemas to LLM-compatible tool definitions
                const toolSchemas: OpenAI.Responses.Tool[] = (tools as ToolInputSchema[]).map(ts => ({
                    name: ts.type,
                    description: "",
                    parameters: ts.properties,
                } as any)); //TODO fix type
                const stream = this.llmRouter.stream({
                    model: chosenModel,
                    previous_response_id: previousResponseId,
                    input: inputs,
                    tools: toolSchemas,
                    parallel_tool_calls: true,
                    world: worldModel.getConfig(),
                    maxCredits: BigInt(limits.maxCreditsPerTurn),
                    userData,
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
                            // Execute the tool call (sync or async) and record result
                            const { toolResult, functionCallEntry } = await this.processFunctionCall(
                                chatId,
                                bot.id,
                                ev as FunctionCallStreamEvent,
                            );

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
                            // Deduct cost for model usage
                            const cost = BigInt(ev.cost);
                            turnStats.creditsUsed += cost;
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
                    }
                    previousResponseId = ev.responseId;
                }

                // Prepare for next iteration
                inputs = nextInputs;
                tools = []; // only original tool calls
            }
        } finally {
            // Send cost incurred event 
            await BusService.get().getBus().publish({
                type: "billing:event",
                id: `${bot.id}-${chatId}-${turnStats.creditsUsed}`,
                accountId: creditAccountId,
                delta: (-turnStats.creditsUsed).toString(), // MAKE SURE THIS IS NEGATIVE
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
        return responseMessage;
    }

    // --------------------------------------------------------------------------
    // Helper: process a function_call event, handling sync vs async tools
    private async processFunctionCall(
        conversationId: string | undefined,
        callerBotId: string,
        ev: FunctionCallStreamEvent,
    ): Promise<{ toolResult: unknown; functionCallEntry: ToolFunctionCall }> {
        // Determine if this call is async (only routines can be async)
        const args = ev.arguments as Record<string, any>;
        const isAsync = args.isAsync === true;
        // Prepare the function call entry
        const fnEntryBase = {
            id: ev.callId,
            function: { name: ev.name, arguments: JSON.stringify(ev.arguments) },
        } as Omit<ToolFunctionCall, "result">;
        if (isAsync) {
            // TODO: emit async routine start event via bus
            // Acknowledge immediately
            const ack = { status: "started", callId: ev.callId };
            const entry: ToolFunctionCall = { ...fnEntryBase, result: { success: true, output: ack } };
            return { toolResult: ack, functionCallEntry: entry };
        } else {
            // Synchronous execution
            const res = await this.toolRunner.run(ev.name, ev.arguments, { conversationId: conversationId || "", callerBotId });
            let entry: ToolFunctionCall;
            let output: unknown = null;
            if (res.ok) {
                output = res.data.output;
                entry = { ...fnEntryBase, result: { success: true, output: res.data.output } };
            } else {
                entry = { ...fnEntryBase, result: { success: false, error: res.error } };
            }
            return { toolResult: output, functionCallEntry: entry };
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
    /**
     * @param reasoningEngine    Low-level engine for generating responses.
     * @param agentGraph         Strategy for picking responding bots (e.g. direct mentions, OpenAI Swarm).
     * @param bus                EventBus implementation (Redis/NATS etc.) so other services can listen for events
     * @param conversationStore  Efficient storage and retrieval of conversation state
     * @param messageStore       Efficient storage and retrieval of message state
     */
    constructor(
        private readonly reasoningEngine: ReasoningEngine,
        private readonly agentGraph: AgentGraph,
        private readonly bus: EventBus,
        private readonly conversationStore: ConversationStateStore,
        private readonly messageStore: MessageStore,
    ) { }

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
            logger.error("Message or conversation state not found in respond", { chatId });
            throw new Error("Message or conversation state not found");
        }

        // Load the triggering message
        const messageState = await this.messageStore.getMessage(messageId);
        if (!messageState) {
            logger.error("Message not found in cache in respond", { messageId });
            throw new Error(`Message ${messageId} not found in cache`);
        }

        // Determine credit account TODO this is NOT the user Id, but the ID of credit_account table associated with the user
        const creditAccountId = "";

        // Get available tools & world model from conversation state
        const availableTools = conversationState.availableTools;
        const worldModel = conversationState.worldModel;

        // Determine responders
        const { responders } = await this.agentGraph.selectResponders(conversationState, messageState);

        // Initialize results
        const results: MessageState[] = [];

        // Run reasoning loop for each responder in parallel
        await Promise.all(responders.map(async (responder) => {
            const response = await this.reasoningEngine.runLoop(
                { id: messageId },
                worldModel,
                availableTools,
                responder,
                creditAccountId,
                conversationState.config,
                userData,
                chatId,
                model,
            );
            results.push(response);
        }));

        // Store responses by adding each message to cache
        await Promise.all(results.map(m => this.messageStore.addMessage(chatId, m)));

        return results;
    }

    async handleInternalEvent(event: SwarmInternalEvent): Promise<void> {
        const { conversationId, type } = event;
        logger.info(`CompletionService handling internal event: ${type} for convo: ${conversationId}`);

        const conversationState = await this.conversationStore.get(conversationId);
        if (!conversationState) {
            logger.error(`Conversation state not found for ${conversationId} in handleInternalEvent`);
            throw new Error(`Conversation state not found for ${conversationId}`);
        }

        // TODO: Resolve how to get the actual SessionUser for Swarm-initiated events.
        // This placeholder is for type-checking only.
        const placeholderSessionUserSession: SessionUserSession = {
            __typename: "SessionUserSession",
            id: "placeholder_session_id",
            lastRefreshAt: new Date(),
        };
        const placeholderUserData: SessionUser = {
            __typename: "SessionUser",
            credits: "0",
            hasPremium: false,
            id: "system_user_id_for_swarm", // ID for the SessionUser
            languages: [DEFAULT_LANGUAGE],
            publicId: "system_user_public_id_for_swarm",
            session: placeholderSessionUserSession,
            // Optional fields can be omitted or set to undefined/null if appropriate
            handle: undefined,
            name: "System (Swarm)",
            profileImage: undefined,
            theme: undefined,
            updatedAt: new Date(),
            // Add any other non-optional fields from SessionUser with defaults
        };

        // TODO: Resolve how to get the correct creditAccountId for Swarm-initiated events.
        // This ID is crucial for billing and should map to the user/entity responsible for the swarm's costs.
        const creditAccountIdForRunLoop = "unknown_credit_account_for_swarm";

        const results: MessageState[] = [];

        switch (event.type) {
            case "swarm_started": {
                const swarmStartEvent = event as SwarmStartedEvent;
                const goal = swarmStartEvent.goal;
                const systemMessageText = `Swarm initiated. Goal: ${goal}.`;

                const startMessage = { text: systemMessageText };

                const msgCfg = MessageConfig.default();
                msgCfg.setRole("system");
                const systemMessageConfigExport = msgCfg.export();

                const syntheticMessageForSelection: MessageState = {
                    id: generatePK().toString(),
                    createdAt: new Date(),
                    config: systemMessageConfigExport, // Use exported config
                    text: systemMessageText,
                    language: DEFAULT_LANGUAGE,
                    parent: null,
                    user: { id: placeholderUserData.id },
                } as MessageState;

                const { responders } = await this.agentGraph.selectResponders(conversationState, syntheticMessageForSelection);

                if (!responders || responders.length === 0) {
                    logger.warn(`No responders found for swarm_started in convo ${conversationId}`);
                    return;
                }

                await Promise.all(responders.map(async (responder) => {
                    const response = await this.reasoningEngine.runLoop(
                        startMessage,
                        conversationState.worldModel,
                        conversationState.availableTools,
                        responder,
                        creditAccountIdForRunLoop, // Use placeholder credit account ID
                        conversationState.config,
                        placeholderUserData,
                        conversationId,
                    );
                    results.push(response);
                }));
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
                const { responders: msgResponders } = await this.agentGraph.selectResponders(conversationState, messageState);

                if (!msgResponders || msgResponders.length === 0) {
                    logger.warn(`No responders found for external_message_created in convo ${conversationId}`, { messageId });
                    return;
                }

                await Promise.all(msgResponders.map(async (responder) => {
                    const response = await this.reasoningEngine.runLoop(
                        { id: messageId },
                        conversationState.worldModel,
                        conversationState.availableTools,
                        responder,
                        creditAccountIdForRunLoop, // Use placeholder credit account ID
                        conversationState.config,
                        placeholderUserData,
                        conversationId,
                    );
                    results.push(response);
                }));
                break;
            }
            default: {
                logger.warn(`Unhandled SwarmInternalEvent type: ${event.type} in convo ${conversationId}`);
                return;
            }
        }

        if (results.length > 0) {
            await Promise.all(results.map(m => this.messageStore.addMessage(conversationId, m)));
            // TODO: Consider emitting events that these responses were created (e.g., via this.bus)
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
 *   2. **ACTIVE**        – processing events / generating sub‑goals
 *   3. **IDLE**          – nothing queued; awaiting new stimuli
 *   4. **PAUSED**        – paused by user / system
 *   5. **FAILED**        – unrecoverable error
 *   6. **TERMINATED**    – shut down; resources released
 *
 * Transitions:
 *   • UNINITIALIZED → ACTIVE  on start()
 *   • ACTIVE ↔ IDLE           as queue fills / drains
 *   • ACTIVE ↔ PAUSED         on pause/resume
 *   • * → FAILED              on fatal error
 *   • * → TERMINATED          on shutdown()
 *
 * Responsibilities:
 *   • Maintain shared conversation context (sub‑goals, outcomes)
 *   • Dispatch agents via ConversationService
 *   • Queue asynchronous tool events until relevant agent is ready
 *   • Publish pause / resume / shutdown signals
 *
 * Routine orchestration lives entirely in **RoutineStateMachine**.
 */
export class SwarmStateMachine {
    private disposed = false;

    /** Swarm lifecycle stages */
    static State = {
        UNINITIALIZED: "UNINITIALIZED",
        ACTIVE: "ACTIVE",
        IDLE: "IDLE",
        PAUSED: "PAUSED",
        FAILED: "FAILED",
        TERMINATED: "TERMINATED",
    } as const;

    // ── Constructor & private fields ───────────────────────────────────
    private state: State = SwarmStateMachine.State.UNINITIALIZED;
    private readonly eventQueue: SwarmInternalEvent[] = [];
    private worldModel: WorldModel | null = null;

    constructor(
        private readonly bus: EventBus,
        private readonly completion: CompletionService,
    ) {
        // SwarmStateMachine will now rely on explicit handleEvent calls
        // or internal event generation.
    }

    // ── World‑model helper ------------------------------------------------------
    /**
     * Generates the system prompt / world‑model seed inserted into the
     * conversation when the swarm starts.  Agents are briefed on:
     *   • Mission and high‑level goal
     *   • Shared‑chat etiquette & how to decide communication style
     *   • Available tools (routines, built‑ins) & when to delegate to them
     */
    private buildWorldModel(goal: string): WorldModel {
        const config = new WorldModel({ goal });
        config.updateConfig({
            systemMessage: `
    # 🐝 Swarm Configuration

    # 🔖 Shared-state Etiquette
    1. Maintain a lightweight **conversation-state JSON block** at the top of the thread.  
       – Send a message in the form \`{{state:update}} {...}\` whenever you mutate it.  
    2. Keep your own turns short; DO NOT dump large data blobs.  
       – Persist artifacts with \`resource_manage(op="add")\` and post a link/reference instead.
    
    # 🗂️ Organising Work
    • **Elect (and rotate) a facilitator** if coordination slows down.  
    • Break the mission into explicit *sub-goals* and track them in the state blob.  
    • If a sub-goal is complex or parallelisable, OFFLOAD IT:  
      → call **run_routine(op="start")** with \`mode="async"\`.
    
    # 🧰 Tooling Cheatsheet
    • \`define_tool\`   → ask the server for the full JSON schema of any tool.  
    • \`send_message\`  → chat or publish events between agents.  
    • \`resource_manage\` → CRUD for notes, prompts, routines, etc.  
    • **run_routine**  → your main lever for branching, retries, heavy lifting.
    
    Respond briefly, update the shared state often, and prefer routines over monolithic chat threads.
            `.trim(),
        });
        return config;
    }

    // ── Lifecycle control -------------------------------------------------------
    async start(convoId: string, goal: string): Promise<void> {
        if (this.state !== SwarmStateMachine.State.UNINITIALIZED) return;
        try {
            // Inject world‑model as a system message (stub – real impl via MessageStore)
            const systemPrompt = this.buildWorldModel(goal);
            logger.info("Starting swarm", { conversationId: convoId, goal, systemPrompt: systemPrompt.getConfig().systemMessage });
            // TODO: persist systemPrompt in conversation state/messageStore if it should be a visible message

            this.state = SwarmStateMachine.State.ACTIVE;
            // Kick off arbitrator / first agent via completion
            const startEvent: SwarmStartedEvent = { type: "swarm_started", conversationId: convoId, goal };
            await this.completion.handleInternalEvent(startEvent);
        } catch (err) {
            this.fail(err);
        }
    }

    async handleEvent(ev: SwarmInternalEvent): Promise<void> {
        if (this.state === SwarmStateMachine.State.TERMINATED) return;
        this.eventQueue.push(ev);
        if (this.state === SwarmStateMachine.State.IDLE && ev.conversationId) {
            this.drain(ev.conversationId).catch(err => logger.error("Error draining swarm queue from handleEvent", { error: err, event: ev }));
        }
    }

    private async drain(convoId: string) {
        if (this.state === SwarmStateMachine.State.PAUSED) return;
        if (this.eventQueue.length === 0) { // Check if queue is empty before setting to active
            this.state = SwarmStateMachine.State.IDLE;
            return;
        }
        this.state = SwarmStateMachine.State.ACTIVE;
        // Process only one event per drain call to prevent re-entrancy issues if handleInternalEvent is slow
        // or if it itself queues more events.
        // For a full drain loop, a while loop would be here, but that can lead to very long event loop blocking.
        // A more robust solution might involve setImmediate or a microtask queue for sequential processing without blocking.
        // For now, processing one and then relying on subsequent triggers to call drain again.
        if (this.eventQueue.length > 0) {
            const ev = this.eventQueue.shift()!;
            if (ev.conversationId !== convoId) {
                logger.warn("Swarm drain called with convoId mismatch", { expected: convoId, actual: ev.conversationId });
                // Re-queue or handle error? For now, re-queue and log.
                this.eventQueue.unshift(ev);
                this.state = SwarmStateMachine.State.IDLE; // Reset state
                return;
            }
            try {
                await this.completion.handleInternalEvent(ev);
            } catch (error) {
                logger.error("Error processing event in SwarmStateMachine drain", { error, event: ev });
                // Decide on error handling: fail the swarm, retry the event, or ignore?
                // For now, log and continue to set IDLE if queue empty.
            }
        }

        if (this.eventQueue.length === 0) {
            this.state = SwarmStateMachine.State.IDLE;
        } else {
            // If there are more events, schedule another drain to continue processing without blocking.
            // This avoids a deep recursion or a long-running while loop.
            // Note: This assumes `drain` can be called again safely.
            // Using setImmediate to yield to the event loop before continuing.
            setImmediate(() => this.drain(convoId).catch(err => logger.error("Error in scheduled subsequent drain", { error: err, conversationId: convoId })));
        }
    }

    async pause(): Promise<void> {
        if (this.state === SwarmStateMachine.State.ACTIVE || this.state === SwarmStateMachine.State.IDLE) {
            this.state = SwarmStateMachine.State.PAUSED;
        }
    }

    async resume(convoId: string): Promise<void> {
        if (this.state === SwarmStateMachine.State.PAUSED) {
            this.state = SwarmStateMachine.State.IDLE; // mark idle so drain() sets to ACTIVE if queue has items
            await this.drain(convoId);
        }
    }

    async shutdown(): Promise<void> {
        this.disposed = true; // Mark as disposed first
        this.eventQueue.length = 0;
        this.state = SwarmStateMachine.State.TERMINATED;
        logger.info("SwarmStateMachine shutdown complete.");
    }

    private fail(err: unknown) {
        logger.error("Swarm failure", { error: err });
        this.state = SwarmStateMachine.State.FAILED;
        // Optionally, perform cleanup or notify other services
    }
}


// Instantiate stores and services that are dependencies
const prismaChatStore = new PrismaChatStore();
// Wrap PrismaChatStore with CachedConversationStateStore
export const conversationStateStore = new CachedConversationStateStore(prismaChatStore);
export const messageStore = new RedisMessageStore(); // Export if used directly elsewhere

const contextBuilder = new RedisContextBuilder();
const toolRunner = new CompositeToolRunner();
const agentGraphInstance = new CompositeGraph();
const llmRouter = new FallbackRouter();
const bus = BusService.get().getBus();

// Instantiate core engines/services
const reasoningEngine = new ReasoningEngine(
    contextBuilder,
    llmRouter,
    toolRunner,
);

export const completionService = new CompletionService(
    reasoningEngine,
    agentGraphInstance,
    bus,
    conversationStateStore, // Use the wrapped CachedConversationStateStore instance
    messageStore,
);

// SwarmStateMachine class definition must come before this instantiation
export const swarmStateMachine = new SwarmStateMachine(
    bus,
    completionService,
);
