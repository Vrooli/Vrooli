/**
 * StatsResource API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import { 
    type StatsResource,
    type StatsResourceSearchInput,
    StatPeriodType,
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
 * StatsResource API response factory
 */
export class StatsResourceResponseFactory {
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
     * Generate unique stats ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful stats resource response
     */
    createSuccessResponse(stats: StatsResource): APIResponse<StatsResource> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/stats/resource/${stats.id}`,
                    related: {
                        // StatsResource doesn't have a resource property
                    },
                },
            },
        };
    }
    
    /**
     * Create stats resource list response
     */
    createStatsListResponse(statsList: StatsResource[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<StatsResource> {
        const paginationData = pagination || {
            page: 1,
            pageSize: statsList.length,
            totalCount: statsList.length,
        };
        
        return {
            data: statsList,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/stats/resource?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/stats/resource",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(resourceId: string): APIErrorResponse {
        return {
            error: {
                code: "RESOURCE_STATS_NOT_FOUND",
                message: `Resource statistics for ID '${resourceId}' were not found`,
                details: {
                    resourceId,
                    searchCriteria: { resourceId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/resource/${resourceId}`,
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
                message: `You do not have permission to ${operation} resource statistics`,
                details: {
                    operation,
                    requiredPermissions: ["stats:read"],
                    userPermissions: [],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/stats/resource",
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
                path: "/api/stats/resource",
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
                path: "/api/stats/resource",
            },
        };
    }
    
    /**
     * Create mock resource statistics
     */
    createMockStats(overrides?: Partial<StatsResource>): StatsResource {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStats: StatsResource = {
            __typename: "StatsResource",
            id,
            periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
            periodEnd: now,
            periodType: StatPeriodType.Weekly,
            references: Math.floor(Math.random() * 50) + 5,
            referencedBy: Math.floor(Math.random() * 100) + 10,
            runCompletionTimeAverage: Math.random() * 3600 + 300,
            runContextSwitchesAverage: Math.random() * 5 + 0.5,
            runsCompleted: Math.floor(Math.random() * 100) + 10,
            runsStarted: Math.floor(Math.random() * 120) + 15,
        };
        
        return {
            ...defaultStats,
            ...overrides,
        };
    }
    
    /**
     * Create trending resource statistics
     */
    createTrendingResourceStats(): StatsResource[] {
        return [
            this.createMockStats({
                references: 250,
                referencedBy: 500,
                runsCompleted: 150,
                runsStarted: 180,
                periodType: StatPeriodType.Monthly,
            }),
            this.createMockStats({
                references: 180,
                referencedBy: 350,
                runsCompleted: 120,
                runsStarted: 140,
                periodType: StatPeriodType.Monthly,
            }),
            this.createMockStats({
                references: 120,
                referencedBy: 200,
                runsCompleted: 80,
                runsStarted: 95,
                periodType: StatPeriodType.Monthly,
            }),
        ];
    }
    
    /**
     * Create statistics for different time periods
     */
    createStatsByPeriod(resourceId: string): Partial<Record<StatPeriodType, StatsResource>> {
        const baseStats = this.createMockStats();
        
        return {
            [StatPeriodType.Daily]: {
                ...baseStats,
                periodType: StatPeriodType.Daily,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                references: 4,
                referencedBy: 8,
                runsCompleted: 3,
                runsStarted: 5,
            },
            [StatPeriodType.Weekly]: {
                ...baseStats,
                periodType: StatPeriodType.Weekly,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                references: 32,
                referencedBy: 65,
                runsCompleted: 25,
                runsStarted: 30,
            },
            [StatPeriodType.Monthly]: {
                ...baseStats,
                periodType: StatPeriodType.Monthly,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                references: 150,
                referencedBy: 280,
                runsCompleted: 110,
                runsStarted: 125,
            },
            [StatPeriodType.Yearly]: {
                ...baseStats,
                periodType: StatPeriodType.Yearly,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                references: 1800,
                referencedBy: 3500,
                runsCompleted: 1200,
                runsStarted: 1350,
            },
            // Note: AllTime doesn't exist in StatPeriodType
        };
    }
    
    /**
     * Create time series data for resource analytics
     */
    createTimeSeriesStats(resourceId: string, days = 30): StatsResource[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            return this.createMockStats({
                periodType: StatPeriodType.Daily,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                references: Math.floor(Math.random() * 10) + 1,
                referencedBy: Math.floor(Math.random() * 20) + 2,
                runsCompleted: Math.floor(Math.random() * 10),
                runsStarted: Math.floor(Math.random() * 15) + 2,
            });
        });
    }
    
    /**
     * Create comparative statistics for multiple resources
     */
    createComparativeStats(resourceIds: string[]): StatsResource[] {
        return resourceIds.map(resourceId => 
            this.createMockStats({
                periodType: StatPeriodType.Monthly,
                references: Math.floor(Math.random() * 200) + 10,
                referencedBy: Math.floor(Math.random() * 400) + 20,
                runsCompleted: Math.floor(Math.random() * 150) + 10,
                runsStarted: Math.floor(Math.random() * 180) + 15,
            }),
        );
    }
}

