/**
 * Form data fixtures for reminder list creation and editing
 * These represent data as it appears in form state before submission
 */

import { DUMMY_ID } from "@vrooli/shared";

/**
 * Minimal reminder list form input
 */
export const minimalReminderListFormInput = {
    label: "My Reminders",
};

/**
 * Complete reminder list form input with description
 */
export const completeReminderListFormInput = {
    label: "Project Milestones",
    description: "Track important project milestones and deadlines for the Q1 2024 release",
};

/**
 * Reminder list update form data
 */
export const minimalReminderListUpdateFormInput = {
    label: "Updated Reminders",
};

export const completeReminderListUpdateFormInput = {
    label: "Updated Project Tasks",
    description: "Updated milestone tracking with new deliverables and adjusted timelines for Q2 2024",
};

/**
 * Reminder list with initial reminders
 */
export const reminderListWithRemindersFormInput = {
    label: "Weekly Tasks",
    description: "Recurring weekly tasks and responsibilities",
    initialReminders: [
        {
            name: "Team standup",
            description: "Daily team sync meeting",
            dueDate: new Date("2024-01-22T10:00:00Z"),
            index: 0,
        },
        {
            name: "Weekly report",
            description: "Submit weekly progress report",
            dueDate: new Date("2024-01-26T17:00:00Z"),
            index: 1,
        },
        {
            name: "Code review",
            description: "Review pending pull requests",
            dueDate: null,
            index: 2,
        },
    ],
};

/**
 * Reminder list with nested reminder items
 */
export const reminderListWithNestedItemsFormInput = {
    label: "Release Checklist",
    description: "Complete checklist for product releases",
    initialReminders: [
        {
            name: "Pre-release testing",
            description: "Complete all testing before release",
            dueDate: new Date("2024-02-01T12:00:00Z"),
            index: 0,
            reminderItems: [
                {
                    name: "Unit tests pass",
                    description: "",
                    isComplete: true,
                    index: 0,
                },
                {
                    name: "Integration tests pass",
                    description: "",
                    isComplete: false,
                    index: 1,
                },
                {
                    name: "E2E tests pass",
                    description: "",
                    isComplete: false,
                    index: 2,
                },
            ],
        },
        {
            name: "Documentation update",
            description: "Update all documentation",
            dueDate: new Date("2024-02-02T12:00:00Z"),
            index: 1,
            reminderItems: [
                {
                    name: "API docs",
                    description: "Update API documentation",
                    isComplete: false,
                    index: 0,
                },
                {
                    name: "User guide",
                    description: "Update user guide with new features",
                    isComplete: false,
                    index: 1,
                },
            ],
        },
    ],
};

/**
 * Private reminder list form input
 */
export const privateReminderListFormInput = {
    label: "Personal Goals",
    description: "My personal development goals and milestones",
    isPrivate: true,
};

/**
 * Shared reminder list form input
 */
export const sharedReminderListFormInput = {
    label: "Team Deadlines",
    description: "Shared deadlines and milestones for the entire team",
    isPrivate: false,
    shareWithTeams: ["team_123456789012345678", "team_234567890123456789"],
};

/**
 * Multiple reminder list creation (batch)
 */
export const batchReminderListFormInput = [
    {
        label: "Daily Tasks",
        description: "Tasks to complete every day",
    },
    {
        label: "Monthly Goals",
        description: "Goals to achieve each month",
    },
    {
        label: "Project Deadlines",
        description: "Important project milestone dates",
    },
];

/**
 * Different reminder list types for various use cases
 */
export const reminderListFormVariants = {
    personal: {
        label: "Personal Tasks",
        description: "My personal to-do list and reminders",
        isPrivate: true,
    },
    work: {
        label: "Work Projects",
        description: "Work-related tasks and project deadlines",
        isPrivate: false,
    },
    shopping: {
        label: "Shopping Lists",
        description: "Things to buy and shopping reminders",
        isPrivate: true,
    },
    household: {
        label: "Household Chores",
        description: "Regular household maintenance and chores",
        isPrivate: true,
    },
    fitness: {
        label: "Fitness Goals",
        description: "Workout schedule and fitness milestones",
        isPrivate: true,
    },
};

/**
 * Form validation test cases
 */
export const invalidReminderListFormInputs = {
    missingLabel: {
        label: "", // Required field
        description: "This should fail validation",
    },
    labelTooLong: {
        label: "x".repeat(129), // Max 128 characters
        description: "Valid description",
    },
    labelTooShort: {
        label: "ab", // Min 3 characters typically
        description: "Valid description",
    },
    onlyWhitespace: {
        label: "   ", // Should be trimmed and fail
        description: "Valid description",
    },
    descriptionTooLong: {
        label: "Valid List",
        description: "x".repeat(2049), // Max 2048 characters
    },
};

/**
 * Form state examples
 */
