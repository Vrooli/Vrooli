import { useState, useCallback, useRef, useEffect } from 'react';
import { buildApiUrl } from '../api/apiBase';
import type { APIError } from '../../types';

interface UseApiCallReturn<T> {
  data: T | null;
  error: APIError | null;
  loading: boolean;
  execute: (url: string, options?: RequestInit) => Promise<T | null>;
}

/**
 * Generic fetch wrapper that handles:
 * - JSON response parsing
 * - Error normalisation into APIError
 * - Loading state
 * - AbortController cleanup on unmount
 *
 * Usage:
 *   const { execute, data, error, loading } = useApiCall<MetricsResponse>();
 *   // later:  const result = await execute('/metrics/current');
 */
export function useApiCall<T = unknown>(): UseApiCallReturn<T> {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<APIError | null>(null);
  const [loading, setLoading] = useState(false);
  const controllerRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    return () => {
      mountedRef.current = false;
      controllerRef.current?.abort();
    };
  }, []);

  const execute = useCallback(async (url: string, options?: RequestInit): Promise<T | null> => {
    // Abort any in-flight request
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(buildApiUrl(url), {
        ...options,
        signal: controller.signal,
      });

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

      const result = (await response.json()) as T;
      if (mountedRef.current) {
        setData(result);
        setError(null);
      }
      return result;
    } catch (err) {
      // Ignore abort errors
      if (err instanceof DOMException && err.name === 'AbortError') {
        return null;
      }

      if (!mountedRef.current) return null;

      const apiError: APIError =
        err && typeof err === 'object' && 'error' in err
          ? (err as APIError)
          : {
              error: 'Network or unknown error',
              details: err instanceof Error ? err.message : String(err),
              timestamp: new Date().toISOString(),
            };
      setError(apiError);
      return null;
    } finally {
      if (mountedRef.current) {
        setLoading(false);
      }
    }
  }, []);

  return { data, error, loading, execute };
}
