/**
 * FeedbackDialog — "Add Feedback" entry point on an initiative.
 *
 * Capture-first modal where a user types free-form feedback, optionally
 * attaches images, and submits. The server spawns a feedback agent that
 * produces a proposal — reviewed afterwards in the FeedbackPanel.
 *
 * Three round types are exposed:
 *   - Feedback: active; full agent round.
 *   - Note: active; saves the submission but doesn't spawn an agent.
 *     Still shows up in the feedback log for future meta-optimizer mining.
 *   - Research: scaffolded but disabled; the chip renders a "Coming soon"
 *     badge so users can see the shape without being able to trigger it.
 *
 * Active-agent awareness is proactive: on open we fetch the initiative's
 * lock status (a single call that returns the holder and any busy backlog
 * items). If the initiative isn't free, the warning block renders
 * immediately and the textarea is disabled until the user explicitly picks
 * override. This is the rule the plan calls out as "should not be able to
 * spawn another agent while one is running" — the UI enforces it up-front
 * instead of letting the user type, submit, and get surprised by a 409.
 *
 * Note-type rounds bypass this guard because they don't spawn an agent and
 * therefore can't violate single-agent-per-initiative.
 *
 * Quick Actions surface (Plan A): when caller passes `items`, the dialog
 * also exposes a selection-driven picker, five quick-action toggles
 * (split / merge / identify-missing / reconcile / reframe), and a
 * collapsible help block. Selecting any action or any item wraps the
 * submission in an XML envelope the feedback skill knows how to
 * interpret. Plain free-prose (no actions, no selection) submits the raw
 * textarea text, exactly as today.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2, Paperclip, SendHorizontal, AlertTriangle, ChevronDown, ChevronRight } from "lucide-react";
import { Dialog } from "../ui/dialog";
import { Button } from "../ui/button";
import { CaptureAttachmentPreview } from "../capture/capture-attachment-preview";
import { useIndexedDBAttachments } from "../../hooks/useIndexedDBAttachments";
import { useAutoResizeTextarea } from "../../hooks/useAutoResizeTextarea";
import {
  feedbackService,
  FeedbackBusyError,
  FeedbackLockConflictError,
} from "../../services/feedback-service";
import { selectors } from "../../consts/selectors";
import type {
  FeedbackRound,
  FeedbackRoundType,
  ItemActivity,
  LockHolder,
} from "../../types";
import { buildEnvelope, pruneActionsForSelection } from "./feedback-dialog-envelope";
import type { QuickActionKey } from "./feedback-dialog-envelope";

const DRAFT_KEY_PREFIX = "swarm-initiative-feedback-draft:";
const MAX_VISIBLE_LINES = 8;
const LINE_HEIGHT_PX = 22;
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;

/** Minimal item shape the picker needs. The parent passes whatever it
 *  already has — we don't reach back into stores. */
export interface FeedbackDialogItem {
  ref: string; // "kind/name"
  title: string;
}

const QUICK_ACTIONS: ReadonlyArray<{
  key: QuickActionKey;
  label: string;
  hint: string;
  testId: string;
}> = [
  {
    key: "split_oversized",
    label: "Split oversized items",
    hint: "Break apart items that bundle multiple units of work.",
    testId: selectors.feedback.dialogQuickActionSplit,
  },
  {
    key: "merge_coupled",
    label: "Merge coupled items",
    hint: "Collapse items that share a substrate or only validate together.",
    testId: selectors.feedback.dialogQuickActionMerge,
  },
  {
    key: "identify_missing_work",
    label: "Identify missing work",
    hint: "Inspect the code state and propose follow-ups, tests, or gaps.",
    testId: selectors.feedback.dialogQuickActionIdentifyMissing,
  },
  {
    key: "reconcile_with_code_drift",
    label: "Reconcile with code drift",
    hint: "Update items that no longer match what the code does.",
    testId: selectors.feedback.dialogQuickActionReconcile,
  },
  {
    key: "reframe_scope",
    label: "Reframe scope",
    hint: "Holistic re-shape: items partitioned along the wrong lines.",
    testId: selectors.feedback.dialogQuickActionReframe,
  },
];

