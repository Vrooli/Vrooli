/**
 * API Client - HTTP infrastructure for Swarm Manager
 *
 * This module provides the core HTTP client implementation with structured
 * error handling and graceful degradation support.
 *
 * It is designed as a seam - the client can be substituted for testing
 * or alternative implementations without changing consuming code.
 *
 * Responsibilities:
 * - URL resolution and construction
 * - HTTP request/response handling
 * - Structured error handling with error type differentiation
 * - Timeout enforcement
 *
 * NOT responsible for:
 * - Domain type definitions (see types/domain.ts)
 * - Business logic or validation
 * - UI state management
 *
 * DOC: docs/internal/SEAMS.md#ui-to-api-seam-improved-in-phase-3
 * DOC: docs/internal/ERROR-SEMANTICS.md#api-error-type-system
 */

import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import { apiConfig } from "../config";

// ============================================================================
// Error Types
// ============================================================================

/**
 * Error types for API failures.
 * These allow consumers to handle different failure modes appropriately.
 */
export type ApiErrorType = "network" | "timeout" | "http" | "parse";

/**
 * Structured API error with type differentiation.
 *
 * This enables graceful degradation by allowing UI to show appropriate
 * messages based on failure type:
 * - network: "Unable to connect - check your internet connection"
 * - timeout: "Request timed out - the server may be busy"
 * - http: Status-specific messages (404, 500, etc.)
 * - parse: "Invalid server response"
 */
export class ApiError extends Error {
  readonly type: ApiErrorType;
  readonly status?: number;
  readonly cause?: unknown;
  readonly isClientError: boolean;
  readonly isServerError: boolean;
  readonly isRetryable: boolean;

  constructor(
    type: ApiErrorType,
    message: string,
    options?: { status?: number; cause?: unknown }
  ) {
    super(message);
    this.name = "ApiError";
    this.type = type;
    this.status = options?.status;
    this.cause = options?.cause;
    this.isClientError = type === "http" && !!options?.status && options.status >= 400 && options.status < 500;
    this.isServerError = type === "http" && !!options?.status && options.status >= 500;
    // Network errors and 5xx are typically retryable; 4xx are not
    this.isRetryable = type === "network" || type === "timeout" || this.isServerError;
  }

  /**
   * Returns a user-friendly message based on error type.
   * Does not expose technical details like URLs or stack traces.
   */
  get userMessage(): string {
    switch (this.type) {
      case "network":
        return "Unable to connect to the server. Please check your internet connection.";
      case "timeout":
        return "The request timed out. The server may be busy - please try again.";
      case "http":
        if (this.status === 401) return "Your session has expired. Please refresh the page.";
        if (this.status === 403) return "You don't have permission to access this resource.";
        if (this.status === 404) return "The requested resource was not found.";
        if (this.isServerError) return "The server encountered an error. Please try again later.";
        return "The request failed. Please try again.";
      case "parse":
        return "Received an invalid response from the server.";
      default:
        return "An unexpected error occurred. Please try again.";
    }
  }
}

/**
 * Type guard to check if an error is an ApiError.
 */
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}

// ============================================================================
// Types
// ============================================================================

/**
 * Options for API requests.
 */
export interface RequestOptions {
  /** Response type override (default: "json") */
  responseType?: "json" | "text" | "blob";
  /** Custom headers to include in the request */
  headers?: Record<string, string>;
  /** Optional external abort signal for caller-controlled cancellation */
  signal?: AbortSignal;
}

/**
 * Interface for the API client.
 * This is the seam - any implementation matching this interface can be used.
 * [REQ:REQ-P0-007] Added patch method for partial updates
 */
export interface IApiClient {
  get<T>(path: string, options?: RequestOptions): Promise<T>;
  post<T>(path: string, body: unknown, options?: RequestOptions): Promise<T>;
  put<T>(path: string, body: unknown, options?: RequestOptions): Promise<T>;
  patch<T>(path: string, body: unknown, options?: RequestOptions): Promise<T>;
  delete<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T>;
}

// ============================================================================
// Implementation
// ============================================================================

