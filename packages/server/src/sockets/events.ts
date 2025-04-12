import { ReservedSocketEvents, RoomSocketEvents, SocketEvent, SocketEventHandler, SocketEventPayloads } from "@local/shared";
import { Socket } from "socket.io";
import { AuthTokensService } from "../auth/auth.js";
import { SessionService } from "../auth/session.js";
import { SessionData } from "../types.js";
import { io, sessionSockets, userSockets } from "./io.js";

type EmitSocketEvent = Exclude<SocketEvent, ReservedSocketEvents | RoomSocketEvents>;
type OnSocketEvent = Exclude<SocketEvent, ReservedSocketEvents>;

/**
 * Emits a socket event to all clients in a specific room.
 * 
 * @param event The custom socket event to emit.
 * @param roomId The ID of the room (e.g. chat) to emit the event to.
 * @param payload The payload data to send along with the event.
 */
export function emitSocketEvent<T extends EmitSocketEvent>(event: T, roomId: string, payload: SocketEventPayloads[T]) {
    io.in(roomId).fetchSockets().then((sockets) => {
        for (const socket of sockets) {
            // Check if socket is associated with a session
            const session = (socket as { session?: SessionData }).session;
            if (!session) {
                socket.emit(event, payload);
                return;
            }
            // Check if the session is expired
            const isExpired = AuthTokensService.isAccessTokenExpired(session);
            // If so, close the socket
            if (isExpired) {
                socket.disconnect();
                // Also remove the socket from session socket maps
                const user = SessionService.getUser({ session });
                const sessionId = user?.session?.id;
                if (sessionId) {
                    closeSessionSockets(sessionId);
                }
                return;
            }
            // Otherwise, emit the event to the socket
            socket.emit(event, payload);
        }
    });
}

/**
 * Registers a socket event listener.
 * 
 * @param socket - The socket object.
 * @param event - The socket event to listen for.
 * @param handler - The event handler function.
 */
export function onSocketEvent<T extends OnSocketEvent>(socket: Socket, event: T, handler: SocketEventHandler<T>) {
    socket.on(event, handler as never);
}

/**
 * Checks if a specific room has open connections.
 *
 * @param roomId - The ID of the room to check.
 * @returns true if the room has one or more open connections, false otherwise.
 */
export function roomHasOpenConnections(roomId: string): boolean {
    const room = io.sockets.adapter.rooms.get(roomId);
    return room ? room.size > 0 : false;
}


/**
 * Closes all socket connections for a user. 
 * Useful when the user logs out or is banned.
 */
export function closeUserSockets(userId: string) {
    const socketsIds = userSockets.get(userId);
    if (socketsIds) {
        for (const socketId of socketsIds) {
            const socket = io.sockets.sockets.get(socketId);
            if (socket && socket.connected) {
                socket.disconnect();
            }
        }
        userSockets.delete(userId);
    }
}

/**
 * Closes all socket connections for a session.
 * Useful when the user logs out or revokes open sessions.
 */
export function closeSessionSockets(sessionId: string) {
    const socketsIds = sessionSockets.get(sessionId);
    if (socketsIds) {
        for (const socketId of socketsIds) {
            const socket = io.sockets.sockets.get(socketId);
            if (socket && socket.connected) {
                socket.disconnect();
            }
        }
        sessionSockets.delete(sessionId);
    }
}
