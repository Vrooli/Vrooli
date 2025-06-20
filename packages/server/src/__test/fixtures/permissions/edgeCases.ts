import { AccountStatus, generatePK } from "@vrooli/shared";
import { type AuthenticatedSessionData } from "../../../types.js";
import { standardUser } from "./userPersonas.js";

/**
 * Edge case scenarios for permission testing
 */

/**
 * Expired session data
 */
export const expiredSession: AuthenticatedSessionData = {
    ...standardUser,
    id: "888888888888888888",
    csrfToken: "expired-token",
    currentToken: "expired-jwt-token",
    // In real scenarios, this would be validated by JWT expiration
};

/**
 * User with partial data (migration scenario)
 */
export const partialUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "999999999999999999",
    email: null, // No email
    emailVerified: false,
    handle: null, // No handle
    name: "Partial User",
};

/**
 * User with conflicting permissions
 */
export const conflictingPermissionsUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "101010101010101010",
    roles: [
        { role: { name: "ReadOnly", permissions: JSON.stringify(["read_all"]) } },
        { role: { name: "Writer", permissions: JSON.stringify(["write_all"]) } },
    ],
};

/**
 * User with maximum permissions (stress test)
 */
export const maxPermissionsUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "111111111111111112",
    hasPremium: true,
    roles: [
        {
            role: {
                name: "SuperAdmin",
                permissions: JSON.stringify([
                    "admin",
                    "moderate_content",
                    "manage_users",
                    "manage_teams",
                    "manage_billing",
                    "view_analytics",
                    "export_data",
                    "import_data",
                    "execute_scripts",
                    "manage_ai",
                ]),
            },
        },
    ],
};

/**
 * User in deletion process
 */
export const deletingUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "121212121212121212",
    accountStatus: AccountStatus.Deleting,
    // Should have limited access while deletion is processing
};

/**
 * User with rate limit exceeded
 */
export const rateLimitedUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "131313131313131313",
    // In real scenario, rate limit would be tracked separately
    // This is for testing rate limit error handling
};

/**
 * Cross-origin session (CORS testing)
 */
export const corsTestUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "141414141414141414",
    // Would include origin headers in actual request
};

/**
 * Session hijacking attempt simulation
 */
export const hijackedSession: AuthenticatedSessionData = {
    ...standardUser,
    id: "151515151515151515",
    csrfToken: "wrong-csrf-token", // Mismatched CSRF
};

/**
 * User with special characters in data
 */
export const specialCharsUser: AuthenticatedSessionData = {
    ...standardUser,
    id: "161616161616161616",
    handle: "user-with_special.chars",
    name: "User <script>alert('xss')</script>",
    email: "test+special@example.com",
};

/**
 * Concurrent session scenario
 */
export const concurrentSessions = {
    session1: {
        ...standardUser,
        id: "171717171717171717",
        currentToken: "session-1-token",
    },
    session2: {
        ...standardUser,
        id: "171717171717171717", // Same user
        currentToken: "session-2-token",
    },
    description: "Same user with multiple active sessions",
};

/**
 * Permission inheritance test cases
 */
export const permissionInheritance = {
    // User inherits team permissions
    teamMemberWithInheritance: {
        user: standardUser,
        teamPermissions: ["create_projects", "view_analytics"],
        userPermissions: ["view_own_data"],
        expectedPermissions: ["create_projects", "view_analytics", "view_own_data"],
    },
    
    // Role conflicts with team permissions
    conflictingTeamPermissions: {
        user: standardUser,
        teamRole: "ReadOnly",
        userRole: "Editor",
        description: "Testing which permission takes precedence",
    },
};

/**
 * Constants for time-based calculations
 */
const ONE_HOUR_MS = 3600000;
const MAX_ID_LENGTH = 30;
const MAX_NAME_LENGTH = 1000;

/**
 * Time-based permission scenarios
 */
export const timeBasedPermissions = {
    // Temporary elevated permissions
    temporaryAdmin: {
        user: standardUser,
        elevatedUntil: new Date(Date.now() + ONE_HOUR_MS), // 1 hour from now
        elevatedPermissions: ["admin"],
    },
    
    // Expired temporary permissions
    expiredElevation: {
        user: standardUser,
        elevatedUntil: new Date(Date.now() - ONE_HOUR_MS), // 1 hour ago
        elevatedPermissions: ["admin"],
    },
};

/**
 * Malformed session data - missing required fields
 */
export const malformedSession: Partial<AuthenticatedSessionData> = {
    // Missing required fields like id, isLoggedIn, etc.
    handle: "malformed_user",
    // @ts-expect-error - Intentionally incomplete session
    csrfToken: undefined,
    currentToken: "invalid-format-token",
};

/**
 * Session with invalid data types
 */
export const invalidTypeSession = {
    ...standardUser,
    // @ts-expect-error - Intentionally wrong types
    id: 12345, // Should be string
    isLoggedIn: "yes", // Should be boolean
    hasPremium: 1, // Should be boolean
    languages: "en,es", // Should be array
    roles: { role: "admin" }, // Should be array
};

/**
 * Helper to create edge case scenarios
 */
export function createEdgeCaseUser(
    scenario: "corrupted" | "null-fields" | "overflow" | "injection" | "malformed",
): AuthenticatedSessionData | Partial<AuthenticatedSessionData> {
    switch (scenario) {
        case "corrupted":
            return {
                ...standardUser,
                id: "corrupted_id_format",
                // @ts-expect-error - Intentionally wrong type
                roles: "not-an-array",
            };
        
        case "null-fields":
            return {
                ...standardUser,
                id: generatePK().toString(),
                // @ts-expect-error - Testing null handling
                languages: null,
                timeZone: null,
            };
        
        case "overflow":
            return {
                ...standardUser,
                id: "9".repeat(MAX_ID_LENGTH), // ID too long
                name: "A".repeat(MAX_NAME_LENGTH), // Name too long
            };
        
        case "injection":
            return {
                ...standardUser,
                id: "'; DROP TABLE users; --",
                handle: "../../../etc/passwd",
                name: "${process.env.SECRET_KEY}",
            };
            
        case "malformed":
            return malformedSession;
    }
}
