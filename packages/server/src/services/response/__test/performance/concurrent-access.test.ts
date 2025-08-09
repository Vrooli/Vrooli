import {
    type BotParticipant,
    type ChatConfigObject,
    type ConversationState,
    type MessageState,
    generatePK,
    LRUCache,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DbProvider } from "../../../../db/provider.js";
import { CacheService } from "../../../../redisConn.js";
import { DbChatStore } from "../../chatStore.js";
import { RedisMessageStore } from "../../messageStore.js";
import { OpenAIService } from "../../services.js";

// Mock dependencies
vi.mock("../../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
        trace: vi.fn(),
    },
}));

vi.mock("../../../../db/provider.js");
vi.mock("../../../../redisConn.js");

// Mock OpenAI to avoid real API calls
vi.mock("openai", () => ({
    default: vi.fn().mockImplementation(() => ({
        moderations: {
            create: vi.fn().mockResolvedValue({
                results: [{ flagged: false }],
            }),
        },
        chat: {
            completions: {
                create: vi.fn(),
            },
        },
    })),
}));

describe("Concurrent Access and Performance Tests", () => {
    let chatStore: DbChatStore;
    let messageStore: MessageStore;
    let mockDb: any;
    let mockCache: any;
    let mockL1Cache: LRUCache<string, ConversationState>;

    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();

        // Mock database
        mockDb = {
            transaction: vi.fn((fn: any) => fn({
                chat: {
                    update: vi.fn().mockResolvedValue({}),
                    findUniqueOrThrow: vi.fn(),
                },
                chat_participants: {
                    findMany: vi.fn().mockResolvedValue([]),
                },
            })),
            chat: {
                findUniqueOrThrow: vi.fn(),
                update: vi.fn(),
            },
            chat_participants: {
                findMany: vi.fn().mockResolvedValue([]),
            },
        };
        (DbProvider.get as any).mockReturnValue(mockDb);

        // Mock cache
        mockCache = {
            getFromCache: vi.fn().mockResolvedValue(null),
            saveToCache: vi.fn().mockResolvedValue(undefined),
            deleteFromCache: vi.fn().mockResolvedValue(undefined),
        };
        (CacheService.get as any).mockReturnValue(mockCache);

        // Initialize services
        mockL1Cache = new LRUCache<string, ConversationState>(100);
        chatStore = new DbChatStore(50, mockL1Cache); // Short debounce for testing
        messageStore = RedisMessageStore.get();
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    describe("ChatStore Concurrent State Updates", () => {
        it("should handle rapid state mutations from multiple sources", async () => {
            const conversationId = "chat_concurrent_1";
            const initialState: ConversationState = {
                id: conversationId,
                participants: [],
                status: "in_progress",
                config: {
                    __version: "1.0",
                    model: "gpt-4o",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        startedAt: null,
                        lastProcessingCycleEndedAt: null,
                    },
                } as ChatConfigObject,
            };

            // Mock fetch to return initial state
            mockDb.chat.findUniqueOrThrow.mockResolvedValue({
                id: conversationId,
                config: initialState.config,
            });

            // Simulate 10 concurrent updates
            const updatePromises = [];
            for (let i = 0; i < 10; i++) {
                updatePromises.push(
                    (async () => {
                        const state = await chatStore.loadState(conversationId);
                        state.config.stats.totalToolCalls += 1;
                        state.config.stats.totalCredits = (parseInt(state.config.stats.totalCredits) + i * 10).toString();
                        chatStore.saveState(conversationId, state);
                    })(),
                );
            }

            await Promise.all(updatePromises);

            // Advance timers to trigger debounced write
            await vi.advanceTimersByTimeAsync(100);

            // Should batch all updates into a single write
            expect(mockDb.transaction).toHaveBeenCalledTimes(1);

            // Final state should reflect all updates
            const finalCall = mockDb.transaction.mock.calls[0][0];
            const mockTx = {
                chat: {
                    update: vi.fn(),
                },
            };
            await finalCall(mockTx);

            expect(mockTx.chat.update).toHaveBeenCalledWith({
                where: { id: conversationId },
                data: expect.objectContaining({
                    config: expect.objectContaining({
                        stats: expect.objectContaining({
                            totalToolCalls: 10,
                            totalCredits: "450", // Sum of 0+10+20+...+90
                        }),
                    }),
                }),
            });
        });

        it("should handle interleaved reads and writes correctly", async () => {
            const conversationId = "chat_interleaved";
            const initialState: ConversationState = {
                id: conversationId,
                participants: [],
                status: "in_progress",
                config: {
                    __version: "1.0",
                    model: "gpt-4o",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "100",
                        startedAt: null,
                        lastProcessingCycleEndedAt: null,
                    },
                } as ChatConfigObject,
            };

            mockDb.chat.findUniqueOrThrow.mockResolvedValue({
                id: conversationId,
                config: initialState.config,
            });

            // Simulate interleaved operations
            const operations = [];

            // Writer 1: Increments tool calls
            operations.push(
                (async () => {
                    for (let i = 0; i < 5; i++) {
                        const state = await chatStore.loadState(conversationId);
                        state.config.stats.totalToolCalls += 1;
                        chatStore.saveState(conversationId, state);
                        await new Promise(resolve => setTimeout(resolve, 10));
                    }
                })(),
            );

            // Reader: Periodically reads state
            const readResults: number[] = [];
            operations.push(
                (async () => {
                    for (let i = 0; i < 10; i++) {
                        const state = await chatStore.loadState(conversationId);
                        readResults.push(state.config.stats.totalToolCalls);
                        await new Promise(resolve => setTimeout(resolve, 5));
                    }
                })(),
            );

            // Writer 2: Increments credits
            operations.push(
                (async () => {
                    for (let i = 0; i < 3; i++) {
                        const state = await chatStore.loadState(conversationId);
                        state.config.stats.totalCredits = (parseInt(state.config.stats.totalCredits) + 50).toString();
                        chatStore.saveState(conversationId, state);
                        await new Promise(resolve => setTimeout(resolve, 15));
                    }
                })(),
            );

            await Promise.all(operations);

            // Reads should show monotonically increasing values
            for (let i = 1; i < readResults.length; i++) {
                expect(readResults[i]).toBeGreaterThanOrEqual(readResults[i - 1]);
            }
        });

        it("should handle cache stampede prevention", async () => {
            const conversationId = "chat_stampede";
            let fetchCallCount = 0;

            // Mock slow database fetch
            mockDb.chat.findUniqueOrThrow.mockImplementation(async () => {
                fetchCallCount++;
                await new Promise(resolve => setTimeout(resolve, 100));
                return {
                    id: conversationId,
                    config: {
                        __version: "1.0",
                        model: "gpt-4o",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    },
                };
            });

            // Simulate 20 concurrent requests for the same conversation
            const loadPromises = [];
            for (let i = 0; i < 20; i++) {
                loadPromises.push(chatStore.loadState(conversationId));
            }

            const results = await Promise.all(loadPromises);

            // Should only fetch from DB once despite concurrent requests
            expect(fetchCallCount).toBe(1);

            // All results should be identical (but different objects)
            for (let i = 1; i < results.length; i++) {
                expect(results[i]).toEqual(results[0]);
                expect(results[i]).not.toBe(results[0]); // Different object references
            }
        });
    });

    describe("MessageStore Concurrent Access", () => {
        it("should handle concurrent message additions", async () => {
            const conversationId = "chat_messages";

            // Add 100 messages concurrently
            const messagePromises = [];
            for (let i = 0; i < 100; i++) {
                const message: MessageState = {
                    id: generatePK().toString(),
                    content: `Message ${i}`,
                    role: i % 2 === 0 ? "user" : "bot",
                    timestamp: Date.now() + i,
                };
                messagePromises.push(
                    messageStore.addMessage(conversationId, message),
                );
            }

            await Promise.all(messagePromises);

            // Retrieve all messages
            const messages = await messageStore.getMessages(conversationId);

            // Should have all 100 messages
            expect(messages).toHaveLength(100);

            // Messages should be in timestamp order
            for (let i = 1; i < messages.length; i++) {
                expect(messages[i].timestamp).toBeGreaterThan(messages[i - 1].timestamp);
            }
        });

        it("should handle concurrent reads while writing", async () => {
            const conversationId = "chat_read_write";

            // Pre-populate some messages
            for (let i = 0; i < 10; i++) {
                await messageStore.addMessage(conversationId, {
                    id: generatePK().toString(),
                    content: `Initial message ${i}`,
                    role: "user",
                    timestamp: Date.now() + i,
                });
            }

            // Concurrent operations
            const operations = [];
            const readCounts: number[] = [];

            // Writer: Adds messages continuously
            operations.push(
                (async () => {
                    for (let i = 0; i < 20; i++) {
                        await messageStore.addMessage(conversationId, {
                            id: generatePK().toString(),
                            content: `New message ${i}`,
                            role: "bot",
                            timestamp: Date.now() + 100 + i,
                        });
                        await new Promise(resolve => setTimeout(resolve, 5));
                    }
                })(),
            );

            // Reader: Reads message count periodically
            operations.push(
                (async () => {
                    for (let i = 0; i < 10; i++) {
                        const messages = await messageStore.getMessages(conversationId);
                        readCounts.push(messages.length);
                        await new Promise(resolve => setTimeout(resolve, 10));
                    }
                })(),
            );

            await Promise.all(operations);

            // Read counts should be non-decreasing
            for (let i = 1; i < readCounts.length; i++) {
                expect(readCounts[i]).toBeGreaterThanOrEqual(readCounts[i - 1]);
            }

            // Final count should be 30 (10 initial + 20 new)
            const finalMessages = await messageStore.getMessages(conversationId);
            expect(finalMessages).toHaveLength(30);
        });
    });

    describe("L1/L2 Cache Performance", () => {
        it("should demonstrate L1 cache hit performance", async () => {
            const conversationId = "chat_l1_perf";
            const state: ConversationState = {
                id: conversationId,
                participants: [],
                status: "in_progress",
                config: {
                    __version: "1.0",
                    model: "gpt-4o",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: "0",
                        startedAt: null,
                        lastProcessingCycleEndedAt: null,
                    },
                } as ChatConfigObject,
            };

            // Warm up L1 cache
            mockL1Cache.set(conversationId, state);

            // Measure L1 cache hits
            const startTime = performance.now();
            const iterations = 10000;

            for (let i = 0; i < iterations; i++) {
                await chatStore.getConversation(conversationId);
            }

            const endTime = performance.now();
            const avgTimeMs = (endTime - startTime) / iterations;

            // L1 cache hits should be very fast (< 0.1ms average)
            expect(avgTimeMs).toBeLessThan(0.1);

            // Should not hit L2 cache or DB
            expect(mockCache.getFromCache).not.toHaveBeenCalled();
            expect(mockDb.chat.findUniqueOrThrow).not.toHaveBeenCalled();
        });

        it("should handle L1 cache eviction under memory pressure", async () => {
            // Create a small L1 cache that will evict entries
            const smallL1Cache = new LRUCache<string, ConversationState>(10);
            const smallCacheStore = new DbChatStore(50, smallL1Cache);

            // Add 20 conversations (more than cache size)
            for (let i = 0; i < 20; i++) {
                const state: ConversationState = {
                    id: `chat_${i}`,
                    participants: [],
                    status: "in_progress",
                    config: {
                        __version: "1.0",
                        model: "gpt-4o",
                        stats: {
                            totalToolCalls: i,
                            totalCredits: String(i * 100),
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    } as ChatConfigObject,
                };

                smallL1Cache.set(`chat_${i}`, state);
            }

            // Early entries should be evicted
            expect(smallL1Cache.get("chat_0")).toBeUndefined();
            expect(smallL1Cache.get("chat_5")).toBeUndefined();

            // Recent entries should still be in cache
            expect(smallL1Cache.get("chat_19")).toBeDefined();
            expect(smallL1Cache.get("chat_15")).toBeDefined();

            // Access an old entry to test L2 fallback
            mockCache.getFromCache.mockResolvedValue(JSON.stringify({
                id: "chat_0",
                participants: [],
                status: "in_progress",
                config: {
                    model: "gpt-4o",
                    stats: { totalToolCalls: 0, totalCredits: "0" },
                },
            }));

            const result = await smallCacheStore.getConversation("chat_0");

            expect(result).toBeDefined();
            expect(mockCache.getFromCache).toHaveBeenCalled();
        });
    });

    describe("High Concurrency Stress Test", () => {
        it("should handle 100+ concurrent conversations", async () => {
            const numConversations = 100;
            const updatesPerConversation = 5;
            const allOperations = [];

            // Create response service with mocked dependencies
            const mockOpenAIClient = {
                moderations: { create: vi.fn().mockResolvedValue({ results: [{ flagged: false }] }) },
                chat: {
                    completions: {
                        create: vi.fn().mockResolvedValue({
                            async *[Symbol.asyncIterator]() {
                                yield {
                                    id: "chunk1",
                                    choices: [{
                                        delta: { content: "Response" },
                                        finish_reason: "stop",
                                    }],
                                    usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
                                };
                            },
                        }),
                    },
                },
            };

            const openAIService = new OpenAIService();
            (openAIService as any).client = mockOpenAIClient;

            // Simulate multiple conversations with concurrent updates
            for (let convIdx = 0; convIdx < numConversations; convIdx++) {
                const conversationId = `stress_test_${convIdx}`;

                // Initial state setup
                const initialState: ConversationState = {
                    id: conversationId,
                    participants: [{
                        id: `bot_${convIdx}`,
                        name: `Bot ${convIdx}`,
                        config: { __version: "1.0", resources: [], model: "gpt-4o" },
                        state: "ready",
                    } as BotParticipant],
                    status: "in_progress",
                    config: {
                        __version: "1.0",
                        model: "gpt-4o",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                        },
                    } as ChatConfigObject,
                };

                // Mock DB responses for this conversation
                mockDb.chat.findUniqueOrThrow.mockResolvedValue({
                    id: conversationId,
                    config: initialState.config,
                });

                // Simulate multiple updates per conversation
                for (let updateIdx = 0; updateIdx < updatesPerConversation; updateIdx++) {
                    allOperations.push(
                        (async () => {
                            const state = await chatStore.loadState(conversationId);
                            state.config.stats.totalToolCalls += 1;
                            state.config.stats.totalCredits = (parseInt(state.config.stats.totalCredits) + Math.random() * 100).toString();
                            chatStore.saveState(conversationId, state);

                            // Add a message
                            await messageStore.addMessage(conversationId, {
                                id: generatePK().toString(),
                                content: `Message ${updateIdx} for conversation ${convIdx}`,
                                role: updateIdx % 2 === 0 ? "user" : "bot",
                                timestamp: Date.now() + updateIdx,
                            });
                        })(),
                    );
                }
            }

            // Execute all operations concurrently
            const startTime = performance.now();
            await Promise.all(allOperations);
            const endTime = performance.now();

            const totalOperations = numConversations * updatesPerConversation;
            const totalTimeMs = endTime - startTime;
            const avgTimePerOperation = totalTimeMs / totalOperations;

            // Performance assertions
            expect(avgTimePerOperation).toBeLessThan(10); // Should average < 10ms per operation

            // Verify data integrity - spot check a few conversations
            for (let i = 0; i < 5; i++) {
                const checkId = `stress_test_${i}`;
                const messages = await messageStore.getMessages(checkId);
                expect(messages.length).toBe(updatesPerConversation);
            }

            // Advance timers to flush all pending writes
            await vi.advanceTimersByTimeAsync(100);

            // Database writes should be batched efficiently
            // With 100ms debounce and concurrent updates, we expect significantly fewer DB writes than total updates
            const dbWriteCount = mockDb.transaction.mock.calls.length;
            expect(dbWriteCount).toBeLessThan(totalOperations / 2);
        });
    });

    describe("Memory Leak Prevention", () => {
        it("should not leak memory when processing many conversations", async () => {
            // Process many conversations sequentially
            for (let i = 0; i < 1000; i++) {
                const conversationId = `memory_test_${i}`;

                // Mock DB response
                mockDb.chat.findUniqueOrThrow.mockResolvedValue({
                    id: conversationId,
                    config: {
                        __version: "1.0",
                        model: "gpt-4o",
                        stats: {
                            totalToolCalls: 0,
                            totalCredits: "0",
                            startedAt: null,
                            lastProcessingCycleEndedAt: null,
                        },
                    },
                });

                // Load, update, and save state
                const state = await chatStore.loadState(conversationId);
                state.config.stats.totalToolCalls = i;
                chatStore.saveState(conversationId, state);

                // Add some messages
                for (let j = 0; j < 10; j++) {
                    await messageStore.addMessage(conversationId, {
                        id: generatePK().toString(),
                        content: `Message ${j}`,
                        role: "user",
                        timestamp: Date.now() + j,
                    });
                }

                // Simulate cleanup every 100 conversations
                if (i % 100 === 0) {
                    await vi.advanceTimersByTimeAsync(100);
                }
            }

            // Check that internal state maps don't grow unbounded
            const internalState = (chatStore as any).state;

            // Should have some entries but not all 1000
            expect(internalState.size).toBeLessThan(1000);

            // L1 cache should respect its size limit
            expect(mockL1Cache.size).toBeLessThanOrEqual(100);
        });
    });
});
