/**
 * Server Readiness Checker
 *
 * DOC: docs/internal/SEAMS.md#server-readiness
 *
 * Validates that the server is truly ready to serve requests.
 * Unlike simple connectivity checks, this module validates:
 * - Response status codes (must be 2xx or configurable)
 * - Optional content validation
 * - Proper timeout handling
 *
 * Testing Seams:
 * - IHttpClient: Mock HTTP requests without network access
 * - ITimer: Mock timing for deterministic tests
 */

/**
 * Result of a server readiness check.
 */
export interface ReadinessResult {
    /** Whether the server is ready */
    ready: boolean;
    /** HTTP status code received, if any */
    statusCode?: number;
    /** Error message if not ready */
    error?: string;
    /** Time taken for the check in milliseconds */
    durationMs: number;
}

/**
 * Configuration for the readiness checker.
 */
export interface ReadinessConfig {
    /** URL to check for readiness */
    url: string;
    /** Maximum time to wait for server to be ready (ms) */
    timeoutMs: number;
    /** Interval between check attempts (ms) */
    pollIntervalMs: number;
    /**
     * HTTP status codes considered "ready".
     * @default [200, 201, 202, 204]
     */
    acceptableStatusCodes: number[];
    /**
     * Whether to accept any 2xx status code.
     * If true, overrides acceptableStatusCodes.
     * @default true
     */
    acceptAny2xx: boolean;
    /**
     * Optional content validator.
     * If provided, response body must pass this check.
     */
    contentValidator?: (body: string) => boolean;
}

/**
 * Default configuration for readiness checks.
 */
export const DEFAULT_READINESS_CONFIG: Omit<ReadinessConfig, "url"> = {
    timeoutMs: 30000,
    pollIntervalMs: 500,
    acceptableStatusCodes: [200, 201, 202, 204],
    acceptAny2xx: true,
};

/**
 * Interface for HTTP client operations.
 * Seam for testing without actual network requests.
 */
export interface IHttpClient {
    /**
     * Make a GET request to the specified URL.
     * @param url - URL to request
     * @param timeoutMs - Request timeout in milliseconds
     * @returns Response with status and optional body
     */
    get(url: string, timeoutMs: number): Promise<{
        statusCode: number;
        body?: string;
    }>;
}

/**
 * Interface for timing operations.
 * Seam for testing time-dependent behavior.
 */
export interface ITimer {
    /** Get current timestamp in milliseconds */
    now(): number;
    /** Sleep for specified milliseconds */
    sleep(ms: number): Promise<void>;
}

/**
 * Default timer implementation using real time.
 */
export const realTimer: ITimer = {
    now: () => Date.now(),
    sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)),
};

/**
 * Progress callback for long-running readiness checks.
 */
export type ReadinessProgressCallback = (attempt: number, elapsed: number) => void;

/**
 * Check if a status code is acceptable based on configuration.
 */
export function isAcceptableStatus(
    statusCode: number,
    config: Pick<ReadinessConfig, "acceptAny2xx" | "acceptableStatusCodes">
): boolean {
    if (config.acceptAny2xx) {
        return statusCode >= 200 && statusCode < 300;
    }
    return config.acceptableStatusCodes.includes(statusCode);
}

/**
 * Check server readiness with polling and timeout.
 *
 * Unlike simple connectivity checks, this function:
 * 1. Validates HTTP status codes (not just "any response")
 * 2. Supports configurable acceptable statuses
 * 3. Optionally validates response content
 * 4. Reports progress for UI updates
 *
 * @param httpClient - HTTP client for making requests
 * @param config - Readiness configuration
 * @param timer - Timer for timing operations
 * @param onProgress - Optional progress callback
 * @returns ReadinessResult indicating success or failure
 */
