/**
 * Bookmark API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for bookmark endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    Bookmark, 
    BookmarkCreateInput, 
    BookmarkUpdateInput,
    BookmarkList,
    BookmarkFor, 
} from "@vrooli/shared";
import { 
    bookmarkValidation,
    BookmarkFor as BookmarkForEnum, 
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
 * Bookmark API response factory
 */
export class BookmarkResponseFactory {
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
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful bookmark response
     */
    createSuccessResponse(bookmark: Bookmark): APIResponse<Bookmark> {
        return {
            data: bookmark,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/bookmark/${bookmark.id}`,
                    related: {
                        list: `${this.baseUrl}/api/bookmark-list/${bookmark.list.id}`,
                        target: `${this.baseUrl}/api/${bookmark.to.__typename.toLowerCase()}/${bookmark.to.id}`,
                        user: `${this.baseUrl}/api/user/${bookmark.by.id}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create bookmark list response
     */
    createBookmarkListResponse(bookmarks: Bookmark[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Bookmark> {
        const paginationData = pagination || {
            page: 1,
            pageSize: bookmarks.length,
            totalCount: bookmarks.length,
        };
        
        return {
            data: bookmarks,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/bookmark?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/bookmark",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(bookmarkId: string): APIErrorResponse {
        return {
            error: {
                code: "BOOKMARK_NOT_FOUND",
                message: `Bookmark with ID '${bookmarkId}' was not found`,
                details: {
                    bookmarkId,
                    searchCriteria: { id: bookmarkId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/bookmark/${bookmarkId}`,
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
                message: `You do not have permission to ${operation} this bookmark`,
                details: {
                    operation,
                    requiredPermissions: ["bookmark:write"],
                    userPermissions: ["bookmark:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/bookmark",
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
                path: "/api/bookmark",
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
                path: "/api/bookmark",
            },
        };
    }
    
    /**
     * Create mock bookmark data
     */
    createMockBookmark(overrides?: Partial<Bookmark>): Bookmark {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultBookmark: Bookmark = {
            __typename: "Bookmark",
            id,
            createdAt: now,
            updatedAt: now,
            by: {
                __typename: "User",
                id: `user_${id}`,
                handle: "testuser",
                name: "Test User",
                createdAt: now,
                updatedAt: now,
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
                __typename: "Resource",
                id: `resource_${id}`,
                createdAt: now,
                updatedAt: now,
                isInternal: false,
                isPrivate: false,
                usedBy: [],
                usedByCount: 0,
                versions: [],
                versionsCount: 0,
                you: {
                    __typename: "ResourceYou",
                    canDelete: false,
                    canUpdate: false,
                    canReport: false,
                    isBookmarked: true,
                    isReacted: false,
                    reaction: null,
                },
            },
            list: {
                __typename: "BookmarkList",
                id: `list_${id}`,
                label: "My Bookmarks",
                createdAt: now,
                updatedAt: now,
                bookmarks: [],
                bookmarksCount: 1,
                you: {
                    __typename: "BookmarkListYou",
                    canDelete: true,
                    canUpdate: true,
                },
            },
        };
        
        return {
            ...defaultBookmark,
            ...overrides,
        };
    }
    
    /**
     * Create bookmark from API input
     */
    createBookmarkFromInput(input: BookmarkCreateInput): Bookmark {
        const bookmark = this.createMockBookmark();
        
        // Update bookmark based on input
        bookmark.to.__typename = input.bookmarkFor;
        bookmark.to.id = input.forConnect;
        
        if (input.listCreate) {
            bookmark.list.id = input.listCreate.id;
            bookmark.list.label = input.listCreate.label;
        } else if (input.listConnect) {
            bookmark.list.id = input.listConnect;
        }
        
        return bookmark;
    }
    
    /**
     * Create multiple bookmarks for different object types
     */
    createBookmarksForAllTypes(): Bookmark[] {
        return Object.values(BookmarkForEnum).map(bookmarkFor => 
            this.createMockBookmark({
                to: {
                    ...this.createMockBookmark().to,
                    __typename: bookmarkFor,
                    id: `${bookmarkFor.toLowerCase()}_${this.generateId()}`,
                },
            }),
        );
    }
    
    /**
     * Validate bookmark create input
     */
    async validateCreateInput(input: BookmarkCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await bookmarkValidation.create.validate(input);
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
 * MSW handlers factory for bookmark endpoints
 */
export class BookmarkMSWHandlers {
    private responseFactory: BookmarkResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new BookmarkResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all bookmark endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create bookmark
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, async (req, res, ctx) => {
                const body = await req.json() as BookmarkCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create bookmark
                const bookmark = this.responseFactory.createBookmarkFromInput(body);
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get bookmark by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const bookmark = this.responseFactory.createMockBookmark({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update bookmark
            http.put(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as BookmarkUpdateInput;
                
                const bookmark = this.responseFactory.createMockBookmark({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete bookmark
            http.delete(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List bookmarks
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const bookmarkFor = url.searchParams.get("bookmarkFor") as BookmarkFor;
                
                let bookmarks = this.responseFactory.createBookmarksForAllTypes();
                
                // Filter by bookmark type if specified
                if (bookmarkFor) {
                    bookmarks = bookmarks.filter(b => b.to.__typename === bookmarkFor);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedBookmarks = bookmarks.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createBookmarkListResponse(
                    paginatedBookmarks,
                    {
                        page,
                        pageSize: limit,
                        totalCount: bookmarks.length,
                    },
                );
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        forConnect: "Target object ID is required",
                        bookmarkFor: "Bookmark type must be specified",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, async (req, res, ctx) => {
                const body = await req.json() as BookmarkCreateInput;
                const bookmark = this.responseFactory.createBookmarkFromInput(body);
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, (req, res, ctx) => {
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
export const bookmarkResponseScenarios = {
    // Success scenarios
    createSuccess: (bookmark?: Bookmark) => {
        const factory = new BookmarkResponseFactory();
        return factory.createSuccessResponse(
            bookmark || factory.createMockBookmark(),
        );
    },
    
    listSuccess: (bookmarks?: Bookmark[]) => {
        const factory = new BookmarkResponseFactory();
        return factory.createBookmarkListResponse(
            bookmarks || factory.createBookmarksForAllTypes(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new BookmarkResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                forConnect: "Target object is required",
                bookmarkFor: "Bookmark type must be specified",
            },
        );
    },
    
    notFoundError: (bookmarkId?: string) => {
        const factory = new BookmarkResponseFactory();
        return factory.createNotFoundErrorResponse(
            bookmarkId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new BookmarkResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new BookmarkResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new BookmarkMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new BookmarkMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new BookmarkMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new BookmarkMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const bookmarkResponseFactory = new BookmarkResponseFactory();
export const bookmarkMSWHandlers = new BookmarkMSWHandlers();
