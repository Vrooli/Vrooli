/* eslint-disable func-style */
import { ChatConfig, type ChatConfigObject } from "@local/shared";
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { SocketService } from "../../sockets/io.js";
import { type LLMCompletionTask } from "../../tasks/taskTypes.js";
import { BusService, BusWorker, type EventBus } from "../bus.js";
import { type ToolInputSchema } from "../mcp/types.js";
import { type AgentGraph, CompositeGraph } from "./agentGraph.js";
import { type ConversationStateStore, PrismaChatStore } from "./chatStore.js";
import { type ContextBuilder, RedisContextBuilder } from "./contextBuilder.js";
import { type MessageStore } from "./messageStore.js";
import { FallbackRouter, type LlmRouter } from "./router.js";
import { CompositeToolRunner, type ToolRunner } from "./toolRunner.js";
import { type BotParticipant, type MessageState, type TurnStats } from "./types.js";
import { WorldModel } from "./worldModel.js";

/** Maximum iterations for the drain loop, if something goes wrong with normal limit logic */
const MAX_DRAIN_ITERATIONS = 1_000;

/**
 * Increment the tool call counter for the given turn and conversation config.
 */
function bumpToolStats(
    turn: TurnStats,
    convoCfg: ChatConfigObject,
    amount = 1,
) {
    turn.toolCalls += amount;      // whole-turn cap
    turn.botToolCalls += amount;      // per-bot cap
    convoCfg.stats.totalToolCalls += amount; // conversation-wide
}

/**
 * Increment the total credits used for the given turn and conversation config.
 */
