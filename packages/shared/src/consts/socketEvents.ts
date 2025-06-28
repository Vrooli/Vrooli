/* c8 ignore start */
/**
 * Unified Event System Types
 * 
 * This file defines all socket event types used across the Vrooli platform.
 * It provides a single source of truth for event types shared between server and client.
 * 
 * Architecture:
 * - UnifiedEvent: Clean, client-relevant event structure (no server metadata)
 * - EventTypes: Centralized event type constants using hierarchical naming
 * - SwarmSocketEventPayloads: Type-safe payload definitions for each event type
 * 
 * @module socketEvents
 */
import { type AITaskInfo } from "../ai/types.js";
import { type ChatMessage, type ChatParticipant, type Notification } from "../api/types.js";
import { type DeferredDecisionData, type RunTaskInfo } from "../run/types.js";
import { type ChatConfigObject } from "../shape/configs/chat.js";
import { type JOIN_CHAT_ROOM_ERRORS, type JOIN_RUN_ROOM_ERRORS, type JOIN_USER_ROOM_ERRORS, type LEAVE_CHAT_ROOM_ERRORS, type LEAVE_RUN_ROOM_ERRORS, type LEAVE_USER_ROOM_ERRORS } from "./api.js";

/**
 * Base event structure for all socket events.
 * Clean and focused on client-relevant data only.
 * Server-side metadata (delivery guarantees, priority, etc.) is stripped before emission.
 * 
 * @template T - The type of the event payload data
 * @interface UnifiedEvent
 */
export interface UnifiedEvent<T = unknown> {
    /** Unique event identifier for tracking and debugging */
    id: string;
    /** Event type using hierarchical naming (e.g., "swarm/state/changed") */
    type: string;
    /** When the event occurred on the server */
    timestamp: Date;
    /** Event-specific payload data */
    data: T;
}

/**
 * Centralized event type constants using hierarchical naming.
 * This replaces the fragmented event type definitions across the codebase.
 * 
 * Naming convention:
 * - Use forward slashes for hierarchy: category/subcategory/action
 * - Categories map to tiers: swarm (Tier 1), run (Tier 2), step (Tier 3)
 * - Actions use past tense for completed events, present tense for ongoing
 * 
 * @constant EventTypes
 */
