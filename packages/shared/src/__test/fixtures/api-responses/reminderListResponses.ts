/* c8 ignore start */
/**
 * Reminder List API Response Fixtures
 * 
 * Comprehensive fixtures for reminder list management including
 * list creation, organization, hierarchy, and task collections.
 */

import type {
    ReminderList,
    ReminderListCreateInput,
    ReminderListUpdateInput,
} from "../../../api/types.js";
import { generatePK } from "../../../id/index.js";
import { BaseAPIResponseFactory } from "./base.js";
import { reminderItemResponseFactory } from "./reminderItemResponses.js";
import type { MockDataOptions } from "./types.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_NAME_LENGTH = 200;
const MAX_DESCRIPTION_LENGTH = 1000;
const MAX_LISTS_PER_USER = 100;
const MAX_ITEMS_PER_LIST = 1000;
const MILLISECONDS_PER_HOUR = 60 * 60 * 1000;
const MILLISECONDS_PER_DAY = 24 * MILLISECONDS_PER_HOUR;
const HOURS_IN_2 = 2;
const HOURS_IN_6 = 6;
const DAYS_IN_1 = 1;
const DAYS_IN_3 = 3;
const DAYS_IN_7 = 7;

// Common list names for different scenarios
const COMMON_LIST_NAMES = [
    "Personal Tasks",
    "Work Projects",
    "Shopping List",
    "Daily Habits",
    "Weekly Goals",
    "Monthly Objectives",
    "Learning Goals",
    "Health & Fitness",
] as const;

/**
 * Reminder List API response factory
 */
export class ReminderListResponseFactory extends BaseAPIResponseFactory<
    ReminderList,
    ReminderListCreateInput,
    ReminderListUpdateInput
