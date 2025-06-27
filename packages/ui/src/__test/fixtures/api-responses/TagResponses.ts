/**
 * Tag API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for tag endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    Tag, 
    TagCreateInput, 
    TagUpdateInput,
} from "@vrooli/shared";
import { 
    tagValidation, 
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
 * Tag API response factory
 */
export class TagResponseFactory {
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
     * Generate unique tag ID
     */
    private generateId(): string {
        return `${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create successful tag response
     */
    createSuccessResponse(tag: Tag): APIResponse<Tag> {
        return {
            data: tag,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/tag/${tag.id}`,
                    related: {
                        tagged: `${this.baseUrl}/api/tag/${tag.id}/tagged`,
                    },
                },
            },
        };
    }
    
    /**
     * Create tag list response
     */
    createTagListResponse(tags: Tag[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<Tag> {
        const paginationData = pagination || {
            page: 1,
            pageSize: tags.length,
            totalCount: tags.length,
        };
        
        return {
            data: tags,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/tag?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
                path: "/api/tag",
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(tagId: string): APIErrorResponse {
        return {
            error: {
                code: "TAG_NOT_FOUND",
                message: `Tag with ID '${tagId}' was not found`,
                details: {
                    tagId,
                    searchCriteria: { id: tagId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/tag/${tagId}`,
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
                message: `You do not have permission to ${operation} this tag`,
                details: {
                    operation,
                    requiredPermissions: ["tag:write"],
                    userPermissions: ["tag:read"],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: "/api/tag",
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
                path: "/api/tag",
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
                path: "/api/tag",
            },
        };
    }
    
    /**
     * Create mock tag data
     */
    createMockTag(overrides?: Partial<Tag>): Tag {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultTag: Tag = {
            __typename: "Tag",
            id,
            created_at: now,
            updated_at: now,
            tag: "example-tag",
            bookmarks: 0,
            translations: [{
                __typename: "TagTranslation",
                id: `trans_${id}`,
                language: "en",
                description: "An example tag for testing purposes",
            }],
            translationsCount: 1,
            you: {
                __typename: "TagYou",
                isOwn: false,
            },
        };
        
        return {
            ...defaultTag,
            ...overrides,
        };
    }
    
    /**
     * Create tag from API input
     */
    createTagFromInput(input: TagCreateInput): Tag {
        const tag = this.createMockTag();
        
        // Update tag based on input
        if (input.tag) {
            tag.tag = input.tag;
        }
        
        if (input.translations?.create) {
            tag.translations = input.translations.create.map(trans => ({
                __typename: "TagTranslation" as const,
                id: `trans_${this.generateId()}`,
                language: trans.language,
                description: trans.description || "",
            }));
            tag.translationsCount = tag.translations.length;
        }
        
        return tag;
    }
    
    /**
     * Create popular programming tags
     */
    createProgrammingTags(): Tag[] {
        const tags = [
            "javascript", "typescript", "python", "react", "nodejs", 
            "api", "database", "web-development", "mobile", "ai",
        ];
        
        return tags.map((tagName, index) => 
            this.createMockTag({
                tag: tagName,
                bookmarks: Math.floor(Math.random() * 1000) + 10,
                translations: [{
                    __typename: "TagTranslation",
                    id: `trans_${tagName}_${this.generateId()}`,
                    language: "en",
                    description: `Tag for ${tagName} related content`,
                }],
            }),
        );
    }
    
    /**
     * Create category-based tags
     */
    createCategoryTags(): Tag[] {
        const categories = [
            { name: "tutorial", desc: "Educational and instructional content" },
            { name: "beginner", desc: "Content suitable for beginners" },
            { name: "advanced", desc: "Advanced level content" },
            { name: "productivity", desc: "Tools and tips for productivity" },
            { name: "automation", desc: "Automated processes and workflows" },
            { name: "security", desc: "Security-related content" },
            { name: "testing", desc: "Testing methodologies and tools" },
            { name: "deployment", desc: "Deployment and DevOps content" },
        ];
        
        return categories.map(category => 
            this.createMockTag({
                tag: category.name,
                bookmarks: Math.floor(Math.random() * 500) + 5,
                translations: [{
                    __typename: "TagTranslation",
                    id: `trans_${category.name}_${this.generateId()}`,
                    language: "en",
                    description: category.desc,
                }],
            }),
        );
    }
    
    /**
     * Create trending tags
     */
    createTrendingTags(): Tag[] {
        return this.createProgrammingTags()
            .sort((a, b) => b.bookmarks - a.bookmarks)
            .slice(0, 5);
    }
    
    /**
     * Create tags with multiple translations
     */
    createMultilingualTags(): Tag[] {
        return [
            this.createMockTag({
                tag: "international",
                translations: [
                    {
                        __typename: "TagTranslation",
                        id: `trans_en_${this.generateId()}`,
                        language: "en",
                        description: "International content",
                    },
                    {
                        __typename: "TagTranslation",
                        id: `trans_es_${this.generateId()}`,
                        language: "es",
                        description: "Contenido internacional",
                    },
                    {
                        __typename: "TagTranslation",
                        id: `trans_fr_${this.generateId()}`,
                        language: "fr",
                        description: "Contenu international",
                    },
                ],
                translationsCount: 3,
            }),
        ];
    }
    
    /**
     * Validate tag create input
     */
    async validateCreateInput(input: TagCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await tagValidation.create.validate(input);
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
 * MSW handlers factory for tag endpoints
 */
export class TagMSWHandlers {
    private responseFactory: TagResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new TagResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all tag endpoints
     */
    createSuccessHandlers(): RestHandler[] {
        return [
            // Create tag
            http.post(`${this.responseFactory["baseUrl"]}/api/tag`, async (req, res, ctx) => {
                const body = await req.json() as TagCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create tag
                const tag = this.responseFactory.createTagFromInput(body);
                const response = this.responseFactory.createSuccessResponse(tag);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get tag by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/tag/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const tag = this.responseFactory.createMockTag({ id: id as string });
                const response = this.responseFactory.createSuccessResponse(tag);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update tag
            http.put(`${this.responseFactory["baseUrl"]}/api/tag/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as TagUpdateInput;
                
                const tag = this.responseFactory.createMockTag({ 
                    id: id as string,
                    updated_at: new Date().toISOString(),
                });
                
                // Apply updates from body
                if (body.tag) {
                    tag.tag = body.tag;
                }
                
                const response = this.responseFactory.createSuccessResponse(tag);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete tag
            http.delete(`${this.responseFactory["baseUrl"]}/api/tag/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List tags
            http.get(`${this.responseFactory["baseUrl"]}/api/tag`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const search = url.searchParams.get("search");
                const trending = url.searchParams.get("trending") === "true";
                const category = url.searchParams.get("category");
                
                let tags: Tag[] = [];
                
                if (trending) {
                    tags = this.responseFactory.createTrendingTags();
                } else if (category === "programming") {
                    tags = this.responseFactory.createProgrammingTags();
                } else if (category === "general") {
                    tags = this.responseFactory.createCategoryTags();
                } else {
                    tags = [
                        ...this.responseFactory.createProgrammingTags(),
                        ...this.responseFactory.createCategoryTags(),
                    ];
                }
                
                // Filter by search if specified
                if (search) {
                    tags = tags.filter(t => 
                        t.tag.toLowerCase().includes(search.toLowerCase()) ||
                        t.translations.some(trans => 
                            trans.description?.toLowerCase().includes(search.toLowerCase()),
                        ),
                    );
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedTags = tags.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createTagListResponse(
                    paginatedTags,
                    {
                        page,
                        pageSize: limit,
                        totalCount: tags.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Get trending tags
            http.get(`${this.responseFactory["baseUrl"]}/api/tag/trending`, (req, res, ctx) => {
                const tags = this.responseFactory.createTrendingTags();
                const response = this.responseFactory.createTagListResponse(tags);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/tag`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        tag: "Tag name is required and must be unique",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/tag/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string)),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/tag`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/tag`, (req, res, ctx) => {
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
            http.post(`${this.responseFactory["baseUrl"]}/api/tag`, async (req, res, ctx) => {
                const body = await req.json() as TagCreateInput;
                const tag = this.responseFactory.createTagFromInput(body);
                const response = this.responseFactory.createSuccessResponse(tag);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/tag`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/tag/:id`, (req, res, ctx) => {
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
export const tagResponseScenarios = {
    // Success scenarios
    createSuccess: (tag?: Tag) => {
        const factory = new TagResponseFactory();
        return factory.createSuccessResponse(
            tag || factory.createMockTag(),
        );
    },
    
    listSuccess: (tags?: Tag[]) => {
        const factory = new TagResponseFactory();
        return factory.createTagListResponse(
            tags || factory.createProgrammingTags(),
        );
    },
    
    trendingTags: () => {
        const factory = new TagResponseFactory();
        return factory.createTagListResponse(
            factory.createTrendingTags(),
        );
    },
    
    programmingTags: () => {
        const factory = new TagResponseFactory();
        return factory.createTagListResponse(
            factory.createProgrammingTags(),
        );
    },
    
    categoryTags: () => {
        const factory = new TagResponseFactory();
        return factory.createTagListResponse(
            factory.createCategoryTags(),
        );
    },
    
    multilingualTags: () => {
        const factory = new TagResponseFactory();
        return factory.createTagListResponse(
            factory.createMultilingualTags(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>) => {
        const factory = new TagResponseFactory();
        return factory.createValidationErrorResponse(
            fieldErrors || {
                tag: "Tag name is required and must be unique",
            },
        );
    },
    
    notFoundError: (tagId?: string) => {
        const factory = new TagResponseFactory();
        return factory.createNotFoundErrorResponse(
            tagId || "non-existent-id",
        );
    },
    
    permissionError: (operation?: string) => {
        const factory = new TagResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
        );
    },
    
    serverError: () => {
        const factory = new TagResponseFactory();
        return factory.createServerErrorResponse();
    },
    
    // MSW handlers
    successHandlers: () => new TagMSWHandlers().createSuccessHandlers(),
    errorHandlers: () => new TagMSWHandlers().createErrorHandlers(),
    loadingHandlers: (delay?: number) => new TagMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new TagMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const tagResponseFactory = new TagResponseFactory();
export const tagMSWHandlers = new TagMSWHandlers();
