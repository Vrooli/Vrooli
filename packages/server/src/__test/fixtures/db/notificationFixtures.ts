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
            // Placeholder IDs for testing
            userPlaceholder: generatePK(),
            teamPlaceholder: generatePK(),
            inviterUser: generatePK(),
            senderUser: generatePK(),
            reactorUser: generatePK(),
            reactionPlaceholder: generatePK(),
            resourcePlaceholder: generatePK(),
            chatPlaceholder: generatePK(),
        };
    }
    return _notificationDbIds;
}

// Lazy initialization for notification fixture IDs
let _notificationFixtureIds: Record<string, bigint> | null = null;
function getNotificationFixtureIds() {
    if (!_notificationFixtureIds) {
        _notificationFixtureIds = {
            minimal: generatePK(),
            complete: generatePK(),
            oldNotification: generatePK(),
            longContent: generatePK(),
            multipleObject: generatePK(),
            system: generatePK(),
            reaction: generatePK(),
            // Invalid cases
            invalidUser: generatePK(),
            invalidObject: generatePK(),
            // Error scenarios
            foreignKeyViolation: generatePK(),
            checkConstraintViolation: generatePK(),
            outOfRange: generatePK(),
            selfNotification: generatePK(),
            orphanedObject: generatePK(),
        };
    }
    return _notificationFixtureIds;
}

// Lazy initialization for subscription fixture IDs
let _subscriptionFixtureIds: Record<string, bigint> | null = null;
function getSubscriptionFixtureIds() {
    if (!_subscriptionFixtureIds) {
        _subscriptionFixtureIds = {
            minimal: generatePK(),
            complete: generatePK(),
            invalidUser: generatePK(),
            invalidTypes: generatePK(),
            silent: generatePK(),
            resource: generatePK(),
            chat: generatePK(),
            // Error scenarios
            foreignKeyViolation: generatePK(),
            checkConstraintViolation: generatePK(),
            outOfRange: generatePK(),
            duplicateSubscription: generatePK(),
        };
    }
    return _subscriptionFixtureIds;
}

/**
 * Enhanced test fixtures for Notification model following standard structure
 */
