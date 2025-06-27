/**
 * ScheduleRecurrence API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for schedule recurrence endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    ScheduleRecurrence, 
    ScheduleRecurrenceCreateInput, 
    ScheduleRecurrenceUpdateInput,
    RecurrenceType,
} from "@vrooli/shared";
import { 
    scheduleRecurrenceValidation,
    RecurrenceType as RecurrenceTypeEnum, 
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
 * ScheduleRecurrence API response factory
 */
export class ScheduleRecurrenceResponseFactory {
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
     * Generate unique schedule recurrence ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful schedule recurrence response
     */
    createSuccessResponse(recurrence: ScheduleRecurrence): APIResponse<ScheduleRecurrence> {
        return {
            data: recurrence,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/schedule-recurrence/${recurrence.id}`,
                    related: {
                        schedule: `${this.baseUrl}/api/schedule/${recurrence.schedule?.id}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create schedule recurrence list response
     */
    createRecurrenceListResponse(recurrences: ScheduleRecurrence[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ScheduleRecurrence> {
        const paginationData = pagination || {
            page: 1,
            pageSize: recurrences.length,
            totalCount: recurrences.length,
        };
        
        return {
            data: recurrences,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/schedule-recurrence?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/schedule-recurrence",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(recurrenceId: string): APIErrorResponse {
        return {
            error: {
                code: "SCHEDULE_RECURRENCE_NOT_FOUND",
                message: `Schedule recurrence with ID '${recurrenceId}' was not found`,
                details: {
                    recurrenceId,
                    searchCriteria: { id: recurrenceId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/schedule-recurrence/${recurrenceId}`,
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
                message: `You do not have permission to ${operation} this schedule recurrence`,
                details: {
                    operation,
                    requiredPermissions: ["schedule:write"],
                    userPermissions: ["schedule:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/schedule-recurrence",
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
                path: "/api/schedule-recurrence",
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
                path: "/api/schedule-recurrence",
            },
        };
    }
    
    /**
     * Create mock schedule recurrence data
     */
    createMockRecurrence(overrides?: Partial<ScheduleRecurrence>): ScheduleRecurrence {
        const now = new Date().toISOString();
        const id = this.generateId();
        const nextYear = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString();
        
        const defaultRecurrence: ScheduleRecurrence = {
            __typename: "ScheduleRecurrence",
            id,
            recurrenceType: RecurrenceTypeEnum.Daily,
            interval: 1,
            dayOfWeek: null,
            dayOfMonth: null,
            month: null,
            endDate: nextYear,
            schedule: {
                __typename: "Schedule",
                id: `schedule_${id}`,
                created_at: now,
                updated_at: now,
                startTime: now,
                endTime: null,
                timezone: "UTC",
                recurrences: [],
                recurrencesCount: 1,
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
                    title: "Recurring Run",
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
            },
        };
        
        return {
            ...defaultRecurrence,
            ...overrides,
        };
    }
    
    /**
     * Create schedule recurrence from API input
     */
    createRecurrenceFromInput(input: ScheduleRecurrenceCreateInput): ScheduleRecurrence {
        const recurrence = this.createMockRecurrence();
        
        // Update recurrence based on input
        if (input.recurrenceType) {
            recurrence.recurrenceType = input.recurrenceType;
        }
        
        if (input.interval) {
            recurrence.interval = input.interval;
        }
        
        if (input.dayOfWeek !== undefined) {
            recurrence.dayOfWeek = input.dayOfWeek;
        }
        
        if (input.dayOfMonth !== undefined) {
            recurrence.dayOfMonth = input.dayOfMonth;
        }
        
        if (input.month !== undefined) {
            recurrence.month = input.month;
        }
        
        if (input.endDate !== undefined) {
            recurrence.endDate = input.endDate;
        }
        
        if (input.scheduleConnect) {
            recurrence.schedule.id = input.scheduleConnect;
        }
        
        return recurrence;
    }
    
    /**
     * Create recurrences for different types
     */
    createRecurrencesForAllTypes(): ScheduleRecurrence[] {
        const now = new Date();
        
        return [
            // Daily recurrence
            this.createMockRecurrence({
                recurrenceType: RecurrenceTypeEnum.Daily,
                interval: 1,
                dayOfWeek: null,
                dayOfMonth: null,
                month: null,
            }),
            
            // Weekly recurrence (every Monday)
            this.createMockRecurrence({
                recurrenceType: RecurrenceTypeEnum.Weekly,
                interval: 1,
                dayOfWeek: 1, // Monday
                dayOfMonth: null,
                month: null,
            }),
            
            // Monthly recurrence (15th of each month)
            this.createMockRecurrence({
                recurrenceType: RecurrenceTypeEnum.Monthly,
                interval: 1,
                dayOfWeek: null,
                dayOfMonth: 15,
                month: null,
            }),
            
            // Yearly recurrence (January 1st)
            this.createMockRecurrence({
                recurrenceType: RecurrenceTypeEnum.Yearly,
                interval: 1,
                dayOfWeek: null,
                dayOfMonth: 1,
                month: 1, // January
            }),
            
            // Every other week
            this.createMockRecurrence({
                recurrenceType: RecurrenceTypeEnum.Weekly,
                interval: 2,
                dayOfWeek: 5, // Friday
                dayOfMonth: null,
                month: null,
            }),
        ];
    }
    
    /**
     * Create recurrences for a specific schedule
     */
    createRecurrencesForSchedule(scheduleId: string, count = 2): ScheduleRecurrence[] {
        return Array.from({ length: count }, (_, index) => {
            const recurrence = this.createMockRecurrence({
                recurrenceType: index === 0 ? RecurrenceTypeEnum.Daily : RecurrenceTypeEnum.Weekly,
                interval: index + 1,
            });
            recurrence.schedule.id = scheduleId;
            return recurrence;
        });
    }
    
    /**
     * Validate schedule recurrence create input
     */
    async validateCreateInput(input: ScheduleRecurrenceCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await scheduleRecurrenceValidation.create.validate(input);
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
 * MSW handlers factory for schedule recurrence endpoints
 */
export class ScheduleRecurrenceMSWHandlers {
    private responseFactory: ScheduleRecurrenceResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ScheduleRecurrenceResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all schedule recurrence endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create schedule recurrence
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, async (req, res, ctx) => {
                const body = await req.json() as ScheduleRecurrenceCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create recurrence
                const recurrence = this.responseFactory.createRecurrenceFromInput(body);
                const response = this.responseFactory.createSuccessResponse(recurrence);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get schedule recurrence by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const recurrence = this.responseFactory.createMockRecurrence({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(recurrence);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update schedule recurrence
            http.put(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ScheduleRecurrenceUpdateInput;
                
                const recurrence = this.responseFactory.createMockRecurrence({ id: id as string });
                
                // Apply updates from body
                if (body.recurrenceType) {
                    recurrence.recurrenceType = body.recurrenceType;
                }
                
                if (body.interval) {
                    recurrence.interval = body.interval;
                }
                
                if (body.dayOfWeek !== undefined) {
                    recurrence.dayOfWeek = body.dayOfWeek;
                }
                
                if (body.dayOfMonth !== undefined) {
                    recurrence.dayOfMonth = body.dayOfMonth;
                }
                
                if (body.month !== undefined) {
                    recurrence.month = body.month;
                }
                
                if (body.endDate !== undefined) {
                    recurrence.endDate = body.endDate;
                }
                
                const response = this.responseFactory.createSuccessResponse(recurrence);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete schedule recurrence
            http.delete(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List schedule recurrences
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const scheduleId = url.searchParams.get("scheduleId");
                const recurrenceType = url.searchParams.get("recurrenceType") as RecurrenceType;
                
                let recurrences = scheduleId 
                    ? this.responseFactory.createRecurrencesForSchedule(scheduleId)
                    : this.responseFactory.createRecurrencesForAllTypes();
                
                // Filter by recurrence type if specified
                if (recurrenceType) {
                    recurrences = recurrences.filter(r => r.recurrenceType === recurrenceType);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedRecurrences = recurrences.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createRecurrenceListResponse(
                    paginatedRecurrences,
                    {
                        page,
                        pageSize: limit,
                        totalCount: recurrences.length,
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        recurrenceType: "Recurrence type is required",
                        interval: "Interval must be greater than 0",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, async (req, res, ctx) => {
                const body = await req.json() as ScheduleRecurrenceCreateInput;
                const recurrence = this.responseFactory.createRecurrenceFromInput(body);
                const response = this.responseFactory.createSuccessResponse(recurrence);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-recurrence/:id`, (req, res, ctx) => {
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
export const scheduleRecurrenceResponseScenarios = {
    // Success scenarios
    createSuccess: (recurrence?: ScheduleRecurrence) => {
        const factory = new ScheduleRecurrenceResponseFactory();
        return factory.createSuccessResponse(
            recurrence || factory.createMockRecurrence(),
        );
    },
    
    listSuccess: (recurrences?: ScheduleRecurrence[]) => {
        const factory = new ScheduleRecurrenceResponseFactory();
        return factory.createRecurrenceListResponse(
            recurrences || factory.createRecurrencesForAllTypes(),
        );
    },
    
    dailyRecurrence: () => {
        const factory = new ScheduleRecurrenceResponseFactory();
        const recurrence = factory.createMockRecurrence({
            recurrenceType: RecurrenceTypeEnum.Daily,
            interval: 1,
        });
        return factory.createSuccessResponse(recurrence);
    },
    
    weeklyRecurrence: () => {
        const factory = new ScheduleRecurrenceResponseFactory();
        const recurrence = factory.createMockRecurrence({
            recurrenceType: RecurrenceTypeEnum.Weekly,
            interval: 1,
            dayOfWeek: 1, // Monday
        });
        return factory.createSuccessResponse(recurrence);
    },
    
    monthlyRecurrence: () => {
        const factory = new ScheduleRecurrenceResponseFactory();
        const recurrence = factory.createMockRecurrence({
            recurrenceType: RecurrenceTypeEnum.Monthly,
            interval: 1,
            dayOfMonth: 15,
        });
        return factory.createSuccessResponse(recurrence);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ScheduleRecurrenceResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                recurrenceType: "Recurrence type is required",
                interval: "Interval must be greater than 0",
            },
        );
    },
    
    notFoundError: (recurrenceId?: string) => {
        const factory = new ScheduleRecurrenceResponseFactory();
        return factory.createNotFoundErrorResponse(
            recurrenceId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ScheduleRecurrenceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ScheduleRecurrenceResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ScheduleRecurrenceMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ScheduleRecurrenceMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ScheduleRecurrenceMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ScheduleRecurrenceMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const scheduleRecurrenceResponseFactory = new ScheduleRecurrenceResponseFactory();
export const scheduleRecurrenceMSWHandlers = new ScheduleRecurrenceMSWHandlers();
