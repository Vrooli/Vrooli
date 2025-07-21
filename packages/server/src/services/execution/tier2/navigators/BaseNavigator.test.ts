import { beforeEach, describe, expect, test } from "vitest";
import { type RoutineVersionConfigObject } from "@vrooli/shared";
import { BaseNavigator } from "./BaseNavigator.js";
import { type Location, type StepInfo } from "../types.js";

/**
 * Concrete test implementation of BaseNavigator for testing
 * since BaseNavigator is abstract and cannot be instantiated directly
 */
class TestNavigator extends BaseNavigator {
    readonly type = "test";
    readonly version = "1.0.0";
    
    canNavigate(routine: unknown): boolean {
        try {
            this.validateRoutine(routine);
            return true;
        } catch {
            return false;
        }
    }
    
    getStartLocation(routine: unknown): Location {
        const config = this.validateRoutine(routine);
        return this.createLocation("start", `test_${config.__version}`);
    }
    
    getNextLocations(_routine: unknown, _current: Location, _context?: Record<string, unknown>): Location[] {
        return [];
    }
    
    isEndLocation(_routine: unknown, _location: Location): boolean {
        return false;
    }
    
    getStepInfo(_routine: unknown, location: Location): StepInfo {
        return {
            id: location.nodeId,
            name: `Step ${location.nodeId}`,
            type: "test",
        };
    }
    
    // Expose protected methods for testing
    public testCreateLocation(nodeId: string, routineId: string): Location {
        return this.createLocation(nodeId, routineId);
    }
    
    public testValidateRoutine(routine: unknown): RoutineVersionConfigObject {
        return this.validateRoutine(routine);
    }
}

