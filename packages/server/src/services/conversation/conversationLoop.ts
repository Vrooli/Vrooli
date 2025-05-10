import { LRUCache } from "@local/shared";
import { AgentGraph } from "./agentGraph.js";
import { ContextBuilder } from "./contextBuilder.js";
import { ConversationRepository } from "./conversationRepo.js";
import { EventBus } from "./eventBus.js";
import { LlmRouter, ToolRunner } from "./toolRunner.js";
import { BotParticipant, Conversation, ConversationEvent, LLM, Message, MessageCreatedEvent, ScheduledTickEvent, ToolResultEvent } from "./types.js";

/** 
 * Buffer of events grouped by turnId. This is used to batch events for a given turnId together. 
 * 
 * Any events which should be responded to immediately (e.g. tool results, so 
 * bots an use multiple tools in the same turn) are added to the existing turn buffer.
 * 
 * Any external events (e.g. user messages) are added to the buffer for the next turn, 
 * so that the conversation can move forward.
 */
type TurnQueue = Map<number, ConversationEvent[]>; // keyed by turnId

/** 
 * Hard stop for extreme fan-out.
 * Limits the number of events that can be processed in a single turn.
 * This is a safeguard to prevent infinite loops or excessive memory usage.
 */
const MAX_TURN_EVENTS = 100;

/**
 * Maximum number of conversations that we can hold in memory at once.
 * This is a safeguard to prevent excessive memory usage.
 */
