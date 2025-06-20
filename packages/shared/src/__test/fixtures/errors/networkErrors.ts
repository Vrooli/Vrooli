/**
 * Network error fixtures for testing connection and network-related errors
 * 
 * These fixtures provide network error scenarios including timeouts,
 * connection failures, offline states, and other network-related issues.
 */

import type {
    EnhancedNetworkError,
    ErrorContext
} from "./types.js";
import { BaseErrorFactory } from "./types.js";

// Legacy interface for backwards compatibility
export interface NetworkError {
    error: Error;
    display?: {
        title: string;
        message: string;
        icon?: string;
        retry?: boolean;
        retryDelay?: number;
    };
    metadata?: {
        url?: string;
        method?: string;
        duration?: number;
        attempt?: number;
    };
}

// Network error factory implementing the unified pattern
export class NetworkErrorFactory extends BaseErrorFactory<EnhancedNetworkError> {
    standard: EnhancedNetworkError = {
        error: new Error("Network request failed"),
        display: {
            title: "Network Error",
            message: "Unable to complete the request. Please check your connection.",
            retry: true,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "retry",
            attempts: 3,
            delay: 1000,
            backoffMultiplier: 2,
        },
    };

    withDetails: EnhancedNetworkError = {
        error: new Error("ECONNREFUSED"),
        display: {
            title: "Connection Failed",
            message: "Unable to connect to the server. Please check if the service is available.",
            icon: "error_outline",
            retry: true,
            retryDelay: 2000,
        },
        metadata: {
            url: "https://api.vrooli.com",
            method: "GET",
            duration: 5000,
            attempt: 1,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "retry",
            attempts: 3,
            delay: 2000,
            backoffMultiplier: 1.5,
            maxDelay: 10000,
        },
        telemetry: {
            traceId: "net-trace-123",
            spanId: "net-span-456",
            tags: { type: "connection", protocol: "https" },
            level: "error",
        },
        context: {
            operation: "api.fetch",
            resource: { type: "endpoint", id: "/api/v1/users" },
            environment: "production",
            timestamp: new Date(),
        },
    };

    variants = {
        minimal: {
            error: new Error("Network error"),
            userImpact: "blocking" as const,
            recovery: {
                strategy: "fail" as const,
            },
        },
        retryable: {
            error: new Error("Temporary network issue"),
            display: {
                title: "Temporary Issue",
                message: "A temporary network issue occurred. Retrying...",
                retry: true,
            },
            userImpact: "degraded" as const,
            recovery: {
                strategy: "retry" as const,
                attempts: 5,
                delay: 500,
            },
        },
        offline: {
            error: new Error("Network offline"),
            display: {
                title: "You're Offline",
                message: "Check your internet connection and try again.",
                icon: "wifi_off",
                retry: false,
            },
            userImpact: "blocking" as const,
            recovery: {
                strategy: "fail" as const,
            },
        },
    };

    create(overrides?: Partial<EnhancedNetworkError>): EnhancedNetworkError {
        return {
            ...this.standard,
            ...overrides,
        };
    }

    createWithContext(context: ErrorContext): EnhancedNetworkError {
        return {
            ...this.withDetails,
            context: {
                ...this.withDetails.context,
                ...context,
            },
        };
    }

    getDisplayMessage(error: EnhancedNetworkError): string {
        return error.display?.message || error.error.message || "Network error occurred";
    }

    isRetryable(error: EnhancedNetworkError): boolean {
        return error.recovery.strategy === "retry" && (error.display?.retry ?? true);
    }
}

// Create factory instance
export const networkErrorFactory = new NetworkErrorFactory();

