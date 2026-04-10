/**
 * useSkillEditorMultiEdit - Multi-item editing state for SkillEditorModal.
 *
 * Manages pending changes across multiple skills when browsing with sidebar,
 * including save-all, dirty tracking, and tree display merging.
 */
import { useCallback, useMemo, useState } from "react";
import type { Skill, SkillWithSource } from "@/lib/types/templates";
import { buildSkillData, type SkillFormState } from "./skillEditorUtils";

interface UseSkillEditorMultiEditParams {
  skill?: Skill;
  readOnly: boolean;
  hasUnsavedChanges: boolean;
  formState: SkillFormState;
  allSkills?: SkillWithSource[];
  onSelectSkill?: (skill: SkillWithSource) => void;
  onSaveAll?: (skills: Array<{ id: string; data: Omit<Skill, "id" | "createdAt" | "updatedAt"> }>) => Promise<void>;
}

export function useSkillEditorMultiEdit({
  skill,
  readOnly,
  hasUnsavedChanges,
  formState,
  allSkills,
  onSelectSkill,
  onSaveAll,
}: UseSkillEditorMultiEditParams) {
  const [pendingChanges, setPendingChanges] = useState<Map<string, SkillFormState>>(new Map());
  const [expandedTreeNodes, setExpandedTreeNodes] = useState<Set<string>>(new Set());
  const [isSavingAll, setIsSavingAll] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  const { name, modes, icon } = formState;

  // Compute set of dirty item IDs for the sidebar
  const dirtyItemIds = useMemo(() => {
    const ids = new Set<string>();
    for (const [id] of pendingChanges.entries()) {
      ids.add(id);
    }
    if (skill?.id && hasUnsavedChanges) {
      ids.add(skill.id);
    }
    return ids;
  }, [pendingChanges, skill?.id, hasUnsavedChanges]);

  const dirtyCount = dirtyItemIds.size;

  // Store current changes in pending when switching skills
  const storeCurrentChanges = useCallback(() => {
    if (!skill?.id || readOnly || !hasUnsavedChanges) return;
    setPendingChanges((prev) => {
      const next = new Map(prev);
      next.set(skill.id, { ...formState });
      return next;
    });
  }, [skill?.id, readOnly, hasUnsavedChanges, formState]);

  // Handle switching to a different skill
  const handleSelectSkill = useCallback((selectedSkill: SkillWithSource) => {
    if (selectedSkill.id === skill?.id) return;
    storeCurrentChanges();
    onSelectSkill?.(selectedSkill);
  }, [skill?.id, storeCurrentChanges, onSelectSkill]);

  // Toggle tree node expansion
  const toggleTreeNode = useCallback((nodeId: string) => {
    setExpandedTreeNodes((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) next.delete(nodeId);
      else next.add(nodeId);
      return next;
    });
  }, []);

  // Compute merged items for tree display
  const itemsForTree = useMemo(() => {
    if (!allSkills) return [];
    return allSkills.map((item) => {
      if (item.id === skill?.id) {
        return { ...item, name, modes, icon };
      }
      const pending = pendingChanges.get(item.id);
      if (pending) {
        return { ...item, name: pending.name, modes: pending.modes, icon: pending.icon };
      }
      return item;
    });
  }, [allSkills, pendingChanges, skill?.id, name, modes, icon]);

  // Handle Save All
  const handleSaveAll = useCallback(async () => {
    if (!onSaveAll || dirtyCount === 0) return;
    setIsSavingAll(true);
    try {
      const updates: Array<{ id: string; data: Omit<Skill, "id" | "createdAt" | "updatedAt"> }> = [];

      if (skill?.id && hasUnsavedChanges) {
        updates.push({ id: skill.id, data: buildSkillData(formState) });
      }

      for (const [id, state] of pendingChanges.entries()) {
        if (id === skill?.id) continue;
        updates.push({ id, data: buildSkillData(state) });
      }

      await onSaveAll(updates);
      setPendingChanges(new Map());
    } finally {
      setIsSavingAll(false);
    }
  }, [onSaveAll, dirtyCount, skill?.id, hasUnsavedChanges, formState, pendingChanges]);

  const resetPendingChanges = useCallback(() => {
    setPendingChanges(new Map());
  }, []);

  return {
    pendingChanges,
    dirtyItemIds,
    dirtyCount,
    expandedTreeNodes,
    isSavingAll,
    isSidebarCollapsed,
    setIsSidebarCollapsed,
    handleSelectSkill,
    toggleTreeNode,
    itemsForTree,
    handleSaveAll,
    resetPendingChanges,
  };
}
