import { ChatConfigObject, MessageConfigObject } from "@local/shared";
import OpenAI from "openai";

/** A participant in a conversation – human or bot. */
export interface BotParticipant {
    /** Stable unique identifier (Snowflake, UUID, etc.) */
    id: string;
    /** Display name (for mentions/UX) */
    name: string;
    /** Arbitrary metadata (roles, capabilities, preferences) */
    meta?: Record<string, unknown> & {
        role?: string;
        extraTools?: JsonSchema[];   // bot-specific
        disabledTools?: string[];    // names to hide
        systemPrompt?: string;       // override / append
    };
}

/** A single chat message. */
export interface MessageState {
    /** Database ID – absent before insert. */
    id?: string;
    /** Primary text payload (may contain @mentions etc). */
    content: string;
    /** Author bot ID (assistant role only). */
    botId?: string;
    createdAt: Date;
    parentId?: string | null;
    language?: string;
    config: MessageConfigObject;
}

/** Generic success/error wrapper returned by many collaborators. */
export type OkErr<T = unknown> =
    | { ok: true; data: T }
    | { ok: false; error: { code: string; message: string } };

/* ------------------------------------------------------------------
 * Event‑bus contracts
 * ------------------------------------------------------------------ */

/** Base shape for every domain event flowing through the system. */
export interface ConversationEvent {
    type: string;
    conversationId: string;
    turnId: number;            // <-- NEW
    payload: unknown;          // (messageId, tool result …)
}

export interface MessageCreatedEvent extends ConversationEvent {
    type: "message.created";
    conversationId: string;
    messageId: string;
}

export interface ToolResultEvent extends ConversationEvent {
    type: "tool.result";
    conversationId: string;
    callerBotId: string;
    callId: string;
    output: unknown;
}

export interface ScheduledTickEvent extends ConversationEvent {
    type: "scheduled.tick";
    conversationId: string;
    topic: string;
}

/** Callback signature used when subscribing to the EventBus. */
export type EventCallback = (evt: ConversationEvent) => void | Promise<void>;



/* ------------------------------------------------------------------
 * ContextBuilder – prompt preparation service
 * ------------------------------------------------------------------ */

/**
 * Builds the prompt slice sent to the LLM for **one** bot turn.
 * Typical tasks:
 *  • Retrieve recent messages & vector summaries
 *  • Apply window trimming / summarisation
 *  • Attach tool‑schemas relevant to this bot
 */
export interface ContextBuildResult {
    inputMessages: OpenAI.ChatCompletionMessageParam[];
    toolSchemas: JsonSchema[];
}

/* ------------------------------------------------------------------
 * ToolRunner – executes MCP tool calls (or other functions)
 * ------------------------------------------------------------------ */

export interface ToolMeta {
    conversationId: string;
    callerBotId: string;
}

/* ------------------------------------------------------------------
 * LlmRouter – multi‑provider Responses‑API gateway
 * ------------------------------------------------------------------ */

/** Reduced JSON Schema marker (compatible with OpenAI function‑calling). */
export type JsonSchema = Record<string, unknown>;

/** 
 * Buffer of events grouped by turnId. This is used to batch events for a given turnId together. 
 * 
 * Any events which should be responded to immediately (e.g. tool results, so 
 * bots an use multiple tools in the same turn) are added to the existing turn buffer.
 * 
 * Any external events (e.g. user messages) are added to the buffer for the next turn, 
 * so that the conversation can move forward.
 */
export type TurnQueue = Map<number, ConversationEvent[]>; // keyed by turnId

/**
 * Stats for a single turn. 
 * This is not persisted to the database.
 */
export type TurnStats = {
    /** Tool-calls made by ALL bots this turn (for maxToolCallsPerTurn) */
    toolCalls: number;
    /** Tool-calls made by the CURRENT bot this turn (for maxToolCallsPerBotTurn) */
    botToolCalls: number;
    /** Credits used this turn */
    creditsUsed: bigint;
}

/**
 * A conversation state
 */
export type ConversationState = {
    /** The conversation ID, stringified */
    id: string;
    /** The most up-to-date config for the conversation (may not be synced to the database yet) */
    config: ChatConfigObject;
    /** Information about the bots (who can respond) in the conversation */
    participants: BotParticipant[];
    /** The queue of events for the conversation, keyed by turnId */
    queue: TurnQueue;
    /** The current turn's stats, for tracking limits */
    turnStats: TurnStats;
}
