/**
 * PullRequest API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for pull request endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-user-resource-types | LAST: 2025-07-02 - Fixed User/ResourceVersion properties and removed invalid fields
// AI_CHECK: TYPE_SAFETY=fixed-premium-user-resource-types | LAST: 2025-07-02 - Fixed Premium, UserYou, ResourceYou, and ResourceVersionYou type errors

import { HttpResponse, http } from "msw";
import type { 
    PullRequest, 
    PullRequestCreateInput, 
    PullRequestUpdateInput,
    PullRequestStatus,
    PullRequestFromObjectType,
    PullRequestToObjectType,
    CommentTranslation,
    User,
    ResourceVersion,
    Resource,
    Comment,
} from "@vrooli/shared";
import { 
    pullRequestValidation,
    PullRequestStatus as PullRequestStatusEnum,
    PullRequestFromObjectType as PullRequestFromObjectTypeEnum,
    PullRequestToObjectType as PullRequestToObjectTypeEnum,
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
 * PullRequest API response factory
 */
export class PullRequestResponseFactory {
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
     * Generate unique public ID for pull requests
     */
    private generatePublicId(): string {
        return `PR-${Date.now().toString().slice(-6)}`;
    }
    
    /**
     * Create successful pull request response
     */
    createSuccessResponse(pullRequest: PullRequest): APIResponse<PullRequest> {
        return {
            data: pullRequest,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/pull-request/${pullRequest.id}`,
                    related: {
                        from: `${this.baseUrl}/api/resource-version/${pullRequest.from.id}`,
                        to: `${this.baseUrl}/api/resource/${pullRequest.to.id}`,
                        user: pullRequest.createdBy ? `${this.baseUrl}/api/user/${pullRequest.createdBy.id}` : undefined,
                        comments: `${this.baseUrl}/api/pull-request/${pullRequest.id}/comments`,
                    },
                },
            },
        };
    }
    
    /**
     * Create pull request list response
     */
    createPullRequestListResponse(pullRequests: PullRequest[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<PullRequest> {
        const paginationData = pagination || {
            page: 1,
            pageSize: pullRequests.length,
            totalCount: pullRequests.length,
        };
        
        return {
            data: pullRequests,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/pull-request?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/pull-request",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(pullRequestId: string): APIErrorResponse {
        return {
            error: {
                code: "PULL_REQUEST_NOT_FOUND",
                message: `Pull request with ID '${pullRequestId}' was not found`,
                details: {
                    pullRequestId,
                    searchCriteria: { id: pullRequestId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/pull-request/${pullRequestId}`,
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
                message: `You do not have permission to ${operation} this pull request`,
                details: {
                    operation,
                    requiredPermissions: ["pull-request:write"],
                    userPermissions: ["pull-request:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/pull-request",
            },
        };
    }
    
    /**
     * Create conflict error response (for merge conflicts)
     */
    createConflictErrorResponse(reason = "Merge conflict detected"): APIErrorResponse {
        return {
            error: {
                code: "MERGE_CONFLICT",
                message: "Pull request cannot be merged due to conflicts",
                details: {
                    reason,
                    conflictResolutionRequired: true,
                    conflictFiles: ["src/main.ts", "package.json"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/pull-request",
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
                path: "/api/pull-request",
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
                path: "/api/pull-request",
            },
        };
    }
    
    /**
     * Create mock user data
     */
    private createMockUser(id?: string): User {
        const userId = id || `user_${this.generateId()}`;
        const now = new Date().toISOString();
        
        return {
            __typename: "User",
            id: userId,
            handle: `user-${userId.slice(-4)}`,
            name: `Test User ${userId.slice(-4)}`,
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
            publicId: `user_public_${userId}`,
            views: 0,
            isBotDepictingPerson: false,
            isPrivateBookmarks: true,
            isPrivateMemberships: true,
            isPrivatePullRequests: true,
            isPrivateResources: false,
            isPrivateResourcesCreated: false,
            isPrivateTeamsCreated: true,
            isPrivateVotes: true,
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
        };
    }
    
    /**
     * Create mock resource version (from)
     */
    private createMockResourceVersion(id?: string): ResourceVersion {
        const versionId = id || `version_${this.generateId()}`;
        const now = new Date().toISOString();
        
        return {
            __typename: "ResourceVersion",
            id: versionId,
            createdAt: now,
            updatedAt: now,
            commentsCount: 0,
            forksCount: 0,
            isDeleted: false,
            isLatest: true,
            isPrivate: false,
            reportsCount: 0,
            versionIndex: 1,
            versionLabel: "1.0.0",
            comments: [],
            translations: [],
            translationsCount: 0,
            root: {
                __typename: "Resource",
                id: `resource_${versionId}`,
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
                publicId: `resource_public_${versionId}`,
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
                versionsCount: 1,
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
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                },
            },
            pullRequest: null,
            publicId: `pub_res_ver_${versionId}`,
            complexity: 1,
            forks: [],
            isComplete: false,
            relatedVersions: [],
            reports: [],
            timesCompleted: 0,
            timesStarted: 0,
            you: {
                __typename: "ResourceVersionYou",
                canBookmark: true,
                canComment: false,
                canCopy: true,
                canDelete: false,
                canReact: true,
                canRead: true,
                canReport: false,
                canRun: true,
                canUpdate: false,
            },
        };
    }
    
    /**
     * Create mock resource (to)
     */
    private createMockResource(id?: string): Resource {
        const resourceId = id || `resource_${this.generateId()}`;
        const now = new Date().toISOString();
        
        return {
            __typename: "Resource",
            id: resourceId,
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
            publicId: `resource_public_${resourceId}`,
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
            versionsCount: 1,
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
                isBookmarked: false,
                isViewed: false,
                reaction: null,
            },
        };
    }
    
    /**
     * Create mock pull request data
     */
    createMockPullRequest(overrides?: Partial<PullRequest>): PullRequest {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultPullRequest: PullRequest = {
            __typename: "PullRequest",
            id,
            createdAt: now,
            updatedAt: now,
            publicId: this.generatePublicId(),
            status: PullRequestStatusEnum.Open,
            closedAt: null,
            createdBy: this.createMockUser(),
            from: this.createMockResourceVersion(),
            to: this.createMockResource(),
            comments: [],
            commentsCount: 0,
            translations: [{
                __typename: "CommentTranslation",
                id: `trans_${id}`,
                language: "en",
                text: "This pull request adds a new feature to improve user experience.",
            }],
            translationsCount: 1,
            you: {
                __typename: "PullRequestYou",
                canComment: true,
                canDelete: false,
                canReport: false,
                canUpdate: false,
            },
        };
        
        return {
            ...defaultPullRequest,
            ...overrides,
        };
    }
    
    /**
     * Create pull request from API input
     */
    createPullRequestFromInput(input: PullRequestCreateInput): PullRequest {
        const pullRequest = this.createMockPullRequest();
        
        // Update pull request based on input
        pullRequest.id = input.id;
        pullRequest.from.id = input.fromConnect;
        pullRequest.to.id = input.toConnect;
        
        if (input.translationsCreate && input.translationsCreate.length > 0) {
            pullRequest.translations = input.translationsCreate.map(trans => ({
                __typename: "CommentTranslation" as const,
                id: trans.id,
                language: trans.language,
                text: trans.text,
            }));
            pullRequest.translationsCount = input.translationsCreate.length;
        }
        
        return pullRequest;
    }
    
    /**
     * Create pull requests for different statuses
     */
    createPullRequestsForAllStatuses(): PullRequest[] {
        return Object.values(PullRequestStatusEnum).map(status => 
            this.createMockPullRequest({
                status,
                publicId: `${String(status).toUpperCase()}-${this.generatePublicId()}`,
                closedAt: [PullRequestStatusEnum.Merged, PullRequestStatusEnum.Rejected, PullRequestStatusEnum.Canceled].includes(status) 
                    ? new Date().toISOString() 
                    : null,
            }),
        );
    }
    
    /**
     * Create pull request with specific workflow state
     */
    createWorkflowPullRequest(workflow: "draft" | "review" | "approved" | "merged" | "rejected"): PullRequest {
        const baseData = this.createMockPullRequest();
        
        switch (workflow) {
            case "draft":
                return {
                    ...baseData,
                    status: PullRequestStatusEnum.Draft,
                    commentsCount: 0,
                    you: {
                        ...baseData.you,
                        canUpdate: true,
                        canDelete: true,
                    },
                };
            
            case "review":
                return {
                    ...baseData,
                    status: PullRequestStatusEnum.Open,
                    commentsCount: 2,
                    you: {
                        ...baseData.you,
                        canComment: true,
                        canUpdate: false,
                    },
                };
            
            case "approved":
                return {
                    ...baseData,
                    status: PullRequestStatusEnum.Open,
                    commentsCount: 3,
                    you: {
                        ...baseData.you,
                        canComment: true,
                        canUpdate: false,
                    },
                };
            
            case "merged":
                return {
                    ...baseData,
                    status: PullRequestStatusEnum.Merged,
                    closedAt: new Date().toISOString(),
                    commentsCount: 4,
                    you: {
                        ...baseData.you,
                        canComment: false,
                        canUpdate: false,
                        canDelete: false,
                    },
                };
            
            case "rejected":
                return {
                    ...baseData,
                    status: PullRequestStatusEnum.Rejected,
                    closedAt: new Date().toISOString(),
                    commentsCount: 2,
                    you: {
                        ...baseData.you,
                        canComment: false,
                        canUpdate: false,
                        canDelete: false,
                    },
                };
            
            default:
                return baseData;
        }
    }
    
    /**
     * Validate pull request create input
     */
    async validateCreateInput(input: PullRequestCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await pullRequestValidation.create({}).validate(input);
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
    
    /**
     * Validate pull request update input
     */
    async validateUpdateInput(input: PullRequestUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await pullRequestValidation.update({}).validate(input);
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
 * MSW handlers factory for pull request endpoints
 */
export class PullRequestMSWHandlers {
    private responseFactory: PullRequestResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new PullRequestResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all pull request endpoints
     */
    createSuccessHandlers() {
        return [
            // Create pull request
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request`, async ({ request }) => {
                const body = await request.json() as PullRequestCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 },
                    );
                }
                
                // Create pull request
                const pullRequest = this.responseFactory.createPullRequestFromInput(body);
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get pull request by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/pull-request/:id`, ({ params }) => {
                const { id } = params;
                
                const pullRequest = this.responseFactory.createMockPullRequest({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                return HttpResponse.json(response);
            }),
            
            // Update pull request
            http.put(`${this.responseFactory["baseUrl"]}/api/pull-request/:id`, async ({ params, request }) => {
                const { id } = params;
                const body = await request.json() as PullRequestUpdateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateUpdateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 },
                    );
                }
                
                const pullRequest = this.responseFactory.createMockPullRequest({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                    status: body.status || PullRequestStatusEnum.Open,
                });
                
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                return HttpResponse.json(response);
            }),
            
            // Delete pull request
            http.delete(`${this.responseFactory["baseUrl"]}/api/pull-request/:id`, () => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // Merge pull request
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request/:id/merge`, ({ params }) => {
                const { id } = params;
                
                const pullRequest = this.responseFactory.createMockPullRequest({ 
                    id: id as string,
                    status: PullRequestStatusEnum.Merged,
                    closedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                return HttpResponse.json(response);
            }),
            
            // Reject pull request
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request/:id/reject`, ({ params }) => {
                const { id } = params;
                
                const pullRequest = this.responseFactory.createMockPullRequest({ 
                    id: id as string,
                    status: PullRequestStatusEnum.Rejected,
                    closedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                return HttpResponse.json(response);
            }),
            
            // List pull requests
            http.get(`${this.responseFactory["baseUrl"]}/api/pull-request`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const status = url.searchParams.get("status") as PullRequestStatus;
                const toId = url.searchParams.get("toId");
                
                let pullRequests = this.responseFactory.createPullRequestsForAllStatuses();
                
                // Filter by status if specified
                if (status) {
                    pullRequests = pullRequests.filter(pr => pr.status === status);
                }
                
                // Filter by target resource if specified
                if (toId) {
                    pullRequests = pullRequests.filter(pr => pr.to.id === toId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedPullRequests = pullRequests.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createPullRequestListResponse(
                    paginatedPullRequests,
                    {
                        page,
                        pageSize: limit,
                        totalCount: pullRequests.length,
                    },
                );
                
                return HttpResponse.json(response);
            }),
        ];
    }
    
    /**
     * Create error handlers for testing error scenarios
     */
    createErrorHandlers() {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request`, () => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        fromConnect: "Source resource version ID is required",
                        toConnect: "Target resource ID is required",
                        toObjectType: "Target object type must be specified",
                    }),
                    { status: 400 },
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/pull-request/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 },
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request`, () => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 },
                );
            }),
            
            // Merge conflict error
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request/:id/merge`, () => {
                return HttpResponse.json(
                    this.responseFactory.createConflictErrorResponse(),
                    { status: 409 },
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request`, () => {
                return HttpResponse.json(
                    this.responseFactory.createServerErrorResponse(),
                    { status: 500 },
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000) {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request`, async ({ request }) => {
                const body = await request.json() as PullRequestCreateInput;
                const pullRequest = this.responseFactory.createPullRequestFromInput(body);
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response, { status: 201 });
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request/:id/merge`, async ({ params }) => {
                const { id } = params;
                
                const pullRequest = this.responseFactory.createMockPullRequest({ 
                    id: id as string,
                    status: PullRequestStatusEnum.Merged,
                    closedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(pullRequest);
                
                await new Promise(resolve => setTimeout(resolve, delay));
                return HttpResponse.json(response);
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers() {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request`, () => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/pull-request/:id`, () => {
                return HttpResponse.error();
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/pull-request/:id/merge`, () => {
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
    }) {
        const { endpoint, method, status, response, delay } = config;
        const fullEndpoint = `${this.responseFactory["baseUrl"]}${endpoint}`;
        
        switch (method) {
            case "GET":
                return http.get(fullEndpoint, async () => {
                    if (delay) await new Promise(resolve => setTimeout(resolve, delay));
                    return HttpResponse.json(response, { status });
                });
            case "POST":
                return http.post(fullEndpoint, async () => {
                    if (delay) await new Promise(resolve => setTimeout(resolve, delay));
                    return HttpResponse.json(response, { status });
                });
            case "PUT":
                return http.put(fullEndpoint, async () => {
                    if (delay) await new Promise(resolve => setTimeout(resolve, delay));
                    return HttpResponse.json(response, { status });
                });
            case "DELETE":
                return http.delete(fullEndpoint, async () => {
                    if (delay) await new Promise(resolve => setTimeout(resolve, delay));
                    return HttpResponse.json(response, { status });
                });
            default:
                throw new Error(`Unsupported method: ${method}`);
        }
    }
}

/**
 * Pre-configured response scenarios
 */
export const pullRequestResponseScenarios = {
    // Success scenarios
    createSuccess: (pullRequest?: PullRequest) => {
        const factory = new PullRequestResponseFactory();
        return factory.createSuccessResponse(
            pullRequest || factory.createMockPullRequest(),
        );
    },
    
    listSuccess: (pullRequests?: PullRequest[]) => {
        const factory = new PullRequestResponseFactory();
        return factory.createPullRequestListResponse(
            pullRequests || factory.createPullRequestsForAllStatuses(),
        );
    },
    
    draftPullRequest: () => {
        const factory = new PullRequestResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("draft"),
        );
    },
    
    reviewPullRequest: () => {
        const factory = new PullRequestResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("review"),
        );
    },
    
    mergedPullRequest: () => {
        const factory = new PullRequestResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("merged"),
        );
    },
    
    rejectedPullRequest: () => {
        const factory = new PullRequestResponseFactory();
        return factory.createSuccessResponse(
            factory.createWorkflowPullRequest("rejected"),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new PullRequestResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                fromConnect: "Source resource version is required",
                toConnect: "Target resource is required",
                toObjectType: "Target object type must be specified",
            },
        );
    },
    
    notFoundError: (pullRequestId?: string) => {
        const factory = new PullRequestResponseFactory();
        return factory.createNotFoundErrorResponse(
            pullRequestId || "non-existent-pr-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new PullRequestResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    mergeConflictError: (reason?: string) => {
        const factory = new PullRequestResponseFactory();
        return factory.createConflictErrorResponse(reason);
    },
    
    serverError: () => {
        const factory = new PullRequestResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new PullRequestMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new PullRequestMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new PullRequestMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new PullRequestMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const pullRequestResponseFactory = new PullRequestResponseFactory();
export const pullRequestMSWHandlers = new PullRequestMSWHandlers();
