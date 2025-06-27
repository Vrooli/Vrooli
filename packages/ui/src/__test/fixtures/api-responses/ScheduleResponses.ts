/**
 * Schedule API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for schedule endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    Schedule, 
    ScheduleCreateInput, 
    ScheduleUpdateInput,
    ScheduleFor,
} from "@vrooli/shared";
import { 
    scheduleValidation,
    ScheduleFor as ScheduleForEnum, 
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
 * Schedule API response factory
 */
export class ScheduleResponseFactory {
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
     * Generate unique schedule ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful schedule response
     */
    createSuccessResponse(schedule: Schedule): APIResponse<Schedule> {
        return {
            data: schedule,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/schedule/${schedule.id}`,
                    related: {
                        recurrences: `${this.baseUrl}/api/schedule/${schedule.id}/recurrences`,
                        exceptions: `${this.baseUrl}/api/schedule/${schedule.id}/exceptions`,
                        runProject: schedule.runProject ? `${this.baseUrl}/api/run-project/${schedule.runProject.id}` : undefined,
                        runRoutine: schedule.runRoutine ? `${this.baseUrl}/api/run-routine/${schedule.runRoutine.id}` : undefined,
                    },
                },
            },
        };
    }
    
    /**
     * Create schedule list response
     */
    createScheduleListResponse(schedules: Schedule[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Schedule> {
        const paginationData = pagination || {
            page: 1,
            pageSize: schedules.length,
            totalCount: schedules.length,
        };
        
        return {
            data: schedules,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/schedule?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/schedule",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(scheduleId: string): APIErrorResponse {
        return {
            error: {
                code: "SCHEDULE_NOT_FOUND",
                message: `Schedule with ID '${scheduleId}' was not found`,
                details: {
                    scheduleId,
                    searchCriteria: { id: scheduleId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/schedule/${scheduleId}`,
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
                message: `You do not have permission to ${operation} this schedule`,
                details: {
                    operation,
                    requiredPermissions: ["schedule:write"],
                    userPermissions: ["schedule:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/schedule",
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
                path: "/api/schedule",
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
                path: "/api/schedule",
            },
        };
    }
    
    /**
     * Create mock schedule data
     */
    createMockSchedule(overrides?: Partial<Schedule>): Schedule {
        const now = new Date().toISOString();
        const id = this.generateId();
        const nextWeek = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString();
        
        const defaultSchedule: Schedule = {
            __typename: "Schedule",
            id,
            created_at: now,
            updated_at: now,
            startTime: nextWeek,
            endTime: null,
            timezone: "UTC",
            recurrences: [],
            recurrencesCount: 0,
            exceptions: [],
            exceptionsCount: 0,
            labels: [],
            labelsCount: 0,
            focusModes: [],
            focusModesCount: 0,
            meetings: [],
            meetingsCount: 0,
            runProject: null,
            runRoutine: {
                __typename: "RunRoutine",
                id: `run_${id}`,
                completedAt: null,
                startedAt: null,
                isPrivate: false,
                timeElapsed: 0,
                title: "Scheduled Routine Run",
                runProject: null,
                steps: [],
                inputs: [],
                outputs: [],
            },
            you: {
                __typename: "ScheduleYou",
                canDelete: true,
                canUpdate: true,
                canRead: true,
            },
        };
        
        return {
            ...defaultSchedule,
            ...overrides,
        };
    }
    
    /**
     * Create schedule from API input
     */
    createScheduleFromInput(input: ScheduleCreateInput): Schedule {
        const schedule = this.createMockSchedule();
        
        // Update schedule based on input
        if (input.startTime) {
            schedule.startTime = input.startTime;
        }
        
        if (input.endTime) {
            schedule.endTime = input.endTime;
        }
        
        if (input.timezone) {
            schedule.timezone = input.timezone;
        }
        
        // Handle run connections
        if (input.runProjectConnect) {
            schedule.runProject = {
                __typename: "RunProject",
                id: input.runProjectConnect,
                completedAt: null,
                startedAt: null,
                isPrivate: false,
                timeElapsed: 0,
                title: "Scheduled Project Run",
                steps: [],
            };
            schedule.runRoutine = null;
        } else if (input.runRoutineConnect) {
            schedule.runRoutine = {
                ...schedule.runRoutine!,
                id: input.runRoutineConnect,
            };
        }
        
        return schedule;
    }
    
    /**
     * Create schedules for different schedule types
     */
    createSchedulesForAllTypes(): Schedule[] {
        return Object.values(ScheduleForEnum).map(scheduleFor => {
            const base = this.createMockSchedule();
            
            if (scheduleFor === ScheduleForEnum.RunProject) {
                return {
                    ...base,
                    runProject: {
                        __typename: "RunProject",
                        id: `project_${this.generateId()}`,
                        completedAt: null,
                        startedAt: null,
                        isPrivate: false,
                        timeElapsed: 0,
                        title: "Project Schedule",
                        steps: [],
                    },
                    runRoutine: null,
                };
            } else {
                return {
                    ...base,
                    runRoutine: {
                        ...base.runRoutine!,
                        title: "Routine Schedule",
                    },
                };
            }
        });
    }
    
    /**
     * Create daily, weekly, and monthly schedules
     */
    createVariousScheduleTypes(): Schedule[] {
        const now = new Date();
        
        return [
            // Daily schedule
            this.createMockSchedule({
                startTime: new Date(now.getTime() + 24 * 60 * 60 * 1000).toISOString(),
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: `daily_${this.generateId()}`,
                    recurrenceType: "Daily",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                }],
                recurrencesCount: 1,
            }),
            
            // Weekly schedule
            this.createMockSchedule({
                startTime: new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000).toISOString(),
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: `weekly_${this.generateId()}`,
                    recurrenceType: "Weekly",
                    interval: 1,
                    dayOfWeek: now.getDay(),
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                }],
                recurrencesCount: 1,
            }),
            
            // Monthly schedule
            this.createMockSchedule({
                startTime: new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000).toISOString(),
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: `monthly_${this.generateId()}`,
                    recurrenceType: "Monthly",
                    interval: 1,
                    dayOfWeek: null,
                    dayOfMonth: now.getDate(),
                    month: null,
                    endDate: null,
                }],
                recurrencesCount: 1,
            }),
        ];
    }
    
    /**
     * Validate schedule create input
     */
    async validateCreateInput(input: ScheduleCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await scheduleValidation.create.validate(input);
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
 * MSW handlers factory for schedule endpoints
 */
export class ScheduleMSWHandlers {
    private responseFactory: ScheduleResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ScheduleResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all schedule endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create schedule
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, async (req, res, ctx) => {
                const body = await req.json() as ScheduleCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create schedule
                const schedule = this.responseFactory.createScheduleFromInput(body);
                const response = this.responseFactory.createSuccessResponse(schedule);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get schedule by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const schedule = this.responseFactory.createMockSchedule({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(schedule);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update schedule
            http.put(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ScheduleUpdateInput;
                
                const schedule = this.responseFactory.createMockSchedule({ 
                    id: id as string,
                    updated_at: new Date().toISOString(),
                });
                
                // Apply updates from body
                if (body.startTime) {
                    schedule.startTime = body.startTime;
                }
                
                if (body.endTime !== undefined) {
                    schedule.endTime = body.endTime;
                }
                
                if (body.timezone) {
                    schedule.timezone = body.timezone;
                }
                
                const response = this.responseFactory.createSuccessResponse(schedule);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete schedule
            http.delete(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List schedules
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const scheduleFor = url.searchParams.get("scheduleFor") as ScheduleFor;
                const upcoming = url.searchParams.get("upcoming") === "true";
                
                let schedules = upcoming 
                    ? this.responseFactory.createVariousScheduleTypes()
                    : this.responseFactory.createSchedulesForAllTypes();
                
                // Filter by schedule type if specified
                if (scheduleFor) {
                    schedules = schedules.filter(s => {
                        if (scheduleFor === ScheduleForEnum.RunProject) {
                            return s.runProject !== null;
                        } else {
                            return s.runRoutine !== null;
                        }
                    });
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedSchedules = schedules.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createScheduleListResponse(
                    paginatedSchedules,
                    {
                        page,
                        pageSize: limit,
                        totalCount: schedules.length,
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
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        startTime: "Start time is required",
                        timezone: "Timezone is required",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, async (req, res, ctx) => {
                const body = await req.json() as ScheduleCreateInput;
                const schedule = this.responseFactory.createScheduleFromInput(body);
                const response = this.responseFactory.createSuccessResponse(schedule);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, (req, res, ctx) => {
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
export const scheduleResponseScenarios = {
    // Success scenarios
    createSuccess: (schedule?: Schedule) => {
        const factory = new ScheduleResponseFactory();
        return factory.createSuccessResponse(
            schedule || factory.createMockSchedule(),
        );
    },
    
    listSuccess: (schedules?: Schedule[]) => {
        const factory = new ScheduleResponseFactory();
        return factory.createScheduleListResponse(
            schedules || factory.createSchedulesForAllTypes(),
        );
    },
    
    upcomingSchedules: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createScheduleListResponse(
            factory.createVariousScheduleTypes(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ScheduleResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                startTime: "Start time is required",
                timezone: "Timezone is required",
            },
        );
    },
    
    notFoundError: (scheduleId?: string) => {
        const factory = new ScheduleResponseFactory();
        return factory.createNotFoundErrorResponse(
            scheduleId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ScheduleResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ScheduleResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ScheduleMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ScheduleMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ScheduleMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ScheduleMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const scheduleResponseFactory = new ScheduleResponseFactory();
export const scheduleMSWHandlers = new ScheduleMSWHandlers();
