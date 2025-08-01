import {
    EventTypes,
    RunState,
    type CoreResourceAllocation,
    type ExecutionStrategy,
    type RoutineExecutionInput,
    type RunContext,
    type SessionUser,
    type TierExecutionRequest,
} from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, test, vi, type Mock, type MockedFunction } from "vitest";
import { logger } from "../../../events/logger.js";
import { EventPublisher } from "../../events/publisher.js";
import type { ISwarmContextManager } from "../shared/SwarmContextManager.js";
import type { StepExecutor } from "../tier3/stepExecutor.js";
import { RoutineExecutor } from "./routineExecutor.js";
import type { IRunContextManager } from "./runContextManager.js";
import type { INavigator, StepInfo } from "./types.js";

// Mock dependencies
vi.mock("../../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        debug: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("../../events/publisher.js", () => ({
    EventPublisher: {
        emit: vi.fn(async () => ({ proceed: true, reason: null })),
    },
}));

// Create mock instance that will be returned by DbProvider.get()
const mockDbInstance = {
    run: {
        findUnique: vi.fn(),
    },
};

vi.mock("../../../db/provider.js", () => ({
    DbProvider: {
        init: vi.fn().mockResolvedValue(undefined),
        get: vi.fn(() => mockDbInstance),
    },
}));

vi.mock("./navigators/navigatorFactory.js", () => ({
    getNavigator: vi.fn(),
}));

// Create a mock state machine instance that will be reused
let mockStateMachineInstance: any;

vi.mock("./routineStateMachine.js", () => ({
    RoutineStateMachine: vi.fn().mockImplementation(() => {
        if (!mockStateMachineInstance) {
            mockStateMachineInstance = {
                initializeExecution: vi.fn().mockResolvedValue(undefined),
                start: vi.fn().mockResolvedValue(undefined),
                complete: vi.fn().mockResolvedValue(undefined),
                fail: vi.fn().mockResolvedValue(undefined),
                getState: vi.fn().mockReturnValue(RunState.RUNNING),
                getResourceUsage: vi.fn().mockReturnValue({
                    creditsUsed: "50",
                    durationMs: 1000,
                    memoryUsedMB: 100,
                    stepsExecuted: 1,
                    startTime: new Date(),
                }),
                addCreditsUsed: vi.fn(),
                incrementStepCount: vi.fn(),
                canProceed: vi.fn().mockResolvedValue(true),
                pause: vi.fn().mockResolvedValue(undefined),
                resume: vi.fn().mockResolvedValue(undefined),
                stop: vi.fn().mockResolvedValue(undefined),
            };
        }
        return mockStateMachineInstance;
    }),
}));

vi.mock("@vrooli/shared", async () => {
    const actual = await vi.importActual("@vrooli/shared");
    return {
        ...actual,
        generatePK: vi.fn(() => "test-pk-123"),
    };
});

// Import mocked navigator factory
import { getNavigator } from "./navigators/navigatorFactory.js";

const mockGetNavigator = getNavigator as MockedFunction<typeof getNavigator>;

// Test data factories
function createMockUser(partial: Partial<SessionUser> = {}): SessionUser {
    const now = new Date().toISOString();
    return {
        __typename: "SessionUser" as const,
        id: "test-user-123",
        publicId: "pub_test-user-123",
        credits: "1000",
        creditAccountId: "credit_test-user-123",
        creditSettings: null,
        handle: "testuser",
        hasPremium: false,
        hasReceivedPhoneVerificationReward: false,
        languages: ["en"],
        name: "Test User",
        phoneNumberVerified: false,
        profileImage: null,
        theme: "light",
        updatedAt: now,
        session: {
            __typename: "SessionUserSession" as const,
            id: "session_test-user-123",
            lastRefreshAt: now,
        },
        ...partial,
    } as SessionUser;
}

function createMockSwarmContextManager(): ISwarmContextManager {
    return {
        updateContext: vi.fn().mockResolvedValue(undefined),
        getSwarmState: vi.fn().mockResolvedValue({
            swarmId: "123456789012345679",
            version: 1,
            chatConfig: { __version: "1.0.0" },
            execution: { status: RunState.RUNNING, agents: [], activeRuns: [] },
            resources: {
                allocated: [],
                consumed: { credits: 0, tokens: 0, time: 0 },
                remaining: { credits: 10000, tokens: 10000, time: 3600 },
            },
            metadata: {
                createdAt: new Date(),
                lastUpdated: new Date(),
                updatedBy: "system",
                subscribers: new Set(),
            },
        }),
        isSwarmActive: vi.fn().mockResolvedValue(true),
        removeContext: vi.fn().mockResolvedValue(undefined),
        saveSwarmState: vi.fn().mockResolvedValue(undefined),
    } as unknown as ISwarmContextManager;
}

