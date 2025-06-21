/**
 * StatsSite API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for site-wide statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    StatsSite,
    StatsSiteSearchInput,
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
 * StatsSite API response factory
 */
export class StatsSiteResponseFactory {
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
     * Create successful site statistics response
     */
    createSuccessResponse(stats: StatsSite): APIResponse<StatsSite> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/stats/site/${stats.id}`
                }
            }
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
            totalCount: statsList.length
        };
        
        return {
            data: statsList,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/stats/site?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/stats/site'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(period: string): APIErrorResponse {
        return {
            error: {
                code: 'SITE_STATS_NOT_FOUND',
                message: `Site statistics for period '${period}' were not found`,
                details: {
                    period,
                    searchCriteria: { period }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/site/${period}`
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
                message: `You do not have permission to ${operation} site statistics`,
                details: {
                    operation,
                    requiredPermissions: ['admin:stats:read'],
                    userPermissions: ['user:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/stats/site'
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
                path: '/api/stats/site'
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
                path: '/api/stats/site'
            }
        };
    }
    
    /**
     * Create mock site statistics
     */
    createMockStats(overrides?: Partial<StatsSite>): StatsSite {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultStats: StatsSite = {
            __typename: "StatsSite",
            id,
            periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
            periodEnd: now,
            periodType: PeriodTypeEnum.Week,
            
            // User statistics
            users: Math.floor(Math.random() * 10000) + 1000,
            usersLoggedIn: Math.floor(Math.random() * 5000) + 500,
            usersCreated: Math.floor(Math.random() * 200) + 20,
            
            // Content statistics  
            apis: Math.floor(Math.random() * 500) + 100,
            apisCreated: Math.floor(Math.random() * 30) + 5,
            projects: Math.floor(Math.random() * 1000) + 200,
            projectsCreated: Math.floor(Math.random() * 50) + 10,
            routines: Math.floor(Math.random() * 2000) + 300,
            routinesCreated: Math.floor(Math.random() * 100) + 15,
            routinesCompleted: Math.floor(Math.random() * 500) + 50,
            runs: Math.floor(Math.random() * 5000) + 1000,
            runsStarted: Math.floor(Math.random() * 1000) + 200,
            runsCompleted: Math.floor(Math.random() * 800) + 150,
            
            // Engagement statistics
            verificationApplications: Math.floor(Math.random() * 50) + 5,
            verificationApplicationsApproved: Math.floor(Math.random() * 30) + 2,
            reports: Math.floor(Math.random() * 20) + 1,
            reportsActionTaken: Math.floor(Math.random() * 15) + 1,
            
            // Community statistics
            teams: Math.floor(Math.random() * 300) + 50,
            teamsCreated: Math.floor(Math.random() * 20) + 3,
            
            // System statistics
            pageViews: Math.floor(Math.random() * 100000) + 10000,
            uniqueVisitors: Math.floor(Math.random() * 20000) + 2000,
            
            // Performance metrics
            averageResponseTime: Math.floor(Math.random() * 500) + 100, // milliseconds
            uptime: 99.9 - Math.random() * 0.1 // percentage
        };
        
        return {
            ...defaultStats,
            ...overrides
        };
    }
    
    /**
     * Create site statistics for different time periods
     */
    createStatsByPeriod(): Record<PeriodType, StatsSite> {
        const baseStats = this.createMockStats();
        
        return {
            [PeriodTypeEnum.Day]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                users: 150,
                usersLoggedIn: 80,
                usersCreated: 5,
                pageViews: 2500,
                uniqueVisitors: 400,
                runsStarted: 45,
                runsCompleted: 35
            },
            [PeriodTypeEnum.Week]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Week,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                users: 1200,
                usersLoggedIn: 650,
                usersCreated: 35,
                pageViews: 18000,
                uniqueVisitors: 2800,
                runsStarted: 320,
                runsCompleted: 280
            },
            [PeriodTypeEnum.Month]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Month,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                users: 5500,
                usersLoggedIn: 3200,
                usersCreated: 180,
                pageViews: 85000,
                uniqueVisitors: 12000,
                runsStarted: 1500,
                runsCompleted: 1200
            },
            [PeriodTypeEnum.Year]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Year,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                users: 25000,
                usersLoggedIn: 18000,
                usersCreated: 2200,
                pageViews: 1200000,
                uniqueVisitors: 180000,
                runsStarted: 25000,
                runsCompleted: 20000
            },
            [PeriodTypeEnum.AllTime]: {
                ...baseStats,
                periodType: PeriodTypeEnum.AllTime,
                periodStart: new Date(Date.now() - 3 * 365 * 24 * 60 * 60 * 1000).toISOString(),
                users: 50000,
                usersLoggedIn: 35000,
                usersCreated: 50000,
                pageViews: 5000000,
                uniqueVisitors: 750000,
                runsStarted: 100000,
                runsCompleted: 80000
            }
        };
    }
    
    /**
     * Create trending time series data
     */
    createTimeSeriesStats(days: number = 30): StatsSite[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            // Simulate growth trend
            const growthFactor = 1 + (index / days) * 0.1;
            
            return this.createMockStats({
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                users: Math.floor((100 + Math.random() * 50) * growthFactor),
                usersLoggedIn: Math.floor((50 + Math.random() * 30) * growthFactor),
                usersCreated: Math.floor((2 + Math.random() * 8) * growthFactor),
                pageViews: Math.floor((1500 + Math.random() * 1000) * growthFactor),
                uniqueVisitors: Math.floor((300 + Math.random() * 200) * growthFactor),
                runsStarted: Math.floor((30 + Math.random() * 20) * growthFactor),
                runsCompleted: Math.floor((25 + Math.random() * 15) * growthFactor)
            });
        });
    }
    
    /**
     * Create admin dashboard statistics
     */
    createAdminDashboardStats(): StatsSite {
        return this.createMockStats({
            periodType: PeriodTypeEnum.Day,
            users: 12500,
            usersLoggedIn: 3400,
            usersCreated: 45,
            apis: 850,
            apisCreated: 12,
            projects: 2400,
            projectsCreated: 28,
            routines: 5800,
            routinesCreated: 65,
            routinesCompleted: 340,
            runs: 15000,
            runsStarted: 480,
            runsCompleted: 420,
            teams: 320,
            teamsCreated: 8,
            pageViews: 45000,
            uniqueVisitors: 8200,
            verificationApplications: 15,
            verificationApplicationsApproved: 8,
            reports: 3,
            reportsActionTaken: 2,
            averageResponseTime: 180,
            uptime: 99.95
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
            performanceScore: 92.4 // percentage
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get current site statistics
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/current`, (req, res, ctx) => {
                const url = new URL(req.url);
                const period = url.searchParams.get('period') as PeriodType || PeriodTypeEnum.Week;
                
                const statsByPeriod = this.responseFactory.createStatsByPeriod();
                const stats = statsByPeriod[period];
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get site statistics by period
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/:period`, (req, res, ctx) => {
                const { period } = req.params;
                
                const statsByPeriod = this.responseFactory.createStatsByPeriod();
                const stats = statsByPeriod[period as PeriodType] || statsByPeriod[PeriodTypeEnum.Week];
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Search site statistics
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/site/search`, async (req, res, ctx) => {
                const body = await req.json() as StatsSiteSearchInput;
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                
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
                        totalCount: statsList.length
                    }
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get admin dashboard statistics
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/admin/dashboard`, (req, res, ctx) => {
                const stats = this.responseFactory.createAdminDashboardStats();
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get growth metrics
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/growth`, (req, res, ctx) => {
                const growthMetrics = this.responseFactory.createGrowthMetrics();
                
                return res(
                    ctx.status(200),
                    ctx.json({
                        data: growthMetrics,
                        meta: {
                            timestamp: new Date().toISOString(),
                            requestId: this.responseFactory['generateRequestId'](),
                            version: '1.0'
                        }
                    })
                );
            }),
            
            // Get time series data
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/timeseries`, (req, res, ctx) => {
                const url = new URL(req.url);
                const days = parseInt(url.searchParams.get('days') || '30');
                
                const timeSeriesData = this.responseFactory.createTimeSeriesStats(days);
                const response = this.responseFactory.createStatsListResponse(timeSeriesData);
                
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
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/:period`, (req, res, ctx) => {
                const { period } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(period as string))
                );
            }),
            
            // Permission error (admin only)
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/admin/dashboard`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('view admin statistics'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/site/search`, (req, res, ctx) => {
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
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/current`, (req, res, ctx) => {
                const stats = this.responseFactory.createMockStats();
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
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/current`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
            }),
            
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/site/admin/dashboard`, (req, res, ctx) => {
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
export const statsSiteResponseScenarios = {
    // Success scenarios
    currentStats: (period?: PeriodType) => {
        const factory = new StatsSiteResponseFactory();
        const statsByPeriod = factory.createStatsByPeriod();
        const stats = statsByPeriod[period || PeriodTypeEnum.Week];
        return factory.createSuccessResponse(stats);
    },
    
    adminDashboard: () => {
        const factory = new StatsSiteResponseFactory();
        return factory.createSuccessResponse(
            factory.createAdminDashboardStats()
        );
    },
    
    growthMetrics: () => {
        const factory = new StatsSiteResponseFactory();
        return {
            data: factory.createGrowthMetrics(),
            meta: {
                timestamp: new Date().toISOString(),
                requestId: factory['generateRequestId'](),
                version: '1.0'
            }
        };
    },
    
    timeSeriesData: (days?: number) => {
        const factory = new StatsSiteResponseFactory();
        return factory.createStatsListResponse(
            factory.createTimeSeriesStats(days)
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
            period || 'invalid-period'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsSiteResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'view admin statistics'
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
    networkErrorHandlers: () => new StatsSiteMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const statsSiteResponseFactory = new StatsSiteResponseFactory();
export const statsSiteMSWHandlers = new StatsSiteMSWHandlers();