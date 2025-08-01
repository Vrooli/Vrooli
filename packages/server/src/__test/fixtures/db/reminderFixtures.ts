import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { BulkSeedResult, DbErrorScenarios, DbTestFixtures } from "./types.js";

/**
 * Database fixtures for Reminder model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Time constants to avoid magic numbers
const SECONDS_PER_MINUTE = 60;
const MS_PER_SECOND = 1000;
const MINUTES_PER_HOUR = 60;
const HOURS_PER_DAY = 24;
// const DAYS_PER_YEAR = 365; // Unused for now

const MINUTE_IN_MS = SECONDS_PER_MINUTE * MS_PER_SECOND;
const HOUR_IN_MS = MINUTES_PER_HOUR * MINUTE_IN_MS;
const DAY_IN_MS = HOURS_PER_DAY * HOUR_IN_MS;
// const YEAR_IN_MS = DAYS_PER_YEAR * DAY_IN_MS; // Unused for now

const MAX_DESCRIPTION_LENGTH = 2048;
const MANY_ITEMS_COUNT = 20;
const MANY_REMINDERS_COUNT = 50;
const EVERY_NTH_COMPLETED = 3;
const DEFAULT_SEED_COUNT = 3;
const WORKDAY_START_HOURS = 8;
const LONG_TERM_DAYS = 400;

// Factory functions to generate IDs when needed
export const reminderDbIds = {
    reminder1: () => generatePK(),
    reminder2: () => generatePK(),
    reminder3: () => generatePK(),
    list1: () => generatePK(),
    list2: () => generatePK(),
    item1: () => generatePK(),
    item2: () => generatePK(),
    item3: () => generatePK(),
};

/**
 * Enhanced test fixtures for Reminder model following standard structure
 */
