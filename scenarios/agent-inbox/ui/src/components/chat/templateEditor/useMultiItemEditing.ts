/**
 * Hook for managing multi-item editing state in the template editor.
 * Handles pending changes across multiple templates, tree sidebar state,
 * and batch save operations.
 */

import { useCallback, useMemo, useState } from "react";
import type { Template, TemplateWithSource } from "@/lib/types/templates";
import type { TemplateFormState, SaveOptions } from "./types";

interface UseMultiItemEditingParams {
  template?: Template;
  templateSource?: string;
  readOnly: boolean;
  allTemplates?: TemplateWithSource[];
  onSelectTemplate?: (template: TemplateWithSource) => void;
  onSaveAll?: (templates: Array<{ id: string; data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">; options?: SaveOptions }>) => Promise<void>;
  // Current form state from the parent hook
  hasUnsavedChanges: boolean;
  getCurrentFormState: () => TemplateFormState;
  // Live form field values for tree display
  name: string;
  modes: string[];
  icon: string;
  // For save operations
  isEditingDefault: boolean;
}

export function useMultiItemEditing({
  template,
  readOnly,
  allTemplates,
  onSelectTemplate,
  onSaveAll,
  hasUnsavedChanges,
  getCurrentFormState,
  name,
  modes,
  icon,
  isEditingDefault,
}: UseMultiItemEditingParams) {
  const [pendingChanges, setPendingChanges] = useState<Map<string, TemplateFormState>>(new Map());
  const [expandedTreeNodes, setExpandedTreeNodes] = useState<Set<string>>(new Set());
  const [isSavingAll, setIsSavingAll] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  // Store current changes in pending when switching templates
  const storeCurrentChanges = useCallback(() => {
    if (!template?.id || readOnly) return;
    if (hasUnsavedChanges) {
      setPendingChanges((prev) => {
        const next = new Map(prev);
        next.set(template.id, getCurrentFormState());
        return next;
      });
    }
  }, [template?.id, readOnly, hasUnsavedChanges, getCurrentFormState]);

  // Handle switching to a different template
  const handleSelectTemplate = useCallback((selectedTemplate: TemplateWithSource) => {
    if (selectedTemplate.id === template?.id) return;
    storeCurrentChanges();
    onSelectTemplate?.(selectedTemplate);
  }, [template?.id, storeCurrentChanges, onSelectTemplate]);

  // Toggle tree node expansion
  const toggleTreeNode = useCallback((nodeId: string) => {
    setExpandedTreeNodes((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }, []);

  // Compute set of dirty item IDs for the sidebar
  const dirtyItemIds = useMemo(() => {
    const ids = new Set<string>();
    for (const [id] of pendingChanges.entries()) {
      ids.add(id);
    }
    if (template?.id && hasUnsavedChanges) {
      ids.add(template.id);
    }
    return ids;
  }, [pendingChanges, template?.id, hasUnsavedChanges]);

  const dirtyCount = dirtyItemIds.size;

  // Compute merged items for tree display
  const itemsForTree = useMemo(() => {
    if (!allTemplates) return [];
    return allTemplates.map((item) => {
      if (item.id === template?.id) {
        return { ...item, name, modes, icon };
      }
      const pending = pendingChanges.get(item.id);
      if (pending) {
        return { ...item, name: pending.name, modes: pending.modes, icon: pending.icon };
      }
      return item;
    });
  }, [allTemplates, pendingChanges, template?.id, name, modes, icon]);

  // Handle Save All
  const handleSaveAll = useCallback(async () => {
    if (!onSaveAll || dirtyCount === 0) return;
    setIsSavingAll(true);
    try {
      const currentState = getCurrentFormState();
      const updates: Array<{ id: string; data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">; options?: SaveOptions }> = [];

      if (template?.id && hasUnsavedChanges) {
        updates.push({
          id: template.id,
          data: {
            name: currentState.name.trim(),
            description: currentState.description.trim(),
            icon: currentState.icon,
            modes: currentState.modes,
            content: currentState.content.trim(),
            variables: currentState.variables,
            draft: currentState.draft || undefined,
          },
          options: isEditingDefault ? { applyToDefault: currentState.applyToDefault } : undefined,
        });
      }

      for (const [id, state] of pendingChanges.entries()) {
        if (id === template?.id) continue;
        const originalTemplate = allTemplates?.find((t) => t.id === id);
        const isDefault = originalTemplate?.source === "default";
        updates.push({
          id,
          data: {
            name: state.name.trim(),
            description: state.description.trim(),
            icon: state.icon,
            modes: state.modes,
            content: state.content.trim(),
            variables: state.variables,
            draft: state.draft || undefined,
          },
          options: isDefault ? { applyToDefault: state.applyToDefault } : undefined,
        });
      }

      await onSaveAll(updates);
      setPendingChanges(new Map());
    } finally {
      setIsSavingAll(false);
    }
  }, [onSaveAll, dirtyCount, template?.id, hasUnsavedChanges, getCurrentFormState, isEditingDefault, pendingChanges, allTemplates]);

  return {
    pendingChanges,
    setPendingChanges,
    expandedTreeNodes,
    isSavingAll,
    isSidebarCollapsed,
    setIsSidebarCollapsed,
    dirtyItemIds,
    dirtyCount,
    itemsForTree,
    handleSelectTemplate,
    toggleTreeNode,
    handleSaveAll,
  };
}
