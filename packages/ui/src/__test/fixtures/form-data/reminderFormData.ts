import { DUMMY_ID, type ReminderShape, type ReminderItemShape } from "@vrooli/shared";

/**
 * Form data fixtures for reminder-related forms
 * These represent data as it appears in form state before submission
 */

/**
 * Reminder creation form data
 */
export const minimalReminderCreateFormInput = {
    name: "Buy groceries",
    description: "",
    dueDate: null,
    index: 0,
    reminderListId: null,
    reminderItems: [],
};

export const completeReminderCreateFormInput = {
    name: "Complete project milestone",
    description: "Finish implementing the authentication module and write tests",
    dueDate: new Date("2024-12-31T17:00:00Z"),
    index: 0,
    reminderListId: "123456789012345678", // Existing list ID
    reminderItems: [
        {
            name: "Write unit tests",
            description: "Create comprehensive test coverage",
            isComplete: false,
            index: 0,
        },
        {
            name: "Update documentation",
            description: "",
            isComplete: false,
            index: 1,
        },
        {
            name: "Run integration tests",
            description: "Ensure all components work together",
            isComplete: false,
            index: 2,
        },
    ],
};

/**
 * Reminder update form data
 */
export const minimalReminderUpdateFormInput = {
    name: "Updated reminder name",
    description: "Updated description",
};

export const completeReminderUpdateFormInput = {
    name: "Updated project milestone",
    description: "Updated description with new requirements and extended scope",
    dueDate: new Date("2025-01-15T17:00:00Z"),
    index: 1,
    reminderItems: [
        {
            id: "123456789012345679", // Existing item to update
            name: "Write unit tests",
            description: "Create comprehensive test coverage with edge cases",
            isComplete: true,
            index: 0,
        },
        {
            id: "123456789012345680", // Existing item to update
            name: "Update documentation",
            description: "Add API documentation and examples",
            isComplete: false,
            index: 1,
        },
        {
            // New item without ID
            name: "Deploy to staging",
            description: "Test in staging environment",
            isComplete: false,
            index: 2,
        },
    ],
    itemsToDelete: ["123456789012345681"], // IDs of items to delete
};

/**
 * Reminder with list creation form data
 */
export const reminderWithNewListFormInput = {
    name: "Vacation planning",
    description: "Plan the summer vacation trip",
    dueDate: new Date("2024-06-15T00:00:00Z"),
    index: 0,
    createNewList: true,
    newListName: "Travel Plans",
    reminderItems: [
        {
            name: "Book flights",
            description: "Compare prices and book best option",
            isComplete: false,
            index: 0,
        },
        {
            name: "Reserve hotel",
            description: "Find accommodation near city center",
            isComplete: false,
            index: 1,
        },
        {
            name: "Plan activities",
            description: "Research and book tours",
            isComplete: false,
            index: 2,
        },
    ],
};

/**
 * Different reminder types for various use cases
 */
export const reminderFormVariants = {
    personal: {
        name: "Doctor appointment",
        description: "Annual checkup at Dr. Smith's office",
        dueDate: new Date("2024-02-15T14:30:00Z"),
        index: 0,
        reminderListId: null,
        reminderItems: [],
    },
    work: {
        name: "Quarterly report",
        description: "Prepare Q4 2023 financial report for board meeting",
        dueDate: new Date("2024-01-31T23:59:59Z"),
        index: 0,
        reminderListId: "234567890123456789", // Work reminders list
        reminderItems: [
            {
                name: "Gather financial data",
                description: "",
                isComplete: false,
                index: 0,
            },
            {
                name: "Create charts and graphs",
                description: "",
                isComplete: false,
                index: 1,
            },
            {
                name: "Write executive summary",
                description: "",
                isComplete: false,
                index: 2,
            },
        ],
    },
    recurring: {
        name: "Weekly team meeting prep",
        description: "Prepare agenda and materials for Monday team meeting",
        dueDate: new Date("2024-01-22T09:00:00Z"),
        index: 0,
        reminderListId: "345678901234567890", // Recurring tasks list
        reminderItems: [
            {
                name: "Review previous action items",
                description: "",
                isComplete: false,
                index: 0,
            },
            {
                name: "Update project status",
                description: "",
                isComplete: false,
                index: 1,
            },
        ],
    },
    urgent: {
        name: "Submit tax documents",
        description: "Submit all required tax documents before deadline",
        dueDate: new Date("2024-04-15T23:59:59Z"),
        index: 0,
        reminderListId: null,
        reminderItems: [
            {
                name: "Gather W-2 forms",
                description: "Collect from all employers",
                isComplete: true,
                index: 0,
            },
            {
                name: "Collect 1099 forms",
                description: "From freelance work",
                isComplete: false,
                index: 1,
            },
            {
                name: "Schedule CPA appointment",
                description: "",
                isComplete: false,
                index: 2,
            },
        ],
    },
};

/**
 * Reminder item form data
 */
export const minimalReminderItemFormInput = {
    name: "Simple task",
    description: "",
    isComplete: false,
    index: 0,
};

export const completeReminderItemFormInput = {
    name: "Complex task with details",
    description: "This task has a detailed description explaining what needs to be done and how to approach it",
    isComplete: false,
    index: 0,
};

/**
 * Form validation test cases
 */
