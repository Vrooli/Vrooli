/**
 * Settings tab for managing templates and Suggestions preferences.
 *
 * Updated to work with file-based template storage:
 * - All templates are now editable (including defaults)
 * - Source indicators show where templates come from (default, user, modified)
 * - Modified defaults can be reset to their original state
 */

import { useState, useCallback } from "react";
import {
  Lightbulb,
  Pencil,
  Trash2,
  RotateCcw,
  Search,
  FileText,
  User,
  RefreshCw,
} from "lucide-react";
import type { ModeHistoryEntry, TemplateWithSource } from "@/lib/types/templates";
import { ModelSelector } from "./ModelSelector";
import type { Model } from "@/lib/api";

interface TemplatesSettingsTabProps {
  suggestionsVisible: boolean;
  onToggleSuggestions: (visible: boolean) => void;
  mergeModel: string;
  onMergeModelChange: (modelId: string) => void;
  templates: TemplateWithSource[];
  onEditTemplate: (template: TemplateWithSource) => void;
  onDeleteTemplate: (templateId: string) => Promise<void>;
  onResetTemplate: (templateId: string) => Promise<void>;
  modeHistory: ModeHistoryEntry[];
  onClearHistory: () => void;
  models: Model[];
  isLoading?: boolean;
}

function getSourceBadge(source: TemplateWithSource["source"]) {
  switch (source) {
    case "default":
      return {
        label: "Default",
        className: "bg-slate-700 text-slate-400",
        icon: FileText,
      };
    case "user":
      return {
        label: "Custom",
        className: "bg-indigo-900/50 text-indigo-400",
        icon: User,
      };
    case "modified":
      return {
        label: "Modified",
        className: "bg-amber-900/50 text-amber-400",
        icon: RefreshCw,
      };
  }
}

export function TemplatesSettingsTab({
  suggestionsVisible,
  onToggleSuggestions,
  mergeModel,
  onMergeModelChange,
  templates,
  onEditTemplate,
  onDeleteTemplate,
  onResetTemplate,
  modeHistory,
  onClearHistory,
  models,
  isLoading,
}: TemplatesSettingsTabProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [deletingId, setDeletingId] = useState<string | null>(null);
  const [resettingId, setResettingId] = useState<string | null>(null);

  // Filter templates
  const filteredTemplates = templates.filter((t) => {
    const matchesSearch =
      !searchQuery ||
      t.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      t.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesSearch;
  });

  // Group templates by source for better organization
  const defaultTemplates = filteredTemplates.filter((t) => t.source === "default");
  const modifiedTemplates = filteredTemplates.filter((t) => t.source === "modified");
  const userTemplates = filteredTemplates.filter((t) => t.source === "user");

  const handleDelete = useCallback(
    async (templateId: string) => {
      setDeletingId(templateId);
      try {
        await onDeleteTemplate(templateId);
      } finally {
        setDeletingId(null);
      }
    },
    [onDeleteTemplate]
  );

  const handleReset = useCallback(
    async (templateId: string) => {
      setResettingId(templateId);
      try {
        await onResetTemplate(templateId);
      } finally {
        setResettingId(null);
      }
    },
    [onResetTemplate]
  );

  const renderTemplateItem = (template: TemplateWithSource) => {
    const badge = getSourceBadge(template.source);
    const BadgeIcon = badge.icon;
    const isDeleting = deletingId === template.id;
    const isResetting = resettingId === template.id;
    const isOperating = isDeleting || isResetting;

    return (
      <div
        key={template.id}
        className={`flex items-center justify-between p-2 bg-slate-800/50 border border-white/5 rounded-lg ${
          isOperating ? "opacity-50" : ""
        }`}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm text-white truncate">{template.name}</p>
            <span
              className={`flex items-center gap-1 px-1.5 py-0.5 text-[10px] rounded ${badge.className}`}
            >
              <BadgeIcon className="h-2.5 w-2.5" />
              {badge.label}
            </span>
          </div>
          <p className="text-xs text-slate-500 truncate">
            {template.modes?.join(" → ") || "No mode"}
          </p>
        </div>
        <div className="flex items-center gap-1 ml-2">
          {/* Edit button - available for ALL templates */}
          <button
            onClick={() => onEditTemplate(template)}
            disabled={isOperating}
            className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors disabled:opacity-50"
            title="Edit template"
          >
            <Pencil className="h-4 w-4" />
          </button>

          {/* Reset button - only for modified templates */}
          {template.source === "modified" && (
            <button
              onClick={() => handleReset(template.id)}
              disabled={isOperating}
              className="p-1.5 rounded hover:bg-white/10 text-amber-400 hover:text-amber-300 transition-colors disabled:opacity-50"
              title="Reset to default"
            >
              <RotateCcw className={`h-4 w-4 ${isResetting ? "animate-spin" : ""}`} />
            </button>
          )}

          {/* Delete button - only for user and modified templates */}
          {(template.source === "user" || template.source === "modified") && (
            <button
              onClick={() => handleDelete(template.id)}
              disabled={isOperating}
              className="p-1.5 rounded hover:bg-white/10 text-slate-400 hover:text-red-400 transition-colors disabled:opacity-50"
              title={template.source === "modified" ? "Remove customization" : "Delete template"}
            >
              <Trash2 className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>
    );
  };

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
            Templates ({templates.length})
            {isLoading && (
              <span className="ml-2 text-xs text-slate-500">Loading...</span>
            )}
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

        {/* Legend */}
        <div className="flex items-center gap-4 mb-3 text-xs text-slate-500">
          <span className="flex items-center gap-1">
            <FileText className="h-3 w-3" />
            Default
          </span>
          <span className="flex items-center gap-1 text-amber-400">
            <RefreshCw className="h-3 w-3" />
            Modified
          </span>
          <span className="flex items-center gap-1 text-indigo-400">
            <User className="h-3 w-3" />
            Custom
          </span>
        </div>

        {/* Templates */}
        <div className="space-y-2 max-h-80 overflow-y-auto">
          {/* User templates first */}
          {userTemplates.length > 0 && (
            <div className="space-y-2">
              <p className="text-[10px] uppercase tracking-wider text-slate-600 font-medium">
                Custom Templates
              </p>
              {userTemplates.map(renderTemplateItem)}
            </div>
          )}

          {/* Modified templates */}
          {modifiedTemplates.length > 0 && (
            <div className="space-y-2">
              <p className="text-[10px] uppercase tracking-wider text-slate-600 font-medium mt-3">
                Modified Templates
              </p>
              {modifiedTemplates.map(renderTemplateItem)}
            </div>
          )}

          {/* Default templates */}
          {defaultTemplates.length > 0 && (
            <div className="space-y-2">
              <p className="text-[10px] uppercase tracking-wider text-slate-600 font-medium mt-3">
                Default Templates
              </p>
              {defaultTemplates.map(renderTemplateItem)}
            </div>
          )}

          {filteredTemplates.length === 0 && (
            <p className="text-sm text-slate-500 text-center py-4">
              {isLoading ? "Loading templates..." : "No templates found"}
            </p>
          )}
        </div>
      </section>

      {/* Info */}
      <section className="text-xs text-slate-500 space-y-1">
        <p>
          <strong className="text-slate-400">Tip:</strong> All templates are editable.
          Editing a default template creates your own customized version.
        </p>
        <p>
          Use the <RotateCcw className="h-3 w-3 inline text-amber-400" /> button to
          reset a modified template back to its original default.
        </p>
      </section>
    </div>
  );
}
