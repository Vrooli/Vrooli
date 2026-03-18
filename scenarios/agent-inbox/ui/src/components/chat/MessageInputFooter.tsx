import { Loader2, X, Sparkles } from "lucide-react";
import { WebSearchIndicator } from "./WebSearchIndicator";
import { ForcedToolIndicator } from "./ForcedToolIndicator";
import { TemplateIndicator } from "./TemplateIndicator";
import { SkillIndicator } from "./SkillIndicator";
import { SuggestedSkills } from "./SuggestedSkills";
import type { ForcedTool } from "./AttachmentButton";
import type { Skill, Template } from "@/lib/types/templates";
import type { SuggestedSkill } from "@/lib/api";

interface ActiveTemplateState {
  template: Template;
}

interface MessageInputFooterProps {
  isEditMode: boolean;
  modKey: string;
  loading: boolean;
  enableWebSearch: boolean;
  modelSupportsWebSearch: boolean;
  webSearchEnabled: boolean;
  setWebSearchEnabled: (enabled: boolean) => void;
  forcedTool: ForcedTool | null;
  handleClearForcedTool: () => void;
  activeTemplate: ActiveTemplateState | null;
  clearTemplate: () => void;
  setShowVariableForm: (show: boolean) => void;
  activeTemplateId?: string | null;
  onTemplateDeactivate?: () => void;
  selectedSkillIds: string[];
  getSelectedSkills: () => Skill[];
  removeSkill: (id: string) => void;
  setShowSkillSelector: (show: boolean) => void;
  suggestedSkills: SuggestedSkill[];
  suggestionsLoading: boolean;
  suggestionsDidSearch: boolean;
  addSkill: (id: string) => void;
  dismissSuggestion: (id: string) => void;
  dismissAllSuggestions: () => void;
}

export function MessageInputFooter({
  isEditMode,
  modKey,
  loading,
  enableWebSearch,
  modelSupportsWebSearch,
  webSearchEnabled,
  setWebSearchEnabled,
  forcedTool,
  handleClearForcedTool,
  activeTemplate,
  clearTemplate,
  setShowVariableForm,
  activeTemplateId,
  onTemplateDeactivate,
  selectedSkillIds,
  getSelectedSkills,
  removeSkill,
  setShowSkillSelector,
  suggestedSkills,
  suggestionsLoading,
  suggestionsDidSearch,
  addSkill,
  dismissSuggestion,
  dismissAllSuggestions,
}: MessageInputFooterProps) {
  return (
    <div className="mt-1.5 sm:mt-2 px-1 space-y-1.5 sm:space-y-2">
      <div className="hidden sm:flex items-center justify-between gap-2">
        <p className="text-xs text-slate-400">
          {isEditMode ? (
            <>
              Press{" "}
              <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">
                {modKey}+Enter
              </kbd>{" "}
              to save,{" "}
              <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">
                Escape
              </kbd>{" "}
              to cancel
            </>
          ) : (
            <>
              Press{" "}
              <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">
                {modKey}+Enter
              </kbd>{" "}
              to send,{" "}
              <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">
                Enter
              </kbd>{" "}
              for new line
            </>
          )}
        </p>
        {!isEditMode && (
          <p className="text-[11px] text-slate-500 shrink-0">
            Type{" "}
            <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">
              /
            </kbd>{" "}
            for tools
          </p>
        )}
      </div>
      {!isEditMode && (
        <p className="sm:hidden text-[11px] text-slate-500">
          <kbd className="px-1 py-0.5 rounded bg-white/10 text-slate-400">
            /
          </kbd>{" "}
          tools
        </p>
      )}
      <div className="flex flex-wrap items-center gap-1.5 sm:gap-3">
        {enableWebSearch && modelSupportsWebSearch && (
          <WebSearchIndicator
            enabled={webSearchEnabled}
            onDisable={() => setWebSearchEnabled(false)}
          />
        )}
        {forcedTool && (
          <ForcedToolIndicator
            scenario={forcedTool.scenario}
            toolName={forcedTool.toolName}
            onClear={handleClearForcedTool}
          />
        )}
        {activeTemplate && (
          <TemplateIndicator
            template={activeTemplate.template}
            onClear={clearTemplate}
            onEdit={() => setShowVariableForm(true)}
          />
        )}
        {activeTemplateId && !activeTemplate && onTemplateDeactivate && (
          <div className="flex items-center gap-1.5 text-xs text-indigo-400 bg-indigo-400/10 px-2 py-1 rounded-full">
            <Sparkles className="h-3 w-3" />
            <span>Template tools active</span>
            <button
              onClick={onTemplateDeactivate}
              className="ml-1 hover:text-indigo-300 transition-colors"
              aria-label="Deactivate template tools"
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}
        {selectedSkillIds.length > 0 && (
          <SkillIndicator
            skills={getSelectedSkills()}
            onRemove={removeSkill}
            onAdd={() => setShowSkillSelector(true)}
          />
        )}
        <div className="hidden sm:block">
          <SuggestedSkills
            suggestions={suggestedSkills}
            isLoading={suggestionsLoading}
            didSearch={suggestionsDidSearch}
            onAttach={addSkill}
            onDismiss={dismissSuggestion}
            onDismissAll={dismissAllSuggestions}
          />
        </div>
        {loading && (
          <span className="text-xs text-indigo-400 flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            AI is responding...
          </span>
        )}
      </div>
    </div>
  );
}
