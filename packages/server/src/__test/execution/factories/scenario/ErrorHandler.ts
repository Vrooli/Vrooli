/**
 * Error Handler
 * 
 * Provides comprehensive error handling for scenario execution
 */

import { ExecutionTestError, SchemaValidationError, MockConfigurationError, ScenarioExecutionError } from "../../types.js";

export interface ErrorContext {
    scenarioName: string;
    operation: string;
    startTime: Date;
    metadata?: Record<string, unknown>;
}

export interface ErrorHandlerOptions {
    maxRetries?: number;
    retryDelay?: number;
    timeoutMs?: number;
    suppressErrors?: boolean;
    onError?: (error: Error, context: ErrorContext) => Promise<void>;
}

export class ErrorHandler {
    private options: ErrorHandlerOptions;
    private errorHistory: Array<{
        error: Error;
        context: ErrorContext;
        timestamp: Date;
    }> = [];

    constructor(options: ErrorHandlerOptions = {}) {
        this.options = {
            maxRetries: 3,
            retryDelay: 1000,
            timeoutMs: 30000,
            suppressErrors: false,
            ...options,
        };
    }

    async withErrorHandling<T>(
        operation: () => Promise<T>,
        context: ErrorContext,
    ): Promise<T> {
        const startTime = Date.now();
        let lastError: Error | null = null;
        let attempt = 0;

        while (attempt <= (this.options.maxRetries || 0)) {
            try {
                // Check timeout
                if (Date.now() - startTime > (this.options.timeoutMs || 30000)) {
                    throw new ExecutionTestError(
                        `Operation timed out after ${this.options.timeoutMs}ms`,
                        "OPERATION_TIMEOUT",
                        { operation: context.operation, attempt },
                    );
                }

                const result = await operation();
                return result;
            } catch (error) {
                lastError = error as Error;
                attempt++;

                // Log error
                this.logError(lastError, context);

                // Call error handler if provided
                if (this.options.onError) {
                    await this.options.onError(lastError, context);
                }

                // Don't retry on these error types
                if (
                    lastError instanceof SchemaValidationError ||
                    lastError instanceof MockConfigurationError ||
                    lastError.message.includes("RESOURCE_LIMIT_EXCEEDED")
                ) {
                    break;
                }

                // Wait before retry
                if (attempt <= (this.options.maxRetries || 0)) {
                    await this.delay(this.options.retryDelay || 1000);
                }
            }
        }

        // If we get here, all retries failed
        if (this.options.suppressErrors) {
            console.error(`Operation failed after ${attempt} attempts:`, lastError);
            return null as T;
        }

        throw this.enrichError(lastError!, context, attempt);
    }

    async withTimeout<T>(
        operation: () => Promise<T>,
        timeoutMs: number,
        context: ErrorContext,
    ): Promise<T> {
        const timeoutPromise = new Promise<never>((_, reject) => {
            setTimeout(() => {
                reject(new ExecutionTestError(
                    `Operation timed out after ${timeoutMs}ms`,
                    "OPERATION_TIMEOUT",
                    { operation: context.operation },
                ));
            }, timeoutMs);
        });

        try {
            return await Promise.race([operation(), timeoutPromise]);
        } catch (error) {
            this.logError(error as Error, context);
            throw this.enrichError(error as Error, context, 1);
        }
    }

    createErrorBoundary<T>(
        operation: () => Promise<T>,
        context: ErrorContext,
        fallback?: () => Promise<T>,
    ): () => Promise<T> {
        return async () => {
            try {
                return await operation();
            } catch (error) {
                this.logError(error as Error, context);
                
                if (fallback) {
                    try {
                        return await fallback();
                    } catch (fallbackError) {
                        throw this.enrichError(fallbackError as Error, context, 1);
                    }
                }
                
                throw this.enrichError(error as Error, context, 1);
            }
        };
    }

    getErrorHistory(): Array<{
        error: Error;
        context: ErrorContext;
        timestamp: Date;
    }> {
        return [...this.errorHistory];
    }

    clearErrorHistory(): void {
        this.errorHistory = [];
    }

    private logError(error: Error, context: ErrorContext): void {
        this.errorHistory.push({
            error,
            context,
            timestamp: new Date(),
        });

        // Keep only last 100 errors
        if (this.errorHistory.length > 100) {
            this.errorHistory.shift();
        }
    }

    private enrichError(error: Error, context: ErrorContext, attempts: number): Error {
        if (error instanceof ExecutionTestError) {
            return error;
        }

        return new ScenarioExecutionError(
            `Operation '${context.operation}' failed after ${attempts} attempts: ${error.message}`,
            context.scenarioName,
        );
    }

    private delay(ms: number): Promise<void> {
        return new Promise(resolve => setTimeout(resolve, ms));
    }
}

// Utility function to create error boundaries
export function createErrorBoundary<T extends unknown[], R>(
    fn: (...args: T) => Promise<R>,
    context: ErrorContext,
    options: ErrorHandlerOptions = {},
): (...args: T) => Promise<R> {
    const handler = new ErrorHandler(options);
    
    return async (...args: T): Promise<R> => {
        return handler.withErrorHandling(
            () => fn(...args),
            context,
        );
    };
}
