/**
 * Modal for creating, editing, and previewing skills.
 * Simpler than TemplateEditorModal since skills have no variables.
 *
 * Features:
 * - Edit name, description, icon, modes, content, tags
 * - Apply changes to default or save as custom
 * - Preview content
 * - Read-only mode for previewing without editing
 */

import { useCallback, useEffect, useMemo, useState, type ComponentType, type SVGProps } from "react";
import { X, Eye, ChevronDown, Info, Tag, Pencil, AlertTriangle, BookOpen, Loader2, Construction } from "lucide-react";
import * as LucideIcons from "lucide-react";
import type { Skill, SkillSource, SkillWithSource } from "@/lib/types/templates";
import { IconSelector, SKILL_ICON_OPTIONS } from "@/components/shared/IconSelector";
import { CategoryPathEditor } from "@/components/shared/CategoryPathEditor";
import { ItemTreeSidebar } from "@/components/shared/ItemTreeSidebar";
import { getSkillModesAtLevel } from "@/data/skills";

// Type for Lucide icon components
type IconComponent = ComponentType<SVGProps<SVGSVGElement> & { className?: string }>;

// Get icon component from name
function getIconComponent(name: string): IconComponent {
  const Icon = (LucideIcons as unknown as Record<string, IconComponent>)[name];
  return Icon || BookOpen;
}

interface SaveOptions {
  applyToDefault?: boolean;
}

interface SkillEditorModalProps {
  open: boolean;
  onClose: () => void;
  skill?: Skill; // Undefined for create, defined for edit
  skillSource?: SkillSource; // Source of the skill being edited
  onSave?: (
    skill: Omit<Skill, "id" | "createdAt" | "updatedAt">,
    options?: SaveOptions
  ) => void;
  readOnly?: boolean; // If true, modal is in preview mode (no editing)
  onEdit?: () => void; // Callback when Edit button is clicked in readOnly mode
  // Multi-item mode props
  allSkills?: SkillWithSource[]; // If provided, shows tree sidebar for navigation
  onSelectSkill?: (skill: SkillWithSource) => void; // Called when switching skills
  onSaveAll?: (skills: Array<{ id: string; data: Omit<Skill, "id" | "createdAt" | "updatedAt">; options?: SaveOptions }>) => Promise<void>;
}

// Form state for tracking changes
interface SkillFormState {
  name: string;
  description: string;
  icon: string;
  modes: string[];
  content: string;
  tagsInput: string;
  targetToolId: string;
  applyToDefault: boolean;
  draft: boolean;
}

