import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { fetchHealth, type HealthResponse } from "../lib/api";
import type { HealthStatus } from "../components/ui/HealthIndicator";

// ─────────────────────────────────────────────────────────────────────────────
// Health Status Derivation Hook
// [REQ:P0-001] Reference Scenario Registry - Centralized health state management
// ─────────────────────────────────────────────────────────────────────────────
//
// This hook centralizes the health check polling and status derivation pattern
// that was previously duplicated across Dashboard and ReferenceDetail pages.
//
// Decision boundary: "What is the health status?"
// - loading: Health check is in progress
// - connected: Health check succeeded
// - disconnected: Health check failed
// ─────────────────────────────────────────────────────────────────────────────

export interface UseHealthStatusOptions {
  /** Polling interval in milliseconds (default: 30000) */
  refetchInterval?: number;
}

export interface UseHealthStatusResult {
  /** The query result for advanced use cases */
  query: UseQueryResult<HealthResponse, Error>;
  /** Whether the API is healthy/connected */
  isHealthy: boolean;
  /** Whether the health check is loading */
  isLoading: boolean;
  /** Whether there was an error */
  hasError: boolean;
  /** Derived health status for HealthIndicator component */
  healthStatus: HealthStatus;
  /** Refresh the health check */
  refetch: () => void;
}

/**
 * Hook for centralized health status management.
 *
 * Consolidates the health check polling pattern and status derivation
 * that is shared across pages, enforcing a single responsibility for
 * health state management.
 *
 * @example
 * ```tsx
 * const { isHealthy, healthStatus, refetch } = useHealthStatus();
 *
 * // Use isHealthy to conditionally enable other queries
 * const dataQuery = useQuery({
 *   queryKey: ["data"],
 *   queryFn: fetchData,
 *   enabled: isHealthy
 * });
 *
 * // Use healthStatus for the indicator
 * <HealthIndicator status={healthStatus} />
 * ```
 */
export function useHealthStatus(
  options: UseHealthStatusOptions = {}
): UseHealthStatusResult {
  const { refetchInterval = 30000 } = options;

  const query = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval
  });

  const isHealthy = query.isSuccess;
  const isLoading = query.isLoading;
  const hasError = query.isError;

  // Decision: Derive health status for display
  // - loading takes precedence (show loading state while checking)
  // - connected if health check succeeded
  // - disconnected otherwise (error or never checked)
  const healthStatus: HealthStatus = isLoading
    ? "loading"
    : isHealthy
      ? "connected"
      : "disconnected";

  return {
    query,
    isHealthy,
    isLoading,
    hasError,
    healthStatus,
    refetch: query.refetch
  };
}
