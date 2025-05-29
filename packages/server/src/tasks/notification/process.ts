import { type Job } from "bullmq";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { SocketService } from "../../sockets/io.js";
import { type NotificationCreateTask } from "../taskTypes.js";

/**
 * Worker that creates a notification record in the database and emits it via WebSocket.
 */
export async function notificationCreateProcess({ data }: Job<NotificationCreateTask>) {
    const { id, userId, category, title, description, link, imgLink } = data;
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
    if (SocketService.get().roomHasOpenConnections(userId)) {
        SocketService.get().emitSocketEvent("notification", userId, {
            category,
            description,
            id,
            imgLink,
            link,
            title,
            __typename: "Notification",
            createdAt: new Date().toISOString(),
            isRead: false,
        });
    }
} 
