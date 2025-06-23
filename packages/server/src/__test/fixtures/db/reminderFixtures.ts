import { generatePK, generatePublicId, nanoid } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Reminder model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const reminderDbIds = {
    reminder1: generatePK(),
    reminder2: generatePK(),
    reminder3: generatePK(),
    list1: generatePK(),
    list2: generatePK(),
    item1: generatePK(),
    item2: generatePK(),
    item3: generatePK(),
};

/**
 * Enhanced test fixtures for Reminder model following standard structure
 */
export const reminderDbFixtures: DbTestFixtures<Prisma.ReminderCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        user: { connect: { id: "user_placeholder_id" } },
        reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000), // 1 day from now
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        user: { connect: { id: "user_placeholder_id" } },
        reminderAt: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000), // 2 days from now
        name: "Complete Reminder",
        description: "A comprehensive reminder with all features",
        dueDate: new Date(Date.now() + 3 * 24 * 60 * 60 * 1000), // 3 days from now
        index: 0,
        isComplete: false,
        reminderList: { connect: { id: "list_placeholder_id" } },
        reminderItems: {
            create: [
                {
                    id: generatePK(),
                    name: "Task 1",
                    description: "Complete the first task",
                    dueDate: new Date(Date.now() + 24 * 60 * 60 * 1000),
                    index: 0,
                    isComplete: false,
                },
                {
                    id: generatePK(),
                    name: "Task 2",
                    description: "Complete the second task",
                    dueDate: new Date(Date.now() + 48 * 60 * 60 * 1000),
                    index: 1,
                    isComplete: true,
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required user and reminderAt
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            reminderAt: "not-a-date", // Should be Date
            dueDate: "not-a-date", // Should be Date
            index: "not-a-number", // Should be number
            isComplete: "yes", // Should be boolean
        },
        invalidUserConnection: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "non-existent-user-id" } },
            reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
        },
        invalidListConnection: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
            reminderList: { connect: { id: "non-existent-list-id" } },
        },
    },
    edgeCases: {
        pastReminder: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            reminderAt: new Date(Date.now() - 24 * 60 * 60 * 1000), // 1 day ago
            name: "Past Reminder",
            description: "This reminder was scheduled in the past",
            isComplete: false,
        },
        completedReminder: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
            name: "Completed Reminder",
            description: "This reminder has been marked as complete",
            isComplete: true,
        },
        manyItemsReminder: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
            name: "Reminder with Many Items",
            reminderItems: {
                create: Array.from({ length: 20 }, (_, i) => ({
                    id: generatePK(),
                    name: `Task ${i + 1}`,
                    description: `Description for task ${i + 1}`,
                    index: i,
                    isComplete: i % 3 === 0, // Every third item completed
                })),
            },
        },
        urgentReminder: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            reminderAt: new Date(Date.now() + 60 * 60 * 1000), // 1 hour from now
            name: "Urgent Reminder",
            description: "This reminder is due very soon",
            dueDate: new Date(Date.now() + 2 * 60 * 60 * 1000), // 2 hours from now
            index: 0,
            isComplete: false,
        },
        longTermReminder: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            reminderAt: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year from now
            name: "Long-term Reminder",
            description: "This reminder is set for the distant future",
            dueDate: new Date(Date.now() + 400 * 24 * 60 * 60 * 1000), // 400 days from now
            index: 0,
            isComplete: false,
        },
    },
};

/**
 * Enhanced factory for creating reminder database fixtures
 */
export class ReminderDbFactory extends EnhancedDbFactory<Prisma.ReminderCreateInput> {
    
