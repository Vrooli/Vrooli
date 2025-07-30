/**
 * Base Permission Factory
 * 
 * Core factory implementation for creating permission fixtures following
 * the unified fixture architecture pattern.
 */

import { type SessionData } from "../../../../types.js";
import {
    type ApiKeyAuthData,
    type PermissionContext,
    type PermissionFixtureFactory,
    type PermissionTestResult,
    type RateLimitInfo,
    type SessionOptions,
} from "../types.js";

/**
 * Constants for ID generation
 */
const TEST_ID_PADDING = 17;

/**
 * Base factory for creating permission-related fixtures
 */
export abstract class BasePermissionFactory<TSession extends SessionData | ApiKeyAuthData>
    implements PermissionFixtureFactory<TSession> {

    /**
     * Default session options
     */
    protected readonly defaultOptions: SessionOptions = {
        withCsrf: true,
        timeZone: "UTC",
        theme: "light",
        languages: ["en"],
        marketplaceUrl: "https://test.vrooli.com",
    };

    /**
     * Base user data shared across all personas
     */
    protected readonly baseUserData = {
        languages: this.defaultOptions.languages ?? ["en"],
        theme: this.defaultOptions.theme ?? "light",
        csrfToken: "test-csrf-token",
        currentToken: "test-token",
        marketplaceUrl: this.defaultOptions.marketplaceUrl ?? "https://test.vrooli.com",
    } as const;

    /**
     * Create a session with specific overrides
     */
    abstract createSession(overrides?: Partial<TSession>): TSession;

    /**
     * Add permissions to a session
     */
    withPermissions(session: TSession, permissions: string[]): TSession {
        if (this.isUserSession(session)) {
            return {
                ...session,
                roles: [{
                    role: {
                        name: "Custom",
                        permissions: JSON.stringify(permissions),
                    },
                }],
            } as TSession;
        }

        // For API keys, permissions are structured differently
        throw new Error("Use withApiKeyPermissions for API key sessions");
    }

    /**
     * Add a role to a session
     */
    withRole(session: TSession, role: string): TSession {
        if (!this.isUserSession(session)) {
            throw new Error("Roles can only be added to user sessions");
        }

        return {
            ...session,
            roles: [{
                role: {
                    name: role,
                    permissions: this.getPermissionsForRole(role),
                },
            }],
        } as TSession;
    }

    /**
     * Add team membership to a session
     */
    withTeam(session: TSession, teamId: string, role: "Owner" | "Admin" | "Member"): TSession {
        if (!this.isUserSession(session)) {
            throw new Error("Team membership can only be added to user sessions");
        }

        // In a real implementation, this would add to session.teams
        // For now, we'll store it in a way that tests can access
        return {
            ...session,
            _testTeamMembership: {
                teamId,
                role,
            },
        } as TSession;
    }

    /**
     * Create an expired session
     */
    asExpired(session: TSession): TSession {
        if (this.isApiKeySession(session)) {
            return {
                ...session,
                isExpired: true,
            } as TSession;
        }

        // For user sessions, we modify the token expiry
        return {
            ...session,
            _testExpired: true,
        } as TSession;
    }

    /**
     * Create a rate-limited session
     */
    asRateLimited(session: TSession, info: RateLimitInfo): TSession {
        return {
            ...session,
            _testRateLimit: info,
        } as TSession;
    }

    /**
     * Check if session is a user session
     */
    protected isUserSession(session: TSession): session is SessionData {
        return !("__type" in session);
    }

    /**
     * Check if session is an API key session
     */
    protected isApiKeySession(session: TSession): session is ApiKeyAuthData {
        return "__type" in session;
    }

    /**
     * Get permissions for a given role
     */
    protected getPermissionsForRole(role: string): string {
        const rolePermissions: Record<string, string[]> = {
            Admin: ["*"],
            Moderator: ["user.read", "user.update", "content.moderate"],
            Member: ["content.create", "content.read", "content.update.own"],
            Guest: ["content.read.public"],
        };

        return JSON.stringify(rolePermissions[role] || []);
    }

    /**
     * Generate a consistent ID for testing
     */
    protected generateTestId(prefix: string, index: number): string {
        // Use consistent IDs for predictable testing
        return `${prefix}${String(index).padStart(TEST_ID_PADDING, "0")}`;
    }

    /**
     * Create test context for permission checking
     */
    createTestContext(session: TSession, additionalContext?: Record<string, unknown>): PermissionContext {
        return {
            session,
            context: additionalContext,
        };
    }

    /**
     * Simulate a permission check with timing
     */
    async simulatePermissionCheck(
        context: PermissionContext,
        checkFn: (ctx: PermissionContext) => boolean | Promise<boolean>,
    ): Promise<PermissionTestResult> {
        const start = Date.now();

        try {
            const allowed = await checkFn(context);
            const end = Date.now();

            return {
                allowed,
                timing: {
                    start,
                    end,
                    duration: end - start,
                },
            };
        } catch (error) {
            const end = Date.now();

            return {
                allowed: false,
                reason: error instanceof Error ? error.message : "Unknown error",
                timing: {
                    start,
                    end,
                    duration: end - start,
                },
            };
        }
    }

    /**
     * Create a batch of sessions for testing
     */
    createBatch(count: number, factory: (index: number) => TSession): TSession[] {
        return Array.from({ length: count }, (_, i) => factory(i));
    }

    /**
     * Merge session data with defaults
     */
    protected mergeWithDefaults(
        base: Partial<TSession>,
        overrides?: Partial<TSession>,
    ): TSession {
        return {
            ...base,
            ...overrides,
        } as TSession;
    }
}
