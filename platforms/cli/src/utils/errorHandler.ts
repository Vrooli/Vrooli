import { type Ora } from "ora";
import { output } from "./output.js";
import { AxiosError } from "axios";

// HTTP status codes
const HTTP_STATUS = {
    UNAUTHORIZED: 401,
    FORBIDDEN: 403,
    NOT_FOUND: 404,
    INTERNAL_SERVER_ERROR: 500,
} as const;

/**
 * Standardized error handler for CLI commands
 * Provides consistent error formatting and exit behavior
 */
export class CommandError extends Error {
    constructor(
        message: string,
        public readonly exitCode: number = 1,
        public readonly details?: unknown,
    ) {
        super(message);
        this.name = "CommandError";
    }
}

/**
 * Handle command errors in a consistent way
 * @param error - The error to handle
 * @param spinner - Optional spinner to fail
 * @param context - Optional context about what was being attempted
 */
export function handleCommandError(
    error: unknown, 
    spinner?: Ora,
    context?: string,
): never {
    // Stop spinner if present
    if (spinner) {
        spinner.fail();
    }

    // Determine error message and details
    let message: string;
    let exitCode = 1;
    let showDetails = false;

    if (error instanceof CommandError) {
        message = error.message;
        exitCode = error.exitCode;
        showDetails = !!error.details;
    } else if (error instanceof AxiosError) {
        // Handle API errors specially
        if (error.response) {
            const status = error.response.status;
            const data = error.response.data;
            
            if (status === HTTP_STATUS.UNAUTHORIZED) {
                message = "Authentication failed. Please login again with 'vrooli auth login'";
            } else if (status === HTTP_STATUS.FORBIDDEN) {
                message = "Permission denied. You don't have access to this resource.";
            } else if (status === HTTP_STATUS.NOT_FOUND) {
                message = "Resource not found.";
            } else if (status >= HTTP_STATUS.INTERNAL_SERVER_ERROR) {
                message = "Server error. Please try again later.";
            } else if (data?.message) {
                message = data.message;
            } else {
                message = `API error (${status})`;
            }
            
            if (data?.error) {
                showDetails = true;
            }
        } else if (error.request) {
            message = "Network error. Please check your connection and try again.";
        } else {
            message = error.message;
        }
    } else if (error instanceof Error) {
        message = error.message;
    } else if (typeof error === "string") {
        message = error;
    } else {
        message = "An unexpected error occurred";
        showDetails = true;
    }

    // Add context if provided
    if (context) {
        message = `${context}: ${message}`;
    }

    // Output error
    output.error(message);

    // Show details in debug mode
    if (showDetails && process.env.DEBUG === "true") {
        output.debug("Error details:", error);
    }

    // Exit with appropriate code
    process.exit(exitCode);
}

/**
 * Wrap an async command function with error handling
 */
export function withErrorHandler<T extends (...args: unknown[]) => Promise<unknown>>(
    fn: T,
    context?: string,
): T {
    return (async (...args: Parameters<T>) => {
        try {
            return await fn(...args);
        } catch (error) {
            handleCommandError(error, undefined, context);
        }
    }) as T;
}

/**
 * Create a safe wrapper for command actions
 */
export function createCommandAction<T extends unknown[]>(
    action: (...args: T) => Promise<void>,
    context?: string,
): (...args: T) => Promise<void> {
    return async (...args: T) => {
        try {
            await action(...args);
        } catch (error) {
            handleCommandError(error, undefined, context);
        }
    };
}
