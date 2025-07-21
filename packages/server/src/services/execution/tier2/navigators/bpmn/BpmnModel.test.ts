import { beforeEach, describe, expect, test } from "vitest";
import { BpmnModel } from "./BpmnModel.js";

describe("BpmnModel", () => {
    let model: BpmnModel;

    const validBpmnXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1" isExecutable="true">
            <bpmn:startEvent id="StartEvent_1">
                <bpmn:outgoing>Flow_1</bpmn:outgoing>
            </bpmn:startEvent>
            <bpmn:task id="Task_1">
                <bpmn:incoming>Flow_1</bpmn:incoming>
                <bpmn:outgoing>Flow_2</bpmn:outgoing>
            </bpmn:task>
            <bpmn:endEvent id="EndEvent_1">
                <bpmn:incoming>Flow_2</bpmn:incoming>
            </bpmn:endEvent>
            <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_1" />
            <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_1" targetRef="EndEvent_1" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(() => {
        model = new BpmnModel();
    });

    describe("constructor", () => {
        test("should initialize with default values", () => {
            expect(model).toBeDefined();
            expect(model.getElementById("nonexistent")).toBeNull();
        });
    });

    describe("parseXml", () => {
        test("should load valid BPMN XML", async () => {
            await model.parseXml(validBpmnXml);
            
            const startEvent = model.getElementById("StartEvent_1");
            expect(startEvent).toBeDefined();
            expect(startEvent.$type).toBe("bpmn:StartEvent");
        });

        test("should throw error for invalid XML", async () => {
            await expect(model.parseXml("invalid xml")).rejects.toThrow();
        });

        test("should throw error for empty XML", async () => {
            await expect(model.parseXml("")).rejects.toThrow();
        });
    });

    describe("getElementById", () => {
        beforeEach(async () => {
            await model.parseXml(validBpmnXml);
        });

        test("should find existing elements", () => {
            expect(model.getElementById("StartEvent_1")).toBeDefined();
            expect(model.getElementById("Task_1")).toBeDefined();
            expect(model.getElementById("EndEvent_1")).toBeDefined();
        });

        test("should return null for non-existent elements", () => {
            expect(model.getElementById("NonExistent")).toBeNull();
        });
    });

    describe("getOutgoingFlows", () => {
        beforeEach(async () => {
            await model.parseXml(validBpmnXml);
        });

        test("should get outgoing flows from start event", () => {
            const flows = model.getOutgoingFlows("StartEvent_1");
            expect(flows).toHaveLength(1);
            expect(flows[0].id).toBe("Flow_1");
            expect((flows[0].targetRef as any).id).toBe("Task_1");
        });

        test("should get outgoing flows from task", () => {
            const flows = model.getOutgoingFlows("Task_1");
            expect(flows).toHaveLength(1);
            expect(flows[0].id).toBe("Flow_2");
        });

        test("should return empty array for end event", () => {
            const flows = model.getOutgoingFlows("EndEvent_1");
            expect(flows).toHaveLength(0);
        });

        test("should return empty array for non-existent element", () => {
            const flows = model.getOutgoingFlows("NonExistent");
            expect(flows).toHaveLength(0);
        });
    });

    describe("getIncomingFlows", () => {
        beforeEach(async () => {
            await model.parseXml(validBpmnXml);
        });

        test("should get incoming flows to task", () => {
            const flows = model.getIncomingFlows("Task_1");
            expect(flows).toHaveLength(1);
            expect(flows[0].id).toBe("Flow_1");
            expect((flows[0].sourceRef as any).id).toBe("StartEvent_1");
        });

        test("should get incoming flows to end event", () => {
            const flows = model.getIncomingFlows("EndEvent_1");
            expect(flows).toHaveLength(1);
            expect(flows[0].id).toBe("Flow_2");
        });

        test("should return empty array for start event", () => {
            const flows = model.getIncomingFlows("StartEvent_1");
            expect(flows).toHaveLength(0);
        });
    });

    describe("getStartEvents", () => {
        test("should find start events in simple process", async () => {
            await model.parseXml(validBpmnXml);
            const startEvents = model.getStartEvents();
            expect(startEvents).toHaveLength(1);
            expect(startEvents[0].id).toBe("StartEvent_1");
        });

        test("should find multiple start events", async () => {
            const multiStartXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:startEvent id="Start1" />
                    <bpmn:startEvent id="Start2" />
                </bpmn:process>
            </bpmn:definitions>`;

            await model.parseXml(multiStartXml);
            const startEvents = model.getStartEvents();
            expect(startEvents).toHaveLength(2);
            expect(startEvents.map(e => e.id)).toContain("Start1");
            expect(startEvents.map(e => e.id)).toContain("Start2");
        });
    });

    describe("getAllProcesses", () => {
        test("should get all processes", async () => {
            await model.parseXml(validBpmnXml);
            const processes = model.getAllProcesses();
            expect(processes).toHaveLength(1);
            expect(processes[0].id).toBe("Process_1");
        });

        test("should handle multiple processes", async () => {
            const multiProcessXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1" />
                <bpmn:process id="Process_2" />
            </bpmn:definitions>`;

            await model.parseXml(multiProcessXml);
            const processes = model.getAllProcesses();
            expect(processes).toHaveLength(2);
            expect(processes.map(p => p.id)).toContain("Process_1");
            expect(processes.map(p => p.id)).toContain("Process_2");
        });
    });

    describe("createAbstractLocation", () => {
        test("should create abstract location with default type", () => {
            const location = model.createAbstractLocation("node1", "routine1");
            
            expect(location).toEqual({
                nodeId: "node1",
                routineId: "routine1",
                type: "node",
                metadata: {},
            });
        });

        test("should create abstract location with custom type and metadata", () => {
            const location = model.createAbstractLocation(
                "gateway1", 
                "routine1", 
                "gateway_evaluation",
                {
                    parentNodeId: "parent1",
                    metadata: { custom: "data" },
                },
            );
            
            expect(location.nodeId).toBe("gateway1");
            expect(location.type).toBe("gateway_evaluation");
            expect(location.metadata.parentNodeId).toBe("parent1");
            expect(location.metadata.metadata).toEqual({ custom: "data" });
        });
    });

    describe("complex BPMN structures", () => {
        test("should handle subprocess elements", async () => {
            const subprocessXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="SubProcess_1">
                        <bpmn:startEvent id="SubStart" />
                        <bpmn:endEvent id="SubEnd" />
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            await model.parseXml(subprocessXml);
            const subprocess = model.getElementById("SubProcess_1");
            expect(subprocess).toBeDefined();
            expect(subprocess.$type).toBe("bpmn:SubProcess");
            
            // Elements inside subprocess should also be findable
            expect(model.getElementById("SubStart")).toBeDefined();
            expect(model.getElementById("SubEnd")).toBeDefined();
        });

        test("should handle gateways", async () => {
            const gatewayXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:exclusiveGateway id="ExclusiveGW" />
                    <bpmn:parallelGateway id="ParallelGW" />
                    <bpmn:inclusiveGateway id="InclusiveGW" />
                </bpmn:process>
            </bpmn:definitions>`;

            await model.parseXml(gatewayXml);
            
            expect(model.getElementById("ExclusiveGW").$type).toBe("bpmn:ExclusiveGateway");
            expect(model.getElementById("ParallelGW").$type).toBe("bpmn:ParallelGateway");
            expect(model.getElementById("InclusiveGW").$type).toBe("bpmn:InclusiveGateway");
        });
    });
});
