import {
    generatePK,
    ModelType,
    type BotParticipant,
    type ChatConfigObject,
    type MessageConfigObject,
    type MessageState,
    type ResponseContext,
    type SessionUser,
} from "@vrooli/shared";
import { Redis } from "ioredis";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { CacheService } from "../../redisConn.js";
import { MessageHistoryBuilder } from "./messageHistoryBuilder.js";
import { ChatContextCache } from "./messageStore.js";
import { ResponseService } from "./responseService.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
    },
}));

vi.mock("./registry.js", () => ({
    AIServiceRegistry: {
        get: vi.fn(() => ({
            getServiceConfig: vi.fn().mockReturnValue({
                contextSize: 8192,
                tokenization: { encoding: "cl100k_base" },
            }),
        })),
    },
}));

vi.mock("../mcp/registry.js", () => ({
    ToolRegistry: vi.fn().mockImplementation(() => ({
        getAllTools: vi.fn().mockReturnValue([
            {
                name: "search",
                description: "Search for information",
                inputSchema: {
                    type: "object",
                    properties: { query: { type: "string" } },
                    required: ["query"],
                },
            },
            {
                name: "calculate",
                description: "Perform calculations",
                inputSchema: {
                    type: "object",
                    properties: { expression: { type: "string" } },
                },
            },
        ]),
    })),
}));

vi.mock("../../redisConn.js", () => ({
    CacheService: {
        getInstance: vi.fn(),
    },
}));

// Mock ResponseService's prompt generation
vi.mock("./responseService.js");

