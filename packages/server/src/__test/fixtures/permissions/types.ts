/**
 * Permission Fixture Type Definitions
 * 
 * Core types and interfaces for the permission testing layer of the
 * unified fixture architecture.
 */

import { type AuthenticatedSessionData } from "../../../types.js";

/**
 * API Key types for testing
 */
export enum ApiKeyType {
    Internal = "Internal",
    External = "External",
}

/**
 * Base permission context that all permission tests require
 */
export interface PermissionContext {
    /** The authenticated session (user or API key) */
    session: AuthenticatedSessionData | ApiKeyAuthData;
    /** Additional context like team membership, resource ownership */
    context?: Record<string, unknown>;
    /** Expected permission result */
    expected?: boolean;
}

/**
 * Permission test result with detailed information
 */
export interface PermissionTestResult {
    /** Whether the permission check passed */
    allowed: boolean;
    /** Reason for denial if applicable */
    reason?: string;
    /** Performance metrics */
    timing?: {
        start: number;
        end: number;
        duration: number;
    };
    /** Additional metadata */
    metadata?: Record<string, unknown>;
}

/**
 * Multi-actor permission scenario for testing complex interactions
 */
export interface PermissionScenario<TResource = Record<string, unknown>> {
    /** Unique identifier for the scenario */
    id: string;
    /** Human-readable description */
    description: string;
    /** The resource being accessed */
    resource: TResource;
    /** Multiple actors trying to access the resource */
    actors: Array<{
        /** Actor identifier */
        id: string;
        /** Actor's session */
        session: AuthenticatedSessionData | ApiKeyAuthData;
        /** Expected permissions for each action */
        permissions: Record<string, boolean>;
    }>;
    /** Actions to test */
    actions: string[];
}

/**
 * Permission matrix for comprehensive testing
 */
export interface PermissionMatrix {
    /** Map of persona name to expected permission result */
    [persona: string]: boolean;
}

/**
 * API Key authentication data structure
 */
export interface ApiKeyAuthData {
    __type: ApiKeyType;
    id: string;
    userId: string;
    permissions: {
        read: "None" | "Public" | "Private" | "Auth";
        write: "None" | "Public" | "Private" | "Auth";
        bot: boolean;
        daily_credits: number;
    };
    csrfToken: string;
    isLoggedIn: boolean;
    languages: string[];
    timeZone: string;
    theme: string;
    isExpired?: boolean;
    isRevoked?: boolean;
}

/**
 * Team member with role information
 */
export interface TeamMember {
    user: AuthenticatedSessionData;
    teamId: string;
    role: "Owner" | "Admin" | "Member";
    permissions?: string[];
    joinedAt?: Date;
}

/**
 * OAuth session data
 */
export interface OAuthSessionData extends AuthenticatedSessionData {
    provider: "google" | "github" | "microsoft" | "discord";
    scope: string[];
    externalId: string;
    refreshToken?: string;
}

/**
 * Rate limiting information
 */
export interface RateLimitInfo {
    limit: number;
    remaining: number;
    reset: Date;
    window: string;
}

/**
 * Permission inheritance configuration
 */
export interface PermissionInheritance {
    /** Parent permission source */
    inherits?: string;
    /** Additional permissions to add */
    adds?: string[];
    /** Permissions to remove from inheritance */
    excludes?: string[];
    /** Permissions that override inherited ones */
    overrides?: Record<string, unknown>;
}

/**
 * Factory method signatures for creating test fixtures
 */
export interface PermissionFixtureFactory<TSession = AuthenticatedSessionData> {
    /** Create a session with specific overrides */
    createSession: (overrides?: Partial<TSession>) => TSession;
    
    /** Create a session with specific permissions */
    withPermissions: (session: TSession, permissions: string[]) => TSession;
    
    /** Create a session with a specific role */
    withRole: (session: TSession, role: string) => TSession;
    
