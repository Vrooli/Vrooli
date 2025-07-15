import { type ChatMessageSearchTreeInput, type ChatMessageSearchTreeResult, type FindByIdInput, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { loggedInUserNoPremiumData, mockApiSession, mockAuthenticatedSession, mockLoggedOutSession, mockReadPublicPermissions } from "../../__test/session.js";
import { ApiKeyEncryptionService } from "../../auth/apiKeyEncryption.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { chatMessage_findMany } from "../generated/chatMessage_findMany.js";
import { chatMessage_findOne } from "../generated/chatMessage_findOne.js";
import { chatMessage_findTree } from "../generated/chatMessage_findTree.js";
import { chatMessage } from "./chatMessage.js";
// Import database fixtures for seeding
import { seedTestChat } from "../../__test/fixtures/db/chatFixtures.js";
import { seedChatMessages, seedConversationTree } from "../../__test/fixtures/db/chatMessageFixtures.js";
import { seedTestUsers } from "../../__test/fixtures/db/userFixtures.js";
import { cleanupGroups } from "../../__test/helpers/testCleanupHelpers.js";
import { validateCleanup } from "../../__test/helpers/testValidation.js";

// Import validation fixtures for API input testing

describe("EndpointsChatMessage", () => {
    beforeAll(async () => {
        // Use Vitest spies to suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
    });

    beforeEach(async () => {
        // Clean up using dependency-ordered cleanup helpers
        await cleanupGroups.chat(DbProvider.get());
    });

    afterEach(async () => {
        // Validate cleanup to detect any missed records
        const orphans = await validateCleanup(DbProvider.get(), {
            tables: ["chat","chat_message","chat_participants","chat_invite","user"],
            logOrphans: true,
        });
        if (orphans.length > 0) {
            console.warn("Test cleanup incomplete:", orphans);
        }
    });

    afterAll(async () => {
        // Restore all mocks
        vi.restoreAllMocks();
    });

    // Helper function to create complex test data structure
    const createTestData = async () => {
        // Seed test users using database fixtures
        const testUsers = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });

        // Create a bot user
        const testBot = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: "bot-" + Math.random(),
                handle: "test-bot-" + Math.floor(Math.random() * 1000),
                name: "Test Bot",
                isBot: true,
                botSettings: "{}",
            },
        });

        // Create test chats using database fixtures
        const normalChat = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[0].id,
            isPrivate: false,
            openToAnyoneWithInvite: false,
            participantIds: [testUsers[0].id, testUsers[1].id],
        });

        const publicChat = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[1].id,
            isPrivate: false,
            openToAnyoneWithInvite: true,
            participantIds: [testUsers[1].id],
        });

        const privateChat = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[1].id,
            isPrivate: true,
            openToAnyoneWithInvite: false,
            participantIds: [testUsers[1].id],
        });

        const seqChat = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[0].id,
            isPrivate: false,
            openToAnyoneWithInvite: false,
            participantIds: [testUsers[0].id],
        });

        const branchChat = await seedTestChat(DbProvider.get(), {
            createdById: testUsers[0].id,
            isPrivate: false,
            openToAnyoneWithInvite: false,
            participantIds: [testUsers[0].id, testUsers[1].id],
        });

        // Seed messages for normal chat
        const normalChatMessages = await seedChatMessages(DbProvider.get(), {
            chatId: normalChat.id,
            messages: [
                { userId: testUsers[0].id, text: "Hello, this is User 1's first message" },
                { userId: testBot.id, text: "Hello User 1, this is the bot's response", parentId: null },
                { userId: testUsers[1].id, text: "This is User 2 joining the conversation", parentId: null },
            ],
        });
        const messages: any = {};
        messages.user1Message1 = normalChatMessages[0];
        messages.botMessage1 = normalChatMessages[1];
        messages.user2Message1 = normalChatMessages[2];

        // Set parent relationships after creation
        await DbProvider.get().chatMessage.update({
            where: { id: messages.botMessage1.id },
            data: { parentId: messages.user1Message1.id },
        });
        await DbProvider.get().chatMessage.update({
            where: { id: messages.user2Message1.id },
            data: { parentId: messages.botMessage1.id },
        });

        // Seed messages for public chat
        const publicChatMessages = await seedChatMessages(DbProvider.get(), {
            chatId: publicChat.id,
            messages: [
                { userId: testUsers[1].id, text: "This is a message in a public chat" },
            ],
        });
        messages.publicChatMessage = publicChatMessages[0];

        // Seed messages for private chat
        const privateChatMessages = await seedChatMessages(DbProvider.get(), {
            chatId: privateChat.id,
            messages: [
                { userId: testUsers[1].id, text: "This is a message in a private chat" },
            ],
        });
        messages.privateChatMessage = privateChatMessages[0];

        // Seed sequential messages
        const seqMessages = await seedChatMessages(DbProvider.get(), {
            chatId: seqChat.id,
            messages: [
                { userId: testUsers[0].id, text: "Seq A", versionIndex: 0 },
                { userId: testUsers[0].id, text: "Seq B", versionIndex: 1 },
                { userId: testUsers[0].id, text: "Seq C", versionIndex: 2 },
                { userId: testUsers[0].id, text: "Seq D", versionIndex: 3 },
            ],
        });
        messages.seqMsgA = seqMessages[0];
        messages.seqMsgB = seqMessages[1];
        messages.seqMsgC = seqMessages[2];
        messages.seqMsgD = seqMessages[3];

        // Seed branching conversation tree
        const branchTree = await seedConversationTree(DbProvider.get(), {
            chatId: branchChat.id,
            structure: [
                {
                    id: "root",
                    userId: testUsers[0].id,
                    text: "Root message",
                    children: [
                        {
                            id: "branch1A",
                            userId: testUsers[1].id,
                            text: "Branch 1A",
                            children: [
                                {
                                    id: "leaf1A1",
                                    userId: testUsers[0].id,
                                    text: "Leaf 1A1",
                                },
                            ],
                        },
                        {
                            id: "branch1B",
                            userId: testUsers[1].id,
                            text: "Branch 1B",
                            children: [
                                {
                                    id: "leaf1B1",
                                    userId: testUsers[0].id,
                                    text: "Leaf 1B1",
                                },
                                {
                                    id: "leaf1B2",
                                    userId: testUsers[0].id,
                                    text: "Leaf 1B2",
                                },
                            ],
                        },
                    ],
                },
            ],
        });
        Object.assign(messages, branchTree);

        return {
            testUsers,
            testBot,
            normalChat,
            publicChat,
            privateChat,
            seqChat,
            branchChat,
            messages,
        };
    };

    describe("findOne", () => {
        describe("valid", () => {
            it("participant can view message in their chat", async () => {
                const testData = await createTestData();
                const { testUsers, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: messages.user1Message1.id };
                const result = await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(messages.user1Message1.id);
                expect(result.translations).toBeInstanceOf(Array);
                expect(result.translations[0].text).toBe("Hello, this is User 1's first message");
            });

            it("returns message with parent relationship", async () => {
                const testData = await createTestData();
                const { testUsers, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: messages.botMessage1.id };
                const result = await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(messages.botMessage1.id);
                expect(result.parentId).toBe(messages.user1Message1.id);
            });

            it("returns message in public chat for API key with public read permissions", async () => {
                const testData = await createTestData();
                const { testUsers, messages } = testData;

                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id.toString() };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input: FindByIdInput = { id: messages.publicChatMessage.id };
                const result = await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(messages.publicChatMessage.id);
            });
        });

        describe("invalid", () => {
            it("non-participant cannot view message", async () => {
                const testData = await createTestData();
                const { testUsers, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id.toString(), // User 2 is not in normalChat
                });

                const input: FindByIdInput = { id: messages.user1Message1.id };

                await expect(async () => {
                    await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot view message", async () => {
                const testData = await createTestData();
                const { messages } = testData;

                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = { id: messages.user1Message1.id };

                await expect(async () => {
                    await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);
                }).rejects.toThrow();
            });

            it("user cannot view message in private chat they're not part of", async () => {
                const testData = await createTestData();
                const { testUsers, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(), // User 0 is not in privateChat
                });

                const input: FindByIdInput = { id: messages.privateChatMessage.id };

                await expect(async () => {
                    await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);
                }).rejects.toThrow();
            });
        });
    });

    describe("findMany", () => {
        describe("valid", () => {
            it("returns messages for participant", async () => {
                const testData = await createTestData();
                const { testUsers, normalChat, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input = { chatId: normalChat.id, take: 10 };
                const result = await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThanOrEqual(3); // At least the 3 messages we created

                const messageIds = result.edges.map(e => e?.node?.id);
                expect(messageIds).toContain(messages.user1Message1.id);
                expect(messageIds).toContain(messages.botMessage1.id);
                expect(messageIds).toContain(messages.user2Message1.id);
            });

            it("returns messages from user", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input = { chatId: normalChat.id, userId: testUsers[0].id, take: 10 };
                const result = await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only user1Message1
                expect(result.edges[0]?.node?.id).toBe(messages.user1Message1.id);
            });

            it("returns messages for public chat with API key", async () => {
                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = { chatId: publicChat.id, take: 10 };
                const result = await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThanOrEqual(1);
            });
        });

        describe("invalid", () => {
            it("non-participant cannot view messages", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not in normalChat
                });

                const input = { chatId: normalChat.id, take: 10 };

                await expect(async () => {
                    await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);
                }).rejects.toThrow();
            });

            it("logged out user cannot view messages", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input = { chatId: normalChat.id, take: 10 };

                await expect(async () => {
                    await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("findTree", () => {
        describe("valid", () => {
            it("returns full conversation tree for participant", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id,
                    take: 50,
                };
                const result = await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree) as ChatMessageSearchTreeResult;

                expect(result).not.toBeNull();
                expect(result.branches).toBeInstanceOf(Array);

                // Find root message
                const rootBranch = result.branches.find(b => b.messages?.[0]?.id === messages.root.id);
                expect(rootBranch).not.toBeNull();
                expect(rootBranch!.messages).toHaveLength(1);
                expect(rootBranch!.messages![0].translations[0].text).toBe("Root message");
            });

            it("returns sequential messages in order", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: seqChat.id,
                    take: 50,
                };
                const result = await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree) as ChatMessageSearchTreeResult;

                expect(result).not.toBeNull();
                expect(result.branches).toBeInstanceOf(Array);
                expect(result.branches).toHaveLength(1); // Sequential messages should be in one branch

                const messages = result.branches[0].messages!;
                expect(messages).toHaveLength(4);
                expect(messages[0].translations[0].text).toBe("Seq A");
                expect(messages[1].translations[0].text).toBe("Seq B");
                expect(messages[2].translations[0].text).toBe("Seq C");
                expect(messages[3].translations[0].text).toBe("Seq D");
            });

            it("handles branching conversations correctly", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id,
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id,
                    parentId: messages.root.id,
                    take: 50,
                };
                const result = await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree) as ChatMessageSearchTreeResult;

                expect(result).not.toBeNull();
                expect(result.branches).toBeInstanceOf(Array);

                // Should have two branches from root
                const branch1A = result.branches.find(b => b.messages?.some(m => m.id === messages.branch1A.id));
                const branch1B = result.branches.find(b => b.messages?.some(m => m.id === messages.branch1B.id));

                expect(branch1A).not.toBeNull();
                expect(branch1B).not.toBeNull();
            });
        });

        describe("invalid", () => {
            it("non-participant cannot view tree", async () => {
                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id, // User 2 is not in branchChat
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id,
                    take: 50,
                };

                await expect(async () => {
                    await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree);
                }).rejects.toThrow();
            });

            it("logged out user cannot view tree", async () => {
                const { req, res } = await mockLoggedOutSession();

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id,
                    take: 50,
                };

                await expect(async () => {
                    await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree);
                }).rejects.toThrow();
            });
        });
    });
});
