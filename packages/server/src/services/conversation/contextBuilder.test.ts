/* eslint-disable @typescript-eslint/ban-ts-comment */
import { TaskContextInfo } from "@local/shared";
import { expect } from "chai";
import { RedisClientType } from "redis";
import sinon from "sinon";
import { initializeRedis } from "../../redisConn.js";
import { UI_URL } from "../../server.js";
import { OpenAIService } from "../../tasks/llm/services/openai.js";
import { PreMapMessageDataDelete, PreMapMessageDataUpdate } from "../../utils/chat.js";
import { ChatContextManager, determineRespondingBots, processMentions, stringifyTaskContexts } from "./contextBuilder.js";

describe("ChatContextManager", () => {
    let chatContextManager: ChatContextManager;
    let redis: RedisClientType;
    const sandbox = sinon.createSandbox();

    // Replace direct instantiation with stubbed versions
    const openAIService = sinon.createStubInstance(OpenAIService);

    const lmServices = [
        { name: "OpenAIService", instance: openAIService },
        // Add other implementations as needed
    ];

    // Test data
    const testChat = {
        id: "chat-test-1",
        name: "Test Chat",
    };

    const testMessage1 = {
        chatId: testChat.id,
        messageId: "msg-test-1",
        parentId: null,
        content: "First test message",
        language: "en",
        text: "First test message",
        userId: "user-test-1",
    };

    const testMessage2 = {
        chatId: testChat.id,
        messageId: "msg-test-2",
        parentId: testMessage1.messageId,
        content: "Second test message",
        language: "en",
        text: "Second test message",
        userId: "user-test-1",
    };

    beforeEach(async () => {
        redis = await initializeRedis() as RedisClientType;
        sandbox.restore();

        // Clear Redis for this test chat
        await redis.del(`chat:${testChat.id}`);
        await redis.del(`message:${testMessage1.messageId}`);
        await redis.del(`message:${testMessage2.messageId}`);
        await redis.del(`children:${testMessage1.messageId}`);
    });

    afterEach(async () => {
        sandbox.restore();

        // Clean up Redis after tests
        await redis.del(`chat:${testChat.id}`);
        await redis.del(`message:${testMessage1.messageId}`);
        await redis.del(`message:${testMessage2.messageId}`);
        await redis.del(`children:${testMessage1.messageId}`);
    });

    lmServices.forEach(({ name: lmServiceName, instance: lmService }) => {
        describe(`${lmServiceName}`, () => {
            beforeEach(async () => {
                // Create a ChatContextManager with the language model service
                chatContextManager = new ChatContextManager("defaultModel");
            });

            // Add message
            it("should add a new message to Redis", async () => {
                const message = { __type: "Create", ...testMessage1 } as const;
                await chatContextManager.addMessage(message);

                // Verify Redis data
                const chatMessages = await redis.zRange(`chat:${message.chatId}`, 0, -1);
                expect(chatMessages).to.include(message.messageId);

                const messageData = await redis.hGetAll(`message:${message.messageId}`);
                expect(messageData).to.have.property("id", message.messageId);
                expect(messageData).to.have.property("userId", message.userId);
                expect(messageData).to.have.property("translatedTokenCounts");
                expect(JSON.parse(messageData.translatedTokenCounts)).to.be.an("object");
            });

            it("should add a message with a parent to Redis", async () => {
                // First add parent message
                const parentMessage = { __type: "Create", ...testMessage1 } as const;
                await chatContextManager.addMessage(parentMessage);

                // Add child message
                const childMessage = { __type: "Create", ...testMessage2 } as const;
                await chatContextManager.addMessage(childMessage);

                // Verify parent-child relationship in Redis
                const children = await redis.sMembers(`children:${childMessage.parentId}`);
                expect(children).to.include(childMessage.messageId);

                const messageData = await redis.hGetAll(`message:${childMessage.messageId}`);
                expect(messageData).to.have.property("parentId", childMessage.parentId);
            });

            // Edit message
            it("should edit an existing message in Redis", async () => {
                // First add a message
                await chatContextManager.addMessage({ __type: "Create", ...testMessage1 } as const);

                // Edit the message
                const editedMessage: PreMapMessageDataUpdate = {
                    __type: "Update",
                    chatId: testMessage1.chatId,
                    messageId: testMessage1.messageId,
                    text: "Edited first message",
                    userId: testMessage1.userId,
                    parentId: null,
                };

                await chatContextManager.editMessage(editedMessage);

                // Verify Redis data
                const messageData = await redis.hGetAll(`message:${testMessage1.messageId}`);
                expect(messageData).to.have.property("id", testMessage1.messageId);
                expect(messageData).to.have.property("userId", testMessage1.userId);
                expect(messageData).to.have.property("translatedTokenCounts");
            });

            it("should edit a message and update parent ID", async () => {
                // Setup: Add two parent messages and a child message
                const parentMsg1 = { ...testMessage1, messageId: "parent1", parentId: null };
                const parentMsg2 = { ...testMessage1, messageId: "parent2", parentId: null };
                const childMsg = { ...testMessage2, messageId: "child1", parentId: "parent1" };

                await chatContextManager.addMessage({ __type: "Create", ...parentMsg1 } as const);
                await chatContextManager.addMessage({ __type: "Create", ...parentMsg2 } as const);
                await chatContextManager.addMessage({ __type: "Create", ...childMsg } as const);

                // Verify initial parent-child relationship
                let children = await redis.sMembers(`children:${childMsg.parentId}`);
                expect(children).to.include(childMsg.messageId);

                // Change parent
                const updatedMsg: PreMapMessageDataUpdate = {
                    __type: "Update",
                    chatId: childMsg.chatId,
                    messageId: childMsg.messageId,
                    parentId: "parent2",
                    text: childMsg.text,
                    userId: childMsg.userId,
                };

                await chatContextManager.editMessage(updatedMsg);

                // Verify updated parent-child relationship
                children = await redis.sMembers(`children:${childMsg.parentId}`);
                expect(children).to.not.include(childMsg.messageId);

                children = await redis.sMembers(`children:${updatedMsg.parentId}`);
                expect(children).to.include(childMsg.messageId);

                const messageData = await redis.hGetAll(`message:${childMsg.messageId}`);
                expect(messageData).to.have.property("parentId", "parent2");
            });

            // Delete message
            it("should delete a message and its references from Redis", async () => {
                // First add a message
                await chatContextManager.addMessage({ __type: "Create", ...testMessage1 } as const);

                // Delete the message
                const deleteMessage: PreMapMessageDataDelete = {
                    __type: "Delete",
                    chatId: testMessage1.chatId,
                    messageId: testMessage1.messageId,
                };

                await chatContextManager.deleteMessage(deleteMessage);

                // Verify message is deleted
                const messageData = await redis.hGetAll(`message:${testMessage1.messageId}`);
                expect(Object.keys(messageData).length).to.equal(0);

                const chatMessages = await redis.zRange(`chat:${testMessage1.chatId}`, 0, -1);
                expect(chatMessages).to.not.include(testMessage1.messageId);
            });

            it("should update children's parent references when deleting a message with children", async () => {
                // Setup: Add parent with a parent, and two children
                const grandparent = { ...testMessage1, messageId: "grandparent", parentId: null };
                const parent = { ...testMessage1, messageId: "parent", parentId: "grandparent" };
                const child1 = { ...testMessage2, messageId: "child1", parentId: "parent" };
                const child2 = { ...testMessage2, messageId: "child2", parentId: "parent" };

                await chatContextManager.addMessage({ __type: "Create", ...grandparent } as const);
                await chatContextManager.addMessage({ __type: "Create", ...parent } as const);
                await chatContextManager.addMessage({ __type: "Create", ...child1 } as const);
                await chatContextManager.addMessage({ __type: "Create", ...child2 } as const);

                // Delete the parent
                await chatContextManager.deleteMessage({
                    __type: "Delete",
                    chatId: parent.chatId,
                    messageId: parent.messageId,
                });

                // Verify children now have grandparent as parent
                const child1Data = await redis.hGetAll(`message:${child1.messageId}`);
                expect(child1Data).to.have.property("parentId", grandparent.messageId);

                const child2Data = await redis.hGetAll(`message:${child2.messageId}`);
                expect(child2Data).to.have.property("parentId", grandparent.messageId);

                // Verify grandparent now has both children
                const grandparentChildren = await redis.sMembers(`children:${grandparent.messageId}`);
                expect(grandparentChildren).to.include.members([child1.messageId, child2.messageId]);
            });

            // Delete chat
            it("should delete a chat and all its messages", async () => {
                // Setup: Add messages to chat
                await chatContextManager.addMessage({ __type: "Create", ...testMessage1 } as const);
                await chatContextManager.addMessage({ __type: "Create", ...testMessage2 } as const);

                // Delete the chat
                await chatContextManager.deleteChat(testChat.id);

                // Verify chat and messages are deleted
                const chatMessages = await redis.zRange(`chat:${testChat.id}`, 0, -1);
                expect(chatMessages.length).to.equal(0);

                const message1Data = await redis.hGetAll(`message:${testMessage1.messageId}`);
                expect(Object.keys(message1Data).length).to.equal(0);

                const message2Data = await redis.hGetAll(`message:${testMessage2.messageId}`);
                expect(Object.keys(message2Data).length).to.equal(0);
            });
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
        const message = "   ";
        const messageFromUserId = userId;
        expect(determineRespondingBots(message, messageFromUserId, chat, bots, userId)).to.deep.equal([]);
    });

    it("returns an empty array if message userId does not match", () => {
        const message = "Hello";
        const messageFromUserId = "anotherUser";
        expect(determineRespondingBots(message, messageFromUserId, chat, bots, userId)).to.deep.equal([]);
    });

    it("returns an empty array if there are no bots in the chat", () => {
        const message = "Hello";
        const messageFromUserId = userId;
        expect(determineRespondingBots(message, messageFromUserId, chat, [], userId)).to.deep.equal([]);
    });

    it("returns the only bot if there is one bot in a two-participant chat", () => {
        const singleBotChat = { botParticipants: ["bot1"], participantsCount: 2 };
        const singleBot = [{ id: "bot1", name: "BotOne" }];
        const message = "Hello";
        const messageFromUserId = userId;
        expect(determineRespondingBots(message, messageFromUserId, singleBotChat, singleBot, userId)).to.deep.equal(["bot1"]);
    });

    it("returns bot IDs based on mentions in the message content", () => {
        const message = `Hello [@BotOne](${UI_URL}/@BotOne)`;
        const messageFromUserId = userId;
        expect(determineRespondingBots(message, messageFromUserId, chat, bots, userId)).to.deep.equal(["bot1"]);
    });

    it("handles unexpected input gracefully", () => {
        const message = null; // Simulate an unexpected null content
        const messageFromUserId = userId;
        // @ts-ignore: Testing runtime scenario
        expect(() => determineRespondingBots(message, messageFromUserId, chat, bots, userId)).to.not.throw();
        // @ts-ignore: Testing runtime scenario
        expect(determineRespondingBots(message, messageFromUserId, chat, bots, userId)).to.deep.equal([]);
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
});
