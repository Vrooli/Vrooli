import { assertFindManyResultIds } from "../../__test/helpers.js";
// Tests for the ChatParticipant endpoint (findOne, findMany, updateOne)
import { type ChatParticipantSearchInput, type ChatParticipantUpdateInput, type FindByIdInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { defaultPublicUserData, loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { chatParticipant_findMany } from "../generated/chatParticipant_findMany.js";
import { chatParticipant_findOne } from "../generated/chatParticipant_findOne.js";
import { chatParticipant_updateOne } from "../generated/chatParticipant_updateOne.js";
import { chatParticipant } from "./chatParticipant.js";

// Test users and chats
const user1Id = uuid();
const user2Id = uuid();
const chat1Id = uuid();
const chat2Id = uuid();

// Entries for chat participants
let cp1: any;
let cp2: any;
let cp3: any;

describe("EndpointsChatParticipant", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // suppress logger output
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // clear Redis and tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        // Create two users
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user1Id,
                name: "Test User 1",
            },
        });
        await DbProvider.get().user.create({
            data: {
                ...defaultPublicUserData(),
                id: user2Id,
                name: "Test User 2",
            },
        });

        // Create two chats with translations
        await DbProvider.get().chat.create({
            data: {
                id: chat1Id,
                isPrivate: false,
                openToAnyoneWithInvite: true,
                creatorId: user1Id,
                translations: { create: { id: uuid(), language: "en", name: "Chat 1", description: "First chat" } },
            },
        });
        await DbProvider.get().chat.create({
            data: {
                id: chat2Id,
                isPrivate: true,
                openToAnyoneWithInvite: false,
                creatorId: user2Id,
                translations: { create: { id: uuid(), language: "en", name: "Chat 2", description: "Second chat" } },
            },
        });

        // Seed participant records
        cp1 = await DbProvider.get().chat_participants.create({
            data: {
                id: uuid(),
                chat: { connect: { id: chat1Id } },
                user: { connect: { id: user1Id } },
            },
        });
        cp2 = await DbProvider.get().chat_participants.create({
            data: {
                id: uuid(),
                chat: { connect: { id: chat1Id } },
                user: { connect: { id: user2Id } },
            },
        });
        cp3 = await DbProvider.get().chat_participants.create({
            data: {
                id: uuid(),
                chat: { connect: { id: chat2Id } },
                user: { connect: { id: user2Id } },
            },
        });
    });

    after(async () => {
        // cleanup and restore logger
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns own participant record", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: cp1.id };
                const result = await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(cp1.id);
            });

            it("API key with public read can find no participants", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
                const input: FindByIdInput = { id: cp3.id };
                try {
                    await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // expected
                }
            });

            it("logged out user can find no participants", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: cp2.id };
                try {
                    await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // expected
                }
            });
        });
        describe("invalid", () => {
            it("does not return participant from a private chat when not involved", async () => {
                // user1 should not see cp3 (chat2-user2)
                const user = { ...loggedInUserNoPremiumData(), id: user1Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: FindByIdInput = { id: cp3.id };
                try {
                    await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // expected
                }
            });
        });
    });

    describe("findMany", () => {
        it("returns only own participants for authenticated user", async () => {
            const user = { ...loggedInUserNoPremiumData(), id: user2Id };
            const { req, res } = await mockAuthenticatedSession(user);
            const input: ChatParticipantSearchInput = { take: 10 };
            const expectedIds = [
                cp2.id,
                cp3.id,
            ];
            const result = await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);
            expect(result).to.not.be.null;
            assertFindManyResultIds(expect, result, expectedIds);
        });

        it("API key with public read returns no participants", async () => {
            const permissions = mockReadPublicPermissions();
            const apiToken = ApiKeyEncryptionService.generateSiteKey();
            const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
            const input: ChatParticipantSearchInput = { take: 10 };
            try {
                await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);
                expect.fail("Expected an error");
            } catch (error) {
                // expected
            }
        });

        it("logged out user can find no participants", async () => {
            const { req, res } = await mockLoggedOutSession();
            const input: ChatParticipantSearchInput = { take: 10 };
            try {
                await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);
                expect.fail("Expected an error");
            } catch (error) {
                // expected
            }
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates own participant record", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user2Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: ChatParticipantUpdateInput = { id: cp2.id };
                const result = await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(cp2.id);
            });

            it("API key with write permissions can update only own participant", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, loggedInUserNoPremiumData());
                const input: ChatParticipantUpdateInput = { id: cp1.id };
                const result = await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                expect(result.id).to.equal(cp1.id);

                // user2 should not be able to update cp1
                const user2 = { ...loggedInUserNoPremiumData(), id: user2Id };
                const { req: req2, res: res2 } = await mockAuthenticatedSession(user2);
                const input2: ChatParticipantUpdateInput = { id: cp1.id };
                try {
                    await chatParticipant.updateOne({ input: input2 }, { req: req2, res: res2 }, chatParticipant_updateOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // expected
                }
            });
        });
        describe("invalid", () => {
            it("cannot update another user's participant record", async () => {
                const user = { ...loggedInUserNoPremiumData(), id: user2Id };
                const { req, res } = await mockAuthenticatedSession(user);
                const input: ChatParticipantUpdateInput = { id: cp1.id };
                try {
                    await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                    expect.fail("Expected an error");
                } catch (error) {
                    // expected
                }
            });
        });
    });
}); 
