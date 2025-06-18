// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-18
import { generatePK, initIdGenerator, MINUTES_1_MS, MINUTES_5_MS } from "@vrooli/shared";
import { type Job } from "bullmq";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { type LLMCompletionTask, QueueTaskType, type SwarmExecutionTask } from "../taskTypes.js";

// TODO: Remove these mocks once the swarm execution modules are optimized for test loading.
// The swarm execution components are too heavy for test environments and cause worker memory issues.
// These mocks should be replaced with proper integration tests that use lightweight test doubles
// or actual swarm implementations in a test-specific configuration.
vi.mock("../../services/execution/swarmExecutionService.js", () => ({
    SwarmExecutionService: vi.fn().mockImplementation(() => ({
        startSwarm: vi.fn().mockResolvedValue({ swarmId: "swarm-123", success: true }),
        cancelSwarm: vi.fn().mockResolvedValue({ success: true }),
    })),
}));

vi.mock("../../services/execution/tier1/coordination/swarmStateMachine.js", () => ({
    SwarmStateMachine: vi.fn().mockImplementation(() => ({})),
}));

// Import after mocking to avoid loading the heavy dependencies
let llmProcess: any;
let activeSwarmRegistry: any;
let NewSwarmStateMachineAdapter: any;

describe("llmProcess", () => {
    beforeEach(async () => {
        vi.clearAllMocks();

        // Initialize ID generator for test data creation
        await initIdGenerator(0);

        // Dynamically import after mocks are set up
        const processModule = await import("./process.js");
        llmProcess = processModule.llmProcess;
        activeSwarmRegistry = processModule.activeSwarmRegistry;
        NewSwarmStateMachineAdapter = processModule.NewSwarmStateMachineAdapter;

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

    const createTestSwarmExecutionTask = (overrides: Partial<SwarmExecutionTask> = {}): SwarmExecutionTask => {
        const swarmId = generatePK().toString();
        const userId = generatePK().toString();

        return {
            taskType: QueueTaskType.SWARM_EXECUTION,
            type: QueueTaskType.SWARM_EXECUTION,
            swarmId,
            config: {
                name: "Test Swarm",
                description: "A test swarm for automated testing",
                goal: "Complete the test objectives efficiently",
                resources: {
                    maxCredits: 1000,
                    maxTokens: 50000,
                    maxTime: MINUTES_5_MS,
                    tools: [
                        { name: "calculator", description: "Perform mathematical calculations" },
                        { name: "web_search", description: "Search the web for information" },
                    ],
                },
                config: {
                    model: "gpt-4",
                    temperature: 0.7,
                    autoApproveTools: false,
                    parallelExecutionLimit: 3,
                },
                organizationId: undefined,
                leaderBotId: undefined,
            },
            userData: {
                id: userId,
                name: "testUser",
                hasPremium: false,
                languages: ["en"],
                roles: [],
                wallets: [],
                theme: "light",
            },
            ...overrides,
        };
    };

    const createTestLLMCompletionTask = (overrides: Partial<LLMCompletionTask> = {}): LLMCompletionTask => {
        const chatId = generatePK().toString();
        const messageId = generatePK().toString();
        const userId = generatePK().toString();

        return {
            taskType: QueueTaskType.LLM_COMPLETION,
            type: QueueTaskType.LLM_COMPLETION,
            chatId,
            messageId,
            model: "gpt-4",
            taskContexts: [],
            userData: {
                id: userId,
                name: "testUser",
                hasPremium: false,
                languages: ["en"],
                roles: [],
                wallets: [],
                theme: "light",
            },
            respondingBot: {
                id: generatePK().toString(),
                publicId: "test-bot",
                handle: "testbot",
            },
            ...overrides,
        };
    };

    const createMockSwarmJob = (data: Partial<SwarmExecutionTask | LLMCompletionTask> = {}): Job<SwarmExecutionTask | LLMCompletionTask> => {
        const defaultData = data.type === QueueTaskType.LLM_COMPLETION
            ? createTestLLMCompletionTask(data as Partial<LLMCompletionTask>)
            : createTestSwarmExecutionTask(data as Partial<SwarmExecutionTask>);

        return {
            id: "swarm-job-id",
            data: defaultData,
            name: "swarm",
            attemptsMade: 0,
            opts: {},
        } as Job<SwarmExecutionTask | LLMCompletionTask>;
    };

    describe("successful swarm execution", () => {
        it("should process swarm execution task", async () => {
            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);

            expect(result).toEqual({ swarmId: "swarm-123", success: true });

            // Verify swarm was added to active registry
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(1);
            expect(activeSwarms[0].id).toBe(swarmTask.swarmId);
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
            expect(result).toEqual({ swarmId: "swarm-123", success: true });
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
            expect(result).toEqual({ swarmId: "swarm-123", success: true });

            // Verify premium user tracking
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms[0].hasPremium).toBe(true);
        });

        it("should handle organization-based swarms", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                config: {
                    name: "Organization Swarm",
                    description: "Swarm for organization use",
                    goal: "Handle organization-specific tasks",
                    resources: {
                        maxCredits: 5000,
                        maxTokens: 200000,
                        maxTime: MINUTES_5_MS * 6,
                        tools: [
                            { name: "team_coordination", description: "Coordinate team activities" },
                            { name: "project_management", description: "Manage project tasks" },
                        ],
                    },
                    config: {
                        model: "gpt-4",
                        temperature: 0.5,
                        autoApproveTools: true,
                        parallelExecutionLimit: 8,
                    },
                    organizationId: generatePK().toString(),
                    leaderBotId: generatePK().toString(),
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: "swarm-123", success: true });
        });
    });

    describe("resource and rate limiting", () => {
        it("should track free user swarms with lower limits", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                userData: {
                    id: generatePK().toString(),
                    name: "freeUser",
                    hasPremium: false,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
                config: {
                    name: "Free Tier Swarm",
                    description: "Limited swarm for free users",
                    goal: "Basic task completion",
                    resources: {
                        maxCredits: 100,
                        maxTokens: 10000,
                        maxTime: MINUTES_1_MS * 5,
                        tools: [
                            { name: "basic_search", description: "Basic search functionality" },
                        ],
                    },
                    config: {
                        model: "gpt-3.5-turbo",
                        temperature: 0.7,
                        autoApproveTools: false,
                        parallelExecutionLimit: 1,
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: "swarm-123", success: true });

            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms[0].hasPremium).toBe(false);
        });

        it("should handle multiple concurrent swarms per user", async () => {
            const userId = generatePK().toString();

            const swarmTask1 = createTestSwarmExecutionTask({
                userData: {
                    id: userId,
                    name: "testUser",
                    hasPremium: true,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
            });

            const swarmTask2 = createTestSwarmExecutionTask({
                userData: {
                    id: userId,
                    name: "testUser",
                    hasPremium: true,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
            });

            const job1 = createMockSwarmJob(swarmTask1);
            const job2 = createMockSwarmJob(swarmTask2);

            const [result1, result2] = await Promise.all([
                llmProcess(job1),
                llmProcess(job2),
            ]);

            expect(result1).toEqual({ swarmId: "swarm-123", success: true });
            expect(result2).toEqual({ swarmId: "swarm-123", success: true });

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

        it("should handle LLM completion task (deprecated)", async () => {
            const llmTask = createTestLLMCompletionTask();
            const job = createMockSwarmJob(llmTask);

            await expect(llmProcess(job)).rejects.toThrow("LLM completion through conversation service deprecated");

            // Verify no task was added to registry on error
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(0);
        });

        it("should handle swarm execution service failures", async () => {
            // Mock the service to throw an error
            const processModule = await import("./process.js");
            const mockSwarmService = {
                startSwarm: vi.fn().mockRejectedValue(new Error("Service unavailable")),
                cancelSwarm: vi.fn(),
            };

            // Temporarily replace the service
            vi.doMock("../../services/execution/swarmExecutionService.js", () => ({
                SwarmExecutionService: vi.fn().mockImplementation(() => mockSwarmService),
            }));

            const swarmTask = createTestSwarmExecutionTask();
            const job = createMockSwarmJob(swarmTask);

            await expect(llmProcess(job)).rejects.toThrow("Service unavailable");

            // Verify no swarm was added to registry on error
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(0);
        });

        it("should handle missing swarm configuration", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                config: {
                    ...createTestSwarmExecutionTask().config,
                    name: "", // Invalid empty name
                },
            });

            const job = createMockSwarmJob(swarmTask);

            // Should still process but service might handle validation
            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: "swarm-123", success: true });
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
            expect(activeSwarms[0].id).toBe(swarmTask.swarmId);
            expect(activeSwarms[0].userId).toBe(swarmTask.userData.id);
            expect(activeSwarms[0].startTime).toBeTypeOf("number");
        });

        it("should create proper state machine adapter", async () => {
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
            });

            const job = createMockSwarmJob(swarmTask);

            await llmProcess(job);

            // Access the registry internals to verify adapter functionality
            const activeSwarms = activeSwarmRegistry.listActive();
            expect(activeSwarms).toHaveLength(1);

            // Test adapter functionality indirectly through registry
            const swarmRecord = activeSwarms[0];
            expect(swarmRecord.id).toBe(swarmTask.swarmId);
            expect(swarmRecord.userId).toBe(swarmTask.userData.id);
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
                config: {
                    name: "Auto-Approve Swarm",
                    description: "Swarm with auto-approved tools",
                    goal: "Complete tasks without manual approval",
                    resources: {
                        maxCredits: 1000,
                        maxTokens: 50000,
                        maxTime: MINUTES_5_MS,
                        tools: [
                            { name: "safe_calculator", description: "Safe mathematical operations" },
                            { name: "text_processor", description: "Process text safely" },
                        ],
                    },
                    config: {
                        model: "gpt-4",
                        temperature: 0.5,
                        autoApproveTools: true, // Auto-approve enabled
                        parallelExecutionLimit: 3,
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: "swarm-123", success: true });
        });

        it("should handle manual tool approval flow", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                config: {
                    name: "Manual Approval Swarm",
                    description: "Swarm requiring manual tool approval",
                    goal: "Complete tasks with manual oversight",
                    resources: {
                        maxCredits: 2000,
                        maxTokens: 75000,
                        maxTime: MINUTES_5_MS * 2,
                        tools: [
                            { name: "external_api", description: "Call external APIs" },
                            { name: "file_operations", description: "Perform file operations" },
                        ],
                    },
                    config: {
                        model: "gpt-4",
                        temperature: 0.3,
                        autoApproveTools: false, // Manual approval required
                        parallelExecutionLimit: 2,
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: "swarm-123", success: true });
        });

        it("should handle different parallel execution limits", async () => {
            const swarmTask = createTestSwarmExecutionTask({
                config: {
                    name: "High Concurrency Swarm",
                    description: "Swarm with high parallel execution",
                    goal: "Handle multiple tasks simultaneously",
                    resources: {
                        maxCredits: 5000,
                        maxTokens: 200000,
                        maxTime: MINUTES_5_MS * 4,
                        tools: [
                            { name: "parallel_processor", description: "Process multiple items" },
                            { name: "batch_analyzer", description: "Analyze data in batches" },
                        ],
                    },
                    config: {
                        model: "gpt-4",
                        temperature: 0.6,
                        autoApproveTools: true,
                        parallelExecutionLimit: 15, // High concurrency
                    },
                },
            });

            const job = createMockSwarmJob(swarmTask);

            const result = await llmProcess(job);
            expect(result).toEqual({ swarmId: "swarm-123", success: true });
        });
    });

    describe("NewSwarmStateMachineAdapter", () => {
        it("should provide correct task ID", () => {
            const adapter = new NewSwarmStateMachineAdapter("swarm-123", {} as any, "user-123");
            expect(adapter.getTaskId()).toBe("swarm-123");
        });

        it("should provide user ID", () => {
            const adapter = new NewSwarmStateMachineAdapter("swarm-123", {} as any, "user-123");
            expect(adapter.getAssociatedUserId?.()).toBe("user-123");
        });

        it("should return default status", () => {
            const adapter = new NewSwarmStateMachineAdapter("swarm-123", {} as any, "user-123");
            expect(adapter.getCurrentSagaStatus()).toBe("RUNNING");
        });

        it("should handle pause requests", async () => {
            const adapter = new NewSwarmStateMachineAdapter("swarm-123", {} as any, "user-123");
            const result = await adapter.requestPause();
            expect(result).toBe(false); // Pause not currently supported
        });
    });
});
