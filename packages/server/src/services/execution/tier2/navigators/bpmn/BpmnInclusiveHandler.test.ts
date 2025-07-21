import { beforeEach, describe, expect, test } from "vitest";
import { BpmnInclusiveHandler } from "./BpmnInclusiveHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation, InclusiveGatewayState } from "../../types.js";

describe("BpmnInclusiveHandler", () => {
    let handler: BpmnInclusiveHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const inclusiveGatewayXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1">
            <bpmn:startEvent id="Start">
                <bpmn:outgoing>ToSplit</bpmn:outgoing>
            </bpmn:startEvent>
            <bpmn:inclusiveGateway id="InclusiveSplit" default="DefaultFlow">
                <bpmn:incoming>ToSplit</bpmn:incoming>
                <bpmn:outgoing>PathA</bpmn:outgoing>
                <bpmn:outgoing>PathB</bpmn:outgoing>
                <bpmn:outgoing>PathC</bpmn:outgoing>
                <bpmn:outgoing>DefaultFlow</bpmn:outgoing>
            </bpmn:inclusiveGateway>
            <bpmn:task id="TaskA">
                <bpmn:incoming>PathA</bpmn:incoming>
                <bpmn:outgoing>FromTaskA</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="TaskB">
                <bpmn:incoming>PathB</bpmn:incoming>
                <bpmn:outgoing>FromTaskB</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="TaskC">
                <bpmn:incoming>PathC</bpmn:incoming>
                <bpmn:outgoing>FromTaskC</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="DefaultTask">
                <bpmn:incoming>DefaultFlow</bpmn:incoming>
                <bpmn:outgoing>FromDefault</bpmn:outgoing>
            </bpmn:task>
            <bpmn:inclusiveGateway id="InclusiveJoin">
                <bpmn:incoming>FromTaskA</bpmn:incoming>
                <bpmn:incoming>FromTaskB</bpmn:incoming>
                <bpmn:incoming>FromTaskC</bpmn:incoming>
                <bpmn:incoming>FromDefault</bpmn:incoming>
                <bpmn:outgoing>ToEnd</bpmn:outgoing>
            </bpmn:inclusiveGateway>
            <bpmn:endEvent id="End">
                <bpmn:incoming>ToEnd</bpmn:incoming>
            </bpmn:endEvent>
            <bpmn:sequenceFlow id="ToSplit" sourceRef="Start" targetRef="InclusiveSplit" />
            <bpmn:sequenceFlow id="PathA" sourceRef="InclusiveSplit" targetRef="TaskA">
                <bpmn:conditionExpression>conditionA == true</bpmn:conditionExpression>
            </bpmn:sequenceFlow>
            <bpmn:sequenceFlow id="PathB" sourceRef="InclusiveSplit" targetRef="TaskB">
                <bpmn:conditionExpression>conditionB == true</bpmn:conditionExpression>
            </bpmn:sequenceFlow>
            <bpmn:sequenceFlow id="PathC" sourceRef="InclusiveSplit" targetRef="TaskC">
                <bpmn:conditionExpression>conditionC == true</bpmn:conditionExpression>
            </bpmn:sequenceFlow>
            <bpmn:sequenceFlow id="DefaultFlow" sourceRef="InclusiveSplit" targetRef="DefaultTask" />
            <bpmn:sequenceFlow id="FromTaskA" sourceRef="TaskA" targetRef="InclusiveJoin" />
            <bpmn:sequenceFlow id="FromTaskB" sourceRef="TaskB" targetRef="InclusiveJoin" />
            <bpmn:sequenceFlow id="FromTaskC" sourceRef="TaskC" targetRef="InclusiveJoin" />
            <bpmn:sequenceFlow id="FromDefault" sourceRef="DefaultTask" targetRef="InclusiveJoin" />
            <bpmn:sequenceFlow id="ToEnd" sourceRef="InclusiveJoin" targetRef="End" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnInclusiveHandler();
        model = new BpmnModel();
        await model.loadXml(inclusiveGatewayXml);

        mockLocation = {
            nodeId: "InclusiveSplit",
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

    describe("processInclusiveGateway", () => {
        describe("split gateway", () => {
            test("should activate multiple paths based on conditions", () => {
                mockContext.variables = {
                    conditionA: true,
                    conditionB: true,
                    conditionC: false,
                };

                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveSplit", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(2);
                expect(result.nextLocations.map(loc => loc.nodeId)).toContain("TaskA");
                expect(result.nextLocations.map(loc => loc.nodeId)).toContain("TaskB");
                expect(result.nextLocations.map(loc => loc.nodeId)).not.toContain("TaskC");
            });

            test("should activate at least one path", () => {
                mockContext.variables = {
                    conditionA: false,
                    conditionB: false,
                    conditionC: false,
                };

                expect(() => {
                    handler.processInclusiveGateway(
                        model, 
                        "InclusiveSplit", 
                        mockLocation, 
                        mockContext,
                    );
                }).toThrow("No paths activated");
            });

            test("should track activated paths in state", () => {
                mockContext.variables = {
                    conditionA: true,
                    conditionB: true,
                    conditionC: true,
                };

                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveSplit", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.gateways.inclusiveStates).toHaveLength(1);
                const state = result.updatedContext.gateways.inclusiveStates[0];
                expect(state.gatewayId).toBe("InclusiveSplit");
                expect(state.activatedPaths).toHaveLength(3);
            });

            test("should record condition evaluations", () => {
                mockContext.variables = {
                    conditionA: true,
                    conditionB: false,
                    conditionC: true,
                };

                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveSplit", 
                    mockLocation, 
                    mockContext,
                );

                const state = result.updatedContext.gateways.inclusiveStates[0];
                expect(state.evaluatedConditions).toHaveLength(4); // Including default
                expect(state.evaluatedConditions[0].result).toBe(true);
                expect(state.evaluatedConditions[1].result).toBe(false);
                expect(state.evaluatedConditions[2].result).toBe(true);
            });

            test("should include path metadata", () => {
                mockContext.variables = { conditionA: true };

                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveSplit", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations[0].metadata.inclusiveGateway).toBe("InclusiveSplit");
                expect(result.nextLocations[0].metadata.pathId).toBeDefined();
                expect(result.pathUpdates).toHaveLength(1);
                expect(result.pathUpdates[0].action).toBe("activated");
            });
        });

        describe("join gateway", () => {
            beforeEach(() => {
                // Set up split state and location for join
                const splitId = "inclusive_split_InclusiveSplit_123";
                const inclusiveState: InclusiveGatewayState = {
                    id: splitId,
                    gatewayId: "InclusiveSplit",
                    evaluatedConditions: [
                        {
                            conditionId: `${splitId}_condition_0`,
                            expression: "conditionA == true",
                            result: true,
                            evaluatedAt: new Date(),
                        },
                        {
                            conditionId: `${splitId}_condition_1`,
                            expression: "conditionB == true",
                            result: true,
                            evaluatedAt: new Date(),
                        },
                    ],
                    activatedPaths: [`${splitId}_path_0`, `${splitId}_path_1`],
                };

                mockContext.gateways.inclusiveStates = [inclusiveState];
                mockContext.variables["completed_paths_InclusiveJoin"] = [];

                mockLocation = {
                    nodeId: "InclusiveJoin",
                    routineId: "routine_1",
                    type: "node",
                    metadata: {
                        pathId: `${splitId}_path_0`,
                    },
                };
            });

            test("should wait when not all paths complete", () => {
                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveJoin", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.processingComplete).toBe(false);
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].type).toBe("gateway_evaluation");
                expect(result.nextLocations[0].metadata.gatewayType).toBe("inclusive_join");
            });

            test("should track completed paths", () => {
                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveJoin", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.pathUpdates).toHaveLength(1);
                expect(result.pathUpdates[0].action).toBe("completed");
                expect(result.pathUpdates[0].pathId).toBe("inclusive_split_InclusiveSplit_123_path_0");
            });

            test("should proceed when all paths complete", () => {
                // Mark first path as already completed
                mockContext.variables["completed_paths_InclusiveJoin"] = [
                    "inclusive_split_InclusiveSplit_123_path_0",
                ];
                mockLocation.metadata.pathId = "inclusive_split_InclusiveSplit_123_path_1";

                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveJoin", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.processingComplete).toBe(true);
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("End");
                expect(result.pathUpdates.some(u => u.action === "synchronized")).toBe(true);
            });

            test("should clean up state after synchronization", () => {
                mockContext.variables["completed_paths_InclusiveJoin"] = [
                    "inclusive_split_InclusiveSplit_123_path_0",
                ];
                mockLocation.metadata.pathId = "inclusive_split_InclusiveSplit_123_path_1";

                const result = handler.processInclusiveGateway(
                    model, 
                    "InclusiveJoin", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.gateways.inclusiveStates).toHaveLength(0);
            });
        });

        describe("pass-through gateway", () => {
            test("should handle single flow gateway", async () => {
                const simpleXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:inclusiveGateway id="PassThrough">
                            <bpmn:incoming>In</bpmn:incoming>
                            <bpmn:outgoing>Out</bpmn:outgoing>
                        </bpmn:inclusiveGateway>
                        <bpmn:task id="NextTask">
                            <bpmn:incoming>Out</bpmn:incoming>
                        </bpmn:task>
                        <bpmn:sequenceFlow id="Out" sourceRef="PassThrough" targetRef="NextTask" />
                    </bpmn:process>
                </bpmn:definitions>`;

                const simpleModel = new BpmnModel();
                await simpleModel.loadXml(simpleXml);

                const result = handler.processInclusiveGateway(
                    simpleModel, 
                    "PassThrough", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("NextTask");
                expect(result.processingComplete).toBe(true);
            });
        });
    });

    describe("condition evaluation", () => {
        test("should evaluate simple variable conditions", () => {
            mockContext.variables = {
                isActive: true,
                status: "approved",
                count: 5,
            };

            expect((handler as any).evaluateCondition("isActive", mockContext.variables)).toBe(true);
            expect((handler as any).evaluateCondition("status == approved", mockContext.variables)).toBe(true);
            expect((handler as any).evaluateCondition("count > 3", mockContext.variables)).toBe(true);
        });

        test("should evaluate OR conditions", () => {
            mockContext.variables = {
                conditionA: true,
                conditionB: false,
            };

            expect((handler as any).evaluateCondition(
                "conditionA || conditionB", 
                mockContext.variables,
            )).toBe(true);

            mockContext.variables.conditionA = false;
            expect((handler as any).evaluateCondition(
                "conditionA || conditionB", 
                mockContext.variables,
            )).toBe(false);
        });

        test("should evaluate AND conditions", () => {
            mockContext.variables = {
                conditionA: true,
                conditionB: true,
            };

            expect((handler as any).evaluateCondition(
                "conditionA && conditionB", 
                mockContext.variables,
            )).toBe(true);

            mockContext.variables.conditionB = false;
            expect((handler as any).evaluateCondition(
                "conditionA && conditionB", 
                mockContext.variables,
            )).toBe(false);
        });

        test("should evaluate numeric comparisons", () => {
            mockContext.variables = { value: 10 };

            expect((handler as any).evaluateCondition("value > 5", mockContext.variables)).toBe(true);
            expect((handler as any).evaluateCondition("value < 20", mockContext.variables)).toBe(true);
            expect((handler as any).evaluateCondition("value >= 10", mockContext.variables)).toBe(true);
            expect((handler as any).evaluateCondition("value <= 10", mockContext.variables)).toBe(true);
        });

        test("should evaluate 'in' operator", () => {
            mockContext.variables = {
                status: "pending",
                allowedStatuses: ["pending", "approved", "rejected"],
            };

            expect((handler as any).evaluateCondition(
                "status in allowedStatuses", 
                mockContext.variables,
            )).toBe(true);
        });

        test("should handle null conditions as true", () => {
            expect((handler as any).evaluateCondition(null, mockContext.variables)).toBe(true);
        });

        test("should handle invalid conditions as false", () => {
            expect((handler as any).evaluateCondition(
                "invalid syntax {{", 
                mockContext.variables,
            )).toBe(false);
        });
    });

    describe("resolveInclusiveDeadlock", () => {
        test("should resolve stuck inclusive states", () => {
            const oldDate = new Date(Date.now() - 60000); // 1 minute ago
            mockContext.gateways.inclusiveStates = [{
                id: "stuck_state",
                gatewayId: "InclusiveJoin",
                evaluatedConditions: [{
                    conditionId: "cond_1",
                    expression: "true",
                    result: true,
                    evaluatedAt: oldDate,
                }],
                activatedPaths: ["path_1", "path_2"],
            }];

            const result = handler.resolveInclusiveDeadlock(
                mockContext, 
                "InclusiveJoin", 
                30000,
            );

            expect(result.gateways.inclusiveStates).toHaveLength(0);
        });

        test("should not affect recent states", () => {
            const recentDate = new Date();
            mockContext.gateways.inclusiveStates = [{
                id: "recent_state",
                gatewayId: "InclusiveJoin",
                evaluatedConditions: [{
                    conditionId: "cond_1",
                    expression: "true",
                    result: true,
                    evaluatedAt: recentDate,
                }],
                activatedPaths: ["path_1"],
            }];

            const result = handler.resolveInclusiveDeadlock(
                mockContext, 
                "InclusiveJoin", 
                30000,
            );

            expect(result.gateways.inclusiveStates).toHaveLength(1);
        });
    });

    describe("getInclusiveGatewaySummary", () => {
        test("should provide correct summary", () => {
            mockContext.gateways.inclusiveStates = [
                {
                    id: "state_1",
                    gatewayId: "Gateway1",
                    evaluatedConditions: [],
                    activatedPaths: ["path_1", "path_2"],
                },
                {
                    id: "state_2",
                    gatewayId: "Gateway2",
                    evaluatedConditions: [],
                    activatedPaths: ["path_3"],
                },
                {
                    id: "state_3",
                    gatewayId: "Gateway1",
                    evaluatedConditions: [],
                    activatedPaths: ["path_4"],
                },
            ];

            const summary = handler.getInclusiveGatewaySummary(mockContext);

            expect(summary.activeGateways).toBe(2); // Unique gateways
            expect(summary.activePaths).toBe(4); // Total paths
            expect(summary.pendingSynchronizations).toBe(3); // Total states
        });
    });

    describe("helper methods", () => {
        test("updatePathCompletion should add path to completed list", () => {
            mockContext.variables["completed_paths_Gateway1"] = ["path_1"];

            const result = handler.updatePathCompletion(
                mockContext, 
                "Gateway1", 
                "path_2",
            );

            expect(result.variables["completed_paths_Gateway1"]).toContain("path_1");
            expect(result.variables["completed_paths_Gateway1"]).toContain("path_2");
        });

        test("isInclusiveGatewayReady should check path completion", () => {
            mockContext.gateways.inclusiveStates = [{
                id: "state_1",
                gatewayId: "Gateway1",
                evaluatedConditions: [],
                activatedPaths: ["path_1", "path_2"],
            }];
            mockContext.variables["completed_paths_Gateway1"] = ["path_1", "path_2"];

            const ready = handler.isInclusiveGatewayReady("Gateway1", mockContext);
            expect(ready).toBe(true);
        });
    });

    describe("error handling", () => {
        test("should throw error for non-existent gateway", () => {
            expect(() => {
                handler.processInclusiveGateway(
                    model, 
                    "NonExistentGateway", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow("Inclusive gateway not found: NonExistentGateway");
        });
    });
});