function createMockStepExecutor(): StepExecutor {
    return {
        execute: vi.fn().mockResolvedValue({
            success: true,
            outputs: { result: "test output" },
            metadata: {
                tokensUsed: 100,
                executionTime: 500,
                creditsUsed: "25",
                toolCalls: 1,
            },
        }),
    } as unknown as StepExecutor;
}

function createMockRunContextManager(): IRunContextManager {
    return {
        getRunContext: vi.fn().mockResolvedValue({
            runId: "123456789012345680",
            routineId: "123456789012345681",
            swarmId: "123456789012345679",
            navigator: null,
            currentLocation: { id: "start", routineId: "123456789012345681", nodeId: "start" },
            visitedLocations: [],
            variables: {},
            outputs: {},
            completedSteps: [],
            parallelBranches: [],
            availableResources: [],
            sharedKnowledge: {},
            resourceLimits: { maxCredits: "1000", maxDurationMs: 300000, maxMemoryMB: 512, maxSteps: 50 },
            resourceUsage: { creditsUsed: "0", durationMs: 0, memoryUsedMB: 0, stepsExecuted: 0, startTime: new Date() },
            progress: { currentStepId: null, completedSteps: [], totalSteps: 0, percentComplete: 0 },
            retryCount: 0,
        }),
        updateRunContext: vi.fn().mockResolvedValue(undefined),
        allocateFromSwarm: vi.fn().mockResolvedValue({
            allocationId: "123",
            runId: "123456789012345680",
            swarmId: "123456789012345679",
            allocated: { credits: "1000", timeoutMs: 300000, memoryMB: 512, concurrentExecutions: 1 },
            remaining: { credits: "1000", timeoutMs: 300000, memoryMB: 512, concurrentExecutions: 1 },
            allocatedAt: new Date(),
            expiresAt: new Date(Date.now() + 300000),
        }),
        allocateForStep: vi.fn().mockResolvedValue({
            allocationId: "456",
            stepId: "789",
            runId: "123456789012345680",
            allocated: { credits: "50", timeoutMs: 60000, memoryMB: 128, concurrentExecutions: 1 },
            allocatedAt: new Date(),
            expiresAt: new Date(Date.now() + 60000),
        }),
        releaseFromStep: vi.fn().mockResolvedValue(undefined),
        releaseToSwarm: vi.fn().mockResolvedValue(undefined),
        checkResourceAvailability: vi.fn().mockResolvedValue(true),
        getResourceUsage: vi.fn().mockResolvedValue({ creditsUsed: "100", durationMs: 60000, memoryUsedMB: 256, stepsExecuted: 5, toolCalls: 3 }),
        validateContext: vi.fn().mockResolvedValue(true),
        clearContext: vi.fn().mockResolvedValue(undefined),
        emitRunStarted: vi.fn().mockResolvedValue(undefined),
        emitRunCompleted: vi.fn().mockResolvedValue(undefined),
        emitRunFailed: vi.fn().mockResolvedValue(undefined),
        allocateStep: vi.fn().mockResolvedValue({
            stepId: "321",
            allocated: { credits: "100", timeoutMs: 60000, memoryMB: 128, concurrentExecutions: 1 },
        }),
        releaseStep: vi.fn().mockResolvedValue(undefined),
    } as unknown as IRunContextManager;
}

function createMockNavigator(): INavigator {
    return {
        type: "sequential",
        version: "1.0.0",
        canNavigate: vi.fn().mockReturnValue(true),
        getStartLocation: vi.fn().mockReturnValue({
            id: "start",
            routineId: "123456789012345681",
            nodeId: "0",
        }),
        getNextLocations: vi.fn().mockReturnValue([{
            id: "1",
            routineId: "123456789012345681",
            nodeId: "1",
        }]),
        isEndLocation: vi.fn().mockReturnValue(false),
        getStepInfo: vi.fn().mockReturnValue({
            id: "123456789012345680",
            name: "Test Step",
            type: "llm_call",
            description: "Test step description",
            config: {
                prompt: "Test prompt",
                model: "gpt-4",
                temperature: 0.7,
            },
        }),
    };
}

