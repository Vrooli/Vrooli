/**
 * Suggestions panel for quick template access.
 * Shows mode chips at top level, navigates into submodes and templates.
 */

import { useCallback, useMemo } from "react";
import { ChevronRight, ChevronLeft, Plus, Pencil, Trash2, RotateCcw, Lightbulb } from "lucide-react";
import type { TemplateWithSource, ModeHistoryEntry } from "@/lib/types/templates";
import { SUGGESTION_MODES } from "@/lib/types/templates";
import { sortByFrecency, serializeModePath } from "@/lib/frecency";
import { getIconComponent, MODE_ICONS } from "./SuggestionsIcons";

interface SuggestionsProps {
  embedded?: boolean;
  templates: TemplateWithSource[];
  currentModePath: string[];
  modeHistory: ModeHistoryEntry[];
  onSelectTemplate: (template: TemplateWithSource) => void;
  onNavigateToMode: (mode: string) => void;
  onNavigateBack: () => void;
  onResetPath: () => void;
  onEditTemplate?: (template: TemplateWithSource) => void;
  onDeleteTemplate?: (templateId: string) => void;
  onResetTemplate?: (templateId: string) => void;
  onCreateTemplate?: (parentModes: string[]) => void;
  onRecordModeUsage?: (path: string[]) => void;
}