function bumpCreditStats(
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
// TODO need turn timeouts
//TODO


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
   * Executes a single reasoning loop for a bot:
   *  1. Constructs context window using the provided messages, world model, and tool schemas
   *  2. Emits events for clients to update the UI (e.g. typing, thinking)
   *  3. Streams LLM responses (chunks, function_call, done)
   *  4. Handles tool calls inline and feeds results back
   *  5. Deducts credits & tool calls via CacheService
   * 
   * @param messageHistory - The messages to include in the context window
   * @param worldModel - The world model to use for the context window
   * @param availableTools - The tools available to the bot
   * @param bot - Information about the bot that's running the loop
   * @param creditAccountId - The credit account to charge for the response (tied to a user or team)
   * @param chatId - The chat this loop is for, if applicable
   * 
   * @returns - A MessageState object representing the final message from the bot, 
   * including any tool calls and their results, and any bots directly mentioned in the message. 
   * It is the responsibility of the caller to decide what to do with the message, 
   * including whether to persist it, whether to trigger responses for the bots mentioned, etc.
   */
    async runLoop(
        messageHistory: MessageState[],
        worldModel: WorldModel,
        availableTools: ToolInputSchema[],
        bot: BotParticipant,
        creditAccountId: string,
        chatId: string | undefined,
    ): Promise<void> {
        // Signal typing start
        if (chatId) {
            SocketService.get().emitSocketEvent("typing", chatId, { starting: [bot.id] });
        }

        // Build context and available tools
        const { contextMessages } = await this.contextBuilder.build({
            availableTools,
            bot,
            messageHistory,
            worldModel,
        });

        let inputs = messageHistory;
        let tools: ToolInputSchema[] = availableTools;
        let draftMessage = "";
        let previousResponseId: string | undefined;
        const turnStats: TurnStats = { toolCalls: 0, botToolCalls: 0, creditsUsed: 0n };

        // Compute limits
        const limits = new ChatConfig({ config: convo.config }).getEffectiveLimits();
        const exceeded = () =>
            turnStats.toolCalls >= limits.maxToolCallsPerTurn ||
            turnStats.botToolCalls >= limits.maxToolCallsPerBotTurn ||
            turnStats.creditsUsed >= BigInt(limits.maxCreditsPerTurn);

        try {
            while (inputs.length && !exceeded()) {
                const stream = this.llmRouter.stream({
                    model: this.llmRouter.bestModelFor(bot.id),
                    previous_response_id: previousResponseId,
                    input: inputs,
                    tools,
                    parallel_tool_calls: true,
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
                            // Process tool call and get structured output
                            const out = await this.processFunctionCall(convo, bot.id, ev, turnStats);
                            nextInputs.push(out.toolResult);
                            break;
                        }

                        case "done":
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

        // TODO: Build and return a MessageState object for the final bot response.
        // Implementation outline:
        //   - Use `draftMessage` as the `text` field.
        //   - Generate or acquire a unique `id` for the message.
        //   - Set `createdAt` to the current timestamp.
        //   - Include the conversation `config` (ChatConfigObject) and response `language`.
        //   - Attach any tool call outputs collected during this loop.
        //   - Reference the previous response via `parent: { id: previousResponseId }`, if applicable.
        //   - Return an object matching the MessageState type, e.g.:
        //       return {
        //         id: "<generated-id>",
        //         createdAt: new Date(),
        //         config: <ChatConfigObject>,
        //         language: "<language-code>",
        //         text: draftMessage,
        //         parent: previousResponseId ? { id: previousResponseId } : null,
        //         user: { id: bot.id },
        //       };
        // Replace this placeholder with actual implementation when ready.
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
        taskContexts,
        userData,
    }: LLMCompletionTask) {
        // Get message and conversation state
        const messageState = await this.messageStore.getMessage(messageId);
        const conversationState = await this.conversationStore.get(chatId);
        if (!messageState || !conversationState) {
            throw new Error("Message or conversation state not found");
        }

        // Get available tools
        const availableTools = conversationState.availableTools;

        // Get world model
        const worldModel = conversationState.worldModel;

        // Determine responders
        const responders = this.agentGraph.selectResponders(conversationState, messageState);

        // Initialize results
        const results: MessageState[] = [];

        // Run reasoning loop for each responder in parallel
        await Promise.all(responders.map(async (responder) => {
            const response = await this.reasoningEngine.runLoop(messageState, worldModel, availableTools, responder, chatId);
            results.push(response);
        }));

        // Store responses
        await this.messageStore.storeMessages(results);

        return results;
    }
}

/*──────────────────────── factory wiring ───────────────────────────────────*/
export const chatStore = new PrismaChatStore();
export const messageStore = new PrismaRedisMessageStore();
const contextBuilder = new RedisContextBuilder();
const toolRunner = new CompositeToolRunner();
const agentGraph = new CompositeGraph();
const llmRouter = new FallbackRouter();

export class ConversationWorker extends BusWorker {
    protected static async init(bus: EventBus) {
        return new ReasoningEngine(
            agentGraph,
            bus,
            chatStore,
            contextBuilder,
            llmRouter,
            messageStore,
            toolRunner,
        );
    }

    static get(): ReasoningEngine {
        return super.get() as ReasoningEngine;
    }

    protected static async shutdown() {
        const loop = ConversationWorker.get();
        await loop.dispose();
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
    private readonly eventQueue: InternalEvent[] = [];
    private worldModel: WorldModel | null = null;

    constructor(
        private readonly bus: EventBus,
        private readonly completionService: CompletionService,
    ) {
        this.bus.subscribe((evt) => {
            if (this.disposed) return;
            if ("conversationId" in evt) {
                this.enqueue(evt as ConversationEvent).catch(logger.error);
            }
        });
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
            // TODO: persist systemPrompt in conversation

            this.state = SwarmStateMachine.State.ACTIVE;
            // Kick off arbitrator / first agent via ConversationService (stub)
            await this.convoService.handleInternalEvent(convoId, { type: "swarm_started" } as any);
        } catch (err) {
            this.fail(err);
        }
    }

    async handleEvent(convoId: string, ev: InternalEvent): Promise<void> {
        if (this.state === SwarmStateMachine.State.TERMINATED) return;
        // Enqueue and maybe drain if ACTIVE/IDLE
        this.eventQueue.push(ev);
        if (this.state === SwarmStateMachine.State.IDLE) this.drain(convoId).catch(console.error);
    }

    private async drain(convoId: string) {
        if (this.state === SwarmStateMachine.State.PAUSED) return;
        this.state = SwarmStateMachine.State.ACTIVE;
        while (this.eventQueue.length) {
            const ev = this.eventQueue.shift()!;
            await this.convoService.handleInternalEvent(convoId, ev);
        }
        this.state = SwarmStateMachine.State.IDLE;
    }

    async pause(): Promise<void> {
        if (this.state === SwarmStateMachine.State.ACTIVE || this.state === SwarmStateMachine.State.IDLE) {
            this.state = SwarmStateMachine.State.PAUSED;
        }
    }

    async resume(convoId: string): Promise<void> {
        if (this.state === SwarmStateMachine.State.PAUSED) {
            this.state = SwarmStateMachine.State.IDLE; // mark idle so drain() sets to ACTIVE
            await this.drain(convoId);
        }
    }

    async shutdown(): Promise<void> {
        this.eventQueue.length = 0;
        this.state = SwarmStateMachine.State.TERMINATED;
    }

    private fail(err: unknown) {
        console.error("Swarm failure", err);
        this.state = SwarmStateMachine.State.FAILED;
    }
}
