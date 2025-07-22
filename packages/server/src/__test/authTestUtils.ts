import { ApiKeyPermission } from "@vrooli/shared";
import type { Request, Response } from "express";
import { mockApiSession, mockLoggedOutSession, loggedInUserNoPremiumData } from "./session.js";
import { ApiKeyEncryptionService } from "../auth/apiKeyEncryption.js";
import type { ApiEndpoint } from "../types.js";
import type { PartialApiInfo } from "../builders/types.js";
import { expectCustomErrorAsync } from "./errorTestUtils.js";

/**
 * Helper function to create an API key session with simplified interface
 */
async function mockApiKeySession(options: {
    permissions: Partial<Record<ApiKeyPermission, boolean>>;
    apiEndpoint?: string;
}): Promise<{ req: Request; res: Response }> {
    const userData = loggedInUserNoPremiumData();
    const apiToken = ApiKeyEncryptionService.generateSiteKey();
    
    // Build full permissions object with defaults
    const fullPermissions: Record<ApiKeyPermission, boolean> = {
        [ApiKeyPermission.ReadPublic]: false,
        [ApiKeyPermission.ReadPrivate]: false,
        [ApiKeyPermission.ReadAuth]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.WritePrivate]: false,
        ...options.permissions,
    };
    
    const { req, res } = await mockApiSession(apiToken, fullPermissions, userData);
    
    // Set API endpoint if provided
    if (options.apiEndpoint) {
        req.headers = req.headers || {};
        req.headers["x-api-endpoint"] = options.apiEndpoint;
    }
    
    return { req, res };
}

/**
 * Authentication test scenario interface
 */
export interface AuthTestScenario {
    name: string;
    setup: () => Promise<{ req: Request; res: Response }>;
    expectedError?: string;
    expectedCode?: string;
    customAssertions?: (error: any) => void;
}

/**
 * Test session interface
 */
export interface TestSession {
    req: Request;
    res: Response;
    user?: { id: string; name: string };
    apiKey?: { id: string; permissions: Record<string, boolean> };
}

/**
 * Pre-defined authentication scenarios for common test cases
 */
export const AUTH_SCENARIOS = {
    notLoggedIn: {
        name: "not logged in",
        setup: mockLoggedOutSession,
        expectedError: "Unauthorized",
        expectedCode: "Unauthorized",
    },
    apiKeyInsteadOfUser: {
        name: "API key instead of user",
        setup: () => mockApiKeySession({
            permissions: {
                [ApiKeyPermission.ReadPublic]: true,
                [ApiKeyPermission.ReadPrivate]: true,
                [ApiKeyPermission.WritePrivate]: true,
            },
        }),
        expectedError: "Unauthorized",
        expectedCode: "Unauthorized",
    },
} as const;


/**
 * Direct assertion helper for endpoints that require authentication
 * Replaces the deprecated testEndpointRequiresAuth function
 */
export async function assertRequiresAuth<TInput, TResult>(
    endpoint: ApiEndpoint<TInput, TResult>,
    input: TInput,
    select: PartialApiInfo,
): Promise<void> {
    // Test 1: Not logged in
    const { req: req1, res: res1 } = await mockLoggedOutSession();
    await expectCustomErrorAsync(
        endpoint((input === undefined ? undefined : { input }) as any, { req: req1, res: res1 }, select),
        "Unauthorized",
        "0323",
    );

    // Test 2: API key instead of user
    const { req: req2, res: res2 } = await mockApiKeySession({
        permissions: {
            [ApiKeyPermission.ReadPublic]: true,
            [ApiKeyPermission.ReadPrivate]: true,
            [ApiKeyPermission.WritePrivate]: true,
        },
    });
    await expectCustomErrorAsync(
        endpoint((input === undefined ? undefined : { input }) as any, { req: req2, res: res2 }, select),
        "Unauthorized",
        "0323",
    );
}

/**
 * Direct assertion helper for endpoints that require API key with read permissions
 * Replaces the deprecated testEndpointRequiresApiKeyReadPermissions function
 */
export async function assertRequiresApiKeyReadPermissions<TInput, TResult>(
    endpoint: ApiEndpoint<TInput, TResult>,
    input: TInput,
    select: PartialApiInfo,
): Promise<void> {
    const { req, res } = await mockApiKeySession({
        permissions: {},
    });
    await expect(endpoint((input === undefined ? undefined : { input }) as any, { req, res }, select))
        .rejects
        .toThrow(expect.objectContaining({
            code: "Unauthorized",
        }));
}


/**
 * Direct assertion helper for endpoints that require API key with write permissions
 * Replaces the deprecated testEndpointRequiresApiKeyWritePermissions function
 */
export async function assertRequiresApiKeyWritePermissions<TInput, TResult>(
    endpoint: ApiEndpoint<TInput, TResult>,
    input: TInput,
    select: PartialApiInfo,
): Promise<void> {
    // Test 1: No write permissions
    const { req: req1, res: res1 } = await mockApiKeySession({
        permissions: {
            [ApiKeyPermission.ReadPublic]: true,
            [ApiKeyPermission.ReadPrivate]: true,
        },
    });
    await expectCustomErrorAsync(
        endpoint((input === undefined ? undefined : { input }) as any, { req: req1, res: res1 }, select),
        "Unauthorized",
        "0323",
    );

    // Test 2: Wrong API endpoint
    const { req: req2, res: res2 } = await mockApiKeySession({
        permissions: {
            [ApiKeyPermission.WritePrivate]: true,
        },
        apiEndpoint: "https://wrong.endpoint.com",
    });
    await expectCustomErrorAsync(
        endpoint((input === undefined ? undefined : { input }) as any, { req: req2, res: res2 }, select),
        "Unauthorized",
        "0323",
    );
}