export interface FeedbackDialogProps {
  /** Name of the initiative this feedback is scoped to. */
  initiativeName: string;
  /** Whether the dialog is open. */
  isOpen: boolean;
  /** Close callback. */
  onClose: () => void;
  /** Called after a round is successfully created. Parent uses this to refetch
   *  the rounds list and switch to the Feedback tab. */
  onSubmitted?: (round: FeedbackRound) => void;
  /** Initiative member items for the picker. When omitted, the
   *  selection picker and quick actions are hidden — the dialog
   *  degrades to the original free-prose surface. */
  items?: FeedbackDialogItem[];
}

type TypeOption = {
  value: FeedbackRoundType;
  label: string;
  hint: string;
  disabled?: boolean;
};

const TYPE_OPTIONS: TypeOption[] = [
  { value: "feedback", label: "Feedback", hint: "Agent proposes mutations" },
  { value: "research", label: "Research", hint: "Coming soon", disabled: true },
  { value: "note", label: "Note", hint: "Logged without agent" },
];

/** Combined "what's blocking a new round" state. Populated by either the
 *  preflight lock query or a post-submit 409. */
interface Blocker {
  holder?: LockHolder;
  activities?: ItemActivity[];
}

function blockerIsEmpty(b: Blocker | null): b is null {
  return b == null || (!b.holder && (!b.activities || b.activities.length === 0));
}

