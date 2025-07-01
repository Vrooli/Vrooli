import { getNavigator, getNavigatorByType, getSupportedTypes } from "./navigatorFactory.js";

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
    });

    describe("getSupportedTypes", () => {
        test("should return supported types", () => {
            const types = getSupportedTypes();
            expect(types).toEqual(["sequential", "bpmn"]);
        });
    });
});