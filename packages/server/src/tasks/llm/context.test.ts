/* eslint-disable @typescript-eslint/ban-ts-comment */
import { TaskContextInfo } from "@local/shared";
import { expect } from "chai";
import { RedisClientType } from "redis";
import pkg from "../../__mocks__/@prisma/client.js";
import { RedisClientMock } from "../../__mocks__/redis.js";
import { initializeRedis } from "../../redisConn.js";
import { UI_URL } from "../../server.js";
import { PreMapMessageData, PreMapMessageDataDelete, PreMapMessageDataUpdate } from "../../utils/chat.js";
import { ChatContextCollector, ChatContextManager, MessageContextInfo, determineRespondingBots, processMentions, stringifyTaskContexts } from "./context.js";
import { EstimateTokensParams } from "./service.js";
import { OpenAIService } from "./services/openai.js";

const { PrismaClient } = pkg;

jest.mock("../../events/logger");

// NOTE: In these tests, you'll see a lot of checks for specific Redis 
// functions being called or not called. Here's a quick guide:
// - hSet/hGetAll: Message data
// - zAdd/zRange: Chat message order
// - sAdd/sRem/sMembers/zRem: Children of a message (for forked messages)
// - del: Deleting any key; non-specific. Should check parameters to verify
describe("ChatContextManager", () => {
    let chatContextManager: ChatContextManager;

    const lmServices = [
        { name: "OpenAIService", instance: new OpenAIService() },
        // Add other implementations as needed
    ];

    // Initial Redis data
    const initialChatData = {
        "chat:chat1": [{ score: Date.now(), value: "message1" }],
    };
    const initialMessageData = {
        "message:message1": {
            messageId: "message1",
            userId: "user1",
            parentId: "parent1",
            translatedTokenCounts: JSON.stringify({ en: { "default": 10 } }),
        },
    };
    const initialChildrenData = {
        "children:parent1": ["message1"],
    };
    const initialData = {
        ...JSON.parse(JSON.stringify(initialChatData)),
        ...JSON.parse(JSON.stringify(initialMessageData)),
        ...JSON.parse(JSON.stringify(initialChildrenData)),
    };

    // Additional data for testing
    const message2 = {
        chatId: "chat1",
        content: "Hello",
        language: "en",
        messageId: "message2",
        parentId: "parent1",
        translations: [{ id: "123", language: "en", text: "Hello" }],
        userId: "user1",
    } as const;

    lmServices.forEach(({ name: lmServiceName, instance: lmService }) => {
        beforeEach(() => {
            jest.clearAllMocks();
            RedisClientMock.resetMock();
            chatContextManager = new ChatContextManager(lmService);

            // Add mock data to Redis
            RedisClientMock.__setAllMockData(JSON.parse(JSON.stringify(initialData)));
        });

        // Add message
        it(`${lmServiceName}: should add a new message to Redis`, async () => {
            const message = { __type: "Create", ...message2 } as const;
            await chatContextManager.addMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.zAdd).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).toHaveBeenCalled();

            // Verify redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message.chatId}`]).toContainEqual(expect.objectContaining({ value: message.messageId }));
            expect(currentData[`message:${message.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    id: message.messageId,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String),
                }),
            );
            expect(currentData[`children:${message.parentId}`]).to.deep.equal(expect.arrayContaining([message.messageId]));
        });
        it(`${lmServiceName}: should add a message without a parent to Redis`, async () => {
            const message = { __type: "Create", ...message2, parentId: null } as const;
            await chatContextManager.addMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.zAdd).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).not.toHaveBeenCalled();

            // Verify redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message.chatId}`]).toContainEqual(expect.objectContaining({ value: message.messageId }));
            expect(currentData[`message:${message.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    id: message.messageId,
                    userId: message.userId,
                    translatedTokenCounts: expect.any(String),
                }),
            );
            expect(currentData[`children:${message.parentId}`]).not.to.deep.equal(expect.arrayContaining([message.messageId]));
        });

        // Edit message
        it(`${lmServiceName}: should edit an existing message in Redis`, async () => {
            const message: PreMapMessageDataUpdate = {
                __type: "Update",
                chatId: "chat1",
                messageId: "message1",
                parentId: "parent1",
                translations: [{ id: "999", language: "en", text: "Hello, edited" }],
                userId: "user1",
            };

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    id: message.messageId,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String), // Adjust the expectation based on your token count calculation logic
                }),
            );
        });
        it(`${lmServiceName}: should edit a message and handle updated parent ID`, async () => {
            // Verify that message is part of original parent's children
            const originalParentId = initialMessageData["message:message1"].parentId;
            const originalChildren = RedisClientMock.__getAllMockData()[`children:${originalParentId}`];
            expect(originalChildren).to.include("message1");

            const message: PreMapMessageData = {
                __type: "Update",
                ...initialMessageData["message:message1"],
                parentId: "newParentId", // Adding a new parent ID
                chatId: "chat1",
                translations: [{ id: "444", language: "en", text: "Hello, edited" }],
            };

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sRem).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    parentId: message.parentId,
                }),
            );
            // Message should be in new parent's children
            expect(currentData[`children:${message.parentId}`]).to.deep.equal(expect.arrayContaining([message.messageId]));
            // Message should be removed from original parent's children
            expect(currentData[`children:${originalParentId}`]).not.to.deep.equal(expect.arrayContaining([message.messageId]));
        });
        it(`${lmServiceName}: should still work when existing message does not exist`, async () => {
            const message: PreMapMessageData = {
                __type: "Update",
                ...initialMessageData["message:message1"],
                messageId: "nonExistentMessageId",
                chatId: "chat1",
                translations: [{ id: "333", language: "en", text: "Hello, edited" }],
            };

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sRem).not.toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    id: message.messageId,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String), // Adjust the expectation based on your token count calculation logic
                }),
            );
            expect(currentData[`children:${message.parentId}`]).to.deep.equal(expect.arrayContaining([message.messageId]));
        });
        it(`${lmServiceName}: should edit message translations and update token counts`, async () => {
            const messageWithEditedTranslations: PreMapMessageData = {
                __type: "Update",
                ...initialMessageData["message:message1"],
                chatId: "chat1",
                translations: [{ id: "111", language: "en", text: "Edited translation" }], // Updated translations
            };

            // Mock `calculateTokenCounts` to return a specific value
            const mockedTokenCounts = { en: { "default": -420 } };// Negative to ensure that original was never conincidentally this exact value
            jest.spyOn(chatContextManager, "calculateTokenCounts").mockReturnValue(mockedTokenCounts);

            await chatContextManager.editMessage(messageWithEditedTranslations);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${messageWithEditedTranslations.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    translatedTokenCounts: JSON.stringify(mockedTokenCounts),
                }),
            );
        });
        it(`${lmServiceName}: should recover from invalid existing data - bad message`, async () => {
            const message: PreMapMessageData = {
                __type: "Update",
                ...initialMessageData["message:message1"],
                chatId: "chat1",
                translations: [{ id: "123", language: "en", text: "Hello, edited" }],
            };

            // Make existing message data invalid
            RedisClientMock.__setMockData(`message:${message.messageId}`, "invalid data"); // Not an object

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.messageId}`]).to.deep.equal(
                expect.objectContaining({
                    id: message.messageId,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String),
                }),
            );
        });

        // Delete message
        it(`${lmServiceName}: should successfully delete a message and its references`, async () => {
            const message: PreMapMessageDataDelete = {
                __type: "Delete",
                chatId: "chat1",
                messageId: "message1",
            };

            await chatContextManager.deleteMessage(message);

            // Verify Redis operations for deletion
            expect(RedisClientMock.instance?.del).toHaveBeenCalledWith([`message:${message.messageId}`, `children:${message.messageId}`]);
            expect(RedisClientMock.instance?.zRem).toHaveBeenCalledWith(`chat:${message.chatId}`, message.messageId);

            // Verify the message and its children references are removed
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.messageId}`]).to.be.undefined;
            expect(currentData[`children:${message.messageId}`]).to.be.undefined;
            expect(currentData[`chat:${message.chatId}`]).not.toContainEqual(expect.objectContaining({ value: message.messageId }));
        });
        it(`${lmServiceName}: should update children's parent references when deleting a message with children`, async () => {
            const message1 = initialMessageData["message:message1"];

            // Create some children for message1
            const childMessage1 = {
                __type: "Create",
                chatId: "chat1",
                content: "Hello",
                language: "en",
                messageId: "childMessage1",
                parentId: message1.messageId,
                translations: [{ id: "123", language: "en", text: "Hello1" }],
                userId: "user1",
            } as const;
            const childMessage2 = {
                __type: "Create",
                chatId: "chat1",
                content: "Hello",
                language: "en",
                messageId: "childMessage2",
                parentId: message1.messageId,
                translations: [{ id: "234", language: "en", text: "Hello2" }],
                userId: "user1",
            } as const;

            // Add the children
            await chatContextManager.addMessage(childMessage1);
            await chatContextManager.addMessage(childMessage2);

            // Make sure they're added
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`children:${childMessage1.parentId}`]).to.deep.equal(expect.arrayContaining([childMessage1.messageId, childMessage2.messageId]));
            expect(currentData[`message:${childMessage1.messageId}`]).to.deep.equal(expect.objectContaining({ parentId: message1.messageId }));
            expect(currentData[`message:${childMessage2.messageId}`]).to.deep.equal(expect.objectContaining({ parentId: message1.messageId }));

            // Delete the parent message (message1)
            await chatContextManager.deleteMessage({ __type: "Delete", chatId: childMessage1.chatId, messageId: childMessage1.parentId } as const);

            // Verify Redis operations for deletion
            expect(RedisClientMock.instance?.del).toHaveBeenCalledWith([`message:${childMessage1.parentId}`, `children:${childMessage1.parentId}`]);
            expect(RedisClientMock.instance?.zRem).toHaveBeenCalledWith(`chat:${childMessage1.chatId}`, childMessage1.parentId);

            // Verify the message and its children references are removed
            const updatedData = RedisClientMock.__getAllMockData();
            // The parent should now be message1's parent (parent1)
            expect(updatedData[`children:${message1.parentId}`]).to.deep.equal(expect.arrayContaining([childMessage1.messageId, childMessage2.messageId]));
            expect(updatedData[`children:${message1.messageId}`]).to.be.undefined;
            expect(updatedData[`message:${message1.messageId}`]).to.be.undefined;
            expect(updatedData[`chat:${childMessage1.chatId}`]).not.toContainEqual(expect.objectContaining({ value: message1.messageId }));
            expect(updatedData[`chat:${childMessage1.chatId}`]).toContainEqual(expect.objectContaining({ value: childMessage1.messageId }));
            expect(updatedData[`chat:${childMessage1.chatId}`]).toContainEqual(expect.objectContaining({ value: childMessage2.messageId }));
            expect(updatedData[`message:${childMessage1.messageId}`]).to.deep.equal(expect.objectContaining({ parentId: message1.parentId }));
            expect(updatedData[`message:${childMessage2.messageId}`]).to.deep.equal(expect.objectContaining({ parentId: message1.parentId }));
        });
        it(`${lmServiceName}: should gracefully handle attempts to delete a non-existent message`, async () => {
            const message = {
                __type: "Delete",
                chatId: "chat1",
                messageId: "nonExistentMessage",
            } as const;

            // Get starting data
            const startingData = RedisClientMock.__getAllMockData();

            await chatContextManager.deleteMessage(message);

            // Verify that ending data is the same as starting data
            const endingData = RedisClientMock.__getAllMockData();
            expect(endingData).to.deep.equal(startingData);
        });

        // Delete chat
        it(`${lmServiceName}: should delete a chat and all its messages`, async () => {
            // Add some messages to the chat for deletion
            await chatContextManager.addMessage({ __type: "Create", ...message2 } as const); // message2 is defined in the initial test setup
            const message3 = { ...message2, messageId: "message3", parentId: "message2" };
            await chatContextManager.addMessage({ __type: "Create", ...message3 } as const);

            // Ensure messages are added
            let currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message2.chatId}`]).toContainEqual(expect.objectContaining({ value: message2.messageId }));
            expect(currentData[`chat:${message3.chatId}`]).toContainEqual(expect.objectContaining({ value: message3.messageId }));

            // Delete the chat
            await chatContextManager.deleteChat(message2.chatId);

            // Verify Redis operations
            expect(RedisClientMock.instance?.del).toHaveBeenCalled();

            // Verify the chat and all its messages are deleted
            currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message2.chatId}`]).to.be.undefined;
            expect(currentData[`message:${message2.messageId}`]).to.be.undefined;
            expect(currentData[`children:${message2.messageId}`]).to.be.undefined;
            expect(currentData[`message:${message2.messageId}`]).to.be.undefined;
            expect(currentData[`children:${message2.messageId}`]).to.be.undefined;
            expect(currentData[`message:${message3.messageId}`]).to.be.undefined;
            expect(currentData[`children:${message3.messageId}`]).to.be.undefined;
        });
        it(`${lmServiceName}: should handle deleting an empty chat`, async () => {
            const emptyChatId = "emptyChat";

            // Delete the empty chat
            await chatContextManager.deleteChat(emptyChatId);

            // Verify Redis operations
            expect(RedisClientMock.instance?.del).toHaveBeenCalledTimes(1); // Only the chat itself should be deleted

            // Verify the empty chat is deleted
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${emptyChatId}`]).to.be.undefined;
        });
        it(`${lmServiceName}: should gracefully handle deleting a non-existent chat`, async () => {
            const nonExistentChatId = "nonExistentChat";

            // Get starting data
            const startingData = RedisClientMock.__getAllMockData();

            // Attempt to delete a non-existent chat
            await chatContextManager.deleteChat(nonExistentChatId);

            // Verify Redis operations
            expect(RedisClientMock.instance?.del).toHaveBeenCalledTimes(1); // Attempt to delete the non-existent chat

            // Verify that ending data is the same as starting data
            const endingData = RedisClientMock.__getAllMockData();
            expect(endingData).to.deep.equal(startingData);
        });

        // Token counts
        const translations = [
            { language: "en", text: "Hello world" },
            { language: "es", text: "Hola mundo" },
        ];
        const estimationTypes = lmService.getEstimationTypes();
        const tokenEstimationMock = ({ text, model }: EstimateTokensParams) => ({ model: model as "default", tokens: text.split(" ").length });
        it(`${lmServiceName}: should calculate token counts for each translation using provided estimation methods`, () => {
            // Mock the estimateTokens method
            jest.spyOn(lmService, "estimateTokens").mockImplementation(tokenEstimationMock);

            const tokenCounts = chatContextManager.calculateTokenCounts(translations, ...estimationTypes);

            // Verify structure and reasonableness of token counts
            Object.entries(tokenCounts).forEach(([language, methods]) => {
                estimationTypes.forEach(method => {
                    expect(methods).to.have.property(method);
                    const count = methods[method];
                    expect(count).to.be.at.least(0); // Token count should be non-negative
                    expect(count).to.be.at.most(translations.find(t => t.language === language)!.text.length); // Token count should not exceed message length
                    expect(Number.isInteger(count)).to.be.ok; // Token count should be an integer
                });
            });
        });
        it(`${lmServiceName}: should handle an empty translations array gracefully`, () => {
            const tokenCounts = chatContextManager.calculateTokenCounts([], ...estimationTypes);

            // Expect an empty object when no translations are provided
            expect(tokenCounts).to.deep.equal({});
        });
        it(`${lmServiceName}: should handle undefined or null estimation methods gracefully`, () => {
            // @ts-ignore: Testing runtime scenario
            const tokenCounts = chatContextManager.calculateTokenCounts(translations, undefined, null);

            // Expect token counts to still be calculated using some default method or skipped
            Object.keys(tokenCounts).forEach(language => {
                expect(tokenCounts[language]).to.exist;
                expect(Object.keys(tokenCounts[language])).to.include("default"); // Assuming 'default' is a fallback method
            });
        });
    });
});

describe("ChatContextCollector", () => {
    let chatContextCollector: ChatContextCollector;
    let redis: RedisClientType;

    const lmServices = [
        { name: "OpenAIService", instance: new OpenAIService() },
        // Add other implementations as needed
    ];

    // Initial Redis data setup
    const initialChatData = {
        "chat:chat1": [
            { score: Date.now() - 5000, value: "message1" },
            { score: Date.now() - 4000, value: "message2" },
            { score: Date.now() - 3000, value: "message3" },
            { score: Date.now() - 2000, value: "message4" },
            { score: Date.now() - 1000, value: "message5" },
            { score: Date.now(), value: "message6" },
        ],
        "chat:emptyChat": [],
    };
    const initialMessageData = {
        "message:message1": {
            id: "message1",
            userId: "alice",
            parentId: null,
            content: "This is the first message",
            language: "en",
            translatedTokenCounts: JSON.stringify({ en: { "default": 100 } }),
        },
        // Child of first message
        "message:message2": {
            id: "message2",
            userId: "bob",
            parentId: "message1",
            content: "This is the second message",
            language: "en",
            translatedTokenCounts: JSON.stringify({ en: { "default": 100 } }),
        },
        // Also a child of first message (i.e. chat has forked)
        "message:message3": {
            id: "message3",
            userId: "bob",
            parentId: "message1",
            content: "This is the third message",
            language: "en",
            translatedTokenCounts: JSON.stringify({ en: { "default": 100 } }),
        },
        // Child of second message
        "message:message4": {
            id: "message4",
            userId: "alice",
            parentId: "message2",
            content: "This is the fourth message",
            language: "en",
            translatedTokenCounts: JSON.stringify({ en: { "default": 100 } }),
        },
        // Child of fourth message
        "message:message5": {
            id: "message5",
            userId: "bob",
            parentId: "message4",
            content: "This is the fifth message",
            language: "en",
            translatedTokenCounts: JSON.stringify({ en: { "default": 100 } }),
        },
        // Child of third message
        "message:message6": {
            id: "message6",
            userId: "bob",
            parentId: "message3",
            content: "This is the sixth message",
            language: "en",
            translatedTokenCounts: JSON.stringify({ en: { "default": 100 } }),
        },
    };
    const initialChildrenData = {
        "children:message1": ["message2", "message3"],
        "children:message2": ["message4"],
        "children:message3": ["message6"],
        "children:message4": ["message5"],
    };
    const initialData = {
        ...JSON.parse(JSON.stringify(initialChatData)),
        ...JSON.parse(JSON.stringify(initialMessageData)),
        ...JSON.parse(JSON.stringify(initialChildrenData)),
    };

    lmServices.forEach(({ name: lmServiceName, instance: lmService }) => {
        beforeEach(async () => {
            jest.clearAllMocks();
            RedisClientMock.resetMock();
            redis = await initializeRedis() as RedisClientType;
            chatContextCollector = new ChatContextCollector(lmService);

            // Add mock data to the databases
            RedisClientMock.__setAllMockData(JSON.parse(JSON.stringify(initialData)));
            PrismaClient.injectData({
                ChatMessage: [
                    {
                        id: "message7",
                        parent: { id: "message3" },
                        translations: [{ language: "en", text: "This is the seventh message" }],
                        user: { id: "charlie" },
                    },
                    {
                        id: "message8",
                        parent: { id: "message4" },
                        translations: [{ language: "en", text: "This is the eighth message" }],
                        user: { id: "dave" },
                    },
                ],
            });

            // Mock language model service methods to simplify testing
            jest.spyOn(lmService, "getContextSize").mockReturnValue(350 as any); // Since we've set each message to have 100 tokens, this should be able to hold 3 messages
            jest.spyOn(lmService, "getEstimationMethod").mockReturnValue("default");
        });

        afterEach(() => {
            PrismaClient.clearData();
        });

        // Collect context
        it(`${lmServiceName}: should collect the correct context for the latest message`, async () => {
            const chatId = "chat1";
            const model = "defaultModel";
            const languages = ["en"];
            const latestMessage = "message6";

            const contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages, latestMessage });

            // Expect to collect context for message6, its parent (message3), and its grandparent (message1)
            expect(contextInfo.length).to.equal(3);
            expect(contextInfo.map(info => (info as MessageContextInfo).messageId)).to.deep.equal(expect.arrayContaining(["message1", "message3", "message6"]));
            contextInfo.forEach(info => {
                expect(info.tokenSize).to.equal(100); // Based on initialMessageData
            });
        });
        it(`${lmServiceName}: should handle cases with no latest message id provided`, async () => {
            const chatId = "chat1";
            const model = "defaultModel";
            const languages = ["en"];

            const contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages });

            // Since the most recent message is message6, the result should be the same as the previous test
            expect(contextInfo.length).to.equal(3);
            expect(contextInfo.map(info => (info as MessageContextInfo).messageId)).to.deep.equal(expect.arrayContaining(["message1", "message3", "message6"]));
            contextInfo.forEach(info => {
                expect(info.tokenSize).to.equal(100); // Based on initialMessageData
            });
        });
        it(`${lmServiceName}: should limit the context collection to the context size`, async () => {
            const chatId = "chat1";
            const model = "defaultModel";
            const languages = ["en"];
            const latestMessage = "message5"; // Starting from a deep child

            let contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages, latestMessage });

            // Given the context size of 350 and token count of 100 per message, expect to collect up to 3 messages
            expect(contextInfo.length).to.equal(3);

            // Now let's lower the context size to 200 and try again
            jest.spyOn(lmService, "getContextSize").mockReturnValue(200 as any);

            contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages, latestMessage });

            // Given the context size of 200 and token count of 100 per message, expect to collect up to 2 messages
            expect(contextInfo.length).to.equal(2);
        });
        it(`${lmServiceName}: should stop early when missing messages in the chain`, async () => {
            // Remove message2 from Redis
            RedisClientMock.__deleteMockData("message:message2");

            const chatId = "chat1";
            const model = "defaultModel";
            const languages = ["en"];
            const latestMessage = "message5";

            const contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages, latestMessage });

            // Should be able to collect message5 and its parent (message4), but not its grandparent (message2) or any other ancestors
            expect(contextInfo.length).to.equal(2);
            expect(contextInfo.map(info => (info as MessageContextInfo).messageId)).to.deep.equal(expect.arrayContaining(["message4", "message5"]));
        });

        // Latest message ID
        it(`${lmServiceName}: should return the ID of the most recent message in a chat with messages`, async () => {
            const latestMessageId = await chatContextCollector.getLatestMessageId(redis, "chat1");

            // Expect the latest message ID to be 'message6', as per the initialMessageData
            expect(latestMessageId).to.equal("message6");
        });
        it(`${lmServiceName}: should return null for an empty chat`, async () => {
            const latestMessageId = await chatContextCollector.getLatestMessageId(redis, "emptyChat");

            // Expect null for an empty chat
            expect(latestMessageId).to.be.null;
        });
        it(`${lmServiceName}: should return null for a non-existent chat ID`, async () => {
            const latestMessageId = await chatContextCollector.getLatestMessageId(redis, "nonExistentChatId");

            // Expect null for a non-existent chat ID
            expect(latestMessageId).to.be.null;
        });

        // Get message details
        it(`${lmServiceName}: should retrieve message details from Redis if available`, async () => {
            const messageId = "message2"; // An ID from the initial Redis mock data
            const messageDetails = await chatContextCollector.getMessageDetails(redis, messageId);

            expect(messageDetails).to.deep.equal(expect.objectContaining({
                id: messageId,
                parentId: "message1", // Based on initialMessageData
                translatedTokenCounts: expect.any(Object), // Token counts are parsed JSON
                userId: "bob", // Based on initialMessageData
            }));
        });
        it(`${lmServiceName}: should fetch message details from the database if not in Redis and store them in Redis`, async () => {
            const messageId = "message7"; // ID not in initial Redis mock data, but in Prisma mock
            const messageDetails = await chatContextCollector.getMessageDetails(redis, messageId);

            // @ts-ignore: Testing runtime scenario
            expect(PrismaClient.instance.chat_message.findUnique).toHaveBeenCalledWith({
                where: { id: messageId },
                select: expect.any(Object),
            });

            expect(messageDetails).to.deep.equal(expect.objectContaining({
                id: messageId,
                parentId: "message3", // Based on Prisma mock data
                translatedTokenCounts: expect.any(Object), // Token counts are calculated and should be an object
                userId: "charlie", // Based on Prisma mock data
            }));

            // Verify Redis now contains the fetched message details
            const redisData = await redis.hGetAll(`message:${messageId}`);
            expect(redisData).to.deep.equal(expect.objectContaining({
                id: messageId,
                parentId: "message3",
                translatedTokenCounts: expect.any(String), // Should be JSON stringified
                userId: "charlie",
            }));
        });
        it(`${lmServiceName}: should return null if message details cannot be fetched from Redis or the database`, async () => {
            const messageId = "nonExistentMessage";
            await expect(chatContextCollector.getMessageDetails(redis, messageId)).resolves.to.be.null;
        });

        // Language token count
        it("should return the token count for a preferred language if available", async () => {
            const messageId = "message1";
            const estimationMethod = "default";
            const languages = ["en"];
            const expectedTokenCount = 100;

            // Mock Redis response
            RedisClientMock.__setMockData(`message:${messageId}`, {
                translatedTokenCounts: JSON.stringify({ en: { "default": expectedTokenCount } }),
            });

            const [tokenCount, language] = await chatContextCollector.getTokenCountForLanguages(redis, messageId, estimationMethod, languages);

            expect(tokenCount).to.equal(expectedTokenCount);
            expect(language).to.equal("en");
        });
        it("should calculate and update token count if not available in cache", async () => {
            const messageId = "message2";
            const estimationMethod = "default";
            const languages = ["fr"]; // Preferred language not initially in cache
            const expectedTokenCount = 150; // Token count after calculation

            // Mock Redis to initially return empty or irrelevant data
            RedisClientMock.__setMockData(`message:${messageId}`, {
                translatedTokenCounts: JSON.stringify({}), // No counts initially
            });

            // Mock Prisma response for message translations
            // @ts-ignore: Testing runtime scenario
            PrismaClient.instance.chat_message.findUnique.mockResolvedValueOnce({
                translations: [{ language: "fr", text: "Ceci est un message" }],
            });

            // Mock token count calculation
            jest.spyOn(ChatContextManager.prototype, "calculateTokenCounts").mockReturnValueOnce({
                fr: { "default": expectedTokenCount },
            });

            const [tokenCount, language] = await chatContextCollector.getTokenCountForLanguages(redis, messageId, estimationMethod, languages);

            expect(tokenCount).to.equal(expectedTokenCount);
            expect(language).to.equal("fr");
            expect(redis.hSet).toHaveBeenCalledWith(`message:${messageId}`, "translatedTokenCounts", expect.any(String)); // Ensure cache is updated
        });
        it("should use the first available language if preferred languages are not available", async () => {
            const messageId = "message3";
            const estimationMethod = "default";
            const languages = ["es"]; // Preferred language not available
            const fallbackLanguage = "en"; // First available language in cache
            const expectedTokenCount = 120;

            // Mock Redis response with only English token count available
            RedisClientMock.__setMockData(`message:${messageId}`, {
                translatedTokenCounts: JSON.stringify({ en: { "default": expectedTokenCount } }),
            });

            const [tokenCount, language] = await chatContextCollector.getTokenCountForLanguages(redis, messageId, estimationMethod, languages);

            expect(tokenCount).to.equal(expectedTokenCount);
            expect(language).to.equal(fallbackLanguage);
        });
        it("should return -1 if token counts cannot be found or calculated", async () => {
            const messageId = "message4";
            const estimationMethod = "default";
            const languages = ["en"];

            // Mock Redis with no data
            RedisClientMock.__setMockData(`message:${messageId}`, {});

            // Mock Prisma to simulate no translations available
            // @ts-ignore: Testing runtime scenario
            PrismaClient.instance.chat_message.findUnique.mockResolvedValueOnce(null);

            const [tokenCount, language] = await chatContextCollector.getTokenCountForLanguages(redis, messageId, estimationMethod, languages);

            expect(tokenCount).to.equal(-1);
            expect(language).to.equal("");
        });
    });
});

describe("processMentions", () => {
    const chat = { botParticipants: ["bot1", "bot2", "bot3"] };
    const bots = [
        { id: "bot1", name: "Alice" },
        { id: "bot2", name: "Bob" },
        // Assuming 'bot3' does not match a bot name to test the failure case
    ];

    it("returns an empty array for messages without mentions", () => {
        const messageContent = "This is a message without any mentions.";
        expect(processMentions(messageContent, chat, bots)).to.deep.equal([]);
    });

    it("ignores non-mention markdown links", () => {
        const messageContent = "Here's a [@link](http://example.com) that is not a mention.";
        expect(processMentions(messageContent, chat, bots)).to.deep.equal([]);
    });

    it("processes valid mentions correctly", () => {
        const messageContent = `Hello [@Alice](${UI_URL})!`;
        expect(processMentions(messageContent, chat, bots)).to.deep.equal(["bot1"]);
    });

    it("returns an empty array for mentions that do not match any bot", () => {
        const messageContent = `Hello [@Charlie](${UI_URL}/@Charlie)!`;
        expect(processMentions(messageContent, chat, bots)).to.deep.equal([]);
    });

    it("responds to @Everyone mention by returning all bot participants", () => {
        const messageContent = `Attention [@Everyone](${UI_URL}/fdksf?asdfa="fdasfs")!`;
        expect(processMentions(messageContent, chat, bots)).to.deep.equal(["bot1", "bot2", "bot3"]);
    });

    it("filters out mentions with incorrect origin", () => {
        const messageContent = "Hello [@Alice](https://another-site.com/@Alice)!";
        expect(processMentions(messageContent, chat, bots)).to.deep.equal([]);
    });

    it("handles malformed markdown links gracefully", () => {
        const messageContent = "This is a malformed link [@Alice](@Alice) without proper markdown.";
        expect(processMentions(messageContent, chat, bots)).to.deep.equal([]);
    });

    it("deduplicates mentions", () => {
        const messageContent = `Hi [@Alice](${UI_URL}) and [@Alice](${UI_URL}) again!`;
        expect(processMentions(messageContent, chat, bots)).to.deep.equal(["bot1"]);
    });
});

describe("determineRespondingBots", () => {
    const userId = "user123";
    const chat = { botParticipants: ["bot1", "bot2"], participantsCount: 3 };
    const bots = [
        { id: "bot1", name: "BotOne" },
        { id: "bot2", name: "BotTwo" },
    ];

    it("returns an empty array if message content is blank", () => {
        const message = { userId, content: "   " };
        expect(determineRespondingBots(message.content, message.userId, chat, bots, userId)).to.deep.equal([]);
    });

    it("returns an empty array if message userId does not match", () => {
        const message = { userId: "anotherUser", content: "Hello" };
        expect(determineRespondingBots(message.content, message.userId, chat, bots, userId)).to.deep.equal([]);
    });

    it("returns an empty array if there are no bots in the chat", () => {
        const message = { userId, content: "Hello" };
        expect(determineRespondingBots(message.content, message.userId, chat, [], userId)).to.deep.equal([]);
    });

    it("returns the only bot if there is one bot in a two-participant chat", () => {
        const singleBotChat = { botParticipants: ["bot1"], participantsCount: 2 };
        const singleBot = [{ id: "bot1", name: "BotOne" }];
        const message = { userId, content: "Hello" };
        expect(determineRespondingBots(message.content, message.userId, singleBotChat, singleBot, userId)).to.deep.equal(["bot1"]);
    });

    // Assuming processMentions is a mockable function that has been properly mocked to return bot IDs based on mentions
    it("returns bot IDs based on mentions in the message content", () => {
        const message = { userId, content: `Hello [@BotOne](${UI_URL}/@BotOne)` };
        // Mock `processMentions` here to return ['bot1'] when called with message.content, chat, and bots
        // For demonstration purposes, this test assumes `processMentions` behavior is correctly implemented elsewhere
        expect(determineRespondingBots(message.content, message.userId, chat, bots, userId)).to.deep.equal(["bot1"]);
    });

    it("handles unexpected input gracefully", () => {
        const message = { userId, content: null }; // Simulate an unexpected null content
        // @ts-ignore: Testing runtime scenario
        expect(() => determineRespondingBots(message.content, message.userId, chat, bots, userId)).not.to.throw();
        // @ts-ignore: Testing runtime scenario
        expect(determineRespondingBots(message.content, message.userId, chat, bots, userId)).to.deep.equal([]);
    });
});

describe("stringifyTaskContexts", () => {
    const task = "Submit Report";

    it("Single context without template (data is an object)", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: { title: "Annual Report", dueDate: "2023-12-31" },
                label: "Existing Data",
            },
        ];

        const expectedOutput = JSON.stringify(contexts[0].data, null, 2);

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Single context with null template (data is an object)", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: { title: "Annual Report", dueDate: "2023-12-31" },
                label: "Existing Data",
                template: null,
            },
        ];

        const expectedOutput = JSON.stringify(contexts[0].data, null, 2);

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Single context without template (data is a string)", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "This is a sample string data.",
                label: "String Data",
            },
        ];

        const expectedOutput = contexts[0].data;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Single context with default template variables", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: { title: "Annual Report" },
                label: "Report Data",
                template: "Task: <TASK>\nData:\n<DATA>",
            },
        ];

        const expectedOutput = `Task: ${task}\nData:\n${JSON.stringify(contexts[0].data, null, 2)}`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Single context with custom template variables", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: { title: "Annual Report" },
                label: "Report Data",
                template: "Here is the data for **{taskName}**:\n{dataContent}",
                templateVariables: {
                    task: "{taskName}",
                    data: "{dataContent}",
                },
            },
        ];

        const expectedOutput = `Here is the data for **${task}**:\n${JSON.stringify(contexts[0].data, null, 2)}`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Single context with variable names containing regex special characters", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Sample data with special variables.",
                label: "",
                template: "Variable [task]: [task*name]\nVariable [data]: [data+content]",
                templateVariables: {
                    task: "[task*name]",
                    data: "[data+content]",
                },
            },
        ];

        const expectedOutput = `Variable [task]: ${task}\nVariable [data]: ${contexts[0].data}`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Multiple contexts with and without templates", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: { name: "Alice" },
                label: "User Data",
                template: "User: <TASK>\nDetails:\n<DATA>",
            },
            {
                id: "context2",
                data: "Additional context information.",
                label: "Extra Info",
            },
        ];

        const expectedOutput1 = `User: ${task}\nDetails:\n${JSON.stringify(contexts[0].data, null, 2)}`;
        const expectedOutput2 = contexts[1].data;
        const expectedOutput = `${expectedOutput1}\n\n${expectedOutput2}`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context without template", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Data without label.",
                label: "unused",
            },
        ];

        const expectedOutput = contexts[0].data;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with undefined templateVariables", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Sample data.",
                label: "unused",
                template: "Task is <TASK> and data is <DATA>",
            },
        ];

        const expectedOutput = `Task is ${task} and data is ${contexts[0].data}`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with null data", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: null,
                label: "Null Data",
            },
        ];

        const expectedOutput = "null";

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with data as number", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: 42,
                label: "Number Data",
            },
        ];

        const expectedOutput = "42";

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with data as boolean", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: true,
                label: "Boolean Data",
            },
        ];

        const expectedOutput = "true";

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with empty data", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: {},
                label: "Empty Data",
            },
        ];

        const expectedOutput = "{}";

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with empty template", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Data with empty template.",
                label: "Empty Template",
                template: "",
            },
        ];

        const expectedOutput = contexts[0].data;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with template variables that are not in the template", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Data.",
                label: "blah",
                template: "This is a template without variables.",
                templateVariables: {
                    task: "{unusedTaskVar}",
                    data: "{unusedDataVar}",
                },
            },
        ];

        const expectedOutput = contexts[0].template;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with overlapping variable names", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Overlapping variables test.",
                label: "Overlap Test",
                template: "Variable VAR and VARIABLE_VAR",
                templateVariables: {
                    task: "VAR",
                    data: "VARIABLE_VAR",
                },
            },
        ];

        const expectedOutput = `Variable ${task} and ${contexts[0].data}`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with variables appearing multiple times", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: "Repeated data",
                label: "Repeat Test",
                template: "Data: DATA, again DATA.",
                templateVariables: {
                    data: "DATA",
                },
            },
        ];

        const expectedOutput = `Data: ${contexts[0].data}, again ${contexts[0].data}.`;

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });

    it("Context with undefined data", () => {
        const contexts: TaskContextInfo[] = [
            {
                id: "context1",
                data: undefined,
                label: "Undefined Data",
            },
        ];

        const expectedOutput = "undefined";

        const result = stringifyTaskContexts(task, contexts);

        expect(result).to.equal(expectedOutput);
    });
});