export async function checkServerReadiness(
    httpClient: IHttpClient,
    config: ReadinessConfig,
    timer: ITimer = realTimer,
    onProgress?: ReadinessProgressCallback
): Promise<ReadinessResult> {
    const startTime = timer.now();
    const deadline = startTime + config.timeoutMs;
    let attempt = 0;
    let lastError: string | undefined;
    let lastStatusCode: number | undefined;

    while (timer.now() < deadline) {
        attempt++;
        const attemptStart = timer.now();

        try {
            // Calculate remaining time for this attempt
            const remaining = deadline - timer.now();
            const requestTimeout = Math.min(config.pollIntervalMs * 2, remaining);

            const response = await httpClient.get(config.url, requestTimeout);
            lastStatusCode = response.statusCode;

            // Check if status code is acceptable
            if (!isAcceptableStatus(response.statusCode, config)) {
                lastError = `Server returned status ${response.statusCode}, expected 2xx`;

                // Log specific guidance for common non-ready statuses
                if (response.statusCode === 404) {
                    lastError += " (404 indicates the URL path may be incorrect or server not fully initialized)";
                } else if (response.statusCode === 503) {
                    lastError += " (503 indicates server is starting up)";
                }

                // Log immediately so the error is visible even if we timeout
                // Only log once per status code to avoid spam
                if (attempt === 1 || (attempt % 10 === 0)) {
                    console.warn(`[ServerReadiness] ${lastError} at ${config.url}`);
                }
            } else if (config.contentValidator && response.body !== undefined) {
                // Validate content if validator is provided
                if (!config.contentValidator(response.body)) {
                    lastError = "Server response failed content validation";
                } else {
                    // Success!
                    return {
                        ready: true,
                        statusCode: response.statusCode,
                        durationMs: timer.now() - startTime,
                    };
                }
            } else {
                // Success without content validation
                return {
                    ready: true,
                    statusCode: response.statusCode,
                    durationMs: timer.now() - startTime,
                };
            }
        } catch (error) {
            // Connection error - server not yet available
            lastError = error instanceof Error ? error.message : String(error);

            // Don't log connection refused errors as they're expected during startup
            if (!lastError.includes("ECONNREFUSED")) {
                console.warn(`[ServerReadiness] Check attempt ${attempt} failed:`, lastError);
            }
        }

        // Report progress
        if (onProgress) {
            const elapsed = timer.now() - startTime;
            onProgress(attempt, elapsed);
        }

        // Wait before next attempt
        const attemptDuration = timer.now() - attemptStart;
        const waitTime = Math.max(0, config.pollIntervalMs - attemptDuration);

        if (timer.now() + waitTime < deadline) {
            await timer.sleep(waitTime);
        } else {
            break; // No time for another attempt
        }
    }

    // Timeout reached
    // Build result object, only including optional properties if defined
    // (required for exactOptionalPropertyTypes compatibility)
    return {
        ready: false,
        ...(lastStatusCode !== undefined && { statusCode: lastStatusCode }),
        error: lastError ?? `Server did not become ready within ${config.timeoutMs}ms`,
        durationMs: timer.now() - startTime,
    };
}

/**
 * Create a readiness checker bound to Electron's net module.
 *
 * This is the production factory - tests should call checkServerReadiness
 * directly with mock dependencies.
 */
export function createElectronReadinessChecker(
    electronNet: typeof Electron.net
): IHttpClient {
    return {
        get: (url, timeoutMs) => {
            return new Promise((resolve, reject) => {
                const request = electronNet.request(url);
                let completed = false;

                // Set up timeout
                const timeout = setTimeout(() => {
                    if (!completed) {
                        completed = true;
                        request.abort();
                        reject(new Error(`Request timed out after ${timeoutMs}ms`));
                    }
                }, timeoutMs);

                // Handle response
                request.on("response", (response) => {
                    if (completed) return;

                    let body = "";
                    response.on("data", (chunk) => {
                        body += chunk.toString();
                    });

                    response.on("end", () => {
                        if (completed) return;
                        completed = true;
                        clearTimeout(timeout);
                        // Build response object, only including body if non-empty
                        // (required for exactOptionalPropertyTypes compatibility)
                        resolve({
                            statusCode: response.statusCode,
                            ...(body && { body }),
                        });
                    });

                    response.on("error", (error) => {
                        if (completed) return;
                        completed = true;
                        clearTimeout(timeout);
                        reject(error);
                    });
                });

                // Handle request errors
                request.on("error", (error) => {
                    if (completed) return;
                    completed = true;
                    clearTimeout(timeout);
                    reject(error);
                });

                // Send the request
                request.end();
            });
        },
    };
}
