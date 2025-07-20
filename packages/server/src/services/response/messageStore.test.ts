import { describe, it, expect, vi, beforeEach, afterEach, beforeAll, afterAll } from "vitest";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { Redis } from "ioredis";
import { 
    type MessageState, 
    type MessageConfigObject,
    generatePK,
    WEEKS_1_DAYS,
    DAYS_1_S,
} from "@vrooli/shared";
import { CacheService } from "../../redisConn.js";
import { ChatContextCache, TokenCounter } from "./messageStore.js";
import { AIServiceRegistry } from "./registry.js";
import { logger } from "../../events/logger.js";

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

// Mock CacheService to use test container Redis
vi.mock("../../redisConn.js", () => ({
    CacheService: {
        getInstance: vi.fn(),
    },
}));

describe("MessageStore (ChatContextCache)", () => {
    let container: StartedTestContainer;
    let redisClient: Redis;
    let messageStore: ChatContextCache;

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
        // Reset singleton instance
        ChatContextCache["instance"] = null;
        messageStore = ChatContextCache.get();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("addMessage", () => {
        it("should add a message to the cache", async () => {
            const conversationId = generatePK();
            const message: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Hello, world!",
                config: {
                    modelType: "gpt-4",
                    role: "user",
                } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = await messageStore.addMessage(conversationId, message);

            expect(result).toEqual(message);

            // Verify message was added to cache
            const cached = await redisClient.hget(`chat:${conversationId}:messages`, message.id);
            expect(cached).toBeTruthy();
            const parsedCached = JSON.parse(cached!);
            expect(parsedCached.id).toBe(message.id);
            expect(parsedCached.content).toBe(message.content);
        });

        it("should add message to conversation order ZSET", async () => {
            const conversationId = generatePK();
            const message: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Test message",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            // Check ZSET contains the message
            const messageIds = await redisClient.zrange(`chat:${conversationId}:order`, 0, -1);
            expect(messageIds).toContain(message.id);
        });

        it("should calculate and store token size", async () => {
            const conversationId = generatePK();
            const message: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "This is a test message for token counting",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            // Verify token size was calculated
            const cached = await redisClient.hget(`chat:${conversationId}:messages`, message.id);
            const parsedCached = JSON.parse(cached!);
            expect(parsedCached.tokenSize).toBeGreaterThan(0);
        });

        it("should set TTL on cache entries", async () => {
            const conversationId = generatePK();
            const message: MessageState = {
                id: generatePK(),
                parent: null,
                userId: "user123",
                content: "Expiring message",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            // Check TTL was set
            const ttl = await redisClient.ttl(`chat:${conversationId}:messages`);
            expect(ttl).toBeGreaterThan(0);
            expect(ttl).toBeLessThanOrEqual(WEEKS_1_DAYS * DAYS_1_S);
        });
    });

    describe("updateMessage", () => {
        it("should update an existing message", async () => {
            const conversationId = generatePK();
            const messageId = generatePK();
            const originalMessage: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "Original content",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            // Add original message
            await messageStore.addMessage(conversationId, originalMessage);

            // Update message
            const updatedMessage: MessageState = {
                ...originalMessage,
                content: "Updated content",
            };

            const result = await messageStore.updateMessage(messageId, updatedMessage);

            expect(result.content).toBe("Updated content");

            // Verify update in cache
            const cached = await redisClient.hget(`chat:${conversationId}:messages`, messageId);
            const parsedCached = JSON.parse(cached!);
            expect(parsedCached.content).toBe("Updated content");
        });

        it("should recalculate token size on content update", async () => {
            const conversationId = generatePK();
            const messageId = generatePK();
            const originalMessage: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "Short",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, originalMessage);

            // Get original token size
            const originalCached = await redisClient.hget(`chat:${conversationId}:messages`, messageId);
            const originalTokenSize = JSON.parse(originalCached!).tokenSize;

            // Update with longer content
            const updatedMessage: MessageState = {
                ...originalMessage,
                content: "This is a much longer message with more tokens",
            };

            await messageStore.updateMessage(messageId, updatedMessage);

            // Verify token size increased
            const updatedCached = await redisClient.hget(`chat:${conversationId}:messages`, messageId);
            const updatedTokenSize = JSON.parse(updatedCached!).tokenSize;
            expect(updatedTokenSize).toBeGreaterThan(originalTokenSize);
        });

        it("should handle updating non-existent message", async () => {
            const nonExistentId = generatePK();
            const message: MessageState = {
                id: nonExistentId,
                parent: null,
                userId: "user123",
                content: "New message",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            // Should handle gracefully (might create new entry or throw)
            await expect(messageStore.updateMessage(nonExistentId, message))
                .resolves.toBeTruthy();
        });
    });

    describe("deleteMessage", () => {
        it("should delete a message from cache", async () => {
            const conversationId = generatePK();
            const messageId = generatePK();
            const message: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "To be deleted",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            // Add message
            await messageStore.addMessage(conversationId, message);

            // Delete message
            await messageStore.deleteMessage(messageId);

            // Verify deletion
            const cached = await redisClient.hget(`chat:${conversationId}:messages`, messageId);
            expect(cached).toBeNull();

            // Verify removed from order ZSET
            const messageIds = await redisClient.zrange(`chat:${conversationId}:order`, 0, -1);
            expect(messageIds).not.toContain(messageId);
        });

        it("should handle deleting non-existent message", async () => {
            const nonExistentId = generatePK();
            
            // Should not throw
            await expect(messageStore.deleteMessage(nonExistentId))
                .resolves.toBeUndefined();
        });
    });

    describe("getMessage", () => {
        it("should retrieve a cached message", async () => {
            const conversationId = generatePK();
            const messageId = generatePK();
            const message: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "Test retrieval",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            const retrieved = await messageStore.getMessage(messageId);

            expect(retrieved).toBeTruthy();
            expect(retrieved?.id).toBe(messageId);
            expect(retrieved?.content).toBe("Test retrieval");
        });

        it("should return null for non-existent message", async () => {
            const nonExistentId = generatePK();
            
            const result = await messageStore.getMessage(nonExistentId);
            
            expect(result).toBeNull();
        });
    });

    describe("getConversationMessages", () => {
        it("should retrieve all messages for a conversation", async () => {
            const conversationId = generatePK();
            const messages: MessageState[] = [];

            // Add multiple messages
            for (let i = 0; i < 5; i++) {
                const message: MessageState = {
                    id: generatePK(),
                    parent: i > 0 ? messages[i - 1].id : null,
                    userId: i % 2 === 0 ? "user123" : "bot123",
                    content: `Message ${i}`,
                    config: { 
                        modelType: "gpt-4", 
                        role: i % 2 === 0 ? "user" : "assistant",
                    } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() + i * 1000).toISOString(),
                };
                messages.push(message);
                await messageStore.addMessage(conversationId, message);
            }

            const retrieved = await messageStore.getConversationMessages(conversationId);

            expect(retrieved).toHaveLength(5);
            // Should be ordered by creation time
            expect(retrieved[0].content).toBe("Message 0");
            expect(retrieved[4].content).toBe("Message 4");
        });

        it("should respect limit option", async () => {
            const conversationId = generatePK();

            // Add 10 messages
            for (let i = 0; i < 10; i++) {
                const message: MessageState = {
                    id: generatePK(),
                    parent: null,
                    userId: "user123",
                    content: `Message ${i}`,
                    config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() + i * 1000).toISOString(),
                };
                await messageStore.addMessage(conversationId, message);
            }

            const retrieved = await messageStore.getConversationMessages(conversationId, { limit: 5 });

            expect(retrieved).toHaveLength(5);
            // Should get the most recent 5
            expect(retrieved[0].content).toBe("Message 5");
            expect(retrieved[4].content).toBe("Message 9");
        });

        it("should handle before/after options", async () => {
            const conversationId = generatePK();
            const messageIds: string[] = [];

            // Add messages with known IDs
            for (let i = 0; i < 10; i++) {
                const messageId = generatePK();
                messageIds.push(messageId);
                const message: MessageState = {
                    id: messageId,
                    parent: null,
                    userId: "user123",
                    content: `Message ${i}`,
                    config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() + i * 1000).toISOString(),
                };
                await messageStore.addMessage(conversationId, message);
            }

            // Get messages after index 3
            const afterResult = await messageStore.getConversationMessages(
                conversationId, 
                { after: messageIds[3] },
            );

            expect(afterResult.length).toBeGreaterThan(0);
            expect(afterResult[0].content).toBe("Message 4");

            // Get messages before index 7
            const beforeResult = await messageStore.getConversationMessages(
                conversationId,
                { before: messageIds[7] },
            );

            expect(beforeResult.length).toBeGreaterThan(0);
            expect(beforeResult[beforeResult.length - 1].content).toBe("Message 6");
        });

        it("should return empty array for conversation with no messages", async () => {
            const emptyConversationId = generatePK();
            
            const result = await messageStore.getConversationMessages(emptyConversationId);
            
            expect(result).toEqual([]);
        });
    });

    describe("getMessageWithSize", () => {
        it("should return message with token count", async () => {
            const conversationId = generatePK();
            const messageId = generatePK();
            const message: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "Calculate my tokens",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            const result = await messageStore.getMessageWithSize(messageId, "gpt-4");

            expect(result?.message).toBeTruthy();
            expect(result?.message.id).toBe(messageId);
            expect(result?.tokenSize).toBeGreaterThan(0);
        });

        it("should return null for non-existent message", async () => {
            const result = await messageStore.getMessageWithSize(generatePK(), "gpt-4");
            
            expect(result).toBeNull();
        });
    });

    describe("cache expiration", () => {
        it("should handle expired cache entries", async () => {
            vi.useFakeTimers();
            
            const conversationId = generatePK();
            const messageId = generatePK();
            const message: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "Will expire",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            // Fast forward past TTL
            vi.advanceTimersByTime((WEEKS_1_DAYS * DAYS_1_S + 1) * 1000);

            // Force Redis to check expiration (in real Redis this happens automatically)
            await redisClient.expire(`chat:${conversationId}:messages`, -1);

            const result = await messageStore.getMessage(messageId);
            expect(result).toBeNull();

            vi.useRealTimers();
        });
    });

    describe("concurrent operations", () => {
        it("should handle concurrent message additions", async () => {
            const conversationId = generatePK();
            const promises: Promise<MessageState>[] = [];

            // Add 20 messages concurrently
            for (let i = 0; i < 20; i++) {
                const message: MessageState = {
                    id: generatePK(),
                    parent: null,
                    userId: `user${i}`,
                    content: `Concurrent message ${i}`,
                    config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() + i).toISOString(),
                };
                promises.push(messageStore.addMessage(conversationId, message));
            }

            const results = await Promise.all(promises);

            expect(results).toHaveLength(20);

            // Verify all messages were added
            const retrieved = await messageStore.getConversationMessages(conversationId);
            expect(retrieved).toHaveLength(20);
        });

        it("should handle concurrent updates to same message", async () => {
            const conversationId = generatePK();
            const messageId = generatePK();
            const originalMessage: MessageState = {
                id: messageId,
                parent: null,
                userId: "user123",
                content: "Original",
                config: { modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, originalMessage);

            // Update concurrently
            const updates = Array(10).fill(0).map((_, i) => 
                messageStore.updateMessage(messageId, {
                    ...originalMessage,
                    content: `Update ${i}`,
                }),
            );

            await Promise.all(updates);

            // One of the updates should have won
            const final = await messageStore.getMessage(messageId);
            expect(final?.content).toMatch(/Update \d/);
        });
    });
});

