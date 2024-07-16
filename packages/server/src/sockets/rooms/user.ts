import { JOIN_USER_ROOM_ERRORS, LEAVE_USER_ROOM_ERRORS } from "@local/shared";
import { Socket } from "socket.io";
import { assertRequestFrom } from "../../auth/request";
import { logger } from "../../events/logger";
import { rateLimitSocket } from "../../middleware";
import { onSocketEvent } from "../events";

/** Socket room for user-specific events */
export function userSocketRoomHandlers(socket: Socket) {
    onSocketEvent(socket, "joinUserRoom", async ({ userId }, callback) => {
        const rateLimitError = await rateLimitSocket({ maxUser: 1000, socket });
        if (rateLimitError) {
            callback({ success: false, error: rateLimitError });
            return;
        }
        // Check if user is authenticated
        const { id } = assertRequestFrom(socket, { isUser: true });
        // Check if user is joining their own room
        if (userId !== id) {
            const message = JOIN_USER_ROOM_ERRORS.UserNotFoundOrUnauthorized;
            logger.error(message, { trace: "0493" });
            callback({ success: false, error: message });
            return;
        }
        // Otherwise, join the room
        socket.join(userId);
        callback({ success: true });
    });

    onSocketEvent(socket, "leaveUserRoom", async ({ userId }, callback) => {
        try {
            socket.leave(userId);
            callback({ success: true });
        } catch (error) {
            const message = LEAVE_USER_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0576", error, userId });
            callback({ success: false, error: message });
        }
    });
}
