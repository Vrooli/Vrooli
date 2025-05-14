/* eslint-disable func-style */
import { ChatConfig, LRUCache, MessageConfigObject, type ChatConfigObject } from "@local/shared";
import { SocketService } from "../../sockets/io.js";
import { BusWorker, EventBus } from "../bus.js";
import { AgentGraph, CompositeGraph } from "./agentGraph.js";
import { ContextBuilder, RedisContextBuilder } from "./contextBuilder.js";
import { ChatPersistence, PrismaChatPersistence } from "./persistence.js";
import { FunctionCallOutput, OutputGenerator } from "./responseUtils.js";
import { FallbackRouter, FunctionCallStreamEvent, LlmRouter } from "./router.js";
import { ToolRunner } from "./toolRunner.js";
import { BotParticipant, ConversationEvent, ConversationState, MessageCreatedEvent, MessageState, ScheduledTickEvent, ToolResultEvent, TurnStats } from "./types.js";

/**
 * Maximum number of conversations that we can hold in memory at once.
 * This is a safeguard to prevent excessive memory usage.
 */
const MAX_CONCURRENT_CONVERSATIONS = 1_000;
/** Maximum number of messages to keep in memory */
const MAX_RECENT_MESSAGES = 5_000;

function incrementToolStats(
    turn: TurnStats,
    convoCfg: ChatConfigObject,
    amount = 1,
) {
    turn.toolCalls += amount;      // whole-turn cap
    turn.botToolCalls += amount;      // per-bot cap
    convoCfg.stats.totalToolCalls += amount; // conversation-wide
}

function incrementCreditStats(
    turn: TurnStats,
    convoCfg: ChatConfigObject,
    amount: bigint,
) {
    turn.creditsUsed += amount;
    convoCfg.stats.totalCredits = (
        BigInt(convoCfg.stats.totalCredits) + amount
    ).toString();
}

// TODO make sure that when messages/chats/particpants are added/updated/deleted (ModelLogic trigger), this class handles it correctly (including adding to context collector cache)
// TODO I don't think reduceUserCredits is being called in the right places (or anywhere). Need to carefully validate that the code reduces spent credits.
// TODO Tthe failure cases could be handled better. E.g. emitting event to show retry button
/* 
* ------------------------------------------------------------------
* A coordinator that advances a chat or swarm conversation forward.
* 
* Progresses the conversation state forward using turns, where each turn 
* processes all events in the buffer for a given turnId. 
* Each event triggers 0 or more bots to respond in the next turn.
* 
* NOTE 1: This is only designed to move the conversation forward by 1 turn at a time. 
* Subsequent turns can only be triggered by external events, such as user input,
* webhook events, scheduled events, or routines called by bots in the conversation. 
* If you want the bots to converse with each other endlessly (until limits are reached of course) 
* without stopping, a separate class that calls `ConversationLoop` is required.
* 
* NOTE 2: You should only have one instance of this class per process
* ------------------------------------------------------------------ 
*/
export class ConversationLoop {
    private disposed = false;
    /** per‑conversation turn buffer */
    private convoStates = new LRUCache<string, ConversationState>(MAX_CONCURRENT_CONVERSATIONS);
    /** conversations currently under drain */
    private processing = new Set<string>();
    /** Recent messages */
    private messageData = new LRUCache<string, MessageState>(MAX_RECENT_MESSAGES);

    /**
     * @param bus            EventBus implementation (Redis/NATS etc.) so other services can listen for events
     * @param repo           Conversation‑aware persistence adapter
     * @param contextBuilder Prepares the prompt slice + tool list for a bot turn. Typically passed into Responses API
     * @param toolRunner     Executes MCP tool calls (performs the actual work of a tool).
     * @param agentGraph     Strategy for picking responding bots (e.g. direct mentions, OpenAI Swarm).
     * @param llmRouter      Streams Responses‑API calls to chosen LLM provider (e.g. OpenAI, Anthropic).
     */
    constructor(
        private readonly bus: EventBus,
        private readonly store: ChatPersistence,
        private readonly contextBuilder: ContextBuilder,
        private readonly toolRunner: ToolRunner,
        private readonly agentGraph: AgentGraph,
        private readonly llmRouter: LlmRouter,
    ) {
        this.bus.subscribe((evt) => this.enqueueEvent(evt).catch(console.error));
    }

    /**
     * Clean shutdown: mark disposed, close bus connection.
     */
    async dispose(): Promise<void> {
        this.disposed = true;
        await this.bus.close();
    }