    /**
     * Get the test fixtures for Reminder model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ReminderCreateInput> {
        return reminderDbFixtures;
    }

    /**
     * Get Reminder-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: reminderDbIds.reminder1, // Duplicate ID
                    publicId: generatePublicId(),
                    user: { connect: { id: "user_placeholder_id" } },
                    reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: "non-existent-user-id" } },
                    reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    user: { connect: { id: "user_placeholder_id" } },
                    reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
                },
            },
            validation: {
                requiredFieldMissing: reminderDbFixtures.invalid.missingRequired,
                invalidDataType: reminderDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: "user_placeholder_id" } },
                    reminderAt: new Date("1900-01-01"), // Too far in past
                    index: -1, // Negative index
                },
            },
            businessLogic: {
                dueDateBeforeReminder: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: "user_placeholder_id" } },
                    reminderAt: new Date(Date.now() + 48 * 60 * 60 * 1000), // 2 days
                    dueDate: new Date(Date.now() + 24 * 60 * 60 * 1000), // 1 day - before reminder
                },
                completedWithIncompleteItems: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: "user_placeholder_id" } },
                    reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000),
                    isComplete: true, // Reminder complete
                    reminderItems: {
                        create: [{
                            id: generatePK(),
                            name: "Incomplete Task",
                            index: 0,
                            isComplete: false, // But has incomplete items
                        }],
                    },
                },
            },
        };
    }

    /**
     * Add user association to a reminder fixture
     */
    protected addUserAssociation(data: Prisma.ReminderCreateInput, userId: string): Prisma.ReminderCreateInput {
        return {
            ...data,
            user: { connect: { id: userId } },
        };
    }

    /**
     * Add list association to a reminder fixture
     */
    protected addListAssociation(data: Prisma.ReminderCreateInput, listId: string): Prisma.ReminderCreateInput {
        return {
            ...data,
            reminderList: { connect: { id: listId } },
        };
    }

    /**
     * Add items to a reminder fixture
     */
    protected addItems(data: Prisma.ReminderCreateInput, items: Array<{
        name: string;
        description?: string;
        dueDate?: Date;
        index?: number;
        isComplete?: boolean;
    }>): Prisma.ReminderCreateInput {
        return {
            ...data,
            reminderItems: {
                create: items.map((item, i) => ({
                    id: generatePK(),
                    name: item.name,
                    description: item.description,
                    dueDate: item.dueDate,
                    index: item.index ?? i,
                    isComplete: item.isComplete ?? false,
                })),
            },
        };
    }