export const reminderListFormStates = {
    pristine: {
        values: {
            label: "",
            description: "",
            initialReminders: [],
        },
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            label: "", // Empty required field
            description: "Some description",
            initialReminders: [],
        },
        errors: {
            label: "Label is required",
        },
        touched: {
            label: true,
            description: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalReminderListFormInput,
        errors: {},
        touched: {
            label: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeReminderListFormInput,
        errors: {},
        touched: {
            label: true,
            description: true,
        },
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create form initial values
 */
export const createReminderListFormInitialValues = (listData?: Partial<any>) => ({
    label: listData?.label || "",
    description: listData?.description || "",
    isPrivate: listData?.isPrivate || false,
    initialReminders: listData?.reminders || [],
    ...listData,
});

/**
 * Helper function to validate reminder list label
 */
export const validateReminderListLabel = (label: string): string | null => {
    if (!label || !label.trim()) return "Label is required";
    if (label.trim().length < 3) return "Label must be at least 3 characters";
    if (label.length > 128) return "Label must be less than 128 characters";
    return null;
};

/**
 * Helper function to validate reminder list description
 */
export const validateReminderListDescription = (description: string): string | null => {
    if (description && description.length > 2048) {
        return "Description must be less than 2048 characters";
    }
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformReminderListFormToApiInput = (formData: any) => ({
    id: formData.id || DUMMY_ID,
    label: formData.label.trim(),
    ...(formData.description && { description: formData.description.trim() }),
    ...(formData.isPrivate !== undefined && { isPrivate: formData.isPrivate }),
    ...(formData.initialReminders && {
        remindersCreate: formData.initialReminders.map((reminder: any, index: number) => ({
            id: DUMMY_ID,
            name: reminder.name,
            description: reminder.description || undefined,
            dueDate: reminder.dueDate || undefined,
            index: reminder.index !== undefined ? reminder.index : index,
            ...(reminder.reminderItems && {
                reminderItemsCreate: reminder.reminderItems.map((item: any, itemIndex: number) => ({
                    id: DUMMY_ID,
                    name: item.name,
                    description: item.description || undefined,
                    isComplete: item.isComplete || false,
                    index: item.index !== undefined ? item.index : itemIndex,
                    reminderConnect: DUMMY_ID,
                })),
            }),
        })),
    }),
});

/**
 * Mock data for autocomplete/suggestions
 */
export const mockReminderListSuggestions = {
    labels: [
        "Daily Tasks",
        "Weekly Goals",
        "Shopping List",
        "Project Deadlines",
        "Personal Goals",
    ],
    recentlyUsedLists: [
        { id: "list_123456789012345678", label: "Work Tasks", reminderCount: 15 },
        { id: "list_234567890123456789", label: "Home Projects", reminderCount: 8 },
        { id: "list_345678901234567890", label: "Shopping", reminderCount: 12 },
    ],
    templates: [
        {
            label: "Weekly Review Template",
            description: "Standard weekly review checklist",
            reminders: [
                { name: "Review last week's goals", index: 0 },
                { name: "Plan this week's priorities", index: 1 },
                { name: "Update project status", index: 2 },
            ],
        },
        {
            label: "Release Checklist Template",
            description: "Standard release preparation checklist",
            reminders: [
                { name: "Code freeze", index: 0 },
                { name: "Run all tests", index: 1 },
                { name: "Update documentation", index: 2 },
                { name: "Deploy to staging", index: 3 },
                { name: "Final approval", index: 4 },
            ],
        },
    ],
};

/**
 * Example form flow data
 */
export const reminderListCreationFlow = {
    step1_initial: {
        label: "",
        description: "",
        initialReminders: [],
    },
    step2_labelEntered: {
        label: "My New List",
        description: "",
        initialReminders: [],
    },
    step3_descriptionAdded: {
        label: "My New List",
        description: "A list to track important tasks",
        initialReminders: [],
    },
    step4_remindersAdded: {
        label: "My New List",
        description: "A list to track important tasks",
        initialReminders: [
            { name: "First task", description: "", dueDate: null, index: 0 },
            { name: "Second task", description: "With details", dueDate: new Date("2024-02-15T10:00:00Z"), index: 1 },
        ],
    },
};

/**
 * Helper function to calculate list statistics
 */
export const calculateReminderListStats = (reminders: any[]) => {
    const total = reminders.length;
    const completed = reminders.filter(r => 
        r.reminderItems?.length > 0 
            ? r.reminderItems.every((item: any) => item.isComplete)
            : false
    ).length;
    const overdue = reminders.filter(r => 
        r.dueDate && new Date(r.dueDate) < new Date()
    ).length;
    
    return {
        total,
        completed,
        pending: total - completed,
        overdue,
        completionRate: total > 0 ? (completed / total) * 100 : 0,
    };
};

/**
 * Helper function to sort reminders by various criteria
 */
export const sortReminders = (reminders: any[], sortBy: 'index' | 'dueDate' | 'name' | 'created') => {
    return [...reminders].sort((a, b) => {
        switch (sortBy) {
            case 'index':
                return (a.index || 0) - (b.index || 0);
            case 'dueDate':
                if (!a.dueDate) return 1;
                if (!b.dueDate) return -1;
                return new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime();
            case 'name':
                return a.name.localeCompare(b.name);
            case 'created':
                return new Date(b.createdAt || 0).getTime() - new Date(a.createdAt || 0).getTime();
            default:
                return 0;
        }
    });
};