import { type BookmarkCreateInput, type Bookmark, generatePK } from "@vrooli/shared";
import { type BookmarkFormData } from "../form-data/bookmarkFormData.js";

/**
 * Transformation utilities for converting between different bookmark data formats
 * These functions handle the data flow between UI forms, API requests, and responses
 */

/**
 * Transform form data to API create request
 * This represents what happens when a user submits a bookmark form
 */
export function transformFormToCreateRequest(formData: BookmarkFormData): BookmarkCreateInput {
    const request: BookmarkCreateInput = {
        id: generatePK().toString(),
        bookmarkFor: formData.bookmarkFor,
        forConnect: formData.forConnect,
    };

    // Handle list creation or connection
    if (formData.createNewList && formData.newListLabel) {
        request.listCreate = {
            id: generatePK().toString(),
            label: formData.newListLabel,
        };
    } else if (formData.listId) {
        request.listConnect = formData.listId;
    }

    return request;
}

/**
 * Transform API response back to form data
 * This would be used when editing an existing bookmark
 */
export function transformApiResponseToForm(bookmark: Bookmark): BookmarkFormData {
    // Determine bookmarkFor based on the __typename of the 'to' object
    const bookmarkFor = bookmark.to.__typename as any; // This should match BookmarkFor enum
    
    return {
        bookmarkFor,
        forConnect: bookmark.to.id,
        listId: bookmark.list?.id,
        createNewList: false, // Existing bookmark, so no new list creation
    };
}

/**
 * Transform form data to update request
 * Used when updating bookmark list assignment
 */
export function transformFormToUpdateRequest(
    bookmarkId: string, 
    formData: Partial<BookmarkFormData>
): { id: string; listConnect?: string } {
    const request: { id: string; listConnect?: string } = {
        id: bookmarkId,
    };

    if (formData.listId) {
        request.listConnect = formData.listId;
    }

    return request;
}

/**
 * Validate form data before submission
 * Returns array of validation errors, empty if valid
 */
export function validateBookmarkFormData(formData: BookmarkFormData): string[] {
    const errors: string[] = [];

    if (!formData.bookmarkFor) {
        errors.push("Bookmark type is required");
    }

    if (!formData.forConnect || formData.forConnect.trim() === "") {
        errors.push("Target object ID is required");
    }

    // Validate ID format (should be snowflake-like)
    if (formData.forConnect && !/^\d{10,19}$/.test(formData.forConnect)) {
        errors.push("Invalid object ID format");
    }

    if (formData.createNewList && (!formData.newListLabel || formData.newListLabel.trim() === "")) {
        errors.push("List label is required when creating a new list");
    }

    return errors;
}

/**
 * Helper to determine if two bookmark form data objects represent the same bookmark
 * Useful for testing data consistency
 */
export function areBookmarkFormsEqual(
    form1: BookmarkFormData, 
    form2: BookmarkFormData
): boolean {
    return (
        form1.bookmarkFor === form2.bookmarkFor &&
        form1.forConnect === form2.forConnect &&
        form1.listId === form2.listId
    );
}

/**
 * Mock API service functions for testing
 * These simulate the actual API calls that would be made
 */
