/**
 * Run API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for run endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    Run, 
    RunCreateInput, 
    RunUpdateInput,
    RunStatus,
    RunStep,
} from "@vrooli/shared";
import { 
    runValidation,
    RunStatus as RunStatusEnum, 
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
 * Run API response factory
 */
export class RunResponseFactory {
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
     * Generate unique run ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful run response
     */
    createSuccessResponse(run: Run): APIResponse<Run> {
        return {
            data: run,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run/${run.id}`,
                    related: {
                        resource: run.resourceVersion ? `${this.baseUrl}/api/resource-version/${run.resourceVersion.id}` : undefined,
                        steps: `${this.baseUrl}/api/run/${run.id}/steps`,
                        outputs: `${this.baseUrl}/api/run/${run.id}/outputs`,
                    },
                },
            },
        };
    }
    
    /**
     * Create run list response
     */
    createRunListResponse(runs: Run[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Run> {
        const paginationData = pagination || {
            page: 1,
            pageSize: runs.length,
            totalCount: runs.length,
        };
        
        return {
            data: runs,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/run",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(runId: string): APIErrorResponse {
        return {
            error: {
                code: "RUN_NOT_FOUND",
                message: `Run with ID '${runId}' was not found`,
                details: {
                    runId,
                    searchCriteria: { id: runId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/run/${runId}`,
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
                message: `You do not have permission to ${operation} this run`,
                details: {
                    operation,
                    requiredPermissions: ["run:write"],
                    userPermissions: ["run:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/run",
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
                path: "/api/run",
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
                path: "/api/run",
            },
        };
    }
    
    /**
     * Create mock run data
     */
    createMockRun(overrides?: Partial<Run>): Run {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultRun = {
            __typename: "Run",
            id,
            isPrivate: false,
            completedComplexity: 0,
            contextSwitches: 0,
            status: RunStatusEnum.Scheduled,
            startedAt: null,
            completedAt: null,
            timeElapsed: null,
            name: "Test Run",
            steps: [],
            stepsCount: 0,
            inputs: [],
            inputsCount: 0,
            outputs: [],
            outputsCount: 0,
            routineVersion: {
                __typename: "RoutineVersion",
                id: `routine_${id}`,
                createdAt: now,
                updatedAt: now,
                versionLabel: "1.0.0",
                versionNotes: "Initial version",
                complexity: 5,
                timesStarted: 10,
                timesCompleted: 8,
                isAutomatable: true,
                isComplete: true,
                isDeleted: false,
                isLatest: true,
                isPrivate: false,
                translations: [],
                translationsCount: 0,
                root: {
                    __typename: "Routine",
                    id: `routine_root_${id}`,
                    createdAt: now,
                    updatedAt: now,
                    isInternal: false,
                    isPrivate: false,
                    completedAt: null,
                    createdBy: null,
                    hasCompleteVersion: true,
                    score: 4.5,
                    bookmarks: 0,
                    views: 100,
                    you: {
                        __typename: "RoutineYou",
                        canComment: true,
                        canDelete: false,
                        canBookmark: true,
                        canUpdate: false,
                        canRead: true,
                        canReact: true,
                        isBookmarked: false,
                        reaction: null,
                    },
                },
                you: {
                    __typename: "VersionYou",
                    canComment: true,
                    canDelete: false,
                    canRead: true,
                    canReport: false,
                    canUpdate: false,
                },
            },
            you: {
                __typename: "RunYou",
                canDelete: true,
                canUpdate: true,
                canRead: true,
            },
        };
        
        return {
            ...defaultRun,
            ...overrides,
        } as unknown as Run;
    }
    
    /**
     * Create run from API input
     */
    createRunFromInput(input: RunCreateInput): Run {
        const run = this.createMockRun();
        
        // Update run based on input
        if (input.isPrivate !== undefined) {
            run.isPrivate = input.isPrivate;
        }
        
        if (input.name) {
            run.name = input.name;
        }
        
        if (input.status) {
            run.status = input.status;
        }
        
        return run;
    }
    
    /**
     * Create runs with different statuses
     */
    createRunsWithAllStatuses(): Run[] {
        return Object.values(RunStatusEnum).map(status => 
            this.createMockRun({
                status,
                name: `${status} Run`,
                startedAt: [RunStatusEnum.InProgress, RunStatusEnum.Completed, RunStatusEnum.Failed, RunStatusEnum.Cancelled].includes(status) 
                    ? new Date(Date.now() - 3600000).toISOString() 
                    : null,
                completedAt: [RunStatusEnum.Completed, RunStatusEnum.Failed, RunStatusEnum.Cancelled].includes(status)
                    ? new Date().toISOString()
                    : null,
                timeElapsed: [RunStatusEnum.Completed, RunStatusEnum.Failed, RunStatusEnum.Cancelled].includes(status)
                    ? 3600
                    : null,
                completedComplexity: status === RunStatusEnum.Completed ? 5 : 0,
            }),
        );
    }
    
    /**
     * Validate run create input
     */
    async validateCreateInput(input: RunCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await runValidation.create({}).validate(input);
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
 * MSW handlers factory for run endpoints
 */
export class RunMSWHandlers {
    private responseFactory: RunResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new RunResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all run endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create run
            http.post(`${this.responseFactory["baseUrl"]}/api/run`, async ({ request, params }) => {
                const body = await request.json() as RunCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create run
                const run = this.responseFactory.createRunFromInput(body);
                const response = this.responseFactory.createSuccessResponse(run);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get run by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/run/:id`, ({ request, params }) => {
                const { id } = params;
                
                const run = this.responseFactory.createMockRun({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(run);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update run
            http.put(`${this.responseFactory["baseUrl"]}/api/run/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as RunUpdateInput;
                
                const run = this.responseFactory.createMockRun({ 
                    id: id as string,
                });
                
                // Apply updates from body
                if (body.status) {
                    run.status = body.status;
                }
                
                const response = this.responseFactory.createSuccessResponse(run);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete run
            http.delete(`${this.responseFactory["baseUrl"]}/api/run/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List runs
            http.get(`${this.responseFactory["baseUrl"]}/api/run`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as RunStatus;
                
                let runs = this.responseFactory.createRunsWithAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    runs = runs.filter(r => r.status === status);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedRuns = runs.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createRunListResponse(
                    paginatedRuns,
                    {
                        page,
                        pageSize: limit,
                        totalCount: runs.length,
                    },
                );
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Start run
            http.post(`${this.responseFactory["baseUrl"]}/api/run/:id/start`, ({ request, params }) => {
                const { id } = params;
                
                const run = this.responseFactory.createMockRun({ 
                    id: id as string,
                    status: RunStatusEnum.InProgress,
                    startedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(run);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Complete run
            http.post(`${this.responseFactory["baseUrl"]}/api/run/:id/complete`, ({ request, params }) => {
                const { id } = params;
                
                const startedAt = new Date(Date.now() - 3600000).toISOString();
                const completedAt = new Date().toISOString();
                
                const run = this.responseFactory.createMockRun({ 
                    id: id as string,
                    status: RunStatusEnum.Completed,
                    startedAt,
                    completedAt,
                    timeElapsed: 3600,
                    completedComplexity: 5,
                });
                
                const response = this.responseFactory.createSuccessResponse(run);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Cancel run
            http.post(`${this.responseFactory["baseUrl"]}/api/run/:id/cancel`, ({ request, params }) => {
                const { id } = params;
                
                const run = this.responseFactory.createMockRun({ 
                    id: id as string,
                    status: RunStatusEnum.Cancelled,
                    completedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(run);
                
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/run`, () => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        routineVersionConnect: "Routine version is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/run/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/run`, () => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/run`, () => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/run`, async ({ request, params }) => {
                const body = await request.json() as RunCreateInput;
                const run = this.responseFactory.createRunFromInput(body);
                const response = this.responseFactory.createSuccessResponse(run);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/run`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/run/:id`, ({ request, params }) => {
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
        return httpMethod(fullEndpoint, async () => {
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
export const runResponseScenarios = {
    // Success scenarios
    createSuccess: (run?: Run) => {
        const factory = new RunResponseFactory();
        return factory.createSuccessResponse(
            run || factory.createMockRun(),
        );
    },
    
    listSuccess: (runs?: Run[]) => {
        const factory = new RunResponseFactory();
        return factory.createRunListResponse(
            runs || factory.createRunsWithAllStatuses(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new RunResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                routineVersionConnect: "Routine version is required",
            },
        );
    },
    
    notFoundError: (runId?: string) => {
        const factory = new RunResponseFactory();
        return factory.createNotFoundErrorResponse(
            runId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new RunResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new RunResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new RunMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new RunMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new RunMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new RunMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const runResponseFactory = new RunResponseFactory();
export const runMSWHandlers = new RunMSWHandlers();
