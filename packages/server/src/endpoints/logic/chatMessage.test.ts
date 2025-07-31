import { type ChatMessageSearchTreeInput, type ChatMessageSearchTreeResult, type FindByIdInput, generatePK, generatePublicId } from "@vrooli/shared";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
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
            tables: ["chat", "chat_message", "chat_participants", "chat_invite", "user"],
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
    async function createTestData() {
        // Seed test users using database fixtures
        const seedResult = await seedTestUsers(DbProvider.get(), 3, { withAuth: true });
        const testUsers = seedResult.records;

        // Create a bot user
        const testBot = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                handle: "test-bot-" + Math.floor(Math.random() * 1000),
                name: "Test Bot",
                isBot: true,
                botSettings: "{}",
            },
        });

        // Create test chats using database fixtures
        const normalChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[0].id.toString(), testUsers[1].id.toString()],
            isPrivate: false,
        });

        const publicChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[1].id.toString()],
            isPrivate: false,
        });

        const privateChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[1].id.toString()],
            isPrivate: true,
        });

        const seqChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[0].id.toString()],
            isPrivate: false,
        });

        const branchChat = await seedTestChat(DbProvider.get(), {
            userIds: [testUsers[0].id.toString(), testUsers[1].id.toString()],
            isPrivate: false,
        });

        // Seed messages for normal chat
        const normalChatMessagesResult = await seedChatMessages(DbProvider.get(), {
            chatId: normalChat.id.toString(),
            messages: [
                { userId: testUsers[0].id.toString(), text: "Hello, this is User 1's first message" },
                { userId: testBot.id.toString(), text: "Hello User 1, this is the bot's response", parentId: undefined },
                { userId: testUsers[1].id.toString(), text: "This is User 2 joining the conversation", parentId: undefined },
            ],
        });
        const normalChatMessages = normalChatMessagesResult.records;
        const messages: any = {};
        messages.user1Message1 = normalChatMessages[0];
        messages.botMessage1 = normalChatMessages[1];
        messages.user2Message1 = normalChatMessages[2];

        // Set parent relationships after creation
        await DbProvider.get().chat_message.update({
            where: { id: messages.botMessage1.id },
            data: { parentId: messages.user1Message1.id },
        });
        await DbProvider.get().chat_message.update({
            where: { id: messages.user2Message1.id },
            data: { parentId: messages.botMessage1.id },
        });

        // Seed messages for public chat
        const publicChatMessagesResult = await seedChatMessages(DbProvider.get(), {
            chatId: publicChat.id.toString(),
            messages: [
                { userId: testUsers[1].id.toString(), text: "This is a message in a public chat" },
            ],
        });
        messages.publicChatMessage = publicChatMessagesResult.records[0];

        // Seed messages for private chat
        const privateChatMessagesResult = await seedChatMessages(DbProvider.get(), {
            chatId: privateChat.id.toString(),
            messages: [
                { userId: testUsers[1].id.toString(), text: "This is a message in a private chat" },
            ],
        });
        messages.privateChatMessage = privateChatMessagesResult.records[0];

        // Seed sequential messages
        const seqMessagesResult = await seedChatMessages(DbProvider.get(), {
            chatId: seqChat.id.toString(),
            messages: [
                { userId: testUsers[0].id.toString(), text: "Seq A", versionIndex: 0 },
                { userId: testUsers[0].id.toString(), text: "Seq B", versionIndex: 1 },
                { userId: testUsers[0].id.toString(), text: "Seq C", versionIndex: 2 },
                { userId: testUsers[0].id.toString(), text: "Seq D", versionIndex: 3 },
            ],
        });
        const seqMessages = seqMessagesResult.records;
        messages.seqMsgA = seqMessages[0];
        messages.seqMsgB = seqMessages[1];
        messages.seqMsgC = seqMessages[2];
        messages.seqMsgD = seqMessages[3];

        // Generate IDs for the branching conversation tree
        const rootId = generatePK();
        const branch1AId = generatePK();
        const leaf1A1Id = generatePK();
        const branch1BId = generatePK();
        const leaf1B1Id = generatePK();
        const leaf1B2Id = generatePK();

        // Seed branching conversation tree
        const branchTree = await seedConversationTree(DbProvider.get(), {
            chatId: branchChat.id.toString(),
            structure: [
                {
                    id: rootId.toString(),
                    userId: testUsers[0].id.toString(),
                    text: "Root message",
                    children: [
                        {
                            id: branch1AId.toString(),
                            userId: testUsers[1].id.toString(),
                            text: "Branch 1A",
                            children: [
                                {
                                    id: leaf1A1Id.toString(),
                                    userId: testUsers[0].id.toString(),
                                    text: "Leaf 1A1",
                                },
                            ],
                        },
                        {
                            id: branch1BId.toString(),
                            userId: testUsers[1].id.toString(),
                            text: "Branch 1B",
                            children: [
                                {
                                    id: leaf1B1Id.toString(),
                                    userId: testUsers[0].id.toString(),
                                    text: "Leaf 1B1",
                                },
                                {
                                    id: leaf1B2Id.toString(),
                                    userId: testUsers[0].id.toString(),
                                    text: "Leaf 1B2",
                                },
                            ],
                        },
                    ],
                },
            ],
        });

        // Map the generated IDs to the expected keys
        messages.root = branchTree.records.find(m => m.id === rootId);
        messages.branch1A = branchTree.records.find(m => m.id === branch1AId);
        messages.leaf1A1 = branchTree.records.find(m => m.id === leaf1A1Id);
        messages.branch1B = branchTree.records.find(m => m.id === branch1BId);
        messages.leaf1B1 = branchTree.records.find(m => m.id === leaf1B1Id);
        messages.leaf1B2 = branchTree.records.find(m => m.id === leaf1B2Id);

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
    }

    describe("findOne", () => {
        describe("valid", () => {
            it("participant can view message in their chat", async () => {
                const testData = await createTestData();
                const { testUsers, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input: FindByIdInput = { id: messages.user1Message1.id.toString() };
                const result = await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);

                expect(result).not.toBeNull();
                expect(result.id).toBe(messages.user1Message1.id.toString());
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

                const input: FindByIdInput = { id: messages.botMessage1.id.toString() };
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

                const input: FindByIdInput = { id: messages.publicChatMessage.id.toString() };
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

                const input: FindByIdInput = { id: messages.user1Message1.id.toString() };

                await expect(async () => {
                    await chatMessage.findOne({ input }, { req, res }, chatMessage_findOne);
                }).rejects.toThrow();
            });

            it("logged out user cannot view message", async () => {
                const testData = await createTestData();
                const { messages } = testData;

                const { req, res } = await mockLoggedOutSession();

                const input: FindByIdInput = { id: messages.user1Message1.id.toString() };

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

                const input: FindByIdInput = { id: messages.privateChatMessage.id.toString() };

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

                const input = { chatId: normalChat.id.toString(), take: 10 };
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
                const testData = await createTestData();
                const { testUsers, normalChat, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input = { chatId: normalChat.id.toString(), userId: testUsers[0].id.toString(), take: 10 };
                const result = await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBe(1); // Only user1Message1
                expect(result.edges[0]?.node?.id).toBe(messages.user1Message1.id);
            });

            it("returns messages for public chat with API key", async () => {
                const testData = await createTestData();
                const { testUsers, publicChat } = testData;

                const testUser = { ...loggedInUserNoPremiumData(), id: testUsers[0].id.toString() };
                const permissions = mockReadPublicPermissions();
                const apiToken = ApiKeyEncryptionService.generateSiteKey();
                const { req, res } = await mockApiSession(apiToken, permissions, testUser);

                const input = { chatId: publicChat.id.toString(), take: 10 };
                const result = await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);

                expect(result).not.toBeNull();
                expect(result.edges).toBeInstanceOf(Array);
                expect(result.edges.length).toBeGreaterThanOrEqual(1);
            });
        });

        describe("invalid", () => {
            it("non-participant cannot view messages", async () => {
                const testData = await createTestData();
                const { testUsers, normalChat } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id.toString(), // User 2 is not in normalChat
                });

                const input = { chatId: normalChat.id.toString(), take: 10 };

                await expect(async () => {
                    await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);
                }).rejects.toThrow();
            });

            it("logged out user cannot view messages", async () => {
                const testData = await createTestData();
                const { normalChat } = testData;

                const { req, res } = await mockLoggedOutSession();

                const input = { chatId: normalChat.id.toString(), take: 10 };

                await expect(async () => {
                    await chatMessage.findMany({ input }, { req, res }, chatMessage_findMany);
                }).rejects.toThrow();
            });
        });
    });

    describe("findTree", () => {
        describe("valid", () => {
            it("returns full conversation tree for participant", async () => {
                const testData = await createTestData();
                const { testUsers, branchChat, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id.toString(),
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
                const testData = await createTestData();
                const { testUsers, seqChat } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: seqChat.id.toString(),
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
                const testData = await createTestData();
                const { testUsers, branchChat, messages } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[0].id.toString(),
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id.toString(),
                    parentId: messages.root.id.toString(),
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
                const testData = await createTestData();
                const { testUsers, branchChat } = testData;

                const { req, res } = await mockAuthenticatedSession({
                    ...loggedInUserNoPremiumData(),
                    id: testUsers[2].id.toString(), // User 2 is not in branchChat
                });

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id.toString(),
                    take: 50,
                };

                await expect(async () => {
                    await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree);
                }).rejects.toThrow();
            });

            it("logged out user cannot view tree", async () => {
                const testData = await createTestData();
                const { branchChat } = testData;

                const { req, res } = await mockLoggedOutSession();

                const input: ChatMessageSearchTreeInput = {
                    chatId: branchChat.id.toString(),
                    take: 50,
                };

                await expect(async () => {
                    await chatMessage.findTree({ input }, { req, res }, chatMessage_findTree);
                }).rejects.toThrow();
            });
        });
    });
});
