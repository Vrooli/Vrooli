import { beforeEach, describe, expect, test, vi } from "vitest";
import { 
    DefaultContextTransformer, 
    ContextUtils, 
} from "./contextTransformer.js";
import { 
    type EnhancedExecutionContext,
    type BoundaryEvent,
    type IntermediateEvent,
    type TimerEvent,
    type ParallelBranch,
    type JoinPoint,
    type SubprocessContext,
    type EventSubprocess,
    type MessageEvent,
    type WebhookEvent,
    type SignalEvent,
    type EventInstance,
    type Location,
} from "../types.js";

describe("DefaultContextTransformer", () => {
    let transformer: DefaultContextTransformer;
    let mockBasicContext: Record<string, unknown>;
    let mockEnhancedContext: EnhancedExecutionContext;

    beforeEach(() => {
        vi.clearAllMocks();
        transformer = new DefaultContextTransformer();
        
        mockBasicContext = {
            userId: "user123",
            stepCount: 5,
            variables: { foo: "bar" },
        };

        mockEnhancedContext = {
            variables: { userId: "user123", stepCount: 5 },
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

    describe("enhance", () => {
        test("should convert basic context to enhanced context", () => {
            const result = transformer.enhance(mockBasicContext);

            expect(result).toEqual({
                variables: mockBasicContext,
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
            });
        });

        test("should return same context if already enhanced", () => {
            const result = transformer.enhance(mockEnhancedContext as Record<string, unknown>);

            expect(result).toEqual(mockEnhancedContext);
        });

        test("should handle empty basic context", () => {
            const result = transformer.enhance({});

            expect(result.variables).toEqual({});
            expect(result.events).toBeDefined();
            expect(result.parallelExecution).toBeDefined();
            expect(result.subprocesses).toBeDefined();
            expect(result.external).toBeDefined();
            expect(result.gateways).toBeDefined();
        });
    });

    describe("simplify", () => {
        test("should extract variables from enhanced context", () => {
            const result = transformer.simplify(mockEnhancedContext);

            expect(result).toEqual({ userId: "user123", stepCount: 5 });
        });

        test("should return empty object if no variables", () => {
            const emptyEnhanced = {
                ...mockEnhancedContext,
                variables: {},
            };

            const result = transformer.simplify(emptyEnhanced);

            expect(result).toEqual({});
        });
    });

    describe("merge", () => {
        test("should merge variables correctly", () => {
            const updates = {
                variables: { newVar: "newValue", stepCount: 10 },
            };

            const result = transformer.merge(mockEnhancedContext, updates);

            expect(result.variables).toEqual({
                userId: "user123",
                stepCount: 10,
                newVar: "newValue",
            });
        });

        test("should merge events correctly", () => {
            const mockBoundaryEvent: BoundaryEvent = {
                id: "event1",
                type: "timer",
                attachedToRef: "task1",
                interrupting: true,
                activatedAt: new Date(),
                config: {},
            };

            const updates = {
                events: {
                    active: [mockBoundaryEvent],
                    pending: [],
                    fired: [],
                    timers: [],
                },
            };

            const result = transformer.merge(mockEnhancedContext, updates);

            expect(result.events.active).toEqual([mockBoundaryEvent]);
        });

        test("should append to fired events array", () => {
            const existingFired: EventInstance[] = [
                { id: "event1", eventId: "event1", type: "timer", payload: {}, firedAt: new Date() },
            ];
            const newFired: EventInstance[] = [
                { id: "event2", eventId: "event2", type: "message", payload: {}, firedAt: new Date() },
            ];

            const base = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    fired: existingFired,
                },
            };

            const updates = {
                events: {
                    ...mockEnhancedContext.events,
                    fired: newFired,
                },
            };

            const result = transformer.merge(base, updates);

            expect(result.events.fired).toHaveLength(2);
            expect(result.events.fired).toEqual([...existingFired, ...newFired]);
        });

        test("should merge parallel execution branches", () => {
            const mockBranch: ParallelBranch = {
                id: "branch1",
                branchId: "branch1",
                currentLocation: { id: "loc1", routineId: "routine1", nodeId: "node1" },
                status: "running",
                startedAt: new Date(),
            };

            const updates = {
                parallelExecution: {
                    ...mockEnhancedContext.parallelExecution,
                    activeBranches: [mockBranch],
                },
            };

            const result = transformer.merge(mockEnhancedContext, updates);

            expect(result.parallelExecution.activeBranches).toEqual([mockBranch]);
        });

        test("should merge subprocess stack", () => {
            const mockSubprocess: SubprocessContext = {
                id: "subprocess1",
                subprocessId: "subprocess1",
                parentLocation: { id: "parent1", routineId: "routine1", nodeId: "parent1" },
                variables: {},
                startedAt: new Date(),
                status: "running",
            };

            const updates = {
                subprocesses: {
                    ...mockEnhancedContext.subprocesses,
                    stack: [mockSubprocess],
                },
            };

            const result = transformer.merge(mockEnhancedContext, updates);

            expect(result.subprocesses.stack).toEqual([mockSubprocess]);
        });

        test("should append to external events arrays", () => {
            const existingWebhook: WebhookEvent[] = [
                { id: "webhook1", webhookId: "webhook1", url: "/api/test", method: "POST", payload: {}, receivedAt: new Date() },
            ];
            const newWebhook: WebhookEvent[] = [
                { id: "webhook2", webhookId: "webhook2", url: "/api/test2", method: "POST", payload: {}, receivedAt: new Date() },
            ];

            const base = {
                ...mockEnhancedContext,
                external: {
                    ...mockEnhancedContext.external,
                    webhookEvents: existingWebhook,
                },
            };

            const updates = {
                external: {
                    ...mockEnhancedContext.external,
                    webhookEvents: newWebhook,
                },
            };

            const result = transformer.merge(base, updates);

            expect(result.external.webhookEvents).toHaveLength(2);
            expect(result.external.webhookEvents).toEqual([...existingWebhook, ...newWebhook]);
        });

        test("should handle partial updates", () => {
            const updates = {
                variables: { newVar: "value" },
                // Only updating variables, not other properties
            };

            const result = transformer.merge(mockEnhancedContext, updates);

            expect(result.variables).toEqual({
                userId: "user123",
                stepCount: 5,
                newVar: "value",
            });
            expect(result.events).toEqual(mockEnhancedContext.events);
            expect(result.parallelExecution).toEqual(mockEnhancedContext.parallelExecution);
        });
    });

    describe("validate", () => {
        test("should validate correct enhanced context", () => {
            const result = transformer.validate(mockEnhancedContext);

            expect(result).toBe(true);
        });

        test("should reject null or undefined context", () => {
            expect(transformer.validate(null as any)).toBe(false);
            expect(transformer.validate(undefined as any)).toBe(false);
        });

        test("should reject non-object context", () => {
            expect(transformer.validate("string" as any)).toBe(false);
            expect(transformer.validate(123 as any)).toBe(false);
            expect(transformer.validate(true as any)).toBe(false);
        });

        test("should reject context missing required top-level properties", () => {
            const incompleteContext = {
                variables: {},
                events: {},
                // Missing other required properties
            };

            expect(transformer.validate(incompleteContext as any)).toBe(false);
        });

        test("should reject context with invalid array properties", () => {
            const invalidContext = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    active: "not an array",
                },
            };

            expect(transformer.validate(invalidContext as any)).toBe(false);
        });

        test("should validate boundary events in active array", () => {
            const validBoundaryEvent: BoundaryEvent = {
                id: "event1",
                type: "timer",
                attachedToRef: "task1",
                interrupting: true,
                activatedAt: new Date(),
                config: {},
            };

            const contextWithBoundaryEvent = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    active: [validBoundaryEvent],
                },
            };

            expect(transformer.validate(contextWithBoundaryEvent)).toBe(true);
        });

        test("should reject invalid boundary events", () => {
            const invalidBoundaryEvent = {
                id: "event1",
                type: "timer",
                // Missing required properties
            };

            const contextWithInvalidEvent = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    active: [invalidBoundaryEvent],
                },
            };

            expect(transformer.validate(contextWithInvalidEvent as any)).toBe(false);
        });

        test("should validate timer events", () => {
            const validTimerEvent: TimerEvent = {
                id: "timer1",
                eventId: "event1",
                expiresAt: new Date(),
                duration: 5000,
            };

            const contextWithTimer = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    timers: [validTimerEvent],
                },
            };

            expect(transformer.validate(contextWithTimer)).toBe(true);
        });

        test("should validate parallel branches", () => {
            const validBranch: ParallelBranch = {
                id: "branch1",
                branchId: "branch1",
                currentLocation: { id: "loc1", routineId: "routine1", nodeId: "node1" },
                status: "running",
                startedAt: new Date(),
                context: {},
            };

            const contextWithBranch = {
                ...mockEnhancedContext,
                parallelExecution: {
                    ...mockEnhancedContext.parallelExecution,
                    activeBranches: [validBranch],
                },
            };

            expect(transformer.validate(contextWithBranch)).toBe(true);
        });

        test("should validate join points", () => {
            const validJoinPoint: JoinPoint = {
                id: "join1",
                gatewayId: "gateway1",
                requiredBranches: ["branch1", "branch2"],
                completedBranches: ["branch1"],
                isReady: false,
            };

            const contextWithJoinPoint = {
                ...mockEnhancedContext,
                parallelExecution: {
                    ...mockEnhancedContext.parallelExecution,
                    joinPoints: [validJoinPoint],
                },
            };

            expect(transformer.validate(contextWithJoinPoint)).toBe(true);
        });

        test("should handle validation errors gracefully", () => {
            // Create a context that would throw during validation
            const problematicContext = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    active: [
                        {
                            // This will cause an error in validateBoundaryEvent
                            get id() { throw new Error("Test error"); },
                        },
                    ],
                },
            };

            expect(transformer.validate(problematicContext as any)).toBe(false);
        });
    });

    describe("prune", () => {
        test("should remove expired timers", () => {
            const now = new Date();
            const pastDate = new Date(now.getTime() - 2 * 60 * 60 * 1000); // 2 hours ago
            const futureDate = new Date(now.getTime() + 2 * 60 * 60 * 1000); // 2 hours from now

            const expiredTimer: TimerEvent = {
                id: "timer1",
                eventId: "event1",
                expiresAt: pastDate,
                duration: 5000,
            };

            const validTimer: TimerEvent = {
                id: "timer2",
                eventId: "event2",
                expiresAt: futureDate,
                duration: 5000,
            };

            const contextWithTimers = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    timers: [expiredTimer, validTimer],
                },
            };

            const result = transformer.prune(contextWithTimers);

            expect(result.events.timers).toHaveLength(1);
            expect(result.events.timers[0]).toEqual(validTimer);
        });

        test("should remove old fired events", () => {
            const now = new Date();
            const oldDate = new Date(now.getTime() - 2 * 60 * 60 * 1000); // 2 hours ago
            const recentDate = new Date(now.getTime() - 30 * 60 * 1000); // 30 minutes ago

            const oldEvent: EventInstance = {
                id: "event1",
                type: "timer",
                firedAt: oldDate,
            };

            const recentEvent: EventInstance = {
                id: "event2",
                type: "message",
                firedAt: recentDate,
            };

            const contextWithEvents = {
                ...mockEnhancedContext,
                events: {
                    ...mockEnhancedContext.events,
                    fired: [oldEvent, recentEvent],
                },
            };

            const result = transformer.prune(contextWithEvents);

            expect(result.events.fired).toHaveLength(1);
            expect(result.events.fired[0]).toEqual(recentEvent);
        });

        test("should remove inactive parallel branches", () => {
            const activeBranch: ParallelBranch = {
                id: "branch1",
                branchId: "branch1",
                currentLocation: { id: "loc1", routineId: "routine1", nodeId: "node1" },
                status: "running",
                startedAt: new Date(),
            };

            const completedBranch: ParallelBranch = {
                id: "branch2",
                branchId: "branch2",
                currentLocation: { id: "loc2", routineId: "routine1", nodeId: "node2" },
                status: "completed",
                startedAt: new Date(),
            };

            const failedBranch: ParallelBranch = {
                id: "branch3",
                branchId: "branch3",
                currentLocation: { id: "loc3", routineId: "routine1", nodeId: "node3" },
                status: "failed",
                startedAt: new Date(),
            };

            const contextWithBranches = {
                ...mockEnhancedContext,
                parallelExecution: {
                    ...mockEnhancedContext.parallelExecution,
                    activeBranches: [activeBranch, completedBranch, failedBranch],
                },
            };

            const result = transformer.prune(contextWithBranches);

            expect(result.parallelExecution.activeBranches).toHaveLength(1);
            expect(result.parallelExecution.activeBranches[0]).toEqual(activeBranch);
        });

        test("should remove completed event subprocesses", () => {
            const activeSubprocess: EventSubprocess = {
                id: "sub1",
                subprocessId: "sub1",
                triggerEvent: "timer",
                interrupting: false,
                status: "running",
                startedAt: new Date(),
            };

            const completedSubprocess: EventSubprocess = {
                id: "sub2",
                subprocessId: "sub2",
                triggerEvent: "message",
                interrupting: false,
                status: "completed",
                startedAt: new Date(),
            };

            const contextWithSubprocesses = {
                ...mockEnhancedContext,
                subprocesses: {
                    ...mockEnhancedContext.subprocesses,
                    eventSubprocesses: [activeSubprocess, completedSubprocess],
                },
            };

            const result = transformer.prune(contextWithSubprocesses);

            expect(result.subprocesses.eventSubprocesses).toHaveLength(1);
            expect(result.subprocesses.eventSubprocesses[0]).toEqual(activeSubprocess);
        });

        test("should remove old external events", () => {
            const now = new Date();
            const oldDate = new Date(now.getTime() - 2 * 60 * 60 * 1000); // 2 hours ago
            const recentDate = new Date(now.getTime() - 30 * 60 * 1000); // 30 minutes ago

            const oldWebhook: WebhookEvent = {
                id: "webhook1",
                webhookId: "webhook1",
                url: "/api/old",
                method: "POST",
                payload: {},
                receivedAt: oldDate,
            };

            const recentWebhook: WebhookEvent = {
                id: "webhook2",
                webhookId: "webhook2",
                url: "/api/recent",
                method: "POST",
                payload: {},
                receivedAt: recentDate,
            };

            const oldSignal: SignalEvent = {
                id: "signal1",
                signalRef: "old-signal",
                scope: "global",
                payload: {},
                propagatedAt: oldDate,
            };

            const recentSignal: SignalEvent = {
                id: "signal2",
                signalRef: "recent-signal",
                scope: "global",
                payload: {},
                propagatedAt: recentDate,
            };

            const contextWithExternalEvents = {
                ...mockEnhancedContext,
                external: {
                    ...mockEnhancedContext.external,
                    webhookEvents: [oldWebhook, recentWebhook],
                    signalEvents: [oldSignal, recentSignal],
                },
            };

            const result = transformer.prune(contextWithExternalEvents);

            expect(result.external.webhookEvents).toHaveLength(1);
            expect(result.external.webhookEvents[0]).toEqual(recentWebhook);
            expect(result.external.signalEvents).toHaveLength(1);
            expect(result.external.signalEvents[0]).toEqual(recentSignal);
        });

        test("should preserve other properties unchanged", () => {
            const result = transformer.prune(mockEnhancedContext);

            expect(result.variables).toEqual(mockEnhancedContext.variables);
            expect(result.gateways).toEqual(mockEnhancedContext.gateways);
            expect(result.subprocesses.stack).toEqual(mockEnhancedContext.subprocesses.stack);
        });
    });
});

