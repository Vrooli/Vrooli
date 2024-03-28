import { ReservedSocketEvents, RoomSocketEvents, SocketEvent, SocketEventCallbackData, SocketEventPayloads } from "@local/shared";
import { io } from "socket.io-client";
import { webSocketUrlBase } from "./fetchData";

export const socket = io(webSocketUrlBase, { withCredentials: true });

type EmitSocketEvent = RoomSocketEvents;
type OnSocketEvent = Exclude<SocketEvent, ReservedSocketEvents | RoomSocketEvents>;

/**
 * Emits a socket event from the client to the server.
 * 
 * @param event - The socket event to emit.
 * @param payload - The payload data to send along with the event.
 * @param callback - An optional callback function to handle the response.
 */
export const emitSocketEvent = <T extends EmitSocketEvent>(
    event: T,
    payload: SocketEventPayloads[T],
    callback?: (response: SocketEventCallbackData) => void,
) => {
    if (callback) {
        socket.emit(event, payload, callback);
    } else {
        socket.emit(event, payload);
    }
};

/**
 * Registers a socket event listener.
 * 
 * @param socket - The socket object.
 * @param event - The socket event to listen for.
 * @param handler - The event handler function.
 */
export const onSocketEvent = <T extends OnSocketEvent>(
    event: T,
    handler: (payload: SocketEventPayloads[T]) => unknown,
) => {
    socket.on(event, handler as never);

    return () => {
        socket.off(event, handler as never);
    }
};
