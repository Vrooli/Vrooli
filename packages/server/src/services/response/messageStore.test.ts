import {
    DAYS_1_S,
    generatePK,
    WEEKS_1_DAYS,
    type MessageConfigObject,
    type MessageState,
} from "@vrooli/shared";
import { Redis } from "ioredis";
import { GenericContainer, type StartedTestContainer } from "testcontainers";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { CacheService } from "../../redisConn.js";
import { RedisMessageStore, TokenCounter } from "./messageStore.js";

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

describe("MessageStore (RedisMessageStore)", () => {
    let container: StartedTestContainer;
    let redisClient: Redis;
    let messageStore: RedisMessageStore;

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
        (CacheService.get as any).mockReturnValue(redisClient);
    }, 30000);

    afterAll(async () => {
        await redisClient.quit();
        await container.stop();
    });

    beforeEach(async () => {
        // Clear Redis before each test
        await redisClient.flushall();
        // Reset singleton instance
        RedisMessageStore["_instance"] = null;
        messageStore = RedisMessageStore.get();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("addMessage", () => {
        it("should add a message to the cache", async () => {
            const conversationId = generatePK().toString();
            const message: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "Hello, world!",
                config: {
                    __version: "1.0",
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
            expect(parsedCached.text).toBe(message.text);
        });

        it("should add message to conversation order ZSET", async () => {
            const conversationId = generatePK().toString();
            const message: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "Test message",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            // Check ZSET contains the message
            const messageIds = await redisClient.zrange(`chat:${conversationId}:order`, 0, -1);
            expect(messageIds).toContain(message.id);
        });

        it("should calculate and store token size", async () => {
            const conversationId = generatePK().toString();
            const message: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "This is a test message for token counting",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            const conversationId = generatePK().toString();
            const message: MessageState = {
                id: generatePK().toString(),
                parent: null,
                user: { id: "user123" },
                text: "Expiring message",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const originalMessage: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "Original content",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            // Add original message
            await messageStore.addMessage(conversationId, originalMessage);

            // Update message
            const updatedMessage: MessageState = {
                ...originalMessage,
                text: "Updated content",
            };

            const result = await messageStore.updateMessage(messageId, updatedMessage);

            expect(result.text).toBe("Updated content");

            // Verify update in cache
            const cached = await redisClient.hget(`chat:${conversationId}:messages`, messageId);
            const parsedCached = JSON.parse(cached!);
            expect(parsedCached.text).toBe("Updated content");
        });

        it("should recalculate token size on content update", async () => {
            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const originalMessage: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "Short",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
                text: "This is a much longer message with more tokens",
            };

            await messageStore.updateMessage(messageId, updatedMessage);

            // Verify token size increased
            const updatedCached = await redisClient.hget(`chat:${conversationId}:messages`, messageId);
            const updatedTokenSize = JSON.parse(updatedCached!).tokenSize;
            expect(updatedTokenSize).toBeGreaterThan(originalTokenSize);
        });

        it("should handle updating non-existent message", async () => {
            const nonExistentId = generatePK().toString();
            const message: MessageState = {
                id: nonExistentId,
                parent: null,
                user: { id: "user123" },
                text: "New message",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const message: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "To be deleted",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            const nonExistentId = generatePK().toString();

            // Should not throw
            await expect(messageStore.deleteMessage(nonExistentId))
                .resolves.toBeUndefined();
        });
    });

    describe("getMessage", () => {
        it("should retrieve a cached message", async () => {
            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const message: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "Test retrieval",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            const retrieved = await messageStore.getMessage(messageId);

            expect(retrieved).toBeTruthy();
            expect(retrieved?.id).toBe(messageId);
            expect(retrieved?.text).toBe("Test retrieval");
        });

        it("should return null for non-existent message", async () => {
            const nonExistentId = generatePK().toString();

            const result = await messageStore.getMessage(nonExistentId);

            expect(result).toBeNull();
        });
    });

    describe("getConversationMessages", () => {
        it("should retrieve all messages for a conversation", async () => {
            const conversationId = generatePK().toString();
            const messages: MessageState[] = [];

            // Add multiple messages
            for (let i = 0; i < 5; i++) {
                const message: MessageState = {
                    id: generatePK().toString(),
                    parent: i > 0 ? { id: messages[i - 1].id } : null,
                    user: { id: i % 2 === 0 ? "user123" : "bot123" },
                    text: `Message ${i}`,
                    config: {
                        __version: "1.0",
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
            expect(retrieved[0].text).toBe("Message 0");
            expect(retrieved[4].text).toBe("Message 4");
        });

        it("should respect limit option", async () => {
            const conversationId = generatePK().toString();

            // Add 10 messages
            for (let i = 0; i < 10; i++) {
                const message: MessageState = {
                    id: generatePK().toString(),
                    parent: null,
                    user: { id: "user123" },
                    text: `Message ${i}`,
                    config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                    language: "en",
                    createdAt: new Date(Date.now() + i * 1000).toISOString(),
                };
                await messageStore.addMessage(conversationId, message);
            }

            const retrieved = await messageStore.getConversationMessages(conversationId, { limit: 5 });

            expect(retrieved).toHaveLength(5);
            // Should get the most recent 5
            expect(retrieved[0].text).toBe("Message 5");
            expect(retrieved[4].text).toBe("Message 9");
        });

        it("should handle before/after options", async () => {
            const conversationId = generatePK().toString();
            const messageIds: string[] = [];

            // Add messages with known IDs
            for (let i = 0; i < 10; i++) {
                const messageId = generatePK().toString();
                messageIds.push(messageId);
                const message: MessageState = {
                    id: messageId,
                    parent: null,
                    user: { id: "user123" },
                    text: `Message ${i}`,
                    config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            expect(afterResult[0].text).toBe("Message 4");

            // Get messages before index 7
            const beforeResult = await messageStore.getConversationMessages(
                conversationId,
                { before: messageIds[7] },
            );

            expect(beforeResult.length).toBeGreaterThan(0);
            expect(beforeResult[beforeResult.length - 1].text).toBe("Message 6");
        });

        it("should return empty array for conversation with no messages", async () => {
            const emptyConversationId = generatePK().toString();

            const result = await messageStore.getConversationMessages(emptyConversationId);

            expect(result).toEqual([]);
        });
    });

    describe("getMessageWithTokenCount", () => {
        it("should return message with token count", async () => {
            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const message: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "Calculate my tokens",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, message);

            const result = await messageStore.getMessageWithTokenCount(messageId, "gpt-4");

            expect(result?.message).toBeTruthy();
            expect(result?.message.id).toBe(messageId);
            expect(result?.tokenSize).toBeGreaterThan(0);
        });

        it("should return null for non-existent message", async () => {
            const result = await messageStore.getMessageWithTokenCount(generatePK().toString(), "gpt-4");

            expect(result).toBeNull();
        });
    });

    describe("cache expiration", () => {
        it("should handle expired cache entries", async () => {
            vi.useFakeTimers();

            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const message: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "Will expire",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            const conversationId = generatePK().toString();
            const promises: Promise<MessageState>[] = [];

            // Add 20 messages concurrently
            for (let i = 0; i < 20; i++) {
                const message: MessageState = {
                    id: generatePK().toString(),
                    parent: null,
                    user: { id: `user${i}` },
                    text: `Concurrent message ${i}`,
                    config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
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
            const conversationId = generatePK().toString();
            const messageId = generatePK().toString();
            const originalMessage: MessageState = {
                id: messageId,
                parent: null,
                user: { id: "user123" },
                text: "Original",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            await messageStore.addMessage(conversationId, originalMessage);

            // Update concurrently
            const updates = Array(10).fill(0).map((_, i) =>
                messageStore.updateMessage(messageId, {
                    ...originalMessage,
                    text: `Update ${i}`,
                }),
            );

            await Promise.all(updates);

            // One of the updates should have won
            const final = await messageStore.getMessage(messageId);
            expect(final?.text).toMatch(/Update \d/);
        });
    });
});

describe("TokenCounter", () => {
    let mockService: any;
    let tokenCounter: TokenCounter;

    beforeEach(() => {
        mockService = {
            getEstimationInfo: vi.fn().mockReturnValue({
                estimationModel: "gpt-4",
                encoding: "cl100k_base",
            }),
            estimateTokens: vi.fn().mockReturnValue({ tokens: 5 }),
        };
        tokenCounter = new TokenCounter(mockService, "gpt-4");
    });

    describe("ensure", () => {
        it("should estimate tokens for simple message", () => {
            const message: MessageState = {
                id: "test-id",
                text: "Hello, world!",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const result = tokenCounter.ensure(message, {});

            expect(result.size).toBe(5);
            expect(mockService.estimateTokens).toHaveBeenCalledWith({
                aiModel: "gpt-4",
                text: expect.stringContaining("Hello, world!"),
            });
        });

        it("should cache token counts for same message", () => {
            const message: MessageState = {
                id: "test-id",
                text: "Hello, world!",
                config: { __version: "1.0", modelType: "gpt-4", role: "user" } as MessageConfigObject,
                language: "en",
                createdAt: new Date().toISOString(),
            };

            const counts = { "gpt-4-cl100k_base": 5 };
            const result = tokenCounter.ensure(message, counts);

            expect(result.size).toBe(5);
            expect(mockService.estimateTokens).not.toHaveBeenCalled();
        });

        it("should parse stored token counts", () => {
            const raw = "{\"gpt-4-cl100k_base\":10,\"gpt-3.5-cl100k_base\":8}";
            const parsed = tokenCounter.parse(raw);

            expect(parsed).toEqual({
                "gpt-4-cl100k_base": 10,
                "gpt-3.5-cl100k_base": 8,
            });
        });

        it("should handle invalid JSON gracefully", () => {
            const parsed = tokenCounter.parse("invalid json");

            expect(parsed).toEqual({});
        });

        it("should generate correct key", () => {
            const key = tokenCounter.getKey();

            expect(key).toBe("gpt-4-cl100k_base");
        });
    });
});
