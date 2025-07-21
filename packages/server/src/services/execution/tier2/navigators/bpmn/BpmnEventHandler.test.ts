import { beforeEach, describe, expect, test, vi } from "vitest";
import { BpmnEventHandler } from "./BpmnEventHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation } from "../../types.js";

describe("BpmnEventHandler", () => {
    let handler: BpmnEventHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const bpmnWithBoundaryEvents = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1">
            <bpmn:task id="Task_1">
                <bpmn:outgoing>Flow_1</bpmn:outgoing>
            </bpmn:task>
            <bpmn:boundaryEvent id="BoundaryTimer" attachedToRef="Task_1">
                <bpmn:timerEventDefinition>
                    <bpmn:timeDuration>PT5M</bpmn:timeDuration>
                </bpmn:timerEventDefinition>
                <bpmn:outgoing>TimerFlow</bpmn:outgoing>
            </bpmn:boundaryEvent>
            <bpmn:boundaryEvent id="BoundaryError" attachedToRef="Task_1" cancelActivity="false">
                <bpmn:errorEventDefinition errorRef="Error_1" />
                <bpmn:outgoing>ErrorFlow</bpmn:outgoing>
            </bpmn:boundaryEvent>
            <bpmn:task id="TimerHandler">
                <bpmn:incoming>TimerFlow</bpmn:incoming>
            </bpmn:task>
            <bpmn:task id="ErrorHandler">
                <bpmn:incoming>ErrorFlow</bpmn:incoming>
            </bpmn:task>
            <bpmn:task id="NextTask">
                <bpmn:incoming>Flow_1</bpmn:incoming>
            </bpmn:task>
            <bpmn:sequenceFlow id="Flow_1" sourceRef="Task_1" targetRef="NextTask" />
            <bpmn:sequenceFlow id="TimerFlow" sourceRef="BoundaryTimer" targetRef="TimerHandler" />
            <bpmn:sequenceFlow id="ErrorFlow" sourceRef="BoundaryError" targetRef="ErrorHandler" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnEventHandler();
        model = new BpmnModel();
        await model.loadXml(bpmnWithBoundaryEvents);

        mockLocation = {
            nodeId: "Task_1",
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

    describe("attachBoundaryEvents", () => {
        test("should attach boundary events to activity", () => {
            const result = handler.attachBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.boundaryEventsAttached).toHaveLength(2);
            expect(result.boundaryEventsAttached.map(e => e.eventId)).toContain("BoundaryTimer");
            expect(result.boundaryEventsAttached.map(e => e.eventId)).toContain("BoundaryError");
        });

        test("should identify interrupting vs non-interrupting events", () => {
            const result = handler.attachBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            const timerEvent = result.boundaryEventsAttached.find(e => e.eventId === "BoundaryTimer");
            const errorEvent = result.boundaryEventsAttached.find(e => e.eventId === "BoundaryError");
            
            expect(timerEvent?.interrupting).toBe(true); // Default is true
            expect(errorEvent?.interrupting).toBe(false); // Explicitly set to false
        });

        test("should update context with active boundary events", () => {
            const result = handler.attachBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.updatedContext.events.active).toHaveLength(2);
            expect(result.updatedContext.events.active[0].activityId).toBe("Task_1");
            expect(result.updatedContext.events.active[0].status).toBe("monitoring");
        });

        test("should return empty array for activity without boundary events", async () => {
            const simpleXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:task id="SimpleTask" />
                </bpmn:process>
            </bpmn:definitions>`;
            
            const simpleModel = new BpmnModel();
            await simpleModel.loadXml(simpleXml);
            
            const result = handler.attachBoundaryEvents(simpleModel, "SimpleTask", mockLocation, mockContext);
            expect(result.boundaryEventsAttached).toHaveLength(0);
        });
    });

    describe("checkBoundaryEvents", () => {
        beforeEach(() => {
            // Add active boundary events to context
            mockContext.events.active = [
                {
                    id: "boundary_1",
                    eventId: "BoundaryTimer",
                    activityId: "Task_1",
                    interrupting: true,
                    eventType: "timer",
                    status: "monitoring",
                    attachedAt: new Date(),
                },
                {
                    id: "boundary_2",
                    eventId: "BoundaryError",
                    activityId: "Task_1",
                    interrupting: false,
                    eventType: "error",
                    status: "monitoring",
                    attachedAt: new Date(),
                },
            ];
        });

        test("should trigger timer boundary event when timer expires", () => {
            // Add expired timer to context
            mockContext.events.timers = [{
                id: "timer_1",
                eventId: "BoundaryTimer",
                expiresAt: new Date(Date.now() - 1000), // Expired
                type: "duration",
            }];

            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.triggeredEvents).toHaveLength(1);
            expect(result.triggeredEvents[0].eventId).toBe("BoundaryTimer");
            expect(result.shouldInterruptActivity).toBe(true);
        });

        test("should trigger error boundary event on matching error", () => {
            // Add error to context
            mockContext.variables.errors = [{
                code: "Error_1",
                message: "Test error",
            }];

            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.triggeredEvents).toHaveLength(1);
            expect(result.triggeredEvents[0].eventId).toBe("BoundaryError");
            expect(result.shouldInterruptActivity).toBe(false); // Non-interrupting
        });

        test("should create correct next locations for triggered events", () => {
            mockContext.events.timers = [{
                id: "timer_1",
                eventId: "BoundaryTimer",
                expiresAt: new Date(Date.now() - 1000),
                type: "duration",
            }];

            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.nextLocations).toHaveLength(1);
            expect(result.nextLocations[0].nodeId).toBe("TimerHandler");
            expect(result.nextLocations[0].type).toBe("boundary_event_triggered");
        });

        test("should handle multiple triggered events", () => {
            // Add both timer and error
            mockContext.events.timers = [{
                id: "timer_1",
                eventId: "BoundaryTimer",
                expiresAt: new Date(Date.now() - 1000),
                type: "duration",
            }];
            mockContext.variables.errors = [{
                code: "Error_1",
                message: "Test error",
            }];

            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.triggeredEvents).toHaveLength(2);
            expect(result.nextLocations).toHaveLength(2);
            expect(result.shouldInterruptActivity).toBe(true); // At least one interrupting
        });

        test("should return no triggers when no events match", () => {
            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.triggeredEvents).toHaveLength(0);
            expect(result.nextLocations).toHaveLength(0);
            expect(result.shouldInterruptActivity).toBe(false);
        });
    });

    describe("event type detection", () => {
        test("should detect timer event types", async () => {
            const timerXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:task id="Task_1" />
                    <bpmn:boundaryEvent id="TimerEvent" attachedToRef="Task_1">
                        <bpmn:timerEventDefinition>
                            <bpmn:timeDuration>PT30S</bpmn:timeDuration>
                        </bpmn:timerEventDefinition>
                    </bpmn:boundaryEvent>
                </bpmn:process>
            </bpmn:definitions>`;

            const timerModel = new BpmnModel();
            await timerModel.loadXml(timerXml);
            
            const result = handler.attachBoundaryEvents(timerModel, "Task_1", mockLocation, mockContext);
            expect(result.boundaryEventsAttached[0].eventType).toBe("timer");
        });

        test("should detect message event types", async () => {
            const messageXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:task id="Task_1" />
                    <bpmn:boundaryEvent id="MessageEvent" attachedToRef="Task_1">
                        <bpmn:messageEventDefinition messageRef="Message_1" />
                    </bpmn:boundaryEvent>
                </bpmn:process>
            </bpmn:definitions>`;

            const messageModel = new BpmnModel();
            await messageModel.loadXml(messageXml);
            
            const result = handler.attachBoundaryEvents(messageModel, "Task_1", mockLocation, mockContext);
            expect(result.boundaryEventsAttached[0].eventType).toBe("message");
        });

        test("should detect signal event types", async () => {
            const signalXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:task id="Task_1" />
                    <bpmn:boundaryEvent id="SignalEvent" attachedToRef="Task_1">
                        <bpmn:signalEventDefinition signalRef="Signal_1" />
                    </bpmn:boundaryEvent>
                </bpmn:process>
            </bpmn:definitions>`;

            const signalModel = new BpmnModel();
            await signalModel.loadXml(signalXml);
            
            const result = handler.attachBoundaryEvents(signalModel, "Task_1", mockLocation, mockContext);
            expect(result.boundaryEventsAttached[0].eventType).toBe("signal");
        });
    });

    describe("completeBoundaryEvent", () => {
        test("should mark boundary event as completed", () => {
            const activeEvent = {
                id: "boundary_1",
                eventId: "BoundaryTimer",
                activityId: "Task_1",
                interrupting: true,
                eventType: "timer",
                status: "monitoring" as const,
                attachedAt: new Date(),
            };
            mockContext.events.active = [activeEvent];

            const result = handler.completeBoundaryEvent(mockContext, "BoundaryTimer");
            
            expect(result.events.active[0].status).toBe("completed");
        });

        test("should not affect other boundary events", () => {
            mockContext.events.active = [
                {
                    id: "boundary_1",
                    eventId: "Event1",
                    activityId: "Task_1",
                    interrupting: true,
                    eventType: "timer",
                    status: "monitoring",
                    attachedAt: new Date(),
                },
                {
                    id: "boundary_2",
                    eventId: "Event2",
                    activityId: "Task_1",
                    interrupting: false,
                    eventType: "error",
                    status: "monitoring",
                    attachedAt: new Date(),
                },
            ];

            const result = handler.completeBoundaryEvent(mockContext, "Event1");
            
            expect(result.events.active[0].status).toBe("completed");
            expect(result.events.active[1].status).toBe("monitoring");
        });
    });

    describe("detachBoundaryEvents", () => {
        test("should remove boundary events for completed activity", () => {
            mockContext.events.active = [
                {
                    id: "boundary_1",
                    eventId: "Event1",
                    activityId: "Task_1",
                    interrupting: true,
                    eventType: "timer",
                    status: "monitoring",
                    attachedAt: new Date(),
                },
                {
                    id: "boundary_2",
                    eventId: "Event2",
                    activityId: "Task_2",
                    interrupting: false,
                    eventType: "error",
                    status: "monitoring",
                    attachedAt: new Date(),
                },
            ];

            const result = handler.detachBoundaryEvents(mockContext, "Task_1");
            
            expect(result.events.active).toHaveLength(1);
            expect(result.events.active[0].activityId).toBe("Task_2");
        });

        test("should not remove completed boundary events", () => {
            mockContext.events.active = [
                {
                    id: "boundary_1",
                    eventId: "Event1",
                    activityId: "Task_1",
                    interrupting: true,
                    eventType: "timer",
                    status: "completed",
                    attachedAt: new Date(),
                    completedAt: new Date(),
                },
            ];

            const result = handler.detachBoundaryEvents(mockContext, "Task_1");
            
            expect(result.events.active).toHaveLength(1);
            expect(result.events.active[0].status).toBe("completed");
        });
    });

    describe("message event handling", () => {
        test("should trigger on matching message", () => {
            mockContext.events.active = [{
                id: "boundary_1",
                eventId: "MessageBoundary",
                activityId: "Task_1",
                interrupting: true,
                eventType: "message",
                status: "monitoring",
                attachedAt: new Date(),
                eventDefinition: { messageRef: "Order_Cancelled" },
            }];

            mockContext.external.messageEvents = [{
                id: "msg_1",
                messageRef: "Order_Cancelled",
                payload: { orderId: "123" },
                receivedAt: new Date(),
            }];

            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.triggeredEvents).toHaveLength(1);
            expect(result.triggeredEvents[0].eventId).toBe("MessageBoundary");
        });
    });

    describe("signal event handling", () => {
        test("should trigger on matching signal", () => {
            mockContext.events.active = [{
                id: "boundary_1",
                eventId: "SignalBoundary",
                activityId: "Task_1",
                interrupting: false,
                eventType: "signal",
                status: "monitoring",
                attachedAt: new Date(),
                eventDefinition: { signalRef: "System_Alert" },
            }];

            mockContext.external.signalEvents = [{
                id: "sig_1",
                signalRef: "System_Alert",
                payload: { severity: "high" },
            }];

            const result = handler.checkBoundaryEvents(model, "Task_1", mockLocation, mockContext);
            
            expect(result.triggeredEvents).toHaveLength(1);
            expect(result.triggeredEvents[0].eventId).toBe("SignalBoundary");
            expect(result.shouldInterruptActivity).toBe(false);
        });
    });
});
