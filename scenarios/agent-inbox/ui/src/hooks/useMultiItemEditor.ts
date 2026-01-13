/**
 * useMultiItemEditor - Hook for managing multiple items with pending changes.
 *
 * Features:
 * - Track pending changes per item in a Map
 * - Compute dirty state by comparing original vs current
 * - Store form state when switching items (preserve in memory)
 * - Batch save all dirty items
 */

import { useState, useCallback, useMemo } from "react";
import { buildTree, type TreeNode } from "@/components/shared/ItemTreeSidebar";

interface BaseItem {
  id: string;
  name: string;
  modes?: string[];
}

export interface PendingChange<T> {
  original: T;
  current: T;
  isDirty: boolean;
}

interface UseMultiItemEditorOptions<T extends BaseItem> {
  items: T[];
  initialSelectedId?: string | null;
  onSaveItem: (id: string, data: T) => Promise<void>;
  compareItems?: (a: T, b: T) => boolean;
}

interface UseMultiItemEditorReturn<T extends BaseItem> {
  // Selection
  selectedItemId: string | null;
  selectedItem: T | null;
  selectItem: (id: string) => void;

  // Pending changes
  pendingChanges: Map<string, PendingChange<T>>;
  getCurrentItemData: () => T | null;
  updateCurrentItem: (updates: Partial<T>) => void;
  hasUnsavedChanges: (id: string) => boolean;
  getDirtyItemIds: () => string[];
  getDirtyCount: () => number;

  // Save operations
  saveItem: (id: string) => Promise<void>;
  saveAllDirty: () => Promise<void>;
  discardChanges: (id: string) => void;
  discardAllChanges: () => void;
  isSaving: boolean;

  // Tree structure
  treeData: TreeNode[];
  expandedNodes: Set<string>;
  toggleNode: (nodeId: string) => void;
  expandToItem: (itemId: string) => void;
}

/**
 * Default comparison function using JSON.stringify.
 */
function defaultCompare<T>(a: T, b: T): boolean {
  return JSON.stringify(a) === JSON.stringify(b);
}

