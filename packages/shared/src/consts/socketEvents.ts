import { ChatMessage } from "../api/generated/graphqlTypes";

export type ReservedSocketEvents = "connect" | "connect_error" | "disconnect";
export type RoomSocketEvents = "joinChatRoom" | "leaveChatRoom";

export interface SocketEventPayloads {
    // Chat-related events
    messages: {
        added?: ChatMessage[];
        deleted?: string[];
        edited?: ChatMessage[];
    }
    typing: {
        /** IDs of users who started typing */
        starting?: string[];
        /** IDs of users who stopped typing */
        stopping?: string[]
    };
    participants: {
        /** IDs of users who joined the chat */
        joining?: string[];
        /** IDs of users who left the chat */
        leaving?: string[]
    };
    joinChatRoom: { chatId: string };
    leaveChatRoom: { chatId: string };
    //TODO need types for these
    llmTasks: unknown[];
    // Notifications
    notification: {
        body: string;
        icon: string;
        link: string | undefined;
        title: string;
    }
    // Types reserved internally by socket.io. These should not be used.
    connect: never;
    connect_error: never;
    disconnect: never;
}

export interface SocketEventCallbackData {
    success?: boolean;
    error?: string;
}
export interface SocketEventCallbacks {
    joinChatRoom: (response: SocketEventCallbackData) => void;
    leaveChatRoom: (response: SocketEventCallbackData) => void;
}

export type SocketEventHandler<T extends SocketEvent> = T extends keyof SocketEventCallbacks
    ? (payload: SocketEventPayloads[T], callback: SocketEventCallbacks[T]) => void
    : (payload: SocketEventPayloads[T]) => void;


export type SocketEvent = keyof SocketEventPayloads;
