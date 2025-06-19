import { BookmarkFor, DUMMY_ID, type BookmarkShape, type BookmarkListShape } from "@vrooli/shared";

/**
 * Form data fixtures for bookmark creation and editing
 * 
 * Bookmarks are a unique case - they're created via button click, not forms.
 * We define both:
 * 1. BookmarkFormData - simplified UI state for the button/dialog interaction
 * 2. BookmarkShape fixtures - the real shape used by shapeBookmark.create()
 */

// Simplified interface for UI interaction (button click + list selection)
export interface BookmarkFormData {
    bookmarkFor: BookmarkFor;
    forConnect: string;
    listId?: string;
    createNewList?: boolean;
    newListLabel?: string;
}

// Helper to transform UI state to BookmarkShape
export function bookmarkFormDataToShape(formData: BookmarkFormData): BookmarkShape {
    return {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: {
            __typename: formData.bookmarkFor,
            id: formData.forConnect,
        },
        list: formData.createNewList && formData.newListLabel ? {
            __typename: "BookmarkList",
            id: DUMMY_ID,
            label: formData.newListLabel,
        } : formData.listId ? {
            __typename: "BookmarkList",
            __connect: true,
            id: formData.listId,
        } : null,
    };
}

/**
 * Minimal bookmark form input - just bookmarking a resource
 */
export const minimalBookmarkFormInput: BookmarkShape = {
    __typename: "Bookmark",
    id: DUMMY_ID,
    to: {
        __typename: BookmarkFor.Resource,
        id: "123456789012345678", // Resource ID (18 digits)
    },
    list: null, // No list for minimal bookmark
};

/**
 * Complete bookmark form input with new list creation
 */
export const completeBookmarkFormInput: BookmarkShape = {
    __typename: "Bookmark",
    id: DUMMY_ID,
    to: {
        __typename: BookmarkFor.User,
        id: "987654321098765432", // User ID (18 digits)
    },
    list: {
        __typename: "BookmarkList",
        id: DUMMY_ID,
        label: "My Favorite Users",
    },
};

/**
 * Bookmark form input with existing list
 */
export const bookmarkWithExistingListFormInput: BookmarkShape = {
    __typename: "Bookmark",
    id: DUMMY_ID,
    to: {
        __typename: BookmarkFor.Team,
        id: "111222333444555666", // Team ID (18 digits)
    },
    list: {
        __typename: "BookmarkList",
        __connect: true,
        id: "777888999000111222", // Existing list ID (18 digits)
    },
};

/**
 * All possible bookmark types for testing
 */
export const bookmarkFormVariants: Record<BookmarkFor, BookmarkShape> = {
    [BookmarkFor.Comment]: {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: { __typename: BookmarkFor.Comment, id: "123456789012345678" },
        list: null,
    },
    [BookmarkFor.Issue]: {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: { __typename: BookmarkFor.Issue, id: "234567890123456789" },
        list: null,
    },
    [BookmarkFor.Resource]: {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: { __typename: BookmarkFor.Resource, id: "345678901234567890" },
        list: null,
    },
    [BookmarkFor.Tag]: {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: { __typename: BookmarkFor.Tag, id: "456789012345678901" },
        list: null,
    },
    [BookmarkFor.Team]: {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: { __typename: BookmarkFor.Team, id: "567890123456789012" },
        list: null,
    },
    [BookmarkFor.User]: {
        __typename: "Bookmark",
        id: DUMMY_ID,
        to: { __typename: BookmarkFor.User, id: "678901234567890123" },
        list: null,
    },
};

/**
 * Form validation test cases
 */
export const invalidBookmarkFormInputs = {
    missingForConnect: {
        bookmarkFor: BookmarkFor.Resource,
        forConnect: "", // Invalid: empty string
    },
    missingBookmarkFor: {
        // @ts-expect-error - Testing invalid input
        bookmarkFor: undefined,
        forConnect: "123456789012345678",
    },
    invalidId: {
        bookmarkFor: BookmarkFor.Resource,
        forConnect: "invalid-id", // Invalid: not a snowflake ID
    },
    newListWithoutLabel: {
        bookmarkFor: BookmarkFor.Resource,
        forConnect: "123456789012345678",
        createNewList: true,
        newListLabel: "", // Invalid: empty label when creating list
    },
};