/**
 * Session fixtures for authentication and authorization testing
 * These represent various user session states and auth contexts
 */

import { type Session, type User } from "@vrooli/shared";

/**
 * Base user data for sessions
 */
const baseUserData = {
    __typename: "User" as const,
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-20T10:00:00Z",
    isBot: false,
    isBotDepictingPerson: false,
    status: "Unlocked",
    // Default counts
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
};

/**
 * Guest session (not authenticated)
 */
export const guestSession = {
    isLoggedIn: false,
    user: null,
    permissions: ["Read"],
    theme: "light",
    language: "en",
};

/**
 * Basic authenticated user session
 */
export const authenticatedUserSession = {
    isLoggedIn: true,
    user: {
        ...baseUserData,
        id: "user_auth_123456789",
        handle: "testuser",
        name: "Test User",
        isPrivate: false,
        theme: "light",
        email: "testuser@example.com",
        emailVerified: true,
        profileImage: null,
        bannerImage: null,
        permissions: ["Read", "Write", "Comment"],
        you: {
            __typename: "UserYou" as const,
            canDelete: true,
            canReport: false,
            canUpdate: true,
            isBookmarked: false,
            isReported: false,
            reaction: null,
        },
    },
    accessToken: "mock_access_token_123456",
    refreshToken: "mock_refresh_token_123456",
    expiresAt: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
    sessionId: "session_123456789",
};

/**
 * Premium user session
 */
export const premiumUserSession = {
    ...authenticatedUserSession,
    user: {
        ...authenticatedUserSession.user,
        id: "user_premium_123456",
        handle: "premiumuser",
        name: "Premium User",
        isPremium: true,
        premiumUntil: "2025-01-20T00:00:00Z",
        profileImage: "https://example.com/premium-avatar.jpg",
        permissions: ["Read", "Write", "Comment", "Premium"],
        credits: 1000,
        creditsUsed: 250,
    },
};

/**
 * Admin user session
 */
export const adminUserSession = {
    ...authenticatedUserSession,
    user: {
        ...authenticatedUserSession.user,
        id: "user_admin_123456789",
        handle: "adminuser",
        name: "Admin User",
        isAdmin: true,
        profileImage: "https://example.com/admin-avatar.jpg",
        permissions: ["Read", "Write", "Comment", "Delete", "Admin", "Moderate"],
        roles: ["Admin", "Moderator"],
    },
};

/**
 * Team member session
 */
export const teamMemberSession = {
    ...authenticatedUserSession,
    user: {
        ...authenticatedUserSession.user,
        id: "user_team_123456789",
        handle: "teammember",
        name: "Team Member",
        teams: [
            {
                id: "team_123456789",
                handle: "awesome-team",
                name: "Awesome Team",
                role: "Member",
                permissions: ["Read", "Write"],
            },
            {
                id: "team_987654321",
                handle: "dev-team",
                name: "Development Team",
                role: "Admin",
                permissions: ["Read", "Write", "Delete", "Admin"],
            },
        ],
    },
    activeTeam: "team_123456789", // Currently acting as this team
};

/**
 * Unverified email session
 */
export const unverifiedEmailSession = {
    ...authenticatedUserSession,
    user: {
        ...authenticatedUserSession.user,
        id: "user_unverified_123456",
        email: "unverified@example.com",
        emailVerified: false,
        permissions: ["Read"], // Limited permissions
    },
    requiresVerification: true,
};

/**
 * Suspended user session
 */
export const suspendedUserSession = {
    ...authenticatedUserSession,
    user: {
        ...authenticatedUserSession.user,
        id: "user_suspended_123456",
        handle: "suspendeduser",
        name: "Suspended User",
        status: "Suspended",
        suspendedUntil: "2024-02-20T00:00:00Z",
        suspendedReason: "Terms of service violation",
    },
    isRestricted: true,
    restrictions: ["NoPost", "NoComment", "NoMessage"],
};

/**
 * Session with 2FA enabled
 */
export const twoFactorSession = {
    ...authenticatedUserSession,
    twoFactorEnabled: true,
    twoFactorVerified: false,
    requiresTwoFactor: true,
    twoFactorMethods: ["app", "sms"],
};

/**
 * Session states
 */
export const sessionStates = {
    /**
     * Fresh login
     */
    fresh: {
        ...authenticatedUserSession,
        loginAt: new Date().toISOString(),
        lastActivity: new Date().toISOString(),
        isNew: true,
    },

    /**
     * About to expire
     */
    expiringSoon: {
        ...authenticatedUserSession,
        expiresAt: new Date(Date.now() + 300000).toISOString(), // 5 minutes
        showExpiryWarning: true,
    },

    /**
     * Expired session
     */
    expired: {
        ...authenticatedUserSession,
        expiresAt: new Date(Date.now() - 60000).toISOString(), // 1 minute ago
        isExpired: true,
        requiresRefresh: true,
    },

    /**
     * Invalid/corrupted session
     */
    invalid: {
        isLoggedIn: false,
        user: null,
        error: "Invalid session token",
        requiresLogin: true,
    },
};

/**
 * OAuth session variants
 */
export const oauthSessions = {
    google: {
        ...authenticatedUserSession,
        provider: "google",
        providerId: "google_123456789",
        providerData: {
            email: "user@gmail.com",
            picture: "https://googleusercontent.com/photo.jpg",
        },
    },
    github: {
        ...authenticatedUserSession,
        provider: "github",
        providerId: "github_123456",
        providerData: {
            login: "githubuser",
            avatar_url: "https://avatars.githubusercontent.com/u/123456",
        },
    },
};

/**
 * API key sessions (for bots/integrations)
 */
export const apiKeySession = {
    isLoggedIn: true,
    isApiKey: true,
    user: {
        ...baseUserData,
        id: "user_bot_123456789",
        handle: "integration-bot",
        name: "Integration Bot",
        isBot: true,
    },
    apiKey: {
        id: "apikey_123456789",
        name: "Production API Key",
        permissions: ["Read", "Write"],
        rateLimit: 1000,
        expiresAt: null,
    },
};

/**
 * Helper function to create custom session
 */
export const createSession = (overrides: Partial<any> = {}) => ({
    ...authenticatedUserSession,
    ...overrides,
    user: {
        ...authenticatedUserSession.user,
        ...overrides.user,
    },
});

/**
 * Helper function to check session validity
 */
export const isSessionValid = (session: any): boolean => {
    if (!session?.isLoggedIn || !session?.user) return false;
    if (session.isExpired) return false;
    if (session.expiresAt && new Date(session.expiresAt) < new Date()) return false;
    if (session.user.status === "Suspended") return false;
    return true;
};

/**
 * Helper function to get session permissions
 */
export const getSessionPermissions = (session: any): string[] => {
    if (!session?.user) return ["Read"]; // Guest permissions
    
    const basePermissions = session.user.permissions || [];
    const teamPermissions = session.activeTeam 
        ? session.user.teams?.find((t: any) => t.id === session.activeTeam)?.permissions || []
        : [];
    
    return [...new Set([...basePermissions, ...teamPermissions])];
};

/**
 * Mock session storage
 */
export const mockSessionStorage = {
    get: (key: string = "session") => {
        const stored = localStorage.getItem(key);
        return stored ? JSON.parse(stored) : null;
    },
    set: (session: any, key: string = "session") => {
        localStorage.setItem(key, JSON.stringify(session));
    },
    clear: (key: string = "session") => {
        localStorage.removeItem(key);
    },
};