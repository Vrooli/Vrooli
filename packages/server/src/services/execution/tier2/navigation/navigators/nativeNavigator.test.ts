import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { type Logger } from "winston";
import { NativeNavigator } from "./nativeNavigator.js";
import { type RoutineVersionConfigObject } from "@vrooli/shared";

// Mock dependencies
vi.mock("../../../../../redisConn.js", () => ({
    CacheService: {
        get: vi.fn(() => ({
            raw: vi.fn(() => Promise.resolve({
                get: vi.fn(),
                set: vi.fn(),
                del: vi.fn(),
            })),
        })),
    },
}));

const mockLogger: Logger = {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
} as Logger;

describe("NativeNavigator", () => {
    let navigator: NativeNavigator;

    beforeEach(() => {
        navigator = new NativeNavigator(mockLogger);
        vi.clearAllMocks();
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("canNavigate", () => {
        it("should accept valid BPMN routine configs", () => {
            const bpmnRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __version: "1.0.0",
                    __type: "BPMN-2.0",
                    schema: {
                        __format: "xml" as const,
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="process1">
                                <bpmn:startEvent id="start" />
                                <bpmn:task id="task1" name="Test Task" />
                                <bpmn:endEvent id="end" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            expect(navigator.canNavigate(bpmnRoutine)).toBe(true);
        });

        it("should accept valid single-step routine configs", () => {
            const singleStepRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                callDataAction: {
                    __version: "1.0.0",
                    schema: {
                        toolName: "ResourceManage" as const,
                        inputTemplate: JSON.stringify({ op: "find" }),
                        outputMapping: {},
                    },
                },
            };

            expect(navigator.canNavigate(singleStepRoutine)).toBe(true);
        });

        it("should reject invalid routine configs", () => {
            const invalidRoutine = {
                // Missing __version
                callDataAction: {
                    __version: "1.0.0",
                    schema: {
                        toolName: "ResourceManage",
                        inputTemplate: "{}",
                        outputMapping: {},
                    },
                },
            };

            expect(navigator.canNavigate(invalidRoutine)).toBe(false);
        });

        it("should reject configs with no call data or graph", () => {
            const emptyRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                // No callData* or graph
            };

            expect(navigator.canNavigate(emptyRoutine)).toBe(false);
        });
    });

    describe("getStartLocation", () => {
        it("should get start location for BPMN routine", () => {
            const bpmnRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __version: "1.0.0",
                    __type: "BPMN-2.0",
                    schema: {
                        __format: "xml" as const,
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="process1">
                                <bpmn:startEvent id="startEvent1" name="Start" />
                                <bpmn:task id="task1" name="Test Task" />
                                <bpmn:endEvent id="endEvent1" name="End" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            const location = navigator.getStartLocation(bpmnRoutine);
            expect(location.nodeId).toBe("startEvent1");
            expect(location.routineId).toMatch(/^bpmn_config_[a-f0-9]+$/);
        });

        it("should get start location for single-step routine", () => {
            const singleStepRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                callDataGenerate: {
                    __version: "1.0.0",
                    schema: {
                        prompt: "Generate a response",
                    },
                },
            };

            const location = navigator.getStartLocation(singleStepRoutine);
            expect(location.nodeId).toBe("single_step");
            expect(location.routineId).toMatch(/^single-step_config_[a-f0-9]+$/);
        });

        it("should throw error for unsupported routine", () => {
            const invalidRoutine = {
                __version: "1.0.0",
                // No supported config
            };

            expect(() => navigator.getStartLocation(invalidRoutine)).toThrow();
        });
    });

    describe("navigation delegation", () => {
        it("should delegate to appropriate navigator based on routine type", async () => {
            const bpmnRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __version: "1.0.0",
                    __type: "BPMN-2.0",
                    schema: {
                        __format: "xml" as const,
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="process1">
                                <bpmn:startEvent id="startEvent1" />
                                <bpmn:endEvent id="endEvent1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                        activityMap: {},
                        rootContext: {
                            inputMap: {},
                            outputMap: {},
                        },
                    },
                },
            };

            // Get start location (this should work through BPMN navigator)
            const startLocation = navigator.getStartLocation(bpmnRoutine);
            expect(startLocation.nodeId).toBe("startEvent1");

            // Test other navigation methods work
            expect(navigator.canNavigate(bpmnRoutine)).toBe(true);
        });

        it("should handle single-step routine navigation", () => {
            const singleStepRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                callDataApi: {
                    __version: "1.0.0",
                    schema: {
                        endpoint: "https://api.example.com",
                        method: "GET",
                    },
                },
            };

            const startLocation = navigator.getStartLocation(singleStepRoutine);
            expect(startLocation.nodeId).toBe("single_step");
        });
    });

    describe("error handling", () => {
        it("should handle malformed routine configs gracefully", () => {
            const malformedRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    // Missing required schema
                },
            };

            expect(navigator.canNavigate(malformedRoutine)).toBe(false);
        });

        it("should provide helpful error messages", () => {
            const unsupportedRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "UnsupportedFormat",
                    schema: {},
                },
            };

            expect(() => navigator.getStartLocation(unsupportedRoutine)).toThrow(
                "No appropriate navigator found for routine configuration",
            );
        });
    });
});
