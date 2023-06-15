import { ChatMessageCreateInput } from "@local/shared";
import { Server, Socket } from "socket.io";
import { assertRequestFrom } from "../auth";
import { logger } from "../events";
import { withPrisma } from "../utils/withPrisma";

type SocketCallback = (data: { success?: boolean; error?: string }) => void;

export const chatSocketHandlers = (io: Server, socket: Socket) => {
    socket.on("joinRoom", async (chatId: string, callback: SocketCallback) => {
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
    socket.on("leaveRoom", (chatId: string, callback: SocketCallback) => {
        socket.leave(chatId);
        callback({ success: true });
    });

    // Listen for chat message events and emit to the right room
    socket.on("message", (message: ChatMessageCreateInput) => {
        // io.to(chatId).emit("message", message);
    });

    // Listen for message edit events and emit to the right room
    socket.on("editMessage", (chatId, messageId, newContent) => {
        io.to(chatId).emit("editMessage", messageId, newContent);
    });

    // Listen for reaction events and emit to the right room
    socket.on("addReaction", (chatId, messageId, reaction) => {
        io.to(chatId).emit("addReaction", messageId, reaction);
    });
};
