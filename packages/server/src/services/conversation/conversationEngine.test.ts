import type {
    BotParticipant,
    ConversationContext,
    ConversationEngineConfig,
    ConversationParams,
    ConversationTrigger,
    ExecutionStrategy,
    ResponseResult,
    SessionUser,
    SwarmId
} from "@vrooli/shared";
import { generatePK, ModelStrategy } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ISwarmContextManager } from "../execution/shared/SwarmContextManager.js";
import type { ResponseService } from "../response/responseService.js";
import type { AgentSelectionResult } from "./agentGraph.js";
import { ConversationEngine } from "./conversationEngine.js";

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

    function createSampleUserData(): SessionUser {
        return {
            __typename: "SessionUser",
            id: generatePK().toString(),
            name: "Test User",
            credits: "1000",
            creditAccountId: null,
            creditSettings: null,
            handle: "testuser",
            hasPremium: false,
            hasReceivedPhoneVerificationReward: false,
            languages: ["en"],
            phoneNumberVerified: false,
            profileImage: null,
            publicId: generatePK().toString(),
            session: {
                __typename: "SessionUserSession",
                id: generatePK().toString(),
                lastRefreshAt: new Date().toISOString(),
            },
            theme: null,
        };
    }

    function createSampleBotParticipant(): BotParticipant {
        return {
            id: generatePK().toString(),
            name: "Test Bot",
            handle: "testbot",
            config: {
                __version: "1.0",
                resources: [],
                modelConfig: {
                    strategy: ModelStrategy.FIXED,
                    preferredModel: "gpt-4o",
                    offlineOnly: false,
                },
                agentSpec: {
                    role: "assistant",
                },
            },
            state: "ready",
            isBotDeprecated: false,
            user: {
                id: generatePK().toString(),
                name: "Test Bot User",
            },
            you: {
                id: generatePK().toString(),
                isInvited: true,
                isViewed: false,
                canDelete: false,
                canUpdate: false,
                canRead: true,
            },
        };
    }

    function createSampleConversationContext(): ConversationContext {
        const userData = createSampleUserData();
        const botParticipant = createSampleBotParticipant();
        return {
            swarmId: generatePK().toString() as SwarmId,
            userData,
            timestamp: new Date(),
            participants: [botParticipant],
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
    }

    function createSampleTrigger(userData: SessionUser): ConversationTrigger {
        return {
            type: "user_message",
            data: {
                messageId: generatePK().toString(),
                content: "Hello",
                userId: userData.id,
            },
        };
    }

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
        let sampleConversationContext: ConversationContext;
        let sampleUserData: SessionUser;
        let sampleBotParticipant: BotParticipant;
        let sampleTrigger: ConversationTrigger;

        beforeEach(() => {
            sampleConversationContext = createSampleConversationContext();
            sampleUserData = sampleConversationContext.userData;
            sampleBotParticipant = sampleConversationContext.participants[0];
            sampleTrigger = createSampleTrigger(sampleUserData);

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
                    id: generatePK().toString(),
                    text: "Hello! How can I help you?",
                    config: { __version: "1.0", role: "assistant" },
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
                id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
                    id: generatePK().toString(),
                    text: "I'll search for that information.",
                    config: { __version: "1.0", role: "assistant" },
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
                        id: generatePK().toString(),
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
                id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
        let sampleConversationContext: ConversationContext;
        let sampleBotParticipant: BotParticipant;
        let sampleTrigger: ConversationTrigger;

        beforeEach(() => {
            sampleConversationContext = createSampleConversationContext();
            sampleBotParticipant = sampleConversationContext.participants[0];
            sampleTrigger = createSampleTrigger(sampleConversationContext.userData);

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
        let sampleConversationContext: ConversationContext;
        let sampleTrigger: ConversationTrigger;

        beforeEach(() => {
            sampleConversationContext = createSampleConversationContext();
            sampleTrigger = createSampleTrigger(sampleConversationContext.userData);
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
            invalidContext.swarmId = generatePK().toString() as SwarmId; // Keep valid swarmId
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
            invalidContext.swarmId = generatePK().toString() as SwarmId; // Keep valid swarmId
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
            invalidContext.swarmId = generatePK().toString() as SwarmId; // Keep valid swarmId
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
            invalidContext.swarmId = generatePK().toString() as SwarmId; // Keep valid swarmId
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
        let sampleConversationContext: ConversationContext;
        let sampleBotParticipant: BotParticipant;
        let sampleTrigger: ConversationTrigger;

        beforeEach(() => {
            sampleConversationContext = createSampleConversationContext();
            sampleBotParticipant = sampleConversationContext.participants[0];
            sampleTrigger = createSampleTrigger(sampleConversationContext.userData);

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
                    id: generatePK().toString(),
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
                responders: [sampleBotParticipant, { ...sampleBotParticipant, id: generatePK().toString() }],
                strategy: "reasoning",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.metadata?.executionMode).toBe("sequential");
        });

        it("should use sequential mode for deterministic strategy", async () => {
            conversationParams.strategy = "deterministic";

            mockSelectResponders.mockResolvedValue({
                responders: [sampleBotParticipant, { ...sampleBotParticipant, id: generatePK().toString() }],
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
                responders: [sampleBotParticipant, { ...sampleBotParticipant, id: generatePK().toString() }],
                strategy: "multi_bot",
            });

            const result = await conversationEngine.orchestrateConversation(conversationParams);

            expect(result.metadata?.executionMode).toBe("parallel");
        });
    });

    describe("error handling", () => {
        let sampleConversationContext: ConversationContext;
        let sampleBotParticipant: BotParticipant;
        let sampleTrigger: ConversationTrigger;

        beforeEach(() => {
            sampleConversationContext = createSampleConversationContext();
            sampleBotParticipant = sampleConversationContext.participants[0];
            sampleTrigger = createSampleTrigger(sampleConversationContext.userData);
        });

        it("should handle partial failures in parallel execution", async () => {
            const secondBot: BotParticipant = {
                ...sampleBotParticipant,
                id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
        let sampleConversationContext: ConversationContext;
        let sampleBotParticipant: BotParticipant;
        let sampleTrigger: ConversationTrigger;

        beforeEach(() => {
            sampleConversationContext = createSampleConversationContext();
            sampleBotParticipant = sampleConversationContext.participants[0];
            sampleTrigger = createSampleTrigger(sampleConversationContext.userData);
        });

        it("should correctly aggregate resources from multiple bots", async () => {
            const secondBot: BotParticipant = {
                ...sampleBotParticipant,
                id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
                        id: generatePK().toString(),
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
