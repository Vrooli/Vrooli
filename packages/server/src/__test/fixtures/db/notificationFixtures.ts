import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Notification model - used for seeding test data
 */

export class NotificationDbFactory {
    static createMinimal(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationCreateInput>
    ): Prisma.NotificationCreateInput {
        return {
            id: generatePK(),
            category,
            user: { connect: { id: userId } },
            isRead: false,
            ...overrides,
        };
    }

    static createWithObject(
        userId: string,
        category: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.NotificationCreateInput>
    ): Prisma.NotificationCreateInput {
        const base = this.createMinimal(userId, category, overrides);
        
        // Add the appropriate connection based on object type
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
            ...base,
            ...(connections[objectType] || {}),
        };
    }

    static createRead(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationCreateInput>
    ): Prisma.NotificationCreateInput {
        return this.createMinimal(userId, category, {
            isRead: true,
            ...overrides,
        });
    }
}

/**
 * Database fixtures for NotificationSubscription model
 */
export class NotificationSubscriptionDbFactory {
    static createMinimal(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationSubscriptionCreateInput>
    ): Prisma.NotificationSubscriptionCreateInput {
        return {
            id: generatePK(),
            category,
            user: { connect: { id: userId } },
            isEnabled: true,
            ...overrides,
        };
    }

    static createWithObject(
        userId: string,
        objectId: string,
        objectType: string,
        overrides?: Partial<Prisma.NotificationSubscriptionCreateInput>
    ): Prisma.NotificationSubscriptionCreateInput {
        const base = this.createMinimal(userId, `${objectType}Updates`, overrides);
        
        // Add the appropriate connection based on object type
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
            ...base,
            ...(connections[objectType] || {}),
        };
    }

    static createDisabled(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.NotificationSubscriptionCreateInput>
    ): Prisma.NotificationSubscriptionCreateInput {
        return this.createMinimal(userId, category, {
            isEnabled: false,
            ...overrides,
        });
    }
}

/**
 * Helper to seed notifications for testing
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
) {
    const notifications = [];
    const subscriptions = [];
    const count = options.count || 5;
    const categories = options.categories || ["Update", "Reminder", "Alert"];

    for (let i = 0; i < count; i++) {
        const category = categories[i % categories.length];
        const isRead = options.withRead && i % 2 === 0;
        
        const notification = await prisma.notification.create({
            data: NotificationDbFactory.createMinimal(
                options.userId,
                category,
                { isRead }
            ),
        });
        notifications.push(notification);
    }

    if (options.withSubscriptions) {
        for (const category of categories) {
            const subscription = await prisma.notificationSubscription.create({
                data: NotificationSubscriptionDbFactory.createMinimal(
                    options.userId,
                    category
                ),
            });
            subscriptions.push(subscription);
        }
    }

    return { notifications, subscriptions };
}