import { type ChatParticipantSearchInput, type ChatParticipantUpdateInput, type FindByIdInput } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions, mockWritePrivatePermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";
import { chatParticipant_findMany } from "../generated/chatParticipant_findMany.js";
import { chatParticipant_findOne } from "../generated/chatParticipant_findOne.js";
import { chatParticipant_updateOne } from "../generated/chatParticipant_updateOne.js";
import { chatParticipant } from "./chatParticipant.js";

// Import database fixtures for seeding
import { seedTestChat } from "../../__test/fixtures/db/chatFixtures.js";
import { seedChatParticipants } from "../../__test/fixtures/db/chatParticipantFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";

// Import validation fixtures for API input testing

describe("EndpointsChatParticipant", () => {
    let testUsers: any[];
    let chat1: any;
    let chat2: any;
    let publicChat: any;
    let participants: any = {};

    beforeAll(() => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Reset Redis and database tables
        await CacheService.get().flushAll();
        await DbProvider.deleteAll();

        // Seed test users using database fixtures
        testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create test chats using database fixtures
        chat1 = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[0].id,
            isPrivate: false,
            openToAnyoneWithInvite: false,
            participantIds: [], // We'll add participants separately
        });

        chat2 = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[1].id,
            isPrivate: false,
            openToAnyoneWithInvite: false,
            participantIds: [], // We'll add participants separately
        });

        publicChat = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[0].id,
            isPrivate: false,
            openToAnyoneWithInvite: true,
            participantIds: [], // We'll add participants separately
        });

        // Seed chat participants using database fixtures
        const chat1Participants = await seedChatParticipants(DbProvider.get(), {
            chatId: chat1.id,
            participants: [
                { userId: testUsers[0].id, role: "admin" }, // User 0 is admin of chat1
                { userId: testUsers[1].id, role: "member" }, // User 1 is member of chat1
            ],
        });
        participants.cp1 = chat1Participants[0]; // User 0 in chat1 (admin)
        participants.cp2 = chat1Participants[1]; // User 1 in chat1 (member)

        const chat2Participants = await seedChatParticipants(DbProvider.get(), {
            chatId: chat2.id,
            participants: [
                { userId: testUsers[1].id, role: "admin" }, // User 1 is admin of chat2
            ],
        });
        participants.cp3 = chat2Participants[0]; // User 1 in chat2 (admin)

        // Add participants to public chat
        const publicChatParticipants = await seedChatParticipants(DbProvider.get(), {
            chatId: publicChat.id,
            participants: [
                { userId: testUsers[0].id, role: "admin" }, // User 0 is admin of publicChat
            ],
        });
        participants.cpPublic = publicChatParticipants[0];
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
            it("participant can view their own participant record", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: FindByIdInput = { id: participants.cp1.id };
                const result = await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(participants.cp1.id);
                expect(result.userId).toBe(testUsers[0].id);
                expect(result.chatId).toBe(chat1.id);
            });

            it("admin can view other participants in their chat", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // Admin of chat1
                });

                const input: FindByIdInput = { id: participants.cp2.id }; // User 1's participation in chat1
                const result = await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(participants.cp2.id);
                expect(result.userId).toBe(testUsers[1].id);
                expect(result.chatId).toBe(chat1.id);
            });

            it("returns participant record for public chat with API key", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: FindByIdInput = { id: participants.cpPublic.id };
                const result = await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(participants.cpPublic.id);
            });
        });

        describe("invalid", () => {
            it("non-participant cannot view participant record", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not in any chat
                });

                const input: FindByIdInput = { id: participants.cp1.id };

                await expect(async () => {
                    await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot view participant record", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = { id: participants.cp1.id };

                await expect(async () => {
                    await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                }).rejects.toThrow();
            });

            it("member cannot view participant from different chat", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 is in chat1 but not chat2
                });

                const input: FindByIdInput = { id: participants.cp3.id }; // User 1's participation in chat2

                await expect(async () => {
                    await chatParticipant.findOne({ input }, { req, res }, chatParticipant_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns participants for chat member", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // User 0 is in chat1
                });

                const input: ChatParticipantSearchInput = { chatId: chat1.id, take: 10 };
                const result = await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(2); // Both participants in chat1

                const participantIds = result.edges.map(e => e?.node?.id);
                expect(participantIds).toContain(participants.cp1.id);
                expect(participantIds).toContain(participants.cp2.id);
            });

            it("returns participants filtered by user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: ChatParticipantSearchInput = {
                    chatId: chat1.id,
                    userId: testUsers[1].id,
                    take: 10
                };
                const result = await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only user 1's participation
                expect(result.edges[0]?.node?.id).toBe(participants.cp2.id);
                expect(result.edges[0]?.node?.userId).toBe(testUsers[1].id);
            });

            it("returns participants for public chat with API key", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ChatParticipantSearchInput = { chatId: publicChat.id, take: 10 };
                const result = await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThanOrEqual(1);
            });
        });

        describe("invalid", () => {
            it("non-participant cannot view participants", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not in chat1
                });

                const input: ChatParticipantSearchInput = { chatId: chat1.id, take: 10 };

                await expect(async () => {
                    await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);
                }).rejects.toThrow();
            });

            it("logged out user cannot view participants", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ChatParticipantSearchInput = { chatId: chat1.id, take: 10 };

                await expect(async () => {
                    await chatParticipant.findMany({ input }, { req, res }, chatParticipant_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("updateOne", () => {
        describe("valid", () => {
            it("admin can update participant permissions", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // Admin of chat1
                });

                const input: ChatParticipantUpdateInput = {
                    id: participants.cp2.id, // Update user 1's permissions
                    permissions: JSON.stringify({
                        canDelete: true,
                        canInvite: true,
                        canKick: false,
                        canUpdate: false,
                    }),
                };

                const result = await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(participants.cp2.id);
                expect(result.permissions).toBe(input.permissions);
            });

            it("API key with write permissions can update participant", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockWritePrivatePermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: ChatParticipantUpdateInput = {
                    id: participants.cp2.id,
                    permissions: JSON.stringify({
                        canDelete: false,
                        canInvite: false,
                        canKick: false,
                        canUpdate: true,
                    }),
                };

                const result = await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(participants.cp2.id);
                expect(result.permissions).toBe(input.permissions);
            });
        });

        describe("invalid", () => {
            it("member cannot update other participant permissions", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[1].id, // Member of chat1, not admin
                });

                const input: ChatParticipantUpdateInput = {
                    id: participants.cp1.id, // Try to update admin's permissions
                    permissions: JSON.stringify({
                        canDelete: false,
                        canInvite: false,
                        canKick: false,
                        canUpdate: false,
                    }),
                };

                await expect(async () => {
                    await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                }).rejects.toThrow();
            });

            it("non-participant cannot update participant", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not in chat1
                });

                const input: ChatParticipantUpdateInput = {
                    id: participants.cp1.id,
                    permissions: JSON.stringify({
                        canDelete: false,
                        canInvite: false,
                        canKick: false,
                        canUpdate: false,
                    }),
                };

                await expect(async () => {
                    await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot update participant", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ChatParticipantUpdateInput = {
                    id: participants.cp1.id,
                    permissions: JSON.stringify({
                        canDelete: false,
                        canInvite: false,
                        canKick: false,
                        canUpdate: false,
                    }),
                };

                await expect(async () => {
                    await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                }).rejects.toThrow();
            });

            it("admin cannot update participant from different chat", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id, // Admin of chat1 but not chat2
                });

                const input: ChatParticipantUpdateInput = {
                    id: participants.cp3.id, // User 1's participation in chat2
                    permissions: JSON.stringify({
                        canDelete: false,
                        canInvite: false,
                        canKick: false,
                        canUpdate: false,
                    }),
                };

                await expect(async () => {
                    await chatParticipant.updateOne({ input }, { req, res }, chatParticipant_updateOne);
                }).rejects.toThrow();
            });
        });
    });
});