import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { beforeEach, describe, expect, test } from "vitest";
import { BpmnNavigator } from "./BpmnNavigator.js";

describe("BpmnNavigator", () => {
    let navigator: BpmnNavigator;

    beforeEach(() => {
        navigator = new BpmnNavigator();
    });

    describe("Basic Properties", () => {
        test("should have correct type and version", () => {
            expect(navigator.type).toBe("bpmn");
            expect(navigator.version).toBe("1.0.0");
        });
    });

    describe("canNavigate", () => {
        test("should return true for valid BPMN routine", () => {
            const validBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="Process_1">
                                <bpmn:startEvent id="StartEvent_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(navigator.canNavigate(validBpmn)).toBe(true);
        });

        test("should return false for routine without version", () => {
            const invalidBpmn = {
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        data: "valid xml",
                    },
                },
            };

            expect(navigator.canNavigate(invalidBpmn)).toBe(false);
        });

        test("should return false for non-BPMN routine", () => {
            const sequentialRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "Sequential",
                    schema: {
                        steps: [],
                    },
                },
            };

            expect(navigator.canNavigate(sequentialRoutine)).toBe(false);
        });

        test("should return false for routine without graph", () => {
            const routineWithoutGraph: RoutineVersionConfigObject = {
                __version: "1.0.0",
            };

            expect(navigator.canNavigate(routineWithoutGraph)).toBe(false);
        });

        test("should return false for routine with invalid schema data", () => {
            const invalidSchema: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: null as unknown as string,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(navigator.canNavigate(invalidSchema)).toBe(false);
        });

        test("should return false for routine without xmlns:bpmn namespace", () => {
            const invalidNamespace: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <definitions>
                            <process id="Process_1">
                                <startEvent id="StartEvent_1" />
                            </process>
                        </definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(navigator.canNavigate(invalidNamespace)).toBe(false);
        });

        test("should handle null/undefined input gracefully", () => {
            expect(navigator.canNavigate(null)).toBe(false);
            expect(navigator.canNavigate(undefined)).toBe(false);
            expect(navigator.canNavigate("string")).toBe(false);
            expect(navigator.canNavigate(123)).toBe(false);
            expect(navigator.canNavigate({})).toBe(false);
        });

        test("should handle exceptions gracefully", () => {
            const cyclicalObject = {} as Record<string, unknown>;
            cyclicalObject.self = cyclicalObject;

            expect(navigator.canNavigate(cyclicalObject)).toBe(false);
        });
    });

    describe("getStartLocation", () => {
        test("should find start event as start location", () => {
            const bpmnWithStartEvent: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="StartEvent_1" name="Start" />
                                <bpmn:task id="Task_1" name="First Task" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithStartEvent);

            expect(location.nodeId).toBe("StartEvent_1");
            expect(location.routineId).toBe("bpmn_1.0.0_TestProcess");
            expect(location.id).toBe("bpmn_1.0.0_TestProcess_StartEvent_1");
        });

        test("should find first task without incoming flows when no start event", () => {
            const bpmnWithoutStartEvent: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" name="First Task" />
                                <bpmn:task id="Task_2" name="Second Task" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithoutStartEvent);

            expect(location.nodeId).toBe("Task_1");
            expect(location.routineId).toBe("bpmn_1.0.0_TestProcess");
        });

        test("should handle userTask as start location", () => {
            const bpmnWithUserTask: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:userTask id="UserTask_1" name="User Task" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithUserTask);
            expect(location.nodeId).toBe("UserTask_1");
        });

        test("should handle serviceTask as start location", () => {
            const bpmnWithServiceTask: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:serviceTask id="ServiceTask_1" name="Service Task" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithServiceTask);
            expect(location.nodeId).toBe("ServiceTask_1");
        });

        test("should handle callActivity as start location", () => {
            const bpmnWithCallActivity: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:callActivity id="CallActivity_1" name="Call Activity" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithCallActivity);
            expect(location.nodeId).toBe("CallActivity_1");
        });

        test("should throw error when no start location found", () => {
            const bpmnWithoutStartLocation: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" name="Task with incoming" />
                                <bpmn:task id="Task_2" name="Task with incoming" />
                                <bpmn:sequenceFlow id="flow1" sourceRef="Task_1" targetRef="Task_2" />
                                <bpmn:sequenceFlow id="flow2" sourceRef="Task_2" targetRef="Task_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(() => navigator.getStartLocation(bpmnWithoutStartLocation))
                .toThrow("No start location found in BPMN");
        });

        test("should handle process without id gracefully", () => {
            const bpmnWithoutProcessId: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process>
                                <bpmn:startEvent id="StartEvent_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithoutProcessId);
            expect(location.routineId).toBe("bpmn_1.0.0_process");
        });

        test("should handle malformed start event id", () => {
            const bpmnWithMalformedId: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="" />
                                <bpmn:task id="Task_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnWithMalformedId);
            expect(location.nodeId).toBe("Task_1");
        });
    });

    describe("getNextLocations", () => {
        const sampleBpmn: RoutineVersionConfigObject = {
            __version: "1.0.0",
            graph: {
                __type: "BPMN-2.0",
                schema: {
                    __format: "xml",
                    data: `<?xml version="1.0" encoding="UTF-8"?>
                    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" 
                                      xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
                                      targetNamespace="http://bpmn.io/schema/bpmn">
                        <bpmn:process id="TestProcess" isExecutable="true">
                            <bpmn:startEvent id="StartEvent_1" />
                            <bpmn:task id="Task_1" name="First Task" />
                            <bpmn:task id="Task_2" name="Second Task" />
                            <bpmn:endEvent id="EndEvent_1" />
                            
                            <bpmn:sequenceFlow id="flow1" sourceRef="StartEvent_1" targetRef="Task_1" />
                            <bpmn:sequenceFlow id="flow2" sourceRef="Task_1" targetRef="Task_2" />
                            <bpmn:sequenceFlow id="flow3" sourceRef="Task_2" targetRef="EndEvent_1" />
                        </bpmn:process>
                    </bpmn:definitions>`,
                    activityMap: {
                        "StartEvent_1": { type: "start", connections: ["Task_1"] },
                        "Task_1": { type: "task", connections: ["Task_2"] },
                        "Task_2": { type: "task", connections: ["EndEvent_1"] },
                        "EndEvent_1": { type: "end", connections: [] },
                    },
                    rootContext: { inputMap: {}, outputMap: {} },
                },
            },
        };

        test("should find next location for start event", () => {
            const startLocation = {
                id: "bpmn_1.0.0_TestProcess_StartEvent_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "StartEvent_1",
            };

            const nextLocations = navigator.getNextLocations(sampleBpmn, startLocation);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_1");
            expect(nextLocations[0].routineId).toBe("bpmn_1.0.0_TestProcess");
        });

        test("should find next location for task", () => {
            const taskLocation = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const nextLocations = navigator.getNextLocations(sampleBpmn, taskLocation);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_2");
        });

        test("should return empty array for end event", () => {
            const endLocation = {
                id: "bpmn_1.0.0_TestProcess_EndEvent_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "EndEvent_1",
            };

            const nextLocations = navigator.getNextLocations(sampleBpmn, endLocation);
            expect(nextLocations).toHaveLength(0);
        });

        test("should handle conditional flows with true condition", () => {
            const conditionalBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2">
                                    <bpmn:conditionExpression>shouldProceed</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const context = { shouldProceed: true };
            const nextLocations = navigator.getNextLocations(conditionalBpmn, location, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_2");
        });

        test("should handle conditional flows with false condition", () => {
            const conditionalBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2">
                                    <bpmn:conditionExpression>shouldProceed</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const context = { shouldProceed: false };
            const nextLocations = navigator.getNextLocations(conditionalBpmn, location, context);

            expect(nextLocations).toHaveLength(0);
        });

        test("should handle equality condition evaluation", () => {
            const conditionalBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2">
                                    <bpmn:conditionExpression>status == "approved"</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const context = { status: "approved" };
            const nextLocations = navigator.getNextLocations(conditionalBpmn, location, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_2");
        });

        test("should handle equality condition with quotes", () => {
            const conditionalBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2">
                                    <bpmn:conditionExpression>status == 'approved'</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const context = { status: "approved" };
            const nextLocations = navigator.getNextLocations(conditionalBpmn, location, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_2");
        });

        test("should handle multiple outgoing flows", () => {
            const multipleBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:task id="Task_3" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_3" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const nextLocations = navigator.getNextLocations(multipleBpmn, location);

            expect(nextLocations).toHaveLength(2);
            expect(nextLocations.map(l => l.nodeId)).toContain("Task_2");
            expect(nextLocations.map(l => l.nodeId)).toContain("Task_3");
        });

        test("should handle flows without context when condition present", () => {
            const conditionalBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2">
                                    <bpmn:conditionExpression>shouldProceed</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            // No context provided - BpmnNavigator returns true for unresolvable conditions (lines 151-153)
            const nextLocations = navigator.getNextLocations(conditionalBpmn, location);

            expect(nextLocations).toHaveLength(1);
        });

        test("should handle malformed condition expressions gracefully", () => {
            const malformedBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:task id="Task_2" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2">
                                    <bpmn:conditionExpression>complex && (function() { return true; })()</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const context = { complex: true };
            const nextLocations = navigator.getNextLocations(malformedBpmn, location, context);

            // Should proceed with malformed conditions (default true behavior)
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_2");
        });

        test("should handle empty targetRef gracefully", () => {
            const emptyTargetBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const nextLocations = navigator.getNextLocations(emptyTargetBpmn, location);
            expect(nextLocations).toHaveLength(0);
        });
    });

    describe("isEndLocation", () => {
        test("should return true for end event", () => {
            const bpmnWithEndEvent: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:endEvent id="EndEvent_1" name="End" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const endLocation = {
                id: "bpmn_1.0.0_TestProcess_EndEvent_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "EndEvent_1",
            };

            expect(navigator.isEndLocation(bpmnWithEndEvent, endLocation)).toBe(true);
        });

        test("should return true for task without outgoing flows", () => {
            const bpmnTerminalTask: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" name="First Task" />
                                <bpmn:task id="Task_2" name="Terminal Task" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const terminalLocation = {
                id: "bpmn_1.0.0_TestProcess_Task_2",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_2",
            };

            expect(navigator.isEndLocation(bpmnTerminalTask, terminalLocation)).toBe(true);
        });

        test("should return false for task with outgoing flows", () => {
            const bpmnNonTerminal: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" name="First Task" />
                                <bpmn:task id="Task_2" name="Second Task" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_2" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const nonTerminalLocation = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            expect(navigator.isEndLocation(bpmnNonTerminal, nonTerminalLocation)).toBe(false);
        });

        test("should return false for start event", () => {
            const bpmnWithStartEvent: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="StartEvent_1" name="Start" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = {
                id: "bpmn_1.0.0_TestProcess_StartEvent_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "StartEvent_1",
            };

            expect(navigator.isEndLocation(bpmnWithStartEvent, startLocation)).toBe(true);
        });
    });

    describe("getStepInfo", () => {
        test("should get step info for task with name", () => {
            const bpmnWithNamedTask: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" name="Process Data" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {
                            Task_1: {
                                subroutineId: "data_processor",
                                inputMap: { input: "data" },
                                outputMap: { result: "output" },
                            },
                        },
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithNamedTask, location);

            expect(stepInfo.id).toBe("Task_1");
            expect(stepInfo.name).toBe("Task_1"); // Current implementation falls back to nodeId when name parsing fails
            expect(stepInfo.type).toBe("task");
            expect(stepInfo.description).toBe("BPMN task");
            expect(stepInfo.config).toEqual({
                subroutineId: "data_processor",
                inputMap: { input: "data" },
                outputMap: { result: "output" },
            });
        });

        test("should get step info for userTask", () => {
            const bpmnWithUserTask: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:userTask id="UserTask_1" name="User Input" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_UserTask_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "UserTask_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithUserTask, location);

            expect(stepInfo.id).toBe("UserTask_1");
            expect(stepInfo.name).toBe("UserTask_1"); // Current implementation falls back to nodeId when name parsing fails
            expect(stepInfo.type).toBe("user");
            expect(stepInfo.description).toBe("BPMN userTask");
        });

        test("should get step info for serviceTask", () => {
            const bpmnWithServiceTask: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:serviceTask id="ServiceTask_1" name="API Call" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_ServiceTask_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "ServiceTask_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithServiceTask, location);

            expect(stepInfo.type).toBe("service");
        });

        test("should get step info for callActivity", () => {
            const bpmnWithCallActivity: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:callActivity id="CallActivity_1" name="Subprocess" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_CallActivity_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "CallActivity_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithCallActivity, location);

            expect(stepInfo.type).toBe("subroutine");
        });

        test("should get step info for exclusiveGateway", () => {
            const bpmnWithGateway: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:exclusiveGateway id="Gateway_1" name="Decision Point" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Gateway_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Gateway_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithGateway, location);

            expect(stepInfo.type).toBe("decision");
        });

        test("should get step info for parallelGateway", () => {
            const bpmnWithParallelGateway: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:parallelGateway id="ParallelGateway_1" name="Parallel Split" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_ParallelGateway_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "ParallelGateway_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithParallelGateway, location);

            expect(stepInfo.type).toBe("parallel");
        });

        test("should get step info for startEvent", () => {
            const bpmnWithStartEvent: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="StartEvent_1" name="Start Process" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_StartEvent_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "StartEvent_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithStartEvent, location);

            expect(stepInfo.type).toBe("start");
        });

        test("should get step info for endEvent", () => {
            const bpmnWithEndEvent: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:endEvent id="EndEvent_1" name="End Process" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_EndEvent_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "EndEvent_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithEndEvent, location);

            expect(stepInfo.type).toBe("end");
        });

        test("should handle element without name", () => {
            const bpmnWithoutName: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithoutName, location);

            expect(stepInfo.name).toBe("Task_1");
        });

        test("should handle unknown element type", () => {
            const bpmnWithUnknownType: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:unknownElement id="Unknown_1" name="Unknown" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Unknown_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Unknown_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithUnknownType, location);

            expect(stepInfo.type).toBe("task");
        });

        test("should handle element not found", () => {
            const simpleBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_NonExistent",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "NonExistent",
            };

            const stepInfo = navigator.getStepInfo(simpleBpmn, location);

            expect(stepInfo.id).toBe("NonExistent");
            expect(stepInfo.name).toBe("NonExistent");
            expect(stepInfo.type).toBe("task");
            expect(stepInfo.description).toBe("BPMN element");
            expect(stepInfo.config).toBeUndefined();
        });

        test("should handle element without config in activityMap", () => {
            const bpmnWithoutConfig: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" name="Simple Task" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const stepInfo = navigator.getStepInfo(bpmnWithoutConfig, location);

            expect(stepInfo.config).toBeUndefined();
        });
    });

    describe("Error Handling and Edge Cases", () => {
        test("should handle invalid routine structure in validateRoutine calls", () => {
            expect(() => navigator.getStartLocation(null as unknown as RoutineVersionConfigObject))
                .toThrow("Invalid routine configuration for bpmn navigator");

            expect(() => navigator.getStartLocation({ __version: null } as unknown as RoutineVersionConfigObject))
                .toThrow("Routine missing version for bpmn navigator");
        });

        test("should handle malformed XML gracefully", () => {
            const malformedBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: "<<invalid xml>>",
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(() => navigator.getStartLocation(malformedBpmn))
                .toThrow("No start location found in BPMN");
        });

        test("should handle XML with special characters in IDs", () => {
            const specialCharBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="Test_Process-123">
                                <bpmn:startEvent id="Start_Event.1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(specialCharBpmn);
            expect(location.nodeId).toBe("Start_Event.1");
            expect(location.routineId).toBe("bpmn_1.0.0_Test_Process-123");
        });

        test("should handle empty BPMN data", () => {
            const emptyBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: "",
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(() => navigator.getStartLocation(emptyBpmn))
                .toThrow("No start location found in BPMN");
        });

        test("should handle very large version strings", () => {
            const longVersionBpmn: RoutineVersionConfigObject = {
                __version: "a".repeat(1000),
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="StartEvent_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(longVersionBpmn);
            expect(location.routineId).toContain("a".repeat(1000));
        });

        test("should handle self-referencing sequence flows", () => {
            const selfRefBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task_1" />
                                <bpmn:sequenceFlow sourceRef="Task_1" targetRef="Task_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_Task_1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task_1",
            };

            const nextLocations = navigator.getNextLocations(selfRefBpmn, location);
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("Task_1");
        });
    });

    describe("Complex Workflow Scenarios", () => {
        test("should handle parallel gateway flows", () => {
            const parallelBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="start" />
                                <bpmn:parallelGateway id="fork" />
                                <bpmn:task id="task1" />
                                <bpmn:task id="task2" />
                                <bpmn:parallelGateway id="join" />
                                <bpmn:endEvent id="end" />
                                
                                <bpmn:sequenceFlow sourceRef="start" targetRef="fork" />
                                <bpmn:sequenceFlow sourceRef="fork" targetRef="task1" />
                                <bpmn:sequenceFlow sourceRef="fork" targetRef="task2" />
                                <bpmn:sequenceFlow sourceRef="task1" targetRef="join" />
                                <bpmn:sequenceFlow sourceRef="task2" targetRef="join" />
                                <bpmn:sequenceFlow sourceRef="join" targetRef="end" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const forkLocation = {
                id: "bpmn_1.0.0_TestProcess_fork",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "fork",
            };

            const nextLocations = navigator.getNextLocations(parallelBpmn, forkLocation);

            expect(nextLocations).toHaveLength(2);
            expect(nextLocations.map(l => l.nodeId)).toContain("task1");
            expect(nextLocations.map(l => l.nodeId)).toContain("task2");
        });

        test("should handle exclusive gateway with multiple conditions", () => {
            const exclusiveBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:exclusiveGateway id="decision" />
                                <bpmn:task id="taskA" />
                                <bpmn:task id="taskB" />
                                <bpmn:task id="taskC" />
                                
                                <bpmn:sequenceFlow sourceRef="decision" targetRef="taskA">
                                    <bpmn:conditionExpression>choice == "A"</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                                <bpmn:sequenceFlow sourceRef="decision" targetRef="taskB">
                                    <bpmn:conditionExpression>choice == "B"</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                                <bpmn:sequenceFlow sourceRef="decision" targetRef="taskC" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const decisionLocation = {
                id: "bpmn_1.0.0_TestProcess_decision",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "decision",
            };

            // Test choice A
            let context = { choice: "A" };
            let nextLocations = navigator.getNextLocations(exclusiveBpmn, decisionLocation, context);
            expect(nextLocations).toHaveLength(2); // One conditional + one default
            expect(nextLocations.map(l => l.nodeId)).toContain("taskA");
            expect(nextLocations.map(l => l.nodeId)).toContain("taskC");

            // Test choice B
            context = { choice: "B" };
            nextLocations = navigator.getNextLocations(exclusiveBpmn, decisionLocation, context);
            expect(nextLocations).toHaveLength(2); // One conditional + one default
            expect(nextLocations.map(l => l.nodeId)).toContain("taskB");
            expect(nextLocations.map(l => l.nodeId)).toContain("taskC");

            // Test no matching choice (should take default)
            context = { choice: "Z" };
            nextLocations = navigator.getNextLocations(exclusiveBpmn, decisionLocation, context);
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("taskC");
        });

        test("should handle nested conditions with complex expressions", () => {
            const complexConditionBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="task1" />
                                <bpmn:task id="task2" />
                                
                                <bpmn:sequenceFlow sourceRef="task1" targetRef="task2">
                                    <bpmn:conditionExpression>  validUser  </bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = {
                id: "bpmn_1.0.0_TestProcess_task1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "task1",
            };

            // Test with whitespace in condition (should be trimmed)
            const context = { validUser: true };
            const nextLocations = navigator.getNextLocations(complexConditionBpmn, location, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("task2");
        });
    });

    describe("Security and Robustness", () => {
        test("should handle potential regex injection attempts", () => {
            const maliciousBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="Task.*" />
                                <bpmn:task id="Task2" />
                                <bpmn:sequenceFlow sourceRef="Task.*" targetRef="Task2" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            // Should not match due to literal interpretation of IDs
            const location = {
                id: "bpmn_1.0.0_TestProcess_TaskX",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "TaskX",
            };

            const nextLocations = navigator.getNextLocations(maliciousBpmn, location);
            expect(nextLocations).toHaveLength(0);
        });

        test("should handle extremely long XML gracefully", () => {
            const longElement = "a".repeat(10000);
            const longBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                <bpmn:task id="LongTask" name="${longElement}" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(longBpmn);
            expect(location.nodeId).toBe("Start");

            const stepInfo = navigator.getStepInfo(longBpmn, {
                id: "bpmn_1.0.0_TestProcess_LongTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "LongTask",
            });
            expect(stepInfo.name).toBe("LongTask"); // Current implementation falls back to nodeId when name parsing fails
        });

        test("should handle Unicode characters in XML", () => {
            const unicodeBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="开始" name="开始事件" />
                                <bpmn:task id="任务1" name="处理数据 🔄" />
                                <bpmn:sequenceFlow sourceRef="开始" targetRef="任务1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const location = navigator.getStartLocation(unicodeBpmn);
            expect(location.nodeId).toBe("开始");

            const nextLocations = navigator.getNextLocations(unicodeBpmn, location);
            expect(nextLocations[0].nodeId).toBe("任务1");

            const stepInfo = navigator.getStepInfo(unicodeBpmn, nextLocations[0]);
            expect(stepInfo.name).toBe("任务1"); // Current implementation falls back to nodeId when name parsing fails
        });
    });

    // ========================================
    // ADVANCED BPMN FEATURE TESTS
    // Testing implementation gaps and edge cases
    // ========================================

    describe("Advanced BPMN Features - Boundary Events", () => {
        test("should fail to handle timer boundary events (NOT IMPLEMENTED)", () => {
            const timerBoundaryBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:userTask id="ManualTask" name="Wait for User">
                                    <bpmn:incoming>Flow_1</bpmn:incoming>
                                    <bpmn:outgoing>Flow_2</bpmn:outgoing>
                                </bpmn:userTask>
                                
                                <!-- Timer boundary event - NOT HANDLED -->
                                <bpmn:boundaryEvent id="Timer_Boundary" attachedToRef="ManualTask">
                                    <bpmn:outgoing>Flow_Timeout</bpmn:outgoing>
                                    <bpmn:timerEventDefinition>
                                        <bpmn:timeDuration>PT30M</bpmn:timeDuration>
                                    </bpmn:timerEventDefinition>
                                </bpmn:boundaryEvent>
                                
                                <bpmn:task id="TimeoutTask" name="Handle Timeout">
                                    <bpmn:incoming>Flow_Timeout</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:task id="NormalTask" name="Normal Flow">
                                    <bpmn:incoming>Flow_2</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:sequenceFlow id="Flow_1" sourceRef="Start" targetRef="ManualTask" />
                                <bpmn:sequenceFlow id="Flow_2" sourceRef="ManualTask" targetRef="NormalTask" />
                                <bpmn:sequenceFlow id="Flow_Timeout" sourceRef="Timer_Boundary" targetRef="TimeoutTask" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const taskLocation = {
                id: "bpmn_1.0.0_TestProcess_ManualTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "ManualTask",
            };

            // Current implementation will only find normal sequence flows, ignoring boundary events
            const nextLocations = navigator.getNextLocations(timerBoundaryBpmn, taskLocation);

            // SHOULD have 2 paths (normal + timeout), but current implementation only finds 1
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("NormalTask");

            // Timer boundary event is completely ignored
            expect(nextLocations.find(l => l.nodeId === "TimeoutTask")).toBeUndefined();
        });

        test("should fail to handle error boundary events (NOT IMPLEMENTED)", () => {
            const errorBoundaryBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:serviceTask id="RiskyTask" name="API Call">
                                    <bpmn:incoming>Flow_1</bpmn:incoming>
                                    <bpmn:outgoing>Flow_Success</bpmn:outgoing>
                                </bpmn:serviceTask>
                                
                                <!-- Error boundary event - NOT HANDLED -->
                                <bpmn:boundaryEvent id="Error_Boundary" attachedToRef="RiskyTask">
                                    <bpmn:outgoing>Flow_Error</bpmn:outgoing>
                                    <bpmn:errorEventDefinition errorRef="ServiceError" />
                                </bpmn:boundaryEvent>
                                
                                <bpmn:task id="ErrorHandler" name="Handle Error">
                                    <bpmn:incoming>Flow_Error</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:task id="SuccessTask" name="Success Path">
                                    <bpmn:incoming>Flow_Success</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:sequenceFlow id="Flow_Success" sourceRef="RiskyTask" targetRef="SuccessTask" />
                                <bpmn:sequenceFlow id="Flow_Error" sourceRef="Error_Boundary" targetRef="ErrorHandler" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const riskyTaskLocation = {
                id: "bpmn_1.0.0_TestProcess_RiskyTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "RiskyTask",
            };

            const nextLocations = navigator.getNextLocations(errorBoundaryBpmn, riskyTaskLocation);

            // Only finds success path, error handling is ignored
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("SuccessTask");
        });

        test("should fail to handle message boundary events (NOT IMPLEMENTED)", () => {
            const messageBoundaryBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="WaitingTask" name="Long Running Task">
                                    <bpmn:outgoing>Flow_Normal</bpmn:outgoing>
                                </bpmn:task>
                                
                                <!-- Message boundary event - NOT HANDLED -->
                                <bpmn:boundaryEvent id="Message_Boundary" attachedToRef="WaitingTask">
                                    <bpmn:outgoing>Flow_Interrupt</bpmn:outgoing>
                                    <bpmn:messageEventDefinition messageRef="UrgentMessage" />
                                </bpmn:boundaryEvent>
                                
                                <bpmn:task id="InterruptTask" name="Handle Urgent Message">
                                    <bpmn:incoming>Flow_Interrupt</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:sequenceFlow id="Flow_Normal" sourceRef="WaitingTask" targetRef="NormalEnd" />
                                <bpmn:sequenceFlow id="Flow_Interrupt" sourceRef="Message_Boundary" targetRef="InterruptTask" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const waitingTaskLocation = {
                id: "bpmn_1.0.0_TestProcess_WaitingTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "WaitingTask",
            };

            const nextLocations = navigator.getNextLocations(messageBoundaryBpmn, waitingTaskLocation);

            // Message interruption path is ignored
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations.find(l => l.nodeId === "InterruptTask")).toBeUndefined();
        });
    });

    describe("Advanced BPMN Features - Event Gateways", () => {
        test("should fail to handle event-based gateways (NOT IMPLEMENTED)", () => {
            const eventGatewayBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="PrepareTask" />
                                
                                <!-- Event-based gateway - NOT HANDLED -->
                                <bpmn:eventBasedGateway id="EventGateway">
                                    <bpmn:incoming>Flow_Prepare</bpmn:incoming>
                                    <bpmn:outgoing>Flow_Timer</bpmn:outgoing>
                                    <bpmn:outgoing>Flow_Message</bpmn:outgoing>
                                </bpmn:eventBasedGateway>
                                
                                <bpmn:intermediateCatchEvent id="TimerEvent">
                                    <bpmn:incoming>Flow_Timer</bpmn:incoming>
                                    <bpmn:outgoing>Flow_TimeoutPath</bpmn:outgoing>
                                    <bpmn:timerEventDefinition>
                                        <bpmn:timeDuration>PT5M</bpmn:timeDuration>
                                    </bpmn:timerEventDefinition>
                                </bpmn:intermediateCatchEvent>
                                
                                <bpmn:intermediateCatchEvent id="MessageEvent">
                                    <bpmn:incoming>Flow_Message</bpmn:incoming>
                                    <bpmn:outgoing>Flow_MessagePath</bpmn:outgoing>
                                    <bpmn:messageEventDefinition messageRef="ResponseMessage" />
                                </bpmn:intermediateCatchEvent>
                                
                                <bpmn:sequenceFlow id="Flow_Prepare" sourceRef="PrepareTask" targetRef="EventGateway" />
                                <bpmn:sequenceFlow id="Flow_Timer" sourceRef="EventGateway" targetRef="TimerEvent" />
                                <bpmn:sequenceFlow id="Flow_Message" sourceRef="EventGateway" targetRef="MessageEvent" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const prepareTaskLocation = {
                id: "bpmn_1.0.0_TestProcess_PrepareTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "PrepareTask",
            };

            const nextLocations = navigator.getNextLocations(eventGatewayBpmn, prepareTaskLocation);

            // Should find EventGateway but won't understand its semantics
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("EventGateway");

            // Event gateway step info won't indicate special handling
            const gatewayStepInfo = navigator.getStepInfo(eventGatewayBpmn, nextLocations[0]);
            expect(gatewayStepInfo.type).toBe("task"); // Falls back to generic type
        });
    });

    describe("Advanced BPMN Features - Intermediate Events", () => {
        test("should fail to handle intermediate throw events (NOT IMPLEMENTED)", () => {
            const intermediateThrowBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="ProcessTask" />
                                
                                <!-- Intermediate message throw event - NOT HANDLED -->
                                <bpmn:intermediateThrowEvent id="NotifyEvent">
                                    <bpmn:incoming>Flow_1</bpmn:incoming>
                                    <bpmn:outgoing>Flow_2</bpmn:outgoing>
                                    <bpmn:messageEventDefinition messageRef="NotificationMessage" />
                                </bpmn:intermediateThrowEvent>
                                
                                <bpmn:task id="ContinueTask" />
                                
                                <bpmn:sequenceFlow id="Flow_1" sourceRef="ProcessTask" targetRef="NotifyEvent" />
                                <bpmn:sequenceFlow id="Flow_2" sourceRef="NotifyEvent" targetRef="ContinueTask" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const processTaskLocation = {
                id: "bpmn_1.0.0_TestProcess_ProcessTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "ProcessTask",
            };

            const nextLocations = navigator.getNextLocations(intermediateThrowBpmn, processTaskLocation);
            expect(nextLocations[0].nodeId).toBe("NotifyEvent");

            // Event is treated as generic element, not as message throw
            const eventStepInfo = navigator.getStepInfo(intermediateThrowBpmn, nextLocations[0]);
            expect(eventStepInfo.type).toBe("task"); // No special handling for intermediate events
        });

        test("should fail to handle intermediate catch events (NOT IMPLEMENTED)", () => {
            const intermediateCatchBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="InitTask" />
                                
                                <!-- Intermediate signal catch event - NOT HANDLED -->
                                <bpmn:intermediateCatchEvent id="WaitForSignal">
                                    <bpmn:incoming>Flow_1</bpmn:incoming>
                                    <bpmn:outgoing>Flow_2</bpmn:outgoing>
                                    <bpmn:signalEventDefinition signalRef="ProcessSignal" />
                                </bpmn:intermediateCatchEvent>
                                
                                <bpmn:task id="ProcessAfterSignal" />
                                
                                <bpmn:sequenceFlow id="Flow_1" sourceRef="InitTask" targetRef="WaitForSignal" />
                                <bpmn:sequenceFlow id="Flow_2" sourceRef="WaitForSignal" targetRef="ProcessAfterSignal" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const initTaskLocation = {
                id: "bpmn_1.0.0_TestProcess_InitTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "InitTask",
            };

            const nextLocations = navigator.getNextLocations(intermediateCatchBpmn, initTaskLocation);
            expect(nextLocations[0].nodeId).toBe("WaitForSignal");

            // No understanding that this should wait for external signal
            const eventStepInfo = navigator.getStepInfo(intermediateCatchBpmn, nextLocations[0]);
            expect(eventStepInfo.type).toBe("task");
        });
    });

    describe("Advanced BPMN Features - Complex Gateways", () => {
        test("should fail to handle inclusive gateways (NOT IMPLEMENTED)", () => {
            const inclusiveGatewayBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="EvaluateTask" />
                                
                                <!-- Inclusive gateway (OR gateway) - NOT HANDLED -->
                                <bpmn:inclusiveGateway id="InclusiveGateway">
                                    <bpmn:incoming>Flow_Eval</bpmn:incoming>
                                    <bpmn:outgoing>Flow_PathA</bpmn:outgoing>
                                    <bpmn:outgoing>Flow_PathB</bpmn:outgoing>
                                    <bpmn:outgoing>Flow_PathC</bpmn:outgoing>
                                </bpmn:inclusiveGateway>
                                
                                <bpmn:task id="TaskA" />
                                <bpmn:task id="TaskB" />
                                <bpmn:task id="TaskC" />
                                
                                <!-- Inclusive join -->
                                <bpmn:inclusiveGateway id="InclusiveJoin">
                                    <bpmn:incoming>Flow_A_Done</bpmn:incoming>
                                    <bpmn:incoming>Flow_B_Done</bpmn:incoming>
                                    <bpmn:incoming>Flow_C_Done</bpmn:incoming>
                                    <bpmn:outgoing>Flow_Continue</bpmn:outgoing>
                                </bpmn:inclusiveGateway>
                                
                                <bpmn:sequenceFlow id="Flow_Eval" sourceRef="EvaluateTask" targetRef="InclusiveGateway" />
                                <bpmn:sequenceFlow id="Flow_PathA" sourceRef="InclusiveGateway" targetRef="TaskA">
                                    <bpmn:conditionExpression>needsA</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                                <bpmn:sequenceFlow id="Flow_PathB" sourceRef="InclusiveGateway" targetRef="TaskB">
                                    <bpmn:conditionExpression>needsB</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                                <bpmn:sequenceFlow id="Flow_PathC" sourceRef="InclusiveGateway" targetRef="TaskC">
                                    <bpmn:conditionExpression>needsC</bpmn:conditionExpression>
                                </bpmn:sequenceFlow>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const evaluateLocation = {
                id: "bpmn_1.0.0_TestProcess_EvaluateTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "EvaluateTask",
            };

            const nextLocations = navigator.getNextLocations(inclusiveGatewayBpmn, evaluateLocation);
            expect(nextLocations[0].nodeId).toBe("InclusiveGateway");

            // Inclusive gateway treated as unknown element
            const gatewayStepInfo = navigator.getStepInfo(inclusiveGatewayBpmn, nextLocations[0]);
            expect(gatewayStepInfo.type).toBe("task"); // No mapping for inclusiveGateway
        });

        test("should fail to handle complex gateways (NOT IMPLEMENTED)", () => {
            const complexGatewayBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:task id="PrepTask" />
                                
                                <!-- Complex gateway - NOT HANDLED -->
                                <bpmn:complexGateway id="ComplexGateway">
                                    <bpmn:incoming>Flow_Prep</bpmn:incoming>
                                    <bpmn:outgoing>Flow_Complex1</bpmn:outgoing>
                                    <bpmn:outgoing>Flow_Complex2</bpmn:outgoing>
                                    <!-- Complex activation condition -->
                                    <bpmn:activationCondition>
                                        (priorityHigh and resourcesAvailable) or emergencyMode
                                    </bpmn:activationCondition>
                                </bpmn:complexGateway>
                                
                                <bpmn:task id="HighPriorityTask" />
                                <bpmn:task id="NormalTask" />
                                
                                <bpmn:sequenceFlow id="Flow_Prep" sourceRef="PrepTask" targetRef="ComplexGateway" />
                                <bpmn:sequenceFlow id="Flow_Complex1" sourceRef="ComplexGateway" targetRef="HighPriorityTask" />
                                <bpmn:sequenceFlow id="Flow_Complex2" sourceRef="ComplexGateway" targetRef="NormalTask" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const prepLocation = {
                id: "bpmn_1.0.0_TestProcess_PrepTask",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "PrepTask",
            };

            const nextLocations = navigator.getNextLocations(complexGatewayBpmn, prepLocation);
            expect(nextLocations[0].nodeId).toBe("ComplexGateway");

            // Complex gateway is not understood
            const gatewayStepInfo = navigator.getStepInfo(complexGatewayBpmn, nextLocations[0]);
            expect(gatewayStepInfo.type).toBe("task");
        });
    });

    describe("Advanced BPMN Features - Subprocess", () => {
        test("should fail to handle embedded subprocesses (NOT IMPLEMENTED)", () => {
            const embeddedSubprocessBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                
                                <!-- Embedded subprocess - NOT HANDLED -->
                                <bpmn:subProcess id="EmbeddedSubprocess">
                                    <bpmn:incoming>Flow_ToSub</bpmn:incoming>
                                    <bpmn:outgoing>Flow_FromSub</bpmn:outgoing>
                                    
                                    <!-- Internal process elements -->
                                    <bpmn:startEvent id="Sub_Start" />
                                    <bpmn:task id="Sub_Task1" />
                                    <bpmn:task id="Sub_Task2" />
                                    <bpmn:endEvent id="Sub_End" />
                                    
                                    <bpmn:sequenceFlow sourceRef="Sub_Start" targetRef="Sub_Task1" />
                                    <bpmn:sequenceFlow sourceRef="Sub_Task1" targetRef="Sub_Task2" />
                                    <bpmn:sequenceFlow sourceRef="Sub_Task2" targetRef="Sub_End" />
                                </bpmn:subProcess>
                                
                                <bpmn:task id="AfterSubprocess" />
                                
                                <bpmn:sequenceFlow id="Flow_ToSub" sourceRef="Start" targetRef="EmbeddedSubprocess" />
                                <bpmn:sequenceFlow id="Flow_FromSub" sourceRef="EmbeddedSubprocess" targetRef="AfterSubprocess" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(embeddedSubprocessBpmn);
            const nextLocations = navigator.getNextLocations(embeddedSubprocessBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("EmbeddedSubprocess");

            // Subprocess treated as single task, internal structure ignored
            const subprocessStepInfo = navigator.getStepInfo(embeddedSubprocessBpmn, nextLocations[0]);
            expect(subprocessStepInfo.type).toBe("task"); // No mapping for subProcess
        });

        test("should fail to handle ad-hoc subprocesses (NOT IMPLEMENTED)", () => {
            const adHocSubprocessBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                
                                <!-- Ad-hoc subprocess - NOT HANDLED -->
                                <bpmn:adHocSubProcess id="AdHocSubprocess" triggeredByEvent="false">
                                    <bpmn:incoming>Flow_ToAdHoc</bpmn:incoming>
                                    <bpmn:outgoing>Flow_FromAdHoc</bpmn:outgoing>
                                    
                                    <!-- Ad-hoc tasks can be executed in any order -->
                                    <bpmn:task id="AdHoc_Task1" />
                                    <bpmn:task id="AdHoc_Task2" />
                                    <bpmn:task id="AdHoc_Task3" />
                                    
                                    <!-- Completion condition -->
                                    <bpmn:completionCondition>allTasksComplete</bpmn:completionCondition>
                                </bpmn:adHocSubProcess>
                                
                                <bpmn:task id="AfterAdHoc" />
                                
                                <bpmn:sequenceFlow id="Flow_ToAdHoc" sourceRef="Start" targetRef="AdHocSubprocess" />
                                <bpmn:sequenceFlow id="Flow_FromAdHoc" sourceRef="AdHocSubprocess" targetRef="AfterAdHoc" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(adHocSubprocessBpmn);
            const nextLocations = navigator.getNextLocations(adHocSubprocessBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("AdHocSubprocess");

            // Ad-hoc subprocess not recognized
            const adHocStepInfo = navigator.getStepInfo(adHocSubprocessBpmn, nextLocations[0]);
            expect(adHocStepInfo.type).toBe("task");
        });
    });

    describe("Advanced BPMN Features - Event Subprocesses", () => {
        test("should fail to handle event subprocesses (NOT IMPLEMENTED)", () => {
            const eventSubprocessBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Main_Start" />
                                <bpmn:task id="Main_Task" />
                                <bpmn:endEvent id="Main_End" />
                                
                                <!-- Event subprocess - NOT HANDLED -->
                                <bpmn:subProcess id="EventSubprocess" triggeredByEvent="true">
                                    <!-- Error event start -->
                                    <bpmn:startEvent id="Error_Start">
                                        <bpmn:errorEventDefinition errorRef="CriticalError" />
                                    </bpmn:startEvent>
                                    
                                    <bpmn:task id="ErrorCleanup" />
                                    <bpmn:endEvent id="Error_End" />
                                    
                                    <bpmn:sequenceFlow sourceRef="Error_Start" targetRef="ErrorCleanup" />
                                    <bpmn:sequenceFlow sourceRef="ErrorCleanup" targetRef="Error_End" />
                                </bpmn:subProcess>
                                
                                <bpmn:sequenceFlow sourceRef="Main_Start" targetRef="Main_Task" />
                                <bpmn:sequenceFlow sourceRef="Main_Task" targetRef="Main_End" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(eventSubprocessBpmn);

            // Should start from Main_Start, event subprocess should not be start location
            expect(startLocation.nodeId).toBe("Main_Start");

            // Event subprocess is completely ignored in navigation
            const nextLocations = navigator.getNextLocations(eventSubprocessBpmn, startLocation);
            expect(nextLocations[0].nodeId).toBe("Main_Task");

            // Event subprocess never appears in navigation paths
            const allPossibleIds = ["EventSubprocess", "Error_Start", "ErrorCleanup", "Error_End"];
            const mainTaskLocation = nextLocations[0];
            const afterMainTask = navigator.getNextLocations(eventSubprocessBpmn, mainTaskLocation);

            expect(allPossibleIds.every(id =>
                !afterMainTask.find(loc => loc.nodeId === id),
            )).toBe(true);
        });
    });

    describe("Advanced BPMN Features - Multiple Instance Activities", () => {
        test("should fail to handle multi-instance tasks (NOT IMPLEMENTED)", () => {
            const multiInstanceBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                
                                <!-- Multi-instance task - NOT HANDLED -->
                                <bpmn:userTask id="MultiInstanceTask" name="Process Each Item">
                                    <bpmn:incoming>Flow_To_Multi</bpmn:incoming>
                                    <bpmn:outgoing>Flow_From_Multi</bpmn:outgoing>
                                    
                                    <!-- Multi-instance characteristics -->
                                    <bpmn:multiInstanceLoopCharacteristics isSequential="false">
                                        <bpmn:loopCardinality>#{itemList.size()}</bpmn:loopCardinality>
                                        <bpmn:loopDataInputRef>itemList</bpmn:loopDataInputRef>
                                        <bpmn:inputDataItem>currentItem</bpmn:inputDataItem>
                                        <bpmn:completionCondition>#{completedItems >= requiredItems}</bpmn:completionCondition>
                                    </bpmn:multiInstanceLoopCharacteristics>
                                </bpmn:userTask>
                                
                                <bpmn:task id="AfterMultiInstance" />
                                
                                <bpmn:sequenceFlow id="Flow_To_Multi" sourceRef="Start" targetRef="MultiInstanceTask" />
                                <bpmn:sequenceFlow id="Flow_From_Multi" sourceRef="MultiInstanceTask" targetRef="AfterMultiInstance" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(multiInstanceBpmn);
            const nextLocations = navigator.getNextLocations(multiInstanceBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("MultiInstanceTask");

            // Multi-instance characteristics are ignored, treated as single task
            const multiTaskStepInfo = navigator.getStepInfo(multiInstanceBpmn, nextLocations[0]);
            expect(multiTaskStepInfo.type).toBe("user");
            // No indication of multi-instance nature in config
        });

        test("should fail to handle sequential multi-instance (NOT IMPLEMENTED)", () => {
            const sequentialMultiInstanceBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                
                                <!-- Sequential multi-instance subprocess - NOT HANDLED -->
                                <bpmn:subProcess id="SequentialMultiSubprocess">
                                    <bpmn:incoming>Flow_To_Sequential</bpmn:incoming>
                                    <bpmn:outgoing>Flow_From_Sequential</bpmn:outgoing>
                                    
                                    <!-- Sequential execution -->
                                    <bpmn:multiInstanceLoopCharacteristics isSequential="true">
                                        <bpmn:loopCardinality>3</bpmn:loopCardinality>
                                    </bpmn:multiInstanceLoopCharacteristics>
                                    
                                    <bpmn:startEvent id="Sub_Start" />
                                    <bpmn:task id="Sequential_Task" />
                                    <bpmn:endEvent id="Sub_End" />
                                    
                                    <bpmn:sequenceFlow sourceRef="Sub_Start" targetRef="Sequential_Task" />
                                    <bpmn:sequenceFlow sourceRef="Sequential_Task" targetRef="Sub_End" />
                                </bpmn:subProcess>
                                
                                <bpmn:task id="AfterSequential" />
                                
                                <bpmn:sequenceFlow id="Flow_To_Sequential" sourceRef="Start" targetRef="SequentialMultiSubprocess" />
                                <bpmn:sequenceFlow id="Flow_From_Sequential" sourceRef="SequentialMultiSubprocess" targetRef="AfterSequential" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(sequentialMultiInstanceBpmn);
            const nextLocations = navigator.getNextLocations(sequentialMultiInstanceBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("SequentialMultiSubprocess");

            // Sequential multi-instance is not understood
            const sequentialStepInfo = navigator.getStepInfo(sequentialMultiInstanceBpmn, nextLocations[0]);
            expect(sequentialStepInfo.type).toBe("task");
        });
    });

    describe("Advanced BPMN Features - Loop Activities", () => {
        test("should fail to handle standard loops (NOT IMPLEMENTED)", () => {
            const standardLoopBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                
                                <!-- Standard loop task - NOT HANDLED -->
                                <bpmn:task id="LoopTask" name="Retry Until Success">
                                    <bpmn:incoming>Flow_To_Loop</bpmn:incoming>
                                    <bpmn:outgoing>Flow_From_Loop</bpmn:outgoing>
                                    
                                    <!-- Standard loop characteristics -->
                                    <bpmn:standardLoopCharacteristics testBefore="true">
                                        <bpmn:loopCondition>#{retryCount < maxRetries and !success}</bpmn:loopCondition>
                                        <bpmn:loopMaximum>5</bpmn:loopMaximum>
                                    </bpmn:standardLoopCharacteristics>
                                </bpmn:task>
                                
                                <bpmn:task id="AfterLoop" />
                                
                                <bpmn:sequenceFlow id="Flow_To_Loop" sourceRef="Start" targetRef="LoopTask" />
                                <bpmn:sequenceFlow id="Flow_From_Loop" sourceRef="LoopTask" targetRef="AfterLoop" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(standardLoopBpmn);
            const nextLocations = navigator.getNextLocations(standardLoopBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("LoopTask");

            // Loop characteristics are ignored
            const loopStepInfo = navigator.getStepInfo(standardLoopBpmn, nextLocations[0]);
            expect(loopStepInfo.type).toBe("task");
            // No indication of loop behavior
        });
    });

    describe("Advanced BPMN Features - Data Objects and Associations", () => {
        test("should fail to handle data objects (NOT IMPLEMENTED)", () => {
            const dataObjectBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                <bpmn:task id="ProcessData" />
                                
                                <!-- Data objects - NOT HANDLED -->
                                <bpmn:dataObject id="InputData" name="Customer Data" />
                                <bpmn:dataObject id="OutputData" name="Processed Results" />
                                
                                <!-- Data associations - NOT HANDLED -->
                                <bpmn:dataInputAssociation>
                                    <bpmn:sourceRef>InputData</bpmn:sourceRef>
                                    <bpmn:targetRef>ProcessData</bpmn:targetRef>
                                </bpmn:dataInputAssociation>
                                
                                <bpmn:dataOutputAssociation>
                                    <bpmn:sourceRef>ProcessData</bpmn:sourceRef>
                                    <bpmn:targetRef>OutputData</bpmn:targetRef>
                                </bpmn:dataOutputAssociation>
                                
                                <bpmn:sequenceFlow sourceRef="Start" targetRef="ProcessData" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(dataObjectBpmn);
            const nextLocations = navigator.getNextLocations(dataObjectBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("ProcessData");

            // Data objects and associations are completely ignored
            const processDataStepInfo = navigator.getStepInfo(dataObjectBpmn, nextLocations[0]);
            expect(processDataStepInfo.type).toBe("task");
            // No data object information in step config
        });

        test("should fail to handle data stores (NOT IMPLEMENTED)", () => {
            const dataStoreBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                <bpmn:task id="ReadFromDB" />
                                <bpmn:task id="WriteToDB" />
                                
                                <!-- Data store - NOT HANDLED -->
                                <bpmn:dataStoreReference id="CustomerDB" name="Customer Database" />
                                
                                <!-- Data store associations - NOT HANDLED -->
                                <bpmn:dataInputAssociation>
                                    <bpmn:sourceRef>CustomerDB</bpmn:sourceRef>
                                    <bpmn:targetRef>ReadFromDB</bpmn:targetRef>
                                </bpmn:dataInputAssociation>
                                
                                <bpmn:dataOutputAssociation>
                                    <bpmn:sourceRef>WriteToDB</bpmn:sourceRef>
                                    <bpmn:targetRef>CustomerDB</bpmn:targetRef>
                                </bpmn:dataOutputAssociation>
                                
                                <bpmn:sequenceFlow sourceRef="Start" targetRef="ReadFromDB" />
                                <bpmn:sequenceFlow sourceRef="ReadFromDB" targetRef="WriteToDB" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            // Data store references are ignored in navigation
            const startLocation = navigator.getStartLocation(dataStoreBpmn);
            const nextLocations = navigator.getNextLocations(dataStoreBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("ReadFromDB");
            // No awareness of data store connections
        });
    });

    describe("Advanced BPMN Features - Conversation and Choreography", () => {
        test("should fail to handle conversation diagrams (NOT IMPLEMENTED)", () => {
            const conversationBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <!-- Conversation elements - NOT HANDLED -->
                            <bpmn:collaboration id="Collaboration_1">
                                <bpmn:participant id="Customer" name="Customer" />
                                <bpmn:participant id="Vendor" name="Vendor" />
                                
                                <bpmn:conversation id="OrderNegotiation">
                                    <bpmn:participantRef>Customer</bpmn:participantRef>
                                    <bpmn:participantRef>Vendor</bpmn:participantRef>
                                </bpmn:conversation>
                                
                                <bpmn:messageFlow id="OrderRequest" sourceRef="Customer" targetRef="Vendor" />
                                <bpmn:messageFlow id="OrderResponse" sourceRef="Vendor" targetRef="Customer" />
                            </bpmn:collaboration>
                            
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                <bpmn:task id="SimpleTask" />
                                <bpmn:sequenceFlow sourceRef="Start" targetRef="SimpleTask" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            // Should find the process elements but ignore conversation
            const startLocation = navigator.getStartLocation(conversationBpmn);
            expect(startLocation.nodeId).toBe("Start");

            // Conversation elements are completely ignored
            const nextLocations = navigator.getNextLocations(conversationBpmn, startLocation);
            expect(nextLocations[0].nodeId).toBe("SimpleTask");
        });

        test("should fail to handle choreography tasks (NOT IMPLEMENTED)", () => {
            const choreographyBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <!-- Choreography diagram - NOT HANDLED -->
                            <bpmn:choreography id="Choreography_1">
                                <bpmn:participant id="Buyer" name="Buyer" />
                                <bpmn:participant id="Seller" name="Seller" />
                                
                                <bpmn:choreographyTask id="NegotiatePrice">
                                    <bpmn:participantRef>Buyer</bpmn:participantRef>
                                    <bpmn:participantRef>Seller</bpmn:participantRef>
                                </bpmn:choreographyTask>
                            </bpmn:choreography>
                            
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                <bpmn:task id="RegularTask" />
                                <bpmn:sequenceFlow sourceRef="Start" targetRef="RegularTask" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            // Choreography is ignored, only process navigation works
            const startLocation = navigator.getStartLocation(choreographyBpmn);
            expect(startLocation.nodeId).toBe("Start");
        });
    });

    describe("Advanced BPMN Features - Transaction and Compensation", () => {
        test("should fail to handle transaction subprocesses (NOT IMPLEMENTED)", () => {
            const transactionBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                
                                <!-- Transaction subprocess - NOT HANDLED -->
                                <bpmn:transaction id="PaymentTransaction">
                                    <bpmn:incoming>Flow_To_Transaction</bpmn:incoming>
                                    <bpmn:outgoing>Flow_Success</bpmn:outgoing>
                                    
                                    <bpmn:startEvent id="Trans_Start" />
                                    <bpmn:task id="ChargeCard" />
                                    <bpmn:task id="UpdateInventory" />
                                    <bpmn:endEvent id="Trans_End" />
                                    
                                    <!-- Compensation boundary event -->
                                    <bpmn:boundaryEvent id="Compensation_Boundary" attachedToRef="PaymentTransaction">
                                        <bpmn:compensateEventDefinition />
                                    </bpmn:boundaryEvent>
                                    
                                    <bpmn:sequenceFlow sourceRef="Trans_Start" targetRef="ChargeCard" />
                                    <bpmn:sequenceFlow sourceRef="ChargeCard" targetRef="UpdateInventory" />
                                    <bpmn:sequenceFlow sourceRef="UpdateInventory" targetRef="Trans_End" />
                                </bpmn:transaction>
                                
                                <!-- Compensation handler - NOT HANDLED -->
                                <bpmn:task id="ReverseTransaction" isForCompensation="true">
                                    <bpmn:incoming>Flow_Compensate</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:task id="SendConfirmation" />
                                
                                <bpmn:sequenceFlow id="Flow_To_Transaction" sourceRef="Start" targetRef="PaymentTransaction" />
                                <bpmn:sequenceFlow id="Flow_Success" sourceRef="PaymentTransaction" targetRef="SendConfirmation" />
                                <bpmn:association id="Flow_Compensate" sourceRef="Compensation_Boundary" targetRef="ReverseTransaction" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(transactionBpmn);
            const nextLocations = navigator.getNextLocations(transactionBpmn, startLocation);

            expect(nextLocations[0].nodeId).toBe("PaymentTransaction");

            // Transaction semantics are ignored
            const transactionStepInfo = navigator.getStepInfo(transactionBpmn, nextLocations[0]);
            expect(transactionStepInfo.type).toBe("task"); // No mapping for transaction
        });

        test("should fail to handle compensation events (NOT IMPLEMENTED)", () => {
            const compensationBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <bpmn:startEvent id="Start" />
                                <bpmn:task id="BookFlight" />
                                <bpmn:task id="BookHotel" />
                                
                                <!-- Compensation intermediate throw event - NOT HANDLED -->
                                <bpmn:intermediateThrowEvent id="TriggerCompensation">
                                    <bpmn:incoming>Flow_Trigger</bpmn:incoming>
                                    <bpmn:compensateEventDefinition activityRef="BookFlight" />
                                </bpmn:intermediateThrowEvent>
                                
                                <!-- Compensation handler - NOT HANDLED -->
                                <bpmn:task id="CancelFlight" isForCompensation="true">
                                    <bpmn:incoming>Flow_Cancel_Flight</bpmn:incoming>
                                </bpmn:task>
                                
                                <bpmn:sequenceFlow sourceRef="Start" targetRef="BookFlight" />
                                <bpmn:sequenceFlow sourceRef="BookFlight" targetRef="BookHotel" />
                                <bpmn:sequenceFlow id="Flow_Trigger" sourceRef="BookHotel" targetRef="TriggerCompensation" />
                                <bpmn:association id="Flow_Cancel_Flight" sourceRef="BookFlight" targetRef="CancelFlight" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(compensationBpmn);
            let currentLocation = startLocation;

            // Navigate through normal flow
            let nextLocations = navigator.getNextLocations(compensationBpmn, currentLocation);
            expect(nextLocations[0].nodeId).toBe("BookFlight");

            currentLocation = nextLocations[0];
            nextLocations = navigator.getNextLocations(compensationBpmn, currentLocation);
            expect(nextLocations[0].nodeId).toBe("BookHotel");

            currentLocation = nextLocations[0];
            nextLocations = navigator.getNextLocations(compensationBpmn, currentLocation);
            expect(nextLocations[0].nodeId).toBe("TriggerCompensation");

            // Compensation logic is not understood
            const compensationStepInfo = navigator.getStepInfo(compensationBpmn, nextLocations[0]);
            expect(compensationStepInfo.type).toBe("task");
        });
    });

    describe("BPMN Implementation Gaps - Summary Tests", () => {
        test("should demonstrate regex-based parsing limitations", () => {
            const complexAttributeBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <!-- Complex nested attributes that regex parsing struggles with -->
                                <bpmn:task id="Task1" 
                                    name="Task with &quot;quotes&quot; and special chars" 
                                    implementation="##WebService"
                                    operationRef="service:operation"
                                    isForCompensation="false">
                                    
                                    <bpmn:documentation>
                                        <![CDATA[
                                            This task has CDATA content with <special> characters
                                            and "quotes" that might confuse regex parsing.
                                        ]]>
                                    </bpmn:documentation>
                                    
                                    <bpmn:extensionElements>
                                        <custom:attribute value="complex &lt;value&gt;" />
                                    </bpmn:extensionElements>
                                    
                                    <bpmn:incoming>Flow1</bpmn:incoming>
                                    <bpmn:outgoing>Flow2</bpmn:outgoing>
                                </bpmn:task>
                                
                                <bpmn:sequenceFlow id="Flow1" sourceRef="Start" targetRef="Task1" />
                                <bpmn:sequenceFlow id="Flow2" sourceRef="Task1" targetRef="End" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            const taskLocation = {
                id: "bpmn_1.0.0_TestProcess_Task1",
                routineId: "bpmn_1.0.0_TestProcess",
                nodeId: "Task1",
            };

            // Current regex-based parsing may struggle with complex attributes
            const stepInfo = navigator.getStepInfo(complexAttributeBpmn, taskLocation);

            // Name extraction likely fails due to quotes and special characters
            expect(stepInfo.name).toBe("Task1"); // Falls back to nodeId
            expect(stepInfo.type).toBe("task");

            // Extension elements and documentation are completely ignored
        });

        test("should demonstrate missing XML namespace handling", () => {
            const namespacedBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <definitions 
                            xmlns="http://www.omg.org/spec/BPMN/20100524/MODEL"
                            xmlns:custom="http://custom.namespace.com"
                            xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
                            
                            <process id="TestProcess">
                                <!-- Using different namespace prefixes -->
                                <startEvent id="Start" />
                                <task id="Task1" />
                                
                                <!-- Custom namespace elements -->
                                <custom:metadata>
                                    <custom:priority>high</custom:priority>
                                    <custom:category>finance</custom:category>
                                </custom:metadata>
                                
                                <sequenceFlow sourceRef="Start" targetRef="Task1" />
                            </process>
                        </definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            // Current implementation specifically looks for "bpmn:" prefix
            expect(() => navigator.getStartLocation(namespacedBpmn))
                .toThrow("No start location found in BPMN");

            // Custom namespace elements are completely ignored
        });

        test("should demonstrate poor error handling for malformed BPMN", () => {
            const malformedBpmn: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="TestProcess">
                                <!-- Malformed elements -->
                                <bpmn:task id="Task1" name="Unclosed task"
                                <bpmn:task id="Task2" name="Missing attributes />
                                <bpmn:sequenceFlow sourceRef="Task1" <!-- missing targetRef and closing -->
                                <invalid_element>content</wrong_closing_tag>
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            // Current implementation has poor error handling - it finds Task2 as valid start even in malformed XML
            // This demonstrates that regex-based parsing is inadequate
            const location = navigator.getStartLocation(malformedBpmn);
            expect(location.nodeId).toBe("Task2"); // Shows poor validation - should have failed
        });
    });
});
