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
import { type DeferredDecisionData } from "../run/types.js";
import { type ChatConfigObject } from "../shape/configs/chat.js";
import { type JOIN_CHAT_ROOM_ERRORS, type JOIN_RUN_ROOM_ERRORS, type JOIN_USER_ROOM_ERRORS, type LEAVE_CHAT_ROOM_ERRORS, type LEAVE_RUN_ROOM_ERRORS, type LEAVE_USER_ROOM_ERRORS } from "./api.js";

/**
 * Base event structure for all socket events.
 * Clean and focused on client-relevant data only.
 * 
 * The socket service should automatically wrap the event in this structure before sending it to the client, 
 * and the client should automatically unwrap it before processing. This ensures that we don't have 
 * to worry about building the event structure manually, and simplifies emitting events.
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
 * Error payload structure for all error events.
 * 
 * This is a standardized error payload that can be used for all error events.
 * It's used to standardize error handling and reporting across the platform.
 * 
 * @interface ErrorPayload
 */
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
 * Base event structure for chat-specific events.
 * 
 * @interface ChatEventData
 */
export interface ChatEventData {
    /** 
     * The ID of the chat the event is associated with.
     * This doesn't strictly need to be in the event data, since if you're receiving an event from a chat room,
     * you already know the chat ID. But it can help simplify event emitting if we include it, since the event bus
     * can parse this out of the event data to emit the event to the correct chat room.
     */
    chatId: string;
}

/**
 * Base event structure for swarm-specific events.
 * Since swarms are currently tied 1-to-1 with a chat, we can use the chat ID to identify the swarm.
 * 
 * @interface SwarmEventData
 */
export type SwarmEventData = ChatEventData;

/**
 * Base event structure for run-specific events.
 * 
 * @interface RunEventData
 */
export interface RunEventData {
    /** The ID of the run the event is associated with */
    runId: string;
}

/**
 * Data sensitivity types for access control.
 * 
 * @enum DataSensitivityType
 */
export enum DataSensitivityType {
    PII = "PII",
    PHI = "PHI",
    FINANCIAL = "FINANCIAL",
    CREDENTIAL = "CREDENTIAL",
    PROPRIETARY = "PROPRIETARY",
    PUBLIC = "PUBLIC",
}

/**
 * Base event structure for security-specific events.
 * These always occur within the context of a swarm, so they extend the SwarmEventData interface 
 * and use the same room ID as swarms.
 * 
 * @interface SecurityEventData
 */
export interface SecurityEventData extends SwarmEventData {
    /** Entity that triggered the security event */
    triggeredBy: {
        id: string;
        type: "user" | "bot" | "system";
        name?: string;
    };
    /** Severity level of the security event */
    severity: "info" | "warning" | "critical";
    /** Additional context about the security event */
    context?: Record<string, unknown>;
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