export const notificationDbFixtures: DbTestFixtures<Prisma.notificationCreateInput> = {
    minimal: {
        id: getNotificationFixtureIds().minimal,
        category: "Update",
        title: "Test Notification",
        user: { connect: { id: getNotificationDbIds().userPlaceholder } },
        isRead: false,
    },
    complete: {
        id: getNotificationFixtureIds().complete,
        category: "TeamInvite",
        title: "Team Invitation",
        description: "You have been invited to join the Development Team",
        user: { connect: { id: getNotificationDbIds().userPlaceholder } },
        isRead: false,
        count: 1,
        link: "https://example.com/team-invite",
        imgLink: "https://example.com/team-avatar.jpg",
        createdAt: new Date(),
    },
    invalid: {
        missingRequired: {
            // Missing required category, title, and user
            isRead: false,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake" as any, // Invalid string ID
            category: 123 as any, // Should be string
            isRead: "yes" as any, // Should be boolean
            title: 456 as any, // Should be string
            description: true as any, // Should be string
            user: "invalid-user-reference" as any, // Should be connect object
        },
        invalidUserConnection: {
            id: getNotificationFixtureIds().invalidUser,
            category: "Update",
            title: "Invalid User Test",
            user: { connect: { id: generatePK() } }, // Non-existent user ID
            isRead: false,
        },
        invalidObjectConnection: {
            id: getNotificationFixtureIds().invalidObject,
            category: "CommentReply",
            title: "Invalid Connection Test",
            user: { connect: { id: getNotificationDbIds().userPlaceholder } },
            isRead: false,
        },
    },
    edgeCases: {
        oldNotification: {
            id: getNotificationFixtureIds().oldNotification,
            category: "Update",
            title: "Old Notification",
            description: "This notification was created a year ago",
            user: { connect: { id: getNotificationDbIds().userPlaceholder } },
            isRead: true,
            createdAt: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000), // 1 year ago
        },
        longContentNotification: {
            id: getNotificationFixtureIds().longContent,
            category: "SystemAlert",
            title: "System Maintenance Alert - Critical Updates Required",
            description: "A".repeat(1000), // Very long description
            user: { connect: { id: getNotificationDbIds().userPlaceholder } },
            isRead: false,
        },
        multipleObjectNotification: {
            id: getNotificationFixtureIds().multipleObject,
            category: "ProjectUpdate",
            title: "Project and Team Update",
            description: "Multiple updates across different areas",
            user: { connect: { id: getNotificationDbIds().userPlaceholder } },
            isRead: false,
            link: "https://example.com/project/123",
        },
        systemNotification: {
            id: getNotificationFixtureIds().system,
            category: "SystemMaintenance",
            title: "Scheduled Maintenance",
            description: "System will be offline for maintenance from 2-4 AM UTC",
            user: { connect: { id: getNotificationDbIds().userPlaceholder } },
            isRead: false,
        },
        reactionNotification: {
            id: getNotificationFixtureIds().reaction,
            category: "Reaction",
            title: "New Reaction",
            description: "Someone liked your comment",
            user: { connect: { id: getNotificationDbIds().userPlaceholder } },
            isRead: false,
            link: "https://example.com/comment/456",
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
                    title: "Duplicate Test",
                    user: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    isRead: false,
                },
                foreignKeyViolation: {
                    id: getNotificationFixtureIds().foreignKeyViolation,
                    category: "Update",
                    title: "Foreign Key Test",
                    user: { connect: { id: generatePK() } }, // Non-existent user ID
                    isRead: false,
                },
                checkConstraintViolation: {
                    id: getNotificationFixtureIds().checkConstraintViolation,
                    category: "", // Empty category violates constraint
                    title: "Constraint Test",
                    user: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    isRead: false,
                },
            },
            validation: {
                requiredFieldMissing: notificationDbFixtures.invalid.missingRequired,
                invalidDataType: notificationDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: getNotificationFixtureIds().outOfRange,
                    category: "A".repeat(256), // Category too long
                    title: "Out of Range Test",
                    user: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    isRead: false,
                },
            },
            businessLogic: {
                notificationToSelf: {
                    id: getNotificationFixtureIds().selfNotification,
                    category: "TeamInvite",
                    title: "Self Notification Test",
                    user: { connect: { id: getNotificationDbIds().userPlaceholder } }, // Same user as sender in business logic
                    isRead: false,
                },
                orphanedObjectNotification: {
                    id: getNotificationFixtureIds().orphanedObject,
                    category: "CommentReply",
                    title: "Orphaned Object Test",
                    user: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    isRead: false,
                },
            },
        };
    }

    /**
     * Add object association to a notification fixture
     */
    protected addObjectAssociation(data: Prisma.notificationCreateInput, objectId: bigint, objectType: string): Prisma.notificationCreateInput {
        // Since notification model doesn't have object relations, we simulate with link field
        const linkUrls: Record<string, string> = {
            Comment: `https://example.com/comment/${objectId}`,
            Issue: `https://example.com/issue/${objectId}`,
            Question: `https://example.com/question/${objectId}`,
            Quiz: `https://example.com/quiz/${objectId}`,
            Reaction: `https://example.com/reaction/${objectId}`,
            Report: `https://example.com/report/${objectId}`,
            Team: `https://example.com/team/${objectId}`,
            User: `https://example.com/user/${objectId}`,
        };

        return {
            ...data,
            link: linkUrls[objectType] || data.link,
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
        if (!data.title) errors.push("Notification title is required");
        if (!data.user) errors.push("Notification must be associated with a user");
        if (data.isRead === undefined) errors.push("isRead flag is required");

        // Check category-specific requirements (simplified since no object relations)
        if (data.category?.includes("Team") && !data.link) {
            warnings.push("Team-related notifications should have a link");
        }

        if (data.category?.includes("Comment") && !data.link) {
            warnings.push("Comment-related notifications should have a link");
        }

        if (data.category?.includes("Reaction") && !data.link) {
            warnings.push("Reaction notifications should have a link");
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
        userId: bigint,
        category: string,
        overrides?: Partial<Prisma.notificationCreateInput>,
    ): Prisma.notificationCreateInput {
        const factory = new NotificationDbFactory();
        return factory.createMinimal({
            category,
            title: "Test Notification",
            user: { connect: { id: userId } },
            ...overrides,
        });
    }

    static createWithObject(
        userId: bigint,
        category: string,
        objectId: bigint,
        objectType: string,
        overrides?: Partial<Prisma.notificationCreateInput>,
    ): Prisma.notificationCreateInput {
        const factory = new NotificationDbFactory();
        const data = factory.createMinimal({
            category,
            title: "Test Notification",
            user: { connect: { id: userId } },
            ...overrides,
        });
        return factory.addObjectAssociation(data, objectId, objectType);
    }

    static createRead(
        userId: bigint,
        category: string,
        overrides?: Partial<Prisma.notificationCreateInput>,
    ): Prisma.notificationCreateInput {
        const factory = new NotificationDbFactory();
        return factory.createMinimal({
            category,
            title: "Test Notification",
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
        id: getSubscriptionFixtureIds().minimal,
        subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
        silent: false,
    },
    complete: {
        id: getSubscriptionFixtureIds().complete,
        subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
        silent: false,
        context: "Test subscription context",
        team: { connect: { id: getNotificationDbIds().teamPlaceholder } },
    },
    invalid: {
        missingRequired: {
            // Missing required subscriber
            silent: false,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake" as any, // Invalid string ID
            silent: "yes" as any, // Should be boolean
            context: 123 as any, // Should be string
            subscriber: "invalid-subscriber-reference" as any, // Should be connect object
        },
        invalidUserConnection: {
            id: getSubscriptionFixtureIds().invalidUser,
            subscriber: { connect: { id: generatePK() } }, // Non-existent user ID
            silent: false,
        },
    },
    edgeCases: {
        silentSubscription: {
            id: getSubscriptionFixtureIds().silent,
            subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
            silent: true,
            context: "Silent notification subscription",
        },
        resourceSubscription: {
            id: getSubscriptionFixtureIds().resource,
            subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
            silent: false,
            resource: { connect: { id: getNotificationDbIds().resourcePlaceholder } },
            context: "Resource update subscription",
        },
        chatSubscription: {
            id: getSubscriptionFixtureIds().chat,
            subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
            silent: false,
            chat: { connect: { id: getNotificationDbIds().chatPlaceholder } },
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
                    subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    silent: false,
                },
                foreignKeyViolation: {
                    id: getSubscriptionFixtureIds().foreignKeyViolation,
                    subscriber: { connect: { id: generatePK() } }, // Non-existent user ID
                    silent: false,
                },
                checkConstraintViolation: {
                    id: getSubscriptionFixtureIds().checkConstraintViolation,
                    subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    silent: false,
                    context: "", // Empty context might violate constraint
                },
            },
            validation: {
                requiredFieldMissing: notificationSubscriptionDbFixtures.invalid.missingRequired,
                invalidDataType: notificationSubscriptionDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: getSubscriptionFixtureIds().outOfRange,
                    context: "A".repeat(2049), // Context too long (max 2048)
                    subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    silent: false,
                },
            },
            businessLogic: {
                duplicateSubscription: {
                    id: getSubscriptionFixtureIds().duplicateSubscription,
                    subscriber: { connect: { id: getNotificationDbIds().userPlaceholder } },
                    silent: false,
                    resource: { connect: { id: getNotificationDbIds().resourcePlaceholder } },
                    // Same user, same resource - potential duplicate
                },
            },
        };
    }

    /**
     * Add object association to a subscription fixture
     */
    protected addObjectAssociation(data: Prisma.notification_subscriptionCreateInput, objectId: bigint, objectType: string): Prisma.notification_subscriptionCreateInput {
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
        const hasEntity = !!(data.resource || data.chat || data.comment ||
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
        userId: bigint,
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
        userId: bigint,
        objectId: bigint,
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
        userId: bigint,
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
        userId: bigint;
        count?: number;
        categories?: string[];
        withRead?: boolean;
        withSubscriptions?: boolean;
    },
): Promise<BulkSeedResult<any>> {
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
        records: [...notifications, ...subscriptions],
        summary: {
            total: notifications.length + subscriptions.length,
            withAuth: 0,
            bots: 0,
            teams: 0,
            notifications: notifications.length,
            subscriptions: subscriptionCount,
            read: readCount,
            categories: categories.length,
        },
    };
}