function createMockTierExecutionRequest(): TierExecutionRequest<RoutineExecutionInput> {
    return {
        context: {
            swarmId: "123456789012345679",
            userData: createMockUser(),
            parentSwarmId: "parent-swarm-123",
            timestamp: new Date(),
        },
        input: {
            resourceVersionId: "123456789012345681",
            runId: "123456789012345680",
            inputs: { inputParam: "test value" },
            metadata: { testKey: "testValue" },
        },
        allocation: {
            maxCredits: "1000",
            maxDurationMs: 300000,
            maxMemoryMB: 512,
            maxConcurrentSteps: 10,
        },
        options: {
            timeout: 30000,
            retryCount: 3,
            priority: "medium",
        },
    };
}

// Test suite
describe("RoutineExecutor", () => {
    let contextManager: ISwarmContextManager;
    let stepExecutor: StepExecutor;
    let runContextManager: IRunContextManager;
    let navigator: INavigator;
    let routineExecutor: RoutineExecutor;

    beforeEach(() => {
        vi.clearAllMocks();

        // Reset the mock state machine instance
        mockStateMachineInstance = undefined;

        contextManager = createMockSwarmContextManager();
        stepExecutor = createMockStepExecutor();
        runContextManager = createMockRunContextManager();
        navigator = createMockNavigator();

        // Setup navigator factory mock
        mockGetNavigator.mockReturnValue(navigator);

        routineExecutor = new RoutineExecutor(
            contextManager,
            stepExecutor,
            "test-context-123",
            runContextManager,
            "test-user-123",
            "parent-swarm-123",
        );

        // Directly replace the state machine with our mock
        (routineExecutor as any).stateMachine = mockStateMachineInstance || {
            initializeExecution: vi.fn().mockResolvedValue(undefined),
            start: vi.fn().mockResolvedValue(undefined),
            complete: vi.fn().mockResolvedValue(undefined),
            fail: vi.fn().mockResolvedValue(undefined),
            getState: vi.fn().mockReturnValue(RunState.RUNNING),
            getResourceUsage: vi.fn().mockReturnValue({
                creditsUsed: "50",
                durationMs: 1000,
                memoryUsedMB: 100,
                stepsExecuted: 1,
                startTime: new Date(),
            }),
            addCreditsUsed: vi.fn(),
            incrementStepCount: vi.fn(),
            canProceed: vi.fn().mockResolvedValue(true),
            pause: vi.fn().mockResolvedValue(undefined),
            resume: vi.fn().mockResolvedValue(undefined),
            stop: vi.fn().mockResolvedValue(undefined),
        };
    });

    afterEach(() => {
        vi.resetAllMocks();
    });

    describe("Constructor", () => {
        test("should create instance with all dependencies", () => {
            expect(routineExecutor).toBeDefined();
            expect(routineExecutor.getStateMachine()).toBeDefined();
        });

        test("should create instance without optional runContextManager", () => {
            const executor = new RoutineExecutor(
                contextManager,
                stepExecutor,
                "test-context-123",
            );
            expect(executor).toBeDefined();
            expect(executor.getStateMachine()).toBeDefined();
        });
    });

    describe("execute()", () => {
        test("should execute routine successfully", async () => {
            const request = createMockTierExecutionRequest();
            const mockStateMachine = routineExecutor.getStateMachine();

            // Mock navigator to return end location on second call
            (navigator.isEndLocation as Mock)
                .mockReturnValueOnce(false)  // First step
                .mockReturnValueOnce(true);  // End after first step

            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(true);
            expect(result.result).toBeDefined();
            expect(result.resourcesUsed).toBeDefined();
            expect(result.duration).toBeGreaterThan(0);
            expect(result.metadata).toEqual({
                swarmId: "123456789012345679",
                strategy: "lean",
                componentCount: 2,
            });

            // Verify state machine calls
            expect(mockStateMachine.initializeExecution).toHaveBeenCalledWith(
                "123456789012345679",
                "123456789012345681",
                "123456789012345679",
                "123456789012345680",
            );
            expect(mockStateMachine.start).toHaveBeenCalled();
            expect(mockStateMachine.complete).toHaveBeenCalled();
        });

        test("should handle execution failure", async () => {
            const request = createMockTierExecutionRequest();
            const mockStateMachine = routineExecutor.getStateMachine();

            // Mock state machine to throw error
            mockStateMachine.initializeExecution = vi.fn().mockRejectedValue(new Error("Initialization failed"));

            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error).toBeDefined();
            expect(result.error?.code).toBe("ROUTINE_EXECUTION_FAILED");
            expect(result.error?.message).toBe("Initialization failed");
            expect(result.error?.tier).toBe("tier2");
            expect(result.error?.strategy).toBe("lean");
            expect(result.metadata?.failurePoint).toBe("execution");

            // Verify failure handling
            expect(mockStateMachine.fail).toHaveBeenCalledWith("Initialization failed");
        });

        test("should handle missing navigator", async () => {
            const request = createMockTierExecutionRequest();
            const mockStateMachine = routineExecutor.getStateMachine();

            // Mock navigator factory to return null
            mockGetNavigator.mockReturnValue(null);

            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error?.message).toBe("No navigator available for this routine type");
            expect(mockStateMachine.fail).toHaveBeenCalled();
        });
    });

    describe("executeRoutineSteps()", () => {
        test("should execute sequential steps successfully", async () => {
            const request = createMockTierExecutionRequest();

            // Mock navigator behavior for multi-step execution
            (navigator.isEndLocation as Mock)
                .mockReturnValueOnce(false)  // Step 0
                .mockReturnValueOnce(false)  // Step 1 
                .mockReturnValueOnce(true);  // End at step 2

            (navigator.getNextLocations as Mock)
                .mockReturnValueOnce([{ id: "123456789012345684", routineId: "123456789012345681", nodeId: "1" }])
                .mockReturnValueOnce([{ id: "123456789012345686", routineId: "123456789012345681", nodeId: "2" }])
                .mockReturnValueOnce([]);  // No more steps

            (navigator.getStepInfo as Mock)
                .mockReturnValueOnce({
                    id: "123456789012345681",
                    name: "First Step",
                    type: "llm_call",
                    config: { prompt: "First prompt" },
                })
                .mockReturnValueOnce({
                    id: "123456789012345684",
                    name: "Second Step",
                    type: "tool_call",
                    config: { tool: "calculator" },
                });

            // Mock step executor responses
            (stepExecutor.execute as Mock)
                .mockResolvedValueOnce({
                    success: true,
                    outputs: { step0Result: "result0" },
                })
                .mockResolvedValueOnce({
                    success: true,
                    outputs: { step1Result: "result1" },
                });

            // Mock resource allocation for steps
            (runContextManager?.allocateForStep as Mock)
                .mockResolvedValueOnce({
                    allocationId: "123456789012345683",
                    stepId: "123456789012345681",
                    allocated: { credits: "25", timeoutMs: 30000, memoryMB: 64, concurrentExecutions: 1 },
                })
                .mockResolvedValueOnce({
                    allocationId: "123456789012345685",
                    stepId: "123456789012345684",
                    allocated: { credits: "25", timeoutMs: 30000, memoryMB: 64, concurrentExecutions: 1 },
                });

            const mockStateMachine = routineExecutor.getStateMachine();

            // Execute the test by calling execute which will internally call executeRoutineSteps
            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(true);
            expect(stepExecutor.execute).toHaveBeenCalledTimes(2);
            expect(navigator.getStepInfo).toHaveBeenCalledTimes(2);
            expect(navigator.getNextLocations).toHaveBeenCalledTimes(2);
            expect(mockStateMachine.addCreditsUsed).toHaveBeenCalledTimes(2);
            expect(mockStateMachine.incrementStepCount).toHaveBeenCalledTimes(2);
        });

        test("should handle step execution failure", async () => {
            const request = createMockTierExecutionRequest();

            // Mock navigator for single step that fails
            (navigator.isEndLocation as Mock).mockReturnValue(false);
            (navigator.getStepInfo as Mock).mockReturnValue({
                id: "failing-step",
                name: "Failing Step",
                type: "llm_call",
                config: { prompt: "This will fail" },
            });

            // Mock step executor to fail
            (stepExecutor.execute as Mock).mockRejectedValue(new Error("Step execution failed"));

            const mockStateMachine = routineExecutor.getStateMachine();

            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Step execution failed");
            expect(mockStateMachine.fail).toHaveBeenCalled();
        });

        test("should handle infinite loop protection", async () => {
            const request = createMockTierExecutionRequest();

            // Mock navigator to never end (infinite loop scenario)
            (navigator.isEndLocation as Mock).mockReturnValue(false);
            (navigator.getNextLocations as Mock).mockReturnValue([{
                id: "infinite-step",
                routineId: "123456789012345681",
                nodeId: "infinite",
            }]);
            (navigator.getStepInfo as Mock).mockReturnValue({
                id: "infinite-step",
                name: "Infinite Step",
                type: "llm_call",
                config: {},
            });

            const mockStateMachine = routineExecutor.getStateMachine();

            const result = await routineExecutor.execute(request);

            // Should complete successfully after hitting MAX_STEPS limit
            expect(result.success).toBe(true);
            expect(stepExecutor.execute).toHaveBeenCalledTimes(1000); // MAX_STEPS
        });
    });

    describe("executeSingleStep()", () => {
        test("should execute step with resource allocation", async () => {
            const stepInfo: StepInfo = {
                id: "123456789012345682",
                name: "Test Step",
                type: "llm_call",
                config: { prompt: "Test prompt" },
            };

            const mockStateMachine = routineExecutor.getStateMachine();

            // Execute single step via the execute method (which calls executeSingleStep internally)
            const request = createMockTierExecutionRequest();
            (navigator.isEndLocation as Mock).mockReturnValueOnce(false).mockReturnValueOnce(true);
            (navigator.getNextLocations as Mock).mockReturnValue([]);
            (navigator.getStepInfo as Mock).mockReturnValue(stepInfo);

            await routineExecutor.execute(request);

            // Verify resource allocation and release were called
            expect(runContextManager?.allocateForStep).toHaveBeenCalledWith("123456789012345679", {
                stepId: "123456789012345682",
                stepType: "llm_call",
                estimatedRequirements: {
                    credits: "50",
                    durationMs: 60000,
                    memoryMB: 128,
                },
            });
            expect(runContextManager?.releaseFromStep).toHaveBeenCalled();

            // Verify events were emitted
            expect(EventPublisher.emit).toHaveBeenCalledWith(EventTypes.RUN.STEP_STARTED, expect.any(Object));
            expect(EventPublisher.emit).toHaveBeenCalledWith(EventTypes.RUN.STEP_COMPLETED, expect.any(Object));
        });

        test("should handle step failure and resource cleanup", async () => {
            const stepInfo: StepInfo = {
                id: "failing-step",
                name: "Failing Step",
                type: "tool_call",
                config: { tool: "broken_tool" },
            };

            // Mock step executor to fail
            (stepExecutor.execute as Mock).mockRejectedValue(new Error("Tool execution failed"));
            (navigator.isEndLocation as Mock).mockReturnValue(false);
            (navigator.getStepInfo as Mock).mockReturnValue(stepInfo);

            const request = createMockTierExecutionRequest();
            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(false);

            // Verify resource cleanup on failure
            expect(runContextManager?.allocateForStep).toHaveBeenCalled();
            expect(runContextManager?.releaseFromStep).toHaveBeenCalledWith(
                "123456789012345679",
                "failing-step",
                expect.objectContaining({
                    creditsUsed: "15", // Partial usage on failure
                    stepsExecuted: 0,
                    toolCalls: 0,
                }),
            );
        });

        test("should handle permission denial", async () => {
            const stepInfo: StepInfo = {
                id: "restricted-step",
                name: "Restricted Step",
                type: "subroutine", // This type is not allowed for regular users
                config: {},
            };

            // Mock the step execution to simulate permission failure by throwing an error
            (stepExecutor.execute as Mock).mockRejectedValue(new Error("Permission denied for routine execution"));

            (navigator.isEndLocation as Mock).mockReturnValue(false);
            (navigator.getStepInfo as Mock).mockReturnValue(stepInfo);

            const request = createMockTierExecutionRequest();

            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Permission denied");
        });
    });

    describe("createTier3Request()", () => {
        test("should create proper Tier 3 request for LLM call", () => {
            const stepInfo: StepInfo = {
                id: "llm-step",
                name: "LLM Step",
                type: "llm_call",
                description: "Generate response",
                config: {
                    prompt: "Hello world",
                    model: "gpt-4",
                    temperature: 0.7,
                },
            };

            const context = createMockTierExecutionRequest().context;
            const allocation: CoreResourceAllocation = {
                maxCredits: "100",
                maxDurationMs: 30000,
                maxMemoryMB: 256,
                maxConcurrentSteps: 1,
            };

            // Use a spy to access the private method
            const routineExecutorAny = routineExecutor as any;
            const tier3Request = routineExecutorAny.createTier3Request(stepInfo, context, allocation);

            expect(tier3Request).toEqual({
                context: {
                    swarmId: context.swarmId,
                    userData: context.userData,
                    parentSwarmId: context.parentSwarmId,
                    timestamp: context.timestamp,
                },
                input: {
                    stepId: "llm-step",
                    stepType: "llm_call",
                    toolName: undefined,
                    parameters: expect.objectContaining({
                        stepName: "LLM Step",
                        stepDescription: "Generate response",
                        prompt: "Hello world",
                        model: "gpt-4",
                        temperature: 0.7,
                    }),
                    strategy: "reasoning",
                },
                allocation,
                options: {
                    timeout: 30000,
                    retryCount: 0,
                    priority: "medium",
                },
            });
        });

        test("should create proper Tier 3 request for tool call", () => {
            const stepInfo: StepInfo = {
                id: "tool-step",
                name: "Tool Step",
                type: "tool_call",
                config: {
                    tool: "calculator",
                    operation: "add",
                    values: [1, 2],
                },
            };

            const context = createMockTierExecutionRequest().context;
            const routineExecutorAny = routineExecutor as any;
            const tier3Request = routineExecutorAny.createTier3Request(stepInfo, context);

            expect(tier3Request.input.toolName).toBe("calculator");
            expect(tier3Request.input.strategy).toBe("deterministic");
            expect(tier3Request.input.parameters).toEqual(expect.objectContaining({
                stepName: "Tool Step",
                tool: "calculator",
                operation: "add",
                values: [1, 2],
            }));
        });

        test("should create proper Tier 3 request for API call", () => {
            const stepInfo: StepInfo = {
                id: "api-step",
                name: "API Step",
                type: "api_call",
                config: {
                    url: "https://api.example.com/data",
                    method: "GET",
                    headers: { "Authorization": "Bearer token" },
                },
            };

            const context = createMockTierExecutionRequest().context;
            const routineExecutorAny = routineExecutor as any;
            const tier3Request = routineExecutorAny.createTier3Request(stepInfo, context);

            expect(tier3Request.input.toolName).toBe("http_request");
            expect(tier3Request.input.strategy).toBe("deterministic");
        });
    });

    describe("extractToolName()", () => {
        test("should extract tool name from config.toolName", () => {
            const stepInfo: StepInfo = {
                id: "test",
                name: "Test",
                type: "tool_call",
                config: { toolName: "custom_tool" },
            };

            const routineExecutorAny = routineExecutor as any;
            const toolName = routineExecutorAny.extractToolName(stepInfo);
            expect(toolName).toBe("custom_tool");
        });

        test("should extract tool name from config.tool", () => {
            const stepInfo: StepInfo = {
                id: "test",
                name: "Test",
                type: "tool_call",
                config: { tool: "calculator" },
            };

            const routineExecutorAny = routineExecutor as any;
            const toolName = routineExecutorAny.extractToolName(stepInfo);
            expect(toolName).toBe("calculator");
        });

        test("should derive tool name for api_call", () => {
            const stepInfo: StepInfo = {
                id: "test",
                name: "Test",
                type: "api_call",
                config: {},
            };

            const routineExecutorAny = routineExecutor as any;
            const toolName = routineExecutorAny.extractToolName(stepInfo);
            expect(toolName).toBe("http_request");
        });

        test("should return undefined for LLM call", () => {
            const stepInfo: StepInfo = {
                id: "test",
                name: "Test",
                type: "llm_call",
                config: {},
            };

            const routineExecutorAny = routineExecutor as any;
            const toolName = routineExecutorAny.extractToolName(stepInfo);
            expect(toolName).toBeUndefined();
        });
    });

    describe("determineStrategy()", () => {
        test("should use explicit strategy from config", () => {
            const stepInfo: StepInfo = {
                id: "test",
                name: "Test",
                type: "llm_call",
                config: { strategy: "deterministic" },
            };

            const routineExecutorAny = routineExecutor as any;
            const strategy = routineExecutorAny.determineStrategy(stepInfo);
            expect(strategy).toBe("deterministic");
        });

        test("should determine strategy from step type", () => {
            const testCases: Array<[string, ExecutionStrategy]> = [
                ["llm_call", "reasoning"],
                ["tool_call", "deterministic"],
                ["api_call", "deterministic"],
                ["code_execution", "deterministic"],
                ["unknown_type", "reasoning"],
            ];

            testCases.forEach(([stepType, expectedStrategy]) => {
                const stepInfo: StepInfo = {
                    id: "test",
                    name: "Test",
                    type: stepType,
                    config: {},
                };

                const routineExecutorAny = routineExecutor as any;
                const strategy = routineExecutorAny.determineStrategy(stepInfo);
                expect(strategy).toBe(expectedStrategy);
            });
        });
    });

    describe("extractParameters()", () => {
        test("should extract all parameters correctly", () => {
            const stepInfo: StepInfo = {
                id: "test",
                name: "Test Step",
                type: "llm_call",
                description: "Test description",
                config: {
                    toolName: "should_be_excluded",
                    strategy: "should_be_excluded",
                    prompt: "Hello world",
                    temperature: 0.8,
                    messages: [{ role: "user", content: "Hello" }],
                    code: "console.log('test')",
                    routineConfig: { version: "1.0" },
                    customParam: "custom value",
                },
            };

            const routineExecutorAny = routineExecutor as any;
            const params = routineExecutorAny.extractParameters(stepInfo);

            expect(params).toEqual({
                stepName: "Test Step",
                stepDescription: "Test description",
                prompt: "Hello world",
                temperature: 0.8,
                messages: [{ role: "user", content: "Hello" }],
                code: "console.log('test')",
                routineConfig: { version: "1.0" },
                customParam: "custom value",
            });

            // Verify excluded fields
            expect(params).not.toHaveProperty("toolName");
            expect(params).not.toHaveProperty("strategy");
        });
    });

    describe("checkStepPermission()", () => {
        test("should allow admin to execute any step", async () => {
            const runContext: RunContext = {
                variables: { userRole: "admin" },
                blackboard: {},
            };

            const routineExecutorAny = routineExecutor as any;
            const allowed = await routineExecutorAny.checkStepPermission(
                "test-run",
                "test-step",
                runContext,
            );

            expect(allowed).toBe(true);
        });

        test("should allow agent to execute allowed steps", async () => {
            const allowedSteps = ["action", "decision", "subroutine", "parallel", "loop"];

            for (const stepType of allowedSteps) {
                const runContext: RunContext = {
                    variables: {
                        userRole: "agent",
                        ["step_test-step_type"]: stepType,
                    },
                    blackboard: {},
                };

                const routineExecutorAny = routineExecutor as any;
                const allowed = await routineExecutorAny.checkStepPermission(
                    "test-run",
                    "test-step",
                    runContext,
                );

                expect(allowed).toBe(true);
            }
        });

        test("should deny guest any step execution", async () => {
            const runContext: RunContext = {
                variables: {
                    userRole: "guest",
                    ["step_test-step_type"]: "action",
                },
                blackboard: {},
            };

            const routineExecutorAny = routineExecutor as any;
            const allowed = await routineExecutorAny.checkStepPermission(
                "test-run",
                "test-step",
                runContext,
            );

            expect(allowed).toBe(false);
        });

        test("should deny user restricted step types", async () => {
            const runContext: RunContext = {
                variables: {
                    userRole: "user",
                    ["step_test-step_type"]: "subroutine", // Not allowed for user
                },
                blackboard: {},
            };

            const routineExecutorAny = routineExecutor as any;
            const allowed = await routineExecutorAny.checkStepPermission(
                "test-run",
                "test-step",
                runContext,
            );

            expect(allowed).toBe(false);
        });
    });

    describe("State Management Methods", () => {
        test("should proxy state management to state machine", async () => {
            const mockStateMachine = routineExecutor.getStateMachine();
            vi.spyOn(mockStateMachine, "getState").mockReturnValue(RunState.RUNNING);
            vi.spyOn(mockStateMachine, "canProceed").mockResolvedValue(true);
            vi.spyOn(mockStateMachine, "pause").mockResolvedValue(undefined);
            vi.spyOn(mockStateMachine, "resume").mockResolvedValue(undefined);
            vi.spyOn(mockStateMachine, "stop").mockResolvedValue(undefined);

            // Test getState
            const state = routineExecutor.getState();
            expect(state).toBe(RunState.RUNNING);
            expect(mockStateMachine.getState).toHaveBeenCalled();

            // Test canProceed
            const canProceed = await routineExecutor.canProceed();
            expect(canProceed).toBe(true);
            expect(mockStateMachine.canProceed).toHaveBeenCalled();

            // Test pause
            await routineExecutor.pause();
            expect(mockStateMachine.pause).toHaveBeenCalled();

            // Test resume
            await routineExecutor.resume();
            expect(mockStateMachine.resume).toHaveBeenCalled();

            // Test stop
            await routineExecutor.stop("test reason");
            expect(mockStateMachine.stop).toHaveBeenCalledWith({ reason: "test reason" });
        });
    });

    describe("Event Emission", () => {
        test("should handle event emission blocking gracefully", async () => {
            // Mock EventPublisher to block events
            (EventPublisher.emit as Mock)
                .mockResolvedValueOnce({ proceed: false, reason: "Event blocked by interceptor" })
                .mockResolvedValueOnce({ proceed: false, reason: "Completion blocked" });

            const request = createMockTierExecutionRequest();
            (navigator.isEndLocation as Mock).mockReturnValueOnce(false).mockReturnValueOnce(true);

            const result = await routineExecutor.execute(request);

            // Should still succeed despite blocked events
            expect(result.success).toBe(true);
            expect(logger.warn).toHaveBeenCalledWith(
                "[RoutineExecutor] Step start event blocked",
                expect.any(Object),
            );
            expect(logger.warn).toHaveBeenCalledWith(
                "[RoutineExecutor] Step completion event blocked",
                expect.any(Object),
            );
        });
    });

    describe("Resource Management", () => {
        test("should work without runContextManager", async () => {
            // Create executor without runContextManager
            const executorWithoutRM = new RoutineExecutor(
                contextManager,
                stepExecutor,
                "test-context-123",
            );

            // Apply the same mock to this instance
            (executorWithoutRM as any).stateMachine = {
                initializeExecution: vi.fn().mockResolvedValue(undefined),
                start: vi.fn().mockResolvedValue(undefined),
                complete: vi.fn().mockResolvedValue(undefined),
                fail: vi.fn().mockResolvedValue(undefined),
                getState: vi.fn().mockReturnValue(RunState.RUNNING),
                getResourceUsage: vi.fn().mockReturnValue({
                    creditsUsed: "50",
                    durationMs: 1000,
                    memoryUsedMB: 100,
                    stepsExecuted: 1,
                    startTime: new Date(),
                }),
                addCreditsUsed: vi.fn(),
                incrementStepCount: vi.fn(),
                canProceed: vi.fn().mockResolvedValue(true),
                pause: vi.fn().mockResolvedValue(undefined),
                resume: vi.fn().mockResolvedValue(undefined),
                stop: vi.fn().mockResolvedValue(undefined),
            };

            const request = createMockTierExecutionRequest();
            (navigator.isEndLocation as Mock).mockReturnValueOnce(false).mockReturnValueOnce(true);

            const result = await executorWithoutRM.execute(request);

            expect(result.success).toBe(true);
            // Should not call any runContextManager methods
            expect(runContextManager?.allocateForStep).not.toHaveBeenCalled();
            expect(runContextManager?.releaseFromStep).not.toHaveBeenCalled();
        });

        test("should handle resource allocation failure", async () => {
            // Mock resource allocation to fail
            (runContextManager?.allocateForStep as Mock).mockRejectedValue(
                new Error("Insufficient resources"),
            );

            const request = createMockTierExecutionRequest();
            (navigator.isEndLocation as Mock).mockReturnValue(false);

            const result = await routineExecutor.execute(request);

            expect(result.success).toBe(false);
            expect(result.error?.message).toContain("Insufficient resources");
        });
    });
});
