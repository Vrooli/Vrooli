// React Query hooks for the health audit endpoints.

import { useQuery } from "@tanstack/react-query";
import {
  fetchHealthAudit,
  fetchModelHealth,
  fetchRunnerHealth,
  healthQueryKeys,
} from "../api/healthClient";
import type {
  HealthAuditFilters,
  HealthAuditResponse,
  ModelHealthListResponse,
  RunnerHealthListResponse,
} from "../api/types";

const SNAPSHOT_REFETCH_MS = 30_000;

export function useModelHealth(options: { enabled?: boolean } = {}) {
  return useQuery<ModelHealthListResponse, Error>({
    queryKey: healthQueryKeys.models(),
    queryFn: fetchModelHealth,
    enabled: options.enabled ?? true,
    refetchInterval: SNAPSHOT_REFETCH_MS,
    staleTime: 10_000,
  });
}

export function useRunnerHealth(options: { enabled?: boolean } = {}) {
  return useQuery<RunnerHealthListResponse, Error>({
    queryKey: healthQueryKeys.runners(),
    queryFn: fetchRunnerHealth,
    enabled: options.enabled ?? true,
    refetchInterval: SNAPSHOT_REFETCH_MS,
    staleTime: 10_000,
  });
}

export function useHealthAudit(filters: HealthAuditFilters, options: { enabled?: boolean } = {}) {
  return useQuery<HealthAuditResponse, Error>({
    queryKey: healthQueryKeys.audit(filters),
    queryFn: () => fetchHealthAudit(filters),
    enabled: options.enabled ?? true,
    staleTime: 10_000,
  });
}
