import { describe, it, expect, beforeEach } from "vitest";
import { createMockLogger } from "../../../../../__test/util.js";
import { SequentialNavigator } from "./sequentialNavigator.js";
import { type RoutineVersionConfigObject } from "@vrooli/shared";

describe("SequentialNavigator", () => {
    let navigator: SequentialNavigator;
    let logger: ReturnType<typeof createMockLogger>;

    beforeEach(() => {
        logger = createMockLogger();
        navigator = new SequentialNavigator(logger);
    });

    describe("canNavigate", () => {
        it("should return true for valid sequential config", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "Sequential",
                    schema: {
                        steps: [
                            {
                                id: "step1",
                                name: "First Step",
                                subroutineId: "sub1",
                                inputMap: { "input1": "subInput1" },
                                outputMap: { "subOutput1": "output1" },
                            },
                        ],
                        rootContext: {
                            inputMap: { "mainInput": "input1" },
                            outputMap: { "output1": "mainOutput" },
                        },
                    },
                },
            };

            expect(navigator.canNavigate(routine)).toBe(true);
        });

        it("should return false for non-sequential config", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "BPMN-2.0",
                    schema: {
                        __format: "xml",
                        data: "<bpmn:process />",
                        activityMap: {},
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            expect(navigator.canNavigate(routine)).toBe(false);
        });

        it("should return false for empty steps", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "Sequential",
                    schema: {
                        steps: [],
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            expect(navigator.canNavigate(routine)).toBe(false);
        });
    });

    describe("getStartLocation", () => {
        it("should return location with nodeId 0", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "Sequential",
                    schema: {
                        steps: [
                            {
                                id: "step1",
                                name: "First Step",
                                subroutineId: "sub1",
                                inputMap: {},
                                outputMap: {},
                            },
                        ],
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            const location = navigator.getStartLocation(routine);
            expect(location.nodeId).toBe("0");
            expect(location.routineId).toMatch(/^sequential_config_/);
        });
    });

    describe("getNextLocations", () => {
        const createRoutineWithSteps = (steps: number, mode?: "sequential" | "parallel") => ({
            __version: "1.0",
            graph: {
                __version: "1.0",
                __type: "Sequential",
                schema: {
                    steps: Array.from({ length: steps }, (_, i) => ({
                        id: `step${i + 1}`,
                        name: `Step ${i + 1}`,
                        subroutineId: `sub${i + 1}`,
                        inputMap: {},
                        outputMap: {},
                    })),
                    rootContext: {
                        inputMap: {},
                        outputMap: {},
                    },
                    executionMode: mode,
                },
            },
        } as RoutineVersionConfigObject);

        it("should return next step in sequential mode", async () => {
            const routine = createRoutineWithSteps(3);
            const startLocation = navigator.getStartLocation(routine);
            
            const nextLocations = await navigator.getNextLocations(startLocation, {});
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("1");
        });

        it("should return empty array at end of sequence", async () => {
            const routine = createRoutineWithSteps(2);
            const startLocation = navigator.getStartLocation(routine);
            const lastLocation = { ...startLocation, nodeId: "1" };
            
            const nextLocations = await navigator.getNextLocations(lastLocation, {});
            expect(nextLocations).toHaveLength(0);
        });

        it("should return empty array in parallel mode (handled by getParallelBranches)", async () => {
            const routine = createRoutineWithSteps(3, "parallel");
            const startLocation = navigator.getStartLocation(routine);
            
            const nextLocations = await navigator.getNextLocations(startLocation, {});
            expect(nextLocations).toHaveLength(0); // Parallel mode uses getParallelBranches instead
        });

        it("should skip steps with skip condition", async () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "Sequential",
                    schema: {
                        steps: [
                            {
                                id: "step1",
                                name: "Step 1",
                                subroutineId: "sub1",
                                inputMap: {},
                                outputMap: {},
                            },
                            {
                                id: "step2",
                                name: "Step 2",
                                subroutineId: "sub2",
                                inputMap: {},
                                outputMap: {},
                                skipCondition: "context.skipStep2",
                            },
                            {
                                id: "step3",
                                name: "Step 3",
                                subroutineId: "sub3",
                                inputMap: {},
                                outputMap: {},
                            },
                        ],
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            const startLocation = navigator.getStartLocation(routine);
            const context = { skipStep2: true };
            
            const nextLocations = await navigator.getNextLocations(startLocation, context);
            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("2"); // Should skip to step 3
        });
    });

    describe("isEndLocation", () => {
        it("should return true for last step in sequential mode", async () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "Sequential",
                    schema: {
                        steps: [
                            {
                                id: "step1",
                                name: "Step 1",
                                subroutineId: "sub1",
                                inputMap: {},
                                outputMap: {},
                            },
                            {
                                id: "step2",
                                name: "Step 2",
                                subroutineId: "sub2",
                                inputMap: {},
                                outputMap: {},
                            },
                        ],
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            const location = navigator.getStartLocation(routine);
            const lastLocation = { ...location, nodeId: "1" };
            
            expect(await navigator.isEndLocation(lastLocation)).toBe(true);
            expect(await navigator.isEndLocation(location)).toBe(false);
        });
    });

    describe("getStepInfo", () => {
        it("should return correct step info", async () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0",
                graph: {
                    __version: "1.0",
                    __type: "Sequential",
                    schema: {
                        steps: [
                            {
                                id: "analyze-data",
                                name: "Analyze Data",
                                description: "Analyzes the input data",
                                subroutineId: "data-analyzer-v1",
                                inputMap: { "rawData": "analyzerInput" },
                                outputMap: { "analyzerOutput": "analysis" },
                                retryPolicy: {
                                    maxAttempts: 3,
                                    backoffMs: 1000,
                                },
                            },
                        ],
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            const location = navigator.getStartLocation(routine);
            const stepInfo = await navigator.getStepInfo(location);
            
            expect(stepInfo.id).toBe("analyze-data");
            expect(stepInfo.name).toBe("Analyze Data");
            expect(stepInfo.description).toBe("Analyzes the input data");
            expect(stepInfo.type).toBe("subroutine");
            expect(stepInfo.config).toEqual({
                subroutineId: "data-analyzer-v1",
                inputMap: { "rawData": "analyzerInput" },
                outputMap: { "analyzerOutput": "analysis" },
                retryPolicy: {
                    maxAttempts: 3,
                    backoffMs: 1000,
                },
            });
        });
    });
});
