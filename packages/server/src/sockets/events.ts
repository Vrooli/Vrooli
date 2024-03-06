import { ReservedSocketEvents, RoomSocketEvents, SocketEvent, SocketEventHandler, SocketEventPayloads } from "@local/shared";
import { Socket } from "socket.io";
import { io } from "./io";

type EmitSocketEvent = Exclude<SocketEvent, ReservedSocketEvents | RoomSocketEvents>;
type OnSocketEvent = Exclude<SocketEvent, ReservedSocketEvents>;

/**
 * Emits a socket event to all clients in a specific room.
 * 
 * @param event The custom socket event to emit.
 * @param roomId The ID of the room (e.g. chat) to emit the event to.
 * @param payload The payload data to send along with the event.
 */
export const emitSocketEvent = <T extends EmitSocketEvent>(event: T, roomId: string, payload: SocketEventPayloads[T]) => {
    io.to(roomId).emit(event, payload);
};

/**
 * Registers a socket event listener.
 * 
 * @param socket - The socket object.
 * @param event - The socket event to listen for.
 * @param handler - The event handler function.
 */
export const onSocketEvent = <T extends OnSocketEvent>(socket: Socket, event: T, handler: SocketEventHandler<T>) => {
    socket.on(event, handler as never);
};