export function FeedbackDialog({ initiativeName, isOpen, onClose, onSubmitted, items }: FeedbackDialogProps) {
  const draftKey = `${DRAFT_KEY_PREFIX}${initiativeName}`;
  const itemsKey = `${draftKey}:items`;
  const actionsKey = `${draftKey}:actions`;
  const helpOpenKey = `${draftKey}:help-open`;
  const pickerOpenKey = `${draftKey}:picker-open`;

  const [text, setText] = useState(() => safeLoadDraft(draftKey));
  const [type, setType] = useState<FeedbackRoundType>("feedback");
  const [override, setOverride] = useState(false);
  const [blocker, setBlocker] = useState<Blocker | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Quick Actions state — only meaningful when `items` is provided.
  const itemRefs = useMemo(() => (items ?? []).map((i) => i.ref), [items]);
  const [selectedItems, setSelectedItems] = useState<Set<string>>(() => safeLoadStringSet(itemsKey, itemRefs));
  const [selectedActions, setSelectedActions] = useState<Set<QuickActionKey>>(() => safeLoadActionSet(actionsKey));
  const [helpOpen, setHelpOpen] = useState<boolean>(() => safeLoadBoolean(helpOpenKey, false));
  const [pickerOpen, setPickerOpen] = useState<boolean>(() => safeLoadBoolean(pickerOpenKey, false));

  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  // Feedback attachments persist in IndexedDB per-initiative so a half-composed
  // dialog survives a refresh — unlike the ephemeral `useAttachments` hook
  // clarification-panel uses, which holds attachments in memory only.
  // Feedback rounds can take real time to compose (screenshots, multi-step
  // thoughts); losing that work on an accidental reload is user-hostile.
  const {
    attachments,
    addFile,
    removeFile,
    clearAll,
    getFiles,
  } = useIndexedDBAttachments({ dbName: `swarm-initiative-feedback-attachments:${initiativeName}` });

  useAutoResizeTextarea(textareaRef, text, { maxHeight: MAX_TEXTAREA_HEIGHT });

  // Proactive preflight: ask the server whether the initiative is free to
  // accept a new agent-spawning round. The query runs only while the dialog
  // is open and the user has picked a type that actually spawns an agent
  // (note-type rounds don't need the guard). Stale-while-revalidate with a
  // short refetch so the moment another round completes, the warning
  // clears without the user having to close and reopen.
  const needsPreflight = isOpen && type !== "note";
  const lockQuery = useQuery({
    queryKey: ["initiative-feedback-lock", initiativeName],
    queryFn: () => feedbackService.lockStatus(initiativeName),
    enabled: needsPreflight,
    refetchInterval: needsPreflight ? 5000 : false,
    staleTime: 2000,
  });

  // Sync the preflight result into the blocker state. Preflight is the
  // authoritative view of "what's running right now"; if it says the
  // initiative is free, we clear any stale blocker (including one set by
  // a prior 409) — the lock was released and the user can proceed. If a
  // new submit races back with a 409, the mutation's onError will set
  // the blocker again.
  useEffect(() => {
    if (!needsPreflight) {
      setBlocker(null);
      return;
    }
    if (!lockQuery.data) return;
    const { locked, holder, item_activities } = lockQuery.data;
    const hasBusy = (item_activities?.length ?? 0) > 0;
    if (locked || hasBusy) {
      setBlocker({
        holder: locked ? holder : undefined,
        activities: hasBusy ? item_activities : undefined,
      });
    } else {
      setBlocker(null);
    }
  }, [lockQuery.data, needsPreflight]);

  // Persist draft debounced so switching initiatives doesn't lose work.
  useEffect(() => {
    if (!isOpen) return;
    const handle = setTimeout(() => safeSaveDraft(draftKey, text), 300);
    return () => clearTimeout(handle);
  }, [draftKey, isOpen, text]);

  // Persist selection-state alongside the textarea draft. The picker /
  // actions / help-open / picker-open flags are localStorage primitives —
  // small enough to write synchronously on each change without debouncing.
  useEffect(() => {
    if (!isOpen) return;
    safeSaveStringSet(itemsKey, selectedItems);
  }, [isOpen, itemsKey, selectedItems]);
  useEffect(() => {
    if (!isOpen) return;
    safeSaveStringSet(actionsKey, selectedActions as Set<string>);
  }, [isOpen, actionsKey, selectedActions]);
  useEffect(() => {
    if (!isOpen) return;
    safeSaveBoolean(helpOpenKey, helpOpen);
  }, [isOpen, helpOpenKey, helpOpen]);
  useEffect(() => {
    if (!isOpen) return;
    safeSaveBoolean(pickerOpenKey, pickerOpen);
  }, [isOpen, pickerOpenKey, pickerOpen]);

  // Reset transient state whenever the dialog reopens.
  useEffect(() => {
    if (isOpen) {
      setError(null);
      setOverride(false);
      setBlocker(null);
      setText(safeLoadDraft(draftKey));
      setSelectedItems(safeLoadStringSet(itemsKey, itemRefs));
      setSelectedActions(safeLoadActionSet(actionsKey));
      setHelpOpen(safeLoadBoolean(helpOpenKey, false));
      setPickerOpen(safeLoadBoolean(pickerOpenKey, false));
    }
  }, [draftKey, isOpen, itemsKey, actionsKey, helpOpenKey, pickerOpenKey, itemRefs]);

  // Quick Actions are available on all feedback rounds — including
  // initiatives with zero live items. The "identify missing work" and
  // "reconcile with code drift" lenses are explicitly designed to run
  // with zero selection (gap-and-drift sweeps on completed initiatives
  // are the most common feedback use case). Note rounds don't spawn an
  // agent so they skip the envelope entirely.
  const actionsAvailable = type === "feedback";
  // The picker only makes sense when there's something to pick from.
  const pickerAvailable = actionsAvailable && (items?.length ?? 0) > 0;

  // Compose the body sent to feedbackService.start. When at least one
  // action OR at least one item is selected, wrap in the XML envelope;
  // otherwise pass the raw textarea content unchanged. Note-type rounds
  // always use raw text because actions are hidden.
  const submissionText = useMemo(() => {
    if (!actionsAvailable) return text;
    const hasActions = selectedActions.size > 0;
    const hasItems = selectedItems.size > 0;
    if (!hasActions && !hasItems) return text;
    return buildEnvelope({ items: [...selectedItems], actions: [...selectedActions], note: text });
  }, [actionsAvailable, selectedActions, selectedItems, text]);

  const mutation = useMutation({
    mutationFn: async () => {
      return feedbackService.start(initiativeName, {
        type,
        text: submissionText.trim(),
        files: getFiles(),
        override,
      });
    },
    onSuccess: (round) => {
      clearAll();
      safeClearDraft(draftKey);
      safeClearKey(itemsKey);
      safeClearKey(actionsKey);
      // Help-open and picker-open are user preferences, not draft data —
      // intentionally preserved across submissions so a user who keeps
      // the help block open keeps it open.
      setText("");
      setSelectedItems(new Set());
      setSelectedActions(new Set());
      setError(null);
      setOverride(false);
      setBlocker(null);
      onSubmitted?.(round);
      onClose();
    },
    onError: (err) => {
      if (err instanceof FeedbackLockConflictError) {
        setBlocker({ holder: err.holder });
        setError(null);
        return;
      }
      if (err instanceof FeedbackBusyError) {
        setBlocker({ activities: err.activities });
        setError(null);
        return;
      }
      setError(err instanceof Error ? err.message : "Failed to submit feedback.");
    },
  });

  const blocked = type !== "note" && !blockerIsEmpty(blocker);
  // Submit is allowed when there is *something* to send. With Quick
  // Actions enabled, picking ≥1 action is enough to send even if the
  // textarea is empty — the agent reads the envelope alone. Without
  // Quick Actions (or when nothing is selected), we still require text.
  const hasMeaningfulInput = actionsAvailable
    ? text.trim().length > 0 || selectedActions.size > 0
    : text.trim().length > 0;
  const canSubmit = hasMeaningfulInput && !mutation.isPending && (!blocked || override);
  const textareaDisabled = mutation.isPending || (blocked && !override);

  const handleSubmit = useCallback(() => {
    if (!canSubmit) return;
    setError(null);
    mutation.mutate();
  }, [canSubmit, mutation]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;
    Array.from(files).forEach(addFile);
    e.target.value = "";
  };

  const placeholder = useMemo(() => {
    if (type === "note") {
      return "A quick note about this initiative… (Ctrl+Enter to save)";
    }
    if (blocked && !override) {
      return "Override the active agent before composing your message.";
    }
    if (actionsAvailable && selectedActions.size > 0) {
      return "Add anything else the agent should know… (Ctrl+Enter to submit)";
    }
    return "What's off? What should change? (Ctrl+Enter to submit)";
  }, [type, blocked, override, actionsAvailable, selectedActions]);

  // Quick action toggle handler. Enforces combinability rules so the
  // UI state is always valid:
  //   - split ⊥ merge
  //   - reframe is solo
  //   - identify_missing + reconcile stack with each other and with
  //     split or merge
  const toggleAction = useCallback((key: QuickActionKey) => {
    setSelectedActions((prev) => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
        return next;
      }
      if (key === "reframe_scope") {
        return new Set<QuickActionKey>(["reframe_scope"]);
      }
      // Selecting any prescriptive lens removes reframe.
      next.delete("reframe_scope");
      // split ⊥ merge
      if (key === "split_oversized") next.delete("merge_coupled");
      if (key === "merge_coupled") next.delete("split_oversized");
      next.add(key);
      return next;
    });
  }, []);

  // Gating: which actions are clickable given current selection count
  // and mutual exclusion. Reframe being active does NOT disable the
  // others — clicking another action while reframe is selected toggles
  // reframe off (per the "selecting any other clears it" rule).
  const actionEnabled = useCallback(
    (key: QuickActionKey): boolean => {
      const hasSplit = selectedActions.has("split_oversized");
      const hasMerge = selectedActions.has("merge_coupled");
      const itemCount = selectedItems.size;
      switch (key) {
        case "split_oversized":
          return !hasMerge && itemCount >= 1;
        case "merge_coupled":
          return !hasSplit && itemCount >= 2;
        case "identify_missing_work":
        case "reconcile_with_code_drift":
        case "reframe_scope":
          return true;
      }
    },
    [selectedActions, selectedItems],
  );

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      title="Add Feedback"
      maxWidth="max-w-2xl"
      isLoading={mutation.isPending}
      testId={selectors.feedback.dialog}
    >
      <div className="space-y-4">
        <p className="text-sm text-slate-400">
          Capture observations, frustrations, or ideas about this initiative.
          Feedback can result in proposed backlog mutations you can review and
          accept.
        </p>

        {/* Type selector */}
        <div className="flex flex-wrap gap-2">
          {TYPE_OPTIONS.map((opt) => {
            const selected = type === opt.value;
            const testId = {
              feedback: selectors.feedback.dialogTypeFeedback,
              research: selectors.feedback.dialogTypeResearch,
              note: selectors.feedback.dialogTypeNote,
            }[opt.value];
            return (
              <button
                key={opt.value}
                type="button"
                disabled={opt.disabled || mutation.isPending}
                onClick={() => setType(opt.value)}
                aria-pressed={selected}
                data-testid={testId}
                className={`rounded-full border px-3 py-1.5 text-xs font-medium transition-colors ${
                  selected
                    ? "border-cyan-400/60 bg-cyan-500/20 text-cyan-200"
                    : "border-slate-700 bg-slate-800/60 text-slate-300 hover:border-slate-500"
                } disabled:cursor-not-allowed disabled:opacity-40`}
              >
                <span>{opt.label}</span>
                <span className="ml-2 text-[10px] font-normal uppercase tracking-wider text-slate-500">
                  {opt.hint}
                </span>
              </button>
            );
          })}
        </div>

        {/* Item-selection picker — only when there are items to pick from. */}
        {pickerAvailable && (
          <ItemPicker
            items={items ?? []}
            selected={selectedItems}
            onToggle={(ref) => {
              setSelectedItems((prev) => {
                const next = new Set(prev);
                if (next.has(ref)) next.delete(ref);
                else next.add(ref);
                return next;
              });
              // Drop split/merge if user removes items below threshold.
              setSelectedActions((prev) => pruneActionsForSelection(prev, computeNextSelectionSize(selectedItems, ref)));
            }}
            onSelectAll={() => {
              const all = new Set(itemRefs);
              setSelectedItems(all);
            }}
            onSelectNone={() => {
              setSelectedItems(new Set());
              setSelectedActions((prev) => pruneActionsForSelection(prev, 0));
            }}
            open={pickerOpen}
            onToggleOpen={() => setPickerOpen((v) => !v)}
            disabled={mutation.isPending}
          />
        )}

        {/* Quick action buttons — visible on every feedback round so
            gap-and-drift sweeps work on initiatives with zero live items. */}
        {actionsAvailable && (
          <div className="flex flex-wrap gap-2">
            {QUICK_ACTIONS.map((action) => {
              const active = selectedActions.has(action.key);
              const enabled = actionEnabled(action.key);
              return (
                <button
                  key={action.key}
                  type="button"
                  data-testid={action.testId}
                  disabled={!enabled || mutation.isPending}
                  aria-pressed={active}
                  onClick={() => toggleAction(action.key)}
                  title={action.hint}
                  className={`rounded-md border px-2.5 py-1.5 text-xs font-medium transition-colors ${
                    active
                      ? "border-cyan-400/60 bg-cyan-500/20 text-cyan-200"
                      : "border-slate-700 bg-slate-800/60 text-slate-300 hover:border-slate-500"
                  } disabled:cursor-not-allowed disabled:opacity-40`}
                >
                  {action.label}
                </button>
              );
            })}
          </div>
        )}

        {/* Help block — visible on all round types so users always see
            what feedback can do. Defaults closed; state persists. */}
        <HelpBlock open={helpOpen} onToggle={() => setHelpOpen((v) => !v)} />

        {/* Active-agent warning. Rendered above the textarea so the user
            sees it before they start composing. When blocked and override
            isn't checked the textarea is disabled; the user has to
            consciously opt in to preempt whatever is running. */}
        {blocked && blocker && (
          <BlockerNotice
            blocker={blocker}
            override={override}
            onOverrideChange={setOverride}
          />
        )}

        {/* Textarea */}
        <div className="rounded-xl border border-slate-700 bg-slate-800/50 p-3 focus-within:border-cyan-500/50">
          <textarea
            ref={textareaRef}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={textareaDisabled}
            className="block w-full resize-none bg-transparent text-sm leading-relaxed text-slate-100 placeholder-slate-500 outline-none disabled:opacity-60"
            data-testid={selectors.feedback.dialogText}
            rows={4}
          />
          <CaptureAttachmentPreview attachments={attachments} onRemove={removeFile} />
        </div>

        {/* Non-lock errors (anything that wasn't a known 409 shape). */}
        {error && (
          <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
            {error}
          </p>
        )}

        {/* Actions */}
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <input
              ref={fileInputRef}
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              multiple
              className="hidden"
              onChange={handleFileSelect}
            />
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={mutation.isPending}
              className="rounded p-2 text-slate-400 transition-colors hover:bg-slate-800 hover:text-slate-200 disabled:opacity-40"
              title="Attach image"
              aria-label="Attach image"
              data-testid={selectors.feedback.dialogAttach}
            >
              <Paperclip className="h-4 w-4" aria-hidden="true" />
            </button>
            <span className="text-[11px] text-slate-500">
              {attachments.length > 0
                ? `${attachments.length} attachment${attachments.length === 1 ? "" : "s"}`
                : "No attachments"}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={onClose}
              disabled={mutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={handleSubmit}
              disabled={!canSubmit}
              data-testid={selectors.feedback.dialogSubmit}
            >
              {mutation.isPending ? (
                <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
              ) : (
                <SendHorizontal className="mr-1.5 h-3.5 w-3.5" />
              )}
              {mutation.isPending
                ? "Submitting…"
                : blocked && override
                  ? "Override & submit"
                  : "Submit"}
            </Button>
          </div>
        </div>
      </div>
    </Dialog>
  );
}

