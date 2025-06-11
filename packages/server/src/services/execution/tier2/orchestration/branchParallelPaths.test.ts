import { describe, it, expect, vi, beforeEach } from "vitest";
import { BranchCoordinator } from "./branchCoordinator.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type StepExecutor } from "./stepExecutor.js";
import { type Logger } from "winston";

describe("BranchCoordinator - Parallel Path Execution", () => {
    let coordinator: BranchCoordinator;
    let mockEventBus: EventBus;
    let mockLogger: Logger;
    let mockStepExecutor: StepExecutor;

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
            executeStep: vi.fn(),
        } as unknown as StepExecutor;

        coordinator = new BranchCoordinator(mockEventBus, mockLogger);
    });

    it("should execute different paths for parallel branches", async () => {
        // Setup test data
        const run = {
            id: "run-123",
            routineId: "routine-456",
            context: {
                variables: {},
                blackboard: {},
                scopes: [{ id: "global", name: "Global", variables: {} }],
            },
            config: { recoveryStrategy: "fail" as const },
        };

        // Mock navigator that returns multiple parallel paths
        const mockNavigator = {
            type: "test",
            version: "1.0",
            getParallelBranches: vi.fn().mockReturnValue([
                // Path 1
                [
                    { id: "p1-1", routineId: run.routineId, nodeId: "path1-step1" },
                    { id: "p1-2", routineId: run.routineId, nodeId: "path1-step2" },
                ],
                // Path 2
                [
                    { id: "p2-1", routineId: run.routineId, nodeId: "path2-step1" },
                    { id: "p2-2", routineId: run.routineId, nodeId: "path2-step2" },
                ],
                // Path 3
                [
                    { id: "p3-1", routineId: run.routineId, nodeId: "path3-step1" },
                    { id: "p3-2", routineId: run.routineId, nodeId: "path3-step2" },
                ],
            ]),
            getStepInfo: vi.fn().mockImplementation((location) => ({
                id: location.nodeId,
                name: `Step ${location.nodeId}`,
                type: "action",
            })),
        };

        // Track which steps were executed
        const executedSteps: string[] = [];
        mockStepExecutor.executeStep = vi.fn().mockImplementation(async (params) => {
            executedSteps.push(params.stepId);
            return {
                success: true,
                outputs: { result: `${params.stepId}-output` },
                duration: 100,
            };
        });

        // Create 3 parallel branches
        const locations = [
            { id: "loc1", routineId: run.routineId, nodeId: "parallel-node" },
            { id: "loc2", routineId: run.routineId, nodeId: "parallel-node" },
            { id: "loc3", routineId: run.routineId, nodeId: "parallel-node" },
        ];
        const branches = await coordinator.createBranches(run.id, locations, true);

        // Verify branch indices were assigned
        expect(branches[0].branchIndex).toBe(0);
        expect(branches[1].branchIndex).toBe(1);
        expect(branches[2].branchIndex).toBe(2);

        // Execute branches
        const results = await coordinator.executeBranches(
            run,
            branches,
            mockNavigator,
            mockStepExecutor,
        );

        // Verify all branches completed successfully
        expect(results).toHaveLength(3);
        results.forEach(result => {
            expect(result.success).toBe(true);
            expect(result.completedSteps).toBe(2); // Each path has 2 steps
        });

        // Verify each branch executed its own path
        expect(executedSteps).toContain("path1-step1");
        expect(executedSteps).toContain("path1-step2");
        expect(executedSteps).toContain("path2-step1");
        expect(executedSteps).toContain("path2-step2");
        expect(executedSteps).toContain("path3-step1");
        expect(executedSteps).toContain("path3-step2");

        // Verify getParallelBranches was called
        expect(mockNavigator.getParallelBranches).toHaveBeenCalled();
    });

    it("should handle branch index out of bounds gracefully", async () => {
        const run = {
            id: "run-123",
            routineId: "routine-456",
            context: {
                variables: {},
                blackboard: {},
                scopes: [{ id: "global", name: "Global", variables: {} }],
            },
            config: { recoveryStrategy: "fail" as const },
        };

        // Mock navigator with only 2 paths
        const mockNavigator = {
            type: "test",
            version: "1.0",
            getParallelBranches: vi.fn().mockReturnValue([
                [{ id: "p1", routineId: run.routineId, nodeId: "path1-step1" }],
                [{ id: "p2", routineId: run.routineId, nodeId: "path2-step1" }],
            ]),
            getStepInfo: vi.fn().mockImplementation((location) => ({
                id: location.nodeId,
                name: `Step ${location.nodeId}`,
                type: "action",
            })),
        };

        mockStepExecutor.executeStep = vi.fn().mockResolvedValue({
            success: true,
            outputs: {},
            duration: 100,
        });

        // Create 3 branches but navigator only has 2 paths
        const locations = [
            { id: "loc1", routineId: run.routineId, nodeId: "parallel-node" },
            { id: "loc2", routineId: run.routineId, nodeId: "parallel-node" },
            { id: "loc3", routineId: run.routineId, nodeId: "parallel-node" },
        ];
        const branches = await coordinator.createBranches(run.id, locations, true);

        // Execute branches
        await coordinator.executeBranches(run, branches, mockNavigator, mockStepExecutor);

        // Verify fallback warning was logged for out-of-bounds index
        expect(mockLogger.warn).toHaveBeenCalledWith(
            "[BranchCoordinator] Using fallback parallel path",
            expect.objectContaining({
                requestedIndex: 2,
                fallbackIndex: 1,
            }),
        );

        // Verify third branch fell back to second path (last available)
        const calls = (mockStepExecutor.executeStep as any).mock.calls;
        const thirdBranchCall = calls.find((call: any[]) => 
            call[0].context.scopes.some((s: any) => s.id === `branch-${branches[2].id}`),
        );
        expect(thirdBranchCall).toBeDefined();
        expect(thirdBranchCall[0].stepId).toBe("path2-step1"); // Fell back to last available path
    });

    it("should handle sequential branches without branch indices", async () => {
        const run = {
            id: "run-123",
            routineId: "routine-456",
            context: {
                variables: {},
                blackboard: {},
                scopes: [{ id: "global", name: "Global", variables: {} }],
            },
            config: { recoveryStrategy: "fail" as const },
        };

        const mockNavigator = {
            type: "test",
            version: "1.0",
            getStepInfo: vi.fn().mockReturnValue({
                id: "test",
                name: "Test Step",
                type: "action",
            }),
        };

        mockStepExecutor.executeStep = vi.fn().mockResolvedValue({
            success: true,
            outputs: {},
            duration: 100,
        });

        // Create sequential branches using the new API (old API now works differently)
        const branches1 = await coordinator.createSequentialBranch(run.id, "node1");
        const branches2 = await coordinator.createSequentialBranch(run.id, "node2");

        // Verify no branch indices for sequential branches
        expect(branches1[0].branchIndex).toBeUndefined();
        expect(branches2[0].branchIndex).toBeUndefined();
        
        // Combine for execution test
        const branches = [...branches1, ...branches2];

        // Execute should still work
        const results = await coordinator.executeBranches(
            run,
            branches,
            mockNavigator,
            mockStepExecutor,
        );

        expect(results).toHaveLength(2);
        results.forEach(result => {
            expect(result.success).toBe(true);
        });
    });

    it("should log debug info when selecting parallel paths", async () => {
        const run = {
            id: "run-123",
            routineId: "routine-456",
            context: {
                variables: {},
                blackboard: {},
                scopes: [{ id: "global", name: "Global", variables: {} }],
            },
            config: { recoveryStrategy: "fail" as const },
        };

        const mockNavigator = {
            type: "test",
            version: "1.0",
            getParallelBranches: vi.fn().mockReturnValue([
                [{ id: "p1", routineId: run.routineId, nodeId: "path1-step1" }],
                [{ id: "p2", routineId: run.routineId, nodeId: "path2-step1" }],
            ]),
            getStepInfo: vi.fn().mockReturnValue({
                id: "test",
                name: "Test Step",
                type: "action",
            }),
        };

        mockStepExecutor.executeStep = vi.fn().mockResolvedValue({
            success: true,
            outputs: {},
            duration: 100,
        });

        const branches = await coordinator.createBranches(
            run.id,
            [{ id: "loc1", routineId: run.routineId, nodeId: "node" }],
            true,
        );

        await coordinator.executeBranches(run, branches, mockNavigator, mockStepExecutor);

        // Verify debug logging
        expect(mockLogger.debug).toHaveBeenCalledWith(
            "[BranchCoordinator] Selected parallel path",
            expect.objectContaining({
                branchIndex: 0,
                pathLength: 1,
                totalPaths: 2,
            }),
        );
    });
});