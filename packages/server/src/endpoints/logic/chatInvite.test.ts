import { type ChatInviteCreateInput, type ChatInviteSearchInput, type ChatInviteUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPrivatePermissions, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { chatInvite_acceptOne } from "../generated/chatInvite_acceptOne.js";
import { chatInvite_createMany } from "../generated/chatInvite_createMany.js";
import { chatInvite_createOne } from "../generated/chatInvite_createOne.js";
import { chatInvite_declineOne } from "../generated/chatInvite_declineOne.js";
import { chatInvite_findMany } from "../generated/chatInvite_findMany.js";
import { chatInvite_findOne } from "../generated/chatInvite_findOne.js";
import { chatInvite_updateMany } from "../generated/chatInvite_updateMany.js";
import { chatInvite_updateOne } from "../generated/chatInvite_updateOne.js";
import { chatInvite } from "./chatInvite.js";

// Import database fixtures for seeding
import { seedTestChat } from "../../__test/fixtures/db/chatFixtures.js";
import { seedChatInvites } from "../../__test/fixtures/db/chatInviteFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing
import { chatInviteTestDataFactory } from "@vrooli/shared";

describe("EndpointsChatInvite", () => {
    let testUsers: any[];
    let chat1: any;
    let chat2: any;
    let invite1: any;
    let invite2: any;

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up tables used in tests
        try {
            const prisma = DbProvider.get();
            if (prisma) {
                testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
            }
        } catch (error) {
            // If database is not initialized, skip cleanup
        }

        // Seed chats using database fixtures
        chat1 = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[0].id],
            isPrivate: false,
        });

        chat2 = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[1].id],
            isPrivate: true,
        });

        // Seed chat invites using database fixtures
        const invites = await seedChatInvites(DbProvider.get(), {
            chatId: chat1.id,
            userIds: [testUsers[1].id],
            withCustomMessages: true,
        });
        invite1 = invites[0];

        const invites2 = await seedChatInvites(DbProvider.get(), {
            chatId: chat2.id,
            userIds: [testUsers[0].id], // User 0 is invited to chat2
            withCustomMessages: true,
        });
        invite2 = invites2[0];
    });

    afterAll(async () => {
        // Clean up
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Restore all mocks
        vi.restoreAllMocks();
    });

    describe("findOne", () => {
        describe("valid", () => {
            it("returns invite when user is the chat creator", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id  // User 0 is the creator of chat1
                });
                const input: FindByIdInput = { id: invite1.id }; // invite1 is for chat1
                const result = await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(invite1.id);
            });

            it("returns invite when user is the invited user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id  // User 0 is invited in invite2
                });
                const input: FindByIdInput = { id: invite2.id };
                const result = await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(invite2.id);
            });

            it("throws error for user with no visibility permission", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: FindByIdInput = { id: invite1.id };
                await expect(async () => {
                    await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                }).rejects.toThrow();
            });

            it("returns invite for API key with private read permissions", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);
                const input: FindByIdInput = { id: invite1.id };
                const result = await chatInvite.findOne({ input }, { req, res }, chatInvite_findOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(invite1.id);
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns only invites visible to authenticated user", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteSearchInput = { take: 10 };
                const result = await chatInvite.findMany({ input }, { req, res }, chatInvite_findMany);
                expect(result).not.toBeNull();
                expect(result).toHaveProperty("edges");
                expect(result.edges).toBeInstanceOf(Array);

                // User0 should see:
                // - invite1 (as creator of chat1)
                // - invite2 (as the invited user)
                expect(result.edges!.length).toBeGreaterThanOrEqual(1);
                expect(result.edges!.length).toBeLessThanOrEqual(2);

                const ids = result.edges!.map(e => e!.node!.id).sort();
                // Check if the visible invites are only those where user0 is involved
                ids.filter((id): id is string => id !== undefined).forEach(id => {
                    expect([invite1.id, invite2.id].includes(id)).toBe(true);
                });
            });

            it("fails for not authenticated user", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteSearchInput = { take: 10 };
                await expect(async () => {
                    await chatInvite.findMany({ input }, { req, res }, chatInvite_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("createOne", () => {
        describe("valid", () => {
            it("creates an invite for authenticated user", async () => {
                // User0 can create invites for chat1 (which they own)
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });

                // Use validation fixtures for API input
                const input: ChatInviteCreateInput = chatInviteTestDataFactory.createMinimal({
                    chatConnect: chat1.id,
                    userConnect: testUsers[2].id,
                    message: "New Invite",
                });

                const result = await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(input.id);
                expect(result.message).toEqual("New Invite");
            });

            it("API key with write permissions can create invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: testUsers[0].id });

                // Use validation fixtures for API input
                const input: ChatInviteCreateInput = chatInviteTestDataFactory.createMinimal({
                    chatConnect: chat1.id,
                    userConnect: testUsers[2].id,
                });

                const result = await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                expect(result).not.toBeNull();
                expect(result.id).toEqual(input.id);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteCreateInput = chatInviteTestDataFactory.createMinimal({
                    chatConnect: chat1.id,
                    userConnect: testUsers[1].id,
                });
                await expect(async () => {
                    await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                }).rejects.toThrow();
            });

            it("authenticated user cannot create invite for chat they don't own", async () => {
                // User2 tries to create invite for chat2 (owned by user1)
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[2].id });
                const input: ChatInviteCreateInput = chatInviteTestDataFactory.createMinimal({
                    chatConnect: chat2.id,
                    userConnect: testUsers[2].id,
                });
                await expect(async () => {
                    await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                }).rejects.toThrow();
            });

            it("API key without write permissions cannot create invite", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteCreateInput = chatInviteTestDataFactory.createMinimal({
                    chatConnect: chat2.id,
                    userConnect: testUsers[2].id,
                });
                await expect(async () => {
                    await chatInvite.createOne({ input }, { req, res }, chatInvite_createOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("createMany", () => {
        describe("valid", () => {
            it("creates multiple invites for chats user owns", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                // Only creates invite for chat1 which user0 owns
                const input: ChatInviteCreateInput[] = [
                    chatInviteTestDataFactory.createMinimal({
                        chatConnect: chat1.id,
                        userConnect: testUsers[2].id,
                        message: "Bulk 1",
                    }),
                ];
                const result = await chatInvite.createMany({ input }, { req, res }, chatInvite_createMany);
                expect(result).toHaveLength(1);
                expect(result[0].id).toEqual(input[0].id);
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot create many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteCreateInput[] = [
                    chatInviteTestDataFactory.createMinimal({
                        chatConnect: chat1.id,
                        userConnect: testUsers[2].id,
                    }),
                ];
                await expect(async () => {
                    await chatInvite.createMany({ input }, { req, res }, chatInvite_createMany);
                }).rejects.toThrow();
            });

            it("API key without write permissions cannot create many invites", async () => {
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteCreateInput[] = [
                    chatInviteTestDataFactory.createMinimal({
                        chatConnect: chat2.id,
                        userConnect: testUsers[2].id,
                    }),
                ];
                await expect(async () => {
                    await chatInvite.createMany({ input }, { req, res }, chatInvite_createMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("updates invite for chat owner", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                // User0 is the owner of chat1, and invite1 is for chat1
                const input: ChatInviteUpdateInput = { id: invite1.id, message: "Updated Msg" };
                const result = await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                expect(result).not.toBeNull();
                expect(result.message).toEqual("Updated Msg");
            });

            it("cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteUpdateInput = { id: invite2.id, message: "Updated By Recipient" };
                await expect(async () => {
                    await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                }).rejects.toThrow();
            });

            it("API key with write permissions can update an invite", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteUpdateInput = { id: invite1.id, message: "API Update" };
                const result = await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                expect(result).not.toBeNull();
                expect(result.message).toEqual("API Update");
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update invite", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteUpdateInput = { id: invite1.id, message: "Fail Update" };
                await expect(async () => {
                    await chatInvite.updateOne({ input }, { req, res }, chatInvite_updateOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateMany", () => {
        describe("valid", () => {
            it("updates multiple invites where user has visibility", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteUpdateInput[] = [
                    { id: invite1.id, message: "Bulk Update 1" }, // User0 is chat owner
                ];
                const result = await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                expect(result).toHaveLength(input.length);
                const messages = result.map(r => r.message).sort();
                expect(messages).toEqual(input.map(i => i.message).sort());
            });

            it("API key with write permissions can update many invites", async () => {
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, { ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteUpdateInput[] = [{ id: invite1.id, message: "API Bulk" }];
                const result = await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                expect(result).toHaveLength(input.length);
                const messages = result.map(r => r.message).sort();
                expect(messages).toEqual(input.map(i => i.message).sort());
            });
        });

        describe("invalid", () => {
            it("not logged in user cannot update many invites", async () => {
                const { req, res } = await mockLoggedOutSession();
                const input: ChatInviteUpdateInput[] = [{ id: invite1.id, message: "Fail Bulk" }];
                await expect(async () => {
                    await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                }).rejects.toThrow();
            });

            it("Cannot update invite as invite recipient", async () => {
                const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
                const input: ChatInviteUpdateInput[] = [{ id: invite2.id, message: "Fail Bulk" }];
                await expect(async () => {
                    await chatInvite.updateMany({ input }, { req, res }, chatInvite_updateMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("acceptOne", () => {
        it("invited user can accept invite", async () => {
            // user1 is invited to chat1 via invite1
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[1].id });
            const input: FindByIdInput = { id: invite1.id };

            const result = await chatInvite.acceptOne({ input }, { req, res }, chatInvite_acceptOne);

            expect(result.status).toEqual("Accepted");

            // Verify user1 is now a participant in chat1
            const participant = await DbProvider.get().chat_participants.findUnique({
                where: { chatId_userId: { chatId: chat1.id, userId: testUsers[1].id } },
            });
            expect(participant).not.toBeNull();
        });

        it("non-invited user cannot accept invite", async () => {
            // user2 tries to accept invite1 (for user1)
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[2].id });
            const input: FindByIdInput = { id: invite1.id };
            await expect(async () => {
                await chatInvite.acceptOne({ input }, { req, res }, chatInvite_acceptOne);
            }).rejects.toThrow();
        });

        it("cannot accept non-pending invite", async () => {
            // First, user1 accepts invite1
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[1].id });
            await chatInvite.acceptOne({ input: { id: invite1.id } }, { req: req1, res: res1 }, chatInvite_acceptOne);

            // Then, user1 tries to accept it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[1].id });
            await expect(async () => {
                await chatInvite.acceptOne({ input: { id: invite1.id } }, { req: req2, res: res2 }, chatInvite_acceptOne);
            }).rejects.toThrow();
        });
    });

    describe("declineOne", () => {
        it("invited user can decline invite", async () => {
            // user0 is invited to chat2 via invite2
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
            const input: FindByIdInput = { id: invite2.id };

            const result = await chatInvite.declineOne({ input }, { req, res }, chatInvite_declineOne);

            expect(result.status).toEqual("Declined");

            // Verify user0 is NOT a participant in chat2
            const participant = await DbProvider.get().chat_participants.findUnique({
                where: { chatId_userId: { chatId: chat2.id, userId: testUsers[0].id } },
            });
            expect(participant).toBeNull();
        });

        it("chat owner can't decline invite (they must delete it)", async () => {
            // user1 owns chat2, user0 is invited via invite2
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[1].id });
            const input: FindByIdInput = { id: invite2.id };

            await expect(async () => {
                await chatInvite.declineOne({ input }, { req, res }, chatInvite_declineOne);
            }).rejects.toThrow();
        });

        it("non-involved user cannot decline invite", async () => {
            // user2 tries to decline invite2 (for user0, chat owned by user1)
            const { req, res } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[2].id });
            const input: FindByIdInput = { id: invite2.id };
            await expect(async () => {
                await chatInvite.declineOne({ input }, { req, res }, chatInvite_declineOne);
            }).rejects.toThrow();
        });

        it("cannot decline non-pending invite", async () => {
            // First, user0 declines invite2
            const { req: req1, res: res1 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
            await chatInvite.declineOne({ input: { id: invite2.id } }, { req: req1, res: res1 }, chatInvite_declineOne);

            // Then, user0 tries to decline it again
            const { req: req2, res: res2 } = await mockAuthenticatedSession({ ...loggedInUserNoPremiumData(), id: testUsers[0].id });
            await expect(async () => {
                await chatInvite.declineOne({ input: { id: invite2.id } }, { req: req2, res: res2 }, chatInvite_declineOne);
            }).rejects.toThrow();
        });
    });
});
