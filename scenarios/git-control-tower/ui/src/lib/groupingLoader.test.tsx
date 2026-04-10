/**
 * Regression tests for grouping rules loading race condition.
 *
 * Bug: When loading git-control-tower behind a proxy, grouping rules disappear
 * on first load after proxy cache expires. Root cause: the loader effect set
 * `groupingLoadedKey` unconditionally, even while the grouping-rules API was
 * still loading. This prevented the effect from processing the data when it
 * arrived later, since the `groupingLoadedKey === repoKey` guard caused an
 * early return.
 *
 * The fix: only set `groupingLoadedKey` inside the branches that actually
 * process data (data available, or done loading with no data).
 */
import "@testing-library/jest-dom";
import { renderHook, waitFor } from "@testing-library/react";
import { useState, useEffect, useCallback, useMemo } from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";

// ---------------------------------------------------------------------------
// Minimal reproduction of the App.tsx grouping-loading effect, isolated as a
// testable hook. This mirrors the EXACT logic from App.tsx lines 1308-1376
// (post-fix) so the test validates the real code pattern.
// ---------------------------------------------------------------------------

interface GroupingRule {
  id: string;
  label: string;
  prefixes: string[];
  mode: "prefix" | "segment";
}

interface GroupingRulesConfig {
  enabled: boolean;
  rules: Array<{
    id: string;
    label: string;
    prefixes: string[];
    mode: string;
  }>;
}

interface QueryResult<T> {
  data: T | undefined;
  isLoading: boolean;
}

/**
 * Reproduces the exact loading logic from App.tsx to test the race condition.
 */
