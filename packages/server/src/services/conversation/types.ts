import { type BotConfigObject, type ChatConfigObject, type ChatMessage, type PendingToolCallEntry, type SessionUser, type SwarmResource, type SwarmSubTask, type TeamConfigObject, type ChatToolCallRecord } from "@vrooli/shared";
import type OpenAI from "openai";
import { type ConversationEvent } from "../bus.js";
import type { Tool } from "../mcp/types.js";

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
    /** The chat this tool call is for, if applicable. */
    conversationId?: string;
    /** The bot that's calling the tool, if applicable. */
    callerBotId?: string;
    /** The user session that initiated the action leading to this tool call. */
    sessionUser?: SessionUser;
    /** An AbortSignal to cancel the tool call if needed. */
    signal?: AbortSignal;
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

/**
 * A conversation state
 */
export type ConversationState = {
    /** The conversation ID, stringified */
    id: string;
    /** 
     * The most up-to-date config for the conversation, as stored in the database
     * (though it may not actually be synced to the database yet).
     * 
     * This includes things like limits, stats, and the world model config (which gets hydrated and turned into ConversationState.worldModel).
     */
    config: ChatConfigObject;
    /** Information about the bots (who can respond) in the conversation */
    participants: BotParticipant[];
    /** The available tools (schemas) for this conversation */
    availableTools: Tool[];
    /** 
     * The system message string generated for the leader bot when the swarm is first started. 
     * This is primarily for initialization and context, as individual bots will have their system messages
     * generated dynamically by CompletionService._buildSystemMessage during their turns.
     */
    initialLeaderSystemMessage: string;
    /**
     * The team configuration fetched from the database if teamId is present in the chat config.
     * Contains organizational structure (MOISE+ hierarchy, etc.) for swarm coordination.
     * 
     * This is runtime-only data, not persisted with the conversation state.
     * Rather, it stored with the team object in the database, so that it can be reused by other conversations.
     */
    teamConfig?: TeamConfigObject;
}

// Individual Swarm Event Interfaces
// These events are internal to the swarm's operation and processing.

/** Event triggered when a swarm is started. */
export interface SwarmStartedEvent {
    type: "swarm_started";
    conversationId: string;
    sessionUser: SessionUser;
    goal: string;
}

/** Event triggered when an external message (e.g., from a user) is created that the swarm needs to process. */
export interface SwarmExternalMessageCreatedEvent {
    type: "external_message_created";
    conversationId: string;
    sessionUser: SessionUser;
    payload: {
        messageId: string;
    };
}

/** Event triggered when a new sub-task is created within the swarm. */
export interface SwarmSubTaskCreatedEvent {
    type: "SubTaskCreated";
    conversationId: string;
    sessionUser: SessionUser;
    task: SwarmSubTask;
}

/** Event triggered when the status of a sub-task changes. */
export interface SwarmStatusChangedEvent {
    type: "StatusChanged";
    conversationId: string;
    sessionUser: SessionUser;
    taskId: string;
    newStatus: SwarmSubTask["status"];
    ts: string; // Timestamp
}

/** Event triggered when the progress of a sub-task is updated. */
export interface SwarmProgressUpdatedEvent {
    type: "ProgressUpdated";
    conversationId: string;
    sessionUser: SessionUser;
    taskId: string;
    pct: number; // Percentage complete
    ts: string;  // Timestamp
}

/** Event triggered when a new resource is added by the swarm. */
export interface SwarmResourceAddedEvent {
    type: "ResourceAdded";
    conversationId: string;
    sessionUser: SessionUser;
    resource: SwarmResource;
}

/** Event triggered when a tool call initiated by the swarm finishes. */
export interface SwarmToolCallFinishedEvent {
    type: "ToolCallFinished";
    conversationId: string;
    sessionUser: SessionUser;
    record: ChatToolCallRecord;
}

/** Event triggered when a previously deferred tool call has been approved and needs execution. */
export interface SwarmApprovedToolExecutionRequestEvent {
    type: "ApprovedToolExecutionRequest";
    conversationId: string;
    sessionUser: SessionUser;
    payload: {
        pendingToolCall: PendingToolCallEntry; // Using PendingToolCallEntry from @vrooli/shared
    };
}

/** Event triggered when a previously deferred tool call has been rejected by the user. */
export interface SwarmRejectedToolExecutionRequestEvent {
    type: "RejectedToolExecutionRequest";
    conversationId: string;
    sessionUser: SessionUser;
    payload: {
        pendingToolCall: PendingToolCallEntry;
        reason?: string; // Optional reason for rejection
    };
}

/** Event triggered to update the swarm's shared state (e.g., blackboard, subtasks). */
export interface SwarmStateUpdateEvent {
    type: "SwarmStateUpdate";
    conversationId: string;
    sessionUser: SessionUser; // User context for the update
    payload: {
        // Partial update to ChatConfigObject, focusing on swarm-related fields
        // This allows targeted updates without needing the full config object.
        // Example: { subtasks: newSubtaskList, blackboard: updatedBlackboard }
        // The exact structure of this payload will depend on how SwarmStateMachine consumes it.
        // For now, let's assume it can handle partial updates to specific ChatConfigObject fields relevant to swarm state.
        updatedState: Partial<Pick<ChatConfigObject, "subtasks" | "blackboard" | "resources" | "records" | "eventSubscriptions" | "subtaskLeaders">>;
    };
}

/**
 * Union type for all swarm events.
 * These events are typically handled by the SwarmStateMachine and CompletionService.
 */
export type SwarmEvent =
    | SwarmStartedEvent
    | SwarmExternalMessageCreatedEvent
    | SwarmSubTaskCreatedEvent
    | SwarmStatusChangedEvent
    | SwarmProgressUpdatedEvent
    | SwarmResourceAddedEvent
    | SwarmToolCallFinishedEvent
    | SwarmApprovedToolExecutionRequestEvent
    | SwarmRejectedToolExecutionRequestEvent
    | SwarmStateUpdateEvent;


