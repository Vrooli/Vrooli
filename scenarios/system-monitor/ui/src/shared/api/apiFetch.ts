// DOC: docs/internal/INTEROP_AUDIT.md
import { buildUrl as buildApiUrl } from '../../lib/api-client';
import type { APIError, ErrorDetail } from '../../types';

/** Parse an error response into a normalized APIError. */
async function parseErrorResponse(response: Response): Promise<APIError> {
  const errorText = await response.text();
  try {
    const parsed = JSON.parse(errorText) as Record<string, unknown>;
    // Unified format: { error: { code, message, retryable, ... } }
    if (parsed.error && typeof parsed.error === 'object' && 'code' in (parsed.error as Record<string, unknown>)) {
      const detail = parsed.error as ErrorDetail;
      return {
        error: detail.message,
        detail,
        timestamp: new Date().toISOString(),
      };
    }
    // Legacy or unexpected JSON — use what we can.
    return {
      error: typeof parsed.error === 'string' ? parsed.error : response.statusText,
      timestamp: new Date().toISOString(),
    };
  } catch {
    return {
      error: `HTTP ${response.status}: ${response.statusText}`,
      timestamp: new Date().toISOString(),
    };
  }
}

/** Build a network-error APIError (fetch itself threw). */
function networkError(): APIError {
  return {
    error: 'Unable to reach the server. Check your connection.',
    detail: { code: 'network', message: 'Unable to reach the server. Check your connection.', retryable: true, recovery: 'wait' },
    timestamp: new Date().toISOString(),
  };
}

/**
 * Shared fetch utility that handles:
 * - Prepending the API base URL
 * - JSON response parsing
 * - Error normalization into APIError
 * - Optional AbortSignal forwarding
 *
 * Use this instead of raw `fetch()` + `buildApiUrl()` in hooks and callbacks.
 * For components that need loading/error state, use `useApiCall` which wraps this.
 */
export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(buildApiUrl(path), options);
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw networkError();
  }

  if (!response.ok) {
    throw await parseErrorResponse(response);
  }

  try {
    return (await response.json()) as T;
  } catch {
    throw {
      error: 'Invalid response from server',
      detail: { code: 'internal', message: 'Failed to parse server response', retryable: false },
      timestamp: new Date().toISOString(),
    } satisfies APIError;
  }
}

/**
 * Fetch + proto-parse in one step.
 *
 * Works like `apiFetch` but passes the raw JSON through a proto `parser`
 * function (from proto-contracts.ts) before returning, giving callers a
 * fully-typed protobuf message shape.
 */
export async function protoFetch<T>(
  path: string,
  parser: (data: unknown) => T,
  options?: RequestInit,
): Promise<T> {
  const url = buildApiUrl(path);
  let response: Response;
  try {
    response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') throw err;
    throw networkError();
  }
  if (!response.ok) {
    throw await parseErrorResponse(response);
  }
  let json: unknown;
  try {
    json = await response.json();
  } catch {
    throw {
      error: 'Invalid response from server',
      detail: { code: 'internal', message: 'Failed to parse server response', retryable: false },
      timestamp: new Date().toISOString(),
    } satisfies APIError;
  }
  try {
    return parser(json);
  } catch {
    throw {
      error: 'Invalid response format',
      detail: { code: 'internal', message: 'Failed to decode server response', retryable: false },
      timestamp: new Date().toISOString(),
    } satisfies APIError;
  }
}

/** Type guard: returns true when `err` is shaped like an APIError. */
export function isApiError(err: unknown): err is APIError {
  return err != null && typeof err === 'object' && 'error' in err;
}

/** Extract a human-readable message from any thrown value. */
export function extractErrorMessage(err: unknown, fallback = 'An unknown error occurred'): string {
  if (isApiError(err)) return err.error;
  if (err instanceof Error) return err.message;
  return fallback;
}

/** Normalize any thrown value into an APIError. */
export function toApiError(err: unknown): APIError {
  if (isApiError(err)) return err;
  if (err instanceof DOMException && err.name === 'AbortError') throw err;
  // Standard JS errors are NOT network errors — preserve their message
  if (err instanceof Error) {
    return {
      error: err.message,
      detail: { code: 'internal', message: err.message, retryable: false },
      timestamp: new Date().toISOString(),
    };
  }
  return networkError();
}
