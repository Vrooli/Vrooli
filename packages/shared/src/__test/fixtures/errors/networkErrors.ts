/**
 * Network error fixtures for testing connection and network-related errors
 * 
 * These fixtures provide network error scenarios including timeouts,
 * connection failures, offline states, and other network-related issues.
 */

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
        } satisfies NetworkError,

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
        } satisfies NetworkError,

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
        } satisfies NetworkError,
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
    } satisfies NetworkError,

    connectionReset: {
        error: new Error("ECONNRESET"),
        display: {
            title: "Connection Lost",
            message: "The connection was unexpectedly closed. Please try again.",
            retry: true,
        },
    } satisfies NetworkError,

    dnsFailure: {
        error: new Error("ENOTFOUND"),
        display: {
            title: "Server Not Found",
            message: "Could not find the server. Please check your internet connection.",
            icon: "dns",
            retry: true,
        },
    } satisfies NetworkError,

    // Offline states
    networkOffline: {
        error: new Error("Network request failed"),
        display: {
            title: "You're Offline",
            message: "Check your internet connection and try again.",
            icon: "wifi_off",
            retry: false,
        },
    } satisfies NetworkError,

    browserOffline: {
        error: new Error("navigator.onLine is false"),
        display: {
            title: "No Internet Connection",
            message: "Please connect to the internet to continue.",
            icon: "signal_wifi_off",
            retry: false,
        },
    } satisfies NetworkError,

    // SSL/TLS errors
    sslError: {
        error: new Error("SSL_ERROR_BAD_CERT_DOMAIN"),
        display: {
            title: "Security Error",
            message: "There was a problem establishing a secure connection.",
            icon: "security",
            retry: false,
        },
    } satisfies NetworkError,

    certificateExpired: {
        error: new Error("CERT_HAS_EXPIRED"),
        display: {
            title: "Certificate Error",
            message: "The server's security certificate has expired.",
            icon: "security",
            retry: false,
        },
    } satisfies NetworkError,

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
    } satisfies NetworkError,

    // CORS errors
    corsError: {
        error: new Error("CORS policy: No 'Access-Control-Allow-Origin' header"),
        display: {
            title: "Access Denied",
            message: "The server is blocking requests from this application.",
            retry: false,
        },
    } satisfies NetworkError,

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
    } satisfies NetworkError,

    packetLoss: {
        error: new Error("High packet loss detected"),
        display: {
            title: "Connection Unstable",
            message: "Your connection is unstable. Please try again.",
            icon: "warning",
            retry: true,
        },
    } satisfies NetworkError,

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
    } satisfies NetworkError,

    websocketError: {
        error: new Error("WebSocket connection failed"),
        display: {
            title: "Real-time Connection Failed",
            message: "Unable to establish real-time connection. Some features may be limited.",
            icon: "sync_disabled",
            retry: true,
            retryDelay: 3000,
        },
    } satisfies NetworkError,

    // Factory functions
    factories: {
        /**
         * Create a timeout error with custom duration
         */
        createTimeoutError: (durationMs: number): NetworkError => ({
            error: new Error(`Request timeout after ${durationMs}ms`),
            display: {
                title: "Request Timed Out",
                message: `The operation took longer than ${durationMs / 1000} seconds.`,
                retry: true,
            },
            metadata: {
                duration: durationMs,
            },
        }),

        /**
         * Create a connection error for a specific host
         */
        createConnectionError: (host: string, port?: number): NetworkError => ({
            error: new Error(`ECONNREFUSED ${host}${port ? `:${port}` : ""}`),
            display: {
                title: "Connection Failed",
                message: `Could not connect to ${host}`,
                retry: true,
            },
            metadata: {
                url: `https://${host}${port ? `:${port}` : ""}`,
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
        ): NetworkError => ({
            error: new Error(message),
            display: {
                title: displayTitle,
                message: displayMessage,
                retry,
            },
        }),

        /**
         * Create a network error with retry configuration
         */
        createRetryableError: (
            error: string,
            maxRetries: number,
            currentAttempt: number,
        ): NetworkError => ({
            error: new Error(error),
            display: {
                title: "Network Error",
                message: `Attempt ${currentAttempt} of ${maxRetries} failed. ${
                    currentAttempt < maxRetries ? "Retrying..." : "Please try again later."
                }`,
                retry: currentAttempt < maxRetries,
                retryDelay: Math.min(1000 * Math.pow(2, currentAttempt), 30000),
            },
            metadata: {
                attempt: currentAttempt,
            },
        }),
    },
};