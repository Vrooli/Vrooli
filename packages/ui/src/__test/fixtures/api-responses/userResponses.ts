import { type User, type UserYou, type Translation } from "@vrooli/shared";

/**
 * API response fixtures for users
 * These represent what components receive from API calls
 */

/**
 * Minimal user API response
 */
export const minimalUserResponse: User = {
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
    // Counts
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
 * Complete user API response with all fields
 */
export const completeUserResponse: User = {
    __typename: "User",
    id: "user_987654321098765432",
    handle: "poweruser",
    name: "Power User",
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: "https://example.com/banner.jpg",
    profileImage: "https://example.com/profile.jpg",
    theme: "dark",
    createdAt: "2023-06-15T10:30:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    // Counts
    awardedCount: 15,
    bookmarksCount: 42,
    commentsCount: 128,
    issuesCount: 7,
    postsCount: 23,
    projectsCount: 5,
    pullRequestsCount: 12,
    questionsCount: 34,
    quizzesCount: 3,
    reportsReceivedCount: 0,
    routinesCount: 18,
    standardsCount: 9,
    teamsCount: 3,
    translations: [
        {
            __typename: "UserTranslation",
            id: "usertrans_123456789012345678",
            language: "en",
            bio: "Experienced developer passionate about AI and automation",
        },
        {
            __typename: "UserTranslation",
            id: "usertrans_987654321098765432",
            language: "es",
            bio: "Desarrollador experimentado apasionado por la IA y la automatización",
        },
    ],
    you: {
        __typename: "UserYou",
        canDelete: false,
        canReport: true,
        canUpdate: false,
        isBookmarked: true,
        isReported: false,
        reaction: "like",
    },
};

/**
 * Current user (self) response
 */
export const currentUserResponse: User = {
    __typename: "User",
    id: "user_current_123456789",
    handle: "currentuser",
    name: "Current User",
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: null,
    profileImage: "https://example.com/myprofile.jpg",
    theme: "light",
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-20T09:00:00Z",
    // Counts
    awardedCount: 5,
    bookmarksCount: 20,
    commentsCount: 50,
    issuesCount: 2,
    postsCount: 10,
    projectsCount: 3,
    pullRequestsCount: 8,
    questionsCount: 15,
    quizzesCount: 1,
    reportsReceivedCount: 0,
    routinesCount: 7,
    standardsCount: 4,
    teamsCount: 2,
    translations: [],
    you: {
        __typename: "UserYou",
        canDelete: true, // Can delete own account
        canReport: false, // Can't report self
        canUpdate: true, // Can update own profile
        isBookmarked: false,
        isReported: false,
        reaction: null,
    },
};

/**
 * Bot user response
 */
export const botUserResponse: User = {
    __typename: "User",
    id: "user_bot_123456789012",
    handle: "helpfulbot",
    name: "Helpful Bot",
    isBot: true,
    isBotDepictingPerson: false,
    isPrivate: false,
    status: "Unlocked",
    bannerImage: null,
    profileImage: "https://example.com/bot-avatar.png",
    theme: "light",
    createdAt: "2023-07-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    // Bots typically have different activity patterns
    awardedCount: 0,
    bookmarksCount: 0,
    commentsCount: 500, // High activity
    issuesCount: 0,
    postsCount: 0,
    projectsCount: 0,
    pullRequestsCount: 0,
    questionsCount: 0,
    quizzesCount: 0,
    reportsReceivedCount: 2,
    routinesCount: 10,
    standardsCount: 0,
    teamsCount: 1,
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
 * Private user response (limited visibility)
 */
export const privateUserResponse: User = {
    __typename: "User",
    id: "user_private_123456789",
    handle: "privateuser",
    name: "Private User",
    isBot: false,
    isBotDepictingPerson: false,
    isPrivate: true,
    status: "Unlocked",
    bannerImage: null,
    profileImage: null, // Private users might not show images
    theme: "light",
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    // Private users show limited counts
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
 * User variant states for testing
 */
export const userResponseVariants = {
    minimal: minimalUserResponse,
    complete: completeUserResponse,
    currentUser: currentUserResponse,
    bot: botUserResponse,
    private: privateUserResponse,
    suspended: {
        ...minimalUserResponse,
        id: "user_suspended_123456789",
        handle: "suspendeduser",
        name: "Suspended User",
        status: "Suspended",
    },
    deleted: {
        ...minimalUserResponse,
        id: "user_deleted_123456789",
        handle: "[deleted]",
        name: "[Deleted User]",
        status: "Deleted",
    },
} as const;

/**
 * User list response for search results
 */
export const userSearchResponse = {
    __typename: "UserSearchResult",
    edges: [
        {
            __typename: "UserEdge",
            cursor: "cursor_1",
            node: userResponseVariants.complete,
        },
        {
            __typename: "UserEdge",
            cursor: "cursor_2",
            node: userResponseVariants.minimal,
        },
        {
            __typename: "UserEdge",
            cursor: "cursor_3",
            node: userResponseVariants.bot,
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
 * Loading and error states for UI testing
 */
export const userUIStates = {
    loading: null,
    error: {
        code: "USER_NOT_FOUND",
        message: "The requested user could not be found",
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