/**
 * RunIO API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for run input/output endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import { 
    type RunIO,
    type RunIOCreateInput,
    type RunIOUpdateInput,
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
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Generate unique run IO ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful run input response
     */
    createInputSuccessResponse(input: RunIO): APIResponse<RunIO> {
        return {
            data: input,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-input/${input.id}`,
                    related: {
                        run: `${this.baseUrl}/api/run/${input.run.id}`,
                        node: `${this.baseUrl}/api/node/${input.nodeName}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create successful run output response
     */
    createOutputSuccessResponse(output: RunIO): APIResponse<RunIO> {
        return {
            data: output,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-output/${output.id}`,
                    related: {
                        run: `${this.baseUrl}/api/run/${output.run.id}`,
                        node: `${this.baseUrl}/api/node/${output.nodeName}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create run input list response
     */
    createInputListResponse(inputs: RunIO[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<RunIO> {
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
    createOutputListResponse(outputs: RunIO[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<RunIO> {
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
    createMockRunInput(overrides?: Partial<RunIO>): RunIO {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultInput: RunIO = {
            __typename: "RunIO",
            id,
            data: JSON.stringify({
                value: "test input value",
                type: "string",
            }),
            nodeInputName: "testInput",
            nodeName: "TestNode",
            run: {
                __typename: "Run",
                id: `run_${id}`,
                createdAt: now,
                updatedAt: now,
                completedAt: null,
                startedAt: now,
                status: "InProgress",
                timeElapsed: 0,
                name: "Test Run",
                contextSwitches: 0,
                steps: [],
                inputs: [],
                lastStep: null,
                runProject: null,
                runRoutine: null,
                user: null,
                you: {
                    __typename: "RunYou",
                    canDelete: true,
                    canUpdate: true,
                    canRead: true,
                },
            } as any,
        };
        
        return {
            ...defaultInput,
            ...overrides,
        };
    }
    
    /**
     * Create mock run output data
     */
    createMockRunOutput(overrides?: Partial<RunIO>): RunIO {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultOutput: RunIO = {
            __typename: "RunIO",
            id,
            data: JSON.stringify({
                result: "test output value",
                type: "string",
            }),
            nodeInputName: "testOutput",
            nodeName: "TestNode",
            run: {
                __typename: "Run",
                id: `run_${id}`,
                createdAt: new Date(Date.now() - 3600000).toISOString(),
                updatedAt: now,
                completedAt: now,
                startedAt: new Date(Date.now() - 3600000).toISOString(),
                status: "Completed",
                timeElapsed: 3600,
                name: "Test Run",
                contextSwitches: 2,
                steps: [],
                inputs: [],
                lastStep: null,
                runProject: null,
                runRoutine: null,
                user: null,
                you: {
                    __typename: "RunYou",
                    canDelete: true,
                    canUpdate: false,
                    canRead: true,
                },
            } as any,
        };
        
        return {
            ...defaultOutput,
            ...overrides,
        };
    }
    
    /**
     * Create run input from API input
     */
    createRunInputFromInput(input: RunIOCreateInput): RunIO {
        const runInput = this.createMockRunInput();
        
        // Update run input based on input
        if (input.data) {
            runInput.data = input.data;
        }
        
        if (input.nodeInputName) {
            runInput.nodeInputName = input.nodeInputName;
        }
        
        if (input.runConnect) {
            runInput.run.id = input.runConnect;
        }
        
        return runInput;
    }
    
    /**
     * Create run output from API input
     */
    createRunOutputFromInput(input: RunIOCreateInput): RunIO {
        const runOutput = this.createMockRunOutput();
        
        // Update run output based on input
        if (input.data) {
            runOutput.data = input.data;
        }
        
        if (input.nodeInputName) {
            runOutput.nodeInputName = input.nodeInputName;
        }
        
        if (input.runConnect) {
            runOutput.run.id = input.runConnect;
        }
        
        return runOutput;
    }
    
    /**
     * Create multiple run inputs
     */
    createMultipleRunInputs(count = 5): RunIO[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockRunInput({
                id: `input_${index}_${this.generateId()}`,
                nodeInputName: `input${index + 1}`,
                nodeName: `Node${index + 1}`,
            }),
        );
    }
    
    /**
     * Create multiple run outputs
     */
    createMultipleRunOutputs(count = 5): RunIO[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockRunOutput({
                id: `output_${index}_${this.generateId()}`,
                nodeInputName: `output${index + 1}`,
                nodeName: `Node${index + 1}`,
            }),
        );
    }
    
    /**
     * Validate run input create input
     */
    async validateInputCreateInput(input: RunIOCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            // Validation would be done here
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
    async validateOutputCreateInput(input: RunIOCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            // Validation would be done here
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
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create run input
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, async ({ request, params }) => {
                const body = await request.json() as RunIOCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateInputCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create run input
                const runInput = this.responseFactory.createRunInputFromInput(body);
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Create run output
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, async ({ request, params }) => {
                const body = await request.json() as RunIOCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateOutputCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create run output
                const runOutput = this.responseFactory.createRunOutputFromInput(body);
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get run input by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, ({ request, params }) => {
                const { id } = params;
                
                const runInput = this.responseFactory.createMockRunInput({ id: id as string });
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get run output by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, ({ request, params }) => {
                const { id } = params;
                
                const runOutput = this.responseFactory.createMockRunOutput({ id: id as string });
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update run input
            http.put(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as RunIOUpdateInput;
                
                const runInput = this.responseFactory.createMockRunInput({ 
                    id: id as string,
                    data: body.data || JSON.stringify({ updated: true }),
                });
                
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update run output
            http.put(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as RunIOUpdateInput;
                
                const runOutput = this.responseFactory.createMockRunOutput({ 
                    id: id as string,
                    data: body.data || JSON.stringify({ updated: true }),
                });
                
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete run input
            http.delete(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // Delete run output
            http.delete(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List run inputs
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const runId = url.searchParams.get("runId");
                
                let inputs = this.responseFactory.createMultipleRunInputs(20);
                
                // Filter by run ID if specified
                if (runId) {
                    inputs = inputs.filter(i => i.run.id === runId);
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
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // List run outputs
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const runId = url.searchParams.get("runId");
                
                let outputs = this.responseFactory.createMultipleRunOutputs(20);
                
                // Filter by run ID if specified
                if (runId) {
                    outputs = outputs.filter(o => o.run.id === runId);
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
                
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Validation error for input
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        data: "Input data is required",
                        nodeInputName: "Node input name is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Validation error for output
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        data: "Output data is required",
                        nodeInputName: "Node input name is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error for input
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse("input", id as string),
                    { status: 404 }
                );
            }),
            
            // Not found error for output
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse("output", id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createServerErrorResponse(),
                    { status: 500 }
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, async ({ request, params }) => {
                const body = await request.json() as RunIOCreateInput;
                const runInput = this.responseFactory.createRunInputFromInput(body);
                const response = this.responseFactory.createInputSuccessResponse(runInput);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 201 });
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, async ({ request, params }) => {
                const body = await request.json() as RunIOCreateInput;
                const runOutput = this.responseFactory.createRunOutputFromInput(body);
                const response = this.responseFactory.createOutputSuccessResponse(runOutput);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 201 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/run-input`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/run-input/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/run-output`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/run-output/:id`, ({ request, params }) => {
                return HttpResponse.error();
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
    }): RequestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        const httpMethod = http[method.toLowerCase() as keyof typeof http] as any;
        return httpMethod(fullEndpoint, async ({ request, params }) => {
            if (delay) {
                await new Promise(resolve => setTimeout(resolve, delay));
            }
            return HttpResponse.json(response, { status });
        });
    }
}

/**
 * Pre-configured response scenarios
 */
export const runIOResponseScenarios = {
    // Success scenarios
    createInputSuccess: (input?: RunIO) => {
        const factory = new RunIOResponseFactory();
        return factory.createInputSuccessResponse(
            input || factory.createMockRunInput(),
        );
    },
    
    createOutputSuccess: (output?: RunIO) => {
        const factory = new RunIOResponseFactory();
        return factory.createOutputSuccessResponse(
            output || factory.createMockRunOutput(),
        );
    },
    
    listInputsSuccess: (inputs?: RunIO[]) => {
        const factory = new RunIOResponseFactory();
        return factory.createInputListResponse(
            inputs || factory.createMultipleRunInputs(),
        );
    },
    
    listOutputsSuccess: (outputs?: RunIO[]) => {
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
