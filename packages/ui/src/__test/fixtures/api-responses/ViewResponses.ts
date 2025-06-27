/**
 * View API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for view endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    View, 
    ViewCreateInput, 
    ViewUpdateInput,
    ViewFor,
} from "@vrooli/shared";
import { 
    viewValidation,
    ViewFor as ViewForEnum, 
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
 * View API response factory
 */
export class ViewResponseFactory {
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
     * Generate unique view ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful view response
     */
    createSuccessResponse(view: View): APIResponse<View> {
        return {
            data: view,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/view/${view.id}`,
                    related: {
                        user: view.by ? `${this.baseUrl}/api/user/${view.by.id}` : undefined,
                        object: `${this.baseUrl}/api/${view.to.__typename.toLowerCase()}/${view.to.id}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create view list response
     */
    createViewListResponse(views: View[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<View> {
        const paginationData = pagination || {
            page: 1,
            pageSize: views.length,
            totalCount: views.length,
        };
        
        return {
            data: views,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/view?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/view",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(viewId: string): APIErrorResponse {
        return {
            error: {
                code: "VIEW_NOT_FOUND",
                message: `View with ID '${viewId}' was not found`,
                details: {
                    viewId,
                    searchCriteria: { id: viewId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/view/${viewId}`,
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
                message: `You do not have permission to ${operation} this view`,
                details: {
                    operation,
                    requiredPermissions: ["view:write"],
                    userPermissions: ["view:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/view",
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
                path: "/api/view",
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
                path: "/api/view",
            },
        };
    }
    
    /**
     * Create mock view data
     */
    createMockView(overrides?: Partial<View>): View {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultView: View = {
            __typename: "View",
            id,
            created_at: now,
            updated_at: now,
            lastViewedAt: now,
            by: {
                __typename: "User",
                id: `user_${id}`,
                handle: "viewer_user",
                name: "Viewer User",
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
                        count: 0,
                    },
                },
            },
            to: {
                __typename: "Routine",
                id: `routine_${id}`,
                created_at: now,
                updated_at: now,
                isInternal: false,
                isPrivate: false,
                completedAt: null,
                createdBy: null,
                hasCompleteVersion: true,
                score: 4.5,
                bookmarks: 25,
                views: 150,
                you: {
                    __typename: "RoutineYou",
                    canComment: true,
                    canDelete: false,
                    canBookmark: true,
                    canUpdate: false,
                    canRead: true,
                    canReact: true,
                    isBookmarked: false,
                    reaction: null,
                },
            },
        };
        
        return {
            ...defaultView,
            ...overrides,
        };
    }
    
    /**
     * Create view from API input
     */
    createViewFromInput(input: ViewCreateInput): View {
        const view = this.createMockView();
        
        // Update view based on input
        if (input.forConnect) {
            view.to.id = input.forConnect;
        }
        
        if (input.viewFor) {
            view.to.__typename = input.viewFor;
        }
        
        return view;
    }
    
    /**
     * Create views for different object types
     */
    createViewsForAllTypes(): View[] {
        return Object.values(ViewForEnum).map(viewFor => 
            this.createMockView({
                to: {
                    ...this.createMockView().to,
                    __typename: viewFor,
                    id: `${viewFor.toLowerCase()}_${this.generateId()}`,
                },
            }),
        );
    }
    
    /**
     * Create trending views (high view count)
     */
    createTrendingViews(): View[] {
        return [
            this.createMockView({
                to: {
                    ...this.createMockView().to,
                    __typename: "Routine",
                    views: 1500,
                    score: 4.8,
                },
            }),
            this.createMockView({
                to: {
                    ...this.createMockView().to,
                    __typename: "Project",
                    views: 1200,
                    score: 4.6,
                },
            }),
            this.createMockView({
                to: {
                    ...this.createMockView().to,
                    __typename: "Api",
                    views: 1000,
                    score: 4.7,
                },
            }),
        ];
    }
    
    /**
     * Create recent views for a user
     */
    createRecentViewsForUser(userId: string, count = 10): View[] {
        const baseTime = Date.now();
        const hour = 60 * 60 * 1000;
        
        return Array.from({ length: count }, (_, index) => 
            this.createMockView({
                by: {
                    ...this.createMockView().by,
                    id: userId,
                },
                lastViewedAt: new Date(baseTime - (index * hour)).toISOString(),
                to: {
                    ...this.createMockView().to,
                    __typename: index % 2 === 0 ? "Routine" : "Project",
                    id: `${index % 2 === 0 ? "routine" : "project"}_${this.generateId()}`,
                },
            }),
        );
    }
    
    /**
     * Create views by time period
     */
    createViewsByTimePeriod(period: "today" | "week" | "month"): View[] {
        const now = Date.now();
        const periods = {
            today: 24 * 60 * 60 * 1000,
            week: 7 * 24 * 60 * 60 * 1000,
            month: 30 * 24 * 60 * 60 * 1000,
        };
        
        const timespan = periods[period];
        const count = period === "today" ? 20 : period === "week" ? 100 : 500;
        
        return Array.from({ length: count }, (_, index) => 
            this.createMockView({
                lastViewedAt: new Date(now - (Math.random() * timespan)).toISOString(),
            }),
        );
    }
    
    /**
     * Validate view create input
     */
    async validateCreateInput(input: ViewCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await viewValidation.create.validate(input);
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
 * MSW handlers factory for view endpoints
 */
export class ViewMSWHandlers {
    private responseFactory: ViewResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ViewResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all view endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create view (track view)
            http.post(`${this.responseFactory["baseUrl"]}/api/view`, async (req, res, ctx) => {
                const body = await req.json() as ViewCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create view
                const view = this.responseFactory.createViewFromInput(body);
                const response = this.responseFactory.createSuccessResponse(view);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get view by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/view/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const view = this.responseFactory.createMockView({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(view);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update view
            http.put(`${this.responseFactory["baseUrl"]}/api/view/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ViewUpdateInput;
                
                const view = this.responseFactory.createMockView({ 
                    id: id as string,
                    lastViewedAt: new Date().toISOString(),
                    updated_at: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(view);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete view
            http.delete(`${this.responseFactory["baseUrl"]}/api/view/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List views
            http.get(`${this.responseFactory["baseUrl"]}/api/view`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const viewFor = url.searchParams.get("viewFor") as ViewFor;
                const userId = url.searchParams.get("userId");
                const trending = url.searchParams.get("trending") === "true";
                const period = url.searchParams.get("period") as "today" | "week" | "month";
                
                let views: View[] = [];
                
                if (trending) {
                    views = this.responseFactory.createTrendingViews();
                } else if (userId) {
                    views = this.responseFactory.createRecentViewsForUser(userId);
                } else if (period) {
                    views = this.responseFactory.createViewsByTimePeriod(period);
                } else {
                    views = this.responseFactory.createViewsForAllTypes();
                }
                
                // Filter by view type if specified
                if (viewFor) {
                    views = views.filter(v => v.to.__typename === viewFor);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedViews = views.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createViewListResponse(
                    paginatedViews,
                    {
                        page,
                        pageSize: limit,
                        totalCount: views.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get trending views
            http.get(`${this.responseFactory["baseUrl"]}/api/view/trending`, (req, res, ctx) => {
                const views = this.responseFactory.createTrendingViews();
                const response = this.responseFactory.createViewListResponse(views);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get recent views for user
            http.get(`${this.responseFactory["baseUrl"]}/api/view/recent/:userId`, (req, res, ctx) => {
                const { userId } = req.params;
                
                const views = this.responseFactory.createRecentViewsForUser(userId as string);
                const response = this.responseFactory.createViewListResponse(views);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/view`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        forConnect: "Target object is required",
                        viewFor: "View type must be specified",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/view/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/view`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/view`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/view`, async (req, res, ctx) => {
                const body = await req.json() as ViewCreateInput;
                const view = this.responseFactory.createViewFromInput(body);
                const response = this.responseFactory.createSuccessResponse(view);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/view`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/view/:id`, (req, res, ctx) => {
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
export const viewResponseScenarios = {
    // Success scenarios
    createSuccess: (view?: View) => {
        const factory = new ViewResponseFactory();
        return factory.createSuccessResponse(
            view || factory.createMockView(),
        );
    },
    
    listSuccess: (views?: View[]) => {
        const factory = new ViewResponseFactory();
        return factory.createViewListResponse(
            views || factory.createViewsForAllTypes(),
        );
    },
    
    trendingViews: () => {
        const factory = new ViewResponseFactory();
        return factory.createViewListResponse(
            factory.createTrendingViews(),
        );
    },
    
    recentViews: (userId?: string) => {
        const factory = new ViewResponseFactory();
        return factory.createViewListResponse(
            factory.createRecentViewsForUser(userId || "user-123"),
        );
    },
    
    todayViews: () => {
        const factory = new ViewResponseFactory();
        return factory.createViewListResponse(
            factory.createViewsByTimePeriod("today"),
        );
    },
    
    weekViews: () => {
        const factory = new ViewResponseFactory();
        return factory.createViewListResponse(
            factory.createViewsByTimePeriod("week"),
        );
    },
    
    monthViews: () => {
        const factory = new ViewResponseFactory();
        return factory.createViewListResponse(
            factory.createViewsByTimePeriod("month"),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ViewResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                forConnect: "Target object is required",
                viewFor: "View type must be specified",
            },
        );
    },
    
    notFoundError: (viewId?: string) => {
        const factory = new ViewResponseFactory();
        return factory.createNotFoundErrorResponse(
            viewId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ViewResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ViewResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ViewMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ViewMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ViewMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ViewMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const viewResponseFactory = new ViewResponseFactory();
export const viewMSWHandlers = new ViewMSWHandlers();
