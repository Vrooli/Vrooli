import { logger } from "../../events/logger.js";

// Timeout constants
const MINUTES_5 = 5;
const SECONDS_PER_MINUTE = 60;
const MS_PER_SECOND = 1000;
const DEFAULT_STREAM_TIMEOUT_MS = MINUTES_5 * SECONDS_PER_MINUTE * MS_PER_SECOND;

/**
 * Configuration for stream timeout wrapper
 */
export interface StreamTimeoutConfig {
    /** Timeout in milliseconds */
    timeoutMs?: number;
    /** Name of the service for logging */
    serviceName: string;
    /** Optional model name for logging */
    modelName?: string;
    /** Optional abort signal to coordinate cancellation */
    signal?: AbortSignal;
}

/**
 * Error thrown when a stream times out
 */
export class StreamTimeoutError extends Error {
    constructor(serviceName: string, timeoutMs: number, modelName?: string) {
        super(`Stream timeout after ${timeoutMs}ms in ${serviceName}${modelName ? ` (model: ${modelName})` : ""}`);
        this.name = "StreamTimeoutError";
    }
}

/**
 * Wraps an async generator with timeout protection
 * 
 * @param generator - The async generator to wrap
 * @param config - Timeout configuration
 * @returns A new async generator that enforces the timeout
 */
export async function* withStreamTimeout<T>(
    generator: AsyncGenerator<T>,
    config: StreamTimeoutConfig,
): AsyncGenerator<T> {
    const { 
        timeoutMs = DEFAULT_STREAM_TIMEOUT_MS, 
        serviceName, 
        modelName,
        signal,
    } = config;

    let timeoutId: NodeJS.Timeout | null = null;
    let isTimedOut = false;
    let lastActivityTime = Date.now();
    
    // Create an abort controller for internal cancellation
    const internalAbortController = new AbortController();
    
    // Link external abort signal if provided
    if (signal) {
        signal.addEventListener("abort", () => {
            internalAbortController.abort();
        });
    }

    // Reset timeout on activity
    function resetTimeout() {
        lastActivityTime = Date.now();
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
        
        timeoutId = setTimeout(function timeoutHandler() {
            isTimedOut = true;
            internalAbortController.abort();
            logger.error("[StreamTimeout] Stream timed out", {
                serviceName,
                modelName,
                timeoutMs,
                lastActivityMs: Date.now() - lastActivityTime,
            });
        }, timeoutMs);
    }

    // Start the timeout
    resetTimeout();

    try {
        // Iterate through the generator
        for await (const value of generator) {
            // Check if we've timed out
            if (isTimedOut || internalAbortController.signal.aborted) {
                throw new StreamTimeoutError(serviceName, timeoutMs, modelName);
            }
            
            // Reset timeout on each yielded value
            resetTimeout();
            
            // Yield the value
            yield value;
        }
    } catch (error) {
        // If it's already a timeout error, re-throw it
        if (error instanceof StreamTimeoutError) {
            throw error;
        }
        
        // If we timed out but got a different error, wrap it
        if (isTimedOut || internalAbortController.signal.aborted) {
            throw new StreamTimeoutError(serviceName, timeoutMs, modelName);
        }
        
        // Otherwise, re-throw the original error
        throw error;
    } finally {
        // Clean up the timeout
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
    }
}

/**
 * Creates a timeout wrapper with pre-configured settings
 * Useful for creating service-specific timeout wrappers
 */
export function createStreamTimeoutWrapper(
    defaultConfig: Partial<StreamTimeoutConfig>,
) {
    return function wrapWithTimeout<T>(
        generator: AsyncGenerator<T>,
        overrideConfig?: Partial<StreamTimeoutConfig>,
    ): AsyncGenerator<T> {
        const config = {
            ...defaultConfig,
            ...overrideConfig,
        } as StreamTimeoutConfig;
        
        return withStreamTimeout(generator, config);
    };
}