export const mockBookmarkService = {
    /**
     * Simulate creating a bookmark via API
     */
    async create(request: BookmarkCreateInput): Promise<Bookmark> {
        // Simulate API delay
        await new Promise(resolve => setTimeout(resolve, 100));
        
        // Create the bookmark response
        const bookmark: Bookmark = {
            __typename: "Bookmark",
            id: request.id,
            by: {
                __typename: "User",
                id: "current_user_123456789012345678",
                handle: "currentuser",
                name: "Current User",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                status: "Unlocked",
                bannerImage: null,
                profileImage: null,
                theme: "light",
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                awardedCount: 0,
                bookmarksCount: 1,
                commentsCount: 0,
                issuesCount: 0,
                postsCount: 0,
                projectsCount: 0,
                pullRequestsCount: 0,
                questionsCount: 0,
                quizzesCount: 0,
                reportsReceivedCount: 0,
                routinesCount: 0,
                standardsCount: 0,
                teamsCount: 0,
                translations: [],
                you: {
                    __typename: "UserYou",
                    canDelete: true,
                    canReport: false,
                    canUpdate: true,
                    isBookmarked: false,
                    isReported: false,
                    reaction: null,
                },
            },
            to: {
                __typename: request.bookmarkFor,
                id: request.forConnect,
                // Add minimal fields based on type
                ...(request.bookmarkFor === "User" && {
                    handle: "targetuser",
                    name: "Target User",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    status: "Unlocked",
                }),
                ...(request.bookmarkFor === "Resource" && {
                    link: "https://example.com/resource",
                    usedFor: "Api",
                    index: 0,
                }),
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
            } as any, // Type assertion since BookmarkTo is a union type
            list: request.listCreate ? {
                __typename: "BookmarkList",
                id: request.listCreate.id,
                label: request.listCreate.label,
                bookmarks: [],
                bookmarksCount: 1,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
            } : null,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
        };

        // Store in global test storage for retrieval by findById
        const storage = (globalThis as any).__testBookmarkStorage || {};
        storage[request.id] = JSON.parse(JSON.stringify(bookmark)); // Store a deep copy
        (globalThis as any).__testBookmarkStorage = storage;
        
        return bookmark;
    },

    /**
     * Simulate fetching a bookmark by ID
     * In a real implementation, this would fetch from database by ID
     */
    async findById(id: string): Promise<Bookmark> {
        await new Promise(resolve => setTimeout(resolve, 50));
        
        // For testing, we'll return the same bookmark that was created
        // In a real app, this would fetch from storage by ID
        // We need to maintain state or use the same creation parameters
        const storedBookmarks = (globalThis as any).__testBookmarkStorage || {};
        if (storedBookmarks[id]) {
            // Return a deep copy to prevent mutations
            return JSON.parse(JSON.stringify(storedBookmarks[id]));
        }
        
        // Fallback for testing - create a minimal response
        return {
            __typename: "Bookmark",
            id,
            by: {
                __typename: "User",
                id: "current_user_123456789012345678",
                handle: "currentuser",
                name: "Current User",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                status: "Unlocked",
                bannerImage: null,
                profileImage: null,
                theme: "light",
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                awardedCount: 0,
                bookmarksCount: 1,
                commentsCount: 0,
                issuesCount: 0,
                postsCount: 0,
                projectsCount: 0,
                pullRequestsCount: 0,
                questionsCount: 0,
                quizzesCount: 0,
                reportsReceivedCount: 0,
                routinesCount: 0,
                standardsCount: 0,
                teamsCount: 0,
                translations: [],
                you: {
                    __typename: "UserYou",
                    canDelete: true,
                    canReport: false,
                    canUpdate: true,
                    isBookmarked: false,
                    isReported: false,
                    reaction: null,
                },
            },
            to: {
                __typename: "Resource",
                id: "123456789012345678",
                link: "https://example.com/resource",
                usedFor: "Api",
                index: 0,
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                you: {
                    __typename: "ResourceYou",
                    canDelete: false,
                    canUpdate: false,
                },
            } as any,
            list: null,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
        };
    },

    /**
     * Simulate updating a bookmark
     */
    async update(id: string, updates: { listConnect?: string }): Promise<Bookmark> {
        await new Promise(resolve => setTimeout(resolve, 75));
        
        const bookmark = await this.findById(id);
        
        // Create a deep copy to avoid mutating the original object
        const updatedBookmark = JSON.parse(JSON.stringify(bookmark));
        
        if (updates.listConnect) {
            updatedBookmark.list = {
                __typename: "BookmarkList",
                id: updates.listConnect,
                label: "Updated List",
                bookmarks: [],
                bookmarksCount: 1,
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: new Date().toISOString(),
            };
        }
        
        updatedBookmark.updatedAt = new Date().toISOString();
        
        // Update in storage
        const storage = (globalThis as any).__testBookmarkStorage || {};
        storage[id] = updatedBookmark;
        (globalThis as any).__testBookmarkStorage = storage;
        
        return updatedBookmark;
    },

    /**
     * Simulate deleting a bookmark
     */
    async delete(id: string): Promise<{ success: boolean }> {
        await new Promise(resolve => setTimeout(resolve, 25));
        return { success: true };
    },
};