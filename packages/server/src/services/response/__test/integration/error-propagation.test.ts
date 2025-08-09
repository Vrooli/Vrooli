import { BotConfig, type BotParticipant, ChatConfig, type ConversationContext, type ConversationParams, type ConversationState, type ConversationTrigger, generatePK, LlmServiceId, type SessionUser, type SwarmState, type Tool } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DbProvider } from "../../../../db/provider.js";
import { logger } from "../../../../events/logger.js";
import { CacheService } from "../../../../redisConn.js";
import { CompositeGraph } from "../../../conversation/agentGraph.js";
import { ConversationEngine } from "../../../conversation/conversationEngine.js";
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

vi.mock("../../../events/publisher.js", () => ({
    EventPublisher: {
        get: vi.fn(() => ({
            publish: vi.fn(),
        })),
    },
}));

// Helper to create mock error streams
function createMockErrorStream(errorMessage: string) {
    return {
        [Symbol.asyncIterator]() {
            // eslint-disable-next-line require-yield
            return (async function* () {
                throw new Error(errorMessage);
            })();
        },
    };
}

// Helper to create timeout streams
function createMockTimeoutStream(timeoutMs: number) {
    return {
        async *[Symbol.asyncIterator]() {
            await new Promise(resolve => setTimeout(resolve, timeoutMs));
            yield {
                choices: [{
                    delta: { content: "This should timeout" },
                    finish_reason: null,
                }],
            };
        },
    };
}

// Helper to create authentication error
function createAuthError() {
    const error = new Error("Invalid API key");
    (error as any).status = 401;
    (error as any).code = "invalid_api_key";
    return error;
}

// Helper to create rate limit error
function createRateLimitError() {
    const error = new Error("Rate limit exceeded");
    (error as any).status = 429;
    (error as any).code = "rate_limit_exceeded";
    return error;
}

