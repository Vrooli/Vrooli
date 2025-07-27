import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { EventInterceptor } from "./EventInterceptor.js";
import { DefaultDecisionMaker } from "./BotPriority.js";
import { SwarmStateAccessor } from "../execution/shared/SwarmStateAccessor.js";
import { RoutineExecutor } from "../execution/tier2/routineExecutor.js";
import { StepExecutor } from "../execution/tier3/stepExecutor.js";
import { getEventBehavior } from "./registry.js";
import { aggregateProgression, aggregateReasons } from "./publisher.js";
import { logger } from "../../events/logger.js";
import { extractChatId } from "./types.js";
import type { 
    BotParticipant, 
    ServiceEvent, 
    SwarmState, 
    BehaviourSpec,
    RoutineAction,
    InvokeAction,
    TierExecutionRequest,
    RoutineExecutionInput,
} from "@vrooli/shared";
import { generatePK } from "@vrooli/shared";
import type { 
    ILockService, 
    ISwarmContextManager, 
    BotEventResponse,
    InterceptionResult,
    BotDecisionContext,
} from "./types.js";

// Mock dependencies
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("./BotPriority.js", () => ({
    DefaultDecisionMaker: vi.fn(),
}));

vi.mock("../execution/shared/SwarmStateAccessor.js", () => ({
    SwarmStateAccessor: vi.fn(),
}));

vi.mock("../execution/tier2/routineExecutor.js", () => ({
    RoutineExecutor: vi.fn(),
}));

vi.mock("../execution/tier3/stepExecutor.js", () => ({
    StepExecutor: vi.fn(),
}));

vi.mock("./registry.js", () => ({
    getEventBehavior: vi.fn(),
}));

vi.mock("./publisher.js", () => ({
    aggregateProgression: vi.fn(),
    aggregateReasons: vi.fn(),
}));

vi.mock("./types.js", () => ({
    extractChatId: vi.fn(),
}));

// Mock JEXL
vi.mock("jexl", () => ({
    default: {
        eval: vi.fn(),
    },
}));

