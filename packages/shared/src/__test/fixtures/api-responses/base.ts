/* c8 ignore start */
/**
 * Base API Response Factory
 * 
 * This abstract class provides the foundation for all API response fixtures,
 * eliminating boilerplate and leveraging the existing error infrastructure.
 */

import { http, type HttpHandler } from "msw";
import { generatePK, nanoid } from "../../../index.js";
import { 
    apiErrorFixtures,
    authErrorFixtures,
    businessErrorFixtures,
    networkErrorFixtures,
    systemErrorFixtures,
    validationErrorFixtures,
} from "../errors/index.js";
import type { ParseableError } from "../../errors/index.js";
import type { 
    APIResponse, 
    APIErrorResponse, 
    PaginatedAPIResponse,
    BatchAPIResponse,
    APIResponseFactoryConfig,
    MockDataOptions,
    MSWHandlerConfig,
} from "./types.js";

/**
 * Base factory class for API response fixtures
 * 
 * @template TData - The data type for successful responses
 * @template TCreateInput - The input type for create operations
 * @template TUpdateInput - The input type for update operations
 */
export abstract class BaseAPIResponseFactory<
    TData,
    TCreateInput = unknown,
    TUpdateInput = unknown
> {
    protected abstract readonly entityName: string;
    protected readonly config: Required<APIResponseFactoryConfig>;

    constructor(config: APIResponseFactoryConfig = {}) {
        const DEFAULT_PAGE_SIZE = 20;
        this.config = {
            baseUrl: config.baseUrl || process.env.VITE_SERVER_URL || "http://localhost:5329",
            version: config.version || "1.0",
            defaultPageSize: config.defaultPageSize || DEFAULT_PAGE_SIZE,
            requestIdGenerator: config.requestIdGenerator || (() => `req_${Date.now()}_${nanoid()}`),
            idGenerator: config.idGenerator || (() => generatePK().toString()),
        };
    }

    /**
     * Get the base endpoint path for this entity
     */
    protected get basePath(): string {
        return `/api/${this.entityName}`;
    }

    /**
     * Generate a unique request ID
     */
    protected generateRequestId(): string {
        return this.config.requestIdGenerator();
    }

    /**
     * Generate a unique entity ID
     */
    protected generateId(): string {
        return this.config.idGenerator();
    }

    /**
     * Create a successful API response
     */
    createSuccessResponse(data: TData, options?: {
        links?: Record<string, string>;
        additionalMeta?: Record<string, unknown>;
    }): APIResponse<TData> {
        return {
            data,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: this.config.version,
                links: options?.links ? {
                    self: `${this.config.baseUrl}${this.basePath}/${(data as Record<string, unknown>).id}`,
                    ...options.links,
                } : undefined,
                ...options?.additionalMeta,
            },
        };
    }

    /**
     * Create a paginated API response
     */
    createPaginatedResponse(
        items: TData[],
        pagination: {
            page: number;
            pageSize?: number;
            totalCount: number;
        },
        options?: {
            links?: Record<string, string>;
            additionalMeta?: Record<string, unknown>;
        },
    ): PaginatedAPIResponse<TData> {
        const pageSize = pagination.pageSize || this.config.defaultPageSize;
        const totalPages = Math.ceil(pagination.totalCount / pageSize);

        return {
            data: items,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: this.config.version,
                links: {
                    self: `${this.config.baseUrl}${this.basePath}?page=${pagination.page}&limit=${pageSize}`,
                    first: `${this.config.baseUrl}${this.basePath}?page=1&limit=${pageSize}`,
                    last: `${this.config.baseUrl}${this.basePath}?page=${totalPages}&limit=${pageSize}`,
                    ...(pagination.page > 1 && {
                        prev: `${this.config.baseUrl}${this.basePath}?page=${pagination.page - 1}&limit=${pageSize}`,
                    }),
                    ...(pagination.page < totalPages && {
                        next: `${this.config.baseUrl}${this.basePath}?page=${pagination.page + 1}&limit=${pageSize}`,
                    }),
                    ...options?.links,
                },
                ...options?.additionalMeta,
            },
            pagination: {
                page: pagination.page,
                pageSize,
                totalCount: pagination.totalCount,
                totalPages,
                hasNextPage: pagination.page < totalPages,
                hasPreviousPage: pagination.page > 1,
            },
        };
    }

    /**
     * Create a batch operation response
     */
    createBatchResponse(
        results: Array<{ success: boolean; data?: TData; error?: APIErrorResponse["error"] }>,
        options?: {
            links?: Record<string, string>;
            additionalMeta?: Record<string, unknown>;
        },
    ): BatchAPIResponse<TData> {
        const succeeded = results.filter(r => r.success);
        const failed = results.filter(r => !r.success);

        return {
            data: succeeded.map(r => r.data as TData),
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: this.config.version,
                links: options?.links,
                ...options?.additionalMeta,
            },
            batch: {
                total: results.length,
                succeeded: succeeded.length,
                failed: failed.length,
                errors: failed.length > 0 ? failed.map((r) => ({
                    index: results.indexOf(r),
                    error: r.error as APIErrorResponse["error"],
                })) : undefined,
            },
        };
    }

    /**
     * Convert a ParseableError to an API error response
     */
    protected errorToAPIResponse(error: ParseableError, path: string): APIErrorResponse {
        const serverError = error.toServerError();
        return {
            error: {
                code: serverError.code,
                message: error.getUserMessage(["en"]),
                details: (error as Record<string, unknown>).data,
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path,
                severity: error.getSeverity(),
                trace: serverError.trace,
            },
        };
    }

    /**
     * Create a validation error response using the shared error infrastructure
     */
    createValidationErrorResponse(
        fieldErrors: Record<string, string | string[]>,
        path?: string,
    ): APIErrorResponse {
        const error = validationErrorFixtures.create({
            fields: fieldErrors,
            message: "Validation failed",
        });
        return this.errorToAPIResponse(error, path || this.basePath);
    }

    /**
     * Create a not found error response
     */
    createNotFoundErrorResponse(
        resourceId: string,
        resourceType?: string,
    ): APIErrorResponse {
        const error = apiErrorFixtures.variants.notFound;
        return {
            error: {
                code: error.code,
                message: `${resourceType || this.entityName} with ID '${resourceId}' was not found`,
                details: {
                    resourceId,
                    resourceType: resourceType || this.entityName,
                    searchCriteria: { id: resourceId },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `${this.basePath}/${resourceId}`,
            },
        };
    }

    /**
     * Create a permission error response
     */
    createPermissionErrorResponse(
        operation: string,
        requiredPermissions?: string[],
    ): APIErrorResponse {
        const error = authErrorFixtures.variants.permissionDenied;
        return {
            error: {
                code: error.code,
                message: error.message,
                details: {
                    operation,
                    requiredPermissions: requiredPermissions || [`${this.entityName}:${operation}`],
                    ...error.details,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: this.basePath,
            },
        };
    }

    /**
     * Create a rate limit error response
     */
    createRateLimitErrorResponse(
        limit: number,
        remaining: number,
        resetTime: Date,
    ): APIErrorResponse {
        const error = apiErrorFixtures.variants.rateLimitExceeded;
        const SECONDS_PER_MS = 1000;
        return {
            error: {
                code: error.code,
                message: error.message,
                details: {
                    limit,
                    remaining,
                    reset: resetTime.toISOString(),
                    retryAfter: Math.ceil((resetTime.getTime() - Date.now()) / SECONDS_PER_MS),
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: this.basePath,
            },
        };
    }

    /**
     * Create a business error response
     */
    createBusinessErrorResponse(
        type: "limit" | "conflict" | "state" | "workflow" | "constraint" | "policy",
        details: Record<string, unknown>,
    ): APIErrorResponse {
        const error = businessErrorFixtures.create({ type, details });
        return this.errorToAPIResponse(error, this.basePath);
    }

    /**
     * Create a network error response
     */
    createNetworkErrorResponse(): APIErrorResponse {
        const error = networkErrorFixtures.variants.timeout;
        return this.errorToAPIResponse(error, this.basePath);
    }

    /**
     * Create a server error response
     */
    createServerErrorResponse(
        component?: string,
        operation?: string,
    ): APIErrorResponse {
        const error = systemErrorFixtures.create({
            component: component || this.entityName,
            details: { operation },
        });
        return this.errorToAPIResponse(error, this.basePath);
    }

    /**
     * Abstract methods that must be implemented by subclasses
     */
    
    /**
     * Create mock data for this entity
     */
    abstract createMockData(options?: MockDataOptions): TData;

    /**
     * Create entity from create input
     */
    abstract createFromInput(input: TCreateInput): TData;

    /**
     * Update entity from update input
     */
    abstract updateFromInput(existing: TData, input: TUpdateInput): TData;

    /**
     * Validate create input
     */
    abstract validateCreateInput(input: TCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }>;

    /**
     * Validate update input
     */
    abstract validateUpdateInput(input: TUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }>;

    /**
     * Create MSW handlers for this entity
     */
    createMSWHandlers(config?: Partial<MSWHandlerConfig>): HttpHandler[] {
        const handlerConfig: MSWHandlerConfig = {
            baseUrl: this.config.baseUrl,
            ...config,
        };

        return [
            this.createGetHandler(handlerConfig),
            this.createListHandler(handlerConfig),
            this.createCreateHandler(handlerConfig),
            this.createUpdateHandler(handlerConfig),
            this.createDeleteHandler(handlerConfig),
        ];
    }

    /**
     * Create GET handler
     */
    protected createGetHandler(config: MSWHandlerConfig): HttpHandler {
        return http.get(`${config.baseUrl}${this.basePath}/:id`, ({ params }) => {
            const { id } = params;
            
            // Simulate network failure
            if (Math.random() < (config.networkFailureRate || 0)) {
                return new Response(null, { status: 0 });
            }

            // Simulate errors
            if (Math.random() < (config.errorRate || 0)) {
                return new Response(
                    JSON.stringify(this.createNotFoundErrorResponse(id as string)),
                    { status: 404, headers: { "Content-Type": "application/json" } },
                );
            }

            const data = this.createMockData({ overrides: { id } });
            const response = this.createSuccessResponse(data);

            return new Response(
                JSON.stringify(config.responseTransformer ? config.responseTransformer(response) : response),
                { 
                    status: 200, 
                    headers: { "Content-Type": "application/json" },
                },
            );
        });
    }

    /**
     * Create LIST handler
     */
    protected createListHandler(config: MSWHandlerConfig): HttpHandler {
        return http.get(`${config.baseUrl}${this.basePath}`, ({ request }) => {
            const url = new URL(request.url);
            const page = parseInt(url.searchParams.get("page") || "1");
            const limit = parseInt(url.searchParams.get("limit") || String(this.config.defaultPageSize));

            // Create mock items
            const MOCK_TOTAL_COUNT = 50;
            const totalCount = MOCK_TOTAL_COUNT;
            const items: TData[] = [];
            for (let i = 0; i < Math.min(limit, totalCount - (page - 1) * limit); i++) {
                items.push(this.createMockData());
            }

            const response = this.createPaginatedResponse(items, { page, pageSize: limit, totalCount });

            return new Response(
                JSON.stringify(config.responseTransformer ? config.responseTransformer(response) : response),
                { 
                    status: 200, 
                    headers: { "Content-Type": "application/json" },
                },
            );
        });
    }

    /**
     * Create CREATE handler
     */
    protected createCreateHandler(config: MSWHandlerConfig): HttpHandler {
        return http.post(`${config.baseUrl}${this.basePath}`, async ({ request }) => {
            const body = await request.json() as TCreateInput;

            // Validate input
            const validation = await this.validateCreateInput(body);
            if (!validation.valid) {
                return new Response(
                    JSON.stringify(this.createValidationErrorResponse(validation.errors || {})),
                    { status: 400, headers: { "Content-Type": "application/json" } },
                );
            }

            const data = this.createFromInput(body);
            const response = this.createSuccessResponse(data);

            return new Response(
                JSON.stringify(config.responseTransformer ? config.responseTransformer(response) : response),
                { 
                    status: 201, 
                    headers: { "Content-Type": "application/json" },
                },
            );
        });
    }

    /**
     * Create UPDATE handler
     */
    protected createUpdateHandler(config: MSWHandlerConfig): HttpHandler {
        return http.put(`${config.baseUrl}${this.basePath}/:id`, async ({ request, params }) => {
            const { id } = params;
            const body = await request.json() as TUpdateInput;

            // Validate input
            const validation = await this.validateUpdateInput(body);
            if (!validation.valid) {
                return new Response(
                    JSON.stringify(this.createValidationErrorResponse(validation.errors || {})),
                    { status: 400, headers: { "Content-Type": "application/json" } },
                );
            }

            const existing = this.createMockData({ overrides: { id } });
            const updated = this.updateFromInput(existing, body);
            const response = this.createSuccessResponse(updated);

            return new Response(
                JSON.stringify(config.responseTransformer ? config.responseTransformer(response) : response),
                { 
                    status: 200, 
                    headers: { "Content-Type": "application/json" },
                },
            );
        });
    }

    /**
     * Create DELETE handler
     */
    protected createDeleteHandler(config: MSWHandlerConfig): HttpHandler {
        return http.delete(`${config.baseUrl}${this.basePath}/:id`, () => {
            return new Response(null, { status: 204 });
        });
    }
}