export const invalidReminderFormInputs = {
    missingName: {
        name: "", // Required field
        description: "This reminder has no name",
        dueDate: null,
        index: 0,
    },
    nameTooShort: {
        name: "AB", // Too short (min 3 chars based on API fixture)
        description: "Valid description",
        dueDate: null,
        index: 0,
    },
    nameTooLong: {
        name: "A".repeat(51), // Too long (max 50 chars)
        description: "Valid description",
        dueDate: null,
        index: 0,
    },
    descriptionTooLong: {
        name: "Valid reminder",
        description: "A".repeat(2049), // Too long (max 2048 chars)
        dueDate: null,
        index: 0,
    },
    negativeIndex: {
        name: "Valid reminder",
        description: "Valid description",
        dueDate: null,
        index: -1, // Should be non-negative
    },
    invalidDate: {
        name: "Valid reminder",
        description: "Valid description",
        // @ts-expect-error - Testing invalid input
        dueDate: "not-a-date",
        index: 0,
    },
    invalidReminderItem: {
        name: "Valid reminder",
        description: "Valid description",
        dueDate: null,
        index: 0,
        reminderItems: [
            {
                name: "", // Missing required name
                description: "Item without name",
                isComplete: false,
                index: 0,
            },
        ],
    },
};

/**
 * Form validation states
 */
export const reminderFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "AB", // Too short
            description: "A".repeat(2049), // Too long
            dueDate: new Date("2024-01-01"),
            index: -1, // Negative
        },
        errors: {
            name: "Reminder name must be at least 3 characters",
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
        values: minimalReminderCreateFormInput,
        errors: {},
        touched: {
            name: true,
            description: true,
            dueDate: true,
            index: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeReminderCreateFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create reminder form initial values
 */
export const createReminderFormInitialValues = (reminderData?: Partial<any>) => ({
    name: reminderData?.name || "",
    description: reminderData?.description || "",
    dueDate: reminderData?.dueDate || null,
    index: reminderData?.index || 0,
    reminderListId: reminderData?.reminderList?.id || null,
    reminderItems: reminderData?.reminderItems || [],
    ...reminderData,
});

/**
 * Helper function to transform form data to API format (ReminderShape)
 */
export const transformReminderFormToApiInput = (formData: any): ReminderShape => ({
    __typename: "Reminder",
    id: formData.id || DUMMY_ID,
    name: formData.name,
    description: formData.description || undefined,
    dueDate: formData.dueDate ? new Date(formData.dueDate) : undefined,
    index: formData.index,
    reminderList: formData.createNewList && formData.newListName
        ? {
            __typename: "ReminderList",
            id: DUMMY_ID,
            label: formData.newListName,
        }
        : formData.reminderListId
        ? {
            __typename: "ReminderList",
            __connect: true,
            id: formData.reminderListId,
        }
        : null,
    reminderItems: formData.reminderItems?.map((item: any, index: number): ReminderItemShape => ({
        __typename: "ReminderItem",
        id: item.id || DUMMY_ID,
        name: item.name,
        description: item.description || undefined,
        isComplete: item.isComplete || false,
        index: item.index !== undefined ? item.index : index,
        reminder: {
            __typename: "Reminder",
            __connect: true,
            id: formData.id || DUMMY_ID,
        },
    })) || [],
});

/**
 * Helper function to validate reminder due date
 */
export const validateReminderDueDate = (dueDate: Date | string | null): string | null => {
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
 * Mock autocomplete suggestions
 */
export const mockReminderSuggestions = {
    reminderLists: [
        { id: "123456789012345678", label: "Personal Tasks" },
        { id: "234567890123456789", label: "Work Tasks" },
        { id: "345678901234567890", label: "Shopping Lists" },
        { id: "456789012345678901", label: "Project Milestones" },
    ],
    quickAddTemplates: [
        { name: "Call", icon: "phone" },
        { name: "Email", icon: "email" },
        { name: "Meeting with", icon: "group" },
        { name: "Buy", icon: "shopping_cart" },
        { name: "Review", icon: "rate_review" },
        { name: "Submit", icon: "send" },
    ],
    dueDatePresets: [
        { label: "Today", value: () => new Date() },
        { label: "Tomorrow", value: () => new Date(Date.now() + 24 * 60 * 60 * 1000) },
        { label: "Next Week", value: () => new Date(Date.now() + 7 * 24 * 60 * 60 * 1000) },
        { label: "Next Month", value: () => new Date(Date.now() + 30 * 24 * 60 * 60 * 1000) },
        { label: "End of Year", value: () => new Date(new Date().getFullYear(), 11, 31) },
    ],
};

/**
 * Reminder item completion states for testing
 */
export const reminderItemStates = {
    allIncomplete: [
        { name: "Task 1", isComplete: false, index: 0 },
        { name: "Task 2", isComplete: false, index: 1 },
        { name: "Task 3", isComplete: false, index: 2 },
    ],
    partiallyComplete: [
        { name: "Task 1", isComplete: true, index: 0 },
        { name: "Task 2", isComplete: false, index: 1 },
        { name: "Task 3", isComplete: true, index: 2 },
    ],
    allComplete: [
        { name: "Task 1", isComplete: true, index: 0 },
        { name: "Task 2", isComplete: true, index: 1 },
        { name: "Task 3", isComplete: true, index: 2 },
    ],
};