/**
 * MSW handlers factory for resource statistics endpoints
 */
export class StatsResourceMSWHandlers {
    private responseFactory: StatsResourceResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new StatsResourceResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all resource statistics endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Get resource statistics by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/:id`, ({ request, params }) => {
                const { id } = params;
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType;
                
                if (period) {
                    const statsByPeriod = this.responseFactory.createStatsByPeriod(id as string);
                    const stats = statsByPeriod[period] || statsByPeriod[StatPeriodType.Monthly];
                    const response = this.responseFactory.createSuccessResponse(stats);
                    
                    return HttpResponse.json(response, { status: 200 });
                }
                
                const stats = this.responseFactory.createMockStats();
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Search resource statistics
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/resource/search`, async ({ request, params }) => {
                const body = await request.json() as StatsResourceSearchInput;
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                let statsList: StatsResource[] = [];
                
                if (body.ids && body.ids.length > 0) {
                    statsList = this.responseFactory.createComparativeStats(body.ids);
                } else {
                    statsList = this.responseFactory.createTrendingResourceStats();
                }
                
                // Filter by period if specified
                if (body.periodType) {
                    statsList = statsList.map(stats => ({
                        ...stats,
                        periodType: body.periodType!,
                    }));
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedStats = statsList.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createStatsListResponse(
                    paginatedStats,
                    {
                        page,
                        pageSize: limit,
                        totalCount: statsList.length,
                    },
                );
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get trending resources
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/trending`, ({ request, params }) => {
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType || StatPeriodType.Monthly;
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                const trendingStats = this.responseFactory.createTrendingResourceStats()
                    .map(stats => ({ ...stats, periodType: period }))
                    .slice(0, limit);
                
                const response = this.responseFactory.createStatsListResponse(trendingStats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get time series data for resource
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/:id/timeseries`, ({ request, params }) => {
                const { id } = params;
                const url = new URL(request.url);
                const days = parseInt(url.searchParams.get("days") || "30");
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(id as string, days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get aggregated statistics for multiple resources
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/resource/aggregate`, async ({ request, params }) => {
                const { resourceIds, periodType } = await request.json() as {
                    resourceIds: string[];
                    periodType: StatPeriodType;
                };
                
                const aggregatedStats = this.responseFactory.createComparativeStats(resourceIds)
                    .map(stats => ({ ...stats, periodType }));
                
                const response = this.responseFactory.createStatsListResponse(aggregatedStats);
                
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/:id`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("view"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/resource/search`, ({ request, params }) => {
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/:id`, async ({ request, params }) => {
                const { id } = params;
                const stats = this.responseFactory.createMockStats();
                const response = this.responseFactory.createSuccessResponse(stats);
                
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/resource/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/resource/search`, ({ request, params }) => {
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
export const statsResourceResponseScenarios = {
    // Success scenarios
    getSuccess: (stats?: StatsResource) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createSuccessResponse(
            stats || factory.createMockStats(),
        );
    },
    
    searchSuccess: (statsList?: StatsResource[]) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createStatsListResponse(
            statsList || factory.createTrendingResourceStats(),
        );
    },
    
    trendingResources: (period?: StatPeriodType) => {
        const factory = new StatsResourceResponseFactory();
        const trending = factory.createTrendingResourceStats();
        if (period) {
            trending.forEach(stats => stats.periodType = period);
        }
        return factory.createStatsListResponse(trending);
    },
    
    timeSeriesData: (resourceId?: string, days?: number) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createStatsListResponse(
            factory.createTimeSeriesStats(resourceId || "resource-123", days),
        );
    },
    
    comparativeStats: (resourceIds?: string[]) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createStatsListResponse(
            factory.createComparativeStats(resourceIds || ["resource-1", "resource-2", "resource-3"]),
        );
    },
    
    // Error scenarios
    notFoundError: (resourceId?: string) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "view",
        );
    },
    
    serverError: () => {
        const factory = new StatsResourceResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new StatsResourceMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new StatsResourceMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new StatsResourceMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new StatsResourceMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const statsResourceResponseFactory = new StatsResourceResponseFactory();
export const statsResourceMSWHandlers = new StatsResourceMSWHandlers();
