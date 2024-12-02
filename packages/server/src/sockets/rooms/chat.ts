import { JOIN_CHAT_ROOM_ERRORS, LEAVE_CHAT_ROOM_ERRORS } from "@local/shared";
import { Socket } from "socket.io";
import { AuthTokensService } from "../../auth/auth";
import { RequestService } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events/logger";
import { onSocketEvent } from "../../sockets/events";

/** Socket room for chat events */
export function chatSocketRoomHandlers(socket: Socket) {
    onSocketEvent(socket, "joinChatRoom", async ({ chatId }, callback) => {
        try {
            if (AuthTokensService.isAccessTokenExpired(socket.session)) {
                callback({ success: false, error: JOIN_CHAT_ROOM_ERRORS.SessionExpired });
            }
            const rateLimitError = await RequestService.get().rateLimitSocket({ maxUser: 1000, socket });
            if (rateLimitError) {
                callback({ success: false, error: rateLimitError });
                return;
            }
            // Check if user is authenticated
            const { id } = RequestService.assertRequestFrom(socket, { isUser: true });
            // Find chat only if permitted
            const chat = await prismaInstance.chat.findMany({
                where: {
                    id: chatId,
                    OR: [
                        { openToAnyoneWithInvite: true },
                        { participants: { some: { user: { id } } } },
                        { creator: { id } },
                    ],
                },
            });
            // If not found, return error
            if (!chat || chat.length === 0) {
                const message = JOIN_CHAT_ROOM_ERRORS.ChatNotFoundOrUnauthorized;
                logger.error(message, { trace: "0490" });
                callback({ success: false, error: message });
                return;
            }
            // Otherwise, join the room
            socket.join(chatId);
            callback({ success: true });
        } catch (error) {
            const message = JOIN_CHAT_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0491", error, chatId });
            callback({ success: false, error: message });
        }
    });

    onSocketEvent(socket, "leaveChatRoom", async ({ chatId }, callback) => {
        try {
            socket.leave(chatId);
            callback({ success: true });
        } catch (error) {
            const message = LEAVE_CHAT_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0578", error, chatId });
            callback({ success: false, error: message });
        }
    });
}
