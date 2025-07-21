import { describe, expect, test } from "vitest";
import { getNavigator, getNavigatorByType, getSupportedNavigatorTypes } from "./navigatorFactory.js";

describe("navigatorFactory", () => {
    describe("getNavigator", () => {
        test("should detect sequential routine", () => {
            const sequentialRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "Sequential",
                    schema: {
                        steps: [
                            { id: "step1", name: "Step 1", subroutineId: "sub1" },
                            { id: "step2", name: "Step 2", subroutineId: "sub2" },
                        ],
                    },
                },
            };

            const navigator = getNavigator(sequentialRoutine);
            expect(navigator).toBeTruthy();
            expect(navigator?.type).toBe("sequential");
        });

        test("should detect BPMN routine", () => {
            const bpmnRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    schema: {
                        data: `<?xml version="1.0" encoding="UTF-8"?>
                        <bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
                            <bpmn:process id="Process_1">
                                <bpmn:startEvent id="StartEvent_1" />
                            </bpmn:process>
                        </bpmn:definitions>`,
                    },
                },
            };

            const navigator = getNavigator(bpmnRoutine);
            expect(navigator).toBeTruthy();
            expect(navigator?.type).toBe("bpmn");
        });

        test("should return null for unsupported routine", () => {
            const unsupportedRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "Unsupported",
                },
            };

            const navigator = getNavigator(unsupportedRoutine);
            expect(navigator).toBeNull();
        });

        // Enhanced edge case tests
        test("should handle null input", () => {
            const navigator = getNavigator(null);
            expect(navigator).toBeNull();
        });

        test("should handle undefined input", () => {
            const navigator = getNavigator(undefined);
            expect(navigator).toBeNull();
        });

        test("should handle empty object", () => {
            const navigator = getNavigator({});
            expect(navigator).toBeNull();
        });

        test("should handle malformed routine without graph", () => {
            const malformedRoutine = {
                __version: "1.0.0",
                // Missing graph property
            };

            const navigator = getNavigator(malformedRoutine);
            expect(navigator).toBeNull();
        });

        test("should handle routine with null graph", () => {
            const routineWithNullGraph = {
                __version: "1.0.0",
                graph: null,
            };

            const navigator = getNavigator(routineWithNullGraph);
            expect(navigator).toBeNull();
        });

        test("should handle routine with missing graph type", () => {
            const routineWithoutType = {
                __version: "1.0.0",
                graph: {
                    schema: {
                        steps: [{ id: "step1" }],
                    },
                },
            };

            const navigator = getNavigator(routineWithoutType);
            expect(navigator).toBeNull();
        });

        test("should handle sequential routine with empty steps", () => {
            const emptySequentialRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "Sequential",
                    schema: {
                        steps: [],
                    },
                },
            };

            const navigator = getNavigator(emptySequentialRoutine);
            // Should still return navigator even with empty steps, as it's structurally valid
            expect(navigator).toBeTruthy();
            expect(navigator?.type).toBe("sequential");
        });

        test("should handle BPMN routine with minimal XML", () => {
            const minimalBpmnRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    schema: {
                        data: "<?xml version=\"1.0\"?><bpmn:definitions xmlns:bpmn=\"http://www.omg.org/spec/BPMN/20100524/MODEL\"/>",
                    },
                },
            };

            const navigator = getNavigator(minimalBpmnRoutine);
            expect(navigator).toBeTruthy();
            expect(navigator?.type).toBe("bpmn");
        });

        test("should handle BPMN routine with empty data", () => {
            const bpmnWithEmptyData = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    schema: {
                        data: "",
                    },
                },
            };

            const navigator = getNavigator(bpmnWithEmptyData);
            // BPMN navigator should be created but may not be fully functional with empty data
            // This tests the factory's ability to create navigators regardless of data validity
            expect(navigator).toBeTruthy();
            expect(navigator?.type).toBe("bpmn");
        });

        test("should prioritize sequential over BPMN when both might match", () => {
            // This tests the factory's ordering - sequential is checked first
            const ambiguousRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "Sequential", // Explicitly sequential
                    schema: {
                        steps: [{ id: "step1" }],
                        // Could potentially have BPMN-like properties too
                        data: "some-bpmn-data",
                    },
                },
            };

            const navigator = getNavigator(ambiguousRoutine);
            expect(navigator).toBeTruthy();
            expect(navigator?.type).toBe("sequential");
        });

        test("should handle primitive inputs gracefully", () => {
            expect(getNavigator("string")).toBeNull();
            expect(getNavigator(123)).toBeNull();
            expect(getNavigator(true)).toBeNull();
            expect(getNavigator([])).toBeNull();
        });
    });

    describe("getNavigatorByType", () => {
        test("should return sequential navigator", () => {
            const navigator = getNavigatorByType("sequential");
            expect(navigator?.type).toBe("sequential");
        });

        test("should return BPMN navigator", () => {
            const navigator = getNavigatorByType("bpmn");
            expect(navigator?.type).toBe("bpmn");
        });

        test("should return null for unknown type", () => {
            const navigator = getNavigatorByType("unknown");
            expect(navigator).toBeNull();
        });

        // Enhanced edge case tests
        test("should handle case sensitivity", () => {
            expect(getNavigatorByType("Sequential")).toBeNull();
            expect(getNavigatorByType("SEQUENTIAL")).toBeNull();
            expect(getNavigatorByType("BPMN")).toBeNull();
            expect(getNavigatorByType("Bpmn")).toBeNull();
        });

        test("should handle empty string", () => {
            const navigator = getNavigatorByType("");
            expect(navigator).toBeNull();
        });

        test("should handle whitespace", () => {
            expect(getNavigatorByType(" sequential ")).toBeNull();
            expect(getNavigatorByType("\t")).toBeNull();
            expect(getNavigatorByType("\n")).toBeNull();
        });

        test("should create new instances each time", () => {
            const nav1 = getNavigatorByType("sequential");
            const nav2 = getNavigatorByType("sequential");
            
            expect(nav1).not.toBe(nav2); // Different instances
            expect(nav1?.type).toBe(nav2?.type); // Same type
        });

        test("should handle all supported types", () => {
            const supportedTypes = getSupportedNavigatorTypes();
            
            for (const type of supportedTypes) {
                const navigator = getNavigatorByType(type);
                expect(navigator).toBeTruthy();
                expect(navigator?.type).toBe(type);
            }
        });
    });

    describe("getSupportedNavigatorTypes", () => {
        test("should return supported types", () => {
            const types = getSupportedNavigatorTypes();
            expect(types).toEqual(["sequential", "bpmn"]);
        });

        test("should return array of strings", () => {
            const types = getSupportedNavigatorTypes();
            expect(Array.isArray(types)).toBe(true);
            expect(types.every(type => typeof type === "string")).toBe(true);
        });

        test("should return same array each time", () => {
            const types1 = getSupportedNavigatorTypes();
            const types2 = getSupportedNavigatorTypes();
            
            expect(types1).toEqual(types2);
        });

        test("should include all factory-supported types", () => {
            const types = getSupportedNavigatorTypes();
            
            // Each supported type should be creatable via getNavigatorByType
            for (const type of types) {
                const navigator = getNavigatorByType(type);
                expect(navigator).toBeTruthy();
            }
        });
    });
});
