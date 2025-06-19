import { type Reminder, type ReminderItem, type ReminderList, type ReminderEdge, type ReminderSearchResult } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";

/**
 * API response fixtures for reminders
 * These represent what components receive from API calls
 */

/**
 * Mock reminder item data
 */
const basicReminderItem: ReminderItem = {
    __typename: "ReminderItem",
    id: "reminderitem_123456789",
    name: "Review pull request",
    description: null,
    dueDate: null,
    index: 0,
    isComplete: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    reminder: {} as Reminder, // Avoid circular reference
};

const completeReminderItem: ReminderItem = {
    __typename: "ReminderItem",
    id: "reminderitem_987654321",
    name: "Update documentation",
    description: "Update API documentation with new endpoints",
    dueDate: "2024-03-15T17:00:00Z",
    index: 1,
    isComplete: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-15T10:30:00Z",
    reminder: {} as Reminder, // Avoid circular reference
};

const completedReminderItem: ReminderItem = {
    __typename: "ReminderItem",
    id: "reminderitem_completed_123",
    name: "Write unit tests",
    description: "Add comprehensive test coverage for new features",
    dueDate: "2024-01-10T17:00:00Z",
    index: 0,
    isComplete: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-10T16:45:00Z",
    reminder: {} as Reminder, // Avoid circular reference
};

const overdueReminderItem: ReminderItem = {
    __typename: "ReminderItem",
    id: "reminderitem_overdue_456",
    name: "Fix critical bug",
    description: "Address memory leak in data processing module",
    dueDate: "2024-01-05T17:00:00Z", // Past due date
    index: 0,
    isComplete: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-05T09:00:00Z",
    reminder: {} as Reminder, // Avoid circular reference
};

/**
 * Mock reminder list data
 */
const personalReminderList: ReminderList = {
    __typename: "ReminderList",
    id: "reminderlist_personal_123",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-20T15:30:00Z",
    reminders: [], // Will be populated by parent reminders
};

const workReminderList: ReminderList = {
    __typename: "ReminderList",
    id: "reminderlist_work_456789",
    createdAt: "2023-12-01T00:00:00Z",
    updatedAt: "2024-01-20T12:00:00Z",
    reminders: [], // Will be populated by parent reminders
};

/**
 * Minimal reminder API response
 */
export const minimalReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_123456789012345",
    name: "Buy groceries",
    description: null,
    dueDate: null,
    index: 0,
    isComplete: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    reminderItems: [],
    reminderList: personalReminderList,
};

/**
 * Complete reminder API response with all fields
 */
export const completeReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_987654321098765",
    name: "Complete project milestone",
    description: "Finish implementing the authentication module and write comprehensive tests",
    dueDate: "2024-03-31T17:00:00Z",
    index: 1,
    isComplete: false,
    createdAt: "2023-12-01T00:00:00Z",
    updatedAt: "2024-01-20T14:45:00Z",
    reminderItems: [
        { ...basicReminderItem, reminder: {} as Reminder },
        { ...completeReminderItem, reminder: {} as Reminder },
    ],
    reminderList: workReminderList,
};

// Set up circular references
completeReminderResponse.reminderItems[0].reminder = completeReminderResponse;
completeReminderResponse.reminderItems[1].reminder = completeReminderResponse;

/**
 * Completed reminder
 */
export const completedReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_completed_123456",
    name: "Setup development environment",
    description: "Install and configure all necessary development tools",
    dueDate: "2024-01-15T17:00:00Z",
    index: 0,
    isComplete: true,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-15T16:30:00Z",
    reminderItems: [
        { ...completedReminderItem, reminder: {} as Reminder },
    ],
    reminderList: workReminderList,
};

// Set up circular reference
completedReminderResponse.reminderItems[0].reminder = completedReminderResponse;

/**
 * Overdue reminder
 */
export const overdueReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_overdue_789012",
    name: "Submit quarterly report",
    description: "Compile and submit Q4 performance metrics and analysis",
    dueDate: "2024-01-05T17:00:00Z", // Past due date
    index: 2,
    isComplete: false,
    createdAt: "2023-12-15T00:00:00Z",
    updatedAt: "2024-01-06T09:15:00Z",
    reminderItems: [
        { ...overdueReminderItem, reminder: {} as Reminder },
    ],
    reminderList: workReminderList,
};

// Set up circular reference
overdueReminderResponse.reminderItems[0].reminder = overdueReminderResponse;

/**
 * Recurring reminder (simulated through multiple similar reminders)
 */
