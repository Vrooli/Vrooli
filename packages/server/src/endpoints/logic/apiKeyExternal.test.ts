import { type ApiKeyExternalCreateInput, type ApiKeyExternalUpdateInput, generatePK } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { assertRequiresAuth } from "../../__test/authTestUtils.js";
import { loggedInUserNoPremiumData, mockApiSession, mockWriteAuthPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import { apiKeyExternal_createOne } from "../generated/apiKeyExternal_createOne.js";
import { apiKeyExternal_updateOne } from "../generated/apiKeyExternal_updateOne.js";
import { apiKeyExternal } from "./apiKeyExternal.js";
// Import database fixtures
import { ApiKeyExternalDbFactory } from "../../__test/fixtures/db/apiKeyExternalFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

describe("EndpointsApiKeyExternal", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["user", "user_auth", "email", "session"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.minimal(DbProvider.get());
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("createOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    apiKeyExternal.createOne,
                    {
                        id: generatePK(),
                        keyData: "test-key-data",
                        name: "Test External API Key",
                        provider: "github",
                    },
                    apiKeyExternal_createOne,
                );
            });

            it("requires official user account", async () => {
                // Create guest user
                const guestUser = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: "guest_user",
                        name: "Guest User",
                        status: "Guest",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                    },
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: guestUser.id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "test-key-data",
                    name: "Test External API Key",
                    provider: "github",
                };

                await expect(apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne))
                    .rejects.toThrow(CustomError);
            });

            it("requires API key with auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWritePrivatePermissions(["ApiKeyExternal"]), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "test-key-data",
                    name: "Test External API Key",
                    provider: "github",
                };

                await expect(apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("creates external API key for official user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "github_token_abc123",
                    name: "GitHub API Key",
                    provider: "github",
                    description: "For accessing GitHub repositories",
                };

                const result = await apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne);

                expect(result).toBeDefined();
                expect(result.__typename).toBe("ApiKeyExternal");
                expect(result.id).toBe(input.id);
                expect(result.name).toBe("GitHub API Key");
                expect(result.provider).toBe("github");
                expect(result.description).toBe("For accessing GitHub repositories");
                expect(result.user?.id).toBe(testUser[0].id.toString());

                // Verify key data is not exposed in response
                expect(result.keyData).toBeUndefined();

                // Verify in database
                const createdKey = await DbProvider.get().apiKeyExternal.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdKey).toBeDefined();
                expect(createdKey?.keyData).toBe("github_token_abc123");
                expect(createdKey?.name).toBe("GitHub API Key");
            });

            it("creates API key with minimal required fields", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "minimal_key_123",
                    name: "Minimal Key",
                    provider: "custom",
                };

                const result = await apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne);

                expect(result.name).toBe("Minimal Key");
                expect(result.provider).toBe("custom");
                expect(result.description).toBeUndefined();
            });

            it("creates API key for different providers", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const providers = ["github", "gitlab", "bitbucket", "openai", "anthropic"];

                for (const provider of providers) {
                    const input: ApiKeyExternalCreateInput = {
                        id: generatePK(),
                        keyData: `${provider}_key_${Date.now()}`,
                        name: `${provider.charAt(0).toUpperCase() + provider.slice(1)} Key`,
                        provider,
                    };

                    const result = await apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne);

                    expect(result.provider).toBe(provider);
                    expect(result.name).toBe(`${provider.charAt(0).toUpperCase() + provider.slice(1)} Key`);
                }
            });

            it("encrypts key data in storage", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const sensitiveKey = "very_secret_api_key_12345";
                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: sensitiveKey,
                    name: "Sensitive Key",
                    provider: "openai",
                };

                const result = await apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne);

                expect(result).toBeDefined();

                // Verify key data is encrypted in database
                const storedKey = await DbProvider.get().apiKeyExternal.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(storedKey?.keyData).toBeDefined();
                // The stored key should be encrypted, not the plain text
                expect(storedKey?.keyData).not.toBe(sensitiveKey);
            });
        });

        describe("invalid", () => {
            it("rejects duplicate name for same user", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create existing API key
                await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Duplicate Name",
                        provider: "github",
                        userId: testUser[0].id,
                    }),
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "new_key_data",
                    name: "Duplicate Name", // Same name
                    provider: "gitlab",
                };

                await expect(apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne))
                    .rejects.toThrow();
            });

            it("rejects empty key data", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "", // Empty key data
                    name: "Empty Key",
                    provider: "github",
                };

                await expect(apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne))
                    .rejects.toThrow();
            });

            it("rejects invalid provider name", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalCreateInput = {
                    id: generatePK(),
                    keyData: "valid_key_data",
                    name: "Invalid Provider Key",
                    provider: "", // Empty provider
                };

                await expect(apiKeyExternal.createOne({ input }, { req, res }, apiKeyExternal_createOne))
                    .rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("authentication", () => {
            it("not logged in", async () => {
                await assertRequiresAuth(
                    apiKeyExternal.updateOne,
                    { id: generatePK().toString() },
                    apiKeyExternal_updateOne,
                );
            });

            it("requires official user account", async () => {
                // Create guest user
                const guestUser = await DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: "guest_user",
                        name: "Guest User",
                        status: "Guest",
                        isBot: false,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                    },
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: guestUser.id,
                });

                const input: ApiKeyExternalUpdateInput = {
                    id: generatePK().toString(),
                    name: "Updated Name",
                };

                await expect(apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("requires API key with auth write permissions", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWritePrivatePermissions(["ApiKeyExternal"]), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalUpdateInput = {
                    id: generatePK().toString(),
                    name: "Updated Name",
                };

                await expect(apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne))
                    .rejects.toThrow(CustomError);
            });
        });

        describe("valid", () => {
            it("updates own API key", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create existing API key
                const existingKey = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Original Name",
                        provider: "github",
                        userId: testUser[0].id,
                        description: "Original description",
                    }),
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalUpdateInput = {
                    id: existingKey.id.toString(),
                    name: "Updated GitHub Key",
                    description: "Updated description for GitHub access",
                };

                const result = await apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne);

                expect(result.name).toBe("Updated GitHub Key");
                expect(result.description).toBe("Updated description for GitHub access");
                expect(result.provider).toBe("github"); // Unchanged
                expect(result.id).toBe(existingKey.id.toString());
            });

            it("updates key data", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                const existingKey = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Key to Update",
                        provider: "openai",
                        userId: testUser[0].id,
                    }),
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const newKeyData = "updated_api_key_data_456";
                const input: ApiKeyExternalUpdateInput = {
                    id: existingKey.id.toString(),
                    keyData: newKeyData,
                    name: "Updated OpenAI Key",
                };

                const result = await apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne);

                expect(result.name).toBe("Updated OpenAI Key");

                // Verify key data is not exposed in response
                expect(result.keyData).toBeUndefined();

                // Verify key data was updated in database (encrypted)
                const updatedKey = await DbProvider.get().apiKeyExternal.findUnique({
                    where: { id: existingKey.id },
                });
                expect(updatedKey?.keyData).toBeDefined();
                expect(updatedKey?.keyData).not.toBe(newKeyData); // Should be encrypted
            });

            it("updates last used timestamp", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                const existingKey = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Key with Usage",
                        provider: "anthropic",
                        userId: testUser[0].id,
                    }),
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const lastUsedAt = new Date();
                const input: ApiKeyExternalUpdateInput = {
                    id: existingKey.id.toString(),
                    lastUsedAt: lastUsedAt.toISOString(),
                };

                const result = await apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne);

                expect(result.lastUsedAt).toBeDefined();
                const resultDate = new Date(result.lastUsedAt!);
                expect(Math.abs(resultDate.getTime() - lastUsedAt.getTime())).toBeLessThan(1000); // Within 1 second
            });

            it("can disable and re-enable API key", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                const existingKey = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Toggle Key",
                        provider: "custom",
                        userId: testUser[0].id,
                        isActive: true,
                    }),
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                // Disable the key
                const disableInput: ApiKeyExternalUpdateInput = {
                    id: existingKey.id.toString(),
                    isActive: false,
                };

                const disabledResult = await apiKeyExternal.updateOne({ input: disableInput }, { req, res }, apiKeyExternal_updateOne);
                expect(disabledResult.isActive).toBe(false);

                // Re-enable the key
                const enableInput: ApiKeyExternalUpdateInput = {
                    id: existingKey.id.toString(),
                    isActive: true,
                };

                const enabledResult = await apiKeyExternal.updateOne({ input: enableInput }, { req, res }, apiKeyExternal_updateOne);
                expect(enabledResult.isActive).toBe(true);
            });
        });

        describe("invalid", () => {
            it("cannot update another user's API key", async () => {
                const testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

                // Create API key for user 1
                const user1Key = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "User 1 Key",
                        provider: "github",
                        userId: testUsers[0].id,
                    }),
                });

                // Try to update as user 2
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id,
                });

                const input: ApiKeyExternalUpdateInput = {
                    id: user1Key.id.toString(),
                    name: "Hacked Key",
                };

                await expect(apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects update to non-existent key", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                const input: ApiKeyExternalUpdateInput = {
                    id: generatePK().toString(),
                    name: "Non-existent Key",
                };

                await expect(apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne))
                    .rejects.toThrow(CustomError);
            });

            it("rejects duplicate name update", async () => {
                const testUser = await seedTestUsers(DbProvider.get(), 1, { withAuth: true });

                // Create two API keys
                const key1 = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Existing Key",
                        provider: "github",
                        userId: testUser[0].id,
                    }),
                });

                const key2 = await DbProvider.get().apiKeyExternal.create({
                    data: ApiKeyExternalDbFactory.createMinimal({
                        name: "Another Key",
                        provider: "gitlab",
                        userId: testUser[0].id,
                    }),
                });

                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, mockWriteAuthPermissions(), {
                    ...loggedInUserNoPremiumData(),
                    id: testUser[0].id,
                });

                // Try to update key2 to have the same name as key1
                const input: ApiKeyExternalUpdateInput = {
                    id: key2.id.toString(),
                    name: "Existing Key", // Duplicate name
                };

                await expect(apiKeyExternal.updateOne({ input }, { req, res }, apiKeyExternal_updateOne))
                    .rejects.toThrow();
            });
        });
    });
});
