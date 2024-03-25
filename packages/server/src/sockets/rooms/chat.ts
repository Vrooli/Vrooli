import { Socket } from "socket.io";
import { assertRequestFrom } from "../../auth/request";
import { prismaInstance } from "../../db/instance";
import { logger } from "../../events/logger";
import { rateLimitSocket } from "../../middleware";
import { onSocketEvent } from "../../sockets/events";

/** Socket room for chat events */
export const chatSocketRoomHandlers = (socket: Socket) => {
    onSocketEvent(socket, "joinChatRoom", async ({ chatId }, callback) => {
        const rateLimitError = await rateLimitSocket({ maxUser: 1000, socket });
        if (rateLimitError) {
            callback({ error: rateLimitError });
            return;
        }
        try {
            // Check if user is authenticated
            const { id } = assertRequestFrom(socket, { isUser: true });
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
                const message = "Chat not found or unauthorized";
                logger.error(message, { trace: "0490" });
                callback({ error: message });
                return;
            }
            // Otherwise, join the room
            socket.join(chatId);
            callback({ success: true });
        } catch (error) {
            logger.error("Caught error in chatSocketRoomHandlers", { trace: "0491", error, chatId });
            callback({ error: "Error joining chat room" });
        }
    });

    onSocketEvent(socket, "leaveChatRoom", async ({ chatId }, callback) => {
        socket.leave(chatId);
        callback({ success: true });
    });
};
