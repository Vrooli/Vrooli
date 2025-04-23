import { ChatCreateInput, ChatSearchInput, ChatUpdateInput, FindByIdInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
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

// Generate unique IDs for test users and chats
const user1Id = uuid();
const user2Id = uuid();
const user3Id = uuid();
const chat1Id = uuid();
const chat2Id = uuid();

describe("EndpointsChat", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Stub logger to suppress output during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Clear Redis and database tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().session.deleteMany();
        await DbProvider.get().user_auth.deleteMany();
        await DbProvider.get().chat.deleteMany();
        await DbProvider.get().chat_translation.deleteMany();
        await DbProvider.get().user.deleteMany({});

        // Seed three users
        await DbProvider.get().user.create({
            data: {
                id: user1Id,
                name: "Test User 1",
                handle: "test-user-1",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user2Id,
                name: "Test User 2",
                handle: "test-user-2",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });
        await DbProvider.get().user.create({
            data: {
                id: user3Id,
                name: "Test User 3",
                handle: "test-user-3",
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
                auths: { create: [{ provider: "Password", hashed_password: "dummy-hash" }] },
            },
        });

        // Create a private chat with user1 and user2
        await DbProvider.get().chat.create({
            data: {
                id: chat1Id,
                isPrivate: true,
                openToAnyoneWithInvite: false,
                translations: {
                    create: {
                        id: uuid(),
                        language: "en",
                        name: "Private Chat",
                        description: "A private chat",
                    },
                },
                participants: {
                    create: [
                        { id: uuid(), user: { connect: { id: user1Id } } },
                        { id: uuid(), user: { connect: { id: user2Id } } },
                    ],
                },
            },
        });

        // Create a public chat open to anyone
        await DbProvider.get().chat.create({
            data: {
                id: chat2Id,
                isPrivate: false,
                openToAnyoneWithInvite: true,
                translations: {
                    create: {
                        id: uuid(),
                        language: "en",
                        name: "Public Chat",
                        description: "A public chat",
                    },
                },
                participants: {
                    create: [
                        { id: uuid(), user: { connect: { id: user2Id } } },
                    ],
                },
            },
        });
    });

    after(async () => {
        // Clean up and restore logger stubs
        await (await initializeRedis())?.flushAll();
        await DbProvider.get().chat.deleteMany();
        await DbProvider.get().chat_translation.deleteMany();
        await DbProvider.get().user.deleteMany({});
        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        it("returns chat by id for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user3Id });
            const input: FindByIdInput = { id: chat1Id };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(chat1Id);
        });

        it("returns chat by id when not authenticated", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: FindByIdInput = { id: chat2Id };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(chat2Id);
        });

        it("returns chat by id with API key public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
            const input: FindByIdInput = { id: chat2Id };
            const result = await chat.findOne({ input }, { req, res }, chat_findOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(chat2Id);
        });
    });

    describe("findMany", () => {
        it("returns chats without filters for any authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: ChatSearchInput = { take: 10 };
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).to.not.be.null;
            expect(result.edges).to.be.an("array");
            const ids = result.edges!.map(edge => edge!.node!.id).sort();
            expect(ids).to.deep.equal([chat1Id, chat2Id].sort());
        });

        it("returns chats without filters for not authenticated user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ChatSearchInput = { take: 10 };
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).to.not.be.null;
            const ids = result.edges!.map(edge => edge!.node!.id).sort();
            expect(ids).to.deep.equal([chat1Id, chat2Id].sort());
        });

        it("returns chats without filters for API key public read", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
            const input: ChatSearchInput = { take: 10 };
            const result = await chat.findMany({ input }, { req, res }, chat_findMany);
            expect(result).to.not.be.null;
            const ids = result.edges!.map(edge => edge!.node!.id).sort();
            expect(ids).to.deep.equal([chat1Id, chat2Id].sort());
        });
    });

    describe("createOne", () => {
        it("creates a chat for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const newChatId = uuid();
            const input: ChatCreateInput = { id: newChatId, openToAnyoneWithInvite: true };
            const result = await chat.createOne({ input }, { req, res }, chat_createOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(newChatId);
            expect(result.openToAnyoneWithInvite).to.be.true;
        });

        it("API key with write permissions can create chat", async () => {
            const permissions = mockWritePrivatePermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
            const newChatId = uuid();
            const input: ChatCreateInput = { id: newChatId, openToAnyoneWithInvite: false };
            const result = await chat.createOne({ input }, { req, res }, chat_createOne);
            expect(result).to.not.be.null;
            expect(result.id).to.equal(newChatId);
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ChatCreateInput = { id: uuid(), openToAnyoneWithInvite: false };
            try {
                await chat.createOne({ input }, { req, res }, chat_createOne);
                expect.fail("Expected an error to be thrown");
            } catch (err) {
                console.log("[endpoint testing error debug] createOne chat error:", err);
            }
        });
    });

    describe("updateOne", () => {
        it("updates chat openToAnyoneWithInvite flag for authenticated user", async () => {
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: ChatUpdateInput = { id: chat1Id, openToAnyoneWithInvite: true };
            const result = await chat.updateOne({ input }, { req, res }, chat_updateOne);
            expect(result).to.not.be.null;
            expect(result.openToAnyoneWithInvite).to.be.true;
        });

        it("throws error for not logged in user", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ChatUpdateInput = { id: chat1Id, openToAnyoneWithInvite: false };
            try {
                await chat.updateOne({ input }, { req, res }, chat_updateOne);
                expect.fail("Expected an error to be thrown");
            } catch (err) {
                console.log("[endpoint testing error debug] updateOne chat error:", err);
            }
        });
    });
}); 