    /**
     * Gets conversation-specific turn queue and config, or creates them if they don't exist.
     * 
     * @param conversationId - The ID of the conversation to get the state for.
     * @param invalidate - If true, the cached state will be invalidated and the conversation will be reloaded from the database.
     */
    private async getConversationState(conversationId: string, invalidate = false): Promise<ConversationState | null> {
        // Check for cached state
        const existingState = this.convoStates.get(conversationId);
        if (existingState && !invalidate) return existingState;
        // Check for persisted state
        const persistedState = await this.store.getConversation(conversationId);
        if (persistedState) {
            this.convoStates.set(conversationId, persistedState);
            return persistedState;
        }
        return null;
    }

    /**
     * Updates the conversation config in memory.
     */
    private async updateConversationState(conversationId: string, updatedState: Partial<ConversationState>) {
        const existingState = await this.getConversationState(conversationId);
        const result = {
            config: ChatConfig.default().export(),
            participants: [],
            ...this.store.initializeTurnState(),
            ...existingState,
            ...updatedState,
            id: conversationId,
        } satisfies ConversationState;
        // Update in memory
        this.convoStates.set(conversationId, result);
        // Persist to database (debounced)
        this.store.saveState(conversationId, result.config);
    }

    /**
     * Removes a conversation from memory only (stills persists to db)
     */
    private removeConversation(conversationId: string) {
        this.convoStates.delete(conversationId);
    }

    /**
     * Gets a message from the message cache.
     */
    private async getMessage(messageId: string): Promise<MessageState | null> {
        const cached = this.messageData.get(messageId);
        if (cached) return cached;
        const persisted = await this.store.getMessage(messageId);
        if (persisted) this.messageData.set(messageId, persisted);
        return persisted ?? null;
    }

    /**
     * Removes a message from the message cache.
     */
    private removeMessage(messageId: string) {
        this.messageData.delete(messageId);
    }

    /**
     * Push a fresh event into the per‑conversation queue, enforce MAX_TURN_EVENTS,
     * and trigger drain if nobody is currently processing that conversation.
     */
    private async enqueueEvent(event: ConversationEvent) {
        if (this.disposed) return;

        const state = await this.getConversationState(event.conversationId);
        if (!state) return;

        const chatConfigInstance = new ChatConfig({ config: state.config });
        const effectiveLimits = chatConfigInstance.getEffectiveLimits();

        const buf = state.queue.get(event.turnId) ?? [];
        if (buf.length >= effectiveLimits.maxEventsPerTurn) {
            console.warn("Turn event cap reached based on config", event);
            return; // discard excess to prevent storms
        }
        buf.push(event);
        state.queue.set(event.turnId, buf);

        // attempt to drain if not already processing
        if (!this.processing.has(event.conversationId)) {
            this.drainTurns(event.conversationId).catch(console.error);
        }
    }

    /**
     * Sequentially drains buffered turns for a single conversation.  Ensures
     * only one worker handles a convo at once (guarded by `processing` set).
     */
    private async drainTurns(conversationId: string) {
        this.processing.add(conversationId);
        try {
            const state = await this.getConversationState(conversationId);
            if (!state) return;

            while (!this.disposed && state.queue && state.queue.size) {
                const nextTurn = Math.min(...state.queue.keys());
                const batch = state.queue.get(nextTurn);
                if (batch) {
                    state.queue.delete(nextTurn);
                    await this.processTurnBatch(batch);
                }
            }
        } finally {
            this.processing.delete(conversationId);
        }
    }

    /**
     * Consumes all events in a turn, aggregates the union of responders, and
     * fires off their bot turns in parallel.
     */
    private async processTurnBatch(events: ConversationEvent[]) {
        const sample = events[0];
        const state = await this.getConversationState(sample.conversationId, true); // Invalidate cache to reload from DB and reset turn stats
        if (!state) {
            console.error("Conversation not found", sample.conversationId);
            return;
        }

        const responderSet = new Map<string, BotParticipant>();

        for (const event of events) {
            switch (event.type) {
                case "message.created":
                    await this.handleMessageForResponderCalc(event as MessageCreatedEvent, state, responderSet);
                    break;
                case "scheduled.tick":
                    await this.handleTickForResponderCalc(event as ScheduledTickEvent, state, responderSet);
                    break;
                case "tool.result":
                    // tool results usually only matter to the originating bot; handled inline later
                    await this.handleToolResult(event as ToolResultEvent);
                    break;
                default:
                    console.warn("ConversationLoop: unknown event type", event.type);
            }
        }

        if (responderSet.size === 0) return;
        await Promise.all(
            [...responderSet.values()].map((bot) =>
                this.runBotTurn(
                    state,
                    bot,
                ).catch((err) => console.error("Bot turn failed", bot.id, err)),
            ),
        );
    }

