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
  DiffResult,
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
  return useQuery<DiffResult | undefined, Error>({
    queryKey: queryKeys.baselineDiff(scenario, name, branch, surface, repoId),
    queryFn: () => diffBaseline({ scenario, name, branch, surface, repoId }),
    // The diff resolves the verdict server-side (start + server-side wait);
    // never poll it automatically.
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

// ── On-demand compare (Plan C Decision 3) ─────────────────────────────────
// The single compare-trigger used by every surface tab AND the full
// BaselineCompareView. A diff re-executes the surface(s) server-side (minutes),
// so comparison is an explicit start/exit action, never an on-mount fetch.
//
// Pass an explicit `baselineName`/`branch` for a specific baseline (compare
// view), or omit them to compare against the per-scenario default baseline
// (Decision 4). A `surface` narrows the diff to one surface; "" diffs all.

export interface CompareOnDemand {
  /** True once the user has started the comparison. */
  comparing: boolean;
  start: () => void;
  exit: () => void;
  /** Resolved baseline being compared against ("" when none selected). */
  baselineName: string;
  /** The diff result, once a comparison has run. */
  diff: DiffResult | undefined;
  /** A diff is in flight. */
  isRunning: boolean;
  error: Error | null;
}

export function useCompareOnDemand(
  scenario: string,
  opts: {
    surface?: string;
    baselineName?: string;
    branch?: string;
    repoId?: string | null;
  } = {},
): CompareOnDemand {
  const { surface = "", branch = "", repoId } = opts;
  const { defaultBaselineName } = useDefaultBaseline(scenario);
  const baselineName = opts.baselineName ?? defaultBaselineName ?? "";

  const [comparing, setComparing] = useState(false);
  const start = useCallback(() => setComparing(true), []);
  const exit = useCallback(() => setComparing(false), []);

  const diffQuery = useBaselineDiff(scenario, baselineName, branch, {
    surface,
    enabled: comparing && Boolean(baselineName),
    repoId,
  });

  return {
    comparing,
    start,
    exit,
    baselineName,
    diff: diffQuery.data,
    isRunning: diffQuery.isLoading || diffQuery.isFetching,
    error: diffQuery.error,
  };
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