export const EventTypes = {

    // ===== Swarm events (Tier 1 - Coordination) =====

    /** Fired when a swarm's execution state changes (e.g., RUNNING → PAUSED) */
    SWARM_STATE_CHANGED: "swarm/state/changed",

    /** Fired when swarm resource allocation or consumption is updated */
    SWARM_RESOURCE_UPDATED: "swarm/resource/updated",

    /** Fired when swarm configuration is modified */
    SWARM_CONFIG_UPDATED: "swarm/config/updated",

    /** Fired when swarm team composition changes */
    SWARM_TEAM_UPDATED: "swarm/team/updated",

    /** Fired when a new swarm goal is created */
    SWARM_GOAL_CREATED: "swarm/goal/created",

    /** Fired when an existing swarm goal is modified */
    SWARM_GOAL_UPDATED: "swarm/goal/updated",

    /** Fired when a swarm goal is successfully completed */
    SWARM_GOAL_COMPLETED: "swarm/goal/completed",

    /** Fired when a swarm goal fails to complete */
    SWARM_GOAL_FAILED: "swarm/goal/failed",

    // ===== Run events (Tier 2 - Process) =====

    /** Fired when a routine run begins execution */
    RUN_STARTED: "run/started",

    /** Fired when a routine run completes successfully */
    RUN_COMPLETED: "run/completed",

    /** Fired when a routine run fails */
    RUN_FAILED: "run/failed",

    /** Fired when a run task is ready for execution */
    RUN_TASK_READY: "run/task/ready",

    /** Fired when a run requires user decision */
    RUN_DECISION_REQUESTED: "run/decision/requested",

    // ===== Step events (Tier 3 - Execution) =====

    /** Fired when an individual step begins execution */
    STEP_STARTED: "step/started",

    /** Fired when a step completes successfully */
    STEP_COMPLETED: "step/completed",

    /** Fired when a step fails */
    STEP_FAILED: "step/failed",

    // ===== Tool events =====

    /** Fired when a tool is invoked */
    TOOL_CALLED: "tool/called",

    /** Fired when a tool execution completes successfully */
    TOOL_COMPLETED: "tool/completed",

    /** Fired when a tool execution fails */
    TOOL_FAILED: "tool/failed",

    /** Fired when a tool requires user approval before execution */
    TOOL_APPROVAL_REQUIRED: "tool/approval/required",

    /** Fired when user approves a pending tool execution */
    TOOL_APPROVAL_GRANTED: "tool/approval/granted",

    /** Fired when user rejects a pending tool execution */
    TOOL_APPROVAL_REJECTED: "tool/approval/rejected",

    /** Fired when tool approval times out without user response */
    TOOL_APPROVAL_TIMEOUT: "tool/approval/timeout",

    // ===== Bot status events =====

    /** Fired when a bot's processing status changes */
    BOT_STATUS_UPDATED: "bot/status/updated",

    /** Fired when a bot starts or stops typing */
    BOT_TYPING_UPDATED: "bot/typing/updated",

    /** Fired when a bot sends a response stream chunk, end, or error */
    BOT_RESPONSE_STREAM: "bot/response/stream",

    /** Fired when a bot sends model reasoning content */
    BOT_MODEL_REASONING_STREAM: "bot/reasoning/stream",

    // ===== Chat events =====

    /** Fired when new messages are added to a chat */
    CHAT_MESSAGE_ADDED: "chat/message/added",

    /** Fired when existing messages are modified */
    CHAT_MESSAGE_UPDATED: "chat/message/updated",

    /** Fired when messages are deleted from a chat */
    CHAT_MESSAGE_REMOVED: "chat/message/removed",

    /** Fired when participants join or leave a chat */
    CHAT_PARTICIPANTS_CHANGED: "chat/participants/changed",

    /** Fired when typing indicators change */
    CHAT_TYPING_UPDATED: "chat/typing/updated",

    // ===== Stream events =====

    /** Fired for each chunk of a response stream */
    RESPONSE_STREAM_CHUNK: "response/stream/chunk",

    /** Fired when a response stream completes */
    RESPONSE_STREAM_END: "response/stream/end",

    /** Fired when a response stream encounters an error */
    RESPONSE_STREAM_ERROR: "response/stream/error",

    /** Fired for each chunk of model reasoning stream */
    REASONING_STREAM_CHUNK: "reasoning/stream/chunk",

    /** Fired when a reasoning stream completes */
    REASONING_STREAM_END: "reasoning/stream/end",

    /** Fired when a reasoning stream encounters an error */
    REASONING_STREAM_ERROR: "reasoning/stream/error",

    // ===== LLM task events =====

    /** Fired when LLM tasks are created or updated */
    LLM_TASKS_UPDATED: "llm/tasks/updated",

    // ===== User events =====

    /** Fired when user's credit balance changes */
    USER_CREDITS_UPDATED: "user/credits/updated",

    /** Fired when user receives a new notification */
    USER_NOTIFICATION_RECEIVED: "user/notification/received",

    // ===== Room events =====

    /** Fired when a client requests to join a room */
    ROOM_JOIN_REQUESTED: "room/join/requested",

    /** Fired when a client requests to leave a room */
    ROOM_LEAVE_REQUESTED: "room/leave/requested",

    // ===== Cancellation events =====

    /** Fired when user requests to cancel ongoing operation */
    CANCELLATION_REQUESTED: "cancellation/requested",
} as const;

/** Type for all event type values */
export type SwarmEventTypeValue = typeof EventTypes[keyof typeof EventTypes];

/**
 * Swarm execution states
 * @enum {string}
 */
export type SwarmState =
    /** Initial state before any execution */
    | "UNINITIALIZED"
    /** Swarm is being initialized */
    | "STARTING"
    /** Swarm is actively executing */
    | "RUNNING"
    /** Swarm execution is temporarily suspended */
    | "PAUSED"
    /** Swarm has finished successfully */
    | "COMPLETED"
    /** Swarm encountered an error */
    | "FAILED"
    /** Swarm was forcefully stopped */
    | "TERMINATED"
    /** Swarm has been moved to long-term storage */
    | "ARCHIVED";

/**
 * Bot processing states
 * @enum {string}
 */