export const networkErrorFixtures = {
    // Timeout errors
    timeout: {
        client: {
            error: new Error("Request timeout after 30000ms"),
            display: {
                title: "Request Timed Out",
                message: "The server is taking too long to respond. Please try again.",
                retry: true,
                retryDelay: 5000,
            },
            metadata: {
                duration: 30000,
                attempt: 1,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 3,
                delay: 5000,
                backoffMultiplier: 1.5,
                maxDelay: 30000,
            },
        } satisfies EnhancedNetworkError,

        server: {
            error: new Error("Gateway timeout"),
            display: {
                title: "Server Timeout",
                message: "The server couldn't process your request in time.",
                retry: true,
            },
            metadata: {
                duration: 60000,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 2,
                delay: 10000,
            },
        } satisfies EnhancedNetworkError,

        custom: {
            error: new Error("Operation timed out after 10s"),
            display: {
                title: "Operation Timed Out",
                message: "This is taking longer than expected. Please check your connection.",
                retry: true,
                retryDelay: 2000,
            },
            metadata: {
                duration: 10000,
            },
            userImpact: "degraded",
            recovery: {
                strategy: "retry",
                attempts: 5,
                delay: 2000,
                backoffMultiplier: 2,
            },
        } satisfies EnhancedNetworkError,
    },

    // Connection errors
    connectionRefused: {
        error: new Error("ECONNREFUSED"),
        display: {
            title: "Connection Failed",
            message: "Unable to connect to the server. Please check if the service is available.",
            icon: "error_outline",
            retry: true,
        },
        metadata: {
            url: "https://api.vrooli.com",
            method: "GET",
        },
        userImpact: "blocking",
        recovery: {
            strategy: "retry",
            attempts: 3,
            delay: 2000,
            backoffMultiplier: 2,
        },
    } satisfies EnhancedNetworkError,

    connectionReset: {
        error: new Error("ECONNRESET"),
        display: {
            title: "Connection Lost",
            message: "The connection was unexpectedly closed. Please try again.",
            retry: true,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "retry",
            attempts: 3,
            delay: 1000,
        },
    } satisfies EnhancedNetworkError,

    dnsFailure: {
        error: new Error("ENOTFOUND"),
        display: {
            title: "Server Not Found",
            message: "Could not find the server. Please check your internet connection.",
            icon: "dns",
            retry: true,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "retry",
            attempts: 2,
            delay: 5000,
        },
    } satisfies EnhancedNetworkError,

    // Offline states
    networkOffline: {
        error: new Error("Network request failed"),
        display: {
            title: "You're Offline",
            message: "Check your internet connection and try again.",
            icon: "wifi_off",
            retry: false,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
    } satisfies EnhancedNetworkError,

    browserOffline: {
        error: new Error("navigator.onLine is false"),
        display: {
            title: "No Internet Connection",
            message: "Please connect to the internet to continue.",
            icon: "signal_wifi_off",
            retry: false,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
    } satisfies EnhancedNetworkError,

    // SSL/TLS errors
    sslError: {
        error: new Error("SSL_ERROR_BAD_CERT_DOMAIN"),
        display: {
            title: "Security Error",
            message: "There was a problem establishing a secure connection.",
            icon: "security",
            retry: false,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
    } satisfies EnhancedNetworkError,

    certificateExpired: {
        error: new Error("CERT_HAS_EXPIRED"),
        display: {
            title: "Certificate Error",
            message: "The server's security certificate has expired.",
            icon: "security",
            retry: false,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
    } satisfies EnhancedNetworkError,

    // Proxy errors
    proxyError: {
        error: new Error("PROXY_CONNECTION_FAILED"),
        display: {
            title: "Proxy Error",
            message: "Unable to connect through the proxy server.",
            retry: true,
        },
        metadata: {
            url: "proxy.company.com:8080",
        },
        userImpact: "blocking",
        recovery: {
            strategy: "retry",
            attempts: 2,
            delay: 3000,
        },
    } satisfies EnhancedNetworkError,

    // CORS errors
    corsError: {
        error: new Error("CORS policy: No 'Access-Control-Allow-Origin' header"),
        display: {
            title: "Access Denied",
            message: "The server is blocking requests from this application.",
            retry: false,
        },
        userImpact: "blocking",
        recovery: {
            strategy: "fail",
        },
    } satisfies EnhancedNetworkError,

    // Network quality issues
    slowConnection: {
        error: new Error("Slow network detected"),
        display: {
            title: "Slow Connection",
            message: "Your internet connection is slow. Some features may not work properly.",
            icon: "network_check",
            retry: true,
            retryDelay: 10000,
        },
        metadata: {
            duration: 45000,
        },
        userImpact: "degraded",
        recovery: {
            strategy: "retry",
            attempts: 2,
            delay: 10000,
        },
    } satisfies EnhancedNetworkError,

    packetLoss: {
        error: new Error("High packet loss detected"),
        display: {
            title: "Connection Unstable",
            message: "Your connection is unstable. Please try again.",
            icon: "warning",
            retry: true,
        },
        userImpact: "degraded",
        recovery: {
            strategy: "retry",
            attempts: 5,
            delay: 2000,
            backoffMultiplier: 1.5,
        },
    } satisfies EnhancedNetworkError,

    // Service specific
    cdnError: {
        error: new Error("CDN unreachable"),
        display: {
            title: "Content Delivery Error",
            message: "Unable to load resources. Please refresh the page.",
            retry: true,
        },
        metadata: {
            url: "cdn.vrooli.com",
        },
        userImpact: "degraded",
        recovery: {
            strategy: "retry",
            attempts: 3,
            delay: 2000,
        },
    } satisfies EnhancedNetworkError,

    websocketError: {
        error: new Error("WebSocket connection failed"),
        display: {
            title: "Real-time Connection Failed",
            message: "Unable to establish real-time connection. Some features may be limited.",
            icon: "sync_disabled",
            retry: true,
            retryDelay: 3000,
        },
        userImpact: "degraded",
        recovery: {
            strategy: "retry",
            attempts: 5,
            delay: 3000,
            backoffMultiplier: 1.5,
            maxDelay: 15000,
        },
    } satisfies EnhancedNetworkError,

    // Factory functions
    factories: {
        /**
         * Create a timeout error with custom duration
         */
        createTimeoutError: (durationMs: number): EnhancedNetworkError => ({
            error: new Error(`Request timeout after ${durationMs}ms`),
            display: {
                title: "Request Timed Out",
                message: `The operation took longer than ${durationMs / 1000} seconds.`,
                retry: true,
            },
            metadata: {
                duration: durationMs,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 3,
                delay: Math.min(durationMs / 10, 5000),
                backoffMultiplier: 1.5,
            },
        }),

        /**
         * Create a connection error for a specific host
         */
        createConnectionError: (host: string, port?: number): EnhancedNetworkError => ({
            error: new Error(`ECONNREFUSED ${host}${port ? `:${port}` : ""}`),
            display: {
                title: "Connection Failed",
                message: `Could not connect to ${host}`,
                retry: true,
            },
            metadata: {
                url: `https://${host}${port ? `:${port}` : ""}`,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 3,
                delay: 2000,
                backoffMultiplier: 2,
            },
        }),

        /**
         * Create a custom network error
         */
        createNetworkError: (
            message: string,
            displayTitle: string,
            displayMessage: string,
            retry = true,
        ): EnhancedNetworkError => ({
            error: new Error(message),
            display: {
                title: displayTitle,
                message: displayMessage,
                retry,
            },
            userImpact: retry ? "degraded" : "blocking",
            recovery: {
                strategy: retry ? "retry" : "fail",
                attempts: retry ? 3 : undefined,
                delay: retry ? 1000 : undefined,
            },
        }),

        /**
         * Create a network error with retry configuration
         */
        createRetryableError: (
            error: string,
            maxRetries: number,
            currentAttempt: number,
        ): EnhancedNetworkError => ({
            error: new Error(error),
            display: {
                title: "Network Error",
                message: `Attempt ${currentAttempt} of ${maxRetries} failed. ${currentAttempt < maxRetries ? "Retrying..." : "Please try again later."
                    }`,
                retry: currentAttempt < maxRetries,
                retryDelay: Math.min(1000 * Math.pow(2, currentAttempt), 30000),
            },
            metadata: {
                attempt: currentAttempt,
            },
            userImpact: currentAttempt < maxRetries ? "degraded" : "blocking",
            recovery: {
                strategy: currentAttempt < maxRetries ? "retry" : "fail",
                attempts: maxRetries - currentAttempt,
                delay: Math.min(1000 * Math.pow(2, currentAttempt), 30000),
                backoffMultiplier: 2,
            },
        }),
    },
};
