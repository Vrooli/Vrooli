/* c8 ignore start */
/**
 * Reminder Item API Response Fixtures
 * 
 * Comprehensive fixtures for reminder item management including
 * task creation, completion tracking, due dates, and reminder lists.
 */

import type {
    ReminderItem,
    ReminderItemCreateInput,
    ReminderItemUpdateInput,
    Reminder,
    ReminderList,
    User,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { userResponseFactory } from "./userResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_NAME_LENGTH = 200;
const MAX_DESCRIPTION_LENGTH = 1000;
const MAX_ITEMS_PER_LIST = 1000;
const MILLISECONDS_PER_HOUR = 60 * 60 * 1000;
const MILLISECONDS_PER_DAY = 24 * MILLISECONDS_PER_HOUR;
const HOURS_IN_2 = 2;
const HOURS_IN_6 = 6;
const HOURS_IN_12 = 12;
const DAYS_IN_1 = 1;
const DAYS_IN_3 = 3;
const DAYS_IN_7 = 7;

// Priority levels
const PRIORITY_LEVELS = ["Low", "Medium", "High", "Urgent"] as const;

/**
 * Reminder Item API response factory
 */
export class ReminderItemResponseFactory extends BaseAPIResponseFactory<
    ReminderItem,
    ReminderItemCreateInput,
    ReminderItemUpdateInput
> {
    protected readonly entityName = "reminder_item";

    /**
     * Create mock reminder item data
     */
    createMockData(options?: MockDataOptions): ReminderItem {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const itemId = options?.overrides?.id || generatePK().toString();

        const baseReminderItem: ReminderItem = {
            __typename: "ReminderItem",
            id: itemId,
            created_at: now,
            updated_at: now,
            name: "Complete project task",
            description: null,
            dueDate: null,
            index: 0,
            isComplete: false,
            reminderList: this.createMockReminderList(),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            const dueDate = scenario === "edge-case" 
                ? new Date(Date.now() - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString() // Overdue
                : new Date(Date.now() + (DAYS_IN_3 * MILLISECONDS_PER_DAY)).toISOString(); // Due in 3 days

            return {
                ...baseReminderItem,
                name: scenario === "edge-case" 
                    ? "A".repeat(MAX_NAME_LENGTH) // Maximum length name
                    : "Review quarterly business metrics and prepare presentation for board meeting",
                description: scenario === "complete" 
                    ? "This task involves analyzing Q3 financial data, identifying key trends, and creating a comprehensive presentation highlighting our achievements and areas for improvement."
                    : scenario === "edge-case"
                    ? "B".repeat(MAX_DESCRIPTION_LENGTH) // Maximum length description
                    : null,
                dueDate,
                index: scenario === "edge-case" ? 999 : 5,
                isComplete: scenario === "edge-case",
                reminderList: this.createMockReminderList({ scenario }),
                you: {
                    canDelete: scenario !== "edge-case",
                    canUpdate: scenario !== "edge-case",
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseReminderItem,
            ...options?.overrides,
        };
    }

    /**
     * Create reminder item from input
     */
    createFromInput(input: ReminderItemCreateInput): ReminderItem {
        const now = new Date().toISOString();
        const itemId = generatePK().toString();

        return {
            __typename: "ReminderItem",
            id: itemId,
            created_at: now,
            updated_at: now,
            name: input.name,
            description: input.description || null,
            dueDate: input.dueDate || null,
            index: input.index || 0,
            isComplete: input.isComplete || false,
            reminderList: this.createMockReminderList({ 
                overrides: { id: input.reminderListConnect },
            }),
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update reminder item from input
     */
    updateFromInput(existing: ReminderItem, input: ReminderItemUpdateInput): ReminderItem {
        const updates: Partial<ReminderItem> = {
            updated_at: new Date().toISOString(),
        };

        if (input.name !== undefined) updates.name = input.name;
        if (input.description !== undefined) updates.description = input.description;
        if (input.dueDate !== undefined) updates.dueDate = input.dueDate;
        if (input.index !== undefined) updates.index = input.index;
        if (input.isComplete !== undefined) updates.isComplete = input.isComplete;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: ReminderItemCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.name || input.name.trim().length === 0) {
            errors.name = "Reminder item name is required";
        } else if (input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        if (input.description && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
        }

        if (!input.reminderListConnect) {
            errors.reminderListConnect = "Reminder list ID is required";
        }

        if (input.index !== undefined && input.index < 0) {
            errors.index = "Index cannot be negative";
        }

        if (input.dueDate && new Date(input.dueDate).getTime() < Date.now() - (DAYS_IN_1 * MILLISECONDS_PER_DAY)) {
            errors.dueDate = "Due date cannot be more than 1 day in the past";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: ReminderItemUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.name !== undefined) {
            if (!input.name || input.name.trim().length === 0) {
                errors.name = "Reminder item name cannot be empty";
            } else if (input.name.length > MAX_NAME_LENGTH) {
                errors.name = `Name must be ${MAX_NAME_LENGTH} characters or less`;
            }
        }

        if (input.description !== undefined && input.description && input.description.length > MAX_DESCRIPTION_LENGTH) {
            errors.description = `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`;
        }

        if (input.index !== undefined && input.index < 0) {
            errors.index = "Index cannot be negative";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create reminder items for a list
     */
    createReminderItemsForList(listId: string, count = 8): ReminderItem[] {
        const reminderList = this.createMockReminderList({ overrides: { id: listId } });
        
        return Array.from({ length: count }, (_, index) => {
            const isCompleted = index % 3 === 0; // Every 3rd item is completed
            const hasDescription = index % 2 === 0; // Every 2nd item has description
            const hasDueDate = index % 4 === 0; // Every 4th item has due date
            
            return this.createMockData({
                overrides: {
                    id: `item_${listId}_${index}`,
                    reminderList,
                    name: `Task ${index + 1}: ${this.getTaskName(index)}`,
                    description: hasDescription ? this.getTaskDescription(index) : null,
                    dueDate: hasDueDate ? this.getDueDate(index) : null,
                    index,
                    isComplete: isCompleted,
                    created_at: new Date(Date.now() - (index * HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(),
                },
            });
        });
    }

    /**
     * Create reminder items with different priorities and states
     */
    createReminderItemsWithVariedStates(): ReminderItem[] {
        const baseTime = Date.now();
        
        return [
            // Overdue high priority task
            this.createMockData({
                overrides: {
                    id: "overdue_urgent_task",
                    name: "Submit quarterly financial report",
                    description: "Critical: Board meeting is tomorrow and we need the Q3 financial analysis completed",
                    dueDate: new Date(baseTime - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(), // 1 day overdue
                    isComplete: false,
                    index: 0,
                },
            }),

            // Due today task
            this.createMockData({
                overrides: {
                    id: "due_today_task",
                    name: "Review and approve marketing campaign",
                    description: "Final review of the holiday marketing materials before launch",
                    dueDate: new Date(baseTime + (HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString(), // Due in 6 hours
                    isComplete: false,
                    index: 1,
                },
            }),

            // Completed task
            this.createMockData({
                overrides: {
                    id: "completed_task",
                    name: "Update employee handbook",
                    description: "Updated remote work policies and benefits information",
                    dueDate: new Date(baseTime - (HOURS_IN_12 * MILLISECONDS_PER_HOUR)).toISOString(), // Was due 12 hours ago
                    isComplete: true,
                    index: 2,
                    updated_at: new Date(baseTime - (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(), // Completed 2 hours ago
                },
            }),

            // Future task without due date
            this.createMockData({
                overrides: {
                    id: "future_task",
                    name: "Plan team building event",
                    description: "Research venues and activities for Q4 team building day",
                    dueDate: null,
                    isComplete: false,
                    index: 3,
                },
            }),

            // Quick task without description
            this.createMockData({
                overrides: {
                    id: "quick_task",
                    name: "Call IT support about printer issue",
                    description: null,
                    dueDate: new Date(baseTime + (HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(), // Due in 2 hours
                    isComplete: false,
                    index: 4,
                },
            }),

            // Long-term project task
            this.createMockData({
                overrides: {
                    id: "longterm_task",
                    name: "Research new CRM system options",
                    description: "Evaluate different CRM platforms including Salesforce, HubSpot, and Pipedrive. Compare features, pricing, and integration capabilities with our existing tools.",
                    dueDate: new Date(baseTime + (DAYS_IN_7 * MILLISECONDS_PER_DAY)).toISOString(), // Due in 1 week
                    isComplete: false,
                    index: 5,
                },
            }),
        ];
    }

    /**
     * Create reminder items by completion status
     */
    createReminderItemsByStatus(completed = false, count = 5): ReminderItem[] {
        return Array.from({ length: count }, (_, index) =>
            this.createMockData({
                overrides: {
                    id: `${completed ? "completed" : "pending"}_item_${index}`,
                    name: `${completed ? "Completed" : "Pending"} Task ${index + 1}`,
                    isComplete: completed,
                    index,
                    updated_at: completed 
                        ? new Date(Date.now() - (index * HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString()
                        : new Date(Date.now() - (index * HOURS_IN_2 * MILLISECONDS_PER_HOUR)).toISOString(),
                },
            }),
        );
    }

    /**
     * Create list limit error response
     */
    createListLimitErrorResponse(listId: string, currentCount = MAX_ITEMS_PER_LIST) {
        return this.createBusinessErrorResponse("list_limit", {
            resource: "reminder_item",
            listId,
            limit: MAX_ITEMS_PER_LIST,
            current: currentCount,
            message: `Reminder list has reached the maximum number of items (${MAX_ITEMS_PER_LIST})`,
        });
    }

    /**
     * Create cannot complete error response
     */
    createCannotCompleteErrorResponse(reason: string) {
        return this.createBusinessErrorResponse("cannot_complete", {
            resource: "reminder_item",
            reason,
            message: `Cannot mark item as complete: ${reason}`,
        });
    }

    /**
     * Create mock reminder list
     */
    private createMockReminderList(options?: { scenario?: string; overrides?: Partial<ReminderList> }): ReminderList {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const listId = options?.overrides?.id || generatePK().toString();

        const baseList: ReminderList = {
            __typename: "ReminderList",
            id: listId,
            created_at: now,
            updated_at: now,
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
            return {
                ...baseList,
                name: scenario === "edge-case" ? "Edge Case List" : "Work Projects",
                description: scenario === "complete" ? "Important work-related tasks and projects" : null,
                user: userResponseFactory.createMockData({ scenario: "complete" }),
                ...options?.overrides,
            };
        }

        return {
            ...baseList,
            ...options?.overrides,
        };
    }

    /**
     * Get task name for index
     */
    private getTaskName(index: number): string {
        const taskNames = [
            "Review quarterly report",
            "Update project documentation",
            "Schedule team meeting",
            "Prepare presentation slides",
            "Call potential client",
            "Analyze market research data",
            "Update website content",
            "Review budget proposal",
        ];
        return taskNames[index % taskNames.length];
    }

    /**
     * Get task description for index
     */
    private getTaskDescription(index: number): string {
        const descriptions = [
            "Review the Q3 financial results and identify key trends",
            "Update the project README and API documentation",
            "Coordinate schedules for the weekly team sync meeting",
            "Create slides for the upcoming client presentation",
            "Follow up on the lead from last week's networking event",
            "Analyze the customer survey results and market trends",
            "Update the company blog with latest news and updates",
            "Review the proposed marketing budget for next quarter",
        ];
        return descriptions[index % descriptions.length];
    }

    /**
     * Get due date for index
     */
    private getDueDate(index: number): string {
        const baseTimes = [
            HOURS_IN_6 * MILLISECONDS_PER_HOUR, // 6 hours from now
            DAYS_IN_1 * MILLISECONDS_PER_DAY, // 1 day from now
            DAYS_IN_3 * MILLISECONDS_PER_DAY, // 3 days from now
            DAYS_IN_7 * MILLISECONDS_PER_DAY, // 1 week from now
        ];
        return new Date(Date.now() + baseTimes[index % baseTimes.length]).toISOString();
    }
}

/**
 * Pre-configured reminder item response scenarios
 */
export const reminderItemResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<ReminderItemCreateInput>) => {
        const factory = new ReminderItemResponseFactory();
        const defaultInput: ReminderItemCreateInput = {
            name: "Complete important task",
            description: "This is a task that needs to be completed",
            reminderListConnect: generatePK().toString(),
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (reminderItem?: ReminderItem) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            reminderItem || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: ReminderItem, updates?: Partial<ReminderItemUpdateInput>) => {
        const factory = new ReminderItemResponseFactory();
        const item = existing || factory.createMockData({ scenario: "complete" });
        const input: ReminderItemUpdateInput = {
            id: item.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(item, input),
        );
    },

    completeSuccess: (itemId?: string) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: itemId,
                    isComplete: true,
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    uncompleteSuccess: (itemId?: string) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: itemId,
                    isComplete: false,
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    listSuccess: (reminderItems?: ReminderItem[]) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createPaginatedResponse(
            reminderItems || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: reminderItems?.length || DEFAULT_COUNT },
        );
    },

    listItemsSuccess: (listId?: string, count?: number) => {
        const factory = new ReminderItemResponseFactory();
        const items = factory.createReminderItemsForList(listId || generatePK().toString(), count);
        return factory.createPaginatedResponse(
            items,
            { page: 1, totalCount: items.length },
        );
    },

    variedStatesSuccess: () => {
        const factory = new ReminderItemResponseFactory();
        const items = factory.createReminderItemsWithVariedStates();
        return factory.createPaginatedResponse(
            items,
            { page: 1, totalCount: items.length },
        );
    },

    completedItemsSuccess: (count?: number) => {
        const factory = new ReminderItemResponseFactory();
        const items = factory.createReminderItemsByStatus(true, count);
        return factory.createPaginatedResponse(
            items,
            { page: 1, totalCount: items.length },
        );
    },

    pendingItemsSuccess: (count?: number) => {
        const factory = new ReminderItemResponseFactory();
        const items = factory.createReminderItemsByStatus(false, count);
        return factory.createPaginatedResponse(
            items,
            { page: 1, totalCount: items.length },
        );
    },

    overdueItemSuccess: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    name: "Overdue Task",
                    dueDate: new Date(Date.now() - (DAYS_IN_1 * MILLISECONDS_PER_DAY)).toISOString(),
                    isComplete: false,
                },
            }),
        );
    },

    dueTodayItemSuccess: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    name: "Due Today Task",
                    dueDate: new Date(Date.now() + (HOURS_IN_6 * MILLISECONDS_PER_HOUR)).toISOString(),
                    isComplete: false,
                },
            }),
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createValidationErrorResponse({
            name: "Reminder item name is required",
            reminderListConnect: "Reminder list ID is required",
        });
    },

    notFoundError: (itemId?: string) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createNotFoundErrorResponse(
            itemId || "non-existent-item",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
            ["reminder:write"],
        );
    },

    listLimitError: (listId?: string, currentCount?: number) => {
        const factory = new ReminderItemResponseFactory();
        return factory.createListLimitErrorResponse(listId || generatePK().toString(), currentCount);
    },

    cannotCompleteError: (reason = "Item has dependencies") => {
        const factory = new ReminderItemResponseFactory();
        return factory.createCannotCompleteErrorResponse(reason);
    },

    nameTooLongError: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createValidationErrorResponse({
            name: `Name must be ${MAX_NAME_LENGTH} characters or less`,
        });
    },

    descriptionTooLongError: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createValidationErrorResponse({
            description: `Description must be ${MAX_DESCRIPTION_LENGTH} characters or less`,
        });
    },

    invalidDueDateError: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createValidationErrorResponse({
            dueDate: "Due date cannot be more than 1 day in the past",
        });
    },

    negativeIndexError: () => {
        const factory = new ReminderItemResponseFactory();
        return factory.createValidationErrorResponse({
            index: "Index cannot be negative",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new ReminderItemResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new ReminderItemResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new ReminderItemResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const reminderItemResponseFactory = new ReminderItemResponseFactory();