export function SkillEditorModal({
  open,
  onClose,
  skill,
  skillSource,
  onSave,
  readOnly = false,
  onEdit,
  allSkills,
  onSelectSkill,
  onSaveAll,
}: SkillEditorModalProps) {
  const isEditing = !!skill;
  const isEditingDefault = isEditing && skillSource === "default" && !readOnly;
  const showSidebar = !!allSkills && allSkills.length > 0;

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("BookOpen");
  const [modes, setModes] = useState<string[]>([]);
  const [content, setContent] = useState("");
  const [tagsInput, setTagsInput] = useState("");
  const [targetToolId, setTargetToolId] = useState("");
  const [showPreview, setShowPreview] = useState(false);
  const [applyToDefault, setApplyToDefault] = useState(false);
  const [draft, setDraft] = useState(false);

  // Validation
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Confirmation dialog state
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  // Multi-item editing state
  const [pendingChanges, setPendingChanges] = useState<Map<string, SkillFormState>>(new Map());
  const [expandedTreeNodes, setExpandedTreeNodes] = useState<Set<string>>(new Set());
  const [isSavingAll, setIsSavingAll] = useState(false);
  const [isSidebarCollapsed, setIsSidebarCollapsed] = useState(false);

  // Track if there are unsaved changes
  const hasUnsavedChanges = useMemo(() => {
    if (readOnly) return false;
    if (!skill) {
      // Creating new - has changes if any field is filled
      return !!(name.trim() || description.trim() || content.trim() || tagsInput.trim() || targetToolId.trim() || modes.length > 0 || draft);
    }
    // Editing - compare to original
    const originalTags = skill.tags?.join(", ") || "";
    return (
      name !== skill.name ||
      description !== skill.description ||
      icon !== (skill.icon || "BookOpen") ||
      JSON.stringify(modes) !== JSON.stringify(skill.modes || []) ||
      content !== skill.content ||
      tagsInput !== originalTags ||
      targetToolId !== (skill.targetToolId || "") ||
      draft !== (skill.draft || false)
    );
  }, [readOnly, skill, name, description, icon, modes, content, tagsInput, targetToolId, draft]);

  // Get current form state as an object
  const getCurrentFormState = useCallback((): SkillFormState => ({
    name,
    description,
    icon,
    modes,
    content,
    tagsInput,
    targetToolId,
    applyToDefault,
    draft,
  }), [name, description, icon, modes, content, tagsInput, targetToolId, applyToDefault, draft]);

  // Store current changes in pending when switching skills
  const storeCurrentChanges = useCallback(() => {
    if (!skill?.id || readOnly) return;
    if (hasUnsavedChanges) {
      setPendingChanges((prev) => {
        const next = new Map(prev);
        next.set(skill.id, getCurrentFormState());
        return next;
      });
    }
  }, [skill?.id, readOnly, hasUnsavedChanges, getCurrentFormState]);

  // Handle switching to a different skill
  const handleSelectSkill = useCallback((selectedSkill: SkillWithSource) => {
    if (selectedSkill.id === skill?.id) return;

    // Store current changes before switching
    storeCurrentChanges();

    // Notify parent to switch skill
    onSelectSkill?.(selectedSkill);
  }, [skill?.id, storeCurrentChanges, onSelectSkill]);

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

    // Add items from pendingChanges
    for (const [id] of pendingChanges.entries()) {
      ids.add(id);
    }

    // Add current skill if it has unsaved changes
    if (skill?.id && hasUnsavedChanges) {
      ids.add(skill.id);
    }

    return ids;
  }, [pendingChanges, skill?.id, hasUnsavedChanges]);

  // Count total dirty items
  const dirtyCount = dirtyItemIds.size;

  // Compute merged items for tree display - reflects pending changes in real-time
  const itemsForTree = useMemo(() => {
    if (!allSkills) return [];

    return allSkills.map((item) => {
      // For current item, use live form state
      if (item.id === skill?.id) {
        return { ...item, name, modes, icon };
      }

      // For other items with pending changes, use their pending values
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
      const updates: Array<{ id: string; data: Omit<Skill, "id" | "createdAt" | "updatedAt">; options?: SaveOptions }> = [];

      // Add current skill if dirty
      if (skill?.id && hasUnsavedChanges) {
        const tags = tagsInput
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean);

        updates.push({
          id: skill.id,
          data: {
            name: name.trim(),
            description: description.trim(),
            icon,
            modes: modes.length > 0 ? modes : undefined,
            content: content.trim(),
            tags: tags.length > 0 ? tags : undefined,
            targetToolId: targetToolId.trim() || undefined,
            draft: draft || undefined,
          },
          options: isEditingDefault ? { applyToDefault } : undefined,
        });
      }

      // Add pending changes
      for (const [id, state] of pendingChanges.entries()) {
        if (id === skill?.id) continue; // Skip if already added
        const originalSkill = allSkills?.find((s) => s.id === id);
        const isDefault = originalSkill?.source === "default";
        const tags = state.tagsInput
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean);

        updates.push({
          id,
          data: {
            name: state.name.trim(),
            description: state.description.trim(),
            icon: state.icon,
            modes: state.modes.length > 0 ? state.modes : undefined,
            content: state.content.trim(),
            tags: tags.length > 0 ? tags : undefined,
            targetToolId: state.targetToolId.trim() || undefined,
            draft: state.draft || undefined,
          },
          options: isDefault ? { applyToDefault: state.applyToDefault } : undefined,
        });
      }

      await onSaveAll(updates);

      // Clear pending changes after successful save
      setPendingChanges(new Map());
    } finally {
      setIsSavingAll(false);
    }
  }, [onSaveAll, dirtyCount, skill?.id, hasUnsavedChanges, name, description, icon, modes, content, tagsInput, targetToolId, isEditingDefault, applyToDefault, pendingChanges, allSkills, draft]);

  // Handle close with unsaved changes check (updated for multi-item)
  const handleClose = useCallback(() => {
    // Check both current unsaved changes and pending changes from other items
    if (hasUnsavedChanges || pendingChanges.size > 0) {
      setShowCloseConfirm(true);
    } else {
      onClose();
    }
  }, [hasUnsavedChanges, pendingChanges.size, onClose]);

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

    // Use capture phase with highest priority
    document.addEventListener("keydown", handleEscape, { capture: true });
    return () => {
      document.removeEventListener("keydown", handleEscape, { capture: true });
    };
  }, [open, handleClose]);

  // Initialize form when skill changes
  useEffect(() => {
    if (skill) {
      // Check if there are pending changes for this skill
      const pending = pendingChanges.get(skill.id);
      if (pending) {
        // Restore from pending changes
        setName(pending.name);
        setDescription(pending.description);
        setIcon(pending.icon);
        setModes(pending.modes);
        setContent(pending.content);
        setTagsInput(pending.tagsInput);
        setTargetToolId(pending.targetToolId);
        setApplyToDefault(pending.applyToDefault);
        setDraft(pending.draft);
      } else {
        // Initialize from skill
        setName(skill.name);
        setDescription(skill.description);
        setIcon(skill.icon || "BookOpen");
        setModes(skill.modes || []);
        setContent(skill.content);
        setTagsInput(skill.tags?.join(", ") || "");
        setTargetToolId(skill.targetToolId || "");
        setApplyToDefault(false);
        setDraft(skill.draft || false);
      }
    } else {
      setName("");
      setDescription("");
      setIcon("BookOpen");
      setModes([]);
      setContent("");
      setTagsInput("");
      setTargetToolId("");
      setApplyToDefault(false);
      setDraft(false);
    }
    setErrors({});
    setShowPreview(false);
  }, [skill, open, pendingChanges]);

  // Get mode suggestions at a specific level
  const getSuggestionsAtLevel = useCallback(
    (level: number, parentPath: string[]): string[] => {
      return getSkillModesAtLevel(level, parentPath);
    },
    []
  );

  // Validate form
  const validate = useCallback(() => {
    const newErrors: Record<string, string> = {};

    if (!name.trim()) {
      newErrors.name = "Name is required";
    }
    if (!description.trim()) {
      newErrors.description = "Description is required";
    }
    if (!content.trim()) {
      newErrors.content = "Content is required";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [name, description, content]);

  // Handle save
  const handleSave = useCallback(() => {
    if (readOnly || !onSave) return;
    if (!validate()) return;

    const tags = tagsInput
      .split(",")
      .map((t) => t.trim())
      .filter(Boolean);

    onSave(
      {
        name: name.trim(),
        description: description.trim(),
        icon,
        modes: modes.length > 0 ? modes : undefined,
        content: content.trim(),
        tags: tags.length > 0 ? tags : undefined,
        targetToolId: targetToolId.trim() || undefined,
        draft: draft || undefined,
      },
      isEditingDefault ? { applyToDefault } : undefined
    );
    onClose();
  }, [readOnly, onSave, validate, name, description, icon, modes, content, tagsInput, targetToolId, onClose, isEditingDefault, applyToDefault, draft]);

  if (!open) return null;

  const modalTitle = readOnly
    ? "Skill Preview"
    : isEditing
      ? "Edit Skill"
      : "Create Skill";

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={handleClose}
      />

      {/* Modal */}
      <div className={`relative bg-slate-900 border border-white/10 rounded-xl w-full max-h-[90vh] shadow-xl mx-4 flex flex-col ${
        showSidebar ? "max-w-6xl" : "max-w-2xl md:max-w-5xl"
      }`}>
        {/* Header */}
        <div className="flex-shrink-0 flex items-center justify-between p-4 border-b border-white/10">
          <h2 className="text-lg font-semibold text-white">
            {modalTitle}
          </h2>
          <div className="flex items-center gap-2">
            {readOnly && onEdit && (
              <button
                onClick={onEdit}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors"
                title="Edit skill"
              >
                <Pencil className="h-4 w-4" />
                Edit
              </button>
            )}
            <button
              onClick={handleClose}
              className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Info banner for editing defaults */}
        {isEditingDefault && (
          <div className={`flex-shrink-0 mx-4 mt-4 p-3 rounded-lg flex items-start gap-3 ${
            applyToDefault
              ? "bg-indigo-900/20 border border-indigo-500/30"
              : "bg-amber-900/20 border border-amber-500/30"
          }`}>
            <Info className={`h-5 w-5 flex-shrink-0 mt-0.5 ${
              applyToDefault ? "text-indigo-400" : "text-amber-400"
            }`} />
            <div className="flex-1">
              <p className={`text-sm font-medium ${
                applyToDefault ? "text-indigo-200" : "text-amber-200"
              }`}>
                {applyToDefault
                  ? "Updating default skill"
                  : "Editing a default skill"}
              </p>
              <p className={`text-xs mt-1 ${
                applyToDefault ? "text-indigo-300/70" : "text-amber-300/70"
              }`}>
                {applyToDefault
                  ? "Your changes will modify the default skill directly."
                  : "Your changes will be saved as a custom version. The original default will remain available."}
              </p>
            </div>
            <button
              type="button"
              onClick={() => setApplyToDefault(!applyToDefault)}
              className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
                applyToDefault
                  ? "bg-amber-600/20 text-amber-300 hover:bg-amber-600/30"
                  : "bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30"
              }`}
            >
              {applyToDefault ? "Save as custom" : "Apply to default"}
            </button>
          </div>
        )}

        {/* Content with optional sidebar */}
        <div className="flex-1 min-h-0 overflow-hidden flex">
          {/* Tree Sidebar */}
          {showSidebar && allSkills && (
            <ItemTreeSidebar
              items={itemsForTree}
              selectedItemId={skill?.id ?? null}
              onSelectItem={(id) => {
                const selected = allSkills.find((s) => s.id === id);
                if (selected) handleSelectSkill(selected);
              }}
              dirtyItemIds={dirtyItemIds}
              expandedNodes={expandedTreeNodes}
              onToggleNode={toggleTreeNode}
              renderItemIcon={(item) => {
                const IconComp = getIconComponent(item.icon || "BookOpen");
                return <IconComp className="h-3.5 w-3.5 flex-shrink-0 text-slate-400" />;
              }}
              title="Skills"
              className={isSidebarCollapsed ? "" : "w-60 flex-shrink-0"}
              isCollapsed={isSidebarCollapsed}
              onToggleCollapse={() => setIsSidebarCollapsed((prev) => !prev)}
            />
          )}

          {/* Main Content */}
          <div className="flex-1 min-h-0 p-4 overflow-hidden">
            <div className="h-full md:grid md:grid-cols-[1fr_2fr] md:gap-6 space-y-4 md:space-y-0 overflow-y-auto md:overflow-hidden">
            {/* Left Column - Metadata */}
            <div className="space-y-4 md:h-full md:min-h-0 md:overflow-y-auto md:pr-2">
              {/* Name */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Name {!readOnly && "*"}
                </label>
                <input
                  type="text"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., Screaming Architecture"
                  disabled={readOnly}
                  className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    errors.name ? "border-red-500" : "border-white/10"
                  } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`}
                />
                {errors.name && (
                  <p className="text-xs text-red-400 mt-1">{errors.name}</p>
                )}
              </div>

              {/* Description */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Description {!readOnly && "*"}
                </label>
                <input
                  type="text"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Brief description of what this skill provides"
                  disabled={readOnly}
                  className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    errors.description ? "border-red-500" : "border-white/10"
                  } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`}
                />
                {errors.description && (
                  <p className="text-xs text-red-400 mt-1">{errors.description}</p>
                )}
              </div>

              {/* Icon */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Icon
                </label>
                <IconSelector
                  value={icon}
                  onChange={setIcon}
                  icons={SKILL_ICON_OPTIONS}
                  disabled={readOnly}
                />
              </div>

              {/* Draft Status */}
              {!readOnly && (
                <div className="flex items-center gap-3 p-3 bg-slate-800/50 border border-white/5 rounded-lg">
                  <input
                    type="checkbox"
                    id="draft"
                    checked={draft}
                    onChange={(e) => setDraft(e.target.checked)}
                    className="rounded bg-slate-700 border-white/20 text-orange-500 focus:ring-orange-500"
                  />
                  <div className="flex-1">
                    <label htmlFor="draft" className="text-sm text-slate-300 cursor-pointer flex items-center gap-2">
                      <Construction className="h-4 w-4 text-orange-400" />
                      Mark as draft
                    </label>
                    <p className="text-xs text-slate-500 mt-0.5">
                      Draft skills show a warning that they may not be fully working
                    </p>
                  </div>
                </div>
              )}

              {/* Category Path */}
              <CategoryPathEditor
                value={modes}
                onChange={setModes}
                getSuggestionsAtLevel={getSuggestionsAtLevel}
                label="Category Path"
                placeholder="Select or type category..."
                disabled={readOnly}
              />

              {/* Tags */}
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <Tag className="h-4 w-4 text-slate-400" />
                  <label className="text-sm font-medium text-slate-300">
                    Tags
                  </label>
                </div>
                <input
                  type="text"
                  value={tagsInput}
                  onChange={(e) => setTagsInput(e.target.value)}
                  placeholder="architecture, audit, clean-code, domain-driven"
                  disabled={readOnly}
                  className={`w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    readOnly ? "opacity-70 cursor-not-allowed" : ""
                  }`}
                />
                {!readOnly && (
                  <p className="text-xs text-slate-500 mt-1">
                    Comma-separated tags for search and filtering
                  </p>
                )}
              </div>

              {/* Target Tool ID (optional) */}
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Target Tool ID {!readOnly && "(optional)"}
                </label>
                <input
                  type="text"
                  value={targetToolId}
                  onChange={(e) => setTargetToolId(e.target.value)}
                  placeholder="e.g., spawn_coding_agent"
                  disabled={readOnly}
                  className={`w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    readOnly ? "opacity-70 cursor-not-allowed" : ""
                  }`}
                />
                {!readOnly && (
                  <p className="text-xs text-slate-500 mt-1">
                    If set, this skill will only be sent to the specified tool
                  </p>
                )}
              </div>
            </div>

            {/* Right Column - Content */}
            <div className="flex flex-col md:h-full md:min-h-0 md:overflow-y-auto space-y-4">
              {/* Content */}
              <div className="flex-1 flex flex-col">
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Skill Content {!readOnly && "*"}
                </label>
                <textarea
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="The methodology, knowledge, or expertise content that will be injected into the agent's context..."
                  rows={14}
                  disabled={readOnly}
                  className={`flex-1 w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-sm min-h-[300px] ${
                    errors.content ? "border-red-500" : "border-white/10"
                  } ${readOnly ? "opacity-70 cursor-not-allowed" : ""}`}
                />
                {errors.content && (
                  <p className="text-xs text-red-400 mt-1">{errors.content}</p>
                )}
                {!readOnly && (
                  <p className="text-xs text-slate-500 mt-1">
                    Use Markdown formatting. This content will be injected into the agent's context when the skill is activated.
                  </p>
                )}
              </div>

              {/* Preview toggle */}
              <div>
                <button
                  type="button"
                  onClick={() => setShowPreview(!showPreview)}
                  className="flex items-center gap-2 text-sm text-slate-300 hover:text-white transition-colors"
                >
                  <Eye className="h-4 w-4" />
                  {showPreview ? "Hide Preview" : "Show Preview"}
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${
                      showPreview ? "rotate-180" : ""
                    }`}
                  />
                </button>
                {showPreview && (
                  <div className="mt-2 p-3 bg-slate-800/50 border border-white/5 rounded-lg max-h-64 overflow-y-auto">
                    <pre className="text-sm text-slate-300 whitespace-pre-wrap font-mono">
                      {content || "(no content)"}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        </div>

        {/* Footer */}
        <div className="flex-shrink-0 flex items-center justify-end gap-3 p-4 border-t border-white/10">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
          >
            {readOnly ? "Close" : "Cancel"}
          </button>
          {!readOnly && (
            <>
              {/* Show Save All when multiple items are dirty and onSaveAll is provided */}
              {showSidebar && dirtyCount > 1 && onSaveAll ? (
                <button
                  onClick={handleSaveAll}
                  disabled={isSavingAll}
                  className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isSavingAll && <Loader2 className="h-4 w-4 animate-spin" />}
                  Save All Changes ({dirtyCount})
                </button>
              ) : (
                <button
                  onClick={handleSave}
                  className="px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors"
                >
                  {isEditing ? "Save Changes" : "Create Skill"}
                </button>
              )}
            </>
          )}
        </div>
      </div>

      {/* Unsaved changes confirmation dialog */}
      {showCloseConfirm && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center">
          <div
            className="absolute inset-0 bg-black/60"
            onClick={() => setShowCloseConfirm(false)}
          />
          <div className="relative bg-slate-900 border border-white/10 rounded-xl p-6 max-w-sm mx-4 shadow-xl">
            <div className="flex items-start gap-3 mb-4">
              <AlertTriangle className="h-6 w-6 text-amber-400 flex-shrink-0" />
              <div>
                <h3 className="text-lg font-semibold text-white">Unsaved Changes</h3>
                <p className="text-sm text-slate-400 mt-1">
                  {dirtyCount > 1
                    ? `You have unsaved changes in ${dirtyCount} skills. Are you sure you want to close without saving?`
                    : "You have unsaved changes. Are you sure you want to close without saving?"}
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowCloseConfirm(false)}
                className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
              >
                Keep Editing
              </button>
              <button
                onClick={() => {
                  setShowCloseConfirm(false);
                  setPendingChanges(new Map()); // Clear pending changes
                  onClose();
                }}
                className="px-4 py-2 text-sm font-medium bg-red-600 text-white rounded-lg hover:bg-red-500 transition-colors"
              >
                Discard {dirtyCount > 1 ? "All Changes" : "Changes"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