    /**
     * Reminder-specific validation
     */
    protected validateSpecific(data: Prisma.ReminderCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Reminder
        if (!data.user) errors.push("Reminder must be associated with a user");
        if (!data.reminderAt) errors.push("Reminder time is required");

        // Check business logic
        if (data.reminderAt && data.reminderAt < new Date()) {
            warnings.push("Reminder is scheduled in the past");
        }

        if (data.dueDate && data.reminderAt && data.dueDate < data.reminderAt) {
            warnings.push("Due date is before reminder time");
        }

        if (data.index !== undefined && data.index < 0) {
            errors.push("Reminder index cannot be negative");
        }

        // Check items logic
        if (data.reminderItems?.create) {
            const items = Array.isArray(data.reminderItems.create) ? data.reminderItems.create : [data.reminderItems.create];
            const allItemsComplete = items.every(item => item.isComplete);
            
            if (data.isComplete && !allItemsComplete) {
                warnings.push("Reminder is marked complete but has incomplete items");
            }

            if (!data.isComplete && allItemsComplete && items.length > 0) {
                warnings.push("All items are complete but reminder is not marked as complete");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.ReminderCreateInput>,
    ): Prisma.ReminderCreateInput {
        const factory = new ReminderDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addUserAssociation(data, userId);
    }

    static createWithName(
        userId: string,
        name: string,
        overrides?: Partial<Prisma.ReminderCreateInput>,
    ): Prisma.ReminderCreateInput {
        const factory = new ReminderDbFactory();
        const data = factory.createMinimal({ name, ...overrides });
        return factory.addUserAssociation(data, userId);
    }

    static createInList(
        userId: string,
        listId: string,
        overrides?: Partial<Prisma.ReminderCreateInput>,
    ): Prisma.ReminderCreateInput {
        const factory = new ReminderDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addUserAssociation(data, userId);
        return factory.addListAssociation(data, listId);
    }

    static createWithItem(
        userId: string,
        itemData: Partial<Prisma.ReminderItemCreateWithoutReminderInput>,
        overrides?: Partial<Prisma.ReminderCreateInput>,
    ): Prisma.ReminderCreateInput {
        const factory = new ReminderDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addUserAssociation(data, userId);
        return factory.addItems(data, [{
            name: itemData.name || "Reminder Item",
            description: itemData.description,
            dueDate: itemData.dueDate,
            index: itemData.index || 0,
            isComplete: itemData.isComplete || false,
        }]);
    }
}

/**
 * Enhanced test fixtures for ReminderList model
 */
export const reminderListDbFixtures: DbTestFixtures<Prisma.ReminderListCreateInput> = {
    minimal: {
        id: generatePK(),
        publicId: generatePublicId(),
        user: { connect: { id: "user_placeholder_id" } },
    },
    complete: {
        id: generatePK(),
        publicId: generatePublicId(),
        user: { connect: { id: "user_placeholder_id" } },
        name: "Personal Reminders",
        description: "My personal reminder list for daily tasks",
        reminders: {
            create: [
                {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Morning Meeting",
                    reminderAt: new Date(Date.now() + 8 * 60 * 60 * 1000), // 8 hours from now
                    index: 0,
                    user: { connect: { id: "user_placeholder_id" } },
                },
                {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Project Deadline",
                    reminderAt: new Date(Date.now() + 2 * 24 * 60 * 60 * 1000), // 2 days from now
                    index: 1,
                    user: { connect: { id: "user_placeholder_id" } },
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required user
            publicId: generatePublicId(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            publicId: 123, // Should be string
            name: 456, // Should be string
            description: true, // Should be string
        },
        invalidUserConnection: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "non-existent-user-id" } },
        },
    },
    edgeCases: {
        emptyList: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            name: "Empty List",
            description: "A list with no reminders",
        },
        manyReminders: {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: "user_placeholder_id" } },
            name: "Busy Schedule",
            description: "A list with many reminders",
            reminders: {
                create: Array.from({ length: 50 }, (_, i) => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: `Reminder ${i + 1}`,
                    reminderAt: new Date(Date.now() + (i + 1) * 60 * 60 * 1000), // Each hour apart
                    index: i,
                    user: { connect: { id: "user_placeholder_id" } },
                })),
            },
        },
    },
};

/**
 * Enhanced factory for creating reminder list database fixtures
 */
export class ReminderListDbFactory extends EnhancedDbFactory<Prisma.ReminderListCreateInput> {
    
    /**
     * Get the test fixtures for ReminderList model
     */
    protected getFixtures(): DbTestFixtures<Prisma.ReminderListCreateInput> {
        return reminderListDbFixtures;
    }

    /**
     * Get ReminderList-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: reminderDbIds.list1, // Duplicate ID
                    publicId: generatePublicId(),
                    user: { connect: { id: "user_placeholder_id" } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: "non-existent-user-id" } },
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    publicId: "", // Empty publicId violates constraint
                    user: { connect: { id: "user_placeholder_id" } },
                },
            },
            validation: {
                requiredFieldMissing: reminderListDbFixtures.invalid.missingRequired,
                invalidDataType: reminderListDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    user: { connect: { id: "user_placeholder_id" } },
                    name: "A".repeat(256), // Name too long
                },
            },
            businessLogic: {},
        };
    }

    /**
     * Add user association to a reminder list fixture
     */
    protected addUserAssociation(data: Prisma.ReminderListCreateInput, userId: string): Prisma.ReminderListCreateInput {
        return {
            ...data,
            user: { connect: { id: userId } },
        };
    }

    /**
     * Add reminders to a reminder list fixture
     */
    protected addReminders(data: Prisma.ReminderListCreateInput, reminders: Array<{ name: string; reminderAt: Date }>, userId: string): Prisma.ReminderListCreateInput {
        return {
            ...data,
            reminders: {
                create: reminders.map((r, index) => ({
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: r.name,
                    reminderAt: r.reminderAt,
                    index,
                    user: { connect: { id: userId } },
                })),
            },
        };
    }

