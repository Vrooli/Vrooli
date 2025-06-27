import { describe, it, expect, beforeEach, vi } from "vitest";
import { ValidationEngine } from "./validationEngine.js";
import * as yup from "yup";
import { type Logger } from "winston";
import { type IEventBus } from "../../../events/index.js";

// Mock logger
const mockLogger = {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    child: vi.fn(() => mockLogger),
} as unknown as Logger;

// Mock event bus
const mockEventBus = {
    publish: vi.fn().mockResolvedValue({ success: true }),
    subscribe: vi.fn().mockResolvedValue("sub-123"),
    unsubscribe: vi.fn().mockResolvedValue(undefined),
    publishBarrierSync: vi.fn().mockResolvedValue({ success: true, responses: [], timedOut: false, duration: 100 }),
    start: vi.fn().mockResolvedValue(undefined),
    stop: vi.fn().mockResolvedValue(undefined),
} as unknown as IEventBus;

describe("ValidationEngine", () => {
    let validationEngine: ValidationEngine;

    beforeEach(() => {
        vi.clearAllMocks();
        validationEngine = new ValidationEngine(mockLogger, mockEventBus);
    });

    describe("constructor", () => {
        it("should initialize with event bus", () => {
            const engine = new ValidationEngine(mockLogger, mockEventBus);
            expect(engine).toBeDefined();
        });

        it("should handle missing event bus gracefully", () => {
            const engine = new ValidationEngine(mockLogger);
            expect(engine).toBeDefined();
        });
    });

    describe("validateOutputs", () => {
        it("should validate simple outputs successfully", async () => {
            const outputs = {
                result: "success",
                data: { count: 42 },
            };

            const result = await validationEngine.validateOutputs(outputs);

            expect(result.valid).toBe(true);
            expect(result.data).toEqual(outputs);
            expect(result.errors).toHaveLength(0);
            
            // Check that event was published with correct structure
            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/validation_requested",
                    data: expect.objectContaining({
                        outputs,
                        schemaValid: true,
                        schemaErrors: [],
                    }),
                }),
            );
        });

        it("should validate outputs with schema", async () => {
            const outputs = {
                result: "success",
                count: 42,
            };

            const schema = yup.object({
                result: yup.string().required(),
                count: yup.number().required(),
            });

            const result = await validationEngine.validateOutputs(outputs, schema);

            expect(result.valid).toBe(true);
            expect(result.data).toEqual(outputs);
            expect(result.errors).toHaveLength(0);
        });

        it("should fail validation with invalid schema", async () => {
            const outputs = {
                result: "success",
                count: "not-a-number", // Invalid type
            };

            const schema = yup.object({
                result: yup.string().required(),
                count: yup.number().required(),
            });

            const result = await validationEngine.validateOutputs(outputs, schema);

            expect(result.valid).toBe(false);
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.errors[0]).toContain("count");
        });

        it("should validate outputs with execution context", async () => {
            const outputs = { message: "Hello World" };
            const context = {
                executionId: "exec-123",
                stepId: "step-456",
                routineId: "routine-789",
                tier: 3 as const,
                strategy: "deterministic",
            };

            const result = await validationEngine.validateOutputs(outputs, undefined, context);

            expect(result.valid).toBe(true);
            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/validation_requested",
                    data: expect.objectContaining({
                        executionId: "exec-123",
                        stepId: "step-456",
                        routineId: "routine-789",
                        tier: 3,
                        strategy: "deterministic",
                        outputs,
                    }),
                }),
            );
        });

        it("should handle empty outputs", async () => {
            const outputs = {};

            const result = await validationEngine.validateOutputs(outputs);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Output data is empty");
        });

        it("should handle outputs with null values", async () => {
            const outputs = {
                validField: "good",
                nullField: null,
            };

            const result = await validationEngine.validateOutputs(outputs);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Output field 'nullField' is null");
        });

        it("should emit completion event", async () => {
            const outputs = { status: "complete" };

            await validationEngine.validateOutputs(outputs);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/validation_completed",
                    data: expect.objectContaining({
                        valid: true,
                        outputs,
                        errors: [],
                    }),
                }),
            );
        });
    });

    describe("validateDataTypes", () => {
        it("should validate correct data types", async () => {
            const outputs = {
                name: "test",
                count: 42,
                active: true,
                items: ["a", "b", "c"],
            };

            const expectedTypes = {
                name: "string",
                count: "number",
                active: "boolean",
                items: "array",
            };

            const result = await validationEngine.validateDataTypes(outputs, expectedTypes);

            expect(result.valid).toBe(true);
            expect(result.errors).toHaveLength(0);
        });

        it("should detect type mismatches", async () => {
            const outputs = {
                name: 123, // Should be string
                count: "42", // Should be number
            };

            const expectedTypes = {
                name: "string",
                count: "number",
            };

            const result = await validationEngine.validateDataTypes(outputs, expectedTypes);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Output 'name' type mismatch: expected string, got number");
            expect(result.errors).toContain("Output 'count' type mismatch: expected number, got string");
        });

        it("should handle missing required fields", async () => {
            const outputs = {
                name: "test",
                // count is missing
            };

            const expectedTypes = {
                name: "string",
                count: "number", // Required
            };

            const result = await validationEngine.validateDataTypes(outputs, expectedTypes);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Required output 'count' is missing");
        });

        it("should handle optional fields", async () => {
            const outputs = {
                name: "test",
                // optionalCount is missing but marked optional
            };

            const expectedTypes = {
                name: "string",
                optionalCount: "optional",
            };

            const result = await validationEngine.validateDataTypes(outputs, expectedTypes);

            expect(result.valid).toBe(true);
            expect(result.errors).toHaveLength(0);
        });
    });

    describe("validateExecutionOutputs", () => {
        it("should validate execution outputs with context", async () => {
            const outputs = { result: "success", data: "processed" };
            const expectedOutputs = {
                result: { required: true },
                data: { required: true },
            };
            const context = {
                executionId: "exec-123",
                stepId: "step-456",
                routineId: "routine-789",
                tier: 3 as const,
            };

            const result = await validationEngine.validateExecutionOutputs(
                outputs,
                expectedOutputs,
                context,
            );

            expect(result.valid).toBe(true);
            expect(result.data).toEqual(outputs);

            // Should emit pre and post validation events
            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/pre_validation",
                    data: expect.objectContaining({
                        executionId: "exec-123",
                        outputs,
                        expectedOutputs,
                    }),
                }),
            );

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/post_validation",
                    data: expect.objectContaining({
                        executionId: "exec-123",
                        valid: true,
                        outputs,
                    }),
                }),
            );
        });

        it("should detect missing required outputs", async () => {
            const outputs = { result: "success" }; // Missing 'data'
            const expectedOutputs = {
                result: { required: true },
                data: { required: true },
            };

            const result = await validationEngine.validateExecutionOutputs(outputs, expectedOutputs);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Required output 'data' is missing");
        });

        it("should handle optional outputs", async () => {
            const outputs = { result: "success" }; // Missing optional 'data'
            const expectedOutputs = {
                result: { required: true },
                data: { required: false },
            };

            const result = await validationEngine.validateExecutionOutputs(outputs, expectedOutputs);

            expect(result.valid).toBe(true);
            expect(result.errors).toHaveLength(0);
        });
    });

    describe("validateValue", () => {
        it("should validate string values", async () => {
            expect(await validationEngine.validateValue("hello", "string")).toBe(true);
            expect(await validationEngine.validateValue(123, "string")).toBe(false);
        });

        it("should validate number values", async () => {
            expect(await validationEngine.validateValue(42, "number")).toBe(true);
            expect(await validationEngine.validateValue("42", "number")).toBe(false);
        });

        it("should validate boolean values", async () => {
            expect(await validationEngine.validateValue(true, "boolean")).toBe(true);
            expect(await validationEngine.validateValue("true", "boolean")).toBe(false);
        });

        it("should validate array values", async () => {
            expect(await validationEngine.validateValue([1, 2, 3], "array")).toBe(true);
            expect(await validationEngine.validateValue("not-array", "array")).toBe(false);
        });

        it("should validate with constraints", async () => {
            const constraints = { minLength: 5, maxLength: 10 };
            expect(await validationEngine.validateValue("hello", "string", constraints)).toBe(true);
            expect(await validationEngine.validateValue("hi", "string", constraints)).toBe(false); // Too short
            expect(await validationEngine.validateValue("verylongstring", "string", constraints)).toBe(false); // Too long
        });

        it("should handle unknown types", async () => {
            expect(await validationEngine.validateValue("anything", "unknown")).toBe(true);
        });
    });

    describe("validateReasoning", () => {
        it("should validate correct reasoning structure", async () => {
            const reasoning = {
                conclusion: "The analysis is complete",
                reasoning: ["Step 1: Analyze data", "Step 2: Draw conclusions"],
                evidence: [
                    {
                        type: "fact",
                        content: "Data shows positive trend",
                        confidence: 0.9,
                    },
                ],
                confidence: 0.85,
                assumptions: ["Data is accurate", "Trend will continue"],
            };

            const result = await validationEngine.validateReasoning(reasoning);

            expect(result.passed).toBe(true);
            expect(result.type).toBe("structure");
            expect(result.message).toBe("Valid reasoning structure");
        });

        it("should fail validation for invalid reasoning structure", async () => {
            const reasoning = {
                conclusion: "Missing required fields",
                // Missing reasoning, evidence, confidence, assumptions
            };

            const result = await validationEngine.validateReasoning(reasoning);

            expect(result.passed).toBe(false);
            expect(result.type).toBe("structure");
            expect(result.message).toContain("required");
        });

        it("should validate evidence structure", async () => {
            const reasoning = {
                conclusion: "Test",
                reasoning: ["Step 1"],
                evidence: [
                    {
                        type: "invalid-type", // Should be 'fact', 'inference', or 'assumption'
                        content: "Test evidence",
                        confidence: 0.8,
                    },
                ],
                confidence: 0.7,
                assumptions: [],
            };

            const result = await validationEngine.validateReasoning(reasoning);

            expect(result.passed).toBe(false);
            expect(result.message).toContain("type");
        });
    });

    describe("event emission", () => {
        it("should emit security scan events", async () => {
            const outputs = { data: "test" };

            await validationEngine.validateOutputs(outputs);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/security_scan_requested",
                    data: expect.objectContaining({
                        data: outputs,
                    }),
                }),
            );
        });

        it("should emit quality check events", async () => {
            const outputs = { data: "test" };

            await validationEngine.validateOutputs(outputs);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/quality_check_requested",
                    data: expect.objectContaining({
                        data: outputs,
                    }),
                }),
            );
        });

        it("should handle event emission failures gracefully", async () => {
            vi.mocked(mockEventBus.publish).mockRejectedValueOnce(new Error("Event bus error"));

            const outputs = { data: "test" };
            const result = await validationEngine.validateOutputs(outputs);

            // Should still complete validation despite event error
            expect(result.valid).toBe(true);
        });
    });

    describe("schema conversion", () => {
        it("should handle plain object schemas", async () => {
            const outputs = { name: "test", age: 25 };
            const schema = {
                name: { type: "string", required: true },
                age: { type: "number", required: true },
            };

            const result = await validationEngine.validateOutputs(outputs, schema);

            expect(result.valid).toBe(true);
        });

        it("should handle complex schema constraints", async () => {
            const outputs = { 
                email: "test@example.com",
                password: "secretpass",
            };
            const schema = {
                email: { 
                    type: "string", 
                    required: true, 
                    pattern: "^[^@]+@[^@]+\\.[^@]+$",
                },
                password: { 
                    type: "string", 
                    required: true, 
                    minLength: 8,
                },
            };

            const result = await validationEngine.validateOutputs(outputs, schema);

            expect(result.valid).toBe(true);
        });
    });

    describe("error handling", () => {
        it("should handle validation errors gracefully", async () => {
            // Mock schema validation to throw
            const invalidSchema = {
                validate: vi.fn().mockRejectedValue(new Error("Schema error")),
                validateSync: vi.fn().mockRejectedValue(new Error("Schema error")),
            };

            const result = await validationEngine.validate({ test: "data" }, invalidSchema as unknown as yup.Schema);

            expect(result.valid).toBe(false);
            expect(result.errors[0]).toContain("Schema error");
        });

        it("should handle malformed outputs", async () => {
            const outputs = {
                "": "empty key", // Invalid key
                validKey: null, // Null value
            };

            const result = await validationEngine.validateOutputs(outputs);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Output contains empty key");
            expect(result.errors).toContain("Output field 'validKey' is null");
        });
    });

    describe("integration with emergent agents", () => {
        it("should provide rich event data for security agents", async () => {
            const outputs = { 
                userInput: "<script>alert('xss')</script>",
                sqlQuery: "SELECT * FROM users WHERE id = 1; DROP TABLE users;--",
            };

            await validationEngine.validateOutputs(outputs);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/security_scan_requested",
                    data: expect.objectContaining({
                        data: outputs,
                        timestamp: expect.any(Date),
                    }),
                }),
            );
        });

        it("should provide rich event data for quality agents", async () => {
            const outputs = { 
                analysis: "Based on the data, I conclude that...",
                confidence: 0.75,
            };

            await validationEngine.validateOutputs(outputs);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/quality_check_requested",
                    data: expect.objectContaining({
                        data: outputs,
                        timestamp: expect.any(Date),
                    }),
                }),
            );
        });

        it("should emit comprehensive validation context", async () => {
            const outputs = { result: "test" };
            const context = {
                executionId: "exec-123",
                stepId: "step-456", 
                routineId: "routine-789",
                tier: 3 as const,
                strategy: "deterministic",
            };

            await validationEngine.validateOutputs(outputs, undefined, context);

            expect(mockEventBus.publish).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "execution/output/validation_requested",
                    data: expect.objectContaining({
                        executionId: "exec-123",
                        stepId: "step-456",
                        routineId: "routine-789",
                        tier: 3,
                        strategy: "deterministic",
                        outputs,
                        schema: "none",
                        timestamp: expect.any(Date),
                    }),
                }),
            );
        });
    });
});
