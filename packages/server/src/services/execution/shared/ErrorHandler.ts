import { EventTypes } from "@vrooli/shared";
import { logger } from "../../../events/logger.js";
import { EventPublisher } from "../../events/publisher.js";

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
    reason?: string;
}

export interface ErrorHandlerConfig {
    /** Name of the component that the error handler is for */
    component: string;
    /** Swarm ID to use for error events */
    chatId?: string | null;
    /** Run ID to use for error events, if no swarm is available */
    runId?: string | null;
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

    constructor(config: ErrorHandlerConfig) {
        this.config = {
            component: config.component,
            logByDefault: config.logByDefault ?? true,
            publishByDefault: config.publishByDefault ?? true,
            chatId: config.chatId ?? null,
            runId: config.runId ?? null,
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
                    logger.info(`[${context.component}] Operation succeeded after ${attempt} retries`, {
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

            logger.warn(`[${context.component}] Retrying operation after ${delay}ms`, {
                operation: context.operation,
                attempt: attempt + 1,
                maxRetries,
                error: lastError.message,
            });

            await new Promise(resolve => setTimeout(resolve, delay));
        }

        if (!lastError) {
            throw new Error("Unexpected: No error recorded in error handling");
        }
        return { success: false, error: lastError };
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
                error: context.reason || errorObj.message,
                errorName: errorObj.name,
                ...context.metadata,
            };

            if (this.config.includeStackTrace && errorObj.stack) {
                logData.stack = errorObj.stack;
            }

            logger.error(`[${context.component}] ${context.operation} failed`, logData);
        }

        // Publish error event if enabled
        if ((context.publishError ?? this.config.publishByDefault)) {
            try {
                const { proceed, reason } = await EventPublisher.emit(EventTypes.SYSTEM.ERROR, {
                    chatId: this.config.chatId || undefined,
                    runId: this.config.runId || undefined,
                    component: context.component,
                    operation: context.operation,
                    error: errorObj,
                    context: context.metadata,
                });

                if (!proceed) {
                    // Error events being blocked is concerning - log it
                    logger.warn(`[${context.component}] Error event blocked by system`, {
                        originalError: errorObj.message,
                        blockReason: reason,
                        operation: context.operation,
                    });
                }
            } catch (publishError) {
                logger.error(`[${context.component}] Failed to publish error event to unified system`, {
                    originalError: errorObj.message,
                    publishError: publishError instanceof Error ? publishError.message : String(publishError),
                });
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
}
