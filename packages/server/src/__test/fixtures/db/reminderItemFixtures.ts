import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for ReminderItem model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const reminderItemDbIds = {
    item1: generatePK(),
    item2: generatePK(),
    item3: generatePK(),
    item4: generatePK(),
    item5: generatePK(),
};

/**
 * Minimal reminder item data for database creation
 */
export const minimalReminderItemDb: Omit<Prisma.reminder_itemCreateInput, "reminder"> = {
    id: reminderItemDbIds.item1,
    name: "Test Reminder Item",
    index: 0,
};

/**
 * Reminder item with description
 */
export const reminderItemWithDescriptionDb: Omit<Prisma.reminder_itemCreateInput, "reminder"> = {
    id: reminderItemDbIds.item2,
    name: "Reminder Item with Description",
    description: "This is a test reminder item with a description",
    index: 1,
};

/**
 * Reminder item with due date
 */
export const reminderItemWithDueDateDb: Omit<Prisma.reminder_itemCreateInput, "reminder"> = {
    id: reminderItemDbIds.item3,
    name: "Reminder Item with Due Date",
    description: "This item has a due date",
    dueDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 7 days from now
    index: 2,
};

/**
 * Completed reminder item
 */
export const completedReminderItemDb: Omit<Prisma.reminder_itemCreateInput, "reminder"> = {
    id: reminderItemDbIds.item4,
    name: "Completed Reminder Item",
    description: "This item has been completed",
    completedAt: new Date(),
    index: 3,
};

/**
 * Complete reminder item with all features
 */
export const completeReminderItemDb: Omit<Prisma.reminder_itemCreateInput, "reminder"> = {
    id: reminderItemDbIds.item5,
    name: "Complete Reminder Item",
    description: "This is a complete reminder item with all features",
    dueDate: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days from now
    index: 4,
};

/**
 * Factory for creating reminder item database fixtures with overrides
 */
export class ReminderItemDbFactory {
    static createMinimal(
        reminderId: string,
        overrides?: Partial<Prisma.reminder_itemCreateInput>,
    ): Prisma.reminder_itemCreateInput {
        return {
            ...minimalReminderItemDb,
            id: generatePK(),
            reminder: { connect: { id: reminderId } },
            ...overrides,
        };
    }

    static createWithDescription(
        reminderId: string,
        overrides?: Partial<Prisma.reminder_itemCreateInput>,
    ): Prisma.reminder_itemCreateInput {
        return {
            ...reminderItemWithDescriptionDb,
            id: generatePK(),
            reminder: { connect: { id: reminderId } },
            ...overrides,
        };
    }

    static createWithDueDate(
        reminderId: string,
        daysFromNow = 7,
        overrides?: Partial<Prisma.reminder_itemCreateInput>,
    ): Prisma.reminder_itemCreateInput {
        return {
            ...reminderItemWithDueDateDb,
            id: generatePK(),
            dueDate: new Date(Date.now() + daysFromNow * 24 * 60 * 60 * 1000),
            reminder: { connect: { id: reminderId } },
            ...overrides,
        };
    }

    static createCompleted(
        reminderId: string,
        overrides?: Partial<Prisma.reminder_itemCreateInput>,
    ): Prisma.reminder_itemCreateInput {
        return {
            ...completedReminderItemDb,
            id: generatePK(),
            completedAt: new Date(),
            reminder: { connect: { id: reminderId } },
            ...overrides,
        };
    }

    static createComplete(
        reminderId: string,
        overrides?: Partial<Prisma.reminder_itemCreateInput>,
    ): Prisma.reminder_itemCreateInput {
        return {
            ...completeReminderItemDb,
            id: generatePK(),
            reminder: { connect: { id: reminderId } },
            ...overrides,
        };
    }

    /**
     * Create multiple reminder items for a reminder
     */
    static createMultiple(
        reminderId: string,
        count: number,
        options?: {
            withDescriptions?: boolean;
            withDueDates?: boolean;
            someCompleted?: boolean;
        },
    ): Prisma.reminder_itemCreateInput[] {
        const items: Prisma.reminder_itemCreateInput[] = [];

        for (let i = 0; i < count; i++) {
            let item: Prisma.reminder_itemCreateInput;

            if (options?.someCompleted && i % 3 === 0) {
                // Every third item is completed
                item = this.createCompleted(reminderId, {
                    name: `Reminder Item ${i + 1}`,
                    index: i,
                });
            } else if (options?.withDueDates) {
                // Items with due dates
                item = this.createWithDueDate(reminderId, (i + 1) * 7, {
                    name: `Reminder Item ${i + 1}`,
                    description: options?.withDescriptions ? `Description for item ${i + 1}` : undefined,
                    index: i,
                });
            } else if (options?.withDescriptions) {
                // Items with descriptions
                item = this.createWithDescription(reminderId, {
                    name: `Reminder Item ${i + 1}`,
                    description: `Description for item ${i + 1}`,
                    index: i,
                });
            } else {
                // Minimal items
                item = this.createMinimal(reminderId, {
                    name: `Reminder Item ${i + 1}`,
                    index: i,
                });
            }

            items.push(item);
        }

        return items;
    }
}

/**
 * Helper to seed reminder items for testing
 */
export async function seedReminderItems(
    prisma: any,
    reminderId: string,
    options: {
        count?: number;
        withDescriptions?: boolean;
        withDueDates?: boolean;
        someCompleted?: boolean;
        items?: Array<{
            name: string;
            description?: string;
            dueDate?: Date;
            completed?: boolean;
        }>;
    } = {},
) {
    const reminderItems = [];

    if (options.items) {
        // Create specific items
        for (let i = 0; i < options.items.length; i++) {
            const itemData = options.items[i];
            const item = await prisma.reminder_item.create({
                data: {
                    id: generatePK(),
                    name: itemData.name,
                    description: itemData.description,
                    dueDate: itemData.dueDate,
                    completedAt: itemData.completed ? new Date() : null,
                    index: i,
                    reminder: { connect: { id: reminderId } },
                },
            });
            reminderItems.push(item);
        }
    } else {
        // Create items using factory
        const count = options.count || 3;
        const items = ReminderItemDbFactory.createMultiple(reminderId, count, {
            withDescriptions: options.withDescriptions,
            withDueDates: options.withDueDates,
            someCompleted: options.someCompleted,
        });

        for (const itemData of items) {
            const item = await prisma.reminder_item.create({ data: itemData });
            reminderItems.push(item);
        }
    }

    return reminderItems;
}

/**
 * Helper to create a checklist-style reminder with items
 */
export async function createChecklistReminder(
    prisma: any,
    reminderId: string,
    checklist: string[],
) {
    const items = [];
    
    for (let i = 0; i < checklist.length; i++) {
        const item = await prisma.reminder_item.create({
            data: ReminderItemDbFactory.createMinimal(reminderId, {
                name: checklist[i],
                index: i,
            }),
        });
        items.push(item);
    }

    return items;
}