describe("ContextUtils", () => {
    describe("createEmpty", () => {
        test("should create empty enhanced context", () => {
            const result = ContextUtils.createEmpty();

            expect(result).toEqual({
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
            });
        });
    });

    describe("addVariable", () => {
        test("should add variable to context", () => {
            const context = ContextUtils.createEmpty();
            const result = ContextUtils.addVariable(context, "testKey", "testValue");

            expect(result.variables).toEqual({
                testKey: "testValue",
            });
        });

        test("should preserve existing variables", () => {
            const context = ContextUtils.createEmpty();
            context.variables.existingVar = "existingValue";
            
            const result = ContextUtils.addVariable(context, "newVar", "newValue");

            expect(result.variables).toEqual({
                existingVar: "existingValue",
                newVar: "newValue",
            });
        });

        test("should handle complex variable values", () => {
            const context = ContextUtils.createEmpty();
            const complexValue = { nested: { data: [1, 2, 3] } };
            
            const result = ContextUtils.addVariable(context, "complex", complexValue);

            expect(result.variables.complex).toEqual(complexValue);
        });
    });

    describe("addBoundaryEvent", () => {
        test("should add boundary event to context", () => {
            const context = ContextUtils.createEmpty();
            const boundaryEvent: BoundaryEvent = {
                id: "event1",
                type: "timer",
                attachedToRef: "task1",
                interrupting: true,
                activatedAt: new Date(),
                config: {},
            };

            const result = ContextUtils.addBoundaryEvent(context, boundaryEvent);

            expect(result.events.active).toHaveLength(1);
            expect(result.events.active[0]).toEqual(boundaryEvent);
        });

        test("should append to existing boundary events", () => {
            const context = ContextUtils.createEmpty();
            const existingEvent: BoundaryEvent = {
                id: "event1",
                type: "timer",
                attachedToRef: "task1",
                interrupting: true,
                activatedAt: new Date(),
                config: {},
            };
            context.events.active = [existingEvent];

            const newEvent: BoundaryEvent = {
                id: "event2",
                type: "message",
                attachedToRef: "task2",
                interrupting: false,
                activatedAt: new Date(),
                config: {},
            };

            const result = ContextUtils.addBoundaryEvent(context, newEvent);

            expect(result.events.active).toHaveLength(2);
            expect(result.events.active).toContain(existingEvent);
            expect(result.events.active).toContain(newEvent);
        });
    });

    describe("addTimerEvent", () => {
        test("should add timer event to context", () => {
            const context = ContextUtils.createEmpty();
            const timerEvent: TimerEvent = {
                id: "timer1",
                eventId: "event1",
                expiresAt: new Date(),
                duration: 5000,
            };

            const result = ContextUtils.addTimerEvent(context, timerEvent);

            expect(result.events.timers).toHaveLength(1);
            expect(result.events.timers[0]).toEqual(timerEvent);
        });
    });

    describe("addParallelBranch", () => {
        test("should add parallel branch to context", () => {
            const context = ContextUtils.createEmpty();
            const branch: ParallelBranch = {
                id: "branch1",
                branchId: "branch1",
                currentLocation: { id: "loc1", routineId: "routine1", nodeId: "node1" },
                status: "running",
                startedAt: new Date(),
            };

            const result = ContextUtils.addParallelBranch(context, branch);

            expect(result.parallelExecution.activeBranches).toHaveLength(1);
            expect(result.parallelExecution.activeBranches[0]).toEqual(branch);
        });
    });

    describe("completeParallelBranch", () => {
        test("should complete parallel branch and update status", () => {
            const context = ContextUtils.createEmpty();
            const branch: ParallelBranch = {
                id: "branch1",
                branchId: "branch1",
                currentLocation: { id: "loc1", routineId: "routine1", nodeId: "node1" },
                status: "running",
                startedAt: new Date(),
            };
            context.parallelExecution.activeBranches = [branch];

            const result = ContextUtils.completeParallelBranch(context, "branch1", "success");

            expect(result.parallelExecution.activeBranches[0].status).toBe("completed");
            expect(result.parallelExecution.activeBranches[0].result).toBe("success");
            expect(result.parallelExecution.activeBranches[0].completedAt).toBeDefined();
            expect(result.parallelExecution.completedBranches).toContain("branch1");
        });

        test("should not affect other branches", () => {
            const context = ContextUtils.createEmpty();
            const branch1: ParallelBranch = {
                id: "branch1",
                branchId: "branch1",
                currentLocation: { id: "loc1", routineId: "routine1", nodeId: "node1" },
                status: "running",
                startedAt: new Date(),
            };
            const branch2: ParallelBranch = {
                id: "branch2",
                branchId: "branch2",
                currentLocation: { id: "loc2", routineId: "routine1", nodeId: "node2" },
                status: "running",
                startedAt: new Date(),
            };
            context.parallelExecution.activeBranches = [branch1, branch2];

            const result = ContextUtils.completeParallelBranch(context, "branch1");

            expect(result.parallelExecution.activeBranches[0].status).toBe("completed");
            expect(result.parallelExecution.activeBranches[1].status).toBe("running");
        });
    });

    describe("enterSubprocess", () => {
        test("should add subprocess to stack", () => {
            const context = ContextUtils.createEmpty();
            const subprocess: SubprocessContext = {
                id: "subprocess1",
                subprocessId: "subprocess1",
                parentLocation: { id: "parent1", routineId: "routine1", nodeId: "parent1" },
                variables: {},
                startedAt: new Date(),
                status: "running",
            };

            const result = ContextUtils.enterSubprocess(context, subprocess);

            expect(result.subprocesses.stack).toHaveLength(1);
            expect(result.subprocesses.stack[0]).toEqual(subprocess);
        });

        test("should stack multiple subprocesses", () => {
            const context = ContextUtils.createEmpty();
            const subprocess1: SubprocessContext = {
                id: "subprocess1",
                subprocessId: "subprocess1",
                parentLocation: { id: "parent1", routineId: "routine1", nodeId: "parent1" },
                variables: {},
                startedAt: new Date(),
                status: "running",
            };
            const subprocess2: SubprocessContext = {
                id: "subprocess2",
                subprocessId: "subprocess2",
                parentLocation: { id: "parent2", routineId: "routine1", nodeId: "parent2" },
                variables: {},
                startedAt: new Date(),
                status: "running",
            };

            let result = ContextUtils.enterSubprocess(context, subprocess1);
            result = ContextUtils.enterSubprocess(result, subprocess2);

            expect(result.subprocesses.stack).toHaveLength(2);
            expect(result.subprocesses.stack[0]).toEqual(subprocess1);
            expect(result.subprocesses.stack[1]).toEqual(subprocess2);
        });
    });

    describe("exitSubprocess", () => {
        test("should remove top subprocess from stack", () => {
            const context = ContextUtils.createEmpty();
            const subprocess1: SubprocessContext = {
                id: "subprocess1",
                type: "embedded",
                parentLocation: { id: "parent1", routineId: "routine1", nodeId: "parent1" },
                currentLocation: { nodeId: "sub1" },
                startedAt: new Date(),
                context: {},
            };
            const subprocess2: SubprocessContext = {
                id: "subprocess2",
                type: "embedded",
                parentLocation: { id: "parent2", routineId: "routine1", nodeId: "parent2" },
                currentLocation: { nodeId: "sub2" },
                startedAt: new Date(),
                context: {},
            };
            context.subprocesses.stack = [subprocess1, subprocess2];

            const result = ContextUtils.exitSubprocess(context);

            expect(result.subprocesses.stack).toHaveLength(1);
            expect(result.subprocesses.stack[0]).toEqual(subprocess1);
        });

        test("should handle empty stack gracefully", () => {
            const context = ContextUtils.createEmpty();

            const result = ContextUtils.exitSubprocess(context);

            expect(result.subprocesses.stack).toHaveLength(0);
        });
    });

    describe("fireEvent", () => {
        test("should add event to fired events", () => {
            const context = ContextUtils.createEmpty();
            const eventInstance: EventInstance = {
                id: "event1",
                eventId: "event1",
                type: "timer",
                payload: {},
                firedAt: new Date(),
            };

            const result = ContextUtils.fireEvent(context, eventInstance);

            expect(result.events.fired).toHaveLength(1);
            expect(result.events.fired[0]).toEqual(eventInstance);
        });

        test("should append to existing fired events", () => {
            const context = ContextUtils.createEmpty();
            const existingEvent: EventInstance = {
                id: "event1",
                eventId: "event1",
                type: "timer",
                payload: {},
                firedAt: new Date(),
            };
            context.events.fired = [existingEvent];

            const newEvent: EventInstance = {
                id: "event2",
                eventId: "event2",
                type: "message",
                payload: {},
                firedAt: new Date(),
            };

            const result = ContextUtils.fireEvent(context, newEvent);

            expect(result.events.fired).toHaveLength(2);
            expect(result.events.fired).toContain(existingEvent);
            expect(result.events.fired).toContain(newEvent);
        });
    });
});