export type BotStatus =
    /** Bot is processing/thinking */
    | "thinking"
    /** Bot is executing a tool */
    | "tool_calling"
    /** Tool execution completed successfully */
    | "tool_completed"
    /** Tool execution failed */
    | "tool_failed"
    /** All processing is complete */
    | "processing_complete"
    /** Internal error occurred */
    | "error_internal"
    /** Tool is waiting for user approval */
    | "tool_pending_approval"
    /** Tool was rejected by user */
    | "tool_rejected_by_user";

export type ReservedSocketEvents = "connect" | "connect_error" | "disconnect";
export type RoomSocketEvents = "joinChatRoom" | "leaveChatRoom" | "joinRunRoom" | "leaveRunRoom" | "joinUserRoom" | "leaveUserRoom" | "requestCancellation";

export interface ErrorPayload {
    /** User-friendly error message */
    message: string;
    /** Standardized error code (e.g., 'LLM_ERROR', 'TOOL_EXECUTION_FAILED', 'TIMEOUT', 'CANCELLED', 'CREDIT_EXHAUSTED') */
    code?: string;
    /** Optional: further technical details or stringified error */
    details?: string;
    /** If the error is related to a specific tool call */
    toolCallId?: string;
    /** Hint for the UI if a retry might be possible */
    retryable?: boolean;
}

/**
 * Type-safe socket event payload map.
 * Maps each event type to its specific payload structure.
 * 
 * @interface SwarmSocketEventPayloads
 */
export interface SwarmSocketEventPayloads {
    // ===== Swarm event payloads =====

    /**
     * Payload for swarm state change events
     * @event SWARM_STATE_CHANGED
     */
    [EventTypes.SWARM_STATE_CHANGED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** Previous state before the change */
        oldState: SwarmState;
        /** New state after the change */
        newState: SwarmState;
        /** Optional human-readable message about the state change */
        message?: string;
    };

    /**
     * Payload for swarm resource update events
     * @event SWARM_RESOURCE_UPDATED
     */
    [EventTypes.SWARM_RESOURCE_UPDATED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** Total credits allocated to the swarm */
        allocated: number;
        /** Credits consumed so far */
        consumed: number;
        /** Credits remaining for use */
        remaining: number;
    };

    /**
     * Payload for swarm configuration update events
     * @event SWARM_CONFIG_UPDATED
     */
    [EventTypes.SWARM_CONFIG_UPDATED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Partial configuration object with updated fields */
        config: Partial<ChatConfigObject> & { __typename?: "ChatConfigObject" };
    };

    /**
     * Payload for swarm team update events
     * @event SWARM_TEAM_UPDATED
     */
    [EventTypes.SWARM_TEAM_UPDATED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** ID of the team assigned to this swarm */
        teamId?: string;
        /** Bot ID of the swarm leader */
        swarmLeader?: string;
        /** Map of subtask IDs to their leader bot IDs */
        subtaskLeaders?: Record<string, string>;
    };

    // ===== Tool event payloads =====

    /**
     * Payload for tool approval request events
     * @event TOOL_APPROVAL_REQUIRED
     */
    [EventTypes.TOOL_APPROVAL_REQUIRED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** Unique ID for this pending approval */
        pendingId: string;
        /** ID of the tool call requiring approval */
        toolCallId: string;
        /** Name of the tool to be executed */
        toolName: string;
        /** Arguments to be passed to the tool */
        toolArguments: Record<string, any>;
        /** ID of the bot requesting tool execution */
        callerBotId: string;
        /** Human-readable name of the bot */
        callerBotName?: string;
        /** Unix timestamp when approval will timeout */
        approvalTimeoutAt?: number;
        /** Estimated cost in credits */
        estimatedCost?: string;
    };

    /**
     * Payload for tool approval rejection events
     * @event TOOL_APPROVAL_REJECTED
     */
    [EventTypes.TOOL_APPROVAL_REJECTED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the pending approval that was rejected */
        pendingId: string;
        /** ID of the tool call that was rejected */
        toolCallId: string;
        /** Name of the tool that was rejected */
        toolName: string;
        /** Reason for rejection */
        reason?: string;
        /** ID of the bot whose request was rejected */
        callerBotId: string;
    };

    // ===== Bot status event payloads =====

