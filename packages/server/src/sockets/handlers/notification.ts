import { Server, Socket } from "socket.io";

export const notificationSocketHandlers = (io: Server, socket: Socket) => {
    // Listen for notification events
    socket.on("notification", (notification) => {
        // When a notification is received, emit it to all connected clients
        io.emit("notification", notification);
    });
};
