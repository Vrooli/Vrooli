import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ReminderList model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const reminderListDbIds = {
    list1: generatePK(),
    list2: generatePK(),
    list3: generatePK(),
    reminder1: generatePK(),
    reminder2: generatePK(),
    reminder3: generatePK(),
    reminder4: generatePK(),
    reminder5: generatePK(),
    reminderItem1: generatePK(),
    reminderItem2: generatePK(),
    reminderItem3: generatePK(),
};

/**
 * Minimal reminder list data for database creation
 */
export const minimalReminderListDb: Prisma.reminder_listCreateInput = {
    id: reminderListDbIds.list1,
    user: { connect: { id: "1" } }, // Will be overridden when used
};

/**
 * Reminder list with a single reminder
 */
export const reminderListWithSingleReminderDb: Prisma.reminder_listCreateInput = {
    id: reminderListDbIds.list2,
    user: { connect: { id: "1" } }, // Will be overridden when used
    reminders: {
        create: [{
            id: reminderListDbIds.reminder1,
            name: "Test Reminder",
            description: "A simple reminder",
            index: 0,
            isComplete: false,
        }],
    },
};

/**
 * Complete reminder list with multiple reminders and items
 */
export const completeReminderListDb: Prisma.reminder_listCreateInput = {
    id: reminderListDbIds.list3,
    user: { connect: { id: "1" } }, // Will be overridden when used
    reminders: {
        create: [
            {
                id: reminderListDbIds.reminder2,
                name: "Daily Tasks",
                description: "Things to do today",
                index: 0,
                isComplete: false,
                dueDate: new Date(Date.now() + 24 * 60 * 60 * 1000), // Tomorrow
                reminderItems: {
                    create: [
                        {
                            id: reminderListDbIds.reminderItem1,
                            name: "Morning routine",
                            description: "Exercise and breakfast",
                            index: 0,
                            isComplete: false,
                        },
                        {
                            id: reminderListDbIds.reminderItem2,
                            name: "Check emails",
                            description: null,
                            index: 1,
                            isComplete: true,
                            completedAt: new Date(),
                        },
                    ],
                },
            },
            {
                id: reminderListDbIds.reminder3,
                name: "Weekly Goals",
                description: "Important tasks for this week",
                index: 1,
                isComplete: false,
                dueDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // Next week
                reminderItems: {
                    create: [{
                        id: reminderListDbIds.reminderItem3,
                        name: "Complete project milestone",
                        description: "Finish the feature implementation",
                        index: 0,
                        isComplete: false,
                    }],
                },
            },
            {
                id: reminderListDbIds.reminder4,
                name: "Completed Task",
                description: "Already done",
                index: 2,
                isComplete: true,
                completedAt: new Date(Date.now() - 24 * 60 * 60 * 1000), // Yesterday
            },
        ],
    },
};

/**
 * Factory for creating reminder list database fixtures with overrides
 */
