/**
 * Modal for creating and editing templates.
 * Provides form for all template fields including variables.
 *
 * Updated to support file-based templates:
 * - All templates are now editable (including defaults)
 * - Editing a default template creates a user override
 */

import { useCallback, useEffect, useState } from "react";
import { X, Plus, Trash2, ChevronDown, Eye, Info } from "lucide-react";
import type { Template, TemplateVariable, TemplateSource } from "@/lib/types/templates";
import { SUGGESTION_MODES } from "@/lib/types/templates";
import { fillTemplateContent } from "@/data/templates";

interface SaveOptions {
  applyToDefault?: boolean;
}

interface TemplateEditorModalProps {
  open: boolean;
  onClose: () => void;
  template?: Template; // Undefined for create, defined for edit
  templateSource?: TemplateSource; // Source of the template being edited
  defaultModes?: string[]; // Pre-fill modes when creating from Suggestions
  onSave: (
    template: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
    options?: SaveOptions
  ) => void;
}

const ICON_OPTIONS = [
  "Sparkles",
  "Bug",
  "Search",
  "RefreshCw",
  "FlaskConical",
  "GraduationCap",
  "Eye",
  "FolderTree",
  "Package",
  "Lightbulb",
  "Gauge",
  "FileType",
  "Server",
  "Layout",
  "Wand2",
  "Building2",
  "ArrowRightLeft",
  "Puzzle",
  "Route",
  "MessageCircleQuestion",
  "Shield",
  "Accessibility",
  "FileCode",
  "Terminal",
  "Database",
  "Cloud",
  "Zap",
];

const VARIABLE_TYPES: TemplateVariable["type"][] = [
  "text",
  "textarea",
  "select",
];

