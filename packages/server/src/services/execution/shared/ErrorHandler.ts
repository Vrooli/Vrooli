import { nanoid } from "nanoid";
import { type Logger } from "winston";
import { type IEventBus, EventUtils, getUnifiedEventSystem } from "../../events/index.js";

export type Result<T, E = Error> =
    | { success: true; data: T }
    | { success: false; error: E };

export interface ErrorContext {
    /** Operation being performed */
    operation: string;
    /** Component or class name */
    component: string;
    /** Additional context data */
    metadata?: Record<string, any>;
    /** Whether to log the error */
    logError?: boolean;
    /** Whether to publish error event */
    publishError?: boolean;
    /** Custom error message */
    errorMessage?: string;
}

export interface ErrorHandlerConfig {
    /** Whether to log all errors by default */
    logByDefault?: boolean;
    /** Whether to publish error events by default */
    publishByDefault?: boolean;
    /** Error types to ignore (won't log or publish) */
    ignoredErrors?: string[];
    /** Whether to include stack traces in logs */
    includeStackTrace?: boolean;
}

/**
 * Centralized error handling utility that provides:
 * - Consistent error wrapping and logging
 * - Result type for better error handling
 * - Automatic error event publishing
 * - Contextual error information
 */
export class ErrorHandler {
    private readonly config: Required<ErrorHandlerConfig>;
    private readonly unifiedEventBus: IEventBus | null;

    constructor(
        private readonly logger: Logger,
        config: ErrorHandlerConfig = {},
    ) {
        this.unifiedEventBus = getUnifiedEventSystem();
        this.config = {
            logByDefault: config.logByDefault ?? true,
            publishByDefault: config.publishByDefault ?? true,
            ignoredErrors: config.ignoredErrors ?? [],
            includeStackTrace: config.includeStackTrace ?? true,
        };
    }

    /**
     * Wrap an async operation with error handling
     */
    async wrap<T>(
        operation: () => Promise<T>,
        context: ErrorContext,
    ): Promise<Result<T>> {
        try {
            const data = await operation();
            return { success: true, data };
        } catch (error) {
            return await this.handleError(error, context);
        }
    }

    /**
     * Wrap a sync operation with error handling
     */
    async wrapSync<T>(
        operation: () => T,
        context: ErrorContext,
    ): Promise<Result<T>> {
        try {
            const data = operation();
            return { success: true, data };
        } catch (error) {
            return await this.handleError(error, context);
        }
    }

    /**
     * Execute operation and throw on error (with logging/publishing)
     */
    async execute<T>(
        operation: () => Promise<T>,
        context: ErrorContext,
    ): Promise<T> {
        const result = await this.wrap(operation, context);
        if (!result.success) {
            const errorResult = result as { success: false; error: Error };
            throw errorResult.error;
        }
        return result.data;
    }

    /**
     * Execute operation with retry logic
     */
    async executeWithRetry<T>(
        operation: () => Promise<T>,
        context: ErrorContext,
        options: {
            maxRetries?: number;
            retryDelay?: number;
            exponentialBackoff?: boolean;
            retryableErrors?: string[];
        } = {},
    ): Promise<Result<T>> {
        const maxRetries = options.maxRetries ?? 3;
        const retryDelay = options.retryDelay ?? 100;
        const exponentialBackoff = options.exponentialBackoff ?? true;
        const retryableErrors = options.retryableErrors ?? ["ECONNREFUSED", "ETIMEDOUT", "ENOTFOUND"];

        let lastError: Error | undefined;

        for (let attempt = 0; attempt <= maxRetries; attempt++) {
            const result = await this.wrap(operation, {
                ...context,
                metadata: {
                    ...context.metadata,
                    attempt: attempt + 1,
                    maxRetries,
                },
            });

            if (result.success) {
                if (attempt > 0) {
                    this.logger.info(`[${context.component}] Operation succeeded after ${attempt} retries`, {
                        operation: context.operation,
                        attempts: attempt + 1,
                    });
                }
                return result;
            }

            const errorResult = result as { success: false; error: Error };
            lastError = errorResult.error;

            // Check if error is retryable
            const isRetryable = retryableErrors.some(code =>
                lastError?.message?.includes(code) ||
                (lastError as any)?.code === code,
            );

            if (!isRetryable || attempt === maxRetries) {
                return result;
            }

            // Calculate delay
            const delay = exponentialBackoff
                ? retryDelay * Math.pow(2, attempt)
                : retryDelay;

            this.logger.warn(`[${context.component}] Retrying operation after ${delay}ms`, {
                operation: context.operation,
                attempt: attempt + 1,
                maxRetries,
                error: lastError.message,
            });

            await new Promise(resolve => setTimeout(resolve, delay));
        }

        return { success: false, error: lastError! };
    }

