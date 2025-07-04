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
 * - SocketEventPayloads: Type-safe payload definitions for each event type
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
 * Centralized event type constants organized by socket room.
 * This provides better separation of concerns and clearer event organization.
 * 
 * Naming convention:
 * - Use forward slashes for hierarchy: category/subcategory/action
 * - Actions use past tense for completed events, present tense for ongoing
 * 
 * @constant EventTypes
 */
export const EventTypes = {

    // ===== Chat Room Events =====
    CHAT: {
        /** Fired when new messages are added to a chat */
        MESSAGE_ADDED: "chat/message/added",

        /** Fired when existing messages are modified */
        MESSAGE_UPDATED: "chat/message/updated",

        /** Fired when messages are deleted from a chat */
        MESSAGE_REMOVED: "chat/message/removed",

        /** Fired when participants join or leave a chat */
        PARTICIPANTS_CHANGED: "chat/participants/changed",

        /** Fired when typing indicators change */
        TYPING_UPDATED: "chat/typing/updated",

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

        /** Fired when LLM tasks are created or updated */
        LLM_TASKS_UPDATED: "llm/tasks/updated",

        /** Fired when a bot's processing status changes */
        BOT_STATUS_UPDATED: "bot/status/updated",

        /** Fired when a bot starts or stops typing */
        BOT_TYPING_UPDATED: "bot/typing/updated",

        /** Fired when a bot sends a response stream chunk, end, or error */
        BOT_RESPONSE_STREAM: "bot/response/stream",

        /** Fired when a bot sends model reasoning content */
        BOT_MODEL_REASONING_STREAM: "bot/reasoning/stream",

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

        /** Fired when user requests to cancel ongoing operation */
        CANCELLATION_REQUESTED: "cancellation/requested",
    },

    // ===== Swarm Room Events =====
    SWARM: {
        /** Fired when a swarm is started */
        STARTED: "swarm/started",

        /** Fired when a swarm's execution state changes (e.g., RUNNING → PAUSED) */
        STATE_CHANGED: "swarm/state/changed",

        /** Fired when swarm resource allocation or consumption is updated */
        RESOURCE_UPDATED: "swarm/resource/updated",

        /** Fired when swarm configuration is modified */
        CONFIG_UPDATED: "swarm/config/updated",

        /** Fired when swarm team composition changes */
        TEAM_UPDATED: "swarm/team/updated",

        /** Fired when a new swarm goal is created */
        GOAL_CREATED: "swarm/goal/created",

        /** Fired when an existing swarm goal is modified */
        GOAL_UPDATED: "swarm/goal/updated",

        /** Fired when a swarm goal is successfully completed */
        GOAL_COMPLETED: "swarm/goal/completed",

        /** Fired when a swarm goal fails to complete */
        GOAL_FAILED: "swarm/goal/failed",
    },

    // ===== Run Room Events =====
    RUN: {
        /** Fired when a routine run begins execution */
        STARTED: "run/started",

        /** Fired when a routine run completes successfully */
        COMPLETED: "run/completed",

        /** Fired when a routine run fails */
        FAILED: "run/failed",

        /** Fired when a run task is ready for execution */
        TASK_READY: "run/task/ready",

        /** Fired when a run requires user decision */
        DECISION_REQUESTED: "run/decision/requested",

        /** Fired when an individual step begins execution */
        STEP_STARTED: "step/started",

        /** Fired when a step completes successfully */
        STEP_COMPLETED: "step/completed",

        /** Fired when a step fails */
        STEP_FAILED: "step/failed",
    },

    // ===== Safety/Interception Events =====
    SAFETY: {
        /** Fired BEFORE any action execution (tool calls, API calls, routine execution, etc.) 
         * Allows agents to intercept and validate/sanitize inputs before execution */
        PRE_ACTION: "safety/pre_action",

        /** Fired AFTER action completion but BEFORE results are returned to user
         * Allows agents to validate outputs, check for compliance, quality, etc. */
        POST_ACTION: "safety/post_action",

        /** Fired when an emergency stop is triggered */
        EMERGENCY_STOP: "safety/emergency_stop",

        /** Fired when a threat is detected */
        THREAT_DETECTED: "safety/threat_detected",
    },

    // ===== User Room Events =====
    USER: {
        /** Fired when user's credit balance changes */
        CREDITS_UPDATED: "user/credits/updated",

        /** Fired when user receives a new notification */
        NOTIFICATION_RECEIVED: "user/notification/received",
    },

    // ===== Room Management Events (shared across rooms) =====
    ROOM: {
        /** Fired when a client requests to join a room */
        JOIN_REQUESTED: "room/join/requested",

        /** Fired when a client requests to leave a room */
        LEAVE_REQUESTED: "room/leave/requested",
    },
} as const;

