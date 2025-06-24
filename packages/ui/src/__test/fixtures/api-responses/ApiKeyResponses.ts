/**
 * ApiKey API Response Fixtures
 * 
 * This file provides comprehensive API response fixtures for apiKey and apiKeyExternal endpoints.
 * It includes success responses, error responses, and MSW handlers for testing.
 */

import { http, type RestHandler } from "msw";
import type { 
    ApiKey, 
    ApiKeyCreated,
    ApiKeyCreateInput, 
    ApiKeyUpdateInput,
    ApiKeyExternal,
    ApiKeyExternalCreateInput,
    ApiKeyExternalUpdateInput,
} from "@vrooli/shared";
import { 
    apiKeyValidation,
    apiKeyExternalValidation,
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
 * ApiKey API response factory
 */
export class ApiKeyResponseFactory {
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
     * Generate API key string
     */
    private generateApiKeyString(): string {
        const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
        let result = "vr_";
        for (let i = 0; i < 48; i++) {
            result += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        return result;
    }
    
    /**
     * Create successful ApiKey response
     */
    createApiKeySuccessResponse(apiKey: ApiKey): APIResponse<ApiKey> {
        return {
            data: apiKey,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/apiKey/${apiKey.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/current`,
                        usage: `${this.baseUrl}/api/apiKey/${apiKey.id}/usage`,
                    },
                },
            },
        };
    }
    
    /**
     * Create successful ApiKeyCreated response (includes the actual key)
     */
    createApiKeyCreatedSuccessResponse(apiKeyCreated: ApiKeyCreated): APIResponse<ApiKeyCreated> {
        return {
            data: apiKeyCreated,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/apiKey/${apiKeyCreated.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/current`,
                        usage: `${this.baseUrl}/api/apiKey/${apiKeyCreated.id}/usage`,
                    },
                },
            },
        };
    }
    
    /**
     * Create successful ApiKeyExternal response
     */
    createApiKeyExternalSuccessResponse(apiKeyExternal: ApiKeyExternal): APIResponse<ApiKeyExternal> {
        return {
            data: apiKeyExternal,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/apiKeyExternal/${apiKeyExternal.id}`,
                    related: {
                        user: `${this.baseUrl}/api/user/current`,
                        service: `${this.baseUrl}/api/services/${apiKeyExternal.service}`,
                    },
                },
            },
        };
    }
    
    /**
     * Create ApiKey list response
     */
    createApiKeyListResponse(apiKeys: ApiKey[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ApiKey> {
        const paginationData = pagination || {
            page: 1,
            pageSize: apiKeys.length,
            totalCount: apiKeys.length,
        };
        
        return {
            data: apiKeys,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/apiKey?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
     * Create ApiKeyExternal list response
     */
    createApiKeyExternalListResponse(apiKeysExternal: ApiKeyExternal[], pagination?: {
        page: number;
        pageSize: number;
        totalCount: number;
    }): PaginatedAPIResponse<ApiKeyExternal> {
        const paginationData = pagination || {
            page: 1,
            pageSize: apiKeysExternal.length,
            totalCount: apiKeysExternal.length,
        };
        
        return {
            data: apiKeysExternal,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: "1.0",
                links: {
                    self: `${this.baseUrl}/api/apiKeyExternal?page=${paginationData.page}&limit=${paginationData.pageSize}`,
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
    createValidationErrorResponse(fieldErrors: Record<string, string>, path = "/api/apiKey"): APIErrorResponse {
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
                path,
            },
        };
    }
    
    /**
     * Create not found error response
     */
    createNotFoundErrorResponse(apiKeyId: string, type: "apiKey" | "apiKeyExternal" = "apiKey"): APIErrorResponse {
        return {
            error: {
                code: type === "apiKey" ? "API_KEY_NOT_FOUND" : "API_KEY_EXTERNAL_NOT_FOUND",
                message: `${type === "apiKey" ? "API key" : "External API key"} with ID '${apiKeyId}' was not found`,
                details: {
                    apiKeyId,
                    searchCriteria: { id: apiKeyId },
                    type,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `${type === "apiKey" ? "/api/apiKey" : "/api/apiKeyExternal"}/${apiKeyId}`,
            },
        };
    }
    
    /**
     * Create permission error response
     */
    createPermissionErrorResponse(operation: string, type: "apiKey" | "apiKeyExternal" = "apiKey"): APIErrorResponse {
        const resource = type === "apiKey" ? "API key" : "external API key";
        return {
            error: {
                code: "PERMISSION_DENIED",
                message: `You do not have permission to ${operation} this ${resource}`,
                details: {
                    operation,
                    resource: type,
                    requiredPermissions: [`${type}:${operation}`],
                    userPermissions: [`${type}:read`],
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: type === "apiKey" ? "/api/apiKey" : "/api/apiKeyExternal",
            },
        };
    }
    
    /**
     * Create rate limit error response
     */
    createRateLimitErrorResponse(apiKeyId: string): APIErrorResponse {
        return {
            error: {
                code: "RATE_LIMIT_EXCEEDED",
                message: "API key has exceeded its rate limit",
                details: {
                    apiKeyId,
                    limitType: "hard",
                    resetTime: new Date(Date.now() + 3600000).toISOString(), // 1 hour from now
                    retryAfter: 3600,
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/apiKey/${apiKeyId}/validate`,
            },
        };
    }
    
    /**
     * Create quota exceeded error response
     */
    createQuotaExceededErrorResponse(apiKeyId: string): APIErrorResponse {
        return {
            error: {
                code: "QUOTA_EXCEEDED",
                message: "API key has exceeded its usage quota",
                details: {
                    apiKeyId,
                    quotaType: "monthly",
                    resetDate: new Date(new Date().getFullYear(), new Date().getMonth() + 1, 1).toISOString(),
                    usageDetails: {
                        used: "5000000",
                        limit: "5000000",
                    },
                },
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                path: `/api/apiKey/${apiKeyId}/validate`,
            },
        };
    }
    
    /**
     * Create network error response
     */
    createNetworkErrorResponse(type: "apiKey" | "apiKeyExternal" = "apiKey"): APIErrorResponse {
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
                path: type === "apiKey" ? "/api/apiKey" : "/api/apiKeyExternal",
            },
        };
    }
    
    /**
     * Create server error response
     */
    createServerErrorResponse(type: "apiKey" | "apiKeyExternal" = "apiKey"): APIErrorResponse {
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
                path: type === "apiKey" ? "/api/apiKey" : "/api/apiKeyExternal",
            },
        };
    }
    
    /**
     * Create mock ApiKey data
     */
    createMockApiKey(overrides?: Partial<ApiKey>): ApiKey {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultApiKey: ApiKey = {
            __typename: "ApiKey",
            id,
            creditsUsed: "1250000",
            disabledAt: null,
            limitHard: "5000000",
            limitSoft: "4000000",
            name: "Development API Key",
            permissions: JSON.stringify({
                read: true,
                write: true,
                execute: true,
                admin: false,
            }),
            stopAtLimit: false,
        };
        
        return {
            ...defaultApiKey,
            ...overrides,
        };
    }
    
    /**
     * Create mock ApiKeyCreated data (includes the actual key)
     */
    createMockApiKeyCreated(overrides?: Partial<ApiKeyCreated>): ApiKeyCreated {
        const baseApiKey = this.createMockApiKey(overrides);
        
        return {
            ...baseApiKey,
            key: this.generateApiKeyString(),
            ...overrides,
        };
    }
    
    /**
     * Create mock ApiKeyExternal data
     */
    createMockApiKeyExternal(overrides?: Partial<ApiKeyExternal>): ApiKeyExternal {
        const now = new Date().toISOString();
        const id = this.generateId();
        
        const defaultApiKeyExternal: ApiKeyExternal = {
            __typename: "ApiKeyExternal",
            id,
            disabledAt: null,
            name: "OpenAI API Key",
            service: "OpenAI",
        };
        
        return {
            ...defaultApiKeyExternal,
            ...overrides,
        };
    }
    
    /**
     * Create ApiKey from API input
     */
    createApiKeyFromInput(input: ApiKeyCreateInput): ApiKeyCreated {
        const apiKey = this.createMockApiKeyCreated();
        
        // Update apiKey based on input
        if (input.name) apiKey.name = input.name;
        if (input.limitHard) apiKey.limitHard = input.limitHard;
        if (input.limitSoft) apiKey.limitSoft = input.limitSoft;
        if (input.stopAtLimit !== undefined) apiKey.stopAtLimit = input.stopAtLimit;
        if (input.permissions) apiKey.permissions = input.permissions;
        if (input.disabled) apiKey.disabledAt = new Date().toISOString();
        
        return apiKey;
    }
    
    /**
     * Create ApiKeyExternal from API input
     */
    createApiKeyExternalFromInput(input: ApiKeyExternalCreateInput): ApiKeyExternal {
        const apiKeyExternal = this.createMockApiKeyExternal();
        
        // Update apiKeyExternal based on input
        if (input.name) apiKeyExternal.name = input.name;
        if (input.service) apiKeyExternal.service = input.service;
        if (input.disabled) apiKeyExternal.disabledAt = new Date().toISOString();
        
        return apiKeyExternal;
    }
    
    /**
     * Create multiple ApiKeys for different scenarios
     */
    createMultipleApiKeys(): ApiKey[] {
        return [
            this.createMockApiKey({
                name: "Production API Key",
                limitHard: "10000000",
                limitSoft: "8000000",
                creditsUsed: "2500000",
                permissions: JSON.stringify({ read: true, write: true, execute: true, admin: true }),
                stopAtLimit: true,
            }),
            this.createMockApiKey({
                name: "Development API Key", 
                limitHard: "1000000",
                limitSoft: "800000",
                creditsUsed: "450000",
                permissions: JSON.stringify({ read: true, write: true, execute: false, admin: false }),
                stopAtLimit: false,
            }),
            this.createMockApiKey({
                name: "Testing API Key",
                limitHard: "500000", 
                limitSoft: "400000",
                creditsUsed: "125000",
                permissions: JSON.stringify({ read: true, write: false, execute: false, admin: false }),
                stopAtLimit: true,
                disabledAt: new Date().toISOString(),
            }),
        ];
    }
    
    /**
     * Create multiple ApiKeyExternals for different services
     */
    createMultipleApiKeyExternals(): ApiKeyExternal[] {
        return [
            this.createMockApiKeyExternal({
                name: "OpenAI GPT-4",
                service: "OpenAI",
            }),
            this.createMockApiKeyExternal({
                name: "Anthropic Claude",
                service: "Anthropic",
            }),
            this.createMockApiKeyExternal({
                name: "Google Gemini",
                service: "Google",
            }),
            this.createMockApiKeyExternal({
                name: "Azure OpenAI",
                service: "Azure",
                disabledAt: new Date().toISOString(),
            }),
        ];
    }
    
    /**
     * Validate ApiKey create input
     */
    async validateApiKeyCreateInput(input: ApiKeyCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await apiKeyValidation.create.validate(input);
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
     * Validate ApiKeyExternal create input
     */
    async validateApiKeyExternalCreateInput(input: ApiKeyExternalCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        try {
            await apiKeyExternalValidation.create.validate(input);
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
 * MSW handlers factory for ApiKey endpoints
 */
export class ApiKeyMSWHandlers {
    private responseFactory: ApiKeyResponseFactory;
    
    constructor(baseUrl?: string) {
        this.responseFactory = new ApiKeyResponseFactory(baseUrl);
    }
    
    /**
     * Create success handlers for all ApiKey endpoints
     */
    createApiKeySuccessHandlers(): RestHandler[] {
        return [
            // Create ApiKey
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey`, async (req, res, ctx) => {
                const body = await req.json() as ApiKeyCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateApiKeyCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {})),
                    );
                }
                
                // Create ApiKey
                const apiKey = this.responseFactory.createApiKeyFromInput(body);
                const response = this.responseFactory.createApiKeyCreatedSuccessResponse(apiKey);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get ApiKey by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKey/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const apiKey = this.responseFactory.createMockApiKey({ id: id as string });
                const response = this.responseFactory.createApiKeySuccessResponse(apiKey);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update ApiKey
            http.put(`${this.responseFactory["baseUrl"]}/api/apiKey/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ApiKeyUpdateInput;
                
                const apiKey = this.responseFactory.createMockApiKey({ 
                    id: id as string,
                    ...body,
                });
                
                const response = this.responseFactory.createApiKeySuccessResponse(apiKey);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete ApiKey
            http.delete(`${this.responseFactory["baseUrl"]}/api/apiKey/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List ApiKeys
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKey`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                
                const apiKeys = this.responseFactory.createMultipleApiKeys();
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedApiKeys = apiKeys.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createApiKeyListResponse(
                    paginatedApiKeys,
                    {
                        page,
                        pageSize: limit,
                        totalCount: apiKeys.length,
                    },
                );
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Validate ApiKey
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey/validate`, (req, res, ctx) => {
                return res(
                    ctx.status(200),
                    ctx.json({
                        valid: true,
                        permissions: JSON.stringify({ read: true, write: true, execute: true }),
                        creditsRemaining: "2750000",
                    }),
                );
            }),
        ];
    }
    
    /**
     * Create success handlers for all ApiKeyExternal endpoints
     */
    createApiKeyExternalSuccessHandlers(): RestHandler[] {
        return [
            // Create ApiKeyExternal
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, async (req, res, ctx) => {
                const body = await req.json() as ApiKeyExternalCreateInput;
                
                // Validate input
                const validation = await this.responseFactory.validateApiKeyExternalCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.responseFactory.createValidationErrorResponse(validation.errors || {}, "/api/apiKeyExternal")),
                    );
                }
                
                // Create ApiKeyExternal
                const apiKeyExternal = this.responseFactory.createApiKeyExternalFromInput(body);
                const response = this.responseFactory.createApiKeyExternalSuccessResponse(apiKeyExternal);
                
                return res(
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            // Get ApiKeyExternal by ID
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal/:id`, (req, res, ctx) => {
                const { id } = req.params;
                
                const apiKeyExternal = this.responseFactory.createMockApiKeyExternal({ id: id as string });
                const response = this.responseFactory.createApiKeyExternalSuccessResponse(apiKeyExternal);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Update ApiKeyExternal
            http.put(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal/:id`, async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json() as ApiKeyExternalUpdateInput;
                
                const apiKeyExternal = this.responseFactory.createMockApiKeyExternal({ 
                    id: id as string,
                    ...body,
                });
                
                const response = this.responseFactory.createApiKeyExternalSuccessResponse(apiKeyExternal);
                
                return res(
                    ctx.status(200),
                    ctx.json(response),
                );
            }),
            
            // Delete ApiKeyExternal
            http.delete(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal/:id`, (req, res, ctx) => {
                return res(ctx.status(204));
            }),
            
            // List ApiKeyExternals
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, (req, res, ctx) => {
                const url = new URL(req.url);
                const page = parseInt(url.searchParams.get("page") || "1");
                const limit = parseInt(url.searchParams.get("limit") || "10");
                const service = url.searchParams.get("service");
                
                let apiKeysExternal = this.responseFactory.createMultipleApiKeyExternals();
                
                // Filter by service if specified
                if (service) {
                    apiKeysExternal = apiKeysExternal.filter(k => k.service === service);
                }
                
                // Paginate
                const startIndex = (page - 1) * limit;
                const paginatedApiKeysExternal = apiKeysExternal.slice(startIndex, startIndex + limit);
                
                const response = this.responseFactory.createApiKeyExternalListResponse(
                    paginatedApiKeysExternal,
                    {
                        page,
                        pageSize: limit,
                        totalCount: apiKeysExternal.length,
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
    createApiKeyErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        name: "API key name is required",
                        limitHard: "Hard limit must be a positive number",
                        permissions: "Permissions must be a valid JSON string",
                    })),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKey/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string, "apiKey")),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create", "apiKey")),
                );
            }),
            
            // Rate limit error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey/validate`, (req, res, ctx) => {
                return res(
                    ctx.status(429),
                    ctx.json(this.responseFactory.createRateLimitErrorResponse("test-api-key-id")),
                );
            }),
            
            // Quota exceeded error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey/validate`, (req, res, ctx) => {
                return res(
                    ctx.status(402),
                    ctx.json(this.responseFactory.createQuotaExceededErrorResponse("test-api-key-id")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse("apiKey")),
                );
            }),
        ];
    }
    
    /**
     * Create error handlers for ApiKeyExternal testing error scenarios
     */
    createApiKeyExternalErrorHandlers(): RestHandler[] {
        return [
            // Validation error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(this.responseFactory.createValidationErrorResponse({
                        name: "External API key name is required",
                        service: "Service provider is required",
                        key: "API key value is required",
                    }, "/api/apiKeyExternal")),
                );
            }),
            
            // Not found error
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal/:id`, (req, res, ctx) => {
                const { id } = req.params;
                return res(
                    ctx.status(404),
                    ctx.json(this.responseFactory.createNotFoundErrorResponse(id as string, "apiKeyExternal")),
                );
            }),
            
            // Permission error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, (req, res, ctx) => {
                return res(
                    ctx.status(403),
                    ctx.json(this.responseFactory.createPermissionErrorResponse("create", "apiKeyExternal")),
                );
            }),
            
            // Server error
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json(this.responseFactory.createServerErrorResponse("apiKeyExternal")),
                );
            }),
        ];
    }
    
    /**
     * Create loading simulation handlers
     */
    createLoadingHandlers(delay = 2000): RestHandler[] {
        return [
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey`, async (req, res, ctx) => {
                const body = await req.json() as ApiKeyCreateInput;
                const apiKey = this.responseFactory.createApiKeyFromInput(body);
                const response = this.responseFactory.createApiKeyCreatedSuccessResponse(apiKey);
                
                return res(
                    ctx.delay(delay),
                    ctx.status(201),
                    ctx.json(response),
                );
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, async (req, res, ctx) => {
                const body = await req.json() as ApiKeyExternalCreateInput;
                const apiKeyExternal = this.responseFactory.createApiKeyExternalFromInput(body);
                const response = this.responseFactory.createApiKeyExternalSuccessResponse(apiKeyExternal);
                
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
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKey`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKey/:id`, (req, res, ctx) => {
                return res.networkError("Connection timeout");
            }),
            
            http.post(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal`, (req, res, ctx) => {
                return res.networkError("Network connection failed");
            }),
            
            http.get(`${this.responseFactory["baseUrl"]}/api/apiKeyExternal/:id`, (req, res, ctx) => {
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
export const apiKeyResponseScenarios = {
    // ApiKey Success scenarios
    createApiKeySuccess: (apiKey?: ApiKeyCreated) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createApiKeyCreatedSuccessResponse(
            apiKey || factory.createMockApiKeyCreated(),
        );
    },
    
    listApiKeySuccess: (apiKeys?: ApiKey[]) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createApiKeyListResponse(
            apiKeys || factory.createMultipleApiKeys(),
        );
    },
    
    // ApiKeyExternal Success scenarios
    createApiKeyExternalSuccess: (apiKeyExternal?: ApiKeyExternal) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createApiKeyExternalSuccessResponse(
            apiKeyExternal || factory.createMockApiKeyExternal(),
        );
    },
    
    listApiKeyExternalSuccess: (apiKeysExternal?: ApiKeyExternal[]) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createApiKeyExternalListResponse(
            apiKeysExternal || factory.createMultipleApiKeyExternals(),
        );
    },
    
    // Error scenarios
    validationError: (fieldErrors?: Record<string, string>, type: "apiKey" | "apiKeyExternal" = "apiKey") => {
        const factory = new ApiKeyResponseFactory();
        const path = type === "apiKey" ? "/api/apiKey" : "/api/apiKeyExternal";
        return factory.createValidationErrorResponse(
            fieldErrors || {
                name: `${type === "apiKey" ? "API key" : "External API key"} name is required`,
                [type === "apiKey" ? "limitHard" : "service"]: `${type === "apiKey" ? "Hard limit is required" : "Service provider is required"}`,
            },
            path,
        );
    },
    
    notFoundError: (apiKeyId?: string, type: "apiKey" | "apiKeyExternal" = "apiKey") => {
        const factory = new ApiKeyResponseFactory();
        return factory.createNotFoundErrorResponse(
            apiKeyId || "non-existent-id",
            type,
        );
    },
    
    permissionError: (operation?: string, type: "apiKey" | "apiKeyExternal" = "apiKey") => {
        const factory = new ApiKeyResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            type,
        );
    },
    
    rateLimitError: (apiKeyId?: string) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createRateLimitErrorResponse(
            apiKeyId || "test-api-key-id",
        );
    },
    
    quotaExceededError: (apiKeyId?: string) => {
        const factory = new ApiKeyResponseFactory();
        return factory.createQuotaExceededErrorResponse(
            apiKeyId || "test-api-key-id",
        );
    },
    
    serverError: (type: "apiKey" | "apiKeyExternal" = "apiKey") => {
        const factory = new ApiKeyResponseFactory();
        return factory.createServerErrorResponse(type);
    },
    
    // MSW handlers
    apiKeySuccessHandlers: () => new ApiKeyMSWHandlers().createApiKeySuccessHandlers(),
    apiKeyExternalSuccessHandlers: () => new ApiKeyMSWHandlers().createApiKeyExternalSuccessHandlers(),
    apiKeyErrorHandlers: () => new ApiKeyMSWHandlers().createApiKeyErrorHandlers(),
    apiKeyExternalErrorHandlers: () => new ApiKeyMSWHandlers().createApiKeyExternalErrorHandlers(),
    loadingHandlers: (delay?: number) => new ApiKeyMSWHandlers().createLoadingHandlers(delay),
    networkErrorHandlers: () => new ApiKeyMSWHandlers().createNetworkErrorHandlers(),
};

// Export factory instances for easy use
export const apiKeyResponseFactory = new ApiKeyResponseFactory();
export const apiKeyMSWHandlers = new ApiKeyMSWHandlers();