describe("Error Propagation Integration Tests", () => {
    let conversationEngine: ConversationEngine;
    let responseService: ResponseService;
    let agentGraph: CompositeGraph;
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
        agentGraph = new CompositeGraph();
        contextManager = new SwarmContextManager();

        responseService = new ResponseService(
            router,
            toolRunner,
            new MessageHistoryBuilder(),
        );

        conversationEngine = new ConversationEngine(
            responseService,
            contextManager,
            {
                maxConcurrentTurns: 3,
                defaultTimeoutMs: 5000, // Short timeout for testing
                fallbackBehavior: "most_capable",
            },
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

    describe("Service-Level Error Propagation", () => {
        it("should propagate ResponseService errors to ConversationEngine with proper context", async () => {
            const errorBot: BotParticipant = {
                id: "error-bot",
                name: "Error Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "error-propagation-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [errorBot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "error-propagation-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [errorBot],
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

            await contextManager.updateContext("error-propagation-test", swarmState);

            // Mock OpenAI to return authentication error
            const authError = createAuthError();
            mockOpenAIClient.chat.completions.create.mockRejectedValueOnce(authError);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should handle error gracefully but mark as failed
            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
            expect(result.error?.code).toBe("CONVERSATION_ORCHESTRATION_FAILED");
            expect(result.error?.tier).toBe("tier2");

            // Verify error was logged with proper context
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Conversation orchestration failed"),
                expect.objectContaining({
                    swarmId: "error-propagation-test",
                    strategy: "conversation",
                }),
            );

            // Should have no successful messages but preserve conversation state
            expect(result.messages).toHaveLength(0);
            expect(result.updatedParticipants).toHaveLength(1);
            expect(result.updatedParticipants[0].success).toBe(false);
        });

        it("should handle LLM service cascading failures with retry logic", async () => {
            const retryBot: BotParticipant = {
                id: "retry-bot",
                name: "Retry Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "retry-cascade-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [retryBot],
                conversationHistory: [],
                availableTools: [],
            };

            // Mock multiple failures followed by success
            const rateLimitError = createRateLimitError();
            const successStream = {
                async *[Symbol.asyncIterator]() {
                    yield {
                        choices: [{
                            delta: { content: "Finally succeeded!" },
                            finish_reason: null,
                        }],
                    };
                    yield {
                        choices: [{
                            delta: {},
                            finish_reason: "stop",
                        }],
                        usage: {
                            prompt_tokens: 25,
                            completion_tokens: 10,
                            total_tokens: 35,
                        },
                    };
                },
            };

            mockOpenAIClient.chat.completions.create
                .mockRejectedValueOnce(rateLimitError) // First attempt fails
                .mockRejectedValueOnce(rateLimitError) // Second attempt fails
                .mockResolvedValueOnce(successStream); // Third attempt succeeds

            // Create a context where multiple retries can occur
            // (Note: This tests the FallbackRouter's retry logic within ResponseService)
            const swarmState: SwarmState = {
                swarmId: "retry-cascade-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [retryBot],
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

            await contextManager.updateContext("retry-cascade-test", swarmState);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should eventually succeed after retries
            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);
            expect(result.messages[0].content).toContain("Finally succeeded!");

            // Should log multiple retry attempts
            expect(logger.warn).toHaveBeenCalled();
        });

        it("should handle tool execution errors and maintain conversation flow", async () => {
            const toolBot: BotParticipant = {
                id: "tool-bot",
                name: "Tool Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "tool-error-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [toolBot],
                conversationHistory: [],
                availableTools: [
                    {
                        name: "failing_tool",
                        description: "A tool that always fails",
                        inputSchema: { type: "object", properties: { input: { type: "string" } } },
                    },
                ],
            };

            // Register a failing tool
            const failingTool: Tool = {
                name: "failing_tool",
                description: "A tool that always fails",
                execute: vi.fn().mockRejectedValue(new Error("Tool execution failed")),
            };
            toolRunner.register(failingTool);

            // Mock OpenAI to make a tool call
            const toolCallStream = {
                async *[Symbol.asyncIterator]() {
                    yield {
                        choices: [{
                            delta: {
                                tool_calls: [{
                                    index: 0,
                                    id: "call_failing_tool",
                                    type: "function",
                                    function: {
                                        name: "failing_tool",
                                        arguments: "{\"input\": \"test\"}",
                                    },
                                }],
                            },
                            finish_reason: null,
                        }],
                    };
                    yield {
                        choices: [{
                            delta: {},
                            finish_reason: "tool_calls",
                        }],
                        usage: {
                            prompt_tokens: 50,
                            completion_tokens: 30,
                            total_tokens: 80,
                        },
                    };
                },
            };

            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(toolCallStream);

            const swarmState: SwarmState = {
                swarmId: "tool-error-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [toolBot],
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

            await contextManager.updateContext("tool-error-test", swarmState);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Conversation should continue despite tool failure
            expect(result.success).toBe(true);
            expect(failingTool.execute).toHaveBeenCalled();

            // Should log tool execution error
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Tool execution error"),
                expect.any(Object),
            );
        });
    });

    describe("Storage Layer Error Propagation", () => {
        it("should handle database connection failures gracefully", async () => {
            const dbBot: BotParticipant = {
                id: "db-bot",
                name: "DB Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            // Simulate database failure
            const originalSaveState = chatStore.saveState;
            chatStore.saveState = vi.fn().mockRejectedValue(new Error("Database connection failed"));

            const conversationContext: ConversationContext = {
                swarmId: "db-error-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [dbBot],
                conversationHistory: [],
                availableTools: [],
            };

            // Mock successful OpenAI response
            const successStream = {
                async *[Symbol.asyncIterator]() {
                    yield {
                        choices: [{
                            delta: { content: "Response generated despite DB issues" },
                            finish_reason: null,
                        }],
                    };
                    yield {
                        choices: [{
                            delta: {},
                            finish_reason: "stop",
                        }],
                        usage: {
                            prompt_tokens: 30,
                            completion_tokens: 15,
                            total_tokens: 45,
                        },
                    };
                },
            };

            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(successStream);

            const swarmState: SwarmState = {
                swarmId: "db-error-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [dbBot],
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

            await contextManager.updateContext("db-error-test", swarmState);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should handle gracefully - response succeeds but persistence might fail
            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);

            // Should log database error
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Database"),
                expect.any(Object),
            );

            // Restore original method
            chatStore.saveState = originalSaveState;
        });

        it("should handle cache service failures with database fallback", async () => {
            const cacheBot: BotParticipant = {
                id: "cache-bot",
                name: "Cache Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationId = generatePK().toString();

            // Set up initial state
            const initialState: ConversationState = {
                id: conversationId,
                participants: [cacheBot],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, initialState);

            // Simulate cache failure
            const originalCacheGet = mockCache.get;
            const originalCacheSet = mockCache.set;
            mockCache.get = vi.fn().mockRejectedValue(new Error("Redis connection failed"));
            mockCache.set = vi.fn().mockRejectedValue(new Error("Redis connection failed"));

            // Should fall back to database
            const loadedState = await conversationStore.get(conversationId);
            expect(loadedState).toBeTruthy();
            expect(loadedState?.id).toBe(conversationId);

            // Should log cache errors but continue operation
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("cache"),
                expect.any(Object),
            );

            // Restore cache methods
            mockCache.get = originalCacheGet;
            mockCache.set = originalCacheSet;
        });

        it("should handle state corruption and recovery", async () => {
            const corruptBot: BotParticipant = {
                id: "corrupt-bot",
                name: "Corrupt Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationId = "state-corruption-test";

            // Save malformed state directly to database to simulate corruption
            const corruptedState = {
                id: conversationId,
                participants: "invalid_data", // Should be array
                config: null, // Should be object
                availableTools: undefined, // Should be array
            };

            // Mock database to return corrupted data
            const originalFindUnique = mockDb.chat.findUnique;
            mockDb.chat.findUnique = vi.fn().mockResolvedValue({
                id: conversationId,
                state: JSON.stringify(corruptedState),
                updatedAt: new Date(),
            });

            try {
                const loadedState = await conversationStore.get(conversationId);
                // Should either return null or throw validation error
                expect(loadedState).toBeNull();
            } catch (error) {
                // Or should catch corruption and handle gracefully
                expect(error).toBeInstanceOf(Error);
            }

            // Should log state corruption error
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("state"),
                expect.any(Object),
            );

            // Restore original method
            mockDb.chat.findUnique = originalFindUnique;
        });
    });

    describe("Cross-Service Error Correlation", () => {
        it("should correlate errors across multiple service layers", async () => {
            const errorBot: BotParticipant = {
                id: "correlation-bot",
                name: "Correlation Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "error-correlation-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [errorBot],
                conversationHistory: [],
                availableTools: [],
            };

            // Create a chain of failures across services

            // 1. Context manager failure
            const originalGetContext = contextManager.getContext;
            contextManager.getContext = vi.fn().mockRejectedValue(new Error("Context service unavailable"));

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should handle gracefully with proper error correlation
            expect(result.success).toBe(true); // ConversationEngine handles missing context gracefully
            expect(result.participantResults.size).toBe(0); // No bots selected due to context failure

            // Verify all error logs include correlation context
            const errorCalls = (logger.error as any).mock.calls;
            expect(errorCalls.some((call: any) =>
                call[1] && typeof call[1] === "object" && call[1].swarmId === "error-correlation-test",
            )).toBe(true);

            // Restore original method
            contextManager.getContext = originalGetContext;
        });

        it("should handle timeout cascades across service boundaries", async () => {
            const timeoutBot: BotParticipant = {
                id: "timeout-bot",
                name: "Timeout Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "timeout-cascade-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [timeoutBot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "timeout-cascade-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [timeoutBot],
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

            await contextManager.updateContext("timeout-cascade-test", swarmState);

            // Mock OpenAI to take longer than the timeout
            const timeoutStream = createMockTimeoutStream(10000); // 10 second timeout > 5 second limit
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(timeoutStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const startTime = Date.now();
            const result = await conversationEngine.orchestrateConversation(params);
            const duration = Date.now() - startTime;

            // Should timeout and handle gracefully
            expect(duration).toBeLessThan(7000); // Should respect timeout limits
            expect(result.success).toBe(false); // Should fail due to timeout
            expect(result.error?.code).toBe("CONVERSATION_ORCHESTRATION_FAILED");

            // Should log timeout-related errors
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Conversation orchestration failed"),
                expect.any(Object),
            );
        });
    });

    describe("Resource Management on Errors", () => {
        it("should clean up resources properly when errors occur", async () => {
            const resourceBot: BotParticipant = {
                id: "resource-bot",
                name: "Resource Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "resource-cleanup-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [resourceBot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "resource-cleanup-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [resourceBot],
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

            await contextManager.updateContext("resource-cleanup-test", swarmState);

            // Mock OpenAI to fail mid-stream
            const errorStream = createMockErrorStream("Connection interrupted");
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(errorStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should handle error and not leave resources in inconsistent state
            expect(result.success).toBe(false);
            expect(result.resourcesUsed.creditsUsed).toBe("0"); // No partial charges
            expect(result.resourcesUsed.durationMs).toBeGreaterThan(0); // Duration tracked even on failure

            // Verify swarm state wasn't corrupted by the error
            const finalSwarmState = await contextManager.getContext("resource-cleanup-test");
            expect(finalSwarmState).toBeTruthy();
            expect(finalSwarmState?.execution.status).toBe("running"); // Status preserved
        });

        it("should handle partial success scenarios with accurate resource tracking", async () => {
            const bot1: BotParticipant = {
                id: "success-bot",
                name: "Success Bot",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const bot2: BotParticipant = {
                id: "failure-bot",
                name: "Failure Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "partial-success-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [bot1, bot2],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "partial-success-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [bot1, bot2],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: 2000, tokens: 20000, time: 600 },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext("partial-success-test", swarmState);

            // Mock one success and one failure
            const successStream = {
                async *[Symbol.asyncIterator]() {
                    yield {
                        choices: [{
                            delta: { content: "Success!" },
                            finish_reason: null,
                        }],
                    };
                    yield {
                        choices: [{
                            delta: {},
                            finish_reason: "stop",
                        }],
                        usage: {
                            prompt_tokens: 40,
                            completion_tokens: 20,
                            total_tokens: 60,
                        },
                    };
                },
            };

            const errorStream = createMockErrorStream("Service unavailable");

            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(successStream)
                .mockResolvedValueOnce(errorStream);

            const trigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: {
                        respondingBots: ["@all"],
                    } as any,
                },
            };

            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
                metadata: {
                    executionMode: "parallel",
                },
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should succeed overall with partial results
            expect(result.success).toBe(true);
            expect(result.participantResults.size).toBe(2);
            expect(result.participantResults.has("success-bot")).toBe(true);
            expect(result.participantResults.has("failure-bot")).toBe(true);

            // Only successful bot should contribute to messages and resource usage
            expect(result.messages).toHaveLength(1);
            expect(result.messages[0].content).toContain("Success!");
            expect(BigInt(result.totalResourceUsage.creditsUsed)).toBeGreaterThan(0n);

            // Failed bot should be marked appropriately
            const failedBotResult = result.updatedParticipants.find(p => p.botId === "failure-bot");
            expect(failedBotResult?.success).toBe(false);
            expect(failedBotResult?.error).toBeDefined();

            // Should log partial failure
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error in parallel execution"),
                expect.any(Object),
            );
        });
    });

    describe("Error Recovery and Resilience", () => {
        it("should recover from transient failures and retry successfully", async () => {
            const resilientBot: BotParticipant = {
                id: "resilient-bot",
                name: "Resilient Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "resilience-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [resilientBot],
                conversationHistory: [],
                availableTools: [],
            };

            // Track retry attempts
            let attemptCount = 0;

            // Mock transient failures followed by success
            mockOpenAIClient.chat.completions.create.mockImplementation(async () => {
                attemptCount++;
                if (attemptCount <= 2) {
                    throw new Error("Transient network error");
                }

                return {
                    async *[Symbol.asyncIterator]() {
                        yield {
                            choices: [{
                                delta: { content: "Recovered successfully!" },
                                finish_reason: null,
                            }],
                        };
                        yield {
                            choices: [{
                                delta: {},
                                finish_reason: "stop",
                            }],
                            usage: {
                                prompt_tokens: 35,
                                completion_tokens: 12,
                                total_tokens: 47,
                            },
                        };
                    },
                };
            });

            const swarmState: SwarmState = {
                swarmId: "resilience-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [resilientBot],
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

            await contextManager.updateContext("resilience-test", swarmState);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should eventually succeed after retries
            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);
            expect(result.messages[0].content).toContain("Recovered successfully!");

            // Should have attempted multiple times
            expect(attemptCount).toBeGreaterThan(1);

            // Should log recovery
            expect(logger.warn).toHaveBeenCalled();
        });
    });
});
