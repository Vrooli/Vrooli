/**
 * Form data fixtures for bookmark list creation and editing
 * These represent data as it appears in form state before submission
 */

import { DUMMY_ID } from "@vrooli/shared";

/**
 * Minimal bookmark list form input - just the label
 */
export const minimalBookmarkListFormInput = {
    label: "My Bookmarks",
};

/**
 * Complete bookmark list form input with description
 */
export const completeBookmarkListFormInput = {
    label: "Favorite Resources",
    description: "A collection of my favorite learning resources, tools, and documentation for AI development",
};

/**
 * Bookmark list update form data
 */
export const minimalBookmarkListUpdateFormInput = {
    label: "Updated Bookmarks",
};

export const completeBookmarkListUpdateFormInput = {
    label: "Updated Resource Collection",
    description: "Updated collection with new resources and better organization. Now includes video tutorials and interactive examples.",
};

/**
 * Multiple bookmark list creation (batch)
 */
export const batchBookmarkListFormInput = [
    {
        label: "Development Tools",
        description: "Essential development tools and utilities",
    },
    {
        label: "Learning Resources",
        description: "Tutorials, courses, and documentation",
    },
    {
        label: "Team Favorites",
        description: "Resources recommended by team members",
    },
];

/**
 * Bookmark list with initial bookmarks
 */
export const bookmarkListWithBookmarksFormInput = {
    label: "Project Resources",
    description: "Resources for the current project",
    initialBookmarks: [
        {
            objectType: "Resource",
            objectId: "123456789012345678",
        },
        {
            objectType: "Tag",
            objectId: "234567890123456789",
        },
        {
            objectType: "User",
            objectId: "345678901234567890",
        },
    ],
};

/**
 * Private bookmark list form input
 */
export const privateBookmarkListFormInput = {
    label: "Private Collection",
    description: "My personal collection of resources",
    isPrivate: true,
};

/**
 * Shared bookmark list form input
 */
export const sharedBookmarkListFormInput = {
    label: "Team Resources",
    description: "Shared resources for the entire team",
    isPrivate: false,
    shareWithTeams: ["team_123456789012345678", "team_234567890123456789"],
};

/**
 * Form validation test cases
 */
export const invalidBookmarkListFormInputs = {
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
};

/**
 * Form state examples
 */
export const bookmarkListFormStates = {
    pristine: {
        values: {
            label: "",
            description: "",
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
        values: minimalBookmarkListFormInput,
        errors: {},
        touched: {
            label: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeBookmarkListFormInput,
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
export const createBookmarkListFormInitialValues = (listData?: Partial<any>) => ({
    label: listData?.label || "",
    description: listData?.description || "",
    isPrivate: listData?.isPrivate || false,
    ...listData,
});

/**
 * Helper function to validate bookmark list label
 */
export const validateBookmarkListLabel = (label: string): string | null => {
    if (!label || !label.trim()) return "Label is required";
    if (label.trim().length < 3) return "Label must be at least 3 characters";
    if (label.length > 128) return "Label must be less than 128 characters";
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformBookmarkListFormToApiInput = (formData: any) => ({
    id: DUMMY_ID,
    label: formData.label.trim(),
    ...(formData.description && { description: formData.description.trim() }),
    ...(formData.isPrivate !== undefined && { isPrivate: formData.isPrivate }),
    ...(formData.initialBookmarks && {
        bookmarksCreate: formData.initialBookmarks.map((bookmark: any) => ({
            id: DUMMY_ID,
            bookmarkFor: bookmark.objectType,
            forConnect: bookmark.objectId,
        })),
    }),
});

/**
 * Mock data for autocomplete/suggestions
 */
export const mockBookmarkListSuggestions = {
    labels: [
        "Frequently Used",
        "Learning Resources",
        "Development Tools",
        "Documentation",
        "Team Resources",
    ],
    recentlyUsedLists: [
        { id: "list_123456789012345678", label: "My Resources", bookmarkCount: 12 },
        { id: "list_234567890123456789", label: "Project Docs", bookmarkCount: 8 },
        { id: "list_345678901234567890", label: "Tutorials", bookmarkCount: 25 },
    ],
};

/**
 * Example form flow data
 */
export const bookmarkListCreationFlow = {
    step1_initial: {
        label: "",
        description: "",
    },
    step2_labelEntered: {
        label: "My New List",
        description: "",
    },
    step3_descriptionAdded: {
        label: "My New List",
        description: "A collection of useful resources",
    },
    step4_bookmarksAdded: {
        label: "My New List",
        description: "A collection of useful resources",
        initialBookmarks: [
            { objectType: "Resource", objectId: "123456789012345678" },
        ],
    },
};