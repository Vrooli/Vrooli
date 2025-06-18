// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-18

import { generatePK, initIdGenerator, MINUTES_5_MS } from "@vrooli/shared";
import { type Job } from "bullmq";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import "../../__test/setup.js";
import { type RunTask, QueueTaskType } from "../taskTypes.js";

// TODO: Remove these mocks once the execution tier modules are optimized for test loading.
// The tier execution components are too heavy for test environments and cause worker memory issues.
// These mocks should be replaced with proper integration tests that use lightweight test doubles
// or actual tier implementations in a test-specific configuration.
vi.mock("../../services/execution/tier2/tierTwoOrchestrator.js", () => ({
    TierTwoOrchestrator: vi.fn().mockImplementation(() => ({
        startRun: vi.fn().mockResolvedValue(undefined),
        cancelRun: vi.fn().mockResolvedValue(undefined),
    })),
}));

vi.mock("../../services/execution/cross-cutting/events/eventBus.js", () => ({
    EventBus: vi.fn().mockImplementation(() => ({})),
}));

vi.mock("../../services/execution/tier3/TierThreeExecutor.js", () => ({
    TierThreeExecutor: vi.fn().mockImplementation(() => ({})),
}));

// Import after mocking to avoid loading the heavy dependencies
let runProcess: any;
let activeRunRegistry: any;
let NewRunStateMachineAdapter: any;

