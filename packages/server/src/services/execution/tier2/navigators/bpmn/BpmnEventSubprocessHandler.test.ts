import { beforeEach, describe, expect, test } from "vitest";
import { BpmnEventSubprocessHandler } from "./BpmnEventSubprocessHandler.js";
import { BpmnModel } from "./BpmnModel.js";
import type { EnhancedExecutionContext, AbstractLocation, EventSubprocess } from "../../types.js";

describe("BpmnEventSubprocessHandler", () => {
    let handler: BpmnEventSubprocessHandler;
    let model: BpmnModel;
    let mockContext: EnhancedExecutionContext;
    let mockLocation: AbstractLocation;

    const eventSubprocessXml = `<?xml version="1.0" encoding="UTF-8"?>
    <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
        <bpmn:process id="Process_1">
            <bpmn:startEvent id="MainStart">
                <bpmn:outgoing>ToTask</bpmn:outgoing>
            </bpmn:startEvent>
            <bpmn:task id="MainTask">
                <bpmn:incoming>ToTask</bpmn:incoming>
                <bpmn:outgoing>ToEnd</bpmn:outgoing>
            </bpmn:task>
            <bpmn:endEvent id="MainEnd">
                <bpmn:incoming>ToEnd</bpmn:incoming>
            </bpmn:endEvent>
            <bpmn:subProcess id="EventSubprocess1" triggeredByEvent="true">
                <bpmn:startEvent id="TimerStart" isInterrupting="true">
                    <bpmn:timerEventDefinition>
                        <bpmn:timeDuration>PT5M</bpmn:timeDuration>
                    </bpmn:timerEventDefinition>
                    <bpmn:outgoing>ToEventTask1</bpmn:outgoing>
                </bpmn:startEvent>
                <bpmn:task id="EventTask1">
                    <bpmn:incoming>ToEventTask1</bpmn:incoming>
                    <bpmn:outgoing>ToEventEnd1</bpmn:outgoing>
                </bpmn:task>
                <bpmn:endEvent id="EventEnd1">
                    <bpmn:incoming>ToEventEnd1</bpmn:incoming>
                </bpmn:endEvent>
                <bpmn:sequenceFlow id="ToEventTask1" sourceRef="TimerStart" targetRef="EventTask1" />
                <bpmn:sequenceFlow id="ToEventEnd1" sourceRef="EventTask1" targetRef="EventEnd1" />
            </bpmn:subProcess>
            <bpmn:subProcess id="EventSubprocess2" triggeredByEvent="true" cancelActivity="false">
                <bpmn:startEvent id="MessageStart" isInterrupting="false">
                    <bpmn:messageEventDefinition messageRef="Message_1" />
                    <bpmn:outgoing>ToEventTask2</bpmn:outgoing>
                </bpmn:startEvent>
                <bpmn:task id="EventTask2">
                    <bpmn:incoming>ToEventTask2</bpmn:incoming>
                    <bpmn:outgoing>ToEventEnd2</bpmn:outgoing>
                </bpmn:task>
                <bpmn:endEvent id="EventEnd2">
                    <bpmn:incoming>ToEventEnd2</bpmn:incoming>
                </bpmn:endEvent>
                <bpmn:sequenceFlow id="ToEventTask2" sourceRef="MessageStart" targetRef="EventTask2" />
                <bpmn:sequenceFlow id="ToEventEnd2" sourceRef="EventTask2" targetRef="EventEnd2" />
            </bpmn:subProcess>
            <bpmn:sequenceFlow id="ToTask" sourceRef="MainStart" targetRef="MainTask" />
            <bpmn:sequenceFlow id="ToEnd" sourceRef="MainTask" targetRef="MainEnd" />
        </bpmn:process>
    </bpmn:definitions>`;

    beforeEach(async () => {
        handler = new BpmnEventSubprocessHandler();
        model = new BpmnModel();
        await model.loadXml(eventSubprocessXml);

        mockLocation = {
            nodeId: "MainTask",
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

    describe("monitorEventSubprocesses", () => {
        test("should detect event subprocesses in the model", () => {
            const result = handler.monitorEventSubprocesses(
                model, 
                mockLocation, 
                mockContext,
            );

            // Initially no triggers
            expect(result.nextLocations).toHaveLength(0);
            expect(result.subprocessActivations).toHaveLength(0);
            expect(result.shouldInterruptMainProcess).toBe(false);
        });

        test("should trigger timer event subprocess when timer expires", () => {
            // Add expired timer
            mockContext.events.timers = [{
                id: "timer_1",
                eventId: "Timer_1",
                expiresAt: new Date(Date.now() - 1000), // Expired
                type: "duration",
            }];

            const result = handler.monitorEventSubprocesses(
                model, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(1);
            expect(result.nextLocations[0].nodeId).toBe("TimerStart");
            expect(result.subprocessActivations).toHaveLength(1);
            expect(result.subprocessActivations[0].subprocessId).toBe("EventSubprocess1");
            expect(result.shouldInterruptMainProcess).toBe(true);
        });

        test("should trigger message event subprocess on message", () => {
            // Add message event
            mockContext.external.messageEvents = [{
                id: "msg_1",
                messageRef: "Message_1",
                payload: { data: "test" },
                receivedAt: new Date(),
            }];

            const result = handler.monitorEventSubprocesses(
                model, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(1);
            expect(result.nextLocations[0].nodeId).toBe("MessageStart");
            expect(result.subprocessActivations[0].subprocessId).toBe("EventSubprocess2");
            expect(result.subprocessActivations[0].interrupting).toBe(false);
            expect(result.shouldInterruptMainProcess).toBe(false);
        });

        test("should handle multiple event subprocess triggers", () => {
            // Add both timer and message
            mockContext.events.timers = [{
                id: "timer_1",
                eventId: "Timer_1",
                expiresAt: new Date(Date.now() - 1000),
                type: "duration",
            }];
            mockContext.external.messageEvents = [{
                id: "msg_1",
                messageRef: "Message_1",
                payload: { data: "test" },
                receivedAt: new Date(),
            }];

            const result = handler.monitorEventSubprocesses(
                model, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(2);
            expect(result.subprocessActivations).toHaveLength(2);
            expect(result.shouldInterruptMainProcess).toBe(true); // At least one interrupting
        });

        test("should consume trigger events", () => {
            mockContext.external.messageEvents = [{
                id: "msg_1",
                messageRef: "Message_1",
                payload: { data: "test" },
                receivedAt: new Date(),
            }];

            const result = handler.monitorEventSubprocesses(
                model, 
                mockLocation, 
                mockContext,
            );

            // Message should be consumed
            expect(result.updatedContext.external.messageEvents).toHaveLength(0);
        });
    });

    describe("processEventSubprocess", () => {
        describe("monitoring state", () => {
            beforeEach(() => {
                mockContext.subprocesses.eventSubprocesses = [{
                    id: "event_sub_1",
                    subprocessId: "EventSubprocess1",
                    triggerEvent: "TimerStart",
                    interrupting: true,
                    status: "monitoring",
                }];
            });

            test("should check for triggers in monitoring state", () => {
                const result = handler.processEventSubprocess(
                    model, 
                    "EventSubprocess1", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(0);
                expect(result.shouldInterruptMainProcess).toBe(false);
            });
        });

        describe("active state", () => {
            beforeEach(() => {
                mockContext.subprocesses.eventSubprocesses = [{
                    id: "event_sub_1",
                    subprocessId: "EventSubprocess1",
                    triggerEvent: "TimerStart",
                    interrupting: true,
                    status: "active",
                    startedAt: new Date(),
                }];
                mockLocation = {
                    nodeId: "EventTask1",
                    routineId: "routine_1",
                    type: "subprocess_context",
                    metadata: {
                        subprocessId: "EventSubprocess1",
                    },
                };
            });

            test("should navigate within active event subprocess", () => {
                const result = handler.processEventSubprocess(
                    model, 
                    "EventSubprocess1", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(1);
                expect(result.nextLocations[0].nodeId).toBe("EventEnd1");
                expect(result.nextLocations[0].type).toBe("subprocess_context");
            });

            test("should complete event subprocess at end event", () => {
                mockLocation.nodeId = "EventEnd1";
                
                const result = handler.processEventSubprocess(
                    model, 
                    "EventSubprocess1", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.updatedContext.subprocesses.eventSubprocesses[0].status).toBe("completed");
            });
        });

        describe("completed state", () => {
            test("should handle completed event subprocess", () => {
                mockContext.subprocesses.eventSubprocesses = [{
                    id: "event_sub_1",
                    subprocessId: "EventSubprocess1",
                    triggerEvent: "TimerStart",
                    interrupting: true,
                    status: "completed",
                    startedAt: new Date(),
                    completedAt: new Date(),
                }];

                const result = handler.processEventSubprocess(
                    model, 
                    "EventSubprocess1", 
                    mockLocation, 
                    mockContext,
                );

                expect(result.nextLocations).toHaveLength(0);
            });
        });
    });

    describe("event type triggers", () => {
        test("should handle signal event triggers", async () => {
            const signalXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="SignalSubprocess" triggeredByEvent="true">
                        <bpmn:startEvent id="SignalStart">
                            <bpmn:signalEventDefinition signalRef="Signal_1" />
                        </bpmn:startEvent>
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            const signalModel = new BpmnModel();
            await signalModel.loadXml(signalXml);

            mockContext.external.signalEvents = [{
                id: "sig_1",
                signalRef: "Signal_1",
                payload: { data: "test" },
            }];

            const result = handler.monitorEventSubprocesses(
                signalModel, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(1);
            expect(result.subprocessActivations[0].eventType).toBe("signal");
        });

        test("should handle error event triggers", async () => {
            const errorXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="ErrorSubprocess" triggeredByEvent="true">
                        <bpmn:startEvent id="ErrorStart">
                            <bpmn:errorEventDefinition errorRef="Error_1" />
                        </bpmn:startEvent>
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            const errorModel = new BpmnModel();
            await errorModel.loadXml(errorXml);

            mockContext.variables.errors = [{
                code: "Error_1",
                message: "Test error",
            }];

            const result = handler.monitorEventSubprocesses(
                errorModel, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(1);
            expect(result.subprocessActivations[0].eventType).toBe("error");
        });

        test("should handle escalation event triggers", async () => {
            const escalationXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="EscalationSubprocess" triggeredByEvent="true">
                        <bpmn:startEvent id="EscalationStart">
                            <bpmn:escalationEventDefinition escalationRef="Escalation_1" />
                        </bpmn:startEvent>
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            const escalationModel = new BpmnModel();
            await escalationModel.loadXml(escalationXml);

            mockContext.events.fired = [{
                id: "esc_1",
                eventId: "EscalationThrow",
                type: "escalation",
                payload: { escalationCode: "Escalation_1" },
                firedAt: new Date(),
                source: "activity",
            }];

            const result = handler.monitorEventSubprocesses(
                escalationModel, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(1);
            expect(result.subprocessActivations[0].eventType).toBe("escalation");
        });

        test("should handle conditional event triggers", async () => {
            const conditionalXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="ConditionalSubprocess" triggeredByEvent="true">
                        <bpmn:startEvent id="ConditionalStart">
                            <bpmn:conditionalEventDefinition>
                                <bpmn:condition>criticalError == true</bpmn:condition>
                            </bpmn:conditionalEventDefinition>
                        </bpmn:startEvent>
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            const conditionalModel = new BpmnModel();
            await conditionalModel.loadXml(conditionalXml);

            mockContext.variables.criticalError = true;

            const result = handler.monitorEventSubprocesses(
                conditionalModel, 
                mockLocation, 
                mockContext,
            );

            expect(result.nextLocations).toHaveLength(1);
            expect(result.subprocessActivations[0].eventType).toBe("conditional");
        });
    });

    describe("getEventSubprocessSummary", () => {
        test("should provide correct summary", () => {
            mockContext.subprocesses.eventSubprocesses = [
                {
                    id: "sub_1",
                    subprocessId: "Sub1",
                    triggerEvent: "Event1",
                    interrupting: true,
                    status: "monitoring",
                },
                {
                    id: "sub_2",
                    subprocessId: "Sub2",
                    triggerEvent: "Event2",
                    interrupting: false,
                    status: "active",
                    startedAt: new Date(),
                },
                {
                    id: "sub_3",
                    subprocessId: "Sub3",
                    triggerEvent: "Event3",
                    interrupting: true,
                    status: "completed",
                    startedAt: new Date(),
                    completedAt: new Date(),
                },
                {
                    id: "sub_4",
                    subprocessId: "Sub4",
                    triggerEvent: "Event4",
                    interrupting: false,
                    status: "active",
                    startedAt: new Date(),
                },
            ];

            const summary = handler.getEventSubprocessSummary(mockContext);

            expect(summary.monitoring).toBe(1);
            expect(summary.active).toBe(2);
            expect(summary.completed).toBe(1);
            expect(summary.interrupting).toBe(2);
            expect(summary.nonInterrupting).toBe(2);
        });
    });

    describe("helper methods", () => {
        test("activateEventSubprocessMonitoring should add monitoring state", () => {
            const result = handler.activateEventSubprocessMonitoring(
                mockContext, 
                "Subprocess1", 
                "Event1",
            );

            expect(result.subprocesses.eventSubprocesses).toHaveLength(1);
            const subprocess = result.subprocesses.eventSubprocesses[0];
            expect(subprocess.subprocessId).toBe("Subprocess1");
            expect(subprocess.triggerEvent).toBe("Event1");
            expect(subprocess.status).toBe("monitoring");
        });

        test("deactivateEventSubprocessMonitoring should remove monitoring state", () => {
            mockContext.subprocesses.eventSubprocesses = [
                {
                    id: "sub_1",
                    subprocessId: "Sub1",
                    triggerEvent: "Event1",
                    interrupting: true,
                    status: "monitoring",
                },
                {
                    id: "sub_2",
                    subprocessId: "Sub1",
                    triggerEvent: "Event2",
                    interrupting: false,
                    status: "active",
                    startedAt: new Date(),
                },
            ];

            const result = handler.deactivateEventSubprocessMonitoring(
                mockContext, 
                "Sub1",
            );

            expect(result.subprocesses.eventSubprocesses).toHaveLength(1);
            expect(result.subprocesses.eventSubprocesses[0].status).toBe("active");
        });
    });

    describe("error handling", () => {
        test("should throw error for non-existent subprocess", () => {
            mockContext.subprocesses.eventSubprocesses = [];

            expect(() => {
                handler.processEventSubprocess(
                    model, 
                    "NonExistentSubprocess", 
                    mockLocation, 
                    mockContext,
                );
            }).toThrow("Event subprocess not found: NonExistentSubprocess");
        });
    });

    describe("interrupting vs non-interrupting", () => {
        test("should detect interrupting status correctly", async () => {
            const mixedXml = `<?xml version="1.0" encoding="UTF-8"?>
            <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                <bpmn:process id="Process_1">
                    <bpmn:subProcess id="Interrupting" triggeredByEvent="true">
                        <bpmn:startEvent id="Start1" />
                    </bpmn:subProcess>
                    <bpmn:subProcess id="NonInterrupting" triggeredByEvent="true" cancelActivity="false">
                        <bpmn:startEvent id="Start2" />
                    </bpmn:subProcess>
                </bpmn:process>
            </bpmn:definitions>`;

            const mixedModel = new BpmnModel();
            await mixedModel.loadXml(mixedXml);

            // Test interrupting (default)
            const interrupting = (handler as any).isEventSubprocessInterrupting(
                mixedModel.getElementById("Interrupting"),
            );
            expect(interrupting).toBe(true);

            // Test non-interrupting
            const nonInterrupting = (handler as any).isEventSubprocessInterrupting(
                mixedModel.getElementById("NonInterrupting"),
            );
            expect(nonInterrupting).toBe(false);
        });
    });
});
