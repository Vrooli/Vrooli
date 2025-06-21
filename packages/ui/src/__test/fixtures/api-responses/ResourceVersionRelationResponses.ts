/**
 * ResourceVersionRelation API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource version relation endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { rest, type RestHandler } from 'msw';
import type { 
    ResourceVersionRelation, 
    ResourceVersionRelationCreateInput, 
    ResourceVersionRelationUpdateInput
} from '@vrooli/shared';
import { 
    resourceVersionRelationValidation
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
 * ResourceVersionRelation API response factory
 */
export class ResourceVersionRelationResponseFactory {
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
     * Generate unique resource version relation ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful resource version relation response
     */
    createSuccessResponse(relation: ResourceVersionRelation): APIResponse<ResourceVersionRelation> {
        return {
            data: relation,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/resource-version-relation/${relation.id}`,
                    related: {
                        from: `${this.baseUrl}/api/resource-version/${relation.from.id}`,
                        to: `${this.baseUrl}/api/resource-version/${relation.to.id}`
                    }
                }
            }
        };
    }
    
    /**
     * Create resource version relation list response
     */
    createRelationListResponse(relations: ResourceVersionRelation[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ResourceVersionRelation> {
        const paginationData = pagination || {
            page: 1,
            pageSize: relations.length,
            totalCount: relations.length
        };
        
        return {
            data: relations,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0',
                links: {
                    self: `${this.baseUrl}/api/resource-version-relation?page=${paginationData.page}&limit=${paginationData.pageSize}`
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
                path: '/api/resource-version-relation'
            }
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(relationId: string): APIErrorResponse {
        return {
            error: {
                code: 'RELATION_NOT_FOUND',
                message: `Resource version relation with ID '${relationId}' was not found`,
                details: {
                    relationId,
                    searchCriteria: { id: relationId }
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/resource-version-relation/${relationId}`
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
                message: `You do not have permission to ${operation} this relation`,
                details: {
                    operation,
                    requiredPermissions: ['resource:write'],
                    userPermissions: ['resource:read']
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: '/api/resource-version-relation'
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
                path: '/api/resource-version-relation'
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
                path: '/api/resource-version-relation'
            }
        };
    }
    
