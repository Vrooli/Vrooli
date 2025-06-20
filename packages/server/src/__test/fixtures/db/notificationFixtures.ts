import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Notification model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const notificationDbIds = {
    notification1: generatePK(),
    notification2: generatePK(),
    notification3: generatePK(),
    subscription1: generatePK(),
    subscription2: generatePK(),
    subscription3: generatePK(),
};

/**
 * Enhanced test fixtures for Notification model following standard structure
 */
export const notificationDbFixtures: DbTestFixtures<Prisma.NotificationCreateInput> = {
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
            user: { connect: { id: "non-existent-user-id" } },
            isRead: false,
        },
        invalidObjectConnection: {
            id: generatePK(),
            category: "CommentReply",
            user: { connect: { id: "user_placeholder_id" } },
            isRead: false,
            comment: { connect: { id: "non-existent-comment-id" } },
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
export class NotificationDbFactory extends EnhancedDbFactory<Prisma.NotificationCreateInput> {
    
    /**
     * Get the test fixtures for Notification model
     */
    protected getFixtures(): DbTestFixtures<Prisma.NotificationCreateInput> {
        return notificationDbFixtures;
    }

    /**
     * Get Notification-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: notificationDbIds.notification1, // Duplicate ID
                    category: "Update",
                    user: { connect: { id: "user_placeholder_id" } },
                    isRead: false,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    category: "Update",
                    user: { connect: { id: "non-existent-user-id" } },
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
                    comment: { connect: { id: "non-existent-comment-id" } },
                },
            },
        };
    }

    /**
     * Add object association to a notification fixture
     */
    protected addObjectAssociation(data: Prisma.NotificationCreateInput, objectId: string, objectType: string): Prisma.NotificationCreateInput {
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
    protected validateSpecific(data: Prisma.NotificationCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Notification
        if (!data.category) errors.push("Notification category is required");
        if (!data.user) errors.push("Notification must be associated with a user");
        if (data.isRead === undefined) errors.push("isRead flag is required");

        // Check business logic
        if (data.fromUser && data.user && 
            typeof data.fromUser === 'object' && 'connect' in data.fromUser &&
            typeof data.user === 'object' && 'connect' in data.user &&
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
        overrides?: Partial<Prisma.NotificationCreateInput>
    ): Prisma.NotificationCreateInput {
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
        overrides?: Partial<Prisma.NotificationCreateInput>
    ): Prisma.NotificationCreateInput {
        const factory = new NotificationDbFactory();
        let data = factory.createMinimal({
            category,
            user: { connect: { id: userId } },
            ...overrides,
        });
        return factory.addObjectAssociation(data, objectId, objectType);
    }

    static createRead(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationCreateInput>
    ): Prisma.NotificationCreateInput {
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
export const notificationSubscriptionDbFixtures: DbTestFixtures<Prisma.NotificationSubscriptionCreateInput> = {
    minimal: {
        id: generatePK(),
        category: "Updates",
        user: { connect: { id: "user_placeholder_id" } },
        isEnabled: true,
    },
    complete: {
        id: generatePK(),
        category: "TeamUpdates",
        user: { connect: { id: "user_placeholder_id" } },
        isEnabled: true,
        team: { connect: { id: "team_placeholder_id" } },
    },
    invalid: {
        missingRequired: {
            // Missing required category and user
            isEnabled: true,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            category: 123, // Should be string
            isEnabled: "yes", // Should be boolean
        },
        invalidUserConnection: {
            id: generatePK(),
            category: "Updates",
            user: { connect: { id: "non-existent-user-id" } },
            isEnabled: true,
        },
    },
    edgeCases: {
        disabledSubscription: {
            id: generatePK(),
            category: "Alerts",
            user: { connect: { id: "user_placeholder_id" } },
            isEnabled: false,
        },
        projectSubscription: {
            id: generatePK(),
            category: "ProjectUpdates",
            user: { connect: { id: "user_placeholder_id" } },
            isEnabled: true,
            project: { connect: { id: "project_placeholder_id" } },
        },
        routineSubscription: {
            id: generatePK(),
            category: "RoutineUpdates",
            user: { connect: { id: "user_placeholder_id" } },
            isEnabled: true,
            routine: { connect: { id: "routine_placeholder_id" } },
        },
    },
};

/**
 * Enhanced factory for creating notification subscription database fixtures
 */
export class NotificationSubscriptionDbFactory extends EnhancedDbFactory<Prisma.NotificationSubscriptionCreateInput> {
    
    /**
     * Get the test fixtures for NotificationSubscription model
     */
    protected getFixtures(): DbTestFixtures<Prisma.NotificationSubscriptionCreateInput> {
        return notificationSubscriptionDbFixtures;
    }

    /**
     * Get NotificationSubscription-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: notificationDbIds.subscription1, // Duplicate ID
                    category: "Updates",
                    user: { connect: { id: "user_placeholder_id" } },
                    isEnabled: true,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    category: "Updates",
                    user: { connect: { id: "non-existent-user-id" } },
                    isEnabled: true,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    category: "", // Empty category violates constraint
                    user: { connect: { id: "user_placeholder_id" } },
                    isEnabled: true,
                },
            },
            validation: {
                requiredFieldMissing: notificationSubscriptionDbFixtures.invalid.missingRequired,
                invalidDataType: notificationSubscriptionDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    category: "A".repeat(256), // Category too long
                    user: { connect: { id: "user_placeholder_id" } },
                    isEnabled: true,
                },
            },
            businessLogic: {
                duplicateSubscription: {
                    id: generatePK(),
                    category: "Updates",
                    user: { connect: { id: "user_placeholder_id" } },
                    isEnabled: true,
                    // Same user, same category - potential duplicate
                },
            },
        };
    }

    /**
     * Add object association to a subscription fixture
     */
    protected addObjectAssociation(data: Prisma.NotificationSubscriptionCreateInput, objectId: string, objectType: string): Prisma.NotificationSubscriptionCreateInput {
        const connections: Record<string, any> = {
            Api: { api: { connect: { id: objectId } } },
            Comment: { comment: { connect: { id: objectId } } },
            Issue: { issue: { connect: { id: objectId } } },
            Meeting: { meeting: { connect: { id: objectId } } },
            Note: { note: { connect: { id: objectId } } },
            Project: { project: { connect: { id: objectId } } },
            Question: { question: { connect: { id: objectId } } },
            Quiz: { quiz: { connect: { id: objectId } } },
            Report: { report: { connect: { id: objectId } } },
            Routine: { routine: { connect: { id: objectId } } },
            SmartContract: { smartContract: { connect: { id: objectId } } },
            Standard: { standard: { connect: { id: objectId } } },
            Team: { team: { connect: { id: objectId } } },
            User: { user: { connect: { id: objectId } } },
        };

        return {
            ...data,
            ...(connections[objectType] || {}),
        };
    }

    /**
     * NotificationSubscription-specific validation
     */
    protected validateSpecific(data: Prisma.NotificationSubscriptionCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to NotificationSubscription
        if (!data.category) errors.push("Subscription category is required");
        if (!data.user) errors.push("Subscription must be associated with a user");
        if (data.isEnabled === undefined) errors.push("isEnabled flag is required");

        // Check business logic
        if (!data.isEnabled) {
            warnings.push("Subscription is disabled - user won't receive notifications");
        }

        // Check category-specific requirements
        if (data.category?.includes("Team") && !data.team) {
            warnings.push("Team-related subscriptions should reference a team");
        }

        if (data.category?.includes("Project") && !data.project) {
            warnings.push("Project-related subscriptions should reference a project");
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationSubscriptionCreateInput>
    ): Prisma.NotificationSubscriptionCreateInput {
        const factory = new NotificationSubscriptionDbFactory();
        return factory.createMinimal({
            category,
            user: { connect: { id: userId } },
            ...overrides,
        });
    }

    static createWithObject(
        userId: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.NotificationSubscriptionCreateInput>
    ): Prisma.NotificationSubscriptionCreateInput {
        const factory = new NotificationSubscriptionDbFactory();
        let data = factory.createMinimal({
            category: `${objectType}Updates`,
            user: { connect: { id: userId } },
            ...overrides,
        });
        return factory.addObjectAssociation(data, objectId, objectType);
    }

    static createDisabled(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationSubscriptionCreateInput>
    ): Prisma.NotificationSubscriptionCreateInput {
        const factory = new NotificationSubscriptionDbFactory();
        return factory.createMinimal({
            category,
            user: { connect: { id: userId } },
            isEnabled: false,
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
    }
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
                }
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
        for (const category of categories) {
            const subscription = await prisma.notificationSubscription.create({
                data: NotificationSubscriptionDbFactory.createMinimal(
                    options.userId,
                    category
                ),
                include: {
                    user: true,
                    team: true,
                    project: true,
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