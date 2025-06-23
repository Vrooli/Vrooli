/**
 * StatsTeam API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for team statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from "msw";
import type { 
    StatsTeam,
    StatsTeamSearchInput,
    PeriodType,
} from "@vrooli/shared";
import { 
    PeriodType as PeriodTypeEnum, 
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
        return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Generate unique stats ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
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
                        team: `${this.baseUrl}/api/team/${stats.team?.id}`,
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
            periodType: PeriodTypeEnum.Week,
            
            // Member statistics
            members: Math.floor(Math.random() * 50) + 5,
            membersLoggedIn: Math.floor(Math.random() * 30) + 3,
            membersJoined: Math.floor(Math.random() * 5) + 1,
            membersLeft: Math.floor(Math.random() * 2),
            
            // Content statistics
            apis: Math.floor(Math.random() * 20) + 2,
            apisCreated: Math.floor(Math.random() * 3),
            projects: Math.floor(Math.random() * 30) + 5,
            projectsCreated: Math.floor(Math.random() * 5) + 1,
            projectsCompleted: Math.floor(Math.random() * 3),
            routines: Math.floor(Math.random() * 50) + 10,
            routinesCreated: Math.floor(Math.random() * 8) + 1,
            routinesCompleted: Math.floor(Math.random() * 15) + 2,
            
            // Activity statistics
            runs: Math.floor(Math.random() * 200) + 50,
            runsStarted: Math.floor(Math.random() * 40) + 10,
            runsCompleted: Math.floor(Math.random() * 35) + 8,
            
            // Engagement statistics
            views: Math.floor(Math.random() * 1000) + 100,
            bookmarks: Math.floor(Math.random() * 50) + 5,
            
            team: {
                __typename: "Team",
                id: `team_${id}`,
                created_at: now,
                updated_at: now,
                handle: `team_${id}`,
                name: "Test Team",
                isOpenToNewMembers: true,
                isPrivate: false,
                bookmarks: Math.floor(Math.random() * 50) + 5,
                views: Math.floor(Math.random() * 500) + 50,
                you: {
                    __typename: "TeamYou",
                    canAddMembers: true,
                    canBookmark: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: true,
                    canRead: true,
                    isBookmarked: false,
                    isViewed: true,
                    yourMembership: {
                        __typename: "Member",
                        id: `member_${id}`,
                        created_at: now,
                        updated_at: now,
                        isAdmin: true,
                        permissions: JSON.stringify(["read", "write", "admin"]),
                        role: {
                            __typename: "Role",
                            id: `role_${id}`,
                            created_at: now,
                            updated_at: now,
                            name: "Admin",
                            permissions: JSON.stringify(["read", "write", "admin"]),
                            translations: [],
                        },
                        you: {
                            __typename: "MemberYou",
                            canUpdate: true,
                            canDelete: false,
                        },
                    },
                },
            },
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
                membersLoggedIn: 22,
                routinesCreated: 15,
                routinesCompleted: 45,
                projectsCreated: 8,
                projectsCompleted: 6,
                runsCompleted: 120,
                views: 2500,
                bookmarks: 150,
                periodType: PeriodTypeEnum.Month,
            }),
            this.createMockStats({
                members: 18,
                membersLoggedIn: 16,
                routinesCreated: 12,
                routinesCompleted: 38,
                projectsCreated: 6,
                projectsCompleted: 5,
                runsCompleted: 95,
                views: 1800,
                bookmarks: 120,
                periodType: PeriodTypeEnum.Month,
            }),
            this.createMockStats({
                members: 12,
                membersLoggedIn: 11,
                routinesCreated: 8,
                routinesCompleted: 25,
                projectsCreated: 4,
                projectsCompleted: 3,
                runsCompleted: 75,
                views: 1200,
                bookmarks: 90,
                periodType: PeriodTypeEnum.Month,
            }),
        ];
    }
    
    /**
     * Create statistics for different time periods
     */
    createStatsByPeriod(teamId: string): Record<PeriodType, StatsTeam> {
        const baseStats = this.createMockStats({
            team: {
                ...this.createMockStats().team,
                id: teamId,
            },
        });
        
        return {
            [PeriodTypeEnum.Day]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                membersLoggedIn: 8,
                runsStarted: 5,
                runsCompleted: 4,
                views: 25,
                bookmarks: 2,
            },
            [PeriodTypeEnum.Week]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Week,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                membersLoggedIn: 15,
                runsStarted: 35,
                runsCompleted: 28,
                views: 180,
                bookmarks: 12,
                routinesCreated: 3,
                projectsCreated: 1,
            },
            [PeriodTypeEnum.Month]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Month,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                membersLoggedIn: 20,
                membersJoined: 3,
                membersLeft: 1,
                runsStarted: 150,
                runsCompleted: 120,
                views: 800,
                bookmarks: 45,
                routinesCreated: 12,
                routinesCompleted: 35,
                projectsCreated: 5,
                projectsCompleted: 3,
            },
            [PeriodTypeEnum.Year]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Year,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                members: 25,
                membersJoined: 18,
                membersLeft: 5,
                runsStarted: 1800,
                runsCompleted: 1500,
                views: 12000,
                bookmarks: 600,
                routinesCreated: 150,
                routinesCompleted: 400,
                projectsCreated: 60,
                projectsCompleted: 45,
            },
            [PeriodTypeEnum.AllTime]: {
                ...baseStats,
                periodType: PeriodTypeEnum.AllTime,
                periodStart: new Date(Date.now() - 2 * 365 * 24 * 60 * 60 * 1000).toISOString(),
                members: 25,
                membersJoined: 25,
                membersLeft: 8,
                runsStarted: 3500,
                runsCompleted: 3000,
                views: 25000,
                bookmarks: 1200,
                routinesCreated: 300,
                routinesCompleted: 800,
                projectsCreated: 120,
                projectsCompleted: 95,
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
                team: {
                    ...this.createMockStats().team,
                    id: teamId,
                },
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                membersLoggedIn: Math.floor(Math.random() * 15) + 3,
                runsStarted: Math.floor(Math.random() * 10) + 2,
                runsCompleted: Math.floor(Math.random() * 8) + 1,
                views: Math.floor(Math.random() * 50) + 10,
                bookmarks: Math.floor(Math.random() * 5),
            });
        });
    }
    
    /**
     * Create comparative statistics for multiple teams
     */
    createComparativeStats(teamIds: string[]): StatsTeam[] {
        return teamIds.map(teamId => 
            this.createMockStats({
                team: {
                    ...this.createMockStats().team,
                    id: teamId,
                    name: `Team ${teamId}`,
                },
                periodType: PeriodTypeEnum.Month,
                members: Math.floor(Math.random() * 30) + 5,
                runsCompleted: Math.floor(Math.random() * 100) + 20,
                routinesCreated: Math.floor(Math.random() * 15) + 2,
                projectsCompleted: Math.floor(Math.random() * 8) + 1,
                views: Math.floor(Math.random() * 1000) + 100,
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get team statistics by ID
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, (req, res, ctx) => {
                const { id } = req.params;
                const url = new URL(req.url);
                const period = url.searchParams.get("period") as PeriodType;
                
                if (period) {
                    const statsByPeriod = this.responseFactory.createStatsByPeriod(id as string);
                    const stats = statsByPeriod[period] || statsByPeriod[PeriodTypeEnum.Month];
                    const response = this.responseFactory.createSuccessResponse(stats);
                    
                    return res(
                        ctx.status(200),
                        ctx.json(response),
                    );
                }
                
                const stats = this.responseFactory.createMockStats({
                    team: {
                        ...this.responseFactory.createMockStats().team,
                        id: id as string,
                    },
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Search team statistics
            rest.post(`${this.responseFactory["baseUrl"]}/api/stats/team/search`, async (req, res, ctx) => {
                const body = await req.json() as StatsTeamSearchInput;
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                let statsList: StatsTeam[] = [];
                
                if (body.teamIds && body.teamIds.length > 0) {
                    statsList = this.responseFactory.createComparativeStats(body.teamIds);
                } else {
                    statsList = this.responseFactory.createHighPerformingTeamStats();
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
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get top performing teams
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/top-performers`, (req, res, ctx) => {
                const url = new URL(req.url);
                const period = url.searchParams.get("period") as PeriodType || PeriodTypeEnum.Month;
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                const topTeams = this.responseFactory.createHighPerformingTeamStats()
                    .map(stats => ({ ...stats, periodType: period }))
                    .slice(0, limit);
                
                const response = this.responseFactory.createStatsListResponse(topTeams);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get time series data for team
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id/timeseries`, (req, res, ctx) => {
                const { id } = req.params;
                const url = new URL(req.url);
                const days = parseInt(url.searchParams.get("days") || "30");
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(id as string, days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get aggregated statistics for multiple teams
            rest.post(`${this.responseFactory["baseUrl"]}/api/stats/team/aggregate`, async (req, res, ctx) => {
                const { teamIds, periodType } = await req.json() as {
                    teamIds: string[];
                    periodType: PeriodType;
                };
                
                const aggregatedStats = this.responseFactory.createComparativeStats(teamIds)
                    .map(stats => ({ ...stats, periodType }));
                
                const response = this.responseFactory.createStatsListResponse(aggregatedStats);
                
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
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("view")),
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory["baseUrl"]}/api/stats/team/search`, (req, res, ctx) => {
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
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, (req, res, ctx) => {
                const { id } = req.params;
                const stats = this.responseFactory.createMockStats({
                    team: {
                        ...this.responseFactory.createMockStats().team,
                        id: id as string,
                    },
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
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
            rest.get(`${this.responseFactory["baseUrl"]}/api/stats/team/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            rest.post(`${this.responseFactory["baseUrl"]}/api/stats/team/search`, (req, res, ctx) => {
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
    
    topPerformers: (period?: PeriodType) => {
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
