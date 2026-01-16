import { resolveApiBase, buildApiUrl } from '@vrooli/api-base';

export const API_BASE = resolveApiBase({ appendSuffix: true });

/**
 * API error types for graceful degradation and user-friendly messages.
 * Components can check error.type to decide how to handle specific failures.
 */
export type ApiErrorType =
  | 'network' // Network unreachable, offline, DNS failure
  | 'timeout' // Request timed out
  | 'unauthorized' // 401 - session expired or invalid credentials
  | 'forbidden' // 403 - lacks permission
  | 'not_found' // 404 - resource doesn't exist
  | 'validation' // 400 - bad request / validation error
  | 'rate_limited' // 429 - too many requests
  | 'server_error' // 500+ - server-side failure
  | 'unknown'; // Unclassified error

export class ApiError extends Error {
  readonly type: ApiErrorType;
  readonly status?: number;
  readonly retryable: boolean;
  readonly userMessage: string;

  constructor(
    message: string,
    type: ApiErrorType,
    status?: number,
    userMessage?: string
  ) {
    super(message);
    this.name = 'ApiError';
    this.type = type;
    this.status = status;
    // Network, timeout, and server errors are typically retryable
    this.retryable = ['network', 'timeout', 'server_error', 'rate_limited'].includes(type);
    this.userMessage = userMessage ?? getDefaultUserMessage(type);
  }
}

function getDefaultUserMessage(type: ApiErrorType): string {
  switch (type) {
    case 'network':
      return 'Unable to reach the server. Please check your connection and try again.';
    case 'timeout':
      return 'The request took too long. Please try again.';
    case 'unauthorized':
      return 'Your session has expired. Please log in again.';
    case 'forbidden':
      return 'You don\'t have permission to perform this action.';
    case 'not_found':
      return 'The requested resource was not found.';
    case 'validation':
      return 'The request was invalid. Please check your input and try again.';
    case 'rate_limited':
      return 'Too many requests. Please wait a moment and try again.';
    case 'server_error':
      return 'Something went wrong on our end. Please try again later.';
    default:
      return 'An unexpected error occurred. Please try again.';
  }
}

function classifyError(status: number): ApiErrorType {
  if (status === 401) return 'unauthorized';
  if (status === 403) return 'forbidden';
  if (status === 404) return 'not_found';
  if (status === 400 || status === 422) return 'validation';
  if (status === 429) return 'rate_limited';
  if (status >= 500) return 'server_error';
  return 'unknown';
}

interface ApiCallOptions extends RequestInit {
  /** Timeout in milliseconds. Defaults to 30000 (30 seconds). */
  timeout?: number;
}

export async function apiCall<T>(endpoint: string, options: ApiCallOptions = {}): Promise<T> {
  const { timeout = 30000, ...fetchOptions } = options;
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });

  // Create abort controller for timeout handling
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeout);

  try {
    const res = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...fetchOptions.headers,
      },
      credentials: 'include',
      signal: controller.signal,
      ...fetchOptions,
    });

    clearTimeout(timeoutId);

    if (!res.ok) {
      const errorType = classifyError(res.status);
      let errorText = '';
      try {
        errorText =
          typeof (res as { text?: () => Promise<string | undefined> }).text === 'function'
            ? (await (res as { text: () => Promise<string | undefined> }).text()) ?? ''
            : res.statusText || 'Unknown error';
      } catch {
        errorText = res.statusText || 'Unknown error';
      }

      // Try to extract a more specific message from JSON response
      let userMessage: string | undefined;
      try {
        const parsed = JSON.parse(errorText);
        userMessage = parsed.error || parsed.message || undefined;
      } catch {
        // Not JSON, use default message
      }

      throw new ApiError(
        `API call failed (${res.status}): ${errorText}`,
        errorType,
        res.status,
        userMessage
      );
    }

    if (typeof (res as { json?: () => Promise<unknown> }).json === 'function') {
      return res.json() as Promise<T>;
    }

    return Promise.resolve(undefined as unknown as T);
  } catch (err) {
    clearTimeout(timeoutId);

    // Handle abort (timeout)
    if (err instanceof Error && err.name === 'AbortError') {
      throw new ApiError(
        `Request to ${endpoint} timed out after ${timeout}ms`,
        'timeout',
        undefined,
        'The request took too long. Please try again.'
      );
    }

    // Handle network errors (TypeError for failed fetch)
    if (err instanceof TypeError) {
      throw new ApiError(
        `Network error calling ${endpoint}: ${err.message}`,
        'network',
        undefined,
        'Unable to reach the server. Please check your connection and try again.'
      );
    }

    // Re-throw ApiError as-is
    if (err instanceof ApiError) {
      throw err;
    }

    // Wrap unknown errors
    throw new ApiError(
      err instanceof Error ? err.message : 'Unknown error',
      'unknown'
    );
  }
}

/**
 * Helper to check if an error is an ApiError of a specific type.
 * Useful for conditional handling in catch blocks.
 */
export function isApiError(error: unknown, type?: ApiErrorType): error is ApiError {
  if (!(error instanceof ApiError)) return false;
  if (type === undefined) return true;
  return error.type === type;
}
