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

import { useCallback, useEffect, useMemo, useState } from "react";
import { X, Eye, ChevronDown, Info, Tag, Pencil, AlertTriangle } from "lucide-react";
import type { Skill, SkillSource } from "@/lib/types/templates";
import { IconSelector, SKILL_ICON_OPTIONS } from "@/components/shared/IconSelector";
import { CategoryPathEditor } from "@/components/shared/CategoryPathEditor";
import { getSkillModesAtLevel } from "@/data/skills";

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
}

export function SkillEditorModal({
  open,
  onClose,
  skill,
  skillSource,
  onSave,
  readOnly = false,
  onEdit,
}: SkillEditorModalProps) {
  const isEditing = !!skill;
  const isEditingDefault = isEditing && skillSource === "default" && !readOnly;

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

  // Validation
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Confirmation dialog state
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  // Track if there are unsaved changes
  const hasUnsavedChanges = useMemo(() => {
    if (readOnly) return false;
    if (!skill) {
      // Creating new - has changes if any field is filled
      return !!(name.trim() || description.trim() || content.trim() || tagsInput.trim() || targetToolId.trim() || modes.length > 0);
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
      targetToolId !== (skill.targetToolId || "")
    );
  }, [readOnly, skill, name, description, icon, modes, content, tagsInput, targetToolId]);

  // Handle close with unsaved changes check
  const handleClose = useCallback(() => {
    if (hasUnsavedChanges) {
      setShowCloseConfirm(true);
    } else {
      onClose();
    }
  }, [hasUnsavedChanges, onClose]);

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
      setName(skill.name);
      setDescription(skill.description);
      setIcon(skill.icon || "BookOpen");
      setModes(skill.modes || []);
      setContent(skill.content);
      setTagsInput(skill.tags?.join(", ") || "");
      setTargetToolId(skill.targetToolId || "");
    } else {
      setName("");
      setDescription("");
      setIcon("BookOpen");
      setModes([]);
      setContent("");
      setTagsInput("");
      setTargetToolId("");
    }
    setErrors({});
    setShowPreview(false);
    setApplyToDefault(false);
  }, [skill, open]);

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
      },
      isEditingDefault ? { applyToDefault } : undefined
    );
    onClose();
  }, [readOnly, onSave, validate, name, description, icon, modes, content, tagsInput, targetToolId, onClose, isEditingDefault, applyToDefault]);

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
      <div className="relative bg-slate-900 border border-white/10 rounded-xl w-full max-w-2xl md:max-w-5xl max-h-[90vh] shadow-xl mx-4 flex flex-col">
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

        {/* Content */}
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

        {/* Footer */}
        <div className="flex-shrink-0 flex items-center justify-end gap-3 p-4 border-t border-white/10">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
          >
            {readOnly ? "Close" : "Cancel"}
          </button>
          {!readOnly && (
            <button
              onClick={handleSave}
              className="px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors"
            >
              {isEditing ? "Save Changes" : "Create Skill"}
            </button>
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
                  You have unsaved changes. Are you sure you want to close without saving?
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
                  onClose();
                }}
                className="px-4 py-2 text-sm font-medium bg-red-600 text-white rounded-lg hover:bg-red-500 transition-colors"
              >
                Discard Changes
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
