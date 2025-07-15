import { ApiKeyPermission } from "@vrooli/shared";
import { expect } from "chai";
import { ApiKeyEncryptionService } from "../auth/apiKeyEncryption.js";
import { type UserDataForPasswordAuth } from "../auth/email.js";
import { mockApiSession, mockLoggedOutSession } from "./session.js";
import { loggedInUserNoPremiumData } from "./session.js";

/**
 * Helper function to test that an endpoint requires authentication.
 * This should be called from within an it() block.
 * @param endpoint - The endpoint to test
 * @param params - The parameters to pass to the endpoint
 * @param select - The selection shape to pass to the endpoint
 */
export async function assertEndpointRequiresAuth(
    endpoint: Function,
    params: object,
    select: object,
) {
    const { req, res } = await mockLoggedOutSession();

    try {
        await endpoint(params, { req, res }, select);
        expect.fail("Expected an error to be thrown");
    } catch (error) {
        expect(error).to.have.property("code", "Unauthorized");
    }
}

/**
 * Helper function to test that an endpoint requires API key with read permissions.
 * This should be called from within an it() block.
 * @param userData - The user data for authentication (optional, defaults to standard test user)
 * @param endpoint - The endpoint to test
 * @param params - The parameters to pass to the endpoint
 * @param select - The selection shape to pass to the endpoint
 * @param objectTypes - Optional array of object types (ignored, for backward compatibility)
 */
export async function assertEndpointRequiresApiKeyReadPermissions(
    userData: UserDataForPasswordAuth | Function,
    endpoint?: Function | object,
    params?: object,
    select?: object | string[],
    objectTypes?: string[],
) {
    // Handle overloaded function signatures
    let actualUserData: UserDataForPasswordAuth;
    let actualEndpoint: Function;
    let actualParams: object;
    let actualSelect: object;
    
    if (typeof userData === "function") {
        // Called without userData parameter
        actualUserData = { ...loggedInUserNoPremiumData(), id: "test-user-id" };
        actualEndpoint = userData;
        actualParams = endpoint as object;
        actualSelect = params as object;
        // The 4th parameter might be objectTypes array
    } else {
        // Called with userData parameter
        actualUserData = userData;
        actualEndpoint = endpoint as Function;
        actualParams = params!;
        actualSelect = select as object;
    }

    const permissions = {
        [ApiKeyPermission.ReadPublic]: false,
        [ApiKeyPermission.ReadPrivate]: false,
        [ApiKeyPermission.WritePrivate]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.ReadAuth]: false,
    } as Record<ApiKeyPermission, boolean>;
    const apiToken = ApiKeyEncryptionService.generateSiteKey();
    const { req, res } = await mockApiSession(apiToken, permissions, actualUserData);

    try {
        await actualEndpoint(actualParams, { req, res }, actualSelect);
        expect.fail("Expected an error to be thrown");
    } catch (error) {
        expect(error).to.have.property("code", "Unauthorized");
    }
}

/**
 * Helper function to test that an endpoint requires API key with private permissions.
 * This should be called from within an it() block.
 * @param userData - The user data for authentication (optional, defaults to standard test user)
 * @param endpoint - The endpoint to test
 * @param params - The parameters to pass to the endpoint
 * @param select - The selection shape to pass to the endpoint
 * @param objectTypes - Optional array of object types (ignored, for backward compatibility)
 */
export async function assertEndpointRequiresApiKeyPrivatePermissions(
    userData: UserDataForPasswordAuth | Function,
    endpoint?: Function | object,
    params?: object,
    select?: object | string[],
    objectTypes?: string[],
) {
    // Handle overloaded function signatures
    let actualUserData: UserDataForPasswordAuth;
    let actualEndpoint: Function;
    let actualParams: object;
    let actualSelect: object;
    
    if (typeof userData === "function") {
        // Called without userData parameter
        actualUserData = { ...loggedInUserNoPremiumData(), id: "test-user-id" };
        actualEndpoint = userData;
        actualParams = endpoint as object;
        actualSelect = params as object;
        // The 4th parameter might be objectTypes array
    } else {
        // Called with userData parameter
        actualUserData = userData;
        actualEndpoint = endpoint as Function;
        actualParams = params!;
        actualSelect = select as object;
    }

    const permissions = {
        [ApiKeyPermission.ReadPublic]: true,
        [ApiKeyPermission.ReadPrivate]: false,
        [ApiKeyPermission.WritePrivate]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.ReadAuth]: false,
    } as Record<ApiKeyPermission, boolean>;
    const apiToken = ApiKeyEncryptionService.generateSiteKey();
    const { req, res } = await mockApiSession(apiToken, permissions, actualUserData);

    try {
        await actualEndpoint(actualParams, { req, res }, actualSelect);
        expect.fail("Expected an error to be thrown");
    } catch (error) {
        expect(error).to.have.property("code", "Unauthorized");
    }
}

