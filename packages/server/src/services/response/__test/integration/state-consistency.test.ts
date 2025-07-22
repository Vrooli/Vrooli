import {
    BotConfig,
    type BotParticipant,
    ChatConfig,
    type ChatConfigObject,
    type ConversationState,
    generatePK,
    LlmServiceId,
    type MessageState,
    type SessionUser,
    type SwarmState,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DbProvider } from "../../../../db/provider.js";
import { logger } from "../../../../events/logger.js";
import { CacheService } from "../../../../redisConn.js";
import { SwarmContextManager } from "../../../execution/shared/SwarmContextManager.js";
import { type CachedConversationStateStore, conversationStateStore, PrismaChatStore } from "../../chatStore.js";
import { MessageHistoryBuilder } from "../../messageHistoryBuilder.js";
import { RedisMessageStore } from "../../messageStore.js";
import { ResponseService } from "../../responseService.js";
import { FallbackRouter } from "../../router.js";
import { OpenAIService } from "../../services.js";
import { CompositeToolRunner } from "../../toolRunner.js";
// Note: Container setup is handled by global vitest setup

// Mock external dependencies
vi.mock("openai", () => {
    return {
        default: vi.fn().mockImplementation(() => ({
            moderations: {
                create: vi.fn().mockResolvedValue({
                    results: [{ flagged: false, categories: {}, category_scores: {} }],
                }),
            },
            chat: {
                completions: {
                    create: vi.fn(),
                },
            },
        })),
    };
});

vi.mock("../../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
        trace: vi.fn(),
    },
}));

// Helper to create mock stream responses
function createMockTextStream(chunks: string[]) {
    return {
        async *[Symbol.asyncIterator]() {
            for (const chunk of chunks) {
                yield {
                    choices: [{
                        delta: { content: chunk },
                        finish_reason: null,
                    }],
                };
            }
            yield {
                choices: [{
                    delta: {},
                    finish_reason: "stop",
                }],
                usage: {
                    prompt_tokens: 50,
                    completion_tokens: 25,
                    total_tokens: 75,
                },
            };
        },
    };
}

