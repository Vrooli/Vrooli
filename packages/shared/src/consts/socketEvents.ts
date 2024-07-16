import { ChatMessage, ChatParticipant } from "../api/generated/graphqlTypes";
import { LlmTaskInfo } from "../llm/types";
import { JOIN_CHAT_ROOM_ERRORS, JOIN_USER_ROOM_ERRORS, LEAVE_CHAT_ROOM_ERRORS, LEAVE_USER_ROOM_ERRORS } from "./api";

export type ReservedSocketEvents = "connect" | "connect_error" | "disconnect";
export type RoomSocketEvents = "joinChatRoom" | "leaveChatRoom" | "joinUserRoom" | "leaveUserRoom";

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
    notification: {
        body: string;
        icon: string;
        link: string | undefined;
        title: string;
    };
}
export type ChatSocketEventPayloads = {
    messages: {
        added?: ChatMessage[];
        deleted?: string[];
        edited?: ChatMessage[];
    }
    responseStream: {
        /** The state of the stream */
        __type: "stream" | "end" | "error";
        /** The ID of the bot sending the message */
        botId?: string;
        /** The current text stream (not the accumulated text) */
        message: string;
    }
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
        tasks?: LlmTaskInfo[];
        /** For updating individual fields (e.g. "status") of a task */
        updates?: Partial<LlmTaskInfo>[];
    };
    joinChatRoom: { chatId: string };
    leaveChatRoom: { chatId: string };
}
export type ReservedSocketEventPayloads = {
    connect: never;
    connect_error: never;
    disconnect: never;
}
export type SocketEventPayloads = UserSocketEventPayloads & ChatSocketEventPayloads & ReservedSocketEventPayloads;

export type JoinUserRoomCallbackData = {
    success: boolean;
    error?: keyof typeof JOIN_USER_ROOM_ERRORS | string;
}

export type LeaveUserRoomCallbackData = {
    success: boolean;
    error?: keyof typeof LEAVE_USER_ROOM_ERRORS;
}

export type JoinChatRoomCallbackData = {
    success: boolean;
    error?: keyof typeof JOIN_CHAT_ROOM_ERRORS | string;
}

export type LeaveChatRoomCallbackData = {
    success: boolean;
    error?: keyof typeof LEAVE_CHAT_ROOM_ERRORS;
}

export type UserSocketEventCallbackPayloads = {
    joinUserRoom: JoinUserRoomCallbackData;
    leaveUserRoom: LeaveUserRoomCallbackData;
}

export type ChatSocketEventCallbackPayloads = {
    joinChatRoom: JoinChatRoomCallbackData;
    leaveChatRoom: LeaveChatRoomCallbackData;
}

export type SocketEventCallbackPayloads = UserSocketEventCallbackPayloads & ChatSocketEventCallbackPayloads;

export type SocketEventHandler<T extends SocketEvent> = T extends keyof SocketEventCallbackPayloads
    ? (payload: SocketEventPayloads[T], callback: (data: SocketEventCallbackPayloads[T]) => void) => void
    : (payload: SocketEventPayloads[T]) => void;


export type SocketEvent = keyof SocketEventPayloads;
