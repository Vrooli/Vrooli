import { Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { ModelSelector } from "./ModelSelector";
import { SettingsSection, SettingsSwitchRow, SettingsNumberField } from "./SettingsControls";
import type { Model } from "../../lib/api";

interface AutoSuggestDraft {
  enabled: boolean;
  debounceMs: number;
  throttleMs: number;
  minInputLength: number;
  minScorePercent: number;
  maxSuggestions: number;
}

interface SuggestionsSettingsTabProps {
  suggestionsVisible: boolean;
  onSuggestionsVisibleChange: (visible: boolean) => void;
  mergeModel: string;
  onMergeModelChange: (model: string) => void;
  models: Model[];
  autoSuggestError?: string | null;
  suggestionsSaveError?: string | null;
  autoSuggestLoading: boolean;
  suggestionsDraft: AutoSuggestDraft;
  onSuggestionsDraftChange: (updater: (prev: AutoSuggestDraft) => AutoSuggestDraft) => void;
  isSavingSuggestions: boolean;
  onSaveSuggestions: () => void;
}

export function SuggestionsSettingsTab({
  suggestionsVisible,
  onSuggestionsVisibleChange,
  mergeModel,
  onMergeModelChange,
  models,
  autoSuggestError,
  suggestionsSaveError,
  autoSuggestLoading,
  suggestionsDraft,
  onSuggestionsDraftChange,
  isSavingSuggestions,
  onSaveSuggestions,
}: SuggestionsSettingsTabProps) {
  return (
    <div className="space-y-6">
      <SettingsSection title="Suggestions Panel">
        <SettingsSwitchRow
          title="Show Suggestions"
          description="Display template suggestions above message input"
          checked={suggestionsVisible}
          onCheckedChange={onSuggestionsVisibleChange}
        />
      </SettingsSection>

      <SettingsSection
        title="AI Merge Model"
        description="Model used when merging your message with a template"
      >
        <ModelSelector
          models={models}
          selectedModel={mergeModel}
          onSelectModel={onMergeModelChange}
          label="Merge model"
          compact
        />
      </SettingsSection>

      <SettingsSection title="Auto Skill Suggestion">
        {autoSuggestError && (
          <p className="text-xs text-amber-400 mb-3">
            Could not load persisted settings: {autoSuggestError}
          </p>
        )}
        {suggestionsSaveError && (
          <p className="text-xs text-red-400 mb-3">{suggestionsSaveError}</p>
        )}

        <SettingsSwitchRow
          title="Enable Auto Suggest"
          description="Suggest relevant skills while typing"
          checked={suggestionsDraft.enabled}
          onCheckedChange={(checked) => onSuggestionsDraftChange((prev) => ({ ...prev, enabled: checked }))}
          disabled={autoSuggestLoading}
        />

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-3">
          <SettingsNumberField
            label="Debounce (ms)"
            value={suggestionsDraft.debounceMs}
            min={100}
            max={10000}
            disabled={autoSuggestLoading}
            onChange={(next) => onSuggestionsDraftChange((prev) => ({ ...prev, debounceMs: next }))}
          />
          <SettingsNumberField
            label="Throttle (ms)"
            value={suggestionsDraft.throttleMs}
            min={1000}
            max={120000}
            disabled={autoSuggestLoading}
            onChange={(next) => onSuggestionsDraftChange((prev) => ({ ...prev, throttleMs: next }))}
          />
          <SettingsNumberField
            label="Min input length"
            value={suggestionsDraft.minInputLength}
            min={1}
            max={200}
            disabled={autoSuggestLoading}
            onChange={(next) => onSuggestionsDraftChange((prev) => ({ ...prev, minInputLength: next }))}
          />
          <SettingsNumberField
            label="Min score (%)"
            value={suggestionsDraft.minScorePercent}
            min={0}
            max={100}
            disabled={autoSuggestLoading}
            onChange={(next) => onSuggestionsDraftChange((prev) => ({ ...prev, minScorePercent: next }))}
          />
          <div className="sm:col-span-2">
            <SettingsNumberField
              label="Max suggestions"
              value={suggestionsDraft.maxSuggestions}
              min={1}
              max={20}
              disabled={autoSuggestLoading}
              onChange={(next) => onSuggestionsDraftChange((prev) => ({ ...prev, maxSuggestions: next }))}
            />
          </div>
        </div>

        <div className="flex items-center justify-end mt-4">
          <Button
            variant="secondary"
            onClick={onSaveSuggestions}
            disabled={isSavingSuggestions || autoSuggestLoading}
            data-testid="save-suggestions-settings-button"
          >
            {isSavingSuggestions ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Saving...
              </>
            ) : "Save suggestions settings"}
          </Button>
        </div>
      </SettingsSection>
    </div>
  );
}
