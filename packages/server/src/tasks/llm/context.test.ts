/* eslint-disable @typescript-eslint/ban-ts-comment */
import { RedisClientType } from "redis";
import pkg from "../../__mocks__/@prisma/client";
import { RedisClientMock } from "../../__mocks__/redis";
import { initializeRedis } from "../../redisConn";
import { UI_URL } from "../../server";
import { PreMapMessageData } from "../../utils/chat";
import { ChatContextCollector, ChatContextManager, determineRespondingBots, MessageContextInfo, processMentions } from "./context";
import { EstimateTokensParams } from "./service";
import { OpenAIService } from "./services/openai";

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
            id: "message1",
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
        id: "message2",
        isNew: true,
        language: "en",
        parentId: "parent1",
        translations: [{ language: "en", text: "Hello" }],
        userId: "user1",
    };

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
            const message = { ...message2, isNew: true };
            await chatContextManager.addMessage(message.chatId, message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.zAdd).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).toHaveBeenCalled();

            // Verify redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message.chatId}`]).toContainEqual(expect.objectContaining({ value: message.id }));
            expect(currentData[`message:${message.id}`]).toEqual(
                expect.objectContaining({
                    id: message.id,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String),
                }),
            );
            expect(currentData[`children:${message.parentId}`]).toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should refuse to add a message not marked as new`, async () => {
            const message = { ...message2, isNew: false };
            await chatContextManager.addMessage(message.chatId, message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).not.toHaveBeenCalled();
            expect(RedisClientMock.instance?.zAdd).not.toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).not.toHaveBeenCalled();

            // Verify redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message.chatId}`]).not.toContainEqual(expect.objectContaining({ value: message.id }));
            expect(currentData[`message:${message.id}`]).toBeUndefined();
            expect(currentData[`children:${message.parentId}`]).not.toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should add a message without a parent to Redis`, async () => {
            const message = { ...message2, isNew: true, parentId: undefined };
            await chatContextManager.addMessage(message.chatId, message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.zAdd).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).not.toHaveBeenCalled();

            // Verify redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${message.chatId}`]).toContainEqual(expect.objectContaining({ value: message.id }));
            expect(currentData[`message:${message.id}`]).toEqual(
                expect.objectContaining({
                    id: message.id,
                    userId: message.userId,
                    translatedTokenCounts: expect.any(String),
                }),
            );
            expect(currentData[`children:${message.parentId}`]).not.toEqual(expect.arrayContaining([message.id]));
        });

        // Edit message
        it(`${lmServiceName}: should edit an existing message in Redis`, async () => {
            const message = {
                ...initialMessageData["message:message1"],
                chatId: "chat1",
                content: "Hello, edited",
                isNew: false,
                language: "en",
                translations: [{ language: "en", text: "Hello, edited" }],
            };

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.id}`]).toEqual(
                expect.objectContaining({
                    id: message.id,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String), // Adjust the expectation based on your token count calculation logic
                }),
            );
        });
        it(`${lmServiceName}: should refuse to edit a message marked as new`, async () => {
            const newMessage: PreMapMessageData = {
                ...message2,
                isNew: true, // Marked as new, so edit should fail
            };

            await chatContextManager.editMessage(newMessage);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).not.toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${newMessage.id}`]).toBeUndefined();
        });
        it(`${lmServiceName}: should edit a message and handle updated parent ID`, async () => {
            // Verify that message is part of original parent's children
            const originalParentId = initialMessageData["message:message1"].parentId;
            const originalChildren = RedisClientMock.__getAllMockData()[`children:${originalParentId}`];
            expect(originalChildren).toContain("message1");

            const message: PreMapMessageData = {
                ...initialMessageData["message:message1"],
                parentId: "newParentId", // Adding a new parent ID
                chatId: "chat1",
                content: "Hello, edited",
                isNew: false,
                language: "en",
                translations: [{ language: "en", text: "Hello, edited" }],
            };

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sRem).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.id}`]).toEqual(
                expect.objectContaining({
                    parentId: message.parentId,
                }),
            );
            // Message should be in new parent's children
            expect(currentData[`children:${message.parentId}`]).toEqual(expect.arrayContaining([message.id]));
            // Message should be removed from original parent's children
            expect(currentData[`children:${originalParentId}`]).not.toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should still work when existing message does not exist`, async () => {
            const message: PreMapMessageData = {
                ...initialMessageData["message:message1"],
                id: "nonExistentMessageId",
                chatId: "chat1",
                content: "Hello, edited",
                isNew: false,
                language: "en",
                translations: [{ language: "en", text: "Hello, edited" }],
            };

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();
            expect(RedisClientMock.instance?.sRem).not.toHaveBeenCalled();
            expect(RedisClientMock.instance?.sAdd).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.id}`]).toEqual(
                expect.objectContaining({
                    id: message.id,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String), // Adjust the expectation based on your token count calculation logic
                }),
            );
            expect(currentData[`children:${message.parentId}`]).toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should edit message translations and update token counts`, async () => {
            const messageWithEditedTranslations: PreMapMessageData = {
                ...initialMessageData["message:message1"],
                chatId: "chat1",
                content: "Edited translation",
                isNew: false,
                language: "en",
                translations: [{ language: "en", text: "Edited translation" }], // Updated translations
            };

            // Mock `calculateTokenCounts` to return a specific value
            const mockedTokenCounts = { en: { "default": -420 } };// Negative to ensure that original was never conincidentally this exact value
            jest.spyOn(chatContextManager, "calculateTokenCounts").mockReturnValue(mockedTokenCounts);

            await chatContextManager.editMessage(messageWithEditedTranslations);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${messageWithEditedTranslations.id}`]).toEqual(
                expect.objectContaining({
                    translatedTokenCounts: JSON.stringify(mockedTokenCounts),
                }),
            );
        });
        it(`${lmServiceName}: should recover from invalid existing data - bad message`, async () => {
            const message: PreMapMessageData = {
                ...initialMessageData["message:message1"],
                chatId: "chat1",
                content: "Hello, edited",
                isNew: false,
                language: "en",
                translations: [{ language: "en", text: "Hello, edited" }],
            };

            // Make existing message data invalid
            RedisClientMock.__setMockData(`message:${message.id}`, "invalid data"); // Not an object

            await chatContextManager.editMessage(message);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${message.id}`]).toEqual(
                expect.objectContaining({
                    id: message.id,
                    userId: message.userId,
                    parentId: message.parentId,
                    translatedTokenCounts: expect.any(String),
                }),
            );
        });

        // Delete message
        it(`${lmServiceName}: should successfully delete a message and its references`, async () => {
            const chatId = "chat1";
            const messageId = "message1";

            await chatContextManager.deleteMessage(chatId, messageId);

            // Verify Redis operations for deletion
            expect(RedisClientMock.instance?.del).toHaveBeenCalledWith([`message:${messageId}`, `children:${messageId}`]);
            expect(RedisClientMock.instance?.zRem).toHaveBeenCalledWith(`chat:${chatId}`, messageId);

            // Verify the message and its children references are removed
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${messageId}`]).toBeUndefined();
            expect(currentData[`children:${messageId}`]).toBeUndefined();
            expect(currentData[`chat:${chatId}`]).not.toContainEqual(expect.objectContaining({ value: messageId }));
        });
        it(`${lmServiceName}: should update children's parent references when deleting a message with children`, async () => {
            const message1 = initialMessageData["message:message1"];

            // Create some children for message1
            const childMessage1 = {
                chatId: "chat1",
                content: "Hello",
                id: "childMessage1",
                isNew: true,
                language: "en",
                parentId: message1.id,
                translations: [{ language: "en", text: "Hello1" }],
                userId: "user1",
            };
            const childMessage2 = {
                chatId: "chat1",
                content: "Hello",
                id: "childMessage2",
                isNew: true,
                language: "en",
                parentId: message1.id,
                translations: [{ language: "en", text: "Hello2" }],
                userId: "user1",
            };

            // Add the children
            await chatContextManager.addMessage(childMessage1.chatId, childMessage1);
            await chatContextManager.addMessage(childMessage2.chatId, childMessage2);

            // Make sure they're added
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`children:${childMessage1.parentId}`]).toEqual(expect.arrayContaining([childMessage1.id, childMessage2.id]));
            expect(currentData[`message:${childMessage1.id}`]).toEqual(expect.objectContaining({ parentId: message1.id }));
            expect(currentData[`message:${childMessage2.id}`]).toEqual(expect.objectContaining({ parentId: message1.id }));

            // Delete the parent message (message1)
            await chatContextManager.deleteMessage(childMessage1.chatId, childMessage1.parentId);

            // Verify Redis operations for deletion
            expect(RedisClientMock.instance?.del).toHaveBeenCalledWith([`message:${childMessage1.parentId}`, `children:${childMessage1.parentId}`]);
            expect(RedisClientMock.instance?.zRem).toHaveBeenCalledWith(`chat:${childMessage1.chatId}`, childMessage1.parentId);

            // Verify the message and its children references are removed
            const updatedData = RedisClientMock.__getAllMockData();
            // The parent should now be message1's parent (parent1)
            expect(updatedData[`children:${message1.parentId}`]).toEqual(expect.arrayContaining([childMessage1.id, childMessage2.id]));
            expect(updatedData[`children:${message1.id}`]).toBeUndefined();
            expect(updatedData[`message:${message1.id}`]).toBeUndefined();
            expect(updatedData[`chat:${childMessage1.chatId}`]).not.toContainEqual(expect.objectContaining({ value: message1.id }));
            expect(updatedData[`chat:${childMessage1.chatId}`]).toContainEqual(expect.objectContaining({ value: childMessage1.id }));
            expect(updatedData[`chat:${childMessage1.chatId}`]).toContainEqual(expect.objectContaining({ value: childMessage2.id }));
            expect(updatedData[`message:${childMessage1.id}`]).toEqual(expect.objectContaining({ parentId: message1.parentId }));
            expect(updatedData[`message:${childMessage2.id}`]).toEqual(expect.objectContaining({ parentId: message1.parentId }));
        });
        it(`${lmServiceName}: should gracefully handle attempts to delete a non-existent message`, async () => {
            const chatId = "chat1";
            const messageId = "nonExistentMessage";

            // Get starting data
            const startingData = RedisClientMock.__getAllMockData();

            await chatContextManager.deleteMessage(chatId, messageId);

            // Verify that ending data is the same as starting data
            const endingData = RedisClientMock.__getAllMockData();
            expect(endingData).toEqual(startingData);
        });

        // Delete chat
        it(`${lmServiceName}: should delete a chat and all its messages`, async () => {
            const chatId = "chat1";
            const message1 = initialMessageData["message:message1"];

            // Add some messages to the chat for deletion
            await chatContextManager.addMessage(chatId, message2); // message2 is defined in the initial test setup
            const message3 = { ...message2, id: "message3", parentId: "message2", isNew: true };
            await chatContextManager.addMessage(chatId, message3);

            // Ensure messages are added
            let currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${chatId}`]).toContainEqual(expect.objectContaining({ value: message2.id }));
            expect(currentData[`chat:${chatId}`]).toContainEqual(expect.objectContaining({ value: message3.id }));

            // Delete the chat
            await chatContextManager.deleteChat(chatId);

            // Verify Redis operations
            expect(RedisClientMock.instance?.del).toHaveBeenCalled();

            // Verify the chat and all its messages are deleted
            currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${chatId}`]).toBeUndefined();
            expect(currentData[`message:${message1.id}`]).toBeUndefined();
            expect(currentData[`children:${message1.id}`]).toBeUndefined();
            expect(currentData[`message:${message2.id}`]).toBeUndefined();
            expect(currentData[`children:${message2.id}`]).toBeUndefined();
            expect(currentData[`message:${message3.id}`]).toBeUndefined();
            expect(currentData[`children:${message3.id}`]).toBeUndefined();
        });
        it(`${lmServiceName}: should handle deleting an empty chat`, async () => {
            const emptyChatId = "emptyChat";

            // Delete the empty chat
            await chatContextManager.deleteChat(emptyChatId);

            // Verify Redis operations
            expect(RedisClientMock.instance?.del).toHaveBeenCalledTimes(1); // Only the chat itself should be deleted

            // Verify the empty chat is deleted
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`chat:${emptyChatId}`]).toBeUndefined();
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
            expect(endingData).toEqual(startingData);
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
                    expect(methods).toHaveProperty(method);
                    const count = methods[method];
                    expect(count).toBeGreaterThanOrEqual(0); // Token count should be non-negative
                    expect(count).toBeLessThanOrEqual(translations.find(t => t.language === language)!.text.length); // Token count should not exceed message length
                    expect(Number.isInteger(count)).toBeTruthy(); // Token count should be an integer
                });
            });
        });
        it(`${lmServiceName}: should handle an empty translations array gracefully`, () => {
            const tokenCounts = chatContextManager.calculateTokenCounts([], ...estimationTypes);

            // Expect an empty object when no translations are provided
            expect(tokenCounts).toEqual({});
        });
        it(`${lmServiceName}: should handle undefined or null estimation methods gracefully`, () => {
            // @ts-ignore: Testing runtime scenario
            const tokenCounts = chatContextManager.calculateTokenCounts(translations, undefined, null);

            // Expect token counts to still be calculated using some default method or skipped
            Object.keys(tokenCounts).forEach(language => {
                expect(tokenCounts[language]).toBeDefined();
                expect(Object.keys(tokenCounts[language])).toContain("default"); // Assuming 'default' is a fallback method
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
            redis = await initializeRedis();
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
            expect(contextInfo.length).toBe(3);
            expect(contextInfo.map(info => (info as MessageContextInfo).messageId)).toEqual(expect.arrayContaining(["message1", "message3", "message6"]));
            contextInfo.forEach(info => {
                expect(info.tokenSize).toBe(100); // Based on initialMessageData
            });
        });
        it(`${lmServiceName}: should handle cases with no latest message id provided`, async () => {
            const chatId = "chat1";
            const model = "defaultModel";
            const languages = ["en"];

            const contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages });

            // Since the most recent message is message6, the result should be the same as the previous test
            expect(contextInfo.length).toBe(3);
            expect(contextInfo.map(info => (info as MessageContextInfo).messageId)).toEqual(expect.arrayContaining(["message1", "message3", "message6"]));
            contextInfo.forEach(info => {
                expect(info.tokenSize).toBe(100); // Based on initialMessageData
            });
        });
        it(`${lmServiceName}: should limit the context collection to the context size`, async () => {
            const chatId = "chat1";
            const model = "defaultModel";
            const languages = ["en"];
            const latestMessage = "message5"; // Starting from a deep child

            let contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages, latestMessage });

            // Given the context size of 350 and token count of 100 per message, expect to collect up to 3 messages
            expect(contextInfo.length).toBe(3);

            // Now let's lower the context size to 200 and try again
            jest.spyOn(lmService, "getContextSize").mockReturnValue(200 as any);

            contextInfo = await chatContextCollector.collectMessageContextInfo({ chatId, model, languages, latestMessage });

            // Given the context size of 200 and token count of 100 per message, expect to collect up to 2 messages
            expect(contextInfo.length).toBe(2);
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
            expect(contextInfo.length).toBe(2);
            expect(contextInfo.map(info => (info as MessageContextInfo).messageId)).toEqual(expect.arrayContaining(["message4", "message5"]));
        });

        // Latest message ID
        it(`${lmServiceName}: should return the ID of the most recent message in a chat with messages`, async () => {
            const latestMessageId = await chatContextCollector.getLatestMessageId(redis, "chat1");

            // Expect the latest message ID to be 'message6', as per the initialMessageData
            expect(latestMessageId).toBe("message6");
        });
        it(`${lmServiceName}: should return null for an empty chat`, async () => {
            const latestMessageId = await chatContextCollector.getLatestMessageId(redis, "emptyChat");

            // Expect null for an empty chat
            expect(latestMessageId).toBeNull();
        });
        it(`${lmServiceName}: should return null for a non-existent chat ID`, async () => {
            const latestMessageId = await chatContextCollector.getLatestMessageId(redis, "nonExistentChatId");

            // Expect null for a non-existent chat ID
            expect(latestMessageId).toBeNull();
        });

        // Get message details
        it(`${lmServiceName}: should retrieve message details from Redis if available`, async () => {
            const messageId = "message2"; // An ID from the initial Redis mock data
            const messageDetails = await chatContextCollector.getMessageDetails(redis, messageId);

            expect(messageDetails).toEqual(expect.objectContaining({
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

            expect(messageDetails).toEqual(expect.objectContaining({
                id: messageId,
                parentId: "message3", // Based on Prisma mock data
                translatedTokenCounts: expect.any(Object), // Token counts are calculated and should be an object
                userId: "charlie", // Based on Prisma mock data
            }));

            // Verify Redis now contains the fetched message details
            const redisData = await redis.hGetAll(`message:${messageId}`);
            expect(redisData).toEqual(expect.objectContaining({
                id: messageId,
                parentId: "message3",
                translatedTokenCounts: expect.any(String), // Should be JSON stringified
                userId: "charlie",
            }));
        });
        it(`${lmServiceName}: should return null if message details cannot be fetched from Redis or the database`, async () => {
            const messageId = "nonExistentMessage";
            await expect(chatContextCollector.getMessageDetails(redis, messageId)).resolves.toBeNull();
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

            expect(tokenCount).toBe(expectedTokenCount);
            expect(language).toBe("en");
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

            expect(tokenCount).toBe(expectedTokenCount);
            expect(language).toBe("fr");
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

            expect(tokenCount).toBe(expectedTokenCount);
            expect(language).toBe(fallbackLanguage);
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

            expect(tokenCount).toBe(-1);
            expect(language).toBe("");
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
        expect(processMentions(messageContent, chat, bots)).toEqual([]);
    });

    it("ignores non-mention markdown links", () => {
        const messageContent = "Here's a [@link](http://example.com) that is not a mention.";
        expect(processMentions(messageContent, chat, bots)).toEqual([]);
    });

    it("processes valid mentions correctly", () => {
        const messageContent = `Hello [@Alice](${UI_URL})!`;
        expect(processMentions(messageContent, chat, bots)).toEqual(["bot1"]);
    });

    it("returns an empty array for mentions that do not match any bot", () => {
        const messageContent = `Hello [@Charlie](${UI_URL}/@Charlie)!`;
        expect(processMentions(messageContent, chat, bots)).toEqual([]);
    });

    it("responds to @Everyone mention by returning all bot participants", () => {
        const messageContent = `Attention [@Everyone](${UI_URL}/fdksf?asdfa="fdasfs")!`;
        expect(processMentions(messageContent, chat, bots)).toEqual(["bot1", "bot2", "bot3"]);
    });

    it("filters out mentions with incorrect origin", () => {
        const messageContent = "Hello [@Alice](https://another-site.com/@Alice)!";
        expect(processMentions(messageContent, chat, bots)).toEqual([]);
    });

    it("handles malformed markdown links gracefully", () => {
        const messageContent = "This is a malformed link [@Alice](@Alice) without proper markdown.";
        expect(processMentions(messageContent, chat, bots)).toEqual([]);
    });

    it("deduplicates mentions", () => {
        const messageContent = `Hi [@Alice](${UI_URL}) and [@Alice](${UI_URL}) again!`;
        expect(processMentions(messageContent, chat, bots)).toEqual(["bot1"]);
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
        expect(determineRespondingBots(message, chat, bots, userId)).toEqual([]);
    });

    it("returns an empty array if message userId does not match", () => {
        const message = { userId: "anotherUser", content: "Hello" };
        expect(determineRespondingBots(message, chat, bots, userId)).toEqual([]);
    });

    it("returns an empty array if there are no bots in the chat", () => {
        const message = { userId, content: "Hello" };
        expect(determineRespondingBots(message, chat, [], userId)).toEqual([]);
    });

    it("returns the only bot if there is one bot in a two-participant chat", () => {
        const singleBotChat = { botParticipants: ["bot1"], participantsCount: 2 };
        const singleBot = [{ id: "bot1", name: "BotOne" }];
        const message = { userId, content: "Hello" };
        expect(determineRespondingBots(message, singleBotChat, singleBot, userId)).toEqual(["bot1"]);
    });

    // Assuming processMentions is a mockable function that has been properly mocked to return bot IDs based on mentions
    it("returns bot IDs based on mentions in the message content", () => {
        const message = { userId, content: `Hello [@BotOne](${UI_URL}/@BotOne)` };
        // Mock `processMentions` here to return ['bot1'] when called with message.content, chat, and bots
        // For demonstration purposes, this test assumes `processMentions` behavior is correctly implemented elsewhere
        expect(determineRespondingBots(message, chat, bots, userId)).toEqual(["bot1"]);
    });

    it("handles unexpected input gracefully", () => {
        const message = { userId, content: null }; // Simulate an unexpected null content
        // @ts-ignore: Testing runtime scenario
        expect(() => determineRespondingBots(message, chat, bots, userId)).not.toThrow();
        // @ts-ignore: Testing runtime scenario
        expect(determineRespondingBots(message, chat, bots, userId)).toEqual([]);
    });
});