export const reminderDbFixtures: DbTestFixtures<Prisma.reminderCreateInput> = {
    minimal: {
        id: generatePK(),
        name: "Minimal Reminder",
        description: null,
        dueDate: new Date(Date.now() + DAY_IN_MS), // 1 day from now
        index: 0,
        reminderList: { connect: { id: generatePK() } },
    },
    complete: {
        id: generatePK(),
        name: "Complete Reminder",
        description: "A comprehensive reminder with all features",
        dueDate: new Date(Date.now() + 3 * DAY_IN_MS), // 3 days from now
        index: 0,
        completedAt: null,
        reminderList: { connect: { id: generatePK() } },
        reminderItems: {
            create: [
                {
                    id: generatePK(),
                    name: "Task 1",
                    description: "Complete the first task",
                    dueDate: new Date(Date.now() + DAY_IN_MS),
                    index: 0,
                    completedAt: null,
                },
                {
                    id: generatePK(),
                    name: "Task 2",
                    description: "Complete the second task",
                    dueDate: new Date(Date.now() + 2 * DAY_IN_MS),
                    index: 1,
                    completedAt: new Date(),
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required name and reminderList
            id: generatePK(),
            index: 0,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake", // Should be bigint
            name: 123, // Should be string
            description: true, // Should be string or null
            dueDate: "not-a-date", // Should be Date
            index: "not-a-number", // Should be number
            completedAt: "yes", // Should be Date or null
        },
        invalidListConnection: {
            id: generatePK(),
            name: "Invalid List Connection",
            index: 0,
            reminderList: { connect: { id: "non-existent-list-id" } },
        },
        duplicateIndex: {
            id: generatePK(),
            name: "Duplicate Index",
            index: 0, // Same index as another reminder in the list
            reminderList: { connect: { id: generatePK() } },
        },
    },
    edgeCases: {
        pastDueDate: {
            id: generatePK(),
            name: "Past Due Date",
            description: "This reminder has a due date in the past",
            dueDate: new Date(Date.now() - DAY_IN_MS), // 1 day ago
            index: 0,
            completedAt: null,
            reminderList: { connect: { id: generatePK() } },
        },
        completedReminder: {
            id: generatePK(),
            name: "Completed Reminder",
            description: "This reminder has been marked as complete",
            dueDate: new Date(Date.now() + DAY_IN_MS),
            index: 0,
            completedAt: new Date(),
            reminderList: { connect: { id: generatePK() } },
        },
        manyItemsReminder: {
            id: generatePK(),
            name: "Reminder with Many Items",
            description: "A reminder with many sub-items",
            dueDate: new Date(Date.now() + DAY_IN_MS),
            index: 0,
            reminderList: { connect: { id: generatePK() } },
            reminderItems: {
                create: Array.from({ length: MANY_ITEMS_COUNT }, (_, i) => ({
                    id: generatePK(),
                    name: `Task ${i + 1}`,
                    description: `Description for task ${i + 1}`,
                    index: i,
                    completedAt: i % EVERY_NTH_COMPLETED === 0 ? new Date() : null, // Every third item completed
                })),
            },
        },
        urgentReminder: {
            id: generatePK(),
            name: "Urgent Reminder",
            description: "This reminder is due very soon",
            dueDate: new Date(Date.now() + 2 * HOUR_IN_MS), // 2 hours from now
            index: 0,
            completedAt: null,
            reminderList: { connect: { id: generatePK() } },
        },
        longTermReminder: {
            id: generatePK(),
            name: "Long-term Reminder",
            description: "This reminder is set for the distant future",
            dueDate: new Date(Date.now() + LONG_TERM_DAYS * DAY_IN_MS), // 400 days from now
            index: 0,
            completedAt: null,
            reminderList: { connect: { id: generatePK() } },
        },
    },
};

/**
 * Enhanced factory for creating reminder database fixtures
 */
export class ReminderDbFactory extends EnhancedDbFactory<Prisma.reminderCreateInput> {

    /**
     * Get the test fixtures for Reminder model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reminderCreateInput> {
        return reminderDbFixtures;
    }

    /**
     * Get Reminder-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: reminderDbIds.reminder1(), // Duplicate ID
                    name: "Duplicate ID Reminder",
                    index: 0,
                    reminderList: { connect: { id: generatePK() } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    name: "Foreign Key Violation",
                    index: 0,
                    reminderList: { connect: { id: generatePK() } }, // Non-existent list
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    name: "", // Empty name violates constraint
                    index: 0,
                    reminderList: { connect: { id: generatePK() } },
                },
            },
            validation: {
                requiredFieldMissing: reminderDbFixtures.invalid.missingRequired,
                invalidDataType: reminderDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    name: "Out of Range",
                    description: "A".repeat(MAX_DESCRIPTION_LENGTH + 1), // Too long (max 2048)
                    dueDate: new Date("1900-01-01"), // Too far in past
                    index: -1, // Negative index
                    reminderList: { connect: { id: generatePK() } },
                },
            },
            businessLogic: {
                completedWithoutDate: {
                    id: generatePK(),
                    name: "Completed Without Date",
                    index: 0,
                    completedAt: new Date(), // Completed but no completedAt should be null if not complete
                    reminderList: { connect: { id: generatePK() } },
                },
                completedWithIncompleteItems: {
                    id: generatePK(),
                    name: "Completed With Incomplete Items",
                    index: 0,
                    completedAt: new Date(), // Reminder complete
                    reminderList: { connect: { id: generatePK() } },
                    reminderItems: {
                        create: [{
                            id: generatePK(),
                            name: "Incomplete Task",
                            index: 0,
                            completedAt: null, // But has incomplete items
                        }],
                    },
                },
            },
        };
    }

    /**
     * Add list association to a reminder fixture
     */
    protected addListAssociation(data: Prisma.reminderCreateInput, listId: bigint): Prisma.reminderCreateInput {
        return {
            ...data,
            reminderList: { connect: { id: listId } },
        };
    }

    /**
     * Override list association in a reminder fixture
     */
    protected overrideListAssociation(data: Prisma.reminderCreateInput, listId: bigint): Prisma.reminderCreateInput {
        return {
            ...data,
            reminderList: { connect: { id: listId } },
        };
    }

    /**
     * Add items to a reminder fixture
     */
    protected addItems(data: Prisma.reminderCreateInput, items: Array<{
        name: string;
        description?: string;
        dueDate?: Date;
        index?: number;
        isComplete?: boolean;
    }>): Prisma.reminderCreateInput {
        return {
            ...data,
            reminderItems: {
                create: items.map((item, i) => ({
                    id: generatePK(),
                    name: item.name,
                    description: item.description,
                    dueDate: item.dueDate,
                    index: item.index ?? i,
                    completedAt: item.isComplete ? new Date() : null,
                })),
            },
        };
    }

    /**
     * Reminder-specific validation
     */
    protected validateSpecific(data: Prisma.reminderCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Reminder
        if (!data.name) errors.push("Reminder name is required");
        if (!data.reminderList) errors.push("Reminder must be associated with a list");
        if (data.index === undefined) errors.push("Reminder index is required");

        // Check business logic
        if (data.dueDate && data.dueDate < new Date()) {
            warnings.push("Due date is in the past");
        }

        if (data.completedAt && !data.dueDate) {
            warnings.push("Reminder is marked complete but has no due date");
        }

        if (data.index !== undefined && data.index < 0) {
            errors.push("Reminder index cannot be negative");
        }

        // Check items logic
        if (data.reminderItems?.create) {
            const items = Array.isArray(data.reminderItems.create) ? data.reminderItems.create : [data.reminderItems.create];
            const allItemsComplete = items.every(item => item.completedAt !== null && item.completedAt !== undefined);

            if (data.completedAt && !allItemsComplete) {
                warnings.push("Reminder is marked complete but has incomplete items");
            }

            if (!data.completedAt && allItemsComplete && items.length > 0) {
                warnings.push("All items are complete but reminder is not marked as complete");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        listId: bigint,
        overrides?: Partial<Prisma.reminderCreateInput>,
    ): Prisma.reminderCreateInput {
        const factory = new ReminderDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addListAssociation(data, listId);
    }

    static createWithName(
        listId: bigint,
        name: string,
        overrides?: Partial<Prisma.reminderCreateInput>,
    ): Prisma.reminderCreateInput {
        const factory = new ReminderDbFactory();
        const data = factory.createMinimal({ name, ...overrides });
        return factory.addListAssociation(data, listId);
    }

    static createInList(
        listId: bigint,
        overrides?: Partial<Prisma.reminderCreateInput>,
    ): Prisma.reminderCreateInput {
        const factory = new ReminderDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addListAssociation(data, listId);
    }

    static createWithItem(
        listId: bigint,
        itemData: Partial<Prisma.reminder_itemCreateWithoutReminderInput>,
        overrides?: Partial<Prisma.reminderCreateInput>,
    ): Prisma.reminderCreateInput {
        const factory = new ReminderDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addListAssociation(data, listId);
        return factory.addItems(data, [{
            name: itemData.name || "Reminder Item",
            description: itemData.description,
            dueDate: itemData.dueDate instanceof Date ? itemData.dueDate : undefined,
            index: itemData.index || 0,
            isComplete: Boolean(itemData.completedAt),
        }]);
    }
}

/**
 * Enhanced test fixtures for ReminderList model
 */
export const reminderListDbFixtures: DbTestFixtures<Prisma.reminder_listCreateInput> = {
    minimal: {
        id: generatePK(),
        user: { connect: { id: generatePK() } },
    },
    complete: {
        id: generatePK(),
        user: { connect: { id: generatePK() } },
        reminders: {
            create: [
                {
                    id: generatePK(),
                    name: "Morning Meeting",
                    description: "Daily standup meeting",
                    dueDate: new Date(Date.now() + WORKDAY_START_HOURS * HOUR_IN_MS), // 8 hours from now
                    index: 0,
                },
                {
                    id: generatePK(),
                    name: "Project Deadline",
                    description: "Final project submission",
                    dueDate: new Date(Date.now() + 2 * DAY_IN_MS), // 2 days from now
                    index: 1,
                },
            ],
        },
    },
    invalid: {
        missingRequired: {
            // Missing required user
            id: generatePK(),
        },
        invalidTypes: {
            id: "not-a-valid-snowflake", // Should be bigint
            user: "not-a-connection", // Should be connection object
        },
        invalidUserConnection: {
            id: generatePK(),
            user: { connect: { id: generatePK() } }, // Non-existent user
        },
    },
    edgeCases: {
        emptyList: {
            id: generatePK(),
            user: { connect: { id: generatePK() } },
        },
        manyReminders: {
            id: generatePK(),
            user: { connect: { id: generatePK() } },
            reminders: {
                create: Array.from({ length: MANY_REMINDERS_COUNT }, (_, i) => ({
                    id: generatePK(),
                    name: `Reminder ${i + 1}`,
                    description: `Description for reminder ${i + 1}`,
                    dueDate: new Date(Date.now() + (i + 1) * HOUR_IN_MS), // Each hour apart
                    index: i,
                })),
            },
        },
    },
};

/**
 * Enhanced factory for creating reminder list database fixtures
 */
export class ReminderListDbFactory extends EnhancedDbFactory<Prisma.reminder_listCreateInput> {

    /**
     * Get the test fixtures for ReminderList model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reminder_listCreateInput> {
        return reminderListDbFixtures;
    }

    /**
     * Get ReminderList-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: reminderDbIds.list1(), // Duplicate ID
                    user: { connect: { id: generatePK() } },
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    user: { connect: { id: generatePK() } }, // Non-existent user
                },
                checkConstraintViolation: {
                    id: BigInt("0"), // Invalid ID (should be positive)
                    user: { connect: { id: generatePK() } },
                },
            },
            validation: {
                requiredFieldMissing: reminderListDbFixtures.invalid.missingRequired,
                invalidDataType: reminderListDbFixtures.invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    user: { connect: { id: generatePK() } },
                    // No specific out of range scenario for reminder_list
                },
            },
            businessLogic: {},
        };
    }

    /**
     * Add user association to a reminder list fixture
     */
    protected addUserAssociation(data: Prisma.reminder_listCreateInput, userId: bigint): Prisma.reminder_listCreateInput {
        return {
            ...data,
            user: { connect: { id: userId } },
        };
    }

    /**
     * Add reminders to a reminder list fixture
     */
    protected addReminders(data: Prisma.reminder_listCreateInput, reminders: Array<{ name: string; dueDate: Date }>): Prisma.reminder_listCreateInput {
        return {
            ...data,
            reminders: {
                create: reminders.map((r, index) => ({
                    id: generatePK(),
                    name: r.name,
                    description: null,
                    dueDate: r.dueDate,
                    index,
                })),
            },
        };
    }

    /**
     * ReminderList-specific validation
     */
    protected validateSpecific(data: Prisma.reminder_listCreateInput): { errors: string[]; warnings: string[] } {
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
        userId: bigint,
        overrides?: Partial<Prisma.reminder_listCreateInput>,
    ): Prisma.reminder_listCreateInput {
        const factory = new ReminderListDbFactory();
        const data = factory.createMinimal(overrides);
        return factory.addUserAssociation(data, userId);
    }

    static createWithReminders(
        userId: bigint,
        reminders: Array<{ name: string; dueDate: Date }>,
        overrides?: Partial<Prisma.reminder_listCreateInput>,
    ): Prisma.reminder_listCreateInput {
        const factory = new ReminderListDbFactory();
        let data = factory.createMinimal(overrides);
        data = factory.addUserAssociation(data, userId);
        return factory.addReminders(data, reminders);
    }
}

/**
 * Enhanced helper to seed multiple test reminders with comprehensive options
 */
interface PrismaClient {
    reminder_list: {
        create: (args: { data: Prisma.reminder_listCreateInput; include?: Record<string, boolean> }) => Promise<{ id: bigint; user: unknown }>;
    };
    reminder: {
        create: (args: { data: Prisma.reminderCreateInput; include?: Record<string, boolean> }) => Promise<unknown>;
    };
}

export async function seedReminders(
    prisma: PrismaClient,
    options: {
        userId: string;
        count?: number;
        withList?: boolean;
        withItems?: boolean;
        datesFrom?: Date;
    },
): Promise<BulkSeedResult<unknown>> {
    // const reminderFactory = new ReminderDbFactory();
    // const listFactory = new ReminderListDbFactory();
    const reminders = [];
    const count = options.count || DEFAULT_SEED_COUNT;
    const startDate = options.datesFrom || new Date();
    let itemCount = 0;
    let completedCount = 0;
    let pastCount = 0;

    // Create list if requested
    let list: { id: bigint; user: unknown } | null = null;
    if (options.withList) {
        list = await prisma.reminder_list.create({
            data: ReminderListDbFactory.createMinimal(BigInt(options.userId), {
                // reminder_list doesn't have name or description fields
            }),
            include: { user: true },
        });
    }

    for (let i = 0; i < count; i++) {
        const dueDate = new Date(startDate.getTime() + (i + 1) * DAY_IN_MS);

        let reminderData: Prisma.reminderCreateInput;

        if (options.withItems) {
            reminderData = ReminderDbFactory.createWithItem(
                list ? list.id : generatePK(),
                {
                    name: `Task ${i + 1}`,
                    description: `Description for task ${i + 1}`,
                    dueDate,
                },
                {
                    name: `Reminder ${i + 1}`,
                    dueDate,
                    description: `Reminder ${i + 1} description`,
                },
            );
            itemCount++;
        } else {
            reminderData = list
                ? ReminderDbFactory.createInList(
                    list.id,
                    {
                        name: `Reminder ${i + 1}`,
                        dueDate,
                        description: `Description for reminder ${i + 1}`,
                    },
                )
                : ReminderDbFactory.createWithName(
                    generatePK(), // Create a new list ID if no list exists
                    `Reminder ${i + 1}`,
                    {
                        dueDate,
                        description: `Description for reminder ${i + 1}`,
                    },
                );
        }

        // Mark some as completed
        if (i % EVERY_NTH_COMPLETED === 0) {
            reminderData.completedAt = new Date();
            completedCount++;
        }

        // Count past reminders
        if (dueDate < new Date()) {
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

    // Include list in records if created
    const allRecords = list ? [list, ...reminders] : reminders;

    return {
        records: allRecords,
        summary: {
            total: allRecords.length,
            withAuth: 0, // No auth in reminders
            bots: 0, // No bots in reminders
            teams: 0, // No teams in reminders
            reminders: reminders.length,
            lists: list ? 1 : 0,
            withItems: itemCount,
            completed: completedCount,
            past: pastCount,
        },
    };
}
