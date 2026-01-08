/**
 * Settings tab for managing templates and Suggestions preferences.
 */

import { useState, useCallback } from "react";
import {
  Lightbulb,
  Eye,
  EyeOff,
  Pencil,
  Trash2,
  RotateCcw,
  Search,
} from "lucide-react";
import type { Template, ModeHistoryEntry } from "@/lib/types/templates";
import { BUILT_IN_TEMPLATES } from "@/data/builtInTemplates";
import { getHiddenBuiltInIds } from "@/data/templates";
import { ModelSelector } from "./ModelSelector";
import type { Model } from "@/lib/api";

interface TemplatesSettingsTabProps {
  suggestionsVisible: boolean;
  onToggleSuggestions: (visible: boolean) => void;
  mergeModel: string;
  onMergeModelChange: (modelId: string) => void;
  templates: Template[];
  onEditTemplate: (template: Template) => void;
  onDeleteTemplate: (templateId: string) => void;
  onHideTemplate: (templateId: string) => void;
  onUnhideTemplate: (templateId: string) => void;
  modeHistory: ModeHistoryEntry[];
  onClearHistory: () => void;
  models: Model[];
}

export function TemplatesSettingsTab({
  suggestionsVisible,
  onToggleSuggestions,
  mergeModel,
  onMergeModelChange,
  templates,
  onEditTemplate,
  onDeleteTemplate,
  onHideTemplate,
  onUnhideTemplate,
  modeHistory,
  onClearHistory,
  models,
}: TemplatesSettingsTabProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [showHidden, setShowHidden] = useState(false);

  // Get hidden built-in IDs
  const hiddenIds = getHiddenBuiltInIds();

  // Filter templates
  const filteredTemplates = templates.filter((t) => {
    const matchesSearch =
      !searchQuery ||
      t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  // Get hidden built-in templates
  const hiddenBuiltIns = BUILT_IN_TEMPLATES.filter((t) =>
    hiddenIds.includes(t.id)
  );

  const handleUnhide = useCallback(
    (templateId: string) => {
      onUnhideTemplate(templateId);
    },
    [onUnhideTemplate]
  );

  return (
    <div className="space-y-6">
      {/* Suggestions Visibility */}
      <section>
        <h3 className="text-sm font-medium text-slate-300 mb-3">
          Suggestions Panel
        </h3>
        <div className="flex items-center justify-between p-3 bg-white/5 border border-white/10 rounded-lg">
          <div className="flex items-center gap-3">
            <Lightbulb className="h-5 w-5 text-indigo-400" />
            <div>
              <p className="text-sm text-white">Show Suggestions</p>
              <p className="text-xs text-slate-500">
                Display template suggestions above message input
              </p>
            </div>
          </div>
          <button
            onClick={() => onToggleSuggestions(!suggestionsVisible)}
            className={`relative w-11 h-6 rounded-full transition-colors ${
              suggestionsVisible ? "bg-indigo-600" : "bg-slate-600"
            }`}
          >
            <span
              className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform ${
                suggestionsVisible ? "translate-x-5" : ""
              }`}
            />
          </button>
        </div>
        <p className="text-xs text-slate-500 mt-2">
          You can also toggle with <kbd className="px-1 bg-slate-700 rounded">/suggestions</kbd>
        </p>
      </section>

      {/* AI Merge Model */}
      <section>
        <h3 className="text-sm font-medium text-slate-300 mb-3">
          AI Merge Model
        </h3>
        <p className="text-xs text-slate-500 mb-3">
          Model used when merging your message with a template
        </p>
        <ModelSelector
          models={models}
          selectedModel={mergeModel}
          onSelectModel={onMergeModelChange}
          label="Merge model"
          compact
        />
      </section>

      {/* Templates List */}
      <section>
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-medium text-slate-300">
            Templates ({filteredTemplates.length})
          </h3>
          {modeHistory.length > 0 && (
            <button
              onClick={onClearHistory}
              className="flex items-center gap-1 text-xs text-slate-400 hover:text-white transition-colors"
            >
              <RotateCcw className="h-3 w-3" />
              Clear history
            </button>
          )}
        </div>

        {/* Search */}
        <div className="relative mb-3">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search templates..."
            className="w-full pl-9 pr-3 py-2 bg-slate-800 border border-white/10 rounded-lg text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>

        {/* Templates */}
        <div className="space-y-2 max-h-64 overflow-y-auto">
          {filteredTemplates.map((template) => (
            <div
              key={template.id}
              className="flex items-center justify-between p-2 bg-slate-800/50 border border-white/5 rounded-lg"
            >
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <p className="text-sm text-white truncate">{template.name}</p>
                  {template.isBuiltIn && (
                    <span className="px-1.5 py-0.5 text-[10px] bg-slate-700 text-slate-400 rounded">
                      Built-in
                    </span>
                  )}
                </div>
                <p className="text-xs text-slate-500 truncate">
                  {template.modes?.join(" → ") || "No mode"}
                </p>
              </div>
              <div className="flex items-center gap-1 ml-2">
                {template.isBuiltIn ? (
                  <button
                    onClick={() => onHideTemplate(template.id)}
                    className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
                    title="Hide template"
                  >
                    <EyeOff className="h-4 w-4" />
                  </button>
                ) : (
                  <>
                    <button
                      onClick={() => onEditTemplate(template)}
                      className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
                      title="Edit template"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => onDeleteTemplate(template.id)}
                      className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors"
                      title="Delete template"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </>
                )}
              </div>
            </div>
          ))}

          {filteredTemplates.length === 0 && (
            <p className="text-sm text-slate-500 text-center py-4">
              No templates found
            </p>
          )}
        </div>

        {/* Hidden templates */}
        {hiddenBuiltIns.length > 0 && (
          <div className="mt-4">
            <button
              onClick={() => setShowHidden(!showHidden)}
              className="flex items-center gap-2 text-xs text-slate-400 hover:text-white transition-colors"
            >
              {showHidden ? (
                <EyeOff className="h-3 w-3" />
              ) : (
                <Eye className="h-3 w-3" />
              )}
              {showHidden ? "Hide" : "Show"} hidden templates (
              {hiddenBuiltIns.length})
            </button>

            {showHidden && (
              <div className="mt-2 space-y-2">
                {hiddenBuiltIns.map((template) => (
                  <div
                    key={template.id}
                    className="flex items-center justify-between p-2 bg-slate-800/30 border border-white/5 rounded-lg opacity-60"
                  >
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-slate-400 truncate">
                        {template.name}
                      </p>
                      <p className="text-xs text-slate-600 truncate">
                        {template.modes?.join(" → ") || "No mode"}
                      </p>
                    </div>
                    <button
                      onClick={() => handleUnhide(template.id)}
                      className="p-1.5 rounded hover:bg-white/10 text-slate-500 hover:text-white transition-colors"
                      title="Unhide template"
                    >
                      <Eye className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </section>
    </div>
  );
}