> {
    protected readonly entityName = "reminder_list";

    /**
     * Create mock reminder list data
     */
    createMockData(options?: MockDataOptions): ReminderList {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const listId = options?.overrides?.id || generatePK().toString();

        const baseReminderList: ReminderList = {
            __typename: "ReminderList",
            id: listId,
            createdAt: now,
            updatedAt: now,
            name: "My Tasks",
            description: null,
            reminderItems: [],
            reminderItemsCount: 0,
            user: userResponseFactory.createMockData(),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const itemCount = scenario === "edge-case" ? MAX_ITEMS_PER_LIST : 8;
            const reminderItems = reminderItemResponseFactory.createReminderItemsForList(listId, itemCount);

            return {
                ...baseReminderList,
                name: scenario === "edge-case"
                    ? "Z".repeat(MAX_NAME_LENGTH) // Maximum length name
                    : "Work Projects & Daily Tasks",
                description: scenario === "complete"
                    ? "Comprehensive list for managing work projects, daily tasks, and personal goals with detailed tracking and progress monitoring."
                    : scenario === "edge-case"
                        ? "Y".repeat(MAX_DESCRIPTION_LENGTH) // Maximum length description
                        : null,
                reminderItems,
                reminderItemsCount: reminderItems.length,
                user: userResponseFactory.createMockData({ scenario: "complete" }),
                you: {
                    canDelete: scenario !== "edge-case",
                    canUpdate: scenario !== "edge-case",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseReminderList,
            ...options?.overrides,
        };
    }

    /**
     * Create reminder list from input
     */
    createFromInput(input: ReminderListCreateInput): ReminderList {
        const now = new Date().toISOString();
        const listId = generatePK().toString();

        return {
            __typename: "ReminderList",
            id: listId,
            createdAt: now,
            updatedAt: now,
            name: input.name,
            description: input.description || null,
            reminderItems: [],
            reminderItemsCount: 0,
            user: userResponseFactory.createMockData({ overrides: { id: input.userConnect } }),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update reminder list from input
     */
    updateFromInput(existing: ReminderList, input: ReminderListUpdateInput): ReminderList {
        const updates: Partial<ReminderList> = {
            updatedAt: new Date().toISOString(),
        };

        if (input.name !== undefined) updates.name = input.name;
        if (input.description !== undefined) updates.description = input.description;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ReminderListCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.name || input.name.trim().length === 0) {
            errors.name = "Reminder list name is required";
        } else if (input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        if (input.description && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
        }

        if (!input.userConnect) {
            errors.userConnect = "User ID is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ReminderListUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined) {
            if (!input.name || input.name.trim().length === 0) {
                errors.name = "Reminder list name cannot be empty";
            } else if (input.name.length > MAX_NAME_LENGTH) {
                errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
            }
        }

        if (input.description !== undefined && input.description && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create reminder lists for a user
     */
    createReminderListsForUser(userId: string, count = 5): ReminderList[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });

        return Array.from({ length: count }, (_, index) => {
            const listName = COMMON_LIST_NAMES[index % COMMON_LIST_NAMES.length];
            const itemCount = Math.floor(Math.random() * 10) + 1; // 1-10 items
            const completedItems = Math.floor(itemCount * Math.random()); // Random completion

            return this.createMockData({
                overrides: {
                    id: `list_${userId}_${index}`,
                    name: listName,
                    description: index === 0 ? `${listName} for managing daily activities` : null,
                    user,
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList(`list_${userId}_${index}`, itemCount),
                    reminderItemsCount: itemCount,
                    createdAt: new Date(Date.now().toISOString() - (index * DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                },
            });
        });
    }

    /**
     * Create reminder lists with different item counts
     */
    createReminderListsWithVariedCounts(): ReminderList[] {
        const baseTime = Date.now();

        return [
            // Empty list
            this.createMockData({
                overrides: {
                    id: "empty_list",
                    name: "New List",
                    description: "A freshly created list with no items yet",
                    reminderItems: [],
                    reminderItemsCount: 0,
                    createdAt: new Date(baseTime - (HOURS_IN_2 * MILLISECONDS_PER_HOUR).toISOString()).toISOString(),
                },
            }),

            // Small list (1-3 items)
            this.createMockData({
                overrides: {
                    id: "small_list",
                    name: "Quick Tasks",
                    description: "Short list of urgent items",
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList("small_list", 2),
                    reminderItemsCount: 2,
                    createdAt: new Date(baseTime - (HOURS_IN_6 * MILLISECONDS_PER_HOUR).toISOString()).toISOString(),
                },
            }),

            // Medium list (4-10 items)
            this.createMockData({
                overrides: {
                    id: "medium_list",
                    name: "Weekly Goals",
                    description: "Tasks to complete this week",
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList("medium_list", 7),
                    reminderItemsCount: 7,
                    createdAt: new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Large list (11+ items)
            this.createMockData({
                overrides: {
                    id: "large_list",
                    name: "Project Backlog",
                    description: "Comprehensive list of all project tasks and future enhancements",
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList("large_list", 15),
                    reminderItemsCount: 15,
                    createdAt: new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Completed list (all items done)
            this.createMockData({
                overrides: {
                    id: "completed_list",
                    name: "Sprint 1 Tasks",
                    description: "All tasks from the first sprint - completed successfully",
                    reminderItems: reminderItemResponseFactory.createReminderItemsByStatus(true, 5),
                    reminderItemsCount: 5,
                    createdAt: new Date(baseTime - (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),
        ];
    }

    /**
     * Create reminder lists for different categories
     */
    createReminderListsForCategories(): ReminderList[] {
        const baseTime = Date.now();

        return [
            // Personal tasks
            this.createMockData({
                overrides: {
                    id: "personal_tasks",
                    name: "Personal Tasks",
                    description: "Daily personal activities and self-care",
                    reminderItems: [
                        ...reminderItemResponseFactory.createReminderItemsForList("personal_tasks", 4),
                    ],
                    reminderItemsCount: 4,
                    createdAt: new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Work projects
            this.createMockData({
                overrides: {
                    id: "work_projects",
                    name: "Work Projects",
                    description: "Professional tasks and project deliverables",
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList("work_projects", 8),
                    reminderItemsCount: 8,
                    createdAt: new Date(baseTime - (DAYS_IN_3 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),

            // Shopping list
            this.createMockData({
                overrides: {
                    id: "shopping_list",
                    name: "Shopping List",
                    description: "Grocery and household items to purchase",
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList("shopping_list", 6),
                    reminderItemsCount: 6,
                    createdAt: new Date(baseTime - (HOURS_IN_6 * MILLISECONDS_PER_HOUR).toISOString()).toISOString(),
                },
            }),

            // Learning goals
            this.createMockData({
                overrides: {
                    id: "learning_goals",
                    name: "Learning Goals",
                    description: "Educational objectives and skill development",
                    reminderItems: reminderItemResponseFactory.createReminderItemsForList("learning_goals", 5),
                    reminderItemsCount: 5,
                    createdAt: new Date(baseTime - (DAYS_IN_7 * MILLISECONDS_PER_DAY).toISOString()).toISOString(),
                },
            }),
        ];
    }

    /**
     * Create user limit reached error response
     */
    createUserLimitReachedErrorResponse(userId: string, currentCount = MAX_LISTS_PER_USER) {
        return this.createBusinessErrorResponse("user_limit", {
            resource: "reminder_list",
            userId,
            limit: MAX_LISTS_PER_USER,
            current: currentCount,
            message: `User has reached the maximum number of reminder lists (${MAX_LISTS_PER_USER})`,
        });
    }

    /**
     * Create list not empty error response
     */
    createListNotEmptyErrorResponse(listId: string, itemCount: number) {
        return this.createBusinessErrorResponse("list_not_empty", {
            resource: "reminder_list",
            listId,
            itemCount,
            message: `Cannot delete reminder list with ${itemCount} items. Please remove all items first.`,
        });
    }

    /**
     * Create duplicate name error response
     */
    createDuplicateNameErrorResponse(name: string, userId: string) {
        return this.createBusinessErrorResponse("duplicate_name", {
            resource: "reminder_list",
            name,
            userId,
            message: `A reminder list named "${name}" already exists for this user`,
        });
    }
}

/**
 * Pre-configured reminder list response scenarios
 */
export const reminderListResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ReminderListCreateInput>) => {
        const factory = new ReminderListResponseFactory();
        const defaultInput: ReminderListCreateInput = {
            name: "My New List",
            description: "A new reminder list for organizing tasks",
            userConnect: generatePK().toString(),
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (reminderList?: ReminderList) => {
        const factory = new ReminderListResponseFactory();
        return factory.createSuccessResponse(
            reminderList || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ReminderListResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ReminderList, updates?: Partial<ReminderListUpdateInput>) => {
        const factory = new ReminderListResponseFactory();
        const list = existing || factory.createMockData({ scenario: "complete" });
        const input: ReminderListUpdateInput = {
            id: list.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(list, input),
        );
    },

    listSuccess: (reminderLists?: ReminderList[]) => {
        const factory = new ReminderListResponseFactory();
        return factory.createPaginatedResponse(
            reminderLists || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: reminderLists?.length || DEFAULT_COUNT },
        );
    },

    userListsSuccess: (userId?: string, count?: number) => {
        const factory = new ReminderListResponseFactory();
        const lists = factory.createReminderListsForUser(userId || generatePK().toString(), count);
        return factory.createPaginatedResponse(
            lists,
            { page: 1, totalCount: lists.length },
        );
    },

    variedCountsSuccess: () => {
        const factory = new ReminderListResponseFactory();
        const lists = factory.createReminderListsWithVariedCounts();
        return factory.createPaginatedResponse(
            lists,
            { page: 1, totalCount: lists.length },
        );
    },

    categoriesSuccess: () => {
        const factory = new ReminderListResponseFactory();
        const lists = factory.createReminderListsForCategories();
        return factory.createPaginatedResponse(
            lists,
            { page: 1, totalCount: lists.length },
        );
    },

    emptyListSuccess: () => {
        const factory = new ReminderListResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    name: "Empty List",
                    reminderItems: [],
                    reminderItemsCount: 0,
                },
            }),
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ReminderListResponseFactory();
        return factory.createValidationErrorResponse({
            name: "Reminder list name is required",
            userConnect: "User ID is required",
        });
    },

    notFoundError: (listId?: string) => {
        const factory = new ReminderListResponseFactory();
        return factory.createNotFoundErrorResponse(
            listId || "non-existent-list",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ReminderListResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["reminder:write"],
        );
    },

    userLimitError: (userId?: string, currentCount?: number) => {
        const factory = new ReminderListResponseFactory();
        return factory.createUserLimitReachedErrorResponse(userId || generatePK().toString(), currentCount);
    },

    listNotEmptyError: (listId?: string, itemCount = 5) => {
        const factory = new ReminderListResponseFactory();
        return factory.createListNotEmptyErrorResponse(listId || generatePK().toString(), itemCount);
    },

    duplicateNameError: (name = "My Tasks", userId?: string) => {
        const factory = new ReminderListResponseFactory();
        return factory.createDuplicateNameErrorResponse(name, userId || generatePK().toString());
    },

    nameTooLongError: () => {
        const factory = new ReminderListResponseFactory();
        return factory.createValidationErrorResponse({
            name: `Name must be ${MAX_NAME_LENGTH} characters or less`,
        });
    },

    descriptionTooLongError: () => {
        const factory = new ReminderListResponseFactory();
        return factory.createValidationErrorResponse({
            description: `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`,
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ReminderListResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ReminderListResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ReminderListResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const reminderListResponseFactory = new ReminderListResponseFactory();

