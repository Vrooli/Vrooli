import { Request, Response } from "express";
import { generatePK } from "@vrooli/shared";
import { type AuthenticatedSessionData } from "../../../types.js";
import { type ApiKeyFullAuthData } from "./apiKeyPermissions.js";
import { adminUser, standardUser, premiumUser, guestUser } from "./userPersonas.js";
import { readOnlyPublicApiKey, writeApiKey } from "./apiKeyPermissions.js";

// Simple client ID generator for testing
function generateClientId(apiKeyId: string): string {
    return `client_${apiKeyId}_${generatePK()}`;
}

/**
 * Helper functions to create test sessions using permission fixtures
 */

/**
 * Create a mock Express request/response pair with session data
 */
export async function createSession(
    userData: AuthenticatedSessionData,
    reqOverrides: Partial<Request> = {},
    resOverrides: Partial<Response> = {},
): Promise<{ req: Request; res: Response }> {
    const req = {
        session: {
            id: `session_${Date.now()}`,
            data: userData,
            touch: jest.fn(),
            regenerate: jest.fn((callback) => callback(null)),
            destroy: jest.fn((callback) => callback(null)),
            reload: jest.fn((callback) => callback(null)),
            save: jest.fn((callback) => callback(null)),
            cookie: {
                originalMaxAge: 86400000,
                httpOnly: true,
                secure: true,
                sameSite: "strict",
            },
        },
        cookies: {},
        headers: {
            "x-csrf-token": userData.csrfToken,
            "user-agent": "Jest Test",
        },
        ip: "127.0.0.1",
        get: jest.fn((header: string) => req.headers[header.toLowerCase()]),
        ...reqOverrides,
    } as unknown as Request;

    const res = {
        locals: {
            userId: userData.id,
            roles: userData.roles,
        },
        status: jest.fn().mockReturnThis(),
        json: jest.fn().mockReturnThis(),
        send: jest.fn().mockReturnThis(),
        setHeader: jest.fn().mockReturnThis(),
        ...resOverrides,
    } as unknown as Response;

    return { req, res };
}

/**
 * Create a mock API key session
 */
export async function createApiKeySession(
    apiKeyData: ApiKeyFullAuthData,
    reqOverrides: Partial<Request> = {},
    resOverrides: Partial<Response> = {},
): Promise<{ req: Request; res: Response }> {
    const req = {
        session: {
            id: `api_session_${Date.now()}`,
            apiToken: generateClientId(apiKeyData.id),
            data: apiKeyData,
            touch: jest.fn(),
            regenerate: jest.fn((callback) => callback(null)),
            destroy: jest.fn((callback) => callback(null)),
            reload: jest.fn((callback) => callback(null)),
            save: jest.fn((callback) => callback(null)),
            cookie: {
                originalMaxAge: 86400000,
                httpOnly: true,
                secure: true,
                sameSite: "strict",
            },
        },
        cookies: {},
        headers: {
            "x-api-key": generateClientId(apiKeyData.id),
            "user-agent": "Jest Test API Client",
        },
        ip: "127.0.0.1",
        get: jest.fn((header: string) => req.headers[header.toLowerCase()]),
        ...reqOverrides,
    } as unknown as Request;

    const res = {
        locals: {
            userId: apiKeyData.userId,
            apiKeyId: apiKeyData.id,
            permissions: apiKeyData.permissions,
        },
        status: jest.fn().mockReturnThis(),
        json: jest.fn().mockReturnThis(),
        send: jest.fn().mockReturnThis(),
        setHeader: jest.fn().mockReturnThis(),
        ...resOverrides,
    } as unknown as Response;

    return { req, res };
}

/**
 * Quick session creators for common scenarios
 */
export const quickSession = {
    // User sessions
    admin: () => createSession(adminUser),
    standard: () => createSession(standardUser),
    premium: () => createSession(premiumUser),
    guest: () => createSession(guestUser),
    
    // API key sessions
    readOnly: () => createApiKeySession(readOnlyPublicApiKey),
    write: () => createApiKeySession(writeApiKey),
    
    // Custom user
    withUser: (userData: AuthenticatedSessionData) => createSession(userData),
    
    // Custom API key
    withApiKey: (apiKeyData: ApiKeyFullAuthData) => createApiKeySession(apiKeyData),
};

/**
 * Create multiple sessions for testing cross-user scenarios
 */
export async function createMultipleSessions(
    users: AuthenticatedSessionData[],
): Promise<Array<{ req: Request; res: Response; user: AuthenticatedSessionData }>> {
    const sessions = await Promise.all(
        users.map(async (user) => {
            const session = await createSession(user);
            return { ...session, user };
        }),
    );
    return sessions;
}