export function useMultiItemEditor<T extends BaseItem>({
  items,
  initialSelectedId = null,
  onSaveItem,
  compareItems = defaultCompare,
}: UseMultiItemEditorOptions<T>): UseMultiItemEditorReturn<T> {
  const [selectedItemId, setSelectedItemId] = useState<string | null>(initialSelectedId);
  const [pendingChanges, setPendingChanges] = useState<Map<string, PendingChange<T>>>(new Map());
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  const [isSaving, setIsSaving] = useState(false);

  // Build tree from items
  const treeData = useMemo(() => buildTree(items), [items]);

  // Get the selected item (either from pending changes or original)
  const selectedItem = useMemo(() => {
    if (!selectedItemId) return null;
    const pending = pendingChanges.get(selectedItemId);
    if (pending) return pending.current;
    return items.find((i) => i.id === selectedItemId) ?? null;
  }, [selectedItemId, pendingChanges, items]);

  // Get current item data (from pending if exists, else original)
  const getCurrentItemData = useCallback((): T | null => {
    if (!selectedItemId) return null;
    const pending = pendingChanges.get(selectedItemId);
    if (pending) return pending.current;
    return items.find((i) => i.id === selectedItemId) ?? null;
  }, [selectedItemId, pendingChanges, items]);

  // Toggle tree node expansion
  const toggleNode = useCallback((nodeId: string) => {
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }, []);

  // Expand tree nodes to show a specific item
  const expandToItem = useCallback(
    (itemId: string) => {
      const item = items.find((i) => i.id === itemId);
      if (!item?.modes) return;

      setExpandedNodes((prev) => {
        const next = new Set(prev);
        let path = "";
        for (const mode of item.modes!) {
          path = path ? `${path}/${mode}` : mode;
          next.add(path);
        }
        return next;
      });
    },
    [items]
  );

  // Select an item (switches to it, preserving any pending changes)
  const selectItem = useCallback(
    (id: string) => {
      if (id === selectedItemId) return;

      // Expand tree to show the item
      expandToItem(id);

      setSelectedItemId(id);
    },
    [selectedItemId, expandToItem]
  );

  // Update the currently selected item
  const updateCurrentItem = useCallback(
    (updates: Partial<T>) => {
      if (!selectedItemId) return;

      setPendingChanges((prev) => {
        const next = new Map(prev);
        const existing = next.get(selectedItemId);
        const original = existing?.original ?? items.find((i) => i.id === selectedItemId);

        if (!original) return prev;

        const current = { ...(existing?.current ?? original), ...updates } as T;
        const isDirty = !compareItems(original, current);

        next.set(selectedItemId, { original, current, isDirty });
        return next;
      });
    },
    [selectedItemId, items, compareItems]
  );

  // Check if a specific item has unsaved changes
  const hasUnsavedChanges = useCallback(
    (id: string): boolean => {
      const pending = pendingChanges.get(id);
      return pending?.isDirty ?? false;
    },
    [pendingChanges]
  );

  // Get all dirty item IDs
  const getDirtyItemIds = useCallback((): string[] => {
    return Array.from(pendingChanges.entries())
      .filter(([, pending]) => pending.isDirty)
      .map(([id]) => id);
  }, [pendingChanges]);

  // Count dirty items
  const getDirtyCount = useCallback((): number => {
    return Array.from(pendingChanges.values()).filter((p) => p.isDirty).length;
  }, [pendingChanges]);

  // Save a single item
  const saveItem = useCallback(
    async (id: string): Promise<void> => {
      const pending = pendingChanges.get(id);
      if (!pending?.isDirty) return;

      setIsSaving(true);
      try {
        await onSaveItem(id, pending.current);

        // Remove from pending changes after successful save
        setPendingChanges((prev) => {
          const next = new Map(prev);
          next.delete(id);
          return next;
        });
      } finally {
        setIsSaving(false);
      }
    },
    [pendingChanges, onSaveItem]
  );

  // Save all dirty items
  const saveAllDirty = useCallback(async (): Promise<void> => {
    const dirtyIds = getDirtyItemIds();
    if (dirtyIds.length === 0) return;

    setIsSaving(true);
    const savedIds: string[] = [];

    try {
      for (const id of dirtyIds) {
        const pending = pendingChanges.get(id);
        if (pending?.isDirty) {
          await onSaveItem(id, pending.current);
          savedIds.push(id);
        }
      }

      // Remove all successfully saved items from pending changes
      setPendingChanges((prev) => {
        const next = new Map(prev);
        for (const id of savedIds) {
          next.delete(id);
        }
        return next;
      });
    } finally {
      setIsSaving(false);
    }
  }, [getDirtyItemIds, pendingChanges, onSaveItem]);

  // Discard changes for a single item
  const discardChanges = useCallback((id: string) => {
    setPendingChanges((prev) => {
      const next = new Map(prev);
      next.delete(id);
      return next;
    });
  }, []);

  // Discard all changes
  const discardAllChanges = useCallback(() => {
    setPendingChanges(new Map());
  }, []);

  // Computed set of dirty item IDs for the sidebar
  const dirtyItemIdsSet = useMemo(() => {
    return new Set(getDirtyItemIds());
  }, [getDirtyItemIds]);

  return {
    selectedItemId,
    selectedItem,
    selectItem,
    pendingChanges,
    getCurrentItemData,
    updateCurrentItem,
    hasUnsavedChanges,
    getDirtyItemIds,
    getDirtyCount,
    saveItem,
    saveAllDirty,
    discardChanges,
    discardAllChanges,
    isSaving,
    treeData,
    expandedNodes,
    toggleNode,
    expandToItem,
  };
}
