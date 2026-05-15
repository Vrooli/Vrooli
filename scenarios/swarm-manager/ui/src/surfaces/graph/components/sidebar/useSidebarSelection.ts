import { useCallback, useEffect, useMemo, useState } from "react";
import type { SidebarTab } from "./types";

export type SelectableSidebarTab = "backlog" | "captures" | "initiatives" | "executions" | "sessions";

export const SELECTABLE_SIDEBAR_TABS: ReadonlySet<SidebarTab> = new Set([
  "backlog",
  "captures",
  "initiatives",
  "executions",
  "sessions",
]);

function isSelectableTab(tab: SidebarTab): tab is SelectableSidebarTab {
  return SELECTABLE_SIDEBAR_TABS.has(tab);
}

export function useSidebarSelection(activeTab: SidebarTab) {
  const [selectionTab, setSelectionTab] = useState<SelectableSidebarTab | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [visibleIds, setVisibleIds] = useState<string[]>([]);

  const selectionMode = selectionTab === activeTab && isSelectableTab(activeTab);
  const selectedCount = selectedIds.size;
  const selectable = isSelectableTab(activeTab);

  useEffect(() => {
    if (selectionTab !== null && selectionTab !== activeTab) {
      setSelectionTab(null);
      setSelectedIds(new Set());
      setVisibleIds([]);
    }
  }, [activeTab, selectionTab]);

  const clear = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  const cancelSelection = useCallback(() => {
    setSelectionTab(null);
    setSelectedIds(new Set());
    setVisibleIds([]);
  }, []);

  const toggleMode = useCallback(() => {
    if (!isSelectableTab(activeTab)) return;
    if (selectionTab === activeTab) {
      setSelectionTab(null);
      setSelectedIds(new Set());
      setVisibleIds([]);
      return;
    }
    setSelectionTab(activeTab);
    setSelectedIds(new Set());
  }, [activeTab, selectionTab]);

  const toggleItem = useCallback((id: string) => {
    setSelectedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const selectVisible = useCallback((ids: string[]) => {
    setSelectedIds(new Set(ids));
  }, []);

  const pruneToVisible = useCallback((ids: string[]) => {
    setVisibleIds(ids);
    const visible = new Set(ids);
    setSelectedIds((current) => {
      let changed = false;
      const next = new Set<string>();
      for (const id of current) {
        if (visible.has(id)) {
          next.add(id);
        } else {
          changed = true;
        }
      }
      return changed ? next : current;
    });
  }, []);

  const isSelected = useCallback((id: string) => selectedIds.has(id), [selectedIds]);
  const selectAllVisible = useCallback(() => setSelectedIds(new Set(visibleIds)), [visibleIds]);

  return useMemo(
    () => ({
      selectable,
      selectionMode,
      selectedIds,
      visibleIds,
      selectedCount,
      toggleMode,
      cancelSelection,
      toggleItem,
      selectVisible,
      selectAllVisible,
      pruneToVisible,
      clear,
      isSelected,
    }),
    [
      selectable,
      selectionMode,
      selectedIds,
      visibleIds,
      selectedCount,
      toggleMode,
      cancelSelection,
      toggleItem,
      selectVisible,
      selectAllVisible,
      pruneToVisible,
      clear,
      isSelected,
    ],
  );
}