describe("EventInterceptor", () => {
    let eventInterceptor: EventInterceptor;
    let mockLockService: ILockService;
    let mockContextManager: ISwarmContextManager;
    let mockStateAccessor: any;
    let mockDecisionMaker: any;
    let mockRoutineExecutor: any;
    let mockStepExecutor: any;

    // Sample test data
    const botId = generatePK().toString();
    const sampleBot: BotParticipant = {
        id: botId,
        name: "Test Bot",
        role: "participant",
        config: {
            agentSpec: {
                role: "coordinator",
                behaviors: [
                    {
                        trigger: {
                            topic: "chat/message",
                            when: "event.type === 'chat/message'",
                            progression: {
                                control: "conditional",
                                condition: "result.approved === true",
                                exclusive: false,
                            },
                        },
                        action: {
                            type: "routine",
                            routineId: generatePK().toString(),
                            label: "Process Message",
                            inputMap: { message: "event.data.message" },
                        },
                    },
                ],
                subscriptions: ["chat/*", "user/login"],
            },
        },
    };

    const eventId = generatePK().toString();
    const chatId = generatePK().toString();
    const swarmId = generatePK().toString();
    
    const sampleEvent: ServiceEvent = {
        id: eventId,
        type: "chat/message",
        timestamp: new Date(),
        data: { message: "Hello world", chatId },
    };

    const sampleSwarmState: SwarmState = {
        id: swarmId,
        name: "Test Swarm",
        participants: [sampleBot],
        goals: [],
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
    };

    beforeEach(() => {
        vi.clearAllMocks();

        // Setup lock service mock
        const mockLock = {
            release: vi.fn().mockResolvedValue(undefined),
        };
        mockLockService = {
            acquire: vi.fn().mockResolvedValue(mockLock),
            release: vi.fn().mockResolvedValue(undefined),
        };

        // Setup context manager mock
        mockContextManager = {
            allocateResources: vi.fn().mockResolvedValue({
                id: generatePK().toString(),
                limits: {
                    maxCredits: "1000",
                    maxDurationMs: 300000,
                    maxMemoryMB: 256,
                    maxConcurrentSteps: 3,
                },
            }),
            releaseResources: vi.fn().mockResolvedValue(undefined),
        } as any;

        // Setup state accessor mock
        mockStateAccessor = {
            buildTriggerContext: vi.fn().mockReturnValue({
                event: sampleEvent,
                bot: sampleBot,
                swarmState: sampleSwarmState,
            }),
        };
        vi.mocked(SwarmStateAccessor).mockImplementation(() => mockStateAccessor);

        // Setup decision maker mock
        mockDecisionMaker = {
            decide: vi.fn().mockResolvedValue({
                shouldHandle: true,
                response: {
                    progression: "continue",
                    reason: "Bot approved event",
                },
            }),
        };
        vi.mocked(DefaultDecisionMaker).mockImplementation(() => mockDecisionMaker);

        // Setup routine executor mock
        const mockStateMachine = {
            stop: vi.fn().mockResolvedValue({ success: true }),
        };
        mockRoutineExecutor = {
            execute: vi.fn().mockResolvedValue({
                success: true,
                result: { output: "routine completed" },
            }),
            getStateMachine: vi.fn().mockReturnValue(mockStateMachine),
        };
        vi.mocked(RoutineExecutor).mockImplementation(() => mockRoutineExecutor);

        // Setup step executor mock
        mockStepExecutor = {
            executeLLMStep: vi.fn().mockResolvedValue({
                response: "Strategy executed successfully",
            }),
        };
        vi.mocked(StepExecutor).mockImplementation(() => mockStepExecutor);

        // Setup registry mock
        vi.mocked(getEventBehavior).mockReturnValue({
            mode: 1, // APPROVAL mode
            interceptable: true,
            barrierConfig: {
                quorum: 1,
                timeoutMs: 5000,
                timeoutAction: "block",
                blockOnFirst: false,
            },
        });

        // Setup publisher mocks
        vi.mocked(aggregateProgression).mockReturnValue("continue");
        vi.mocked(aggregateReasons).mockReturnValue("Aggregated reasons");

        // Setup extractChatId mock
        vi.mocked(extractChatId).mockReturnValue(chatId);

        // Mock RoutineExecutor factory
        const mockRoutineExecutorFactory = vi.fn().mockResolvedValue({
            start: vi.fn().mockResolvedValue(undefined),
            stop: vi.fn().mockResolvedValue({ success: true }),
            getState: vi.fn().mockReturnValue("READY"),
            getStateMachine: vi.fn().mockReturnValue({
                stop: vi.fn().mockResolvedValue({ success: true }),
            }),
        });

        eventInterceptor = new EventInterceptor(mockLockService, mockContextManager, mockRoutineExecutorFactory);
    });

    afterEach(async () => {
        await eventInterceptor.stopAllActiveExecutions();
    });

    describe("constructor", () => {
        it("should initialize with required dependencies", () => {
            expect(eventInterceptor).toBeInstanceOf(EventInterceptor);
            expect(SwarmStateAccessor).toHaveBeenCalled();
        });
    });

    describe("bot registration", () => {
        it("should register bot successfully", () => {
            eventInterceptor.registerBot(sampleBot);

            expect(logger.info).toHaveBeenCalledWith(
                "Bot registered for event interception",
                expect.objectContaining({
                    botId: sampleBot.id,
                    patterns: expect.arrayContaining(["chat/message", "chat/*", "user/login"]),
                }),
            );
        });

        it("should extract patterns from bot configuration", () => {
            eventInterceptor.registerBot(sampleBot);

            const stats = eventInterceptor.getStats();
            expect(stats.registeredBots).toBe(1);
            expect(stats.totalPatterns).toBeGreaterThan(0);
        });

        it("should handle bot with no behaviors gracefully", () => {
            const botWithoutBehaviors: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [],
                    },
                },
            };

            eventInterceptor.registerBot(botWithoutBehaviors);

            const stats = eventInterceptor.getStats();
            expect(stats.registeredBots).toBe(1);
        });

        it("should generate default patterns based on bot role", () => {
            const coordinatorBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [],
                    },
                },
            };

            eventInterceptor.registerBot(coordinatorBot);

            const stats = eventInterceptor.getStats();
            expect(stats.totalPatterns).toBeGreaterThan(0);
        });

        it("should generate monitor patterns for monitor role", () => {
            const monitorBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "monitor",
                        behaviors: [],
                    },
                },
            };

            eventInterceptor.registerBot(monitorBot);

            const stats = eventInterceptor.getStats();
            expect(stats.totalPatterns).toBeGreaterThan(0);
        });

        it("should unregister bot successfully", () => {
            eventInterceptor.registerBot(sampleBot);
            eventInterceptor.unregisterBot(sampleBot.id);

            expect(logger.info).toHaveBeenCalledWith(
                "Bot unregistered from event interception",
                { botId: sampleBot.id },
            );

            const stats = eventInterceptor.getStats();
            expect(stats.registeredBots).toBe(0);
        });

        it("should handle unregistering non-existent bot", () => {
            eventInterceptor.unregisterBot("non-existent-bot");

            expect(logger.info).toHaveBeenCalledWith(
                "Bot unregistered from event interception",
                { botId: "non-existent-bot" },
            );
        });
    });

    describe("event interception", () => {
        beforeEach(() => {
            eventInterceptor.registerBot(sampleBot);
        });

        it("should intercept matching events successfully", async () => {
            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(result.progression).toBe("continue");
            expect(result.responses).toHaveLength(1);
            expect(mockLockService.acquire).toHaveBeenCalledWith(
                `event_interception:${sampleEvent.id}`,
                expect.any(Object),
            );
        });

        it("should skip non-interceptable events", async () => {
            vi.mocked(getEventBehavior).mockReturnValueOnce({
                mode: 0, // PASSIVE mode
                interceptable: false,
            });

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(false);
            expect(result.progression).toBe("continue");
            expect(mockLockService.acquire).not.toHaveBeenCalled();
        });

        it("should handle no matching bots", async () => {
            const nonMatchingEvent: ServiceEvent = {
                ...sampleEvent,
                type: "system/internal",
            };

            const result = await eventInterceptor.checkInterception(nonMatchingEvent, sampleSwarmState);

            expect(result.intercepted).toBe(false);
            expect(result.progression).toBe("continue");
        });

        it("should handle already processed events", async () => {
            const processedEvent: ServiceEvent = {
                ...sampleEvent,
                progression: {
                    finalDecision: "block",
                    processedBy: [
                        {
                            botId: sampleBot.id,
                            response: { progression: "block", reason: "Already processed" },
                            timestamp: new Date(),
                        },
                    ],
                },
            };

            const result = await eventInterceptor.checkInterception(processedEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(result.progression).toBe("block");
        });

        it("should handle lock acquisition failure gracefully", async () => {
            mockLockService.acquire = vi.fn().mockRejectedValue(new Error("Lock failed"));

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result).toBeDefined();
            expect(logger.error).toHaveBeenCalled();
        });

        it("should process bots with blockOnFirst strategy", async () => {
            vi.mocked(getEventBehavior).mockReturnValueOnce({
                mode: 1,
                interceptable: true,
                barrierConfig: {
                    blockOnFirst: true,
                    quorum: 1,
                    timeoutMs: 5000,
                    timeoutAction: "block",
                },
            });

            mockDecisionMaker.decide.mockResolvedValueOnce({
                shouldHandle: true,
                response: { progression: "block", reason: "Blocked by first bot" },
            });

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(vi.mocked(aggregateProgression)).toHaveBeenCalled();
        });
    });

    describe("pattern matching", () => {
        it("should match exact event types", async () => {
            const chatBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "specialist",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: { type: "routine", routineId: generatePK().toString() },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.registerBot(chatBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);
            expect(result.intercepted).toBe(true);
        });

        it("should match wildcard patterns", async () => {
            const wildcardBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "monitor",
                        behaviors: [
                            {
                                trigger: { topic: "chat/+" },
                                action: { type: "invoke", purpose: "Monitor chat events" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.registerBot(wildcardBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);
            expect(result.intercepted).toBe(true);
        });

        it("should match multi-level wildcard patterns", async () => {
            const multiLevelBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "monitor",
                        behaviors: [
                            {
                                trigger: { topic: "#" },
                                action: { type: "invoke", purpose: "Monitor all events" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.registerBot(multiLevelBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);
            expect(result.intercepted).toBe(true);
        });

        it("should not match non-matching patterns", async () => {
            const nonMatchingBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "specialist",
                        behaviors: [
                            {
                                trigger: { topic: "system/error" },
                                action: { type: "routine", routineId: "error-handler" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.registerBot(nonMatchingBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);
            expect(result.intercepted).toBe(false);
        });
    });

    describe("bot priority calculation", () => {
        it("should prioritize coordinator bots", async () => {
            const coordinatorBot: BotParticipant = {
                ...sampleBot,
                id: "coordinator-bot",
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: { type: "routine", routineId: "coordinator-routine" },
                            },
                        ],
                    },
                },
            };

            const specialistBot: BotParticipant = {
                ...sampleBot,
                id: "specialist-bot",
                config: {
                    agentSpec: {
                        role: "specialist",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: { type: "routine", routineId: "specialist-routine" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.registerBot(coordinatorBot);
            eventInterceptor.registerBot(specialistBot);

            // Mock to track call order
            const callOrder: string[] = [];
            mockDecisionMaker.decide.mockImplementation(async (context: BotDecisionContext) => {
                callOrder.push(context.bot.id);
                return {
                    shouldHandle: true,
                    response: { progression: "continue", reason: "Bot response" },
                };
            });

            await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(callOrder[0]).toBe("coordinator-bot"); // Coordinator should be first
        });

        it("should give bonus for exact pattern match", async () => {
            const exactMatchBot: BotParticipant = {
                ...sampleBot,
                id: "exact-match-bot",
                config: {
                    agentSpec: {
                        role: "specialist",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: { type: "routine", routineId: "exact-routine" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.registerBot(exactMatchBot);
            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
        });
    });

    describe("JEXL expression evaluation", () => {
        it("should evaluate JEXL conditions successfully", async () => {
            const jexlMock = await import("jexl");
            vi.mocked(jexlMock.default.eval).mockResolvedValue(true);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(jexlMock.default.eval).toHaveBeenCalled();
        });

        it("should skip bot when JEXL condition fails", async () => {
            const jexlMock = await import("jexl");
            vi.mocked(jexlMock.default.eval).mockResolvedValue(false);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(false);
        });

        it("should handle JEXL evaluation errors", async () => {
            const jexlMock = await import("jexl");
            vi.mocked(jexlMock.default.eval).mockRejectedValue(new Error("JEXL error"));

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(logger.error).toHaveBeenCalled();
        });
    });

    describe("bot action execution", () => {
        beforeEach(() => {
            eventInterceptor.registerBot(sampleBot);
        });

        it("should execute routine actions successfully", async () => {
            const routineBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: {
                                    type: "routine",
                                    routineId: generatePK().toString(),
                                    label: "Process Message",
                                    inputMap: { message: "event.data.message" },
                                },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.unregisterBot(sampleBot.id);
            eventInterceptor.registerBot(routineBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(mockContextManager.allocateResources).toHaveBeenCalled();
            expect(RoutineExecutor).toHaveBeenCalled();
        });

        it("should execute invoke actions successfully", async () => {
            const invokeBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "specialist",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: {
                                    type: "invoke",
                                    purpose: "Analyze message sentiment",
                                },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.unregisterBot(sampleBot.id);
            eventInterceptor.registerBot(invokeBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(StepExecutor).toHaveBeenCalled();
        });

        it("should handle action execution errors", async () => {
            mockContextManager.allocateResources.mockRejectedValue(new Error("Resource allocation failed"));

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(logger.error).toHaveBeenCalled();
            expect(result.responses[0].response.progression).toBe("block");
        });

        it("should handle routine action without routineId", async () => {
            const invalidRoutineBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [
                            {
                                trigger: { topic: "chat/message" },
                                action: {
                                    type: "routine",
                                    label: "Invalid Routine",
                                    // Missing routineId
                                },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.unregisterBot(sampleBot.id);
            eventInterceptor.registerBot(invalidRoutineBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(logger.error).toHaveBeenCalled();
            expect(result.responses[0].response.progression).toBe("block");
        });
    });

    describe("progression control", () => {
        beforeEach(() => {
            eventInterceptor.registerBot(sampleBot);
        });

        it("should handle conditional progression", async () => {
            const jexlMock = await import("jexl");
            vi.mocked(jexlMock.default.eval)
                .mockResolvedValueOnce(true) // For trigger condition
                .mockResolvedValueOnce(true); // For progression condition

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(result.progression).toBe("continue");
        });

        it("should handle blocking progression", async () => {
            const blockingBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [
                            {
                                trigger: {
                                    topic: "chat/message",
                                    progression: {
                                        control: "block",
                                        exclusive: false,
                                    },
                                },
                                action: { type: "routine", routineId: "blocking-routine" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.unregisterBot(sampleBot.id);
            eventInterceptor.registerBot(blockingBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
        });

        it("should handle exclusive responses", async () => {
            const exclusiveBot: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "coordinator",
                        behaviors: [
                            {
                                trigger: {
                                    topic: "chat/message",
                                    progression: {
                                        control: "continue",
                                        exclusive: true,
                                    },
                                },
                                action: { type: "routine", routineId: "exclusive-routine" },
                            },
                        ],
                    },
                },
            };

            eventInterceptor.unregisterBot(sampleBot.id);
            eventInterceptor.registerBot(exclusiveBot);

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(result.intercepted).toBe(true);
            expect(result.responses).toHaveLength(1); // Should stop after exclusive response
        });
    });

    describe("resource management", () => {
        beforeEach(() => {
            eventInterceptor.registerBot(sampleBot);
        });

        it("should allocate and release resources properly", async () => {
            await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(mockContextManager.allocateResources).toHaveBeenCalledWith(
                chatId,
                expect.objectContaining({
                    consumerType: "run",
                    limits: expect.objectContaining({
                        maxCredits: "1000",
                        maxDurationMs: 300000,
                    }),
                }),
            );
        });

        it("should track active executions", async () => {
            await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            const activeExecutions = eventInterceptor.getActiveExecutions();
            expect(activeExecutions.size).toBeGreaterThan(0);
        });

        it("should stop all active executions", async () => {
            await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            await eventInterceptor.stopAllActiveExecutions();

            const activeExecutions = eventInterceptor.getActiveExecutions();
            expect(activeExecutions.size).toBe(0);
        });

        it("should handle resource cleanup errors gracefully", async () => {
            mockContextManager.releaseResources.mockRejectedValue(new Error("Release failed"));

            await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            // Should not throw error
            expect(logger.error).not.toHaveBeenCalledWith(
                expect.stringContaining("Resource cleanup"),
                expect.any(Object),
            );
        });
    });

    describe("statistics and monitoring", () => {
        it("should provide accurate statistics", () => {
            eventInterceptor.registerBot(sampleBot);

            const stats = eventInterceptor.getStats();

            expect(stats.registeredBots).toBe(1);
            expect(stats.totalPatterns).toBeGreaterThan(0);
            expect(stats.patternDistribution).toBeInstanceOf(Map);
            expect(stats.activeExecutions).toBe(0);
        });

        it("should track multiple bots correctly", () => {
            const bot2: BotParticipant = {
                ...sampleBot,
                id: generatePK().toString(),
                name: "Second Bot",
            };

            eventInterceptor.registerBot(sampleBot);
            eventInterceptor.registerBot(bot2);

            const stats = eventInterceptor.getStats();
            expect(stats.registeredBots).toBe(2);
        });

        it("should handle empty state correctly", () => {
            const stats = eventInterceptor.getStats();

            expect(stats.registeredBots).toBe(0);
            expect(stats.totalPatterns).toBe(0);
            expect(stats.activeExecutions).toBe(0);
        });
    });

    describe("error handling", () => {
        beforeEach(() => {
            eventInterceptor.registerBot(sampleBot);
        });

        it("should handle decision maker errors", async () => {
            mockDecisionMaker.decide.mockRejectedValue(new Error("Decision failed"));

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(logger.error).toHaveBeenCalled();
            expect(result.responses[0].response.progression).toBe("block");
        });

        it("should handle state accessor errors", async () => {
            mockStateAccessor.buildTriggerContext.mockImplementation(() => {
                throw new Error("Context build failed");
            });

            const result = await eventInterceptor.checkInterception(sampleEvent, sampleSwarmState);

            expect(logger.error).toHaveBeenCalled();
        });

        it("should handle execution cleanup errors", async () => {
            // Setup the factory to return an executor with a failing stop method
            const failingStateMachine = {
                stop: vi.fn().mockRejectedValue(new Error("Stop failed")),
            };
            const mockRoutineExecutorFactory = vi.fn().mockResolvedValue({
                start: vi.fn().mockResolvedValue(undefined),
                stop: vi.fn().mockResolvedValue({ success: true }),
                getState: vi.fn().mockReturnValue("READY"),
                getStateMachine: vi.fn().mockReturnValue(failingStateMachine),
            });
            
            // Create new interceptor with failing factory
            const testInterceptor = new EventInterceptor(mockLockService, mockContextManager, mockRoutineExecutorFactory);
            
            await testInterceptor.checkInterception(sampleEvent, sampleSwarmState);
            await testInterceptor.stopAllActiveExecutions();

            expect(logger.error).toHaveBeenCalledWith(
                "Failed to stop execution",
                expect.any(Object),
            );
        });
    });

    describe("edge cases", () => {
        it("should handle bot with missing config", () => {
            const botWithoutConfig: BotParticipant = {
                id: "bot-no-config",
                name: "Bot Without Config",
                role: "participant",
                // Missing config
            };

            eventInterceptor.registerBot(botWithoutConfig);

            const stats = eventInterceptor.getStats();
            expect(stats.registeredBots).toBe(1);
        });

        it("should handle event with missing data", async () => {
            const eventWithoutData: ServiceEvent = {
                id: "event-no-data",
                type: "chat/message",
                timestamp: new Date(),
                // Missing data
            };

            eventInterceptor.registerBot(sampleBot);

            const result = await eventInterceptor.checkInterception(eventWithoutData, sampleSwarmState);
            expect(result).toBeDefined();
        });

        it("should handle duplicate bot registration", () => {
            eventInterceptor.registerBot(sampleBot);
            eventInterceptor.registerBot(sampleBot);

            const stats = eventInterceptor.getStats();
            expect(stats.registeredBots).toBe(1); // Should deduplicate
        });

        it("should handle empty behavior arrays", () => {
            const botWithEmptyBehaviors: BotParticipant = {
                ...sampleBot,
                config: {
                    agentSpec: {
                        role: "specialist",
                        behaviors: [],
                    },
                },
            };

            eventInterceptor.registerBot(botWithEmptyBehaviors);
            expect(logger.info).toHaveBeenCalled();
        });
    });
});