export function Suggestions({
  templates, currentModePath, modeHistory, onSelectTemplate,
  onNavigateToMode, onNavigateBack, onResetPath, onEditTemplate,
  onDeleteTemplate, onResetTemplate, onCreateTemplate, onRecordModeUsage,
  embedded = false,
}: SuggestionsProps) {
  const { submodes, levelTemplates } = useMemo(() => {
    const templatesList = templates;

    if (currentModePath.length === 0) {
      const availableModes = new Set<string>();
      templatesList.forEach((t) => { const firstMode = t.modes?.[0]; if (firstMode) availableModes.add(firstMode); });
      const orderedModes = SUGGESTION_MODES.filter((m) => availableModes.has(m));
      return { submodes: orderedModes as string[], levelTemplates: [] as TemplateWithSource[] };
    }

    const submodesSet = new Set<string>();
    const pathTemplates: TemplateWithSource[] = [];

    templatesList.forEach((t) => {
      if (!t.modes || t.modes.length === 0) return;
      if (t.modes.length < currentModePath.length) return;
      for (let i = 0; i < currentModePath.length; i++) { if (t.modes[i] !== currentModePath[i]) return; }
      if (t.modes.length === currentModePath.length) pathTemplates.push(t);
      else { const nextMode = t.modes[currentModePath.length]; if (nextMode) submodesSet.add(nextMode); }
    });

    return { submodes: Array.from(submodesSet).sort(), levelTemplates: pathTemplates };
  }, [templates, currentModePath]);

  const sortedSubmodes = useMemo(() => {
    const submodesWithPath = submodes.map((mode) => ({ mode, path: serializeModePath([...currentModePath, mode]) }));
    return sortByFrecency(submodesWithPath, modeHistory).map((s) => s.mode);
  }, [submodes, currentModePath, modeHistory]);

  const sortedTemplates = useMemo(() => {
    const templatesWithPath = levelTemplates.map((t) => ({ ...t, path: serializeModePath(t.modes || []) }));
    return sortByFrecency(templatesWithPath, modeHistory);
  }, [levelTemplates, modeHistory]);

  const handleModeClick = useCallback((mode: string) => {
    onNavigateToMode(mode);
    onRecordModeUsage?.([...currentModePath, mode]);
  }, [onNavigateToMode, currentModePath, onRecordModeUsage]);

  const handleTemplateClick = useCallback((template: TemplateWithSource) => {
    onSelectTemplate(template);
    if (template.modes) onRecordModeUsage?.(template.modes);
  }, [onSelectTemplate, onRecordModeUsage]);

  const isAtRoot = currentModePath.length === 0;
  const containerClass = embedded
    ? "rounded-lg border border-white/10 bg-slate-800/40 overflow-hidden"
    : "mb-3 rounded-xl border border-white/10 bg-slate-800/50 overflow-hidden";

  return (
    <div className={containerClass}>
      {/* Header with breadcrumb */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-white/10 bg-slate-800/30">
        <div className="flex items-center gap-2 text-sm">
          {!isAtRoot && (
            <button onClick={onNavigateBack} className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors" aria-label="Go back">
              <ChevronLeft className="h-4 w-4" />
            </button>
          )}
          <nav className="flex items-center gap-1 text-slate-400">
            <button onClick={onResetPath} className={`hover:text-white transition-colors ${isAtRoot ? "text-white font-medium" : ""}`}>{embedded ? "All" : "Suggestions"}</button>
            {currentModePath.map((mode, index) => (
              <span key={index} className="flex items-center gap-1">
                <ChevronRight className="h-3 w-3" />
                <button onClick={() => { const newPath = currentModePath.slice(0, index + 1); onResetPath(); newPath.forEach((m) => onNavigateToMode(m)); }} className={`hover:text-white transition-colors ${index === currentModePath.length - 1 ? "text-white font-medium" : ""}`}>{mode}</button>
              </span>
            ))}
          </nav>
        </div>
        {onCreateTemplate && (
          <button onClick={() => onCreateTemplate(currentModePath)} className="flex items-center gap-1 px-2 py-1 text-xs rounded-md bg-indigo-600/20 text-indigo-300 hover:bg-indigo-600/30 transition-colors">
            <Plus className="h-3 w-3" />New
          </button>
        )}
      </div>

      {/* Content */}
      <div className="p-2 flex flex-wrap gap-2">
        {sortedSubmodes.map((mode) => {
          const ModeIcon = MODE_ICONS[mode] || Lightbulb;
          return (
            <button key={mode} onClick={() => handleModeClick(mode)} className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10 text-sm text-slate-300 hover:bg-white/10 hover:border-white/20 hover:text-white transition-all">
              <ModeIcon className="h-4 w-4" />{mode}<ChevronRight className="h-3 w-3 text-slate-500" />
            </button>
          );
        })}

        {sortedTemplates.map((template) => {
          const IconComponent = getIconComponent(template.icon);
          return (
            <div key={template.id} className="group relative flex items-center gap-2 px-3 py-1.5 rounded-lg bg-indigo-600/10 border border-indigo-500/20 text-sm text-indigo-200 hover:bg-indigo-600/20 hover:border-indigo-500/30 transition-all cursor-pointer" onClick={() => handleTemplateClick(template)}>
              {IconComponent && <IconComponent className="h-4 w-4" />}
              <span>{template.name}</span>
              <div className="hidden group-hover:flex items-center gap-1 ml-1">
                {onEditTemplate && <button onClick={(e) => { e.stopPropagation(); onEditTemplate(template); }} className="p-0.5 rounded hover:bg-white/20 text-slate-400 hover:text-white" title="Edit template"><Pencil className="h-3 w-3" /></button>}
                {template.source === "modified" && onResetTemplate && <button onClick={(e) => { e.stopPropagation(); onResetTemplate(template.id); }} className="p-0.5 rounded hover:bg-white/20 text-amber-400 hover:text-amber-300" title="Reset to default"><RotateCcw className="h-3 w-3" /></button>}
                {(template.source === "user" || template.source === "modified") && onDeleteTemplate && <button onClick={(e) => { e.stopPropagation(); onDeleteTemplate(template.id); }} className="p-0.5 rounded hover:bg-white/20 text-slate-400 hover:text-red-400" title={template.source === "modified" ? "Remove customization" : "Delete template"}><Trash2 className="h-3 w-3" /></button>}
              </div>
            </div>
          );
        })}

        {sortedSubmodes.length === 0 && sortedTemplates.length === 0 && (
          <div className="w-full text-center py-4 text-slate-500 text-sm">
            No templates at this level.{" "}
            {onCreateTemplate && <button onClick={() => onCreateTemplate(currentModePath)} className="text-indigo-400 hover:text-indigo-300">Create one?</button>}
          </div>
        )}
      </div>
    </div>
  );
}
