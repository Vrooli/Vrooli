/* c8 ignore start */
import { type AITaskInfo } from "../ai/types.js";
import { type ChatMessage, type ChatParticipant, type Notification } from "../api/types.js";
import { type DeferredDecisionData, type RunTaskInfo } from "../run/types.js";
import { type JOIN_CHAT_ROOM_ERRORS, type JOIN_RUN_ROOM_ERRORS, type JOIN_USER_ROOM_ERRORS, type LEAVE_CHAT_ROOM_ERRORS, type LEAVE_RUN_ROOM_ERRORS, type LEAVE_USER_ROOM_ERRORS } from "./api.js";

export type ReservedSocketEvents = "connect" | "connect_error" | "disconnect";
export type RoomSocketEvents = "joinChatRoom" | "leaveChatRoom" | "joinRunRoom" | "leaveRunRoom" | "joinUserRoom" | "leaveUserRoom" | "requestCancellation";

export interface StreamErrorPayload {
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

export type UserSocketEventPayloads = {
    /**
     * Updates to the number of API credits the user has
     */
    apiCredits: {
        /** Stringified BigInt */
        credits: string;
    };
    joinUserRoom: { userId: string };
    leaveUserRoom: { userId: string };
    notification: Notification;
}
export type RunSocketEventPayloads = {
    joinRunRoom: { runId: string };
    leaveRunRoom: { runId: string };
    /** Runs that can or have been performed */
    runTask: RunTaskInfo;
    /** Requests a decision from the user */
    runTaskDecisionRequest: DeferredDecisionData;
}
export type ChatSocketEventPayloads = {
    messages: {
        added?: ChatMessage[];
        updated?: (Partial<ChatMessage> & { id: string })[];
        removed?: string[];
    }
    responseStream: {
        /** The state of the stream */
        __type: "stream" | "end" | "error";
        /** The ID of the bot sending the message */
        botId?: string;
        /** The current text stream (not the accumulated text) */
        chunk?: string;             // For __type: "stream"
        /** The full constructed message, if the stream is ended */
        finalMessage?: string;      // For __type: "end"
        /** Detailed error information if the stream encounters an error */
        error?: StreamErrorPayload; // For __type: "error"
    };
    modelReasoningStream: {
        /** The state of the stream */
        __type: "stream" | "end" | "error";
        /** The ID of the bot sending the reasoning */
        botId?: string;
        /** The current reasoning text stream (not the accumulated text) */
        chunk?: string; // For __type: "stream"
        /** Detailed error information if the stream encounters an error */
        error?: StreamErrorPayload; // For __type: "error"
    };
    typing: {
        /** IDs of users who started typing */
        starting?: string[];
        /** IDs of users who stopped typing */
        stopping?: string[]
    };
    participants: {
        /** Users who joined the chat */
        joining?: Omit<ChatParticipant, "chat">[];
        /** IDs of users who left the chat */
        leaving?: string[]
    };
    /** Tasks that can or have been performed */
    llmTasks: {
        /** Provides full task information for tasks */
        tasks?: AITaskInfo[];
        /** For updating individual fields (e.g. "status") of a task */
        updates?: Partial<AITaskInfo>[];
    };
    joinChatRoom: { chatId: string };
    leaveChatRoom: { chatId: string };
    requestCancellation: { chatId: string; };
    botStatusUpdate: {
        chatId: string;
        botId: string;
        status: "thinking" | "tool_calling" | "tool_completed" | "tool_failed" | "processing_complete" | "error_internal" | "tool_pending_approval" | "tool_rejected_by_user";
        message?: string;
        toolInfo?: {
            callId: string;
            name: string;
            args?: string;
            result?: string;
            error?: string;
            pendingId?: string;
            reason?: string;
        };
        error?: StreamErrorPayload; // For status: "error_internal"
    };
    tool_approval_required: {
        pendingId: string;
        toolCallId: string;
        toolName: string;
        toolArguments: Record<string, any>;
        callerBotId: string;
        callerBotName?: string;
        approvalTimeoutAt?: number;
        estimatedCost?: string;
    };
    tool_approval_rejected: {
        pendingId: string;
        toolCallId: string;
        toolName: string;
        reason?: string;
        callerBotId: string;
    };
}
export type ReservedSocketEventPayloads = {
    connect: never;
    connect_error: never;
    disconnect: never;
}
export type SocketEventPayloads = UserSocketEventPayloads & RunSocketEventPayloads & ChatSocketEventPayloads & ReservedSocketEventPayloads;

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
