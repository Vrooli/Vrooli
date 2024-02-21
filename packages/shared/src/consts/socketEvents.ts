import { ChatMessage } from "../api/generated/graphqlTypes";

export interface SocketEventPayloads {
    addMessage: ChatMessage;
    deleteMessage: { messageId: string };
    editMessage: ChatMessage;
    typing: { starting?: string[]; stopping?: string[] };
    // Types reserved internally by socket.io
    connect: never;
    connect_error: never;
    disconnect: never;
}

export type SocketEvent = keyof SocketEventPayloads;
