import { Server, Socket } from "socket.io";
import { assertRequestFrom } from "../../auth/request";
import { logger } from "../../events/logger";
import { rateLimitSocket } from "../../middleware";
import { onSocketEvent } from "../../sockets/events";
import { withPrisma } from "../../utils/withPrisma";

/**
 * Handles socket events for chats, which are not handled by endpoints
 */
export const chatSocketHandlers = (io: Server, socket: Socket) => {
    onSocketEvent(socket, "joinChatRoom", async ({ chatId }, callback) => {
        const rateLimitError = await rateLimitSocket({ maxUser: 250, socket });
        if (rateLimitError) {
            callback({ error: rateLimitError });
            return;
        }
        const success = await withPrisma({
            process: async (prisma) => {
                // Check if user is authenticated
                const { id } = assertRequestFrom(socket, { isUser: true });
                // Find chat only if permitted
                const chat = await prisma.chat.findMany({
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
            },
            trace: "0491",
            traceObject: { chatId },
        });
        // If failed, return error
        if (!success) {
            callback({ error: "Error joining chat" });
        }
    });

    // Leave a specific room
    onSocketEvent(socket, "leaveChatRoom", async ({ chatId }, callback) => {
        socket.leave(chatId);
        callback({ success: true });
    });
};