    /** Create a session with team membership */
    withTeam: (session: TSession, teamId: string, role: "Owner" | "Admin" | "Member") => TSession;
    
    /** Create an expired session */
    asExpired: (session: TSession) => TSession;
    
    /** Create a rate-limited session */
    asRateLimited: (session: TSession, info: RateLimitInfo) => TSession;
}

/**
 * Permission validation utilities
 */
export interface PermissionValidator {
    /** Check if a session has a specific permission */
    hasPermission: (session: AuthenticatedSessionData | ApiKeyAuthData, permission: string) => boolean;
    
    /** Check if a session can perform an action on a resource */
    canAccess: (session: AuthenticatedSessionData | ApiKeyAuthData, action: string, resource: Record<string, unknown>) => boolean;
    
    /** Get all permissions for a session */
    getPermissions: (session: AuthenticatedSessionData | ApiKeyAuthData) => string[];
    
    /** Validate permission inheritance chain */
    validateInheritance: (chain: PermissionInheritance[]) => boolean;
}

/**
 * Session creation options
 */
export interface SessionOptions {
    /** Include CSRF token */
    withCsrf?: boolean;
    /** Set specific timezone */
    timeZone?: string;
    /** Set UI theme */
    theme?: "light" | "dark";
    /** Set languages */
    languages?: string[];
    /** Set marketplace URL */
    marketplaceUrl?: string;
}

/**
 * Test helper function signatures
 */
export interface PermissionTestHelpers {
    /** Expect a permission to be denied */
    expectPermissionDenied: (fn: () => Promise<unknown>, expectedError?: string | RegExp) => Promise<void>;
    
    /** Expect a permission to be granted */
    expectPermissionGranted: (fn: () => Promise<unknown>) => Promise<void>;
    
    /** Test a permission matrix against multiple personas */
    testPermissionMatrix: (
        testFn: (session: AuthenticatedSessionData | ApiKeyAuthData) => Promise<unknown>,
        matrix: PermissionMatrix
    ) => Promise<void>;
    
    /** Test permission changes over time */
    testPermissionChange: (
        testFn: (session: AuthenticatedSessionData | ApiKeyAuthData) => Promise<unknown>,
        before: AuthenticatedSessionData,
        after: AuthenticatedSessionData,
        expectations: { beforeShouldPass: boolean; afterShouldPass: boolean }
    ) => Promise<void>;
    
    /** Test bulk permissions */
    testBulkPermissions: (
        operations: Array<{ name: string; fn: (session: AuthenticatedSessionData | ApiKeyAuthData) => Promise<unknown> }>,
        sessions: Array<{ name: string; session: AuthenticatedSessionData | ApiKeyAuthData; isApiKey?: boolean }>,
        expectations: Record<string, Record<string, boolean>>
    ) => Promise<void>;
}

/**
 * Edge case categories for security testing
 */
export interface SecurityEdgeCases {
    /** Session-related edge cases */
    sessionEdgeCases: {
        expired: AuthenticatedSessionData;
        malformed: Record<string, unknown>;
        hijacked: AuthenticatedSessionData;
        replayed: AuthenticatedSessionData;
    };
    
    /** Input validation edge cases */
    inputEdgeCases: {
        sqlInjection: string;
        xssAttempt: string;
        pathTraversal: string;
        commandInjection: string;
    };
    
    /** Permission escalation attempts */
    escalationEdgeCases: {
        roleManipulation: AuthenticatedSessionData;
        scopeExpansion: ApiKeyAuthData;
        ownershipTakeover: AuthenticatedSessionData;
    };
}

/**
 * Performance testing configuration
 */
export interface PerformanceTestConfig {
    /** Number of iterations */
    iterations: number;
    /** Warmup runs before measurement */
    warmupRuns: number;
    /** Expected time thresholds */
    thresholds: {
        p50: number;
        p95: number;
        p99: number;
    };
}