    /**
     * Payload for bot status update events
     * @event BOT_STATUS_UPDATED
     */
    [EventTypes.BOT_STATUS_UPDATED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the bot whose status changed */
        botId: string;
        /** Current status of the bot */
        status: BotStatus;
        /** Optional status message */
        message?: string;
        /** Tool execution details (when status involves tools) */
        toolInfo?: {
            /** ID of the tool call */
            callId: string;
            /** Name of the tool */
            name: string;
            /** Stringified tool arguments */
            args?: string;
            /** Stringified tool result */
            result?: string;
            /** Error message if tool failed */
            error?: string;
            /** Pending approval ID */
            pendingId?: string;
            /** Reason for tool rejection */
            reason?: string;
        };
        /** Error details (when status is error_internal) */
        error?: ErrorPayload;
    };

    /**
     * Payload for bot typing update events
     * @event BOT_TYPING_UPDATED
     */
    [EventTypes.BOT_TYPING_UPDATED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** Bot IDs who started typing */
        starting?: string[];
        /** Bot IDs who stopped typing */
        stopping?: string[];
    };

    /**
     * Payload for bot response stream events
     * @event BOT_RESPONSE_STREAM
     */
    [EventTypes.BOT_RESPONSE_STREAM]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the bot sending the stream */
        botId?: string;
        /** Stream type */
        __type: "stream" | "end" | "error";
        /** Text chunk (for stream type) */
        chunk?: string;
        /** Complete final message (for end type) */
        finalMessage?: string;
        /** Error details (for error type) */
        error?: ErrorPayload;
    };

    /**
     * Payload for bot model reasoning stream events
     * @event BOT_MODEL_REASONING_STREAM
     */
    [EventTypes.BOT_MODEL_REASONING_STREAM]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the bot sending reasoning */
        botId?: string;
        /** Stream type */
        __type: "stream" | "end" | "error";
        /** Reasoning text chunk (for stream type) */
        chunk?: string;
        /** Error details (for error type) */
        error?: ErrorPayload;
    };

    // ===== Chat event payloads =====

    /**
     * Payload for chat message addition events
     * @event CHAT_MESSAGE_ADDED
     */
    [EventTypes.CHAT_MESSAGE_ADDED]: {
        /** ID of the chat */
        chatId: string;
        /** Array of new messages */
        messages: ChatMessage[];
    };

    /**
     * Payload for chat message update events
     * @event CHAT_MESSAGE_UPDATED
     */
    [EventTypes.CHAT_MESSAGE_UPDATED]: {
        /** ID of the chat */
        chatId: string;
        /** Array of message updates (must include id) */
        messages: (Partial<ChatMessage> & { id: string })[];
    };

    /**
     * Payload for chat message removal events
     * @event CHAT_MESSAGE_REMOVED
     */
    [EventTypes.CHAT_MESSAGE_REMOVED]: {
        /** ID of the chat */
        chatId: string;
        /** IDs of messages that were removed */
        messageIds: string[];
    };

    /**
     * Payload for chat participant change events
     * @event CHAT_PARTICIPANTS_CHANGED
     */
    [EventTypes.CHAT_PARTICIPANTS_CHANGED]: {
        /** ID of the chat */
        chatId: string;
        /** Participants who joined the chat */
        joining?: Omit<ChatParticipant, "chat">[];
        /** IDs of participants who left the chat */
        leaving?: string[];
    };

    /**
     * Payload for chat typing indicator events
     * @event CHAT_TYPING_UPDATED
     */
    [EventTypes.CHAT_TYPING_UPDATED]: {
        /** ID of the chat */
        chatId: string;
        /** User IDs who started typing */
        starting?: string[];
        /** User IDs who stopped typing */
        stopping?: string[];
    };

    // ===== Stream event payloads =====

    /**
     * Payload for response stream chunk events
     * @event RESPONSE_STREAM_CHUNK
     */
    [EventTypes.RESPONSE_STREAM_CHUNK]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot sending the stream */
        botId?: string;
        /** Text chunk (not accumulated) */
        chunk: string;
    };

    /**
     * Payload for response stream completion events
     * @event RESPONSE_STREAM_END
     */
    [EventTypes.RESPONSE_STREAM_END]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot that sent the stream */
        botId?: string;
        /** Complete final message */
        finalMessage: string;
    };

    /**
     * Payload for response stream error events
     * @event RESPONSE_STREAM_ERROR
     */
    [EventTypes.RESPONSE_STREAM_ERROR]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot that encountered the error */
        botId?: string;
        /** Error details */
        error: ErrorPayload;
    };

    /**
     * Payload for reasoning stream chunk events
     * @event REASONING_STREAM_CHUNK
     */
    [EventTypes.REASONING_STREAM_CHUNK]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot sending reasoning */
        botId?: string;
        /** Reasoning text chunk (not accumulated) */
        chunk: string;
    };

    /**
     * Payload for reasoning stream completion events
     * @event REASONING_STREAM_END
     */
    [EventTypes.REASONING_STREAM_END]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot that sent reasoning */
        botId?: string;
    };

    /**
     * Payload for reasoning stream error events
     * @event REASONING_STREAM_ERROR
     */
    [EventTypes.REASONING_STREAM_ERROR]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot that encountered the error */
        botId?: string;
        /** Error details */
        error: ErrorPayload;
    };

    // ===== LLM task event payloads =====

    /**
     * Payload for LLM task update events
     * @event LLM_TASKS_UPDATED
     */
    [EventTypes.LLM_TASKS_UPDATED]: {
        /** ID of the chat */
        chatId: string;
        /** Full task information for new/complete updates */
        tasks?: AITaskInfo[];
        /** Partial updates for existing tasks */
        updates?: Partial<AITaskInfo>[];
    };

    // ===== Run event payloads =====

    /**
     * Payload for run task ready events
     * @event RUN_TASK_READY
     */
    [EventTypes.RUN_TASK_READY]: {
        /** ID of the run */
        runId: string;
        /** Task information */
        task: RunTaskInfo;
    };

    /**
     * Payload for run decision request events
     * @event RUN_DECISION_REQUESTED
     */
    [EventTypes.RUN_DECISION_REQUESTED]: {
        /** ID of the run */
        runId: string;
        /** Decision data requiring user input */
        decision: DeferredDecisionData;
    };

    // ===== User event payloads =====

    /**
     * Payload for user credit update events
     * @event USER_CREDITS_UPDATED
     */
    [EventTypes.USER_CREDITS_UPDATED]: {
        /** ID of the user */
        userId: string;
        /** Updated credit balance (stringified BigInt) */
        credits: string;
    };

    /**
     * Payload for user notification events
     * @event USER_NOTIFICATION_RECEIVED
     */
    [EventTypes.USER_NOTIFICATION_RECEIVED]: {
        /** ID of the user */
        userId: string;
        /** Notification object */
        notification: Notification;
    };

    // ===== Room event payloads =====

    /**
     * Payload for room join request events
     * @event ROOM_JOIN_REQUESTED
     */
    [EventTypes.ROOM_JOIN_REQUESTED]: {
        /** Type of room to join */
        roomType: "chat" | "run" | "user";
        /** ID of the specific room */
        roomId: string;
    };

    /**
     * Payload for room leave request events
     * @event ROOM_LEAVE_REQUESTED
     */
    [EventTypes.ROOM_LEAVE_REQUESTED]: {
        /** Type of room to leave */
        roomType: "chat" | "run" | "user";
        /** ID of the specific room */
        roomId: string;
    };

    // ===== Cancellation event payloads =====

    /**
     * Payload for cancellation request events
     * @event CANCELLATION_REQUESTED
     */
    [EventTypes.CANCELLATION_REQUESTED]: {
        /** ID of the chat to cancel operations in */
        chatId: string;
    };

    // ===== Additional missing event payloads =====

    /**
     * Payload for tool approval granted events
     * @event TOOL_APPROVAL_GRANTED
     */
    [EventTypes.TOOL_APPROVAL_GRANTED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the pending approval that was granted */
        pendingId: string;
        /** ID of the tool call */
        toolCallId: string;
        /** Name of the approved tool */
        toolName: string;
        /** ID of the bot whose request was approved */
        callerBotId: string;
        /** ID of the user who approved the request */
        approvedBy?: string;
    };

    /**
     * Payload for tool approval timeout events
     * @event TOOL_APPROVAL_TIMEOUT
     */
    [EventTypes.TOOL_APPROVAL_TIMEOUT]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the pending approval that timed out */
        pendingId: string;
        /** ID of the tool call */
        toolCallId: string;
        /** Name of the tool that timed out */
        toolName: string;
        /** ID of the bot whose request timed out */
        callerBotId: string;
        /** Duration before timeout occurred (ms) */
        timeoutDuration?: number;
        /** Whether the request was auto-rejected on timeout */
        autoRejected?: boolean;
    };

    /**
     * Payload for tool called events
     * @event TOOL_CALLED
     */
    [EventTypes.TOOL_CALLED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** Unique ID for this tool call */
        toolCallId: string;
        /** Name of the tool being called */
        toolName: string;
        /** Arguments passed to the tool */
        parameters?: Record<string, unknown>;
        /** ID of the bot making the call */
        callerBotId: string;
    };

    /**
     * Payload for tool completed events
     * @event TOOL_COMPLETED
     */
    [EventTypes.TOOL_COMPLETED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the completed tool call */
        toolCallId: string;
        /** Name of the completed tool */
        toolName: string;
        /** Result returned by the tool */
        result?: unknown;
        /** Execution duration in milliseconds */
        duration?: number;
        /** Credits consumed by this tool call */
        creditsUsed?: string;
        /** ID of the bot that made the call */
        callerBotId: string;
    };

    /**
     * Payload for tool failed events
     * @event TOOL_FAILED
     */
    [EventTypes.TOOL_FAILED]: {
        /** ID of the chat/conversation */
        chatId: string;
        /** ID of the failed tool call */
        toolCallId: string;
        /** Name of the failed tool */
        toolName: string;
        /** Error message */
        error: string;
        /** Execution duration before failure (ms) */
        duration?: number;
        /** ID of the bot that made the call */
        callerBotId: string;
    };
}