describe("BaseNavigator", () => {
    let navigator: TestNavigator;

    beforeEach(() => {
        navigator = new TestNavigator();
    });

    // Helper function to create test routine configurations
    const createTestRoutine = (version = "1.0.0", additionalProps = {}): RoutineVersionConfigObject => ({
        __version: version,
        ...additionalProps,
    });

    describe("Abstract Properties", () => {
        test("should have correct type and version in concrete implementation", () => {
            expect(navigator.type).toBe("test");
            expect(navigator.version).toBe("1.0.0");
        });
    });

    describe("createLocation", () => {
        test("should create location with correct structure", () => {
            const nodeId = "step1";
            const routineId = "routine_123";
            
            const location = navigator.testCreateLocation(nodeId, routineId);
            
            expect(location).toEqual({
                id: "routine_123_step1",
                routineId: "routine_123",
                nodeId: "step1",
            });
        });

        test("should handle empty strings", () => {
            const location = navigator.testCreateLocation("", "");
            
            expect(location).toEqual({
                id: "_",
                routineId: "",
                nodeId: "",
            });
        });

        test("should handle special characters in IDs", () => {
            const nodeId = "step-1_2.3";
            const routineId = "routine.test_123";
            
            const location = navigator.testCreateLocation(nodeId, routineId);
            
            expect(location).toEqual({
                id: "routine.test_123_step-1_2.3",
                routineId: "routine.test_123",
                nodeId: "step-1_2.3",
            });
        });

        test("should handle unicode characters", () => {
            const nodeId = "步骤1";
            const routineId = "例程_123";
            
            const location = navigator.testCreateLocation(nodeId, routineId);
            
            expect(location).toEqual({
                id: "例程_123_步骤1",
                routineId: "例程_123",
                nodeId: "步骤1",
            });
        });

        test("should handle very long IDs", () => {
            const longNodeId = "a".repeat(1000);
            const longRoutineId = "b".repeat(1000);
            
            const location = navigator.testCreateLocation(longNodeId, longRoutineId);
            
            expect(location.nodeId).toBe(longNodeId);
            expect(location.routineId).toBe(longRoutineId);
            expect(location.id).toBe(`${longRoutineId}_${longNodeId}`);
            expect(location.id).toHaveLength(2001); // 1000 + 1 + 1000
        });

        test("should handle numeric strings", () => {
            const location = navigator.testCreateLocation("123", "456");
            
            expect(location).toEqual({
                id: "456_123",
                routineId: "456",
                nodeId: "123",
            });
        });
    });

    describe("validateRoutine", () => {
        test("should validate and return valid routine configuration", () => {
            const routine = createTestRoutine("2.0.0", {
                callDataAction: { schema: { tool: "test" } },
            });
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated).toEqual(routine);
            expect(validated.__version).toBe("2.0.0");
        });

        test("should throw error for null routine", () => {
            expect(() => navigator.testValidateRoutine(null))
                .toThrow("Invalid routine configuration for test navigator");
        });

        test("should throw error for undefined routine", () => {
            expect(() => navigator.testValidateRoutine(undefined))
                .toThrow("Invalid routine configuration for test navigator");
        });

        test("should throw error for non-object routine", () => {
            expect(() => navigator.testValidateRoutine("string"))
                .toThrow("Invalid routine configuration for test navigator");
            
            expect(() => navigator.testValidateRoutine(123))
                .toThrow("Invalid routine configuration for test navigator");
            
            expect(() => navigator.testValidateRoutine(true))
                .toThrow("Invalid routine configuration for test navigator");
            
            // Arrays are objects in JavaScript, so they pass the object check but fail the version check
            expect(() => navigator.testValidateRoutine([]))
                .toThrow("Routine missing version for test navigator");
        });

        test("should throw error for routine missing version", () => {
            const routineWithoutVersion = {
                callDataAction: { schema: { tool: "test" } },
            };
            
            expect(() => navigator.testValidateRoutine(routineWithoutVersion))
                .toThrow("Routine missing version for test navigator");
        });

        test("should throw error for routine with empty version", () => {
            const routineWithEmptyVersion = {
                __version: "",
                callDataAction: { schema: { tool: "test" } },
            };
            
            expect(() => navigator.testValidateRoutine(routineWithEmptyVersion))
                .toThrow("Routine missing version for test navigator");
        });

        test("should throw error for routine with null version", () => {
            const routineWithNullVersion = {
                __version: null,
                callDataAction: { schema: { tool: "test" } },
            };
            
            expect(() => navigator.testValidateRoutine(routineWithNullVersion))
                .toThrow("Routine missing version for test navigator");
        });

        test("should accept routine with minimal valid structure", () => {
            const minimalRoutine = { __version: "1.0.0" };
            
            const validated = navigator.testValidateRoutine(minimalRoutine);
            
            expect(validated.__version).toBe("1.0.0");
        });

        test("should handle routine with complex nested data", () => {
            const complexRoutine = createTestRoutine("1.5.0", {
                callDataAction: {
                    __version: "1.5.0",
                    schema: {
                        inputTemplate: JSON.stringify({
                            nested: {
                                deep: {
                                    value: "test",
                                    array: [1, 2, 3],
                                },
                            },
                        }),
                    },
                },
                formInput: {
                    __version: "1.5.0",
                    schema: {
                        elements: [
                            {
                                fieldName: "input1",
                                type: "text",
                                props: { defaultValue: "default" },
                            },
                        ],
                        containers: [],
                    },
                },
            });
            
            const validated = navigator.testValidateRoutine(complexRoutine);
            
            expect(validated.__version).toBe("1.5.0");
            expect(validated.callDataAction).toBeDefined();
            expect(validated.formInput).toBeDefined();
        });

        test("should preserve all properties in validated routine", () => {
            const routine = createTestRoutine("3.0.0", {
                callDataAction: {
                    __version: "3.0.0",
                    schema: {
                        inputTemplate: JSON.stringify({
                            prop1: "value1",
                            prop2: 42,
                            prop3: true,
                        }),
                    },
                },
                formOutput: {
                    __version: "3.0.0",
                    schema: {
                        elements: [
                            {
                                fieldName: "output1",
                                type: "text",
                                props: { label: "Output 1" },
                            },
                        ],
                        containers: [],
                    },
                },
            });
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated).toEqual(routine);
            expect(validated.__version).toBe("3.0.0");
            expect(validated.callDataAction).toBeDefined();
            expect(validated.formOutput).toBeDefined();
        });

        test("should handle version with special characters", () => {
            const routine = createTestRoutine("1.0.0-alpha.1");
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated.__version).toBe("1.0.0-alpha.1");
        });

        test("should handle routine with circular references gracefully", () => {
            const circularRef: any = { __version: "1.0.0" };
            circularRef.metadata = circularRef;
            
            // Should not throw during validation (the validation just checks for version existence)
            const validated = navigator.testValidateRoutine(circularRef);
            
            expect(validated.__version).toBe("1.0.0");
            expect(validated.metadata).toBe(validated); // Circular reference preserved
        });
    });

    describe("Error Messages", () => {
        test("should include navigator type in error messages", () => {
            expect(() => navigator.testValidateRoutine(null))
                .toThrow("Invalid routine configuration for test navigator");
            
            expect(() => navigator.testValidateRoutine({}))
                .toThrow("Routine missing version for test navigator");
        });

        test("should provide descriptive error for different invalid types", () => {
            const invalidInputs = [
                { input: "string", description: "string", expectedError: "Invalid routine configuration for test navigator" },
                { input: 123, description: "number", expectedError: "Invalid routine configuration for test navigator" },
                { input: true, description: "boolean", expectedError: "Invalid routine configuration for test navigator" },
                { input: [], description: "array", expectedError: "Routine missing version for test navigator" }, // Arrays pass object check
                { input: new Date(), description: "date object", expectedError: "Routine missing version for test navigator" }, // Dates pass object check
                { input: () => {}, description: "function", expectedError: "Invalid routine configuration for test navigator" },
            ];

            invalidInputs.forEach(({ input, description, expectedError }) => {
                expect(() => navigator.testValidateRoutine(input))
                    .toThrow(expectedError);
            });
        });
    });

    describe("Integration with Abstract Methods", () => {
        test("should work correctly with canNavigate implementation", () => {
            const validRoutine = createTestRoutine("1.0.0");
            const invalidRoutine = { missingVersion: true };
            
            expect(navigator.canNavigate(validRoutine)).toBe(true);
            expect(navigator.canNavigate(invalidRoutine)).toBe(false);
            expect(navigator.canNavigate(null)).toBe(false);
        });

        test("should work correctly with getStartLocation implementation", () => {
            const routine = createTestRoutine("2.5.0");
            
            const startLocation = navigator.getStartLocation(routine);
            
            expect(startLocation.nodeId).toBe("start");
            expect(startLocation.routineId).toBe("test_2.5.0");
            expect(startLocation.id).toBe("test_2.5.0_start");
        });

        test("should throw error in getStartLocation for invalid routine", () => {
            expect(() => navigator.getStartLocation(null))
                .toThrow("Invalid routine configuration for test navigator");
            
            expect(() => navigator.getStartLocation({}))
                .toThrow("Routine missing version for test navigator");
        });
    });

    describe("Edge Cases and Boundary Conditions", () => {
        test("should handle routine with extremely long version string", () => {
            const longVersion = "1.0.0-" + "a".repeat(1000);
            const routine = createTestRoutine(longVersion);
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated.__version).toBe(longVersion);
        });

        test("should handle routine with version containing unicode", () => {
            const unicodeVersion = "1.0.0-测试版本";
            const routine = createTestRoutine(unicodeVersion);
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated.__version).toBe(unicodeVersion);
        });

        test("should handle routine with numeric version", () => {
            const numericVersion: any = 1.5;
            const routine = { __version: numericVersion };
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated.__version).toBe(1.5);
        });

        test("should handle routine with boolean version", () => {
            const booleanVersion: any = true;
            const routine = { __version: booleanVersion };
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated.__version).toBe(true);
        });

        test("should handle routine with zero version", () => {
            const zeroVersion = "0";
            const routine = createTestRoutine(zeroVersion);
            
            const validated = navigator.testValidateRoutine(routine);
            
            expect(validated.__version).toBe("0");
        });

        test("should handle very large routine object", () => {
            const largeRoutine = createTestRoutine("1.0.0", {
                callDataAction: {
                    __version: "1.0.0",
                    schema: {
                        inputTemplate: JSON.stringify({
                            largeData: Array(1000).fill(0).map((_, i) => ({ [`key${i}`]: `value${i}` })),
                        }),
                    },
                },
            });
            
            const validated = navigator.testValidateRoutine(largeRoutine);
            
            expect(validated.__version).toBe("1.0.0");
            expect(validated.callDataAction).toBeDefined();
            if (validated.callDataAction?.schema?.inputTemplate) {
                const parsed = JSON.parse(validated.callDataAction.schema.inputTemplate);
                expect(parsed.largeData).toHaveLength(1000);
                expect(parsed.largeData[0].key0).toBe("value0");
                expect(parsed.largeData[999].key999).toBe("value999");
            }
        });
    });

    describe("Type Safety and TypeScript Integration", () => {
        test("should properly type the returned config", () => {
            const routine = createTestRoutine("1.0.0", {
                callDataAction: {
                    __version: "1.0.0",
                    schema: {
                        inputTemplate: "test template",
                    },
                },
            });
            
            const validated = navigator.testValidateRoutine(routine);
            
            // TypeScript should recognize these properties
            expect(validated.__version).toBeDefined();
            expect(typeof validated.__version).toBe("string");
            
            // Should allow access to optional properties
            if (validated.callDataAction) {
                expect(validated.callDataAction).toBeDefined();
            }
        });

        test("should handle routine with all possible RoutineVersionConfigObject properties", () => {
            const fullRoutine: RoutineVersionConfigObject = {
                __version: "1.0.0",
                callDataAction: { 
                    __version: "1.0.0", 
                    schema: { 
                        inputTemplate: "test template",
                    },
                },
                callDataApi: { 
                    __version: "1.0.0", 
                    schema: { 
                        url: "http://test.com",
                        method: "GET",
                        headers: {},
                    },
                },
                callDataCode: { 
                    __version: "1.0.0", 
                    schema: { 
                        code: "console.log('test')",
                    },
                },
                callDataGenerate: { 
                    __version: "1.0.0", 
                    schema: { 
                        model: { name: "test", value: "test" }, 
                        prompt: "test",
                    },
                },
                formInput: { 
                    __version: "1.0.0", 
                    schema: { 
                        elements: [], 
                        containers: [],
                    },
                },
                formOutput: { 
                    __version: "1.0.0", 
                    schema: { 
                        elements: [], 
                        containers: [],
                    },
                },
                graph: { 
                    __version: "1.0.0", 
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
            
            const validated = navigator.testValidateRoutine(fullRoutine);
            
            expect(validated).toEqual(fullRoutine);
            expect(validated.__version).toBe("1.0.0");
        });
    });
});
