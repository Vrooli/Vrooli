import { type Bookmark, type BookmarkList, BookmarkFor, type User, type Resource } from "@vrooli/shared";

/**
 * API response fixtures for bookmarks
 * These represent what components receive from API calls
 */

/**
 * Mock user data for bookmark relationships
 */
const mockUser: User = {
    __typename: "User",
    id: "user_123456789012345678",
    handle: "testuser",
    name: "Test User",
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: null,
    profileImage: null,
    theme: "light",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    // Add minimal required User fields
    awardedCount: 0,
    bookmarksCount: 0,
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
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * Mock resource data for bookmark target
 */
const mockResource: Resource = {
    __typename: "Resource",
    id: "resource_123456789012345678",
    index: 0,
    link: "https://example.com/resource",
    usedFor: "Api",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "ResourceYou", 
        canDelete: false,
        canUpdate: false,
    },
};

/**
 * Mock bookmark list for organization
 */
const mockBookmarkList: BookmarkList = {
    __typename: "BookmarkList",
    id: "list_123456789012345678",
    label: "My Resources",
    bookmarks: [],
    bookmarksCount: 1,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
};

/**
 * Minimal bookmark API response
 */
export const minimalBookmarkResponse: Bookmark = {
    __typename: "Bookmark",
    id: "bookmark_123456789012345678",
    by: mockUser,
    to: mockResource,
    list: mockBookmarkList,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
};

/**
 * Complete bookmark API response with all fields
 */
export const completeBookmarkResponse: Bookmark = {
    __typename: "Bookmark",
    id: "bookmark_987654321098765432",
    by: {
        ...mockUser,
        id: "user_987654321098765432",
        handle: "poweruser",
        name: "Power User",
        profileImage: "https://example.com/profile.jpg",
        you: {
            __typename: "UserYou",
            canDelete: false,
            canReport: false, // Can't report yourself
            canUpdate: true,  // Can update own profile
            isBookmarked: false,
            isReported: false,
            reaction: null,
        },
    },
    to: {
        ...mockUser,
        id: "user_111222333444555666", // Bookmarking another user
        handle: "bookmarkeduser",
        name: "Bookmarked User",
        you: {
            __typename: "UserYou",
            canDelete: false,
            canReport: true,
            canUpdate: false,
            isBookmarked: true, // This is the bookmark we're looking at
            isReported: false,
            reaction: "like",
        },
    },
    list: {
        ...mockBookmarkList,
        id: "list_987654321098765432",
        label: "Favorite Users",
        bookmarksCount: 3,
    },
    createdAt: "2024-01-01T12:00:00Z",
    updatedAt: "2024-01-01T12:30:00Z",
};

/**
 * Bookmark responses for each BookmarkFor type
 */
export const bookmarkResponseVariants: Record<BookmarkFor, Bookmark> = {
    [BookmarkFor.Comment]: {
        ...minimalBookmarkResponse,
        id: "bookmark_comment_123456789012345678",
        to: {
            __typename: "Comment",
            id: "comment_123456789012345678",
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            // Add minimal Comment fields - you may need to adjust based on actual Comment type
        } as any, // Type assertion for now since Comment structure may vary
    },
    [BookmarkFor.Issue]: {
        ...minimalBookmarkResponse,
        id: "bookmark_issue_123456789012345678",
        to: {
            __typename: "Issue",
            id: "issue_123456789012345678",
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            // Add minimal Issue fields
        } as any,
    },
    [BookmarkFor.Resource]: {
        ...minimalBookmarkResponse,
        id: "bookmark_resource_123456789012345678",
        to: mockResource,
    },
    [BookmarkFor.Tag]: {
        ...minimalBookmarkResponse,
        id: "bookmark_tag_123456789012345678",
        to: {
            __typename: "Tag",
            id: "tag_123456789012345678",
            tag: "artificial-intelligence",
            // Add minimal Tag fields
        } as any,
    },
    [BookmarkFor.Team]: {
        ...minimalBookmarkResponse,
        id: "bookmark_team_123456789012345678",
        to: {
            __typename: "Team",
            id: "team_123456789012345678",
            handle: "awesome-team",
            name: "Awesome Team",
            isPrivate: false,
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            // Add minimal Team fields
        } as any,
    },
    [BookmarkFor.User]: {
        ...minimalBookmarkResponse,
        id: "bookmark_user_123456789012345678",
        to: mockUser,
    },
};

/**
 * Bookmark list with multiple bookmarks for testing lists
 */
export const bookmarkListWithBookmarksResponse: BookmarkList = {
    __typename: "BookmarkList",
    id: "list_111222333444555666",
    label: "My Favorite Resources",
    bookmarks: [
        bookmarkResponseVariants[BookmarkFor.Resource],
        bookmarkResponseVariants[BookmarkFor.User],
        bookmarkResponseVariants[BookmarkFor.Team],
    ],
    bookmarksCount: 3,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T15:00:00Z",
};

/**
 * Loading and error states for UI testing
 */
export const bookmarkUIStates = {
    loading: null,
    error: {
        code: "BOOKMARK_NOT_FOUND",
        message: "The requested bookmark could not be found",
    },
    empty: {
        bookmarks: [],
        bookmarksCount: 0,
    },
};