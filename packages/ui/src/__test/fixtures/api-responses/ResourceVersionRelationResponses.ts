/**
 * ResourceVersionRelation API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for resource version relation endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 * 
 * NOTE: Migrated to MSW v2 - uses HttpResponse.json() and new handler syntax
 */

import { http, HttpResponse, type RequestHandler } from "msw";
import type { 
    Resource,
    ResourceVersion,
    ResourceVersionRelation, 
    ResourceVersionRelationCreateInput, 
    ResourceVersionRelationUpdateInput,
} from "@vrooli/shared";
import { 
    resourceVersionRelationValidation,
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
 * ResourceVersionRelation API response factory
 */
export class ResourceVersionRelationResponseFactory {
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
     * Generate unique resource version relation ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substring(2, 11)}`;
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
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/resource-version-relation/${relation.id}`,
                    related: {
                        toVersion: `${this.baseUrl}/api/resource-version/${relation.toVersion.id}`,
                    },
                },
            },
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
            totalCount: relations.length,
        };
        
        return {
            data: relations,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/resource-version-relation?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/resource-version-relation",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(relationId: string): APIErrorResponse {
        return {
            error: {
                code: "RELATION_NOT_FOUND",
                message: `Resource version relation with ID '${relationId}' was not found`,
                details: {
                    relationId,
                    searchCriteria: { id: relationId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/resource-version-relation/${relationId}`,
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
                message: `You do not have permission to ${operation} this relation`,
                details: {
                    operation,
                    requiredPermissions: ["resource:write"],
                    userPermissions: ["resource:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/resource-version-relation",
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
                path: "/api/resource-version-relation",
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
                path: "/api/resource-version-relation",
            },
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
            toVersion: {
                __typename: "ResourceVersion",
                id: `to_${id}`,
                codeLanguage: null,
                comments: [],
                commentsCount: 0,
                completedAt: null,
                complexity: 1,
                config: null,
                createdAt: now,
                forks: [],
                forksCount: 0,
                isAutomatable: false,
                isComplete: true,
                isDeleted: false,
                isLatest: true,
                isPrivate: false,
                publicId: `pub_${id}`,
                pullRequest: null,
                relatedVersions: [],
                reports: [],
                reportsCount: 0,
                root: {} as ResourceVersion["root"], // Simplified for mock
                resourceSubType: null,
                timesCompleted: 0,
                timesStarted: 0,
                translations: [],
                translationsCount: 0,
                updatedAt: now,
                versionIndex: 1,
                versionLabel: "1.0.0",
                versionNotes: null,
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
            },
            labels: ["dependency", "upgrade"],
        };
        
        return {
            ...defaultRelation,
            ...overrides,
        };
    }
    
    /**
     * Create resource version relation from API input
     */
    createRelationFromInput(input: ResourceVersionRelationCreateInput): ResourceVersionRelation {
        const relation = this.createMockRelation();
        
        // Update relation based on input
        if (input.toVersionConnect) {
            relation.toVersion.id = input.toVersionConnect;
        }
        
        if (input.labels) {
            relation.labels = input.labels;
        }
        
        return relation;
    }
    
    /**
     * Create multiple relations for testing
     */
    createMultipleRelations(count = 10): ResourceVersionRelation[] {
        return Array.from({ length: count }, (_, index) => 
            this.createMockRelation({
                id: `relation_${index}_${this.generateId()}`,
                toVersion: {
                    ...this.createMockRelation().toVersion,
                    id: `to_${index}_${this.generateId()}`,
                    versionLabel: `${index + 1}.0.0`,
                },
                labels: [`label_${index + 1}`, "test"],
            }),
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
            await resourceVersionRelationValidation.create({}).validate(input);
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
    createSuccessHandlers(): RequestHandler[] {
        return [
            // Create resource version relation
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, async ({ request }) => {
                const body = await request.json() as ResourceVersionRelationCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return HttpResponse.json(
                        this.responseFactory.createValidationErrorResponse(validation.errors || {}),
                        { status: 400 },
                    );
                }
                
                // Create relation
                const relation = this.responseFactory.createRelationFromInput(body);
                const response = this.responseFactory.createSuccessResponse(relation);
                
                return HttpResponse.json(response, { status: 201 });
            }),
            
            // Get resource version relation by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version-relation/:id`, ({ params }) => {
                const { id } = params;
                
                const relation = this.responseFactory.createMockRelation({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(relation);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Update resource version relation
            http.put(`${this.responseFactory["baseUrl"]}/api/resource-version-relation/:id`, async ({ params, request }) => {
                const { id } = params;
                const _body = await request.json() as ResourceVersionRelationUpdateInput;
                
                const relation = this.responseFactory.createMockRelation({ 
                    id: id as string,
                });
                
                const response = this.responseFactory.createSuccessResponse(relation);
                
                return HttpResponse.json(response, { status: 200 });
            }),
            
            // Delete resource version relation
            http.delete(`${this.responseFactory["baseUrl"]}/api/resource-version-relation/:id`, () => {
                return new HttpResponse(null, { status: 204 });
            }),
            
            // List resource version relations
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, ({ request }) => {
                const url = new URL(request.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const _fromId = url.searchParams.get("fromId");
                const toId = url.searchParams.get("toId");
                
                let relations = this.responseFactory.createMultipleRelations(50);
                
                // Filter by toId if specified
                if (toId) {
                    relations = relations.filter(r => r.toVersion.id === toId);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedRelations = relations.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createRelationListResponse(
                    paginatedRelations,
                    {
                        page,
                        pageSize: limit,
                        totalCount: relations.length,
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
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, () => {
                return HttpResponse.json(
                    this.responseFactory.createValidationErrorResponse({
                        toVersionConnect: "Target resource version is required",
                        labels: "Labels must be an array of strings",
                    }),
                    { status: 400 },
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version-relation/:id`, ({ params }) => {
                const { id } = params;
                return HttpResponse.json(
                    this.responseFactory.createNotFoundErrorResponse(id as string),
                    { status: 404 },
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, () => {
                return HttpResponse.json(
                    this.responseFactory.createPermissionErrorResponse("create"),
                    { status: 403 },
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, () => {
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
    createLoadingHandlers(delay = 5000): RequestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, async ({ request }) => {
                const body = await request.json() as ResourceVersionRelationCreateInput;
                const relation = this.responseFactory.createRelationFromInput(body);
                const response = this.responseFactory.createSuccessResponse(relation);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/resource-version-relation`, () => {
                return HttpResponse.error();
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/resource-version-relation/:id`, () => {
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
        
        const httpMethod = http[method.toLowerCase() as keyof typeof http] as any;
        
        return httpMethod(fullEndpoint, async () => {
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
export const resourceVersionRelationResponseScenarios = {
    // Success scenarios
    createSuccess: (relation?: ResourceVersionRelation) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createSuccessResponse(
            relation || factory.createMockRelation(),
        );
    },
    
    listSuccess: (relations?: ResourceVersionRelation[]) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createRelationListResponse(
            relations || factory.createMultipleRelations(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                toVersionConnect: "Target resource version is required",
                labels: "Labels must be an array of strings",
            },
        );
    },
    
    notFoundError: (relationId?: string) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createNotFoundErrorResponse(
            relationId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new ResourceVersionRelationResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
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
    networkErrorHandlers: () => new ResourceVersionRelationMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const resourceVersionRelationResponseFactory = new ResourceVersionRelationResponseFactory();
export const resourceVersionRelationMSWHandlers = new ResourceVersionRelationMSWHandlers();
