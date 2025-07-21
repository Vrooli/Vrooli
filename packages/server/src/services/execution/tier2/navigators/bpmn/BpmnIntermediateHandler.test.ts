import { beforeEach, describe, expect, test } from "vitest";
import { BpmnIntermediateHandler } from "./BpmnIntermediateHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation } from "../../types.js";

describe("BpmnIntermediateHandler", () => {
    let handler: BpmnIntermediateHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const intermediateEventsXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1">
            <bpmn:task id="Task1">
                <bpmn:outgoing>ToThrow</bpmn:outgoing>
            </bpmn:task>
            <bpmn:intermediateThrowEvent id="ThrowEvent">
                <bpmn:incoming>ToThrow</bpmn:incoming>
                <bpmn:outgoing>FromThrow</bpmn:outgoing>
                <bpmn:messageEventDefinition messageRef="Message_1" />
            </bpmn:intermediateThrowEvent>
            <bpmn:intermediateCatchEvent id="CatchEvent">
                <bpmn:incoming>FromThrow</bpmn:incoming>
                <bpmn:outgoing>ToCatch</bpmn:outgoing>
                <bpmn:messageEventDefinition messageRef="Message_1" />
            </bpmn:intermediateCatchEvent>
            <bpmn:intermediateCatchEvent id="TimerEvent">
                <bpmn:timerEventDefinition>
                    <bpmn:timeDuration>PT5M</bpmn:timeDuration>
                </bpmn:timerEventDefinition>
                <bpmn:outgoing>FromTimer</bpmn:outgoing>
            </bpmn:intermediateCatchEvent>
            <bpmn:intermediateThrowEvent id="SignalThrow">
                <bpmn:signalEventDefinition signalRef="Signal_1" />
                <bpmn:outgoing>FromSignal</bpmn:outgoing>
            </bpmn:intermediateThrowEvent>
            <bpmn:task id="Task2">
                <bpmn:incoming>ToCatch</bpmn:incoming>
            </bpmn:task>
            <bpmn:sequenceFlow id="ToThrow" sourceRef="Task1" targetRef="ThrowEvent" />
            <bpmn:sequenceFlow id="FromThrow" sourceRef="ThrowEvent" targetRef="CatchEvent" />
            <bpmn:sequenceFlow id="ToCatch" sourceRef="CatchEvent" targetRef="Task2" />
            <bpmn:sequenceFlow id="FromTimer" sourceRef="TimerEvent" targetRef="Task2" />
            <bpmn:sequenceFlow id="FromSignal" sourceRef="SignalThrow" targetRef="Task2" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnIntermediateHandler();
        model = new BpmnModel();
        await model.loadXml(intermediateEventsXml);

        mockLocation = {
            nodeId: "ThrowEvent",
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

    describe("processIntermediateEvent", () => {
        describe("throw events", () => {
            test("should process message throw event", () => {
                const result = handler.processIntermediateEvent(
                    model, 
                    "ThrowEvent", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventThrown).toBeTruthy();
                expect(result.eventThrown?.type).toBe("message");
                expect(result.eventThrown?.payload.messageRef).toBe("Message_1");
            });

            test("should fire event and continue flow", () => {
                const result = handler.processIntermediateEvent(
                    model, 
                    "ThrowEvent", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.events.fired).toHaveLength(1);
                expect(result.updatedContext.events.fired[0].eventId).toBe("ThrowEvent");
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("CatchEvent");
            });

            test("should process signal throw event", () => {
                mockLocation.nodeId = "SignalThrow";
                const result = handler.processIntermediateEvent(
                    model, 
                    "SignalThrow", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventThrown?.type).toBe("signal");
                expect(result.eventThrown?.payload.signalRef).toBe("Signal_1");
                expect(result.updatedContext.external.signalEvents).toHaveLength(1);
            });

            test("should handle link throw events", async () => {
                const linkXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:intermediateThrowEvent id="LinkThrow">
                            <bpmn:linkEventDefinition name="LinkA" />
                            <bpmn:outgoing>Flow1</bpmn:outgoing>
                        </bpmn:intermediateThrowEvent>
                        <bpmn:intermediateCatchEvent id="LinkCatch">
                            <bpmn:linkEventDefinition name="LinkA" />
                            <bpmn:outgoing>Flow2</bpmn:outgoing>
                        </bpmn:intermediateCatchEvent>
                        <bpmn:task id="LinkedTask">
                            <bpmn:incoming>Flow2</bpmn:incoming>
                        </bpmn:task>
                        <bpmn:sequenceFlow id="Flow1" sourceRef="LinkThrow" targetRef="LinkedTask" />
                        <bpmn:sequenceFlow id="Flow2" sourceRef="LinkCatch" targetRef="LinkedTask" />
                    </bpmn:process>
                </bpmn:definitions>`;

                const linkModel = new BpmnModel();
                await linkModel.loadXml(linkXml);

                const result = handler.processIntermediateEvent(
                    linkModel, 
                    "LinkThrow", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventThrown?.type).toBe("link");
                expect(result.nextLocations[0].nodeId).toBe("LinkCatch");
            });
        });

        describe("catch events", () => {
            test("should wait for message catch event", () => {
                mockLocation.nodeId = "CatchEvent";
                const result = handler.processIntermediateEvent(
                    model, 
                    "CatchEvent", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.waitingForEvent).toBeTruthy();
                expect(result.waitingForEvent?.type).toBe("message");
                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].type).toBe("intermediate_waiting");
            });

            test("should trigger message catch event when message available", () => {
                mockLocation.nodeId = "CatchEvent";
                mockContext.external.messageEvents = [{
                    id: "msg_1",
                    messageRef: "Message_1",
                    payload: { data: "test" },
                    receivedAt: new Date(),
                }];

                const result = handler.processIntermediateEvent(
                    model, 
                    "CatchEvent", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventCaught).toBeTruthy();
                expect(result.eventCaught?.eventId).toBe("CatchEvent");
                expect(result.nextLocations[0].nodeId).toBe("Task2");
                expect(result.updatedContext.external.messageEvents).toHaveLength(0);
            });

            test("should set up timer for timer catch event", () => {
                mockLocation.nodeId = "TimerEvent";
                const result = handler.processIntermediateEvent(
                    model, 
                    "TimerEvent", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.waitingForEvent?.type).toBe("timer");
                expect(result.updatedContext.events.timers).toHaveLength(1);
                expect(result.updatedContext.events.timers[0].type).toBe("duration");
            });

            test("should trigger timer event when expired", () => {
                mockLocation.nodeId = "TimerEvent";
                mockContext.events.timers = [{
                    id: "timer_1",
                    eventId: "TimerEvent",
                    expiresAt: new Date(Date.now() - 1000), // Expired
                    type: "duration",
                }];

                const result = handler.processIntermediateEvent(
                    model, 
                    "TimerEvent", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventCaught).toBeTruthy();
                expect(result.nextLocations[0].nodeId).toBe("Task2");
                expect(result.updatedContext.events.timers).toHaveLength(0);
            });
        });

        describe("special event types", () => {
            test("should handle escalation events", async () => {
                const escalationXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:intermediateThrowEvent id="EscalationThrow">
                            <bpmn:escalationEventDefinition escalationRef="Escalation_1" />
                            <bpmn:outgoing>Flow1</bpmn:outgoing>
                        </bpmn:intermediateThrowEvent>
                        <bpmn:task id="Task">
                            <bpmn:incoming>Flow1</bpmn:incoming>
                        </bpmn:task>
                        <bpmn:sequenceFlow id="Flow1" sourceRef="EscalationThrow" targetRef="Task" />
                    </bpmn:process>
                </bpmn:definitions>`;

                const escalationModel = new BpmnModel();
                await escalationModel.loadXml(escalationXml);

                const result = handler.processIntermediateEvent(
                    escalationModel, 
                    "EscalationThrow", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventThrown?.type).toBe("escalation");
                expect(result.eventThrown?.payload.escalationCode).toBe("Escalation_1");
            });

            test("should handle compensation events", async () => {
                const compensationXml = `<?xml version="1.0" encoding="UTF-8"?>
                <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                    <bpmn:process id="Process_1">
                        <bpmn:intermediateThrowEvent id="CompensationThrow">
                            <bpmn:compensateEventDefinition activityRef="CompensableTask" />
                            <bpmn:outgoing>Flow1</bpmn:outgoing>
                        </bpmn:intermediateThrowEvent>
                        <bpmn:task id="Task">
                            <bpmn:incoming>Flow1</bpmn:incoming>
                        </bpmn:task>
                        <bpmn:sequenceFlow id="Flow1" sourceRef="CompensationThrow" targetRef="Task" />
                    </bpmn:process>
                </bpmn:definitions>`;

                const compensationModel = new BpmnModel();
                await compensationModel.loadXml(compensationXml);

                const result = handler.processIntermediateEvent(
                    compensationModel, 
                    "CompensationThrow", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.eventThrown?.type).toBe("compensation");
                expect(result.eventThrown?.payload.activityRef).toBe("CompensableTask");
            });
        });
    });

    describe("getPendingIntermediateEvents", () => {
        test("should return all pending events", () => {
            mockContext.events.pending = [
                {
                    id: "pending_1",
                    eventId: "Event1",
                    type: "message",
                    status: "waiting",
                    metadata: { messageRef: "Message_1" },
                },
                {
                    id: "pending_2",
                    eventId: "Event2",
                    type: "timer",
                    status: "waiting",
                },
            ];

            const pendingEvents = handler.getPendingIntermediateEvents(mockContext);
            
            expect(pendingEvents).toHaveLength(2);
            expect(pendingEvents.map(e => e.type)).toContain("message");
            expect(pendingEvents.map(e => e.type)).toContain("timer");
        });

        test("should filter by event type", () => {
            mockContext.events.pending = [
                {
                    id: "pending_1",
                    eventId: "Event1",
                    type: "message",
                    status: "waiting",
                },
                {
                    id: "pending_2",
                    eventId: "Event2",
                    type: "timer",
                    status: "waiting",
                },
                {
                    id: "pending_3",
                    eventId: "Event3",
                    type: "message",
                    status: "waiting",
                },
            ];

            const messageEvents = handler.getPendingIntermediateEvents(mockContext, "message");
            
            expect(messageEvents).toHaveLength(2);
            expect(messageEvents.every(e => e.type === "message")).toBe(true);
        });
    });

    describe("completeIntermediateEvent", () => {
        test("should complete pending event", () => {
            mockContext.events.pending = [
                {
                    id: "pending_1",
                    eventId: "Event1",
                    type: "message",
                    status: "waiting",
                },
            ];

            const result = handler.completeIntermediateEvent(mockContext, "Event1");
            
            expect(result.events.pending).toHaveLength(0);
        });

        test("should not affect other pending events", () => {
            mockContext.events.pending = [
                {
                    id: "pending_1",
                    eventId: "Event1",
                    type: "message",
                    status: "waiting",
                },
                {
                    id: "pending_2",
                    eventId: "Event2",
                    type: "timer",
                    status: "waiting",
                },
            ];

            const result = handler.completeIntermediateEvent(mockContext, "Event1");
            
            expect(result.events.pending).toHaveLength(1);
            expect(result.events.pending[0].eventId).toBe("Event2");
        });
    });

    describe("error handling", () => {
        test("should throw error for non-existent event", () => {
            expect(() => {
                handler.processIntermediateEvent(
                    model, 
                    "NonExistentEvent", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow("Intermediate event not found: NonExistentEvent");
        });

        test("should handle events without definitions gracefully", async () => {
            const noDefXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:intermediateThrowEvent id="NoDefEvent">
                        <bpmn:outgoing>Flow1</bpmn:outgoing>
                    </bpmn:intermediateThrowEvent>
                    <bpmn:task id="Task">
                        <bpmn:incoming>Flow1</bpmn:incoming>
                    </bpmn:task>
                    <bpmn:sequenceFlow id="Flow1" sourceRef="NoDefEvent" targetRef="Task" />
                </bpmn:process>
            </bpmn:definitions>`;

            const noDefModel = new BpmnModel();
            await noDefModel.loadXml(noDefXml);

            const result = handler.processIntermediateEvent(
                noDefModel, 
                "NoDefEvent", 
                mockLocation, 
                mockContext,
            );

            expect(result.eventThrown).toBeFalsy();
            expect(result.nextLocations[0].nodeId).toBe("Task");
        });
    });

    describe("conditional events", () => {
        test("should evaluate conditional catch events", async () => {
            const conditionalXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:intermediateCatchEvent id="ConditionalEvent">
                        <bpmn:conditionalEventDefinition>
                            <bpmn:condition>isReady</bpmn:condition>
                        </bpmn:conditionalEventDefinition>
                        <bpmn:outgoing>Flow1</bpmn:outgoing>
                    </bpmn:intermediateCatchEvent>
                    <bpmn:task id="Task">
                        <bpmn:incoming>Flow1</bpmn:incoming>
                    </bpmn:task>
                    <bpmn:sequenceFlow id="Flow1" sourceRef="ConditionalEvent" targetRef="Task" />
                </bpmn:process>
            </bpmn:definitions>`;

            const conditionalModel = new BpmnModel();
            await conditionalModel.loadXml(conditionalXml);

            // Condition not met
            mockContext.variables.isReady = false;
            let result = handler.processIntermediateEvent(
                conditionalModel, 
                "ConditionalEvent", 
                mockLocation, 
                mockContext,
            );
            expect(result.waitingForEvent?.type).toBe("conditional");

            // Condition met
            mockContext.variables.isReady = true;
            result = handler.processIntermediateEvent(
                conditionalModel, 
                "ConditionalEvent", 
                mockLocation, 
                mockContext,
            );
            expect(result.eventCaught).toBeTruthy();
            expect(result.nextLocations[0].nodeId).toBe("Task");
        });
    });
});