function useGroupingLoaderLogic(
  repoDir: string | undefined,
  groupingRulesQuery: QueryResult<GroupingRulesConfig>,
) {
  const repoKey = useMemo(
    () => (repoDir ? encodeURIComponent(repoDir) : "unknown"),
    [repoDir],
  );
  const [groupingRules, setGroupingRules] = useState<GroupingRule[]>([]);
  const [groupingLoadedKey, setGroupingLoadedKey] = useState<string | null>(null);
  const [groupingDefaultsPending, setGroupingDefaultsPending] = useState(false);
  const [saveCallCount, setSaveCallCount] = useState(0);
  const [lastSavedRules, setLastSavedRules] = useState<GroupingRule[]>([]);

  const normalizeGroupingRules = useCallback((rawRules: GroupingRule[]) => rawRules, []);

  // --- This effect mirrors App.tsx lines 1308-1376 (post-fix) ---
  useEffect(() => {
    if (!repoDir) return;
    if (groupingLoadedKey === repoKey) return;

    if (groupingRulesQuery.data) {
      const apiRules = groupingRulesQuery.data.rules ?? [];
      const uiRules: GroupingRule[] = apiRules.map((r) => ({
        id: r.id,
        label: r.label,
        prefixes: r.prefixes,
        mode: r.mode as "prefix" | "segment",
      }));
      setGroupingRules(normalizeGroupingRules(uiRules));
      setGroupingDefaultsPending(apiRules.length === 0);
      setGroupingLoadedKey(repoKey);
    } else if (!groupingRulesQuery.isLoading) {
      setGroupingRules([]);
      setGroupingDefaultsPending(true);
      setGroupingLoadedKey(repoKey);
    }
    // When isLoading=true and data=undefined, do NOT set groupingLoadedKey.
  }, [repoDir, repoKey, groupingLoadedKey, normalizeGroupingRules, groupingRulesQuery.data, groupingRulesQuery.isLoading]);

  // --- Mirrors the save effect at App.tsx lines 1406-1421 ---
  useEffect(() => {
    if (!repoDir || groupingLoadedKey !== repoKey) return;
    setSaveCallCount((c) => c + 1);
    setLastSavedRules([...groupingRules]);
  }, [repoDir, repoKey, groupingLoadedKey, groupingRules]);

  return { groupingRules, groupingLoadedKey, groupingDefaultsPending, saveCallCount, lastSavedRules };
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

interface HookProps {
  repoDir: string | undefined;
  query: QueryResult<GroupingRulesConfig>;
}

describe("grouping loader race condition (regression)", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("loads rules when API data arrives after repoDir (slow proxy)", async () => {
    // Simulate: status API responds first (repoDir available), but
    // grouping-rules API is still loading.
    const savedRules: GroupingRulesConfig = {
      enabled: true,
      rules: [
        { id: "g1", label: "Scenarios", prefixes: ["scenarios/"], mode: "segment" },
      ],
    };

    // Phase 1: repoDir available, grouping query still loading
    const { result, rerender } = renderHook(
      ({ repoDir, query }: HookProps) =>
        useGroupingLoaderLogic(repoDir, query),
      {
        initialProps: {
          repoDir: "/home/user/repo",
          query: { data: undefined, isLoading: true } as QueryResult<GroupingRulesConfig>,
        },
      },
    );

    // groupingLoadedKey should NOT be set while loading
    expect(result.current.groupingLoadedKey).toBeNull();
    expect(result.current.groupingRules).toEqual([]);

    // Phase 2: API data arrives
    rerender({
      repoDir: "/home/user/repo",
      query: { data: savedRules, isLoading: false },
    });

    // Now the rules should be loaded
    await waitFor(() => {
      expect(result.current.groupingLoadedKey).not.toBeNull();
      expect(result.current.groupingRules).toHaveLength(1);
      expect((result.current.groupingRules[0] as { label: string } | undefined)?.label).toBe("Scenarios");
    });
  });

  it("loads rules immediately when API data is already cached (warm proxy)", () => {
    const savedRules: GroupingRulesConfig = {
      enabled: true,
      rules: [
        { id: "g1", label: "Scenarios", prefixes: ["scenarios/"], mode: "segment" },
      ],
    };

    const { result } = renderHook(() =>
      useGroupingLoaderLogic("/home/user/repo", {
        data: savedRules,
        isLoading: false,
      }),
    );

    expect(result.current.groupingLoadedKey).not.toBeNull();
    expect(result.current.groupingRules).toHaveLength(1);
    expect((result.current.groupingRules[0] as { label: string } | undefined)?.label).toBe("Scenarios");
  });

  it("does not save empty rules while API is still loading", async () => {
    const savedRules: GroupingRulesConfig = {
      enabled: true,
      rules: [
        { id: "g1", label: "Scenarios", prefixes: ["scenarios/"], mode: "segment" },
      ],
    };

    // Phase 1: loading
    const { result, rerender } = renderHook(
      ({ repoDir, query }: HookProps) =>
        useGroupingLoaderLogic(repoDir, query),
      {
        initialProps: {
          repoDir: "/home/user/repo",
          query: { data: undefined, isLoading: true } as QueryResult<GroupingRulesConfig>,
        },
      },
    );

    // Save should NOT have been called yet (loadedKey is still null)
    expect(result.current.saveCallCount).toBe(0);

    // Phase 2: data arrives
    rerender({
      repoDir: "/home/user/repo",
      query: { data: savedRules, isLoading: false },
    });

    await waitFor(() => {
      expect(result.current.saveCallCount).toBeGreaterThan(0);
    });

    // The save should contain the real rules, not empty
    expect(result.current.lastSavedRules).toHaveLength(1);
    expect((result.current.lastSavedRules[0] as { label: string } | undefined)?.label).toBe("Scenarios");
  });

  it("falls back to defaults when API returns empty and is done loading", () => {
    const { result } = renderHook(() =>
      useGroupingLoaderLogic("/home/user/repo", {
        data: undefined,
        isLoading: false,
      }),
    );

    expect(result.current.groupingLoadedKey).not.toBeNull();
    expect(result.current.groupingRules).toEqual([]);
    expect(result.current.groupingDefaultsPending).toBe(true);
  });

  it("does not load when repoDir is undefined", () => {
    const { result } = renderHook(() =>
      useGroupingLoaderLogic(undefined, {
        data: { enabled: true, rules: [{ id: "g1", label: "X", prefixes: ["x/"], mode: "prefix" }] },
        isLoading: false,
      }),
    );

    expect(result.current.groupingLoadedKey).toBeNull();
    expect(result.current.groupingRules).toEqual([]);
  });

  it("handles repoDir arriving after API data", async () => {
    // Simulate: grouping-rules API responds before status API
    const savedRules: GroupingRulesConfig = {
      enabled: true,
      rules: [
        { id: "g1", label: "Resources", prefixes: ["resources/"], mode: "segment" },
      ],
    };

    // Phase 1: no repoDir yet, but API data is ready
    const { result, rerender } = renderHook(
      ({ repoDir, query }: HookProps) =>
        useGroupingLoaderLogic(repoDir, query),
      {
        initialProps: {
          repoDir: undefined as string | undefined,
          query: { data: savedRules, isLoading: false },
        },
      },
    );

    expect(result.current.groupingLoadedKey).toBeNull();

    // Phase 2: repoDir arrives
    rerender({
      repoDir: "/home/user/repo",
      query: { data: savedRules, isLoading: false },
    });

    await waitFor(() => {
      expect(result.current.groupingRules).toHaveLength(1);
      expect((result.current.groupingRules[0] as { label: string } | undefined)?.label).toBe("Resources");
    });
  });
});