// ---------------------------------------------------------------------------
// Item-selection picker
// ---------------------------------------------------------------------------

interface ItemPickerProps {
  items: FeedbackDialogItem[];
  selected: Set<string>;
  onToggle: (ref: string) => void;
  onSelectAll: () => void;
  onSelectNone: () => void;
  open: boolean;
  onToggleOpen: () => void;
  disabled?: boolean;
}

function ItemPicker({ items, selected, onToggle, onSelectAll, onSelectNone, open, onToggleOpen, disabled }: ItemPickerProps) {
  const summary = `Target items — ${selected.size} of ${items.length} selected`;
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800/40" data-testid={selectors.feedback.dialogTargetPicker}>
      <button
        type="button"
        onClick={onToggleOpen}
        disabled={disabled}
        aria-expanded={open}
        data-testid={selectors.feedback.dialogTargetPickerToggle}
        className="flex w-full items-center justify-between px-3 py-2 text-left text-xs text-slate-200 hover:bg-slate-800/60 disabled:opacity-40"
      >
        <span>{summary}</span>
        {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
      </button>
      {open && (
        <div className="border-t border-slate-700 p-2">
          <div className="mb-2 flex gap-2">
            <button
              type="button"
              data-testid={selectors.feedback.dialogTargetPickerSelectAll}
              onClick={onSelectAll}
              disabled={disabled}
              className="rounded border border-slate-700 bg-slate-800/60 px-2 py-0.5 text-[11px] text-slate-300 hover:border-slate-500 disabled:opacity-40"
            >
              Select all
            </button>
            <button
              type="button"
              data-testid={selectors.feedback.dialogTargetPickerSelectNone}
              onClick={onSelectNone}
              disabled={disabled}
              className="rounded border border-slate-700 bg-slate-800/60 px-2 py-0.5 text-[11px] text-slate-300 hover:border-slate-500 disabled:opacity-40"
            >
              Select none
            </button>
          </div>
          <ul className="max-h-48 space-y-0.5 overflow-y-auto">
            {items.map((item) => {
              const isChecked = selected.has(item.ref);
              return (
                <li key={item.ref}>
                  <label
                    className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-[12px] text-slate-300 hover:bg-slate-800/60"
                    data-testid={selectors.feedback.dialogTargetPickerItem}
                    data-ref={item.ref}
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => onToggle(item.ref)}
                      disabled={disabled}
                      className="h-3.5 w-3.5 accent-cyan-400"
                    />
                    <code className="text-cyan-300/80">{item.ref}</code>
                    <span className="truncate text-slate-400">{item.title}</span>
                  </label>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Help block
// ---------------------------------------------------------------------------

interface HelpBlockProps {
  open: boolean;
  onToggle: () => void;
}

/** Plain-language summary of what feedback can produce. The skill prompt
 *  carries the canonical contract; this is the operator-facing version. */
function HelpBlock({ open, onToggle }: HelpBlockProps) {
  return (
    <div
      className="rounded-lg border border-slate-700 bg-slate-800/40 text-xs text-slate-300"
      data-testid={selectors.feedback.dialogHelpBlock}
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={open}
        data-testid={selectors.feedback.dialogHelpBlockToggle}
        className="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-slate-800/60"
      >
        <span>What can feedback do?</span>
        {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
      </button>
      {open && (
        <div className="space-y-2 border-t border-slate-700 p-3 text-slate-400">
          <p>
            The agent reads your selection, your text, and any attachments, then
            proposes a checklist of mutations you accept or reject in the
            proposal panel. Quick actions are starting points for common
            investigation tasks — the agent can also propose any of the
            mutations below from your free-form text.
          </p>
          <ul className="list-disc space-y-0.5 pl-4">
            <li>Split an item into smaller items, or merge several into one.</li>
            <li>Archive items that are no longer relevant.</li>
            <li>Change priority or non-lifecycle status.</li>
            <li>Add or remove a depends-on edge between two items.</li>
            <li>Move an item to a different initiative.</li>
            <li>Interrupt a running execution before it finishes.</li>
            <li>Update an item's title, description, tags, or acceptance globs.</li>
            <li>Add a brand-new item for missing work.</li>
          </ul>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Warning block
// ---------------------------------------------------------------------------

interface BlockerNoticeProps {
  blocker: Blocker;
  override: boolean;
  onOverrideChange: (value: boolean) => void;
}

/** The amber panel that surfaces whatever is blocking a new round.
 *  Renders both the initiative-level lock holder and any busy backlog
 *  items — they can coexist (e.g. a feedback round holds the lock AND a
 *  workshop agent is running on an item). */
function BlockerNotice({ blocker, override, onOverrideChange }: BlockerNoticeProps) {
  const { holder, activities } = blocker;
  const heading =
    holder && activities && activities.length > 0
      ? "Initiative has an active agent and busy items"
      : holder
        ? "Initiative is currently locked"
        : "Member items have active agent runs";

  return (
    <div
      className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-xs text-amber-200"
      data-testid={selectors.feedback.dialogBlockerNotice}
    >
      <div className="mb-1 flex items-center gap-2 font-medium">
        <AlertTriangle className="h-4 w-4" />
        {heading}
      </div>

      {holder && (
        <p className="text-amber-300/80">
          Run <code className="text-amber-200">{holder.run_id}</code>{" "}
          ({holder.purpose}
          {holder.round_number ? `, round ${holder.round_number}` : ""}) holds the lock.
        </p>
      )}

      {activities && activities.length > 0 && (
        <ul className="mt-1 space-y-0.5 text-amber-300/80">
          {activities.map((a) => (
            <li key={a.ref}>
              <code className="text-amber-200">{a.ref}</code>
              {a.purpose ? ` — ${a.purpose}` : ""}
              {a.run_id ? ` (run ${a.run_id})` : ""}
            </li>
          ))}
        </ul>
      )}

      <label className="mt-2 flex cursor-pointer items-center gap-2">
        <input
          type="checkbox"
          checked={override}
          onChange={(e) => onOverrideChange(e.target.checked)}
          className="h-3.5 w-3.5 accent-amber-400"
          data-testid={selectors.feedback.dialogOverrideConfirm}
        />
        <span>I understand — cancel the active run and start a new one.</span>
      </label>
    </div>
  );
}

function computeNextSelectionSize(prev: Set<string>, toggledRef: string): number {
  return prev.has(toggledRef) ? prev.size - 1 : prev.size + 1;
}

// ---------------------------------------------------------------------------
// Draft persistence helpers
// ---------------------------------------------------------------------------

function safeLoadDraft(key: string): string {
  try {
    return localStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

function safeSaveDraft(key: string, value: string) {
  try {
    if (value.trim()) localStorage.setItem(key, value);
    else localStorage.removeItem(key);
  } catch {
    /* ignore */
  }
}

function safeClearDraft(key: string) {
  try {
    localStorage.removeItem(key);
  } catch {
    /* ignore */
  }
}

function safeClearKey(key: string) {
  try {
    localStorage.removeItem(key);
  } catch {
    /* ignore */
  }
}

function safeLoadStringSet(key: string, validRefs?: string[]): Set<string> {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return new Set();
    const out = new Set<string>();
    const validSet = validRefs ? new Set(validRefs) : null;
    for (const v of parsed) {
      if (typeof v !== "string") continue;
      // When validRefs is provided, drop any persisted refs that no
      // longer exist — the initiative may have evolved since the
      // draft was last saved.
      if (validSet && !validSet.has(v)) continue;
      out.add(v);
    }
    return out;
  } catch {
    return new Set();
  }
}

function safeSaveStringSet(key: string, set: Set<string>) {
  try {
    if (set.size === 0) localStorage.removeItem(key);
    else localStorage.setItem(key, JSON.stringify([...set]));
  } catch {
    /* ignore */
  }
}

const VALID_ACTION_KEYS: ReadonlySet<QuickActionKey> = new Set([
  "split_oversized",
  "merge_coupled",
  "identify_missing_work",
  "reconcile_with_code_drift",
  "reframe_scope",
]);

function safeLoadActionSet(key: string): Set<QuickActionKey> {
  try {
    const raw = localStorage.getItem(key);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return new Set();
    const out = new Set<QuickActionKey>();
    for (const v of parsed) {
      if (typeof v === "string" && VALID_ACTION_KEYS.has(v as QuickActionKey)) {
        out.add(v as QuickActionKey);
      }
    }
    return out;
  } catch {
    return new Set();
  }
}

function safeLoadBoolean(key: string, fallback: boolean): boolean {
  try {
    const raw = localStorage.getItem(key);
    if (raw == null) return fallback;
    return raw === "true";
  } catch {
    return fallback;
  }
}

function safeSaveBoolean(key: string, value: boolean) {
  try {
    localStorage.setItem(key, value ? "true" : "false");
  } catch {
    /* ignore */
  }
}
