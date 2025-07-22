import {
    BotConfig,
    type BotParticipant,
    ChatConfig,
    type ConversationContext,
    type ConversationParams,
    type ConversationState,
    type ConversationTrigger,
    generatePK,
    LlmServiceId,
    type SessionUser,
    type SwarmState,
    type Tool,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DbProvider } from "../../../../db/provider.js";
import { logger } from "../../../../events/logger.js";
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

function createMockToolCallStream(toolCall: { name: string; arguments: string; id: string }) {
    return {
        async *[Symbol.asyncIterator]() {
            yield {
                choices: [{
                    delta: {
                        tool_calls: [{
                            index: 0,
                            id: toolCall.id,
                            type: "function",
                            function: {
                                name: toolCall.name,
                                arguments: toolCall.arguments,
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
                    prompt_tokens: 75,
                    completion_tokens: 30,
                    total_tokens: 105,
                },
            };
        },
    };
}

describe("Conversation Orchestration Integration Tests", () => {
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
    let testUser: SessionUser;

    beforeEach(async () => {
        vi.clearAllMocks();

        // Note: Test containers are already set up by global vitest setup
        // Initialize core services
        mockDb = DbProvider.get();

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

    describe("Multi-Agent Conversation Flows", () => {
        it("should orchestrate a complex research and writing workflow", async () => {
            // Create a team of specialized bots
            const researchBot: BotParticipant = {
                id: "researcher-bot",
                name: "Research Assistant",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "researcher" },
                }),
                state: "ready",
            };

            const writerBot: BotParticipant = {
                id: "writer-bot",
                name: "Content Writer",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "writer" },
                }),
                state: "ready",
            };

            const coordinatorBot: BotParticipant = {
                id: "coordinator-bot",
                name: "Team Coordinator",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "arbitrator" },
                }),
                state: "ready",
            };

            // Set up conversation context
            const conversationContext: ConversationContext = {
                swarmId: "research-workflow",
                userData: testUser,
                timestamp: new Date(),
                participants: [researchBot, writerBot, coordinatorBot],
                conversationHistory: [],
                availableTools: [
                    {
                        name: "search_web",
                        description: "Search the web for information",
                        inputSchema: { type: "object", properties: { query: { type: "string" } } },
                    },
                    {
                        name: "handoff_to_bot",
                        description: "Hand off conversation to another bot",
                        inputSchema: { type: "object", properties: { botId: { type: "string" }, message: { type: "string" } } },
                    },
                ],
                teamConfig: {
                    id: "research-team",
                    name: "Research Team",
                    config: {},
                },
            };

            // Initialize swarm state in context manager
            const initialSwarmState: SwarmState = {
                swarmId: "research-workflow",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [researchBot, writerBot, coordinatorBot],
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

            await contextManager.updateContext("research-workflow", initialSwarmState);

            // Create conversation state for persistence
            const conversationState: ConversationState = {
                id: "research-conv-123",
                participants: [researchBot, writerBot, coordinatorBot],
                config: ChatConfig.default().export(),
                availableTools: conversationContext.availableTools,
                initialLeaderSystemMessage: "You are part of a research team. Coordinate with other bots to complete tasks efficiently.",
                teamConfig: conversationContext.teamConfig,
            };

            await chatStore.saveState(conversationState.id, conversationState);

            // **Phase 1: User initiates research request**
            const userTrigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: undefined, // No specific bot targeting - should use arbitrator
                },
            };

            // Mock coordinator response with handoff
            const coordinatorStream = createMockTextStream([
                "I'll coordinate this research task. ",
                "Let me delegate to our research specialist.",
            ]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(coordinatorStream);

            const conversationParams: ConversationParams = {
                context: conversationContext,
                trigger: userTrigger,
                strategy: "conversation",
                constraints: {
                    maxTurns: 5,
                    maxDurationMs: 30000,
                },
                metadata: {
                    initiatedBy: "user123",
                    requestId: "req-123",
                },
            };

            const phase1Result = await conversationEngine.orchestrateConversation(conversationParams);

            // Verify coordinator was selected (arbitrator role)
            expect(phase1Result.success).toBe(true);
            expect(phase1Result.participantResults.size).toBeGreaterThan(0);
            expect(logger.info).toHaveBeenCalledWith(
                expect.stringContaining("Bot selection completed"),
                expect.objectContaining({
                    strategy: "swarm_baton",
                }),
            );

            // **Phase 2: Handoff to researcher bot**
            // Update active bot to researcher via swarm state
            const updatedSwarmState = await contextManager.getContext("research-workflow");
            if (updatedSwarmState) {
                updatedSwarmState.chatConfig.activeBotId = "researcher-bot";
                await contextManager.updateContext("research-workflow", updatedSwarmState);
            }

            // Mock researcher response with tool usage
            const toolCallStream = createMockToolCallStream({
                name: "search_web",
                arguments: "{\"query\": \"latest AI research trends 2024\"}",
                id: "call_search_123",
            });
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(toolCallStream);

            // Mock tool execution
            const mockTool: Tool = {
                name: "search_web",
                description: "Search the web",
                execute: vi.fn().mockResolvedValue({
                    success: true,
                    result: "Found 10 articles about AI research trends including transformer innovations and multimodal systems.",
                }),
            };
            toolRunner.register(mockTool);

            const researchTrigger: ConversationTrigger = {
                type: "continue",
                lastEvent: {} as any,
            };

            const phase2Params: ConversationParams = {
                ...conversationParams,
                trigger: researchTrigger,
            };

            const phase2Result = await conversationEngine.orchestrateConversation(phase2Params);

            // Verify researcher was active and tool was used
            expect(phase2Result.success).toBe(true);
            expect(mockTool.execute).toHaveBeenCalledWith({ query: "latest AI research trends 2024" });
            expect(phase2Result.totalResourceUsage.toolCalls).toBeGreaterThan(0);

            // **Phase 3: Handoff to writer bot**
            if (updatedSwarmState) {
                updatedSwarmState.chatConfig.activeBotId = "writer-bot";
                await contextManager.updateContext("research-workflow", updatedSwarmState);
            }

            const writerStream = createMockTextStream([
                "Based on the research findings, ",
                "I'll now write a comprehensive summary ",
                "of the latest AI research trends.",
            ]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(writerStream);

            const phase3Result = await conversationEngine.orchestrateConversation(phase2Params);

            // Verify writer was active and produced content
            expect(phase3Result.success).toBe(true);
            expect(phase3Result.participantResults.has("writer-bot")).toBe(true);

            // **Verify overall workflow integrity**
            // Check that all phases maintained state consistency
            const finalConversationState = await chatStore.loadState(conversationState.id);
            expect(finalConversationState.config.stats.totalCredits).toBeGreaterThan(0);
            expect(finalConversationState.config.stats.totalToolCalls).toBeGreaterThan(0);

            // Verify resource tracking across all phases
            const allResourceUsage = [phase1Result, phase2Result, phase3Result]
                .map(r => r.totalResourceUsage)
                .reduce((total, usage) => ({
                    creditsUsed: (BigInt(total.creditsUsed) + BigInt(usage.creditsUsed)).toString(),
                    toolCalls: total.toolCalls + usage.toolCalls,
                    durationMs: total.durationMs + usage.durationMs,
                    stepsExecuted: total.stepsExecuted + usage.stepsExecuted,
                    memoryUsedMB: total.memoryUsedMB + usage.memoryUsedMB,
                }));

            expect(BigInt(allResourceUsage.creditsUsed)).toBeGreaterThan(0n);
            expect(allResourceUsage.toolCalls).toBeGreaterThan(0);
        });

        it("should handle direct bot targeting and override swarm baton", async () => {
            const assistantBot: BotParticipant = {
                id: "assistant-bot",
                name: "General Assistant",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const expertBot: BotParticipant = {
                id: "expert-bot",
                name: "Domain Expert",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "specialist" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "direct-targeting-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [assistantBot, expertBot],
                conversationHistory: [],
                availableTools: [],
            };

            // Set up swarm state with assistant as active bot
            const swarmState: SwarmState = {
                swarmId: "direct-targeting-test",
                version: 1,
                chatConfig: {
                    ...ChatConfig.default().export(),
                    activeBotId: "assistant-bot", // Assistant has the baton
                },
                execution: {
                    status: "running",
                    agents: [assistantBot, expertBot],
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

            await contextManager.updateContext("direct-targeting-test", swarmState);

            // Mock expert bot response
            const expertStream = createMockTextStream([
                "As a domain expert, ",
                "I can provide specialized knowledge on this topic.",
            ]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(expertStream);

            // User directly targets the expert bot (should override active bot)
            const directTargetTrigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: {
                        respondingBots: ["expert-bot"], // Direct targeting
                    } as any,
                },
            };

            const params: ConversationParams = {
                context: conversationContext,
                trigger: directTargetTrigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Verify expert bot was selected despite assistant having the baton
            expect(result.success).toBe(true);
            expect(result.participantResults.has("expert-bot")).toBe(true);
            expect(result.participantResults.has("assistant-bot")).toBe(false);

            // Verify the selection strategy was direct_mention
            expect(logger.info).toHaveBeenCalledWith(
                expect.stringContaining("Bot selection completed"),
                expect.objectContaining({
                    strategy: "direct_mention",
                }),
            );
        });

        it("should handle @all targeting for parallel execution", async () => {
            const bot1: BotParticipant = {
                id: "bot1",
                name: "Bot One",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const bot2: BotParticipant = {
                id: "bot2",
                name: "Bot Two",
                config: BotConfig.parse({
                    model: "gpt-4o-mini",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "parallel-execution-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [bot1, bot2],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "parallel-execution-test",
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

            await contextManager.updateContext("parallel-execution-test", swarmState);

            // Mock responses for both bots
            const bot1Stream = createMockTextStream(["Hello from Bot One!"]);
            const bot2Stream = createMockTextStream(["Hello from Bot Two!"]);
            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(bot1Stream)
                .mockResolvedValueOnce(bot2Stream);

            // User targets all bots
            const allTargetTrigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: {
                        respondingBots: ["@all"],
                    } as any,
                },
            };

            const params: ConversationParams = {
                context: conversationContext,
                trigger: allTargetTrigger,
                strategy: "conversation",
                metadata: {
                    executionMode: "parallel",
                },
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Verify both bots responded
            expect(result.success).toBe(true);
            expect(result.participantResults.size).toBe(2);
            expect(result.participantResults.has("bot1")).toBe(true);
            expect(result.participantResults.has("bot2")).toBe(true);

            // Verify parallel execution was used
            expect(logger.debug).toHaveBeenCalledWith(
                expect.stringContaining("Executing conversation turn"),
                expect.objectContaining({
                    participantCount: 2,
                    executionMode: "parallel",
                }),
            );
        });
    });

    describe("Error Handling and Recovery", () => {
        it("should gracefully handle bot failures and continue with available bots", async () => {
            const workingBot: BotParticipant = {
                id: "working-bot",
                name: "Working Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const failingBot: BotParticipant = {
                id: "failing-bot",
                name: "Failing Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            const conversationContext: ConversationContext = {
                swarmId: "error-recovery-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [workingBot, failingBot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "error-recovery-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [workingBot, failingBot],
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

            await contextManager.updateContext("error-recovery-test", swarmState);

            // Mock one successful response and one failure
            const workingStream = createMockTextStream(["I'm working fine!"]);
            mockOpenAIClient.chat.completions.create
                .mockResolvedValueOnce(workingStream)
                .mockRejectedValueOnce(new Error("Service unavailable"));

            const allTargetTrigger: ConversationTrigger = {
                type: "user_message",
                message: {
                    config: {
                        respondingBots: ["@all"],
                    } as any,
                },
            };

            const params: ConversationParams = {
                context: conversationContext,
                trigger: allTargetTrigger,
                strategy: "conversation",
                metadata: {
                    executionMode: "parallel",
                },
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should succeed overall despite one bot failure
            expect(result.success).toBe(true);
            expect(result.participantResults.has("working-bot")).toBe(true);

            // Should log the error but continue execution
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Error in parallel execution"),
                expect.any(Object),
            );
        });

        it("should handle context manager failures gracefully", async () => {
            const conversationContext: ConversationContext = {
                swarmId: "nonexistent-swarm",
                userData: testUser,
                timestamp: new Date(),
                participants: [],
                conversationHistory: [],
                availableTools: [],
            };

            // Don't initialize swarm state - simulate missing context
            const trigger: ConversationTrigger = { type: "start" };

            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Should handle gracefully with empty bot selection
            expect(result.success).toBe(true);
            expect(result.participantResults.size).toBe(0);
            expect(result.totalResourceUsage.creditsUsed).toBe("0");

            expect(logger.warn).toHaveBeenCalledWith(
                expect.stringContaining("No unified context found"),
                expect.any(Object),
            );
        });
    });

    describe("State Consistency and Persistence", () => {
        it("should maintain conversation state consistency across service boundaries", async () => {
            const bot: BotParticipant = {
                id: "persistence-bot",
                name: "Persistence Bot",
                config: BotConfig.parse({
                    model: "gpt-4o",
                    agentSpec: { role: "assistant" },
                }),
                state: "ready",
            };

            // Initialize conversation state
            const conversationState: ConversationState = {
                id: generatePK(),
                participants: [bot],
                config: ChatConfig.default().export(),
                availableTools: [],
                initialLeaderSystemMessage: "Test system message",
            };

            await chatStore.saveState(conversationState.id, conversationState);

            const conversationContext: ConversationContext = {
                swarmId: "persistence-test",
                userData: testUser,
                timestamp: new Date(),
                participants: [bot],
                conversationHistory: [],
                availableTools: [],
            };

            const swarmState: SwarmState = {
                swarmId: "persistence-test",
                version: 1,
                chatConfig: ChatConfig.default().export(),
                execution: {
                    status: "running",
                    agents: [bot],
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

            await contextManager.updateContext("persistence-test", swarmState);

            // Mock successful response
            const responseStream = createMockTextStream(["Test response for persistence"]);
            mockOpenAIClient.chat.completions.create.mockResolvedValueOnce(responseStream);

            const trigger: ConversationTrigger = { type: "start" };
            const params: ConversationParams = {
                context: conversationContext,
                trigger,
                strategy: "conversation",
            };

            const result = await conversationEngine.orchestrateConversation(params);

            // Verify orchestration succeeded
            expect(result.success).toBe(true);

            // Check state consistency across all stores

            // 1. ChatStore should have updated stats
            const updatedConversationState = await chatStore.loadState(conversationState.id);
            expect(updatedConversationState.config.stats.totalCredits).toBeGreaterThan(0);

            // 2. ConversationStore cache should be consistent
            const cachedState = await conversationStore.get(conversationState.id);
            expect(cachedState?.config.stats.totalCredits).toBe(
                updatedConversationState.config.stats.totalCredits,
            );

            // 3. SwarmContextManager should maintain resource tracking
            const finalSwarmState = await contextManager.getContext("persistence-test");
            expect(finalSwarmState).toBeTruthy();
            expect(finalSwarmState?.resources.consumed.credits).toBeGreaterThan(0);

            // 4. Resource usage should be consistent across all systems
            expect(BigInt(result.totalResourceUsage.creditsUsed)).toBeGreaterThan(0n);
        });
    });
});