/**
 * HTTP client implementation using fetch with structured error handling.
 *
 * Provides typed methods for standard HTTP verbs. All requests include:
 * - Content-Type: application/json header
 * - cache: no-store to prevent stale data
 * - Configurable timeout via AbortController
 */
export class ApiClient implements IApiClient {
  private baseUrl: string;
  private timeoutMs: number;

  constructor(baseUrl: string, timeoutMs: number = apiConfig.requestTimeoutMs) {
    this.baseUrl = baseUrl;
    this.timeoutMs = timeoutMs;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: RequestOptions
  ): Promise<T> {
    const url = buildApiUrl(path, { baseUrl: this.baseUrl });

    // Set up timeout using AbortController
    const controller = new AbortController();
    let timedOut = false;
    let abortedByCaller = false;
    const timeoutId = setTimeout(() => {
      timedOut = true;
      controller.abort();
    }, this.timeoutMs);
    const handleAbort = () => {
      abortedByCaller = true;
      controller.abort();
    };
    options?.signal?.addEventListener("abort", handleAbort);

    try {
      // Determine headers and body based on content type
      const isFormData = body instanceof FormData;
      const headers: Record<string, string> = isFormData
        ? {} // Let browser set Content-Type with boundary for FormData
        : { "Content-Type": "application/json" };

      // Apply custom headers if provided
      if (options?.headers) {
        Object.assign(headers, options.headers);
      }

      const res = await fetch(url, {
        method,
        headers,
        body: isFormData ? body : body ? JSON.stringify(body) : undefined,
        cache: "no-store",
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!res.ok) {
        let detail = "";
        try {
          detail = (await res.text()).trim();
        } catch { /* ignore read failures */ }
        const message = detail || `Request failed with status ${res.status}`;
        throw new ApiError("http", message, { status: res.status });
      }

      // Handle response based on requested type or content-type
      const responseType = options?.responseType || "json";

      if (responseType === "text") {
        return (await res.text()) as T;
      }

      if (responseType === "blob") {
        return (await res.blob()) as T;
      }

      // Handle empty responses (e.g., 204 No Content)
      const contentType = res.headers.get("content-type");
      if (!contentType || !contentType.includes("application/json")) {
        // Return undefined for non-JSON responses (cast needed for generic)
        return undefined as T;
      }

      try {
        return (await res.json()) as T;
      } catch (parseError) {
        throw new ApiError("parse", "Failed to parse server response", {
          cause: parseError,
        });
      }
    } catch (error) {
      clearTimeout(timeoutId);

      // Re-throw ApiErrors as-is
      if (isApiError(error)) {
        throw error;
      }

      // Handle abort (timeout)
      if (error instanceof DOMException && error.name === "AbortError") {
        if (abortedByCaller && !timedOut) {
          throw error;
        }
        throw new ApiError("timeout", "Request timed out", { cause: error });
      }

      // Handle network errors (fetch failed entirely)
      if (error instanceof TypeError) {
        throw new ApiError("network", "Network request failed", { cause: error });
      }

      // Unknown error - wrap as network error
      throw new ApiError("network", "An unexpected error occurred", { cause: error });
    } finally {
      options?.signal?.removeEventListener("abort", handleAbort);
    }
  }

  async get<T>(path: string, options?: RequestOptions): Promise<T> {
    return this.request<T>("GET", path, undefined, options);
  }

  async post<T>(path: string, body: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("POST", path, body, options);
  }

  async put<T>(path: string, body: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("PUT", path, body, options);
  }

  async patch<T>(path: string, body: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("PATCH", path, body, options);
  }

  async delete<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    return this.request<T>("DELETE", path, body, options);
  }
}

// ============================================================================
// Factory & Default Instance
// ============================================================================

/**
 * Creates an API client with the given base URL.
 * Use this factory when you need a client with a custom base URL.
 */
export function createApiClient(baseUrl: string, timeoutMs?: number): IApiClient {
  return new ApiClient(baseUrl, timeoutMs);
}

/**
 * Default API base URL - resolved from VITE_API_BASE_URL or window.location.
 * This is computed once at module load.
 */
export const DEFAULT_API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * Default API client instance.
 * Most application code should use this unless testing or using a custom base URL.
 */
export const defaultApiClient: IApiClient = createApiClient(DEFAULT_API_BASE);
