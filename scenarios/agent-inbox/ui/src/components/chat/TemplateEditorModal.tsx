/**
 * Modal for creating and editing templates.
 * Provides form for all template fields including variables.
 *
 * Updated to support file-based templates:
 * - All templates are now editable (including defaults)
 * - Editing a default template creates a user override
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { X, Plus, Trash2, ChevronDown, ChevronRight, Eye, Info, Wrench, Pencil, AlertTriangle } from "lucide-react";
import type { Template, TemplateVariable, TemplateSource } from "@/lib/types/templates";
import { fillTemplateContent, getTemplateModesAtLevel } from "@/data/templates";
import { useTools } from "@/hooks/useTools";
import type { EffectiveTool } from "@/lib/api";
import { IconSelector, TEMPLATE_ICON_OPTIONS } from "@/components/shared/IconSelector";
import { CategoryPathEditor } from "@/components/shared/CategoryPathEditor";

interface SaveOptions {
  applyToDefault?: boolean;
}

interface TemplateEditorModalProps {
  open: boolean;
  onClose: () => void;
  template?: Template; // Undefined for create, defined for edit
  templateSource?: TemplateSource; // Source of the template being edited
  defaultModes?: string[]; // Pre-fill modes when creating from Suggestions
  onSave?: (
    template: Omit<Template, "id" | "createdAt" | "updatedAt" | "isBuiltIn">,
    options?: SaveOptions
  ) => void;
  readOnly?: boolean; // If true, modal is in preview mode (no editing)
  onEdit?: () => void; // Callback when Edit button is clicked in readOnly mode
}

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
  readOnly = false,
  onEdit,
}: TemplateEditorModalProps) {
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

  // Tool selection state
  const [selectedToolIds, setSelectedToolIds] = useState<string[]>([]);
  const [expandedScenarios, setExpandedScenarios] = useState<Set<string>>(new Set());

  // Fetch available tools
  const { toolsByScenario, isLoading: isLoadingTools } = useTools();

  // Validation
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Confirmation dialog state
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);

  // Track if there are unsaved changes
  const hasUnsavedChanges = useMemo(() => {
    if (readOnly) return false;
    if (!template) {
      // Creating new - has changes if any field is filled
      return !!(name.trim() || description.trim() || content.trim() || modes.length > 0 || variables.length > 0 || selectedToolIds.length > 0);
    }
    // Editing - compare to original
    return (
      name !== template.name ||
      description !== template.description ||
      icon !== (template.icon || "Sparkles") ||
      JSON.stringify(modes) !== JSON.stringify(template.modes || []) ||
      content !== template.content ||
      JSON.stringify(variables) !== JSON.stringify(template.variables || []) ||
      JSON.stringify(selectedToolIds.sort()) !== JSON.stringify((template.suggestedToolIds || []).sort())
    );
  }, [readOnly, template, name, description, icon, modes, content, variables, selectedToolIds]);

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

  // Normalize tool IDs from old format (just tool name) to new format (scenario:toolName)
  // This handles backwards compatibility with templates saved before the scenario prefix was added
  const normalizeToolIds = useCallback((toolIds: string[], availableTools: Map<string, EffectiveTool[]>): string[] => {
    if (!toolIds || availableTools.size === 0) return toolIds || [];

    return toolIds.map(toolId => {
      // If already in scenario:toolName format, keep as-is
      if (toolId.includes(":")) return toolId;

      // Otherwise, find the scenario that has this tool
      for (const [scenario, tools] of availableTools.entries()) {
        const found = tools.find(t => t.tool.name === toolId);
        if (found) {
          return `${scenario}:${toolId}`;
        }
      }
      // If tool not found in any scenario, keep original (might be removed tool)
      return toolId;
    });
  }, []);

  // Initialize form when template changes
  useEffect(() => {
    if (template) {
      setName(template.name);
      setDescription(template.description);
      setIcon(template.icon || "Sparkles");
      setModes(template.modes || []);
      setContent(template.content);
      setVariables(template.variables || []);
      // Normalize tool IDs to scenario:toolName format
      const normalizedIds = normalizeToolIds(template.suggestedToolIds || [], toolsByScenario);
      setSelectedToolIds(normalizedIds);
    } else {
      setName("");
      setDescription("");
      setIcon("Sparkles");
      setModes(defaultModes || []);
      setContent("");
      setVariables([]);
      setSelectedToolIds([]);
    }
    setErrors({});
    setShowPreview(false);
    setApplyToDefault(false);
    setExpandedScenarios(new Set());
  }, [template, defaultModes, open, toolsByScenario, normalizeToolIds]);

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

  // Get mode suggestions at a specific level
  const getSuggestionsAtLevel = useCallback(
    (level: number, parentPath: string[]): string[] => {
      return getTemplateModesAtLevel(level, parentPath);
    },
    []
  );

  // Tool selection helpers
  const toggleToolSelection = useCallback((toolId: string) => {
    setSelectedToolIds((prev) =>
      prev.includes(toolId) ? prev.filter((id) => id !== toolId) : [...prev, toolId]
    );
  }, []);

  const toggleScenario = useCallback((scenario: string) => {
    setExpandedScenarios((prev) => {
      const next = new Set(prev);
      if (next.has(scenario)) {
        next.delete(scenario);
      } else {
        next.add(scenario);
      }
      return next;
    });
  }, []);

  // Count selected tools per scenario
  const selectedCountByScenario = useMemo(() => {
    const counts = new Map<string, number>();
    for (const toolId of selectedToolIds) {
      const [scenario] = toolId.split(":");
      counts.set(scenario, (counts.get(scenario) || 0) + 1);
    }
    return counts;
  }, [selectedToolIds]);

  // Detect undefined variables in content
  const undefinedVariables = useMemo(() => {
    // Extract all {{variable_name}} patterns from content
    const matches = content.match(/\{\{(\w+)\}\}/g) || [];
    const contentVars = [...new Set(matches.map(m => m.slice(2, -2)))];

    // Get defined variable names
    const definedVars = new Set(variables.map(v => v.name));

    // Find variables used in content but not defined
    return contentVars.filter(v => !definedVars.has(v));
  }, [content, variables]);

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
    if (readOnly || !onSave) return;
    if (!validate()) return;

    onSave(
      {
        name: name.trim(),
        description: description.trim(),
        icon,
        modes,
        content: content.trim(),
        variables,
        suggestedToolIds: selectedToolIds.length > 0 ? selectedToolIds : undefined,
      },
      isEditingDefault ? { applyToDefault } : undefined
    );
    onClose();
  }, [readOnly, onSave, validate, name, description, icon, modes, content, variables, selectedToolIds, onClose, isEditingDefault, applyToDefault]);

  // Generate preview content
  const previewContent = useCallback(() => {
    const values: Record<string, string> = {};
    variables.forEach((v) => {
      values[v.name] = v.placeholder || `[${v.label}]`;
    });
    return fillTemplateContent({ content, variables } as Template, values);
  }, [content, variables]);

  if (!open) return null;

  const modalTitle = readOnly
    ? "Template Preview"
    : isEditing
      ? "Edit Template"
      : "Create Template";

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
                  title="Edit template"
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
                    placeholder="e.g., Debug Performance Issue"
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
                    placeholder="Short description of what this template does"
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
                    icons={TEMPLATE_ICON_OPTIONS}
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
                  required={!readOnly}
                  error={errors.modes}
                />

                {/* Variables */}
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-sm font-medium text-slate-300">
                      Variables
                    </label>
                    {!readOnly && (
                      <button
                        onClick={addVariable}
                        className="flex items-center gap-1 px-2 py-1 text-xs rounded bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors"
                      >
                        <Plus className="h-3 w-3" />
                        Add Variable
                      </button>
                    )}
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

                {/* Suggested Tools */}
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <Wrench className="h-4 w-4 text-slate-400" />
                    <label className="text-sm font-medium text-slate-300">
                      Suggested Tools
                    </label>
                    {selectedToolIds.length > 0 && (
                      <span className="text-xs bg-indigo-600/30 text-indigo-300 px-2 py-0.5 rounded-full">
                        {selectedToolIds.length} selected
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-slate-500 mb-3">
                    These tools will be auto-enabled when this template is selected
                  </p>

                  {isLoadingTools ? (
                    <p className="text-sm text-slate-500 italic">Loading tools...</p>
                  ) : toolsByScenario.size === 0 ? (
                    <p className="text-sm text-slate-500 italic">No tools available</p>
                  ) : (
                    <div className="space-y-2">
                      {Array.from(toolsByScenario.entries()).map(([scenario, tools]) => {
                        const isExpanded = expandedScenarios.has(scenario);
                        const selectedCount = selectedCountByScenario.get(scenario) || 0;

                        return (
                          <div
                            key={scenario}
                            className="bg-slate-800/50 border border-white/5 rounded-lg overflow-hidden"
                          >
                            <button
                              type="button"
                              onClick={() => toggleScenario(scenario)}
                              className="w-full flex items-center justify-between px-3 py-2 hover:bg-white/5 transition-colors"
                            >
                              <div className="flex items-center gap-2">
                                {isExpanded ? (
                                  <ChevronDown className="h-4 w-4 text-slate-400" />
                                ) : (
                                  <ChevronRight className="h-4 w-4 text-slate-400" />
                                )}
                                <span className="text-sm text-slate-300">{scenario}</span>
                              </div>
                              <span className="text-xs text-slate-500">
                                {selectedCount}/{tools.length} selected
                              </span>
                            </button>

                            {isExpanded && (
                              <div className="border-t border-white/5 px-3 py-2 space-y-1.5">
                                {tools.map((tool) => {
                                  const toolId = `${scenario}:${tool.tool.name}`;
                                  const isSelected = selectedToolIds.includes(toolId);

                                  return (
                                    <label
                                      key={tool.tool.name}
                                      className="flex items-start gap-2 cursor-pointer group"
                                    >
                                      <input
                                        type="checkbox"
                                        checked={isSelected}
                                        onChange={() => toggleToolSelection(toolId)}
                                        className="mt-0.5 rounded bg-slate-700 border-white/20 text-indigo-500 focus:ring-indigo-500"
                                      />
                                      <div className="flex-1 min-w-0">
                                        <span className="text-sm text-white group-hover:text-indigo-300 transition-colors">
                                          {tool.tool.name}
                                        </span>
                                        {tool.tool.description && (
                                          <p className="text-xs text-slate-500 truncate">
                                            {tool.tool.description}
                                          </p>
                                        )}
                                      </div>
                                    </label>
                                  );
                                })}
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              </div>

              {/* Right Column - Content */}
              <div className="flex flex-col md:h-full md:min-h-0 md:overflow-y-auto space-y-4">
                {/* Content */}
                <div className="flex-1 flex flex-col">
                  <label className="block text-sm font-medium text-slate-300 mb-1">
                    Template Content {!readOnly && "*"}
                  </label>
                  <textarea
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                    placeholder="Template text with {{variable_name}} placeholders..."
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
                      Use {"{{variable_name}}"} syntax for placeholders
                    </p>
                  )}

                  {/* Undefined variable warning */}
                  {undefinedVariables.length > 0 && (
                    <div className="mt-2 p-2 bg-amber-900/20 border border-amber-500/30 rounded-lg flex items-start gap-2">
                      <AlertTriangle className="h-4 w-4 text-amber-400 flex-shrink-0 mt-0.5" />
                      <p className="text-xs text-amber-300">
                        Undefined variables: {undefinedVariables.map(v => `{{${v}}}`).join(', ')}
                      </p>
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
                {isEditing ? "Save Changes" : "Create Template"}
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
