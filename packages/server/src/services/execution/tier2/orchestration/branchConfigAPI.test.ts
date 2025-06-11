import { describe, it, expect, vi, beforeEach } from "vitest";
import { BranchCoordinator } from "./branchCoordinator.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type StepExecutor } from "./stepExecutor.js";
import { type Logger } from "winston";

describe("BranchCoordinator - New Configuration API", () => {
    let coordinator: BranchCoordinator;
    let mockEventBus: EventBus;
    let mockLogger: Logger;
    let mockStepExecutor: StepExecutor;
    let mockNavigator: any;

    beforeEach(() => {
        // Create mocks
        mockEventBus = {
            publish: vi.fn().mockResolvedValue(undefined),
            subscribe: vi.fn(),
            unsubscribe: vi.fn(),
        } as unknown as EventBus;

        mockLogger = {
            info: vi.fn(),
            debug: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
        } as unknown as Logger;

        mockStepExecutor = {
            executeStep: vi.fn().mockResolvedValue({
                success: true,
                outputs: {},
                duration: 100,
            }),
        } as unknown as StepExecutor;

        mockNavigator = {
            type: "test",
            version: "1.0",
            getParallelBranches: vi.fn(),
            getStepInfo: vi.fn().mockReturnValue({
                id: "test",
                name: "Test Step",
                type: "action",
            }),
        };

        coordinator = new BranchCoordinator(mockEventBus, mockLogger);
    });

    describe("createBranchesFromConfig", () => {
        it("should create parallel branches using navigator", async () => {
            // Setup navigator to return 3 parallel paths
            mockNavigator.getParallelBranches.mockReturnValue([
                [{ id: "p1", routineId: "r1", nodeId: "path1-step1" }],
                [{ id: "p2", routineId: "r1", nodeId: "path2-step1" }],
                [{ id: "p3", routineId: "r1", nodeId: "path3-step1" }],
            ]);

            const config = {
                parentStepId: "parallel-step",
                parallel: true,
            };

            const branches = await coordinator.createBranchesFromConfig(
                "run-123",
                config,
                mockNavigator,
            );

            // Verify correct number of branches created
            expect(branches).toHaveLength(3);
            
            // Verify each branch has correct properties
            branches.forEach((branch, index) => {
                expect(branch.parentStepId).toBe("parallel-step");
                expect(branch.parallel).toBe(true);
                expect(branch.branchIndex).toBe(index);
                expect(branch.state).toBe("pending");
            });

            // Verify navigator was called
            expect(mockNavigator.getParallelBranches).toHaveBeenCalledWith({
                id: "parallel-step",
                routineId: "run-123",
                nodeId: "parallel-step",
            });

            // Verify debug logging
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[BranchCoordinator] Derived branch count from navigator",
                expect.objectContaining({
                    branchCount: 3,
                    parentStepId: "parallel-step",
                }),
            );
        });

        it("should create branches with explicit branch count", async () => {
            const config = {
                parentStepId: "parallel-step",
                parallel: true,
                branchCount: 5,
            };

            const branches = await coordinator.createBranchesFromConfig("run-123", config);

            expect(branches).toHaveLength(5);
            branches.forEach((branch, index) => {
                expect(branch.branchIndex).toBe(index);
                expect(branch.parallel).toBe(true);
            });

            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[BranchCoordinator] Using explicit branch count",
                expect.objectContaining({
                    branchCount: 5,
                }),
            );
        });

        it("should create branches with predefined paths", async () => {
            const predefinedPaths = [
                [
                    { id: "p1-1", routineId: "r1", nodeId: "path1-step1" },
                    { id: "p1-2", routineId: "r1", nodeId: "path1-step2" },
                ],
                [
                    { id: "p2-1", routineId: "r1", nodeId: "path2-step1" },
                ],
            ];

            const config = {
                parentStepId: "parallel-step",
                parallel: true,
                predefinedPaths,
            };

            const branches = await coordinator.createBranchesFromConfig("run-123", config);

            expect(branches).toHaveLength(2);
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[BranchCoordinator] Using predefined paths",
                expect.objectContaining({
                    pathCount: 2,
                }),
            );
        });

        it("should create single sequential branch", async () => {
            const config = {
                parentStepId: "sequential-step",
                parallel: false,
            };

            const branches = await coordinator.createBranchesFromConfig("run-123", config);

            expect(branches).toHaveLength(1);
            expect(branches[0].parallel).toBe(false);
            expect(branches[0].branchIndex).toBeUndefined();
        });

        it("should handle navigator with no parallel paths", async () => {
            mockNavigator.getParallelBranches.mockReturnValue([]);

            const config = {
                parentStepId: "parallel-step",
                parallel: true,
            };

            const branches = await coordinator.createBranchesFromConfig(
                "run-123",
                config,
                mockNavigator,
            );

            // Should fall back to 1 branch
            expect(branches).toHaveLength(1);
            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[BranchCoordinator] No parallel paths found",
                expect.objectContaining({
                    parentStepId: "parallel-step",
                }),
            );
        });

        it("should throw error for parallel branches without valid configuration", async () => {
            const config = {
                parentStepId: "parallel-step",
                parallel: true,
                // No branchCount, predefinedPaths, or navigator
            };

            await expect(
                coordinator.createBranchesFromConfig("run-123", config),
            ).rejects.toThrow("Parallel branches require branchCount, predefinedPaths, or navigator");
        });

        it("should emit events with enhanced data", async () => {
            const config = {
                parentStepId: "parallel-step",
                parallel: true,
                branchCount: 2,
            };

            await coordinator.createBranchesFromConfig("run-123", config);

            // Check that events were published with enhanced data
            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    data: expect.objectContaining({
                        runId: "run-123",
                        parentStepId: "parallel-step",
                        branchIndex: 0,
                        parallel: true,
                        totalBranches: 2,
                    }),
                }),
            );

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    data: expect.objectContaining({
                        branchIndex: 1,
                        totalBranches: 2,
                    }),
                }),
            );
        });
    });

    describe("Convenience Methods", () => {
        it("createParallelBranches should use navigator", async () => {
            mockNavigator.getParallelBranches.mockReturnValue([
                [{ id: "p1", routineId: "r1", nodeId: "path1" }],
                [{ id: "p2", routineId: "r1", nodeId: "path2" }],
            ]);

            const branches = await coordinator.createParallelBranches(
                "run-123",
                "parent-step",
                mockNavigator,
            );

            expect(branches).toHaveLength(2);
            expect(branches.every(b => b.parallel)).toBe(true);
        });

        it("createSequentialBranch should create single sequential branch", async () => {
            const branches = await coordinator.createSequentialBranch("run-123", "parent-step");

            expect(branches).toHaveLength(1);
            expect(branches[0].parallel).toBe(false);
            expect(branches[0].branchIndex).toBeUndefined();
        });

        it("createBranchesWithPredefinedPaths should use provided paths", async () => {
            const paths = [
                [{ id: "p1", routineId: "r1", nodeId: "path1" }],
                [{ id: "p2", routineId: "r1", nodeId: "path2" }],
                [{ id: "p3", routineId: "r1", nodeId: "path3" }],
            ];

            const branches = await coordinator.createBranchesWithPredefinedPaths(
                "run-123",
                "parent-step",
                paths,
            );

            expect(branches).toHaveLength(3);
            expect(branches.every(b => b.parallel)).toBe(true);
        });

        it("createParallelBranchesWithCount should create exact number", async () => {
            const branches = await coordinator.createParallelBranchesWithCount(
                "run-123",
                "parent-step",
                4,
            );

            expect(branches).toHaveLength(4);
            expect(branches.every(b => b.parallel)).toBe(true);
            branches.forEach((branch, index) => {
                expect(branch.branchIndex).toBe(index);
            });
        });
    });

    describe("Backward Compatibility", () => {
        it("should log deprecation warning for old createBranches API", async () => {
            const locations = [
                { id: "loc1", routineId: "r1", nodeId: "step1" },
                { id: "loc2", routineId: "r1", nodeId: "step2" },
            ];

            await coordinator.createBranches("run-123", locations, true);

            expect(mockLogger.warn).toHaveBeenCalledWith(
                "[BranchCoordinator] createBranches is deprecated, use createBranchesFromConfig instead",
                expect.objectContaining({
                    runId: "run-123",
                    locationCount: 2,
                    parallel: true,
                }),
            );
        });

        it("should convert old API to new API correctly", async () => {
            const locations = [
                { id: "loc1", routineId: "r1", nodeId: "step1" },
                { id: "loc2", routineId: "r1", nodeId: "step2" },
                { id: "loc3", routineId: "r1", nodeId: "step3" },
            ];

            const branches = await coordinator.createBranches("run-123", locations, true);

            // Should create 3 branches (length of locations array)
            expect(branches).toHaveLength(3);
            
            // All should be parallel with proper indices
            branches.forEach((branch, index) => {
                expect(branch.parallel).toBe(true);
                expect(branch.branchIndex).toBe(index);
                expect(branch.parentStepId).toBe("step1"); // First location's nodeId
            });
        });

        it("should handle empty locations array gracefully", async () => {
            const branches = await coordinator.createBranches("run-123", [], true);

            expect(branches).toHaveLength(0);
        });
    });

    describe("Error Handling", () => {
        it("should handle invalid configuration gracefully", async () => {
            const config = {
                parentStepId: "",
                parallel: true,
            };

            await expect(
                coordinator.createBranchesFromConfig("run-123", config),
            ).rejects.toThrow();
        });

        it("should handle navigator errors gracefully", async () => {
            mockNavigator.getParallelBranches.mockImplementation(() => {
                throw new Error("Navigator error");
            });

            const config = {
                parentStepId: "parallel-step",
                parallel: true,
            };

            await expect(
                coordinator.createBranchesFromConfig("run-123", config, mockNavigator),
            ).rejects.toThrow("Navigator error");
        });
    });
});