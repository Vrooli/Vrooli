import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ReputationHistory model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const reputationHistoryDbIds = {
    history1: generatePK(),
    history2: generatePK(),
    history3: generatePK(),
    history4: generatePK(),
    history5: generatePK(),
};

/**
 * Minimal reputation history data for database creation
 */
export const minimalReputationHistoryDb: Prisma.Reputation_historyCreateInput = {
    id: reputationHistoryDbIds.history1,
    amount: 10,
    event: "test_event",
    user: { connect: { id: "user_123" } },
};

/**
 * Complete reputation history with all fields
 */
export const completeReputationHistoryDb: Prisma.Reputation_historyCreateInput = {
    id: reputationHistoryDbIds.history2,
    amount: 25,
    event: "routine_completed",
    objectId1: "routine_456",
    objectId2: "project_789",
    user: { connect: { id: "user_123" } },
};

/**
 * Factory for creating reputation history database fixtures with overrides
 */
export class ReputationHistoryDbFactory {
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.Reputation_historyCreateInput>
    ): Prisma.Reputation_historyCreateInput {
        return {
            id: generatePK(),
            amount: 10,
            event: "test_event",
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createComplete(
        userId: string,
        overrides?: Partial<Prisma.Reputation_historyCreateInput>
    ): Prisma.Reputation_historyCreateInput {
        return {
            id: generatePK(),
            amount: 25,
            event: "routine_completed",
            objectId1: "routine_456",
            objectId2: "project_789",
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create reputation history for positive events
     */
    static createPositive(
        userId: string,
        amount: number = 15,
        event: string = "positive_action",
        overrides?: Partial<Prisma.Reputation_historyCreateInput>
    ): Prisma.Reputation_historyCreateInput {
        return {
            id: generatePK(),
            amount,
            event,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create reputation history for negative events
     */
    static createNegative(
        userId: string,
        amount: number = -5,
        event: string = "negative_action",
        overrides?: Partial<Prisma.Reputation_historyCreateInput>
    ): Prisma.Reputation_historyCreateInput {
        return {
            id: generatePK(),
            amount,
            event,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create reputation history with object references
     */
    static createWithObjects(
        userId: string,
        objectId1: string,
        objectId2?: string,
        overrides?: Partial<Prisma.Reputation_historyCreateInput>
    ): Prisma.Reputation_historyCreateInput {
        return {
            id: generatePK(),
            amount: 20,
            event: "object_interaction",
            objectId1,
            objectId2,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create reputation history for specific event types
     */
    static createForEvent(
        userId: string,
        eventType: 'routine_completed' | 'project_created' | 'comment_liked' | 'api_used' | 'resource_shared',
        overrides?: Partial<Prisma.Reputation_historyCreateInput>
    ): Prisma.Reputation_historyCreateInput {
        const eventConfig = {
            routine_completed: { amount: 50, event: "routine_completed" },
            project_created: { amount: 30, event: "project_created" },
            comment_liked: { amount: 5, event: "comment_liked" },
            api_used: { amount: 2, event: "api_used" },
            resource_shared: { amount: 10, event: "resource_shared" },
        };

        const config = eventConfig[eventType];
        
        return {
            id: generatePK(),
            amount: config.amount,
            event: config.event,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }
}

/**
 * Helper to seed reputation history for testing
 */
export async function seedReputationHistory(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        events?: Array<{
            amount: number;
            event: string;
            objectId1?: string;
            objectId2?: string;
        }>;
        includeNegative?: boolean;
    }
) {
    const histories = [];
    const count = options.count || 5;

    if (options.events) {
        // Create specific events
        for (const eventData of options.events) {
            const history = await prisma.reputation_history.create({
                data: ReputationHistoryDbFactory.createMinimal(options.userId, {
                    amount: eventData.amount,
                    event: eventData.event,
                    objectId1: eventData.objectId1,
                    objectId2: eventData.objectId2,
                }),
            });
            histories.push(history);
        }
    } else {
        // Create a mix of different event types
        const eventTypes: Array<'routine_completed' | 'project_created' | 'comment_liked' | 'api_used' | 'resource_shared'> = [
            'routine_completed',
            'project_created', 
            'comment_liked',
            'api_used',
            'resource_shared',
        ];

        for (let i = 0; i < count; i++) {
            const eventType = eventTypes[i % eventTypes.length];
            const history = await prisma.reputation_history.create({
                data: ReputationHistoryDbFactory.createForEvent(options.userId, eventType),
            });
            histories.push(history);
        }

        // Add some negative events if requested
        if (options.includeNegative) {
            const negativeHistory = await prisma.reputation_history.create({
                data: ReputationHistoryDbFactory.createNegative(options.userId, -10, "violation_reported"),
            });
            histories.push(negativeHistory);
        }
    }

    return histories;
}

/**
 * Helper to calculate total reputation from history
 */
export function calculateTotalReputation(histories: Array<{ amount: number }>): number {
    return histories.reduce((total, history) => total + history.amount, 0);
}

/**
 * Helper to seed reputation history with timeline progression
 */
export async function seedReputationTimeline(
    prisma: any,
    userId: string,
    options?: {
        startDate?: Date;
        dayIncrement?: number;
        eventCount?: number;
    }
) {
    const startDate = options?.startDate || new Date();
    const dayIncrement = options?.dayIncrement || 1;
    const eventCount = options?.eventCount || 7;
    
    const histories = [];
    
    for (let i = 0; i < eventCount; i++) {
        const eventDate = new Date(startDate);
        eventDate.setDate(eventDate.getDate() + (i * dayIncrement));
        
        const history = await prisma.reputation_history.create({
            data: {
                ...ReputationHistoryDbFactory.createForEvent(userId, 'routine_completed'),
                createdAt: eventDate,
                updatedAt: eventDate,
            },
        });
        histories.push(history);
    }
    
    return histories;
}