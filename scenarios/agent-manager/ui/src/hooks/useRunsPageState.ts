// Shared Runs page state for list controls and investigation selection.
// AI_CHECK: react_coherence=1 | LAST: 2026-02-06

import { useCallback, useState } from "react";

interface RunListItem {
  id: string;
}

export function useRunsPageState() {
  const initialParams = typeof window === "undefined" ? new URLSearchParams() : new URLSearchParams(window.location.search);
  const [searchQuery, setSearchQueryState] = useState(initialParams.get("q") ?? "");
  const [statusFilter, setStatusFilterState] = useState<string>(initialParams.get("status") ?? "all");
  const [sortBy, setSortByState] = useState<string>(initialParams.get("sort") ?? "newest");

  const updateUrl = useCallback((key: string, value: string, defaultValue: string) => {
    if (typeof window === "undefined") return;
    const params = new URLSearchParams(window.location.search);
    if (!value || value === defaultValue) params.delete(key);
    else params.set(key, value);
    const query = params.toString();
    window.history.replaceState(window.history.state, "", `${window.location.pathname}${query ? `?${query}` : ""}${window.location.hash}`);
  }, []);
  const setSearchQuery = useCallback((value: string) => { setSearchQueryState(value); updateUrl("q", value, ""); }, [updateUrl]);
  const setStatusFilter = useCallback((value: string) => { setStatusFilterState(value); updateUrl("status", value, "all"); }, [updateUrl]);
  const setSortBy = useCallback((value: string) => { setSortByState(value); updateUrl("sort", value, "newest"); }, [updateUrl]);

  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedRunIds, setSelectedRunIds] = useState<Set<string>>(new Set());
  const [lastClickedIndex, setLastClickedIndex] = useState<number | null>(null);

  const [investigateModalOpen, setInvestigateModalOpen] = useState(false);
  const [investigateLoading, setInvestigateLoading] = useState(false);
  const [investigateError, setInvestigateError] = useState<string | null>(null);

  const toggleSelectionMode = useCallback(() => {
    setSelectionMode((prevSelectionMode) => {
      if (prevSelectionMode) {
        setSelectedRunIds(new Set());
        setLastClickedIndex(null);
      }
      return !prevSelectionMode;
    });
  }, []);

  const clearSelection = useCallback(() => {
    setSelectedRunIds(new Set());
    setLastClickedIndex(null);
  }, []);

  const handleRunCheckboxChange = useCallback(
    (runId: string, index: number, shiftKey: boolean, visibleRuns: RunListItem[]) => {
      setSelectedRunIds((prev) => {
        const next = new Set(prev);

        if (shiftKey && lastClickedIndex !== null) {
          const start = Math.min(lastClickedIndex, index);
          const end = Math.min(Math.max(lastClickedIndex, index), visibleRuns.length - 1);
          for (let i = start; i <= end; i++) {
            const run = visibleRuns[i];
            if (run) next.add(run.id);
          }
        } else if (next.has(runId)) {
          next.delete(runId);
        } else {
          next.add(runId);
        }

        return next;
      });
      setLastClickedIndex(index);
    },
    [lastClickedIndex]
  );

  return {
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    sortBy,
    setSortBy,
    selectionMode,
    setSelectionMode,
    selectedRunIds,
    setSelectedRunIds,
    investigateModalOpen,
    setInvestigateModalOpen,
    investigateLoading,
    setInvestigateLoading,
    investigateError,
    setInvestigateError,
    toggleSelectionMode,
    clearSelection,
    handleRunCheckboxChange,
  };
}
