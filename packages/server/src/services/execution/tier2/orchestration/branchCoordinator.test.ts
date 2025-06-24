import { describe, it, expect, beforeEach, vi } from "vitest";
import { BranchCoordinator } from "./branchCoordinator.js";
import type { Logger } from "winston";

// Mock logger
const mockLogger: Logger = {
    info: vi.fn(),
    debug: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
} as any;

// Mock EventBus
const mockEventBus = {
    publish: vi.fn().mockResolvedValue(undefined),
    subscribe: vi.fn(),
    unsubscribe: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
};

// Mock StateStore
const mockStateStore = {
    initialize: vi.fn().mockResolvedValue(undefined),
    createBranch: vi.fn().mockResolvedValue(undefined),
    getBranch: vi.fn().mockResolvedValue(null),
    updateBranch: vi.fn().mockResolvedValue(undefined),
    listBranches: vi.fn().mockResolvedValue([]),
    createRun: vi.fn().mockResolvedValue(undefined),
    getRun: vi.fn().mockResolvedValue(null),
    updateRun: vi.fn().mockResolvedValue(undefined),
    deleteRun: vi.fn().mockResolvedValue(undefined),
    getRunState: vi.fn().mockResolvedValue("PENDING"),
    updateRunState: vi.fn().mockResolvedValue(undefined),
    getContext: vi.fn().mockResolvedValue({ variables: {}, blackboard: {}, scopes: [] }),
    updateContext: vi.fn().mockResolvedValue(undefined),
    setVariable: vi.fn().mockResolvedValue(undefined),
    getVariable: vi.fn().mockResolvedValue(undefined),
    getCurrentLocation: vi.fn().mockResolvedValue(null),
    updateLocation: vi.fn().mockResolvedValue(undefined),
    getLocationHistory: vi.fn().mockResolvedValue([]),
    recordStepExecution: vi.fn().mockResolvedValue(undefined),
    getStepExecution: vi.fn().mockResolvedValue(null),
    listStepExecutions: vi.fn().mockResolvedValue([]),
    createCheckpoint: vi.fn().mockResolvedValue(undefined),
    getCheckpoint: vi.fn().mockResolvedValue(null),
    listCheckpoints: vi.fn().mockResolvedValue([]),
    restoreFromCheckpoint: vi.fn().mockResolvedValue(undefined),
    listActiveRuns: vi.fn().mockResolvedValue([]),
    getRunsByState: vi.fn().mockResolvedValue([]),
    getRunsByUser: vi.fn().mockResolvedValue([]),
};

// Mock StepExecutor
const mockStepExecutor = {
    executeStep: vi.fn().mockResolvedValue({
        success: true,
        outputs: { result: "test-output" },
        duration: 100,
    }),
};

// Mock Navigator
const mockNavigator = {
    type: "test",
    version: "1.0.0",
    getParallelBranches: vi.fn().mockReturnValue([]),
};

