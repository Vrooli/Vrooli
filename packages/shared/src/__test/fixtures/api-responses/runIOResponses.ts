/* c8 ignore start */
/**
 * Run Input/Output API Response Fixtures
 * 
 * Comprehensive API response fixtures for run input/output endpoints, covering
 * execution workflows, data flow validation, and runtime scenarios.
 */

import type {
    RoutineVersionInput,
    RoutineVersionOutput,
    RunRoutine,
    RunRoutineInput,
    RunRoutineInputCreateInput,
    RunRoutineInputUpdateInput,
    RunRoutineOutput,
    RunRoutineOutputCreateInput,
    RunRoutineOutputUpdateInput,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

// Constants for realistic data generation
const INPUT_TYPES = ["string", "number", "boolean", "object", "array"] as const;
const OUTPUT_TYPES = ["string", "number", "boolean", "object", "array", "file"] as const;
const MIN_PROCESSING_TIME_MS = 100;
const MAX_PROCESSING_TIME_RANGE_MS = 1000;
const HOUR_IN_MS = 3600000;
const HALF_HOUR_IN_MS = 1800;
const DEFAULT_LIST_SIZE = 5;
const DATA_SAMPLES = {
    string: "sample text value",
    number: 42,
    boolean: true,
    object: { key: "value", nested: { data: "example" } },
    array: ["item1", "item2", "item3"],
    file: { name: "output.txt", content: "Generated file content", mimeType: "text/plain" },
} as const;

/**
 * Factory for generating Run I/O API responses
 */
export class RunIOAPIResponseFactory extends BaseAPIResponseFactory<
    RunRoutineInput | RunRoutineOutput,
    RunRoutineInputCreateInput | RunRoutineOutputCreateInput,
    RunRoutineInputUpdateInput | RunRoutineOutputUpdateInput
> {
    protected readonly entityName = "run-io";

    /**
     * Generate realistic input data based on type
     */
    private generateInputData(type = "string", isRequired = true): string {
        const dataType = INPUT_TYPES.includes(type as any) ? type as keyof typeof DATA_SAMPLES : "string";
        const value = isRequired ? DATA_SAMPLES[dataType] : null;

        return JSON.stringify({
            value,
            type: dataType,
            metadata: {
                provided: isRequired,
                timestamp: new Date().toISOString(),
            },
        });
    }

    /**
     * Generate realistic output data based on type
     */
    private generateOutputData(type = "string", hasResult = true): string {
        const dataType = OUTPUT_TYPES.includes(type as any) ? type as keyof typeof DATA_SAMPLES : "string";
        const result = hasResult ? DATA_SAMPLES[dataType] : null;

        return JSON.stringify({
            result,
            type: dataType,
            metadata: {
                generated: hasResult,
                timestamp: new Date().toISOString(),
                processingTime: Math.floor(Math.random() * MAX_PROCESSING_TIME_RANGE_MS) + MIN_PROCESSING_TIME_MS,
            },
        });
    }

    /**
     * Create mock routine version input
     */
    private createMockRoutineVersionInput(index = 0, name?: string, isRequired = true): RoutineVersionInput {
        const id = this.generateId();
        const inputName = name || `input${index + 1}`;

        return {
            __typename: "RoutineVersionInput",
            id: `input_def_${id}`,
            index,
            isRequired,
            name: inputName,
            translations: [{
                __typename: "RoutineVersionInputTranslation",
                id: `trans_${id}`,
                language: "en",
                description: `Input parameter ${inputName} for the routine`,
                helpText: `Provide a value for ${inputName}`,
            }],
        };
    }

    /**
     * Create mock routine version output
     */
    private createMockRoutineVersionOutput(index = 0, name?: string): RoutineVersionOutput {
        const id = this.generateId();
        const outputName = name || `output${index + 1}`;

        return {
            __typename: "RoutineVersionOutput",
            id: `output_def_${id}`,
            index,
            name: outputName,
            translations: [{
                __typename: "RoutineVersionOutputTranslation",
                id: `trans_${id}`,
                language: "en",
                description: `Output result ${outputName} from the routine`,
                helpText: `This contains the ${outputName} result`,
            }],
        };
    }

    /**
     * Create mock run routine
     */
    private createMockRunRoutine(isCompleted = false): RunRoutine {
        const id = this.generateId();
        const now = new Date().toISOString();
        const startTime = new Date(Date.now().toISOString() - HOUR_IN_MS).toISOString(); // 1 hour ago
        const timeElapsed = isCompleted ? HOUR_IN_MS / 1000 : Math.floor(Math.random() * HALF_HOUR_IN_MS); // Random up to 30 min or 1 hour if completed

        return {
            __typename: "RunRoutine",
            id: `run_${id}`,
            completedAt: isCompleted ? now : null,
            startedAt: startTime,
            isPrivate: false,
            timeElapsed,
            title: "Test Routine Execution",
            runProject: null,
            steps: [],
            inputs: [],
            outputs: [],
        };
    }

    /**
     * Create mock run input data
     */
    createMockRunInput(options?: MockDataOptions): RunRoutineInput {
        const id = this.generateId();
        const index = options?.overrides?.index as number || 0;
        const inputName = options?.overrides?.name as string || `input${index + 1}`;
        const isRequired = options?.overrides?.isRequired as boolean ?? true;
        const dataType = options?.overrides?.type as string || "string";

        const routineInput = this.createMockRoutineVersionInput(index, inputName, isRequired);
        const runRoutine = options?.withRelations !== false ? this.createMockRunRoutine() : null;

        const baseInput: RunRoutineInput = {
            __typename: "RunRoutineInput",
            id,
            data: this.generateInputData(dataType, isRequired),
            input: routineInput,
            runRoutine,
        };

        // Apply scenario-specific overrides
        if (options?.scenario) {
            switch (options.scenario) {
                case "minimal":
                    baseInput.runRoutine = null;
                    baseInput.data = JSON.stringify({ value: null, type: "string" });
                    break;
                case "complete":
                    baseInput.data = this.generateInputData("object", true);
                    break;
                case "edge-case":
                    baseInput.data = this.generateInputData("array", false);
                    baseInput.input.isRequired = false;
                    break;
            }
        }

        // Apply explicit overrides
        if (options?.overrides) {
            Object.assign(baseInput, options.overrides);
        }

        return baseInput;
    }

    /**
     * Create mock run output data
     */
    createMockRunOutput(options?: MockDataOptions): RunRoutineOutput {
        const id = this.generateId();
        const index = options?.overrides?.index as number || 0;
        const outputName = options?.overrides?.name as string || `output${index + 1}`;
        const dataType = options?.overrides?.type as string || "string";
        const hasResult = options?.overrides?.hasResult as boolean ?? true;

        const routineOutput = this.createMockRoutineVersionOutput(index, outputName);
        const runRoutine = options?.withRelations !== false ? this.createMockRunRoutine(true) : null;

        const baseOutput: RunRoutineOutput = {
            __typename: "RunRoutineOutput",
            id,
            data: this.generateOutputData(dataType, hasResult),
            output: routineOutput,
            runRoutine,
        };

        // Apply scenario-specific overrides
        if (options?.scenario) {
            switch (options.scenario) {
                case "minimal":
                    baseOutput.runRoutine = null;
                    baseOutput.data = JSON.stringify({ result: null, type: "string" });
                    break;
                case "complete":
                    baseOutput.data = this.generateOutputData("file", true);
                    break;
                case "edge-case":
                    baseOutput.data = this.generateOutputData("object", false);
                    break;
            }
        }

        // Apply explicit overrides
        if (options?.overrides) {
            Object.assign(baseOutput, options.overrides);
        }

        return baseOutput;
    }

    /**
     * Create mock data - supports both input and output
     */
    createMockData(options?: MockDataOptions & { type?: "input" | "output" }): RunRoutineInput | RunRoutineOutput {
        const type = options?.type || "input";
        return type === "input" ? this.createMockRunInput(options) : this.createMockRunOutput(options);
    }

    /**
     * Create entity from create input
     */
    createFromInput(input: RunRoutineInputCreateInput | RunRoutineOutputCreateInput): RunRoutineInput | RunRoutineOutput {
        if ("inputConnect" in input) {
            // This is a RunRoutineInputCreateInput
            const runInput = this.createMockRunInput() as RunRoutineInput;

            if (input.id) runInput.id = input.id;
            if (input.data) runInput.data = input.data;
            if (input.inputConnect) runInput.input.id = input.inputConnect;
            if (input.runRoutineConnect) runInput.runRoutine!.id = input.runRoutineConnect;

            return runInput;
        } else {
            // This is a RunRoutineOutputCreateInput
            const runOutput = this.createMockRunOutput() as RunRoutineOutput;

            if (input.id) runOutput.id = input.id;
            if (input.data) runOutput.data = input.data;
            if (input.outputConnect) runOutput.output.id = input.outputConnect;
            if (input.runRoutineConnect) runOutput.runRoutine!.id = input.runRoutineConnect;

            return runOutput;
        }
    }

    /**
     * Update entity from update input
     */
    updateFromInput(
        existing: RunRoutineInput | RunRoutineOutput,
        input: RunRoutineInputUpdateInput | RunRoutineOutputUpdateInput,
    ): RunRoutineInput | RunRoutineOutput {
        const updated = { ...existing };

        if (input.data !== undefined) {
            updated.data = input.data;
        }

        return updated;
    }

    /**
     * Validate input create input
     */
    private async validateInputCreate(input: RunRoutineInputCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.data?.trim()) {
            errors.data = "Input data is required";
        } else {
            try {
                JSON.parse(input.data);
            } catch {
                errors.data = "Input data must be valid JSON";
            }
        }

        if (!input.inputConnect) {
            errors.inputConnect = "Input definition reference is required";
        }

        if (!input.runRoutineConnect) {
            errors.runRoutineConnect = "Run routine reference is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate output create input
     */
    private async validateOutputCreate(input: RunRoutineOutputCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.data?.trim()) {
            errors.data = "Output data is required";
        } else {
            try {
                JSON.parse(input.data);
            } catch {
                errors.data = "Output data must be valid JSON";
            }
        }

        if (!input.outputConnect) {
            errors.outputConnect = "Output definition reference is required";
        }

        if (!input.runRoutineConnect) {
            errors.runRoutineConnect = "Run routine reference is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: RunRoutineInputCreateInput | RunRoutineOutputCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        if ("inputConnect" in input) {
            return this.validateInputCreate(input);
        } else {
            return this.validateOutputCreate(input);
        }
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: RunRoutineInputUpdateInput | RunRoutineOutputUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.id) {
            errors.id = "ID is required for updates";
        }

        if (input.data) {
            try {
                JSON.parse(input.data);
            } catch {
                errors.data = "Data must be valid JSON";
            }
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create multiple run inputs
     */
    createMultipleRunInputs(count = DEFAULT_LIST_SIZE, runId?: string): RunRoutineInput[] {
        return Array.from({ length: count }, (_, index) =>
            this.createMockRunInput({
                overrides: {
                    index,
                    name: `input${index + 1}`,
                    runRoutine: runId ? { id: runId } : undefined,
                },
                withRelations: true,
            }) as RunRoutineInput,
        );
    }

    /**
     * Create multiple run outputs
     */
    createMultipleRunOutputs(count = DEFAULT_LIST_SIZE, runId?: string): RunRoutineOutput[] {
        return Array.from({ length: count }, (_, index) =>
            this.createMockRunOutput({
                overrides: {
                    index,
                    name: `output${index + 1}`,
                    runRoutine: runId ? { id: runId } : undefined,
                },
                withRelations: true,
            }) as RunRoutineOutput,
        );
    }

    /**
     * Create business error for invalid data type
     */
    createDataTypeValidationError(expectedType: string, actualType: string): any {
        return this.createBusinessErrorResponse("constraint", {
            reason: "Data type mismatch",
            expectedType,
            actualType,
            message: `Expected ${expectedType} but received ${actualType}`,
        });
    }

    /**
     * Create business error for missing required input
     */
    createMissingRequiredInputError(inputName: string): any {
        return this.createBusinessErrorResponse("constraint", {
            reason: "Required input missing",
            inputName,
            message: `Required input '${inputName}' was not provided`,
        });
    }

    /**
     * Create business error for execution timeout
     */
    createExecutionTimeoutError(timeoutSeconds: number): any {
        return this.createBusinessErrorResponse("limit", {
            reason: "Execution timeout",
            timeoutSeconds,
            message: `Execution exceeded timeout limit of ${timeoutSeconds} seconds`,
        });
    }
}

/**
 * Pre-configured response scenarios for common use cases
 */
export const runIOResponseScenarios = {
    // Success scenarios - Inputs
    createInputSuccess: (input?: RunRoutineInput) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createSuccessResponse(
            input || factory.createMockRunInput(),
        );
    },

    listInputsSuccess: (inputs?: RunRoutineInput[], pagination?: { page: number; pageSize: number; totalCount: number }) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createPaginatedResponse(
            inputs || factory.createMultipleRunInputs(),
            pagination || { page: 1, pageSize: 20, totalCount: 50 },
        );
    },

    requiredInputScenario: () => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockRunInput({
                overrides: { isRequired: true, type: "string" },
                scenario: "complete",
            }),
        );
    },

    optionalInputScenario: () => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockRunInput({
                overrides: { isRequired: false, type: "object" },
                scenario: "edge-case",
            }),
        );
    },

    // Success scenarios - Outputs
    createOutputSuccess: (output?: RunRoutineOutput) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createSuccessResponse(
            output || factory.createMockRunOutput(),
        );
    },

    listOutputsSuccess: (outputs?: RunRoutineOutput[], pagination?: { page: number; pageSize: number; totalCount: number }) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createPaginatedResponse(
            outputs || factory.createMultipleRunOutputs(),
            pagination || { page: 1, pageSize: 20, totalCount: 50 },
        );
    },

    fileOutputScenario: () => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockRunOutput({
                overrides: { type: "file", hasResult: true },
                scenario: "complete",
            }),
        );
    },

    complexDataScenario: () => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockRunOutput({
                overrides: { type: "object", hasResult: true },
                scenario: "complete",
            }),
        );
    },

    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                data: "Data must be valid JSON",
                inputConnect: "Input definition reference is required",
                runRoutineConnect: "Run routine reference is required",
            },
        );
    },

    notFoundError: (type: "input" | "output", id?: string) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createNotFoundErrorResponse(
            id || "non-existent-io-id",
            `run ${type}`,
        );
    },

    permissionError: (operation?: string) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["run:write"],
        );
    },

    dataTypeValidationError: (expectedType?: string, actualType?: string) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createDataTypeValidationError(
            expectedType || "string",
            actualType || "number",
        );
    },

    missingRequiredInputError: (inputName?: string) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createMissingRequiredInputError(inputName || "requiredParam");
    },

    executionTimeoutError: (timeoutSeconds?: number) => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createExecutionTimeoutError(timeoutSeconds || 300);
    },

    serverError: () => {
        const factory = new RunIOAPIResponseFactory();
        return factory.createServerErrorResponse("run-io-service", "process");
    },

    rateLimitError: () => {
        const factory = new RunIOAPIResponseFactory();
        const resetTime = new Date(Date.now().toISOString() + MINUTES_IN_MS); // 1 minute from now
        return factory.createRateLimitErrorResponse(1000, 0, resetTime);
    },
};

// Export factory instance for direct use
export const runIOAPIResponseFactory = new RunIOAPIResponseFactory();

