/* eslint-disable no-magic-numbers */
// Using this.generateId() instead of generatePK
import { type Prisma, type PrismaClient, type notification } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface NotificationRelationConfig extends RelationConfig {
    user: { userId: bigint };
}

/**
 * Enhanced database fixture factory for Notification model
 * Provides comprehensive testing capabilities for notification systems
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different notification categories
 * - Read/unread status tracking
 * - Notification grouping (count)
 * - Link and image support
 * - Predefined test scenarios
 * - Bulk notification operations
 */
export class NotificationDbFactory extends EnhancedDatabaseFactory<
    notification,
    Prisma.notificationCreateInput,
    Prisma.notificationInclude,
    Prisma.notificationUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("Notification", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.notification;
    }

    /**
     * Get complete test fixtures for Notification model
     */
    protected getFixtures(): DbTestFixtures<Prisma.notificationCreateInput, Prisma.notificationUpdateInput> {
        return {
            minimal: {
                id: this.generateId(),
                category: "system",
                isRead: false,
                title: "System Update",
                count: 1,
                user: { connect: { id: this.generateId() } },
            },
            complete: {
                id: this.generateId(),
                category: "message",
                isRead: false,
                title: "New Message",
                description: "You have received a new message from John Doe",
                count: 1,
                link: "/messages/chat-456",
                imgLink: "https://example.com/avatars/john-doe.jpg",
                user: { connect: { id: this.generateId() } },
            },
            invalid: {
                missingRequired: {
                    // Missing id, category, title, user
                    isRead: false,
                    count: 1,
                },
                invalidTypes: {
                    id: BigInt("123456789"), // Invalid snowflake value but properly typed
                    category: 123, // Should be string
                    isRead: "no", // Should be boolean
                    title: null, // Should be string
                    description: true, // Should be string
                    count: "one", // Should be number
                    link: 456, // Should be string
                    imgLink: false, // Should be string
                    user: null, // Should be object
                },
                titleTooLong: {
                    id: this.generateId(),
                    category: "system",
                    isRead: false,
                    title: "a".repeat(129), // Exceeds max length of 128
                    count: 1,
                    user: { connect: { id: this.generateId() } },
                },
                descriptionTooLong: {
                    id: this.generateId(),
                    category: "system",
                    isRead: false,
                    title: "Notification",
                    description: "a".repeat(2049), // Exceeds max length of 2048
                    count: 1,
                    user: { connect: { id: this.generateId() } },
                },
                invalidCategory: {
                    id: this.generateId(),
                    category: "a".repeat(65), // Exceeds max length of 64
                    isRead: false,
                    title: "Invalid Category",
                    count: 1,
                    user: { connect: { id: this.generateId() } },
                },
            },
            edgeCases: {
                maxLengthTitle: {
                    id: this.generateId(),
                    category: "system",
                    isRead: false,
                    title: "a".repeat(128), // Max length
                    count: 1,
                    user: { connect: { id: this.generateId() } },
                },
                maxLengthDescription: {
                    id: this.generateId(),
                    category: "system",
                    isRead: false,
                    title: "Long Description",
                    description: "a".repeat(2048), // Max length
                    count: 1,
                    user: { connect: { id: this.generateId() } },
                },
                unicodeNotification: {
                    id: this.generateId(),
                    category: "social",
                    isRead: false,
                    title: "👋 New follower!",
                    description: "ユーザー様 started following you 🎉",
                    count: 1,
                    user: { connect: { id: this.generateId() } },
                },
                groupedNotification: {
                    id: this.generateId(),
                    category: "likes",
                    isRead: false,
                    title: "Your post received likes",
                    description: "25 people liked your recent post",
                    count: 25,
                    link: "/posts/post-789",
                    user: { connect: { id: this.generateId() } },
                },
                readNotification: {
                    id: this.generateId(),
                    category: "update",
                    isRead: true,
                    title: "App Update Available",
                    description: "Version 2.0 is now available with new features",
                    count: 1,
                    link: "/settings/updates",
                    user: { connect: { id: this.generateId() } },
                },
                urgentNotification: {
                    id: this.generateId(),
                    category: "security",
                    isRead: false,
                    title: "⚠️ Security Alert",
                    description: "Unusual login attempt detected from new location",
                    count: 1,
                    link: "/settings/security",
                    imgLink: "https://example.com/icons/warning.png",
                    user: { connect: { id: this.generateId() } },
                },
                noDescriptionNotification: {
                    id: this.generateId(),
                    category: "reminder",
                    isRead: false,
                    title: "Meeting in 15 minutes",
                    description: null,
                    count: 1,
                    link: "/meetings/meeting-123",
                    user: { connect: { id: this.generateId() } },
                },
            },
            updates: {
                minimal: {
                    isRead: true,
                },
                complete: {
                    isRead: true,
                    count: { increment: 5 },
                    description: "Updated notification description",
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.notificationCreateInput>): Prisma.notificationCreateInput {
        return {
            id: this.generateId(),
            category: "system",
            isRead: false,
            title: "Notification",
            count: 1,
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.notificationCreateInput>): Prisma.notificationCreateInput {
        return {
            id: this.generateId(),
            category: "message",
            isRead: false,
            title: "New Activity",
            description: "You have new activity on your account",
            count: 1,
            link: "/activity",
            imgLink: "https://example.com/icons/activity.png",
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            messageNotification: {
                name: "messageNotification",
                description: "New message notification",
                config: {
                    overrides: {
                        category: "message",
                        title: "New message from Alice",
                        description: "Hey! How are you doing?",
                        link: "/messages/chat-alice",
                        imgLink: "https://example.com/avatars/alice.jpg",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            mentionNotification: {
                name: "mentionNotification",
                description: "User mentioned notification",
                config: {
                    overrides: {
                        category: "mention",
                        title: "@johndoe mentioned you",
                        description: "You were mentioned in a comment: 'Great work @you!'",
                        link: "/posts/post-123#comment-456",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            likeNotification: {
                name: "likeNotification",
                description: "Content liked notification",
                config: {
                    overrides: {
                        category: "like",
                        title: "Your post was liked",
                        description: "Sarah and 5 others liked your post",
                        count: 6,
                        link: "/posts/post-789",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            followNotification: {
                name: "followNotification",
                description: "New follower notification",
                config: {
                    overrides: {
                        category: "follow",
                        title: "New follower",
                        description: "TechGuru started following you",
                        link: "/profile/techguru",
                        imgLink: "https://example.com/avatars/techguru.jpg",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            systemNotification: {
                name: "systemNotification",
                description: "System update notification",
                config: {
                    overrides: {
                        category: "system",
                        title: "System Maintenance",
                        description: "Scheduled maintenance on Sunday 2-4 AM UTC",
                        link: "/announcements/maintenance",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            securityAlert: {
                name: "securityAlert",
                description: "Security alert notification",
                config: {
                    overrides: {
                        category: "security",
                        title: "New login detected",
                        description: "Login from Chrome on Windows in New York",
                        link: "/settings/security/sessions",
                        imgLink: "https://example.com/icons/security.png",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            achievementNotification: {
                name: "achievementNotification",
                description: "Achievement unlocked notification",
                config: {
                    overrides: {
                        category: "achievement",
                        title: "Achievement Unlocked! 🏆",
                        description: "You've earned the 'Active Contributor' badge",
                        link: "/profile/achievements",
                        imgLink: "https://example.com/badges/contributor.png",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
            reminderNotification: {
                name: "reminderNotification",
                description: "Reminder notification",
                config: {
                    overrides: {
                        category: "reminder",
                        title: "Meeting starting soon",
                        description: "Team standup in 10 minutes",
                        link: "/meetings/standup-daily",
                    },
                    user: { user: { connect: { id: this.generateId() } } },
                },
            },
        };
    }

    protected getDefaultInclude(): Prisma.notificationInclude {
        return {
            user: {
                select: {
                    id: true,
                    publicId: true,
                    name: true,
                    handle: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.notificationCreateInput,
        config: NotificationRelationConfig,
        tx: any,
    ): Promise<Prisma.notificationCreateInput> {
        const data = { ...baseData };

        // Handle user connection (required)
        if (config.user) {
            data.user = {
                connect: { id: config.user.userId },
            };
        } else {
            throw new Error("Notification requires a user connection");
        }

        return data;
    }

    /**
     * Create a notification
     */
    async createNotification(
        userId: bigint,
        category: string,
        title: string,
        options?: {
            description?: string;
            link?: string;
            imgLink?: string;
            count?: number;
            isRead?: boolean;
        },
    ): Promise<notification> {
        return await this.createWithRelations({
            overrides: {
                category,
                title,
                description: options?.description,
                link: options?.link,
                imgLink: options?.imgLink,
                count: options?.count ?? 1,
                isRead: options?.isRead ?? false,
            },
            user: { userId },
        });
    }

    /**
     * Create a grouped notification
     */
    async createGroupedNotification(
        userId: bigint,
        category: string,
        title: string,
        count: number,
        description?: string,
    ): Promise<notification> {
        return await this.createNotification(userId, category, title, {
            description,
            count,
        });
    }

    /**
     * Mark notification as read
     */
    async markAsRead(notificationId: bigint): Promise<notification> {
        return await this.prisma.notification.update({
            where: { id: notificationId },
            data: { isRead: true },
            include: this.getDefaultInclude(),
        });
    }

    /**
     * Mark multiple notifications as read
     */
    async markMultipleAsRead(notificationIds: bigint[]): Promise<void> {
        await this.prisma.notification.updateMany({
            where: {
                id: { in: notificationIds },
            },
            data: { isRead: true },
        });
    }

    /**
     * Mark all user notifications as read
     */
    async markAllAsRead(userId: bigint): Promise<void> {
        await this.prisma.notification.updateMany({
            where: {
                userId,
                isRead: false,
            },
            data: { isRead: true },
        });
    }

    /**
     * Increment notification count
     */
    async incrementCount(notificationId: bigint, increment = 1): Promise<notification> {
        return await this.prisma.notification.update({
            where: { id: notificationId },
            data: { count: { increment } },
            include: this.getDefaultInclude(),
        });
    }

    protected async checkModelConstraints(record: notification): Promise<string[]> {
        const violations: string[] = [];

        // Check title length
        if (record.title.length > 128) {
            violations.push("Title exceeds maximum length of 128 characters");
        }

        // Check description length
        if (record.description && record.description.length > 2048) {
            violations.push("Description exceeds maximum length of 2048 characters");
        }

        // Check category length
        if (record.category.length > 64) {
            violations.push("Category exceeds maximum length of 64 characters");
        }

        // Check count is positive
        if (record.count < 1) {
            violations.push("Count must be at least 1");
        }

        // Check link format
        if (record.link && record.link.length > 2048) {
            violations.push("Link exceeds maximum length of 2048 characters");
        }

        // Check image link format
        if (record.imgLink && record.imgLink.length > 2048) {
            violations.push("Image link exceeds maximum length of 2048 characters");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            // Notification has no dependent records
        };
    }

    protected async deleteRelatedRecords(
        record: notification,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // Notification has no dependent records to delete
    }

    /**
     * Get unread notifications for user
     */
    async getUnreadNotifications(userId: bigint, limit?: number): Promise<notification[]> {
        return await this.prisma.notification.findMany({
            where: {
                userId,
                isRead: false,
            },
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
            take: limit,
        });
    }

    /**
     * Get notifications by category
     */
    async getNotificationsByCategory(
        userId: bigint,
        category: string,
        options?: { isRead?: boolean; limit?: number },
    ): Promise<notification[]> {
        const where: Prisma.notificationWhereInput = {
            userId,
            category,
        };

        if (options?.isRead !== undefined) {
            where.isRead = options.isRead;
        }

        return await this.prisma.notification.findMany({
            where,
            include: this.getDefaultInclude(),
            orderBy: { createdAt: "desc" },
            take: options?.limit,
        });
    }

    /**
     * Create bulk notifications for multiple users
     */
    async createBulkNotifications(
        userIds: bigint[],
        category: string,
        title: string,
        description?: string,
    ): Promise<notification[]> {
        const notifications = await Promise.all(
            userIds.map(userId =>
                this.createNotification(userId, category, title, { description }),
            ),
        );
        return notifications;
    }

    /**
     * Delete old read notifications
     */
    async deleteOldReadNotifications(userId: bigint, daysOld = 30): Promise<number> {
        const cutoffDate = new Date();
        cutoffDate.setDate(cutoffDate.getDate() - daysOld);

        const result = await this.prisma.notification.deleteMany({
            where: {
                userId,
                isRead: true,
                updatedAt: { lt: cutoffDate },
            },
        });

        return result.count;
    }

    /**
     * Create notification set for testing
     */
    async createTestNotificationSet(userId: bigint): Promise<{
        unread: notification[];
        read: notification[];
        grouped: notification;
        urgent: notification;
    }> {
        const unread = await Promise.all([
            this.createNotification(userId, "message", "New message", {
                description: "You have a new message",
                link: "/messages",
            }),
            this.createNotification(userId, "like", "Post liked", {
                description: "Someone liked your post",
                link: "/posts/123",
            }),
        ]);

        const readNotif = await this.createNotification(userId, "system", "Update complete", {
            description: "System update completed successfully",
        });
        const read = [await this.markAsRead(readNotif.id)];

        const grouped = await this.createGroupedNotification(
            userId,
            "follow",
            "New followers",
            15,
            "15 people started following you",
        );

        const urgent = await this.createNotification(userId, "security", "⚠️ Security Alert", {
            description: "Suspicious activity detected",
            link: "/security",
            imgLink: "https://example.com/warning.png",
        });

        return {
            unread,
            read,
            grouped,
            urgent,
        };
    }

    /**
     * Get notification statistics for user
     */
    async getNotificationStats(userId: bigint): Promise<{
        total: number;
        unread: number;
        byCategory: Record<string, number>;
    }> {
        const notifications = await this.prisma.notification.findMany({
            where: { userId },
            select: {
                category: true,
                isRead: true,
            },
        });

        const stats = {
            total: notifications.length,
            unread: notifications.filter(n => !n.isRead).length,
            byCategory: {} as Record<string, number>,
        };

        notifications.forEach(notif => {
            stats.byCategory[notif.category] = (stats.byCategory[notif.category] || 0) + 1;
        });

        return stats;
    }
}

// Export factory creator function
export const createNotificationDbFactory = (prisma: PrismaClient) =>
    new NotificationDbFactory(prisma);

// Export the class for type usage
export { NotificationDbFactory as NotificationDbFactoryClass };
