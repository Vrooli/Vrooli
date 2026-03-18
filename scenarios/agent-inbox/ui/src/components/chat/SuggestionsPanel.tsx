import { Sparkles, ChevronDown, ChevronUp } from "lucide-react";
import { Suggestions } from "./Suggestions";
import type {
  TemplateWithSource,
  ModeHistoryEntry,
} from "@/lib/types/templates";

interface SuggestionsPanelProps {
  suggestionsExpanded: boolean;
  setSuggestionsExpanded: (expanded: boolean) => void;
  suggestionsToggleTestId: string;
  templates: TemplateWithSource[];
  currentModePath: string[];
  modeHistory: ModeHistoryEntry[];
  handleTemplateSelect: (template: TemplateWithSource) => void;
  navigateToMode: (mode: string) => void;
  navigateBack: () => void;
  resetModePath: () => void;
  handleOpenTemplateEditor: (template?: TemplateWithSource) => void;
  handleDeleteTemplateFromSuggestions: (id: string) => void;
  handleResetTemplateFromSuggestions: (id: string) => void;
  setDefaultEditorModes: (modes: string[]) => void;
  setEditingTemplate: (template: TemplateWithSource | undefined) => void;
  setShowTemplateEditor: (show: boolean) => void;
  recordModeUsage: (path: string[]) => void;
}

export function SuggestionsPanel({
  suggestionsExpanded,
  setSuggestionsExpanded,
  suggestionsToggleTestId,
  templates,
  currentModePath,
  modeHistory,
  handleTemplateSelect,
  navigateToMode,
  navigateBack,
  resetModePath,
  handleOpenTemplateEditor,
  handleDeleteTemplateFromSuggestions,
  handleResetTemplateFromSuggestions,
  setDefaultEditorModes,
  setEditingTemplate,
  setShowTemplateEditor,
  recordModeUsage,
}: SuggestionsPanelProps) {
  return (
    <div className="mb-2 rounded-xl border border-white/10 bg-slate-900/50 overflow-hidden">
      <button
        type="button"
        onClick={() => setSuggestionsExpanded(!suggestionsExpanded)}
        className="w-full flex items-center justify-between px-3 py-2 text-left hover:bg-white/5 transition-colors"
        data-testid={suggestionsToggleTestId}
      >
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-indigo-400" />
          <span className="text-sm text-slate-200">Suggestions</span>
          <span className="text-xs text-slate-500">{templates.length}</span>
        </div>
        {suggestionsExpanded ? (
          <ChevronUp className="h-4 w-4 text-slate-400" />
        ) : (
          <ChevronDown className="h-4 w-4 text-slate-400" />
        )}
      </button>

      {suggestionsExpanded && (
        <div className="px-2 pb-2">
          <Suggestions
            embedded
            templates={templates}
            currentModePath={currentModePath}
            modeHistory={modeHistory}
            onSelectTemplate={handleTemplateSelect}
            onNavigateToMode={navigateToMode}
            onNavigateBack={navigateBack}
            onResetPath={resetModePath}
            onEditTemplate={handleOpenTemplateEditor}
            onDeleteTemplate={handleDeleteTemplateFromSuggestions}
            onResetTemplate={handleResetTemplateFromSuggestions}
            onCreateTemplate={(modes) => {
              setDefaultEditorModes(modes);
              setEditingTemplate(undefined);
              setShowTemplateEditor(true);
            }}
            onRecordModeUsage={recordModeUsage}
          />
        </div>
      )}
    </div>
  );
}