/**
 * Test helper to verify permission denied
 */
export function expectPermissionDenied(
    fn: () => Promise<any>,
    expectedError: string | RegExp = /Unauthorized|Forbidden|Permission denied/,
): Promise<void> {
    return expect(fn()).rejects.toThrow(expectedError);
}

/**
 * Test helper to verify permission granted
 */
export async function expectPermissionGranted(
    fn: () => Promise<any>,
): Promise<void> {
    await expect(fn()).resolves.not.toThrow();
}

/**
 * Batch permission testing helper
 */
export async function testPermissionMatrix(
    operation: (session: { req: Request; res: Response }) => Promise<any>,
    expectations: Record<string, boolean>,
): Promise<void> {
    const results: Record<string, { passed: boolean; error?: any }> = {};

    // Test each scenario
    for (const [scenario, shouldPass] of Object.entries(expectations)) {
        try {
            let session;
            
            // Create appropriate session based on scenario name
            switch (scenario) {
                case "admin":
                    session = await quickSession.admin();
                    break;
                case "standard":
                    session = await quickSession.standard();
                    break;
                case "premium":
                    session = await quickSession.premium();
                    break;
                case "guest":
                    session = await quickSession.guest();
                    break;
                case "readOnly":
                    session = await quickSession.readOnly();
                    break;
                case "write":
                    session = await quickSession.write();
                    break;
                default:
                    throw new Error(`Unknown scenario: ${scenario}`);
            }

            await operation(session);
            results[scenario] = { passed: true };

            // If we expected it to fail, this is a problem
            if (!shouldPass) {
                throw new Error(`Expected ${scenario} to fail but it passed`);
            }
        } catch (error) {
            results[scenario] = { passed: false, error };

            // If we expected it to pass, this is a problem
            if (shouldPass) {
                throw new Error(`Expected ${scenario} to pass but it failed: ${error}`);
            }
        }
    }

    return;
}

/**
 * Test helper for permission changes (before/after scenarios)
 */
export async function testPermissionChange(
    operation: (session: { req: Request; res: Response }) => Promise<any>,
    beforeSession: AuthenticatedSessionData,
    afterSession: AuthenticatedSessionData,
    expectations: {
        beforeShouldPass: boolean;
        afterShouldPass: boolean;
    },
): Promise<void> {
    // Test with original permissions
    const beforeSessionObj = await createSession(beforeSession);
    const beforeResult = await operation(beforeSessionObj)
        .then(() => true)
        .catch(() => false);
    
    if (beforeResult !== expectations.beforeShouldPass) {
        throw new Error(
            `Before permission test failed: expected ${expectations.beforeShouldPass}, got ${beforeResult}`
        );
    }
    
    // Test with changed permissions
    const afterSessionObj = await createSession(afterSession);
    const afterResult = await operation(afterSessionObj)
        .then(() => true)
        .catch(() => false);
    
    if (afterResult !== expectations.afterShouldPass) {
        throw new Error(
            `After permission test failed: expected ${expectations.afterShouldPass}, got ${afterResult}`
        );
    }
}

/**
 * Bulk permission testing helper
 */
export async function testBulkPermissions(
    operations: Array<{
        name: string;
        fn: (session: { req: Request; res: Response }) => Promise<any>;
    }>,
    sessions: Array<{
        name: string;
        session: AuthenticatedSessionData | ApiKeyFullAuthData;
        isApiKey?: boolean;
    }>,
    expectedMatrix: Record<string, Record<string, boolean>>,
): Promise<void> {
    const results: Record<string, Record<string, boolean>> = {};
    
    for (const { name: operationName, fn } of operations) {
        results[operationName] = {};
        
        for (const { name: sessionName, session, isApiKey } of sessions) {
            try {
                const sessionObj = isApiKey 
                    ? await createApiKeySession(session as ApiKeyFullAuthData)
                    : await createSession(session as AuthenticatedSessionData);
                    
                await fn(sessionObj);
                results[operationName][sessionName] = true;
            } catch (error) {
                results[operationName][sessionName] = false;
            }
            
            // Verify against expected result
            const expected = expectedMatrix[operationName]?.[sessionName];
            const actual = results[operationName][sessionName];
            
            if (expected !== undefined && expected !== actual) {
                throw new Error(
                    `Permission test failed for ${operationName} with ${sessionName}: ` +
                    `expected ${expected}, got ${actual}`
                );
            }
        }
    }
    
    return;
}