/**
 * Helper function to test that an endpoint requires API key with write permissions.
 * This should be called from within an it() block.
 * @param userData - The user data for authentication (optional, defaults to standard test user)
 * @param endpoint - The endpoint to test
 * @param params - The parameters to pass to the endpoint
 * @param select - The selection shape to pass to the endpoint
 * @param objectTypes - Optional array of object types (ignored, for backward compatibility)
 */
export async function assertEndpointRequiresApiKeyWritePermissions(
    userData: UserDataForPasswordAuth | Function,
    endpoint?: Function | object,
    params?: object,
    select?: object | string[],
    objectTypes?: string[],
) {
    // Handle overloaded function signatures
    let actualUserData: UserDataForPasswordAuth;
    let actualEndpoint: Function;
    let actualParams: object;
    let actualSelect: object;
    
    if (typeof userData === "function") {
        // Called without userData parameter
        actualUserData = { ...loggedInUserNoPremiumData(), id: "test-user-id" };
        actualEndpoint = userData;
        actualParams = endpoint as object;
        actualSelect = params as object;
        // The 4th parameter might be objectTypes array
    } else {
        // Called with userData parameter
        actualUserData = userData;
        actualEndpoint = endpoint as Function;
        actualParams = params!;
        actualSelect = select as object;
    }

    const permissions = {
        [ApiKeyPermission.ReadPublic]: true,
        [ApiKeyPermission.ReadPrivate]: true,
        [ApiKeyPermission.WritePrivate]: false,
        [ApiKeyPermission.WriteAuth]: false,
        [ApiKeyPermission.ReadAuth]: false,
    } as Record<ApiKeyPermission, boolean>;
    const apiToken = ApiKeyEncryptionService.generateSiteKey();
    const { req, res } = await mockApiSession(apiToken, permissions, actualUserData);

    try {
        await actualEndpoint(actualParams, { req, res }, actualSelect);
        expect.fail("Expected an error to be thrown");
    } catch (error) {
        expect(error).to.have.property("code", "Unauthorized");
    }
}

/**
 * @deprecated Use assertEndpointRequiresAuth instead - this function creates nested test blocks
 */
export function testEndpointRequiresAuth(
    endpoint: Function,
    params: object,
    select: object,
) {
    it("not logged in", async () => {
        await assertEndpointRequiresAuth(endpoint, params, select);
    });
}

/**
 * @deprecated Use assertEndpointRequiresApiKeyReadPermissions instead - this function creates nested test blocks
 */
export function testEndpointRequiresApiKeyReadPermissions(
    userData: UserDataForPasswordAuth | Function,
    endpoint?: Function | object,
    params?: object,
    select?: object | string[],
    objectTypes?: string[],
) {
    it("API key - no read permissions", async () => {
        if (typeof userData === "function") {
            await assertEndpointRequiresApiKeyReadPermissions(userData, endpoint, params, select, objectTypes);
        } else {
            await assertEndpointRequiresApiKeyReadPermissions(userData, endpoint, params, select, objectTypes);
        }
    });
}

/**
 * @deprecated Use assertEndpointRequiresApiKeyPrivatePermissions instead - this function creates nested test blocks
 */
export function testEndpointRequiresApiKeyPrivatePermissions(
    userData: UserDataForPasswordAuth,
    endpoint: Function,
    params: object,
    select: object,
) {
    it("API key - no private permissions", async () => {
        await assertEndpointRequiresApiKeyPrivatePermissions(userData, endpoint, params, select);
    });
}

/**
 * @deprecated Use assertEndpointRequiresApiKeyWritePermissions instead - this function creates nested test blocks
 */
export function testEndpointRequiresApiKeyWritePermissions(
    userData: UserDataForPasswordAuth | Function,
    endpoint?: Function | object,
    params?: object,
    select?: object | string[],
    objectTypes?: string[],
) {
    it("API key - no write permissions", async () => {
        if (typeof userData === "function") {
            await assertEndpointRequiresApiKeyWritePermissions(userData, endpoint, params, select, objectTypes);
        } else {
            await assertEndpointRequiresApiKeyWritePermissions(userData, endpoint, params, select, objectTypes);
        }
    });
}

//TODO add test for fake api key