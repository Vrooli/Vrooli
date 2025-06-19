/**
 * Error state fixtures for UI components
 * These represent various error states that components might display
 */

/**
 * API error responses
 */
export const apiErrors = {
    /**
     * 400 Bad Request
     */
    badRequest: {
        code: "BAD_REQUEST",
        message: "Invalid request parameters",
        status: 400,
        details: {
            fields: {
                email: "Invalid email format",
                password: "Password must be at least 8 characters",
            },
        },
    },

    /**
     * 401 Unauthorized
     */
    unauthorized: {
        code: "UNAUTHORIZED",
        message: "Authentication required",
        status: 401,
        details: {
            reason: "Token expired",
            action: "Please log in again",
        },
    },

    /**
     * 403 Forbidden
     */
    forbidden: {
        code: "FORBIDDEN",
        message: "You don't have permission to perform this action",
        status: 403,
        details: {
            requiredPermission: "Admin",
            currentRole: "Member",
        },
    },

    /**
     * 404 Not Found
     */
    notFound: {
        code: "NOT_FOUND",
        message: "The requested resource could not be found",
        status: 404,
        details: {
            resource: "User",
            id: "user_123456789",
        },
    },

    /**
     * 409 Conflict
     */
    conflict: {
        code: "CONFLICT",
        message: "Resource already exists",
        status: 409,
        details: {
            field: "email",
            value: "user@example.com",
            suggestion: "Try a different email address",
        },
    },

    /**
     * 422 Unprocessable Entity
     */
    unprocessableEntity: {
        code: "UNPROCESSABLE_ENTITY",
        message: "The request could not be processed",
        status: 422,
        details: {
            reason: "Business logic validation failed",
            rules: ["User must verify email before posting"],
        },
    },

    /**
     * 429 Too Many Requests
     */
    rateLimited: {
        code: "RATE_LIMITED",
        message: "Too many requests. Please try again later.",
        status: 429,
        details: {
            limit: 100,
            window: "1 hour",
            retryAfter: 3600, // seconds
        },
    },

    /**
     * 500 Internal Server Error
     */
    serverError: {
        code: "INTERNAL_SERVER_ERROR",
        message: "An unexpected error occurred. Please try again later.",
        status: 500,
        details: {
            requestId: "req_123456789",
            timestamp: "2024-01-20T10:00:00Z",
        },
    },

    /**
     * 503 Service Unavailable
     */
    serviceUnavailable: {
        code: "SERVICE_UNAVAILABLE",
        message: "Service temporarily unavailable. Please try again later.",
        status: 503,
        details: {
            reason: "Maintenance",
            estimatedDowntime: "30 minutes",
        },
    },
} as const;

/**
 * Network errors
 */
export const networkErrors = {
    /**
     * Network timeout
     */
    timeout: {
        code: "NETWORK_TIMEOUT",
        message: "Request timed out. Please check your connection and try again.",
        isRetryable: true,
        details: {
            timeoutMs: 30000,
            attempt: 1,
            maxAttempts: 3,
        },
    },

    /**
     * No internet connection
     */
    offline: {
        code: "NETWORK_OFFLINE",
        message: "No internet connection. Please check your network settings.",
        isRetryable: true,
        details: {
            lastOnline: "2024-01-20T09:55:00Z",
        },
    },

    /**
     * DNS resolution failed
     */
    dnsFailure: {
        code: "DNS_FAILURE",
        message: "Could not reach the server. Please try again later.",
        isRetryable: true,
        details: {
            host: "api.vrooli.com",
        },
    },

    /**
     * SSL/TLS error
     */
    sslError: {
        code: "SSL_ERROR",
        message: "Secure connection failed",
        isRetryable: false,
        details: {
            reason: "Certificate validation failed",
        },
    },
} as const;

/**
 * Validation errors
 */
export const validationErrors = {
    /**
     * Form validation errors
     */
    formValidation: {
        code: "VALIDATION_ERROR",
        message: "Please fix the errors below",
        fields: {
            email: {
                type: "format",
                message: "Please enter a valid email address",
            },
            password: {
                type: "minLength",
                message: "Password must be at least 8 characters",
            },
            confirmPassword: {
                type: "match",
                message: "Passwords do not match",
            },
            age: {
                type: "range",
                message: "Age must be between 13 and 120",
            },
        },
    },

    /**
     * File validation errors
     */
    fileValidation: {
        code: "FILE_VALIDATION_ERROR",
        message: "File validation failed",
        details: {
            maxSize: "10MB",
            actualSize: "15MB",
            allowedTypes: ["image/jpeg", "image/png", "image/gif"],
            actualType: "image/bmp",
        },
    },

    /**
     * Data validation errors
     */
    dataValidation: {
        code: "DATA_VALIDATION_ERROR",
        message: "Invalid data format",
        details: {
            expected: "Array",
            received: "Object",
            path: "data.items",
        },
    },
} as const;

