// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-18
import { generatePK, initIdGenerator, MINUTES_1_MS, MINUTES_5_MS, type ExecutionOptions } from "@vrooli/shared";
import { type Job } from "bullmq";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type LLMCompletionTask, QueueTaskType, type SwarmExecutionTask } from "../taskTypes.js";

// Mock the simplified swarm coordinator factory
vi.mock("../../services/execution/swarmCoordinatorFactory.js", () => ({
    getSwarmCoordinator: vi.fn(),
    resetSwarmCoordinator: vi.fn(),
    isSwarmCoordinatorInitialized: vi.fn().mockReturnValue(false),
}));

// Import after mocking to avoid loading the heavy dependencies
let llmProcess: any;
let activeSwarmRegistry: any;

describe("llmProcess", () => {
    beforeEach(async () => {
        vi.clearAllMocks();

        // Initialize ID generator for test data creation
        await initIdGenerator(0);

        // Set up mock implementations with generated IDs
        const mockExecutionId = generatePK().toString();
        const mockSwarmId = generatePK().toString();
        const mockConversationId = generatePK().toString();
        const mockUserId = generatePK().toString();

        const { getSwarmCoordinator } = await import("../../services/execution/swarmCoordinatorFactory.js");
        vi.mocked(getSwarmCoordinator).mockReturnValue({
            execute: vi.fn().mockResolvedValue({
                executionId: mockExecutionId,
                status: "completed",
                result: {
                    swarmId: mockSwarmId,
                    swarmName: "Test Swarm",
                    agentCount: 2,
                    conversationId: mockConversationId,
                },
                resourceUsage: {
                    credits: 0,
                    tokens: 0,
                    duration: 100,
                },
                duration: 100,
            }),
            start: vi.fn().mockResolvedValue({
                success: true,
                executionId: mockExecutionId,
                swarmId: mockSwarmId,
            }),
            getTaskId: vi.fn().mockReturnValue(mockSwarmId),
            getState: vi.fn().mockReturnValue("RUNNING"),
            requestPause: vi.fn().mockResolvedValue(true),
            requestStop: vi.fn().mockResolvedValue(true),
            getAssociatedUserId: vi.fn().mockReturnValue(mockUserId),
        });

        // Dynamically import after mocks are set up
        const processModule = await import("./process.js");
        llmProcess = processModule.llmProcess;
        activeSwarmRegistry = processModule.activeSwarmRegistry;

        // Clear the active swarm registry
        if (activeSwarmRegistry && activeSwarmRegistry.clear) {
            activeSwarmRegistry.clear();
        }
    });

    afterEach(async () => {
        // Clear the active swarm registry
        if (activeSwarmRegistry && activeSwarmRegistry.clear) {
            activeSwarmRegistry.clear();
        }
    });

    function createTestSwarmExecutionTask(overrides: Partial<SwarmExecutionTask> = {}): SwarmExecutionTask {
        const swarmId = generatePK().toString();
        const userId = generatePK().toString();

        const base: SwarmExecutionTask = {
            id: generatePK().toString(),
            type: QueueTaskType.SWARM_EXECUTION,
            context: {
                swarmId,
                userData: {
                    id: userId,
                    name: "testUser",
                    hasPremium: false,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
                timestamp: new Date(),
            },
            input: {
                swarmId,
                goal: "Complete the test objectives efficiently",
                availableTools: [
                    { name: "calculator", description: "Perform mathematical calculations" },
                    { name: "researcher", description: "Research and gather information" },
                ],
                executionConfig: {
                    model: "gpt-4",
                    temperature: 0.7,
                    parallelExecutionLimit: 3,
                },
            },
            allocation: {
                maxCredits: "1000",
                maxTokens: 50000,
                maxDurationMs: MINUTES_5_MS,
            },
            options: {
                strategy: "conversational",
                timeout: 30000,
            },
        };

        return {
            ...base,
            ...overrides,
        };
    }

    function createTestLLMCompletionTask(overrides: Partial<LLMCompletionTask> = {}): LLMCompletionTask {
        const chatId = generatePK().toString();
        const messageId = generatePK().toString();
        const userId = generatePK().toString();
        const userData = overrides.userData || {
            id: userId,
            name: "testUser",
            hasPremium: false,
            languages: ["en"],
            roles: [],
            wallets: [],
            theme: "light",
        };

        return {
            taskType: QueueTaskType.LLM_COMPLETION,
            type: QueueTaskType.LLM_COMPLETION,
            chatId,
            messageId,
            model: "gpt-4",
            taskContexts: [],
            userData,
            respondingBot: {
                id: generatePK().toString(),
                publicId: "test-bot",
                handle: "testbot",
            },
            allocation: {
                maxCredits: userData.hasPremium ? "200" : "100",
                maxDurationMs: 60000,
                maxMemoryMB: 256,
                maxConcurrentSteps: 1,
            },
            options: {
                priority: userData.hasPremium ? "high" : "medium",
                timeout: 60000,
                retryPolicy: {
                    maxRetries: 3,
                    backoffMs: 1000,
                    backoffMultiplier: 2,
                    maxBackoffMs: 30000,
                },
            },
            ...overrides,
        };
    }

    function createMockSwarmJob(data: Partial<SwarmExecutionTask | LLMCompletionTask> = {}): Job<SwarmExecutionTask | LLMCompletionTask> {
        const defaultData = data.type === QueueTaskType.LLM_COMPLETION
            ? createTestLLMCompletionTask(data as Partial<LLMCompletionTask>)
            : createTestSwarmExecutionTask(data as Partial<SwarmExecutionTask>);

        return {
            id: generatePK().toString(),
            data: defaultData,
            name: "swarm",
            attemptsMade: 0,
            opts: {},
        } as Job<SwarmExecutionTask | LLMCompletionTask>;
    }

    describe("successful swarm execution", () => {
        it("should process swarm execution task", async () => {
            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);

            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });

            // Verify swarm was added to active registry
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(1);
            expect(activeSwarms[0].id).toBe(swarmTask.context.swarmId);
        });

        it("should handle different model configurations", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                config: {
                    name: "Advanced Swarm",
                    description: "Advanced swarm with Claude model",
                    goal: "Test advanced AI capabilities",
                    resources: {
                        maxCredits: 2000,
                        maxTokens: 100000,
                        maxTime: MINUTES_5_MS * 2,
                        tools: [
                            { name: "code_execution", description: "Execute code safely" },
                            { name: "data_analysis", description: "Analyze datasets" },
                        ],
                    },
                    config: {
                        model: "claude-3-sonnet",
                        temperature: 0.3,
                        autoApproveTools: true,
                        parallelExecutionLimit: 5,
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });
        });

        it("should respect premium user configurations", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                userData: {
                    id: generatePK().toString(),
                    name: "premiumUser",
                    hasPremium: true,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
                config: {
                    name: "Premium Swarm",
                    description: "High-resource swarm for premium users",
                    goal: "Leverage premium capabilities",
                    resources: {
                        maxCredits: 10000,
                        maxTokens: 500000,
                        maxTime: MINUTES_5_MS * 12, // 1 hour
                        tools: [
                            { name: "advanced_search", description: "Premium search capabilities" },
                            { name: "data_export", description: "Export large datasets" },
                        ],
                    },
                    config: {
                        model: "gpt-4",
                        temperature: 0.8,
                        autoApproveTools: false,
                        parallelExecutionLimit: 10,
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });

            // Verify premium user tracking
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms[0].hasPremium).toBe(true);
        });

        it("should handle organization-based swarms", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                input: {
                    ...createTestSwarmExecutionTask().input,
                    goal: "Handle organization-specific tasks",
                    teamConfiguration: {
                        leaderAgentId: generatePK().toString(),
                        preferredTeamSize: 5,
                        requiredSkills: ["team_coordination", "project_management"],
                    },
                    availableTools: [
                        { name: "team_coordination", description: "Coordinate team activities" },
                        { name: "project_management", description: "Manage project tasks" },
                    ],
                    executionConfig: {
                        model: "gpt-4",
                        temperature: 0.5,
                        parallelExecutionLimit: 8,
                    },
                },
                allocation: {
                    maxCredits: "5000",
                    maxTokens: 200000,
                    maxTime: MINUTES_5_MS * 6,
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });
        });
    });

    describe("resource and rate limiting", () => {
        it("should track free user swarms with lower limits", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                context: {
                    ...createTestSwarmExecutionTask().context,
                    userData: {
                        id: generatePK().toString(),
                        name: "freeUser",
                        hasPremium: false,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
                input: {
                    ...createTestSwarmExecutionTask().input,
                    goal: "Basic task completion",
                    availableTools: [
                        { name: "basic_search", description: "Basic search functionality" },
                    ],
                    executionConfig: {
                        model: "gpt-3.5-turbo",
                        temperature: 0.7,
                        parallelExecutionLimit: 1,
                    },
                },
                allocation: {
                    maxCredits: "100",
                    maxTokens: 10000,
                    maxTime: MINUTES_1_MS * 5,
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });

            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms[0].hasPremium).toBe(false);
        });

        it("should handle multiple concurrent swarms per user", async () => {
            const userId = generatePK().toString();

            const swarmTask1 = createTestSwarmExecutionTask({
                context: {
                    ...createTestSwarmExecutionTask().context,
                    userData: {
                        id: userId,
                        name: "testUser",
                        hasPremium: true,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
            });

            const swarmTask2 = createTestSwarmExecutionTask({
                context: {
                    ...createTestSwarmExecutionTask().context,
                    userData: {
                        id: userId,
                        name: "testUser",
                        hasPremium: true,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
            });

            const job1 = createMockSwarmJob(swarmTask1);
            const job2 = createMockSwarmJob(swarmTask2);

            const [result1, result2] = await Promise.all([
                llmProcess(job1),
                llmProcess(job2),
            ]);

            expect(result1).toEqual({ swarmId: swarmTask1.context.swarmId });
            expect(result2).toEqual({ swarmId: swarmTask2.context.swarmId });

            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(2);
            expect(activeSwarms.every(swarm => swarm.userId === userId)).toBe(true);
        });
    });

    describe("error handling", () => {
        it("should handle invalid task type", async () => {
            const invalidTask = {
                ...createTestSwarmExecutionTask(),
                type: "invalid-task-type" as any,
            };

            const job = createMockSwarmJob(invalidTask);

            await expect(llmProcess(job)).rejects.toThrow();

            // Verify no swarm was added to registry on error
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(0);
        });

        it("should handle LLM completion task successfully", async () => {
            const llmTask = createTestLLMCompletionTask();
            const job = createMockSwarmJob(llmTask);

            const result = await llmProcess(job);

            expect(result).toBeDefined();
            expect(result.success).toBe(true);
            expect(result.messageId).toBeDefined();
        });
        // AI_CHECK: COMPLETION_SERVICE_FIX=completionService-test-fix | LAST: 2025-07-09 - Updated test to expect success instead of deprecated error

        it("should handle swarm state machine failures", async () => {
            // Save the original mock
            const { getSwarmCoordinator } = await import("../../services/execution/swarmCoordinatorFactory.js");
            const originalMock = vi.mocked(getSwarmCoordinator).getMockImplementation();
            
            // Mock getSwarmCoordinator to throw an error
            vi.mocked(getSwarmCoordinator).mockImplementationOnce(() => {
                throw new Error("Service unavailable");
            });

            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            await expect(llmProcess(job)).rejects.toThrow("Service unavailable");

            // Verify no swarm was added to registry on error
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(0);
            
            // Restore the original mock
            if (originalMock) {
                vi.mocked(getSwarmCoordinator).mockImplementation(originalMock);
            }
        });

        it("should handle missing swarm configuration", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                input: {
                    ...createTestSwarmExecutionTask().input,
                    goal: "", // Invalid empty goal
                },
            });

            const job = createMockSwarmJob(swarmTask);

            // Should still process but service might handle validation
            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });
        });
    });

    describe("active swarm registry", () => {
        it("should track active swarms", async () => {
            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            // Verify registry is initially empty
            expect(activeSwarmRegistry.listActive()).toHaveLength(0);

            await llmProcess(job);

            // Verify swarm was tracked
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(1);
            expect(activeSwarms[0].id).toBe(swarmTask.context.swarmId);
            expect(activeSwarms[0].userId).toBe(swarmTask.context.userData.id);
            expect(activeSwarms[0].startTime).toBeTypeOf("number");
        });

        it("should create proper state machine adapter", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                context: {
                    ...createTestSwarmExecutionTask().context,
                    userData: {
                        id: generatePK().toString(),
                        name: "premiumUser",
                        hasPremium: true,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            await llmProcess(job);

            // Access the registry internals to verify adapter functionality
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(1);

            // Test adapter functionality indirectly through registry
            const swarmRecord = activeSwarms[0];
            expect(swarmRecord.id).toBe(swarmTask.context.swarmId);
            expect(swarmRecord.userId).toBe(swarmTask.context.userData.id);
        });

        it("should handle registry cleanup", async () => {
            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            await llmProcess(job);

            // Verify swarm was tracked
            expect(activeSwarmRegistry.listActive()).toHaveLength(1);

            // Clear registry
            activeSwarmRegistry.clear();

            // Verify cleanup
            expect(activeSwarmRegistry.listActive()).toHaveLength(0);
        });
    });

    describe("tool and configuration handling", () => {
        it("should handle auto-approved tools", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                input: {
                    ...createTestSwarmExecutionTask().input,
                    goal: "Complete tasks without manual approval",
                    availableTools: [
                        { name: "safe_calculator", description: "Safe mathematical operations" },
                        { name: "text_processor", description: "Process text safely" },
                    ],
                    executionConfig: {
                        model: "gpt-4",
                        temperature: 0.5,
                        parallelExecutionLimit: 3,
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });
        });

        it("should handle manual tool approval flow", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                input: {
                    ...createTestSwarmExecutionTask().input,
                    goal: "Complete tasks with manual oversight",
                    availableTools: [
                        { name: "external_api", description: "Call external APIs" },
                        { name: "file_operations", description: "Perform file operations" },
                    ],
                    executionConfig: {
                        model: "gpt-4",
                        temperature: 0.3,
                        parallelExecutionLimit: 2,
                    },
                },
                allocation: {
                    maxCredits: "2000",
                    maxTokens: 75000,
                    maxTime: MINUTES_5_MS * 2,
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });
        });

        it("should handle different parallel execution limits", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                input: {
                    ...createTestSwarmExecutionTask().input,
                    goal: "Handle multiple tasks simultaneously",
                    availableTools: [
                        { name: "parallel_processor", description: "Process multiple items" },
                        { name: "batch_analyzer", description: "Analyze data in batches" },
                    ],
                    executionConfig: {
                        model: "gpt-4",
                        temperature: 0.6,
                        parallelExecutionLimit: 15, // High concurrency
                    },
                },
                allocation: {
                    maxCredits: "5000",
                    maxTokens: 200000,
                    maxTime: MINUTES_5_MS * 4,
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });
        });
    });

    describe("SwarmCoordinator Integration", () => {
        it("should use SwarmCoordinator directly without adapter", async () => {
            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);

            expect(result).toEqual({ swarmId: swarmTask.context.swarmId });

            // Verify that getSwarmCoordinator was called
            const { getSwarmCoordinator } = await import("../../services/execution/swarmCoordinatorFactory.js");
            expect(getSwarmCoordinator).toHaveBeenCalled();
        });

        it("should add SwarmCoordinator directly to registry", async () => {
            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            await llmProcess(job);

            // Verify swarm was added to registry
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(1);
            expect(activeSwarms[0].id).toBe(swarmTask.context.swarmId);

            // Verify the coordinator is stored in the registry (not an adapter)
            const coordinatorInRegistry = activeSwarmRegistry.get(swarmTask.context.swarmId);
            expect(coordinatorInRegistry).toBeDefined();
            expect(coordinatorInRegistry.getTaskId).toBeDefined();
            expect(coordinatorInRegistry.getState).toBeDefined();
            expect(coordinatorInRegistry.requestPause).toBeDefined();
            expect(coordinatorInRegistry.requestStop).toBeDefined();
        });

        it("should handle coordinator execution errors", async () => {
            // Mock coordinator to return error
            const { getSwarmCoordinator } = await import("../../services/execution/swarmCoordinatorFactory.js");
            vi.mocked(getSwarmCoordinator).mockReturnValueOnce({
                execute: vi.fn().mockResolvedValue({
                    status: "failed",
                    error: { message: "Test execution error" },
                }),
                start: vi.fn().mockResolvedValue({
                    success: false,
                    error: { message: "Test execution error" },
                }),
                getTaskId: vi.fn(),
                getState: vi.fn(),
                requestPause: vi.fn(),
                requestStop: vi.fn(),
                getAssociatedUserId: vi.fn(),
            } as any);

            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            await expect(llmProcess(job)).rejects.toThrow("Test execution error");
        });

        it("should use singleton pattern for coordinator", async () => {
            const swarmTask1 = createTestSwarmExecutionTask();
            const swarmTask2 = createTestSwarmExecutionTask();
            const job1 = createMockSwarmJob(swarmTask1);
            const job2 = createMockSwarmJob(swarmTask2);

            await llmProcess(job1);
            await llmProcess(job2);

            // Verify getSwarmCoordinator was called but returns same instance
            const { getSwarmCoordinator } = await import("../../services/execution/swarmCoordinatorFactory.js");
            expect(getSwarmCoordinator).toHaveBeenCalledTimes(2);

            // Both swarms should be in registry
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(2);
        });
    });
});