/** Type for all event type values */
export type SwarmEventTypeValue =
    | typeof EventTypes.CHAT[keyof typeof EventTypes.CHAT]
    | typeof EventTypes.SWARM[keyof typeof EventTypes.SWARM]
    | typeof EventTypes.RUN[keyof typeof EventTypes.RUN]
    | typeof EventTypes.SAFETY[keyof typeof EventTypes.SAFETY]
    | typeof EventTypes.USER[keyof typeof EventTypes.USER]
    | typeof EventTypes.ROOM[keyof typeof EventTypes.ROOM];

// Convenience aliases for backward compatibility
export const ChatEvents = EventTypes.CHAT;
export const SwarmEvents = EventTypes.SWARM;
export const RunEvents = EventTypes.RUN;
export const SafetyEvents = EventTypes.SAFETY;
export const UserEvents = EventTypes.USER;
export const RoomEvents = EventTypes.ROOM;

// Flattened event types for backward compatibility
export const FlatEventTypes = {
    // Chat events
    CHAT_MESSAGE_ADDED: EventTypes.CHAT.MESSAGE_ADDED,
    CHAT_MESSAGE_UPDATED: EventTypes.CHAT.MESSAGE_UPDATED,
    CHAT_MESSAGE_REMOVED: EventTypes.CHAT.MESSAGE_REMOVED,
    CHAT_PARTICIPANTS_CHANGED: EventTypes.CHAT.PARTICIPANTS_CHANGED,
    CHAT_TYPING_UPDATED: EventTypes.CHAT.TYPING_UPDATED,
    RESPONSE_STREAM_CHUNK: EventTypes.CHAT.RESPONSE_STREAM_CHUNK,
    RESPONSE_STREAM_END: EventTypes.CHAT.RESPONSE_STREAM_END,
    RESPONSE_STREAM_ERROR: EventTypes.CHAT.RESPONSE_STREAM_ERROR,
    REASONING_STREAM_CHUNK: EventTypes.CHAT.REASONING_STREAM_CHUNK,
    REASONING_STREAM_END: EventTypes.CHAT.REASONING_STREAM_END,
    REASONING_STREAM_ERROR: EventTypes.CHAT.REASONING_STREAM_ERROR,
    LLM_TASKS_UPDATED: EventTypes.CHAT.LLM_TASKS_UPDATED,
    BOT_STATUS_UPDATED: EventTypes.CHAT.BOT_STATUS_UPDATED,
    BOT_TYPING_UPDATED: EventTypes.CHAT.BOT_TYPING_UPDATED,
    BOT_RESPONSE_STREAM: EventTypes.CHAT.BOT_RESPONSE_STREAM,
    BOT_MODEL_REASONING_STREAM: EventTypes.CHAT.BOT_MODEL_REASONING_STREAM,
    TOOL_CALLED: EventTypes.CHAT.TOOL_CALLED,
    TOOL_COMPLETED: EventTypes.CHAT.TOOL_COMPLETED,
    TOOL_FAILED: EventTypes.CHAT.TOOL_FAILED,
    TOOL_APPROVAL_REQUIRED: EventTypes.CHAT.TOOL_APPROVAL_REQUIRED,
    TOOL_APPROVAL_GRANTED: EventTypes.CHAT.TOOL_APPROVAL_GRANTED,
    TOOL_APPROVAL_REJECTED: EventTypes.CHAT.TOOL_APPROVAL_REJECTED,
    TOOL_APPROVAL_TIMEOUT: EventTypes.CHAT.TOOL_APPROVAL_TIMEOUT,
    CANCELLATION_REQUESTED: EventTypes.CHAT.CANCELLATION_REQUESTED,
    // Swarm events
    SWARM_STATE_CHANGED: EventTypes.SWARM.STATE_CHANGED,
    SWARM_RESOURCE_UPDATED: EventTypes.SWARM.RESOURCE_UPDATED,
    SWARM_CONFIG_UPDATED: EventTypes.SWARM.CONFIG_UPDATED,
    SWARM_TEAM_UPDATED: EventTypes.SWARM.TEAM_UPDATED,
    SWARM_GOAL_CREATED: EventTypes.SWARM.GOAL_CREATED,
    SWARM_GOAL_UPDATED: EventTypes.SWARM.GOAL_UPDATED,
    SWARM_GOAL_COMPLETED: EventTypes.SWARM.GOAL_COMPLETED,
    SWARM_GOAL_FAILED: EventTypes.SWARM.GOAL_FAILED,
    // Run events
    RUN_STARTED: EventTypes.RUN.STARTED,
    RUN_COMPLETED: EventTypes.RUN.COMPLETED,
    RUN_FAILED: EventTypes.RUN.FAILED,
    RUN_TASK_READY: EventTypes.RUN.TASK_READY,
    RUN_DECISION_REQUESTED: EventTypes.RUN.DECISION_REQUESTED,
    STEP_STARTED: EventTypes.RUN.STEP_STARTED,
    STEP_COMPLETED: EventTypes.RUN.STEP_COMPLETED,
    STEP_FAILED: EventTypes.RUN.STEP_FAILED,
    // User events
    USER_CREDITS_UPDATED: EventTypes.USER.CREDITS_UPDATED,
    USER_NOTIFICATION_RECEIVED: EventTypes.USER.NOTIFICATION_RECEIVED,
    // Room events
    ROOM_JOIN_REQUESTED: EventTypes.ROOM.JOIN_REQUESTED,
    ROOM_LEAVE_REQUESTED: EventTypes.ROOM.LEAVE_REQUESTED,
    // Safety events
    SAFETY_PRE_ACTION: EventTypes.SAFETY.PRE_ACTION,
    SAFETY_POST_ACTION: EventTypes.SAFETY.POST_ACTION,
} as const;

