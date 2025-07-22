// AI_CHECK: TEST_QUALITY=1, TEST_COVERAGE=1 | LAST: 2025-06-18

import { generatePK, initIdGenerator, MINUTES_5_MS } from "@vrooli/shared";
import { type Job } from "bullmq";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createRunTask, runTaskScenarios } from "../../__test/fixtures/tasks/runTaskFactory.js";
import "../../__test/setup.js";
import { createMockJob } from "../taskFactory.js";
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

vi.mock("../../services/events/eventBus.js", () => ({
    EventBus: vi.fn().mockImplementation(() => ({})),
}));

vi.mock("../../services/execution/tier3/stepExecutor.js", () => ({
    StepExecutor: vi.fn().mockImplementation(() => ({
        execute: vi.fn().mockResolvedValue({ success: true, outputs: {} }),
    })),
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

    function createMockRunJob(overrides: Partial<RunTask> = {}): Job<RunTask> {
        const runTask = createRunTask(overrides);
        return createMockJob<RunTask>(
            QueueTaskType.RUN_START,
            runTask,
        );
    }

    describe("successful execution", () => {
        it("should execute simple routine", async () => {
            const job = createMockRunJob();

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify run was added to active registry
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].id).toBe(job.data.input.runId);
        });

        it("should handle new run initialization", async () => {
            const job = createMockRunJob({
                ...runTaskScenarios.newRun(),
                ...runTaskScenarios.premiumUser(),
            });

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify registry tracking
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].userId).toBe(runTask.userData.id);
            expect(activeRuns[0].hasPremium).toBe(true);
        });

        it("should handle existing run resumption", async () => {
            const runId = generatePK().toString();
            const job = createMockRunJob(runTaskScenarios.resumeRun(runId));

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify run was added to active registry even for resumed runs
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
        });

        it("should respect premium user configuration", async () => {
            const job = createMockRunJob({
                ...runTaskScenarios.premiumUser(),
                input: {
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
                },
            });

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify premium user tracking
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns[0].hasPremium).toBe(true);
        });

        it("should handle complex configuration", async () => {
            const job = createMockRunJob({
                input: {
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
                },
            });

            const result = await runProcess(job);

            expect(result).toEqual({ __typename: "Success", success: true });
        });
    });

    describe("error handling", () => {
        it("should handle invalid task type", async () => {
            const job = createMockRunJob({
                type: "invalid-task-type" as any,
            });

            await expect(runProcess(job)).rejects.toThrow();

            // Verify no run was added to registry on error
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(0);
        });

        it("should handle missing resource version", async () => {
            const job = createMockRunJob({
                input: {
                    resourceVersionId: "invalid-resource-id",
                },
            });

            await expect(runProcess(job)).rejects.toThrow();

            // Verify no run was added to registry on error
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(0);
        });

        it("should handle invalid user data", async () => {
            const job = createMockRunJob({
                context: {
                    userData: {
                        id: "invalid-user-id",
                        name: "invalidUser",
                        hasPremium: false,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
            });

            // This should still process but with invalid user data
            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });

            // Verify run tracking with invalid user ID
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].userId).toBe("invalid-user-id");
        });

        it("should handle missing configuration gracefully", async () => {
            const job = createMockRunJob({
                input: {
                    config: undefined, // Missing config
                },
            });

            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });
        });
    });

    describe("user tier and limits", () => {
        it("should track free user runs", async () => {
            const job = createMockRunJob({
                context: {
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
                },
            });

            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });

            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns[0].hasPremium).toBe(false);
        });

        it("should track premium user runs", async () => {
            const job = createMockRunJob(runTaskScenarios.premiumUser());

            const result = await runProcess(job);
            expect(result).toEqual({ __typename: "Success", success: true });

            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns[0].hasPremium).toBe(true);
        });

        it("should handle multiple concurrent runs per user", async () => {
            const userId = generatePK().toString();

            const job1 = createMockRunJob({
                context: {
                    userData: {
                        id: userId,
                        name: "testUser",
                        hasPremium: false,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
            });

            const job2 = createMockRunJob({
                context: {
                    userData: {
                        id: userId,
                        name: "testUser",
                        hasPremium: false,
                        languages: ["en"],
                        roles: [],
                        wallets: [],
                        theme: "light",
                    },
                },
            });

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
            const job = createMockRunJob();

            // Verify registry is initially empty
            expect(activeRunRegistry.listActive()).toHaveLength(0);

            await runProcess(job);

            // Verify run was tracked
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);
            expect(activeRuns[0].id).toBe(job.data.input.runId);
            expect(activeRuns[0].userId).toBe(job.data.context.userData.id);
            expect(activeRuns[0].startTime).toBeTypeOf("number");
        });

        it("should create proper state machine adapter", async () => {
            const job = createMockRunJob(runTaskScenarios.premiumUser());

            await runProcess(job);

            // Access the registry internals to verify adapter functionality
            const activeRuns = activeRunRegistry.listActive();
            expect(activeRuns).toHaveLength(1);

            // Test adapter functionality indirectly through registry
            const runRecord = activeRuns[0];
            expect(runRecord.id).toBe(job.data.input.runId);
            expect(runRecord.userId).toBe(job.data.context.userData.id);
        });

        it("should handle registry cleanup", async () => {
            const job = createMockRunJob();

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
            expect(adapter.getState()).toBe("RUNNING");
        });

        it("should handle pause requests", async () => {
            const adapter = new NewRunStateMachineAdapter("test-run-123", {} as any, "user-123");
            const result = await adapter.requestPause();
            expect(result).toBe(false); // Pause not currently supported
        });
    });
});
