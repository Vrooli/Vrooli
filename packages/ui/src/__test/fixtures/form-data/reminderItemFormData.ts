import { DUMMY_ID, type ReminderItemShape } from "@vrooli/shared";

/**
 * Form data fixtures for reminder item-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Reminder item creation form data
 */
export const minimalReminderItemCreateFormInput = {
    name: "Simple task",
    description: "",
    dueDate: null,
    index: 0,
    isComplete: false,
    reminderId: "123456789012345678", // Parent reminder ID
};

export const completeReminderItemCreateFormInput = {
    name: "Review pull request",
    description: "Review and approve the latest pull request for the authentication module",
    dueDate: new Date("2024-12-31T15:00:00Z"),
    index: 0,
    isComplete: false,
    reminderId: "123456789012345678", // Parent reminder ID
};

/**
 * Reminder item update form data
 */
export const minimalReminderItemUpdateFormInput = {
    name: "Updated task name",
    isComplete: true,
};

export const completeReminderItemUpdateFormInput = {
    name: "Updated task name",
    description: "Updated description with more details",
    dueDate: new Date("2025-01-15T10:00:00Z"),
    index: 1,
    isComplete: true,
};

/**
 * Different reminder item types for various use cases
 */
export const reminderItemFormVariants = {
    simple: {
        name: "Buy milk",
        description: "",
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderId: "123456789012345678",
    },
    detailed: {
        name: "Complete quarterly report",
        description: "Compile financial data, create visualizations, and write executive summary",
        dueDate: new Date("2024-03-31T17:00:00Z"),
        index: 0,
        isComplete: false,
        reminderId: "234567890123456789",
    },
    urgent: {
        name: "Fix critical bug",
        description: "Address memory leak in production server before next deployment",
        dueDate: new Date("2024-01-20T12:00:00Z"),
        index: 0,
        isComplete: false,
        reminderId: "345678901234567890",
    },
    completed: {
        name: "Setup development environment",
        description: "Install dependencies and configure local database",
        dueDate: new Date("2024-01-15T09:00:00Z"),
        index: 0,
        isComplete: true,
        reminderId: "456789012345678901",
    },
    overdue: {
        name: "Submit expense report",
        description: "Submit Q4 2023 expense report with receipts",
        dueDate: new Date("2023-12-31T23:59:59Z"),
        index: 0,
        isComplete: false,
        reminderId: "567890123456789012",
    },
};

/**
 * Reminder item batch creation for multiple items
 */
export const reminderItemBatchFormInput = [
    {
        name: "Research competitors",
        description: "Analyze top 5 competitors' features and pricing",
        index: 0,
        isComplete: false,
    },
    {
        name: "Create comparison matrix",
        description: "Build feature comparison spreadsheet",
        index: 1,
        isComplete: false,
    },
    {
        name: "Present findings",
        description: "Prepare and deliver presentation to team",
        index: 2,
        isComplete: false,
    },
];

/**
 * Form validation test cases
 */
export const invalidReminderItemFormInputs = {
    missingName: {
        name: "", // Required field
        description: "Task without a name",
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderId: "123456789012345678",
    },
    nameTooLong: {
        name: "A".repeat(51), // Too long (max 50 chars)
        description: "Valid description",
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderId: "123456789012345678",
    },
    descriptionTooLong: {
        name: "Valid task",
        description: "A".repeat(2049), // Too long (max 2048 chars)
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderId: "123456789012345678",
    },
    negativeIndex: {
        name: "Valid task",
        description: "Valid description",
        dueDate: null,
        index: -1, // Should be non-negative
        isComplete: false,
        reminderId: "123456789012345678",
    },
    invalidDate: {
        name: "Valid task",
        description: "Valid description",
        // @ts-expect-error - Testing invalid input
        dueDate: "not-a-date",
        index: 0,
        isComplete: false,
        reminderId: "123456789012345678",
    },
    invalidBoolean: {
        name: "Valid task",
        description: "Valid description",
        dueDate: null,
        index: 0,
        // @ts-expect-error - Testing invalid input
        isComplete: "yes", // Should be boolean
        reminderId: "123456789012345678",
    },
    missingReminderId: {
        name: "Task without reminder",
        description: "This task has no parent reminder",
        dueDate: null,
        index: 0,
        isComplete: false,
        // Missing required reminderId for creation
    },
    invalidReminderId: {
        name: "Task with invalid reminder",
        description: "This task has an invalid parent reminder ID",
        dueDate: null,
        index: 0,
        isComplete: false,
        reminderId: "invalid-id", // Not a valid snowflake ID
    },
};

/**
 * Form validation states
 */
