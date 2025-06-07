import { type ChatCreateInput, type ChatSearchInput, type ChatUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { describe, it, expect, beforeAll, afterAll, beforeEach, vi } from "vitest";
import { assertFindManyResultIds } from "../../__test/helpers.js";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { chat_createOne } from "../generated/chat_createOne.js";
import { chat_findMany } from "../generated/chat_findMany.js";
import { chat_findOne } from "../generated/chat_findOne.js";
import { chat_updateOne } from "../generated/chat_updateOne.js";
import { chat } from "./chat.js";

// Import database fixtures for seeding
import { ChatDbFactory, seedTestChat } from "../../__test/fixtures/chatFixtures.js";
import { UserDbFactory, seedTestUsers } from "../../__test/fixtures/userFixtures.js";

// Import validation fixtures for API input testing
import { chatTestDataFactory } from "@vrooli/shared/src/validation/models/__test__/fixtures/chatFixtures.js";

describe("EndpointsChat", () => {
    let testUsers: any[];
    let privateChat: any;
    let publicChat: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clear Redis and database tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create a private chat using database fixtures
        privateChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[0].id, testUsers[1].id],
            isPrivate: true,
            withMessages: true,
        });

        // Create a public chat using database fixtures
        publicChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[1].id],
            isPrivate: false,
            withInvites: true,
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
        it("returns chat by id for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[2].id 
            });
            const input: FindByIdInput = { id: privateChat.id };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(privateChat.id);
        });

        it("returns chat by id when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: publicChat.id };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(publicChat.id);
        });

        it("returns chat by id with API key public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            const input: FindByIdInput = { id: publicChat.id };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).not.toBeNull();
            expect(result.id).toEqual(publicChat.id);
        });
    });

    describe("findMany", () => {
        it("returns chats without filters for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            const input: ChatSearchInput = { take: 10 };
            const expectedIds = [privateChat.id, publicChat.id];
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).not.toBeNull();
            expect(result.edges).toBeInstanceOf(Array);
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("returns chats without filters for not authenticated user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ChatSearchInput = { take: 10 };
            const expectedIds = [privateChat.id, publicChat.id];
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).not.toBeNull();
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("returns chats without filters for API key public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            const input: ChatSearchInput = { take: 10 };
            const expectedIds = [privateChat.id, publicChat.id];
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).not.toBeNull();
            assertFindManyResultIds(expect, result, expectedIds);
        });
    });

    describe("createOne", () => {
        it("creates a chat for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            
            // Use validation fixtures for API input
            const input: ChatCreateInput = chatTestDataFactory.createMinimal({ 
                openToAnyoneWithInvite: true 
            });
            
            const result = await chat.createOne({ input }, { req, res }, chat_createOne);
            expect(result).not.toBeNull();
            expect(result.id).toBeDefined();
            expect(result.openToAnyoneWithInvite).toBe(true);
        });

        it("API key with write permissions can create chat", async () => {
            const permissions = mockWritePrivatePermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            
            // Use complete validation fixture for more comprehensive test
            const input: ChatCreateInput = chatTestDataFactory.createComplete({ 
                openToAnyoneWithInvite: false 
            });
            
            const result = await chat.createOne({ input }, { req, res }, chat_createOne);
            expect(result).not.toBeNull();
            expect(result.id).toBeDefined();
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            // Use validation fixture for consistency
            const input: ChatCreateInput = chatTestDataFactory.createMinimal({ 
                openToAnyoneWithInvite: false 
            });
            
            await expect(async () => {
                await chat.createOne({ input }, { req, res }, chat_createOne);
            }).rejects.toThrow();
        });
    });

    describe("updateOne", () => {
        it("updates chat openToAnyoneWithInvite flag for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ 
                ...loggedInUserNoPremiumData(), 
                id: testUsers[0].id 
            });
            
            // Use the actual chat ID from seeded data
            const input: ChatUpdateInput = { 
                id: privateChat.id, 
                openToAnyoneWithInvite: true 
            };
            
            const result = await chat.updateOne({ input }, { req, res }, chat_updateOne);
            expect(result).not.toBeNull();
            expect(result.openToAnyoneWithInvite).toBe(true);
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();
            
            const input: ChatUpdateInput = { 
                id: privateChat.id, 
                openToAnyoneWithInvite: false 
            };
            
            await expect(async () => {
                await chat.updateOne({ input }, { req, res }, chat_updateOne);
            }).rejects.toThrow();
        });
    });
});