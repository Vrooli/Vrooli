import { SocketEvent, SocketEventPayloads } from "@local/shared";
import { io } from "./io";

export const emitEvent = <T extends SocketEvent>(event: T, chatId: string, payload: SocketEventPayloads[T]) => {
    io.to(chatId).emit(event, payload);
};

