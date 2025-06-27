/**
 * Resource API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    Resource, 
    ResourceCreateInput, 
    ResourceUpdateInput,
    ResourceShape,
    ResourceUsedFor, 
} from "@vrooli/shared";
import { 
    resourceValidation,
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
 * Resource API response factory
 */
export class ResourceResponseFactory {
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
     * Create successful resource response
     */
    createSuccessResponse(resource: Resource): APIResponse<Resource> {
        return {
            data: resource,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/resource/${resource.id}`,
                    related: {
                        versions: `${this.baseUrl}/api/resource/${resource.id}/versions`,
                        usedBy: `${this.baseUrl}/api/resource/${resource.id}/used-by`,
                    },
                },
            },
        };
    }
    
    /**
     * Create resource list response
     */
    createResourceListResponse(resources: Resource[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Resource> {
        const paginationData = pagination || {
            page: 1,
            pageSize: resources.length,
            totalCount: resources.length,
        };
        
        return {
            data: resources,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/resource?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/resource",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(resourceId: string): APIErrorResponse {
        return {
            error: {
                code: "RESOURCE_NOT_FOUND",
                message: `Resource with ID '${resourceId}' was not found`,
                details: {
                    resourceId,
                    searchCriteria: { id: resourceId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/resource/${resourceId}`,
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
                message: `You do not have permission to ${operation} this resource`,
                details: {
                    operation,
                    requiredPermissions: ["resource:write"],
                    userPermissions: ["resource:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/resource",
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
                path: "/api/resource",
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
                path: "/api/resource",
            },
        };
    }
    
    /**
     * Create mock resource data
     */
    createMockResource(overrides?: Partial<Resource>): Resource {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultResource: Resource = {
            __typename: "Resource",
            id,
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
                canDelete: true,
                canUpdate: true,
                canReport: false,
                isBookmarked: false,
                isReacted: false,
                reaction: null,
            },
        };
        
        return {
            ...defaultResource,
            ...overrides,
        };
    }
    
    /**
     * Create resource from API input
     */
    createResourceFromInput(input: ResourceCreateInput): Resource {
        const resource = this.createMockResource();
        
        // Update resource based on input
        if (input.isInternal !== undefined) {
            resource.isInternal = input.isInternal;
        }
        
        if (input.isPrivate !== undefined) {
            resource.isPrivate = input.isPrivate;
        }
        
        return resource;
    }
    
    /**
     * Create resources for different purposes
     */
    createResourcesForAllPurposes(): Resource[] {
        return Object.values(ResourceUsedForEnum).map(usedFor => 
            this.createMockResource({
                id: `resource_${usedFor.toLowerCase()}_${this.generateId()}`,
            }),
        );
    }
    
    /**
     * Validate resource create input
     */
    async validateCreateInput(input: ResourceCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await resourceValidation.create.validate(input);
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
 * MSW handlers factory for resource endpoints
 */
export class ResourceMSWHandlers {
    private responseFactory: ResourceResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ResourceResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all resource endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create resource
            http.post(`${this.responseFactory["baseUrl"]}/api/resource`, async (req, res, ctx) => {
                const body = await req.json() as ResourceCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create resource
                const resource = this.responseFactory.createResourceFromInput(body);
                const response = this.responseFactory.createSuccessResponse(resource);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get resource by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/resource/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const resource = this.responseFactory.createMockResource({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(resource);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update resource
            http.put(`${this.responseFactory["baseUrl"]}/api/resource/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ResourceUpdateInput;
                
                const resource = this.responseFactory.createMockResource({ 
                    id: id as string,
                    updatedAt: new Date().toISOString(),
                });
                
                const response = this.responseFactory.createSuccessResponse(resource);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete resource
            http.delete(`${this.responseFactory["baseUrl"]}/api/resource/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List resources
            http.get(`${this.responseFactory["baseUrl"]}/api/resource`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const isInternal = url.searchParams.get("isInternal") === "true";
                
                let resources = this.responseFactory.createResourcesForAllPurposes();
                
                // Filter by internal flag if specified
                if (isInternal !== null) {
                    resources = resources.filter(r => r.isInternal === isInternal);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedResources = resources.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createResourceListResponse(
                    paginatedResources,
                    {
                        page,
                        pageSize: limit,
                        totalCount: resources.length,
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
            http.post(`${this.responseFactory["baseUrl"]}/api/resource`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        versions: "At least one version is required",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/resource/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/resource`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/resource`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/resource`, async (req, res, ctx) => {
                const body = await req.json() as ResourceCreateInput;
                const resource = this.responseFactory.createResourceFromInput(body);
                const response = this.responseFactory.createSuccessResponse(resource);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/resource`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/resource/:id`, (req, res, ctx) => {
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
export const resourceResponseScenarios = {
    // Success scenarios
    createSuccess: (resource?: Resource) => {
        const factory = new ResourceResponseFactory();
        return factory.createSuccessResponse(
            resource || factory.createMockResource(),
        );
    },
    
    listSuccess: (resources?: Resource[]) => {
        const factory = new ResourceResponseFactory();
        return factory.createResourceListResponse(
            resources || factory.createResourcesForAllPurposes(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ResourceResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                versions: "At least one version is required",
            },
        );
    },
    
    notFoundError: (resourceId?: string) => {
        const factory = new ResourceResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ResourceResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new ResourceResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ResourceMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ResourceMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ResourceMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ResourceMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const resourceResponseFactory = new ResourceResponseFactory();
export const resourceMSWHandlers = new ResourceMSWHandlers();
