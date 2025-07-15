/**
 * Bookmark API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for bookmark endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-bookmark-types | LAST: 2025-07-02 - Fixed User/Resource properties, MSW v2 syntax, and BookmarkTo type compatibility

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    Bookmark, 
    BookmarkCreateInput, 
    BookmarkUpdateInput,
    BookmarkList,
    BookmarkFor,
    BookmarkTo,
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
        return `req_${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Generate unique resource ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
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
                isBotDepictingPerson: false,
                isPrivate: false,
                isPrivateBookmarks: true,
                isPrivateMemberships: true,
                isPrivatePullRequests: true,
                isPrivateResources: false,
                isPrivateResourcesCreated: false,
                isPrivateTeamsCreated: true,
                isPrivateVotes: true,
                bookmarkedBy: [],
                bookmarks: 0,
                profileImage: null,
                bannerImage: null,
                publicId: `user_public_${id}`,
                views: 0,
                premium: null,
                wallets: [],
                translations: [],
                membershipsCount: 0,
                reportsReceived: [],
                reportsReceivedCount: 0,
                resourcesCount: 0,
                you: {
                    __typename: "UserYou",
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isViewed: false,
                },
            },
            to: {
                __typename: "Resource",
                id: `resource_${id}`,
                createdAt: now,
                updatedAt: now,
                bookmarkedBy: [],
                bookmarks: 0,
                completedAt: null,
                createdBy: null,
                hasCompleteVersion: false,
                isDeleted: false,
                isInternal: false,
                isPrivate: false,
                issues: [],
                issuesCount: 0,
                owner: null,
                parent: null,
                permissions: "{}",
                publicId: `resource_public_${id}`,
                pullRequests: [],
                pullRequestsCount: 0,
                resourceType: "Api" as any,
                score: 0,
                stats: [],
                tags: [],
                transfers: [],
                transfersCount: 0,
                translatedName: "Test Resource",
                versions: [],
                versionsCount: 0,
                views: 0,
                you: {
                    __typename: "ResourceYou",
                    canBookmark: true,
                    canComment: true,
                    canDelete: false,
                    canReact: true,
                    canRead: true,
                    canTransfer: false,
                    canUpdate: false,
                    isBookmarked: true,
                    isViewed: false,
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
        return Object.values(BookmarkForEnum).map(bookmarkFor => {
            const now = new Date().toISOString();
            const id = `${(bookmarkFor as string).toLowerCase()}_${this.generateId()}`;
            
            let to: BookmarkTo;
            
            // Create appropriate object based on type
            if (bookmarkFor === "User") {
                to = {
                    __typename: "User",
                    id,
                    handle: "testuser",
                    name: "Test User",
                    createdAt: now,
                    updatedAt: now,
                    isBot: false,
                    isPrivate: false,
                    profileImage: null,
                    bannerImage: null,
                    premium: null,
                        wallets: [],
                    translations: [],
                    bookmarkedBy: [],
                    bookmarks: 0,
                    publicId: `user_public_${id}`,
                    views: 0,
                    membershipsCount: 0,
                    reportsReceived: [],
                    reportsReceivedCount: 0,
                    resourcesCount: 0,
                    isBotDepictingPerson: false,
                    isPrivateBookmarks: true,
                    isPrivateMemberships: true,
                    isPrivatePullRequests: true,
                    isPrivateResources: false,
                    isPrivateResourcesCreated: false,
                    isPrivateTeamsCreated: true,
                    isPrivateVotes: true,
                    you: {
                        __typename: "UserYou",
                        canDelete: false,
                        canReport: false,
                        canUpdate: false,
                        isBookmarked: false,
                        isViewed: false,
                    },
                };
            } else {
                // For now, default to Resource for other types
                to = {
                    __typename: "Resource",
                    id,
                    createdAt: now,
                    updatedAt: now,
                    bookmarkedBy: [],
                    bookmarks: 0,
                    completedAt: null,
                    createdBy: null,
                    hasCompleteVersion: false,
                    isDeleted: false,
                    isInternal: false,
                    isPrivate: false,
                    issues: [],
                    issuesCount: 0,
                    owner: null,
                    parent: null,
                    permissions: "{}",
                    publicId: `resource_public_${id}`,
                    pullRequests: [],
                    pullRequestsCount: 0,
                    resourceType: "Api" as any,
                    score: 0,
                    stats: [],
                    tags: [],
                    transfers: [],
                    transfersCount: 0,
                    translatedName: "Test Resource",
                    versions: [],
                    versionsCount: 0,
                    views: 0,
                    you: {
                        __typename: "ResourceYou",
                        canBookmark: true,
                        canComment: true,
                        canDelete: false,
                        canReact: true,
                        canRead: true,
                        canTransfer: false,
                        canUpdate: false,
                        isBookmarked: true,
                        isViewed: false,
                        reaction: null,
                    },
                };
            }
            
            return this.createMockBookmark({ to });
        });
    }
    
    /**
     * Validate bookmark create input
     */
    async validateCreateInput(input: BookmarkCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await bookmarkValidation.create({}).validate(input);
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
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create bookmark
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, async ({ request, params }) => {
                const body = await request.json() as BookmarkCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 }
                    );
                }
                
                // Create bookmark
                const bookmark = this.responseFactory.createBookmarkFromInput(body);
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get bookmark by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, ({ request, params }) => {
                const { id } = params;
                
                const bookmark = this.responseFactory.createMockBookmark({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update bookmark
            http.put(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as BookmarkUpdateInput;
                
                const bookmark = this.responseFactory.createMockBookmark({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete bookmark
            http.delete(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, ({ request, params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List bookmarks
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark`, ({ request, params }) => {
                const url = new URL(request.url);
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
                
                return HttpResponse.json(response, { status: 200 });
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers(): RequestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        forConnect: "Target object ID is required",
                        bookmarkFor: "Bookmark type must be specified",
                    }),
                    { status: 400 }
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, ({ request, params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 }
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, ({ request, params }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 }
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, ({ request, params }) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, async ({ request, params }) => {
                const body = await request.json() as BookmarkCreateInput;
                const bookmark = this.responseFactory.createBookmarkFromInput(body);
                const response = this.responseFactory.createSuccessResponse(bookmark);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 201 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/bookmark`, ({ request, params }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/bookmark/:id`, ({ request, params }) => {
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
        
        const httpMethod = method.toLowerCase() as keyof typeof http;
        return http[httpMethod](fullEndpoint, async ({ request, params }) => {
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
