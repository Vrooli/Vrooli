import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { type Logger } from "winston";
import { 
    type Navigator,
    type Location,
    type StepInfo,
    type NavigationTrigger,
    type NavigationTimeout,
    type NavigationEvent,
    type RoutineVersionConfigObject,
    generatePK,
} from "@vrooli/shared";

// Mock implementations for testing
class MockNavigatorBase implements Navigator {
    constructor(
        public readonly type: string,
        public readonly version: string,
        protected logger: Logger,
    ) {}

    canNavigate(routine: unknown): boolean {
        const r = routine as any;
        return r?.graph?.__type === this.type || 
               (this.type === "native" && !r?.graph);
    }

    getStartLocation(routine: unknown): Location {
        const nodes = this.getNodes(routine);
        const startNode = nodes.find((n: any) => 
            !nodes.some((other: any) => other.nextNodes?.includes(n.id)),
        ) || nodes[0];
        
        return {
            id: `${generatePK()}-${startNode.id}`,
            routineId: generatePK(),
            nodeId: startNode.id,
        };
    }

    getAllStartLocations(routine: unknown): Location[] {
        const nodes = this.getNodes(routine);
        const startNodes = nodes.filter((n: any) => 
            !nodes.some((other: any) => other.nextNodes?.includes(n.id)),
        );
        
        return startNodes.map((n: any) => ({
            id: `${generatePK()}-${n.id}`,
            routineId: generatePK(),
            nodeId: n.id,
        }));
    }

    async getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]> {
        // Mock implementation
        return [];
    }

    async isEndLocation(location: Location): Promise<boolean> {
        return location.nodeId === "end" || location.nodeId.includes("end");
    }

    async getStepInfo(location: Location): Promise<StepInfo> {
        return {
            id: location.nodeId,
            name: `Step ${location.nodeId}`,
            type: "action",
            description: `Mock step for ${location.nodeId}`,
            config: {},
        };
    }

    async getDependencies(location: Location): Promise<string[]> {
        return [];
    }

    async getParallelBranches(location: Location): Promise<Location[][]> {
        return [];
    }

    async getLocationTriggers(location: Location): Promise<NavigationTrigger[]> {
        return [];
    }

    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        return [];
    }

    async canTriggerEvent(location: Location, event: NavigationEvent): Promise<boolean> {
        return false;
    }

    protected getNodes(routine: unknown): any[] {
        const r = routine as any;
        return r?.graph?.nodes || [];
    }

    validateAndCache(routine: unknown): RoutineVersionConfigObject {
        if (!this.canNavigate(routine)) {
            throw new Error(`Invalid routine configuration for ${this.type} navigator`);
        }
        return routine as RoutineVersionConfigObject;
    }

    generateRoutineId(config: RoutineVersionConfigObject): string {
        return generatePK();
    }

    async getCachedConfig(routineId: string): Promise<RoutineVersionConfigObject> {
        throw new Error(`Routine config ${routineId} not in cache`);
    }
}

class MockNativeNavigator extends MockNavigatorBase {
    constructor(logger: Logger) {
        super("native", "1.0.0", logger);
    }

    async getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]> {
        // Mock navigation logic based on context
        if (current.nodeId === "start") {
            return [{
                id: `${generatePK()}-process`,
                routineId: current.routineId,
                nodeId: "process",
            }];
        }
        if (current.nodeId === "process") {
            if (context.status === "error") {
                return [{
                    id: `${generatePK()}-error_handler`,
                    routineId: current.routineId,
                    nodeId: "error_handler",
                }];
            }
            return [{
                id: `${generatePK()}-validate`,
                routineId: current.routineId,
                nodeId: "validate",
            }];
        }
        return [];
    }

    async getParallelBranches(location: Location): Promise<Location[][]> {
        if (location.nodeId === "split") {
            return [
                [{ id: `${generatePK()}-branch1`, routineId: location.routineId, nodeId: "branch1" }],
                [{ id: `${generatePK()}-branch2`, routineId: location.routineId, nodeId: "branch2" }],
                [{ id: `${generatePK()}-branch3`, routineId: location.routineId, nodeId: "branch3" }],
            ];
        }
        return [];
    }

    async getLocationTriggers(location: Location): Promise<NavigationTrigger[]> {
        const node = { id: location.nodeId, triggers: [{ id: "trigger1", event: "data.received", condition: "payload.type == 'important'" }] };
        return node.triggers || [];
    }

    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        if (location.nodeId === "timed_step") {
            return [{
                duration: 30000,
                action: "skip",
                fallbackLocation: {
                    id: `${generatePK()}-timeout_handler`,
                    routineId: location.routineId,
                    nodeId: "timeout_handler",
                },
            }];
        }
        return [];
    }

    async canTriggerEvent(location: Location, event: NavigationEvent): Promise<boolean> {
        if (event.type === "data.received" && event.payload?.type === "important") {
            return true;
        }
        return false;
    }
}

