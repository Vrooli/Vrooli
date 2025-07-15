/**
 * StatsUser API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for user statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import { 
    type StatsUser,
    type StatsUserSearchInput,
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
 * StatsUser API response factory
 */
export class StatsUserResponseFactory {
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
     * Create successful user statistics response
     */
    createSuccessResponse(stats: StatsUser): APIResponse<StatsUser> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/stats/user/${stats.id}`,
                    related: {
                        // StatsUser doesn't have a user property
                    },
                },
            },
        };
    }
    
    /**
     * Create user statistics list response
     */
    createStatsListResponse(statsList: StatsUser[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<StatsUser> {
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
                    self: `${this.baseUrl}/api/stats/user?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/stats/user",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(userId: string): APIErrorResponse {
        return {
            error: {
                code: "USER_STATS_NOT_FOUND",
                message: `User statistics for ID '${userId}' were not found`,
                details: {
                    userId,
                    searchCriteria: { userId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/user/${userId}`,
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
                message: `You do not have permission to ${operation} user statistics`,
                details: {
                    operation,
                    requiredPermissions: ["user:stats:read"],
                    userPermissions: ["user:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/stats/user",
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
                path: "/api/stats/user",
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
                path: "/api/stats/user",
            },
        };
    }
    
    /**
     * Create mock user statistics
     */
    createMockStats(overrides?: Partial<StatsUser>): StatsUser {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStats: StatsUser = {
            __typename: "StatsUser",
            id,
            periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
            periodEnd: now,
            periodType: StatPeriodType.Weekly,
            
            // StatsUser specific properties
            resourcesCreatedByType: JSON.stringify({
                "routine": Math.floor(Math.random() * 25) + 5,
                "project": Math.floor(Math.random() * 15) + 2,
                "api": Math.floor(Math.random() * 10) + 1
            }),
            resourcesCompletedByType: JSON.stringify({
                "routine": Math.floor(Math.random() * 12) + 2,
                "project": Math.floor(Math.random() * 3),
                "api": Math.floor(Math.random() * 2)
            }),
            resourceCompletionTimeAverageByType: JSON.stringify({
                "routine": Math.floor(Math.random() * 3600) + 300,
                "project": Math.floor(Math.random() * 7200) + 600,
                "api": Math.floor(Math.random() * 1800) + 200
            }),
            runsStarted: Math.floor(Math.random() * 20) + 5,
            runsCompleted: Math.floor(Math.random() * 18) + 4,
            runCompletionTimeAverage: Math.random() * 3600 + 300,
            runContextSwitchesAverage: Math.random() * 5 + 0.5,
            teamsCreated: Math.floor(Math.random() * 3),
        };
        
        return {
            ...defaultStats,
            ...overrides,
        };
    }
    
    /**
     * Create high-performing user statistics
     */
    createHighPerformingUserStats(): StatsUser[] {
        return [
            this.createMockStats({
                resourcesCreatedByType: JSON.stringify({
                    "routine": 25,
                    "project": 12,
                    "api": 8
                }),
                resourcesCompletedByType: JSON.stringify({
                    "routine": 80,
                    "project": 10,
                    "api": 5
                }),
                runsCompleted: 200,
                runsStarted: 220,
                runCompletionTimeAverage: 1200,
                runContextSwitchesAverage: 1.5,
                teamsCreated: 5,
                periodType: StatPeriodType.Monthly,
            }),
            this.createMockStats({
                resourcesCreatedByType: JSON.stringify({
                    "routine": 18,
                    "project": 8,
                    "api": 5
                }),
                resourcesCompletedByType: JSON.stringify({
                    "routine": 65,
                    "project": 7,
                    "api": 3
                }),
                runsCompleted: 150,
                runsStarted: 165,
                runCompletionTimeAverage: 1400,
                runContextSwitchesAverage: 2.1,
                teamsCreated: 3,
                periodType: StatPeriodType.Monthly,
            }),
            this.createMockStats({
                resourcesCreatedByType: JSON.stringify({
                    "routine": 12,
                    "project": 6,
                    "api": 3
                }),
                resourcesCompletedByType: JSON.stringify({
                    "routine": 45,
                    "project": 5,
                    "api": 2
                }),
                runsCompleted: 120,
                runsStarted: 130,
                runCompletionTimeAverage: 1600,
                runContextSwitchesAverage: 2.5,
                teamsCreated: 2,
                periodType: StatPeriodType.Monthly,
            }),
        ];
    }
    
    /**
     * Create statistics for different time periods
     */
    createStatsByPeriod(userId: string): Partial<Record<StatPeriodType, StatsUser>> {
        const baseStats = this.createMockStats();
        
        return {
            [StatPeriodType.Daily]: {
                ...baseStats,
                periodType: StatPeriodType.Daily,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 3,
                runsCompleted: 2,
                resourcesCreatedByType: JSON.stringify({"routine": 1}),
                resourcesCompletedByType: JSON.stringify({"routine": 2}),
                resourceCompletionTimeAverageByType: JSON.stringify({"routine": 300}),
                runCompletionTimeAverage: 250.5,
                runContextSwitchesAverage: 1.2,
                teamsCreated: 0,
            },
            [StatPeriodType.Weekly]: {
                ...baseStats,
                periodType: StatPeriodType.Weekly,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 20,
                runsCompleted: 16,
                resourcesCreatedByType: JSON.stringify({"routine": 2, "project": 1}),
                resourcesCompletedByType: JSON.stringify({"routine": 12}),
                resourceCompletionTimeAverageByType: JSON.stringify({"routine": 450, "project": 1200}),
                runCompletionTimeAverage: 680.3,
                runContextSwitchesAverage: 2.1,
                teamsCreated: 1,
            },
            [StatPeriodType.Monthly]: {
                ...baseStats,
                periodType: StatPeriodType.Monthly,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 85,
                runsCompleted: 70,
                resourcesCreatedByType: JSON.stringify({"routine": 8, "project": 3, "api": 1}),
                resourcesCompletedByType: JSON.stringify({"routine": 45, "project": 2}),
                resourceCompletionTimeAverageByType: JSON.stringify({"routine": 520, "project": 1800, "api": 350}),
                runCompletionTimeAverage: 890.7,
                runContextSwitchesAverage: 2.8,
                teamsCreated: 2,
            },
            [StatPeriodType.Yearly]: {
                ...baseStats,
                periodType: StatPeriodType.Yearly,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 1000,
                runsCompleted: 850,
                resourcesCreatedByType: JSON.stringify({"routine": 95, "project": 35, "api": 12}),
                resourcesCompletedByType: JSON.stringify({"routine": 500, "project": 28}),
                resourceCompletionTimeAverageByType: JSON.stringify({"routine": 480, "project": 2100, "api": 420}),
                runCompletionTimeAverage: 720.5,
                runContextSwitchesAverage: 3.2,
                teamsCreated: 5,
            },
            // Note: AllTime doesn't exist in StatPeriodType
        };
    }
    
    /**
     * Create time series data for user analytics
     */
    createTimeSeriesStats(userId: string, days = 30): StatsUser[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            return this.createMockStats({
                // StatsUser doesn't have a user property
                periodType: StatPeriodType.Daily,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                runsStarted: Math.floor(Math.random() * 8) + 1,
                runsCompleted: Math.floor(Math.random() * 6) + 1,
                resourcesCompletedByType: JSON.stringify({"routine": Math.floor(Math.random() * 4)}),
            });
        });
    }
    
    /**
     * Create comparative statistics for multiple users
     */
    createComparativeStats(userIds: string[]): StatsUser[] {
        return userIds.map(userId => 
            this.createMockStats({
                id: `stats_${userId}`,
                periodType: StatPeriodType.Monthly,
                runsCompleted: Math.floor(Math.random() * 100) + 20,
                resourcesCreatedByType: JSON.stringify({"routine": Math.floor(Math.random() * 15) + 2}),
                resourcesCompletedByType: JSON.stringify({"project": Math.floor(Math.random() * 8) + 1}),
            }),
        );
    }
    
    /**
     * Create personal dashboard statistics
     */
    createPersonalDashboardStats(userId: string): StatsUser {
        return this.createMockStats({
            // StatsUser doesn't have a user property
            periodType: StatPeriodType.Monthly,
            resourcesCreatedByType: JSON.stringify({"routine": 12, "project": 6, "api": 3}),
            resourcesCompletedByType: JSON.stringify({"routine": 38, "project": 4}),
            runsStarted: 65,
            runsCompleted: 58,
        });
    }
}

/**
 * MSW handlers factory for user statistics endpoints
 */
export class StatsUserMSWHandlers {
    private responseFactory: StatsUserResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new StatsUserResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all user statistics endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Get user statistics by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/:id`, ({ request, params }) => {
                const { id } = params;
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType;
                
                if (period) {
                    const statsByPeriod = this.responseFactory.createStatsByPeriod(id as string);
                    const stats = statsByPeriod[period] || statsByPeriod[StatPeriodType.Monthly];
                    const response = this.responseFactory.createSuccessResponse(stats);
                    
                    return HttpResponse.json(response, { status: 200 });
                }
                
                const stats = this.responseFactory.createMockStats({
                    id: `stats_${id}`,
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get current user's statistics
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/me`, ({ request, params }) => {
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType || StatPeriodType.Monthly;
                
                const stats = this.responseFactory.createPersonalDashboardStats("current-user");
                stats.periodType = period;
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Search user statistics
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/user/search`, async ({ request, params }) => {
                const body = await request.json() as StatsUserSearchInput;
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                let statsList: StatsUser[] = [];
                
                if (body.ids && body.ids.length > 0) {
                    statsList = this.responseFactory.createComparativeStats(body.ids);
                } else {
                    statsList = this.responseFactory.createHighPerformingUserStats();
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
            
            // Get top performing users
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/top-performers`, ({ request, params }) => {
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType || StatPeriodType.Monthly;
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                const topUsers = this.responseFactory.createHighPerformingUserStats()
                    .map(stats => ({ ...stats, periodType: period }))
                    .slice(0, limit);
                
                const response = this.responseFactory.createStatsListResponse(topUsers);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get time series data for user
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/:id/timeseries`, ({ request, params }) => {
                const { id } = params;
                const url = new URL(request.url);
                const days = parseInt(url.searchParams.get("days") || "30");
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(id as string, days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get aggregated statistics for multiple users
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/user/aggregate`, async ({ request, params }) => {
                const { userIds, periodType } = await request.json() as {
                    userIds: string[];
                    periodType: StatPeriodType;
                };
                
                const aggregatedStats = this.responseFactory.createComparativeStats(userIds)
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/:id`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("view"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/user/search`, ({ request, params }) => {
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/:id`, async ({ request, params }) => {
                const { id } = params;
                const stats = this.responseFactory.createMockStats({
                    id: `stats_${id}`,
                });
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/user/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/user/search`, ({ request, params }) => {
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
export const statsUserResponseScenarios = {
    // Success scenarios
    getSuccess: (stats?: StatsUser) => {
        const factory = new StatsUserResponseFactory();
        return factory.createSuccessResponse(
            stats || factory.createMockStats(),
        );
    },
    
    personalDashboard: (userId?: string) => {
        const factory = new StatsUserResponseFactory();
        return factory.createSuccessResponse(
            factory.createPersonalDashboardStats(userId || "current-user"),
        );
    },
    
    searchSuccess: (statsList?: StatsUser[]) => {
        const factory = new StatsUserResponseFactory();
        return factory.createStatsListResponse(
            statsList || factory.createHighPerformingUserStats(),
        );
    },
    
    topPerformers: (period?: StatPeriodType) => {
        const factory = new StatsUserResponseFactory();
        const topUsers = factory.createHighPerformingUserStats();
        if (period) {
            topUsers.forEach(stats => stats.periodType = period);
        }
        return factory.createStatsListResponse(topUsers);
    },
    
    timeSeriesData: (userId?: string, days?: number) => {
        const factory = new StatsUserResponseFactory();
        return factory.createStatsListResponse(
            factory.createTimeSeriesStats(userId || "user-123", days),
        );
    },
    
    comparativeStats: (userIds?: string[]) => {
        const factory = new StatsUserResponseFactory();
        return factory.createStatsListResponse(
            factory.createComparativeStats(userIds || ["user-1", "user-2", "user-3"]),
        );
    },
    
    // Error scenarios
    notFoundError: (userId?: string) => {
        const factory = new StatsUserResponseFactory();
        return factory.createNotFoundErrorResponse(
            userId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsUserResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "view",
        );
    },
    
    serverError: () => {
        const factory = new StatsUserResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new StatsUserMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new StatsUserMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new StatsUserMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new StatsUserMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const statsUserResponseFactory = new StatsUserResponseFactory();
export const statsUserMSWHandlers = new StatsUserMSWHandlers();
