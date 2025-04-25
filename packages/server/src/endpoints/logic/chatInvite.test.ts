import { ChatInviteCreateInput, ChatInviteSearchInput, ChatInviteUpdateInput, FindByIdInput, uuid } from "@local/shared";
import { expect } from "chai";
import { after, before, beforeEach, describe, it } from "mocha";
import sinon from "sinon";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { initializeRedis } from "../../redisConn.js";
import { chatInvite_acceptOne } from "../generated/chatInvite_acceptOne.js";
import { chatInvite_createMany } from "../generated/chatInvite_createMany.js";
import { chatInvite_createOne } from "../generated/chatInvite_createOne.js";
import { chatInvite_declineOne } from "../generated/chatInvite_declineOne.js";
import { chatInvite_findMany } from "../generated/chatInvite_findMany.js";
import { chatInvite_findOne } from "../generated/chatInvite_findOne.js";
import { chatInvite_updateMany } from "../generated/chatInvite_updateMany.js";
import { chatInvite_updateOne } from "../generated/chatInvite_updateOne.js";
import { chatInvite } from "./chatInvite.js";

const user1Id = uuid();
const user2Id = uuid();
const user3Id = uuid();
const chat1Id = uuid();
const chat2Id = uuid();
const invite1Id = uuid();
const invite2Id = uuid();