/**
 * Legacy type aliases for backward compatibility.
 * @deprecated Use SwarmSocketEventPayloads with EventTypes constants instead
 */
export type UserSocketEventPayloads = {
    [EventTypes.USER_CREDITS_UPDATED]: SwarmSocketEventPayloads[typeof EventTypes.USER_CREDITS_UPDATED];
    [EventTypes.USER_NOTIFICATION_RECEIVED]: SwarmSocketEventPayloads[typeof EventTypes.USER_NOTIFICATION_RECEIVED];
    [EventTypes.ROOM_JOIN_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.ROOM_JOIN_REQUESTED] & { roomType: "user" };
    [EventTypes.ROOM_LEAVE_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.ROOM_LEAVE_REQUESTED] & { roomType: "user" };
}

/**
 * Legacy type aliases for backward compatibility.
 * @deprecated Use SwarmSocketEventPayloads with EventTypes constants instead
 */
export type RunSocketEventPayloads = {
    [EventTypes.RUN_TASK_READY]: SwarmSocketEventPayloads[typeof EventTypes.RUN_TASK_READY];
    [EventTypes.RUN_DECISION_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.RUN_DECISION_REQUESTED];
    [EventTypes.ROOM_JOIN_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.ROOM_JOIN_REQUESTED] & { roomType: "run" };
    [EventTypes.ROOM_LEAVE_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.ROOM_LEAVE_REQUESTED] & { roomType: "run" };
}

