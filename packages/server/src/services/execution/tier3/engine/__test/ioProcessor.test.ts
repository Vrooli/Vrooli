import { describe, it, expect, beforeEach } from "vitest";
import { IOProcessor } from "../ioProcessor.js";
import { type RoutineContextImpl } from "@vrooli/shared";
import { createMockLogger } from "../../../__test/mocks/logger.js";

describe("IOProcessor", () => {
    let ioProcessor: IOProcessor;
    let mockLogger: any;
    let mockContext: Partial<RoutineContextImpl>;

    beforeEach(() => {
        mockLogger = createMockLogger();
        ioProcessor = new IOProcessor(mockLogger);
        
        // Create a mutable context object for testing
        const mockSubroutineContext = {
            allOutputsMap: {
                "step-1": {
                    result: "Hello World",
                    count: 42,
                    data: {
                        nested: "value",
                    },
                },
                "step-2": {
                    processed: true,
                    items: ["a", "b", "c"],
                },
            },
            allOutputsList: [
                { key: "step-1.result", value: "Hello World" },
                { key: "step-1.count", value: 42 },
                { key: "step-2.processed", value: true },
            ],
            allInputsMap: {},
            allInputsList: [],
        };
        
        // Create a mock context with the required getSubroutineContext method
        mockContext = {
            runId: "test-run-123",
            currentLocation: {
                nodeId: "step-123",
                routineId: "routine-456",
            },
            userData: {
                id: "user-789",
            },
            getSubroutineContext: () => mockSubroutineContext,
        } as RoutineContextImpl;
    });

    describe("buildInputPayload", () => {
        it("should resolve step references correctly", async () => {
            const inputs = {
                message: "$ref:step-1.result",
                counter: "$ref:step-1.count",
                flag: "$ref:step-2.processed",
                nested: "$ref:step-1.data.nested",
            };

            const result = await ioProcessor.buildInputPayload(inputs, mockContext as RoutineContextImpl);

            expect(result.message).toBe("Hello World");
            expect(result.counter).toBe(42);
            expect(result.flag).toBe(true);
            expect(result.nested).toBe("value");
        });

        it("should handle missing step references gracefully", async () => {
            const inputs = {
                validRef: "$ref:step-1.result",
                invalidStep: "$ref:nonexistent-step.output",
                invalidKey: "$ref:step-1.nonexistent",
            };

            // Valid reference should work
            const result = await ioProcessor.buildInputPayload(
                { validRef: inputs.validRef }, 
                mockContext as RoutineContextImpl,
            );
            expect(result.validRef).toBe("Hello World");

            // Invalid step should throw
            await expect(
                ioProcessor.buildInputPayload(
                    { invalidStep: inputs.invalidStep }, 
                    mockContext as RoutineContextImpl,
                ),
            ).rejects.toThrow("Referenced step 'nonexistent-step' not found");

            // Invalid key should throw
            await expect(
                ioProcessor.buildInputPayload(
                    { invalidKey: inputs.invalidKey }, 
                    mockContext as RoutineContextImpl,
                ),
            ).rejects.toThrow("Invalid step reference: Output key 'nonexistent' not found in step 'step-1'");
        });

        it("should handle template variables", async () => {
            const inputs = {
                template: "User {{user.id}} in run {{run.id}}",
                simpleUserTemplate: "Hello {{user.id}}",
            };

            const result = await ioProcessor.buildInputPayload(inputs, mockContext as RoutineContextImpl);

            expect(result.template).toBe("User user-789 in run test-run-123");
            expect(result.simpleUserTemplate).toBe("Hello user-789");
        });

        it("should inject context data", async () => {
            const inputs = {
                simpleValue: "test",
            };

            const result = await ioProcessor.buildInputPayload(inputs, mockContext as RoutineContextImpl);

            expect(result._context).toEqual({
                runId: "test-run-123",
                routineId: "routine-456",
                userId: "user-789",
                timestamp: expect.any(String),
            });
        });
    });

    describe("processOutputs", () => {
        it("should store outputs for future reference", async () => {
            const outputs = {
                result: "Processing complete",
                count: 100,
                data: { status: "success" },
            };

            const result = await ioProcessor.processOutputs(
                outputs, 
                undefined, 
                mockContext as RoutineContextImpl,
            );

            expect(result).toEqual(outputs);
            
            // Verify outputs were stored (check via the mock context)
            const context = mockContext.getSubroutineContext!();
            expect(context.allOutputsMap["step-123"]).toEqual(outputs);
        });

        it("should normalize non-object outputs", async () => {
            const rawOutput = "simple string result";

            const result = await ioProcessor.processOutputs(
                rawOutput, 
                undefined, 
                mockContext as RoutineContextImpl,
            );

            expect(result).toEqual({
                result: "simple string result",
            });
        });
    });

    describe("step reference validation", () => {
        it("should validate step references", async () => {
            // Test private method through public interface
            const validInputs = { test: "$ref:step-1.result" };
            const invalidInputs = { test: "$ref:invalid-step.output" };

            // Valid reference should work
            await expect(
                ioProcessor.buildInputPayload(validInputs, mockContext as RoutineContextImpl),
            ).resolves.toBeDefined();

            // Invalid reference should throw
            await expect(
                ioProcessor.buildInputPayload(invalidInputs, mockContext as RoutineContextImpl),
            ).rejects.toThrow("Referenced step 'invalid-step' not found");
        });
    });
});
