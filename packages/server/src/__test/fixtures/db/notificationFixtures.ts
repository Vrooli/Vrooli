/* eslint-disable no-magic-numbers */
import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for Notification model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing - using lazy initialization to avoid module-level generatePK() calls
let _notificationDbIds: Record<string, bigint> | null = null;
export function getNotificationDbIds() {
    if (!_notificationDbIds) {
        _notificationDbIds = {
            notification1: generatePK(),
            notification2: generatePK(),
            notification3: generatePK(),
            subscription1: generatePK(),
            subscription2: generatePK(),
            subscription3: generatePK(),
        };
    }
    return _notificationDbIds;
}

/**
 * Enhanced test fixtures for Notification model following standard structure
 */
export const notificationDbFixtures: DbTestFixtures<Prisma.notificationCreateInput> = {
    minimal: {
        id: generatePK(),
        category: "Update",
        user: { connect: { id: "user_placeholder_id" } },
        isRead: false,
    },
    complete: {
        id: generatePK(),
        category: "TeamInvite",
        user: { connect: { id: "user_placeholder_id" } },
        isRead: false,
        title: "Team Invitation",
        description: "You have been invited to join the Development Team",
        team: { connect: { id: "team_placeholder_id" } },
        fromUser: { connect: { id: "inviter_user_id" } },
        createdAt: new Date(),
    },
    invalid: {
        missingRequired: {
            // Missing required category and user
            isRead: false,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            category: 123, // Should be string
            isRead: "yes", // Should be boolean
            title: 456, // Should be string
            description: true, // Should be string
        },
        invalidUserConnection: {
            id: generatePK(),
            category: "Update",
            user: { connect: { id: generatePK() } },
            isRead: false,
        },
        invalidObjectConnection: {
            id: generatePK(),
            category: "CommentReply",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: false,
            comment: { connect: { id: generatePK() } },
        },
    },
    edgeCases: {
        oldNotification: {
            id: generatePK(),
            category: "Update",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: true,
            title: "Old Notification",
            description: "This notification was created a year ago",
            createdAt: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000), // 1 year ago
        },
        longContentNotification: {
            id: generatePK(),
            category: "SystemAlert",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: false,
            title: "System Maintenance Alert - Critical Updates Required",
            description: "A".repeat(1000), // Very long description
        },
        multipleObjectNotification: {
            id: generatePK(),
            category: "ProjectUpdate",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: false,
            title: "Project and Team Update",
            description: "Multiple updates across different areas",
            team: { connect: { id: "team_placeholder_id" } },
            fromUser: { connect: { id: "sender_user_id" } },
        },
        systemNotification: {
            id: generatePK(),
            category: "SystemMaintenance",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: false,
            title: "Scheduled Maintenance",
            description: "System will be offline for maintenance from 2-4 AM UTC",
            // No fromUser - system generated
        },
        reactionNotification: {
            id: generatePK(),
            category: "Reaction",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: false,
            title: "New Reaction",
            description: "Someone liked your comment",
            reaction: { connect: { id: "reaction_placeholder_id" } },
            fromUser: { connect: { id: "reactor_user_id" } },
        },
    },
};

/**
 * Enhanced factory for creating notification database fixtures
 */
export class NotificationDbFactory extends EnhancedDbFactory<Prisma.notificationCreateInput> {

    /**
     * Get the test fixtures for Notification model
     */
    protected getFixtures(): DbTestFixtures<Prisma.notificationCreateInput> {
        return notificationDbFixtures;
    }

