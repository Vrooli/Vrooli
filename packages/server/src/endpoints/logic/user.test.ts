import { type FindByIdOrHandleInput, type UserSearchInput, type UserUpdateInput, SEEDED_IDS, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { testEndpointRequiresApiKeyWritePermissions, testEndpointRequiresAuth } from "../../__test/endpoints.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { user_findMany } from "../generated/user_findMany.js";
import { user_findOne } from "../generated/user_findOne.js";
import { user_updateOne } from "../generated/user_updateOne.js";
import { user } from "./user.js";

// Import database fixtures for seeding
import { UserDbFactory } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing
import { userTestDataFactory } from "@vrooli/shared";

describe("EndpointsUser", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        const prisma = DbProvider.get();
        await prisma.user.deleteMany();
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("own profile", async () => {
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
                    id: testUser.id.toString()
                });

                const input: FindByIdOrHandleInput = { id: testUser.id.toString() };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser.id.toString());
                expect(result.name).toBe(testUser.name);
                expect(result.handle).toBe(testUser.handle);
            });

            it("own private profile", async () => {
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
                    id: privateUser.id.toString()
                });

                const input: FindByIdOrHandleInput = { id: privateUser.id.toString() };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(privateUser.id.toString());
                expect(result.isPrivate).toBe(true);
            });

            it("public profile", async () => {
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
                    id: testUser1.id.toString()
                });

                const input: FindByIdOrHandleInput = { id: testUser2.id.toString() };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser2.id.toString());
                expect(result.name).toBe(testUser2.name);
                expect(result.handle).toBe(testUser2.handle);
            });

            it("API key - public permissions", async () => {
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
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = { id: testUser2.id.toString() };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser2.id.toString());
                expect(result.name).toBe(testUser2.name);
                expect(result.handle).toBe(testUser2.handle);
            });

            it("not logged in", async () => {
                // Create test user
                const testUser = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const { req, res } = await mockLoggedOutSession();

                const input = { id: testUser.id.toString() };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUser.id.toString());
                expect(result.name).toBe(testUser.name);
                expect(result.handle).toBe(testUser.handle);
            });
        });

        describe("invalid", () => {
            it("private profile", async () => {
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

                const input = { id: testUser2.id.toString() };

                await expect(async () => {
                    await user.findOne({ input }, { req, res }, user_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("retuns public profiles", async () => {
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

            it("search input returns results", async () => {
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

            it("API key - public permissions", async () => {
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

            it("not logged in", async () => {
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

                const { req, res } = await mockLoggedOutSession();

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
        describe("valid", () => {
            it("should create a bot user that you own", async () => {
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
                const input = userTestDataFactory.createBot({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                        systemPrompt: "You are a helpful assistant.",
                    }),
                });

                const result = await user.botCreateOne({ input }, { req, res }, user_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.name).toBe(input.name);
                expect(result.handle).toBe(input.handle);
                expect(result.isBot).toBe(true);
                expect(result.isBotDepictingPerson).toBe(input.isBotDepictingPerson);

                // Verify bot was created in database
                const createdBot = await DbProvider.get().user.findUnique({
                    where: { id: input.id },
                });
                expect(createdBot).not.toBeNull();
                expect(createdBot?.botSettings).toBe(input.botSettings);
            });

            it("API key - write permissions", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = userTestDataFactory.createBot({
                    name: "API Key Bot",
                    handle: "api-bot-" + Math.floor(Math.random() * 1000),
                    isBotDepictingPerson: true,
                    isPrivate: true,
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "You are a specialized bot created via API.",
                    }),
                });

                const result = await user.botCreateOne({ input }, { req, res }, user_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(input.id);
                expect(result.name).toBe(input.name);
                expect(result.isBot).toBe(true);
                // isPrivate is not returned by the endpoint, so we'll query the database directly
                const createdBot = await DbProvider.get().user.findUnique({
                    where: { id: input.id },
                });
                expect(createdBot).not.toBeNull();
                expect(createdBot?.isPrivate).toBe(input.isPrivate);
            });
        });

        describe("invalid", () => {
            it("same ID as existing user", async () => {
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

                const input = userTestDataFactory.createBot({
                    id: testUser2.id.toString(), // Using existing user ID
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                });

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("same ID as admin", async () => {
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

                const input = userTestDataFactory.createBot({
                    id: SEEDED_IDS.User.Admin, // Using admin ID
                    name: "Admin Bot",
                    handle: "admin-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                });

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("trying to set `isBot` to false", async () => {
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
                    ...userTestDataFactory.createBot({
                        name: "Not A Bot",
                        handle: "not-a-bot-" + Math.floor(Math.random() * 1000),
                        botSettings: JSON.stringify({
                            model: "gpt-3.5-turbo",
                        }),
                    }),
                    isBot: false, // This should cause validation to fail
                };

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("trying to update user-specific fields", async () => {
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
                    ...userTestDataFactory.createBot({
                        name: "Test Bot",
                        handle: "test-bot-" + Math.floor(Math.random() * 1000),
                        botSettings: JSON.stringify({
                            model: "gpt-3.5-turbo",
                        }),
                    }),
                    status: "HardLocked", // User-specific field that shouldn't be allowed
                    notificationSettings: JSON.stringify({ disable: true }), // User-specific field
                };

                await expect(async () => {
                    await user.botCreateOne({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            // Skip this test for now since it requires shared state

            // Skip this test for now since it requires shared state
        });
    });

    describe("botUpdateOne", () => {
        describe("valid", () => {
            it("should update a bot user that you own", async () => {
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

                const botData = userTestDataFactory.createBot({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                        systemPrompt: "Initial prompt",
                    }),
                });

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_updateOne);

                // Update the bot
                const updateInput: UserUpdateInput = {
                    id: botData.id,
                    name: "Updated Bot Name",
                    isPrivate: true,
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "Updated prompt",
                    }),
                };

                const result = await user.botUpdateOne({ input: updateInput }, { req, res }, user_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(botData.id);
                expect(result.name).toBe(updateInput.name);

                // Verify bot was updated in database
                const updatedBot = await DbProvider.get().user.findUnique({
                    where: { id: botData.id },
                });
                expect(updatedBot).not.toBeNull();
                expect(updatedBot?.name).toBe(updateInput.name);
                expect(updatedBot?.botSettings).toBe(updateInput.botSettings);
                expect(updatedBot?.isPrivate).toBe(updateInput.isPrivate);
            });

            it("API key - write permissions", async () => {
                // Create test user
                const testUserRecord = await DbProvider.get().user.create({
                    data: UserDbFactory.createWithAuth({
                        id: generatePK(),
                        name: "Test User",
                        handle: "test-user-" + Math.floor(Math.random() * 1000),
                    }),
                });

                const testUser = { ...loggedInUserNoPremiumData(), id: testUserRecord.id.toString() };
                const { req: createReq, res: createRes } = await mockAuthenticatedSession(testUser);

                const botData = userTestDataFactory.createBot({
                    name: "API Bot",
                    handle: "api-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                        systemPrompt: "Initial API prompt",
                    }),
                });

                // Create the bot
                await user.botCreateOne({ input: botData }, { req: createReq, res: createRes }, user_updateOne);

                // Update the bot with API key
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const updateInput: UserUpdateInput = {
                    id: botData.id,
                    name: "API Updated Bot",
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "Updated via API",
                    }),
                };

                const result = await user.botUpdateOne({ input: updateInput }, { req, res }, user_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(botData.id);
                expect(result.name).toBe(updateInput.name);

                // Verify bot was updated in database
                const updatedBot = await DbProvider.get().user.findUnique({
                    where: { id: botData.id },
                });
                expect(updatedBot).not.toBeNull();
                expect(updatedBot?.name).toBe(updateInput.name);
                expect(updatedBot?.botSettings).toBe(updateInput.botSettings);
            });
        });

        describe("invalid", () => {
            it("updating a different user's bot", async () => {
                // First create a bot owned by user1
                const user2 = { ...loggedInUserNoPremiumData(), id: testUsers[1].id };
                const { req: user2Req, res: user2Res } = await mockAuthenticatedSession(user2);

                const botData = userTestDataFactory.createBot({
                    name: "User2's Bot",
                    handle: "user2-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                });

                // Create the bot
                await user.botCreateOne({ input: botData }, { req: user2Req, res: user2Res }, user_updateOne);

                // Try to update the bot as user0
                const user1 = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(user1);

                const updateInput: UserUpdateInput = {
                    id: botData.id,
                    name: "Hijacked Bot",
                };

                await expect(async () => {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("updating admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const updateInput: UserUpdateInput = {
                    id: SEEDED_IDS.User.Admin,
                    name: "Hacked Admin",
                };

                await expect(async () => {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("trying to set `isBot` to false", async () => {
                // First create a bot
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botData = userTestDataFactory.createBot({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                });

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_updateOne);

                // Try to update isBot to false
                const updateInput = {
                    id: botData.id,
                    isBot: false, // This should cause validation to fail
                } as UserUpdateInput;

                await expect(async () => {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("trying to update user-specific fields", async () => {
                // First create a bot
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botData = userTestDataFactory.createBot({
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                });

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_updateOne);

                // Try to update with user-specific fields
                const updateInput = {
                    id: botData.id,
                    name: "Updated Bot",
                    status: "HardLocked", // User-specific field
                    notificationSettings: JSON.stringify({ disable: true }), // User-specific field
                } as any;

                await expect(async () => {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            testEndpointRequiresAuth(user.botUpdateOne, {
                input: {
                    id: userTestDataFactory.createBot({}).id,
                    name: "Test Bot Update",
                },
            }, user_updateOne);

            const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
            testEndpointRequiresApiKeyWritePermissions(
                testUser,
                user.botUpdateOne,
                {
                    input: {
                        id: userTestDataFactory.createBot({}).id,
                        name: "Test Bot Update",
                    },
                },
                user_updateOne,
            );
        });
    });

    describe("profileUpdate", () => {
        describe("valid", () => {
            it("should update user profile information", async () => {
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

                const input: UserUpdateInput = {
                    id: testUser.id,
                    name: "Updated Name",
                };
                const result = await user.profileUpdate({ input }, { req, res }, user_updateOne);

                expect(result.id).toBe(testUser.id);
                expect(result.name).toBe(input.name);

                const updatedUser = await DbProvider.get().user.findUnique({
                    where: { id: testUser.id },
                    select: { id: true, name: true },
                });
                expect(updatedUser).not.toBeNull();
                expect(updatedUser?.name).toBe(input.name);
            });

            it("should handle translationsCreate correctly", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: UserUpdateInput = {
                    id: testUser.id,
                    translationsCreate: [{
                        id: userTestDataFactory.createMinimal({}).id,
                        language: "es",
                        bio: "Biografía de prueba",
                    }],
                };
                const result = await user.profileUpdate({ input }, { req, res }, user_updateOne);

                expect(result.id).toBe(testUser.id);
                expect(result.translations).toBeInstanceOf(Array);

                const updatedUser = await DbProvider.get().user.findUnique({
                    where: { id: testUser.id },
                    include: { translations: true },
                });
                expect(updatedUser).not.toBeNull();
                expect(updatedUser?.translations).toBeInstanceOf(Array);
                expect(updatedUser?.translations.length).toBeGreaterThan(0);
                const createdTranslation = updatedUser?.translations.find((t) => t.language === input.translationsCreate![0].language);
                expect(createdTranslation).not.toBeNull();
                expect(createdTranslation?.id).toBe(input.translationsCreate![0].id);
                expect(createdTranslation?.bio).toBe(input.translationsCreate![0].bio);
            });

            it("API key - write permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: UserUpdateInput = {
                    id: testUser.id,
                    name: "Updated Name",
                };
                const result = await user.profileUpdate({ input }, { req, res }, user_updateOne);

                expect(result.id).toBe(testUser.id);
                expect(result.name).toBe(input.name);
            });
        });

        describe("invalid", () => {
            it("updating a different user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: UserUpdateInput = {
                    id: testUsers[1].id,
                    name: "Updated Name",
                };
                await expect(async () => {
                    await user.profileUpdate({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            it("updating admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: UserUpdateInput = {
                    id: SEEDED_IDS.User.Admin,
                    name: "Updated Name",
                };
                await expect(async () => {
                    await user.profileUpdate({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });
            it("updating a non-existent user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input: UserUpdateInput = {
                    id: userTestDataFactory.createMinimal({}).id,
                    name: "Updated Name",
                };
                await expect(async () => {
                    await user.profileUpdate({ input }, { req, res }, user_updateOne);
                }).rejects.toThrow();
            });

            testEndpointRequiresAuth(user.profileUpdate, { input: { id: testUsers[0].id, name: "Updated Name" } }, user_updateOne);

            const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
            testEndpointRequiresApiKeyWritePermissions(testUser, user.profileUpdate, { input: { id: testUsers[0].id, name: "Updated Name" } }, user_updateOne);
        });
    });
}); 
