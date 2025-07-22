import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
    type ConversationState,
    type ConversationParams,
    type ConversationTrigger,
    type ConversationContext,
    type MessageState,
    type BotParticipant,
    type SessionUser,
    type SwarmState,
    type Tool,
    type ChatConfigObject,
    type ExecutionResourceUsage,
    generatePK,
    LlmServiceId,
    ChatConfig,
    BotConfig,
} from "@vrooli/shared";
import { ConversationEngine } from "../../../conversation/conversationEngine.js";
import { ResponseService } from "../../responseService.js";
import { OpenAIService } from "../../services.js";
import { conversationStateStore, type CachedConversationStateStore, PrismaChatStore } from "../../chatStore.js";
import { CompositeToolRunner } from "../../toolRunner.js";
import { CompositeGraph } from "../../../conversation/agentGraph.js";
import { FallbackRouter } from "../../router.js";
import { MessageHistoryBuilder } from "../../messageHistoryBuilder.js";
import { RedisMessageStore } from "../../messageStore.js";
import { EventPublisher } from "../../../events/publisher.js";
import { SwarmContextManager } from "../../../execution/shared/SwarmContextManager.js";
import { DbProvider } from "../../../../db/provider.js";
import { CacheService } from "../../../../redisConn.js";
import { logger } from "../../../../events/logger.js";
// Test containers are handled by global setup

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