    /**
     * Get Notification-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getNotificationDbIds().notification1, // Duplicate ID
                    category: "Update",
                    user: { connect: { id: "user_placeholder_id" } },
                    isRead: false,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    category: "Update",
                    user: { connect: { id: generatePK() } },
                    isRead: false,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    category: "", // Empty category violates constraint
                    user: { connect: { id: "user_placeholder_id" } },
                    isRead: false,
                },
            },
            validation: {
                requiredFieldMissing: notificationDbFixtures.invalid.missingRequired,
                invalidDataType: notificationDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    category: "A".repeat(256), // Category too long
                    user: { connect: { id: "user_placeholder_id" } },
                    isRead: false,
                },
            },
            businessLogic: {
                notificationToSelf: {
                    id: generatePK(),
                    category: "TeamInvite",
                    user: { connect: { id: "user_placeholder_id" } },
                    fromUser: { connect: { id: "user_placeholder_id" } }, // Same user
                    isRead: false,
                },
                orphanedObjectNotification: {
                    id: generatePK(),
                    category: "CommentReply",
                    user: { connect: { id: "user_placeholder_id" } },
                    isRead: false,
                    comment: { connect: { id: generatePK() } },
                },
            },
        };
    }

    /**
     * Add object association to a notification fixture
     */
    protected addObjectAssociation(data: Prisma.notificationCreateInput, objectId: string, objectType: string): Prisma.notificationCreateInput {
        const connections: Record<string, any> = {
            Comment: { comment: { connect: { id: objectId } } },
            Issue: { issue: { connect: { id: objectId } } },
            Question: { question: { connect: { id: objectId } } },
            Quiz: { quiz: { connect: { id: objectId } } },
            Reaction: { reaction: { connect: { id: objectId } } },
            Report: { report: { connect: { id: objectId } } },
            Team: { team: { connect: { id: objectId } } },
            User: { fromUser: { connect: { id: objectId } } },
        };

        return {
            ...data,
            ...(connections[objectType] || {}),
        };
    }

    /**
     * Notification-specific validation
     */
    protected validateSpecific(data: Prisma.notificationCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Notification
        if (!data.category) errors.push("Notification category is required");
        if (!data.user) errors.push("Notification must be associated with a user");
        if (data.isRead === undefined) errors.push("isRead flag is required");

        // Check business logic
        if (data.fromUser && data.user &&
            typeof data.fromUser === "object" && "connect" in data.fromUser &&
            typeof data.user === "object" && "connect" in data.user &&
            data.fromUser.connect?.id === data.user.connect?.id) {
            warnings.push("Notification sender and recipient are the same user");
        }

        // Check category-specific requirements
        if (data.category?.includes("Team") && !data.team) {
            warnings.push("Team-related notifications should reference a team");
        }

        if (data.category?.includes("Comment") && !data.comment) {
            warnings.push("Comment-related notifications should reference a comment");
        }

        if (data.category?.includes("Reaction") && !data.reaction) {
            warnings.push("Reaction notifications should reference a reaction");
        }

        // Check content length
        if (data.title && data.title.length > 255) {
            errors.push("Notification title is too long (max 255 characters)");
        }

        if (data.description && data.description.length > 2000) {
            errors.push("Notification description is too long (max 2000 characters)");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.notificationCreateInput>,
    ): Prisma.notificationCreateInput {
        const factory = new NotificationDbFactory();
        return factory.createMinimal({
            category,
            user: { connect: { id: userId } },
            ...overrides,
        });
    }

    static createWithObject(
        userId: string,
        category: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.notificationCreateInput>,
    ): Prisma.notificationCreateInput {
        const factory = new NotificationDbFactory();
        const data = factory.createMinimal({
            category,
            user: { connect: { id: userId } },
            ...overrides,
        });
        return factory.addObjectAssociation(data, objectId, objectType);
    }

    static createRead(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.notificationCreateInput>,
    ): Prisma.notificationCreateInput {
        const factory = new NotificationDbFactory();
        return factory.createMinimal({
            category,
            user: { connect: { id: userId } },
            isRead: true,
            ...overrides,
        });
    }
}

/**
 * Enhanced test fixtures for NotificationSubscription model
 */
export const notificationSubscriptionDbFixtures: DbTestFixtures<Prisma.notification_subscriptionCreateInput> = {
    minimal: {
        id: generatePK(),
        subscriber: { connect: { id: "user_placeholder_id" } },
        silent: false,
    },
    complete: {
        id: generatePK(),
        subscriber: { connect: { id: "user_placeholder_id" } },
        silent: false,
        context: "Test subscription context",
        team: { connect: { id: "team_placeholder_id" } },
    },
    invalid: {
        missingRequired: {
            // Missing required subscriber
            silent: false,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            silent: "yes", // Should be boolean
            context: 123, // Should be string
        },
        invalidUserConnection: {
            id: generatePK(),
            subscriber: { connect: { id: generatePK() } },
            silent: false,
        },
    },
    edgeCases: {
        silentSubscription: {
            id: generatePK(),
            subscriber: { connect: { id: "user_placeholder_id" } },
            silent: true,
            context: "Silent notification subscription",
        },
        resourceSubscription: {
            id: generatePK(),
            subscriber: { connect: { id: "user_placeholder_id" } },
            silent: false,
            resource: { connect: { id: "resource_placeholder_id" } },
            context: "Resource update subscription",
        },
        chatSubscription: {
            id: generatePK(),
            subscriber: { connect: { id: "user_placeholder_id" } },
            silent: false,
            chat: { connect: { id: "chat_placeholder_id" } },
            context: "Chat notification subscription",
        },
    },
};

/**
 * Enhanced factory for creating notification subscription database fixtures
 */
export class NotificationSubscriptionDbFactory extends EnhancedDbFactory<Prisma.notification_subscriptionCreateInput> {

