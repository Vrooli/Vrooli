/**
 * Custom hook for managing TemplateEditorModal form state.
 * Orchestrates form initialization, validation, unsaved changes tracking,
 * and multi-item editing.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import type { Template, TemplateVariable, TemplateWithSource } from "@/lib/types/templates";
import { fillTemplateContent, getTemplateModesAtLevel } from "@/data/templates";
import type { SaveOptions } from "./types";
import { useMultiItemEditing } from "./useMultiItemEditing";

interface UseTemplateEditorFormParams {
  template?: Template;
  templateSource?: string;
  defaultModes?: string[];
  open: boolean;
  readOnly: boolean;
  allTemplates?: TemplateWithSource[];
  onSave?: (
    template: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
    options?: SaveOptions
  ) => void;
  onClose: () => void;
  onSelectTemplate?: (template: TemplateWithSource) => void;
  onSaveAll?: (templates: Array<{ id: string; data: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">; options?: SaveOptions }>) => Promise<void>;
}

export function useTemplateEditorForm({
  template,
  templateSource,
  defaultModes,
  open,
  readOnly,
  allTemplates,
  onSave,
  onClose,
  onSelectTemplate,
  onSaveAll,
}: UseTemplateEditorFormParams) {
  const isEditing = !!template;
  const isEditingDefault = isEditing && templateSource === "default" && !readOnly;

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("Sparkles");
  const [modes, setModes] = useState<string[]>([]);
  const [content, setContent] = useState("");
  const [variables, setVariables] = useState<TemplateVariable[]>([]);
  const [showPreview, setShowPreview] = useState(false);
  const [applyToDefault, setApplyToDefault] = useState(false);
  const [draft, setDraft] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  // Track unsaved changes
  const hasUnsavedChanges = useMemo(() => {
    if (readOnly) return false;
    if (!template) {
      return !!(name.trim() || description.trim() || content.trim() || modes.length > 0 || variables.length > 0 || draft);
    }
    return (
      name !== template.name ||
      description !== template.description ||
      icon !== (template.icon || "Sparkles") ||
      JSON.stringify(modes) !== JSON.stringify(template.modes || []) ||
      content !== template.content ||
      JSON.stringify(variables) !== JSON.stringify(template.variables) ||
      draft !== (template.draft || false)
    );
  }, [readOnly, template, name, description, icon, modes, content, variables, draft]);

  const getCurrentFormState = useCallback(() => ({
    name, description, icon, modes, content, variables,
    selectedToolIds: [] as string[], applyToDefault, draft,
  }), [name, description, icon, modes, content, variables, applyToDefault, draft]);

  // Multi-item editing (delegated hook)
  const multiItem = useMultiItemEditing({
    template, templateSource, readOnly, allTemplates,
    onSelectTemplate, onSaveAll, hasUnsavedChanges, getCurrentFormState,
    name, modes, icon, isEditingDefault,
  });

  // Handle close with unsaved changes check
  const handleClose = useCallback(() => {
    if (hasUnsavedChanges || multiItem.pendingChanges.size > 0) {
      setShowCloseConfirm(true);
    } else {
      onClose();
    }
  }, [hasUnsavedChanges, multiItem.pendingChanges.size, onClose]);

  // Handle escape key
  useEffect(() => {
    if (!open) return;
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        e.stopImmediatePropagation();
        handleClose();
      }
    };
    document.addEventListener("keydown", handleEscape, { capture: true });
    return () => document.removeEventListener("keydown", handleEscape, { capture: true });
  }, [open, handleClose]);

  // Initialize form when template changes
  useEffect(() => {
    if (template) {
      const pending = multiItem.pendingChanges.get(template.id);
      if (pending) {
        setName(pending.name);
        setDescription(pending.description);
        setIcon(pending.icon);
        setModes(pending.modes);
        setContent(pending.content);
        setVariables(pending.variables);
        setApplyToDefault(pending.applyToDefault);
        setDraft(pending.draft);
      } else {
        setName(template.name);
        setDescription(template.description);
        setIcon(template.icon || "Sparkles");
        setModes(template.modes || []);
        setContent(template.content);
        setVariables(template.variables);
        setApplyToDefault(false);
        setDraft(template.draft || false);
      }
    } else {
      setName("");
      setDescription("");
      setIcon("Sparkles");
      setModes(defaultModes || []);
      setContent("");
      setVariables([]);
      setApplyToDefault(false);
      setDraft(false);
    }
    setErrors({});
    setShowPreview(false);
  }, [
    template,
    defaultModes,
    open,
    multiItem.pendingChanges,
  ]);

  // Variable management
  const addVariable = useCallback(() => {
    setVariables((prev) => [
      ...prev,
      { name: `variable_${prev.length + 1}`, label: `Variable ${prev.length + 1}`, type: "text", required: false },
    ]);
  }, []);

  const updateVariable = useCallback(
    (index: number, updates: Partial<TemplateVariable>) => {
      setVariables((prev) => prev.map((v, i) => (i === index ? { ...v, ...updates } : v)));
    }, []
  );

  const removeVariable = useCallback((index: number) => {
    setVariables((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const getSuggestionsAtLevel = useCallback(
    (level: number, parentPath: string[]): string[] => getTemplateModesAtLevel(level, parentPath), []
  );

  // Detect undefined variables in content
  const undefinedVariables = useMemo(() => {
    const matches = content.match(/\{\{(\w+)\}\}/g) || [];
    const contentVars = [...new Set(matches.map(m => m.slice(2, -2)))];
    const definedVars = new Set(variables.map(v => v.name));
    return contentVars.filter(v => !definedVars.has(v));
  }, [content, variables]);

  // Validate form
  const validate = useCallback(() => {
    const newErrors: Record<string, string> = {};
    if (!name.trim()) newErrors.name = "Name is required";
    if (!description.trim()) newErrors.description = "Description is required";
    if (!content.trim()) newErrors.content = "Content is required";
    if (modes.length === 0) newErrors.modes = "At least one mode is required";
    variables.forEach((v, i) => {
      if (!v.name.trim()) newErrors[`variable_${i}_name`] = "Variable name is required";
      if (!v.label.trim()) newErrors[`variable_${i}_label`] = "Variable label is required";
      if (v.type === "select" && (!v.options || v.options.length === 0)) {
        newErrors[`variable_${i}_options`] = "Select type requires at least one option";
      }
    });
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [name, description, content, modes, variables]);

  // Handle save
  const handleSave = useCallback(() => {
    if (readOnly || !onSave) return;
    if (!validate()) return;
    onSave(
      {
        name: name.trim(), description: description.trim(), icon, modes,
        content: content.trim(), variables,
        draft: draft || undefined,
      },
      isEditingDefault ? { applyToDefault } : undefined
    );
    onClose();
  }, [readOnly, onSave, validate, name, description, icon, modes, content, variables, onClose, isEditingDefault, applyToDefault, draft]);

  // Generate preview content
  const previewContent = useCallback(() => {
    const values: Record<string, string> = {};
    variables.forEach((v) => { values[v.name] = v.placeholder || `[${v.label}]`; });
    return fillTemplateContent({ content, variables } as Template, values);
  }, [content, variables]);

  return {
    isEditing, isEditingDefault,
    name, setName, description, setDescription, icon, setIcon,
    modes, setModes, content, setContent, variables,
    showPreview, setShowPreview, applyToDefault, setApplyToDefault,
    draft, setDraft, errors, undefinedVariables,
    // Variable management
    addVariable, updateVariable, removeVariable, getSuggestionsAtLevel,
    // Multi-item editing
    pendingChanges: multiItem.pendingChanges,
    setPendingChanges: multiItem.setPendingChanges,
    expandedTreeNodes: multiItem.expandedTreeNodes,
    isSavingAll: multiItem.isSavingAll,
    isSidebarCollapsed: multiItem.isSidebarCollapsed,
    setIsSidebarCollapsed: multiItem.setIsSidebarCollapsed,
    dirtyItemIds: multiItem.dirtyItemIds,
    dirtyCount: multiItem.dirtyCount,
    itemsForTree: multiItem.itemsForTree,
    handleSelectTemplate: multiItem.handleSelectTemplate,
    toggleTreeNode: multiItem.toggleTreeNode,
    handleSaveAll: multiItem.handleSaveAll,
    // Actions
    handleClose, handleSave, previewContent,
    showCloseConfirm, setShowCloseConfirm, hasUnsavedChanges,
  };
}
