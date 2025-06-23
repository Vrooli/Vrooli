/**
 * RunStep API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for run step endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from "msw";
import type { 
    RunStep,
    RunProjectStep,
    RunRoutineStep,
    RunStepStatus,
} from "@vrooli/shared";
import { 
    runStepValidation,
    RunStepStatus as RunStepStatusEnum, 
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
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique run step ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
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
        
        if (runStep.__typename === "RunRoutineStep" && runStep.runRoutine) {
            links.runRoutine = `${this.baseUrl}/api/run/${runStep.runRoutine.id}`;
            if (runStep.step) {
                links.step = `${this.baseUrl}/api/routine-step/${runStep.step.id}`;
            }
        } else if (runStep.__typename === "RunProjectStep" && runStep.runProject) {
            links.runProject = `${this.baseUrl}/api/run-project/${runStep.runProject.id}`;
            if (runStep.directory) {
                links.directory = `${this.baseUrl}/api/project-directory/${runStep.directory.id}`;
            }
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
     * Create mock routine run step data
     */
    createMockRoutineRunStep(overrides?: Partial<RunRoutineStep>): RunRoutineStep {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStep: RunRoutineStep = {
            __typename: "RunRoutineStep",
            id,
            order: 1,
            contextSwitches: 0,
            startedAt: null,
            completedAt: null,
            timeElapsed: null,
            status: RunStepStatusEnum.NotStarted,
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
            step: {
                __typename: "RoutineVersionStep",
                id: `step_${id}`,
                index: 0,
                name: "Test Step",
                optional: false,
                translations: [{
                    __typename: "RoutineVersionStepTranslation",
                    id: `trans_${id}`,
                    language: "en",
                    description: "Test step description",
                    title: "Test Step",
                }],
            },
        };
        
        return {
            ...defaultStep,
            ...overrides,
        };
    }
    
    /**
     * Create mock project run step data
     */
    createMockProjectRunStep(overrides?: Partial<RunProjectStep>): RunProjectStep {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStep: RunProjectStep = {
            __typename: "RunProjectStep",
            id,
            order: 1,
            contextSwitches: 0,
            startedAt: null,
            completedAt: null,
            timeElapsed: null,
            status: RunStepStatusEnum.NotStarted,
            runProject: {
                __typename: "RunProject",
                id: `project_${id}`,
                completedAt: null,
                startedAt: now,
                isPrivate: false,
                timeElapsed: 0,
                title: "Test Project Run",
                steps: [],
            },
            directory: {
                __typename: "ProjectVersionDirectory",
                id: `dir_${id}`,
                created_at: now,
                updated_at: now,
                isRoot: false,
                childOrder: 0,
            },
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
            const baseStep = index % 2 === 0 
                ? this.createMockRoutineRunStep()
                : this.createMockProjectRunStep();
            
            return {
                ...baseStep,
                status,
                order: index + 1,
                startedAt: [RunStepStatusEnum.InProgress, RunStepStatusEnum.Completed, RunStepStatusEnum.Failed, RunStepStatusEnum.Skipped].includes(status)
                    ? new Date(Date.now() - 1800000).toISOString()
                    : null,
                completedAt: [RunStepStatusEnum.Completed, RunStepStatusEnum.Failed, RunStepStatusEnum.Skipped].includes(status)
                    ? new Date().toISOString()
                    : null,
                timeElapsed: [RunStepStatusEnum.Completed, RunStepStatusEnum.Failed, RunStepStatusEnum.Skipped].includes(status)
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
        
        if ([RunStepStatusEnum.Completed, RunStepStatusEnum.Failed, RunStepStatusEnum.Skipped].includes(status)) {
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get run step by ID
            rest.get(`${this.responseFactory["baseUrl"]}/api/run-step/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                // Randomly return routine or project step
                const runStep = Math.random() > 0.5
                    ? this.responseFactory.createMockRoutineRunStep({ id: id as string })
                    : this.responseFactory.createMockProjectRunStep({ id: id as string });
                
                const response = this.responseFactory.createSuccessResponse(runStep);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update run step status
            rest.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, async (req, res, ctx) => {
                const { id } = req.params;
                const { status } = await req.json() as { status: RunStepStatus };
                
                // Create a run step and update its status
                const runStep = this.responseFactory.createMockRoutineRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, status);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // List run steps
            rest.get(`${this.responseFactory["baseUrl"]}/api/run-step`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as RunStepStatus;
                const runId = url.searchParams.get("runId");
                
                let runSteps = this.responseFactory.createRunStepsWithAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    runSteps = runSteps.filter(s => s.status === status);
                }
                
                // Filter by run ID if specified
                if (runId) {
                    runSteps = runSteps.filter(s => {
                        if (s.__typename === "RunRoutineStep") {
                            return s.runRoutine?.id === runId;
                        } else if (s.__typename === "RunProjectStep") {
                            return s.runProject?.id === runId;
                        }
                        return false;
                    });
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
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Start run step
            rest.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/start`, (req, res, ctx) => {
                const { id } = req.params;
                
                const runStep = this.responseFactory.createMockRoutineRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, RunStepStatusEnum.InProgress);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Complete run step
            rest.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/complete`, (req, res, ctx) => {
                const { id } = req.params;
                
                const runStep = this.responseFactory.createMockRoutineRunStep({ 
                    id: id as string,
                    startedAt: new Date(Date.now() - 1800000).toISOString(),
                });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, RunStepStatusEnum.Completed);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Skip run step
            rest.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/skip`, (req, res, ctx) => {
                const { id } = req.params;
                
                const runStep = this.responseFactory.createMockRoutineRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, RunStepStatusEnum.Skipped);
                
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
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
            // Not found error
            rest.get(`${this.responseFactory["baseUrl"]}/api/run-step/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            rest.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("update")),
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory["baseUrl"]}/api/run-step/:id/start`, (req, res, ctx) => {
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
            rest.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, async (req, res, ctx) => {
                const { id } = req.params;
                const { status } = await req.json() as { status: RunStepStatus };
                
                const runStep = this.responseFactory.createMockRoutineRunStep({ id: id as string });
                const updatedStep = this.responseFactory.updateRunStepStatus(runStep, status);
                const response = this.responseFactory.createSuccessResponse(updatedStep);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
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
            rest.get(`${this.responseFactory["baseUrl"]}/api/run-step/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            rest.put(`${this.responseFactory["baseUrl"]}/api/run-step/:id/status`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
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
export const runStepResponseScenarios = {
    // Success scenarios
    createRoutineStepSuccess: (step?: RunRoutineStep) => {
        const factory = new RunStepResponseFactory();
        return factory.createSuccessResponse(
            step || factory.createMockRoutineRunStep(),
        );
    },
    
    createProjectStepSuccess: (step?: RunProjectStep) => {
        const factory = new RunStepResponseFactory();
        return factory.createSuccessResponse(
            step || factory.createMockProjectRunStep(),
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