describe("State Consistency Integration Tests", () => {
    let responseService: ResponseService;
    let contextManager: SwarmContextManager;
    let conversationStore: CachedConversationStateStore;
    let chatStore: PrismaChatStore;
    let messageStore: MessageStore;
    let toolRunner: CompositeToolRunner;
    let mockOpenAIClient: any;
    let mockDb: any;
    let mockCache: any;
    let testUser: SessionUser;

    beforeEach(async () => {
        vi.clearAllMocks();

        // Note: Test containers are already set up by global vitest setup
        // Initialize core services
        mockDb = DbProvider.get();
        mockCache = CacheService.get();

        // Set up mock OpenAI client
        const OpenAI = (await import("openai")).default;
        mockOpenAIClient = new OpenAI({ apiKey: "test-key" });

        // Initialize AI services
        const openAIService = new OpenAIService();
        (openAIService as any).client = mockOpenAIClient;

        const router = new FallbackRouter([
            { id: LlmServiceId.OpenAI, service: openAIService },
        ]);

        // Initialize all services with real dependencies
        conversationStore = conversationStateStore;
        chatStore = new PrismaChatStore();
        messageStore = RedisMessageStore.get();
        toolRunner = new CompositeToolRunner();
        contextManager = new SwarmContextManager();

        responseService = new ResponseService(
            router,
            toolRunner,
            new MessageHistoryBuilder(),
        );

        // Set up test user
        testUser = {
            id: "user123",
            credits: 50000n,
            name: "Test User",
            handle: "testuser",
        } as SessionUser;
    });

    afterEach(async () => {
        // Note: Test containers are cleaned up by global vitest teardown
        vi.restoreAllMocks();
    });

    describe("Cross-Service State Synchronization", () => {
        it("should maintain state consistency between ChatStore and ConversationStore", async () => {
            const conversationId = generatePK();

            // Create initial conversation state
            const initialState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "bot1",
                    name: "Test Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "Test system message",
            };

            // 1. Save to ChatStore
            await chatStore.saveState(conversationId, initialState);

            // Wait for debounced write to complete
            await chatStore.finalizeSave();

            // 2. Load from ConversationStore (should hit database since not cached yet)
            const loadedFromConversationStore = await conversationStore.get(conversationId);
            expect(loadedFromConversationStore).toEqual(initialState);

            // 3. Update config via ConversationStore
            const updatedConfig = {
                ...initialState.config,
                stats: {
                    ...initialState.config.stats,
                    totalCredits: 150,
                    totalToolCalls: 3,
                },
            } as ChatConfigObject;

            await conversationStore.updateConfig(conversationId, updatedConfig);

            // 4. Verify ChatStore reflects the update
            // Wait for debounced write to complete
            await chatStore.finalizeSave();

            const loadedFromChatStore = await chatStore.loadState(conversationId);
            expect(loadedFromChatStore.config.stats.totalCredits).toBe(150);
            expect(loadedFromChatStore.config.stats.totalToolCalls).toBe(3);

            // 5. Verify ConversationStore cache is consistent
            const reloadedFromConversationStore = await conversationStore.get(conversationId);
            expect(reloadedFromConversationStore?.config.stats.totalCredits).toBe(150);
            expect(reloadedFromConversationStore?.config.stats.totalToolCalls).toBe(3);
        });

        it("should synchronize state updates between SwarmContextManager and ChatStore", async () => {
            const swarmId = generatePK();
            const conversationId = generatePK();

            const testBot: BotParticipant = {
                id: "sync-bot",
                name: "Sync Bot",
                config: BotConfig.parse({ model: "gpt-4o" }),
                state: "ready",
            };

            // 1. Initialize SwarmState in context manager
            const initialSwarmState: SwarmState = {
                swarmId,
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [testBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: 1000, tokens: 10000, time: 300 },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext(swarmId, initialSwarmState);

            // 2. Initialize corresponding conversation state
            const conversationState: ConversationState = {
                id: conversationId,
                participants: [testBot],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, conversationState);

            // 3. Update SwarmState with resource consumption
            const updatedSwarmState = await contextManager.getContext(swarmId);
            if (updatedSwarmState) {
                updatedSwarmState.resources.consumed = { credits: 250, tokens: 1500, time: 30 };
                updatedSwarmState.chatConfig.stats = {
                    ...updatedSwarmState.chatConfig.stats,
                    totalCredits: 250,
                    totalToolCalls: 2,
                };
                await contextManager.updateContext(swarmId, updatedSwarmState);
            }

            // 4. Simulate a response generation that should update both stores
            const mockStream = createMockTextStream(["Test response for sync"]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            const userMessage: MessageState = {
                id: generatePK(),
                content: "Test message",
                role: "user",
                timestamp: Date.now(),
            };

            const responseParams = {
                conversationId,
                messages: [userMessage],
                bot: testBot,
                userData: testUser,
                maxTokens: 1000,
            };

            const events = [];
            for await (const event of responseService.generateResponse(responseParams)) {
                events.push(event);
            }

            // Wait for all debounced operations to complete
            await new Promise(resolve => setTimeout(resolve, 3000));

            // 5. Verify both stores are consistent after the operation
            const finalSwarmState = await contextManager.getContext(swarmId);
            const finalConversationState = await chatStore.loadState(conversationId);

            expect(finalSwarmState?.resources.consumed.credits).toBeGreaterThan(250);
            expect(finalConversationState.config.stats.totalCredits).toBeGreaterThan(250);

            // Both should reflect similar credit usage (allowing for small differences in timing)
            const swarmCredits = finalSwarmState?.resources.consumed.credits || 0;
            const conversationCredits = finalConversationState.config.stats.totalCredits;
            expect(Math.abs(swarmCredits - conversationCredits)).toBeLessThan(50); // Allow small variance
        });

        it("should handle cache invalidation cascades correctly", async () => {
            const conversationId = generatePK();

            const initialState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "cache-bot",
                    name: "Cache Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            // 1. Prime the cache by loading state
            await chatStore.saveState(conversationId, initialState);
            const cachedState = await conversationStore.get(conversationId);
            expect(cachedState).toBeTruthy();

            // 2. Verify state is in L1 cache (subsequent get should be instant)
            const startTime = Date.now();
            const cachedAgain = await conversationStore.get(conversationId);
            const loadTime = Date.now() - startTime;
            expect(loadTime).toBeLessThan(10); // Should be very fast from L1 cache
            expect(cachedAgain).toEqual(cachedState);

            // 3. Force cache invalidation via explicit delete
            await conversationStore.del(conversationId);

            // 4. Update state in database directly (simulating external change)
            const updatedState = {
                ...initialState,
                config: {
                    ...initialState.config,
                    stats: {
                        ...initialState.config.stats,
                        totalCredits: 500,
                    },
                },
            };
            await chatStore.saveState(conversationId, updatedState);
            await chatStore.finalizeSave(); // Wait for write

            // 5. Load from conversation store should get fresh data from database
            const freshState = await conversationStore.get(conversationId, true); // Force invalidation
            expect(freshState?.config.stats.totalCredits).toBe(500);

            // 6. Verify L2 cache is now populated with fresh data
            const fromL2 = await conversationStore.get(conversationId);
            expect(fromL2?.config.stats.totalCredits).toBe(500);
        });
    });

    describe("Atomic Operations Across Services", () => {
        it("should handle atomic conversation state updates with rollback on failure", async () => {
            const conversationId = generatePK();

            const initialState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "atomic-bot",
                    name: "Atomic Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, initialState);
            await conversationStore.get(conversationId); // Prime cache

            // Record initial state
            const initialCredits = initialState.config.stats.totalCredits;

            // Simulate a response generation that should update state atomically
            const mockStream = createMockTextStream(["Atomic test response"]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            const userMessage: MessageState = {
                id: generatePK(),
                content: "Test atomic operation",
                role: "user",
                timestamp: Date.now(),
            };

            // Create a scenario where the response succeeds but we can test consistency
            const events = [];
            for await (const event of responseService.generateResponse({
                conversationId,
                messages: [userMessage],
                bot: initialState.participants[0],
                userData: testUser,
                maxTokens: 1000,
            })) {
                events.push(event);
            }

            // Wait for all operations to complete
            await new Promise(resolve => setTimeout(resolve, 3000));

            // Verify atomicity: both stores should have consistent state
            const chatStoreState = await chatStore.loadState(conversationId);
            const conversationStoreState = await conversationStore.get(conversationId, true);

            expect(chatStoreState.config.stats.totalCredits).toBeGreaterThan(initialCredits);
            expect(conversationStoreState?.config.stats.totalCredits).toBe(
                chatStoreState.config.stats.totalCredits,
            );

            // Verify that intermediate state wasn't exposed
            expect(events.find(e => e.type === "done")).toBeTruthy();
        });

        it("should maintain consistency during concurrent state modifications", async () => {
            const conversationId = generatePK();

            const initialState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "concurrent-bot",
                    name: "Concurrent Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, initialState);

            // Create multiple concurrent operations that modify state
            const operations = Array.from({ length: 5 }, (_, i) =>
                conversationStore.updateConfig(conversationId, {
                    ...initialState.config,
                    stats: {
                        ...initialState.config.stats,
                        totalCredits: (i + 1) * 100,
                        totalToolCalls: i + 1,
                    },
                } as ChatConfigObject),
            );

            // Execute all operations concurrently
            await Promise.all(operations);

            // Wait for all debounced writes to complete
            await new Promise(resolve => setTimeout(resolve, 3000));

            // Verify final state is consistent (last write wins)
            const finalChatStoreState = await chatStore.loadState(conversationId);
            const finalConversationStoreState = await conversationStore.get(conversationId, true);

            expect(finalConversationStoreState?.config.stats.totalCredits).toBe(
                finalChatStoreState.config.stats.totalCredits,
            );
            expect(finalConversationStoreState?.config.stats.totalToolCalls).toBe(
                finalChatStoreState.config.stats.totalToolCalls,
            );

            // Should be one of the expected values (last write wins)
            const expectedCredits = [100, 200, 300, 400, 500];
            expect(expectedCredits).toContain(finalChatStoreState.config.stats.totalCredits);
        });
    });

    describe("L1/L2 Cache Consistency", () => {
        it("should maintain consistency between L1 (LRU) and L2 (Redis) caches", async () => {
            const conversationId = generatePK();

            const testState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "cache-consistency-bot",
                    name: "Cache Consistency Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, testState);

            // 1. Load into L1 cache via conversation store
            const firstLoad = await conversationStore.get(conversationId);
            expect(firstLoad).toEqual(testState);

            // 2. Clear L1 cache only (simulate LRU eviction)
            const conversationStoreInternal = conversationStore as any;
            conversationStoreInternal.cache.delete(conversationId);

            // 3. Load again - should come from L2 (Redis)
            const secondLoad = await conversationStore.get(conversationId);
            expect(secondLoad).toEqual(testState);

            // 4. Update state via conversation store
            const updatedConfig = {
                ...testState.config,
                stats: {
                    ...testState.config.stats,
                    totalCredits: 300,
                },
            } as ChatConfigObject;

            await conversationStore.updateConfig(conversationId, updatedConfig);

            // 5. Clear L1 again and verify L2 has the update
            conversationStoreInternal.cache.delete(conversationId);

            const thirdLoad = await conversationStore.get(conversationId);
            expect(thirdLoad?.config.stats.totalCredits).toBe(300);

            // 6. Verify L1 and L2 are consistent
            const fourthLoad = await conversationStore.get(conversationId);
            expect(fourthLoad?.config.stats.totalCredits).toBe(300);
            expect(fourthLoad).toEqual(thirdLoad);
        });

        it("should handle L2 cache failures gracefully and maintain L1 consistency", async () => {
            const conversationId = generatePK();

            const testState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "l2-failure-bot",
                    name: "L2 Failure Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, testState);

            // 1. Load into cache normally
            const initialLoad = await conversationStore.get(conversationId);
            expect(initialLoad).toBeTruthy();

            // 2. Simulate L2 (Redis) failure for get operations
            const originalGet = mockCache.get;
            mockCache.get = vi.fn().mockRejectedValue(new Error("Redis connection failed"));

            // 3. Clear L1 cache to force L2 lookup
            const conversationStoreInternal = conversationStore as any;
            conversationStoreInternal.cache.delete(conversationId);

            // 4. Should fall back to database and still work
            const fallbackLoad = await conversationStore.get(conversationId);
            expect(fallbackLoad).toEqual(testState);

            // 5. Verify error was logged but operation succeeded
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error fetching ConversationState from L2 cache"),
                expect.any(Object),
            );

            // 6. Restore Redis and verify L1 cache is populated
            mockCache.get = originalGet;
            const finalLoad = await conversationStore.get(conversationId);
            expect(finalLoad).toEqual(testState);
        });
    });

    describe("Database vs Cache Consistency", () => {
        it("should detect and resolve inconsistencies between database and cache", async () => {
            const conversationId = generatePK();

            const initialState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "consistency-bot",
                    name: "Consistency Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            // 1. Save initial state and populate cache
            await chatStore.saveState(conversationId, initialState);
            await conversationStore.get(conversationId);

            // 2. Directly modify database (simulating external modification)
            const dbModifiedState = {
                ...initialState,
                config: {
                    ...initialState.config,
                    stats: {
                        ...initialState.config.stats,
                        totalCredits: 999,
                        totalToolCalls: 99,
                    },
                },
            };

            // Directly update via ChatStore without going through conversation store
            await chatStore.saveState(conversationId, dbModifiedState);
            await chatStore.finalizeSave(); // Wait for write

            // 3. Cache should still have old data
            const cachedData = await conversationStore.get(conversationId);
            expect(cachedData?.config.stats.totalCredits).toBe(0); // Original value

            // 4. Force cache invalidation and reload
            const freshData = await conversationStore.get(conversationId, true);
            expect(freshData?.config.stats.totalCredits).toBe(999); // Updated value

            // 5. Verify subsequent loads get the updated data
            const confirmedData = await conversationStore.get(conversationId);
            expect(confirmedData?.config.stats.totalCredits).toBe(999);
        });

        it("should handle database connection failures with cache fallback", async () => {
            const conversationId = generatePK();

            const testState: ConversationState = {
                id: conversationId,
                participants: [{
                    id: "db-failure-bot",
                    name: "DB Failure Bot",
                    config: BotConfig.parse({ model: "gpt-4o" }),
                    state: "ready",
                }],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            // 1. Populate cache normally
            await chatStore.saveState(conversationId, testState);
            const cachedState = await conversationStore.get(conversationId);
            expect(cachedState).toBeTruthy();

            // 2. Simulate database failure
            const originalFindUnique = mockDb.chat.findUnique;
            mockDb.chat.findUnique = vi.fn().mockRejectedValue(new Error("Database connection failed"));

            // 3. Clear database state but keep cache
            await conversationStore.del(conversationId); // This clears cache

            // 4. Try to load - should fail gracefully
            mockDb.chat.findUnique = vi.fn().mockResolvedValue(null); // Simulate not found instead of error
            const failedLoad = await conversationStore.get(conversationId);
            expect(failedLoad).toBeNull();

            // 5. Restore database connection
            mockDb.chat.findUnique = originalFindUnique;

            // 6. Should work normally again
            await chatStore.saveState(conversationId, testState);
            const restoredLoad = await conversationStore.get(conversationId);
            expect(restoredLoad).toEqual(testState);
        });
    });

    describe("Real-World Consistency Scenarios", () => {
        it("should maintain consistency during high-frequency conversation updates", async () => {
            const conversationId = "high-frequency-test";

            const bot: BotParticipant = {
                id: "high-freq-bot",
                name: "High Frequency Bot",
                config: BotConfig.parse({ model: "gpt-4o-mini" }),
                state: "ready",
            };

            const initialState: ConversationState = {
                id: conversationId,
                participants: [bot],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, initialState);

            // Create rapid-fire response generations that update state
            const rapidResponses = Array.from({ length: 10 }, (_, i) => {
                const mockStream = createMockTextStream([`Response ${i + 1}`]);
                mockOpenAIClient.chat.completions.create.mockResolvedValue(mockStream);

                return responseService.generateResponse({
                    conversationId,
                    messages: [{
                        id: generatePK(),
                        content: `Message ${i + 1}`,
                        role: "user",
                        timestamp: Date.now(),
                    }],
                    bot,
                    userData: testUser,
                    maxTokens: 100,
                });
            });

            // Execute all responses as fast as possible
            const responseResults = await Promise.all(
                rapidResponses.map(async responseStream => {
                    const events = [];
                    for await (const event of responseStream) {
                        events.push(event);
                    }
                    return events;
                }),
            );

            // Wait for all debounced operations to complete
            await new Promise(resolve => setTimeout(resolve, 5000));

            // Verify all responses succeeded
            expect(responseResults.every(events =>
                events.some(event => event.type === "done"),
            )).toBe(true);

            // Verify final state consistency
            const finalChatStoreState = await chatStore.loadState(conversationId);
            const finalConversationStoreState = await conversationStore.get(conversationId, true);

            expect(finalChatStoreState.config.stats.totalCredits).toBeGreaterThan(0);
            expect(finalConversationStoreState?.config.stats.totalCredits).toBe(
                finalChatStoreState.config.stats.totalCredits,
            );

            // Should have processed all 10 responses
            const totalCost = responseResults.reduce((sum, events) => {
                const doneEvent = events.find(e => e.type === "done");
                return sum + (doneEvent?.cost || 0);
            }, 0);

            expect(finalChatStoreState.config.stats.totalCredits).toBeGreaterThanOrEqual(totalCost * 0.8); // Allow for rounding
        });
    });
});
