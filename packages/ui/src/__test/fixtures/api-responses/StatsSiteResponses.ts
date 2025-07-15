/**
 * StatsSite API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for site-wide statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    StatsSite,
    StatsSiteSearchInput,
    StatPeriodType,
} from "@vrooli/shared";
import { 
    StatPeriodType as StatPeriodTypeEnum, 
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
 * StatsSite API response factory
 */
export class StatsSiteResponseFactory {
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
     * Create successful site statistics response
     */
    createSuccessResponse(stats: StatsSite): APIResponse<StatsSite> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/stats/site/${stats.id}`,
                },
            },
        };
    }
    
    /**
     * Create site statistics list response
     */
    createStatsListResponse(statsList: StatsSite[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<StatsSite> {
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
                    self: `${this.baseUrl}/api/stats/site?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/stats/site",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(period: string): APIErrorResponse {
        return {
            error: {
                code: "SITE_STATS_NOT_FOUND",
                message: `Site statistics for period '${period}' were not found`,
                details: {
                    period,
                    searchCriteria: { period },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/site/${period}`,
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
                message: `You do not have permission to ${operation} site statistics`,
                details: {
                    operation,
                    requiredPermissions: ["admin:stats:read"],
                    userPermissions: ["user:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/stats/site",
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
                path: "/api/stats/site",
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
                path: "/api/stats/site",
            },
        };
    }
    
    /**
     * Create mock site statistics
     */
    createMockStats(overrides?: Partial<StatsSite>): StatsSite {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStats = {
            __typename: "StatsSite",
            id,
            periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
            periodEnd: now,
            periodType: StatPeriodTypeEnum.Weekly,
            
            // StatsSite actual properties
            activeUsers: Math.floor(Math.random() * 5000) + 500,
            teamsCreated: Math.floor(Math.random() * 20) + 3,
            verifiedEmailsCreated: Math.floor(Math.random() * 100) + 10,
            verifiedWalletsCreated: Math.floor(Math.random() * 50) + 5,
            resourcesCreatedByType: JSON.stringify({
                "routine": Math.floor(Math.random() * 100) + 15,
                "project": Math.floor(Math.random() * 50) + 10,
                "api": Math.floor(Math.random() * 30) + 5
            }),
            resourcesCompletedByType: JSON.stringify({
                "routine": Math.floor(Math.random() * 80) + 12,
                "project": Math.floor(Math.random() * 40) + 8,
                "api": Math.floor(Math.random() * 25) + 3
            }),
            resourceCompletionTimeAverageByType: JSON.stringify({
                "routine": Math.floor(Math.random() * 3600) + 300,
                "project": Math.floor(Math.random() * 7200) + 600,
                "api": Math.floor(Math.random() * 1800) + 200
            }),
            routineComplexityAverage: Math.random() * 10 + 1,
            runCompletionTimeAverage: Math.random() * 3600 + 300,
            runContextSwitchesAverage: Math.random() * 5 + 0.5,
            runsStarted: Math.floor(Math.random() * 1000) + 200,
            runsCompleted: Math.floor(Math.random() * 800) + 150
        };
        
        return {
            ...defaultStats,
            ...overrides,
        } as unknown as StatsSite;
    }
    
    /**
     * Create site statistics for different time periods
     */
    createStatsByPeriod(): any {
        const baseStats = this.createMockStats();
        
        return {
            [StatPeriodTypeEnum.Daily]: {
                ...baseStats,
                periodType: StatPeriodTypeEnum.Daily,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                activeUsers: 80,
                runsStarted: 45,
                runsCompleted: 35,
            },
            [StatPeriodTypeEnum.Weekly]: {
                ...baseStats,
                periodType: StatPeriodTypeEnum.Weekly,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                activeUsers: 650,
                runsStarted: 320,
                runsCompleted: 280,
            },
            [StatPeriodTypeEnum.Monthly]: {
                ...baseStats,
                periodType: StatPeriodTypeEnum.Monthly,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                activeUsers: 3200,
                runsStarted: 1500,
                runsCompleted: 1200,
            },
            [StatPeriodTypeEnum.Yearly]: {
                ...baseStats,
                periodType: StatPeriodTypeEnum.Yearly,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                activeUsers: 18000,
                runsStarted: 25000,
                runsCompleted: 20000,
            },
        };
    }
    
    /**
     * Create trending time series data
     */
    createTimeSeriesStats(days = 30): StatsSite[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            // Simulate growth trend
            const growthFactor = 1 + (index / days) * 0.1;
            
            return this.createMockStats({
                periodType: StatPeriodTypeEnum.Daily,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                activeUsers: Math.floor((50 + Math.random() * 30) * growthFactor),
                runsStarted: Math.floor((30 + Math.random() * 20) * growthFactor),
                runsCompleted: Math.floor((25 + Math.random() * 15) * growthFactor),
            });
        });
    }
    
    /**
     * Create admin dashboard statistics
     */
    createAdminDashboardStats(): StatsSite {
        return this.createMockStats({
            periodType: StatPeriodTypeEnum.Daily,
            activeUsers: 3400,
            teamsCreated: 8,
            verifiedEmailsCreated: 15,
            verifiedWalletsCreated: 8,
            runsStarted: 480,
            runsCompleted: 420,
        });
    }
    
    /**
     * Create growth metrics over time
     */
    createGrowthMetrics(): {
        userGrowth: number;
        contentGrowth: number;
        engagementGrowth: number;
        performanceScore: number;
    } {
        return {
            userGrowth: 12.5, // percentage
            contentGrowth: 18.3, // percentage
            engagementGrowth: 8.7, // percentage
            performanceScore: 92.4, // percentage
        };
    }
}

/**
 * MSW handlers factory for site statistics endpoints
 */
export class StatsSiteMSWHandlers {
    private responseFactory: StatsSiteResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new StatsSiteResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all site statistics endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Get current site statistics
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/current`, ({ request, params }) => {
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType || StatPeriodTypeEnum.Weekly;
                
                const statsByPeriod = this.responseFactory.createStatsByPeriod();
                const stats = statsByPeriod[period];
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get site statistics by period
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/:period`, ({ request, params }) => {
                const { period } = params;
                
                const statsByPeriod = this.responseFactory.createStatsByPeriod();
                const stats = statsByPeriod[period as StatPeriodType] || statsByPeriod[StatPeriodTypeEnum.Weekly];
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Search site statistics
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/site/search`, async ({ request, params }) => {
                const body = await request.json() as StatsSiteSearchInput;
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                let statsList: StatsSite[] = [];
                
                if (body.periodType) {
                    const statsByPeriod = this.responseFactory.createStatsByPeriod();
                    statsList = [statsByPeriod[body.periodType]];
                } else {
                    // Return time series data
                    statsList = this.responseFactory.createTimeSeriesStats(30);
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
            
            // Get admin dashboard statistics
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/admin/dashboard`, ({ request, params }) => {
                const stats = this.responseFactory.createAdminDashboardStats();
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get growth metrics
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/growth`, ({ request, params }) => {
                const growthMetrics = this.responseFactory.createGrowthMetrics();
                
                return HttpResponse.json({
                        data: growthMetrics,
                        meta: {
                            timestamp: new Date().toISOString(),
                            requestId: this.responseFactory["generateRequestId"](),
                            version: "1.0",
                        },
                    }, { status: 200 });
            }),
            
            // Get time series data
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/timeseries`, ({ request, params }) => {
                const url = new URL(request.url);
                const days = parseInt(url.searchParams.get("days") || "30");
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/:period`, ({ request, params }) => {
                const { period } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(period as string),
                    { status: 404 }
                );
            }),
            
            // Permission error (admin only)
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/admin/dashboard`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("view admin statistics"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/site/search`, ({ request, params }) => {
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/current`, async ({ request, params }) => {
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/current`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/site/admin/dashboard`, ({ request, params }) => {
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
export const statsSiteResponseScenarios = {
    // Success scenarios
    currentStats: (period?: StatPeriodType) => {
        const factory = new StatsSiteResponseFactory();
        const statsByPeriod = factory.createStatsByPeriod();
        const stats = statsByPeriod[period || StatPeriodTypeEnum.Weekly];
        return factory.createSuccessResponse(stats);
    },
    
    adminDashboard: () => {
        const factory = new StatsSiteResponseFactory();
        return factory.createSuccessResponse(
            factory.createAdminDashboardStats(),
        );
    },
    
    growthMetrics: () => {
        const factory = new StatsSiteResponseFactory();
        return {
            data: factory.createGrowthMetrics(),
            meta: {
                timestamp: new Date().toISOString(),
                requestId: factory["generateRequestId"](),
                version: "1.0",
            },
        };
    },
    
    timeSeriesData: (days?: number) => {
        const factory = new StatsSiteResponseFactory();
        return factory.createStatsListResponse(
            factory.createTimeSeriesStats(days),
        );
    },
    
    allPeriods: () => {
        const factory = new StatsSiteResponseFactory();
        const statsByPeriod = factory.createStatsByPeriod();
        return factory.createStatsListResponse(Object.values(statsByPeriod));
    },
    
    // Error scenarios
    notFoundError: (period?: string) => {
        const factory = new StatsSiteResponseFactory();
        return factory.createNotFoundErrorResponse(
            period || "invalid-period",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsSiteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "view admin statistics",
        );
    },
    
    serverError: () => {
        const factory = new StatsSiteResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new StatsSiteMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new StatsSiteMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new StatsSiteMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new StatsSiteMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const statsSiteResponseFactory = new StatsSiteResponseFactory();
export const statsSiteMSWHandlers = new StatsSiteMSWHandlers();
