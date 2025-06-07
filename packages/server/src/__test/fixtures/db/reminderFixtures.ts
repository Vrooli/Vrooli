import { generatePK, generatePublicId } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Reminder model - used for seeding test data
 */

export class ReminderDbFactory {
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.ReminderCreateInput>
    ): Prisma.ReminderCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: userId } },
            reminderAt: new Date(Date.now() + 24 * 60 * 60 * 1000), // 1 day from now
            ...overrides,
        };
    }

    static createWithName(
        userId: string,
        name: string,
        overrides?: Partial<Prisma.ReminderCreateInput>
    ): Prisma.ReminderCreateInput {
        return {
            ...this.createMinimal(userId, overrides),
            name,
        };
    }

    static createInList(
        userId: string,
        listId: string,
        overrides?: Partial<Prisma.ReminderCreateInput>
    ): Prisma.ReminderCreateInput {
        return {
            ...this.createMinimal(userId, overrides),
            reminderList: { connect: { id: listId } },
        };
    }

    static createWithItem(
        userId: string,
        itemData: Partial<Prisma.ReminderItemCreateWithoutReminderInput>,
        overrides?: Partial<Prisma.ReminderCreateInput>
    ): Prisma.ReminderCreateInput {
        return {
            ...this.createMinimal(userId, overrides),
            reminderItems: {
                create: [{
                    id: generatePK(),
                    name: itemData.name || "Reminder Item",
                    description: itemData.description,
                    dueDate: itemData.dueDate,
                    index: itemData.index || 0,
                    isComplete: itemData.isComplete || false,
                }],
            },
        };
    }
}

/**
 * Database fixtures for ReminderList model
 */
export class ReminderListDbFactory {
    static createMinimal(
        userId: string,
        overrides?: Partial<Prisma.ReminderListCreateInput>
    ): Prisma.ReminderListCreateInput {
        return {
            id: generatePK(),
            publicId: generatePublicId(),
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    static createWithReminders(
        userId: string,
        reminders: Array<{ name: string; reminderAt: Date }>,
        overrides?: Partial<Prisma.ReminderListCreateInput>
    ): Prisma.ReminderListCreateInput {
        return {
            ...this.createMinimal(userId, overrides),
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
}

/**
 * Helper to seed reminders for testing
 */
export async function seedReminders(
    prisma: any,
    options: {
        userId: string;
        count?: number;
        withList?: boolean;
        withItems?: boolean;
        datesFrom?: Date;
    }
) {
    const reminders = [];
    const count = options.count || 3;
    const startDate = options.datesFrom || new Date();

    // Create list if requested
    let list = null;
    if (options.withList) {
        list = await prisma.reminderList.create({
            data: ReminderListDbFactory.createMinimal(options.userId),
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
                    ...(list && { reminderList: { connect: { id: list.id } } }),
                }
            );
        } else {
            reminderData = list
                ? ReminderDbFactory.createInList(
                    options.userId,
                    list.id,
                    {
                        name: `Reminder ${i + 1}`,
                        reminderAt,
                    }
                )
                : ReminderDbFactory.createWithName(
                    options.userId,
                    `Reminder ${i + 1}`,
                    { reminderAt }
                );
        }

        const reminder = await prisma.reminder.create({
            data: reminderData,
            include: { reminderItems: true },
        });
        reminders.push(reminder);
    }

    return { reminders, list };
}