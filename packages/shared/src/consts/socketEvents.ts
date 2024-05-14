import { ChatMessage, ChatParticipant } from "../api/generated/graphqlTypes";
import { LlmTaskInfo } from "../llm/types";

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
        tasks: LlmTaskInfo[];
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

export interface SocketEventCallbackData {
    success?: boolean;
    error?: string;
}
export interface SocketEventCallbacks {
    joinChatRoom: (response: SocketEventCallbackData) => void;
    leaveChatRoom: (response: SocketEventCallbackData) => void;
    joinUserRoom: (response: SocketEventCallbackData) => void;
    leaveUserRoom: (response: SocketEventCallbackData) => void;
}

export type SocketEventHandler<T extends SocketEvent> = T extends keyof SocketEventCallbacks
    ? (payload: SocketEventPayloads[T], callback: SocketEventCallbacks[T]) => void
    : (payload: SocketEventPayloads[T]) => void;


export type SocketEvent = keyof SocketEventPayloads;