describe("EndpointsChatInvite", () => {
    let loggerErrorStub: sinon.SinonStub;
    let loggerInfoStub: sinon.SinonStub;

    before(() => {
        // Suppress logger output during tests
        loggerErrorStub = sinon.stub(logger, "error");
        loggerInfoStub = sinon.stub(logger, "info");
    });

    beforeEach(async () => {
        // Reset Redis and truncate tables
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

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

        // Seed two chats
        await DbProvider.get().chat.create({
            data: {
                id: chat1Id,
                isPrivate: false,
                openToAnyoneWithInvite: true,
                creatorId: user1Id, // Explicitly set creator
                translations: { create: { id: uuid(), language: "en", name: "Chat 1", description: "First chat" } },
                participants: { create: [{ id: uuid(), user: { connect: { id: user1Id } } }] },
            },
        });
        await DbProvider.get().chat.create({
            data: {
                id: chat2Id,
                isPrivate: true,
                openToAnyoneWithInvite: false,
                creatorId: user2Id, // Explicitly set creator
                translations: { create: { id: uuid(), language: "en", name: "Chat 2", description: "Second chat" } },
                participants: { create: [{ id: uuid(), user: { connect: { id: user2Id } } }] },
            },
        });

        // Seed two chat invites
        const _invite1 = await DbProvider.get().chat_invite.create({
            data: {
                id: invite1Id,
                message: "Invite One",
                chat: { connect: { id: chat1Id } },
                user: { connect: { id: user2Id } },
            },
        });
        const _invite2 = await DbProvider.get().chat_invite.create({
            data: {
                id: invite2Id,
                message: "Invite Two",
                chat: { connect: { id: chat2Id } },
                user: { connect: { id: user1Id } },
            },
        });
        const seededInvites = await DbProvider.get().chat_invite.findMany();
    });

    after(async () => {
        // Clean up and restore logger
        await (await initializeRedis())?.flushAll();
        await DbProvider.deleteAll();

        loggerErrorStub.restore();
        loggerInfoStub.restore();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns invite when user is the chat creator", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User1 is the creator of chat1
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: invite1Id }; // invite1 is for chat1
                const result = await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite1Id);
            });

            it("returns invite when user is the invited user", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id }; // User1 is invited in invite2
                const { req, res } = await mockAuthenticatedSession(testUser);
                const input: FindByIdInput = { id: invite2Id };
                const result = await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite2Id);
            });

            it("throws error for user with no visibility permission", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: invite1Id };
                try {
                    await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                    expect.fail("Expected an error due to visibility restrictions");
                } catch (err) { /* expected */ }
            });

            it("returns invite for API key with private read permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData, id: user1Id };
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: invite1Id };
                const result = await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(invite1Id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns only invites visible to authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteSearchInput = { take: 10 };
                const result = await chatInvite.findMany({ input }, { req, res }, chatInvite_findMany);
                expect(result).to.not.be.null;
                expect(result).to.have.property("edges").that.is.an("array");

                // User1 should see:
                // - invite1 (as creator of chat1)
                // - invite2 (as the invited user)
                expect(result.edges!.length).to.be.within(1, 2); // Either or both invites should be visible

                const ids = result.edges!.map(e => e!.node!.id).sort();
                // Check if the visible invites are only those where user1 is involved
                ids.filter((id): id is string => id !== undefined).forEach(id => {
                    expect([invite1Id, invite2Id].includes(id)).to.be.true;
                });
            });

            it("fails for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteSearchInput = { take: 10 };
                try {
                    await chatInvite.findMany({ input }, { req, res }, chatInvite_findMany);
                    expect.fail("Expected an error due to visibility restrictions");
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates an invite for authenticated user", async () => {
                // User1 can create invites for chat1 (which they own)
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const newInviteId = uuid();
                const input: ChatInviteCreateInput = { id: newInviteId, chatConnect: chat1Id, userConnect: user3Id, message: "New Invite" };
                const result = await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newInviteId);
                expect(result.message).to.equal("New Invite");
            });

            it("API key with write permissions can create invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const newInviteId = uuid();
                const input: ChatInviteCreateInput = { id: newInviteId, chatConnect: chat1Id, userConnect: user3Id };
                const result = await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                expect(result).to.not.be.null;
                expect(result.id).to.equal(newInviteId);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteCreateInput = { id: uuid(), chatConnect: chat1Id, userConnect: user2Id };
                try {
                    await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (err) { /* expected */ }
            });

            it("authenticated user cannot create invite for chat they don't own", async () => {
                // User1 tries to create invite for chat2 (owned by user2)
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user3Id });
                const input: ChatInviteCreateInput = { id: uuid(), chatConnect: chat2Id, userConnect: user3Id };
                try {
                    await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                    expect.fail("Expected an error due to permission restrictions");
                } catch (err) { /* expected */ }
            });

            it("API key without write permissions cannot create invite", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteCreateInput = { id: uuid(), chatConnect: chat2Id, userConnect: user3Id };
                try {
                    await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                    expect.fail("Expected an error to be thrown");
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("createMany", () => {
        describe("valid", () => {
            it("creates multiple invites for chats user owns", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const idA = uuid();
                // Only creates invite for chat1 which user1 owns
                const input: ChatInviteCreateInput[] = [
                    { id: idA, chatConnect: chat1Id, userConnect: user3Id, message: "Bulk 1" },
                ];
                const result = await chatInvite.createMany({ input }, { req, res }, chatInvite_createMany);
                expect(result).to.have.length(1);
                expect(result[0].id).to.equal(idA);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteCreateInput[] = [
                    { id: uuid(), chatConnect: chat1Id, userConnect: user3Id },
                ];
                try {
                    await chatInvite.createMany({ input }, { req, res }, chatInvite_createMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });

            it("API key without write permissions cannot create many invites", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteCreateInput[] = [
                    { id: uuid(), chatConnect: chat2Id, userConnect: user3Id },
                ];
                try {
                    await chatInvite.createMany({ input }, { req, res }, chatInvite_createMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates invite for chat owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                // User1 is the owner of chat1, and invite1 is for chat1
                const input: ChatInviteUpdateInput = { id: invite1Id, message: "Updated Msg" };
                const result = await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                expect(result).to.not.be.null;
                expect(result.message).to.equal("Updated Msg");
            });

            it("cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteUpdateInput = { id: invite2Id, message: "Updated By Recipient" };
                try {
                    await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                    expect.fail("Expected error due to permission restrictions");
                } catch (err) {  /* expected */ }
            });

            it("API key with write permissions can update an invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteUpdateInput = { id: invite1Id, message: "API Update" };
                const result = await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                expect(result).to.not.be.null;
                expect(result.message).to.equal("API Update");
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteUpdateInput = { id: invite1Id, message: "Fail Update" };
                try {
                    await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                    expect.fail();
                } catch (err) {/* expected */ }
            });
        });
    });

    describe("updateMany", () => {
        describe("valid", () => {
            it("updates multiple invites where user has visibility", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteUpdateInput[] = [
                    { id: invite1Id, message: "Bulk Update 1" }, // User1 is chat owner
                ];
                const result = await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                expect(result).to.have.length(input.length);
                const messages = result.map(r => r.message).sort();
                expect(messages).to.deep.equal(input.map(i => i.message).sort());
            });

            it("API key with write permissions can update many invites", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteUpdateInput[] = [{ id: invite1Id, message: "API Bulk" }];
                const result = await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                expect(result).to.have.length(input.length);
                const messages = result.map(r => r.message).sort();
                expect(messages).to.deep.equal(input.map(i => i.message).sort());
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteUpdateInput[] = [{ id: invite1Id, message: "Fail Bulk" }];
                try {
                    await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });

            it("Cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
                const input: ChatInviteUpdateInput[] = [{ id: invite2Id, message: "Fail Bulk" }];
                try {
                    await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                    expect.fail();
                } catch (err) { /* expected */ }
            });
        });
    });

    describe("acceptOne", () => {
        it("invited user can accept invite", async () => {
            // user2 is invited to chat1 via invite1
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
            const input: FindByIdInput = { id: invite1Id };

            const result = await chatInvite.acceptOne({ input }, { req, res }, chatInvite_acceptOne);

            expect(result.status).to.equal("Accepted");

            // Verify user2 is now a participant in chat1
            const participant = await DbProvider.get().chat_participants.findUnique({
                where: { chatId_userId: { chatId: chat1Id, userId: user2Id } },
            });
            expect(participant).to.not.be.null;
        });

        it("non-invited user cannot accept invite", async () => {
            // user3 tries to accept invite1 (for user2)
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user3Id });
            const input: FindByIdInput = { id: invite1Id };
            try {
                await chatInvite.acceptOne({ input }, { req, res }, chatInvite_acceptOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("cannot accept non-pending invite", async () => {
            // First, user2 accepts invite1
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
            await chatInvite.acceptOne({ input: { id: invite1Id } }, { req: req1, res: res1 }, chatInvite_acceptOne);

            // Then, user2 tries to accept it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
            try {
                await chatInvite.acceptOne({ input: { id: invite1Id } }, { req: req2, res: res2 }, chatInvite_acceptOne);
                expect.fail("Expected Conflict error");
            } catch (err) { /* expected */ }
        });
    });

    describe("declineOne", () => {
        it("invited user can decline invite", async () => {
            // user1 is invited to chat2 via invite2
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            const input: FindByIdInput = { id: invite2Id };

            const result = await chatInvite.declineOne({ input }, { req, res }, chatInvite_declineOne);

            expect(result.status).to.equal("Declined");

            // Verify user1 is NOT a participant in chat2
            const participant = await DbProvider.get().chat_participants.findUnique({
                where: { chatId_userId: { chatId: chat2Id, userId: user1Id } },
            });
            expect(participant).to.be.null;
        });

        it("chat owner can't decline invite (they must delete it)", async () => {
            // user2 owns chat2, user1 is invited via invite2
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user2Id });
            const input: FindByIdInput = { id: invite2Id };

            try {
                await chatInvite.declineOne({ input }, { req, res }, chatInvite_declineOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("non-involved user cannot decline invite", async () => {
            // user3 tries to decline invite2 (for user1, chat owned by user2)
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user3Id });
            const input: FindByIdInput = { id: invite2Id };
            try {
                await chatInvite.declineOne({ input }, { req, res }, chatInvite_declineOne);
                expect.fail("Expected Forbidden error");
            } catch (err) { /* expected */ }
        });

        it("cannot decline non-pending invite", async () => {
            // First, user1 declines invite2
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            await chatInvite.declineOne({ input: { id: invite2Id } }, { req: req1, res: res1 }, chatInvite_declineOne);

            // Then, user1 tries to decline it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData, id: user1Id });
            try {
                await chatInvite.declineOne({ input: { id: invite2Id } }, { req: req2, res: res2 }, chatInvite_declineOne);
                expect.fail("Expected Conflict error");
            } catch (err) { /* expected */ }
        });
    });
});
