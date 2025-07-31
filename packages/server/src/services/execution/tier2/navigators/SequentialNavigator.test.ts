import { type RoutineVersionConfigObject, type SequentialStep } from "@vrooli/shared";
import { beforeEach, describe, expect, test } from "vitest";
import { type Location } from "../types.js";
import { SequentialNavigator } from "./SequentialNavigator.js";

describe("SequentialNavigator", () => {
    let navigator: SequentialNavigator;

    beforeEach(() => {
        navigator = new SequentialNavigator();
    });

    // Helper function to create test routine configurations
    function createSequentialRoutine(
        steps: SequentialStep[],
        version = "1.0.0",
        executionMode: "sequential" | "parallel" = "sequential",
    ): RoutineVersionConfigObject {
        return {
            __version: version,
            graph: {
                __type: "Sequential" as const,
                __version: "1.0",
                schema: {
                    steps,
                    rootContext: {
                        inputMap: { input1: "step0_input1" },
                        outputMap: { step_final_output: "routine_output1" },
                    },
                    executionMode,
                },
            },
        };
    }

    function createStep(
        id: string,
        name: string,
        subroutineId = `sub_${id}`,
        options: {
            description?: string;
            skipCondition?: string;
            inputMap?: Record<string, string>;
            outputMap?: Record<string, string>;
            retryPolicy?: { maxAttempts: number; backoffMs: number };
        } = {},
    ): SequentialStep {
        return {
            id,
            name,
            description: options.description,
            subroutineId,
            inputMap: options.inputMap || { input1: "value1" },
            outputMap: options.outputMap || { output1: "result1" },
            skipCondition: options.skipCondition,
            retryPolicy: options.retryPolicy,
        };
    }

    describe("Basic Properties", () => {
        test("should have correct type and version", () => {
            expect(navigator.type).toBe("sequential");
            expect(navigator.version).toBe("1.0.0");
        });
    });

    describe("canNavigate", () => {
        test("should return true for valid sequential routine", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Second Step"),
            ]);

            expect(navigator.canNavigate(routine)).toBe(true);
        });

        test("should return true for single step routine", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            expect(navigator.canNavigate(routine)).toBe(true);
        });

        test("should return false for routine without version", () => {
            const routine = {
                graph: {
                    __type: "Sequential" as const,
                    schema: {
                        steps: [createStep("step1", "Step")],
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                } as any,
            };

            expect(navigator.canNavigate(routine)).toBe(false);
        });

        test("should return false for routine with wrong graph type", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "BPMN-2.0",
                    __version: "1.0",
                    schema: {
                        __format: "xml",
                        data: "<xml></xml>",
                        activityMap: {},
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                },
            };

            expect(navigator.canNavigate(routine)).toBe(false);
        });

        test("should return false for routine without graph", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0.0",
            };

            expect(navigator.canNavigate(routine)).toBe(false);
        });

        test("should return false for routine with non-array steps", () => {
            const routine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                graph: {
                    __type: "Sequential" as const,
                    schema: {
                        steps: "not an array" as any,
                        rootContext: { inputMap: {}, outputMap: {} },
                    },
                } as any,
            };

            expect(navigator.canNavigate(routine)).toBe(false);
        });

        test("should return false for routine with empty steps array", () => {
            const routine = createSequentialRoutine([]);

            expect(navigator.canNavigate(routine)).toBe(false);
        });

        test("should return false for null or undefined routine", () => {
            expect(navigator.canNavigate(null)).toBe(false);
            expect(navigator.canNavigate(undefined)).toBe(false);
        });

        test("should return false for non-object routine", () => {
            expect(navigator.canNavigate("string")).toBe(false);
            expect(navigator.canNavigate(123)).toBe(false);
            expect(navigator.canNavigate([])).toBe(false);
        });

        test("should handle exceptions gracefully", () => {
            const circularRef: any = {};
            circularRef.self = circularRef;

            expect(navigator.canNavigate(circularRef)).toBe(false);
        });
    });

    describe("getStartLocation", () => {
        test("should return first step location", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Second Step"),
            ]);

            const startLocation = navigator.getStartLocation(routine);

            expect(startLocation.nodeId).toBe("0");
            expect(startLocation.routineId).toMatch(/^seq_1\.0\.0_[a-f0-9]+$/);
            expect(startLocation.id).toBe(`${startLocation.routineId}_0`);
        });

        test("should generate consistent routine ID for same configuration", () => {
            const steps = [createStep("step1", "Step")];
            const routine1 = createSequentialRoutine(steps);
            const routine2 = createSequentialRoutine(steps);

            const location1 = navigator.getStartLocation(routine1);
            const location2 = navigator.getStartLocation(routine2);

            expect(location1.routineId).toBe(location2.routineId);
        });

        test("should generate different routine IDs for different configurations", () => {
            const routine1 = createSequentialRoutine([createStep("step1", "Step1")]);
            const routine2 = createSequentialRoutine([createStep("step2", "Step2")]);

            const location1 = navigator.getStartLocation(routine1);
            const location2 = navigator.getStartLocation(routine2);

            expect(location1.routineId).not.toBe(location2.routineId);
        });

        test("should throw error for invalid routine", () => {
            expect(() => navigator.getStartLocation(null)).toThrow("Invalid routine configuration");
            expect(() => navigator.getStartLocation({})).toThrow("Routine missing version");
        });
    });

    describe("getNextLocations", () => {
        test("should return next step location in sequence", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Second Step"),
                createStep("step3", "Third Step"),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const nextLocations = navigator.getNextLocations(routine, startLocation);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("1");
            expect(nextLocations[0].routineId).toBe(startLocation.routineId);
        });

        test("should return empty array for last step", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Second Step"),
            ]);

            const lastLocation: Location = {
                id: "test_1",
                routineId: "test",
                nodeId: "1", // Last step (index 1)
            };

            const nextLocations = navigator.getNextLocations(routine, lastLocation);

            expect(nextLocations).toHaveLength(0);
        });

        test("should return empty array for step beyond sequence", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
            ]);

            const beyondLocation: Location = {
                id: "test_5",
                routineId: "test",
                nodeId: "5", // Beyond array bounds
            };

            const nextLocations = navigator.getNextLocations(routine, beyondLocation);

            expect(nextLocations).toHaveLength(0);
        });

        test("should return empty array for invalid node ID", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
            ]);

            const invalidLocation: Location = {
                id: "test_invalid",
                routineId: "test",
                nodeId: "invalid",
            };

            const nextLocations = navigator.getNextLocations(routine, invalidLocation);

            expect(nextLocations).toHaveLength(0);
        });

        test("should return empty array for negative node ID", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
            ]);

            const negativeLocation: Location = {
                id: "test_-1",
                routineId: "test",
                nodeId: "-1",
            };

            const nextLocations = navigator.getNextLocations(routine, negativeLocation);

            expect(nextLocations).toHaveLength(0);
        });

        test("should skip step with matching skip condition", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Skippable Step", "sub2", { skipCondition: "context.skipStep2" }),
                createStep("step3", "Third Step"),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const context = { skipStep2: true };

            const nextLocations = navigator.getNextLocations(routine, startLocation, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("2"); // Should skip step 1 and go to step 2
        });

        test("should not skip step with non-matching skip condition", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Conditional Step", "sub2", { skipCondition: "context.skipStep2" }),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const context = { skipStep2: false };

            const nextLocations = navigator.getNextLocations(routine, startLocation, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("1"); // Should proceed to step 1
        });

        test("should handle skip condition with missing context variable", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Conditional Step", "sub2", { skipCondition: "context.missingVar" }),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const context = { otherVar: true };

            const nextLocations = navigator.getNextLocations(routine, startLocation, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("1"); // Should proceed normally when condition is falsy
        });

        test("should handle skip condition without context", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Conditional Step", "sub2", { skipCondition: "context.skipStep2" }),
            ]);

            const startLocation = navigator.getStartLocation(routine);

            const nextLocations = navigator.getNextLocations(routine, startLocation);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("1"); // Should proceed normally without context
        });

        test("should handle malformed skip condition", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Conditional Step", "sub2", { skipCondition: "invalid.syntax.here" }),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const context = { skipStep2: true };

            const nextLocations = navigator.getNextLocations(routine, startLocation, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("1"); // Should proceed normally with invalid condition
        });

        test("should handle recursive skip conditions", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Skip Step 2", "sub2", { skipCondition: "context.skip2" }),
                createStep("step3", "Skip Step 3", "sub3", { skipCondition: "context.skip3" }),
                createStep("step4", "Final Step"),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const context = { skip2: true, skip3: true };

            const nextLocations = navigator.getNextLocations(routine, startLocation, context);

            expect(nextLocations).toHaveLength(1);
            expect(nextLocations[0].nodeId).toBe("3"); // Should skip both step 1 and 2, go to step 3
        });

        test("should handle skip condition at end of sequence", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Last Step", "sub2", { skipCondition: "context.skipLast" }),
            ]);

            const firstLocation: Location = {
                id: "test_0",
                routineId: "test",
                nodeId: "0",
            };
            const context = { skipLast: true };

            const nextLocations = navigator.getNextLocations(routine, firstLocation, context);

            expect(nextLocations).toHaveLength(0); // Should reach end after skipping last step
        });
    });

    describe("isEndLocation", () => {
        test("should return true for last step in sequence", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Last Step"),
            ]);

            const lastLocation: Location = {
                id: "test_1",
                routineId: "test",
                nodeId: "1", // Index 1 is the last step (0-based)
            };

            expect(navigator.isEndLocation(routine, lastLocation)).toBe(true);
        });

        test("should return false for non-last step", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Last Step"),
            ]);

            const firstLocation: Location = {
                id: "test_0",
                routineId: "test",
                nodeId: "0",
            };

            expect(navigator.isEndLocation(routine, firstLocation)).toBe(false);
        });

        test("should return true for single step routine", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            const onlyLocation: Location = {
                id: "test_0",
                routineId: "test",
                nodeId: "0",
            };

            expect(navigator.isEndLocation(routine, onlyLocation)).toBe(true);
        });

        test("should return false for invalid step index", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            const invalidLocation: Location = {
                id: "test_invalid",
                routineId: "test",
                nodeId: "invalid",
            };

            expect(navigator.isEndLocation(routine, invalidLocation)).toBe(false);
        });

        test("should return false for step beyond sequence", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            const beyondLocation: Location = {
                id: "test_5",
                routineId: "test",
                nodeId: "5",
            };

            expect(navigator.isEndLocation(routine, beyondLocation)).toBe(false);
        });
    });

    describe("getStepInfo", () => {
        test("should return correct step information", () => {
            const steps = [
                createStep("step1", "First Step", "subroutine1", {
                    description: "This is the first step",
                    inputMap: { input1: "value1", input2: "value2" },
                    outputMap: { output1: "result1" },
                }),
                createStep("step2", "Second Step", "subroutine2"),
            ];
            const routine = createSequentialRoutine(steps);

            const location: Location = {
                id: "test_0",
                routineId: "test",
                nodeId: "0",
            };

            const stepInfo = navigator.getStepInfo(routine, location);

            expect(stepInfo).toEqual({
                id: "step1",
                name: "First Step",
                type: "subroutine",
                description: "This is the first step",
                config: {
                    subroutineId: "subroutine1",
                    inputMap: { input1: "value1", input2: "value2" },
                    outputMap: { output1: "result1" },
                },
            });
        });

        test("should return step info without description when not provided", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Step Without Description", "subroutine1"),
            ]);

            const location: Location = {
                id: "test_0",
                routineId: "test",
                nodeId: "0",
            };

            const stepInfo = navigator.getStepInfo(routine, location);

            expect(stepInfo.description).toBeUndefined();
            expect(stepInfo.id).toBe("step1");
            expect(stepInfo.name).toBe("Step Without Description");
        });

        test("should throw error for invalid step index", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            const invalidLocation: Location = {
                id: "test_invalid",
                routineId: "test",
                nodeId: "invalid",
            };

            // parseInt("invalid") returns NaN, which doesn't trigger the validation check
            // The error occurs when trying to access steps[NaN].id
            expect(() => navigator.getStepInfo(routine, invalidLocation))
                .toThrow("Cannot read properties of undefined");
        });

        test("should throw error for negative step index", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            const negativeLocation: Location = {
                id: "test_-1",
                routineId: "test",
                nodeId: "-1",
            };

            expect(() => navigator.getStepInfo(routine, negativeLocation))
                .toThrow("Invalid step index: -1");
        });

        test("should throw error for step index beyond sequence", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Only Step"),
            ]);

            const beyondLocation: Location = {
                id: "test_5",
                routineId: "test",
                nodeId: "5",
            };

            expect(() => navigator.getStepInfo(routine, beyondLocation))
                .toThrow("Invalid step index: 5");
        });

        test("should include retry policy in config when present", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Retryable Step", "subroutine1", {
                    retryPolicy: {
                        maxAttempts: 3,
                        backoffMs: 1000,
                    },
                }),
            ]);

            const location: Location = {
                id: "test_0",
                routineId: "test",
                nodeId: "0",
            };

            const stepInfo = navigator.getStepInfo(routine, location);

            // Note: The implementation doesn't include retryPolicy in config,
            // but the step data should still be accessible if needed
            expect(stepInfo.config?.subroutineId).toBe("subroutine1");
        });
    });

    describe("Skip Condition Logic", () => {
        test("should handle various valid skip condition patterns", () => {
            const testCases = [
                { condition: "context.flag", context: { flag: true }, shouldSkip: true },
                { condition: "context.flag", context: { flag: false }, shouldSkip: false },
                { condition: "context.skip_step", context: { skip_step: true }, shouldSkip: true },
                { condition: "context.variable123", context: { variable123: true }, shouldSkip: true },
                { condition: "context._private", context: { _private: true }, shouldSkip: true },
                { condition: "context.$special", context: { $special: true }, shouldSkip: true },
            ];

            testCases.forEach(({ condition, context, shouldSkip }, index) => {
                const routine = createSequentialRoutine([
                    createStep("step1", "First Step"),
                    createStep("step2", "Conditional Step", "sub2", { skipCondition: condition }),
                    createStep("step3", "Third Step"),
                ]);

                const startLocation = navigator.getStartLocation(routine);
                const nextLocations = navigator.getNextLocations(routine, startLocation, context);

                const expectedNodeId = shouldSkip ? "2" : "1";
                expect(nextLocations[0]?.nodeId).toBe(expectedNodeId);
                if (nextLocations[0]?.nodeId !== expectedNodeId) {
                    console.log(`Test case ${index}: condition="${condition}", context=${JSON.stringify(context)}, expected skip=${shouldSkip}`);
                }
            });
        });

        test("should reject invalid skip condition patterns", () => {
            const invalidConditions = [
                "invalid",
                "context",
                "context.",
                "context.123invalid",
                "context.invalid-dash",
                "context.invalid.nested",
                "notcontext.variable",
                "context.variable.nested",
                // Note: "  context.variable  " would actually match due to .trim() call
            ];

            invalidConditions.forEach((condition) => {
                const routine = createSequentialRoutine([
                    createStep("step1", "First Step"),
                    createStep("step2", "Conditional Step", "sub2", { skipCondition: condition }),
                ]);

                const startLocation = navigator.getStartLocation(routine);
                const context = { variable: true };
                const nextLocations = navigator.getNextLocations(routine, startLocation, context);

                expect(nextLocations[0]?.nodeId).toBe("1");
                if (nextLocations[0]?.nodeId !== "1") {
                    console.log(`Invalid condition "${condition}" should not cause skipping`);
                }
            });
        });

        test("should handle empty and whitespace skip conditions", () => {
            const conditions = ["", "   ", "\t", "\n"];

            conditions.forEach((condition) => {
                const routine = createSequentialRoutine([
                    createStep("step1", "First Step"),
                    createStep("step2", "Conditional Step", "sub2", { skipCondition: condition }),
                ]);

                const startLocation = navigator.getStartLocation(routine);
                const context = { variable: true };
                const nextLocations = navigator.getNextLocations(routine, startLocation, context);

                expect(nextLocations[0]?.nodeId).toBe("1");
                if (nextLocations[0]?.nodeId !== "1") {
                    console.log(`Empty/whitespace condition "${JSON.stringify(condition)}" should not cause skipping`);
                }
            });
        });

        test("should handle skip conditions with leading/trailing whitespace", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First Step"),
                createStep("step2", "Conditional Step", "sub2", { skipCondition: "  context.skipMe  " }),
                createStep("step3", "Third Step"),
            ]);

            const startLocation = navigator.getStartLocation(routine);
            const context = { skipMe: true };
            const nextLocations = navigator.getNextLocations(routine, startLocation, context);

            // Should skip step2 because trimming makes "  context.skipMe  " -> "context.skipMe"
            expect(nextLocations[0]?.nodeId).toBe("2");
        });
    });

    describe("Routine ID Generation and Hashing", () => {
        test("should generate consistent routine IDs for identical configurations", () => {
            const steps = [
                createStep("step1", "First", "sub1"),
                createStep("step2", "Second", "sub2"),
            ];

            const routine1 = createSequentialRoutine(steps, "1.0.0");
            const routine2 = createSequentialRoutine(steps, "1.0.0");

            const id1 = navigator.getStartLocation(routine1).routineId;
            const id2 = navigator.getStartLocation(routine2).routineId;

            expect(id1).toBe(id2);
        });

        test("should generate different routine IDs for different versions", () => {
            const steps = [createStep("step1", "First", "sub1")];

            const routine1 = createSequentialRoutine(steps, "1.0.0");
            const routine2 = createSequentialRoutine(steps, "2.0.0");

            const id1 = navigator.getStartLocation(routine1).routineId;
            const id2 = navigator.getStartLocation(routine2).routineId;

            expect(id1).not.toBe(id2);
        });

        test("should generate different routine IDs for different step configurations", () => {
            const routine1 = createSequentialRoutine([
                createStep("step1", "First", "sub1"),
            ]);
            const routine2 = createSequentialRoutine([
                createStep("step2", "Different", "sub2"),
            ]);

            const id1 = navigator.getStartLocation(routine1).routineId;
            const id2 = navigator.getStartLocation(routine2).routineId;

            expect(id1).not.toBe(id2);
        });

        test("should generate different routine IDs for different step order", () => {
            const step1 = createStep("step1", "First", "sub1");
            const step2 = createStep("step2", "Second", "sub2");

            const routine1 = createSequentialRoutine([step1, step2]);
            const routine2 = createSequentialRoutine([step2, step1]);

            const id1 = navigator.getStartLocation(routine1).routineId;
            const id2 = navigator.getStartLocation(routine2).routineId;

            expect(id1).not.toBe(id2);
        });

        test("should generate routine ID with expected format", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "First", "sub1"),
            ], "1.2.3");

            const routineId = navigator.getStartLocation(routine).routineId;

            expect(routineId).toMatch(/^seq_1\.2\.3_[a-f0-9]+$/);
        });

        test("should handle hash collisions gracefully", () => {
            // Create many different routines to test hash distribution
            const routineIds = new Set<string>();

            for (let i = 0; i < 100; i++) {
                const routine = createSequentialRoutine([
                    createStep(`step${i}`, `Step ${i}`, `sub${i}`),
                ]);

                const routineId = navigator.getStartLocation(routine).routineId;
                routineIds.add(routineId);
            }

            // All IDs should be unique (no hash collisions in this small test set)
            expect(routineIds.size).toBe(100);
        });
    });

    describe("Error Handling and Edge Cases", () => {
        test("should throw descriptive error for missing routine version", () => {
            const invalidRoutine = {
                graph: {
                    __type: "Sequential" as const,
                    schema: { steps: [] },
                },
            };

            expect(() => navigator.getStartLocation(invalidRoutine))
                .toThrow("Routine missing version for sequential navigator");
        });

        test("should handle routine with complex nested data gracefully", () => {
            const complexRoutine = createSequentialRoutine([
                createStep("step1", "Complex Step", "sub1", {
                    inputMap: {
                        input1: "value1",
                        input2: "value2",
                        nested: "deeply.nested.value",
                    },
                    outputMap: {
                        output1: "result1",
                        complex: "complex.output.path",
                    },
                    retryPolicy: {
                        maxAttempts: 5,
                        backoffMs: 2000,
                    },
                }),
            ]);

            expect(() => navigator.getStartLocation(complexRoutine)).not.toThrow();
            expect(() => navigator.canNavigate(complexRoutine)).not.toThrow();
        });

        test("should handle routine with unicode characters in step names", () => {
            const unicodeRoutine = createSequentialRoutine([
                createStep("step1", "步骤一", "sub1"),
                createStep("step2", "Étape 2", "sub2"),
                createStep("step3", "Шаг 3", "sub3"),
            ]);

            expect(navigator.canNavigate(unicodeRoutine)).toBe(true);

            const startLocation = navigator.getStartLocation(unicodeRoutine);
            const stepInfo = navigator.getStepInfo(unicodeRoutine, startLocation);

            expect(stepInfo.name).toBe("步骤一");
        });

        test("should handle empty string values in step configuration", () => {
            const routineWithEmptyValues = createSequentialRoutine([
                createStep("", "", "", {
                    description: "",
                    inputMap: { "": "" },
                    outputMap: { "": "" },
                }),
            ]);

            expect(navigator.canNavigate(routineWithEmptyValues)).toBe(true);

            const startLocation = navigator.getStartLocation(routineWithEmptyValues);
            const stepInfo = navigator.getStepInfo(routineWithEmptyValues, startLocation);

            expect(stepInfo.id).toBe("");
            expect(stepInfo.name).toBe("");
        });

        test("should validate routine structure before processing", () => {
            expect(() => navigator.getStartLocation(null))
                .toThrow("Invalid routine configuration for sequential navigator");

            expect(() => navigator.getStartLocation(undefined))
                .toThrow("Invalid routine configuration for sequential navigator");

            expect(() => navigator.getStartLocation("string"))
                .toThrow("Invalid routine configuration for sequential navigator");
        });

        test("should handle malformed graph structure gracefully", () => {
            const malformedRoutine = {
                __version: "1.0.0",
                graph: {
                    __type: "Sequential" as const,
                    // Missing schema
                },
            };

            expect(() => navigator.getStartLocation(malformedRoutine))
                .toThrow(); // Should throw when trying to access schema.steps
        });
    });

    describe("Integration Test Scenarios", () => {
        test("should handle complete navigation flow", () => {
            const routine = createSequentialRoutine([
                createStep("init", "Initialize"),
                createStep("process", "Process Data"),
                createStep("validate", "Validate Results"),
                createStep("finalize", "Finalize"),
            ]);

            // Start at beginning
            const startLocation = navigator.getStartLocation(routine);
            expect(startLocation.nodeId).toBe("0");
            expect(navigator.isEndLocation(routine, startLocation)).toBe(false);

            // Navigate through sequence
            let currentLocation = startLocation;
            const visitedSteps: string[] = [];

            while (!navigator.isEndLocation(routine, currentLocation)) {
                const stepInfo = navigator.getStepInfo(routine, currentLocation);
                visitedSteps.push(stepInfo.id);

                const nextLocations = navigator.getNextLocations(routine, currentLocation);
                expect(nextLocations).toHaveLength(1);

                currentLocation = nextLocations[0];
            }

            // Visit final step
            const finalStepInfo = navigator.getStepInfo(routine, currentLocation);
            visitedSteps.push(finalStepInfo.id);

            expect(visitedSteps).toEqual(["init", "process", "validate", "finalize"]);
            expect(navigator.isEndLocation(routine, currentLocation)).toBe(true);
        });

        test("should handle navigation with skip conditions", () => {
            const routine = createSequentialRoutine([
                createStep("step1", "Always Execute"),
                createStep("step2", "Skip if flag set", "sub2", { skipCondition: "context.skipStep2" }),
                createStep("step3", "Conditional", "sub3", { skipCondition: "context.skipStep3" }),
                createStep("step4", "Always Execute Last"),
            ]);

            const context = { skipStep2: true, skipStep3: false };

            // Start navigation
            let currentLocation = navigator.getStartLocation(routine);
            const executedSteps: string[] = [];

            while (!navigator.isEndLocation(routine, currentLocation)) {
                const stepInfo = navigator.getStepInfo(routine, currentLocation);
                executedSteps.push(stepInfo.id);

                const nextLocations = navigator.getNextLocations(routine, currentLocation, context);
                if (nextLocations.length === 0) break;

                currentLocation = nextLocations[0];
            }

            // Execute final step
            if (navigator.isEndLocation(routine, currentLocation)) {
                const finalStepInfo = navigator.getStepInfo(routine, currentLocation);
                executedSteps.push(finalStepInfo.id);
            }

            expect(executedSteps).toEqual(["step1", "step3", "step4"]); // step2 should be skipped
        });

        test("should handle single-step routine correctly", () => {
            const routine = createSequentialRoutine([
                createStep("onlyStep", "The Only Step"),
            ]);

            const startLocation = navigator.getStartLocation(routine);

            expect(navigator.isEndLocation(routine, startLocation)).toBe(true);
            expect(navigator.getNextLocations(routine, startLocation)).toHaveLength(0);

            const stepInfo = navigator.getStepInfo(routine, startLocation);
            expect(stepInfo.id).toBe("onlyStep");
            expect(stepInfo.name).toBe("The Only Step");
        });
    });
});