export const recurringReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_recurring_345678",
    name: "Weekly team standup prep",
    description: "Prepare updates and blockers for Monday morning standup",
    dueDate: "2024-01-22T09:00:00Z",
    index: 0,
    isComplete: false,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-20T00:00:00Z",
    reminderItems: [
        {
            __typename: "ReminderItem",
            id: "reminderitem_standup_1",
            name: "Review completed tasks",
            description: null,
            dueDate: null,
            index: 0,
            isComplete: false,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            reminder: {} as Reminder,
        },
        {
            __typename: "ReminderItem",
            id: "reminderitem_standup_2",
            name: "Identify current blockers",
            description: null,
            dueDate: null,
            index: 1,
            isComplete: false,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            reminder: {} as Reminder,
        },
    ],
    reminderList: workReminderList,
};

// Set up circular references
recurringReminderResponse.reminderItems[0].reminder = recurringReminderResponse;
recurringReminderResponse.reminderItems[1].reminder = recurringReminderResponse;

/**
 * High priority reminder
 */
export const highPriorityReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_priority_567890",
    name: "Security vulnerability patch",
    description: "URGENT: Apply critical security patch to production servers",
    dueDate: "2024-01-21T12:00:00Z", // Today with specific time
    index: 0,
    isComplete: false,
    createdAt: "2024-01-20T08:00:00Z",
    updatedAt: "2024-01-20T08:00:00Z",
    reminderItems: [
        {
            __typename: "ReminderItem",
            id: "reminderitem_patch_1",
            name: "Test patch in staging",
            description: "Verify patch works correctly in staging environment",
            dueDate: "2024-01-21T10:00:00Z",
            index: 0,
            isComplete: false,
            createdAt: "2024-01-20T08:00:00Z",
            updatedAt: "2024-01-20T08:00:00Z",
            reminder: {} as Reminder,
        },
        {
            __typename: "ReminderItem",
            id: "reminderitem_patch_2",
            name: "Deploy to production",
            description: "Apply patch to all production servers",
            dueDate: "2024-01-21T11:30:00Z",
            index: 1,
            isComplete: false,
            createdAt: "2024-01-20T08:00:00Z",
            updatedAt: "2024-01-20T08:00:00Z",
            reminder: {} as Reminder,
        },
        {
            __typename: "ReminderItem",
            id: "reminderitem_patch_3",
            name: "Verify deployment success",
            description: "Confirm all systems are running normally",
            dueDate: "2024-01-21T12:00:00Z",
            index: 2,
            isComplete: false,
            createdAt: "2024-01-20T08:00:00Z",
            updatedAt: "2024-01-20T08:00:00Z",
            reminder: {} as Reminder,
        },
    ],
    reminderList: workReminderList,
};

// Set up circular references
highPriorityReminderResponse.reminderItems.forEach(item => {
    item.reminder = highPriorityReminderResponse;
});

/**
 * Personal reminder
 */
export const personalReminderResponse: Reminder = {
    __typename: "Reminder",
    id: "reminder_personal_678901",
    name: "Doctor appointment",
    description: "Annual check-up with Dr. Smith",
    dueDate: "2024-02-15T14:30:00Z",
    index: 0,
    isComplete: false,
    createdAt: "2024-01-10T00:00:00Z",
    updatedAt: "2024-01-10T00:00:00Z",
    reminderItems: [
        {
            __typename: "ReminderItem",
            id: "reminderitem_doctor_1",
            name: "Bring insurance card",
            description: null,
            dueDate: null,
            index: 0,
            isComplete: false,
            createdAt: "2024-01-10T00:00:00Z",
            updatedAt: "2024-01-10T00:00:00Z",
            reminder: {} as Reminder,
        },
        {
            __typename: "ReminderItem",
            id: "reminderitem_doctor_2",
            name: "List of current medications",
            description: null,
            dueDate: null,
            index: 1,
            isComplete: false,
            createdAt: "2024-01-10T00:00:00Z",
            updatedAt: "2024-01-10T00:00:00Z",
            reminder: {} as Reminder,
        },
    ],
    reminderList: personalReminderList,
};

// Set up circular references
personalReminderResponse.reminderItems[0].reminder = personalReminderResponse;
personalReminderResponse.reminderItems[1].reminder = personalReminderResponse;

/**
 * Reminder variant states for testing
 */
