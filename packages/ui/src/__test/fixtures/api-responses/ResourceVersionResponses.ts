/**
 * ResourceVersion API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource version endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */
// AI_CHECK: TYPE_SAFETY=fixed-resource-properties | LAST: 2025-07-02 - Fixed Resource properties and removed invalid 'link' and 'usedFor' fields

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    ResourceVersion, 
    ResourceVersionCreateInput, 
    ResourceVersionUpdateInput,
    ResourceUsedFor,
} from "@vrooli/shared";
import { 
    resourceVersionValidation,
    ResourceUsedFor as ResourceUsedForEnum, 
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
 * ResourceVersion API response factory
 */
export class ResourceVersionResponseFactory {
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
     * Generate unique resource version ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
    }
    
    /**
     * Create successful resource version response
     */
    createSuccessResponse(resourceVersion: ResourceVersion): APIResponse<ResourceVersion> {
        return {
            data: resourceVersion,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/resource-version/${resourceVersion.id}`,
                    related: {
                        resource: `${this.baseUrl}/api/resource/${resourceVersion.root?.id}`,
                        translations: `${this.baseUrl}/api/resource-version/${resourceVersion.id}/translations`,
                    },
                },
            },
        };
    }
    
    /**
     * Create resource version list response
     */
    createResourceVersionListResponse(resourceVersions: ResourceVersion[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ResourceVersion> {
        const paginationData = pagination || {
            page: 1,
            pageSize: resourceVersions.length,
            totalCount: resourceVersions.length,
        };
        
        return {
            data: resourceVersions,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/resource-version?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/resource-version",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(resourceVersionId: string): APIErrorResponse {
        return {
            error: {
                code: "RESOURCE_VERSION_NOT_FOUND",
                message: `Resource version with ID '${resourceVersionId}' was not found`,
                details: {
                    resourceVersionId,
                    searchCriteria: { id: resourceVersionId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/resource-version/${resourceVersionId}`,
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
                message: `You do not have permission to ${operation} this resource version`,
                details: {
                    operation,
                    requiredPermissions: ["resource:write"],
                    userPermissions: ["resource:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/resource-version",
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
                path: "/api/resource-version",
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
                path: "/api/resource-version",
            },
        };
    }
    
    /**
     * Create mock resource version data
     */
    createMockResourceVersion(overrides?: Partial<ResourceVersion>): ResourceVersion {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultResourceVersion: ResourceVersion = {
            __typename: "ResourceVersion",
            id,
            createdAt: now,
            updatedAt: now,
            versionLabel: "1.0.0",
            isDeleted: false,
            isLatest: true,
            isPrivate: false,
            translations: [{
                __typename: "ResourceVersionTranslation",
                id: `trans_${id}`,
                language: "en",
                name: "Test Resource Version",
                description: "A test resource version for unit testing",
            }],
            translationsCount: 1,
            comments: [],
            commentsCount: 0,
            complexity: 1,
            forks: [],
            forksCount: 0,
            isComplete: false,
            pullRequest: null,
            reports: [],
            reportsCount: 0,
            versionIndex: 1,
            root: {
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
                versionsCount: 1,
                views: 0,
                you: {
                    __typename: "ResourceYou",
                    canBookmark: true,
                    canComment: true,
                    canDelete: true,
                    canReact: true,
                    canRead: true,
                    canTransfer: false,
                    canUpdate: true,
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                },
            },
            publicId: `pub_resource_ver_${id}`,
            relatedVersions: [],
            timesCompleted: 0,
            timesStarted: 0,
            you: {
                __typename: "ResourceVersionYou",
                canBookmark: true,
                canComment: true,
                canCopy: true,
                canDelete: true,
                canReact: true,
                canRead: true,
                canReport: false,
                canRun: true,
                canUpdate: true,
            },
        };
        
        return {
            ...defaultResourceVersion,
            ...overrides,
        };
    }
    
    /**
     * Create resource version from API input
     */
    createResourceVersionFromInput(input: ResourceVersionCreateInput): ResourceVersion {
        const resourceVersion = this.createMockResourceVersion();
        
        // Update resource version based on input
        if (input.versionLabel) {
            resourceVersion.versionLabel = input.versionLabel;
        }
        
        
        if (input.isPrivate !== undefined) {
            resourceVersion.isPrivate = input.isPrivate;
        }
        
        return resourceVersion;
    }
    
    /**
     * Create resource versions for different purposes
     */
    createResourceVersionsForAllPurposes(): ResourceVersion[] {
        return Object.values(ResourceUsedForEnum).map(usedFor => 
            this.createMockResourceVersion({
                versionLabel: `${usedFor}-1.0.0`,
            }),
        );
    }
    
    /**
     * Validate resource version create input
     */
    async validateCreateInput(input: ResourceVersionCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await resourceVersionValidation.create({}).validate(input);
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
 * MSW handlers factory for resource version endpoints
 */
export class ResourceVersionMSWHandlers {
    private responseFactory: ResourceVersionResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ResourceVersionResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all resource version endpoints
     */
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create resource version
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version`, async ({ request }) => {
                const body = await request.json() as ResourceVersionCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}), { status: 400 });
                }
                
                // Create resource version
                const resourceVersion = this.responseFactory.createResourceVersionFromInput(body);
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get resource version by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version/:id`, ({ params }) => {
                const { id } = params;
                
                const resourceVersion = this.responseFactory.createMockResourceVersion({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update resource version
            http.put(`${this.responseFactory["baseUrl"]}/api/resource-version/:id`, async ({ request, params }) => {
                const { id } = params;
                const body = await request.json() as ResourceVersionUpdateInput;
                
                const resourceVersion = this.responseFactory.createMockResourceVersion({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete resource version
            http.delete(`${this.responseFactory["baseUrl"]}/api/resource-version/:id`, ({ params }) => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List resource versions
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const usedFor = url.searchParams.get("usedFor") as ResourceUsedFor;
                
                let resourceVersions = this.responseFactory.createResourceVersionsForAllPurposes();
                
                // Filter by used for if specified (usedFor filter removed since property doesn't exist)
                if (usedFor) {
                    // No-op: usedFor property doesn't exist on ResourceVersion
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedResourceVersions = resourceVersions.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createResourceVersionListResponse(
                    paginatedResourceVersions,
                    {
                        page,
                        pageSize: limit,
                        totalCount: resourceVersions.length,
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
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version`, ({ request }) => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        link: "A valid URL is required",
                        versionLabel: "Version label is required",
                    }), { status: 400 });
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string), { status: 404 });
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version`, ({ request }) => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"), { status: 403 });
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version`, ({ request }) => {
                return HttpResponse.json(
                    this.responseFactory.createServerErrorResponse(), { status: 500 });
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version`, async ({ request }) => {
                const body = await request.json() as ResourceVersionCreateInput;
                const resourceVersion = this.responseFactory.createResourceVersionFromInput(body);
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return HttpResponse.json(response, { status: 201 });
            }),
        ];
    }
    
    /**
     * Create network error handlers
     */
    createNetworkErrorHandlers(): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version`, ({ request }) => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version/:id`, ({ params }) => {
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
        
        return http[method.toLowerCase() as keyof typeof http](fullEndpoint, async ({ request, params }) => {
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
export const resourceVersionResponseScenarios = {
    // Success scenarios
    createSuccess: (resourceVersion?: ResourceVersion) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createSuccessResponse(
            resourceVersion || factory.createMockResourceVersion(),
        );
    },
    
    listSuccess: (resourceVersions?: ResourceVersion[]) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createResourceVersionListResponse(
            resourceVersions || factory.createResourceVersionsForAllPurposes(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                link: "A valid URL is required",
                versionLabel: "Version label is required",
            },
        );
    },
    
    notFoundError: (resourceVersionId?: string) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceVersionId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ResourceVersionMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ResourceVersionMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ResourceVersionMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ResourceVersionMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const resourceVersionResponseFactory = new ResourceVersionResponseFactory();
export const resourceVersionMSWHandlers = new ResourceVersionMSWHandlers();
