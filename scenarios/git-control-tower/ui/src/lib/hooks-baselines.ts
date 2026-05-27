// ============================================================================
// Baseline hooks (Plan B §4.2) — React Query over BaselinesService
// ============================================================================

import { useCallback, useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";
import {
  listBaselines,
  getBaseline,
  diffBaseline,
  snapshotForBaseline,
  deleteBaseline,
  editBaseline,
  type SnapshotBaselineParams,
  type EditBaselineParams,
} from "./api-baselines";
import type {
  BaselineManifest,
  DiffBaselineResponse,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

export function useBaselines(
  scenario: string,
  opts: { allBranches?: boolean; branch?: string; enabled?: boolean; repoId?: string | null } = {},
) {
  const { allBranches = false, branch, enabled = true, repoId } = opts;
  const scope = allBranches ? "all" : branch ?? "current";
  return useQuery<BaselineManifest[], Error>({
    queryKey: queryKeys.baselines(scenario, scope, repoId),
    queryFn: () => listBaselines({ scenario, allBranches, branch, repoId }),
    enabled: enabled && Boolean(scenario),
    staleTime: 10_000,
  });
}

export function useBaseline(
  scenario: string,
  name: string,
  branch: string,
  opts: { enabled?: boolean; repoId?: string | null } = {},
) {
  const { enabled = true, repoId } = opts;
  return useQuery<BaselineManifest | undefined, Error>({
    queryKey: queryKeys.baseline(scenario, name, branch, repoId),
    queryFn: () => getBaseline({ scenario, name, branch, repoId }),
    enabled: enabled && Boolean(scenario) && Boolean(name),
    staleTime: 30_000,
  });
}

export function useBaselineDiff(
  scenario: string,
  name: string,
  branch: string,
  opts: { surface?: string; enabled?: boolean; repoId?: string | null } = {},
) {
  const { surface = "", enabled = true, repoId } = opts;
  return useQuery<DiffBaselineResponse, Error>({
    queryKey: queryKeys.baselineDiff(scenario, name, branch, surface, repoId),
    queryFn: () => diffBaseline({ scenario, name, branch, surface, repoId }),
    // A diff re-runs the surfaces server-side; never poll it automatically.
    enabled: enabled && Boolean(scenario) && Boolean(name),
    staleTime: 0,
    gcTime: 60_000,
  });
}

export function useCreateBaseline(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<unknown, Error, SnapshotBaselineParams>({
    mutationFn: (params) => snapshotForBaseline({ ...params, repoId }),
    onSuccess: (_data, params) => {
      queryClient.invalidateQueries({ queryKey: ["baselines", repoId ?? "default", params.scenario] });
    },
  });
}

export function useDeleteBaseline(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<boolean, Error, { scenario: string; name: string; branch: string }>({
    mutationFn: ({ scenario, name, branch }) => deleteBaseline({ scenario, name, branch, repoId }),
    onSuccess: (_data, { scenario }) => {
      queryClient.invalidateQueries({ queryKey: ["baselines", repoId ?? "default", scenario] });
    },
  });
}

export function useEditBaseline(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<unknown, Error, EditBaselineParams>({
    mutationFn: (params) => editBaseline({ ...params, repoId }),
    onSuccess: (_data, params) => {
      queryClient.invalidateQueries({ queryKey: ["baselines", repoId ?? "default", params.scenario] });
    },
  });
}

// ── Default baseline (Decision 4) ────────────────────────────────────────
// UI-only convenience: one baseline per scenario is marked "default" so the
// Tests/Rules/Overview tabs know which baseline to diff against. Stored in
// localStorage — per-device UX, not part of the substrate. The CLI/API always
// require an explicit name.

const DEFAULT_BASELINE_PREFIX = "gct.defaultBaseline.";

function readDefaultBaseline(scenario: string): string | null {
  if (typeof window === "undefined") return null;
  try {
    return window.localStorage.getItem(DEFAULT_BASELINE_PREFIX + scenario);
  } catch {
    return null;
  }
}

export function useDefaultBaseline(scenario: string) {
  const [name, setNameState] = useState<string | null>(() => readDefaultBaseline(scenario));

  // Re-sync when switching scenarios (the initializer only runs once).
  useEffect(() => {
    setNameState(readDefaultBaseline(scenario));
  }, [scenario]);

  const setDefaultBaseline = useCallback(
    (next: string | null) => {
      setNameState(next);
      if (typeof window === "undefined") return;
      try {
        if (next) {
          window.localStorage.setItem(DEFAULT_BASELINE_PREFIX + scenario, next);
        } else {
          window.localStorage.removeItem(DEFAULT_BASELINE_PREFIX + scenario);
        }
      } catch {
        // Ignore storage errors; default selection still works in-memory.
      }
    },
    [scenario],
  );

  return { defaultBaselineName: name, setDefaultBaseline };
}