/**
 * Legacy type aliases for backward compatibility.
 * @deprecated Use SwarmSocketEventPayloads with EventTypes constants instead
 */
export type ChatSocketEventPayloads = {
    [EventTypes.CHAT_MESSAGE_ADDED]: SwarmSocketEventPayloads[typeof EventTypes.CHAT_MESSAGE_ADDED];
    [EventTypes.CHAT_MESSAGE_UPDATED]: SwarmSocketEventPayloads[typeof EventTypes.CHAT_MESSAGE_UPDATED];
    [EventTypes.CHAT_MESSAGE_REMOVED]: SwarmSocketEventPayloads[typeof EventTypes.CHAT_MESSAGE_REMOVED];
    [EventTypes.CHAT_PARTICIPANTS_CHANGED]: SwarmSocketEventPayloads[typeof EventTypes.CHAT_PARTICIPANTS_CHANGED];
    [EventTypes.CHAT_TYPING_UPDATED]: SwarmSocketEventPayloads[typeof EventTypes.CHAT_TYPING_UPDATED];
    [EventTypes.RESPONSE_STREAM_CHUNK]: SwarmSocketEventPayloads[typeof EventTypes.RESPONSE_STREAM_CHUNK];
    [EventTypes.RESPONSE_STREAM_END]: SwarmSocketEventPayloads[typeof EventTypes.RESPONSE_STREAM_END];
    [EventTypes.RESPONSE_STREAM_ERROR]: SwarmSocketEventPayloads[typeof EventTypes.RESPONSE_STREAM_ERROR];
    [EventTypes.REASONING_STREAM_CHUNK]: SwarmSocketEventPayloads[typeof EventTypes.REASONING_STREAM_CHUNK];
    [EventTypes.REASONING_STREAM_END]: SwarmSocketEventPayloads[typeof EventTypes.REASONING_STREAM_END];
    [EventTypes.REASONING_STREAM_ERROR]: SwarmSocketEventPayloads[typeof EventTypes.REASONING_STREAM_ERROR];
    [EventTypes.LLM_TASKS_UPDATED]: SwarmSocketEventPayloads[typeof EventTypes.LLM_TASKS_UPDATED];
    [EventTypes.BOT_STATUS_UPDATED]: SwarmSocketEventPayloads[typeof EventTypes.BOT_STATUS_UPDATED];
    [EventTypes.TOOL_APPROVAL_REQUIRED]: SwarmSocketEventPayloads[typeof EventTypes.TOOL_APPROVAL_REQUIRED];
    [EventTypes.TOOL_APPROVAL_REJECTED]: SwarmSocketEventPayloads[typeof EventTypes.TOOL_APPROVAL_REJECTED];
    [EventTypes.ROOM_JOIN_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.ROOM_JOIN_REQUESTED] & { roomType: "chat" };
    [EventTypes.ROOM_LEAVE_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.ROOM_LEAVE_REQUESTED] & { roomType: "chat" };
    [EventTypes.CANCELLATION_REQUESTED]: SwarmSocketEventPayloads[typeof EventTypes.CANCELLATION_REQUESTED];
}

