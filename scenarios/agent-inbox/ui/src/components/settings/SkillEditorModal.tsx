/**
 * Modal for creating and editing skills.
 * Simpler than TemplateEditorModal since skills have no variables.
 *
 * Features:
 * - Edit name, description, icon, modes, content, tags
 * - Apply changes to default or save as custom
 * - Preview content
 */

import { useCallback, useEffect, useState } from "react";
import { X, Eye, ChevronDown, Info, Tag } from "lucide-react";
import type { Skill, SkillSource } from "@/lib/types/templates";

interface SaveOptions {
  applyToDefault?: boolean;
}

interface SkillEditorModalProps {
  open: boolean;
  onClose: () => void;
  skill?: Skill; // Undefined for create, defined for edit
  skillSource?: SkillSource; // Source of the skill being edited
  onSave: (
    skill: Omit<Skill, "id" | "createdAt" | "updatedAt">,
    options?: SaveOptions
  ) => void;
}

const ICON_OPTIONS = [
  "BookOpen",
  "Brain",
  "Building2",
  "Code",
  "Database",
  "FileCode",
  "FlaskConical",
  "Gauge",
  "GraduationCap",
  "Layout",
  "Lightbulb",
  "Package",
  "Puzzle",
  "Route",
  "Search",
  "Server",
  "Shield",
  "Sparkles",
  "Target",
  "Terminal",
  "Wand2",
  "Wrench",
  "Zap",
];

// Skill-focused modes (different from template modes)
const SKILL_MODES = [
  "Architecture",
  "Best Practices",
  "Code Quality",
  "Design Patterns",
  "Documentation",
  "Performance",
  "Security",
  "Testing",
  "Tooling",
] as const;

export function SkillEditorModal({
  open,
  onClose,
  skill,
  skillSource,
  onSave,
}: SkillEditorModalProps) {
  const isEditing = !!skill;
  const isEditingDefault = isEditing && skillSource === "default";

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

  // Add/update mode at a level
  const setModeAtLevel = useCallback((level: number, value: string) => {
    setModes((prev) => {
      const newModes = [...prev];
      if (value) {
        newModes[level] = value;
        // Truncate any modes after this level
        return newModes.slice(0, level + 1);
      } else {
        // Clear this level and all after
        return newModes.slice(0, level);
      }
    });
  }, []);

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
  }, [validate, name, description, icon, modes, content, tagsInput, targetToolId, onSave, onClose, isEditingDefault, applyToDefault]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-slate-900 border border-white/10 rounded-xl w-full max-w-2xl max-h-[90vh] shadow-xl mx-4 flex flex-col">
        {/* Header */}
        <div className="flex-shrink-0 flex items-center justify-between p-4 border-b border-white/10">
          <h2 className="text-lg font-semibold text-white">
            {isEditing ? "Edit Skill" : "Create Skill"}
          </h2>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
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
        <div className="flex-1 min-h-0 p-4 overflow-y-auto space-y-4">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Name *
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Screaming Architecture"
              className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                errors.name ? "border-red-500" : "border-white/10"
              }`}
            />
            {errors.name && (
              <p className="text-xs text-red-400 mt-1">{errors.name}</p>
            )}
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Description *
            </label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Brief description of what this skill provides"
              className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                errors.description ? "border-red-500" : "border-white/10"
              }`}
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
            <select
              value={icon}
              onChange={(e) => setIcon(e.target.value)}
              className="w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              {ICON_OPTIONS.map((opt) => (
                <option key={opt} value={opt}>
                  {opt}
                </option>
              ))}
            </select>
          </div>

          {/* Modes */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Category Path
            </label>
            <div className="flex flex-wrap gap-2">
              {/* Level 0: Top-level mode */}
              <select
                value={modes[0] || ""}
                onChange={(e) => setModeAtLevel(0, e.target.value)}
                className="px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="">Select category...</option>
                {SKILL_MODES.map((mode) => (
                  <option key={mode} value={mode}>
                    {mode}
                  </option>
                ))}
              </select>

              {/* Level 1: Subcategory */}
              {modes[0] && (
                <input
                  type="text"
                  value={modes[1] || ""}
                  onChange={(e) => setModeAtLevel(1, e.target.value)}
                  placeholder="Subcategory (optional)"
                  className="px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              )}

              {/* Level 2: Sub-subcategory */}
              {modes[1] && (
                <input
                  type="text"
                  value={modes[2] || ""}
                  onChange={(e) => setModeAtLevel(2, e.target.value)}
                  placeholder="Sub-subcategory (optional)"
                  className="px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              )}
            </div>
            <p className="text-xs text-slate-500 mt-1">
              Path: {modes.length > 0 ? modes.join(" / ") : "(none)"}
            </p>
          </div>

          {/* Content */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Skill Content *
            </label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder="The methodology, knowledge, or expertise content that will be injected into the agent's context..."
              rows={10}
              className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-sm ${
                errors.content ? "border-red-500" : "border-white/10"
              }`}
            />
            {errors.content && (
              <p className="text-xs text-red-400 mt-1">{errors.content}</p>
            )}
            <p className="text-xs text-slate-500 mt-1">
              Use Markdown formatting. This content will be injected into the agent's context when the skill is activated.
            </p>
          </div>

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
              className="w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <p className="text-xs text-slate-500 mt-1">
              Comma-separated tags for search and filtering
            </p>
          </div>

          {/* Target Tool ID (optional) */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Target Tool ID (optional)
            </label>
            <input
              type="text"
              value={targetToolId}
              onChange={(e) => setTargetToolId(e.target.value)}
              placeholder="e.g., spawn_coding_agent"
              className="w-full px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            />
            <p className="text-xs text-slate-500 mt-1">
              If set, this skill will only be sent to the specified tool
            </p>
          </div>

          {/* Preview toggle */}
          <div>
            <button
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

        {/* Footer */}
        <div className="flex-shrink-0 flex items-center justify-end gap-3 p-4 border-t border-white/10">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="px-4 py-2 text-sm font-medium bg-indigo-600 text-white rounded-lg hover:bg-indigo-500 transition-colors"
          >
            {isEditing ? "Save Changes" : "Create Skill"}
          </button>
        </div>
      </div>
    </div>
  );
}
