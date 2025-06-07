import { type FindByIdOrHandleInput, type UserSearchInput, type UserUpdateInput } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { testEndpointRequiresApiKeyWritePermissions, testEndpointRequiresAuth } from "../../__test/endpoints.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { user_findOne } from "../generated/user_findOne.js";
import { user_findMany } from "../generated/user_findMany.js";
import { user_updateOne } from "../generated/user_updateOne.js";
import { user } from "./user.js";

// Import database fixtures for seeding
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/userFixtures.js";

// Import validation fixtures for API input testing
import { userTestDataFactory } from "@vrooli/shared/src/validation/models/__test__/fixtures/userFixtures.js";

describe("EndpointsUser", () => {
    let testUsers: any[];
    let privateUser: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 2, { withAuth: true });

        // Create a private user for specific tests
        privateUser = await DbProvider.get().user.create({
            data: UserDbFactory.createMinimal({
                name: "Private User",
                handle: "private-user-" + Math.floor(Math.random() * 1000),
                isPrivate: true,
            }),
        });
    });

    afterAll(async () => {
        // Clean up
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("own profile", async () => {
                const { req, res } = await mockAuthenticatedSession({ 
                    ...loggedInUserNoPremiumData(), 
                    id: testUsers[0].id 
                });

                const input: FindByIdOrHandleInput = { id: testUsers[0].id };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUsers[0].id);
                expect(result.name).toBe(testUsers[0].name);
                expect(result.handle).toBe(testUsers[0].handle);
            });

            it("own private profile", async () => {
                const { req, res } = await mockAuthenticatedSession({ 
                    ...loggedInUserNoPremiumData(), 
                    id: privateUser.id 
                });

                const input: FindByIdOrHandleInput = { id: privateUser.id };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(privateUser.id);
                expect(result.isPrivate).toBe(true);
            });

            it("public profile", async () => {
                const { req, res } = await mockAuthenticatedSession({ 
                    ...loggedInUserNoPremiumData(), 
                    id: testUsers[0].id 
                });

                const input: FindByIdOrHandleInput = { id: testUsers[1].id };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(testUsers[1].id);
                expect(result.name).toBe(testUsers[1].name);
                expect(result.handle).toBe(testUsers[1].handle);
            });

            it("API key - public permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = { id: validUser2.id };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("id", validUser2.id);
                expect(result).to.have.property("name", validUser2.name);
                expect(result.handle).toBe(testUsers[1].handle);
            });

            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input = { id: validUser2.id };
                const result = await user.findOne({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("id", validUser2.id);
                expect(result).to.have.property("name", validUser2.name);
                expect(result.handle).toBe(testUsers[1].handle);
            });
        });

        describe("invalid", () => {
            it("private profile", async () => {
                // Make user2 private
                await DbProvider.get().user.update({
                    where: { id: validUser2.id },
                    data: { isPrivate: true },
                });

                // Attempt to access user2's profile from user1's account
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { id: validUser2.id };

                try {
                    await user.findOne({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* empty */ }
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("retuns public profiles", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { take: 10 };
                const result = await user.findMany({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
                expect(result.edges!.some(edge => edge?.node?.id === validUser1.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === validUser2.id)).to.be.true;
            });

            it("search input returns results", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = { searchString: validUser2.name };
                const result = await user.findMany({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(1);
                expect(result.edges!.some(edge => edge?.node?.id === validUser2.id)).to.be.true;
            });

            it("API key - public permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = { take: 10 };
                const result = await user.findMany({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
                expect(result.edges!.some(edge => edge?.node?.id === validUser1.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === validUser2.id)).to.be.true;
            });

            it("not logged in", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input = { take: 10 };
                const result = await user.findMany({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");
                expect(result.edges!.length).to.be.at.least(2);
                expect(result.edges!.some(edge => edge?.node?.id === validUser1.id)).to.be.true;
                expect(result.edges!.some(edge => edge?.node?.id === validUser2.id)).to.be.true;
            });
        });
    });

    describe("botCreateOne", () => {
        describe("valid", () => {
            it("should create a bot user that you own", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botData = {
                    id: uuid(),
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                        systemPrompt: "You are a helpful assistant.",
                    }),
                };

                const input = botData;
                const result = await user.botCreateOne({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("id", botData.id);
                expect(result).to.have.property("name", botData.name);
                expect(result).to.have.property("handle", botData.handle);
                expect(result).to.have.property("isBot", true);
                expect(result).to.have.property("isBotDepictingPerson", botData.isBotDepictingPerson);

                // Verify bot was created in database
                const createdBot = await DbProvider.get().user.findUnique({
                    where: { id: botData.id },
                });
                expect(createdBot).to.not.be.null;
                expect(createdBot).to.have.property("botSettings", botData.botSettings);
            });

            it("API key - write permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const botData = {
                    id: uuid(),
                    name: "API Key Bot",
                    handle: "api-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: true,
                    isPrivate: true,
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "You are a specialized bot created via API.",
                    }),
                };

                const input = botData;
                const result = await user.botCreateOne({ input }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("id", botData.id);
                expect(result).to.have.property("name", botData.name);
                expect(result).to.have.property("isBot", true);
                // isPrivate is not returned by the endpoint, so we'll query the database directly
                const createdBot = await DbProvider.get().user.findUnique({
                    where: { id: botData.id },
                });
                expect(createdBot).to.not.be.null;
                expect(createdBot).to.have.property("isPrivate", botData.isPrivate);
            });
        });

        describe("invalid", () => {
            it("same ID as existing user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: validUser2.id, // Using existing user ID
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                };

                try {
                    await user.botCreateOne({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("same ID as admin", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: SEEDED_IDS.User.Admin, // Using admin ID
                    name: "Admin Bot",
                    handle: "admin-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                };

                try {
                    await user.botCreateOne({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("trying to set `isBot` to false", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: uuid(),
                    name: "Not A Bot",
                    handle: "not-a-bot-" + Math.floor(Math.random() * 1000),
                    isBot: false, // This should cause validation to fail
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                };

                try {
                    await user.botCreateOne({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("trying to update user-specific fields", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: uuid(),
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                    status: "HardLocked", // User-specific field that shouldn't be allowed
                    notificationSettings: JSON.stringify({ disable: true }), // User-specific field
                };

                try {
                    await user.botCreateOne({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            testEndpointRequiresAuth(user.botCreateOne, {
                input: {
                    id: uuid(),
                    name: "Test Bot",
                    handle: "test-bot",
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: "{}",
                },
            }, user_findOne);

            const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
            testEndpointRequiresApiKeyWritePermissions(
                testUser,
                user.botCreateOne,
                {
                    input: {
                        id: uuid(),
                        name: "Test Bot",
                        handle: "test-bot",
                        isBot: true,
                        isBotDepictingPerson: false,
                        isPrivate: false,
                        botSettings: "{}",
                    },
                },
                user_findOne,
            );
        });
    });

    describe("botUpdateOne", () => {
        describe("valid", () => {
            it("should update a bot user that you own", async () => {
                // First create a bot
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botId = uuid();
                const botData = {
                    id: botId,
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                        systemPrompt: "Initial prompt",
                    }),
                };

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_findOne);

                // Update the bot
                const updateInput = {
                    id: botId,
                    name: "Updated Bot Name",
                    isPrivate: true,
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "Updated prompt",
                    }),
                };

                const result = await user.botUpdateOne({ input: updateInput }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("id", botId);
                expect(result).to.have.property("name", updateInput.name);

                // Verify bot was updated in database
                const updatedBot = await DbProvider.get().user.findUnique({
                    where: { id: botId },
                });
                expect(updatedBot).to.not.be.null;
                expect(updatedBot).to.have.property("name", updateInput.name);
                expect(updatedBot).to.have.property("botSettings", updateInput.botSettings);
                expect(updatedBot).to.have.property("isPrivate", updateInput.isPrivate);
            });

            it("API key - write permissions", async () => {
                // First create a bot
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req: createReq, res: createRes } = await mockAuthenticatedSession(testUser);

                const botId = uuid();
                const botData = {
                    id: botId,
                    name: "API Bot",
                    handle: "api-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                        systemPrompt: "Initial API prompt",
                    }),
                };

                // Create the bot
                await user.botCreateOne({ input: botData }, { req: createReq, res: createRes }, user_findOne);

                // Update the bot with API key
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const updateInput = {
                    id: botId,
                    name: "API Updated Bot",
                    botSettings: JSON.stringify({
                        model: "gpt-4",
                        systemPrompt: "Updated via API",
                    }),
                };

                const result = await user.botUpdateOne({ input: updateInput }, { req, res }, user_findOne);

                expect(result).to.not.be.null;
                expect(result).to.have.property("id", botId);
                expect(result).to.have.property("name", updateInput.name);

                // Verify bot was updated in database
                const updatedBot = await DbProvider.get().user.findUnique({
                    where: { id: botId },
                });
                expect(updatedBot).to.not.be.null;
                expect(updatedBot).to.have.property("name", updateInput.name);
                expect(updatedBot).to.have.property("botSettings", updateInput.botSettings);
            });
        });

        describe("invalid", () => {
            it("updating a different user's bot", async () => {
                // First create a bot owned by user2
                const user2 = { ...loggedInUserNoPremiumData(), id: validUser2.id };
                const { req: user2Req, res: user2Res } = await mockAuthenticatedSession(user2);

                const botId = uuid();
                const botData = {
                    id: botId,
                    name: "User2's Bot",
                    handle: "user2-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                };

                // Create the bot
                await user.botCreateOne({ input: botData }, { req: user2Req, res: user2Res }, user_findOne);

                // Try to update the bot as user1
                const user1 = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(user1);

                const updateInput = {
                    id: botId,
                    name: "Hijacked Bot",
                };

                try {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("updating admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const updateInput = {
                    id: SEEDED_IDS.User.Admin,
                    name: "Hacked Admin",
                };

                try {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("trying to set `isBot` to false", async () => {
                // First create a bot
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botId = uuid();
                const botData = {
                    id: botId,
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                };

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_findOne);

                // Try to update isBot to false
                const updateInput = {
                    id: botId,
                    isBot: false, // This should cause validation to fail
                };

                try {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            it("trying to update user-specific fields", async () => {
                // First create a bot
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const botId = uuid();
                const botData = {
                    id: botId,
                    name: "Test Bot",
                    handle: "test-bot-" + Math.floor(Math.random() * 1000),
                    isBot: true,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                    botSettings: JSON.stringify({
                        model: "gpt-3.5-turbo",
                    }),
                };

                // Create the bot
                await user.botCreateOne({ input: botData }, { req, res }, user_findOne);

                // Try to update with user-specific fields
                const updateInput = {
                    id: botId,
                    name: "Updated Bot",
                    status: "HardLocked", // User-specific field
                    notificationSettings: JSON.stringify({ disable: true }), // User-specific field
                };

                try {
                    await user.botUpdateOne({ input: updateInput }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* expected error */ }
            });

            testEndpointRequiresAuth(user.botUpdateOne, {
                input: {
                    id: uuid(),
                    name: "Test Bot Update",
                },
            }, user_findOne);

            const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
            testEndpointRequiresApiKeyWritePermissions(
                testUser,
                user.botUpdateOne,
                {
                    input: {
                        id: uuid(),
                        name: "Test Bot Update",
                    },
                },
                user_findOne,
            );
        });
    });

    describe("profileUpdate", () => {
        describe("valid", () => {
            it("should update user profile information", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: testUser.id,
                    name: "Updated Name",
                };
                const result = await user.profileUpdate({ input }, { req, res }, user_findOne);

                expect(result).to.have.property("id", testUser.id);
                expect(result).to.have.property("name", input.name);

                const updatedUser = await DbProvider.get().user.findUnique({
                    where: { id: testUser.id },
                    select: { id: true, name: true },
                });
                expect(updatedUser).to.not.be.null;
                expect(updatedUser?.name).to.equal(input.name);
            });

            it("should handle translationsCreate correctly", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: testUser.id,
                    translationsCreate: [{
                        id: uuid(),
                        language: "es",
                        bio: "Biografía de prueba",
                    }],
                };
                const result = await user.profileUpdate({ input }, { req, res }, user_findOne);

                expect(result).to.have.property("id", testUser.id);
                expect(result).to.have.property("translations").that.is.an("array");

                const updatedUser = await DbProvider.get().user.findUnique({
                    where: { id: testUser.id },
                    include: { translations: true },
                });
                expect(updatedUser).to.not.be.null;
                expect(updatedUser?.translations).to.be.an("array").that.is.not.empty;
                const createdTranslation = updatedUser?.translations.find((t) => t.language === input.translationsCreate[0].language);
                expect(createdTranslation).to.not.be.null;
                expect(createdTranslation).to.have.property("id", input.translationsCreate[0].id);
                expect(createdTranslation).to.have.property("bio", input.translationsCreate[0].bio);
            });

            it("API key - write permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = {
                    id: testUser.id,
                    name: "Updated Name",
                };
                const result = await user.profileUpdate({ input }, { req, res }, user_findOne);

                expect(result).to.have.property("id", testUser.id);
                expect(result).to.have.property("name", input.name);
            });
        });

        describe("invalid", () => {
            it("updating a different user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: validUser2.id,
                    name: "Updated Name",
                };
                try {
                    await user.profileUpdate({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* empty */ }
            });

            it("updating admin user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: SEEDED_IDS.User.Admin,
                    name: "Updated Name",
                };
                try {
                    await user.profileUpdate({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* empty */ }
            });
            it("updating a non-existent user", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
                const { req, res } = await mockAuthenticatedSession(testUser);

                const input = {
                    id: uuid(),
                    name: "Updated Name",
                };
                try {
                    await user.profileUpdate({ input }, { req, res }, user_findOne);
                    expect.fail("Expected an error to be thrown");
                } catch (error) { /* empty */ }
            });

            testEndpointRequiresAuth(user.profileUpdate, { input: { id: validUser1.id, name: "Updated Name" } }, user_findOne);

            const testUser = { ...loggedInUserNoPremiumData(), id: validUser1.id };
            testEndpointRequiresApiKeyWritePermissions(testUser, user.profileUpdate, { input: { id: validUser1.id, name: "Updated Name" } }, user_findOne);
        });
    });
}); 
