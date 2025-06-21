/**
 * ResourceVersion API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource version endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    ResourceVersion, 
    ResourceVersionCreateInput, 
    ResourceVersionUpdateInput,
    ResourceUsedFor,
    ResourceList
} from '@vrooli/shared';
import { 
    resourceVersionValidation,
    ResourceUsedFor as ResourceUsedForEnum 
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
 * ResourceVersion API response factory
 */
export class ResourceVersionResponseFactory {
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
     * Generate unique resource version ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
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
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/resource-version/${resourceVersion.id}`,
                    related: {
                        resource: `${this.baseUrl}/api/resource/${resourceVersion.root?.id}`,
                        translations: `${this.baseUrl}/api/resource-version/${resourceVersion.id}/translations`
                    }
                }
            }
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
            totalCount: resourceVersions.length
        };
        
        return {
            data: resourceVersions,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/resource-version?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/resource-version'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(resourceVersionId: string): APIErrorResponse {
        return {
            error: {
                code: 'RESOURCE_VERSION_NOT_FOUND',
                message: `Resource version with ID '${resourceVersionId}' was not found`,
                details: {
                    resourceVersionId,
                    searchCriteria: { id: resourceVersionId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/resource-version/${resourceVersionId}`
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
                message: `You do not have permission to ${operation} this resource version`,
                details: {
                    operation,
                    requiredPermissions: ['resource:write'],
                    userPermissions: ['resource:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/resource-version'
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
                path: '/api/resource-version'
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
                path: '/api/resource-version'
            }
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
            created_at: now,
            updated_at: now,
            versionLabel: "1.0.0",
            link: `https://example.com/resource/${id}`,
            usedFor: ResourceUsedForEnum.Display,
            title: "Test Resource Version",
            description: "A test resource version for unit testing",
            isLatest: true,
            isPrivate: false,
            translations: [{
                __typename: "ResourceVersionTranslation",
                id: `trans_${id}`,
                language: "en",
                title: "Test Resource Version",
                description: "A test resource version for unit testing"
            }],
            translationsCount: 1,
            resourceList: {
                __typename: "ResourceList",
                id: `list_${id}`,
                created_at: now,
                updated_at: now,
                listFor: {
                    __typename: "ApiVersion",
                    id: `api_${id}`,
                    versionLabel: "1.0.0",
                    created_at: now,
                    updated_at: now
                },
                resources: [],
                translations: []
            },
            root: {
                __typename: "Resource",
                id: `resource_${id}`,
                created_at: now,
                updated_at: now,
                isInternal: false,
                isPrivate: false,
                usedBy: [],
                usedByCount: 0,
                versions: [],
                versionsCount: 1,
                you: {
                    __typename: "ResourceYou",
                    canDelete: true,
                    canUpdate: true,
                    canReport: false,
                    isBookmarked: false,
                    isReacted: false,
                    reaction: null
                }
            },
            you: {
                __typename: "VersionYou",
                canComment: true,
                canDelete: true,
                canRead: true,
                canReport: false,
                canUpdate: true
            }
        };
        
        return {
            ...defaultResourceVersion,
            ...overrides
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
        
        if (input.link) {
            resourceVersion.link = input.link;
        }
        
        if (input.usedFor) {
            resourceVersion.usedFor = input.usedFor;
        }
        
        if (input.isPrivate !== undefined) {
            resourceVersion.isPrivate = input.isPrivate;
        }
        
        if (input.translations?.create) {
            resourceVersion.translations = input.translations.create.map(trans => ({
                __typename: "ResourceVersionTranslation" as const,
                id: `trans_${this.generateId()}`,
                language: trans.language,
                title: trans.title || "",
                description: trans.description || ""
            }));
            resourceVersion.translationsCount = resourceVersion.translations.length;
        }
        
        return resourceVersion;
    }
    
    /**
     * Create resource versions for different purposes
     */
    createResourceVersionsForAllPurposes(): ResourceVersion[] {
        return Object.values(ResourceUsedForEnum).map(usedFor => 
            this.createMockResourceVersion({
                usedFor,
                title: `${usedFor} Resource`,
                description: `A resource used for ${usedFor.toLowerCase()}`
            })
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
            await resourceVersionValidation.create.validate(input);
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
                errors: fieldErrors
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
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create resource version
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version`, async (req, res, ctx) => {
                const body = await req.json() as ResourceVersionCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}))
                    );
                }
                
                // Create resource version
                const resourceVersion = this.responseFactory.createResourceVersionFromInput(body);
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return res(
                    ctx.status(201),
                    ctx.json(response)
                );
            }),
            
            // Get resource version by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const resourceVersion = this.responseFactory.createMockResourceVersion({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Update resource version
            rest.put(`${this.responseFactory['baseUrl']}/api/resource-version/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ResourceVersionUpdateInput;
                
                const resourceVersion = this.responseFactory.createMockResourceVersion({ 
                    id: id as string,
                    updated_at: new Date().toISOString()
                });
                
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Delete resource version
            rest.delete(`${this.responseFactory['baseUrl']}/api/resource-version/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List resource versions
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                const usedFor = url.searchParams.get('usedFor') as ResourceUsedFor;
                
                let resourceVersions = this.responseFactory.createResourceVersionsForAllPurposes();
                
                // Filter by used for if specified
                if (usedFor) {
                    resourceVersions = resourceVersions.filter(rv => rv.usedFor === usedFor);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedResourceVersions = resourceVersions.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createResourceVersionListResponse(
                    paginatedResourceVersions,
                    {
                        page,
                        pageSize: limit,
                        totalCount: resourceVersions.length
                    }
                );
                
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
            // Validation error
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        link: 'A valid URL is required',
                        versionLabel: 'Version label is required'
                    }))
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('create'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version`, (req, res, ctx) => {
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
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version`, async (req, res, ctx) => {
                const body = await req.json() as ResourceVersionCreateInput;
                const resourceVersion = this.responseFactory.createResourceVersionFromInput(body);
                const response = this.responseFactory.createSuccessResponse(resourceVersion);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
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
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version`, (req, res, ctx) => {
                return res.networkError('Network connection failed');
            }),
            
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version/:id`, (req, res, ctx) => {
                return res.networkError('Connection timeout');
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
export const resourceVersionResponseScenarios = {
    // Success scenarios
    createSuccess: (resourceVersion?: ResourceVersion) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createSuccessResponse(
            resourceVersion || factory.createMockResourceVersion()
        );
    },
    
    listSuccess: (resourceVersions?: ResourceVersion[]) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createResourceVersionListResponse(
            resourceVersions || factory.createResourceVersionsForAllPurposes()
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                link: 'A valid URL is required',
                versionLabel: 'Version label is required'
            }
        );
    },
    
    notFoundError: (resourceVersionId?: string) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createNotFoundErrorResponse(
            resourceVersionId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ResourceVersionResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'create'
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
    networkErrorHandlers: () => new ResourceVersionMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const resourceVersionResponseFactory = new ResourceVersionResponseFactory();
export const resourceVersionMSWHandlers = new ResourceVersionMSWHandlers();