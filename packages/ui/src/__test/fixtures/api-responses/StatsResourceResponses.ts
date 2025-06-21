/**
 * StatsResource API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource statistics endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    StatsResource,
    StatsResourceSearchInput,
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
 * StatsResource API response factory
 */
export class StatsResourceResponseFactory {
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
     * Create successful stats resource response
     */
    createSuccessResponse(stats: StatsResource): APIResponse<StatsResource> {
        return {
            data: stats,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/stats/resource/${stats.id}`,
                    related: {
                        resource: `${this.baseUrl}/api/resource/${stats.resource?.id}`
                    }
                }
            }
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
            totalCount: statsList.length
        };
        
        return {
            data: statsList,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/stats/resource?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/stats/resource'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(resourceId: string): APIErrorResponse {
        return {
            error: {
                code: 'RESOURCE_STATS_NOT_FOUND',
                message: `Resource statistics for ID '${resourceId}' were not found`,
                details: {
                    resourceId,
                    searchCriteria: { resourceId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/stats/resource/${resourceId}`
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
                message: `You do not have permission to ${operation} resource statistics`,
                details: {
                    operation,
                    requiredPermissions: ['stats:read'],
                    userPermissions: []
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/stats/resource'
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
                path: '/api/stats/resource'
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
                path: '/api/stats/resource'
            }
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
            periodType: PeriodTypeEnum.Week,
            views: Math.floor(Math.random() * 500) + 50,
            downloads: Math.floor(Math.random() * 100) + 10,
            bookmarks: Math.floor(Math.random() * 50) + 5,
            uses: Math.floor(Math.random() * 200) + 20,
            resource: {
                __typename: "Resource",
                id: `resource_${id}`,
                created_at: now,
                updated_at: now,
                isInternal: false,
                isPrivate: false,
                usedBy: [],
                usedByCount: Math.floor(Math.random() * 10),
                versions: [],
                versionsCount: Math.floor(Math.random() * 5) + 1,
                you: {
                    __typename: "ResourceYou",
                    canDelete: false,
                    canUpdate: false,
                    canReport: false,
                    isBookmarked: false,
                    isReacted: false,
                    reaction: null
                }
            }
        };
        
        return {
            ...defaultStats,
            ...overrides
        };
    }
    
    /**
     * Create trending resource statistics
     */
    createTrendingResourceStats(): StatsResource[] {
        return [
            this.createMockStats({
                views: 2500,
                downloads: 500,
                bookmarks: 150,
                uses: 800,
                periodType: PeriodTypeEnum.Month
            }),
            this.createMockStats({
                views: 1800,
                downloads: 350,
                bookmarks: 120,
                uses: 600,
                periodType: PeriodTypeEnum.Month
            }),
            this.createMockStats({
                views: 1200,
                downloads: 200,
                bookmarks: 80,
                uses: 400,
                periodType: PeriodTypeEnum.Month
            })
        ];
    }
    
