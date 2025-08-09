/**
 * Session Helpers v2
 * 
 * Enhanced session creation and testing utilities using the factory pattern
 * for the unified fixture architecture.
 */

import { type Request, type Response } from "express";
import { expect, vi } from "vitest";
import { type SessionData } from "../../../types.js";
import { ApiKeyFactory } from "./factories/ApiKeyFactory.js";
import { UserSessionFactory } from "./factories/UserSessionFactory.js";
import {
    type ApiKeyAuthData,
    type PermissionMatrix,
    type PermissionTestHelpers,
    type TestSessionData,
} from "./types.js";
import { PermissionValidator } from "./validators/PermissionValidator.js";

// Factory instances
const userFactory = new UserSessionFactory();
const apiKeyFactory = new ApiKeyFactory();
const validator = new PermissionValidator();

/**
 * Create a mock Express request with session data
 */
export function createMockRequest(session: SessionData | TestSessionData | ApiKeyAuthData): Request {
    return {
        session,
        headers: {
            "x-csrf-token": "csrfToken" in session ? session.csrfToken : undefined,
            "authorization": session.isLoggedIn ? `Bearer ${"currentToken" in session ? session.currentToken || "test-token" : "test-token"}` : undefined,
        },
        ip: "127.0.0.1",
        get: (header: string) => {
            const headers = {
                "x-csrf-token": "csrfToken" in session ? session.csrfToken : undefined,
                "user-agent": "test-agent",
            };
            return headers[header.toLowerCase()];
        },
    } as unknown as Request;
}

/**
 * Create a mock Express response
 */
export function createMockResponse(): Response {
    const res = {
        status: vi.fn().mockReturnThis(),
        json: vi.fn().mockReturnThis(),
        send: vi.fn().mockReturnThis(),
        set: vi.fn().mockReturnThis(),
        cookie: vi.fn().mockReturnThis(),
        clearCookie: vi.fn().mockReturnThis(),
        locals: {},
    } as unknown as Response;

    return res;
}

/**
 * Create a session with request/response objects
 */
export async function createSession(
    sessionData: SessionData | TestSessionData | ApiKeyAuthData,
): Promise<{ req: Request; res: Response }> {
    const req = createMockRequest(sessionData);
    const res = createMockResponse();

    return { req, res };
}

/**
 * Create an API key session
 */
export async function createApiKeySession(
    apiKeyData: ApiKeyAuthData,
): Promise<{ req: Request; res: Response }> {
    return createSession(apiKeyData);
}

/**
 * Create multiple sessions for testing
 */
export async function createMultipleSessions(
    sessions: Array<SessionData | ApiKeyAuthData>,
): Promise<Array<{ req: Request; res: Response }>> {
    return Promise.all(sessions.map(session => createSession(session)));
}

/**
 * Quick session creators using factories
 */
export const quickSession = {
    admin: () => createSession(userFactory.createAdmin()),
    standard: () => createSession(userFactory.createStandard()),
    premium: () => createSession(userFactory.createPremium()),
    guest: () => createSession(userFactory.createGuest()),
    banned: () => createSession(userFactory.createBanned()),
    bot: () => createSession(userFactory.createBot()),

    // API key sessions
    readOnly: () => createSession(apiKeyFactory.createReadOnlyPublic()),
    writeEnabled: () => createSession(apiKeyFactory.createWrite()),
    botKey: () => createSession(apiKeyFactory.createBot()),
    expired: () => createSession(apiKeyFactory.createExpired()),

    // Custom sessions
    withUser: (user: SessionData) => createSession(user),
    withApiKey: (apiKey: ApiKeyAuthData) => createSession(apiKey),

    // With specific permissions
    withPermissions: (permissions: string[]) => {
        const user = userFactory.createWithCustomRole("Custom", permissions);
        return createSession(user);
    },
};

/**
 * Enhanced test helpers implementation
 */
