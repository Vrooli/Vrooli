/**
 * Schedule API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for schedule endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-schedule-recurrence-types | LAST: 2025-07-02 - Fixed ScheduleRecurrence properties to include required fields

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    Schedule, 
    ScheduleCreateInput, 
    ScheduleUpdateInput,
    ScheduleFor,
} from "@vrooli/shared";
import { 
    scheduleValidation,
    ScheduleFor as ScheduleForEnum,
    ScheduleRecurrenceType,
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
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Generate unique schedule ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
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
                        runs: `${this.baseUrl}/api/schedule/${schedule.id}/runs`,
                        meetings: `${this.baseUrl}/api/schedule/${schedule.id}/meetings`,
                        user: `${this.baseUrl}/api/user/${schedule.user?.id}`,
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
            createdAt: now,
            updatedAt: now,
            startTime: nextWeek,
            endTime: new Date(Date.now() + 8 * 24 * 60 * 60 * 1000).toISOString(),
            timezone: "UTC",
            publicId: `pub_${id}`,
            recurrences: [],
            exceptions: [],
            meetings: [],
            runs: [],
            user: {
                __typename: "User",
                id: "user-123",
                createdAt: now,
                updatedAt: now,
                name: "Test User",
                handle: "testuser",
                isBot: false,
                premium: "None",
                you: {
                    __typename: "UserYou",
                    canDelete: false,
                    canUpdate: false,
                    canReport: false,
                },
            } as any,
            // Remove 'you' property - it doesn't exist on Schedule type
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
        if (input.runConnect) {
            // Add a run to the runs array
            const newRun = {
                __typename: "Run",
                id: input.runConnect,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                startedAt: null,
                completedAt: null,
                status: "Scheduled",
                timeElapsed: 0,
                name: "Scheduled Run",
                contextSwitches: 0,
            } as any;
            schedule.runs = [newRun];
        }
        
        return schedule;
    }
    
    /**
     * Create schedules for different schedule types
     */
    createSchedulesForAllTypes(): Schedule[] {
        return [
            // Schedule with runs
            this.createMockSchedule({
                runs: [{
                    __typename: "Run",
                    id: `run_${this.generateId()}`,
                    startedAt: null,
                    completedAt: null,
                    status: "Scheduled",
                    timeElapsed: 0,
                    name: "Scheduled Run",
                    contextSwitches: 0,
                } as any],
            }),
            // Schedule with meetings
            this.createMockSchedule({
                meetings: [{
                    __typename: "Meeting",
                    id: `meeting_${this.generateId()}`,
                    openToAnyoneWithInvite: false,
                    showOnTeamProfile: false,
                } as any],
            }),
            // Basic schedule
            this.createMockSchedule(),
        ];
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
                    recurrenceType: ScheduleRecurrenceType.Daily,
                    interval: 1,
                    duration: 3600,
                    dayOfWeek: null,
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                    schedule: null,
                }],
            }),
            
            // Weekly schedule
            this.createMockSchedule({
                startTime: new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000).toISOString(),
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: `weekly_${this.generateId()}`,
                    recurrenceType: ScheduleRecurrenceType.Weekly,
                    interval: 1,
                    duration: 3600,
                    dayOfWeek: now.getDay(),
                    dayOfMonth: null,
                    month: null,
                    endDate: null,
                    schedule: null,
                }],
            }),
            
            // Monthly schedule
            this.createMockSchedule({
                startTime: new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000).toISOString(),
                recurrences: [{
                    __typename: "ScheduleRecurrence",
                    id: `monthly_${this.generateId()}`,
                    recurrenceType: ScheduleRecurrenceType.Monthly,
                    interval: 1,
                    duration: 3600,
                    dayOfWeek: null,
                    dayOfMonth: now.getDate(),
                    month: null,
                    endDate: null,
                    schedule: null,
                }],
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
            await scheduleValidation.create({}).validate(input);
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
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create schedule
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, async ({ request, params }) => {
                const body = await request.json() as ScheduleCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create schedule
                const schedule = this.responseFactory.createScheduleFromInput(body);
                const response = this.responseFactory.createSuccessResponse(schedule);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get schedule by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, ({ request, params }) => {
                const { id } = params;
                
                const schedule = this.responseFactory.createMockSchedule({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(schedule);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update schedule
            http.put(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as ScheduleUpdateInput;
                
                const schedule = this.responseFactory.createMockSchedule({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
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
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete schedule
            http.delete(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List schedules
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule`, ({ request, params }) => {
                const url = new URL(request.url);
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
                        if (scheduleFor === "Run") {
                            return s.runs.length > 0;
                        } else {
                            return s.meetings.length > 0;
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        startTime: "Start time is required",
                        timezone: "Timezone is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, ({ request, params }) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, async ({ request, params }) => {
                const body = await request.json() as ScheduleCreateInput;
                const schedule = this.responseFactory.createScheduleFromInput(body);
                const response = this.responseFactory.createSuccessResponse(schedule);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule/:id`, ({ request, params }) => {
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
