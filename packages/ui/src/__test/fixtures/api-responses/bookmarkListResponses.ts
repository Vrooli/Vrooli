import { type BookmarkList, type BookmarkListSearchResult, type BookmarkListEdge, type Bookmark, BookmarkFor } from "@vrooli/shared";
import { minimalBookmarkResponse, completeBookmarkResponse, bookmarkResponseVariants } from "./bookmarkResponses.js";
import { minimalUserResponse } from "./userResponses.js";

/**
 * API response fixtures for bookmark lists
 * These represent what components receive from API calls
 */

/**
 * Minimal bookmark list API response
 */
export const minimalBookmarkListResponse: BookmarkList = {
    __typename: "BookmarkList",
    id: "list_123456789012345678",
    label: "My Bookmarks",
    bookmarks: [],
    bookmarksCount: 0,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
};

/**
 * Complete bookmark list API response with bookmarks
 */
export const completeBookmarkListResponse: BookmarkList = {
    __typename: "BookmarkList",
    id: "list_987654321098765432",
    label: "My Favorite Resources",
    bookmarks: [
        bookmarkResponseVariants[BookmarkFor.Resource],
        bookmarkResponseVariants[BookmarkFor.User],
        bookmarkResponseVariants[BookmarkFor.Team],
        {
            ...minimalBookmarkResponse,
            id: "bookmark_additional_123456789012345678",
            to: {
                __typename: "Tag",
                id: "tag_favorite_123456789012345678",
                tag: "machine-learning",
                createdAt: "2024-01-01T00:00:00Z",
                updatedAt: "2024-01-01T00:00:00Z",
                you: {
                    __typename: "TagYou",
                    canUpdate: false,
                },
            } as any,
        },
    ],
    bookmarksCount: 4,
    createdAt: "2023-12-15T10:30:00Z",
    updatedAt: "2024-01-20T14:45:00Z",
};

/**
 * Bookmark list variants for different scenarios
 */
export const bookmarkListResponseVariants = {
    minimal: minimalBookmarkListResponse,
    complete: completeBookmarkListResponse,
    
    /**
     * Empty list (no bookmarks)
     */
    empty: {
        __typename: "BookmarkList",
        id: "list_empty_123456789012345678",
        label: "Empty List",
        bookmarks: [],
        bookmarksCount: 0,
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-01T00:00:00Z",
    },

    /**
     * Large populated list
     */
    populated: {
        __typename: "BookmarkList",
        id: "list_populated_123456789012345678",
        label: "Comprehensive Collection",
        bookmarks: [
            bookmarkResponseVariants[BookmarkFor.Resource],
            bookmarkResponseVariants[BookmarkFor.User],
            bookmarkResponseVariants[BookmarkFor.Team],
            bookmarkResponseVariants[BookmarkFor.Tag],
            bookmarkResponseVariants[BookmarkFor.Comment],
            bookmarkResponseVariants[BookmarkFor.Issue],
        ],
        bookmarksCount: 6,
        createdAt: "2023-11-01T00:00:00Z",
        updatedAt: "2024-01-25T09:15:00Z",
    },

    /**
     * Single bookmark list
     */
    single: {
        __typename: "BookmarkList",
        id: "list_single_123456789012345678",
        label: "Quick Save",
        bookmarks: [bookmarkResponseVariants[BookmarkFor.Resource]],
        bookmarksCount: 1,
        createdAt: "2024-01-20T08:00:00Z",
        updatedAt: "2024-01-20T08:00:00Z",
    },

    /**
     * Private bookmark list
     */
    private: {
        __typename: "BookmarkList",
        id: "list_private_123456789012345678",
        label: "Private Collection",
        bookmarks: [
            {
                ...bookmarkResponseVariants[BookmarkFor.User],
                id: "bookmark_private_123456789012345678",
                to: {
                    ...minimalUserResponse,
                    id: "user_private_123456789012345678",
                    handle: "privateuser",
                    name: "Private User",
                    isPrivate: true,
                },
            },
        ],
        bookmarksCount: 1,
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-01T00:00:00Z",
    },

    /**
     * Recently updated list
     */
    recentlyUpdated: {
        __typename: "BookmarkList",
        id: "list_recent_123456789012345678",
        label: "Recently Updated",
        bookmarks: [
            bookmarkResponseVariants[BookmarkFor.Resource],
            bookmarkResponseVariants[BookmarkFor.Team],
        ],
        bookmarksCount: 2,
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: new Date().toISOString(), // Current timestamp
    },

    /**
     * Mixed content types list
     */
    mixed: {
        __typename: "BookmarkList",
        id: "list_mixed_123456789012345678",
        label: "Mixed Content",
        bookmarks: [
            bookmarkResponseVariants[BookmarkFor.User],
            bookmarkResponseVariants[BookmarkFor.Resource],
            bookmarkResponseVariants[BookmarkFor.Tag],
        ],
        bookmarksCount: 3,
        createdAt: "2024-01-10T00:00:00Z",
        updatedAt: "2024-01-15T12:00:00Z",
    },

    /**
     * Work-related bookmarks
     */
    work: {
        __typename: "BookmarkList",
        id: "list_work_123456789012345678",
        label: "Work Resources",
        bookmarks: [
            {
                ...bookmarkResponseVariants[BookmarkFor.Resource],
                id: "bookmark_work_resource_123456789012345678",
                to: {
                    __typename: "Resource",
                    id: "resource_work_123456789012345678",
                    index: 0,
                    link: "https://docs.company.com/api",
                    usedFor: "Api",
                    createdAt: "2024-01-01T00:00:00Z",
                    updatedAt: "2024-01-01T00:00:00Z",
                    you: {
                        __typename: "ResourceYou",
                        canDelete: true,
                        canUpdate: true,
                    },
                },
            },
            {
                ...bookmarkResponseVariants[BookmarkFor.Team],
                id: "bookmark_work_team_123456789012345678",
                to: {
                    __typename: "Team",
                    id: "team_work_123456789012345678",
                    handle: "engineering-team",
                    name: "Engineering Team",
                    isPrivate: false,
                    createdAt: "2024-01-01T00:00:00Z",
                    updatedAt: "2024-01-01T00:00:00Z",
                    you: {
                        __typename: "TeamYou",
                        canDelete: false,
                        canReport: false,
                        canUpdate: true,
                        isBookmarked: true,
                        isReported: false,
                        reaction: null,
                    },
                } as any,
            },
        ],
        bookmarksCount: 2,
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-18T16:30:00Z",
    },
} as const;

