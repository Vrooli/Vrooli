import { describe, it, beforeEach, vi, expect } from "vitest";
import { type Logger } from "winston";
import { type ExecutionContext } from "@vrooli/shared";
import { LegacyInputHandler } from "../legacyInputHandler.js";

describe("LegacyInputHandler", () => {
    let handler: LegacyInputHandler;
    let mockLogger: Logger;
    let baseContext: ExecutionContext;

    beforeEach(() => {
        mockLogger = {
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        } as any;

        baseContext = {
            stepId: "test-step-123",
            stepType: "analyze_data",
            inputs: {},
            config: {},
            resources: {
                models: [],
                tools: [],
                apis: [],
                credits: 5000,
            },
            history: {
                recentSteps: [],
                totalExecutions: 0,
                successRate: 1.0,
            },
            constraints: {},
        };

        handler = new LegacyInputHandler(mockLogger);
    });

    describe("handleMissingInputs", () => {
        it("should return context unchanged when no inputs are missing", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    existingInput1: "value1",
                    existingInput2: "value2",
                },
                config: {
                    expectedInputs: {
                        existingInput1: { type: "Text" },
                        existingInput2: { type: "Text" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result).toEqual(context);
            expect(mockLogger.debug).not.toHaveBeenCalled();
        });

        it("should return context unchanged when no expected inputs are defined", async () => {
            const context = {
                ...baseContext,
                inputs: {},
                config: {},
            };

            const result = await handler.handleMissingInputs(context);

            expect(result).toEqual(context);
        });

        it("should generate missing Boolean inputs", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    existingInput: "test",
                },
                config: {
                    expectedInputs: {
                        existingInput: { type: "Text" },
                        missingBoolean: { type: "Boolean" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs).toHaveProperty("missingBoolean");
            expect(typeof result.inputs.missingBoolean).toBe("boolean");
            expect(result.inputs.missingBoolean).toBe(true); // Should be true when other inputs exist
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyInputHandler] Generating missing inputs",
                expect.objectContaining({
                    stepId: "test-step-123",
                    missingInputs: ["missingBoolean"],
                })
            );
        });

        it("should generate missing Boolean as false when no other inputs exist", async () => {
            const context = {
                ...baseContext,
                inputs: {},
                config: {
                    expectedInputs: {
                        missingBoolean: { type: "Boolean" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs.missingBoolean).toBe(false); // Should be false when no other inputs exist
        });

        it("should generate missing Integer inputs", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    input1: "test",
                    input2: "test",
                },
                config: {
                    expectedInputs: {
                        input1: { type: "Text" },
                        input2: { type: "Text" },
                        missingInteger: { type: "Integer" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs).toHaveProperty("missingInteger");
            expect(typeof result.inputs.missingInteger).toBe("number");
            expect(result.inputs.missingInteger).toBe(20); // 2 existing inputs * 10
            expect(Number.isInteger(result.inputs.missingInteger as number)).toBe(true);
        });

        it("should generate missing Number inputs", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    input1: "test",
                    input2: "test",
                    input3: "test",
                },
                config: {
                    expectedInputs: {
                        input1: { type: "Text" },
                        input2: { type: "Text" },
                        input3: { type: "Text" },
                        missingNumber: { type: "Number" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs).toHaveProperty("missingNumber");
            expect(typeof result.inputs.missingNumber).toBe("number");
            expect(result.inputs.missingNumber).toBeCloseTo(9.42); // 3 existing inputs * 3.14
        });

        it("should generate missing Text inputs with contextual information", async () => {
            const context = {
                ...baseContext,
                stepType: "data_analysis",
                inputs: {},
                config: {
                    expectedInputs: {
                        missingText: { type: "Text" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs).toHaveProperty("missingText");
            expect(typeof result.inputs.missingText).toBe("string");
            expect(result.inputs.missingText).toBe("AI-generated value for missingText in data_analysis");
        });

        it("should use default values when available", async () => {
            const context = {
                ...baseContext,
                inputs: {},
                config: {
                    expectedInputs: {
                        inputWithDefault: {
                            type: "Text",
                            defaultValue: "default_value",
                        },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs.inputWithDefault).toBe("default_value");
        });

        it("should handle multiple missing inputs of different types", async () => {
            const context = {
                ...baseContext,
                stepType: "complex_analysis",
                inputs: {
                    existingInput: "test",
                },
                config: {
                    expectedInputs: {
                        existingInput: { type: "Text" },
                        missingBoolean: { type: "Boolean" },
                        missingInteger: { type: "Integer" },
                        missingNumber: { type: "Number" },
                        missingText: { type: "Text" },
                        inputWithDefault: {
                            type: "Text",
                            defaultValue: "default",
                        },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            // Verify all missing inputs were generated
            expect(result.inputs).toHaveProperty("existingInput", "test");
            expect(result.inputs).toHaveProperty("missingBoolean", true);
            expect(result.inputs).toHaveProperty("missingInteger", 10);
            expect(result.inputs).toHaveProperty("missingNumber", 3.14);
            expect(result.inputs).toHaveProperty("missingText", "AI-generated value for missingText in complex_analysis");
            expect(result.inputs).toHaveProperty("inputWithDefault", "default");

            // Verify logging
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyInputHandler] Generating missing inputs",
                expect.objectContaining({
                    stepId: "test-step-123",
                    missingInputs: ["missingBoolean", "missingInteger", "missingNumber", "missingText", "inputWithDefault"],
                })
            );

            // Verify individual input generation logging
            expect(mockLogger.debug).toHaveBeenCalledWith(
                "[LegacyInputHandler] Generated input value",
                expect.objectContaining({
                    inputName: "missingBoolean",
                    type: "Boolean",
                    value: true,
                })
            );
        });

        it("should handle unknown input types with fallback generation", async () => {
            const context = {
                ...baseContext,
                inputs: {},
                config: {
                    expectedInputs: {
                        unknownType: { type: "CustomType" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs).toHaveProperty("unknownType");
            expect(result.inputs.unknownType).toBe("Generated value for unknownType");
        });

        it("should handle inputs with undefined values", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    definedInput: "value",
                    undefinedInput: undefined,
                },
                config: {
                    expectedInputs: {
                        definedInput: { type: "Text" },
                        undefinedInput: { type: "Text" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs.definedInput).toBe("value");
            expect(result.inputs.undefinedInput).toBe("AI-generated value for undefinedInput in analyze_data");
        });

        it("should preserve original inputs while adding generated ones", async () => {
            const originalInputs = {
                input1: "original1",
                input2: 42,
                input3: true,
                nested: { key: "value" },
            };

            const context = {
                ...baseContext,
                inputs: originalInputs,
                config: {
                    expectedInputs: {
                        input1: { type: "Text" },
                        input2: { type: "Integer" },
                        input3: { type: "Boolean" },
                        nested: { type: "Object" },
                        newInput: { type: "Text" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            // Original inputs should be preserved
            expect(result.inputs.input1).toBe("original1");
            expect(result.inputs.input2).toBe(42);
            expect(result.inputs.input3).toBe(true);
            expect(result.inputs.nested).toEqual({ key: "value" });

            // New input should be generated
            expect(result.inputs.newInput).toBe("AI-generated value for newInput in analyze_data");
        });

        it("should handle edge cases gracefully", async () => {
            const context = {
                ...baseContext,
                stepType: "", // Empty step type
                inputs: {},
                config: {
                    expectedInputs: {
                        testInput: { type: "Text" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            expect(result.inputs.testInput).toBe("AI-generated value for testInput in reasoning");
        });
    });

    describe("legacy pattern preservation", () => {
        it("should maintain legacy Boolean generation logic", async () => {
            // Test with no existing inputs - should generate false
            const contextNoInputs = {
                ...baseContext,
                inputs: {},
                config: {
                    expectedInputs: {
                        boolInput: { type: "Boolean" },
                    },
                },
            };

            const resultNoInputs = await handler.handleMissingInputs(contextNoInputs);
            expect(resultNoInputs.inputs.boolInput).toBe(false);

            // Test with existing inputs - should generate true
            const contextWithInputs = {
                ...baseContext,
                inputs: { existing: "value" },
                config: {
                    expectedInputs: {
                        existing: { type: "Text" },
                        boolInput: { type: "Boolean" },
                    },
                },
            };

            const resultWithInputs = await handler.handleMissingInputs(contextWithInputs);
            expect(resultWithInputs.inputs.boolInput).toBe(true);
        });

        it("should maintain legacy numeric scaling patterns", async () => {
            const context = {
                ...baseContext,
                inputs: {
                    input1: "a",
                    input2: "b",
                    input3: "c",
                },
                config: {
                    expectedInputs: {
                        input1: { type: "Text" },
                        input2: { type: "Text" },
                        input3: { type: "Text" },
                        integerInput: { type: "Integer" },
                        numberInput: { type: "Number" },
                    },
                },
            };

            const result = await handler.handleMissingInputs(context);

            // Legacy pattern: availableInputs * 10 for integers
            expect(result.inputs.integerInput).toBe(30); // 3 * 10

            // Legacy pattern: availableInputs * 3.14 for numbers
            expect(result.inputs.numberInput).toBeCloseTo(9.42); // 3 * 3.14
        });

        it("should maintain legacy text generation patterns", async () => {
            const testCases = [
                { stepType: "data_analysis", inputName: "description" },
                { stepType: "validation", inputName: "criteria" },
                { stepType: "optimization", inputName: "objective" },
            ];

            for (const testCase of testCases) {
                const context = {
                    ...baseContext,
                    stepType: testCase.stepType,
                    inputs: {},
                    config: {
                        expectedInputs: {
                            [testCase.inputName]: { type: "Text" },
                        },
                    },
                };

                const result = await handler.handleMissingInputs(context);
                const expectedText = `AI-generated value for ${testCase.inputName} in ${testCase.stepType}`;

                expect(result.inputs[testCase.inputName]).toBe(expectedText);
            }
        });
    });
});