    /**
     * Handle an error with consistent logging and publishing
     */
    private async handleError<E = Error>(
        error: unknown,
        context: ErrorContext,
    ): Promise<Result<never, E>> {
        const errorObj = this.normalizeError(error);

        // Check if error should be ignored
        if (this.config.ignoredErrors.includes(errorObj.name)) {
            return { success: false, error: errorObj as E };
        }

        // Log error if enabled
        if (context.logError ?? this.config.logByDefault) {
            const logData: Record<string, any> = {
                operation: context.operation,
                error: context.errorMessage || errorObj.message,
                errorName: errorObj.name,
                ...context.metadata,
            };

            if (this.config.includeStackTrace && errorObj.stack) {
                logData.stack = errorObj.stack;
            }

            this.logger.error(`[${context.component}] ${context.operation} failed`, logData);
        }

        // Publish error event if enabled
        if ((context.publishError ?? this.config.publishByDefault)) {
            // Try unified event system first, fallback to legacy
            if (this.unifiedEventBus) {
                try {
                    const event = EventUtils.createBaseEvent(
                        "component.error",
                        {
                            component: context.component,
                            operation: context.operation,
                            error: {
                                name: errorObj.name,
                                message: errorObj.message,
                                stack: errorObj.stack,
                            },
                            ...context.metadata,
                        },
                        EventUtils.createEventSource("cross-cutting", "ErrorHandler", nanoid()),
                        EventUtils.createEventMetadata("reliable", "high"),
                    );
                    await this.unifiedEventBus.publish(event);
                } catch (publishError) {
                    this.logger.error(`[${context.component}] Failed to publish error event to unified system`, {
                        originalError: errorObj.message,
                        publishError: publishError instanceof Error ? publishError.message : String(publishError),
                    });
                }
            } else {
                // No event bus available, error will only be logged
                this.logger.debug("[ErrorHandler] No unified event bus available for error event publication");
            }
        }

        return { success: false, error: errorObj as E };
    }

    /**
     * Normalize various error types to Error objects
     */
    private normalizeError(error: unknown): Error {
        if (error instanceof Error) {
            return error;
        }

        if (typeof error === "string") {
            return new Error(error);
        }

        if (error && typeof error === "object" && "message" in error) {
            const err = new Error(String(error.message));
            if ("name" in error) err.name = String(error.name);
            if ("stack" in error) err.stack = String(error.stack);
            return err;
        }

        return new Error(String(error));
    }

    /**
     * Create a specialized error handler for a component
     */
    createComponentHandler(component: string): ComponentErrorHandler {
        return new ComponentErrorHandler(this, component);
    }

    /**
     * Guard function for narrowing Result types
     */
    static isSuccess<T>(result: Result<T>): result is { success: true; data: T } {
        return result.success;
    }

    /**
     * Guard function for narrowing Result types
     */
    static isError<T, E = Error>(result: Result<T, E>): result is { success: false; error: E } {
        return !result.success;
    }

    /**
     * Unwrap a Result or throw
     */
    static unwrap<T>(result: Result<T>): T {
        if (result.success) {
            return result.data;
        }
        const errorResult = result as { success: false; error: Error };
        throw errorResult.error;
    }

    /**
     * Map over a successful result
     */
    static map<T, U>(result: Result<T>, fn: (data: T) => U): Result<U> {
        if (result.success) {
            try {
                return { success: true, data: fn(result.data) };
            } catch (error) {
                return { success: false, error: error instanceof Error ? error : new Error(String(error)) };
            }
        }
        const errorResult = result as { success: false; error: Error };
        return { success: false, error: errorResult.error };
    }

    /**
     * Async map over a successful result
     */
    static async mapAsync<T, U>(
        result: Result<T>,
        fn: (data: T) => Promise<U>,
    ): Promise<Result<U>> {
        if (result.success) {
            try {
                const data = await fn(result.data);
                return { success: true, data };
            } catch (error) {
                return { success: false, error: error instanceof Error ? error : new Error(String(error)) };
            }
        }
        const errorResult = result as { success: false; error: Error };
        return { success: false, error: errorResult.error };
    }
}

/**
 * Component-specific error handler for cleaner API
 */
export class ComponentErrorHandler {
    constructor(
        private readonly errorHandler: ErrorHandler,
        private readonly component: string,
    ) { }

    async wrap<T>(
        operation: () => Promise<T>,
        operationName: string,
        metadata?: Record<string, any>,
    ): Promise<Result<T>> {
        return this.errorHandler.wrap(operation, {
            operation: operationName,
            component: this.component,
            metadata,
        });
    }

    async execute<T>(
        operation: () => Promise<T>,
        operationName: string,
        metadata?: Record<string, any>,
    ): Promise<T> {
        return this.errorHandler.execute(operation, {
            operation: operationName,
            component: this.component,
            metadata,
        });
    }

    async executeWithRetry<T>(
        operation: () => Promise<T>,
        operationName: string,
        options?: Parameters<ErrorHandler["executeWithRetry"]>[2],
    ): Promise<Result<T>> {
        return this.errorHandler.executeWithRetry(
            operation,
            {
                operation: operationName,
                component: this.component,
            },
            options,
        );
    }
}
