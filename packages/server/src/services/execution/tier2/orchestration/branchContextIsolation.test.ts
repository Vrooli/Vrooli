import { describe, it, expect, vi, beforeEach } from "vitest";
import { BranchCoordinator } from "./branchCoordinator.js";
import { type EventBus } from "../../cross-cutting/events/eventBus.js";
import { type StepExecutor, type StepExecutionResult } from "./stepExecutor.js";
import { type Logger } from "winston";

describe("BranchCoordinator - Context Isolation", () => {
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
            executeStep: vi.fn().mockResolvedValue({
                success: true,
                outputs: { result: "test" },
                duration: 100,
            }),
        } as unknown as StepExecutor;

        coordinator = new BranchCoordinator(mockEventBus, mockLogger);
    });

    describe("Parallel Branch Isolation", () => {
        it("should isolate blackboard changes between parallel branches", async () => {
            // Setup test data
            const run = {
                id: "run-123",
                routineId: "routine-456",
                context: {
                    variables: { global: "value" },
                    blackboard: { 
                        shared: "initial",
                        counter: 0,
                        nested: { value: "test" },
                    },
                    scopes: [{ id: "global", name: "Global", variables: {} }],
                },
                config: { recoveryStrategy: "fail" as const },
            };

            // Create parallel branches
            const config = {
                parentStepId: "node1",
                parallel: true,
                branchCount: 2,
            };
            const branches = await coordinator.createBranchesFromConfig(run.id, config);

            // Track which execution this is
            let executionCount = 0;

            // Mock step executor to modify blackboard
            mockStepExecutor.executeStep = vi.fn().mockImplementation(async (params) => {
                const currentExecution = executionCount++;
                
                // Each branch execution modifies the blackboard differently
                params.context.blackboard.shared = `modified-by-branch${currentExecution + 1}`;
                params.context.blackboard.counter = currentExecution + 1;
                params.context.blackboard.nested.value = `branch${currentExecution + 1}`;
                
                return {
                    success: true,
                    outputs: { 
                        branchId: `branch${currentExecution + 1}`,
                        blackboardState: JSON.parse(JSON.stringify(params.context.blackboard)),
                    },
                    duration: 100,
                };
            });

            const mockNavigator = {
                type: "test",
                version: "1.0",
                getStepInfo: vi.fn().mockReturnValue({
                    id: "test",
                    name: "Test Step",
                    type: "action",
                }),
            };

            // Execute branches
            const results = await coordinator.executeBranches(
                run,
                branches,
                mockNavigator,
                mockStepExecutor,
            );

            // Verify results
            expect(results).toHaveLength(2);
            expect(results[0].success).toBe(true);
            expect(results[1].success).toBe(true);

            // Check that each branch had its own isolated blackboard
            const branch1Blackboard = results[0].outputs.blackboardState as Record<string, any>;
            const branch2Blackboard = results[1].outputs.blackboardState as Record<string, any>;

            // Branch 1 should have its modifications
            expect(branch1Blackboard.shared).toBe("modified-by-branch1");
            expect(branch1Blackboard.counter).toBe(1);
            expect(branch1Blackboard.nested.value).toBe("branch1");

            // Branch 2 should have its modifications
            expect(branch2Blackboard.shared).toBe("modified-by-branch2");
            expect(branch2Blackboard.counter).toBe(2);
            expect(branch2Blackboard.nested.value).toBe("branch2");

            // Original context should remain unchanged
            expect(run.context.blackboard.shared).toBe("initial");
            expect(run.context.blackboard.counter).toBe(0);
            expect(run.context.blackboard.nested.value).toBe("test");
        });

        it("should handle race conditions in parallel branches", async () => {
            // Setup test data with shared counter
            const run = {
                id: "run-123",
                routineId: "routine-456",
                context: {
                    variables: {},
                    blackboard: { 
                        operations: [] as string[],
                        timestamp: Date.now(),
                    },
                    scopes: [{ id: "global", name: "Global", variables: {} }],
                },
                config: { recoveryStrategy: "fail" as const },
            };

            // Create multiple parallel branches
            const branchCount = 5;
            const config = {
                parentStepId: "node0",
                parallel: true,
                branchCount,
            };

            const branches = await coordinator.createBranchesFromConfig(run.id, config);

            // Mock step executor to simulate concurrent operations
            mockStepExecutor.executeStep = vi.fn().mockImplementation(async (params) => {
                // Simulate concurrent modification
                const operations = params.context.blackboard.operations as string[];
                operations.push(`branch-${params.stepId}`);
                
                // Simulate some async work
                await new Promise(resolve => setTimeout(resolve, Math.random() * 10));
                
                return {
                    success: true,
                    outputs: { 
                        operationCount: operations.length,
                    },
                    duration: 100,
                };
            });

            const mockNavigator = {
                type: "test",
                version: "1.0",
                getStepInfo: vi.fn().mockReturnValue({
                    id: "test",
                    name: "Test Step",
                    type: "action",
                }),
            };

            // Execute branches
            const results = await coordinator.executeBranches(
                run,
                branches,
                mockNavigator,
                mockStepExecutor,
            );

            // Each branch should see only its own operation
            results.forEach((result, index) => {
                expect(result.outputs.operationCount).toBe(1);
            });

            // Original blackboard should be unchanged
            expect(run.context.blackboard.operations).toHaveLength(0);
        });
    });

    describe("Sequential Branch Isolation", () => {
        it("should isolate context in sequential branch execution", async () => {
            const run = {
                id: "run-123",
                routineId: "routine-456",
                context: {
                    variables: {},
                    blackboard: { 
                        value: "initial",
                        history: [] as string[],
                    },
                    scopes: [{ id: "global", name: "Global", variables: {} }],
                },
                config: { recoveryStrategy: "fail" as const },
            };

            // Create two separate sequential branches to test isolation
            const branch1 = await coordinator.createSequentialBranch(run.id, "node1");
            const branch2 = await coordinator.createSequentialBranch(run.id, "node2");

            // Track executions
            let executionCount = 0;

            // Mock step executor
            mockStepExecutor.executeStep = vi.fn().mockImplementation(async (params) => {
                const currentExecution = executionCount++;
                const history = params.context.blackboard.history as string[];
                history.push(`execution-${currentExecution + 1}`);
                params.context.blackboard.value = `modified-by-execution-${currentExecution + 1}`;
                
                return {
                    success: true,
                    outputs: { 
                        blackboardSnapshot: JSON.parse(JSON.stringify(params.context.blackboard)),
                    },
                    duration: 100,
                };
            });

            const mockNavigator = {
                type: "test",
                version: "1.0",
                getStepInfo: vi.fn().mockReturnValue({
                    id: "test",
                    name: "Test Step",
                    type: "action",
                }),
            };

            // Execute branches separately (sequential execution)
            const result1 = await coordinator.executeBranches(run, branch1, mockNavigator, mockStepExecutor);
            const result2 = await coordinator.executeBranches(run, branch2, mockNavigator, mockStepExecutor);

            // Each branch should have isolated history
            const branch1Snapshot = result1[0].outputs.blackboardSnapshot as any;
            const branch2Snapshot = result2[0].outputs.blackboardSnapshot as any;

            expect(branch1Snapshot.history).toEqual(["execution-1"]);
            expect(branch2Snapshot.history).toEqual(["execution-2"]);

            // Original context unchanged
            expect(run.context.blackboard.history).toHaveLength(0);
            expect(run.context.blackboard.value).toBe("initial");
        });
    });

    describe("Deep Object Isolation", () => {
        it("should properly deep clone nested objects and arrays", async () => {
            const run = {
                id: "run-123",
                routineId: "routine-456",
                context: {
                    variables: {},
                    blackboard: { 
                        nested: {
                            level1: {
                                level2: {
                                    array: [1, 2, { value: 3 }],
                                    value: "deep",
                                },
                            },
                        },
                        dates: {
                            created: new Date("2024-01-01"),
                        },
                    },
                    scopes: [{ id: "global", name: "Global", variables: {} }],
                },
                config: { recoveryStrategy: "fail" as const },
            };

            const config = {
                parentStepId: "node1",
                parallel: false,
            };
            const branches = await coordinator.createBranchesFromConfig(run.id, config);

            // Mock step executor to modify deep nested values
            mockStepExecutor.executeStep = vi.fn().mockImplementation(async (params) => {
                // Modify nested values
                params.context.blackboard.nested.level1.level2.value = "modified";
                params.context.blackboard.nested.level1.level2.array.push(4);
                (params.context.blackboard.nested.level1.level2.array[2] as any).value = 99;
                
                return {
                    success: true,
                    outputs: { 
                        modifiedBlackboard: params.context.blackboard,
                    },
                    duration: 100,
                };
            });

            const mockNavigator = {
                type: "test",
                version: "1.0",
                getStepInfo: vi.fn().mockReturnValue({
                    id: "test",
                    name: "Test Step",
                    type: "action",
                }),
            };

            // Execute branch
            await coordinator.executeBranches(
                run,
                branches,
                mockNavigator,
                mockStepExecutor,
            );

            // Original nested values should be unchanged
            expect(run.context.blackboard.nested.level1.level2.value).toBe("deep");
            expect(run.context.blackboard.nested.level1.level2.array).toHaveLength(3);
            expect((run.context.blackboard.nested.level1.level2.array[2] as any).value).toBe(3);
        });
    });

    describe("Output Propagation", () => {
        it("should only share outputs when subroutine completes", async () => {
            const run = {
                id: "run-123",
                routineId: "routine-456",
                context: {
                    variables: {},
                    blackboard: { 
                        sharedData: {},
                    },
                    scopes: [{ id: "global", name: "Global", variables: {} }],
                },
                config: { recoveryStrategy: "fail" as const },
            };

            // Create branches
            const config = {
                parentStepId: "node1", 
                parallel: true,
                branchCount: 2,
            };
            const branches = await coordinator.createBranchesFromConfig(run.id, config);

            // Track branch executions
            let executionCount = 0;

            // Mock step executor
            mockStepExecutor.executeStep = vi.fn().mockImplementation(async (params) => {
                const branchNum = executionCount++;
                // Try to share data through blackboard (should not work)
                const stepId = params.location?.nodeId || params.stepId;
                params.context.blackboard.sharedData[`branch${branchNum}`] = "data";
                
                return {
                    success: true,
                    outputs: { 
                        [`result_branch${branchNum}`]: `value${branchNum}`,
                        commonOutput: `branch${branchNum}_result`,
                    },
                    duration: 100,
                };
            });

            const mockNavigator = {
                type: "test",
                version: "1.0",
                getStepInfo: vi.fn().mockReturnValue({
                    id: "test",
                    name: "Test Step",
                    type: "action",
                }),
            };

            // Execute branches
            const results = await coordinator.executeBranches(
                run,
                branches,
                mockNavigator,
                mockStepExecutor,
            );

            // Merge results (simulating subroutine completion)
            const mergedContext = await coordinator.mergeBranchResults(
                run.context,
                results,
            );

            // Original blackboard should be unchanged
            expect(Object.keys(run.context.blackboard.sharedData)).toHaveLength(0);

            // Check merged outputs
            expect(mergedContext.variables.result_branch0).toBe("value0");
            expect(mergedContext.variables.result_branch1).toBe("value1");
            
            // Common outputs from both branches should be merged into an array
            expect(mergedContext.variables.commonOutput_merged).toEqual([
                "branch0_result",
                "branch1_result",
            ]);

            // Merged context should have its own isolated blackboard
            expect(mergedContext.blackboard).not.toBe(run.context.blackboard);
        });
    });
});