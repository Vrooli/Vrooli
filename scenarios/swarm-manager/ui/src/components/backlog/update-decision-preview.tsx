/**
 * UpdateDecisionPreview — Shows a before/after diff of what "Update decision"
 * will change, letting the user confirm or cancel before applying.
 */

import { useMemo } from "react";
import { AlertTriangle, ArrowLeft, Check, Loader2 } from "lucide-react";
import type { WorkshopItem, DecisionOption } from "../../types/domain";

interface UpdateDecisionPreviewProps {
  currentItem: WorkshopItem;
  suggestedUpdate: string;
  contextNote: string;
  onConfirm: () => void;
  onCancel: () => void;
  isApplying: boolean;
}

interface ParsedUpdate {
  topic?: string;
  context?: string;
  options?: DecisionOption[];
}

export function UpdateDecisionPreview({
  currentItem,
  suggestedUpdate,
  contextNote,
  onConfirm,
  onCancel,
  isApplying,
}: UpdateDecisionPreviewProps) {
  const { parsed, parseError } = useMemo(() => {
    try {
      const obj = JSON.parse(suggestedUpdate) as ParsedUpdate;
      return { parsed: obj, parseError: null };
    } catch {
      return { parsed: null, parseError: "Failed to parse suggested update." };
    }
  }, [suggestedUpdate]);

  const topicChanged = parsed?.topic != null && parsed.topic !== currentItem.topic;
  const contextChanged = parsed?.context != null && parsed.context !== currentItem.context;
  const optionsChanged = parsed?.options != null && !optionsEqual(currentItem.options, parsed.options);
  const hasFieldChanges = topicChanged || contextChanged || optionsChanged;
  const hasSelection = currentItem.selected != null;

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center gap-2 border-b border-slate-700 px-3 py-2">
        <button
          type="button"
          onClick={onCancel}
          disabled={isApplying}
          className="rounded p-0.5 text-slate-400 transition-colors hover:bg-slate-700 hover:text-slate-200 disabled:opacity-50"
          title="Back"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
        </button>
        <span className="text-xs font-medium text-slate-300">Preview update</span>
      </div>

      {/* Scrollable diff area */}
      <div className="flex-1 overflow-y-auto px-3 py-2 space-y-3">
        {parseError ? (
          <div className="rounded-md border border-red-500/20 bg-red-500/10 px-3 py-2">
            <p className="text-xs text-red-300">{parseError}</p>
          </div>
        ) : (
          <>
            {/* Topic diff */}
            {topicChanged && (
              <DiffSection label="Topic">
                <OldValue>{currentItem.topic}</OldValue>
                <NewValue>{parsed?.topic}</NewValue>
              </DiffSection>
            )}

            {/* Context diff */}
            {contextChanged && (
              <DiffSection label="Context">
                <OldValue>{currentItem.context}</OldValue>
                <NewValue>{parsed?.context}</NewValue>
              </DiffSection>
            )}

            {/* Options diff */}
            {optionsChanged && (
              <DiffSection label="Options">
                <OptionsDiff
                  current={currentItem.options ?? []}
                  proposed={parsed?.options ?? []}
                />
              </DiffSection>
            )}

            {/* No field changes */}
            {!hasFieldChanges && (
              <p className="text-xs text-slate-500">
                No changes to topic, context, or options.
              </p>
            )}

            {/* Context note */}
            {contextNote && (
              <DiffSection label="Context note (will be attached)">
                <p className="text-xs text-cyan-400/80">{contextNote}</p>
              </DiffSection>
            )}

            {/* Selection warning */}
            {hasSelection && (
              <div className="flex items-start gap-2 rounded-md border border-amber-500/20 bg-amber-500/5 px-3 py-2">
                <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0 text-amber-400" />
                <p className="text-xs text-amber-300">
                  Your current selection will be cleared — you'll need to re-answer this decision.
                </p>
              </div>
            )}
          </>
        )}
      </div>

      {/* Footer buttons */}
      <div className="flex items-center justify-end gap-2 border-t border-slate-700 px-3 py-2">
        <button
          type="button"
          onClick={onCancel}
          disabled={isApplying}
          className="rounded-md border border-slate-600 bg-slate-800/50 px-3 py-1.5 text-xs font-medium text-slate-400 transition-colors hover:border-slate-500 hover:text-slate-300 disabled:opacity-50"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={onConfirm}
          disabled={isApplying || !!parseError}
          className="flex items-center gap-1.5 rounded-md border border-cyan-500/40 bg-cyan-500/10 px-3 py-1.5 text-xs font-medium text-cyan-400 transition-colors hover:bg-cyan-500/20 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isApplying ? (
            <Loader2 className="h-3 w-3 animate-spin" />
          ) : (
            <Check className="h-3 w-3" />
          )}
          Apply update
        </button>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function DiffSection({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <span className="text-[10px] font-medium uppercase tracking-wide text-slate-500">
        {label}
      </span>
      <div className="mt-1 space-y-1">{children}</div>
    </div>
  );
}

function OldValue({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-xs text-red-400/60 line-through">{children}</p>
  );
}

function NewValue({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-xs text-emerald-400">{children}</p>
  );
}

function OptionsDiff({ current, proposed }: { current: DecisionOption[]; proposed: DecisionOption[] }) {
  const currentByKey = new Map(current.map((o) => [o.key, o]));
  const proposedByKey = new Map(proposed.map((o) => [o.key, o]));
  const allKeys = [...new Set([...currentByKey.keys(), ...proposedByKey.keys()])];

  return (
    <div className="space-y-1.5">
      {allKeys.map((key) => {
        const cur = currentByKey.get(key);
        const prop = proposedByKey.get(key);

        if (cur && !prop) {
          // Removed
          return (
            <div key={key} className="text-xs">
              <span className="text-red-400/60 line-through">
                {key}: {cur.label}
              </span>
              <span className="ml-1 text-[10px] text-red-400/40">(removed)</span>
            </div>
          );
        }
        if (!cur && prop) {
          // Added
          return (
            <div key={key} className="text-xs">
              <span className="text-emerald-400">
                {key}: {prop.label}
                {prop.recommended && " ★"}
              </span>
              <span className="ml-1 text-[10px] text-emerald-400/60">(new)</span>
            </div>
          );
        }
        if (cur && prop && (cur.label !== prop.label || cur.rationale !== prop.rationale || cur.recommended !== prop.recommended)) {
          // Modified
          return (
            <div key={key} className="text-xs space-y-0.5">
              <div className="text-red-400/60 line-through">
                {key}: {cur.label}
              </div>
              <div className="text-emerald-400">
                {key}: {prop.label}
                {prop.recommended && " ★"}
              </div>
            </div>
          );
        }
        // Unchanged — skip
        return null;
      })}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function optionsEqual(a?: DecisionOption[], b?: DecisionOption[]): boolean {
  if (!a && !b) return true;
  if (!a || !b) return false;
  if (a.length !== b.length) return false;
  return a.every((opt, i) => {
    const other = b[i];
    if (!other) return false;
    return (
      opt.key === other.key &&
      opt.label === other.label &&
      opt.rationale === other.rationale &&
      opt.recommended === other.recommended
    );
  });
}
