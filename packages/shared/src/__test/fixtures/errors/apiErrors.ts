/**
 * API error fixtures for testing HTTP error responses
 * 
 * These fixtures provide standard HTTP error scenarios including
 * bad requests, authentication errors, rate limiting, and server errors.
 */

export interface ApiError {
    status: number;
    code: string;
    message: string;
    details?: any;
    retryAfter?: number;
    limit?: number;
    remaining?: number;
    reset?: string;
}

export const apiErrorFixtures = {
    // 400 Bad Request
    badRequest: {
        minimal: {
            status: 400,
            code: "BAD_REQUEST",
            message: "Invalid request parameters",
        } satisfies ApiError,
        
        withDetails: {
            status: 400,
            code: "BAD_REQUEST", 
            message: "Invalid request parameters",
            details: {
                fields: {
                    email: "Invalid email format",
                    password: "Password must be at least 8 characters",
                },
            },
        } satisfies ApiError,
        
        malformedJson: {
            status: 400,
            code: "MALFORMED_JSON",
            message: "Request body contains invalid JSON",
        } satisfies ApiError,
    },

    // 401 Unauthorized
    unauthorized: {
        standard: {
            status: 401,
            code: "UNAUTHORIZED",
            message: "Authentication required",
        } satisfies ApiError,
        
        invalidToken: {
            status: 401,
            code: "INVALID_TOKEN", 
            message: "The provided authentication token is invalid",
        } satisfies ApiError,
        
        expiredToken: {
            status: 401,
            code: "TOKEN_EXPIRED",
            message: "Your session has expired. Please log in again",
        } satisfies ApiError,
    },

    // 403 Forbidden
    forbidden: {
        standard: {
            status: 403,
            code: "FORBIDDEN",
            message: "You do not have permission to access this resource",
        } satisfies ApiError,
        
        insufficientRole: {
            status: 403,
            code: "INSUFFICIENT_ROLE",
            message: "Your role does not have permission for this action",
            details: {
                requiredRole: "admin",
                currentRole: "member",
            },
        } satisfies ApiError,
        
        resourceOwner: {
            status: 403,
            code: "NOT_OWNER",
            message: "Only the resource owner can perform this action",
        } satisfies ApiError,
    },

    // 404 Not Found
    notFound: {
        standard: {
            status: 404,
            code: "NOT_FOUND",
            message: "Resource not found",
        } satisfies ApiError,
        
        withDetails: {
            status: 404,
            code: "NOT_FOUND",
            message: "The requested resource could not be found",
            details: {
                resource: "User",
                id: "user_123456789",
            },
        } satisfies ApiError,
        
        endpoint: {
            status: 404,
            code: "ENDPOINT_NOT_FOUND",
            message: "The requested endpoint does not exist",
        } satisfies ApiError,
    },

    // 429 Too Many Requests
    rateLimit: {
        standard: {
            status: 429,
            code: "RATE_LIMIT_EXCEEDED",
            message: "Too many requests. Please try again later",
            retryAfter: 60,
            limit: 100,
            remaining: 0,
            reset: new Date(Date.now() + 60000).toISOString(),
        } satisfies ApiError,
        
        daily: {
            status: 429,
            code: "DAILY_LIMIT_EXCEEDED",
            message: "Daily request limit exceeded",
            retryAfter: 86400,
            limit: 10000,
            remaining: 0,
            reset: new Date(Date.now() + 86400000).toISOString(),
        } satisfies ApiError,
        
        burst: {
            status: 429,
            code: "BURST_LIMIT_EXCEEDED",
            message: "Request rate too high. Please slow down",
            retryAfter: 5,
            limit: 10,
            remaining: 0, 
            reset: new Date(Date.now() + 5000).toISOString(),
        } satisfies ApiError,
    },

    // 500 Internal Server Error
    serverError: {
        standard: {
            status: 500,
            code: "INTERNAL_SERVER_ERROR",
            message: "An unexpected error occurred. Please try again later",
        } satisfies ApiError,
        
        database: {
            status: 500,
            code: "DATABASE_ERROR",
            message: "A database error occurred while processing your request",
        } satisfies ApiError,
        
        withRequestId: {
            status: 500,
            code: "INTERNAL_SERVER_ERROR",
            message: "An unexpected error occurred",
            details: {
                requestId: "req_abc123def456",
                timestamp: new Date().toISOString(),
            },
        } satisfies ApiError,
    },

    // 502 Bad Gateway
    badGateway: {
        standard: {
            status: 502,
            code: "BAD_GATEWAY",
            message: "The server received an invalid response from the upstream server",
        } satisfies ApiError,
    },

    // 503 Service Unavailable
    serviceUnavailable: {
        standard: {
            status: 503,
            code: "SERVICE_UNAVAILABLE",
            message: "The service is temporarily unavailable. Please try again later",
        } satisfies ApiError,
        
        maintenance: {
            status: 503,
            code: "MAINTENANCE_MODE",
            message: "The service is undergoing scheduled maintenance",
            details: {
                estimatedEndTime: new Date(Date.now() + 3600000).toISOString(),
            },
        } satisfies ApiError,
    },

    // 504 Gateway Timeout
    gatewayTimeout: {
        standard: {
            status: 504,
            code: "GATEWAY_TIMEOUT",
            message: "The request timed out",
        } satisfies ApiError,
    },

    // Factory functions
    factories: {
        /**
         * Create a validation error with custom field errors
         */
        createValidationError: (fields: Record<string, string>): ApiError => ({
            status: 400,
            code: "VALIDATION_ERROR",
            message: "Validation failed",
            details: { fields },
        }),

        /**
         * Create a custom API error
         */
        createApiError: (status: number, code: string, message: string, details?: any): ApiError => ({
            status,
            code,
            message,
            ...(details && { details }),
        }),

        /**
         * Create a rate limit error with custom limits
         */
        createRateLimitError: (limit: number, windowSeconds: number): ApiError => ({
            status: 429,
            code: "RATE_LIMIT_EXCEEDED",
            message: `Rate limit of ${limit} requests per ${windowSeconds} seconds exceeded`,
            retryAfter: windowSeconds,
            limit,
            remaining: 0,
            reset: new Date(Date.now() + windowSeconds * 1000).toISOString(),
        }),
    },
};