/**
 * Core types and interfaces for the unified error fixture architecture
 * 
 * This file defines the foundational types for all error fixtures,
 * implementing the unified factory pattern described in the README.
 */

// Base error context for rich error scenarios
export interface ErrorContext {
    user?: { id: string; role: string };
    operation?: string;
    resource?: { type: string; id: string };
    environment?: "development" | "staging" | "production";
    timestamp?: Date;
    attempt?: number;
    maxAttempts?: number;
}

// Recovery strategy configuration
export interface RecoveryStrategy {
    strategy: "retry" | "fallback" | "fail" | "circuit-break" | "queue";
    attempts?: number;
    delay?: number;
    backoffMultiplier?: number;
    maxDelay?: number;
    fallbackAction?: string;
    condition?: (error: any, attempt: number) => boolean;
}

// Telemetry integration for error tracking
export interface ErrorTelemetry {
    traceId: string;
    spanId: string;
    tags: Record<string, string>;
    level: "debug" | "info" | "warn" | "error" | "fatal";
}

// Enhanced base interfaces for all error types
export interface EnhancedApiError {
    // Existing API error properties
    status: number;
    code: string;
    message: string;
    details?: any;
    retryAfter?: number;
    limit?: number;
    remaining?: number;
    reset?: string;

    // Enhanced properties
    cause?: Error;
    userImpact: "blocking" | "degraded" | "none";
    recovery: RecoveryStrategy;
    telemetry?: ErrorTelemetry;
    context?: ErrorContext;
}

export interface EnhancedNetworkError {
    // Existing network error properties
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

    // Enhanced properties
    cause?: Error;
    userImpact: "blocking" | "degraded" | "none";
    recovery: RecoveryStrategy;
    telemetry?: ErrorTelemetry;
    context?: ErrorContext;
}

export interface EnhancedValidationError {
    // Existing validation error properties
    fields?: Record<string, string | string[] | undefined>;
    message?: string;

    // Enhanced properties
    cause?: Error;
    userImpact: "blocking" | "degraded" | "none";
    recovery: RecoveryStrategy;
    telemetry?: ErrorTelemetry;
    context?: ErrorContext;
    // Validation-specific enhancements
    fieldPath?: string;
    invalidValue?: any;
    constraint?: string;
    suggestion?: string;
}

export interface EnhancedAuthError {
    // Existing auth error properties
    code: string;
    message: string;
    details?: {
        reason?: string;
        requiredRole?: string;
        requiredPermission?: string;
        expiresAt?: string;
        remainingAttempts?: number;
        lockoutDuration?: number;
        [key: string]: any; // Allow additional properties
    };
    action?: {
        type: string;
        label: string;
        url?: string; // Make url optional
    };

    // Enhanced properties
    cause?: Error;
    userImpact: "blocking" | "degraded" | "none";
    recovery: RecoveryStrategy;
    telemetry?: ErrorTelemetry;
    context?: ErrorContext;
}

export interface EnhancedBusinessError {
    // Existing business error properties
    code: string;
    message: string;
    type: "limit" | "conflict" | "state" | "workflow" | "constraint" | "policy";
    details?: {
        current?: any;
        limit?: any;
        required?: any;
        conflictWith?: any;
        suggestion?: string;
        missingSteps?: string[];
        yourVersion?: string;
        currentVersion?: string;
        [key: string]: any; // Allow additional properties
    };
    userAction?: {
        label: string;
        action: string;
        url?: string;
    };

    // Enhanced properties
    cause?: Error;
    userImpact: "blocking" | "degraded" | "none";
    recovery: RecoveryStrategy;
    telemetry?: ErrorTelemetry;
    context?: ErrorContext;
}

export interface EnhancedSystemError {
    // Existing system error properties
    code: string;
    message: string;
    severity: "critical" | "error" | "warning";
    component: string;
    details?: {
        service?: string;
        operation?: string;
        errorCode?: string;
        stackTrace?: string;
        metadata?: Record<string, any>;
    };
    recovery?: {
        automatic?: boolean;
        retryable?: boolean;
        estimatedRecovery?: string;
        fallback?: string;
    };