export const reminderItemFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "", // Empty name
            description: "A".repeat(2049), // Too long
            dueDate: new Date("2024-01-01"),
            index: -1, // Negative
            isComplete: false,
            reminderId: "123456789012345678",
        },
        errors: {
            name: "Task name is required",
            description: "Description cannot exceed 2048 characters",
            index: "Index must be a positive number",
        },
        touched: {
            name: true,
            description: true,
            index: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalReminderItemCreateFormInput,
        errors: {},
        touched: {
            name: true,
            description: true,
            dueDate: true,
            index: true,
            isComplete: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeReminderItemCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create reminder item form initial values
 */
export const createReminderItemFormInitialValues = (itemData?: Partial<any>) => ({
    name: itemData?.name || "",
    description: itemData?.description || "",
    dueDate: itemData?.dueDate || null,
    index: itemData?.index || 0,
    isComplete: itemData?.isComplete || false,
    reminderId: itemData?.reminder?.id || itemData?.reminderId || null,
    ...itemData,
});

/**
 * Helper function to transform form data to API format (ReminderItemShape)
 */
export const transformReminderItemFormToApiInput = (formData: any): ReminderItemShape => ({
    __typename: "ReminderItem",
    id: formData.id || DUMMY_ID,
    name: formData.name,
    description: formData.description || undefined,
    dueDate: formData.dueDate ? new Date(formData.dueDate) : undefined,
    index: formData.index,
    isComplete: formData.isComplete || false,
    reminder: formData.reminderId ? {
        __typename: "Reminder",
        __connect: true,
        id: formData.reminderId,
    } : undefined,
});

/**
 * Helper function to validate reminder item due date
 */
export const validateReminderItemDueDate = (dueDate: Date | string | null): string | null => {
    if (!dueDate) return null; // Due date is optional
    
    const date = new Date(dueDate);
    if (isNaN(date.getTime())) {
        return "Please enter a valid date";
    }
    
    // Optional: warn if date is in the past
    if (date < new Date()) {
        return "Due date is in the past";
    }
    
    return null;
};

/**
 * Helper function to sort reminder items by index
 */
export const sortReminderItems = (items: any[]) => {
    return [...items].sort((a, b) => a.index - b.index);
};

/**
 * Mock autocomplete suggestions for reminder items
 */
export const mockReminderItemSuggestions = {
    quickAddTemplates: [
        { name: "Call", icon: "phone" },
        { name: "Email", icon: "email" },
        { name: "Review", icon: "rate_review" },
        { name: "Fix", icon: "build" },
        { name: "Test", icon: "bug_report" },
        { name: "Deploy", icon: "cloud_upload" },
    ],
    dueDatePresets: [
        { label: "Today", value: () => new Date() },
        { label: "Tomorrow", value: () => new Date(Date.now() + 24 * 60 * 60 * 1000) },
        { label: "End of Week", value: () => {
            const date = new Date();
            const day = date.getDay();
            const diff = 5 - day; // Friday
            date.setDate(date.getDate() + (diff > 0 ? diff : 7 + diff));
            return date;
        }},
        { label: "Next Week", value: () => new Date(Date.now() + 7 * 24 * 60 * 60 * 1000) },
        { label: "End of Month", value: () => {
            const date = new Date();
            return new Date(date.getFullYear(), date.getMonth() + 1, 0);
        }},
    ],
};

/**
 * Reminder item priority levels (for future use)
 */
export const reminderItemPriorityLevels = {
    low: {
        name: "Low priority task",
        description: "Can be done whenever",
        priority: 1,
    },
    medium: {
        name: "Medium priority task",
        description: "Should be done soon",
        priority: 2,
    },
    high: {
        name: "High priority task",
        description: "Must be done ASAP",
        priority: 3,
    },
    critical: {
        name: "Critical priority task",
        description: "Drop everything and do this",
        priority: 4,
    },
};

/**
 * Reminder item completion progress for sub-items
 */
export const reminderItemProgressStates = {
    notStarted: {
        items: [
            { name: "Step 1", isComplete: false },
            { name: "Step 2", isComplete: false },
            { name: "Step 3", isComplete: false },
        ],
        progress: 0,
    },
    inProgress: {
        items: [
            { name: "Step 1", isComplete: true },
            { name: "Step 2", isComplete: false },
            { name: "Step 3", isComplete: false },
        ],
        progress: 33,
    },
    almostDone: {
        items: [
            { name: "Step 1", isComplete: true },
            { name: "Step 2", isComplete: true },
            { name: "Step 3", isComplete: false },
        ],
        progress: 67,
    },
    completed: {
        items: [
            { name: "Step 1", isComplete: true },
            { name: "Step 2", isComplete: true },
            { name: "Step 3", isComplete: true },
        ],
        progress: 100,
    },
};