/**
 * ScheduleException API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for schedule exception endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    ScheduleException, 
    ScheduleExceptionCreateInput, 
    ScheduleExceptionUpdateInput,
} from "@vrooli/shared";
import { 
    scheduleExceptionValidation, 
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
 * ScheduleException API response factory
 */
export class ScheduleExceptionResponseFactory {
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
     * Generate unique schedule exception ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful schedule exception response
     */
    createSuccessResponse(exception: ScheduleException): APIResponse<ScheduleException> {
        return {
            data: exception,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/schedule-exception/${exception.id}`,
                    related: {
                        schedule: `${this.baseUrl}/api/schedule/${exception.schedule?.id}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create schedule exception list response
     */
    createExceptionListResponse(exceptions: ScheduleException[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ScheduleException> {
        const paginationData = pagination || {
            page: 1,
            pageSize: exceptions.length,
            totalCount: exceptions.length,
        };
        
        return {
            data: exceptions,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/schedule-exception?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/schedule-exception",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(exceptionId: string): APIErrorResponse {
        return {
            error: {
                code: "SCHEDULE_EXCEPTION_NOT_FOUND",
                message: `Schedule exception with ID '${exceptionId}' was not found`,
                details: {
                    exceptionId,
                    searchCriteria: { id: exceptionId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/schedule-exception/${exceptionId}`,
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
                message: `You do not have permission to ${operation} this schedule exception`,
                details: {
                    operation,
                    requiredPermissions: ["schedule:write"],
                    userPermissions: ["schedule:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/schedule-exception",
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
                path: "/api/schedule-exception",
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
                path: "/api/schedule-exception",
            },
        };
    }
    
    /**
     * Create mock schedule exception data
     */
    createMockException(overrides?: Partial<ScheduleException>): ScheduleException {
        const now = new Date().toISOString();
        const id = this.generateId();
        const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
        
        const defaultException: ScheduleException = {
            __typename: "ScheduleException",
            id,
            originalStartTime: tomorrow,
            newStartTime: null,
            newEndTime: null,
            schedule: {
                __typename: "Schedule",
                id: `schedule_${id}`,
                createdAt: now,
                updatedAt: now,
                startTime: tomorrow,
                endTime: null,
                timezone: "UTC",
                recurrences: [],
                exceptions: [],
            } as any,
        };
        
        return {
            ...defaultException,
            ...overrides,
        } as unknown as ScheduleException;
    }
    
    /**
     * Create schedule exception from API input
     */
    createExceptionFromInput(input: ScheduleExceptionCreateInput): ScheduleException {
        const exception = this.createMockException();
        
        // Update exception based on input
        if (input.originalStartTime) {
            exception.originalStartTime = input.originalStartTime;
        }
        
        if (input.newStartTime !== undefined) {
            exception.newStartTime = input.newStartTime;
        }
        
        if (input.newEndTime !== undefined) {
            exception.newEndTime = input.newEndTime;
        }
        
        if (input.scheduleConnect) {
            exception.schedule.id = input.scheduleConnect;
        }
        
        return exception;
    }
    
    /**
     * Create different types of schedule exceptions
     */
    createVariousExceptionTypes(): ScheduleException[] {
        const baseTime = Date.now();
        const day = 24 * 60 * 60 * 1000;
        
        return [
            // Cancelled exception (null new times)
            this.createMockException({
                originalStartTime: new Date(baseTime + day).toISOString(),
                newStartTime: null,
                newEndTime: null,
            }),
            
            // Rescheduled exception (new start time)
            this.createMockException({
                originalStartTime: new Date(baseTime + 2 * day).toISOString(),
                newStartTime: new Date(baseTime + 3 * day).toISOString(),
                newEndTime: new Date(baseTime + 3 * day + 2 * 60 * 60 * 1000).toISOString(),
            }),
            
            // Time change exception (different duration)
            this.createMockException({
                originalStartTime: new Date(baseTime + 4 * day).toISOString(),
                newStartTime: new Date(baseTime + 4 * day + 60 * 60 * 1000).toISOString(),
                newEndTime: new Date(baseTime + 4 * day + 4 * 60 * 60 * 1000).toISOString(),
            }),
            
            // Next week exception
            this.createMockException({
                originalStartTime: new Date(baseTime + 7 * day).toISOString(),
                newStartTime: new Date(baseTime + 8 * day).toISOString(),
                newEndTime: null,
            }),
        ];
    }
    
    /**
     * Create exceptions for a specific schedule
     */
    createExceptionsForSchedule(scheduleId: string, count = 3): ScheduleException[] {
        const baseTime = Date.now();
        const day = 24 * 60 * 60 * 1000;
        
        return Array.from({ length: count }, (_, index) => {
            const exception = this.createMockException({
                originalStartTime: new Date(baseTime + (index + 1) * day).toISOString(),
            });
            exception.schedule.id = scheduleId;
            return exception;
        });
    }
    
    /**
     * Validate schedule exception create input
     */
    async validateCreateInput(input: ScheduleExceptionCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await scheduleExceptionValidation.create({}).validate(input);
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
 * MSW handlers factory for schedule exception endpoints
 */
export class ScheduleExceptionMSWHandlers {
    private responseFactory: ScheduleExceptionResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ScheduleExceptionResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all schedule exception endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create schedule exception
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, async ({ request }) => {
                const body = await request.json() as ScheduleExceptionCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create exception
                const exception = this.responseFactory.createExceptionFromInput(body);
                const response = this.responseFactory.createSuccessResponse(exception);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get schedule exception by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-exception/:id`, ({ params }) => {
                const { id } = params;
                
                const exception = this.responseFactory.createMockException({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(exception);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update schedule exception
            http.put(`${this.responseFactory["baseUrl"]}/api/schedule-exception/:id`, async ({ params, request }) => {
                const { id } = params;
                const body = await request.json() as ScheduleExceptionUpdateInput;
                
                const exception = this.responseFactory.createMockException({ 
                    id: id as string,
                });
                
                // Apply updates from body
                if (body.newStartTime !== undefined) {
                    exception.newStartTime = body.newStartTime;
                }
                
                if (body.newEndTime !== undefined) {
                    exception.newEndTime = body.newEndTime;
                }
                
                const response = this.responseFactory.createSuccessResponse(exception);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete schedule exception
            http.delete(`${this.responseFactory["baseUrl"]}/api/schedule-exception/:id`, () => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List schedule exceptions
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, ({ request, params }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const scheduleId = url.searchParams.get("scheduleId");
                const cancelled = url.searchParams.get("cancelled") === "true";
                
                let exceptions = scheduleId 
                    ? this.responseFactory.createExceptionsForSchedule(scheduleId)
                    : this.responseFactory.createVariousExceptionTypes();
                
                // Filter by cancelled status if specified
                if (cancelled !== null) {
                    exceptions = exceptions.filter(e => 
                        cancelled ? (e.newStartTime === null) : (e.newStartTime !== null),
                    );
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedExceptions = exceptions.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createExceptionListResponse(
                    paginatedExceptions,
                    {
                        page,
                        pageSize: limit,
                        totalCount: exceptions.length,
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, () => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        originalStartTime: "Original start time is required",
                        scheduleConnect: "Schedule is required",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-exception/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, () => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, () => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, async ({ request }) => {
                const body = await request.json() as ScheduleExceptionCreateInput;
                const exception = this.responseFactory.createExceptionFromInput(body);
                const response = this.responseFactory.createSuccessResponse(exception);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/schedule-exception`, () => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/schedule-exception/:id`, () => {
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
export const scheduleExceptionResponseScenarios = {
    // Success scenarios
    createSuccess: (exception?: ScheduleException) => {
        const factory = new ScheduleExceptionResponseFactory();
        return factory.createSuccessResponse(
            exception || factory.createMockException(),
        );
    },
    
    listSuccess: (exceptions?: ScheduleException[]) => {
        const factory = new ScheduleExceptionResponseFactory();
        return factory.createExceptionListResponse(
            exceptions || factory.createVariousExceptionTypes(),
        );
    },
    
    cancelledExceptions: () => {
        const factory = new ScheduleExceptionResponseFactory();
        const exceptions = factory.createVariousExceptionTypes().filter(e => e.newStartTime === null);
        return factory.createExceptionListResponse(exceptions);
    },
    
    rescheduledExceptions: () => {
        const factory = new ScheduleExceptionResponseFactory();
        const exceptions = factory.createVariousExceptionTypes().filter(e => e.newStartTime !== null);
        return factory.createExceptionListResponse(exceptions);
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ScheduleExceptionResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                originalStartTime: "Original start time is required",
                scheduleConnect: "Schedule is required",
            },
        );
    },
    
    notFoundError: (exceptionId?: string) => {
        const factory = new ScheduleExceptionResponseFactory();
        return factory.createNotFoundErrorResponse(
            exceptionId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ScheduleExceptionResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ScheduleExceptionResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ScheduleExceptionMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ScheduleExceptionMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ScheduleExceptionMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ScheduleExceptionMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const scheduleExceptionResponseFactory = new ScheduleExceptionResponseFactory();
export const scheduleExceptionMSWHandlers = new ScheduleExceptionMSWHandlers();
