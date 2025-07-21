import { beforeEach, describe, expect, test } from "vitest";
import { BpmnParallelHandler } from "./BpmnParallelHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation, ParallelBranch } from "../../types.js";

describe("BpmnParallelHandler", () => {
    let handler: BpmnParallelHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const parallelGatewayXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1">
            <bpmn:startEvent id="Start">
                <bpmn:outgoing>ToSplit</bpmn:outgoing>
            </bpmn:startEvent>
            <bpmn:parallelGateway id="SplitGateway">
                <bpmn:incoming>ToSplit</bpmn:incoming>
                <bpmn:outgoing>ToBranch1</bpmn:outgoing>
                <bpmn:outgoing>ToBranch2</bpmn:outgoing>
                <bpmn:outgoing>ToBranch3</bpmn:outgoing>
            </bpmn:parallelGateway>
            <bpmn:task id="Task1">
                <bpmn:incoming>ToBranch1</bpmn:incoming>
                <bpmn:outgoing>FromTask1</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="Task2">
                <bpmn:incoming>ToBranch2</bpmn:incoming>
                <bpmn:outgoing>FromTask2</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="Task3">
                <bpmn:incoming>ToBranch3</bpmn:incoming>
                <bpmn:outgoing>FromTask3</bpmn:outgoing>
            </bpmn:task>
            <bpmn:parallelGateway id="JoinGateway">
                <bpmn:incoming>FromTask1</bpmn:incoming>
                <bpmn:incoming>FromTask2</bpmn:incoming>
                <bpmn:incoming>FromTask3</bpmn:incoming>
                <bpmn:outgoing>ToEnd</bpmn:outgoing>
            </bpmn:parallelGateway>
            <bpmn:endEvent id="End">
                <bpmn:incoming>ToEnd</bpmn:incoming>
            </bpmn:endEvent>
            <bpmn:sequenceFlow id="ToSplit" sourceRef="Start" targetRef="SplitGateway" />
            <bpmn:sequenceFlow id="ToBranch1" sourceRef="SplitGateway" targetRef="Task1" />
            <bpmn:sequenceFlow id="ToBranch2" sourceRef="SplitGateway" targetRef="Task2" />
            <bpmn:sequenceFlow id="ToBranch3" sourceRef="SplitGateway" targetRef="Task3" />
            <bpmn:sequenceFlow id="FromTask1" sourceRef="Task1" targetRef="JoinGateway" />
            <bpmn:sequenceFlow id="FromTask2" sourceRef="Task2" targetRef="JoinGateway" />
            <bpmn:sequenceFlow id="FromTask3" sourceRef="Task3" targetRef="JoinGateway" />
            <bpmn:sequenceFlow id="ToEnd" sourceRef="JoinGateway" targetRef="End" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnParallelHandler();
        model = new BpmnModel();
        await model.loadXml(parallelGatewayXml);

        mockLocation = {
            nodeId: "SplitGateway",
            routineId: "routine_1",
            type: "node",
            metadata: {},
        };

        mockContext = {
            variables: {},
            events: {
                active: [],
                pending: [],
                fired: [],
                timers: [],
            },
            parallelExecution: {
                activeBranches: [],
                completedBranches: [],
                joinPoints: [],
            },
            subprocesses: {
                activeSubprocesses: [],
                completedSubprocesses: [],
                eventSubprocesses: [],
            },
            gateways: {
                exclusiveStates: [],
                inclusiveStates: [],
            },
            compensation: {
                completedActivities: [],
                compensationHandlers: [],
            },
            external: {
                messageEvents: [],
                signalEvents: [],
            },
            loopIterations: {},
            multiInstanceStates: {},
        };
    });

    describe("processParallelGateway", () => {
        describe("split gateway", () => {
            test("should process parallel split with multiple outgoing flows", () => {
                const result = handler.processParallelGateway(
                    model, 
                    "SplitGateway", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(3);
                expect(result.nextLocations.map(loc => loc.nodeId)).toContain("Task1");
                expect(result.nextLocations.map(loc => loc.nodeId)).toContain("Task2");
                expect(result.nextLocations.map(loc => loc.nodeId)).toContain("Task3");
            });

            test("should create active branches for split", () => {
                const result = handler.processParallelGateway(
                    model, 
                    "SplitGateway", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.parallelExecution.activeBranches).toHaveLength(3);
                expect(result.updatedContext.parallelExecution.activeBranches[0].sourceGatewayId).toBe("SplitGateway");
                expect(result.branchingInfo?.activatedBranches).toBe(3);
            });

            test("should create join point for corresponding join gateway", () => {
                const result = handler.processParallelGateway(
                    model, 
                    "SplitGateway", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.parallelExecution.joinPoints).toHaveLength(1);
                expect(result.updatedContext.parallelExecution.joinPoints[0].gatewayId).toBe("JoinGateway");
                expect(result.updatedContext.parallelExecution.joinPoints[0].expectedBranches).toHaveLength(3);
            });

            test("should include metadata in branch locations", () => {
                const result = handler.processParallelGateway(
                    model, 
                    "SplitGateway", 
                    mockLocation, 
                    mockContext,
                );

                result.nextLocations.forEach((loc, index) => {
                    expect(loc.metadata.parallelGateway).toBe("SplitGateway");
                    expect(loc.metadata.branchIndex).toBe(index);
                    expect(loc.metadata.totalBranches).toBe(3);
                });
            });
        });

        describe("join gateway", () => {
            beforeEach(() => {
                // Set up context with active branches and join point
                const splitId = "split_SplitGateway_123";
                mockContext.parallelExecution = {
                    activeBranches: [
                        {
                            id: `${splitId}_0`,
                            sourceGatewayId: "SplitGateway",
                            targetNodeId: "Task1",
                            status: "active",
                        },
                        {
                            id: `${splitId}_1`,
                            sourceGatewayId: "SplitGateway",
                            targetNodeId: "Task2",
                            status: "active",
                        },
                        {
                            id: `${splitId}_2`,
                            sourceGatewayId: "SplitGateway",
                            targetNodeId: "Task3",
                            status: "active",
                        },
                    ],
                    completedBranches: [],
                    joinPoints: [{
                        gatewayId: "JoinGateway",
                        expectedBranches: [`${splitId}_0`, `${splitId}_1`, `${splitId}_2`],
                        arrivedBranches: [],
                        sourceGatewayId: "SplitGateway",
                    }],
                };

                mockLocation = {
                    nodeId: "JoinGateway",
                    routineId: "routine_1",
                    type: "node",
                    metadata: {
                        branchId: `${splitId}_0`,
                    },
                };
            });

            test("should wait when not all branches have arrived", () => {
                const result = handler.processParallelGateway(
                    model, 
                    "JoinGateway", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].type).toBe("parallel_join_waiting");
                expect(result.branchingInfo?.synchronizationComplete).toBe(false);
            });

            test("should record arrived branch", () => {
                const result = handler.processParallelGateway(
                    model, 
                    "JoinGateway", 
                    mockLocation, 
                    mockContext,
                );

                const joinPoint = result.updatedContext.parallelExecution.joinPoints[0];
                expect(joinPoint.arrivedBranches).toContain("split_SplitGateway_123_0");
            });

            test("should proceed when all branches have arrived", () => {
                // Mark two branches as already arrived
                mockContext.parallelExecution.joinPoints[0].arrivedBranches = [
                    "split_SplitGateway_123_1",
                    "split_SplitGateway_123_2",
                ];

                const result = handler.processParallelGateway(
                    model, 
                    "JoinGateway", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("End");
                expect(result.branchingInfo?.synchronizationComplete).toBe(true);
            });

            test("should clean up completed branches and join points", () => {
                // All branches arrived
                mockContext.parallelExecution.joinPoints[0].arrivedBranches = [
                    "split_SplitGateway_123_1",
                    "split_SplitGateway_123_2",
                ];

                const result = handler.processParallelGateway(
                    model, 
                    "JoinGateway", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.parallelExecution.activeBranches).toHaveLength(0);
                expect(result.updatedContext.parallelExecution.completedBranches).toContain("split_SplitGateway_123_0");
                expect(result.updatedContext.parallelExecution.joinPoints).toHaveLength(0);
            });
        });

        describe("pass-through gateway", () => {
            test("should handle gateway with single flow", async () => {
                const simpleXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:parallelGateway id="PassThrough">
                            <bpmn:incoming>In</bpmn:incoming>
                            <bpmn:outgoing>Out</bpmn:outgoing>
                        </bpmn:parallelGateway>
                        <bpmn:task id="NextTask">
                            <bpmn:incoming>Out</bpmn:incoming>
                        </bpmn:task>
                        <bpmn:sequenceFlow id="Out" sourceRef="PassThrough" targetRef="NextTask" />
                    </bpmn:process>
                </bpmn:definitions>`;

                const simpleModel = new BpmnModel();
                await simpleModel.loadXml(simpleXml);

                const result = handler.processParallelGateway(
                    simpleModel, 
                    "PassThrough", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("NextTask");
                expect(result.updatedContext.parallelExecution.activeBranches).toHaveLength(0);
            });
        });
    });

    describe("findCorrespondingJoinGateway", () => {
        test("should find join gateway by following all paths", () => {
            const joinGateway = (handler as any).findCorrespondingJoinGateway(
                model, 
                "SplitGateway",
            );

            expect(joinGateway).toBe("JoinGateway");
        });

        test("should handle nested parallel gateways", async () => {
            const nestedXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:parallelGateway id="OuterSplit">
                        <bpmn:outgoing>ToInner</bpmn:outgoing>
                        <bpmn:outgoing>ToTask</bpmn:outgoing>
                    </bpmn:parallelGateway>
                    <bpmn:parallelGateway id="InnerSplit">
                        <bpmn:incoming>ToInner</bpmn:incoming>
                        <bpmn:outgoing>ToInner1</bpmn:outgoing>
                        <bpmn:outgoing>ToInner2</bpmn:outgoing>
                    </bpmn:parallelGateway>
                    <bpmn:task id="InnerTask1">
                        <bpmn:incoming>ToInner1</bpmn:incoming>
                        <bpmn:outgoing>FromInner1</bpmn:outgoing>
                    </bpmn:task>
                    <bpmn:task id="InnerTask2">
                        <bpmn:incoming>ToInner2</bpmn:incoming>
                        <bpmn:outgoing>FromInner2</bpmn:outgoing>
                    </bpmn:task>
                    <bpmn:parallelGateway id="InnerJoin">
                        <bpmn:incoming>FromInner1</bpmn:incoming>
                        <bpmn:incoming>FromInner2</bpmn:incoming>
                        <bpmn:outgoing>FromInner</bpmn:outgoing>
                    </bpmn:parallelGateway>
                    <bpmn:task id="OuterTask">
                        <bpmn:incoming>ToTask</bpmn:incoming>
                        <bpmn:outgoing>FromTask</bpmn:outgoing>
                    </bpmn:task>
                    <bpmn:parallelGateway id="OuterJoin">
                        <bpmn:incoming>FromInner</bpmn:incoming>
                        <bpmn:incoming>FromTask</bpmn:incoming>
                        <bpmn:outgoing>ToEnd</bpmn:outgoing>
                    </bpmn:parallelGateway>
                    <bpmn:sequenceFlow id="ToInner" sourceRef="OuterSplit" targetRef="InnerSplit" />
                    <bpmn:sequenceFlow id="ToTask" sourceRef="OuterSplit" targetRef="OuterTask" />
                    <bpmn:sequenceFlow id="ToInner1" sourceRef="InnerSplit" targetRef="InnerTask1" />
                    <bpmn:sequenceFlow id="ToInner2" sourceRef="InnerSplit" targetRef="InnerTask2" />
                    <bpmn:sequenceFlow id="FromInner1" sourceRef="InnerTask1" targetRef="InnerJoin" />
                    <bpmn:sequenceFlow id="FromInner2" sourceRef="InnerTask2" targetRef="InnerJoin" />
                    <bpmn:sequenceFlow id="FromInner" sourceRef="InnerJoin" targetRef="OuterJoin" />
                    <bpmn:sequenceFlow id="FromTask" sourceRef="OuterTask" targetRef="OuterJoin" />
                </bpmn:process>
            </bpmn:definitions>`;

            const nestedModel = new BpmnModel();
            await nestedModel.loadXml(nestedXml);

            const innerJoin = (handler as any).findCorrespondingJoinGateway(
                nestedModel, 
                "InnerSplit",
            );
            expect(innerJoin).toBe("InnerJoin");

            const outerJoin = (handler as any).findCorrespondingJoinGateway(
                nestedModel, 
                "OuterSplit",
            );
            expect(outerJoin).toBe("OuterJoin");
        });
    });

    describe("completeBranch", () => {
        test("should mark branch as completed", () => {
            const branch: ParallelBranch = {
                id: "branch_1",
                sourceGatewayId: "SplitGateway",
                targetNodeId: "Task1",
                status: "active",
            };
            mockContext.parallelExecution.activeBranches = [branch];

            const result = handler.completeBranch(mockContext, "branch_1");

            expect(result.parallelExecution.activeBranches).toHaveLength(0);
            expect(result.parallelExecution.completedBranches).toContain("branch_1");
        });

        test("should not affect other branches", () => {
            mockContext.parallelExecution.activeBranches = [
                {
                    id: "branch_1",
                    sourceGatewayId: "Gateway1",
                    targetNodeId: "Task1",
                    status: "active",
                },
                {
                    id: "branch_2",
                    sourceGatewayId: "Gateway1",
                    targetNodeId: "Task2",
                    status: "active",
                },
            ];

            const result = handler.completeBranch(mockContext, "branch_1");

            expect(result.parallelExecution.activeBranches).toHaveLength(1);
            expect(result.parallelExecution.activeBranches[0].id).toBe("branch_2");
            expect(result.parallelExecution.completedBranches).toContain("branch_1");
        });
    });

    describe("isAllBranchesComplete", () => {
        test("should return true when all branches complete", () => {
            const joinPoint = {
                gatewayId: "JoinGateway",
                expectedBranches: ["branch_1", "branch_2"],
                arrivedBranches: ["branch_1", "branch_2"],
                sourceGatewayId: "SplitGateway",
            };
            mockContext.parallelExecution.joinPoints = [joinPoint];

            const result = handler.isAllBranchesComplete(mockContext, "JoinGateway");
            expect(result).toBe(true);
        });

        test("should return false when branches are pending", () => {
            const joinPoint = {
                gatewayId: "JoinGateway",
                expectedBranches: ["branch_1", "branch_2", "branch_3"],
                arrivedBranches: ["branch_1"],
                sourceGatewayId: "SplitGateway",
            };
            mockContext.parallelExecution.joinPoints = [joinPoint];

            const result = handler.isAllBranchesComplete(mockContext, "JoinGateway");
            expect(result).toBe(false);
        });

        test("should return true when no join point exists", () => {
            const result = handler.isAllBranchesComplete(mockContext, "NonExistentGateway");
            expect(result).toBe(true);
        });
    });

    describe("getParallelExecutionSummary", () => {
        test("should provide correct summary", () => {
            mockContext.parallelExecution = {
                activeBranches: [
                    { id: "b1", sourceGatewayId: "G1", targetNodeId: "T1", status: "active" },
                    { id: "b2", sourceGatewayId: "G1", targetNodeId: "T2", status: "active" },
                    { id: "b3", sourceGatewayId: "G2", targetNodeId: "T3", status: "active" },
                ],
                completedBranches: ["b4", "b5"],
                joinPoints: [
                    {
                        gatewayId: "J1",
                        expectedBranches: ["b1", "b2"],
                        arrivedBranches: ["b1"],
                        sourceGatewayId: "G1",
                    },
                    {
                        gatewayId: "J2",
                        expectedBranches: ["b3"],
                        arrivedBranches: [],
                        sourceGatewayId: "G2",
                    },
                ],
            };

            const summary = handler.getParallelExecutionSummary(mockContext);

            expect(summary.activeBranches).toBe(3);
            expect(summary.completedBranches).toBe(2);
            expect(summary.pendingJoins).toBe(2);
            expect(summary.activeGateways).toBe(2);
        });
    });

    describe("error handling", () => {
        test("should throw error for non-existent gateway", () => {
            expect(() => {
                handler.processParallelGateway(
                    model, 
                    "NonExistentGateway", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow("Parallel gateway not found: NonExistentGateway");
        });
    });
});
