import { type BotConfigObject, type ChatConfigObject, type ChatMessage, type SessionUser } from "@local/shared";
import type OpenAI from "openai";
import { type ConversationEvent } from "../bus.js";
import { type ToolInputSchema } from "../mcp/types.js";
import { type WorldModel } from "./worldModel.js";

/** A participant in a conversation – human or bot. */
export interface BotParticipant {
    /** Stable unique identifier (Snowflake, UUID, etc.) */
    id: string;
    config: BotConfigObject;
    /** Display name (for mentions/UX) */
    name: string;
    /** Arbitrary metadata (roles, capabilities, preferences) */
    meta?: {
        role?: string;
        extraTools?: JsonSchema[];   // bot-specific
        disabledTools?: string[];    // names to hide
        systemPrompt?: string;       // override / append
    };
}

/** 
 * A single chat message. 
 * 
 * If this is a new message that hasn't been persisted yet, the `id` and `createdAt` fields will be dummy values.
 */
export type MessageState = Pick<ChatMessage, "id" | "createdAt" | "config" | "language" | "text"> & {
    parent?: {
        id: string;
    } | null;
    user?: {
        id: string;
    } | null;
}

/** Generic success/error wrapper returned by many collaborators. */
export type OkErr<T = unknown> =
    | { ok: true; data: T }
    | { ok: false; error: { code: string; message: string; creditsUsed?: string; } };


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
 * Stats accumulated during a single bot's response generation loop (a single execution of `ReasoningEngine.runLoop`).
 */
export interface ResponseStats {
    /** Number of tool calls made in this response loop. */
    toolCalls: number;
    /** Credits used in this response loop (for LLM calls and tool executions). */
    creditsUsed: bigint;
}

/** @deprecated Use ResponseStats instead */
export interface TurnStats {
    toolCalls: number;
    botToolCalls?: number; // Kept for now if any transitive internal dep, but should be fully removed
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
    /** The current turn counter for the conversation */
    turnCounter: number;
    /** The current turn's stats, for tracking limits */
    turnStats: TurnStats;
    /** The available tools (schemas) for this conversation */
    availableTools: ToolInputSchema[];
    /** The world model configuration for this conversation */
    worldModel: WorldModel;
}

// Added Swarm event type definitions
export interface SwarmInternalEvent {
    type: string;
    conversationId: string;
    payload?: any;
    actingBot?: BotParticipant;
    sessionUser: SessionUser;
}

export interface SwarmStartedEvent extends SwarmInternalEvent {
    type: "swarm_started";
    goal: string;
}
// End of added Swarm event type definitions
