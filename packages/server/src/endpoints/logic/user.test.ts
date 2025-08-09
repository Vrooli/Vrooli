import { type BotUpdateInput, type FindByPublicIdInput, generatePK, ModelStrategy, type ProfileUpdateInput, SEEDED_PUBLIC_IDS, type UserSearchInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, test, vi } from "vitest";
import { assertRequiresAuth } from "../../__test/authTestUtils.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { user_botCreateOne } from "../generated/user_botCreateOne.js";
import { user_botUpdateOne } from "../generated/user_botUpdateOne.js";
import { user_findMany } from "../generated/user_findMany.js";
import { user_findOne } from "../generated/user_findOne.js";
import { user_profileUpdate } from "../generated/user_profileUpdate.js";
import { user } from "./user.js";
// Import database fixtures for seeding
import { UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";
// Import validation fixtures for API input testing
// eslint-disable-next-line import/extensions
import { userTestDataFactory } from "@vrooli/shared/test-fixtures/api-inputs";
import { cleanupGroups, ensureCleanState, performTestCleanup } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

// Helper function to create admin user for tests (idempotent)
async function createTestAdminUser() {
    try {
        // Try to find existing admin user first
        const existing = await DbProvider.get().user.findUnique({
            where: { publicId: SEEDED_PUBLIC_IDS.Admin },
        });
        
        if (existing) {
            return existing; // Admin user already exists
        }
        
        // Create new admin user if it doesn't exist
        return await DbProvider.get().user.create({
            data: UserDbFactory.createWithAuth({
                id: generatePK(),
                publicId: SEEDED_PUBLIC_IDS.Admin,
                name: "Admin User",
                handle: "__admin__",
                isPrivate: false,
                theme: "light",
            }),
        });
    } catch (error) {
        // If creation fails due to duplicate, try to find the existing one
        const existing = await DbProvider.get().user.findUnique({
            where: { publicId: SEEDED_PUBLIC_IDS.Admin },
        });
        if (existing) {
            return existing;
        }
        throw error; // Re-throw if it's a different error
    }
}

describe("EndpointsUser", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warning").mockImplementation(() => logger);
        
        // Create admin user required by the system
        await createTestAdminUser();
    });

    beforeEach(async () => {
        // Ensure clean database state with race condition protection
        await ensureCleanState(DbProvider.get(), {
            cleanupFn: cleanupGroups.minimal,
            tables: ["user", "user_auth", "email", "session"],
            throwOnFailure: true,
        });
        
        // Recreate admin user after cleanup (required by system)
        await createTestAdminUser();
        
        // Clear Redis cache to reset rate limiting
        await CacheService.get().flushAll();
    });

    afterEach(async () => {
        // Perform immediate cleanup after test to prevent test pollution
        await performTestCleanup(DbProvider.get(), {
            cleanupFn: cleanupGroups.minimal,
            tables: ["user", "user_auth", "email", "session"],
        });
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("authentication", () => {
            // Public endpoint - no authentication required
            it("allows unauthenticated access to public profiles", async () => {
                // Create test user
                const testUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const { req, res } = await mockLoggedOutSession();

                const input = { publicId: testUser.publicId };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser.id.toString());
            });
        });

        describe("valid", () => {
            it("returns own profile", async () => {
                // Create test user
                const testUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUser.id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: testUser.publicId };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser.id.toString());
                expect(result.name).toBe(testUser.name);
                expect(result.handle).toBe(testUser.handle);
            });

            it("returns own private profile", async () => {
                // Create private test user
                const privateUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createMinimal({
                        id: generatePK(),
                        name: "Private User",
                        handle: "private-user-" + Math.floor(Math.random() * 1000),
                        isPrivate: true,
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: privateUser.id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: privateUser.publicId };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(privateUser.id.toString());
                expect(result.isPrivate).toBe(true);
            });

            it("returns public profile", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUser1.id.toString(),
                });

                const input: FindByPublicIdInput = { publicId: testUser2.publicId };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser2.id.toString());
                expect(result.name).toBe(testUser2.name);
                expect(result.handle).toBe(testUser2.handle);
            });

            it("supports API key with public permissions", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = { publicId: testUser2.publicId };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser2.id.toString());
                expect(result.name).toBe(testUser2.name);
                expect(result.handle).toBe(testUser2.handle);
            });
        });

        describe("invalid", () => {
            it("blocks access to private profile", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                        isPrivate: true,
                    }),
                });

                // Attempt to access user2's profile from user1's account
                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { publicId: testUser2.publicId };

                await expect(async () => {
                    await user.findOne({ input }, { req, res }, user_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("authentication", () => {
            // Public endpoint - no authentication required
            it("allows unauthenticated access", async () => {
                // Create test users
                await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const { req, res } = await mockLoggedOutSession();

                const input: UserSearchInput = { take: 10 };
                const result = await user.findMany({ input }, { req, res }, user_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
            });
        });

        describe("valid", () => {
            it("returns public profiles", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: UserSearchInput = { take: 10 };
                const result = await user.findMany({ input }, { req, res }, user_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges!.length).toBeGreaterThanOrEqual(2);
                expect(result.edges!.some(edge => edge?.node?.id === testUser1.id.toString())).toBe(true);
                expect(result.edges!.some(edge => edge?.node?.id === testUser2.id.toString())).toBe(true);
            });

            it("filters results by search string", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: UserSearchInput = { searchString: testUser2.name };
                const result = await user.findMany({ input }, { req, res }, user_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges!.length).toBeGreaterThanOrEqual(1);
                expect(result.edges!.some(edge => edge?.node?.id === testUser2.id.toString())).toBe(true);
            });

            it("supports API key with public permissions", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: UserSearchInput = { take: 10 };
                const result = await user.findMany({ input }, { req, res }, user_findMany);

                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges!.length).toBeGreaterThanOrEqual(2);
                expect(result.edges!.some(edge => edge?.node?.id === testUser1.id.toString())).toBe(true);
                expect(result.edges!.some(edge => edge?.node?.id === testUser2.id.toString())).toBe(true);
            });
        });
    });

    describe("botCreateOne", () => {
        describe("authentication", () => {
            // Hybrid approach: use direct assertion for simple auth check
            it("not logged in", async () => {
                const input = userTestDataFactory.createComplete({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: {
                        __version: "1.0",
                        modelConfig: {
                            strategy: ModelStrategy.FIXED,
                            preferredModel: "gpt-3.5-turbo",
                            offlineOnly: false,
                        },
                        resources: [],
                    },
                });
                await assertRequiresAuth(user.botCreateOne, input, user_botCreateOne);
            });

            // Hybrid approach: use test.each for API key scenarios
            test.each([
                {
                    name: "API key with write permissions",
                    permissions: mockWritePrivatePermissions(),
                    shouldSucceed: true,
                },
            ])("$name", async ({ permissions, shouldSucceed }) => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id };
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = userTestDataFactory.createComplete({
                    name: "API Key Bot",
                    handle: "api-bot-" + Math.floor(Math.random() * 1000),
                    isBotDepictingPerson: true,
                    isPrivate: true,
                    botSettings: {
                        __version: "1.0",
                        modelConfig: {
                            strategy: ModelStrategy.FIXED,
                            preferredModel: "gpt-4",
                            offlineOnly: false,
                        },
                        resources: [],
                    },
                });

                if (shouldSucceed) {
                    const result = await user.botCreateOne({ input }, { req, res }, user_botCreateOne);
                    expect(result).not.toBeNull();
                    expect(result.id).toBe(input.id);
                } else {
                    await expect(async () => {
                        await user.botCreateOne({ input }, { req, res }, user_botCreateOne);
                    }).rejects.toThrow();
                }
            });
        });

        describe("valid", () => {
            it("creates a bot user that you own", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                // Use validation fixtures for API input
                const input = userTestDataFactory.createComplete({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: {
                        __version: "1.0",
                        modelConfig: {
                            strategy: ModelStrategy.FIXED,
                            preferredModel: "gpt-3.5-turbo",
                            offlineOnly: false,
                        },
                        resources: [],
                    },
                });

                const result = await user.botCreateOne({ input }, { req, res }, user_botCreateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.name).toBe(input.name);
                expect(result.handle).toBe(input.handle);
                expect(result.isBot).toBe(true);
                expect(result.isBotDepictingPerson).toBe(input.isBotDepictingPerson);

                // Verify bot was created in database
                const createdBot = await DbProvider.get().user.findUnique({
                    where: { id: BigInt(input.id) },
                });
                expect(createdBot).not.toBeNull();
                expect(createdBot?.botSettings).toBe(input.botSettings);
            });
        });

        describe("invalid", () => {
            it("rejects duplicate ID", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = userTestDataFactory.createComplete({
                    id: testUser2.id.toString(), // Using existing user ID
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: {
                        __version: "1.0",
                        modelConfig: {
                            strategy: ModelStrategy.FIXED,
                            preferredModel: "gpt-3.5-turbo",
                            offlineOnly: false,
                        },
                        resources: [],
                    },
                });

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_botCreateOne);
                }).rejects.toThrow();
            });

            it("rejects admin ID", async () => {
                // Create test admin user
                const adminUser = await createTestAdminUser();

                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = userTestDataFactory.createComplete({
                    id: adminUser.id.toString(), // Using admin ID
                    name: "Admin Bot",
                    handle: "admin-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: {
                        __version: "1.0",
                        modelConfig: {
                            strategy: ModelStrategy.FIXED,
                            preferredModel: "gpt-3.5-turbo",
                            offlineOnly: false,
                        },
                        resources: [],
                    },
                });

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_botCreateOne);
                }).rejects.toThrow();
            });

            it("rejects isBot set to false", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    ...userTestDataFactory.createComplete({
                        name: "Not A Bot",
                        handle: "not-a-bot-" + Math.floor(Math.random() * 1000),
                        botSettings: {
                            __version: "1.0",
                            model: "gpt-3.5-turbo",
                        },
                    }),
                    isBot: false, // This should cause validation to fail
                };

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_botCreateOne);
                }).rejects.toThrow();
            });

            it("rejects user-specific fields", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    ...userTestDataFactory.createComplete({
                        name: "Test Bot",
                        handle: "test-bot-" + Math.floor(Math.random() * 1000),
                        botSettings: {
                            __version: "1.0",
                            model: "gpt-3.5-turbo",
                        },
                    }),
                    status: "HardLocked", // User-specific field that shouldn't be allowed
                    notificationSettings: JSON.stringify({ disable: true }), // User-specific field
                };

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_botCreateOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("botUpdateOne", () => {
        describe("authentication", () => {
            // Hybrid approach: use direct assertion for simple auth check
            it("not logged in", async () => {
                const input: BotUpdateInput = {
                    id: generatePK().toString(),
                    name: "Updated Bot Name",
                };
                await assertRequiresAuth(user.botUpdateOne, input, user_botUpdateOne);
            });
        });

        describe("valid", () => {
            it("updates a bot user that you own", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botData = userTestDataFactory.createComplete({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: {
                        __version: "1.0",
                        model: "gpt-3.5-turbo",
                        systemPrompt: "Initial prompt",
                    },
                });

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_botCreateOne);

                // Update the bot
                const updateInput: BotUpdateInput = {
                    id: botData.id,
                    name: "Updated Bot Name",
                    isPrivate: true,
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "Updated prompt",
                    }),
                };

                const result = await user.botUpdateOne({ input: updateInput }, { req, res }, user_botUpdateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(botData.id);
                expect(result.name).toBe("Updated Bot Name");
                expect(result.isBot).toBe(true);

                // Verify updates in database
                const updatedBot = await DbProvider.get().user.findUnique({
                    where: { id: BigInt(botData.id) },
                });
                expect(updatedBot).not.toBeNull();
                expect(updatedBot?.isPrivate).toBe(true);
                expect(updatedBot?.botSettings).toEqual({
                    model: "gpt-4",
                    systemPrompt: "Updated prompt",
                });
            });
        });

        describe("invalid", () => {
            it("cannot update bot owned by another user", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                // Create bot owned by user1
                const user1Session = { ...loggedInUserNoPremiumData(), id: testUser1.id.toString() };
                const { req: req1, res: res1 } = await mockAuthenticatedSession(user1Session);

                const botData = userTestDataFactory.createComplete({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: {
                        __version: "1.0",
                        model: "gpt-3.5-turbo",
                    },
                });

                await user.botCreateOne({ input: botData }, { req: req1, res: res1 }, user_botCreateOne);

                // Try to update as user2
                const user2Session = { ...loggedInUserNoPremiumData(), id: testUser2.id.toString() };
                const { req: req2, res: res2 } = await mockAuthenticatedSession(user2Session);

                const updateInput: BotUpdateInput = {
                    id: botData.id,
                    name: "Hacked Bot Name",
                };

                await expect(async () => {
                    await user.botUpdateOne({ input: updateInput }, { req: req2, res: res2 }, user_botUpdateOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("profileUpdate", () => {
        describe("authentication", () => {
            // Hybrid approach: use direct assertion for simple auth check
            it("not logged in", async () => {
                const input: ProfileUpdateInput = {
                    id: generatePK().toString(),
                    name: "Updated Name",
                };
                await assertRequiresAuth(user.profileUpdate, input, user_profileUpdate);
            });
        });

        describe("valid", () => {
            it("updates own profile", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const updateInput: ProfileUpdateInput = {
                    id: testUserRecord.id.toString(),
                    name: "Updated Name",
                    bio: "Updated bio",
                    theme: "dark",
                };

                const result = await user.profileUpdate({ input: updateInput }, { req, res }, user_profileUpdate);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUserRecord.id.toString());
                expect(result.name).toBe("Updated Name");
                expect(result.bio).toBe("Updated bio");
                expect(result.theme).toBe("dark");
            });
        });

        describe("invalid", () => {
            it("cannot update another user's profile", async () => {
                // Create test users
                const testUser1 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 1",
                        handle: "test-user-1-" + Math.floor(Math.random() * 1000),
                    }),
                });
                const testUser2 = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User 2",
                        handle: "test-user-2-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUser1.id.toString() };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const updateInput: ProfileUpdateInput = {
                    id: testUser2.id.toString(), // Trying to update user2
                    name: "Hacked Name",
                };

                await expect(async () => {
                    await user.profileUpdate({ input: updateInput }, { req, res }, user_profileUpdate);
                }).rejects.toThrow();
            });
        });
    });
});
