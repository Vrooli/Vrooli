import { type Job } from "bullmq";
import { EventTypes } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { SocketService } from "../../sockets/io.js";
import { type NotificationCreateTask } from "../taskTypes.js";

/**
 * Worker that creates a notification record in the database and emits it via WebSocket.
 */
export async function notificationCreateProcess({ data }: Job<NotificationCreateTask>) {
    const { id, userId, category, title, description, link, imgLink, sendWebSocketEvent } = data;
    // Create the notification record
    try {
        await DbProvider.get().notification.create({
            data: {
                id: BigInt(id),
                userId: BigInt(userId),
                category,
                title,
                description,
                link,
                imgLink,
            },
        });
    } catch (err) {
        logger.error("[notificationCreate] DB create error", { error: err, notificationId: id });
    }
    // Emit via WebSocket if user is connected
    if (sendWebSocketEvent) {
        try {
            const socketService = SocketService.get();
            if (socketService.roomHasOpenConnections(userId)) {
                socketService.emitSocketEvent(EventTypes.USER.NOTIFICATION_RECEIVED, userId, {
                    userId,
                    notification: {
                        id,
                        category,
                        description,
                        imgLink,
                        link,
                        title,
                        __typename: "Notification" as const,
                        createdAt: new Date(),
                        updatedAt: new Date(),
                        isRead: false,
                        count: 1,
                        user: { id: userId },
                    },
                });
            }
        } catch (socketError) {
            logger.warn("[notificationCreate] SocketService not available or failed", { 
                error: socketError, 
                notificationId: id,
                userId, 
            });
        }
    }
} 
