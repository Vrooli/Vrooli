/**
 * RunStep API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for run step endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-runstep-types | LAST: 2025-07-02 - Fixed RunStep type usage and MSW v2 syntax

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    RunStep,
    RunStepStatus,
    RunStepCreateInput,
    RunStepUpdateInput,
    ResourceVersion,
    Run,
} from "@vrooli/shared";
import { RunStepStatus as RunStepStatusEnum } from "@vrooli/shared";

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
 * RunStep API response factory
 */
export class RunStepResponseFactory {
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
     * Generate unique run step ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful run step response
     */
    createSuccessResponse(runStep: RunStep): APIResponse<RunStep> {
        return {
            data: runStep,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-step/${runStep.id}`,
                    related: this.getRelatedLinks(runStep),
                },
            },
        };
    }
    
    /**
     * Get related links based on run step type
     */
    private getRelatedLinks(runStep: RunStep): Record<string, string> {
        const links: Record<string, string> = {};
        
        if (runStep.resourceVersion) {
            links.resourceVersion = `${this.baseUrl}/api/resource-version/${runStep.resourceVersion.id}`;
        }
        if (runStep.nodeId) {
            links.node = `${this.baseUrl}/api/node/${runStep.nodeId}`;
        }
        
        return links;
    }
    
    /**
     * Create run step list response
     */
    createRunStepListResponse(runSteps: RunStep[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<RunStep> {
        const paginationData = pagination || {
            page: 1,
            pageSize: runSteps.length,
            totalCount: runSteps.length,
        };
        
        return {
            data: runSteps,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/run-step?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/run-step",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(runStepId: string): APIErrorResponse {
        return {
            error: {
                code: "RUN_STEP_NOT_FOUND",
                message: `Run step with ID '${runStepId}' was not found`,
                details: {
                    runStepId,
                    searchCriteria: { id: runStepId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/run-step/${runStepId}`,
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
                message: `You do not have permission to ${operation} this run step`,
                details: {
                    operation,
                    requiredPermissions: ["run:write"],
                    userPermissions: ["run:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/run-step",
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
                path: "/api/run-step",
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
                path: "/api/run-step",
            },
        };
    }
    
    /**
     * Create mock run step data
     */
    createMockRunStep(overrides?: Partial<RunStep>): RunStep {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStep: RunStep = {
            __typename: "RunStep",
            id,
            completedAt: null,
            complexity: 1,
            contextSwitches: 0,
            name: "Test Step",
            nodeId: `node_${id}`,
            order: 1,
            resourceInId: null,
            resourceVersion: null,
            startedAt: null,
            status: RunStepStatusEnum.InProgress,
            timeElapsed: null,
        };
        
        return {
            ...defaultStep,
            ...overrides,
        };
    }
    
    /**
     * Create run steps with different statuses
     */
    createRunStepsWithAllStatuses(): RunStep[] {
        return Object.values(RunStepStatusEnum).map((status, index) => {
            const baseStep = this.createMockRunStep();
            
            return {
                ...baseStep,
                status,
                order: index + 1,
                startedAt: [RunStepStatusEnum.InProgress, RunStepStatusEnum.Completed, RunStepStatusEnum.Skipped].includes(status)
                    ? new Date(Date.now() - 1800000).toISOString()
                    : null,
                completedAt: [RunStepStatusEnum.Completed, RunStepStatusEnum.Skipped].includes(status)
                    ? new Date().toISOString()
                    : null,
                timeElapsed: [RunStepStatusEnum.Completed, RunStepStatusEnum.Skipped].includes(status)
                    ? 1800
                    : null,
            };
        });
    }
    
    /**
     * Update run step status
     */
    updateRunStepStatus(runStep: RunStep, status: RunStepStatus): RunStep {
        const now = new Date().toISOString();
        const updatedStep = { ...runStep, status };
        
        if (status === RunStepStatusEnum.InProgress && !updatedStep.startedAt) {
            updatedStep.startedAt = now;
        }
        
        if ([RunStepStatusEnum.Completed, RunStepStatusEnum.Skipped].includes(status)) {
            if (!updatedStep.startedAt) {
                updatedStep.startedAt = new Date(Date.now() - 1800000).toISOString();
            }
            updatedStep.completedAt = now;
            updatedStep.timeElapsed = updatedStep.startedAt 
                ? Math.floor((new Date(now).getTime() - new Date(updatedStep.startedAt).getTime()) / 1000)
                : 0;
        }
        
        return updatedStep;
    }
}

/**
 * MSW handlers factory for run step endpoints
 */
export class RunStepMSWHandlers {
    private responseFactory: RunStepResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new RunStepResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all run step endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Get run step by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/run-step/:id`, ({ request, params }) => {
                const { id } = params;
                
                const runStep = this.responseFactory.createMockRunStep({ id: id as string });
                
                const response = this.responseFactory.createSuccessResponse(runStep);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update run step status
            http.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, async ({ request, params }) => {
                const { id } = params;
                const { status } = await request.json() as { status: RunStepStatus };
                
                // Create a run step and update its status
                const runStep = this.responseFactory.createMockRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, status);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // List run steps
            http.get(`${this.responseFactory["baseUrl"]}/api/run-step`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as RunStepStatus;
                const runId = url.searchParams.get("runId");
                
                let runSteps = this.responseFactory.createRunStepsWithAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    runSteps = runSteps.filter(s => s.status === status);
                }
                
                // Filter by run ID if specified - RunStep doesn't have direct run reference
                if (runId) {
                    // For now, we'll filter based on a pattern in the ID
                    runSteps = runSteps.filter(s => s.id.includes(runId));
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedSteps = runSteps.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createRunStepListResponse(
                    paginatedSteps,
                    {
                        page,
                        pageSize: limit,
                        totalCount: runSteps.length,
                    },
                );
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Start run step
            http.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/start`, ({ request, params }) => {
                const { id } = params;
                
                const runStep = this.responseFactory.createMockRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, RunStepStatusEnum.InProgress);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Complete run step
            http.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/complete`, ({ request, params }) => {
                const { id } = params;
                
                const runStep = this.responseFactory.createMockRunStep({ 
                    id: id as string,
                    startedAt: new Date(Date.now() - 1800000).toISOString(),
                });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, RunStepStatusEnum.Completed);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Skip run step
            http.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/skip`, ({ request, params }) => {
                const { id } = params;
                
                const runStep = this.responseFactory.createMockRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, RunStepStatusEnum.Skipped);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/run-step/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("update"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/start`, ({ request, params }) => {
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
            http.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, async ({ request, params }) => {
                const { id } = params;
                const { status } = await request.json() as { status: RunStepStatus };
                
                const runStep = this.responseFactory.createMockRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, status);
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.get(`${this.responseFactory["baseUrl"]}/api/run-step/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, ({ request, params }) => {
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
        
        const httpMethod = method.toLowerCase() as keyof typeof http;
        return http[httpMethod](fullEndpoint, async ({ request, params }) => {
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
export const runStepResponseScenarios = {
    // Success scenarios
    createStepSuccess: (step?: RunStep) => {
        const factory = new RunStepResponseFactory();
        return factory.createSuccessResponse(
            step || factory.createMockRunStep(),
        );
    },
    
    listSuccess: (runSteps?: RunStep[]) => {
        const factory = new RunStepResponseFactory();
        return factory.createRunStepListResponse(
            runSteps || factory.createRunStepsWithAllStatuses(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new RunStepResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                status: "Invalid status value",
            },
        );
    },
    
    notFoundError: (runStepId?: string) => {
        const factory = new RunStepResponseFactory();
        return factory.createNotFoundErrorResponse(
            runStepId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new RunStepResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "update",
        );
    },
    
    serverError: () => {
        const factory = new RunStepResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new RunStepMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new RunStepMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new RunStepMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new RunStepMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const runStepResponseFactory = new RunStepResponseFactory();
export const runStepMSWHandlers = new RunStepMSWHandlers();