describe("runProcess", () => {
    beforeEach(async () => {
        vi.clearAllMocks();

        // Initialize ID generator for test data creation
        await initIdGenerator(0);

        // Dynamically import after mocks are set up
        const processModule = await import("./process.js");
        runProcess = processModule.runProcess;
        activeRunRegistry = processModule.activeRunRegistry;
        NewRunStateMachineAdapter = processModule.NewRunStateMachineAdapter;

        // Clear the active run registry
        if (activeRunRegistry && activeRunRegistry.clear) {
            activeRunRegistry.clear();
        }
    });

    afterEach(async () => {
        // Clear the active run registry
        if (activeRunRegistry && activeRunRegistry.clear) {
            activeRunRegistry.clear();
        }
    });

    const createTestRunTask = (overrides: Partial<RunTask> = {}): RunTask => {
        const userId = generatePK();
        const runId = generatePK();
        const resourceVersionId = generatePK();

        return {
            taskType: QueueTaskType.RUN_START,
            type: QueueTaskType.RUN_START,
            runId: runId.toString(),
            resourceVersionId: resourceVersionId.toString(),
            isNewRun: true,
            runFrom: "Trigger",
            startedById: userId.toString(),
            status: "Scheduled",
            userData: {
                id: userId.toString(),
                name: "testUser",
                hasPremium: false,
                languages: ["en"],
                roles: [],
                wallets: [],
                theme: "light",
            },
            config: {
                botConfig: {
                    strategy: "reasoning",
                    model: "gpt-4",
                },
                limits: {
                    maxSteps: 50,
                    maxTime: MINUTES_5_MS,
                },
            },
            formValues: {
                input1: "test value",
                input2: 42,
            },
            ...overrides,
        };
    };

    const createMockRunJob = (data: Partial<RunTask> = {}): Job<RunTask> => {
        const defaultData: RunTask = createTestRunTask(data);

        return {
            id: "run-job-id",
            data: defaultData,
            name: "run",
            attemptsMade: 0,
            opts: {},
        } as Job<RunTask>;
    };

    describe("successful execution", () => {
        it("should execute simple routine", async () => {
            const runTask = createTestRunTask();
            const job = createMockRunJob(runTask);

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify run was added to active registry
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].id).toBe(runTask.runId);
        });

        it("should handle new run initialization", async () => {
            const runTask = createTestRunTask({
                isNewRun: true,
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

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify registry tracking
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].userId).toBe(runTask.userData.id);
            expect(activeRuns[0].hasPremium).toBe(true);
        });

        it("should handle existing run resumption", async () => {
            const runTask = createTestRunTask({
                isNewRun: false, // Resuming existing run
            });

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify run was added to active registry even for resumed runs
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
        });

        it("should respect premium user configuration", async () => {
            const runTask = createTestRunTask({
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
                    botConfig: {
                        strategy: "advanced",
                        model: "gpt-4",
                    },
                    limits: {
                        maxSteps: 200,
                        maxTime: MINUTES_5_MS * 12, // 1 hour
                    },
                },
            });

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify premium user tracking
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns[0].hasPremium).toBe(true);
        });

        it("should handle complex configuration", async () => {
            const runTask = createTestRunTask({
                config: {
                    botConfig: {
                        strategy: "advanced",
                        model: "gpt-4",
                    },
                    limits: {
                        maxSteps: 100,
                        maxTime: MINUTES_5_MS * 12, // 1 hour
                    },
                },
            });

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });
        });
    });

    describe("error handling", () => {
        it("should handle invalid task type", async () => {
            const runTask = createTestRunTask({
                type: "invalid-task-type" as any,
            });

            const job = createMockRunJob(runTask);

            await expect(runProcess(job)).rejects.toThrow();

            // Verify no run was added to registry on error
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(0);
        });

        it("should handle missing resource version", async () => {
            const runTask = createTestRunTask({
                resourceVersionId: "invalid-resource-id",
            });

            const job = createMockRunJob(runTask);

            await expect(runProcess(job)).rejects.toThrow();

            // Verify no run was added to registry on error
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(0);
        });

        it("should handle invalid user data", async () => {
            const runTask = createTestRunTask({
                userData: {
                    id: "invalid-user-id",
                    name: "invalidUser",
                    hasPremium: false,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
            });

            const job = createMockRunJob(runTask);

            // This should still process but with invalid user data
            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify run tracking with invalid user ID
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].userId).toBe("invalid-user-id");
        });

        it("should handle missing configuration gracefully", async () => {
            const runTask = createTestRunTask({
                config: undefined, // Missing config
            });

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });
        });
    });

    describe("user tier and limits", () => {
        it("should track free user runs", async () => {
            const runTask = createTestRunTask({
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
                    botConfig: {
                        strategy: "simple",
                        model: "gpt-3.5-turbo",
                    },
                    limits: {
                        maxSteps: 10,
                        maxTime: MINUTES_5_MS / 2,
                    },
                },
            });

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });

            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns[0].hasPremium).toBe(false);
        });

        it("should track premium user runs", async () => {
            const runTask = createTestRunTask({
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
                    botConfig: {
                        strategy: "advanced",
                        model: "gpt-4",
                    },
                    limits: {
                        maxSteps: 200,
                        maxTime: MINUTES_5_MS * 12, // 1 hour
                    },
                },
            });

            const job = createMockRunJob(runTask);

            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });

            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns[0].hasPremium).toBe(true);
        });

        it("should handle multiple concurrent runs per user", async () => {
            const userId = generatePK().toString();

            const runTask1 = createTestRunTask({
                userData: {
                    id: userId,
                    name: "testUser",
                    hasPremium: false,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
            });

            const runTask2 = createTestRunTask({
                userData: {
                    id: userId,
                    name: "testUser",
                    hasPremium: false,
                    languages: ["en"],
                    roles: [],
                    wallets: [],
                    theme: "light",
                },
            });

            const job1 = createMockRunJob(runTask1);
            const job2 = createMockRunJob(runTask2);

            const [result1, result2] = await Promise.all([
                runProcess(job1),
                runProcess(job2),
            ]);

            expect(result1).toEqual({ __typename: "Success", success: true });
            expect(result2).toEqual({ __typename: "Success", success: true });

            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(2);
            expect(activeRuns.every(run => run.userId === userId)).toBe(true);
        });
    });

    describe("active run registry", () => {
        it("should track active runs", async () => {
            const runTask = createTestRunTask();
            const job = createMockRunJob(runTask);

            // Verify registry is initially empty
            expect(activeRunRegistry.listActive()).toHaveLength(0);

            await runProcess(job);

            // Verify run was tracked
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].id).toBe(runTask.runId);
            expect(activeRuns[0].userId).toBe(runTask.userData.id);
            expect(activeRuns[0].startTime).toBeTypeOf("number");
        });

        it("should create proper state machine adapter", async () => {
            const runTask = createTestRunTask({
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

            const job = createMockRunJob(runTask);

            await runProcess(job);

            // Access the registry internals to verify adapter functionality
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);

            // Test adapter functionality indirectly through registry
            const runRecord = activeRuns[0];
            expect(runRecord.id).toBe(runTask.runId);
            expect(runRecord.userId).toBe(runTask.userData.id);
        });

        it("should handle registry cleanup", async () => {
            const runTask = createTestRunTask();
            const job = createMockRunJob(runTask);

            await runProcess(job);

            // Verify run was tracked
            expect(activeRunRegistry.listActive()).toHaveLength(1);

            // Clear registry
            activeRunRegistry.clear();

            // Verify cleanup
            expect(activeRunRegistry.listActive()).toHaveLength(0);
        });
    });

    describe("NewRunStateMachineAdapter", () => {
        it("should provide correct task ID", () => {
            const adapter = new NewRunStateMachineAdapter("test-run-123", {} as any, "user-123");
            expect(adapter.getTaskId()).toBe("test-run-123");
        });

        it("should provide user ID", () => {
            const adapter = new NewRunStateMachineAdapter("test-run-123", {} as any, "user-123");
            expect(adapter.getAssociatedUserId?.()).toBe("user-123");
        });

        it("should return default status", () => {
            const adapter = new NewRunStateMachineAdapter("test-run-123", {} as any, "user-123");
            expect(adapter.getCurrentSagaStatus()).toBe("RUNNING");
        });

        it("should handle pause requests", async () => {
            const adapter = new NewRunStateMachineAdapter("test-run-123", {} as any, "user-123");
            const result = await adapter.requestPause();
            expect(result).toBe(false); // Pause not currently supported
        });
    });
});