/**
 * Unified socket event payload map.
 * Use this instead of individual payload types for full type safety.
 */
export type SocketEventPayloads = SwarmSocketEventPayloads;

export type JoinChatRoomCallbackData = {
    success: boolean;
    error?: keyof typeof JOIN_CHAT_ROOM_ERRORS | string;
}

export type LeaveChatRoomCallbackData = {
    success: boolean;
    error?: keyof typeof LEAVE_CHAT_ROOM_ERRORS;
}

export type RequestCancellationCallbackData = {
    success: boolean;
    error?: string;
    message?: string;
}

export type JoinRunRoomCallbackData = {
    success: boolean;
    error?: keyof typeof JOIN_RUN_ROOM_ERRORS | string;
}

export type LeaveRunRoomCallbackData = {
    success: boolean;
    error?: keyof typeof LEAVE_RUN_ROOM_ERRORS;
}

export type JoinUserRoomCallbackData = {
    success: boolean;
    error?: keyof typeof JOIN_USER_ROOM_ERRORS | string;
}

export type LeaveUserRoomCallbackData = {
    success: boolean;
    error?: keyof typeof LEAVE_USER_ROOM_ERRORS;
}

export type ChatSocketEventCallbackPayloads = {
    joinChatRoom: JoinChatRoomCallbackData;
    leaveChatRoom: LeaveChatRoomCallbackData;
    requestCancellation: RequestCancellationCallbackData;
}

export type RunSocketEventCallbackPayloads = {
    joinRunRoom: JoinRunRoomCallbackData;
    leaveRunRoom: LeaveRunRoomCallbackData;
}

export type UserSocketEventCallbackPayloads = {
    joinUserRoom: JoinUserRoomCallbackData;
    leaveUserRoom: LeaveUserRoomCallbackData;
}

export type SocketEventCallbackPayloads = ChatSocketEventCallbackPayloads & RunSocketEventCallbackPayloads & UserSocketEventCallbackPayloads;

export type SocketEventHandler<T extends SocketEvent> = T extends keyof SocketEventCallbackPayloads
    ? (payload: SocketEventPayloads[T], callback: (data: SocketEventCallbackPayloads[T]) => void) => void
    : (payload: SocketEventPayloads[T]) => void;


export type SocketEvent = keyof SocketEventPayloads;
