import { HttpStatus } from "../consts/api.js";

export interface FetchWrapperOptions {
    /** Maximum number of retry attempts */
    maxRetries?: number;
    /** Delay between retries in milliseconds */
    retryDelay?: number;
    /** Function to determine if a retry should be attempted */
    shouldRetry?: (error: Error, attempt: number) => boolean;
    /** Request timeout in milliseconds */
    timeout?: number;
    /** Transform the response before returning */
    transformResponse?: (response: Response) => Promise<unknown>;
    /** Transform errors before throwing */
    transformError?: (error: Error) => Error;
    /** Headers to include with every request */
    defaultHeaders?: HeadersInit;
}

export interface LazyFetchOptions extends FetchWrapperOptions {
    /** Cache duration in milliseconds */
    cacheDuration?: number;
    /** Whether to cache failed requests */
    cacheErrors?: boolean;
}

interface CacheEntry {
    data: unknown;
    timestamp: number;
    error?: Error;
}

const DEFAULT_OPTIONS: Required<Omit<FetchWrapperOptions, "transformResponse" | "transformError" | "defaultHeaders">> = {
    maxRetries: 3,
    retryDelay: 1000,
    shouldRetry: undefined as any, // Will be set in fetchWrapper
    timeout: 30000,
};

export class FetchError extends Error {
    constructor(
        message: string,
        public status?: number,
        public statusText?: string,
        public response?: Response,
    ) {
        super(message);
        this.name = "FetchError";
    }
}

export class TimeoutError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "TimeoutError";
    }
}

export class NetworkError extends Error {
    constructor(message: string) {
        super(message);
        this.name = "NetworkError";
    }
}

/**
 * Enhanced fetch wrapper with retry logic, timeout, and error transformation
 */
export async function fetchWrapper(
    url: string,
    init?: RequestInit,
    options?: FetchWrapperOptions,
): Promise<Response> {
    const mergedOptions = { ...DEFAULT_OPTIONS, ...options };
    
    // Set up default shouldRetry if not provided
    if (!mergedOptions.shouldRetry) {
        mergedOptions.shouldRetry = (error: Error, attempt: number) => {
            // Retry on network errors and 5xx status codes
            if (error.name === "NetworkError" || error.name === "TimeoutError") {
                return true;
            }
            if (error instanceof FetchError && error.status && error.status >= HttpStatus.InternalServerError) {
                return true;
            }
            return false;
        };
    }
    const headers = new Headers(init?.headers);
    
    // Add default headers
    if (mergedOptions.defaultHeaders) {
        const defaultHeaders = new Headers(mergedOptions.defaultHeaders);
        defaultHeaders.forEach((value, key) => {
            if (!headers.has(key)) {
                headers.set(key, value);
            }
        });
    }

    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= mergedOptions.maxRetries; attempt++) {
        try {
            // Create abort controller for timeout
            const controller = new AbortController();
            const timeoutId = setTimeout(() => {
                controller.abort();
            }, mergedOptions.timeout);

            try {
                const response = await fetch(url, {
                    ...init,
                    headers,
                    signal: controller.signal,
                });

                clearTimeout(timeoutId);

                // Check if response is ok
                if (!response.ok) {
                    throw new FetchError(
                        `HTTP ${response.status}: ${response.statusText}`,
                        response.status,
                        response.statusText,
                        response,
                    );
                }

                // Apply response transformation if provided
                if (mergedOptions.transformResponse) {
                    const transformed = await mergedOptions.transformResponse(response);
                    // Create a new response with the transformed data
                    return new Response(JSON.stringify(transformed), {
                        status: response.status,
                        statusText: response.statusText,
                        headers: response.headers,
                    });
                }

                return response;
            } catch (error) {
                clearTimeout(timeoutId);

                if (error instanceof Error) {
                    if (error.name === "AbortError") {
                        throw new TimeoutError(`Request timeout after ${mergedOptions.timeout}ms`);
                    }
                    if (error instanceof FetchError) {
                        throw error;
                    }
                    // Wrap other errors as NetworkError
                    throw new NetworkError("Unknown network error");
                }
                throw new NetworkError("Unknown network error");
            }
        } catch (error) {
            lastError = error as Error;

            // Apply error transformation if provided
            if (mergedOptions.transformError) {
                lastError = mergedOptions.transformError(lastError);
            }

            // Check if we should retry
            if (attempt < mergedOptions.maxRetries && mergedOptions.shouldRetry(lastError, attempt + 1)) {
                // Wait before retrying
                await new Promise(resolve => setTimeout(resolve, mergedOptions.retryDelay * (attempt + 1)));
                continue;
            }

            throw lastError;
        }
    }

    throw lastError || new Error("Unexpected error in fetchWrapper");
}

