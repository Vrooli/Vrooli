import { ChatConfigObject } from "@local/shared";
import OpenAI from "openai";

/** A participant in a conversation – human or bot. */
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

/** Persisted conversation header plus participants & metadata. */
export interface Conversation {
    id: string;
    participants: BotParticipant[];
    /** Free‑form JSON – goals, quotas, custom fields */
    meta: ChatConfigObject;
    turnCounter: number;
}

export interface MessageMetaPayload {
    contextHints?: string[];
    eventTopic?: string;
    respondingBots?: Array<string | "@all">;
    role?: "user" | "assistant" | "system";
    turnId?: number;                 // Persisted for crash safety
}

/** A single chat message. */
export interface Message {
    /** Database ID – absent before insert. */
    id?: string;
    role: "user" | "assistant" | "system";
    /** Primary text payload (may contain @mentions etc). */
    content: string;
    /** Author bot ID (assistant role only). */
    botId?: string;
    createdAt: Date;
    language?: string;
    contextHints?: MessageMetaPayload["contextHints"];
    respondingBots?: MessageMetaPayload["respondingBots"];
    eventTopic?: MessageMetaPayload["eventTopic"];
    turnId?: MessageMetaPayload["turnId"];
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
 * AgentGraph – responder selection
 * ------------------------------------------------------------------ */

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

export namespace LLM {
    /** Normalised streaming event emitted by the router. */
    export type StreamEvent =
        | {
            type: "message";
            content: string;
            response_id: string;
        }
        | {
            type: "function_call";
            name: string;
            arguments: unknown;
            call_id: string;
            response_id: string;
        }
        | {
            type: "reasoning";
            content: string;
            response_id: string;
        };
}

export interface StreamOptions {
    /** Model name (provider‑specific). */
    model: string;
    /** Optional reference to continue a threaded conversation on provider side. */
    previous_response_id?: string;
    /** Message / input items per the Responses API. */
    input: Array<Record<string, unknown>>;
    /** JSON‑Schema tool list (built‑ins + custom). */
    tools: JsonSchema[];
    /** Allow model to emit several tool calls at once. */
    parallel_tool_calls?: boolean;
}