    /**
     * Override to only generate fields that exist in notification_subscription schema
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
            // notification_subscription doesn't have publicId or handle fields
        };
    }

    /**
     * Get the test fixtures for NotificationSubscription model
     */
    protected getFixtures(): DbTestFixtures<Prisma.notification_subscriptionCreateInput> {
        return notificationSubscriptionDbFixtures;
    }

    /**
     * Get NotificationSubscription-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getNotificationDbIds().subscription1, // Duplicate ID
                    subscriber: { connect: { id: "user_placeholder_id" } },
                    silent: false,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    subscriber: { connect: { id: generatePK() } },
                    silent: false,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    subscriber: { connect: { id: "user_placeholder_id" } },
                    silent: false,
                    context: "", // Empty context might violate constraint
                },
            },
            validation: {
                requiredFieldMissing: notificationSubscriptionDbFixtures.invalid.missingRequired,
                invalidDataType: notificationSubscriptionDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    context: "A".repeat(2049), // Context too long (max 2048)
                    subscriber: { connect: { id: "user_placeholder_id" } },
                    silent: false,
                },
            },
            businessLogic: {
                duplicateSubscription: {
                    id: generatePK(),
                    subscriber: { connect: { id: "user_placeholder_id" } },
                    silent: false,
                    resource: { connect: { id: "resource_placeholder_id" } },
                    // Same user, same resource - potential duplicate
                },
            },
        };
    }

    /**
     * Add object association to a subscription fixture
     */
    protected addObjectAssociation(data: Prisma.notification_subscriptionCreateInput, objectId: string, objectType: string): Prisma.notification_subscriptionCreateInput {
        const connections: Record<string, any> = {
            Resource: { resource: { connect: { id: objectId } } },
            Chat: { chat: { connect: { id: objectId } } },
            Comment: { comment: { connect: { id: objectId } } },
            Issue: { issue: { connect: { id: objectId } } },
            Meeting: { meeting: { connect: { id: objectId } } },
            PullRequest: { pullRequest: { connect: { id: objectId } } },
            Report: { report: { connect: { id: objectId } } },
            Schedule: { schedule: { connect: { id: objectId } } },
            Team: { team: { connect: { id: objectId } } },
        };

        return {
            ...data,
            ...(connections[objectType] || {}),
        };
    }

