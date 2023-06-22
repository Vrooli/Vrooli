import { ChatMessage, ChatMessageCreateInput } from "@local/shared";
import { Server, Socket } from "socket.io";
import { createHelper } from "../actions";
import { assertRequestFrom } from "../auth";
import { chatMessage_create } from "../endpoints";
import { logger } from "../events";
import { rateLimitSocket } from "../middleware";
import { CreateOneResult } from "../types";
import { withPrisma } from "../utils/withPrisma";

type SocketCallbackSuccess = (data: { success?: boolean; error?: string }) => void;
type SocketCallbackResponse<T> = (data: { response?: T; error?: string }) => void;

export const chatSocketHandlers = (io: Server, socket: Socket) => {
    socket.on("joinRoom", async (chatId: string, callback: SocketCallbackSuccess) => {
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
    socket.on("leaveRoom", (chatId: string, callback: SocketCallbackSuccess) => {
        socket.leave(chatId);
        callback({ success: true });
    });

    // Listen for chat message events and emit to the right room
    socket.on("message", async (message: ChatMessageCreateInput, callback: SocketCallbackResponse<CreateOneResult<ChatMessage>>) => {
        const rateLimitError = await rateLimitSocket({ maxUser: 1000, socket });
        if (rateLimitError) {
            callback({ error: rateLimitError });
            return;
        }
        const success = await withPrisma({
            process: async (prisma) => {
                // Check if user is authenticated
                const { id } = assertRequestFrom(socket, { isUser: true });
                // Create message. This will trigger the socket event automatically, 
                // so we don't need to emit it here.
                const response = await createHelper({
                    info: chatMessage_create,
                    input: message,
                    objectType: "ChatMessage",
                    prisma,
                    req: socket,
                });
                callback({ response });
            },
            trace: "0493",
            traceObject: { message },
        });
        // If failed, return error
        if (!success) {
            callback({ error: "Error sending message" });
        }
    });

    // Listen for message edit events and emit to the right room
    socket.on("editMessage", (chatId, messageId, newContent) => {
        io.to(chatId).emit("editMessage", messageId, newContent);
    });

    // Listen for message delete events and emit to the right room
    socket.on("deleteMessage", (chatId, messageId) => {
        io.to(chatId).emit("deleteMessage", messageId);
    });

    // Listen for reaction events and emit to the right room
    socket.on("addReaction", (chatId, messageId, reaction) => {
        io.to(chatId).emit("addReaction", messageId, reaction);
    });
};
