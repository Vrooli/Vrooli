import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { 
    type ChatConfigObject, 
    type ConversationState,
    type BotParticipant,
    generatePK,
    MINUTES_1_S,
    ChatConfig,
    LRUCache,
} from "@vrooli/shared";
import { 
    ChatStore, 
    PrismaChatStore, 
    CachedConversationStateStore,
    type ConversationStatsPatch, 
} from "./chatStore.js";
import { DbProvider } from "../../db/provider.js";
import { logger } from "../../events/logger.js";
import { CacheService } from "../../redisConn.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
        debug: vi.fn(),
    },
}));

vi.mock("../../db/provider.js", () => ({
    DbProvider: {
        get: vi.fn().mockReturnValue({
            chat: {
                findUnique: vi.fn(),
                update: vi.fn(),
            },
        }),
    },
}));

vi.mock("../../redisConn.js", () => ({
    CacheService: {
        get: vi.fn().mockReturnValue({
            get: vi.fn(),
            set: vi.fn(),
            del: vi.fn(),
        }),
    },
}));

// Test implementation of abstract ChatStore for testing base functionality
class TestChatStore extends ChatStore {
    public mockUpsertState = vi.fn();
    public mockFetchState = vi.fn();
    public mockGetConversation = vi.fn();

    constructor(debounceMs?: number) {
        super(debounceMs);
    }

    protected async upsertState(conversationId: string, state: ConversationState): Promise<void> {
        return this.mockUpsertState(conversationId, state);
    }

    protected async fetchState(conversationId: string): Promise<ConversationState> {
        return this.mockFetchState(conversationId);
    }

    async getConversation(id: string): Promise<ConversationState | null> {
        return this.mockGetConversation(id);
    }

    // Expose protected state for testing
    getInternalState() {
        return this.state;
    }
}