export function TemplateEditorModal({
  open,
  onClose,
  template,
  templateSource,
  defaultModes,
  onSave,
}: TemplateEditorModalProps) {
  const isEditing = !!template;
  const isEditingDefault = isEditing && templateSource === "default";

  // Form state
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [icon, setIcon] = useState("Sparkles");
  const [modes, setModes] = useState<string[]>([]);
  const [content, setContent] = useState("");
  const [variables, setVariables] = useState<TemplateVariable[]>([]);
  const [showPreview, setShowPreview] = useState(false);
  const [applyToDefault, setApplyToDefault] = useState(false);

  // Validation
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Initialize form when template changes
  useEffect(() => {
    if (template) {
      setName(template.name);
      setDescription(template.description);
      setIcon(template.icon || "Sparkles");
      setModes(template.modes || []);
      setContent(template.content);
      setVariables(template.variables || []);
    } else {
      setName("");
      setDescription("");
      setIcon("Sparkles");
      setModes(defaultModes || []);
      setContent("");
      setVariables([]);
    }
    setErrors({});
    setShowPreview(false);
    setApplyToDefault(false);
  }, [template, defaultModes, open]);

  // Add a new variable
  const addVariable = useCallback(() => {
    setVariables((prev) => [
      ...prev,
      {
        name: `variable_${prev.length + 1}`,
        label: `Variable ${prev.length + 1}`,
        type: "text",
        required: false,
      },
    ]);
  }, []);

  // Update a variable
  const updateVariable = useCallback(
    (index: number, updates: Partial<TemplateVariable>) => {
      setVariables((prev) =>
        prev.map((v, i) => (i === index ? { ...v, ...updates } : v))
      );
    },
    []
  );

  // Remove a variable
  const removeVariable = useCallback((index: number) => {
    setVariables((prev) => prev.filter((_, i) => i !== index));
  }, []);

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
    if (modes.length === 0) {
      newErrors.modes = "At least one mode is required";
    }

    // Validate variables
    variables.forEach((v, i) => {
      if (!v.name.trim()) {
        newErrors[`variable_${i}_name`] = "Variable name is required";
      }
      if (!v.label.trim()) {
        newErrors[`variable_${i}_label`] = "Variable label is required";
      }
      if (v.type === "select" && (!v.options || v.options.length === 0)) {
        newErrors[`variable_${i}_options`] =
          "Select type requires at least one option";
      }
    });

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [name, description, content, modes, variables]);

  // Handle save
  const handleSave = useCallback(() => {
    if (!validate()) return;

    onSave(
      {
        name: name.trim(),
        description: description.trim(),
        icon,
        modes,
        content: content.trim(),
        variables,
      },
      isEditingDefault ? { applyToDefault } : undefined
    );
    onClose();
  }, [validate, name, description, icon, modes, content, variables, onSave, onClose, isEditingDefault, applyToDefault]);

  // Generate preview content
  const previewContent = useCallback(() => {
    const values: Record<string, string> = {};
    variables.forEach((v) => {
      values[v.name] = v.placeholder || `[${v.label}]`;
    });
    return fillTemplateContent({ content, variables } as Template, values);
  }, [content, variables]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-slate-900 border border-white/10 rounded-xl w-full max-w-2xl max-h-[90vh] overflow-hidden shadow-xl mx-4">
          {/* Header */}
          <div className="flex items-center justify-between p-4 border-b border-white/10">
            <h2 className="text-lg font-semibold text-white">
              {isEditing ? "Edit Template" : "Create Template"}
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
            <div className={`mx-4 mt-4 p-3 rounded-lg flex items-start gap-3 ${
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
                    ? "Updating default template"
                    : "Editing a default template"}
                </p>
                <p className={`text-xs mt-1 ${
                  applyToDefault ? "text-indigo-300/70" : "text-amber-300/70"
                }`}>
                  {applyToDefault
                    ? "Your changes will modify the default template directly. This affects all users."
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
          <div className="p-4 overflow-y-auto max-h-[calc(90vh-130px)] space-y-4">
            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Name
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., Debug Performance Issue"
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
                Description
              </label>
              <input
                type="text"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Short description of what this template does"
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
                Mode Path
              </label>
              <div className="flex flex-wrap gap-2">
                {/* Level 0: Top-level mode */}
                <select
                  value={modes[0] || ""}
                  onChange={(e) => setModeAtLevel(0, e.target.value)}
                  className={`px-3 py-2 bg-slate-800 border rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 ${
                    errors.modes ? "border-red-500" : "border-white/10"
                  }`}
                >
                  <option value="">Select mode...</option>
                  {SUGGESTION_MODES.map((mode) => (
                    <option key={mode} value={mode}>
                      {mode}
                    </option>
                  ))}
                </select>

                {/* Level 1+: Subcategories */}
                {modes[0] && (
                  <input
                    type="text"
                    value={modes[1] || ""}
                    onChange={(e) => setModeAtLevel(1, e.target.value)}
                    placeholder="Subcategory (optional)"
                    className="px-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                )}

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
              {errors.modes && (
                <p className="text-xs text-red-400 mt-1">{errors.modes}</p>
              )}
              <p className="text-xs text-slate-500 mt-1">
                Path: {modes.length > 0 ? modes.join(" → ") : "(none)"}
              </p>
            </div>

            {/* Content */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Template Content
              </label>
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Template text with {{variable_name}} placeholders..."
                rows={6}
                className={`w-full px-3 py-2 bg-slate-800 border rounded-lg text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-sm ${
                  errors.content ? "border-red-500" : "border-white/10"
                }`}
              />
              {errors.content && (
                <p className="text-xs text-red-400 mt-1">{errors.content}</p>
              )}
              <p className="text-xs text-slate-500 mt-1">
                Use {"{{variable_name}}"} syntax for placeholders
              </p>
            </div>

            {/* Variables */}
            <div>
              <div className="flex items-center justify-between mb-2">
                <label className="text-sm font-medium text-slate-300">
                  Variables
                </label>
                <button
                  onClick={addVariable}
                  className="flex items-center gap-1 px-2 py-1 text-xs rounded bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors"
                >
                  <Plus className="h-3 w-3" />
                  Add Variable
                </button>
              </div>

              {variables.length === 0 ? (
                <p className="text-sm text-slate-500 italic">
                  No variables defined yet.
                </p>
              ) : (
                <div className="space-y-3">
                  {variables.map((variable, index) => (
                    <div
                      key={index}
                      className="p-3 bg-slate-800/50 border border-white/5 rounded-lg"
                    >
                      <div className="flex items-start gap-2">
                        <div className="flex-1 grid grid-cols-2 gap-2">
                          <div>
                            <input
                              type="text"
                              value={variable.name}
                              onChange={(e) =>
                                updateVariable(index, {
                                  name: e.target.value.replace(/\s+/g, "_"),
                                })
                              }
                              placeholder="variable_name"
                              className={`w-full px-2 py-1 text-sm bg-slate-700 border rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 ${
                                errors[`variable_${index}_name`]
                                  ? "border-red-500"
                                  : "border-white/10"
                              }`}
                            />
                            <p className="text-xs text-slate-500 mt-0.5">
                              Name (no spaces)
                            </p>
                          </div>
                          <div>
                            <input
                              type="text"
                              value={variable.label}
                              onChange={(e) =>
                                updateVariable(index, { label: e.target.value })
                              }
                              placeholder="Display Label"
                              className={`w-full px-2 py-1 text-sm bg-slate-700 border rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 ${
                                errors[`variable_${index}_label`]
                                  ? "border-red-500"
                                  : "border-white/10"
                              }`}
                            />
                            <p className="text-xs text-slate-500 mt-0.5">
                              Label
                            </p>
                          </div>
                          <div>
                            <select
                              value={variable.type}
                              onChange={(e) =>
                                updateVariable(index, {
                                  type: e.target.value as TemplateVariable["type"],
                                })
                              }
                              className="w-full px-2 py-1 text-sm bg-slate-700 border border-white/10 rounded text-white focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            >
                              {VARIABLE_TYPES.map((t) => (
                                <option key={t} value={t}>
                                  {t}
                                </option>
                              ))}
                            </select>
                            <p className="text-xs text-slate-500 mt-0.5">
                              Type
                            </p>
                          </div>
                          <div>
                            <input
                              type="text"
                              value={variable.placeholder || ""}
                              onChange={(e) =>
                                updateVariable(index, {
                                  placeholder: e.target.value,
                                })
                              }
                              placeholder="Placeholder text..."
                              className="w-full px-2 py-1 text-sm bg-slate-700 border border-white/10 rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                            />
                            <p className="text-xs text-slate-500 mt-0.5">
                              Placeholder
                            </p>
                          </div>
                        </div>
                        <button
                          onClick={() => removeVariable(index)}
                          className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors"
                        >
                          <Trash2 className="h-4 w-4" />
                        </button>
                      </div>

                      {/* Options for select type */}
                      {variable.type === "select" && (
                        <div className="mt-2">
                          <input
                            type="text"
                            value={variable.options?.join(", ") || ""}
                            onChange={(e) =>
                              updateVariable(index, {
                                options: e.target.value
                                  .split(",")
                                  .map((o) => o.trim())
                                  .filter(Boolean),
                              })
                            }
                            placeholder="Option 1, Option 2, Option 3"
                            className={`w-full px-2 py-1 text-sm bg-slate-700 border rounded text-white placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 ${
                              errors[`variable_${index}_options`]
                                ? "border-red-500"
                                : "border-white/10"
                            }`}
                          />
                          <p className="text-xs text-slate-500 mt-0.5">
                            Options (comma-separated)
                          </p>
                        </div>
                      )}

                      {/* Required checkbox */}
                      <div className="mt-2 flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={`required_${index}`}
                          checked={variable.required || false}
                          onChange={(e) =>
                            updateVariable(index, { required: e.target.checked })
                          }
                          className="rounded bg-slate-700 border-white/20 text-indigo-500 focus:ring-indigo-500"
                        />
                        <label
                          htmlFor={`required_${index}`}
                          className="text-xs text-slate-400"
                        >
                          Required
                        </label>
                      </div>
                    </div>
                  ))}
                </div>
              )}
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
                <div className="mt-2 p-3 bg-slate-800/50 border border-white/5 rounded-lg">
                  <pre className="text-sm text-slate-300 whitespace-pre-wrap font-mono">
                    {previewContent()}
                  </pre>
                </div>
              )}
            </div>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-end gap-3 p-4 border-t border-white/10">
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
              {isEditing ? "Save Changes" : "Create Template"}
            </button>
          </div>
      </div>
    </div>
  );
}
