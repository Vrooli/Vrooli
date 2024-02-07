/* eslint-disable @typescript-eslint/ban-ts-comment */
import { RedisClientMock } from "../../__mocks__/redis";
import { PreMapMessageData } from "../../models/base/chatMessage";
import { ChatContextManager } from "./context";
import { OpenAIService } from "./service";

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
        "chat:chat1": [{ score: Date.now(), value: 'message1' }],
    };
    const initialMessageData = {
        'message:message1': {
            id: 'message1',
            userId: 'user1',
            parentId: 'parent1',
            translatedTokenCounts: JSON.stringify({ en: 10 }),
        }
    }
    const initialChildrenData = {
        'children:parent1': ['message1'],
    }
    const initialData = {
        ...JSON.parse(JSON.stringify(initialChatData)),
        ...JSON.parse(JSON.stringify(initialMessageData)),
        ...JSON.parse(JSON.stringify(initialChildrenData)),
    }

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
            const message = { ...message2, isNew: true }
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
                })
            );
            expect(currentData[`children:${message.parentId}`]).toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should refuse to add a message not marked as new`, async () => {
            const message = { ...message2, isNew: false }
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
            const message = { ...message2, isNew: true, parentId: undefined }
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
                })
            );
            expect(currentData[`children:${message.parentId}`]).not.toEqual(expect.arrayContaining([message.id]));
        });

        // Edit message
        it(`${lmServiceName}: should edit an existing message in Redis`, async () => {
            const message = {
                ...initialMessageData['message:message1'],
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
                })
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
            const originalParentId = initialMessageData['message:message1'].parentId;
            const originalChildren = RedisClientMock.__getAllMockData()[`children:${originalParentId}`];
            expect(originalChildren).toContain("message1");

            const message: PreMapMessageData = {
                ...initialMessageData['message:message1'],
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
                })
            );
            // Message should be in new parent's children
            expect(currentData[`children:${message.parentId}`]).toEqual(expect.arrayContaining([message.id]));
            // Message should be removed from original parent's children
            expect(currentData[`children:${originalParentId}`]).not.toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should still work when existing message does not exist`, async () => {
            const message: PreMapMessageData = {
                ...initialMessageData['message:message1'],
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
                })
            );
            expect(currentData[`children:${message.parentId}`]).toEqual(expect.arrayContaining([message.id]));
        });
        it(`${lmServiceName}: should edit message translations and update token counts`, async () => {
            const messageWithEditedTranslations: PreMapMessageData = {
                ...initialMessageData['message:message1'],
                chatId: "chat1",
                content: "Edited translation",
                isNew: false,
                language: "en",
                translations: [{ language: "en", text: "Edited translation" }], // Updated translations
            };

            // Mock `calculateTokenCounts` to return a specific value
            const mockedTokenCounts = { en: { 'default': -420 } };// Negative to ensure that original was never conincidentally this exact value
            jest.spyOn(chatContextManager, "calculateTokenCounts").mockReturnValue(mockedTokenCounts);

            await chatContextManager.editMessage(messageWithEditedTranslations);

            // Verify Redis operations
            expect(RedisClientMock.instance?.hSet).toHaveBeenCalled();

            // Verify Redis data
            const currentData = RedisClientMock.__getAllMockData();
            expect(currentData[`message:${messageWithEditedTranslations.id}`]).toEqual(
                expect.objectContaining({
                    translatedTokenCounts: JSON.stringify(mockedTokenCounts),
                })
            );
        });
        it(`${lmServiceName}: should recover from invalid existing data - bad message`, async () => {
            const message: PreMapMessageData = {
                ...initialMessageData['message:message1'],
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
                })
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
            const message1 = initialMessageData['message:message1'];

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
            const message1 = initialMessageData['message:message1'];

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
            { language: "es", text: "Hola mundo" }
        ];
        const estimationTypes = lmService.getEstimationTypes();
        const tokenEstimationMock = (message: string, method: string | null | undefined) => ([method ?? "default", message.split(" ").length] as readonly [any, number])
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
                expect(Object.keys(tokenCounts[language])).toContain('default'); // Assuming 'default' is a fallback method
            });
        });
    });
});