    /**
     * Create statistics for different time periods
     */
    createStatsByPeriod(resourceId: string): Record<PeriodType, StatsResource> {
        const baseStats = this.createMockStats({
            resource: {
                ...this.createMockStats().resource,
                id: resourceId
            }
        });
        
        return {
            [PeriodTypeEnum.Day]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
                views: 45,
                downloads: 8,
                bookmarks: 3,
                uses: 12
            },
            [PeriodTypeEnum.Week]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Week,
                periodStart: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
                views: 320,
                downloads: 65,
                bookmarks: 25,
                uses: 95
            },
            [PeriodTypeEnum.Month]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Month,
                periodStart: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString(),
                views: 1500,
                downloads: 280,
                bookmarks: 110,
                uses: 450
            },
            [PeriodTypeEnum.Year]: {
                ...baseStats,
                periodType: PeriodTypeEnum.Year,
                periodStart: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000).toISOString(),
                views: 18000,
                downloads: 3500,
                bookmarks: 1200,
                uses: 5800
            },
            [PeriodTypeEnum.AllTime]: {
                ...baseStats,
                periodType: PeriodTypeEnum.AllTime,
                periodStart: new Date(Date.now() - 2 * 365 * 24 * 60 * 60 * 1000).toISOString(),
                views: 35000,
                downloads: 7200,
                bookmarks: 2500,
                uses: 12000
            }
        };
    }
    
    /**
     * Create time series data for resource analytics
     */
    createTimeSeriesStats(resourceId: string, days: number = 30): StatsResource[] {
        const oneDay = 24 * 60 * 60 * 1000;
        const baseTime = Date.now();
        
        return Array.from({ length: days }, (_, index) => {
            const dayStart = baseTime - (days - index) * oneDay;
            const dayEnd = dayStart + oneDay;
            
            return this.createMockStats({
                resource: {
                    ...this.createMockStats().resource,
                    id: resourceId
                },
                periodType: PeriodTypeEnum.Day,
                periodStart: new Date(dayStart).toISOString(),
                periodEnd: new Date(dayEnd).toISOString(),
                views: Math.floor(Math.random() * 100) + 10,
                downloads: Math.floor(Math.random() * 20) + 1,
                bookmarks: Math.floor(Math.random() * 10),
                uses: Math.floor(Math.random() * 30) + 5
            });
        });
    }
    
    /**
     * Create comparative statistics for multiple resources
     */
    createComparativeStats(resourceIds: string[]): StatsResource[] {
        return resourceIds.map(resourceId => 
            this.createMockStats({
                resource: {
                    ...this.createMockStats().resource,
                    id: resourceId
                },
                periodType: PeriodTypeEnum.Month,
                views: Math.floor(Math.random() * 2000) + 100,
                downloads: Math.floor(Math.random() * 400) + 20,
                bookmarks: Math.floor(Math.random() * 150) + 10,
                uses: Math.floor(Math.random() * 600) + 50
            })
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // Get resource statistics by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/:id`, (req, res, ctx) => {
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
                    resource: {
                        ...this.responseFactory.createMockStats().resource,
                        id: id as string
                    }
                });
                const response = this.responseFactory.createSuccessResponse(stats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Search resource statistics
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/resource/search`, async (req, res, ctx) => {
                const body = await req.json() as StatsResourceSearchInput;
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                
                let statsList: StatsResource[] = [];
                
                if (body.resourceIds && body.resourceIds.length > 0) {
                    statsList = this.responseFactory.createComparativeStats(body.resourceIds);
                } else {
                    statsList = this.responseFactory.createTrendingResourceStats();
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
            
            // Get trending resources
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/trending`, (req, res, ctx) => {
                const url = new URL(req.url);
                const period = url.searchParams.get('period') as PeriodType || PeriodTypeEnum.Month;
                const limit = parseInt(url.searchParams.get('limit') || '10');
                
                const trendingStats = this.responseFactory.createTrendingResourceStats()
                    .map(stats => ({ ...stats, periodType: period }))
                    .slice(0, limit);
                
                const response = this.responseFactory.createStatsListResponse(trendingStats);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Get time series data for resource
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/:id/timeseries`, (req, res, ctx) => {
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
            
            // Get aggregated statistics for multiple resources
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/resource/aggregate`, async (req, res, ctx) => {
                const { resourceIds, periodType } = await req.json() as {
                    resourceIds: string[];
                    periodType: PeriodType;
                };
                
                const aggregatedStats = this.responseFactory.createComparativeStats(resourceIds)
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
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/:id`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('view'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/resource/search`, (req, res, ctx) => {
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
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/:id`, (req, res, ctx) => {
                const { id } = req.params;
                const stats = this.responseFactory.createMockStats({
                    resource: {
                        ...this.responseFactory.createMockStats().resource,
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
            rest.get(`${this.responseFactory['baseUrl']}/api/stats/resource/:id`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
            }),
            
            rest.post(`${this.responseFactory['baseUrl']}/api/stats/resource/search`, (req, res, ctx) => {
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
export const statsResourceResponseScenarios = {
    // Success scenarios
    getSuccess: (stats?: StatsResource) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createSuccessResponse(
            stats || factory.createMockStats()
        );
    },
    
    searchSuccess: (statsList?: StatsResource[]) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createStatsListResponse(
            statsList || factory.createTrendingResourceStats()
        );
    },
    
    trendingResources: (period?: PeriodType) => {
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
            factory.createTimeSeriesStats(resourceId || 'resource-123', days)
        );
    },
    
    comparativeStats: (resourceIds?: string[]) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createStatsListResponse(
            factory.createComparativeStats(resourceIds || ['resource-1', 'resource-2', 'resource-3'])
        );
    },
    
    // Error scenarios
    notFoundError: (resourceId?: string) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new StatsResourceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'view'
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
    networkErrorHandlers: () => new StatsResourceMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const statsResourceResponseFactory = new StatsResourceResponseFactory();
export const statsResourceMSWHandlers = new StatsResourceMSWHandlers();