/**
 * Lazy loading fetch wrapper that caches responses
 */
export function createLazyFetch(defaultOptions?: LazyFetchOptions) {
    const cache = new Map<string, CacheEntry>();
    const DEFAULT_CACHE_DURATION = 5 * 60 * 1000; // 5 minutes

    return async function lazyFetch(
        url: string,
        init?: RequestInit,
        options?: LazyFetchOptions,
    ): Promise<Response> {
        const mergedOptions = { 
            ...defaultOptions, 
            ...options,
            cacheDuration: options?.cacheDuration ?? defaultOptions?.cacheDuration ?? DEFAULT_CACHE_DURATION,
        };
        
        // Create cache key from URL and relevant request options
        const cacheKey = `${url}:${init?.method || "GET"}:${JSON.stringify(init?.body || "")}`;
        
        // Check cache
        const cached = cache.get(cacheKey);
        if (cached) {
            const isExpired = Date.now() - cached.timestamp > mergedOptions.cacheDuration;
            
            if (!isExpired) {
                if (cached.error) {
                    if (mergedOptions.cacheErrors) {
                        throw cached.error;
                    }
                } else {
                    // Return cached response
                    return new Response(JSON.stringify(cached.data), {
                        status: HttpStatus.Ok,
                        statusText: "OK",
                        headers: { "X-From-Cache": "true" },
                    });
                }
            } else {
                // Remove expired entry
                cache.delete(cacheKey);
            }
        }

        try {
            const response = await fetchWrapper(url, init, mergedOptions);
            
            // Cache successful responses
            const data = await response.clone().json();
            cache.set(cacheKey, {
                data,
                timestamp: Date.now(),
            });
            
            return response;
        } catch (error) {
            // Cache errors if configured
            if (mergedOptions.cacheErrors && error instanceof Error) {
                cache.set(cacheKey, {
                    data: null,
                    timestamp: Date.now(),
                    error,
                });
            }
            throw error;
        }
    };
}

/**
 * Create a fetch wrapper with preset configuration
 */
export function createFetchClient(baseURL: string, defaultOptions?: FetchWrapperOptions) {
    return {
        async get(path: string, options?: FetchWrapperOptions): Promise<Response> {
            return fetchWrapper(`${baseURL}${path}`, { method: "GET" }, { ...defaultOptions, ...options });
        },
        
        async post(path: string, body?: unknown, options?: FetchWrapperOptions): Promise<Response> {
            return fetchWrapper(
                `${baseURL}${path}`,
                {
                    method: "POST",
                    body: body ? JSON.stringify(body) : undefined,
                    headers: { "Content-Type": "application/json" },
                },
                { ...defaultOptions, ...options },
            );
        },
        
        async put(path: string, body?: unknown, options?: FetchWrapperOptions): Promise<Response> {
            return fetchWrapper(
                `${baseURL}${path}`,
                {
                    method: "PUT",
                    body: body ? JSON.stringify(body) : undefined,
                    headers: { "Content-Type": "application/json" },
                },
                { ...defaultOptions, ...options },
            );
        },
        
        async delete(path: string, options?: FetchWrapperOptions): Promise<Response> {
            return fetchWrapper(`${baseURL}${path}`, { method: "DELETE" }, { ...defaultOptions, ...options });
        },
        
        async patch(path: string, body?: unknown, options?: FetchWrapperOptions): Promise<Response> {
            return fetchWrapper(
                `${baseURL}${path}`,
                {
                    method: "PATCH",
                    body: body ? JSON.stringify(body) : undefined,
                    headers: { "Content-Type": "application/json" },
                },
                { ...defaultOptions, ...options },
            );
        },
    };
}