export class ReminderListDbFactory {
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.reminder_listCreateInput>
    ): Prisma.reminder_listCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createWithReminder(
        userId: string,
        reminderData?: Partial<Prisma.reminderCreateWithoutReminderListInput>,
        overrides?: Partial<Prisma.reminder_listCreateInput>
    ): Prisma.reminder_listCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            reminders: {
                create: [{
                    id: generatePK(),
                    name: "Test Reminder",
                    description: "A test reminder",
                    index: 0,
                    isComplete: false,
                    ...reminderData,
                }],
            },
            ...overrides,
        };
    }

    static createWithMultipleReminders(
        userId: string,
        reminders: Array<Partial<Prisma.reminderCreateWithoutReminderListInput>>,
        overrides?: Partial<Prisma.reminder_listCreateInput>
    ): Prisma.reminder_listCreateInput {
        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            reminders: {
                create: reminders.map((reminder, index) => ({
                    id: generatePK(),
                    name: `Reminder ${index + 1}`,
                    index,
                    isComplete: false,
                    ...reminder,
                })),
            },
            ...overrides,
        };
    }

    static createComplete(
        userId: string,
        overrides?: Partial<Prisma.reminder_listCreateInput>
    ): Prisma.reminder_listCreateInput {
        const baseData = { ...completeReminderListDb };
        baseData.user = { connect: { id: userId } };
        
        return {
            ...baseData,
            id: generatePK(),
            reminders: {
                create: (baseData.reminders?.create as any[])?.map(reminder => ({
                    ...reminder,
                    id: generatePK(),
                    reminderItems: reminder.reminderItems ? {
                        create: reminder.reminderItems.create.map((item: any) => ({
                            ...item,
                            id: generatePK(),
                        })),
                    } : undefined,
                })),
            },
            ...overrides,
        };
    }

    /**
     * Create a reminder list for testing time-based scenarios
     */
    static createWithDueDates(
        userId: string,
        overrides?: Partial<Prisma.reminder_listCreateInput>
    ): Prisma.reminder_listCreateInput {
        const now = new Date();
        const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
        const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
        const nextWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);

        return {
            id: generatePK(),
            user: { connect: { id: userId } },
            reminders: {
                create: [
                    {
                        id: generatePK(),
                        name: "Overdue Task",
                        description: "Should have been done yesterday",
                        index: 0,
                        isComplete: false,
                        dueDate: yesterday,
                    },
                    {
                        id: generatePK(),
                        name: "Due Tomorrow",
                        description: "Upcoming deadline",
                        index: 1,
                        isComplete: false,
                        dueDate: tomorrow,
                    },
                    {
                        id: generatePK(),
                        name: "Future Task",
                        description: "Not urgent",
                        index: 2,
                        isComplete: false,
                        dueDate: nextWeek,
                    },
                    {
                        id: generatePK(),
                        name: "No Due Date",
                        description: "Whenever you get to it",
                        index: 3,
                        isComplete: false,
                    },
                ],
            },
            ...overrides,
        };
    }
}

/**
 * Helper to seed reminder lists for testing
 */
export async function seedReminderLists(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        withReminders?: boolean;
        withItems?: boolean;
    }
) {
    const lists = [];
    const count = options.count || 1;

    for (let i = 0; i < count; i++) {
        let listData: Prisma.reminder_listCreateInput;

        if (options.withItems) {
            // Create complete list with reminders and items
            listData = ReminderListDbFactory.createComplete(options.userId);
        } else if (options.withReminders) {
            // Create list with basic reminders only
            listData = ReminderListDbFactory.createWithMultipleReminders(
                options.userId,
                [
                    { name: `Reminder ${i + 1}-1`, description: "First reminder" },
                    { name: `Reminder ${i + 1}-2`, description: "Second reminder" },
                ]
            );
        } else {
            // Create minimal list
            listData = ReminderListDbFactory.createMinimal(options.userId);
        }

        const list = await prisma.reminder_list.create({
            data: listData,
            include: {
                reminders: {
                    include: {
                        reminderItems: true,
                    },
                },
            },
        });

        lists.push(list);
    }

    return lists;
}

/**
 * Helper to create a reminder list with specific completion states
 */
export async function seedReminderListWithCompletionStates(
    prisma: any,
    userId: string
) {
    const listData = ReminderListDbFactory.createWithMultipleReminders(
        userId,
        [
            {
                name: "Completed reminder",
                isComplete: true,
                completedAt: new Date(Date.now() - 24 * 60 * 60 * 1000),
            },
            {
                name: "Incomplete reminder",
                isComplete: false,
            },
            {
                name: "Partially complete reminder",
                isComplete: false,
                reminderItems: {
                    create: [
                        {
                            id: generatePK(),
                            name: "Completed item",
                            index: 0,
                            isComplete: true,
                            completedAt: new Date(),
                        },
                        {
                            id: generatePK(),
                            name: "Incomplete item",
                            index: 1,
                            isComplete: false,
                        },
                    ],
                },
            },
        ]
    );

    return prisma.reminder_list.create({
        data: listData,
        include: {
            reminders: {
                include: {
                    reminderItems: true,
                },
            },
        },
    });
}