    // Enhanced properties
    cause?: Error;
    userImpact: "blocking" | "degraded" | "none";
    telemetry?: ErrorTelemetry;
    context?: ErrorContext;
}

// Type-safe error factory pattern interface
export interface ErrorFixtureFactory<TError> {
    // Standard variants
    standard: TError;              // Most common error scenario
    withDetails: TError;           // Error with detailed information
    variants: {                    // Common error variations
        [key: string]: TError;
    };

    // Factory methods
    create: (overrides?: Partial<TError>) => TError;
    createWithContext: (context: ErrorContext) => TError;

    // Testing helpers
    isRetryable: (error: TError) => boolean;
    getDisplayMessage: (error: TError) => string;
    getStatusCode?: (error: TError) => number;

    // Simulation helpers
    simulateRecovery: (error: TError) => TError | null;
    escalate: (error: TError) => TError;
}

// Recovery scenario for comprehensive testing
export interface RecoveryScenario<TError> {
    error: TError;
    recoverySteps: RecoveryStep[];
    expectedOutcome: "success" | "partial" | "failure";
    fallbackBehavior?: string;
    description?: string;
}

export interface RecoveryStep {
    action: "retry" | "backoff" | "circuit-break" | "fallback" | "escalate";
    delay?: number;
    condition?: (error: any, attempt: number) => boolean;
    transform?: (error: any) => any;
    description?: string;
}

// Error composition types
export interface ComposedError {
    primary: any;
    causes: any[];
    composition: "chain" | "cascade" | "context" | "timed";
    metadata: {
        created: Date;
        severity: "low" | "medium" | "high" | "critical";
        tags: string[];
    };
}

export interface ErrorSequence {
    errors: any[];
    timing: "sequential" | "parallel" | "cascading";
    options: {
        failFast?: boolean;
        continueOnError?: boolean;
        timeout?: number;
    };
}

export interface CascadeOptions {
    timing: "sequential" | "parallel";
    failureMode: "fail-fast" | "continue" | "retry-each";
    maxAttempts?: number;
    delay?: number;
}

export interface TimingOptions {
    delay?: number;
    duration?: number;
    schedule?: Date[];
    pattern?: "immediate" | "delayed" | "scheduled" | "exponential";
}

// Utility types for error factories
export type ErrorVariants<T> = {
    minimal: T;
    standard: T;
    withDetails: T;
    [key: string]: T;
};

export type ErrorFactories<T> = {
    create: (overrides?: Partial<T>) => T;
    createWithContext: (context: ErrorContext) => T;
    createComposed: (primary: T, causes: any[]) => ComposedError;
    createRecoveryScenario: (error: T, steps: RecoveryStep[]) => RecoveryScenario<T>;
};

// Base factory class that can be extended
export abstract class BaseErrorFactory<TError> implements ErrorFixtureFactory<TError> {
    abstract standard: TError;
    abstract withDetails: TError;
    abstract variants: { [key: string]: TError };

    abstract create(overrides?: Partial<TError>): TError;
    abstract createWithContext(context: ErrorContext): TError;

    isRetryable(error: TError): boolean {
        return (error as any).recovery?.strategy === "retry";
    }

    getDisplayMessage(error: TError): string {
        return (error as any).message || "An error occurred";
    }

    getStatusCode(error: TError): number {
        return (error as any).status || 500;
    }

    simulateRecovery(error: TError): TError | null {
        const recovery = (error as any).recovery;
        if (!recovery || recovery.strategy === "fail") {
            return null;
        }
        return error; // Simulate successful recovery
    }

    escalate(error: TError): TError {
        const escalated = { ...error } as any;
        escalated.severity = "critical";
        escalated.userImpact = "blocking";
        return escalated;
    }
}
