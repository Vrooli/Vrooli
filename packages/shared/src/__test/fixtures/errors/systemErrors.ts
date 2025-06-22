/**
 * System and infrastructure error fixtures
 * 
 * These fixtures provide system-level error scenarios including
 * database errors, service failures, configuration issues, and infrastructure problems.
 */

import {
    type EnhancedSystemError,
    type ErrorContext,
    type RecoveryStrategy,
    BaseErrorFactory,
} from "./types.js";

// Legacy interface for backward compatibility
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

// Helper function to convert legacy recovery to RecoveryStrategy
function toRecoveryStrategy(legacy?: SystemError["recovery"]): RecoveryStrategy {
    if (!legacy) {
        return { strategy: "fail" };
    }

    if (legacy.retryable) {
        return {
            strategy: "retry",
            attempts: 3,
            delay: 1000,
            backoffMultiplier: 2,
            fallbackAction: legacy.fallback,
        };
    }

    if (legacy.fallback) {
        return {
            strategy: "fallback",
            fallbackAction: legacy.fallback,
        };
    }

    return { strategy: "fail" };
}

/**
 * System Error Factory - Provides enhanced system error creation
 * with telemetry, recovery strategies, and user impact analysis
 */
export class SystemErrorFactory extends BaseErrorFactory<EnhancedSystemError> {
    standard: EnhancedSystemError = {
        code: "SYSTEM_ERROR",
        message: "A system error occurred",
        severity: "error",
        component: "System",
        userImpact: "degraded",
        recovery: { automatic: false, retryable: true },
    };

    withDetails: EnhancedSystemError = {
        code: "SYSTEM_ERROR_DETAILED",
        message: "System error with detailed information",
        severity: "error",
        component: "System",
        details: {
            service: "unknown",
            operation: "unknown",
            errorCode: "UNKNOWN",
            metadata: {
                timestamp: new Date().toISOString(),
                environment: "production",
            },
        },
        userImpact: "degraded",
        recovery: { automatic: false, retryable: true },
        telemetry: {
            traceId: "trace-123",
            spanId: "span-456",
            tags: { service: "system", severity: "error" },
            level: "error",
        },
    };

    variants = {
        // Critical system failures
        criticalFailure: {
            code: "CRITICAL_SYSTEM_FAILURE",
            message: "Critical system failure - immediate attention required",
            severity: "critical" as const,
            component: "Core System",
            userImpact: "blocking" as const,
            recovery: { automatic: false, retryable: false },
        } satisfies EnhancedSystemError,

        // Degraded performance
        degradedPerformance: {
            code: "SYSTEM_DEGRADED",
            message: "System performance is degraded",
            severity: "warning" as const,
            component: "Performance Monitor",
            userImpact: "degraded" as const,
            recovery: {
                automatic: true,
                retryable: true,
                fallback: "reduce_load",
            },
        } satisfies EnhancedSystemError,

        // Monitoring/telemetry error
        telemetryError: {
            code: "TELEMETRY_ERROR",
            message: "Error in monitoring/telemetry system",
            severity: "warning" as const,
            component: "Telemetry",
            userImpact: "none" as const,
            recovery: { automatic: true, retryable: false },
        } satisfies EnhancedSystemError,
    };

    create(overrides?: Partial<EnhancedSystemError>): EnhancedSystemError {
        const base = { ...this.standard };
        if (overrides) {
            Object.assign(base, overrides);
        }
        return base;
    }

    createWithContext(context: ErrorContext): EnhancedSystemError {
        return {
            ...this.withDetails,
            context,
            details: {
                ...this.withDetails.details,
                metadata: {
                    ...this.withDetails.details?.metadata,
                    userId: context.user?.id,
                    operation: context.operation,
                    resourceType: context.resource?.type,
                    resourceId: context.resource?.id,
                },
            },
        };
    }

    /**
     * Convert legacy SystemError to EnhancedSystemError
     */
    fromLegacy(legacy: SystemError): EnhancedSystemError {
        return {
            code: legacy.code,
            message: legacy.message,
            severity: legacy.severity,
            component: legacy.component || "System",
            details: legacy.details,
            recovery: legacy.recovery,
            userImpact: this.determineUserImpact(legacy),
        };
    }

    /**
     * Determine user impact based on legacy error properties
     */
    determineUserImpact(error: SystemError): EnhancedSystemError["userImpact"] {
        if (error.severity === "critical") {
            return "blocking";
        }
        if (error.severity === "warning" || error.recovery?.automatic) {
            return "none";
        }
        return "degraded";
    }
}

// Create factory instance
export const systemErrorFactory = new SystemErrorFactory();

// Export helper function
export { toRecoveryStrategy };

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "degraded",
        } satisfies EnhancedSystemError,

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
            userImpact: "degraded",
        } satisfies EnhancedSystemError,

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,
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
            userImpact: "degraded",
        } satisfies EnhancedSystemError,

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
            userImpact: "none",
        } satisfies EnhancedSystemError,

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
            userImpact: "none",
        } satisfies EnhancedSystemError,
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
            userImpact: "degraded",
        } satisfies EnhancedSystemError,

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "none",
        } satisfies EnhancedSystemError,
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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,
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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "degraded",
        } satisfies EnhancedSystemError,

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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,
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
            userImpact: "blocking",
        } satisfies EnhancedSystemError,

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
            userImpact: "none",
        } satisfies EnhancedSystemError,

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
            userImpact: "none",
        } satisfies EnhancedSystemError,
    },

    // Factory functions
    factories: {
        /**
         * Create a database error
         */
        createDatabaseError: (operation: string, errorCode: string): EnhancedSystemError => ({
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
            userImpact: "degraded",
        }),

        /**
         * Create a service error
         */
        createServiceError: (
            service: string,
            operation: string,
            retryable = true,
        ): EnhancedSystemError => ({
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
            userImpact: "degraded",
        }),

        /**
         * Create a critical system error
         */
        createCriticalError: (
            component: string,
            message: string,
            fallback?: string,
        ): EnhancedSystemError => ({
            code: "CRITICAL_ERROR",
            message,
            severity: "critical",
            component,
            recovery: {
                automatic: false,
                retryable: false,
                ...(fallback && { fallback }),
            },
            userImpact: "blocking",
        }),

        /**
         * Create a custom system error
         */
        createSystemError: (
            code: string,
            message: string,
            severity: EnhancedSystemError["severity"],
            details?: EnhancedSystemError["details"],
            component = "System",
        ): EnhancedSystemError => ({
            code,
            message,
            severity,
            component,
            ...(details && { details }),
            recovery: {
                automatic: false,
                retryable: false,
            },
            userImpact: systemErrorFactory.determineUserImpact({ code, message, severity, component, recovery: { automatic: false, retryable: false } } as SystemError),
        }),
    },
};