/**
 * States for execution state machine
 * @enum {string}
 */
export type StateMachineState =
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
 * Type-safe socket event payload map organized by room.
 * Maps each event type to its specific payload structure.
 * 
 * @interface SocketEventPayloads
 */
export interface SocketEventPayloads {
    // ===== Chat Room event payloads =====

    /**
     * Payload for chat message addition events
     * @event CHAT_MESSAGE_ADDED
     */
    [EventTypes.CHAT.MESSAGE_ADDED]: {
        /** ID of the chat */
        chatId: string;
        /** Array of new messages */
        messages: ChatMessage[];
    };

    /**
     * Payload for chat message update events
     * @event CHAT_MESSAGE_UPDATED
     */
    [EventTypes.CHAT.MESSAGE_UPDATED]: {
        /** ID of the chat */
        chatId: string;
        /** Array of message updates (must include id) */
        messages: (Partial<ChatMessage> & { id: string })[];
    };

    /**
     * Payload for chat message removal events
     * @event CHAT_MESSAGE_REMOVED
     */
    [EventTypes.CHAT.MESSAGE_REMOVED]: {
        /** ID of the chat */
        chatId: string;
        /** IDs of messages that were removed */
        messageIds: string[];
    };

    /**
     * Payload for chat participant change events
     * @event CHAT_PARTICIPANTS_CHANGED
     */
    [EventTypes.CHAT.PARTICIPANTS_CHANGED]: {
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
    [EventTypes.CHAT.TYPING_UPDATED]: {
        /** ID of the chat */
        chatId: string;
        /** User IDs who started typing */
        starting?: string[];
        /** User IDs who stopped typing */
        stopping?: string[];
    };

    /**
     * Payload for response stream chunk events
     * @event RESPONSE_STREAM_CHUNK
     */
    [EventTypes.CHAT.RESPONSE_STREAM_CHUNK]: {
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
    [EventTypes.CHAT.RESPONSE_STREAM_END]: {
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
    [EventTypes.CHAT.RESPONSE_STREAM_ERROR]: {
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
    [EventTypes.CHAT.REASONING_STREAM_CHUNK]: {
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
    [EventTypes.CHAT.REASONING_STREAM_END]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot that sent reasoning */
        botId?: string;
    };

    /**
     * Payload for reasoning stream error events
     * @event REASONING_STREAM_ERROR
     */
    [EventTypes.CHAT.REASONING_STREAM_ERROR]: {
        /** ID of the chat */
        chatId: string;
        /** ID of the bot that encountered the error */
        botId?: string;
        /** Error details */
        error: ErrorPayload;
    };

    /**
     * Payload for LLM task update events
     * @event LLM_TASKS_UPDATED
     */
    [EventTypes.CHAT.LLM_TASKS_UPDATED]: {
        /** ID of the chat */
        chatId: string;
        /** Full task information for new/complete updates */
        tasks?: AITaskInfo[];
        /** Partial updates for existing tasks */
        updates?: Partial<AITaskInfo>[];
    };

    /**
     * Payload for bot status update events
     * @event BOT_STATUS_UPDATED
     */
    [EventTypes.CHAT.BOT_STATUS_UPDATED]: {
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
    [EventTypes.CHAT.BOT_TYPING_UPDATED]: {
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
    [EventTypes.CHAT.BOT_RESPONSE_STREAM]: {
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
    [EventTypes.CHAT.BOT_MODEL_REASONING_STREAM]: {
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

    /**
     * Payload for tool called events
     * @event TOOL_CALLED
     */
    [EventTypes.CHAT.TOOL_CALLED]: {
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
    [EventTypes.CHAT.TOOL_COMPLETED]: {
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
    [EventTypes.CHAT.TOOL_FAILED]: {
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

    /**
     * Payload for tool approval request events
     * @event TOOL_APPROVAL_REQUIRED
     */
    [EventTypes.CHAT.TOOL_APPROVAL_REQUIRED]: {
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
     * Payload for tool approval granted events
     * @event TOOL_APPROVAL_GRANTED
     */
    [EventTypes.CHAT.TOOL_APPROVAL_GRANTED]: {
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
     * Payload for tool approval rejection events
     * @event TOOL_APPROVAL_REJECTED
     */
    [EventTypes.CHAT.TOOL_APPROVAL_REJECTED]: {
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

    /**
     * Payload for tool approval timeout events
     * @event TOOL_APPROVAL_TIMEOUT
     */
    [EventTypes.CHAT.TOOL_APPROVAL_TIMEOUT]: {
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
     * Payload for cancellation request events
     * @event CANCELLATION_REQUESTED
     */
    [EventTypes.CHAT.CANCELLATION_REQUESTED]: {
        /** ID of the chat to cancel operations in */
        chatId: string;
    };

    // ===== Swarm Room event payloads =====

    /**
     * Payload for the start of a swarm
     * @event SWARM_STARTED
     */
    [EventTypes.SWARM.STARTED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** The goal of the swarm */
        goal: string;
        /** The ID of the bot or user that started the swarm */
        initiatingUser: string;
    };

    /**
     * Payload for swarm state change events
     * @event SWARM_STATE_CHANGED
     */
    [EventTypes.SWARM.STATE_CHANGED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** Previous state before the change */
        oldState: StateMachineState;
        /** New state after the change */
        newState: StateMachineState;
        /** Optional human-readable message about the state change */
        message?: string;
    };

    /**
     * Payload for swarm resource update events
     * @event SWARM_RESOURCE_UPDATED
     */
    [EventTypes.SWARM.RESOURCE_UPDATED]: {
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
    [EventTypes.SWARM.CONFIG_UPDATED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Partial configuration object with updated fields */
        config: Partial<ChatConfigObject> & { __typename?: "ChatConfigObject" };
    };

    /**
     * Payload for swarm team update events
     * @event SWARM_TEAM_UPDATED
     */
    [EventTypes.SWARM.TEAM_UPDATED]: {
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

    /**
     * Payload for swarm goal created events
     * @event SWARM_GOAL_CREATED
     */
    [EventTypes.SWARM.GOAL_CREATED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** ID of the newly created goal */
        goalId: string;
        /** Goal description */
        description: string;
        /** Goal priority level */
        priority?: number;
    };

    /**
     * Payload for swarm goal updated events
     * @event SWARM_GOAL_UPDATED
     */
    [EventTypes.SWARM.GOAL_UPDATED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** ID of the updated goal */
        goalId: string;
        /** Updated goal description */
        description?: string;
        /** Updated goal priority level */
        priority?: number;
        /** Updated goal progress (0-1) */
        progress?: number;
    };

    /**
     * Payload for swarm goal completed events
     * @event SWARM_GOAL_COMPLETED
     */
    [EventTypes.SWARM.GOAL_COMPLETED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** ID of the completed goal */
        goalId: string;
        /** Goal description */
        description: string;
        /** Time taken to complete the goal (ms) */
        duration?: number;
    };

    /**
     * Payload for swarm goal failed events
     * @event SWARM_GOAL_FAILED
     */
    [EventTypes.SWARM.GOAL_FAILED]: {
        /** ID of the chat/conversation containing the swarm */
        chatId: string;
        /** Unique identifier of the swarm */
        swarmId: string;
        /** ID of the failed goal */
        goalId: string;
        /** Goal description */
        description: string;
        /** Reason for failure */
        error: string;
        /** Time spent before failure (ms) */
        duration?: number;
    };

    // ===== Run Room event payloads =====

    /**
     * Payload for run started events
     * @event RUN_STARTED
     */
    [EventTypes.RUN.STARTED]: {
        /** ID of the run */
        runId: string;
        /** ID of the routine being executed */
        routineId: string;
        /** Run configuration parameters */
        inputs?: Record<string, unknown>;
        /** Estimated completion time (ms) */
        estimatedDuration?: number;
    };

    /**
     * Payload for run completed events
     * @event RUN_COMPLETED
     */
    [EventTypes.RUN.COMPLETED]: {
        /** ID of the run */
        runId: string;
        /** Run outputs */
        outputs?: Record<string, unknown>;
        /** Actual execution time (ms) */
        duration?: number;
        /** Success status message */
        message?: string;
    };

    /**
     * Payload for run failed events
     * @event RUN_FAILED
     */
    [EventTypes.RUN.FAILED]: {
        /** ID of the run */
        runId: string;
        /** Error message */
        error: string;
        /** Execution time before failure (ms) */
        duration?: number;
        /** Whether retry is possible */
        retryable?: boolean;
    };

    /**
     * Payload for run task ready events
     * @event RUN_TASK_READY
     */
    [EventTypes.RUN.TASK_READY]: {
        /** ID of the run */
        runId: string;
        /** Task information */
        task: RunTaskInfo;
    };

    /**
     * Payload for run decision request events
     * @event RUN_DECISION_REQUESTED
     */
    [EventTypes.RUN.DECISION_REQUESTED]: {
        /** ID of the run */
        runId: string;
        /** Decision data requiring user input */
        decision: DeferredDecisionData;
    };

    /**
     * Payload for step started events
     * @event STEP_STARTED
     */
    [EventTypes.RUN.STEP_STARTED]: {
        /** ID of the run */
        runId: string;
        /** ID of the step */
        stepId: string;
        /** Step name/description */
        name: string;
        /** Step inputs */
        inputs?: Record<string, unknown>;
    };

    /**
     * Payload for step completed events
     * @event STEP_COMPLETED
     */
    [EventTypes.RUN.STEP_COMPLETED]: {
        /** ID of the run */
        runId: string;
        /** ID of the routine */
        routineId: string;
        /** ID of the step */
        stepId: string;
        /** Step outputs */
        outputs?: Record<string, unknown>;
        /** Whether the routine completed successfully */
        success: boolean;
        /** Performance metrics for optimization agents */
        metrics: {
            /** Total execution time in milliseconds */
            duration: number;
            /** Credits consumed during execution */
            creditsUsed: string;
            /** Number of steps executed */
            stepsExecuted: number;
            /** Number of steps that failed */
            stepsFailed: number;
            /** Memory usage in bytes (if available) */
            memoryUsed?: number;
            /** Number of API calls made */
            apiCallsCount?: number;
            /** Number of tool calls made */
            toolCallsCount?: number;
            /** Custom metrics specific to the routine */
            customMetrics?: Record<string, number | string>;
        };
        /** Error message if routine failed */
        error?: string;
        /** Execution context */
        context?: {
            /** User who initiated the run */
            userId?: string;
            /** Team context if applicable */
            teamId?: string;
            /** Parent swarm if part of swarm execution */
            swarmId?: string;
        };
    };

    /**
     * Payload for step failed events
     * @event STEP_FAILED
     */
    [EventTypes.RUN.STEP_FAILED]: {
        /** ID of the run */
        runId: string;
        /** ID of the step */
        stepId: string;
        /** Error message */
        error: string;
        /** Execution time before failure (ms) */
        duration?: number;
    };

    // ===== User Room event payloads =====

    /**
     * Payload for user credit update events
     * @event USER_CREDITS_UPDATED
     */
    [EventTypes.USER.CREDITS_UPDATED]: {
        /** ID of the user */
        userId: string;
        /** Updated credit balance (stringified BigInt) */
        credits: string;
    };

    /**
     * Payload for user notification events
     * @event USER_NOTIFICATION_RECEIVED
     */
    [EventTypes.USER.NOTIFICATION_RECEIVED]: {
        /** ID of the user */
        userId: string;
        /** Notification object */
        notification: Notification;
    };

    // ===== Safety/Interception event payloads =====

    /**
     * Payload for pre-action safety checks
     * @event SAFETY_PRE_ACTION
     */
    [EventTypes.SAFETY.PRE_ACTION]: {
        /** Type of action being performed */
        actionType: "tool_call" | "api_call" | "routine_execution" | "data_access" | "external_request";
        /** ID of the entity performing the action (user, bot, agent, etc.) */
        actorId: string;
        /** Type of actor (user, bot, agent, system) */
        actorType: "user" | "bot" | "agent" | "system";
        /** Context where action is happening (chat, run, etc.) */
        contextId?: string;
        /** Context type */
        contextType?: "chat" | "run" | "swarm";
        /** Details about the action being attempted */
        actionDetails: {
            /** Name or identifier of the action */
            name: string;
            /** Parameters or payload of the action */
            parameters?: Record<string, unknown>;
            /** Target resource if applicable */
            targetResource?: string;
            /** Estimated cost in credits if applicable */
            estimatedCost?: string;
        };
        /** Unique ID for tracking this action through pre/post */
        actionId: string;
    };

    /**
     * Payload for post-action safety checks
     * @event SAFETY_POST_ACTION
     */
    [EventTypes.SAFETY.POST_ACTION]: {
        /** Same action ID from pre_action for correlation */
        actionId: string;
        /** Type of action that was performed */
        actionType: "tool_call" | "api_call" | "routine_execution" | "data_access" | "external_request";
        /** ID of the entity that performed the action */
        actorId: string;
        /** Type of actor */
        actorType: "user" | "bot" | "agent" | "system";
        /** Context where action happened */
        contextId?: string;
        /** Context type */
        contextType?: "chat" | "run" | "swarm";
        /** Action name or identifier */
        actionName: string;
        /** Result of the action */
        result?: unknown;
        /** Whether the action succeeded */
        success: boolean;
        /** Error if action failed */
        error?: string;
        /** Duration of action execution in milliseconds */
        duration?: number;
        /** Actual credits consumed */
        creditsUsed?: string;
    };

    /**
     * Payload for emergency stop events
     * @event EMERGENCY_STOP
     */
    [EventTypes.SAFETY.EMERGENCY_STOP]: {
        /** ID of the entity that triggered the emergency stop */
        actorId: string;
        /** Type of actor */
        actorType: "user" | "bot" | "agent" | "system";
        /** Context where emergency stop happened */
        contextId?: string;
        /** Context type */
        contextType?: "chat" | "run" | "swarm";
    };

    /**
     * Payload for threat detected events
     * @event THREAT_DETECTED
     */
    [EventTypes.SAFETY.THREAT_DETECTED]: {
        /** ID of the entity that triggered the threat */
        actorId: string;
        /** Type of actor */
        actorType: "user" | "bot" | "agent" | "system";
        /** Context where threat happened */
        contextId?: string;
        /** Context type */
        contextType?: "chat" | "run" | "swarm";
    };

    // ===== Room Management event payloads =====

    /**
     * Payload for room join request events
     * @event ROOM_JOIN_REQUESTED
     */
    [EventTypes.ROOM.JOIN_REQUESTED]: {
        /** Type of room to join */
        roomType: "chat" | "run" | "user" | "swarm";
        /** ID of the specific room */
        roomId: string;
    };

    /**
     * Payload for room leave request events
     * @event ROOM_LEAVE_REQUESTED
     */
    [EventTypes.ROOM.LEAVE_REQUESTED]: {
        /** Type of room to leave */
        roomType: "chat" | "run" | "user" | "swarm";
        /** ID of the specific room */
        roomId: string;
    };
}

/**
 * Legacy type aliases for backward compatibility.
 * @deprecated Use SocketEventPayloads with EventTypes constants instead
 */
export type UserSocketEventPayloads = {
    [EventTypes.USER.CREDITS_UPDATED]: SocketEventPayloads[typeof EventTypes.USER.CREDITS_UPDATED];
    [EventTypes.USER.NOTIFICATION_RECEIVED]: SocketEventPayloads[typeof EventTypes.USER.NOTIFICATION_RECEIVED];
    [EventTypes.ROOM.JOIN_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.JOIN_REQUESTED] & { roomType: "user" };
    [EventTypes.ROOM.LEAVE_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.LEAVE_REQUESTED] & { roomType: "user" };
}

/**
 * Legacy type aliases for backward compatibility.
 * @deprecated Use SocketEventPayloads with EventTypes constants instead
 */
export type RunSocketEventPayloads = {
    [EventTypes.RUN.TASK_READY]: SocketEventPayloads[typeof EventTypes.RUN.TASK_READY];
    [EventTypes.RUN.DECISION_REQUESTED]: SocketEventPayloads[typeof EventTypes.RUN.DECISION_REQUESTED];
    [EventTypes.ROOM.JOIN_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.JOIN_REQUESTED] & { roomType: "run" };
    [EventTypes.ROOM.LEAVE_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.LEAVE_REQUESTED] & { roomType: "run" };
}

/**
 * Legacy type aliases for backward compatibility.
 * @deprecated Use SocketEventPayloads with EventTypes constants instead
 */
export type ChatSocketEventPayloads = {
    [EventTypes.CHAT.MESSAGE_ADDED]: SocketEventPayloads[typeof EventTypes.CHAT.MESSAGE_ADDED];
    [EventTypes.CHAT.MESSAGE_UPDATED]: SocketEventPayloads[typeof EventTypes.CHAT.MESSAGE_UPDATED];
    [EventTypes.CHAT.MESSAGE_REMOVED]: SocketEventPayloads[typeof EventTypes.CHAT.MESSAGE_REMOVED];
    [EventTypes.CHAT.PARTICIPANTS_CHANGED]: SocketEventPayloads[typeof EventTypes.CHAT.PARTICIPANTS_CHANGED];
    [EventTypes.CHAT.TYPING_UPDATED]: SocketEventPayloads[typeof EventTypes.CHAT.TYPING_UPDATED];
    [EventTypes.CHAT.RESPONSE_STREAM_CHUNK]: SocketEventPayloads[typeof EventTypes.CHAT.RESPONSE_STREAM_CHUNK];
    [EventTypes.CHAT.RESPONSE_STREAM_END]: SocketEventPayloads[typeof EventTypes.CHAT.RESPONSE_STREAM_END];
    [EventTypes.CHAT.RESPONSE_STREAM_ERROR]: SocketEventPayloads[typeof EventTypes.CHAT.RESPONSE_STREAM_ERROR];
    [EventTypes.CHAT.REASONING_STREAM_CHUNK]: SocketEventPayloads[typeof EventTypes.CHAT.REASONING_STREAM_CHUNK];
    [EventTypes.CHAT.REASONING_STREAM_END]: SocketEventPayloads[typeof EventTypes.CHAT.REASONING_STREAM_END];
    [EventTypes.CHAT.REASONING_STREAM_ERROR]: SocketEventPayloads[typeof EventTypes.CHAT.REASONING_STREAM_ERROR];
    [EventTypes.CHAT.LLM_TASKS_UPDATED]: SocketEventPayloads[typeof EventTypes.CHAT.LLM_TASKS_UPDATED];
    [EventTypes.CHAT.BOT_STATUS_UPDATED]: SocketEventPayloads[typeof EventTypes.CHAT.BOT_STATUS_UPDATED];
    [EventTypes.CHAT.TOOL_APPROVAL_REQUIRED]: SocketEventPayloads[typeof EventTypes.CHAT.TOOL_APPROVAL_REQUIRED];
    [EventTypes.CHAT.TOOL_APPROVAL_REJECTED]: SocketEventPayloads[typeof EventTypes.CHAT.TOOL_APPROVAL_REJECTED];
    [EventTypes.ROOM.JOIN_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.JOIN_REQUESTED] & { roomType: "chat" };
    [EventTypes.ROOM.LEAVE_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.LEAVE_REQUESTED] & { roomType: "chat" };
    [EventTypes.CHAT.CANCELLATION_REQUESTED]: SocketEventPayloads[typeof EventTypes.CHAT.CANCELLATION_REQUESTED];
}

/**
 * Swarm room socket event payloads
 */
export type SwarmRoomSocketEventPayloads = {
    [EventTypes.SWARM.STATE_CHANGED]: SocketEventPayloads[typeof EventTypes.SWARM.STATE_CHANGED];
    [EventTypes.SWARM.RESOURCE_UPDATED]: SocketEventPayloads[typeof EventTypes.SWARM.RESOURCE_UPDATED];
    [EventTypes.SWARM.CONFIG_UPDATED]: SocketEventPayloads[typeof EventTypes.SWARM.CONFIG_UPDATED];
    [EventTypes.SWARM.TEAM_UPDATED]: SocketEventPayloads[typeof EventTypes.SWARM.TEAM_UPDATED];
    [EventTypes.SWARM.GOAL_CREATED]: SocketEventPayloads[typeof EventTypes.SWARM.GOAL_CREATED];
    [EventTypes.SWARM.GOAL_UPDATED]: SocketEventPayloads[typeof EventTypes.SWARM.GOAL_UPDATED];
    [EventTypes.SWARM.GOAL_COMPLETED]: SocketEventPayloads[typeof EventTypes.SWARM.GOAL_COMPLETED];
    [EventTypes.SWARM.GOAL_FAILED]: SocketEventPayloads[typeof EventTypes.SWARM.GOAL_FAILED];
    [EventTypes.ROOM.JOIN_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.JOIN_REQUESTED] & { roomType: "swarm" };
    [EventTypes.ROOM.LEAVE_REQUESTED]: SocketEventPayloads[typeof EventTypes.ROOM.LEAVE_REQUESTED] & { roomType: "swarm" };
}

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