/**
 * Business logic errors
 */
export const businessErrors = {
    /**
     * Insufficient permissions
     */
    insufficientPermissions: {
        code: "INSUFFICIENT_PERMISSIONS",
        message: "You need additional permissions to perform this action",
        details: {
            required: ["Admin", "Moderator"],
            current: ["Member"],
            action: "Delete post",
        },
    },

    /**
     * Quota exceeded
     */
    quotaExceeded: {
        code: "QUOTA_EXCEEDED",
        message: "You've reached your usage limit",
        details: {
            limit: 100,
            used: 100,
            resetDate: "2024-02-01T00:00:00Z",
            upgradeUrl: "/settings/billing",
        },
    },

    /**
     * Payment required
     */
    paymentRequired: {
        code: "PAYMENT_REQUIRED",
        message: "This feature requires a premium subscription",
        details: {
            feature: "Advanced Analytics",
            requiredPlan: "Premium",
            currentPlan: "Free",
            upgradeUrl: "/settings/billing",
        },
    },

    /**
     * Account suspended
     */
    accountSuspended: {
        code: "ACCOUNT_SUSPENDED",
        message: "Your account has been suspended",
        details: {
            reason: "Terms of service violation",
            suspendedAt: "2024-01-15T10:00:00Z",
            appealUrl: "/support/appeal",
        },
    },
} as const;

/**
 * Component error states
 */
export const componentErrorStates = {
    /**
     * List/table error
     */
    list: {
        items: [],
        isLoading: false,
        hasError: true,
        error: apiErrors.serverError,
        canRetry: true,
    },

    /**
     * Form error
     */
    form: {
        values: {},
        isLoading: false,
        isSubmitting: false,
        submitError: apiErrors.badRequest,
        fieldErrors: validationErrors.formValidation.fields,
    },

    /**
     * Detail view error
     */
    detail: {
        data: null,
        isLoading: false,
        hasError: true,
        error: apiErrors.notFound,
        canGoBack: true,
    },

    /**
     * Upload error
     */
    upload: {
        files: [],
        isUploading: false,
        hasError: true,
        error: validationErrors.fileValidation,
        failedFiles: ["large-file.zip"],
    },
} as const;

/**
 * Error recovery suggestions
 */
export const errorRecovery = {
    /**
     * Retry suggestions
     */
    retry: {
        immediate: {
            message: "Try again",
            delay: 0,
        },
        delayed: {
            message: "Try again in a few seconds",
            delay: 5000,
        },
        exponential: {
            message: "Retrying automatically...",
            delays: [1000, 2000, 4000, 8000],
        },
    },

    /**
     * Action suggestions
     */
    actions: {
        login: {
            message: "Please log in to continue",
            action: "Login",
            url: "/login",
        },
        upgrade: {
            message: "Upgrade your plan to access this feature",
            action: "View Plans",
            url: "/settings/billing",
        },
        contact: {
            message: "If this problem persists, contact support",
            action: "Contact Support",
            url: "/support",
        },
        refresh: {
            message: "Try refreshing the page",
            action: "Refresh",
            handler: () => window.location.reload(),
        },
    },
} as const;

/**
 * Helper function to create custom error state
 */
export const createErrorState = (
    code: string,
    message: string,
    details?: Record<string, any>,
) => ({
    code,
    message,
    details: details || {},
    timestamp: new Date().toISOString(),
});

/**
 * Helper function to determine if error is retryable
 */
export const isRetryableError = (error: { code: string; status?: number }) => {
    const retryableCodes = [
        "NETWORK_TIMEOUT",
        "NETWORK_OFFLINE",
        "RATE_LIMITED",
        "SERVICE_UNAVAILABLE",
    ];
    const retryableStatuses = [408, 429, 502, 503, 504];

    return (
        retryableCodes.includes(error.code) ||
        (error.status && retryableStatuses.includes(error.status))
    );
};

/**
 * Helper function to get user-friendly error message
 */
export const getUserFriendlyError = (error: { code: string; message: string }) => {
    const friendlyMessages: Record<string, string> = {
        NETWORK_TIMEOUT: "Connection timed out. Please try again.",
        NETWORK_OFFLINE: "You're offline. Check your internet connection.",
        UNAUTHORIZED: "Please log in to continue.",
        FORBIDDEN: "You don't have permission to do that.",
        NOT_FOUND: "We couldn't find what you're looking for.",
        RATE_LIMITED: "You're doing that too often. Please slow down.",
        INTERNAL_SERVER_ERROR: "Something went wrong on our end. Please try again.",
    };

    return friendlyMessages[error.code] || error.message;
};