export const reminderResponseVariants = {
    minimal: minimalReminderResponse,
    complete: completeReminderResponse,
    completed: completedReminderResponse,
    overdue: overdueReminderResponse,
    recurring: recurringReminderResponse,
    highPriority: highPriorityReminderResponse,
    personal: personalReminderResponse,
    simpleTask: {
        ...minimalReminderResponse,
        id: "reminder_simple_456789",
        name: "Call client",
        description: "Follow up on project proposal",
        dueDate: "2024-01-25T10:00:00Z",
        index: 1,
    },
    longTerm: {
        ...completeReminderResponse,
        id: "reminder_longterm_890123",
        name: "Plan summer vacation",
        description: "Research destinations, book flights and accommodation",
        dueDate: "2024-04-30T17:00:00Z",
        index: 0,
        reminderItems: [
            {
                __typename: "ReminderItem",
                id: "reminderitem_vacation_1",
                name: "Research destinations",
                description: "Look into Europe vs Asia options",
                dueDate: "2024-02-28T17:00:00Z",
                index: 0,
                isComplete: false,
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                reminder: {} as Reminder,
            },
            {
                __typename: "ReminderItem",
                id: "reminderitem_vacation_2",
                name: "Book flights",
                description: "Compare prices and book tickets",
                dueDate: "2024-03-31T17:00:00Z",
                index: 1,
                isComplete: false,
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                reminder: {} as Reminder,
            },
        ],
    },
    noItems: {
        ...minimalReminderResponse,
        id: "reminder_noitems_234567",
        name: "Simple reminder without subtasks",
        description: "Just a basic reminder with no breakdown",
        dueDate: "2024-01-30T12:00:00Z",
        reminderItems: [],
    },
} as const;

// Set up circular references for variant reminders
reminderResponseVariants.longTerm.reminderItems[0].reminder = reminderResponseVariants.longTerm;
reminderResponseVariants.longTerm.reminderItems[1].reminder = reminderResponseVariants.longTerm;

/**
 * Reminder search response
 */
export const reminderSearchResponse: ReminderSearchResult = {
    __typename: "ReminderSearchResult",
    edges: [
        {
            __typename: "ReminderEdge",
            cursor: "cursor_1",
            node: reminderResponseVariants.complete,
        },
        {
            __typename: "ReminderEdge",
            cursor: "cursor_2",
            node: reminderResponseVariants.overdue,
        },
        {
            __typename: "ReminderEdge",
            cursor: "cursor_3",
            node: reminderResponseVariants.highPriority,
        },
        {
            __typename: "ReminderEdge",
            cursor: "cursor_4",
            node: reminderResponseVariants.personal,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_4",
    },
};

/**
 * Filtered search responses for different reminder states
 */
export const pendingRemindersSearchResponse: ReminderSearchResult = {
    __typename: "ReminderSearchResult",
    edges: [
        {
            __typename: "ReminderEdge",
            cursor: "cursor_pending_1",
            node: reminderResponseVariants.complete,
        },
        {
            __typename: "ReminderEdge",
            cursor: "cursor_pending_2",
            node: reminderResponseVariants.highPriority,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_pending_1",
        endCursor: "cursor_pending_2",
    },
};

export const completedRemindersSearchResponse: ReminderSearchResult = {
    __typename: "ReminderSearchResult",
    edges: [
        {
            __typename: "ReminderEdge",
            cursor: "cursor_completed_1",
            node: reminderResponseVariants.completed,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_completed_1",
        endCursor: "cursor_completed_1",
    },
};

export const overdueRemindersSearchResponse: ReminderSearchResult = {
    __typename: "ReminderSearchResult",
    edges: [
        {
            __typename: "ReminderEdge",
            cursor: "cursor_overdue_1",
            node: reminderResponseVariants.overdue,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_overdue_1",
        endCursor: "cursor_overdue_1",
    },
};

/**
 * Loading and error states for UI testing
 */
export const reminderUIStates = {
    loading: null,
    error: {
        code: "REMINDER_NOT_FOUND",
        message: "The requested reminder could not be found",
    },
    listError: {
        code: "REMINDER_LIST_NOT_FOUND", 
        message: "The requested reminder list could not be found",
    },
    createError: {
        code: "REMINDER_CREATE_FAILED",
        message: "Failed to create reminder. Please try again.",
    },
    updateError: {
        code: "REMINDER_UPDATE_FAILED",
        message: "Failed to update reminder. Please try again.",
    },
    deleteError: {
        code: "REMINDER_DELETE_FAILED",
        message: "Failed to delete reminder. Please try again.",
    },
    permissionError: {
        code: "REMINDER_PERMISSION_DENIED",
        message: "You don't have permission to modify this reminder",
    },
    overdueWarning: {
        code: "REMINDER_OVERDUE",
        message: "This reminder is past its due date",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};