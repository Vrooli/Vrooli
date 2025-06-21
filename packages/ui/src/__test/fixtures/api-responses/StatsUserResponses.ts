/**
 * StatsUser API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for user statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    StatsUser,
    StatsUserSearchInput,
    PeriodType
} from '@vrooli/shared';
import { 
    PeriodType as PeriodTypeEnum 
} from '@vrooli/shared';

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
    
    constructor(baseUrl: string = process.env.VITE_SERVER_URL || 'http://localhost:5329') {
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
     * Create successful user statistics response
     */
    createSuccessResponse(stats: StatsUser): APIResponse<StatsUser> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/stats/user/${stats.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/${stats.user?.id}`
                    }
                }
            }
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
            totalCount: statsList.length
        };
        
        return {
            data: statsList,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/stats/user?page=${paginationData.page}&limit=${paginationData.pageSize}`
                }
            },
            pagination: {
                ...paginationData,
                totalPages: Math.ceil(paginationData.totalCount / paginationData.pageSize),
                hasNextPage: paginationData.page * paginationData.pageSize < paginationData.totalCount,
                hasPreviousPage: paginationData.page > 1
            }
        };
    }
    
    /**
     * Create validation error response
     */
    createValidationErrorResponse(fieldErrors: Record<string, string>): APIErrorResponse {
        return {
            error: {
                code: 'VALIDATION_ERROR',
                message: 'The request contains invalid data',
                details: {
                    fieldErrors,
                    invalidFields: Object.keys(fieldErrors)
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/stats/user'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(userId: string): APIErrorResponse {
        return {
            error: {
                code: 'USER_STATS_NOT_FOUND',
                message: `User statistics for ID '${userId}' were not found`,
                details: {
                    userId,
                    searchCriteria: { userId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/user/${userId}`
            }
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string): APIErrorResponse {
        return {
            error: {
                code: 'PERMISSION_DENIED',
                message: `You do not have permission to ${operation} user statistics`,
                details: {
                    operation,
                    requiredPermissions: ['user:stats:read'],
                    userPermissions: ['user:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/stats/user'
            }
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: 'NETWORK_ERROR',
                message: 'Network request failed',
                details: {
                    reason: 'Connection timeout',
                    retryable: true,
                    retryAfter: 5000
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/stats/user'
            }
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(): APIErrorResponse {
        return {
            error: {
                code: 'INTERNAL_SERVER_ERROR',
                message: 'An unexpected server error occurred',
                details: {
                    errorId: `ERR_${Date.now()}`,
                    reportable: true,
                    retryable: true
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/stats/user'
            }
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
            periodType: PeriodTypeEnum.Week,
            
            // Content creation statistics
            apis: Math.floor(Math.random() * 10) + 1,
            apisCreated: Math.floor(Math.random() * 3),
            projects: Math.floor(Math.random() * 15) + 2,
            projectsCreated: Math.floor(Math.random() * 5) + 1,
            projectsCompleted: Math.floor(Math.random() * 3),
            routines: Math.floor(Math.random() * 25) + 5,
            routinesCreated: Math.floor(Math.random() * 8) + 1,
            routinesCompleted: Math.floor(Math.random() * 12) + 2,
            
            // Activity statistics
            runs: Math.floor(Math.random() * 50) + 10,
            runsStarted: Math.floor(Math.random() * 20) + 5,
            runsCompleted: Math.floor(Math.random() * 18) + 4,
            
            // Engagement statistics
            views: Math.floor(Math.random() * 500) + 50,
            bookmarks: Math.floor(Math.random() * 25) + 3,
            
            user: {
                __typename: "User",
                id: `user_${id}`,
                handle: `user_${id}`,
                name: "Test User",
                created_at: now,
                updated_at: now,
                isBot: false,
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                premium: false,
                premiumExpiration: null,
                roles: [],
                wallets: [],
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "UserYou",
                    isBlocked: false,
                    isBlockedByYou: false,
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isReacted: false,
                    reactionSummary: {
                        __typename: "ReactionSummary",
                        emotion: null,
                        count: 0
                    }
                }
            }
        };
        
        return {
            ...defaultStats,
            ...overrides
        };
    }
    
    /**
     * Create high-performing user statistics
     */
    createHighPerformingUserStats(): StatsUser[] {
        return [
            this.createMockStats({
                routinesCreated: 25,
                routinesCompleted: 80,
                projectsCreated: 12,
                projectsCompleted: 10,
                runsCompleted: 200,
                views: 5000,
                bookmarks: 300,
                periodType: PeriodTypeEnum.Month,
                user: {
                    ...this.createMockStats().user,
                    name: "Top Contributor",
                    handle: "top_contributor"
                }
            }),
            this.createMockStats({
                routinesCreated: 18,
                routinesCompleted: 65,
                projectsCreated: 8,
                projectsCompleted: 7,
                runsCompleted: 150,
                views: 3500,
                bookmarks: 220,
                periodType: PeriodTypeEnum.Month,
                user: {
                    ...this.createMockStats().user,
                    name: "Active Creator",
                    handle: "active_creator"
                }
            }),
            this.createMockStats({
                routinesCreated: 12,
                routinesCompleted: 45,
                projectsCreated: 6,
                projectsCompleted: 5,
                runsCompleted: 120,
                views: 2800,
                bookmarks: 180,
                periodType: PeriodTypeEnum.Month,
                user: {
                    ...this.createMockStats().user,
                    name: "Productive User",
                    handle: "productive_user"
                }
            })
        ];
    }
    
    /**
     * Create statistics for different time periods
     */
    createStatsByPeriod(userId: string): Record<PeriodType, StatsUser> {
        const baseStats = this.createMockStats({
            user: {
                ...this.createMockStats().user,
                id: userId
            }
        });
        
        return {
            [PeriodTypeEnum.Day]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 3,
                runsCompleted: 2,
                views: 15,
                bookmarks: 1,
                routinesCompleted: 2
            },
            [PeriodTypeEnum.Week]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Week,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 20,
                runsCompleted: 16,
                views: 120,
                bookmarks: 8,
                routinesCreated: 2,
                routinesCompleted: 12,
                projectsCreated: 1
            },
            [PeriodTypeEnum.Month]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Month,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 85,
                runsCompleted: 70,
                views: 500,
                bookmarks: 35,
                routinesCreated: 8,
                routinesCompleted: 45,
                projectsCreated: 3,
                projectsCompleted: 2,
                apisCreated: 1
            },
            [PeriodTypeEnum.Year]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Year,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 1000,
                runsCompleted: 850,
                views: 8000,
                bookmarks: 450,
                routinesCreated: 95,
                routinesCompleted: 500,
                projectsCreated: 35,
                projectsCompleted: 28,
                apisCreated: 12
            },
            [PeriodTypeEnum.AllTime]: {
                ...baseStats,
                periodType: PeriodTypeEnum.AllTime,
                periodStart: new Date(Date.now() - 2 * 365 * 24 * 60 * 60 * 1000).toISOString(),
                runsStarted: 2000,
                runsCompleted: 1700,
                views: 15000,
                bookmarks: 900,
                routinesCreated: 180,
                routinesCompleted: 1000,
                projectsCreated: 70,
                projectsCompleted: 58,
                apisCreated: 25
            }
        };
    }
    
    /**
     * Create time series data for user analytics
     */
    createTimeSeriesStats(userId: string, days: number = 30): StatsUser[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            return this.createMockStats({
                user: {
                    ...this.createMockStats().user,
                    id: userId
                },
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                runsStarted: Math.floor(Math.random() * 8) + 1,
                runsCompleted: Math.floor(Math.random() * 6) + 1,
                views: Math.floor(Math.random() * 30) + 5,
                bookmarks: Math.floor(Math.random() * 3),
                routinesCompleted: Math.floor(Math.random() * 4)
            });
        });
    }
    
    /**
     * Create comparative statistics for multiple users
     */
    createComparativeStats(userIds: string[]): StatsUser[] {
        return userIds.map(userId => 
            this.createMockStats({
                user: {
                    ...this.createMockStats().user,
                    id: userId,
                    name: `User ${userId}`,
                    handle: `user_${userId}`
                },
                periodType: PeriodTypeEnum.Month,
                runsCompleted: Math.floor(Math.random() * 100) + 20,
                routinesCreated: Math.floor(Math.random() * 15) + 2,
                projectsCompleted: Math.floor(Math.random() * 8) + 1,
                views: Math.floor(Math.random() * 1000) + 100
            })
        );
    }
    
    /**
     * Create personal dashboard statistics
     */
    createPersonalDashboardStats(userId: string): StatsUser {
        return this.createMockStats({
            user: {
                ...this.createMockStats().user,
                id: userId,
                name: "Your Profile",
                handle: "your_handle"
            },
            periodType: PeriodTypeEnum.Month,
            routines: 45,
            routinesCreated: 12,
            routinesCompleted: 38,
            projects: 18,
            projectsCreated: 6,
            projectsCompleted: 4,
            apis: 8,
            apisCreated: 3,
            runs: 180,
            runsStarted: 65,
            runsCompleted: 58,
            views: 1200,
            bookmarks: 85
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get user statistics by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/:id`, (req, res, ctx) => {
                const { id } = req.params;
                const url = new URL(req.url);
                const period = url.searchParams.get('period') as PeriodType;
                
                if (period) {
                    const statsByPeriod = this.responseFactory.createStatsByPeriod(id as string);
                    const stats = statsByPeriod[period] || statsByPeriod[PeriodTypeEnum.Month];
                    const response = this.responseFactory.createSuccessResponse(stats);
                    
                    return res(
                        ctx.status(200),
                        ctx.json(response)
                    );
                }
                
                const stats = this.responseFactory.createMockStats({
                    user: {
                        ...this.responseFactory.createMockStats().user,
                        id: id as string
                    }
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get current user's statistics
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/me`, (req, res, ctx) => {
                const url = new URL(req.url);
                const period = url.searchParams.get('period') as PeriodType || PeriodTypeEnum.Month;
                
                const stats = this.responseFactory.createPersonalDashboardStats('current-user');
                stats.periodType = period;
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Search user statistics
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/user/search`, async (req, res, ctx) => {
                const body = await req.json() as StatsUserSearchInput;
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                
                let statsList: StatsUser[] = [];
                
                if (body.userIds && body.userIds.length > 0) {
                    statsList = this.responseFactory.createComparativeStats(body.userIds);
                } else {
                    statsList = this.responseFactory.createHighPerformingUserStats();
                }
                
                // Filter by period if specified
                if (body.periodType) {
                    statsList = statsList.map(stats => ({
                        ...stats,
                        periodType: body.periodType!
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
                        totalCount: statsList.length
                    }
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get top performing users
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/top-performers`, (req, res, ctx) => {
                const url = new URL(req.url);
                const period = url.searchParams.get('period') as PeriodType || PeriodTypeEnum.Month;
                const limit = parseInt(url.searchParams.get('limit') || '10');
                
                const topUsers = this.responseFactory.createHighPerformingUserStats()
                    .map(stats => ({ ...stats, periodType: period }))
                    .slice(0, limit);
                
                const response = this.responseFactory.createStatsListResponse(topUsers);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get time series data for user
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/:id/timeseries`, (req, res, ctx) => {
                const { id } = req.params;
                const url = new URL(req.url);
                const days = parseInt(url.searchParams.get('days') || '30');
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(id as string, days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get aggregated statistics for multiple users
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/user/aggregate`, async (req, res, ctx) => {
                const { userIds, periodType } = await req.json() as {
                    userIds: string[];
                    periodType: PeriodType;
                };
                
                const aggregatedStats = this.responseFactory.createComparativeStats(userIds)
                    .map(stats => ({ ...stats, periodType }));
                
                const response = this.responseFactory.createStatsListResponse(aggregatedStats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            })
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RestHandler[] {
        return [
            // Not found error
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('view'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/user/search`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse())
                );
            })
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay: number = 2000): RestHandler[] {
        return [
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/:id`, (req, res, ctx) => {
                const { id } = req.params;
                const stats = this.responseFactory.createMockStats({
                    user: {
                        ...this.responseFactory.createMockStats().user,
                        id: id as string
                    }
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(200),
                    ctx.json(response)
                );
            })
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/user/:id`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
            }),
            
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/user/search`, (req, res, ctx) => {
                return res.networkError('Network connection failed');
            })
        ];
    }
    
    /**
     * Create custom handler with specific configuration
     */
    createCustomHandler(config: {
        endpoint: string;
        method: 'GET' | 'POST' | 'PUT' | 'DELETE';
        status: number;
        response: any;
        delay?: number;
    }): RestHandler {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory['baseUrl']}${endpoint}`;
        
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
export const statsUserResponseScenarios = {
    // Success scenarios
    getSuccess: (stats?: StatsUser) => {
        const factory = new StatsUserResponseFactory();
        return factory.createSuccessResponse(
            stats || factory.createMockStats()
        );
    },
    
    personalDashboard: (userId?: string) => {
        const factory = new StatsUserResponseFactory();
        return factory.createSuccessResponse(
            factory.createPersonalDashboardStats(userId || 'current-user')
        );
    },
    
    searchSuccess: (statsList?: StatsUser[]) => {
        const factory = new StatsUserResponseFactory();
        return factory.createStatsListResponse(
            statsList || factory.createHighPerformingUserStats()
        );
    },
    
    topPerformers: (period?: PeriodType) => {
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
            factory.createTimeSeriesStats(userId || 'user-123', days)
        );
    },
    
    comparativeStats: (userIds?: string[]) => {
        const factory = new StatsUserResponseFactory();
        return factory.createStatsListResponse(
            factory.createComparativeStats(userIds || ['user-1', 'user-2', 'user-3'])
        );
    },
    
    // Error scenarios
    notFoundError: (userId?: string) => {
        const factory = new StatsUserResponseFactory();
        return factory.createNotFoundErrorResponse(
            userId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsUserResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'view'
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
    networkErrorHandlers: () => new StatsUserMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const statsUserResponseFactory = new StatsUserResponseFactory();
export const statsUserMSWHandlers = new StatsUserMSWHandlers();