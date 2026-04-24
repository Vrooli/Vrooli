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
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Loader2, Paperclip, SendHorizontal, AlertTriangle } from "lucide-react";
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

const DRAFT_KEY_PREFIX = "swarm-initiative-feedback-draft:";
const MAX_VISIBLE_LINES = 8;
const LINE_HEIGHT_PX = 22;
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;

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

export function FeedbackDialog({ initiativeName, isOpen, onClose, onSubmitted }: FeedbackDialogProps) {
  const draftKey = `${DRAFT_KEY_PREFIX}${initiativeName}`;
  const [text, setText] = useState(() => safeLoadDraft(draftKey));
  const [type, setType] = useState<FeedbackRoundType>("feedback");
  const [override, setOverride] = useState(false);
  const [blocker, setBlocker] = useState<Blocker | null>(null);
  const [error, setError] = useState<string | null>(null);

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

  // Reset transient state whenever the dialog reopens.
  useEffect(() => {
    if (isOpen) {
      setError(null);
      setOverride(false);
      setBlocker(null);
      setText(safeLoadDraft(draftKey));
    }
  }, [draftKey, isOpen]);

  const mutation = useMutation({
    mutationFn: async () => {
      return feedbackService.start(initiativeName, {
        type,
        text: text.trim(),
        files: getFiles(),
        override,
      });
    },
    onSuccess: (round) => {
      clearAll();
      safeClearDraft(draftKey);
      setText("");
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
  const canSubmit =
    text.trim().length > 0 && !mutation.isPending && (!blocked || override);
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
    return "What's off? What should change? (Ctrl+Enter to submit)";
  }, [type, blocked, override]);

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
