import { ApiKeyPermission } from "@local/shared";
import { expect } from "chai";
import { ApiKeyEncryptionService } from "../auth/apiKeyEncryption.js";
import { UserDataForPasswordAuth } from "../auth/email.js";
import { mockApiSession, mockLoggedOutSession } from "./session.js";

/**
 * Tests that the endpoint requires authentication
 * @param endpoint - The endpoint to test
 * @param params - The parameters to pass to the endpoint
 * @param select - The selection shape to pass to the endpoint
 */
export function testEndpointRequiresAuth(
    endpoint: Function,
    params: object,
    select: object,
) {
    it("not logged in", async () => {
        const { req, res } = await mockLoggedOutSession();

        try {
            await endpoint(params, { req, res }, select);
            expect.fail("Expected an error to be thrown");
        } catch (error) {
            expect(error).to.have.property("code", "Unauthorized");
        }
    });
}

export function testEndpointRequiresApiKeyPrivatePermissions(
    userData: UserDataForPasswordAuth,
    endpoint: Function,
    params: object,
    select: object,
) {
    it("API key - no private permissions", async () => {
        const permissions = {
            [ApiKeyPermission.ReadPublic]: true,
            [ApiKeyPermission.ReadPrivate]: false,
            [ApiKeyPermission.WritePrivate]: false,
            [ApiKeyPermission.WriteAuth]: false,
            [ApiKeyPermission.ReadAuth]: false,
        } as Record<ApiKeyPermission, boolean>;
        const apiToken = ApiKeyEncryptionService.generateSiteKey();
        const { req, res } = await mockApiSession(apiToken, permissions, userData);

        try {
            await endpoint(params, { req, res }, select);
            expect.fail("Expected an error to be thrown");
        } catch (error) {
            expect(error).to.have.property("code", "Unauthorized");
        }
    });
}

export function testEndpointRequiresApiKeyWritePermissions(
    userData: UserDataForPasswordAuth,
    endpoint: Function,
    params: object,
    select: object,
) {
    it("API key - no write permissions", async () => {
        const permissions = {
            [ApiKeyPermission.ReadPublic]: true,
            [ApiKeyPermission.ReadPrivate]: true,
            [ApiKeyPermission.WritePrivate]: false,
            [ApiKeyPermission.WriteAuth]: false,
            [ApiKeyPermission.ReadAuth]: false,
        } as Record<ApiKeyPermission, boolean>;
        const apiToken = ApiKeyEncryptionService.generateSiteKey();
        const { req, res } = await mockApiSession(apiToken, permissions, userData);

        try {
            await endpoint(params, { req, res }, select);
            expect.fail("Expected an error to be thrown");
        } catch (error) {
            expect(error).to.have.property("code", "Unauthorized");
        }
    });
}

//TODO add test for fake api key