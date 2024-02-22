import { Server, Socket } from "socket.io";
import { onSocketEvent } from "../../sockets/events";

export const notificationSocketHandlers = (io: Server, socket: Socket) => {
    // Listen for notification events
    onSocketEvent(socket, "notification", async (notification) => {
        // Emit it to all connected clients
        io.emit("notification", notification); //TODO not sure if right. Shouldn't there be a room like with chats?
    });
};
