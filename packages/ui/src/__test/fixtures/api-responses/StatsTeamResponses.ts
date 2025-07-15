/**
 * StatsTeam API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for team statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import { 
    type StatsTeam,
    type StatsTeamSearchInput,
    type StatPeriodType,
    StatPeriodType as PeriodType,
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
 * StatsTeam API response factory
 */
export class StatsTeamResponseFactory {
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
     * Create successful team statistics response
     */
    createSuccessResponse(stats: StatsTeam): APIResponse<StatsTeam> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/stats/team/${stats.id}`,
                    related: {
                        // team: `${this.baseUrl}/api/team/${stats.team?.id}`, // StatsTeam doesn't have team property
                    },
                },
            },
        };
    }
    
    /**
     * Create team statistics list response
     */
    createStatsListResponse(statsList: StatsTeam[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<StatsTeam> {
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
                    self: `${this.baseUrl}/api/stats/team?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/stats/team",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(teamId: string): APIErrorResponse {
        return {
            error: {
                code: "TEAM_STATS_NOT_FOUND",
                message: `Team statistics for ID '${teamId}' were not found`,
                details: {
                    teamId,
                    searchCriteria: { teamId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/team/${teamId}`,
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
                message: `You do not have permission to ${operation} team statistics`,
                details: {
                    operation,
                    requiredPermissions: ["team:stats:read"],
                    userPermissions: ["team:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/stats/team",
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
                path: "/api/stats/team",
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
                path: "/api/stats/team",
            },
        };
    }
    
    /**
     * Create mock team statistics
     */
    createMockStats(overrides?: Partial<StatsTeam>): StatsTeam {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStats: StatsTeam = {
            __typename: "StatsTeam",
            id,
            periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
            periodEnd: now,
            periodType: PeriodType.Weekly,
            
            // StatsTeam only has these properties
            members: Math.floor(Math.random() * 50) + 5,
            projects: Math.floor(Math.random() * 30) + 5,
            resources: Math.floor(Math.random() * 50) + 10,
            runsStarted: Math.floor(Math.random() * 40) + 10,
            runsCompleted: Math.floor(Math.random() * 35) + 8,
            runCompletionTimeAverage: Math.random() * 3600 + 300, // Average completion time in seconds
            runContextSwitchesAverage: Math.random() * 5 + 0.5, // Average context switches per run
        };
        
        return {
            ...defaultStats,
            ...overrides,
        };
    }
    
    /**
     * Create high-performing team statistics
     */
    createHighPerformingTeamStats(): StatsTeam[] {
        return [
            this.createMockStats({
                members: 25,
                projects: 15,
                resources: 45,
                runsStarted: 150,
                runsCompleted: 120,
                runCompletionTimeAverage: 450,
                runContextSwitchesAverage: 0.8,
                periodType: PeriodType.Monthly,
            }),
            this.createMockStats({
                members: 18,
                projects: 12,
                resources: 38,
                runsStarted: 110,
                runsCompleted: 95,
                runCompletionTimeAverage: 520,
                runContextSwitchesAverage: 1.2,
                periodType: PeriodType.Monthly,
            }),
            this.createMockStats({
                members: 12,
                projects: 8,
                resources: 25,
                runsStarted: 85,
                runsCompleted: 75,
                runCompletionTimeAverage: 600,
                runContextSwitchesAverage: 1.5,
                periodType: PeriodType.Monthly,
            }),
        ];
    }
    
    /**
     * Create statistics for different time periods
     */
    createStatsByPeriod(teamId: string): Record<StatPeriodType, StatsTeam> {
        const baseStats = this.createMockStats();
        
        return {
            [PeriodType.Hourly]: {
                ...baseStats,
                periodType: PeriodType.Hourly,
                periodStart: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
                members: 8,
                runsStarted: 2,
                runsCompleted: 1,
                runCompletionTimeAverage: 300,
                runContextSwitchesAverage: 0.5,
            },
            [PeriodType.Daily]: {
                ...baseStats,
                periodType: PeriodType.Daily,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                members: 12,
                runsStarted: 5,
                runsCompleted: 4,
                runCompletionTimeAverage: 450,
                runContextSwitchesAverage: 1.2,
            },
            [PeriodType.Weekly]: {
                ...baseStats,
                periodType: PeriodType.Weekly,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                members: 15,
                runsStarted: 35,
                runsCompleted: 28,
                runCompletionTimeAverage: 600,
                runContextSwitchesAverage: 1.5,
                projects: 8,
                resources: 22,
            },
            [PeriodType.Monthly]: {
                ...baseStats,
                periodType: PeriodType.Monthly,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                members: 20,
                runsStarted: 150,
                runsCompleted: 120,
                runCompletionTimeAverage: 780,
                runContextSwitchesAverage: 2.1,
                projects: 15,
                resources: 45,
            },
            [PeriodType.Yearly]: {
                ...baseStats,
                periodType: PeriodType.Yearly,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                members: 25,
                runsStarted: 1800,
                runsCompleted: 1500,
                runCompletionTimeAverage: 920,
                runContextSwitchesAverage: 2.8,
                projects: 60,
                resources: 180,
            },
        };
    }
    
    /**
     * Create time series data for team analytics
     */
    createTimeSeriesStats(teamId: string, days = 30): StatsTeam[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            return this.createMockStats({
                periodType: PeriodType.Daily,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                members: Math.floor(Math.random() * 15) + 3,
                projects: Math.floor(Math.random() * 5) + 1,
                resources: Math.floor(Math.random() * 10) + 2,
                runsStarted: Math.floor(Math.random() * 10) + 2,
                runsCompleted: Math.floor(Math.random() * 8) + 1,
                runCompletionTimeAverage: Math.random() * 1200 + 300,
                runContextSwitchesAverage: Math.random() * 3 + 0.5,
            });
        });
    }
    
    /**
     * Create comparative statistics for multiple teams
     */
    createComparativeStats(teamIds: string[]): StatsTeam[] {
        return teamIds.map(teamId => 
            this.createMockStats({
                periodType: PeriodType.Monthly,
                members: Math.floor(Math.random() * 30) + 5,
                projects: Math.floor(Math.random() * 15) + 2,
                resources: Math.floor(Math.random() * 50) + 10,
                runsStarted: Math.floor(Math.random() * 120) + 25,
                runsCompleted: Math.floor(Math.random() * 100) + 20,
                runCompletionTimeAverage: Math.random() * 2400 + 600,
                runContextSwitchesAverage: Math.random() * 4 + 1,
            }),
        );
    }
}

/**
 * MSW handlers factory for team statistics endpoints
 */
export class StatsTeamMSWHandlers {
    private responseFactory: StatsTeamResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new StatsTeamResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all team statistics endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Get team statistics by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, ({ request, params }) => {
                const { id } = params;
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType;
                
                if (period) {
                    const statsByPeriod = this.responseFactory.createStatsByPeriod(id as string);
                    const stats = statsByPeriod[period] || statsByPeriod[PeriodType.Monthly];
                    const response = this.responseFactory.createSuccessResponse(stats);
                    
                    return HttpResponse.json(response, { status: 200 });
                }
                
                const stats = this.responseFactory.createMockStats({
                    id: id as string,
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Search team statistics
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/team/search`, async ({ request, params }) => {
                const body = await request.json() as StatsTeamSearchInput;
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                let statsList: StatsTeam[] = [];
                
                // Always return high performing team stats for search
                statsList = this.responseFactory.createHighPerformingTeamStats();
                
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
            
            // Get top performing teams
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/top-performers`, ({ request, params }) => {
                const url = new URL(request.url);
                const period = url.searchParams.get("period") as StatPeriodType || PeriodType.Monthly;
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                const topTeams = this.responseFactory.createHighPerformingTeamStats()
                    .map(stats => ({ ...stats, periodType: period }))
                    .slice(0, limit);
                
                const response = this.responseFactory.createStatsListResponse(topTeams);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get time series data for team
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id/timeseries`, ({ request, params }) => {
                const { id } = params;
                const url = new URL(request.url);
                const days = parseInt(url.searchParams.get("days") || "30");
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(id as string, days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Get aggregated statistics for multiple teams
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/team/aggregate`, async ({ request, params }) => {
                const { teamIds, periodType } = await request.json() as {
                    teamIds: string[];
                    periodType: StatPeriodType;
                };
                
                const aggregatedStats = this.responseFactory.createComparativeStats(teamIds)
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("view"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/team/search`, ({ request, params }) => {
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, async ({ request, params }) => {
                const { id } = params;
                const stats = this.responseFactory.createMockStats({
                    id: id as string,
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
            http.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/stats/team/search`, ({ request, params }) => {
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
export const statsTeamResponseScenarios = {
    // Success scenarios
    getSuccess: (stats?: StatsTeam) => {
        const factory = new StatsTeamResponseFactory();
        return factory.createSuccessResponse(
            stats || factory.createMockStats(),
        );
    },
    
    searchSuccess: (statsList?: StatsTeam[]) => {
        const factory = new StatsTeamResponseFactory();
        return factory.createStatsListResponse(
            statsList || factory.createHighPerformingTeamStats(),
        );
    },
    
    topPerformers: (period?: StatPeriodType) => {
        const factory = new StatsTeamResponseFactory();
        const topTeams = factory.createHighPerformingTeamStats();
        if (period) {
            topTeams.forEach(stats => stats.periodType = period);
        }
        return factory.createStatsListResponse(topTeams);
    },
    
    timeSeriesData: (teamId?: string, days?: number) => {
        const factory = new StatsTeamResponseFactory();
        return factory.createStatsListResponse(
            factory.createTimeSeriesStats(teamId || "team-123", days),
        );
    },
    
    comparativeStats: (teamIds?: string[]) => {
        const factory = new StatsTeamResponseFactory();
        return factory.createStatsListResponse(
            factory.createComparativeStats(teamIds || ["team-1", "team-2", "team-3"]),
        );
    },
    
    // Error scenarios
    notFoundError: (teamId?: string) => {
        const factory = new StatsTeamResponseFactory();
        return factory.createNotFoundErrorResponse(
            teamId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsTeamResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "view",
        );
    },
    
    serverError: () => {
        const factory = new StatsTeamResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new StatsTeamMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new StatsTeamMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new StatsTeamMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new StatsTeamMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const statsTeamResponseFactory = new StatsTeamResponseFactory();
export const statsTeamMSWHandlers = new StatsTeamMSWHandlers();