    /**
     * ReminderList-specific validation
     */
    protected validateSpecific(data: Prisma.ReminderListCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to ReminderList
        if (!data.user) errors.push("Reminder list must be associated with a user");

        // Check business logic
        if (data.reminders?.create) {
            const reminders = Array.isArray(data.reminders.create) ? data.reminders.create : [data.reminders.create];
            
            if (reminders.length === 0) {
                warnings.push("Reminder list has no reminders");
            }

            // Check for duplicate indices
            const indices = reminders.map(r => r.index).filter(i => i !== undefined);
            const uniqueIndices = new Set(indices);
            if (indices.length !== uniqueIndices.size) {
                warnings.push("Reminder list has duplicate indices");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.ReminderListCreateInput>,
    ): Prisma.ReminderListCreateInput {
        const factory = new ReminderListDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addUserAssociation(data, userId);
    }

    static createWithReminders(
        userId: string,
        reminders: Array<{ name: string; reminderAt: Date }>,
        overrides?: Partial<Prisma.ReminderListCreateInput>,
    ): Prisma.ReminderListCreateInput {
        const factory = new ReminderListDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addUserAssociation(data, userId);
        return factory.addReminders(data, reminders, userId);
    }
}

/**
 * Enhanced helper to seed multiple test reminders with comprehensive options
 */
export async function seedReminders(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        withList?: boolean;
        withItems?: boolean;
        datesFrom?: Date;
    },
): Promise<BulkSeedResult<any>> {
    const reminderFactory = new ReminderDbFactory();
    const listFactory = new ReminderListDbFactory();
    const reminders = [];
    const count = options.count || 3;
    const startDate = options.datesFrom || new Date();
    let itemCount = 0;
    let completedCount = 0;
    let pastCount = 0;

    // Create list if requested
    let list = null;
    if (options.withList) {
        list = await prisma.reminderList.create({
            data: ReminderListDbFactory.createMinimal(options.userId, {
                name: "Test Reminder List",
                description: "Seeded reminder list for testing",
            }),
            include: { user: true },
        });
    }

    for (let i = 0; i < count; i++) {
        const reminderAt = new Date(startDate.getTime() + (i + 1) * 24 * 60 * 60 * 1000);
        
        let reminderData: Prisma.ReminderCreateInput;
        
        if (options.withItems) {
            reminderData = ReminderDbFactory.createWithItem(
                options.userId,
                {
                    name: `Task ${i + 1}`,
                    description: `Description for task ${i + 1}`,
                    dueDate: reminderAt,
                },
                {
                    name: `Reminder ${i + 1}`,
                    reminderAt,
                    description: `Reminder ${i + 1} description`,
                    ...(list && { reminderList: { connect: { id: list.id } } }),
                },
            );
            itemCount++;
        } else {
            reminderData = list
                ? ReminderDbFactory.createInList(
                    options.userId,
                    list.id,
                    {
                        name: `Reminder ${i + 1}`,
                        reminderAt,
                        description: `Description for reminder ${i + 1}`,
                    },
                )
                : ReminderDbFactory.createWithName(
                    options.userId,
                    `Reminder ${i + 1}`,
                    { 
                        reminderAt,
                        description: `Description for reminder ${i + 1}`,
                    },
                );
        }

        // Mark some as completed
        if (i % 3 === 0) {
            reminderData.isComplete = true;
            completedCount++;
        }

        // Count past reminders
        if (reminderAt < new Date()) {
            pastCount++;
        }

        const reminder = await prisma.reminder.create({
            data: reminderData,
            include: { 
                reminderItems: true,
                reminderList: true,
                user: true,
            },
        });
        reminders.push(reminder);
    }

    return {
        records: { reminders, list },
        summary: {
            total: reminders.length + (list ? 1 : 0),
            reminders: reminders.length,
            withItems: itemCount,
            completed: completedCount,
            past: pastCount,
            withList: list ? 1 : 0,
        },
    };
}
