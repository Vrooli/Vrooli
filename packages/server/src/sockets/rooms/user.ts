import { Socket } from "socket.io";
import { assertRequestFrom } from "../../auth/request";
import { logger } from "../../events/logger";
import { rateLimitSocket } from "../../middleware";
import { onSocketEvent } from "../events";

/** Socket room for user-specific events */
export const userSocketRoomHandlers = (socket: Socket) => {
    onSocketEvent(socket, "joinUserRoom", async ({ userId }, callback) => {
        const rateLimitError = await rateLimitSocket({ maxUser: 1000, socket });
        if (rateLimitError) {
            callback({ error: rateLimitError });
            return;
        }
        // Check if user is authenticated
        const { id } = assertRequestFrom(socket, { isUser: true });
        // Check if user is joining their own room
        if (userId !== id) {
            const message = "Unauthorized";
            logger.error(message, { trace: "0493" });
            callback({ error: message });
            return;
        }
        // Otherwise, join the room
        socket.join(userId);
        callback({ success: true });
    });

    onSocketEvent(socket, "leaveUserRoom", async ({ userId }, callback) => {
        socket.leave(userId);
        callback({ success: true });
    });
};