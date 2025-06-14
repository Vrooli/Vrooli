import { JOIN_CHAT_ROOM_ERRORS, LEAVE_CHAT_ROOM_ERRORS } from "@vrooli/shared";
import { type Socket } from "socket.io";
import { AuthTokensService } from "../../auth/auth.js";
import { RequestService } from "../../auth/request.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { completionService } from "../../services/conversation/responseEngine.js";
import { SocketService } from "../io.js";

/** Socket room for chat events */
export function chatSocketRoomHandlers(socket: Socket) {
    SocketService.get().onSocketEvent(socket, "joinChatRoom", async ({ chatId }, callback) => {
        try {
            if (AuthTokensService.isAccessTokenExpired(socket.session)) {
                callback({ success: false, error: JOIN_CHAT_ROOM_ERRORS.SessionExpired });
                return;
            }
            const rateLimitError = await RequestService.get().rateLimitSocket({ maxUser: 1000, socket });
            if (rateLimitError) {
                callback({ success: false, error: rateLimitError });
                return;
            }
            // Check if user is authenticated
            const { id } = RequestService.assertRequestFrom(socket, { isUser: true });
            // Find chat only if permitted
            const chat = await DbProvider.get().chat.findMany({
                where: {
                    id: BigInt(chatId),
                    OR: [
                        { openToAnyoneWithInvite: true },
                        { participants: { some: { user: { id: BigInt(id) } } } },
                        { creator: { id: BigInt(id) } },
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

    SocketService.get().onSocketEvent(socket, "leaveChatRoom", async ({ chatId }, callback) => {
        try {
            socket.leave(chatId);
            callback({ success: true });
        } catch (error) {
            const message = LEAVE_CHAT_ROOM_ERRORS.ErrorUnknown;
            logger.error(message, { trace: "0578", error, chatId });
            callback({ success: false, error: message });
        }
    });

    SocketService.get().onSocketEvent(socket, "requestCancellation", async (data: { chatId: string }, callback) => {
        const { chatId } = data;
        try {
            if (AuthTokensService.isAccessTokenExpired(socket.session)) {
                callback({ success: false, error: "SessionExpired" });
                return;
            }

            const { id: userId } = RequestService.assertRequestFrom(socket, { isUser: true });

            if (!chatId) {
                logger.warn("Invalid cancellation request: chatId missing", { socketId: socket.id, userId });
                callback({ success: false, error: "Chat ID is required for cancellation." });
                return;
            }

            // Validate user's relationship to the chat
            const chat = await DbProvider.get().chat.findFirst({
                where: {
                    id: BigInt(chatId),
                    OR: [
                        { participants: { some: { userId: BigInt(userId) } } },
                        { creatorId: BigInt(userId) },
                        // Add other conditions if necessary, e.g. openToAnyoneWithInvite, team admin, etc.
                        // For cancellation, being a participant or creator should generally be sufficient.
                    ],
                },
                select: { id: true }, // We only need to know if it exists and user is authorized
            });

            if (!chat) {
                logger.warn("Unauthorized or non-existent chat cancellation request", { chatId, userId, socketId: socket.id });
                callback({ success: false, error: "ChatNotFoundOrUnauthorized" });
                return;
            }

            logger.info(`Received valid cancellation request for chatId: ${chatId} from user: ${userId}, socket: ${socket.id}`);
            completionService.requestCancellation(chatId);
            callback({ success: true, message: "Cancellation request processed." });

        } catch (error: unknown) {
            let errorMessage = "Error processing cancellation request.";
            // Check if the error is from assertRequestFrom (e.g. not authenticated)
            if (error instanceof Error && error.message.includes("Request is not from an authenticated user")) {
                errorMessage = "UserNotAuthenticated";
                logger.warn("Unauthenticated user tried to cancel response", { chatId, socketId: socket.id, error: error.message });
            } else {
                logger.error(errorMessage, { trace: "chatSocketRoomHandlers-requestCancellation", error, chatId, socketId: socket.id });
            }
            callback({ success: false, error: errorMessage });
        }
    });
}