    /**
     * Handle a MessageCreatedEvent to determine additional responders and add
     * them to `responderSet`.
     */
    private async handleMessageForResponderCalc(
        event: MessageCreatedEvent,
        conversation: ConversationState,
        responders: Map<string, BotParticipant>,
    ) {
        const message = await this.getMessage(event.messageId);

        // escape‑hatch early exit
        if (
            !message ||
            message.config.role === "assistant" &&
            !message.config.respondingBots?.length &&
            !message.config.eventTopic
        ) {
            return;
        }

        (await this.agentGraph.selectResponders(conversation, message)).forEach((p) =>
            responders.set(p.id, p),
        );
    }

    /**
     * Expand responders for a ScheduledTickEvent based on agentGraph rules.
     */
    private async handleTickForResponderCalc(
        event: ScheduledTickEvent,
        conversation: ConversationState,
        responders: Map<string, BotParticipant>,
    ) {
        (await this.agentGraph.selectResponders(conversation, {
            type: "system",
            content: `⏰ tick:${event.topic}`,
        })).forEach((p) => responders.set(p.id, p));
    }

    /**
    * Build context & tool schemas, then stream the Responses‑API loop until the
    * bot finishes (or tool‑retry limit reached).
    */
    private async runBotTurn(
        conversation: ConversationState,
        bot: BotParticipant,
    ) {
        const state = await this.getConversationState(conversation.id);
        if (!state) return;

        state.turnStats.botToolCalls = 0;

        /* Tell clients the bot has started typing */
        SocketService.get().emitSocketEvent("typing", conversation.id, {
            starting: [bot.id],
        });

        try {
            const { inputMessages, toolSchemas } = await this.contextBuilder.build(conversation, bot);

            /* Stream the reply (continues to emit "responseStream" chunks) */
            await this.continueStream(
                conversation,
                bot.id,
                inputMessages,
                toolSchemas,
            );

        } finally {
            /* Regardless of success/failure, stop the indicator */
            SocketService.get().emitSocketEvent("typing", conversation.id, {
                stopping: [bot.id],
            });
        }
    }

    /**
     * Executes the LLM streaming loop with on‑the‑fly tool handling.
     * Re‑enters for chained tool calls up to `maxToolCallsPerBotTurn` times.
     */
    private async continueStream(
        conversation: ConversationState,
        botId: string,
        inputItems: any[],
        extraTools: any[] = [],
    ): Promise<void> {
        let previousResponseId: string | undefined;
        let draftMessage = "";
        let finalTurnCost = BigInt(0);

        // Compute effective conversation limits and turn stats
        const state = await this.getConversationState(conversation.id);
        if (!state) return;
        const { config, turnStats } = state;
        const limits = (new ChatConfig({ config })).getEffectiveLimits();

        const turnToolLimitReached = () => turnStats.toolCalls >= limits.maxToolCallsPerTurn;
        const botToolLimitReached = () => turnStats.botToolCalls >= limits.maxToolCallsPerBotTurn;
        const turnCreditLimitReached = () => turnStats.creditsUsed >= BigInt(limits.maxCreditsPerTurn);
        const convoCreditLimitReached = () => BigInt(config.stats.totalCredits) >= BigInt(limits.maxCredits);
        const convoToolLimitReached = () => config.stats.totalToolCalls >= limits.maxToolCalls;

        const canContinue = () => {
            if (!inputItems?.length) return false;
            if (turnToolLimitReached()) return false;
            if (botToolLimitReached()) return false;
            if (turnCreditLimitReached()) return false;
            if (convoCreditLimitReached()) return false;
            if (convoToolLimitReached()) return false;
            return true;
        };

        while (canContinue()) {
            const stream = this.llmRouter.stream({
                model: this.llmRouter.bestModelFor(botId),
                previous_response_id: previousResponseId,
                input: inputItems,
                tools: this.llmRouter.defaultTools().concat(extraTools),
                parallel_tool_calls: true,
            });

            const nextInputs: any[] = [];

            try {
                for await (const event of stream) {
                    switch (event.type) {
                        /* ─ messages (streaming) ─────────────────── */
                        case "message": {
                            draftMessage = event.final ? event.content : draftMessage + event.content;
                            SocketService.get().emitSocketEvent("responseStream", conversation.id, {
                                botId,
                                turnId: conversation.turnCounter,
                                chunk: event.content,
                                done: event.final ?? false,
                            });
                            break;
                        }

                        /* ─ tool calls ───────────────────────────── */
                        case "function_call": {
                            if (botToolLimitReached()) {
                                nextInputs.push(
                                    OutputGenerator.functionCallOutputError(
                                        event.callId,
                                        "BOT_LIMIT_EXCEEDED",
                                        "Bot tool-call limit reached",
                                    ),
                                );
                                break;
                            }
                            const { toolResult } = await this.processFunctionCall(
                                conversation,
                                botId,
                                event,
                                config,
                                turnStats,
                            );
                            nextInputs.push(toolResult);
                            break;
                        }

                        /* ─ provider reasoning, we ignore for now TODO ── */
                        case "reasoning":
                            break;

                        /* ─ done – we finally know the cost ─────── */
                        case "done":
                            finalTurnCost = BigInt(event.cost);
                            incrementCreditStats(turnStats, config, finalTurnCost);
                            break;
                    }
                    previousResponseId = event.responseId;
                }

            } catch (error) {
                console.error("Error processing stream", error);
            }

            /* persist the assembled assistant reply (full text) */
            if (draftMessage) {
                await this.persistAssistantMessage(
                    conversation.id,
                    botId,
                    draftMessage.trim(),
                    conversation.turnCounter,
                    previousResponseId,
                );
                draftMessage = ""; // reset for chained tool calls
            }
            inputItems = nextInputs;
        }

        // Update conversation state
        config.stats.lastTurnEndedAt = Date.now();
        this.updateConversationState(conversation.id, { config, turnStats });
    }

