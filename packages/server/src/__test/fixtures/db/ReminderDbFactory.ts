import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface ReminderRelationConfig extends RelationConfig {
    withList?: { reminderListId: string };
    withItems?: boolean | number;
}

interface ReminderListRelationConfig extends RelationConfig {
    withUser?: { userId: string };
    withReminders?: boolean | number;
}

/**
 * Enhanced database fixture factory for Reminder model
 * Provides comprehensive testing capabilities for reminders with items
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for reminder items
 * - Due date management
 * - Completion tracking
 * - Embedding support for AI features
 * - Index-based ordering
 * - Predefined test scenarios
 */
export class ReminderDbFactory extends EnhancedDatabaseFactory<
    Prisma.reminder,
    Prisma.reminderCreateInput,
    Prisma.reminderInclude,
    Prisma.reminderUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Reminder', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.reminder;
    }

    /**
     * Get complete test fixtures for Reminder model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reminderCreateInput, Prisma.reminderUpdateInput> {
        const now = new Date();
        const tomorrow = new Date(now.getTime() + 24 * 60 * 60 * 1000);
        const nextWeek = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
        const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);

        return {
            minimal: {
                id: generatePK().toString(),
                name: "Test Reminder",
                index: 0,
                reminderList: {
                    connect: { id: "reminder_list_id" }
                },
            },
            complete: {
                id: generatePK().toString(),
                name: "Complete Test Reminder",
                description: "A comprehensive reminder with all features",
                index: 0,
                dueDate: tomorrow,
                reminderList: {
                    connect: { id: "reminder_list_id" }
                },
                reminderItems: {
                    create: [
                        {
                            id: generatePK().toString(),
                            name: "Item 1",
                            description: "First task to complete",
                            index: 0,
                            dueDate: tomorrow,
                        },
                        {
                            id: generatePK().toString(),
                            name: "Item 2",
                            description: "Second task to complete",
                            index: 1,
                            dueDate: nextWeek,
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, name, index, reminderList
                    description: "Invalid reminder",
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    name: null, // Should be string
                    index: "0", // Should be number
                    dueDate: "tomorrow", // Should be Date
                    reminderListId: 123, // Should be string
                },
                negativeIndex: {
                    id: generatePK().toString(),
                    name: "Negative Index",
                    index: -1, // Should be non-negative
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
                tooLongName: {
                    id: generatePK().toString(),
                    name: "a".repeat(129), // Exceeds 128 character limit
                    index: 0,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
                tooLongDescription: {
                    id: generatePK().toString(),
                    name: "Long Description",
                    description: "a".repeat(2049), // Exceeds 2048 character limit
                    index: 0,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
            },
            edgeCases: {
                completedReminder: {
                    id: generatePK().toString(),
                    name: "Completed Reminder",
                    description: "This reminder has been completed",
                    index: 0,
                    dueDate: yesterday,
                    completedAt: now,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
                overdueReminder: {
                    id: generatePK().toString(),
                    name: "Overdue Reminder",
                    description: "This reminder is past its due date",
                    index: 0,
                    dueDate: yesterday,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
                reminderWithManyItems: {
                    id: generatePK().toString(),
                    name: "Multi-Item Reminder",
                    description: "Reminder with many sub-items",
                    index: 0,
                    dueDate: nextWeek,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                    reminderItems: {
                        create: Array.from({ length: 20 }, (_, i) => ({
                            id: generatePK().toString(),
                            name: `Task ${i + 1}`,
                            description: `Description for task ${i + 1}`,
                            index: i,
                            dueDate: new Date(tomorrow.getTime() + i * 24 * 60 * 60 * 1000),
                        })),
                    },
                },
                reminderWithEmbedding: {
                    id: generatePK().toString(),
                    name: "AI-Enhanced Reminder",
                    description: "Reminder with embedding for AI features",
                    index: 0,
                    dueDate: tomorrow,
                    embeddingExpiredAt: new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000),
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
                partiallyCompletedReminder: {
                    id: generatePK().toString(),
                    name: "Partially Complete",
                    description: "Some items are completed",
                    index: 0,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                    reminderItems: {
                        create: [
                            {
                                id: generatePK().toString(),
                                name: "Completed Task",
                                index: 0,
                                completedAt: now,
                            },
                            {
                                id: generatePK().toString(),
                                name: "Pending Task",
                                index: 1,
                            },
                        ],
                    },
                },
                highIndexReminder: {
                    id: generatePK().toString(),
                    name: "High Priority",
                    description: "High index indicates lower display priority",
                    index: 999,
                    reminderList: {
                        connect: { id: "reminder_list_id" }
                    },
                },
            },
            updates: {
                minimal: {
                    name: "Updated Reminder",
                },
                complete: {
                    name: "Completely Updated Reminder",
                    description: "Updated description",
                    dueDate: nextWeek,
                    completedAt: now,
                    index: 5,
                    reminderItems: {
                        create: [{
                            id: generatePK().toString(),
                            name: "New Task",
                            index: 0,
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.reminderCreateInput>): Prisma.reminderCreateInput {
        return {
            id: generatePK().toString(),
            name: `Reminder_${nanoid(8)}`,
            index: 0,
            reminderList: {
                connect: { id: "default_reminder_list_id" }
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.reminderCreateInput>): Prisma.reminderCreateInput {
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000);
        
        return {
            id: generatePK().toString(),
            name: `Complete_Reminder_${nanoid(8)}`,
            description: "A complete reminder with items",
            index: 0,
            dueDate: tomorrow,
            reminderList: {
                connect: { id: "default_reminder_list_id" }
            },
            reminderItems: {
                create: [
                    {
                        id: generatePK().toString(),
                        name: "First Item",
                        description: "First item to complete",
                        index: 0,
                        dueDate: tomorrow,
                    },
                    {
                        id: generatePK().toString(),
                        name: "Second Item",
                        description: "Second item to complete",
                        index: 1,
                        dueDate: new Date(tomorrow.getTime() + 24 * 60 * 60 * 1000),
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            dailyReminder: {
                name: "dailyReminder",
                description: "Daily reminder with recurring tasks",
                config: {
                    overrides: {
                        name: "Daily Tasks",
                        description: "Tasks to complete today",
                        dueDate: new Date(new Date().setHours(23, 59, 59, 999)),
                    },
                    withItems: 3,
                },
            },
            projectReminder: {
                name: "projectReminder",
                description: "Project milestone reminder",
                config: {
                    overrides: {
                        name: "Project Milestone",
                        description: "Important project deadline",
                        dueDate: new Date(Date.now() + 14 * 24 * 60 * 60 * 1000),
                    },
                    withItems: 10,
                },
            },
            shoppingList: {
                name: "shoppingList",
                description: "Shopping list reminder",
                config: {
                    overrides: {
                        name: "Shopping List",
                        description: "Items to buy at the store",
                    },
                    withItems: 5,
                },
            },
            completedReminder: {
                name: "completedReminder",
                description: "Completed reminder with all items done",
                config: {
                    overrides: {
                        name: "Completed Tasks",
                        description: "All tasks have been completed",
                        completedAt: new Date(),
                    },
                    withItems: 3,
                },
            },
            urgentReminder: {
                name: "urgentReminder",
                description: "Urgent reminder due soon",
                config: {
                    overrides: {
                        name: "URGENT: Action Required",
                        description: "This needs immediate attention",
                        dueDate: new Date(Date.now() + 60 * 60 * 1000), // 1 hour
                        index: 0, // High priority
                    },
                    withItems: 2,
                },
            },
        };
    }

    /**
     * Create specific reminder types
     */
    async createSimpleReminder(listId: string, name: string): Promise<Prisma.reminder> {
        return await this.createMinimal({
            name,
            reminderList: { connect: { id: listId } },
        });
    }

    async createReminderWithItems(listId: string, itemCount: number = 3): Promise<Prisma.reminder> {
        return await this.createWithRelations({
            overrides: {
                reminderList: { connect: { id: listId } },
            },
            withItems: itemCount,
        });
    }

    async createCompletedReminder(listId: string): Promise<Prisma.reminder> {
        return await this.createMinimal({
            completedAt: new Date(),
            reminderList: { connect: { id: listId } },
        });
    }

    async createOverdueReminder(listId: string): Promise<Prisma.reminder> {
        return await this.createMinimal({
            dueDate: new Date(Date.now() - 24 * 60 * 60 * 1000),
            reminderList: { connect: { id: listId } },
        });
    }

    protected getDefaultInclude(): Prisma.reminderInclude {
        return {
            reminderList: true,
            reminderItems: {
                orderBy: { index: 'asc' },
            },
            _count: {
                select: {
                    reminderItems: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.reminderCreateInput,
        config: ReminderRelationConfig,
        tx: any
    ): Promise<Prisma.reminderCreateInput> {
        let data = { ...baseData };

        // Handle reminder list relationship
        if (config.withList) {
            data.reminderList = {
                connect: { id: config.withList.reminderListId }
            };
        }

        // Handle reminder items
        if (config.withItems) {
            const itemCount = typeof config.withItems === 'number' ? config.withItems : 1;
            const baseDueDate = (data.dueDate as Date) || new Date(Date.now() + 24 * 60 * 60 * 1000);
            
            data.reminderItems = {
                create: Array.from({ length: itemCount }, (_, i) => ({
                    id: generatePK().toString(),
                    name: `Item ${i + 1}`,
                    description: `Task item ${i + 1}`,
                    index: i,
                    dueDate: new Date(baseDueDate.getTime() + i * 24 * 60 * 60 * 1000),
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.reminder): Promise<string[]> {
        const violations: string[] = [];
        
        // Check name length
        if (record.name.length > 128) {
            violations.push('Reminder name exceeds 128 character limit');
        }

        // Check description length
        if (record.description && record.description.length > 2048) {
            violations.push('Reminder description exceeds 2048 character limit');
        }

        // Check index
        if (record.index < 0) {
            violations.push('Reminder index must be non-negative');
        }

        // Check completion logic
        if (record.completedAt && record.dueDate && record.completedAt < record.dueDate) {
            // This is actually valid - completing before due date
        }

        // Check reminder items
        if (record.reminderItems) {
            const indexSet = new Set<number>();
            for (const item of record.reminderItems) {
                if (indexSet.has(item.index)) {
                    violations.push('Reminder item indexes must be unique within a reminder');
                }
                indexSet.add(item.index);

                if (item.name.length > 128) {
                    violations.push('Reminder item name exceeds 128 character limit');
                }

                if (item.description && item.description.length > 2048) {
                    violations.push('Reminder item description exceeds 2048 character limit');
                }
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            reminderItems: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.reminder,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete reminder items
        if (shouldDelete('reminderItems') && record.reminderItems?.length) {
            await tx.reminder_item.deleteMany({
                where: { reminderId: record.id },
            });
        }
    }

    /**
     * Create a reminder with specific items
     */
    async createReminderWithSpecificItems(
        listId: string,
        items: Array<{ name: string; description?: string; dueDate?: Date }>
    ): Promise<Prisma.reminder> {
        return await this.createComplete({
            reminderList: { connect: { id: listId } },
            reminderItems: {
                create: items.map((item, i) => ({
                    id: generatePK().toString(),
                    name: item.name,
                    description: item.description,
                    dueDate: item.dueDate,
                    index: i,
                })),
            },
        });
    }

    /**
     * Mark reminder items as completed
     */
    async markItemsCompleted(reminderId: string, itemIds: string[]): Promise<void> {
        await this.prisma.reminder_item.updateMany({
            where: {
                id: { in: itemIds },
                reminderId,
            },
            data: {
                completedAt: new Date(),
            },
        });
    }
}

/**
 * Enhanced database fixture factory for ReminderList model
 * Provides comprehensive testing capabilities for reminder lists
 * 
 * Features:
 * - Type-safe Prisma integration
 * - User association
 * - Multiple reminders support
 * - Predefined test scenarios
 */
export class ReminderListDbFactory extends EnhancedDatabaseFactory<
    Prisma.reminder_list,
    Prisma.reminder_listCreateInput,
    Prisma.reminder_listInclude,
    Prisma.reminder_listUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('ReminderList', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.reminder_list;
    }

    /**
     * Get complete test fixtures for ReminderList model
     */
    protected getFixtures(): DbTestFixtures<Prisma.reminder_listCreateInput, Prisma.reminder_listUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                user: {
                    connect: { id: "user_id" }
                },
            },
            complete: {
                id: generatePK().toString(),
                user: {
                    connect: { id: "user_id" }
                },
                reminders: {
                    create: [
                        {
                            id: generatePK().toString(),
                            name: "First Reminder",
                            description: "First reminder in the list",
                            index: 0,
                        },
                        {
                            id: generatePK().toString(),
                            name: "Second Reminder",
                            description: "Second reminder in the list",
                            index: 1,
                            dueDate: new Date(Date.now() + 24 * 60 * 60 * 1000),
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id and user
                    createdAt: new Date(),
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    userId: "string-not-bigint", // Should be BigInt
                },
            },
            edgeCases: {
                emptyList: {
                    id: generatePK().toString(),
                    user: {
                        connect: { id: "user_id" }
                    },
                },
                largeList: {
                    id: generatePK().toString(),
                    user: {
                        connect: { id: "user_id" }
                    },
                    reminders: {
                        create: Array.from({ length: 50 }, (_, i) => ({
                            id: generatePK().toString(),
                            name: `Reminder ${i + 1}`,
                            index: i,
                        })),
                    },
                },
                allCompletedList: {
                    id: generatePK().toString(),
                    user: {
                        connect: { id: "user_id" }
                    },
                    reminders: {
                        create: Array.from({ length: 5 }, (_, i) => ({
                            id: generatePK().toString(),
                            name: `Completed Reminder ${i + 1}`,
                            index: i,
                            completedAt: new Date(),
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    updatedAt: new Date(),
                },
                complete: {
                    reminders: {
                        create: [{
                            id: generatePK().toString(),
                            name: "New Reminder",
                            index: 2,
                        }],
                    },
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.reminder_listCreateInput>): Prisma.reminder_listCreateInput {
        return {
            id: generatePK().toString(),
            user: {
                connect: { id: "default_user_id" }
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.reminder_listCreateInput>): Prisma.reminder_listCreateInput {
        return {
            id: generatePK().toString(),
            user: {
                connect: { id: "default_user_id" }
            },
            reminders: {
                create: [
                    {
                        id: generatePK().toString(),
                        name: "Default Reminder 1",
                        index: 0,
                    },
                    {
                        id: generatePK().toString(),
                        name: "Default Reminder 2",
                        index: 1,
                        dueDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000),
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            personalList: {
                name: "personalList",
                description: "Personal reminder list",
                config: {
                    withReminders: 5,
                },
            },
            workList: {
                name: "workList",
                description: "Work-related reminders",
                config: {
                    withReminders: 10,
                },
            },
            emptyList: {
                name: "emptyList",
                description: "Empty reminder list",
                config: {},
            },
            completedList: {
                name: "completedList",
                description: "List with all reminders completed",
                config: {
                    withReminders: 3,
                },
            },
        };
    }

    /**
     * Create specific list types
     */
    async createPersonalList(userId: string): Promise<Prisma.reminder_list> {
        return await this.seedScenario('personalList');
    }

    async createWorkList(userId: string): Promise<Prisma.reminder_list> {
        return await this.createWithRelations({
            overrides: {
                user: { connect: { id: userId } },
            },
            withReminders: 10,
        });
    }

    async createEmptyList(userId: string): Promise<Prisma.reminder_list> {
        return await this.createMinimal({
            user: { connect: { id: userId } },
        });
    }

    protected getDefaultInclude(): Prisma.reminder_listInclude {
        return {
            user: true,
            reminders: {
                orderBy: { index: 'asc' },
                include: {
                    reminderItems: {
                        orderBy: { index: 'asc' },
                    },
                },
            },
            _count: {
                select: {
                    reminders: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.reminder_listCreateInput,
        config: ReminderListRelationConfig,
        tx: any
    ): Promise<Prisma.reminder_listCreateInput> {
        let data = { ...baseData };

        // Handle user relationship
        if (config.withUser) {
            data.user = {
                connect: { id: config.withUser.userId }
            };
        }

        // Handle reminders
        if (config.withReminders) {
            const reminderCount = typeof config.withReminders === 'number' ? config.withReminders : 1;
            
            data.reminders = {
                create: Array.from({ length: reminderCount }, (_, i) => ({
                    id: generatePK().toString(),
                    name: `Reminder ${i + 1}`,
                    description: `Auto-generated reminder ${i + 1}`,
                    index: i,
                    dueDate: i % 2 === 0 ? new Date(Date.now() + (i + 1) * 24 * 60 * 60 * 1000) : undefined,
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.reminder_list): Promise<string[]> {
        const violations: string[] = [];
        
        // Check user association
        if (!record.userId) {
            violations.push('Reminder list must belong to a user');
        }

        // Check reminder ordering
        if (record.reminders) {
            const indexSet = new Set<number>();
            for (const reminder of record.reminders) {
                if (indexSet.has(reminder.index)) {
                    violations.push('Reminder indexes must be unique within a list');
                }
                indexSet.add(reminder.index);
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            reminders: {
                include: {
                    reminderItems: true,
                },
            },
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.reminder_list,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete reminders (which will cascade delete their items)
        if (shouldDelete('reminders') && record.reminders?.length) {
            await tx.reminder.deleteMany({
                where: { reminderListId: record.id },
            });
        }
    }

    /**
     * Create a list with specific reminder configuration
     */
    async createListWithReminders(
        userId: string,
        reminders: Array<{ name: string; description?: string; itemCount?: number }>
    ): Promise<Prisma.reminder_list> {
        return await this.createComplete({
            user: { connect: { id: userId } },
            reminders: {
                create: reminders.map((reminder, i) => ({
                    id: generatePK().toString(),
                    name: reminder.name,
                    description: reminder.description,
                    index: i,
                    reminderItems: reminder.itemCount ? {
                        create: Array.from({ length: reminder.itemCount }, (_, j) => ({
                            id: generatePK().toString(),
                            name: `${reminder.name} - Item ${j + 1}`,
                            index: j,
                        })),
                    } : undefined,
                })),
            },
        });
    }
}

// Export factory creator functions
export const createReminderDbFactory = (prisma: PrismaClient) => 
    ReminderDbFactory.getInstance('Reminder', prisma);

export const createReminderListDbFactory = (prisma: PrismaClient) => 
    ReminderListDbFactory.getInstance('ReminderList', prisma);

// Export the classes for type usage
export { ReminderDbFactory as ReminderDbFactoryClass };
export { ReminderListDbFactory as ReminderListDbFactoryClass };