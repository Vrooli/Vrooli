// DOC: docs/guides/idea-agent-workflow.md#phase-2-suggest
import { useEffect, useMemo, useState } from "react";
import { Check, ChevronDown, ChevronRight, Pencil, Plus, Sparkles, Trash2 } from "lucide-react";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import {
  getSuggestionSynthesisStatus,
  computeSuggestionsSynthesisSummary,
  type SynthesisStatus,
} from "../../lib/idea-agent-files";
import type { IdeaSuggestion, IdeaSuggestionDecision } from "../../types";

interface IdeaSuggestionsPanelProps {
  suggestions: IdeaSuggestion[];
  filePath: string;
  parseError?: string;
  isSubmitting: boolean;
  submitError?: string | null;
  onSubmit: (suggestions: IdeaSuggestion[]) => void;
  onAdd?: () => void;
  onEdit?: (suggestion: IdeaSuggestion) => void;
  onDelete?: (suggestionId: string) => void;
  disabled?: boolean;
  enhanceCount?: number;
}

const DECISION_OPTIONS: Array<{ value: IdeaSuggestionDecision; label: string }> = [
  { value: "accepted", label: "Accept" },
  { value: "rejected", label: "Reject" },
  { value: "pending", label: "Pending" },
];

const SYNTHESIS_BADGE: Record<SynthesisStatus, { label: string; className: string }> = {
  new: { label: "New", className: "bg-cyan-500/20 text-cyan-300" },
  updated: { label: "Updated", className: "bg-amber-500/20 text-amber-300" },
  incorporated: { label: "", className: "" },
};