    /**
     * Dispatch one MCP tool call and return a `function_call_output` object the
     * model can ingest on the next step.
     */
    private async processFunctionCall(
        conversation: ConversationState,
        botId: string,
        event: FunctionCallStreamEvent,
        config: ChatConfigObject,
        turnStats: TurnStats,
    ): Promise<FunctionCallOutput> {
        const limits = new ChatConfig({ config }).getEffectiveLimits();

        if (config.stats.totalToolCalls >= limits.maxToolCalls) {
            return OutputGenerator.functionCallOutputError(event.callId, "CONVERSATION_TOOL_LIMIT_EXCEEDED", "Conversation tool‑call cap hit");
        }
        if (turnStats.toolCalls >= limits.maxToolCallsPerTurn) {
            return OutputGenerator.functionCallOutputError(event.callId, "TURN_TOOL_LIMIT_EXCEEDED", "Turn tool‑call cap hit");
        }

        // Assume toolRunner.run returns { output: any, creditsUsed: bigint }
        // We'll get the cost after running the tool.
        let toolCost = BigInt(0);
        let toolOutputResult: any;

        try {
            const { output, creditsUsed } = await this.toolRunner.run(event.name, event.arguments, {
                conversationId: conversation.id,
                callerBotId: botId,
            });
            toolOutputResult = output;
            toolCost = creditsUsed ?? BigInt(0);

            incrementToolStats(turnStats, config, 1);
            incrementCreditStats(turnStats, config, toolCost);

            // Check credit limits
            if (turnStats.creditsUsed > BigInt(limits.maxCreditsPerTurn)) {
                return OutputGenerator.functionCallOutputError(event.callId, "TURN_CREDIT_LIMIT_EXCEEDED", `Turn credit limit exceeded by tool ${event.name}. Cost: ${toolCost}`);
            }
            if (BigInt(config.stats.totalCredits) > BigInt(limits.maxCredits)) {
                return OutputGenerator.functionCallOutputError(event.callId, "CONVERSATION_CREDIT_LIMIT_EXCEEDED", `Conversation credit limit exceeded by tool ${event.name}. Cost: ${toolCost}`);
            }

            return OutputGenerator.functionCallOutputSuccess(event.callId, toolOutputResult);

        } catch (err) {
            console.error("Tool execution failed", event.name, err);
            // If tool itself failed, we might still have incurred cost if toolRunner reported it before throwing
            // For now, assuming no cost if toolRunner.run throws before returning creditsUsed.
            return OutputGenerator.functionCallOutputError(event.callId, "TOOL_ERROR", String(err));
        }
    }

    /**
     * Persist an assistant reply then publish a `message.created` event with the
     * **same turnId** so it does NOT trigger a new turn.
     */
    private async persistAssistantMessage(
        conversationId: string,
        botId: string,
        draft: string,
        turnId: number,
        parentId: string | null,
    ) {
        const config: MessageConfigObject = {
            __version: "1.0.0",
            role: "assistant",
            turnId,
        };
        const saved = await this.store.saveMessage(conversationId, {
            config,
            content: draft.trim(),
            botId,
            parentId,
        });
        await this.bus.publish({
            type: "message.created",
            conversationId,
            payload: { messageId: saved.id },
            turnId, // same turn, so no new turn queued
        });
    }
}

export const chatStore = new PrismaChatPersistence();
const contextBuilder = new RedisContextBuilder();
const toolRunner = new McpToolRunner();
const agentGraph = new CompositeGraph();
const llmRouter = new FallbackRouter();

export class ConversationWorker extends BusWorker {
    protected static async init(bus: EventBus) {
        return new ConversationLoop(
            bus,
            chatStore,
            contextBuilder,
            toolRunner,
            agentGraph,
            llmRouter,
        );
    }

    static get(): ConversationLoop {
        return super.get() as ConversationLoop;
    }

    protected static async shutdown() {
        const loop = ConversationWorker.get();
        await loop.dispose();
    }
}