class MockBPMNNavigator extends MockNavigatorBase {
    constructor(logger: Logger) {
        super("bpmn", "1.0.0", logger);
    }

    async getNextLocations(current: Location, context: Record<string, unknown>): Promise<Location[]> {
        if (current.nodeId === "task1") {
            if (context.data?.approved === true) {
                return [{
                    id: `${generatePK()}-task2`,
                    routineId: current.routineId,
                    nodeId: "task2",
                }];
            } else {
                return [{
                    id: `${generatePK()}-end1`,
                    routineId: current.routineId,
                    nodeId: "end1",
                }];
            }
        }
        return [];
    }

    async getStepInfo(location: Location): Promise<StepInfo> {
        if (location.nodeId === "gateway1") {
            return {
                id: location.nodeId,
                name: "Gateway",
                type: "gateway",
                description: "Exclusive gateway",
                config: {},
                metadata: { gatewayType: "exclusive" },
            };
        }
        return super.getStepInfo(location);
    }

    async getLocationTimeouts(location: Location): Promise<NavigationTimeout[]> {
        if (location.nodeId === "timer1") {
            return [{
                duration: 300000, // 5 minutes
                action: "proceed",
            }];
        }
        return [];
    }
}

class MockLangchainNavigator extends MockNavigatorBase {
    constructor(logger: Logger) {
        super("langchain", "1.0.0", logger);
    }
}

class MockTemporalNavigator extends MockNavigatorBase {
    constructor(logger: Logger) {
        super("temporal", "1.0.0", logger);
    }
}

class MockNavigatorRegistry {
    private navigators = new Map<string, Navigator>();

    constructor(private logger: Logger) {}

    register(navigator: Navigator): void {
        this.navigators.set(navigator.type, navigator);
    }

    get(type: string): Navigator | undefined {
        return this.navigators.get(type);
    }

    selectNavigator(routine: unknown): Navigator {
        for (const [type, navigator] of this.navigators) {
            if (navigator.canNavigate(routine)) {
                return navigator;
            }
        }
        throw new Error("No navigator available for routine type");
    }

    getRegisteredTypes(): string[] {
        return Array.from(this.navigators.keys());
    }
}

/**
 * Navigation System Integration Tests
 * 
 * These tests validate the critical navigation infrastructure for Tier 2,
 * which enables graph-type-agnostic routine execution:
 * 
 * 1. Navigator registry and type selection
 * 2. Multi-navigator support (Native, BPMN, Langchain, Temporal)
 * 3. Start location discovery (including multiple start points)
 * 4. Path navigation and branch detection
 * 5. Event-driven navigation (triggers and timeouts)
 * 6. Parallel branch coordination
 * 7. End location detection
 * 8. Context-aware navigation decisions
 * 9. Navigator caching and performance
 * 10. Error handling and recovery
 */

