import { useQuery } from "@tanstack/react-query";
import {
  fetchFallbackInsights,
  fetchHealthSummary,
  operationalQueryKeys,
} from "../api/operationalClient";
import type { FallbackInsights, HealthSummary } from "../api/operationalTypes";

const REFETCH_MS = 60_000;

export function useFallbackInsights(options: { enabled?: boolean } = {}) {
  return useQuery<FallbackInsights, Error>({
    queryKey: operationalQueryKeys.fallback(),
    queryFn: fetchFallbackInsights,
    enabled: options.enabled ?? true,
    refetchInterval: REFETCH_MS,
    staleTime: 30_000,
  });
}

export function useHealthSummary(options: { enabled?: boolean } = {}) {
  return useQuery<HealthSummary, Error>({
    queryKey: operationalQueryKeys.health(),
    queryFn: fetchHealthSummary,
    enabled: options.enabled ?? true,
    refetchInterval: REFETCH_MS,
    staleTime: 30_000,
  });
}
