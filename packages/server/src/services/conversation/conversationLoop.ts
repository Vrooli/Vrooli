/* eslint-disable func-style */
import { ChatConfig, MessageConfigObject, type ChatConfigObject } from "@local/shared";
import { SocketService } from "../../sockets/io.js";
import { BusWorker, ConversationBaseEvent, ConversationEvent, EventBus, MessageCreatedEvent, ScheduledTickEvent } from "../bus.js";
import { AgentGraph, CompositeGraph } from "./agentGraph.js";
import { ConversationStateStore, PrismaChatStore } from "./chatStore.js";
import { ContextBuilder, RedisContextBuilder } from "./contextBuilder.js";
import { MessageStore, PrismaRedisMessageStore } from "./messageStore.js";
import { FunctionCallOutput, OutputGenerator } from "./responseUtils.js";
import { FallbackRouter, FunctionCallStreamEvent, LlmRouter } from "./router.js";
import { CompositeToolRunner, ToolRunner } from "./toolRunner.js";
import { BotParticipant, ConversationState, TurnStats } from "./types.js";

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
// TODO make sure that chat updates update or invalidate the conversation state cache
// TODO I don't think reduceUserCredits is being called in the right places (or anywhere). Need to carefully validate that the code reduces spent credits.
// TODO Tthe failure cases could be handled better. E.g. emitting event to show retry button
//TODO handle message updates (branching conversation)
// TODO make sure preferred model is stored in the conversation state
//TODO canceling responses
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
    /** conversations currently under drain */
    private processing = new Set<string>();

    /**
     * @param agentGraph         Strategy for picking responding bots (e.g. direct mentions, OpenAI Swarm).
     * @param bus                EventBus implementation (Redis/NATS etc.) so other services can listen for events
     * @param conversationStore  Efficient storage and retrieval of conversation state
     * @param contextBuilder     Collects all chat history that can fit into the context window of a bot turn.
     * @param llmRouter          Streams Responses‑API calls to chosen LLM provider (e.g. OpenAI, Anthropic).
     * @param messageStore       Efficient storage and retrieval of message state
     * @param toolRunner         Executes MCP tool calls (performs the actual work of a tool).
     */
    constructor(
        private readonly agentGraph: AgentGraph,
        private readonly bus: EventBus,
        private readonly conversationStore: ConversationStateStore,
        private readonly contextBuilder: ContextBuilder,
        private readonly llmRouter: LlmRouter,
        private readonly messageStore: MessageStore,
        private readonly toolRunner: ToolRunner,
    ) {
        this.bus.subscribe((event) => {
            // Ensure that we only process events that are relevant to the conversation loop
            if ("conversationId" in event && "turnId" in event) {
                this.enqueueEvent(event).catch(console.error);
            }
        });
    }

    /**
     * Clean shutdown: mark disposed, close bus connection.
     */
    async dispose(): Promise<void> {
        this.disposed = true;
        await this.bus.close();
    }

    /**
     * Push a fresh event into the per‑conversation queue, enforce MAX_TURN_EVENTS,
     * and trigger drain if nobody is currently processing that conversation.
     */
    private async enqueueEvent(event: ConversationBaseEvent) {
        if (this.disposed) return;

        const state = await this.conversationStore.get(event.conversationId);
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
            const state = await this.conversationStore.get(conversationId);
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
        const state = await this.conversationStore.get(sample.conversationId, true); // Invalidate cache to reload from DB and reset turn stats
        if (!state) {
            console.error("Conversation not found", sample.conversationId);
            return;
        }

        const responderSet = new Map<string, BotParticipant>();

        for (const event of events) {
            switch (event.type) {
                case "message.created":
                    await this.handleMessageCreatedEvent(event, state, responderSet);
                    break;
                case "scheduled.tick":
                    await this.handleTickEvent(event, state, responderSet);
                    break;
                case "tool.result":
                    await this.handleToolResultEvent(event);
                    break;
                default:
                    // @ts-expect-error Property 'type' is expected to cause an error here due to 'never' type
                    console.warn("ConversationLoop: unknown event type", event.type);
                    break;
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
    private async handleMessageCreatedEvent(
        event: MessageCreatedEvent,
        conversation: ConversationState,
        responders: Map<string, BotParticipant>,
    ) {
        const message = await this.messageStore.getMessage(event.messageId);

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
    private async handleTickEvent(
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
        const state = await this.conversationStore.get(conversation.id);
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
        const state = await this.conversationStore.get(conversation.id);
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
        this.conversationStore.update(conversation.id, { config, turnStats });
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

        let toolCost = BigInt(0);
        let toolOutputResult: any;

        try {
            const result = await this.toolRunner.run(event.name, event.arguments, {
                conversationId: conversation.id,
                callerBotId: botId,
            });
            if (!result.ok) {
                return OutputGenerator.functionCallOutputError(event.callId, "TOOL_ERROR", result.error.message);
            }
            toolOutputResult = result.data.output;
            toolCost = result.data.creditsUsed;

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
        const saved = await this.messageStore.addMessage(conversationId, {
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

    /**
     * Handles a ToolResultEvent by updating message state and notifying clients.
     * This ensures that the result of a tool call is properly processed and surfaced to the UI.
     *
     * @param event - The ToolResultEvent containing the tool call result and metadata.
     */
    private async handleToolResultEvent(event: ToolResultEvent): Promise<void> {
        // Extract relevant fields from the event
        const { conversationId, callId, output, callerBotId, payload } = event;

        // If the event is associated with a message, update the message cache/state
        // Convention: payload may contain messageId
        let messageId: string | undefined = undefined;
        if (typeof payload === "object" && payload !== null && "messageId" in payload) {
            messageId = (payload as { messageId: string }).messageId;
        }

        if (messageId) {
            const message = await this.messageStore.getMessage(messageId);
            if (message && message.config.toolCalls) {
                // Find the tool call by id and update its result
                const updatedToolCalls = message.config.toolCalls.map(tc => {
                    if (tc.id === callId) {
                        return {
                            ...tc,
                            result: { success: true, output },
                        };
                    }
                    return tc;
                });
                message.config.toolCalls = updatedToolCalls;
                this.messageStore.updateMessage(messageId, message);
            }
        }

        // Emit a socket event to notify the client/UI of the tool result
        SocketService.get().emitSocketEvent("toolResult", conversationId, {
            callId,
            output,
            callerBotId,
            messageId,
        });

        // If the result indicates an error, log it and optionally emit a retry UI event
        if (error) {
            console.error(`Tool call failed (toolCallId: ${toolCallId}, botId: ${botId}):`, error);
            // Optionally, emit a retry UI event or update conversation state for retry logic
            // SocketService.get().emitSocketEvent("toolResultRetry", conversationId, { toolCallId, error, messageId, botId });
            // TODO: Integrate retry logic if required by the UI/UX
        }
    }
}

export const chatStore = new PrismaChatStore();
export const messageStore = new PrismaRedisMessageStore();
const contextBuilder = new RedisContextBuilder();
const toolRunner = new CompositeToolRunner();
const agentGraph = new CompositeGraph();
const llmRouter = new FallbackRouter();

export class ConversationWorker extends BusWorker {
    protected static async init(bus: EventBus) {
        return new ConversationLoop(
            agentGraph,
            bus,
            chatStore,
            contextBuilder,
            llmRouter,
            messageStore,
            toolRunner,
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

