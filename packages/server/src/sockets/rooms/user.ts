import { JOIN_USER_ROOM_ERRORS, LEAVE_USER_ROOM_ERRORS } from "@vrooli/shared";
import { type Socket } from "socket.io";
import { AuthTokensService } from "../../auth/auth.js";
import { RequestService } from "../../auth/request.js";
import { logger } from "../../events/logger.js";
import { SocketService } from "../io.js";

/** Socket room for user-specific events */
export function userSocketRoomHandlers(socket: Socket) {
    SocketService.get().onSocketEvent(socket, "joinUserRoom", async ({ userId }, callback) => {
        try {
            if (AuthTokensService.isAccessTokenExpired(socket.session)) {
                callback({ success: false, error: JOIN_USER_ROOM_ERRORS.SessionExpired });
                return;
            }
            const rateLimitError = await RequestService.get().rateLimitSocket({ maxUser: 1000, socket });
            if (rateLimitError) {
                callback({ success: false, error: rateLimitError });
                return;
            }
            // Check if user is authenticated
            const { id } = RequestService.assertRequestFrom(socket, { isUser: true });
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
        } catch (error) {
            const message = JOIN_USER_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0494", error, userId });
            callback({ success: false, error: message });
        }
    });

    SocketService.get().onSocketEvent(socket, "leaveUserRoom", async ({ userId }, callback) => {
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