describe("ChatStore (abstract base)", () => {
    let store: TestChatStore;
    let mockState: ConversationState;

    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
        
        store = new TestChatStore(100); // Use short debounce for testing
        
        mockState = {
            id: "chat123",
            participants: [
                {
                    id: "bot1",
                    name: "TestBot",
                    config: {},
                    state: "ready",
                } as BotParticipant,
            ],
            status: "in_progress",
            config: {
                model: "gpt-4o",
                stats: {
                    totalToolCalls: 0,
                    totalCredits: 0,
                },
            } as ChatConfigObject,
        };
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    describe("loadState", () => {
        it("should fetch and cache state on first load", async () => {
            store.mockFetchState.mockResolvedValue(mockState);

            const result = await store.loadState("chat123");

            expect(store.mockFetchState).toHaveBeenCalledWith("chat123");
            expect(result).toEqual(mockState);
            expect(result).not.toBe(mockState); // Should be a clone
            
            // Check that state is cached
            const internalState = store.getInternalState();
            expect(internalState.has("chat123")).toBe(true);
        });

        it("should return cached state on subsequent loads", async () => {
            store.mockFetchState.mockResolvedValue(mockState);

            // First load
            await store.loadState("chat123");
            
            // Second load
            const result = await store.loadState("chat123");

            // Should only fetch once
            expect(store.mockFetchState).toHaveBeenCalledTimes(1);
            expect(result).toEqual(mockState);
        });

        it("should return cloned state to prevent external mutations", async () => {
            store.mockFetchState.mockResolvedValue(mockState);

            const state1 = await store.loadState("chat123");
            const state2 = await store.loadState("chat123");

            // Modify state1
            state1.config.stats.totalToolCalls = 999;

            // state2 should be unaffected
            expect(state2.config.stats.totalToolCalls).toBe(0);
        });
    });

    describe("saveState", () => {
        it("should schedule debounced writes", async () => {
            store.mockFetchState.mockResolvedValue(mockState);
            
            // Load initial state
            await store.loadState("chat123");

            // Update state
            const updatedState = {
                ...mockState,
                config: {
                    ...mockState.config,
                    stats: {
                        totalToolCalls: 5,
                        totalCredits: 100,
                    },
                },
            };

            store.saveState("chat123", updatedState);

            // Should not write immediately
            expect(store.mockUpsertState).not.toHaveBeenCalled();

            // Advance timers past debounce period
            await vi.advanceTimersByTimeAsync(150);

            // Now should have written
            expect(store.mockUpsertState).toHaveBeenCalledWith("chat123", updatedState);
        });

        it("should batch multiple rapid updates", async () => {
            store.mockFetchState.mockResolvedValue(mockState);
            await store.loadState("chat123");

            // Make multiple rapid updates
            for (let i = 1; i <= 5; i++) {
                const updatedState = {
                    ...mockState,
                    config: {
                        ...mockState.config,
                        stats: {
                            totalToolCalls: i,
                            totalCredits: i * 10,
                        },
                    },
                };
                store.saveState("chat123", updatedState);
                await vi.advanceTimersByTimeAsync(50); // Less than debounce period
            }

            // Should not have written yet
            expect(store.mockUpsertState).not.toHaveBeenCalled();

            // Advance past final debounce
            await vi.advanceTimersByTimeAsync(100);

            // Should write only once with final state
            expect(store.mockUpsertState).toHaveBeenCalledTimes(1);
            expect(store.mockUpsertState).toHaveBeenCalledWith("chat123", expect.objectContaining({
                config: expect.objectContaining({
                    stats: {
                        totalToolCalls: 5,
                        totalCredits: 50,
                    },
                }),
            }));
        });

        it("should handle saveState without prior loadState", async () => {
            // Save without loading first
            store.saveState("chat123", mockState);

            await vi.advanceTimersByTimeAsync(150);

            expect(store.mockUpsertState).toHaveBeenCalledWith("chat123", mockState);
        });

        it("should handle concurrent updates during flush", async () => {
            store.mockFetchState.mockResolvedValue(mockState);
            await store.loadState("chat123");

            // Make upsertState slow
            store.mockUpsertState.mockImplementation(async () => {
                await new Promise(resolve => setTimeout(resolve, 200));
            });

            // First update
            const update1 = { ...mockState, config: { ...mockState.config, stats: { totalToolCalls: 1, totalCredits: 10 } } };
            store.saveState("chat123", update1);

            // Wait for flush to start
            await vi.advanceTimersByTimeAsync(150);
            
            // Second update while first is still flushing
            const update2 = { ...mockState, config: { ...mockState.config, stats: { totalToolCalls: 2, totalCredits: 20 } } };
            store.saveState("chat123", update2);

            // Complete first flush
            await vi.advanceTimersByTimeAsync(200);

            // Should schedule another flush for the second update
            await vi.advanceTimersByTimeAsync(150);

            expect(store.mockUpsertState).toHaveBeenCalledTimes(2);
            expect(store.mockUpsertState).toHaveBeenNthCalledWith(1, "chat123", update1);
            expect(store.mockUpsertState).toHaveBeenNthCalledWith(2, "chat123", update2);
        });
    });

    describe("finalizeSave", () => {
        it("should wait for all pending writes to complete", async () => {
            store.mockFetchState.mockResolvedValue(mockState);
            
            // Create multiple conversations with pending writes
            await store.loadState("chat1");
            await store.loadState("chat2");
            
            store.saveState("chat1", { ...mockState, id: "chat1" });
            store.saveState("chat2", { ...mockState, id: "chat2" });

            // Start finalize
            const finalizePromise = store.finalizeSave(1000);

            // Advance timers to trigger flushes
            await vi.advanceTimersByTimeAsync(150);

            const result = await finalizePromise;

            expect(result).toBe(true);
            expect(store.mockUpsertState).toHaveBeenCalledTimes(2);
        });

        it("should timeout if writes take too long", async () => {
            store.mockFetchState.mockResolvedValue(mockState);
            await store.loadState("chat123");

            // Make upsertState hang indefinitely
            store.mockUpsertState.mockImplementation(() => new Promise(() => {}));

            store.saveState("chat123", mockState);

            // Use real timers for this test
            vi.useRealTimers();
            
            const result = await store.finalizeSave(100);

            expect(result).toBe(false);
            
            vi.useFakeTimers();
        });

        it("should return immediately if no pending writes", async () => {
            const result = await store.finalizeSave();
            expect(result).toBe(true);
        });
    });

    describe("error handling", () => {
        it("should handle flush errors gracefully", async () => {
            const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            store.mockFetchState.mockResolvedValue(mockState);
            store.mockUpsertState.mockRejectedValue(new Error("DB error"));

            await store.loadState("chat123");
            store.saveState("chat123", mockState);

            await vi.advanceTimersByTimeAsync(150);

            expect(consoleErrorSpy).toHaveBeenCalledWith(
                "ChatPersistence flush failed",
                "chat123",
                expect.any(Error),
            );

            // Store should continue to function
            const internalState = store.getInternalState();
            expect(internalState.get("chat123")?.storeInProgress).toBe(false);
            
            consoleErrorSpy.mockRestore();
        });
    });
});