/**
 * Bookmark list search response for paginated results
 */
export const bookmarkListSearchResponse: BookmarkListSearchResult = {
    __typename: "BookmarkListSearchResult",
    edges: [
        {
            __typename: "BookmarkListEdge",
            cursor: "cursor_1",
            node: bookmarkListResponseVariants.complete,
        },
        {
            __typename: "BookmarkListEdge",
            cursor: "cursor_2",
            node: bookmarkListResponseVariants.populated,
        },
        {
            __typename: "BookmarkListEdge",
            cursor: "cursor_3",
            node: bookmarkListResponseVariants.work,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_3",
    },
};

/**
 * Empty search results
 */
export const emptyBookmarkListSearchResponse: BookmarkListSearchResult = {
    __typename: "BookmarkListSearchResult",
    edges: [],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: null,
        endCursor: null,
    },
};

/**
 * Single page search results
 */
export const singlePageBookmarkListSearchResponse: BookmarkListSearchResult = {
    __typename: "BookmarkListSearchResult",
    edges: [
        {
            __typename: "BookmarkListEdge",
            cursor: "cursor_1",
            node: bookmarkListResponseVariants.minimal,
        },
        {
            __typename: "BookmarkListEdge",
            cursor: "cursor_2",
            node: bookmarkListResponseVariants.single,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: false,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_2",
    },
};

/**
 * Bookmark lists organized by common use cases
 */
export const bookmarkListUseCases = {
    /**
     * Personal organization lists
     */
    personal: [
        bookmarkListResponseVariants.minimal,
        bookmarkListResponseVariants.complete,
        bookmarkListResponseVariants.mixed,
    ],
    
    /**
     * Work-related lists
     */
    professional: [
        bookmarkListResponseVariants.work,
        {
            ...bookmarkListResponseVariants.populated,
            label: "Project References",
        },
    ],
    
    /**
     * Learning and educational lists
     */
    educational: [
        {
            ...bookmarkListResponseVariants.populated,
            id: "list_learning_123456789012345678",
            label: "Learning Resources",
            bookmarks: [
                bookmarkResponseVariants[BookmarkFor.Resource],
                bookmarkResponseVariants[BookmarkFor.Tag],
            ],
            bookmarksCount: 2,
        },
    ],
    
    /**
     * Empty and edge case lists
     */
    edgeCases: [
        bookmarkListResponseVariants.empty,
        bookmarkListResponseVariants.private,
        bookmarkListResponseVariants.recentlyUpdated,
    ],
};

/**
 * Loading and error states for UI testing
 */
export const bookmarkListUIStates = {
    loading: null,
    error: {
        code: "BOOKMARK_LIST_NOT_FOUND",
        message: "The requested bookmark list could not be found",
    },
    empty: emptyBookmarkListSearchResponse,
    networkError: {
        code: "NETWORK_ERROR",
        message: "Unable to load bookmark lists. Please check your connection.",
    },
    forbidden: {
        code: "FORBIDDEN",
        message: "You don't have permission to access this bookmark list",
    },
};