import { AccountStatus, generatePK } from "@vrooli/shared";
import { type AuthenticatedSessionData } from "../../../types.js";

/**
 * User personas representing common user types in the system.
 * Each persona has consistent IDs and properties for predictable testing.
 */

// Base user data that all personas share
const baseUserData = {
    languages: ["en"],
    theme: "light",
    csrfToken: "test-csrf-token",
    currentToken: "test-token",
    marketplaceUrl: "https://test.vrooli.com",
} as const;

/**
 * System administrator with full access to all features
 */
export const adminUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "111111111111111111", // Consistent ID for admin
    handle: "admin",
    name: "System Admin",
    email: "admin@vrooli.com",
    emailVerified: true,
    accountStatus: AccountStatus.ActiveGood,
    isLoggedIn: true,
    timeZone: "America/New_York",
    hasPremium: true,
    roles: [{
        role: { name: "Admin", permissions: JSON.stringify(["admin"]) },
    }],
};

/**
 * Regular user with standard permissions
 */
export const standardUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "222222222222222222",
    handle: "johndoe",
    name: "John Doe",
    email: "john@example.com",
    emailVerified: true,
    accountStatus: AccountStatus.ActiveGood,
    isLoggedIn: true,
    timeZone: "America/Chicago",
    hasPremium: false,
    roles: [],
};

/**
 * Premium user with enhanced features
 */
export const premiumUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "333333333333333333",
    handle: "premium_user",
    name: "Premium User",
    email: "premium@example.com",
    emailVerified: true,
    accountStatus: AccountStatus.ActiveGood,
    isLoggedIn: true,
    timeZone: "Europe/London",
    hasPremium: true,
    roles: [],
};

/**
 * New user with unverified email
 */
export const unverifiedUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "444444444444444444",
    handle: "newuser",
    name: "New User",
    email: "new@example.com",
    emailVerified: false,
    accountStatus: AccountStatus.ActiveUnverified,
    isLoggedIn: true,
    timeZone: "UTC",
    hasPremium: false,
    roles: [],
};

/**
 * Banned user - should be denied most operations
 */
export const bannedUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "555555555555555555",
    handle: "banned_user",
    name: "Banned User",
    email: "banned@example.com",
    emailVerified: true,
    accountStatus: AccountStatus.SoftLocked,
    isLoggedIn: true,
    timeZone: "UTC",
    hasPremium: false,
    roles: [],
};

/**
 * Guest user (not logged in)
 */
export const guestUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "",
    handle: null,
    name: null,
    email: null,
    emailVerified: false,
    accountStatus: AccountStatus.Deleted,
    isLoggedIn: false,
    timeZone: "UTC",
    hasPremium: false,
    roles: [],
};

/**
 * Bot user with special permissions
 */
export const botUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "666666666666666666",
    handle: "test_bot",
    name: "Test Bot",
    email: "bot@vrooli.com",
    emailVerified: true,
    accountStatus: AccountStatus.ActiveGood,
    isLoggedIn: true,
    timeZone: "UTC",
    hasPremium: false,
    isBot: true,
    roles: [],
};

/**
 * User with custom role permissions
 */
export const customRoleUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "777777777777777777",
    handle: "custom_role",
    name: "Custom Role User",
    email: "custom@example.com",
    emailVerified: true,
    accountStatus: AccountStatus.ActiveGood,
    isLoggedIn: true,
    timeZone: "Asia/Tokyo",
    hasPremium: false,
    roles: [{
        role: { 
            name: "ContentModerator", 
            permissions: JSON.stringify(["moderate_content", "view_reports"]), 
        },
    }],
};

/**
 * Helper function to create a user with specific permissions
 */
export function createUserWithPermissions(
    overrides: Partial<AuthenticatedSessionData>,
    permissions: string[] = [],
): AuthenticatedSessionData {
    const id = overrides.id || generatePK().toString();
    return {
        ...standardUser,
        ...overrides,
        id,
        roles: permissions.length > 0 ? [{
            role: { 
                name: "CustomRole", 
                permissions: JSON.stringify(permissions), 
            },
        }] : [],
    };
}

/**
 * Suspended user - temporarily suspended, different from banned
 */
export const suspendedUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "888888888888888888",
    handle: "suspended_user",
    name: "Suspended User",
    email: "suspended@example.com",
    emailVerified: true,
    accountStatus: AccountStatus.HardLocked,
    isLoggedIn: true,
    timeZone: "UTC",
    hasPremium: false,
    roles: [],
};

/**
 * User with expired premium subscription
 */
export const expiredPremiumUser: AuthenticatedSessionData = {
    ...baseUserData,
    id: "999999999999999999",
    handle: "expired_premium",
    name: "Expired Premium User",
    email: "expired@example.com",
    emailVerified: true,
    accountStatus: AccountStatus.ActiveGood,
    isLoggedIn: true,
    timeZone: "America/Los_Angeles",
    hasPremium: false, // Was premium, now expired
    roles: [],
    // In a real scenario, this would have additional metadata about the expired subscription
};

/**
 * Create multiple related users for testing cross-user permissions
 */
export function createUserGroup(baseName: string, count = 3): AuthenticatedSessionData[] {
    return Array(count).fill(null).map((_, index) => ({
        ...standardUser,
        id: generatePK().toString(),
        handle: `${baseName}_${index}`,
        name: `${baseName} User ${index}`,
        email: `${baseName}${index}@example.com`,
    }));
}
