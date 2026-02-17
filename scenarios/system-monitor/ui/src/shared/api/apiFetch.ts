import { buildApiUrl } from './apiBase';
import type { APIError } from '../../types';

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
  const response = await fetch(buildApiUrl(path), options);

  if (!response.ok) {
    const errorText = await response.text();
    let errorData: APIError;
    try {
      errorData = JSON.parse(errorText) as APIError;
    } catch {
      errorData = {
        error: `HTTP ${response.status}: ${response.statusText}`,
        details: errorText,
        timestamp: new Date().toISOString(),
      };
    }
    throw errorData;
  }

  return (await response.json()) as T;
}

/** Type guard: returns true when `err` is shaped like an APIError. */
export function isApiError(err: unknown): err is APIError {
  return err != null && typeof err === 'object' && 'error' in err;
}

/** Normalize any thrown value into an APIError. */
export function toApiError(err: unknown): APIError {
  if (isApiError(err)) return err;
  return {
    error: 'Network or unknown error',
    details: err instanceof Error ? err.message : String(err),
    timestamp: new Date().toISOString(),
  };
}