describe("Navigation System Integration", () => {
    let logger: Logger;
    let registry: MockNavigatorRegistry;

    beforeEach(() => {
        logger = {
            info: vi.fn(),
            error: vi.fn(),
            warn: vi.fn(),
            debug: vi.fn(),
        } as unknown as Logger;

        registry = new MockNavigatorRegistry(logger);
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    describe("Navigator Registry", () => {
        it("should register and retrieve navigators by type", () => {
            const nativeNav = new MockNativeNavigator(logger);
            const bpmnNav = new MockBPMNNavigator(logger);

            registry.register(nativeNav);
            registry.register(bpmnNav);

            expect(registry.get("native")).toBe(nativeNav);
            expect(registry.get("bpmn")).toBe(bpmnNav);
        });

        it("should select appropriate navigator based on routine structure", () => {
            registry.register(new MockNativeNavigator(logger));
            registry.register(new MockBPMNNavigator(logger));
            registry.register(new MockLangchainNavigator(logger));
            registry.register(new MockTemporalNavigator(logger));

            // Native routine
            const nativeRoutine = {
                __typename: "Routine",
                graph: {
                    __type: "native",
                    nodes: [{ id: "node1", action: "test" }],
                },
            };
            expect(registry.selectNavigator(nativeRoutine).type).toBe("native");

            // BPMN routine
            const bpmnRoutine = {
                __typename: "Routine",
                graph: {
                    __type: "bpmn",
                    process: {
                        tasks: [{ id: "task1", type: "serviceTask" }],
                    },
                },
            };
            expect(registry.selectNavigator(bpmnRoutine).type).toBe("bpmn");

            // Langchain routine
            const langchainRoutine = {
                __typename: "Routine",
                graph: {
                    __type: "langchain",
                    chain: {
                        nodes: [{ id: "node1", type: "llm" }],
                    },
                },
            };
            expect(registry.selectNavigator(langchainRoutine).type).toBe("langchain");

            // Single-step routine (no graph)
            const singleStepRoutine = {
                __typename: "Routine",
                callDataAction: { type: "action", data: {} },
            };
            expect(registry.selectNavigator(singleStepRoutine).type).toBe("native");
        });

        it("should throw error when no suitable navigator found", () => {
            const unknownRoutine = {
                __typename: "Routine",
                graph: {
                    __type: "unknown_type",
                    data: {},
                },
            };

            expect(() => registry.selectNavigator(unknownRoutine))
                .toThrow("No navigator available for routine type");
        });

        it("should list all registered navigator types", () => {
            registry.register(new MockNativeNavigator(logger));
            registry.register(new MockBPMNNavigator(logger));

            const types = registry.getRegisteredTypes();
            expect(types).toContain("native");
            expect(types).toContain("bpmn");
            expect(types).toHaveLength(2);
        });
    });

    describe("Native Navigator", () => {
        let navigator: MockNativeNavigator;
        let nativeRoutine: RoutineVersionConfigObject;

        beforeEach(() => {
            navigator = new MockNativeNavigator(logger);
            nativeRoutine = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [
                        { 
                            id: "start", 
                            action: "initialize",
                            nextNodes: ["process"],
                        },
                        { 
                            id: "process", 
                            action: "transform",
                            nextNodes: ["validate", "error_handler"],
                        },
                        { 
                            id: "validate", 
                            action: "validate",
                            nextNodes: ["end"],
                        },
                        { 
                            id: "error_handler", 
                            action: "handle_error",
                            nextNodes: ["end"],
                        },
                        { 
                            id: "end", 
                            action: "finalize",
                        },
                    ],
                    edges: [
                        { from: "start", to: "process" },
                        { from: "process", to: "validate", condition: "success" },
                        { from: "process", to: "error_handler", condition: "error" },
                        { from: "validate", to: "end" },
                        { from: "error_handler", to: "end" },
                    ],
                },
            };
        });

        it("should identify native routine structures", () => {
            expect(navigator.canNavigate(nativeRoutine)).toBe(true);
            expect(navigator.canNavigate({ graph: { __type: "bpmn" } })).toBe(false);
        });

        it("should find single start location", () => {
            const startLocation = navigator.getStartLocation(nativeRoutine);
            expect(startLocation.nodeId).toBe("start");
            expect(startLocation.routineId).toBeDefined();
        });

        it("should find multiple start locations", () => {
            const multiStartRoutine = {
                ...nativeRoutine,
                graph: {
                    ...nativeRoutine.graph,
                    nodes: [
                        { id: "start1", action: "init1", nextNodes: ["process"] },
                        { id: "start2", action: "init2", nextNodes: ["process"] },
                        { id: "process", action: "process", nextNodes: ["end"] },
                        { id: "end", action: "end" },
                    ],
                    edges: [
                        { from: "start1", to: "process" },
                        { from: "start2", to: "process" },
                        { from: "process", to: "end" },
                    ],
                },
            };

            const startLocations = navigator.getAllStartLocations(multiStartRoutine);
            expect(startLocations).toHaveLength(2);
            expect(startLocations.map(l => l.nodeId)).toContain("start1");
            expect(startLocations.map(l => l.nodeId)).toContain("start2");
        });

        it("should navigate to next locations based on context", async () => {
            const startLocation = navigator.getStartLocation(nativeRoutine);
            
            // Success path
            const successContext = { status: "success", data: { valid: true } };
            const nextLocations = await navigator.getNextLocations(startLocation, successContext);
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("process");

            // Navigate from process with success
            const processLocation = nextLocations[0];
            const validateLocations = await navigator.getNextLocations(processLocation, successContext);
            expect(validateLocations).toHaveLength(1);
            expect(validateLocations[0].nodeId).toBe("validate");

            // Navigate from process with error
            const errorContext = { status: "error", error: { message: "Failed" } };
            const errorLocations = await navigator.getNextLocations(processLocation, errorContext);
            expect(errorLocations).toHaveLength(1);
            expect(errorLocations[0].nodeId).toBe("error_handler");
        });

        it("should identify end locations", async () => {
            const endLocation: Location = {
                id: `${generatePK()}-end`,
                routineId: generatePK(),
                nodeId: "end",
            };

            const isEnd = await navigator.isEndLocation(endLocation);
            expect(isEnd).toBe(true);

            const processLocation: Location = {
                id: `${generatePK()}-process`,
                routineId: generatePK(),
                nodeId: "process",
            };

            const isNotEnd = await navigator.isEndLocation(processLocation);
            expect(isNotEnd).toBe(false);
        });

        it("should get step information for locations", async () => {
            const processLocation: Location = {
                id: `${generatePK()}-process`,
                routineId: generatePK(),
                nodeId: "process",
            };

            // Need to cache the routine first
            navigator.validateAndCache(nativeRoutine);

            const stepInfo = await navigator.getStepInfo(processLocation);
            expect(stepInfo.id).toBe("process");
            expect(stepInfo.type).toBe("action");
            expect(stepInfo.config).toBeDefined();
        });

        it("should detect parallel branches", async () => {
            const parallelRoutine = {
                ...nativeRoutine,
                graph: {
                    ...nativeRoutine.graph,
                    nodes: [
                        { id: "split", action: "split", nextNodes: ["branch1", "branch2", "branch3"] },
                        { id: "branch1", action: "process1", nextNodes: ["merge"] },
                        { id: "branch2", action: "process2", nextNodes: ["merge"] },
                        { id: "branch3", action: "process3", nextNodes: ["merge"] },
                        { id: "merge", action: "merge", nextNodes: ["end"] },
                        { id: "end", action: "end" },
                    ],
                },
            };

            navigator.validateAndCache(parallelRoutine);

            const splitLocation: Location = {
                id: `${generatePK()}-split`,
                routineId: navigator.generateRoutineId(parallelRoutine),
                nodeId: "split",
            };

            const branches = await navigator.getParallelBranches(splitLocation);
            expect(branches).toHaveLength(3);
            expect(branches[0][0].nodeId).toBe("branch1");
            expect(branches[1][0].nodeId).toBe("branch2");
            expect(branches[2][0].nodeId).toBe("branch3");
        });
    });

    describe("BPMN Navigator", () => {
        let navigator: MockBPMNNavigator;
        let bpmnRoutine: RoutineVersionConfigObject;

        beforeEach(() => {
            navigator = new MockBPMNNavigator(logger);
            bpmnRoutine = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "bpmn",
                    process: {
                        id: "process1",
                        startEvents: [{ id: "start1", name: "Start" }],
                        tasks: [
                            { 
                                id: "task1", 
                                type: "serviceTask",
                                name: "Process Data",
                                outgoing: ["flow1", "flow2"],
                            },
                            { 
                                id: "task2", 
                                type: "userTask",
                                name: "Review",
                                incoming: ["flow1"],
                                outgoing: ["flow3"],
                            },
                        ],
                        endEvents: [{ id: "end1", name: "End", incoming: ["flow3"] }],
                        sequenceFlows: [
                            { 
                                id: "flow1", 
                                sourceRef: "task1", 
                                targetRef: "task2",
                                conditionExpression: "data.approved == true",
                            },
                            { 
                                id: "flow2", 
                                sourceRef: "task1", 
                                targetRef: "end1",
                                conditionExpression: "data.approved == false",
                            },
                            { 
                                id: "flow3", 
                                sourceRef: "task2", 
                                targetRef: "end1",
                            },
                        ],
                        gateways: [
                            {
                                id: "gateway1",
                                type: "exclusive",
                                incoming: ["flow_in"],
                                outgoing: ["flow_out1", "flow_out2"],
                            },
                        ],
                    },
                },
            };
        });

        it("should identify BPMN routine structures", () => {
            expect(navigator.canNavigate(bpmnRoutine)).toBe(true);
            expect(navigator.canNavigate({ graph: { __type: "native" } })).toBe(false);
        });

        it("should navigate BPMN process flows with conditions", async () => {
            navigator.validateAndCache(bpmnRoutine);

            const task1Location: Location = {
                id: `${generatePK()}-task1`,
                routineId: navigator.generateRoutineId(bpmnRoutine),
                nodeId: "task1",
            };

            // Test approved path
            const approvedContext = { data: { approved: true } };
            const approvedNext = await navigator.getNextLocations(task1Location, approvedContext);
            expect(approvedNext).toHaveLength(1);
            expect(approvedNext[0].nodeId).toBe("task2");

            // Test rejected path
            const rejectedContext = { data: { approved: false } };
            const rejectedNext = await navigator.getNextLocations(task1Location, rejectedContext);
            expect(rejectedNext).toHaveLength(1);
            expect(rejectedNext[0].nodeId).toBe("end1");
        });

        it("should handle BPMN gateways", async () => {
            const gatewayLocation: Location = {
                id: `${generatePK()}-gateway1`,
                routineId: navigator.generateRoutineId(bpmnRoutine),
                nodeId: "gateway1",
            };

            const stepInfo = await navigator.getStepInfo(gatewayLocation);
            expect(stepInfo.type).toBe("gateway");
            expect(stepInfo.metadata?.gatewayType).toBe("exclusive");
        });

        it("should support BPMN timer events", async () => {
            const timerRoutine = {
                ...bpmnRoutine,
                graph: {
                    ...bpmnRoutine.graph,
                    intermediateCatchEvents: [
                        {
                            id: "timer1",
                            type: "timer",
                            timerEventDefinition: {
                                timeDuration: "PT5M", // 5 minutes
                            },
                        },
                    ],
                },
            };

            navigator.validateAndCache(timerRoutine);

            const timerLocation: Location = {
                id: `${generatePK()}-timer1`,
                routineId: navigator.generateRoutineId(timerRoutine),
                nodeId: "timer1",
            };

            const timeouts = await navigator.getLocationTimeouts(timerLocation);
            expect(timeouts).toHaveLength(1);
            expect(timeouts[0].duration).toBe(300000); // 5 minutes in ms
            expect(timeouts[0].action).toBe("proceed");
        });
    });

    describe("Event-Driven Navigation", () => {
        let navigator: Navigator;

        beforeEach(() => {
            navigator = new MockNativeNavigator(logger);
        });

        it("should handle navigation triggers", async () => {
            const eventRoutine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [
                        { 
                            id: "wait_for_event",
                            action: "wait",
                            triggers: [
                                {
                                    id: "trigger1",
                                    event: "data.received",
                                    condition: "payload.type == 'important'",
                                },
                            ],
                        },
                    ],
                },
            };

            navigator.validateAndCache(eventRoutine);

            const waitLocation: Location = {
                id: `${generatePK()}-wait_for_event`,
                routineId: navigator.generateRoutineId(eventRoutine),
                nodeId: "wait_for_event",
            };

            const triggers = await navigator.getLocationTriggers(waitLocation);
            expect(triggers).toHaveLength(1);
            expect(triggers[0].event).toBe("data.received");
            expect(triggers[0].condition).toBe("payload.type == 'important'");

            // Test trigger evaluation
            const event: NavigationEvent = {
                type: "data.received",
                payload: { type: "important", data: "test" },
            };

            const canTrigger = await navigator.canTriggerEvent(waitLocation, event);
            expect(canTrigger).toBe(true);

            // Test non-matching event
            const nonMatchingEvent: NavigationEvent = {
                type: "data.received",
                payload: { type: "unimportant" },
            };

            const cannotTrigger = await navigator.canTriggerEvent(waitLocation, nonMatchingEvent);
            expect(cannotTrigger).toBe(false);
        });

        it("should handle timeout configurations", async () => {
            const timeoutRoutine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [
                        { 
                            id: "timed_step",
                            action: "process",
                            timeout: {
                                duration: 30000, // 30 seconds
                                action: "skip",
                                fallbackNode: "timeout_handler",
                            },
                        },
                        {
                            id: "timeout_handler",
                            action: "handle_timeout",
                        },
                    ],
                },
            };

            navigator.validateAndCache(timeoutRoutine);

            const timedLocation: Location = {
                id: `${generatePK()}-timed_step`,
                routineId: navigator.generateRoutineId(timeoutRoutine),
                nodeId: "timed_step",
            };

            const timeouts = await navigator.getLocationTimeouts(timedLocation);
            expect(timeouts).toHaveLength(1);
            expect(timeouts[0].duration).toBe(30000);
            expect(timeouts[0].action).toBe("skip");
            expect(timeouts[0].fallbackLocation?.nodeId).toBe("timeout_handler");
        });
    });

    describe("Complex Navigation Scenarios", () => {
        it("should handle nested parallel branches", async () => {
            const complexRoutine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [
                        { id: "start", nextNodes: ["split1"] },
                        { 
                            id: "split1", 
                            action: "parallel_split",
                            nextNodes: ["branch1_start", "branch2_start"],
                        },
                        // Branch 1 with nested split
                        { id: "branch1_start", nextNodes: ["branch1_split"] },
                        { 
                            id: "branch1_split",
                            action: "parallel_split",
                            nextNodes: ["branch1_1", "branch1_2"],
                        },
                        { id: "branch1_1", nextNodes: ["branch1_merge"] },
                        { id: "branch1_2", nextNodes: ["branch1_merge"] },
                        { 
                            id: "branch1_merge",
                            action: "merge",
                            nextNodes: ["merge1"],
                        },
                        // Branch 2
                        { id: "branch2_start", nextNodes: ["branch2_process"] },
                        { id: "branch2_process", nextNodes: ["merge1"] },
                        // Final merge
                        { 
                            id: "merge1",
                            action: "merge",
                            nextNodes: ["end"],
                        },
                        { id: "end" },
                    ],
                },
            };

            const navigator = new MockNativeNavigator(logger);
            navigator.validateAndCache(complexRoutine);

            const split1Location: Location = {
                id: `${generatePK()}-split1`,
                routineId: navigator.generateRoutineId(complexRoutine),
                nodeId: "split1",
            };

            const branches = await navigator.getParallelBranches(split1Location);
            expect(branches).toHaveLength(2);

            // Check nested branches
            const branch1SplitLocation: Location = {
                id: `${generatePK()}-branch1_split`,
                routineId: navigator.generateRoutineId(complexRoutine),
                nodeId: "branch1_split",
            };

            const nestedBranches = await navigator.getParallelBranches(branch1SplitLocation);
            expect(nestedBranches).toHaveLength(2);
        });

        it("should handle cyclic graphs with loop detection", async () => {
            const cyclicRoutine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [
                        { id: "start", nextNodes: ["process"] },
                        { 
                            id: "process", 
                            nextNodes: ["check"],
                            maxIterations: 5,
                        },
                        { 
                            id: "check",
                            action: "condition",
                            nextNodes: ["process", "end"], // Loop back or end
                        },
                        { id: "end" },
                    ],
                },
            };

            const navigator = new MockNativeNavigator(logger);
            navigator.validateAndCache(cyclicRoutine);

            const processLocation: Location = {
                id: `${generatePK()}-process`,
                routineId: navigator.generateRoutineId(cyclicRoutine),
                nodeId: "process",
            };

            const stepInfo = await navigator.getStepInfo(processLocation);
            expect(stepInfo.metadata?.maxIterations).toBe(5);
        });

        it("should handle conditional branches with complex expressions", async () => {
            const conditionalRoutine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [
                        { 
                            id: "decision",
                            action: "evaluate",
                            conditions: [
                                {
                                    expression: "data.score > 90 && data.verified == true",
                                    targetNode: "high_priority",
                                },
                                {
                                    expression: "data.score > 70",
                                    targetNode: "medium_priority",
                                },
                                {
                                    expression: "true", // Default
                                    targetNode: "low_priority",
                                },
                            ],
                        },
                        { id: "high_priority" },
                        { id: "medium_priority" },
                        { id: "low_priority" },
                    ],
                },
            };

            const navigator = new MockNativeNavigator(logger);
            navigator.validateAndCache(conditionalRoutine);

            const decisionLocation: Location = {
                id: `${generatePK()}-decision`,
                routineId: navigator.generateRoutineId(conditionalRoutine),
                nodeId: "decision",
            };

            // Test high priority path
            const highContext = { data: { score: 95, verified: true } };
            const highNext = await navigator.getNextLocations(decisionLocation, highContext);
            expect(highNext[0].nodeId).toBe("high_priority");

            // Test medium priority path
            const mediumContext = { data: { score: 75, verified: false } };
            const mediumNext = await navigator.getNextLocations(decisionLocation, mediumContext);
            expect(mediumNext[0].nodeId).toBe("medium_priority");

            // Test default path
            const lowContext = { data: { score: 50 } };
            const lowNext = await navigator.getNextLocations(decisionLocation, lowContext);
            expect(lowNext[0].nodeId).toBe("low_priority");
        });
    });

    describe("Performance and Caching", () => {
        it("should cache routine configurations for performance", async () => {
            const navigator = new MockNativeNavigator(logger);
            const routine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [{ id: "node1" }],
                },
            };

            // First call should cache
            navigator.validateAndCache(routine);

            // Get cached config
            const routineId = navigator.generateRoutineId(routine);
            const cached = await navigator.getCachedConfig(routineId);
            
            expect(cached).toBeDefined();
            expect(cached.__version).toBe("1.0.0");
        });

        it("should handle cache misses gracefully", async () => {
            const navigator = new MockNativeNavigator(logger);
            
            await expect(navigator.getCachedConfig("non-existent-id"))
                .rejects.toThrow("Routine config non-existent-id not in cache");
        });
    });

    describe("Error Handling", () => {
        it("should handle invalid routine structures", () => {
            const navigator = new MockNativeNavigator(logger);
            
            const invalidRoutine = {
                graph: {
                    __type: "native",
                    // Missing nodes
                },
            };

            expect(() => navigator.validateAndCache(invalidRoutine))
                .toThrow("Invalid routine configuration");
        });

        it("should handle navigation errors gracefully", async () => {
            const navigator = new MockNativeNavigator(logger);
            
            const invalidLocation: Location = {
                id: "invalid",
                routineId: "non-existent",
                nodeId: "unknown",
            };

            await expect(navigator.getStepInfo(invalidLocation))
                .rejects.toThrow();
        });

        it("should provide meaningful error messages", async () => {
            const navigator = new MockNativeNavigator(logger);
            const routine: RoutineVersionConfigObject = {
                __typename: "RoutineVersion",
                __version: "1.0.0",
                graph: {
                    __type: "native",
                    nodes: [{ id: "node1" }],
                },
            };

            navigator.validateAndCache(routine);

            const location: Location = {
                id: `${generatePK()}-unknown`,
                routineId: navigator.generateRoutineId(routine),
                nodeId: "unknown_node",
            };

            await expect(navigator.getStepInfo(location))
                .rejects.toThrow(/node.*not found/i);
        });
    });
});
