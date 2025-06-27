/**
 * RunIO API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for run input/output endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    RunRoutineInput,
    RunRoutineOutput,
    RunRoutineInputCreateInput,
    RunRoutineOutputCreateInput,
    RunRoutineInputUpdateInput,
    RunRoutineOutputUpdateInput,
} from "@vrooli/shared";
import { 
    runIOValidation, 
} from "@vrooli/shared";

/**
 * Standard API response wrapper
 */
export interface APIResponse<T> {
    data: T;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
        links?: {
            self?: string;
            related?: Record<string, string>;
        };
    };
}

/**
 * API error response structure
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, any>;
        timestamp: string;
        requestId: string;
        path: string;
    };
}

/**
 * Paginated response structure
 */
export interface PaginatedAPIResponse<T> extends APIResponse<T[]> {
    pagination: {
        page: number;
        pageSize: number;
        totalCount: number;
        totalPages: number;
        hasNextPage: boolean;
        hasPreviousPage: boolean;
    };
}

/**
 * RunIO API response factory
 */
export class RunIOResponseFactory {
    private readonly baseUrl: string;
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || "http://localhost:5329") {
        this.baseUrl = baseUrl;
    }
    
    /**
     * Generate unique request ID
     */
    private generateRequestId(): string {
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique run IO ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful run input response
     */
    createInputSuccessResponse(input: RunRoutineInput): APIResponse<RunRoutineInput> {
        return {
            data: input,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-input/${input.id}`,
                    related: {
                        run: `${this.baseUrl}/api/run/${input.runRoutine?.id}`,
                        input: input.input?.id ? `${this.baseUrl}/api/routine-input/${input.input.id}` : undefined,
                    },
                },
            },
        };
    }
    
    /**
     * Create successful run output response
     */
    createOutputSuccessResponse(output: RunRoutineOutput): APIResponse<RunRoutineOutput> {
        return {
            data: output,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-output/${output.id}`,
                    related: {
                        run: `${this.baseUrl}/api/run/${output.runRoutine?.id}`,
                        output: output.output?.id ? `${this.baseUrl}/api/routine-output/${output.output.id}` : undefined,
                    },
                },
            },
        };
    }
    
    /**
     * Create run input list response
     */
    createInputListResponse(inputs: RunRoutineInput[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<RunRoutineInput> {
        const paginationData = pagination || {
            page: 1,
            pageSize: inputs.length,
            totalCount: inputs.length,
        };
        
        return {
            data: inputs,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-input?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create run output list response
     */
    createOutputListResponse(outputs: RunRoutineOutput[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<RunRoutineOutput> {
        const paginationData = pagination || {
            page: 1,
            pageSize: outputs.length,
            totalCount: outputs.length,
        };
        
        return {
            data: outputs,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-output?page=${paginationData.page}&limit=${paginationData.pageSize}`,
                },
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1,
            },
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: "VALIDATION_ERROR",
                message: "The request contains invalid data",
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/run-io",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(type: "input" | "output", id: string): APIErrorResponse {
        return {
            error: {
                code: `RUN_${type.toUpperCase()}_NOT_FOUND`,
                message: `Run ${type} with ID '${id}' was not found`,
                details: {
                    [`${type}Id`]: id,
                    searchCriteria: { id },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/run-${type}/${id}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} this run data`,
                details: {
                    operation,
                    requiredPermissions: ["run:write"],
                    userPermissions: ["run:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/run-io",
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "NETWORK_ERROR",
                message: "Network request failed",
                details: {
                    reason: "Connection timeout",
                    retryable: true,
                    retryAfter: 5000,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/run-io",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: "INTERNAL_SERVER_ERROR",
                message: "An unexpected server error occurred",
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/run-io",
            },
        };
    }
    
    /**
     * Create mock run input data
     */
    createMockRunInput(overrides?: Partial<RunRoutineInput>): RunRoutineInput {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultInput: RunRoutineInput = {
            __typename: "RunRoutineInput",
            id,
            data: JSON.stringify({
                value: "test input value",
                type: "string",
            }),
            input: {
                __typename: "RoutineVersionInput",
                id: `input_${id}`,
                index: 0,
                isRequired: true,
                name: "testInput",
                translations: [{
                    __typename: "RoutineVersionInputTranslation",
                    id: `trans_${id}`,
                    language: "en",
                    description: "Test input for the routine",
                    helpText: "Enter a test value",
                }],
            },
            runRoutine: {
                __typename: "RunRoutine",
                id: `run_${id}`,
                completedAt: null,
                startedAt: now,
                isPrivate: false,
                timeElapsed: 0,
                title: "Test Run",
                runProject: null,
                steps: [],
                inputs: [],
                outputs: [],
            },
        };
        
        return {
            ...defaultInput,
            ...overrides,
        };
    }
    
    /**
     * Create mock run output data
     */
    createMockRunOutput(overrides?: Partial<RunRoutineOutput>): RunRoutineOutput {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultOutput: RunRoutineOutput = {
            __typename: "RunRoutineOutput",
            id,
            data: JSON.stringify({
                result: "test output value",
                type: "string",
            }),
            output: {
                __typename: "RoutineVersionOutput",
                id: `output_${id}`,
                index: 0,
                name: "testOutput",
                translations: [{
                    __typename: "RoutineVersionOutputTranslation",
                    id: `trans_${id}`,
                    language: "en",
                    description: "Test output from the routine",
                    helpText: "This is the result",
                }],
            },
            runRoutine: {
                __typename: "RunRoutine",
                id: `run_${id}`,
                completedAt: now,
                startedAt: new Date(Date.now() - 3600000).toISOString(),
                isPrivate: false,
                timeElapsed: 3600,
                title: "Test Run",
                runProject: null,
                steps: [],
                inputs: [],
                outputs: [],
            },
        };
        
        return {
            ...defaultOutput,
            ...overrides,
        };
    }
    
    /**
     * Create run input from API input
     */
    createRunInputFromInput(input: RunRoutineInputCreateInput): RunRoutineInput {
        const runInput = this.createMockRunInput();
        
        // Update run input based on input
        if (input.data) {
            runInput.data = input.data;
        }
        
        if (input.inputConnect) {
            runInput.input.id = input.inputConnect;
        }
        
        if (input.runRoutineConnect) {
            runInput.runRoutine.id = input.runRoutineConnect;
        }
        
        return runInput;
    }
    
    /**
     * Create run output from API input
     */
    createRunOutputFromInput(input: RunRoutineOutputCreateInput): RunRoutineOutput {
        const runOutput = this.createMockRunOutput();
        
        // Update run output based on input
        if (input.data) {
            runOutput.data = input.data;
        }
        
        if (input.outputConnect) {
            runOutput.output.id = input.outputConnect;
        }
        
        if (input.runRoutineConnect) {
            runOutput.runRoutine.id = input.runRoutineConnect;
        }
        
        return runOutput;
    }
    
    /**
     * Create multiple run inputs
     */
    createMultipleRunInputs(count = 5): RunRoutineInput[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockRunInput({
                id: `input_${index}_${this.generateId()}`,
                input: {
                    ...this.createMockRunInput().input,
                    index,
                    name: `input${index + 1}`,
                },
            }),
        );
    }
    
    /**
     * Create multiple run outputs
     */
    createMultipleRunOutputs(count = 5): RunRoutineOutput[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockRunOutput({
                id: `output_${index}_${this.generateId()}`,
                output: {
                    ...this.createMockRunOutput().output,
                    index,
                    name: `output${index + 1}`,
                },
            }),
        );
    }
    
    /**
     * Validate run input create input
     */
    async validateInputCreateInput(input: RunRoutineInputCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await runIOValidation.runRoutineInputCreate.validate(input);
            return { valid: true };
        } catch (error: any) {
            const fieldErrors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        fieldErrors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                fieldErrors.general = error.message;
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }
    
    /**
     * Validate run output create input
     */
    async validateOutputCreateInput(input: RunRoutineOutputCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await runIOValidation.runRoutineOutputCreate.validate(input);
            return { valid: true };
        } catch (error: any) {
            const fieldErrors: Record<string, string> = {};
            
            if (error.inner) {
                error.inner.forEach((err: any) => {
                    if (err.path) {
                        fieldErrors[err.path] = err.message;
                    }
                });
            } else if (error.message) {
                fieldErrors.general = error.message;
            }
            
            return {
                valid: false,
                errors: fieldErrors,
            };
        }
    }
}

/**
 * MSW handlers factory for run IO endpoints
 */
export class RunIOMSWHandlers {
    private responseFactory: RunIOResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new RunIOResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all run IO endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create run input
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, async (req, res, ctx) => {
                const body = await req.json() as RunRoutineInputCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateInputCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create run input
                const runInput = this.responseFactory.createRunInputFromInput(body);
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Create run output
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, async (req, res, ctx) => {
                const body = await req.json() as RunRoutineOutputCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateOutputCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create run output
                const runOutput = this.responseFactory.createRunOutputFromInput(body);
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get run input by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const runInput = this.responseFactory.createMockRunInput({ id: id as string });
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get run output by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const runOutput = this.responseFactory.createMockRunOutput({ id: id as string });
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update run input
            http.put(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as RunRoutineInputUpdateInput;
                
                const runInput = this.responseFactory.createMockRunInput({ 
                    id: id as string,
                    data: body.data || JSON.stringify({ updated: true }),
                });
                
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update run output
            http.put(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as RunRoutineOutputUpdateInput;
                
                const runOutput = this.responseFactory.createMockRunOutput({ 
                    id: id as string,
                    data: body.data || JSON.stringify({ updated: true }),
                });
                
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete run input
            http.delete(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // Delete run output
            http.delete(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List run inputs
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const runId = url.searchParams.get("runId");
                
                let inputs = this.responseFactory.createMultipleRunInputs(20);
                
                // Filter by run ID if specified
                if (runId) {
                    inputs = inputs.filter(i => i.runRoutine.id === runId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedInputs = inputs.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createInputListResponse(
                    paginatedInputs,
                    {
                        page,
                        pageSize: limit,
                        totalCount: inputs.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // List run outputs
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const runId = url.searchParams.get("runId");
                
                let outputs = this.responseFactory.createMultipleRunOutputs(20);
                
                // Filter by run ID if specified
                if (runId) {
                    outputs = outputs.filter(o => o.runRoutine.id === runId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedOutputs = outputs.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createOutputListResponse(
                    paginatedOutputs,
                    {
                        page,
                        pageSize: limit,
                        totalCount: outputs.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Validation error for input
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        data: "Input data is required",
                        inputConnect: "Input reference is required",
                    })),
                );
            }),
            
            // Validation error for output
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        data: "Output data is required",
                        outputConnect: "Output reference is required",
                    })),
                );
            }),
            
            // Not found error for input
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse("input", id as string)),
                );
            }),
            
            // Not found error for output
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse("output", id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse()),
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, async (req, res, ctx) => {
                const body = await req.json() as RunRoutineInputCreateInput;
                const runInput = this.responseFactory.createRunInputFromInput(body);
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, async (req, res, ctx) => {
                const body = await req.json() as RunRoutineOutputCreateInput;
                const runOutput = this.responseFactory.createRunOutputFromInput(body);
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: "GET" | "POST" | "PUT" | "DELETE";
        status: number;
        response: any;
        delay?: number;
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        return rest[method.toLowerCase() as keyof typeof rest](fullEndpoint, (req, res, ctx) => {
            const responseCtx = [ctx.status(status), ctx.json(response)];
            
            if (delay) {
                responseCtx.unshift(ctx.delay(delay));
            }
            
            return res(...responseCtx);
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const runIOResponseScenarios = {
    // Success scenarios
    createInputSuccess: (input?: RunRoutineInput) => {
        const factory = new RunIOResponseFactory();
        return factory.createInputSuccessResponse(
            input || factory.createMockRunInput(),
        );
    },
    
    createOutputSuccess: (output?: RunRoutineOutput) => {
        const factory = new RunIOResponseFactory();
        return factory.createOutputSuccessResponse(
            output || factory.createMockRunOutput(),
        );
    },
    
    listInputsSuccess: (inputs?: RunRoutineInput[]) => {
        const factory = new RunIOResponseFactory();
        return factory.createInputListResponse(
            inputs || factory.createMultipleRunInputs(),
        );
    },
    
    listOutputsSuccess: (outputs?: RunRoutineOutput[]) => {
        const factory = new RunIOResponseFactory();
        return factory.createOutputListResponse(
            outputs || factory.createMultipleRunOutputs(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new RunIOResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                data: "Data is required",
                connect: "Reference is required",
            },
        );
    },
    
    notFoundError: (type: "input" | "output", id?: string) => {
        const factory = new RunIOResponseFactory();
        return factory.createNotFoundErrorResponse(
            type,
            id || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new RunIOResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new RunIOResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new RunIOMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new RunIOMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new RunIOMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new RunIOMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const runIOResponseFactory = new RunIOResponseFactory();
export const runIOMSWHandlers = new RunIOMSWHandlers();