describe("TokenCounter", () => {
    let tokenCounter: TokenCounter;

    beforeEach(() => {
        tokenCounter = TokenCounter.get();
    });

    describe("estimateTokens", () => {
        it("should estimate tokens for simple text", () => {
            const tokens = tokenCounter.estimateTokens("Hello, world!", "gpt-4");
            
            expect(tokens).toBeGreaterThan(0);
            expect(tokens).toBeLessThan(10); // Simple greeting should be < 10 tokens
        });

        it("should handle empty string", () => {
            const tokens = tokenCounter.estimateTokens("", "gpt-4");
            
            expect(tokens).toBe(0);
        });

        it("should handle long text", () => {
            const longText = "Lorem ipsum ".repeat(1000);
            const tokens = tokenCounter.estimateTokens(longText, "gpt-4");
            
            expect(tokens).toBeGreaterThan(1000);
        });

        it("should handle special characters and emojis", () => {
            const text = "Hello 👋 World! 🌍 Special chars: @#$%^&*()";
            const tokens = tokenCounter.estimateTokens(text, "gpt-4");
            
            expect(tokens).toBeGreaterThan(0);
        });

        it("should handle different models", () => {
            const text = "Test message for different models";
            
            const gpt4Tokens = tokenCounter.estimateTokens(text, "gpt-4");
            const gpt35Tokens = tokenCounter.estimateTokens(text, "gpt-3.5-turbo");
            
            // Both should return reasonable estimates
            expect(gpt4Tokens).toBeGreaterThan(0);
            expect(gpt35Tokens).toBeGreaterThan(0);
        });

        it("should handle code snippets", () => {
            const code = `
                function fibonacci(n) {
                    if (n <= 1) return n;
                    return fibonacci(n - 1) + fibonacci(n - 2);
                }
            `;
            
            const tokens = tokenCounter.estimateTokens(code, "gpt-4");
            
            expect(tokens).toBeGreaterThan(20); // Code should use more tokens
        });

        it("should handle multi-language text", () => {
            const multiLang = "Hello 你好 Bonjour こんにちは مرحبا";
            const tokens = tokenCounter.estimateTokens(multiLang, "gpt-4");
            
            expect(tokens).toBeGreaterThan(5);
        });
    });

    describe("token estimation accuracy", () => {
        it("should provide consistent estimates for same text", () => {
            const text = "The quick brown fox jumps over the lazy dog";
            
            const estimate1 = tokenCounter.estimateTokens(text, "gpt-4");
            const estimate2 = tokenCounter.estimateTokens(text, "gpt-4");
            
            expect(estimate1).toBe(estimate2);
        });

        it("should scale linearly for repeated text", () => {
            const baseText = "Test message. ";
            
            const single = tokenCounter.estimateTokens(baseText, "gpt-4");
            const double = tokenCounter.estimateTokens(baseText.repeat(2), "gpt-4");
            const triple = tokenCounter.estimateTokens(baseText.repeat(3), "gpt-4");
            
            // Should scale approximately linearly
            expect(double).toBeGreaterThan(single * 1.8);
            expect(double).toBeLessThan(single * 2.2);
            expect(triple).toBeGreaterThan(single * 2.8);
            expect(triple).toBeLessThan(single * 3.2);
        });
    });
});