        /** Fired when user requests to cancel ongoing operation */
        CANCELLATION_REQUESTED: "cancellation/requested",
    },

    // ===== Tool Events =====
    TOOL: {
        /** Fired when a tool is invoked */
        CALLED: "tool/called",

        /** Fired when a tool execution completes successfully */
        COMPLETED: "tool/completed",

        /** Fired when a tool execution fails */
        FAILED: "tool/failed",

        /** Fired when a tool requires user approval before execution */
        APPROVAL_REQUIRED: "tool/approval/required",

        /** Fired when user approves a pending tool execution */
        APPROVAL_GRANTED: "tool/approval/granted",

        /** Fired when user rejects a pending tool execution */
        APPROVAL_REJECTED: "tool/approval/rejected",

        /** Fired when tool approval times out without user response */
        APPROVAL_TIMEOUT: "tool/approval/timeout",

        /** Fired before tool execution for pre-checks */
        EXECUTION_REQUESTED: "tool/execution/requested",

        /** Fired after tool execution for post-checks */
        EXECUTION_COMPLETED: "tool/execution/completed",
    },

    // ===== Data Access Events =====
    DATA: {
        /** Fired before data access for pre-checks */
        ACCESS_REQUESTED: "data/access/requested",

        /** Fired after data access for post-checks */
        ACCESS_COMPLETED: "data/access/completed",

        /** Fired when data access is denied */
        ACCESS_DENIED: "data/access/denied",

        /** Fired before data modification */
        MODIFICATION_REQUESTED: "data/modification/requested",

        /** Fired after data modification */
        MODIFICATION_COMPLETED: "data/modification/completed",
    },

    // ===== API Events =====
    API: {
        /** Fired before external API call */
        CALL_REQUESTED: "api/call/requested",

        /** Fired after external API call */
        CALL_COMPLETED: "api/call/completed",

        /** Fired when API call fails */
        CALL_FAILED: "api/call/failed",
    },

    // ===== Resource Events (Extended) =====
    RESOURCE: {
        /** Fired before resource allocation */
        ALLOCATION_REQUESTED: "resource/allocation/requested",

        /** Fired after resource allocation */
        ALLOCATION_COMPLETED: "resource/allocation/completed",

        /** Fired when resource allocation is denied */
        ALLOCATION_DENIED: "resource/allocation/denied",

        /** Fired when resource limit is exceeded */
        LIMIT_EXCEEDED: "resource/limit/exceeded",

        /** Fired when resources are released */
        RELEASED: "resource/released",
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

        /** Fired when a run requires user decision */
        DECISION_REQUESTED: "run/decision/requested",

        /** Fired when an individual step begins execution */
        STEP_STARTED: "step/started",

        /** Fired when a step completes successfully */
        STEP_COMPLETED: "step/completed",

        /** Fired when a step fails */
        STEP_FAILED: "step/failed",

        /** Fired before routine execution for pre-checks */
        EXECUTION_REQUESTED: "run/execution/requested",

        /** Fired after routine execution for post-checks */
        EXECUTION_COMPLETED: "run/execution/completed",
    },

    // ===== Security Events =====
    SECURITY: {
        /** Fired when a security threat is detected */
        THREAT_DETECTED: "security/threat/detected",

        /** Fired when an emergency stop is triggered */
        EMERGENCY_STOP: "security/emergency/stop",

        /** Fired when permission check is needed */
        PERMISSION_CHECK: "security/permission/check",

        /** Fired when security audit is logged */
        AUDIT_LOGGED: "security/audit/logged",

        /** Fired when access is blocked for security reasons */
        ACCESS_BLOCKED: "security/access/blocked",

        /** Fired when security policy is violated */
        POLICY_VIOLATED: "security/policy/violated",
    },

    // ===== System Events =====
    SYSTEM: {
        /** Fired when a component encounters an error */
        ERROR: "system/error",

        /** Fired when a component state changes */
        STATE_CHANGED: "system/state/changed",
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
    | typeof EventTypes.TOOL[keyof typeof EventTypes.TOOL]
    | typeof EventTypes.DATA[keyof typeof EventTypes.DATA]
    | typeof EventTypes.API[keyof typeof EventTypes.API]
    | typeof EventTypes.RESOURCE[keyof typeof EventTypes.RESOURCE]
    | typeof EventTypes.SWARM[keyof typeof EventTypes.SWARM]
    | typeof EventTypes.RUN[keyof typeof EventTypes.RUN]
    | typeof EventTypes.SECURITY[keyof typeof EventTypes.SECURITY]
    | typeof EventTypes.SYSTEM[keyof typeof EventTypes.SYSTEM]
    | typeof EventTypes.USER[keyof typeof EventTypes.USER]
    | typeof EventTypes.ROOM[keyof typeof EventTypes.ROOM];

// Convenience aliases for backward compatibility
export const ChatEvents = EventTypes.CHAT;
export const ToolEvents = EventTypes.TOOL;
export const DataEvents = EventTypes.DATA;
export const ApiEvents = EventTypes.API;
export const ResourceEvents = EventTypes.RESOURCE;
export const SwarmEvents = EventTypes.SWARM;
export const RunEvents = EventTypes.RUN;
export const SecurityEvents = EventTypes.SECURITY;
export const SystemEvents = EventTypes.SYSTEM;
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
    // Tool events (extended)
    TOOL_EXECUTION_REQUESTED: EventTypes.TOOL.EXECUTION_REQUESTED,
    TOOL_EXECUTION_COMPLETED: EventTypes.TOOL.EXECUTION_COMPLETED,
    // Data events
    DATA_ACCESS_REQUESTED: EventTypes.DATA.ACCESS_REQUESTED,
    DATA_ACCESS_COMPLETED: EventTypes.DATA.ACCESS_COMPLETED,
    DATA_ACCESS_DENIED: EventTypes.DATA.ACCESS_DENIED,
    DATA_MODIFICATION_REQUESTED: EventTypes.DATA.MODIFICATION_REQUESTED,
    DATA_MODIFICATION_COMPLETED: EventTypes.DATA.MODIFICATION_COMPLETED,
    // API events
    API_CALL_REQUESTED: EventTypes.API.CALL_REQUESTED,
    API_CALL_COMPLETED: EventTypes.API.CALL_COMPLETED,
    API_CALL_FAILED: EventTypes.API.CALL_FAILED,
    // Resource events (extended)
    RESOURCE_ALLOCATION_REQUESTED: EventTypes.RESOURCE.ALLOCATION_REQUESTED,
    RESOURCE_ALLOCATION_COMPLETED: EventTypes.RESOURCE.ALLOCATION_COMPLETED,
    RESOURCE_ALLOCATION_DENIED: EventTypes.RESOURCE.ALLOCATION_DENIED,
    RESOURCE_LIMIT_EXCEEDED: EventTypes.RESOURCE.LIMIT_EXCEEDED,
    RESOURCE_RELEASED: EventTypes.RESOURCE.RELEASED,
    // Run events (extended)
    RUN_EXECUTION_REQUESTED: EventTypes.RUN.EXECUTION_REQUESTED,
    RUN_EXECUTION_COMPLETED: EventTypes.RUN.EXECUTION_COMPLETED,
    // Security events
    SECURITY_THREAT_DETECTED: EventTypes.SECURITY.THREAT_DETECTED,
    SECURITY_EMERGENCY_STOP: EventTypes.SECURITY.EMERGENCY_STOP,
    SECURITY_PERMISSION_CHECK: EventTypes.SECURITY.PERMISSION_CHECK,
    SECURITY_AUDIT_LOGGED: EventTypes.SECURITY.AUDIT_LOGGED,
    SECURITY_ACCESS_BLOCKED: EventTypes.SECURITY.ACCESS_BLOCKED,
    SECURITY_POLICY_VIOLATED: EventTypes.SECURITY.POLICY_VIOLATED,
    // System events
    SYSTEM_ERROR: EventTypes.SYSTEM.ERROR,
    SYSTEM_STATE_CHANGED: EventTypes.SYSTEM.STATE_CHANGED,
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
    [EventTypes.CHAT.MESSAGE_ADDED]: ChatEventData & {
        /** Array of new messages */
        messages: ChatMessage[];
    };

    /**
     * Payload for chat message update events
     * @event CHAT_MESSAGE_UPDATED
     */
    [EventTypes.CHAT.MESSAGE_UPDATED]: ChatEventData & {
        /** Array of message updates (must include id) */
        messages: (Partial<ChatMessage> & { id: string })[];
    };

    /**
     * Payload for chat message removal events
     * @event CHAT_MESSAGE_REMOVED
     */
    [EventTypes.CHAT.MESSAGE_REMOVED]: ChatEventData & {
        /** IDs of messages that were removed */
        messageIds: string[];
    };

    /**
     * Payload for chat participant change events
     * @event CHAT_PARTICIPANTS_CHANGED
     */
    [EventTypes.CHAT.PARTICIPANTS_CHANGED]: ChatEventData & {
        /** Participants who joined the chat */
        joining?: Omit<ChatParticipant, "chat">[];
        /** IDs of participants who left the chat */
        leaving?: string[];
    };

    /**
     * Payload for chat typing indicator events
     * @event CHAT_TYPING_UPDATED
     */
    [EventTypes.CHAT.TYPING_UPDATED]: ChatEventData & {
        /** User IDs who started typing */
        starting?: string[];
        /** User IDs who stopped typing */
        stopping?: string[];
    };

    /**
     * Payload for response stream chunk events
     * @event RESPONSE_STREAM_CHUNK
     */
    [EventTypes.CHAT.RESPONSE_STREAM_CHUNK]: ChatEventData & {
        /** ID of the bot sending the stream */
        botId?: string;
        /** Text chunk (not accumulated) */
        chunk: string;
    };

    /**
     * Payload for response stream completion events
     * @event RESPONSE_STREAM_END
     */
    [EventTypes.CHAT.RESPONSE_STREAM_END]: ChatEventData & {
        /** ID of the bot that sent the stream */
        botId?: string;
        /** Complete final message */
        finalMessage: string;
    };

    /**
     * Payload for response stream error events
     * @event RESPONSE_STREAM_ERROR
     */
    [EventTypes.CHAT.RESPONSE_STREAM_ERROR]: ChatEventData & {
        /** ID of the bot that encountered the error */
        botId?: string;
        /** Error details */
        error: ErrorPayload;
    };

    /**
     * Payload for reasoning stream chunk events
     * @event REASONING_STREAM_CHUNK
     */
    [EventTypes.CHAT.REASONING_STREAM_CHUNK]: ChatEventData & {
        /** ID of the bot sending reasoning */
        botId?: string;
        /** Reasoning text chunk (not accumulated) */
        chunk: string;
    };

    /**
     * Payload for reasoning stream completion events
     * @event REASONING_STREAM_END
     */
    [EventTypes.CHAT.REASONING_STREAM_END]: ChatEventData & {
        /** ID of the bot that sent reasoning */
        botId?: string;
    };

    /**
     * Payload for reasoning stream error events
     * @event REASONING_STREAM_ERROR
     */
    [EventTypes.CHAT.REASONING_STREAM_ERROR]: ChatEventData & {
        /** ID of the bot that encountered the error */
        botId?: string;
        /** Error details */
        error: ErrorPayload;
    };

    /**
     * Payload for LLM task update events
     * @event LLM_TASKS_UPDATED
     */
    [EventTypes.CHAT.LLM_TASKS_UPDATED]: ChatEventData & {
        /** Full task information for new/complete updates */
        tasks?: AITaskInfo[];
        /** Partial updates for existing tasks */
        updates?: Partial<AITaskInfo>[];
    };

    /**
     * Payload for bot status update events
     * @event BOT_STATUS_UPDATED
     */
    [EventTypes.CHAT.BOT_STATUS_UPDATED]: ChatEventData & {
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
    [EventTypes.CHAT.BOT_TYPING_UPDATED]: ChatEventData & {
        /** Bot IDs who started typing */
        starting?: string[];
        /** Bot IDs who stopped typing */
        stopping?: string[];
    };

    /**
     * Payload for bot response stream events
     * @event BOT_RESPONSE_STREAM
     */
    [EventTypes.CHAT.BOT_RESPONSE_STREAM]: ChatEventData & {
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
    [EventTypes.CHAT.BOT_MODEL_REASONING_STREAM]: ChatEventData & {
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
    [EventTypes.TOOL.CALLED]: ChatEventData & {
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
    [EventTypes.TOOL.COMPLETED]: ChatEventData & {
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
    [EventTypes.TOOL.FAILED]: ChatEventData & {
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
    [EventTypes.TOOL.APPROVAL_REQUIRED]: ChatEventData & {
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
    [EventTypes.TOOL.APPROVAL_GRANTED]: ChatEventData & {
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
    [EventTypes.TOOL.APPROVAL_REJECTED]: ChatEventData & {
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
    [EventTypes.TOOL.APPROVAL_TIMEOUT]: ChatEventData & {
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
    [EventTypes.CHAT.CANCELLATION_REQUESTED]: ChatEventData & {
        // Add any additional fields here if needed
    };

    // ===== Swarm Room event payloads =====

    /**
     * Payload for the start of a swarm
     * @event SWARM_STARTED
     */
    [EventTypes.SWARM.STARTED]: SwarmEventData & {
        /** The goal of the swarm */
        goal: string;
        /** The ID of the bot or user that started the swarm */
        initiatingUser: string;
    };

    /**
     * Payload for swarm state change events
     * @event SWARM_STATE_CHANGED
     */
    [EventTypes.SWARM.STATE_CHANGED]: SwarmEventData & {
        /** Unique identifier of the swarm */
        chatId: string;
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
    [EventTypes.SWARM.RESOURCE_UPDATED]: SwarmEventData & {
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
    [EventTypes.SWARM.CONFIG_UPDATED]: SwarmEventData & {
        /** Partial configuration object with updated fields */
        config: Partial<ChatConfigObject> & { __typename?: "ChatConfigObject" };
    };

    /**
     * Payload for swarm team update events
     * @event SWARM_TEAM_UPDATED
     */
    [EventTypes.SWARM.TEAM_UPDATED]: SwarmEventData & {
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
    [EventTypes.SWARM.GOAL_CREATED]: SwarmEventData & {
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
    [EventTypes.SWARM.GOAL_UPDATED]: SwarmEventData & {
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
    [EventTypes.SWARM.GOAL_COMPLETED]: SwarmEventData & {
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
    [EventTypes.SWARM.GOAL_FAILED]: SwarmEventData & {
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
    [EventTypes.RUN.STARTED]: RunEventData & {
        /** ID of the routine being executed */
        routineId: string;
        /** Run configuration parameters */
        inputs?: Record<string, unknown>;
        /** Estimated completion time (ms) */
        estimatedDuration?: number;
        /** Parent swarm ID */
        parentSwarmId?: string;
    };

    /**
     * Payload for run completed events
     * @event RUN_COMPLETED
     */
    [EventTypes.RUN.COMPLETED]: RunEventData & {
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
    [EventTypes.RUN.FAILED]: RunEventData & {
        /** Error message */
        error: string;
        /** Execution time before failure (ms) */
        duration?: number;
        /** Whether retry is possible */
        retryable?: boolean;
        /** Parent swarm ID */
        parentSwarmId?: string;
    };

    /**
     * Payload for run decision request events
     * @event RUN_DECISION_REQUESTED
     */
    [EventTypes.RUN.DECISION_REQUESTED]: RunEventData & {
        /** Decision data requiring user input */
        decision: DeferredDecisionData;
        /** Parent swarm ID */
        parentSwarmId?: string;
    };

    /**
     * Payload for step started events
     * @event STEP_STARTED
     */
    [EventTypes.RUN.STEP_STARTED]: RunEventData & {
        /** ID of the step */
        stepId: string;
        /** Step name/description */
        name: string;
        /** Step inputs */
        inputs?: Record<string, unknown>;
        /** Parent swarm ID */
        parentSwarmId?: string;
    };

    /**
     * Payload for step completed events
     * @event STEP_COMPLETED
     */
    [EventTypes.RUN.STEP_COMPLETED]: RunEventData & {
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
            chatId?: string;
        };
    };

    /**
     * Payload for step failed events
     * @event STEP_FAILED
     */
    [EventTypes.RUN.STEP_FAILED]: RunEventData & {
        /** ID of the step */
        stepId: string;
        /** Error message */
        error: string;
        /** Execution time before failure (ms) */
        duration?: number;
        /** Parent swarm ID */
        parentSwarmId?: string;
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

    // ===== Tool Execution event payloads =====

    /**
     * Payload for tool execution requested events
     * @event TOOL_EXECUTION_REQUESTED
     */
    [EventTypes.TOOL.EXECUTION_REQUESTED]: ChatEventData & {
        /** Name of the tool to be executed */
        toolName: string;
        /** Unique ID for this tool call */
        toolCallId: string;
        /** Arguments to be passed to the tool */
        arguments: Record<string, unknown>;
        /** ID of the bot requesting execution */
        callerBotId: string;
        /** Estimated cost in credits */
        estimatedCost?: string;
        /** Risk level assessment */
        riskLevel?: "low" | "medium" | "high" | "critical";
        /** Whether this requires approval */
        requiresApproval?: boolean;
    };

    /**
     * Payload for tool execution completed events
     * @event TOOL_EXECUTION_COMPLETED
     */
    [EventTypes.TOOL.EXECUTION_COMPLETED]: ChatEventData & {
        /** Name of the executed tool */
        toolName: string;
        /** ID of the completed tool call */
        toolCallId: string;
        /** Result returned by the tool */
        result?: unknown;
        /** Execution duration in milliseconds */
        duration: number;
        /** Credits consumed by this tool call */
        creditsUsed: string;
        /** ID of the bot that made the call */
        callerBotId: string;
        /** Whether execution was blocked */
        blocked?: boolean;
        /** IDs of agents that blocked execution */
        blockedBy?: string[];
    };

    // ===== Data Access event payloads =====

    /**
     * Payload for data access requested events
     * @event DATA_ACCESS_REQUESTED
     */
    [EventTypes.DATA.ACCESS_REQUESTED]: SwarmEventData & {
        /** Path to the data being accessed */
        path: string;
        /** Type of operation */
        operation: "read" | "write" | "delete";
        /** Data sensitivity level */
        sensitivity?: DataSensitivityType;
        /** ID of the requester */
        requesterId: string;
        /** Type of requester */
        requesterType: "user" | "bot" | "system";
        /** Additional context */
        context?: Record<string, unknown>;
    };

    /**
     * Payload for data access completed events
     * @event DATA_ACCESS_COMPLETED
     */
    [EventTypes.DATA.ACCESS_COMPLETED]: SwarmEventData & {
        /** Path to the accessed data */
        path: string;
        /** Type of operation performed */
        operation: "read" | "write" | "delete";
        /** Whether access was successful */
        success: boolean;
        /** Size of data accessed */
        dataSize?: number;
        /** Whether data was sanitized */
        sanitized?: boolean;
        /** ID of the requester */
        requesterId: string;
        /** Access duration in milliseconds */
        duration: number;
    };

    /**
     * Payload for data access denied events
     * @event DATA_ACCESS_DENIED
     */
    [EventTypes.DATA.ACCESS_DENIED]: SwarmEventData & {
        /** Path to the data */
        path: string;
        /** Type of operation attempted */
        operation: "read" | "write" | "delete";
        /** ID of the requester */
        requesterId: string;
        /** Reason for denial */
        reason: string;
        /** IDs of agents that denied access */
        deniedBy?: string[];
    };

    // ===== API Call event payloads =====

    /**
     * Payload for API call requested events
     * @event API_CALL_REQUESTED
     */
    [EventTypes.API.CALL_REQUESTED]: SwarmEventData & {
        /** API endpoint URL */
        endpoint: string;
        /** HTTP method */
        method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
        /** Request headers */
        headers?: Record<string, string>;
        /** Request body */
        body?: unknown;
        /** ID of the requester */
        requesterId: string;
        /** Estimated response time */
        estimatedDuration?: number;
        /** Risk assessment */
        riskLevel?: "low" | "medium" | "high" | "critical";
    };

    /**
     * Payload for API call completed events
     * @event API_CALL_COMPLETED
     */
    [EventTypes.API.CALL_COMPLETED]: SwarmEventData & {
        /** API endpoint URL */
        endpoint: string;
        /** HTTP method used */
        method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
        /** Response status code */
        statusCode: number;
        /** Response size in bytes */
        responseSize?: number;
        /** Call duration in milliseconds */
        duration: number;
        /** ID of the requester */
        requesterId: string;
        /** Whether response was modified */
        responseModified?: boolean;
    };

    // ===== Resource event payloads =====

    /**
     * Payload for resource allocation requested events
     * @event RESOURCE_ALLOCATION_REQUESTED
     */
    [EventTypes.RESOURCE.ALLOCATION_REQUESTED]: SwarmEventData & {
        /** Type of resource requested */
        resourceType: "credits" | "memory" | "time" | "concurrency";
        /** Amount requested */
        amount: string | number;
        /** Purpose of allocation */
        purpose: string;
        /** ID of the requester */
        requesterId: string;
        /** Priority level */
        priority?: "low" | "medium" | "high" | "critical";
        /** Estimated duration of use */
        estimatedDuration?: number;
    };

    /**
     * Payload for resource allocation completed events
     * @event RESOURCE_ALLOCATION_COMPLETED
     */
    [EventTypes.RESOURCE.ALLOCATION_COMPLETED]: SwarmEventData & {
        /** Type of resource allocated */
        resourceType: "credits" | "memory" | "time" | "concurrency";
        /** Amount allocated */
        allocated: string | number;
        /** Amount originally requested */
        requested: string | number;
        /** Allocation ID for tracking */
        allocationId: string;
        /** ID of the requester */
        requesterId: string;
        /** Expiry time for allocation */
        expiresAt?: Date;
    };

    // ===== Run Execution event payloads =====

    /**
     * Payload for run execution requested events
     * @event RUN_EXECUTION_REQUESTED
     */
    [EventTypes.RUN.EXECUTION_REQUESTED]: RunEventData & {
        /** ID of the routine to execute */
        routineId: string;
        /** Execution parameters */
        parameters?: Record<string, unknown>;
        /** ID of the requester */
        requesterId: string;
        /** Estimated resource requirements */
        estimatedResources?: {
            credits: string;
            duration: number;
            memory: number;
        };
    };

    /**
     * Payload for run execution completed events
     * @event RUN_EXECUTION_COMPLETED
     */
    [EventTypes.RUN.EXECUTION_COMPLETED]: RunEventData & {
        /** ID of the executed routine */
        routineId: string;
        /** Execution results */
        results?: Record<string, unknown>;
        /** Resources actually used */
        resourcesUsed: {
            credits: string;
            duration: number;
            memory: number;
        };
        /** Whether execution was successful */
        success: boolean;
    };

    // ===== Security event payloads =====

    /**
     * Payload for threat detected events
     * @event SECURITY_THREAT_DETECTED
     */
    [EventTypes.SECURITY.THREAT_DETECTED]: SecurityEventData & {
        /** Type of threat detected */
        threatType: "injection" | "privilege_escalation" | "data_exfiltration" | "dos" | "unauthorized_access" | "other";
        /** Threat description */
        description: string;
        /** Affected resources */
        affectedResources?: string[];
        /** Recommended actions */
        recommendations?: string[];
    };

    /**
     * Payload for emergency stop events
     * @event SECURITY_EMERGENCY_STOP
     */
    [EventTypes.SECURITY.EMERGENCY_STOP]: SecurityEventData & {
        /** Reason for emergency stop */
        reason: string;
        /** Components affected */
        affectedComponents: string[];
        /** Whether stop is reversible */
        reversible: boolean;
    };

    /**
     * Payload for permission check events
     * @event SECURITY_PERMISSION_CHECK
     */
    [EventTypes.SECURITY.PERMISSION_CHECK]: SecurityEventData & {
        /** Resource being accessed */
        resource: string;
        /** Action being attempted */
        action: string;
        /** Required permissions */
        requiredPermissions: string[];
        /** Actual permissions */
        actualPermissions?: string[];
        /** Check result */
        allowed?: boolean;
    };

    /**
     * Payload for security audit events
     * @event SECURITY_AUDIT_LOGGED
     */
    [EventTypes.SECURITY.AUDIT_LOGGED]: SecurityEventData & {
        /** Audit event type */
        auditType: string;
        /** Action that was audited */
        action: string;
        /** Outcome of the action */
        outcome: "success" | "failure" | "blocked";
        /** Additional audit details */
        details?: Record<string, unknown>;
    };

    // ===== System event payloads =====

    /**
     * Payload for system error events
     * @event SYSTEM_ERROR
     */
    [EventTypes.SYSTEM.ERROR]: {
        /** The swarm ID, if applicable */
        chatId?: string;
        /** The run ID, if applicable and not in a swarm */
        runId?: string;
        /** Component that encountered the error */
        component: string;
        /** Operation that failed */
        operation: string;
        /** Error details */
        error: {
            name: string;
            message: string;
            stack?: string;
            code?: string;
        };
        /** Context information */
        context?: Record<string, unknown>;
    };

    /**
     * Payload for system state change events
     * @event SYSTEM_STATE_CHANGED
     */
    [EventTypes.SYSTEM.STATE_CHANGED]: {
        /** The swarm ID, if applicable */
        chatId?: string;
        /** The run ID, if applicable and not in a swarm */
        runId?: string;
        /** Task or component ID */
        taskId: string;
        /** Component name */
        componentName: string;
        /** Previous state */
        previousState: string;
        /** New state */
        newState: string;
        /** Additional context */
        context?: Record<string, unknown>;
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

    // ===== Room Socket Event Payloads =====

    /**
     * Payload for joining a chat room
     * @event joinChatRoom
     */
    joinChatRoom: {
        /** ID of the chat room to join */
        chatId: string;
    };

    /**
     * Payload for leaving a chat room
     * @event leaveChatRoom
     */
    leaveChatRoom: {
        /** ID of the chat room to leave */
        chatId: string;
    };

    /**
     * Payload for joining a run room
     * @event joinRunRoom
     */
    joinRunRoom: {
        /** ID of the run room to join */
        runId: string;
    };

    /**
     * Payload for leaving a run room
     * @event leaveRunRoom
     */
    leaveRunRoom: {
        /** ID of the run room to leave */
        runId: string;
    };

    /**
     * Payload for joining a user room
     * @event joinUserRoom
     */
    joinUserRoom: {
        /** ID of the user room to join */
        userId: string;
    };

    /**
     * Payload for leaving a user room
     * @event leaveUserRoom
     */
    leaveUserRoom: {
        /** ID of the user room to leave */
        userId: string;
    };

    /**
     * Payload for requesting cancellation in a chat
     * @event requestCancellation
     */
    requestCancellation: {
        /** ID of the chat to request cancellation for */
        chatId: string;
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
    [EventTypes.TOOL.APPROVAL_REQUIRED]: SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_REQUIRED];
    [EventTypes.TOOL.APPROVAL_REJECTED]: SocketEventPayloads[typeof EventTypes.TOOL.APPROVAL_REJECTED];
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

/**
 * Type guard to check if a payload is wrapped in UnifiedEvent
 */
export function isUnifiedEvent<T = unknown>(payload: unknown): payload is UnifiedEvent<T> {
    return (
        typeof payload === "object" &&
        payload !== null &&
        "id" in payload &&
        "type" in payload &&
        "timestamp" in payload &&
        "data" in payload
    );
}

/**
 * Extract the event data from a UnifiedEvent or return the payload as-is
 */
export function extractEventData<T>(payload: T | UnifiedEvent<T>): T {
    return isUnifiedEvent(payload) ? payload.data : payload;
}

/**
 * Type for EmitSocketEvent that includes UnifiedEvent wrapper
 */
export type UnifiedSocketEvent<T extends keyof SocketEventPayloads> = UnifiedEvent<SocketEventPayloads[T]>;