export const testHelpers: PermissionTestHelpers = {
    /**
     * Expect a permission to be denied
     */
    expectPermissionDenied: async (
        fn: () => Promise<unknown>,
        expectedError?: string | RegExp,
    ): Promise<void> => {
        try {
            await fn();
            throw new Error("Expected permission to be denied, but it was granted");
        } catch (error) {
            if (error instanceof Error) {
                if (error.message === "Expected permission to be denied, but it was granted") {
                    throw error;
                }

                if (expectedError) {
                    if (typeof expectedError === "string") {
                        expect(error.message).toContain(expectedError);
                    } else {
                        expect(error.message).toMatch(expectedError);
                    }
                }
            }
        }
    },

    /**
     * Expect a permission to be granted
     */
    expectPermissionGranted: async (fn: () => Promise<unknown>): Promise<void> => {
        try {
            const result = await fn();
            expect(result).toBeDefined();
        } catch (error) {
            throw new Error(`Expected permission to be granted, but it was denied: ${error}`);
        }
    },

    /**
     * Test a permission matrix
     */
    testPermissionMatrix: async (
        testFn: (session: { req: Request; res: Response }) => Promise<unknown>,
        matrix: PermissionMatrix,
    ): Promise<void> => {
        const personas = {
            admin: userFactory.createAdmin(),
            standard: userFactory.createStandard(),
            premium: userFactory.createPremium(),
            guest: userFactory.createGuest(),
            banned: userFactory.createBanned(),
            bot: userFactory.createBot(),
            readOnly: apiKeyFactory.createReadOnlyPublic(),
            writeEnabled: apiKeyFactory.createWrite(),
        };

        for (const [personaName, shouldPass] of Object.entries(matrix)) {
            const persona = personas[personaName];
            if (!persona) {
                throw new Error(`Unknown persona: ${personaName}`);
            }

            const { req, res } = await createSession(persona);

            try {
                await testFn({ req, res });
                if (!shouldPass) {
                    throw new Error(`Expected ${personaName} to be denied, but was granted`);
                }
            } catch (error) {
                if (shouldPass) {
                    throw new Error(`Expected ${personaName} to be granted, but was denied: ${error}`);
                }
            }
        }
    },

    /**
     * Test permission changes
     */
    testPermissionChange: async (
        testFn: (session: { req: Request; res: Response }) => Promise<unknown>,
        before: SessionData,
        after: SessionData,
        expectations: { beforeShouldPass: boolean; afterShouldPass: boolean },
    ): Promise<void> => {
        // Test before state
        const beforeSession = await createSession(before);
        try {
            await testFn(beforeSession);
            if (!expectations.beforeShouldPass) {
                throw new Error("Expected before state to be denied, but was granted");
            }
        } catch (error) {
            if (expectations.beforeShouldPass) {
                throw new Error(`Expected before state to be granted, but was denied: ${error}`);
            }
        }

        // Test after state
        const afterSession = await createSession(after);
        try {
            await testFn(afterSession);
            if (!expectations.afterShouldPass) {
                throw new Error("Expected after state to be denied, but was granted");
            }
        } catch (error) {
            if (expectations.afterShouldPass) {
                throw new Error(`Expected after state to be granted, but was denied: ${error}`);
            }
        }
    },

    /**
     * Test bulk permissions
     */
    testBulkPermissions: async (
        operations: Array<{ name: string; fn: (session: { req: Request; res: Response }) => Promise<unknown> }>,
        sessions: Array<{ name: string; session: SessionData | ApiKeyAuthData; isApiKey?: boolean }>,
        expectations: Record<string, Record<string, boolean>>,
    ): Promise<void> => {
        for (const operation of operations) {
            for (const sessionConfig of sessions) {
                const expected = expectations[operation.name]?.[sessionConfig.name];
                if (expected === undefined) {
                    throw new Error(`Missing expectation for ${operation.name} with ${sessionConfig.name}`);
                }

                const { req, res } = await createSession(sessionConfig.session);

                try {
                    await operation.fn({ req, res });
                    if (!expected) {
                        throw new Error(`Expected ${sessionConfig.name} to be denied for ${operation.name}, but was granted`);
                    }
                } catch (error) {
                    if (expected) {
                        throw new Error(`Expected ${sessionConfig.name} to be granted for ${operation.name}, but was denied: ${error}`);
                    }
                }
            }
        }
    },
};

// Export individual functions for backward compatibility
export const {
    expectPermissionDenied,
    expectPermissionGranted,
    testPermissionMatrix,
    testPermissionChange,
    testBulkPermissions,
} = testHelpers;

/**
 * Utility to check permissions without throwing
 */
export async function checkPermission(
    session: SessionData | ApiKeyAuthData,
    permission: string,
): Promise<boolean> {
    return validator.hasPermission(session, permission);
}

/**
 * Utility to check resource access
 */
export async function checkAccess(
    session: SessionData | ApiKeyAuthData,
    action: string,
    resource: Record<string, unknown>,
): Promise<boolean> {
    return validator.canAccess(session, action, resource);
}

/**
 * Create a permission testing context
 */
export function createPermissionContext(
    session: SessionData | ApiKeyAuthData,
    additionalContext?: Record<string, unknown>,
) {
    return {
        session,
        context: additionalContext,
        validator,
        factories: {
            user: userFactory,
            apiKey: apiKeyFactory,
        },
    };
}
