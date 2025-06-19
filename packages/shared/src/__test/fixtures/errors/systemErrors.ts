/**
 * System and infrastructure error fixtures
 * 
 * These fixtures provide system-level error scenarios including
 * database errors, service failures, configuration issues, and infrastructure problems.
 */

export interface SystemError {
    code: string;
    message: string;
    severity: "critical" | "error" | "warning";
    component?: string;
    details?: {
        service?: string;
        operation?: string;
        errorCode?: string;
        stackTrace?: string;
        metadata?: Record<string, any>;
    };
    recovery?: {
        automatic: boolean;
        retryable: boolean;
        estimatedRecovery?: string;
        fallback?: string;
    };
}

export const systemErrorFixtures = {
    // Database errors
    database: {
        connectionLost: {
            code: "DATABASE_CONNECTION_LOST",
            message: "Lost connection to the database",
            severity: "critical",
            component: "PostgreSQL",
            details: {
                service: "primary_db",
                operation: "query",
                errorCode: "ECONNRESET",
            },
            recovery: {
                automatic: true,
                retryable: true,
                estimatedRecovery: new Date(Date.now() + 30000).toISOString(),
            },
        } satisfies SystemError,

        queryTimeout: {
            code: "DATABASE_QUERY_TIMEOUT",
            message: "Database query exceeded timeout limit",
            severity: "error",
            component: "PostgreSQL",
            details: {
                operation: "complex_aggregation",
                errorCode: "57014",
                metadata: {
                    query: "SELECT ... (truncated)",
                    duration: 30000,
                    timeout: 30000,
                },
            },
            recovery: {
                automatic: false,
                retryable: true,
            },
        } satisfies SystemError,

        deadlock: {
            code: "DATABASE_DEADLOCK",
            message: "Database deadlock detected",
            severity: "error",
            component: "PostgreSQL",
            details: {
                operation: "transaction",
                errorCode: "40P01",
                metadata: {
                    tables: ["users", "projects"],
                    transactionId: "txn_123456",
                },
            },
            recovery: {
                automatic: false,
                retryable: true,
            },
        } satisfies SystemError,

        diskFull: {
            code: "DATABASE_DISK_FULL",
            message: "Database disk space exhausted",
            severity: "critical",
            component: "PostgreSQL",
            details: {
                service: "primary_db",
                errorCode: "53100",
                metadata: {
                    availableSpace: "0 MB",
                    requiredSpace: "100 MB",
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
                fallback: "read_only_mode",
            },
        } satisfies SystemError,
    },

    // Cache errors
    cache: {
        redisDown: {
            code: "REDIS_UNAVAILABLE",
            message: "Cache service is unavailable",
            severity: "error",
            component: "Redis",
            details: {
                service: "redis_primary",
                operation: "get",
                errorCode: "ECONNREFUSED",
            },
            recovery: {
                automatic: true,
                retryable: true,
                fallback: "bypass_cache",
            },
        } satisfies SystemError,

        cacheMemoryFull: {
            code: "CACHE_MEMORY_FULL",
            message: "Cache memory limit reached",
            severity: "warning",
            component: "Redis",
            details: {
                service: "redis_primary",
                metadata: {
                    maxMemory: "4GB",
                    usedMemory: "4GB",
                    evictionPolicy: "allkeys-lru",
                },
            },
            recovery: {
                automatic: true,
                retryable: true,
                fallback: "evict_old_keys",
            },
        } satisfies SystemError,

        serializationError: {
            code: "CACHE_SERIALIZATION_ERROR",
            message: "Failed to serialize data for caching",
            severity: "error",
            component: "Redis",
            details: {
                operation: "set",
                metadata: {
                    key: "user:123:profile",
                    error: "Circular reference detected",
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
                fallback: "skip_cache",
            },
        } satisfies SystemError,
    },

    // External service errors
    externalService: {
        aiServiceDown: {
            code: "AI_SERVICE_UNAVAILABLE",
            message: "AI service is temporarily unavailable",
            severity: "error",
            component: "LLM Service",
            details: {
                service: "openai",
                operation: "completion",
                errorCode: "503",
            },
            recovery: {
                automatic: true,
                retryable: true,
                estimatedRecovery: new Date(Date.now() + 300000).toISOString(),
                fallback: "queue_for_later",
            },
        } satisfies SystemError,

        paymentGatewayError: {
            code: "PAYMENT_GATEWAY_ERROR",
            message: "Payment processing service error",
            severity: "critical",
            component: "Stripe",
            details: {
                service: "stripe",
                operation: "charge",
                errorCode: "api_connection_error",
            },
            recovery: {
                automatic: false,
                retryable: true,
                fallback: "alternative_gateway",
            },
        } satisfies SystemError,

        emailServiceError: {
            code: "EMAIL_SERVICE_ERROR",
            message: "Failed to send email",
            severity: "warning",
            component: "SendGrid",
            details: {
                service: "sendgrid",
                operation: "send",
                metadata: {
                    to: "user@example.com",
                    template: "welcome_email",
                    error: "Rate limit exceeded",
                },
            },
            recovery: {
                automatic: true,
                retryable: true,
                fallback: "queue_email",
            },
        } satisfies SystemError,
    },

    // Configuration errors
    configuration: {
        missingEnvVar: {
            code: "MISSING_ENV_VARIABLE",
            message: "Required environment variable is not set",
            severity: "critical",
            component: "Configuration",
            details: {
                metadata: {
                    variable: "DATABASE_URL",
                    required: true,
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
            },
        } satisfies SystemError,

        invalidConfig: {
            code: "INVALID_CONFIGURATION",
            message: "System configuration is invalid",
            severity: "critical",
            component: "Configuration",
            details: {
                metadata: {
                    file: "config/production.json",
                    error: "JSON parse error at line 42",
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
                fallback: "default_config",
            },
        } satisfies SystemError,

        secretsManagerError: {
            code: "SECRETS_MANAGER_ERROR",
            message: "Unable to retrieve secrets",
            severity: "critical",
            component: "AWS Secrets Manager",
            details: {
                service: "aws_secrets_manager",
                operation: "getSecretValue",
                errorCode: "AccessDeniedException",
            },
            recovery: {
                automatic: false,
                retryable: true,
            },
        } satisfies SystemError,
    },

    // Infrastructure errors
    infrastructure: {
        memoryExhausted: {
            code: "MEMORY_EXHAUSTED",
            message: "System memory exhausted",
            severity: "critical",
            component: "System",
            details: {
                metadata: {
                    availableMemory: "10MB",
                    totalMemory: "8GB",
                    process: "node",
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
                fallback: "restart_process",
            },
        } satisfies SystemError,

        diskSpaceError: {
            code: "DISK_SPACE_ERROR",
            message: "Insufficient disk space",
            severity: "critical",
            component: "File System",
            details: {
                operation: "write",
                metadata: {
                    path: "/var/log",
                    availableSpace: "100MB",
                    requiredSpace: "500MB",
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
                fallback: "cleanup_old_files",
            },
        } satisfies SystemError,

        cpuOverload: {
            code: "CPU_OVERLOAD",
            message: "CPU usage critically high",
            severity: "warning",
            component: "System",
            details: {
                metadata: {
                    cpuUsage: "95%",
                    loadAverage: [15.2, 14.8, 13.9],
                    cores: 4,
                },
            },
            recovery: {
                automatic: true,
                retryable: true,
                fallback: "throttle_requests",
            },
        } satisfies SystemError,

        networkPartition: {
            code: "NETWORK_PARTITION",
            message: "Network partition detected",
            severity: "critical",
            component: "Network",
            details: {
                metadata: {
                    affectedServices: ["db-primary", "cache-1", "api-2"],
                    partition: "datacenter-east",
                },
            },
            recovery: {
                automatic: true,
                retryable: true,
                estimatedRecovery: new Date(Date.now() + 600000).toISOString(),
                fallback: "failover_to_west",
            },
        } satisfies SystemError,
    },

    // Process errors
    process: {
        uncaughtException: {
            code: "UNCAUGHT_EXCEPTION",
            message: "An unexpected error occurred",
            severity: "critical",
            component: "Application",
            details: {
                stackTrace: "Error: Unexpected token...\n    at Object.parse...",
                metadata: {
                    processId: process.pid,
                    timestamp: new Date().toISOString(),
                },
            },
            recovery: {
                automatic: false,
                retryable: false,
                fallback: "restart_required",
            },
        } satisfies SystemError,

        workerCrash: {
            code: "WORKER_CRASH",
            message: "Background worker process crashed",
            severity: "error",
            component: "Worker",
            details: {
                service: "job_processor",
                metadata: {
                    workerId: "worker_3",
                    exitCode: 1,
                    signal: "SIGTERM",
                },
            },
            recovery: {
                automatic: true,
                retryable: true,
                fallback: "spawn_new_worker",
            },
        } satisfies SystemError,

        zombieProcess: {
            code: "ZOMBIE_PROCESS",
            message: "Zombie process detected",
            severity: "warning",
            component: "Process Manager",
            details: {
                metadata: {
                    pid: 12345,
                    ppid: 12340,
                    state: "Z",
                },
            },
            recovery: {
                automatic: true,
                retryable: false,
                fallback: "cleanup_process",
            },
        } satisfies SystemError,
    },

    // Factory functions
    factories: {
        /**
         * Create a database error
         */
        createDatabaseError: (operation: string, errorCode: string): SystemError => ({
            code: "DATABASE_ERROR",
            message: `Database error during ${operation}`,
            severity: "error",
            component: "Database",
            details: {
                operation,
                errorCode,
            },
            recovery: {
                automatic: false,
                retryable: true,
            },
        }),

        /**
         * Create a service error
         */
        createServiceError: (
            service: string,
            operation: string,
            retryable = true,
        ): SystemError => ({
            code: `${service.toUpperCase()}_ERROR`,
            message: `${service} service error`,
            severity: "error",
            component: service,
            details: {
                service,
                operation,
            },
            recovery: {
                automatic: false,
                retryable,
            },
        }),

        /**
         * Create a critical system error
         */
        createCriticalError: (
            component: string,
            message: string,
            fallback?: string,
        ): SystemError => ({
            code: "CRITICAL_ERROR",
            message,
            severity: "critical",
            component,
            recovery: {
                automatic: false,
                retryable: false,
                ...(fallback && { fallback }),
            },
        }),

        /**
         * Create a custom system error
         */
        createSystemError: (
            code: string,
            message: string,
            severity: SystemError["severity"],
            details?: SystemError["details"],
        ): SystemError => ({
            code,
            message,
            severity,
            ...(details && { details }),
            recovery: {
                automatic: false,
                retryable: false,
            },
        }),
    },
};