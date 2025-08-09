import { beforeEach, describe, expect, test } from "vitest";
import { BpmnAdvancedActivityHandler } from "./BpmnAdvancedActivityHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation } from "../../types.js";

describe("BpmnAdvancedActivityHandler", () => {
    let handler: BpmnAdvancedActivityHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const advancedActivitiesXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1" isExecutable="true">
            <bpmn:task id="MultiInstanceTask">
                <bpmn:multiInstanceLoopCharacteristics isSequential="false">
                    <bpmn:loopCardinality>#{3}</bpmn:loopCardinality>
                </bpmn:multiInstanceLoopCharacteristics>
                <bpmn:outgoing>Flow1</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="SequentialMultiTask">
                <bpmn:multiInstanceLoopCharacteristics isSequential="true">
                    <bpmn:loopDataInputRef>items</bpmn:loopDataInputRef>
                    <bpmn:inputDataItem>item</bpmn:inputDataItem>
                </bpmn:multiInstanceLoopCharacteristics>
                <bpmn:outgoing>Flow2</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="LoopTask">
                <bpmn:standardLoopCharacteristics testBefore="true">
                    <bpmn:loopCondition>#{counter &lt; 5}</bpmn:loopCondition>
                    <bpmn:loopMaximum>10</bpmn:loopMaximum>
                </bpmn:standardLoopCharacteristics>
                <bpmn:outgoing>Flow3</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="CompensationTask" isForCompensation="true">
                <bpmn:outgoing>Flow4</bpmn:outgoing>
            </bpmn:task>
            <bpmn:task id="NextTask">
                <bpmn:incoming>Flow1</bpmn:incoming>
                <bpmn:incoming>Flow2</bpmn:incoming>
                <bpmn:incoming>Flow3</bpmn:incoming>
                <bpmn:incoming>Flow4</bpmn:incoming>
            </bpmn:task>
            <bpmn:sequenceFlow id="Flow1" sourceRef="MultiInstanceTask" targetRef="NextTask" />
            <bpmn:sequenceFlow id="Flow2" sourceRef="SequentialMultiTask" targetRef="NextTask" />
            <bpmn:sequenceFlow id="Flow3" sourceRef="LoopTask" targetRef="NextTask" />
            <bpmn:sequenceFlow id="Flow4" sourceRef="CompensationTask" targetRef="NextTask" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnAdvancedActivityHandler();
        model = new BpmnModel();
        await model.loadXml(advancedActivitiesXml);

        mockLocation = {
            nodeId: "MultiInstanceTask",
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
                stack: [],
                eventSubprocesses: [],
            },
            external: {
                messageEvents: [],
                webhookEvents: [],
                signalEvents: [],
            },
            gateways: {
                inclusiveStates: [],
                complexConditions: [],
            },
        };
    });

    describe("processAdvancedActivity", () => {
        test("should detect multi-instance characteristics", () => {
            const result = handler.processAdvancedActivity(
                model, 
                "MultiInstanceTask", 
                mockLocation, 
                mockContext,
            );

            expect(result.activityComplete).toBe(false);
            expect(result.nextLocations).toHaveLength(3); // Parallel instances
            expect(result.instanceInfo?.type).toBe("multi_instance");
        });

        test("should detect loop characteristics", () => {
            mockLocation.nodeId = "LoopTask";
            mockContext.variables.counter = 0;

            const result = handler.processAdvancedActivity(
                model, 
                "LoopTask", 
                mockLocation, 
                mockContext,
            );

            expect(result.activityComplete).toBe(false);
            expect(result.instanceInfo?.type).toBe("loop");
        });

        test("should detect compensation handlers", () => {
            mockLocation.nodeId = "CompensationTask";

            const result = handler.processAdvancedActivity(
                model, 
                "CompensationTask", 
                mockLocation, 
                mockContext,
            );

            expect(result.activityComplete).toBe(true);
        });

        test("should return empty result for normal activities", async () => {
            const normalXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:task id="NormalTask" />
                </bpmn:process>
            </bpmn:definitions>`;

            const normalModel = new BpmnModel();
            await normalModel.loadXml(normalXml);

            const result = handler.processAdvancedActivity(
                normalModel, 
                "NormalTask", 
                mockLocation, 
                mockContext,
            );

            expect(result.activityComplete).toBe(false);
            expect(result.nextLocations).toHaveLength(0);
        });
    });

    describe("multi-instance activities", () => {
        describe("parallel multi-instance", () => {
            test("should start all instances in parallel", () => {
                const result = handler.processAdvancedActivity(
                    model, 
                    "MultiInstanceTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(3);
                expect(result.updatedContext.variables.multiInstanceStates).toBeDefined();
                const states = result.updatedContext.variables.multiInstanceStates as any[];
                expect(states).toHaveLength(1);
                expect(states[0].isSequential).toBe(false);
                expect(states[0].totalInstances).toBe(3);
            });

            test("should create instance locations with metadata", () => {
                const result = handler.processAdvancedActivity(
                    model, 
                    "MultiInstanceTask", 
                    mockLocation, 
                    mockContext,
                );

                result.nextLocations.forEach((loc, index) => {
                    expect(loc.type).toBe("multi_instance_execution");
                    expect(loc.metadata.instanceIndex).toBe(index);
                    expect(loc.metadata.executionType).toBe("parallel");
                });
            });

            test("should handle collection-based multi-instance", () => {
                mockLocation.nodeId = "SequentialMultiTask";
                mockContext.variables.items = ["item1", "item2", "item3"];

                const result = handler.processAdvancedActivity(
                    model, 
                    "SequentialMultiTask", 
                    mockLocation, 
                    mockContext,
                );

                const states = result.updatedContext.variables.multiInstanceStates as any[];
                expect(states[0].totalInstances).toBe(3);
                expect(states[0].collection).toEqual(["item1", "item2", "item3"]);
            });
        });

        describe("sequential multi-instance", () => {
            test("should start first instance only", () => {
                mockLocation.nodeId = "SequentialMultiTask";
                mockContext.variables.items = ["item1", "item2", "item3"];

                const result = handler.processAdvancedActivity(
                    model, 
                    "SequentialMultiTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.instanceInfo?.currentIteration).toBe(1);
                expect(result.instanceInfo?.totalIterations).toBe(3);
            });

            test("should set element variable for current instance", () => {
                mockLocation.nodeId = "SequentialMultiTask";
                mockContext.variables.items = ["item1", "item2", "item3"];

                const result = handler.processAdvancedActivity(
                    model, 
                    "SequentialMultiTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.variables.item).toBe("item1");
                expect(result.updatedContext.variables.loopCounter).toBe(0);
            });

            test("should continue to next instance", () => {
                // Set up existing multi-instance state
                const state = {
                    id: "multi_instance_123",
                    activityId: "SequentialMultiTask",
                    isSequential: true,
                    totalInstances: 3,
                    currentInstance: 0,
                    completedInstances: 0,
                    instanceResults: [
                        { instanceId: "inst_0", result: "done", status: "completed" as const },
                    ],
                    collection: ["item1", "item2", "item3"],
                    elementVariable: "item",
                    startedAt: new Date(),
                    status: "running" as const,
                };
                mockContext.variables.multiInstanceStates = [state];
                mockContext.variables.items = ["item1", "item2", "item3"];
                mockLocation.nodeId = "SequentialMultiTask";
                mockLocation.metadata = { instanceId: "inst_0" };

                const result = handler.processAdvancedActivity(
                    model, 
                    "SequentialMultiTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.instanceInfo?.currentIteration).toBe(2);
            });

            test("should complete after all instances", () => {
                const state = {
                    id: "multi_instance_123",
                    activityId: "SequentialMultiTask",
                    isSequential: true,
                    totalInstances: 2,
                    currentInstance: 1,
                    completedInstances: 1,
                    instanceResults: [
                        { instanceId: "inst_0", result: "done", status: "completed" as const },
                        { instanceId: "inst_1", result: undefined, status: "running" as const },
                    ],
                    collection: ["item1", "item2"],
                    elementVariable: "item",
                    startedAt: new Date(),
                    status: "running" as const,
                };
                mockContext.variables.multiInstanceStates = [state];
                mockLocation.metadata = { instanceId: "inst_1" };

                const result = handler.processAdvancedActivity(
                    model, 
                    "SequentialMultiTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(true);
                expect(result.nextLocations[0].nodeId).toBe("NextTask");
            });
        });

        describe("empty collections", () => {
            test("should complete immediately with empty collection", () => {
                mockLocation.nodeId = "SequentialMultiTask";
                mockContext.variables.items = [];

                const result = handler.processAdvancedActivity(
                    model, 
                    "SequentialMultiTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(true);
                expect(result.nextLocations[0].nodeId).toBe("NextTask");
            });
        });
    });

    describe("loop activities", () => {
        describe("standard loops", () => {
            test("should start loop execution with testBefore", () => {
                mockLocation.nodeId = "LoopTask";
                mockContext.variables.counter = 0;

                const result = handler.processAdvancedActivity(
                    model, 
                    "LoopTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(false);
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].type).toBe("loop_execution");
                expect(result.instanceInfo?.type).toBe("loop");
            });

            test("should skip loop when condition false", () => {
                mockLocation.nodeId = "LoopTask";
                mockContext.variables.counter = 10;

                const result = handler.processAdvancedActivity(
                    model, 
                    "LoopTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(true);
                expect(result.nextLocations[0].nodeId).toBe("NextTask");
            });

            test("should continue loop iterations", () => {
                const loopState = {
                    id: "loop_123",
                    activityId: "LoopTask",
                    currentIteration: 2,
                    maxIterations: 10,
                    testBefore: true,
                    loopCondition: "counter < 5",
                    startedAt: new Date(),
                    status: "running" as const,
                };
                mockContext.variables.loopStates = [loopState];
                mockContext.variables.counter = 3;
                mockLocation.nodeId = "LoopTask";

                const result = handler.processAdvancedActivity(
                    model, 
                    "LoopTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(false);
                expect(loopState.currentIteration).toBe(3);
            });

            test("should complete loop when condition false", () => {
                const loopState = {
                    id: "loop_123",
                    activityId: "LoopTask",
                    currentIteration: 4,
                    maxIterations: 10,
                    testBefore: true,
                    loopCondition: "counter < 5",
                    startedAt: new Date(),
                    status: "running" as const,
                };
                mockContext.variables.loopStates = [loopState];
                mockContext.variables.counter = 5;
                mockLocation.nodeId = "LoopTask";

                const result = handler.processAdvancedActivity(
                    model, 
                    "LoopTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(true);
                expect(loopState.status).toBe("completed");
            });

            test("should respect maximum iterations", () => {
                const loopState = {
                    id: "loop_123",
                    activityId: "LoopTask",
                    currentIteration: 9,
                    maxIterations: 10,
                    testBefore: true,
                    loopCondition: "counter < 100",
                    startedAt: new Date(),
                    status: "running" as const,
                };
                mockContext.variables.loopStates = [loopState];
                mockContext.variables.counter = 50;
                mockLocation.nodeId = "LoopTask";

                const result = handler.processAdvancedActivity(
                    model, 
                    "LoopTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(true);
            });
        });

        describe("do-while loops", () => {
            test("should execute at least once with testBefore=false", async () => {
                const doWhileXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:task id="DoWhileTask">
                            <bpmn:standardLoopCharacteristics testBefore="false">
                                <bpmn:loopCondition>shouldContinue</bpmn:loopCondition>
                            </bpmn:standardLoopCharacteristics>
                        </bpmn:task>
                    </bpmn:process>
                </bpmn:definitions>`;

                const doWhileModel = new BpmnModel();
                await doWhileModel.loadXml(doWhileXml);
                mockContext.variables.shouldContinue = false;

                const result = handler.processAdvancedActivity(
                    doWhileModel, 
                    "DoWhileTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.activityComplete).toBe(false);
                expect(result.nextLocations).toHaveLength(1);
            });
        });
    });

    describe("compensation", () => {
        test("should process compensation activity", () => {
            mockLocation.nodeId = "CompensationTask";

            const result = handler.processAdvancedActivity(
                model, 
                "CompensationTask", 
                mockLocation, 
                mockContext,
            );

            expect(result.activityComplete).toBe(true);
            expect(result.nextLocations[0].nodeId).toBe("NextTask");
        });
    });

    describe("condition evaluation", () => {
        test("should evaluate simple conditions", () => {
            const evaluate = (handler as any).evaluateLoopCondition;

            mockContext.variables = {
                isActive: true,
                counter: 5,
            };

            expect(evaluate("isActive", mockContext.variables)).toBe(true);
            expect(evaluate("counter", mockContext.variables)).toBe(true);
            expect(evaluate("nonExistent", mockContext.variables)).toBe(false);
        });

        test("should evaluate comparison conditions", () => {
            const evaluate = (handler as any).evaluateLoopCondition;

            mockContext.variables = {
                counter: 5,
                status: "active",
            };

            expect(evaluate("counter == 5", mockContext.variables)).toBe(true);
            expect(evaluate("counter != 10", mockContext.variables)).toBe(true);
            expect(evaluate("counter > 3", mockContext.variables)).toBe(true);
            expect(evaluate("counter < 10", mockContext.variables)).toBe(true);
            expect(evaluate("counter >= 5", mockContext.variables)).toBe(true);
            expect(evaluate("counter <= 5", mockContext.variables)).toBe(true);
            expect(evaluate("status == active", mockContext.variables)).toBe(true);
        });

        test("should handle invalid conditions", () => {
            const evaluate = (handler as any).evaluateLoopCondition;

            expect(evaluate("invalid {{", mockContext.variables)).toBe(false);
            expect(evaluate("", mockContext.variables)).toBe(true); // Default
        });
    });

    describe("completion conditions", () => {
        test("should handle multi-instance completion condition", async () => {
            const completionXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:task id="CompletionTask">
                        <bpmn:multiInstanceLoopCharacteristics>
                            <bpmn:loopCardinality>5</bpmn:loopCardinality>
                            <bpmn:completionCondition>nrOfCompletedInstances >= 3</bpmn:completionCondition>
                        </bpmn:multiInstanceLoopCharacteristics>
                    </bpmn:task>
                </bpmn:process>
            </bpmn:definitions>`;

            const completionModel = new BpmnModel();
            await completionModel.loadXml(completionXml);

            const result = handler.processAdvancedActivity(
                completionModel, 
                "CompletionTask", 
                mockLocation, 
                mockContext,
            );

            const states = result.updatedContext.variables.multiInstanceStates as any[];
            expect(states[0].totalInstances).toBe(5);
            // Completion condition would be evaluated during execution
        });
    });

    describe("error handling", () => {
        test("should throw error for non-existent activity", () => {
            expect(() => {
                handler.processAdvancedActivity(
                    model, 
                    "NonExistentActivity", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow("Activity not found: NonExistentActivity");
        });
    });

    describe("result aggregation", () => {
        test("should aggregate multi-instance results", () => {
            const state = {
                id: "multi_instance_123",
                activityId: "MultiInstanceTask",
                isSequential: false,
                totalInstances: 3,
                currentInstance: 2,
                completedInstances: 3,
                instanceResults: [
                    { instanceId: "inst_0", result: "result1", status: "completed" as const },
                    { instanceId: "inst_1", result: "result2", status: "completed" as const },
                    { instanceId: "inst_2", result: "result3", status: "completed" as const },
                ],
                startedAt: new Date(),
                status: "running" as const,
            };
            mockContext.variables.multiInstanceStates = [state];
            mockLocation.metadata = { instanceId: "inst_2", instanceResult: "result3" };

            const result = handler.processAdvancedActivity(
                model, 
                "MultiInstanceTask", 
                mockLocation, 
                mockContext,
            );

            expect(result.updatedContext.variables["MultiInstanceTask_results"]).toEqual([
                "result1", "result2", "result3",
            ]);
        });
    });
});
