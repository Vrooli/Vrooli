import { useState, useCallback, useRef, useEffect } from 'react';
import { apiFetch, toApiError } from '../api/apiFetch';
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
      const result = await apiFetch<T>(url, {
        ...options,
        signal: controller.signal,
      });
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

      const apiError = toApiError(err);
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