describe("PrismaChatStore (concrete implementation)", () => {
    let store: PrismaChatStore;
    let mockDb: any;

    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
        
        mockDb = DbProvider.get();
        store = new PrismaChatStore();
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    describe("getConversation", () => {
        const mockChatData = {
            id: BigInt("123"),
            config: {
                model: "gpt-4o",
                stats: {
                    totalToolCalls: 5,
                    totalCredits: 100,
                },
            },
            participants: [
                {
                    id: "participant1",
                    user: {
                        id: "bot1",
                        name: "TestBot",
                        handle: "testbot",
                        isBot: true,
                        botSettings: { model: "gpt-4o" },
                    },
                },
            ],
        };

        it("should fetch chat from database and map participants correctly", async () => {
            mockDb.chat.findUnique.mockResolvedValue(mockChatData);

            const result = await store.getConversation("123");

            expect(result).toBeDefined();
            expect(result?.id).toBe("123");
            expect(result?.config.stats).toEqual({
                totalToolCalls: 5,
                totalCredits: 100,
            });
            expect(result?.participants).toHaveLength(1);
            expect(result?.participants[0].name).toBe("TestBot");
            expect(result?.participants[0].id).toBe("bot1");
        });

        it("should return null for missing chat", async () => {
            mockDb.chat.findUnique.mockResolvedValue(null);

            const result = await store.getConversation("nonexistent");

            expect(result).toBeNull();
        });

        it("should use default config when chat has no config", async () => {
            const chatWithoutConfig = {
                ...mockChatData,
                config: null,
            };
            mockDb.chat.findUnique.mockResolvedValue(chatWithoutConfig);

            const result = await store.getConversation("123");

            expect(result?.config).toEqual(ChatConfig.default().export());
        });
    });

    describe("upsertState (via saveState)", () => {
        it("should calculate and persist only changed fields", async () => {
            const initialState: ConversationState = {
                id: "123",
                participants: [],
                config: {
                    model: "gpt-4o",
                    stats: {
                        totalToolCalls: 0,
                        totalCredits: 0,
                    },
                } as ChatConfigObject,
                availableTools: [],
                initialLeaderSystemMessage: "",
                teamConfig: undefined,
            };

            const updatedState: ConversationState = {
                ...initialState,
                config: {
                    ...initialState.config,
                    stats: {
                        totalToolCalls: 5,
                        totalCredits: 100,
                    },
                } as ChatConfigObject,
            };

            // Mock getConversation to return initial state
            store.getConversation = vi.fn().mockResolvedValue(initialState);

            // Call public API
            await store.loadState("123");
            store.saveState("123", updatedState);
            
            // Trigger flush
            await vi.advanceTimersByTimeAsync(2500);

            expect(mockDb.chat.update).toHaveBeenCalledWith({
                where: { id: BigInt("123") },
                data: {
                    config: {
                        path: [],
                        value: {
                            stats: {
                                totalToolCalls: 5,
                                totalCredits: 100,
                            },
                        },
                    },
                },
            });
        });

        it("should handle database errors during update", async () => {
            const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
            
            const state: ConversationState = {
                id: "123",
                participants: [],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
                teamConfig: undefined,
            };

            mockDb.chat.update.mockRejectedValue(new Error("DB connection lost"));
            store.getConversation = vi.fn().mockResolvedValue(state);

            await store.loadState("123");
            store.saveState("123", state);
            
            await vi.advanceTimersByTimeAsync(2500);

            expect(consoleErrorSpy).toHaveBeenCalled();
            
            consoleErrorSpy.mockRestore();
        });
    });
});