export function IdeaSuggestionsPanel({
  suggestions = [],
  filePath,
  parseError,
  isSubmitting,
  submitError,
  onSubmit,
  onAdd,
  onEdit,
  onDelete,
  disabled,
  enhanceCount,
}: IdeaSuggestionsPanelProps) {
  const [localSuggestions, setLocalSuggestions] = useState<IdeaSuggestion[]>(suggestions);
  const [expanded, setExpanded] = useState(true);

  useEffect(() => {
    setLocalSuggestions(suggestions);
  }, [suggestions]);

  const hasSuggestions = localSuggestions.length > 0;
  const synthesisSummary = useMemo(
    () => (enhanceCount && enhanceCount > 0 ? computeSuggestionsSynthesisSummary(localSuggestions) : null),
    [localSuggestions, enhanceCount],
  );

  // If parsing failed completely with no recovered items, show error-only card
  if (parseError && !hasSuggestions) {
    return (
      <Card padding="sm" data-testid={selectors.backlogDetails.suggestionsPanel}>
        <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
          <Sparkles className="h-4 w-4 text-cyan-400" />
          Suggestions Review
        </div>
        <p className="mt-3 text-sm text-red-300">Unable to parse {filePath}: {parseError}</p>
        <p className="mt-2 text-xs text-slate-400">Open the file in the preview to review or fix formatting.</p>
      </Card>
    );
  }
  const hasPending = localSuggestions.some((item) => (item.status ?? "pending") === "pending");

  return (
    <Card padding="sm" data-testid={selectors.backlogDetails.suggestionsPanel}>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 border-b border-slate-800 pb-2 text-left"
      >
        {expanded ? (
          <ChevronDown className="h-4 w-4 text-slate-400" />
        ) : (
          <ChevronRight className="h-4 w-4 text-slate-400" />
        )}
        <Sparkles className="h-4 w-4 text-cyan-400" />
        <span className="flex-1 text-sm font-medium text-slate-200">Suggestions Review</span>
        <span className="rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-400">
          {localSuggestions.length} suggestion{localSuggestions.length === 1 ? "" : "s"}
        </span>
        {synthesisSummary && (
          <span className="text-[11px] text-slate-500">
            {synthesisSummary.incorporated > 0 && <>{synthesisSummary.incorporated} synthesized</>}
            {synthesisSummary.incorporated > 0 && (synthesisSummary.updated > 0 || synthesisSummary.new > 0) && " · "}
            {synthesisSummary.updated > 0 && <span className="text-amber-400">{synthesisSummary.updated} updated</span>}
            {synthesisSummary.updated > 0 && synthesisSummary.new > 0 && " · "}
            {synthesisSummary.new > 0 && <span className="text-cyan-400">{synthesisSummary.new} new</span>}
          </span>
        )}
        {onAdd && (
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onAdd(); }}
            className="ml-1 rounded-md p-1 text-slate-400 hover:bg-slate-700 hover:text-slate-200"
            title="Add suggestion"
          >
            <Plus className="h-4 w-4" />
          </button>
        )}
      </button>

      {expanded && (
        <>
          <p className="mt-3 text-sm text-slate-400">
            Accept, reject, or edit each suggestion. Decisions are saved to {filePath} before running Enhance.
          </p>

          {parseError && (
            <div className="mt-3 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-2 text-sm text-yellow-300">
              {parseError}
            </div>
          )}

          {!hasSuggestions && (
            <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/40 p-4 text-sm text-slate-400">
              No suggestions found yet.{" "}
              {onAdd ? (
                <button type="button" onClick={onAdd} className="text-cyan-400 hover:text-cyan-300 underline">
                  Add one manually
                </button>
              ) : (
                "If the agent is still running, refresh the file list once it finishes."
              )}
            </div>
          )}

          {hasSuggestions && (
            <div className="mt-4 space-y-4">
              {localSuggestions.map((suggestion, index) => {
                const status = enhanceCount && enhanceCount > 0 ? getSuggestionSynthesisStatus(suggestion) : null;
                const badge = status ? SYNTHESIS_BADGE[status] : null;
                return (
                  <div key={suggestion.id} className="rounded-lg border border-white/10 bg-slate-800/40 p-4">
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium text-slate-100">Suggestion {index + 1}</p>
                        {badge && badge.label && (
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${badge.className}`}>
                            {badge.label}
                          </span>
                        )}
                        {status === "incorporated" && (
                          <Check className="h-3.5 w-3.5 text-emerald-500" />
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        {onEdit && (
                          <button
                            type="button"
                            onClick={() => onEdit(suggestion)}
                            className="rounded p-1 text-slate-500 hover:bg-slate-700 hover:text-slate-200"
                            title="Edit suggestion"
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </button>
                        )}
                        {onDelete && (
                          <button
                            type="button"
                            onClick={() => onDelete(suggestion.id)}
                            className="rounded p-1 text-slate-500 hover:bg-red-500/20 hover:text-red-300"
                            title="Delete suggestion"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        )}
                      </div>
                      <Select
                        value={suggestion.status ?? "pending"}
                        onChange={(event) => {
                          const s = event.target.value as IdeaSuggestionDecision;
                          setLocalSuggestions((current) =>
                            current.map((item) => (item.id === suggestion.id ? { ...item, status: s } : item))
                          );
                        }}
                        variant="compact"
                        disabled={disabled}
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
                      disabled={disabled}
                      className="mt-3 w-full rounded-md border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed"
                      rows={3}
                    />
                    {suggestion.details && (
                      <p className="mt-2 text-xs text-slate-400">{suggestion.details}</p>
                    )}
                    {(suggestion.status === "accepted" || suggestion.status === "rejected") && (
                      <textarea
                        value={suggestion.notes ?? ""}
                        onChange={(event) => {
                          const value = event.target.value;
                          setLocalSuggestions((current) =>
                            current.map((item) => (item.id === suggestion.id ? { ...item, notes: value } : item))
                          );
                        }}
                        disabled={disabled}
                        placeholder="Add a note about this decision..."
                        className="mt-2 w-full rounded-md border border-white/10 bg-slate-900/40 px-3 py-2 text-sm text-slate-300 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500 disabled:opacity-50 disabled:cursor-not-allowed"
                        rows={2}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {!disabled && (
            <>
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
            </>
          )}
        </>
      )}
    </Card>
  );
}