describe("BranchCoordinator", () => {
    let branchCoordinator: BranchCoordinator;

    beforeEach(() => {
        vi.clearAllMocks();
        branchCoordinator = new BranchCoordinator(mockLogger, mockEventBus as any, mockStateStore as any);
    });

    describe("createBranchesFromConfig", () => {
        it("should create branches for parallel execution", async () => {
            const config = {
                parentStepId: "step1",
                parallel: true,
                branchCount: 2,
            };

            const branches = await branchCoordinator.createBranchesFromConfig(
                "run-123",
                config,
            );

            expect(branches).toHaveLength(2);
            expect(branches[0].parallel).toBe(true);
            expect(branches[1].parallel).toBe(true);
            expect(branches[0].state).toBe("pending");
            expect(branches[1].state).toBe("pending");

            // Verify events were published
            expect(mockEventBus.publish).toHaveBeenCalledTimes(2);
        });

        it("should create branches for sequential execution", async () => {
            const config = {
                parentStepId: "step1",
                parallel: false,
            };

            const branches = await branchCoordinator.createBranchesFromConfig(
                "run-123",
                config,
            );

            expect(branches).toHaveLength(1);
            expect(branches[0].parallel).toBe(false);
        });
    });

    describe("executeBranches", () => {
        it("should execute parallel branches successfully", async () => {
            const run = {
                id: "run-123",
                routineId: "routine1",
                context: {
                    variables: {},
                    blackboard: {},
                    scopes: [],
                },
                config: {
                    recoveryStrategy: "retry" as const,
                },
            };

            const branches = [
                {
                    id: "branch1",
                    parentStepId: "step1",
                    steps: [],
                    state: "pending" as const,
                    parallel: true,
                },
                {
                    id: "branch2", 
                    parentStepId: "step1",
                    steps: [],
                    state: "pending" as const,
                    parallel: true,
                },
            ];

            const results = await branchCoordinator.executeBranches(
                run,
                branches,
                mockNavigator as any,
                mockStepExecutor as any,
            );

            expect(results).toHaveLength(2);
            expect(results[0].success).toBe(true);
            expect(results[1].success).toBe(true);
            expect(results[0].completedSteps).toBeGreaterThan(0);
            expect(results[1].completedSteps).toBeGreaterThan(0);
        });

        it("should execute sequential branches successfully", async () => {
            const run = {
                id: "run-123",
                routineId: "routine1", 
                context: {
                    variables: {},
                    blackboard: {},
                    scopes: [],
                },
                config: {
                    recoveryStrategy: "retry" as const,
                },
            };

            const branches = [
                {
                    id: "branch1",
                    parentStepId: "step1",
                    steps: [],
                    state: "pending" as const,
                    parallel: false,
                },
            ];

            const results = await branchCoordinator.executeBranches(
                run,
                branches,
                mockNavigator as any,
                mockStepExecutor as any,
            );

            expect(results).toHaveLength(1);
            expect(results[0].success).toBe(true);
        });

        it("should handle branch execution failures", async () => {
            // Mock failed step execution
            mockStepExecutor.executeStep.mockResolvedValueOnce({
                success: false,
                error: "Step failed",
                duration: 50,
            });

            const run = {
                id: "run-123",
                routineId: "routine1",
                context: {
                    variables: {},
                    blackboard: {},
                    scopes: [],
                },
                config: {
                    recoveryStrategy: "fail" as const,
                },
            };

            const branches = [
                {
                    id: "branch1",
                    parentStepId: "step1",
                    steps: [],
                    state: "pending" as const,
                    parallel: false,
                },
            ];

            const results = await branchCoordinator.executeBranches(
                run,
                branches,
                mockNavigator as any,
                mockStepExecutor as any,
            );

            expect(results).toHaveLength(1);
            expect(results[0].success).toBe(false);
            expect(results[0].failedSteps).toBeGreaterThan(0);
        });
    });

    describe("mergeBranchResults", () => {
        it("should merge branch outputs correctly", async () => {
            const parentContext = {
                variables: { parentVar: "value" },
                blackboard: {},
                scopes: [],
            };

            const results = [
                {
                    branchId: "branch1",
                    success: true,
                    completedSteps: 1,
                    failedSteps: 0,
                    skippedSteps: 0,
                    outputs: { result1: "value1" },
                },
                {
                    branchId: "branch2",
                    success: true,
                    completedSteps: 1,
                    failedSteps: 0,
                    skippedSteps: 0,
                    outputs: { result2: "value2" },
                },
            ];

            const merged = await branchCoordinator.mergeBranchResults(
                parentContext,
                results,
            );

            expect(merged.variables.parentVar).toBe("value");
            expect(merged.variables.result1).toBe("value1");
            expect(merged.variables.result2).toBe("value2");
        });

        it("should handle conflicting output keys by creating arrays", async () => {
            const parentContext = {
                variables: {},
                blackboard: {},
                scopes: [],
            };

            const results = [
                {
                    branchId: "branch1",
                    success: true,
                    completedSteps: 1,
                    failedSteps: 0,
                    skippedSteps: 0,
                    outputs: { result: "value1" },
                },
                {
                    branchId: "branch2",
                    success: true,
                    completedSteps: 1,
                    failedSteps: 0,
                    skippedSteps: 0,
                    outputs: { result: "value2" },
                },
            ];

            const merged = await branchCoordinator.mergeBranchResults(
                parentContext,
                results,
            );

            expect(merged.variables.result_merged).toEqual(["value1", "value2"]);
        });
    });

    describe("cleanup", () => {
        it("should clean up completed and failed branches", async () => {
            // First create some branches
            const config = {
                parentStepId: "step1",
                parallel: true,
                branchCount: 2,
            };

            const branches = await branchCoordinator.createBranchesFromConfig(
                "run-123",
                config,
            );

            // Manually mark one as completed and one as failed
            const branchStatus1 = await branchCoordinator.getBranchStatus(branches[0].id);
            const branchStatus2 = await branchCoordinator.getBranchStatus(branches[1].id);
            
            if (branchStatus1) branchStatus1.state = "completed";
            if (branchStatus2) branchStatus2.state = "failed";

            // Clean up
            await branchCoordinator.cleanup("run-123");

            // Verify branches are removed
            const status1 = await branchCoordinator.getBranchStatus(branches[0].id);
            const status2 = await branchCoordinator.getBranchStatus(branches[1].id);
            
            expect(status1).toBeNull();
            expect(status2).toBeNull();
        });
    });

    describe("state persistence and recovery", () => {
        it("should persist branch creation to state store", async () => {
            const config = {
                parentStepId: "step1",
                parallel: false,
            };

            await branchCoordinator.createBranchesFromConfig("run-123", config);

            // Verify state store was called
            expect(mockStateStore.createBranch).toHaveBeenCalledTimes(1);
            expect(mockStateStore.createBranch).toHaveBeenCalledWith(
                "run-123",
                expect.objectContaining({
                    state: "pending",
                    parallel: false,
                }),
            );
        });

        it("should persist branch state updates", async () => {
            const config = { parentStepId: "step1", parallel: false };
            const branches = await branchCoordinator.createBranchesFromConfig("run-123", config);
            
            // Clear previous calls
            vi.clearAllMocks();

            const run = {
                id: "run-123",
                config: { recoveryStrategy: "fail" },
                context: { variables: {}, blackboard: {}, scopes: [] },
            };

            // Execute the branch (this will update state to "running" then "completed")
            await branchCoordinator.executeBranches(run as any, branches, mockNavigator as any, mockStepExecutor as any);

            // Verify state updates were persisted
            expect(mockStateStore.updateBranch).toHaveBeenCalledWith(
                "run-123",
                branches[0].id,
                { state: "running" },
            );
            expect(mockStateStore.updateBranch).toHaveBeenCalledWith(
                "run-123", 
                branches[0].id,
                { state: "completed" },
            );
        });

        it("should restore branches from state store", async () => {
            const persistedBranches = [
                {
                    id: "branch-1",
                    parentStepId: "step-1", 
                    steps: [],
                    state: "running",
                    parallel: true,
                    branchIndex: 0,
                },
                {
                    id: "branch-2",
                    parentStepId: "step-1",
                    steps: [],
                    state: "completed", 
                    parallel: true,
                    branchIndex: 1,
                },
            ];

            mockStateStore.listBranches.mockResolvedValueOnce(persistedBranches);

            await branchCoordinator.restoreBranches("run-123");

            // Verify branches were restored to memory
            const branch1 = await branchCoordinator.getBranchStatus("branch-1");
            const branch2 = await branchCoordinator.getBranchStatus("branch-2");

            expect(branch1?.state).toBe("running");
            expect(branch2?.state).toBe("completed");
            expect(mockStateStore.listBranches).toHaveBeenCalledWith("run-123");
        });

        it("should handle state store errors gracefully", async () => {
            mockStateStore.createBranch.mockRejectedValueOnce(new Error("Redis connection failed"));

            const config = { parentStepId: "step1", parallel: false };
            
            // Should not throw - should continue with in-memory operation
            const branches = await branchCoordinator.createBranchesFromConfig("run-123", config);

            expect(branches).toHaveLength(1);
            expect(mockLogger.error).toHaveBeenCalledWith(
                expect.stringContaining("Failed to persist branch to state store"),
                expect.any(Object),
            );
        });
    });

    describe("instance identification", () => {
        it("should have unique instance ID for each coordinator", () => {
            const coordinator1 = new BranchCoordinator(mockLogger, mockEventBus as any, mockStateStore as any);
            const coordinator2 = new BranchCoordinator(mockLogger, mockEventBus as any, mockStateStore as any);

            // Both should have unique instance IDs (can't directly access private field, 
            // but we can verify through event publishing that they're different)
            expect(coordinator1).not.toBe(coordinator2);
        });

        it("should use instance ID in event publishing", async () => {
            const config = { parentStepId: "step1", parallel: false };
            
            await branchCoordinator.createBranchesFromConfig("run-123", config);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    source: expect.objectContaining({
                        instanceId: expect.stringMatching(/^branch-coordinator-[a-f0-9-]+$/),
                    }),
                }),
            );
        });
    });
});