describe("CachedConversationStateStore (L1/L2 cache integration)", () => {
    let store: CachedConversationStateStore;
    let mockChatStore: PrismaChatStore;
    let mockCache: any;

    beforeEach(() => {
        vi.clearAllMocks();
        
        mockChatStore = new PrismaChatStore();
        mockCache = CacheService.get();
        store = new CachedConversationStateStore(mockChatStore, 100);
    });

    const mockState: ConversationState = {
        id: "123",
        participants: [],
        config: ChatConfig.default().export(),
        availableTools: [],
        initialLeaderSystemMessage: "",
        teamConfig: undefined,
    };

    describe("get (L1/L2 cache hierarchy)", () => {
        it("should return from L1 cache if available", async () => {
            // Manually populate L1 cache by calling get method first
            mockChatStore.getConversation = vi.fn().mockResolvedValue(mockState);
            mockCache.get.mockResolvedValue(null); // L2 miss
            
            // First call to populate L1
            await store.get("123");
            
            // Second call should hit L1
            mockChatStore.getConversation = vi.fn(); // Reset mock
            const result = await store.get("123");

            expect(result).toEqual(mockState);
            expect(mockChatStore.getConversation).not.toHaveBeenCalled();
            expect(mockCache.get).toHaveBeenCalledTimes(1); // Only called in first get
        });

        it("should check L2 cache (Redis) if L1 miss", async () => {
            mockCache.get.mockResolvedValue(mockState);
            mockChatStore.getConversation = vi.fn();

            const result = await store.get("123");

            expect(result).toEqual(mockState);
            expect(mockCache.get).toHaveBeenCalledWith("conversation:123");
            expect(mockChatStore.getConversation).not.toHaveBeenCalled();
        });

        it("should fetch from database if both caches miss", async () => {
            mockCache.get.mockResolvedValue(null);
            mockChatStore.getConversation = vi.fn().mockResolvedValue(mockState);

            const result = await store.get("123");

            expect(result).toEqual(mockState);
            expect(mockChatStore.getConversation).toHaveBeenCalledWith("123");
            expect(mockCache.set).toHaveBeenCalledWith("conversation:123", mockState, 900);
        });

        it("should handle Redis errors gracefully", async () => {
            mockCache.get.mockRejectedValue(new Error("Redis connection failed"));
            mockChatStore.getConversation = vi.fn().mockResolvedValue(mockState);

            const result = await store.get("123");

            expect(result).toEqual(mockState);
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error fetching ConversationState from L2 cache"),
                expect.any(Object),
            );
        });

        it("should invalidate both caches when forced", async () => {
            // Populate caches first
            mockCache.get.mockResolvedValue(mockState);
            await store.get("123");

            // Force invalidation
            mockChatStore.getConversation = vi.fn().mockResolvedValue(mockState);
            await store.get("123", true);

            expect(mockCache.del).toHaveBeenCalledWith("conversation:123");
            expect(mockChatStore.getConversation).toHaveBeenCalled();
        });
    });

    describe("updateConfig", () => {
        it("should update L1 cache and trigger database save", async () => {
            const newConfig = {
                ...ChatConfig.default().export(),
                stats: { totalToolCalls: 5, totalCredits: 100 },
            } as ChatConfigObject;

            mockChatStore.getConversation = vi.fn().mockResolvedValue(mockState);
            mockChatStore.saveState = vi.fn();

            await store.updateConfig("123", newConfig);

            expect(mockChatStore.saveState).toHaveBeenCalledWith("123", expect.objectContaining({
                id: "123",
                config: newConfig,
            }));
        });

        it("should create fallback state if conversation not found", async () => {
            const newConfig = {
                ...ChatConfig.default().export(),
                stats: { totalToolCalls: 1, totalCredits: 10 },
            } as ChatConfigObject;

            mockChatStore.getConversation = vi.fn().mockResolvedValue(null);
            mockChatStore.saveState = vi.fn();

            await store.updateConfig("nonexistent", newConfig);

            expect(mockChatStore.saveState).toHaveBeenCalledWith("nonexistent", expect.objectContaining({
                id: "nonexistent",
                config: newConfig,
                participants: [],
            }));
        });
    });

    describe("updateTeamConfig", () => {
        it("should update team config in both L1 and L2 caches", async () => {
            const teamConfig = {
                id: "team123",
                name: "Test Team",
                config: {},
            };

            // Populate cache first
            mockChatStore.getConversation = vi.fn().mockResolvedValue(mockState);
            await store.get("123");

            await store.updateTeamConfig("123", teamConfig);

            expect(mockCache.set).toHaveBeenCalledWith(
                "conversation:123",
                expect.objectContaining({ teamConfig }),
                900,
            );
        });

        it("should warn if conversation state not found", async () => {
            const teamConfig = { id: "team123", name: "Test Team", config: {} };

            mockChatStore.getConversation = vi.fn().mockResolvedValue(null);
            mockCache.get.mockResolvedValue(null);

            await store.updateTeamConfig("nonexistent", teamConfig);

            expect(logger.warn).toHaveBeenCalledWith(
                expect.stringContaining("Cannot update team config for conversation nonexistent"),
            );
        });
    });

    describe("del (cache invalidation)", () => {
        it("should remove from both L1 and L2 caches", async () => {
            await store.del("123");

            expect(mockCache.del).toHaveBeenCalledWith("conversation:123");
        });

        it("should handle Redis deletion errors gracefully", async () => {
            mockCache.del.mockRejectedValue(new Error("Redis error"));

            await store.del("123");

            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error deleting ConversationState from L2 cache"),
                expect.any(Object),
            );
        });
    });
});