    /**
     * NotificationSubscription-specific validation
     */
    protected validateSpecific(data: Prisma.notification_subscriptionCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to NotificationSubscription
        if (!data.subscriber) errors.push("Subscription must be associated with a subscriber");
        if (data.silent === undefined) errors.push("silent flag is required");

        // Check business logic
        if (data.silent) {
            warnings.push("Subscription is silent - user won't receive audio notifications");
        }

        // Check that at least one entity is being subscribed to
        const hasEntity = !!(data.resourceId || data.chatId || data.commentId ||
            data.issueId || data.meetingId || data.pullRequestId ||
            data.reportId || data.scheduleId || data.teamId ||
            data.resource || data.chat || data.comment ||
            data.issue || data.meeting || data.pullRequest ||
            data.report || data.schedule || data.team);

        if (!hasEntity) {
            warnings.push("Subscription should reference at least one entity to subscribe to");
        }

        // Check context length
        if (data.context && data.context.length > 2048) {
            errors.push("Subscription context is too long (max 2048 characters)");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.notification_subscriptionCreateInput>,
    ): Prisma.notification_subscriptionCreateInput {
        const factory = new NotificationSubscriptionDbFactory();
        return factory.createMinimal({
            subscriber: { connect: { id: userId } },
            silent: false,
            ...overrides,
        });
    }

    static createWithObject(
        userId: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.notification_subscriptionCreateInput>,
    ): Prisma.notification_subscriptionCreateInput {
        const factory = new NotificationSubscriptionDbFactory();
        const data = factory.createMinimal({
            subscriber: { connect: { id: userId } },
            silent: false,
            context: `${objectType} updates subscription`,
            ...overrides,
        });
        return factory.addObjectAssociation(data, objectId, objectType);
    }

    static createSilent(
        userId: string,
        overrides?: Partial<Prisma.notification_subscriptionCreateInput>,
    ): Prisma.notification_subscriptionCreateInput {
        const factory = new NotificationSubscriptionDbFactory();
        return factory.createMinimal({
            subscriber: { connect: { id: userId } },
            silent: true,
            context: "Silent subscription",
            ...overrides,
        });
    }
}

/**
 * Enhanced helper to seed multiple test notifications with comprehensive options
 */
export async function seedNotifications(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        categories?: string[];
        withRead?: boolean;
        withSubscriptions?: boolean;
    },
): Promise<BulkSeedResult<any>> {
    const notificationFactory = new NotificationDbFactory();
    const subscriptionFactory = new NotificationSubscriptionDbFactory();
    const notifications = [];
    const subscriptions = [];
    const count = options.count || 5;
    const categories = options.categories || ["Update", "Reminder", "Alert"];
    let readCount = 0;
    let subscriptionCount = 0;

    for (let i = 0; i < count; i++) {
        const category = categories[i % categories.length];
        const isRead = options.withRead && i % 2 === 0;

        const notification = await prisma.notification.create({
            data: NotificationDbFactory.createMinimal(
                options.userId,
                category,
                {
                    isRead,
                    title: `${category} Notification ${i + 1}`,
                    description: `Description for ${category.toLowerCase()} notification ${i + 1}`,
                },
            ),
            include: {
                user: true,
                fromUser: true,
                team: true,
                comment: true,
                reaction: true,
            },
        });
        notifications.push(notification);

        if (isRead) readCount++;
    }

    if (options.withSubscriptions) {
        for (let i = 0; i < categories.length; i++) {
            const subscription = await prisma.notification_subscription.create({
                data: NotificationSubscriptionDbFactory.createMinimal(
                    options.userId,
                    {
                        context: `Subscription for ${categories[i]} notifications`,
                        silent: i % 2 === 0, // Alternate between silent and non-silent
                    },
                ),
                include: {
                    subscriber: true,
                    team: true,
                    resource: true,
                    chat: true,
                },
            });
            subscriptions.push(subscription);
            subscriptionCount++;
        }
    }

    return {
        records: { notifications, subscriptions },
        summary: {
            total: notifications.length + subscriptions.length,
            notifications: notifications.length,
            subscriptions: subscriptionCount,
            read: readCount,
            categories: categories.length,
        },
    };
}
