import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import type {
    BotParticipant,
    ConversationContext,
    ConversationEngineConfig,
    ConversationParams,
    ConversationTrigger,
    ExecutionStrategy,
    MessageState,
    ResponseResult,
    SwarmId,
    UserData,
} from "@vrooli/shared";
import { toTurnId } from "@vrooli/shared";
import { ConversationEngine } from "./conversationEngine.js";
import type { ResponseService } from "../response/responseService.js";
import type { ISwarmContextManager } from "../execution/shared/SwarmContextManager.js";
import type { AgentSelectionResult } from "./agentGraph.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

const mockSelectResponders = vi.fn();

vi.mock("./agentGraph.js", () => ({
    CompositeGraph: vi.fn().mockImplementation(() => ({
        selectResponders: mockSelectResponders,
    })),
}));

describe("ConversationEngine", () => {
    let conversationEngine: ConversationEngine;
    let mockResponseService: jest.Mocked<ResponseService>;
    let mockContextManager: jest.Mocked<ISwarmContextManager>;

    const defaultConfig: ConversationEngineConfig = {
        maxConcurrentTurns: 5,
        defaultTimeoutMs: 300000,
        fallbackBehavior: "most_capable",
    };

    const sampleUserData: UserData = {
        id: "user-123",
        name: "Test User",
        email: "test@example.com",
        language: "en",
        timezone: "UTC",
    };

    const sampleBotParticipant: BotParticipant = {
        id: "bot-456",
        name: "Test Bot",
        handle: "testbot",
        config: {
            persona: "Helpful assistant",
            role: "assistant",
            capabilities: ["chat", "search"],
            model: "gpt-4o",
        },
        state: "ready",
        isBotDeprecated: false,
        user: {
            id: "bot-user-456",
            name: "Test Bot User",
        },
        you: {
            id: "rel-123",
            isInvited: true,
            isViewed: false,
            canDelete: false,
            canUpdate: false,
            canRead: true,
        },
    };

    const sampleConversationContext: ConversationContext = {
        swarmId: "swarm-789" as SwarmId,
        userData: sampleUserData,
        participants: [sampleBotParticipant],
        conversationHistory: [],
        availableTools: [
            {
                name: "search",
                description: "Search for information",
                parameters: {
                    type: "object",
                    properties: {
                        query: { type: "string" },
                    },
                    required: ["query"],
                },
            },
        ],
    };

    const sampleTrigger: ConversationTrigger = {
        type: "user_message",
        data: {
            messageId: "msg-123",
            content: "Hello",
            userId: sampleUserData.id,
        },
    };

    beforeEach(() => {
        // Reset all mocks
        vi.clearAllMocks();

        // Create mock ResponseService
        mockResponseService = {
            generateResponse: vi.fn(),
            generateResponseStreaming: vi.fn(),
            safeInputCheck: vi.fn(),
            getServiceConfig: vi.fn(),
            updateState: vi.fn(),
            getNativeToolCapabilities: vi.fn(),
        } as any;

        // Create mock SwarmContextManager
        mockContextManager = {
            getContext: vi.fn(),
            updateContext: vi.fn(),
            allocateResources: vi.fn(),
            releaseResources: vi.fn(),
            cleanup: vi.fn(),
        } as any;

        // Initialize ConversationEngine
        conversationEngine = new ConversationEngine(
            mockResponseService,
            mockContextManager,
            defaultConfig,
        );
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("constructor", () => {
        it("should initialize with default config", () => {
            expect(conversationEngine).toBeInstanceOf(ConversationEngine);
        });

        it("should merge custom config with defaults", () => {
            const customConfig: Partial<ConversationEngineConfig> = {
                maxConcurrentTurns: 10,
                defaultTimeoutMs: 600000,
            };

            const engine = new ConversationEngine(
                mockResponseService,
                mockContextManager,
                customConfig,
            );

            expect(engine).toBeInstanceOf(ConversationEngine);
        });
    });

    describe("orchestrateConversation", () => {
        let conversationParams: ConversationParams;

        beforeEach(() => {
            conversationParams = {
                context: sampleConversationContext,
                strategy: "conversation" as ExecutionStrategy,
                trigger: sampleTrigger,
            };

            // Setup default mocks
            mockContextManager.getContext.mockResolvedValue({
                swarmId: sampleConversationContext.swarmId,
                participants: [sampleBotParticipant],
                sharedState: {},
                executionHistory: [],
                resourceAllocation: {},
            });

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant],
                strategy: "direct",
            } as AgentSelectionResult);

            const mockResponseResult: ResponseResult = {
                success: true,
                message: {
                    id: "response-msg-123",
                    text: "Hello! How can I help you?",
                    config: { role: "assistant" },
                    userId: sampleBotParticipant.id,
                    language: "en",
                },
                resourcesUsed: {
                    creditsUsed: "100",
                    durationMs: 1500,
                    toolCalls: 0,
                    memoryUsedMB: 50,
                    stepsExecuted: 1,
                },
                confidence: 0.95,
                continueConversation: false,
            };

            mockResponseService.generateResponse.mockResolvedValue(mockResponseResult);
        });

        it("should orchestrate a successful conversation", async () => {
            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(1);
            expect(result.messages[0].text).toBe("Hello! How can I help you?");
            expect(result.updatedParticipants).toHaveLength(1);
            expect(result.conversationComplete).toBe(false);
            expect(result.swarmId).toBe(sampleConversationContext.swarmId);
            expect(result.resourcesUsed.creditsUsed).toBe("100");
            expect(result.nextAction?.type).toBe("wait_for_user");
        });

        it("should handle reasoning strategy with sequential execution", async () => {
            conversationParams.strategy = "reasoning";

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.metadata?.executionMode).toBe("sequential");
        });

        it("should handle deterministic strategy with sequential execution", async () => {
            conversationParams.strategy = "deterministic";

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.metadata?.executionMode).toBe("sequential");
        });

        it("should handle parallel execution with multiple bots", async () => {
            const secondBot: BotParticipant = {
                ...sampleBotParticipant,
                id: "bot-789",
                name: "Second Bot",
            };

            conversationParams.context.participants = [sampleBotParticipant, secondBot];

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, secondBot],
                strategy: "multi_bot",
            });

            // Mock responses for both bots
            mockResponseService.generateResponse
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "response-1",
                        text: "Response from bot 1",
                        config: { role: "assistant" },
                        userId: sampleBotParticipant.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "50",
                        durationMs: 1000,
                        toolCalls: 0,
                        memoryUsedMB: 25,
                        stepsExecuted: 1,
                    },
                    confidence: 0.9,
                })
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "response-2",
                        text: "Response from bot 2",
                        config: { role: "assistant" },
                        userId: secondBot.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "75",
                        durationMs: 1200,
                        toolCalls: 0,
                        memoryUsedMB: 30,
                        stepsExecuted: 1,
                    },
                    confidence: 0.85,
                });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(2);
            expect(result.resourcesUsed.creditsUsed).toBe("125"); // 50 + 75
            expect(result.metadata?.executionMode).toBe("parallel");
        });

        it("should handle bot selection failure gracefully", async () => {
            mockSelectResponders.mockResolvedValue({
                responders: [],
                strategy: "fallback",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.messages).toHaveLength(0);
            expect(result.metadata?.botSelection?.fallbackUsed).toBe(true);
        });

        it("should handle tool calls and continue conversation", async () => {
            const mockResponseWithTools: ResponseResult = {
                success: true,
                message: {
                    id: "response-with-tools",
                    text: "I'll search for that information.",
                    config: { role: "assistant" },
                    userId: sampleBotParticipant.id,
                    language: "en",
                },
                resourcesUsed: {
                    creditsUsed: "150",
                    durationMs: 2000,
                    toolCalls: 1,
                    memoryUsedMB: 60,
                    stepsExecuted: 2,
                },
                toolCalls: [
                    {
                        id: "tool-call-123",
                        function: {
                            name: "search",
                            arguments: "{\"query\": \"test query\"}",
                        },
                        result: "Search results here",
                    },
                ],
                confidence: 0.92,
                continueConversation: true,
            };

            mockResponseService.generateResponse.mockResolvedValue(mockResponseWithTools);

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.nextAction?.type).toBe("continue");
            expect(result.nextAction?.reason).toContain("Tool calls");
            expect(result.resourcesUsed.toolCalls).toBe(1);
        });

        it("should handle context validation errors", async () => {
            conversationParams.context.swarmId = "" as SwarmId;

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("CONVERSATION_ORCHESTRATION_FAILED");
            expect(result.error?.message).toContain("Missing swarmId");
            expect(result.conversationComplete).toBe(true);
        });

        it("should handle response service errors", async () => {
            mockResponseService.generateResponse.mockRejectedValue(
                new Error("Response service error"),
            );

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("CONVERSATION_ORCHESTRATION_FAILED");
            expect(result.conversationComplete).toBe(true);
        });

        it("should handle context manager errors", async () => {
            mockContextManager.getContext.mockRejectedValue(
                new Error("Context manager error"),
            );

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(false);
            expect(result.error?.code).toBe("CONVERSATION_ORCHESTRATION_FAILED");
        });

        it("should calculate correct average confidence", async () => {
            const secondBot: BotParticipant = {
                ...sampleBotParticipant,
                id: "bot-789",
            };

            conversationParams.context.participants = [sampleBotParticipant, secondBot];

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, secondBot],
                strategy: "multi_bot",
            });

            mockResponseService.generateResponse
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "response-1",
                        text: "Response 1",
                        config: { role: "assistant" },
                        userId: sampleBotParticipant.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "50",
                        durationMs: 1000,
                        toolCalls: 0,
                        memoryUsedMB: 25,
                        stepsExecuted: 1,
                    },
                    confidence: 0.8,
                })
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "response-2",
                        text: "Response 2",
                        config: { role: "assistant" },
                        userId: secondBot.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "50",
                        durationMs: 1000,
                        toolCalls: 0,
                        memoryUsedMB: 25,
                        stepsExecuted: 1,
                    },
                    confidence: 0.6,
                });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.metadata?.turnMetrics?.averageConfidence).toBe(0.7); // (0.8 + 0.6) / 2
        });
    });

    describe("selectRespondingBots", () => {
        beforeEach(() => {
            mockContextManager.getContext.mockResolvedValue({
                swarmId: sampleConversationContext.swarmId,
                participants: [sampleBotParticipant],
                sharedState: {},
                executionHistory: [],
                resourceAllocation: {},
            });
        });

        it("should select bots successfully", async () => {
            const mockSelectionResult: AgentSelectionResult = {
                responders: [sampleBotParticipant],
                strategy: "direct",
            };

            mockSelectResponders.mockResolvedValue(mockSelectionResult);

            const result = await conversationEngine.selectRespondingBots(
                sampleTrigger,
                sampleConversationContext,
            );

            expect(result.responders).toHaveLength(1);
            expect(result.responders[0]).toBe(sampleBotParticipant);
            expect(result.strategy).toBe("direct");
        });

        it("should handle missing context gracefully", async () => {
            mockContextManager.getContext.mockResolvedValue(null);

            const result = await conversationEngine.selectRespondingBots(
                sampleTrigger,
                sampleConversationContext,
            );

            expect(result.responders).toHaveLength(0);
            expect(result.strategy).toBe("fallback");
        });

        it("should handle agent graph errors", async () => {
            mockSelectResponders.mockRejectedValue(
                new Error("Agent graph error"),
            );

            const result = await conversationEngine.selectRespondingBots(
                sampleTrigger,
                sampleConversationContext,
            );

            expect(result.responders).toHaveLength(0);
            expect(result.strategy).toBe("fallback");
        });
    });

    describe("context validation", () => {
        let invalidContext: ConversationContext;

        beforeEach(() => {
            invalidContext = { ...sampleConversationContext };
        });

        it("should throw error for missing swarmId", async () => {
            invalidContext.swarmId = "" as SwarmId;

            const params: ConversationParams = {
                context: invalidContext,
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            const result = await conversationEngine.orchestrateConversation(params);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Missing swarmId");
        });

        it("should throw error for missing userData", async () => {
            invalidContext.swarmId = "valid-swarm" as SwarmId; // Keep valid swarmId
            invalidContext.userData = undefined as any;

            const params: ConversationParams = {
                context: invalidContext,
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            const result = await conversationEngine.orchestrateConversation(params);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Missing userData");
        });

        it("should throw error for invalid participants", async () => {
            invalidContext.swarmId = "valid-swarm" as SwarmId; // Keep valid swarmId
            invalidContext.userData = sampleUserData; // Keep valid userData
            invalidContext.participants = undefined as any;

            const params: ConversationParams = {
                context: invalidContext,
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            const result = await conversationEngine.orchestrateConversation(params);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Invalid participants");
        });

        it("should throw error for invalid conversationHistory", async () => {
            invalidContext.swarmId = "valid-swarm" as SwarmId; // Keep valid swarmId
            invalidContext.userData = sampleUserData; // Keep valid userData
            invalidContext.participants = [sampleBotParticipant]; // Keep valid participants
            invalidContext.conversationHistory = undefined as any;

            const params: ConversationParams = {
                context: invalidContext,
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            const result = await conversationEngine.orchestrateConversation(params);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Invalid conversationHistory");
        });

        it("should throw error for invalid availableTools", async () => {
            invalidContext.swarmId = "valid-swarm" as SwarmId; // Keep valid swarmId
            invalidContext.userData = sampleUserData; // Keep valid userData
            invalidContext.participants = [sampleBotParticipant]; // Keep valid participants
            invalidContext.conversationHistory = []; // Keep valid conversationHistory
            invalidContext.availableTools = undefined as any;

            const params: ConversationParams = {
                context: invalidContext,
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            const result = await conversationEngine.orchestrateConversation(params);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Invalid availableTools");
        });
    });

    describe("execution mode determination", () => {
        let conversationParams: ConversationParams;

        beforeEach(() => {
            conversationParams = {
                context: sampleConversationContext,
                strategy: "conversation" as ExecutionStrategy,
                trigger: sampleTrigger,
            };

            mockContextManager.getContext.mockResolvedValue({
                swarmId: sampleConversationContext.swarmId,
                participants: [sampleBotParticipant],
                sharedState: {},
                executionHistory: [],
                resourceAllocation: {},
            });

            mockResponseService.generateResponse.mockResolvedValue({
                success: true,
                message: {
                    id: "response",
                    text: "Test response",
                    config: { role: "assistant" },
                    userId: sampleBotParticipant.id,
                    language: "en",
                },
                resourcesUsed: {
                    creditsUsed: "100",
                    durationMs: 1000,
                    toolCalls: 0,
                    memoryUsedMB: 50,
                    stepsExecuted: 1,
                },
            });
        });

        it("should use sequential mode for reasoning strategy", async () => {
            conversationParams.strategy = "reasoning";

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, { ...sampleBotParticipant, id: "bot-2" }],
                strategy: "reasoning",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.metadata?.executionMode).toBe("sequential");
        });

        it("should use sequential mode for deterministic strategy", async () => {
            conversationParams.strategy = "deterministic";

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, { ...sampleBotParticipant, id: "bot-2" }],
                strategy: "deterministic",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.metadata?.executionMode).toBe("sequential");
        });

        it("should use sequential mode for single bot", async () => {
            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant],
                strategy: "single",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.metadata?.executionMode).toBe("sequential");
        });

        it("should use parallel mode for conversation strategy with multiple bots", async () => {
            conversationParams.strategy = "conversation";

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, { ...sampleBotParticipant, id: "bot-2" }],
                strategy: "multi_bot",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.metadata?.executionMode).toBe("parallel");
        });
    });

    describe("error handling", () => {
        it("should handle partial failures in parallel execution", async () => {
            const secondBot: BotParticipant = {
                ...sampleBotParticipant,
                id: "bot-failure",
            };

            const conversationParams: ConversationParams = {
                context: {
                    ...sampleConversationContext,
                    participants: [sampleBotParticipant, secondBot],
                },
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            mockContextManager.getContext.mockResolvedValue({
                swarmId: sampleConversationContext.swarmId,
                participants: [sampleBotParticipant, secondBot],
                sharedState: {},
                executionHistory: [],
                resourceAllocation: {},
            });

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, secondBot],
                strategy: "multi_bot",
            });

            // First bot succeeds, second bot fails
            mockResponseService.generateResponse
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "success-response",
                        text: "Success response",
                        config: { role: "assistant" },
                        userId: sampleBotParticipant.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "100",
                        durationMs: 1000,
                        toolCalls: 0,
                        memoryUsedMB: 50,
                        stepsExecuted: 1,
                    },
                })
                .mockRejectedValueOnce(new Error("Bot failure"));

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true); // Should still succeed with partial results
            expect(result.messages).toHaveLength(1); // Only successful response
            expect(result.updatedParticipants).toHaveLength(2); // Both participants tracked
        });

        it("should handle failures in sequential execution", async () => {
            const conversationParams: ConversationParams = {
                context: sampleConversationContext,
                strategy: "reasoning", // Forces sequential
                trigger: sampleTrigger,
            };

            mockContextManager.getContext.mockResolvedValue({
                swarmId: sampleConversationContext.swarmId,
                participants: [sampleBotParticipant],
                sharedState: {},
                executionHistory: [],
                resourceAllocation: {},
            });

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant],
                strategy: "reasoning",
            });

            mockResponseService.generateResponse.mockResolvedValue({
                success: false,
                error: new Error("Sequential bot failure"),
                resourcesUsed: {
                    creditsUsed: "0",
                    durationMs: 500,
                    toolCalls: 0,
                    memoryUsedMB: 0,
                    stepsExecuted: 0,
                },
            } as ResponseResult);

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true); // Should still complete
            expect(result.messages).toHaveLength(0); // No successful messages
            expect(result.updatedParticipants[0].success).toBe(false);
        });
    });

    describe("resource aggregation", () => {
        it("should correctly aggregate resources from multiple bots", async () => {
            const secondBot: BotParticipant = {
                ...sampleBotParticipant,
                id: "bot-2",
            };

            const conversationParams: ConversationParams = {
                context: {
                    ...sampleConversationContext,
                    participants: [sampleBotParticipant, secondBot],
                },
                strategy: "conversation",
                trigger: sampleTrigger,
            };

            mockContextManager.getContext.mockResolvedValue({
                swarmId: sampleConversationContext.swarmId,
                participants: [sampleBotParticipant, secondBot],
                sharedState: {},
                executionHistory: [],
                resourceAllocation: {},
            });

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, secondBot],
                strategy: "multi_bot",
            });

            mockResponseService.generateResponse
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "response-1",
                        text: "Response 1",
                        config: { role: "assistant" },
                        userId: sampleBotParticipant.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "150",
                        durationMs: 1000,
                        toolCalls: 2,
                        memoryUsedMB: 75,
                        stepsExecuted: 3,
                    },
                })
                .mockResolvedValueOnce({
                    success: true,
                    message: {
                        id: "response-2",
                        text: "Response 2",
                        config: { role: "assistant" },
                        userId: secondBot.id,
                        language: "en",
                    },
                    resourcesUsed: {
                        creditsUsed: "250",
                        durationMs: 1500,
                        toolCalls: 1,
                        memoryUsedMB: 100,
                        stepsExecuted: 2,
                    },
                });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.success).toBe(true);
            expect(result.resourcesUsed.creditsUsed).toBe("400"); // 150 + 250
            expect(result.resourcesUsed.toolCalls).toBe(3); // 2 + 1
            expect(result.resourcesUsed.memoryUsedMB).toBe(175); // 75 + 100
            expect(result.resourcesUsed.stepsExecuted).toBe(5); // 3 + 2
        });
    });
});
