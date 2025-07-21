import { beforeEach, describe, expect, test } from "vitest";
import { BpmnSubprocessHandler } from "./BpmnSubprocessHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation } from "../../types.js";

describe("BpmnSubprocessHandler", () => {
    let handler: BpmnSubprocessHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const subprocessXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1">
            <bpmn:startEvent id="MainStart">
                <bpmn:outgoing>ToSubprocess</bpmn:outgoing>
            </bpmn:startEvent>
            <bpmn:subProcess id="EmbeddedSubprocess">
                <bpmn:incoming>ToSubprocess</bpmn:incoming>
                <bpmn:outgoing>FromSubprocess</bpmn:outgoing>
                <bpmn:startEvent id="SubStart">
                    <bpmn:outgoing>ToSubTask</bpmn:outgoing>
                </bpmn:startEvent>
                <bpmn:task id="SubTask">
                    <bpmn:incoming>ToSubTask</bpmn:incoming>
                    <bpmn:outgoing>ToSubEnd</bpmn:outgoing>
                </bpmn:task>
                <bpmn:endEvent id="SubEnd">
                    <bpmn:incoming>ToSubEnd</bpmn:incoming>
                </bpmn:endEvent>
                <bpmn:sequenceFlow id="ToSubTask" sourceRef="SubStart" targetRef="SubTask" />
                <bpmn:sequenceFlow id="ToSubEnd" sourceRef="SubTask" targetRef="SubEnd" />
            </bpmn:subProcess>
            <bpmn:callActivity id="CallActivity" calledElement="ExternalProcess">
                <bpmn:incoming>ToCall</bpmn:incoming>
                <bpmn:outgoing>FromCall</bpmn:outgoing>
            </bpmn:callActivity>
            <bpmn:endEvent id="MainEnd">
                <bpmn:incoming>FromSubprocess</bpmn:incoming>
            </bpmn:endEvent>
            <bpmn:sequenceFlow id="ToSubprocess" sourceRef="MainStart" targetRef="EmbeddedSubprocess" />
            <bpmn:sequenceFlow id="FromSubprocess" sourceRef="EmbeddedSubprocess" targetRef="MainEnd" />
            <bpmn:sequenceFlow id="ToCall" sourceRef="MainStart" targetRef="CallActivity" />
            <bpmn:sequenceFlow id="FromCall" sourceRef="CallActivity" targetRef="MainEnd" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnSubprocessHandler();
        model = new BpmnModel();
        await model.loadXml(subprocessXml);

        mockLocation = {
            nodeId: "EmbeddedSubprocess",
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

    describe("processSubprocess", () => {
        describe("embedded subprocess", () => {
            test("should enter embedded subprocess", () => {
                const result = handler.processSubprocess(
                    model, 
                    "EmbeddedSubprocess", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.subprocessType).toBe("embedded");
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("SubStart");
                expect(result.nextLocations[0].type).toBe("subprocess_context");
            });

            test("should create subprocess tracking", () => {
                const result = handler.processSubprocess(
                    model, 
                    "EmbeddedSubprocess", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.subprocesses.activeSubprocesses).toHaveLength(1);
                const subprocess = result.updatedContext.subprocesses.activeSubprocesses[0];
                expect(subprocess.subprocessId).toBe("EmbeddedSubprocess");
                expect(subprocess.type).toBe("embedded");
                expect(subprocess.status).toBe("active");
            });

            test("should maintain parent context reference", () => {
                const result = handler.processSubprocess(
                    model, 
                    "EmbeddedSubprocess", 
                    mockLocation, 
                    mockContext,
                );

                const subprocess = result.updatedContext.subprocesses.activeSubprocesses[0];
                expect(subprocess.parentProcessId).toBe("routine_1");
            });
        });

        describe("call activity", () => {
            beforeEach(() => {
                mockLocation.nodeId = "CallActivity";
            });

            test("should process call activity", () => {
                const result = handler.processSubprocess(
                    model, 
                    "CallActivity", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.subprocessType).toBe("call");
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].type).toBe("call_activity");
            });

            test("should reference called element", () => {
                const result = handler.processSubprocess(
                    model, 
                    "CallActivity", 
                    mockLocation, 
                    mockContext,
                );

                const subprocess = result.updatedContext.subprocesses.activeSubprocesses[0];
                expect(subprocess.calledElement).toBe("ExternalProcess");
            });

            test("should handle input/output mappings", async () => {
                const mappingXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:callActivity id="CallWithMapping" calledElement="Process_2">
                            <bpmn:extensionElements>
                                <inputOutput>
                                    <inputParameter name="input1">value1</inputParameter>
                                    <outputParameter name="output1">result</outputParameter>
                                </inputOutput>
                            </bpmn:extensionElements>
                        </bpmn:callActivity>
                    </bpmn:process>
                </bpmn:definitions>`;

                const mappingModel = new BpmnModel();
                await mappingModel.loadXml(mappingXml);

                const result = handler.processSubprocess(
                    mappingModel, 
                    "CallWithMapping", 
                    mockLocation, 
                    mockContext,
                );

                const subprocess = result.updatedContext.subprocesses.activeSubprocesses[0];
                expect(subprocess.inputMappings).toBeDefined();
                expect(subprocess.outputMappings).toBeDefined();
            });
        });

        describe("subprocess completion", () => {
            test("should complete subprocess when end event reached", () => {
                // Set up active subprocess
                mockContext.subprocesses.activeSubprocesses = [{
                    id: "sub_1",
                    subprocessId: "EmbeddedSubprocess",
                    type: "embedded",
                    parentProcessId: "routine_1",
                    status: "active",
                    startedAt: new Date(),
                }];

                mockLocation = {
                    nodeId: "SubEnd",
                    routineId: "routine_1",
                    type: "subprocess_context",
                    metadata: {
                        subprocessId: "EmbeddedSubprocess",
                    },
                };

                const result = handler.completeSubprocess(
                    model, 
                    "EmbeddedSubprocess", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("MainEnd");
                expect(result.updatedContext.subprocesses.activeSubprocesses).toHaveLength(0);
                expect(result.updatedContext.subprocesses.completedSubprocesses).toHaveLength(1);
            });

            test("should preserve subprocess results", () => {
                mockContext.subprocesses.activeSubprocesses = [{
                    id: "sub_1",
                    subprocessId: "EmbeddedSubprocess",
                    type: "embedded",
                    parentProcessId: "routine_1",
                    status: "active",
                    startedAt: new Date(),
                }];
                mockContext.variables.subprocessResult = "completed";

                const result = handler.completeSubprocess(
                    model, 
                    "EmbeddedSubprocess", 
                    mockLocation, 
                    mockContext,
                );

                const completed = result.updatedContext.subprocesses.completedSubprocesses[0];
                expect(completed.completedAt).toBeDefined();
                expect(result.updatedContext.variables.subprocessResult).toBe("completed");
            });
        });

        describe("error handling in subprocess", () => {
            test("should propagate errors from subprocess", async () => {
                const errorXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:subProcess id="ErrorSubprocess">
                            <bpmn:endEvent id="ErrorEnd">
                                <bpmn:errorEventDefinition errorRef="Error_1" />
                            </bpmn:endEvent>
                        </bpmn:subProcess>
                        <bpmn:boundaryEvent id="ErrorBoundary" attachedToRef="ErrorSubprocess">
                            <bpmn:errorEventDefinition errorRef="Error_1" />
                            <bpmn:outgoing>ToErrorHandler</bpmn:outgoing>
                        </bpmn:boundaryEvent>
                    </bpmn:process>
                </bpmn:definitions>`;

                const errorModel = new BpmnModel();
                await errorModel.loadXml(errorXml);

                mockContext.subprocesses.activeSubprocesses = [{
                    id: "sub_1",
                    subprocessId: "ErrorSubprocess",
                    type: "embedded",
                    parentProcessId: "routine_1",
                    status: "active",
                    startedAt: new Date(),
                }];

                mockLocation = {
                    nodeId: "ErrorEnd",
                    routineId: "routine_1",
                    type: "subprocess_context",
                    metadata: {
                        subprocessId: "ErrorSubprocess",
                    },
                };

                const result = handler.completeSubprocess(
                    errorModel, 
                    "ErrorSubprocess", 
                    mockLocation, 
                    mockContext,
                    { errorCode: "Error_1" },
                );

                expect(result.updatedContext.variables.errors).toBeDefined();
                expect(result.updatedContext.subprocesses.completedSubprocesses[0].error).toBeDefined();
            });
        });

        describe("navigateWithinSubprocess", () => {
            test("should navigate within subprocess context", () => {
                mockContext.subprocesses.activeSubprocesses = [{
                    id: "sub_1",
                    subprocessId: "EmbeddedSubprocess",
                    type: "embedded",
                    parentProcessId: "routine_1",
                    status: "active",
                    startedAt: new Date(),
                }];

                mockLocation = {
                    nodeId: "SubTask",
                    routineId: "routine_1",
                    type: "subprocess_context",
                    metadata: {
                        subprocessId: "EmbeddedSubprocess",
                    },
                };

                const result = handler.navigateWithinSubprocess(
                    model, 
                    "SubTask", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("SubEnd");
                expect(result.nextLocations[0].type).toBe("subprocess_context");
                expect(result.nextLocations[0].metadata.subprocessId).toBe("EmbeddedSubprocess");
            });
        });

        describe("getActiveSubprocesses", () => {
            test("should return active subprocesses", () => {
                mockContext.subprocesses.activeSubprocesses = [
                    {
                        id: "sub_1",
                        subprocessId: "Sub1",
                        type: "embedded",
                        parentProcessId: "routine_1",
                        status: "active",
                        startedAt: new Date(),
                    },
                    {
                        id: "sub_2",
                        subprocessId: "Sub2",
                        type: "call",
                        parentProcessId: "routine_1",
                        status: "active",
                        startedAt: new Date(),
                    },
                ];

                const active = handler.getActiveSubprocesses(mockContext);
                expect(active).toHaveLength(2);
                expect(active.map(s => s.subprocessId)).toContain("Sub1");
                expect(active.map(s => s.subprocessId)).toContain("Sub2");
            });

            test("should filter by type", () => {
                mockContext.subprocesses.activeSubprocesses = [
                    {
                        id: "sub_1",
                        subprocessId: "Sub1",
                        type: "embedded",
                        parentProcessId: "routine_1",
                        status: "active",
                        startedAt: new Date(),
                    },
                    {
                        id: "sub_2",
                        subprocessId: "Sub2",
                        type: "call",
                        parentProcessId: "routine_1",
                        status: "active",
                        startedAt: new Date(),
                    },
                ];

                const embedded = handler.getActiveSubprocesses(mockContext, "embedded");
                expect(embedded).toHaveLength(1);
                expect(embedded[0].type).toBe("embedded");
            });
        });

        describe("transaction subprocess", () => {
            test("should handle transaction subprocess", async () => {
                const transactionXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:transaction id="TransactionSub">
                            <bpmn:startEvent id="TxStart" />
                            <bpmn:task id="TxTask" />
                            <bpmn:endEvent id="TxEnd" />
                        </bpmn:transaction>
                    </bpmn:process>
                </bpmn:definitions>`;

                const transactionModel = new BpmnModel();
                await transactionModel.loadXml(transactionXml);

                const result = handler.processSubprocess(
                    transactionModel, 
                    "TransactionSub", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.subprocessType).toBe("transaction");
                const subprocess = result.updatedContext.subprocesses.activeSubprocesses[0];
                expect(subprocess.isTransaction).toBe(true);
            });
        });

        describe("adhoc subprocess", () => {
            test("should handle adhoc subprocess", async () => {
                const adhocXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:adHocSubProcess id="AdhocSub" ordering="Parallel">
                            <bpmn:task id="AdTask1" />
                            <bpmn:task id="AdTask2" />
                            <bpmn:completionCondition>allTasksComplete</bpmn:completionCondition>
                        </bpmn:adHocSubProcess>
                    </bpmn:process>
                </bpmn:definitions>`;

                const adhocModel = new BpmnModel();
                await adhocModel.loadXml(adhocXml);

                const result = handler.processSubprocess(
                    adhocModel, 
                    "AdhocSub", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.subprocessType).toBe("adhoc");
                const subprocess = result.updatedContext.subprocesses.activeSubprocesses[0];
                expect(subprocess.isAdhoc).toBe(true);
            });
        });
    });

    describe("error handling", () => {
        test("should throw error for non-existent subprocess", () => {
            expect(() => {
                handler.processSubprocess(
                    model, 
                    "NonExistentSubprocess", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow("Subprocess not found: NonExistentSubprocess");
        });

        test("should handle missing start event in subprocess", async () => {
            const noStartXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="NoStartSub">
                        <bpmn:task id="Task1" />
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            const noStartModel = new BpmnModel();
            await noStartModel.loadXml(noStartXml);

            expect(() => {
                handler.processSubprocess(
                    noStartModel, 
                    "NoStartSub", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow();
        });
    });
});