const MAX_CONCURRENT_CONVERSATIONS = 1_000;

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
    /** per‑conversation buffer */
    private convoQueues = new LRUCache<string, TurnQueue>(MAX_CONCURRENT_CONVERSATIONS);
    /** conversations currently under drain */
    private processing = new Set<string>();

    /**
     * @param bus            EventBus implementation (Redis/NATS etc.) so other services can listen for events
     * @param repo           Conversation‑aware persistence adapter
     * @param contextBuilder Prepares the prompt slice + tool list for a bot turn. Typically passed into Responses API
     * @param toolRunner     Executes MCP tool calls (performs the actual work of a tool).
     * @param agentGraph     Strategy for picking responding bots (e.g. direct mentions, OpenAI Swarm).
     * @param llmRouter      Streams Responses‑API calls to chosen LLM provider (e.g. OpenAI, Anthropic).
     * @param maxToolRetries How many times a single LLM turn may recurse via
     *                       tool‑call → tool‑result before we abort
     */
    constructor(
        private readonly bus: EventBus,
        private readonly repo: ConversationRepository,
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
     * Push a fresh event into the per‑conversation queue, enforce MAX_TURN_EVENTS,
     * and trigger drain if nobody is currently processing that conversation.
     */
    private async enqueueEvent(evt: ConversationEvent) {
        if (this.disposed) return;

        const q = this.convoQueues.get(evt.conversationId) ?? new Map();
        this.convoQueues.set(evt.conversationId, q);

        const buf = q.get(evt.turnId) ?? [];
        if (buf.length >= MAX_TURN_EVENTS) {
            console.warn("Turn event cap reached", evt);
            return; // discard excess to prevent storms
        }
        buf.push(evt);
        q.set(evt.turnId, buf);

        // attempt to drain if not already processing
        if (!this.processing.has(evt.conversationId)) {
            this.drainTurns(evt.conversationId).catch(console.error);
        }
    }

    /**
     * Sequentially drains buffered turns for a single conversation.  Ensures
     * only one worker handles a convo at once (guarded by `processing` set).
     */
    private async drainTurns(conversationId: string) {
        this.processing.add(conversationId);
        try {
            const q = this.convoQueues.get(conversationId)!;
            while (!this.disposed && q && q.size) {
                const nextTurn = Math.min(...q.keys());
                const batch = q.get(nextTurn)!;
                q.delete(nextTurn);
                await this.processTurnBatch(batch);
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
        const conversation = await this.repo.getConversation(sample.conversationId);
        if (!conversation) {
            console.error("Conversation not found", sample.conversationId);
            return;
        }
        const responderSet = new Map<string, BotParticipant>();

        for (const evt of events) {
            switch (evt.type) {
                case "message.created":
                    await this.handleMessageForResponderCalc(evt as MessageCreatedEvent, conversation, responderSet);
                    break;
                case "scheduled.tick":
                    await this.handleTickForResponderCalc(evt as ScheduledTickEvent, conversation, responderSet);
                    break;
                case "tool.result":
                    // tool results usually only matter to the originating bot; handled inline later
                    await this.handleToolResult(evt as ToolResultEvent);
                    break;
                default:
                    console.warn("ConversationLoop: unknown event type", evt.type);
            }
        }

        if (responderSet.size === 0) return;
        await Promise.all(
            [...responderSet.values()].map((bot) =>
                this.runBotTurn(conversation, bot).catch((err) =>
                    console.error("Bot turn failed", bot.id, err)
                )
            )
        );
    }

    /**
     * Handle a MessageCreatedEvent to determine additional responders and add
     * them to `responderSet`.
     */
    private async handleMessageForResponderCalc(
        evt: MessageCreatedEvent,
        conversation: Conversation,
        responders: Map<string, BotParticipant>
    ) {
        const msg = await this.repo.getMessage(evt.messageId);

        // escape‑hatch early exit
        if (
            msg.role === "assistant" &&
            !msg.respondingBots?.length &&
            !msg.eventTopic
        ) {
            return;
        }

        (await this.agentGraph.selectResponders(conversation, msg)).forEach((p) =>
            responders.set(p.id, p)
        );
    }

    /**
     * Expand responders for a ScheduledTickEvent based on agentGraph rules.
     */
    private async handleTickForResponderCalc(
        evt: ScheduledTickEvent,
        conversation: Conversation,
        responders: Map<string, BotParticipant>
    ) {
        (await this.agentGraph.selectResponders(conversation, {
            type: "system",
            content: `⏰ tick:${evt.topic}`,
        } as Message)).forEach((p) => responders.set(p.id, p));
    }

    /**
    * Build context & tool schemas, then stream the Responses‑API loop until the
    * bot finishes (or tool‑retry limit reached).
    */
    private async runBotTurn(conversation: Conversation, bot: BotParticipant) {
        const { inputMessages, toolSchemas } =
            await this.contextBuilder.build(conversation, bot);
        await this.continueStream(conversation, bot.id, inputMessages, toolSchemas);
    }

    /**
     * Executes the LLM streaming loop with on‑the‑fly tool handling.
     * Re‑enters for chained tool calls up to `maxToolRetries` times.
     */
    private async continueStream(
        conversation: Conversation,
        botId: string,
        inputItems: any[],
        extraTools: any[] = []
    ): Promise<void> {
        let previousResponseId: string | undefined;
        let retriesLeft = this.maxToolRetries;

        while (true) {
            const stream = await this.llmRouter.stream({
                model: this.llmRouter.bestModelFor(botId),
                previous_response_id: previousResponseId,
                input: inputItems,
                tools: this.llmRouter.defaultTools().concat(extraTools),
                parallel_tool_calls: true,
            });

            const followUpInputs: any[] = [];
            for await (const ev of stream) {
                switch (ev.type) {
                    case "message":
                        await this.persistAssistantMessage(conversation.id, botId, ev);
                        break;
                    case "function_call":
                        if (ev.name === "handoff_to_bot") {
                            const { bot_id } = ev.arguments;
                            await this.repo.updateConversationMeta(conversation.id, {
                                activeBotId: bot_id === "arbitrator" ? undefined : bot_id
                            });
                            // No follow-up input → break; the next user message will trigger new turn
                            continue;
                        }
                        followUpInputs.push(
                            await this.processFunctionCall(conversation, botId, ev)
                        );
                        break;
                    default:
                        break; // ignore
                }
                previousResponseId = ev.response_id;
            }

            if (followUpInputs.length === 0) break;
            if (--retriesLeft < 0) {
                console.error("Too many tool retries, aborting", botId);
                break;
            }
            inputItems = followUpInputs;
        }
    }

    /**
     * Dispatch one MCP tool call and return a `function_call_output` object the
     * model can ingest on the next step.
     */
    private async processFunctionCall(
        conversation: Conversation,
        botId: string,
        ev: LLM.StreamEvent.FunctionCall
    ) {
        try {
            const output = await this.toolRunner.run(ev.name, ev.arguments, {
                conversationId: conversation.id,
                callerBotId: botId,
            });
            return { type: "function_call_output", call_id: ev.call_id, output };
        } catch (err) {
            console.error("Tool execution failed", ev.name, err);
            return {
                type: "function_call_output",
                call_id: ev.call_id,
                output: { ok: false, error: { code: "TOOL_ERROR", message: String(err) } },
            };
        }
    }

    /**
     * Persist an assistant reply then publish a `message.created` event with the
     * **same turnId** so it does NOT trigger a new turn.
     */
    private async persistAssistantMessage(
        conversationId: string,
        botId: string,
        ev: StreamEvent.Message
    ) {
        const msg: Message = {
            role: "assistant",
            content: ev.content,
            botId,
            createdAt: new Date(),
            turnId: ev.turnId,
        } as Message; // extend with turnId inside repo if desired

        const saved = await this.repo.saveMessage(conversationId, msg);
        await this.bus.publish({
            type: "message.created",
            conversationId,
            messageId: saved.id!,
            turnId: ev.turnId, // keep same turn
        } as ConversationEvent);
    }
}
