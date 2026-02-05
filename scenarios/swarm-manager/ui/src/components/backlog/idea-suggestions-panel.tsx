import { useEffect, useState } from "react";
import { Sparkles } from "lucide-react";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import type { IdeaSuggestion, IdeaSuggestionDecision } from "../../types";

interface IdeaSuggestionsPanelProps {
  suggestions: IdeaSuggestion[];
  filePath: string;
  parseError?: string;
  isSubmitting: boolean;
  submitError?: string | null;
  onSubmit: (suggestions: IdeaSuggestion[]) => void;
}

const DECISION_OPTIONS: Array<{ value: IdeaSuggestionDecision; label: string }> = [
  { value: "accepted", label: "Accept" },
  { value: "rejected", label: "Reject" },
  { value: "pending", label: "Pending" },
];

export function IdeaSuggestionsPanel({
  suggestions = [],
  filePath,
  parseError,
  isSubmitting,
  submitError,
  onSubmit,
}: IdeaSuggestionsPanelProps) {
  const [localSuggestions, setLocalSuggestions] = useState<IdeaSuggestion[]>(suggestions);

  useEffect(() => {
    setLocalSuggestions(suggestions);
  }, [suggestions]);

  if (parseError) {
    return (
      <Card padding="lg" data-testid={selectors.backlogDetails.suggestionsPanel}>
        <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
          <Sparkles className="h-4 w-4 text-cyan-400" />
          Suggestions Review
        </div>
        <p className="mt-3 text-sm text-red-300">Unable to parse {filePath}: {parseError}</p>
        <p className="mt-2 text-xs text-slate-400">Open the file in the preview to review or fix formatting.</p>
      </Card>
    );
  }

  const hasSuggestions = localSuggestions.length > 0;
  const hasPending = localSuggestions.some((item) => (item.status ?? "pending") === "pending");

  return (
    <Card padding="lg" data-testid={selectors.backlogDetails.suggestionsPanel}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
            <Sparkles className="h-4 w-4 text-cyan-400" />
            Suggestions Review
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Accept, reject, or edit each suggestion. Decisions are saved to {filePath} before running Enhance.
          </p>
        </div>
        <span className="rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-400">
          {localSuggestions.length} suggestion{localSuggestions.length === 1 ? "" : "s"}
        </span>
      </div>

      {!hasSuggestions && (
        <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/40 p-4 text-sm text-slate-400">
          No suggestions found yet. If the agent is still running, refresh the file list once it finishes.
        </div>
      )}

      {hasSuggestions && (
        <div className="mt-4 space-y-4">
          {localSuggestions.map((suggestion, index) => (
            <div key={suggestion.id} className="rounded-lg border border-white/10 bg-slate-800/40 p-4">
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm font-medium text-slate-100">Suggestion {index + 1}</p>
                <Select
                  value={suggestion.status ?? "pending"}
                  onChange={(event) => {
                    const status = event.target.value as IdeaSuggestionDecision;
                    setLocalSuggestions((current) =>
                      current.map((item) => (item.id === suggestion.id ? { ...item, status } : item))
                    );
                  }}
                  variant="compact"
                >
                  {DECISION_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </Select>
              </div>
              <textarea
                value={suggestion.suggestion ?? ""}
                onChange={(event) => {
                  const value = event.target.value;
                  setLocalSuggestions((current) =>
                    current.map((item) => (item.id === suggestion.id ? { ...item, suggestion: value } : item))
                  );
                }}
                className="mt-3 w-full rounded-md border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                rows={3}
              />
              {suggestion.details && (
                <p className="mt-2 text-xs text-slate-400">{suggestion.details}</p>
              )}
            </div>
          ))}
        </div>
      )}

      {hasPending && hasSuggestions && (
        <div className="mt-3 text-xs text-slate-400">
          Choose accept or reject for each suggestion before continuing.
        </div>
      )}

      {submitError && (
        <div className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {submitError}
        </div>
      )}

      <div className="mt-4 flex justify-end">
        <Button
          onClick={() => onSubmit(localSuggestions)}
          disabled={isSubmitting || !hasSuggestions || hasPending}
          data-testid={selectors.backlogDetails.suggestionsSubmit}
        >
          {isSubmitting ? "Submitting..." : "Save Decisions & Run Enhance"}
        </Button>
      </div>
    </Card>
  );
}
