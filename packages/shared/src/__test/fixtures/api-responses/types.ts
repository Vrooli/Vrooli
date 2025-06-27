/* c8 ignore start */
/**
 * Core types for API response fixtures
 * 
 * These types define the standard API response structures used throughout
 * the application, providing consistency across all endpoints.
 */

/**
 * Standard API response wrapper for successful responses
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
 * API error response structure that aligns with VrooliError architecture
 */
export interface APIErrorResponse {
    error: {
        code: string;
        message: string;
        details?: Record<string, unknown>;
        timestamp: string;
        requestId: string;
        path: string;
        severity?: "Error" | "Warning" | "Info";
        trace?: string;
    };
}

/**
 * Paginated response structure for list endpoints
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
 * Batch operation response for multiple operations
 */
export interface BatchAPIResponse<T> extends APIResponse<T[]> {
    batch: {
        total: number;
        succeeded: number;
        failed: number;
        errors?: Array<{
            index: number;
            error: APIErrorResponse["error"];
        }>;
    };
}

/**
 * Configuration for API response factory
 */
export interface APIResponseFactoryConfig {
    baseUrl?: string;
    version?: string;
    defaultPageSize?: number;
    requestIdGenerator?: () => string;
    idGenerator?: () => string;
}

/**
 * Mock data generation options
 */
export interface MockDataOptions {
    /** Override specific fields */
    overrides?: Record<string, unknown>;
    /** Generate relationships */
    withRelations?: boolean;
    /** Include optional fields */
    includeOptional?: boolean;
    /** Scenario preset */
    scenario?: "minimal" | "complete" | "edge-case";
}

/**
 * MSW handler configuration
 */
export interface MSWHandlerConfig {
    /** Base URL for endpoints */
    baseUrl: string;
    /** Response delay in milliseconds */
    delay?: number;
    /** Error rate for chaos testing (0-1) */
    errorRate?: number;
    /** Network failure rate (0-1) */
    networkFailureRate?: number;
    /** Custom response transformer */
    responseTransformer?: (response: unknown) => unknown;
}