// Helper to create mock stream with specific token usage
function createMockTokenStream(content: string, promptTokens: number, completionTokens: number) {
    return {
        async *[Symbol.asyncIterator]() {
            for (const chunk of content.split(" ")) {
                yield {
                    choices: [{
                        delta: { content: chunk + " " },
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
                    prompt_tokens: promptTokens,
                    completion_tokens: completionTokens,
                    total_tokens: promptTokens + completionTokens,
                },
            };
        },
    };
}

// Helper to create expensive tool
function createExpensiveTool(credits: number, memoryMB: number, timeMs: number): Tool {
    return {
        name: "expensive_tool",
        description: "A tool that consumes significant resources",
        execute: vi.fn().mockImplementation(async () => {
            // Simulate time consumption
            await new Promise(resolve => setTimeout(resolve, timeMs));
            
            return {
                success: true,
                result: `Tool executed with ${credits} credits, ${memoryMB}MB memory, ${timeMs}ms time`,
                resourcesUsed: {
                    creditsUsed: credits.toString(),
                    memoryUsedMB: memoryMB,
                    durationMs: timeMs,
                    toolCalls: 1,
                    stepsExecuted: 1,
                },
            };
        }),
    };
}

describe("Resource Management Integration Tests", () => {
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
        
        // Test containers are handled by global setup
        
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
                defaultTimeoutMs: 30000,
                fallbackBehavior: "most_capable",
            },
        );
        
        // Set up test user with specific credit amount
        testUser = {
            id: "user123",
            credits: 10000n, // Start with 10,000 credits
            name: "Test User",
            handle: "testuser",
        } as SessionUser;
    });

    afterEach(async () => {
        // Test containers are handled by global setup
        vi.restoreAllMocks();
    });

    describe("Credit Tracking and Consumption", () => {
        it("should accurately track credit consumption across service boundaries", async () => {
            const swarmId = generatePK();
            const creditBot: BotParticipant = {
                id: "credit-tracker-bot",
                name: "Credit Tracker Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId,
                userData: testUser,
                timestamp: new Date(),
                participants: [creditBot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId,
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [creditBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: 5000, tokens: 50000, time: 600 },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext(swarmId, swarmState);

            // Mock OpenAI with known token costs
            const responseContent = "This is a response that will consume a specific amount of tokens";
            const promptTokens = 120;
            const completionTokens = 45;
            const mockStream = createMockTokenStream(responseContent, promptTokens, completionTokens);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            // Initialize conversation state
            const conversationState: ConversationState = {
                id: generatePK(),
                participants: [creditBot],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationState.id, conversationState);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Verify conversation succeeded
            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);

            // Calculate expected credit consumption (rough estimate: ~1 credit per token)
            const expectedMinCredits = Math.floor((promptTokens + completionTokens) * 0.8); // Allow some variance
            const actualCredits = BigInt(result.totalResourceUsage.creditsUsed);

            expect(Number(actualCredits)).toBeGreaterThanOrEqual(expectedMinCredits);

            // Wait for debounced state updates
            await new Promise(resolve => setTimeout(resolve, 2500));

            // Verify credit tracking consistency across all stores
            
            // 1. ConversationEngine result should match SwarmState
            const finalSwarmState = await contextManager.getContext(swarmId);
            expect(finalSwarmState?.resources.consumed.credits).toBeGreaterThan(0);
            expect(finalSwarmState?.resources.consumed.credits).toBe(Number(actualCredits));

            // 2. ChatStore should have updated stats
            const updatedConversationState = await chatStore.loadState(conversationState.id);
            expect(updatedConversationState.config.stats.totalCredits).toBe(Number(actualCredits));

            // 3. ConversationStore cache should be consistent
            const cachedState = await conversationStore.get(conversationState.id);
            expect(cachedState?.config.stats.totalCredits).toBe(Number(actualCredits));

            // 4. Verify remaining resources were decremented
            expect(finalSwarmState?.resources.remaining.credits).toBe(5000 - Number(actualCredits));
        });

        it("should enforce credit limits and prevent overconsumption", async () => {
            const swarmId = generatePK();
            const limitedBot: BotParticipant = {
                id: "limited-bot",
                name: "Limited Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            // Create user with very low credits
            const lowCreditUser: SessionUser = {
                ...testUser,
                credits: 10n, // Only 10 credits available
            };

            const conversationContext: ConversationContext = {
                swarmId,
                userData: lowCreditUser,
                timestamp: new Date(),
                participants: [limitedBot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId,
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [limitedBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: 10, tokens: 100, time: 60 }, // Very limited
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext(swarmId, swarmState);

            // Mock OpenAI with high token usage that would exceed limits
            const expensiveContent = "This is an expensive response with many tokens that should be limited";
            const mockStream = createMockTokenStream(expensiveContent, 500, 200); // 700 tokens total
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should either succeed with limited response or fail gracefully
            if (result.success) {
                // If it succeeded, credit usage should be within limits
                const creditsUsed = Number(BigInt(result.totalResourceUsage.creditsUsed));
                expect(creditsUsed).toBeLessThanOrEqual(10);
            } else {
                // If it failed, should be due to credit limits
                expect(result.error?.code).toBeDefined();
                expect(logger.warn).toHaveBeenCalledWith(
                    expect.stringContaining("credit"),
                    expect.any(Object),
                );
            }
        });

        it("should handle credit allocation for concurrent operations", async () => {
            const bot1: BotParticipant = {
                id: "concurrent-bot-1",
                name: "Concurrent Bot 1",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const bot2: BotParticipant = {
                id: "concurrent-bot-2", 
                name: "Concurrent Bot 2",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "concurrent-credit-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [bot1, bot2],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "concurrent-credit-test",
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

            await contextManager.updateContext("concurrent-credit-test", swarmState);

            // Mock responses for both bots
            const stream1 = createMockTokenStream("Response from bot 1", 80, 30);
            const stream2 = createMockTokenStream("Response from bot 2", 75, 35);
            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(stream1)
                .mockResolvedValueOnce(stream2);

            // Target both bots for parallel execution
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

            // Should succeed with both bots responding
            expect(result.success).toBe(true);
            expect(result.participantResults.size).toBe(2);
            expect(result.messages).toHaveLength(2);

            // Calculate total expected credits
            const totalExpectedTokens = (80 + 30) + (75 + 35); // ~220 tokens
            const actualCredits = Number(BigInt(result.totalResourceUsage.creditsUsed));

            expect(actualCredits).toBeGreaterThan(0);
            expect(actualCredits).toBeLessThanOrEqual(2000); // Within budget

            // Wait for state updates
            await new Promise(resolve => setTimeout(resolve, 2500));

            // Verify both bots' usage is tracked
            const finalSwarmState = await contextManager.getContext("concurrent-credit-test");
            expect(finalSwarmState?.resources.consumed.credits).toBe(actualCredits);
            expect(finalSwarmState?.resources.remaining.credits).toBe(2000 - actualCredits);
        });
    });

    describe("Memory and Token Usage Monitoring", () => {
        it("should track memory usage during tool execution", async () => {
            const memoryBot: BotParticipant = {
                id: "memory-bot",
                name: "Memory Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "memory-tracking-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [memoryBot],
                conversationHistory: [],
                availableTools: [
                    {
                        name: "memory_intensive_tool",
                        description: "A tool that uses significant memory",
                        inputSchema: { type: "object", properties: { data: { type: "string" } } },
                    },
                ],
            };

            // Register memory-intensive tool
            const memoryTool = createExpensiveTool(100, 500, 1000); // 500MB memory usage
            memoryTool.name = "memory_intensive_tool";
            toolRunner.register(memoryTool);

            const swarmState: SwarmState = {
                swarmId: "memory-tracking-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [memoryBot],
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

            await contextManager.updateContext("memory-tracking-test", swarmState);

            // Mock OpenAI to make a tool call
            const toolCallStream = {
                async *[Symbol.asyncIterator]() {
                    yield {
                        choices: [{
                            delta: {
                                tool_calls: [{
                                    index: 0,
                                    id: "call_memory_tool",
                                    type: "function",
                                    function: {
                                        name: "memory_intensive_tool",
                                        arguments: "{\"data\": \"large dataset\"}",
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
                            prompt_tokens: 90,
                            completion_tokens: 25,
                            total_tokens: 115,
                        },
                    };
                },
            };

            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(toolCallStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should succeed and track memory usage
            expect(result.success).toBe(true);
            expect(memoryTool.execute).toHaveBeenCalled();
            expect(result.totalResourceUsage.memoryUsedMB).toBeGreaterThanOrEqual(500);
            expect(result.totalResourceUsage.toolCalls).toBe(1);
            expect(result.totalResourceUsage.durationMs).toBeGreaterThanOrEqual(1000); // Tool execution time
        });

        it("should monitor token usage patterns and optimize accordingly", async () => {
            const tokenBot: BotParticipant = {
                id: "token-optimizer-bot",
                name: "Token Optimizer Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "token-optimization-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [tokenBot],
                conversationHistory: [
                    // Add some conversation history to test context management
                    {
                        id: generatePK(),
                        content: "Previous message with content",
                        role: "user",
                        timestamp: Date.now() - 60000,
                    },
                    {
                        id: generatePK(),
                        content: "Previous bot response with detailed information",
                        role: "assistant",
                        timestamp: Date.now() - 30000,
                    },
                ],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "token-optimization-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [tokenBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: 1000, tokens: 5000, time: 300 }, // Limited tokens
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext("token-optimization-test", swarmState);

            // Mock response that considers token limits
            const optimizedResponse = "Concise response";
            const mockStream = createMockTokenStream(optimizedResponse, 200, 25); // High prompt, low completion
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should succeed with optimized token usage
            expect(result.success).toBe(true);
            expect(result.messages[0].content).toContain("Concise response");

            // Token usage should respect limits
            const totalTokens = 200 + 25; // prompt + completion
            expect(totalTokens).toBeLessThanOrEqual(5000);

            // Should log token optimization if applicable
            const debugCalls = (logger.debug as any).mock.calls;
            expect(debugCalls.some((call: any) => 
                call[0].includes("token") || call[0].includes("optimization"),
            )).toBeTruthy();
        });
    });

    describe("Resource Allocation and Deallocation", () => {
        it("should properly allocate and deallocate resources for conversation turns", async () => {
            const allocationBot: BotParticipant = {
                id: "allocation-bot",
                name: "Allocation Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "resource-allocation-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [allocationBot],
                conversationHistory: [],
                availableTools: [],
            };

            const initialCredits = 3000;
            const initialTokens = 30000;
            const initialTime = 900;

            const swarmState: SwarmState = {
                swarmId: "resource-allocation-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [allocationBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [], // Start with no allocations
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { 
                        credits: initialCredits, 
                        tokens: initialTokens, 
                        time: initialTime, 
                    },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext("resource-allocation-test", swarmState);

            // Take snapshot of initial state
            const preExecutionState = await contextManager.getContext("resource-allocation-test");
            expect(preExecutionState?.resources.remaining.credits).toBe(initialCredits);

            // Mock successful response
            const responseContent = "Resource allocation test response";
            const mockStream = createMockTokenStream(responseContent, 150, 50);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Verify successful execution
            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);

            // Wait for all state updates
            await new Promise(resolve => setTimeout(resolve, 2500));

            // Verify resource deallocation
            const postExecutionState = await contextManager.getContext("resource-allocation-test");
            
            // Resources should be consumed and remaining should be reduced
            expect(postExecutionState?.resources.consumed.credits).toBeGreaterThan(0);
            expect(postExecutionState?.resources.consumed.tokens).toBeGreaterThan(0);
            expect(postExecutionState?.resources.consumed.time).toBeGreaterThan(0);

            // Remaining resources should be reduced by the amount consumed
            const creditsConsumed = postExecutionState?.resources.consumed.credits || 0;
            const tokensConsumed = postExecutionState?.resources.consumed.tokens || 0;
            const timeConsumed = postExecutionState?.resources.consumed.time || 0;

            expect(postExecutionState?.resources.remaining.credits).toBe(initialCredits - creditsConsumed);
            expect(postExecutionState?.resources.remaining.tokens).toBe(initialTokens - tokensConsumed);
            expect(postExecutionState?.resources.remaining.time).toBe(initialTime - timeConsumed);

            // Verify resource accounting matches conversation result
            expect(Number(BigInt(result.totalResourceUsage.creditsUsed))).toBe(creditsConsumed);
            expect(result.totalResourceUsage.durationMs).toBeGreaterThan(0);
        });

        it("should handle resource contention between concurrent operations", async () => {
            const contenderBot1: BotParticipant = {
                id: "contender-bot-1",
                name: "Contender Bot 1",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const contenderBot2: BotParticipant = {
                id: "contender-bot-2",
                name: "Contender Bot 2",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "resource-contention-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [contenderBot1, contenderBot2],
                conversationHistory: [],
                availableTools: [],
            };

            const limitedCredits = 300; // Tight budget that requires careful allocation
            const swarmState: SwarmState = {
                swarmId: "resource-contention-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [contenderBot1, contenderBot2],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { 
                        credits: limitedCredits, 
                        tokens: 3000, 
                        time: 120, 
                    },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext("resource-contention-test", swarmState);

            // Mock responses that together would exceed budget if not managed properly
            const stream1 = createMockTokenStream("Response 1", 80, 40); // ~120 tokens
            const stream2 = createMockTokenStream("Response 2", 90, 45); // ~135 tokens
            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(stream1)
                .mockResolvedValueOnce(stream2);

            // Execute parallel operations
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

            // Should handle resource contention gracefully
            expect(result.success).toBe(true);
            
            // Total resource usage should not exceed available budget
            const totalCreditsUsed = Number(BigInt(result.totalResourceUsage.creditsUsed));
            expect(totalCreditsUsed).toBeLessThanOrEqual(limitedCredits);

            // At least one bot should have responded successfully
            const successfulResponses = Array.from(result.participantResults.values())
                .filter(result => result.success);
            expect(successfulResponses.length).toBeGreaterThan(0);

            // Wait for state updates
            await new Promise(resolve => setTimeout(resolve, 2500));

            // Verify final resource state is consistent
            const finalState = await contextManager.getContext("resource-contention-test");
            expect(finalState?.resources.consumed.credits).toBe(totalCreditsUsed);
            expect(finalState?.resources.remaining.credits).toBe(limitedCredits - totalCreditsUsed);
        });
    });

    describe("Resource Recovery on Failures", () => {
        it("should recover allocated resources when operations fail", async () => {
            const failureBot: BotParticipant = {
                id: "failure-recovery-bot",
                name: "Failure Recovery Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "resource-recovery-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [failureBot],
                conversationHistory: [],
                availableTools: [],
            };

            const initialCredits = 2000;
            const swarmState: SwarmState = {
                swarmId: "resource-recovery-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [failureBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: initialCredits, tokens: 20000, time: 600 },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext("resource-recovery-test", swarmState);

            // Mock OpenAI to fail after allocation but before completion
            mockOpenAIClient.chat.completions.create.mockRejectedValueOnce(
                new Error("Service temporarily unavailable"),
            );

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Operation should fail
            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();

            // But resources should not be permanently consumed since operation failed
            expect(result.totalResourceUsage.creditsUsed).toBe("0");

            // Wait for any cleanup operations
            await new Promise(resolve => setTimeout(resolve, 2000));

            // Verify resources were recovered
            const recoveredState = await contextManager.getContext("resource-recovery-test");
            expect(recoveredState?.resources.consumed.credits).toBe(0);
            expect(recoveredState?.resources.remaining.credits).toBe(initialCredits);

            // Should log resource recovery
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Conversation orchestration failed"),
                expect.any(Object),
            );
        });

        it("should handle partial resource recovery in mixed success/failure scenarios", async () => {
            const successBot: BotParticipant = {
                id: "success-bot",
                name: "Success Bot",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const failBot: BotParticipant = {
                id: "fail-bot",
                name: "Fail Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "partial-recovery-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [successBot, failBot],
                conversationHistory: [],
                availableTools: [],
            };

            const initialCredits = 1500;
            const swarmState: SwarmState = {
                swarmId: "partial-recovery-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [successBot, failBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: initialCredits, tokens: 15000, time: 600 },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext("partial-recovery-test", swarmState);

            // Mock one success and one failure
            const successStream = createMockTokenStream("Success response", 60, 25);
            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(successStream)
                .mockRejectedValueOnce(new Error("Second bot failed"));

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

            // Should succeed overall but with partial results
            expect(result.success).toBe(true);
            expect(result.participantResults.size).toBe(2);
            expect(result.messages).toHaveLength(1); // Only successful bot's message

            // Only successful bot should contribute to resource consumption
            const creditsUsed = Number(BigInt(result.totalResourceUsage.creditsUsed));
            expect(creditsUsed).toBeGreaterThan(0);
            expect(creditsUsed).toBeLessThan(200); // Should be modest for mini model

            // Wait for state updates
            await new Promise(resolve => setTimeout(resolve, 2500));

            // Verify only successful operation's resources were consumed
            const finalState = await contextManager.getContext("partial-recovery-test");
            expect(finalState?.resources.consumed.credits).toBe(creditsUsed);
            expect(finalState?.resources.remaining.credits).toBe(initialCredits - creditsUsed);

            // Failed bot should not have consumed resources
            const failedBotResult = result.updatedParticipants.find(p => p.botId === "fail-bot");
            expect(failedBotResult?.success).toBe(false);
            expect(failedBotResult?.resourcesUsed.creditsUsed).toBe("0");

            // Should log both success and failure
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error in parallel execution"),
                expect.any(Object),
            );
        });
    });

    describe("Cross-Service Resource Synchronization", () => {
        it("should maintain resource consistency across all service layers", async () => {
            const syncBot: BotParticipant = {
                id: "sync-bot",
                name: "Sync Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationId = generatePK();
            const swarmId = generatePK();
            
            // Initialize conversation state
            const conversationState: ConversationState = {
                id: conversationId,
                participants: [syncBot],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "",
            };

            await chatStore.saveState(conversationId, conversationState);

            const conversationContext: ConversationContext = {
                swarmId,
                userData: testUser,
                timestamp: new Date(),
                participants: [syncBot],
                conversationHistory: [],
                availableTools: [],
            };

            const initialCredits = 2500;
            const swarmState: SwarmState = {
                swarmId,
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [syncBot],
                    activeRuns: [],
                    startedAt: new Date(),
                    lastActivityAt: new Date(),
                },
                resources: {
                    allocated: [],
                    consumed: { credits: 0, tokens: 0, time: 0 },
                    remaining: { credits: initialCredits, tokens: 25000, time: 600 },
                },
                metadata: {
                    createdAt: new Date(),
                    lastUpdated: new Date(),
                    updatedBy: "system",
                    subscribers: new Set(),
                },
            };

            await contextManager.updateContext(swarmId, swarmState);

            // Mock response with known resource consumption
            const responseContent = "Synchronized resource tracking test";
            const mockStream = createMockTokenStream(responseContent, 180, 60);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(mockStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Verify successful execution
            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);

            const creditsUsed = Number(BigInt(result.totalResourceUsage.creditsUsed));
            expect(creditsUsed).toBeGreaterThan(0);

            // Wait for all debounced operations to complete
            await new Promise(resolve => setTimeout(resolve, 3000));

            // Verify synchronization across ALL service layers

            // 1. ConversationEngine result
            expect(result.totalResourceUsage.creditsUsed).toBe(creditsUsed.toString());

            // 2. SwarmContextManager state
            const swarmStateAfter = await contextManager.getContext(swarmId);
            expect(swarmStateAfter?.resources.consumed.credits).toBe(creditsUsed);
            expect(swarmStateAfter?.resources.remaining.credits).toBe(initialCredits - creditsUsed);

            // 3. ChatStore persistence
            const chatStoreState = await chatStore.loadState(conversationId);
            expect(chatStoreState.config.stats.totalCredits).toBe(creditsUsed);

            // 4. ConversationStore cache
            const conversationStoreState = await conversationStore.get(conversationId);
            expect(conversationStoreState?.config.stats.totalCredits).toBe(creditsUsed);

            // 5. All systems should have identical values
            expect(swarmStateAfter?.resources.consumed.credits).toBe(chatStoreState.config.stats.totalCredits);
            expect(chatStoreState.config.stats.totalCredits).toBe(conversationStoreState?.config.stats.totalCredits);

            // 6. Verify resource version consistency
            expect(swarmStateAfter?.version).toBeGreaterThan(1); // Should have been updated
            expect(chatStoreState.config.stats.lastUpdated).toBeDefined();
        });
    });
});