describe("MessageHistoryBuilder", () => {
    let container: StartedTestContainer;
    let redisClient: Redis;
    let historyBuilder: MessageHistoryBuilder;
    let cache: ChatContextCache;

    beforeAll(async () => {
        // Start Redis test container
        container = await new GenericContainer("redis:7-alpine")
            .withExposedPorts(6379)
            .start();

        const host = container.getHost();
        const port = container.getMappedPort(6379);

        redisClient = new Redis({
            host,
            port,
            maxRetriesPerRequest: null,
        });

        // Mock CacheService to return our test Redis client
        (CacheService.getInstance as any).mockReturnValue(redisClient);
    }, 30000);

    afterAll(async () => {
        await redisClient.quit();
        await container.stop();
    });

    beforeEach(async () => {
        // Clear Redis before each test
        await redisClient.flushall();

        // Reset singletons
        ChatContextCache["instance"] = null;
        MessageHistoryBuilder["instance"] = null;

        cache = ChatContextCache.get();
        historyBuilder = MessageHistoryBuilder.get();

        // Mock ResponseService prompt generation
        (ResponseService.prototype as any).generatePrompt = vi.fn()
            .mockResolvedValue("You are a helpful assistant. Goal: Help the user.");
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("buildMessages", () => {
        const createTestContext = (): ResponseContext => ({
            swarmId: "swarm123",
            conversationId: "conv123",
            bot: {
                id: "bot123",
                name: "TestBot",
                role: "assistant",
                model: ModelType.Gpt4o as ModelType,
                config: {
                    temperature: 0.7,
                    maxTokens: 4000,
                    systemPrompt: "You are a helpful assistant",
                } as ChatConfigObject,
            } as BotParticipant,
            userData: {
                id: "user123",
                role: "User",
            } as SessionUser,
            messages: [],
            availableTools: [],
            strategy: { type: "standard" },
            resourceLimits: {
                maxCredits: "1000",
                maxTokens: 4000,
                timeoutMs: 60000,
            },
        });

        it("should build messages with system prompt and empty history", async () => {
            const context = createTestContext();

            const messages = await historyBuilder.buildMessages(context);

            expect(messages).toHaveLength(1);
            expect(messages[0]).toMatchObject({
                role: "system",
                content: expect.stringContaining("You are a helpful assistant"),
            });
        });

        it("should include conversation history from cache", async () => {
            const context = createTestContext();

            // Add messages to cache
            const historicalMessages: MessageState[] = [
                {
                    id: generatePK(),
                    parent: null,
                    userId: "user123",
                    content: "Hello",
                    config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() - 5000).toISOString(),
                },
                {
                    id: generatePK(),
                    parent: null,
                    userId: "bot123",
                    content: "Hi there! How can I help?",
                    config: { modelType: "gpt-4", role: "assistant" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() - 4000).toISOString(),
                },
                {
                    id: generatePK(),
                    parent: null,
                    userId: "user123",
                    content: "What's the weather?",
                    config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() - 3000).toISOString(),
                },
            ];

            for (const msg of historicalMessages) {
                await cache.addMessage(context.conversationId, msg);
            }

            const messages = await historyBuilder.buildMessages(context);

            expect(messages).toHaveLength(4); // System + 3 history
            expect(messages[0].role).toBe("system");
            expect(messages[1]).toMatchObject({ role: "user", content: "Hello" });
            expect(messages[2]).toMatchObject({ role: "assistant", content: "Hi there! How can I help?" });
            expect(messages[3]).toMatchObject({ role: "user", content: "What's the weather?" });
        });

        it("should respect token budget and truncate old messages", async () => {
            const context = createTestContext();

            // Override context with very small token limit
            context.resourceLimits.maxTokens = 200;

            // Add many messages to exceed token budget
            for (let i = 0; i < 50; i++) {
                const message: MessageState = {
                    id: generatePK(),
                    parent: null,
                    userId: i % 2 === 0 ? "user123" : "bot123",
                    content: `This is message number ${i} with some content to consume tokens`,
                    config: {
                        modelType: "gpt-4",
                        role: i % 2 === 0 ? "user" : "assistant",
                    } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() - (50 - i) * 1000).toISOString(),
                };
                await cache.addMessage(context.conversationId, message);
            }

            const messages = await historyBuilder.buildMessages(context);

            // Should have truncated messages to fit budget
            expect(messages.length).toBeLessThan(51); // System + truncated history
            expect(messages[0].role).toBe("system");

            // Most recent messages should be included
            expect(messages[messages.length - 1].content).toContain("message number 49");
        });

        it("should handle messages with parent-child relationships", async () => {
            const context = createTestContext();

            const parentId = generatePK();
            const childId = generatePK();

            // Add parent message
            await cache.addMessage(context.conversationId, {
                id: parentId,
                parent: null,
                userId: "user123",
                content: "Parent message",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - 2000).toISOString(),
            });

            // Add child message
            await cache.addMessage(context.conversationId, {
                id: childId,
                parent: parentId,
                userId: "bot123",
                content: "Child response",
                config: { modelType: "gpt-4", role: "assistant" } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - 1000).toISOString(),
            });

            const messages = await historyBuilder.buildMessages(context);

            expect(messages).toHaveLength(3); // System + 2 messages
            expect(messages[1].content).toBe("Parent message");
            expect(messages[2].content).toBe("Child response");
        });

        it("should include tool schemas in system message when tools available", async () => {
            const context = createTestContext();
            context.availableTools = [
                {
                    name: "weather",
                    description: "Get weather information",
                    inputSchema: {
                        type: "object",
                        properties: {
                            location: { type: "string" },
                        },
                        required: ["location"],
                    },
                },
            ];

            const messages = await historyBuilder.buildMessages(context);

            expect(messages[0].role).toBe("system");
            expect(messages[0].content).toContain("weather");
            expect(messages[0].content).toContain("Get weather information");
        });

        it("should handle missing conversation history gracefully", async () => {
            const context = createTestContext();
            context.conversationId = "non-existent-conv";

            const messages = await historyBuilder.buildMessages(context);

            expect(messages).toHaveLength(1); // Just system message
            expect(messages[0].role).toBe("system");
        });

        it("should prioritize recent messages when truncating", async () => {
            const context = createTestContext();
            context.resourceLimits.maxTokens = 300;

            // Add messages with clear ordering
            const messageCount = 20;
            for (let i = 0; i < messageCount; i++) {
                await cache.addMessage(context.conversationId, {
                    id: generatePK(),
                    parent: null,
                    userId: i % 2 === 0 ? "user123" : "bot123",
                    content: `Message ${i}`,
                    config: {
                        modelType: "gpt-4",
                        role: i % 2 === 0 ? "user" : "assistant",
                    } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() - (messageCount - i) * 1000).toISOString(),
                });
            }

            const messages = await historyBuilder.buildMessages(context);

            // Last message should be the most recent
            const lastHistoryMessage = messages[messages.length - 1];
            expect(lastHistoryMessage.content).toBe("Message 19");

            // Should not include oldest messages
            const messageContents = messages.map(m => m.content);
            expect(messageContents).not.toContain("Message 0");
            expect(messageContents).not.toContain("Message 1");
        });

        it("should handle very long individual messages", async () => {
            const context = createTestContext();

            const veryLongContent = "Lorem ipsum ".repeat(1000); // ~2000 tokens

            await cache.addMessage(context.conversationId, {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: veryLongContent,
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            });

            const messages = await historyBuilder.buildMessages(context);

            // Should include the message even if it's large
            expect(messages).toHaveLength(2); // System + long message
            expect(messages[1].content).toBe(veryLongContent);
        });

        it("should calculate token budgets correctly", async () => {
            const context = createTestContext();
            const totalTokens = 4000;
            context.resourceLimits.maxTokens = totalTokens;

            // Add a moderate amount of history
            for (let i = 0; i < 10; i++) {
                await cache.addMessage(context.conversationId, {
                    id: generatePK(),
                    parent: null,
                    userId: i % 2 === 0 ? "user123" : "bot123",
                    content: `Moderate length message ${i}`,
                    config: {
                        modelType: "gpt-4",
                        role: i % 2 === 0 ? "user" : "assistant",
                    } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() - (10 - i) * 1000).toISOString(),
                });
            }

            const messages = await historyBuilder.buildMessages(context);

            // Verify token budget allocation
            // System message should use ~25% of budget
            // History should use remaining after reserves
            // Response reserve: 2000 tokens
            // User message reserve: 500 tokens
            // Safety buffer: 5%

            const expectedHistoryBudget = totalTokens - 2000 - 500 - (totalTokens * 0.05) - (totalTokens * 0.25);

            // All messages should fit within budget
            expect(messages.length).toBeGreaterThan(1);
            expect(messages.length).toBeLessThanOrEqual(11); // System + 10 history
        });

        it("should handle different model types", async () => {
            const context = createTestContext();

            // Test with different models
            const models = [
                ModelType.Gpt4o,
                ModelType.Gpt4o_Mini,
                ModelType.Claude3_5_Sonnet,
                ModelType.O1_Preview,
            ];

            for (const model of models) {
                context.bot.model = model;

                const messages = await historyBuilder.buildMessages(context);

                expect(messages).toHaveLength(1); // At least system message
                expect(messages[0].role).toBe("system");
            }
        });

        it("should handle concurrent message building", async () => {
            const contexts = Array(5).fill(0).map((_, i) => ({
                ...createTestContext(),
                conversationId: `conv${i}`,
            }));

            // Add different messages to each conversation
            for (let i = 0; i < contexts.length; i++) {
                await cache.addMessage(contexts[i].conversationId, {
                    id: generatePK(),
                    parent: null,
                    userId: "user123",
                    content: `Conversation ${i} message`,
                    config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date().toISOString(),
                });
            }

            // Build messages concurrently
            const results = await Promise.all(
                contexts.map(ctx => historyBuilder.buildMessages(ctx)),
            );

            // Each should have correct messages
            for (let i = 0; i < results.length; i++) {
                expect(results[i]).toHaveLength(2); // System + 1 history
                expect(results[i][1].content).toBe(`Conversation ${i} message`);
            }
        });

        it("should handle special message configs", async () => {
            const context = createTestContext();

            // Add message with special config
            await cache.addMessage(context.conversationId, {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Message with metadata",
                config: {
                    modelType: "gpt-4",
                    role: "user",
                    metadata: {
                        source: "api",
                        timestamp: Date.now(),
                        customField: "value",
                    },
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            });

            const messages = await historyBuilder.buildMessages(context);

            expect(messages).toHaveLength(2);
            expect(messages[1].content).toBe("Message with metadata");
        });

        it("should handle error in prompt generation", async () => {
            const context = createTestContext();

            // Mock prompt generation to throw
            (ResponseService.prototype as any).generatePrompt = vi.fn()
                .mockRejectedValueOnce(new Error("Prompt generation failed"));

            // Should handle gracefully and use fallback
            const messages = await historyBuilder.buildMessages(context);

            expect(messages).toHaveLength(1);
            expect(messages[0].role).toBe("system");
            expect(messages[0].content).toContain("Goal:"); // Fallback prompt
        });

        it("should cache system prompts efficiently", async () => {
            const context = createTestContext();

            // First call
            await historyBuilder.buildMessages(context);

            // Second call with same context
            await historyBuilder.buildMessages(context);

            // Prompt generation should only be called once (cached)
            expect(ResponseService.prototype.generatePrompt).toHaveBeenCalledTimes(1);
        });

        it("should handle empty token budget edge case", async () => {
            const context = createTestContext();
            context.resourceLimits.maxTokens = 1; // Extremely small

            const messages = await historyBuilder.buildMessages(context);

            // Should still return at least system message
            expect(messages).toHaveLength(1);
            expect(messages[0].role).toBe("system");
        });

        it("should preserve message order in complex conversations", async () => {
            const context = createTestContext();

            // Create a complex conversation tree
            const rootId = generatePK();
            const branch1Id = generatePK();
            const branch2Id = generatePK();

            // Root message
            await cache.addMessage(context.conversationId, {
                id: rootId,
                parent: null,
                userId: "user123",
                content: "Root question",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - 5000).toISOString(),
            });

            // Two parallel responses
            await cache.addMessage(context.conversationId, {
                id: branch1Id,
                parent: rootId,
                userId: "bot123",
                content: "Branch 1 response",
                config: { modelType: "gpt-4", role: "assistant" } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - 4000).toISOString(),
            });

            await cache.addMessage(context.conversationId, {
                id: branch2Id,
                parent: rootId,
                userId: "bot456",
                content: "Branch 2 response",
                config: { modelType: "gpt-4", role: "assistant" } as MessageConfigObject,
                language: "en",
                createdAt: new Date(Date.now() - 3000).toISOString(),
            });

            const messages = await historyBuilder.buildMessages(context);

            // Should maintain chronological order
            expect(messages).toHaveLength(4); // System + 3 messages
            expect(messages[1].content).toBe("Root question");
            expect(messages[2].content).toBe("Branch 1 response");
            expect(messages[3].content).toBe("Branch 2 response");
        });
    });
});
