import { memo } from "react";
import { Loader2 } from "lucide-react";
import { WebSearchIndicator } from "./WebSearchIndicator";
import { TemplateIndicator } from "./TemplateIndicator";
import { SkillIndicator } from "./SkillIndicator";
import { SuggestedSkills } from "./SuggestedSkills";
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
  activeTemplate: ActiveTemplateState | null;
  clearTemplate: () => void;
  setShowVariableForm: (show: boolean) => void;
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

export const MessageInputFooter = memo(function MessageInputFooter({
  isEditMode,
  modKey,
  loading,
  enableWebSearch,
  modelSupportsWebSearch,
  webSearchEnabled,
  setWebSearchEnabled,
  activeTemplate,
  clearTemplate,
  setShowVariableForm,
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
        {activeTemplate && (
          <TemplateIndicator
            template={activeTemplate.template}
            onClear={clearTemplate}
            onEdit={() => setShowVariableForm(true)}
          />
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
});