    /**
     * Create mock resource version relation data
     */
    createMockRelation(overrides?: Partial<ResourceVersionRelation>): ResourceVersionRelation {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultRelation: ResourceVersionRelation = {
            __typename: "ResourceVersionRelation",
            id,
            created_at: now,
            updated_at: now,
            from: {
                __typename: "ResourceVersion",
                id: `from_${id}`,
                created_at: now,
                updated_at: now,
                versionLabel: "1.0.0",
                link: `https://example.com/resource/from/${id}`,
                title: "Source Resource",
                description: "The source resource in this relation",
                isLatest: true,
                isPrivate: false,
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "VersionYou",
                    canComment: true,
                    canDelete: true,
                    canRead: true,
                    canReport: false,
                    canUpdate: true
                }
            },
            to: {
                __typename: "ResourceVersion",
                id: `to_${id}`,
                created_at: now,
                updated_at: now,
                versionLabel: "1.0.0",
                link: `https://example.com/resource/to/${id}`,
                title: "Target Resource",
                description: "The target resource in this relation",
                isLatest: true,
                isPrivate: false,
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "VersionYou",
                    canComment: true,
                    canDelete: true,
                    canRead: true,
                    canReport: false,
                    canUpdate: true
                }
            }
        };
        
        return {
            ...defaultRelation,
            ...overrides
        };
    }
    
    /**
     * Create resource version relation from API input
     */
    createRelationFromInput(input: ResourceVersionRelationCreateInput): ResourceVersionRelation {
        const relation = this.createMockRelation();
        
        // Update relation based on input
        if (input.fromConnect) {
            relation.from.id = input.fromConnect;
        }
        
        if (input.toConnect) {
            relation.to.id = input.toConnect;
        }
        
        return relation;
    }
    
    /**
     * Create multiple relations for testing
     */
    createMultipleRelations(count: number = 5): ResourceVersionRelation[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockRelation({
                id: `relation_${index}_${this.generateId()}`,
                from: {
                    ...this.createMockRelation().from,
                    id: `from_${index}_${this.generateId()}`,
                    title: `Source Resource ${index + 1}`
                },
                to: {
                    ...this.createMockRelation().to,
                    id: `to_${index}_${this.generateId()}`,
                    title: `Target Resource ${index + 1}`
                }
            })
        );
    }
    
    /**
     * Validate resource version relation create input
     */
    async validateCreateInput(input: ResourceVersionRelationCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await resourceVersionRelationValidation.create.validate(input);
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
 * MSW handlers factory for resource version relation endpoints
 */
export class ResourceVersionRelationMSWHandlers {
    private responseFactory: ResourceVersionRelationResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ResourceVersionRelationResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all resource version relation endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create resource version relation
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, async (req, res, ctx) => {
                const body = await req.json() as ResourceVersionRelationCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}))
                    );
                }
                
                // Create relation
                const relation = this.responseFactory.createRelationFromInput(body);
                const response = this.responseFactory.createSuccessResponse(relation);
                
                return res(
                    ctx.status(201),
                    ctx.json(response)
                );
            }),
            
            // Get resource version relation by ID
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version-relation/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const relation = this.responseFactory.createMockRelation({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(relation);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Update resource version relation
            rest.put(`${this.responseFactory['baseUrl']}/api/resource-version-relation/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ResourceVersionRelationUpdateInput;
                
                const relation = this.responseFactory.createMockRelation({ 
                    id: id as string,
                    updated_at: new Date().toISOString()
                });
                
                const response = this.responseFactory.createSuccessResponse(relation);
                
                return res(
                    ctx.status(200),
                    ctx.json(response)
                );
            }),
            
            // Delete resource version relation
            rest.delete(`${this.responseFactory['baseUrl']}/api/resource-version-relation/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List resource version relations
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get('page') || '1');
                const limit = parseInt(url.searchParams.get('limit') || '10');
                const fromId = url.searchParams.get('fromId');
                const toId = url.searchParams.get('toId');
                
                let relations = this.responseFactory.createMultipleRelations(20);
                
                // Filter by fromId if specified
                if (fromId) {
                    relations = relations.filter(r => r.from.id === fromId);
                }
                
                // Filter by toId if specified
                if (toId) {
                    relations = relations.filter(r => r.to.id === toId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedRelations = relations.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createRelationListResponse(
                    paginatedRelations,
                    {
                        page,
                        pageSize: limit,
                        totalCount: relations.length
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
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        fromConnect: 'Source resource version is required',
                        toConnect: 'Target resource version is required'
                    }))
                );
            }),
            
            // Not found error
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version-relation/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string))
                );
            }),
            
            // Permission error
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse('create'))
                );
            }),
            
            // Server error
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, (req, res, ctx) => {
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
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, async (req, res, ctx) => {
                const body = await req.json() as ResourceVersionRelationCreateInput;
                const relation = this.responseFactory.createRelationFromInput(body);
                const response = this.responseFactory.createSuccessResponse(relation);
                
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
            rest.post(`${this.responseFactory['baseUrl']}/api/resource-version-relation`, (req, res, ctx) => {
                return res.networkError('Network connection failed');
            }),
            
            rest.get(`${this.responseFactory['baseUrl']}/api/resource-version-relation/:id`, (req, res, ctx) => {
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
export const resourceVersionRelationResponseScenarios = {
    // Success scenarios
    createSuccess: (relation?: ResourceVersionRelation) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createSuccessResponse(
            relation || factory.createMockRelation()
        );
    },
    
    listSuccess: (relations?: ResourceVersionRelation[]) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createRelationListResponse(
            relations || factory.createMultipleRelations()
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                fromConnect: 'Source resource version is required',
                toConnect: 'Target resource version is required'
            }
        );
    },
    
    notFoundError: (relationId?: string) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createNotFoundErrorResponse(
            relationId || 'non-existent-id'
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || 'create'
        );
    },
    
    serverError: () => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new ResourceVersionRelationMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new ResourceVersionRelationMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new ResourceVersionRelationMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ResourceVersionRelationMSWHandlers().createNetworkErrorHandlers()
};

// Export factory instances for easy use
export const resourceVersionRelationResponseFactory = new ResourceVersionRelationResponseFactory();
export const resourceVersionRelationMSWHandlers = new ResourceVersionRelationMSWHandlers();