/**
 * ProposalReview — per-mutation accept/reject UI for the current proposal
 * on an `awaiting_user` feedback round.
 *
 * Each mutation renders as a card showing its op, target, rationale, and a
 * human-readable summary of the payload. The user checks off mutations
 * to accept and clicks Apply; rejected mutations are the complement set.
 *
 * If the proposal has parse warnings (the agent produced malformed JSON
 * somewhere), they render at the top as non-blocking hints.
 *
 * The component is stateless about the apply result — the parent (FeedbackPanel)
 * owns the mutation, the error surface, and the post-apply outcome rendering.
 */

import { memo, useEffect, useMemo, useState } from "react";
import { CheckCircle2, CircleOff, XCircle, AlertTriangle, ArrowRight, GitBranch, GitMerge, Trash2, Edit3, Plus, MoveRight, Zap } from "lucide-react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type { BacklogStatus, ProposalMutation, ProposalOp, ProposalRevision, ApplyResult } from "../../types";
import { InitiativeDependencyGraph } from "./InitiativeDependencyGraph";
import { buildOverlay } from "./proposal-overlay";
import {
  PROPOSAL_CLEAR,
  PROPOSAL_DISMISS,
  PROPOSAL_RATIONALE_PLACEHOLDER,
  PROPOSAL_REJECT,
  PROPOSAL_SELECT_ALL,
} from "./feedback-strings";

interface PreviewItem {
  kind: string;
  name: string;
  title: string;
  status: BacklogStatus;
  dependsOn: string[];
  priority?: number;
  archivedAt?: string;
  missing?: boolean;
}

export interface ProposalReviewProps {
  revision: ProposalRevision;
  /** Called with the set of accepted mutation IDs and a rationale on Apply. */
  onAccept: (acceptedIds: string[], rationale: string) => void;
  /** Called on reject (no mutations applied, round becomes `rejected`). */
  onReject: (rationale: string) => void;
  /** Called on dismiss (round ends as `dismissed`). */
  onDismiss: (rationale: string) => void;
  /** True while the parent's mutation is in flight. */
  isPending?: boolean;
  /** Non-fatal error message to surface above the actions. */
  error?: string | null;
  /** Apply-result payload to render inline after a successful apply. */
  applyResult?: ApplyResult | null;
  /** When the round is already terminal the review renders read-only. */
  readOnly?: boolean;
  /**
   * Current items in the initiative. When provided, the review renders a
   * before/after dependency graph overlay above the mutation list so the
   * user sees what the accepted mutations would look like topologically.
   */
  previewItems?: PreviewItem[];
}

export const ProposalReview = memo(function ProposalReview({
  revision,
  onAccept,
  onReject,
  onDismiss,
  isPending,
  error,
  applyResult,
  readOnly,
  previewItems,
}: ProposalReviewProps) {
  const mutations = useMemo(
    () => revision.proposal.mutations ?? [],
    [revision.proposal.mutations],
  );
  const [selected, setSelected] = useState<Set<string>>(
    () => new Set(mutations.map((m) => m.id)),
  );
  const [rationale, setRationale] = useState("");

  // Re-seed selection whenever the revision identity changes (i.e., the
  // agent produced a new proposal revision). Keyed on the revision ID rather
  // than the array identity so we don't reset mid-interaction.
  useEffect(() => {
    setSelected(new Set(mutations.map((m) => m.id)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [revision.id]);

  const toggle = (id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const acceptedIds = useMemo(() => Array.from(selected), [selected]);
  const hasNone = mutations.length === 0;
  const overlay = useMemo(
    () => (previewItems && !hasNone ? buildOverlay(revision.proposal, { acceptedIds }) : null),
    [previewItems, hasNone, revision.proposal, acceptedIds],
  );
  const allSelected = mutations.length > 0 && acceptedIds.length === mutations.length;

  if (revision.proposal.form !== "mutation_list") {
    return (
      <div className="rounded-xl border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
        <div className="flex items-center gap-2 font-medium">
          <AlertTriangle className="h-4 w-4" />
          Full-graph proposals are not yet accepted directly.
        </div>
        <p className="mt-1 text-xs text-amber-200/80">
          Ask the agent to revise as a <code>mutation_list</code> — we normalize
          that shape into per-mutation accept/reject decisions on the server.
        </p>
      </div>
    );
  }

  return (
    <section
      className="space-y-3 rounded-xl border border-slate-700/60 bg-slate-900/40 p-3"
      data-testid={selectors.feedback.proposalReview}
    >
      <header className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h4 className="text-sm font-semibold text-slate-100">
            Proposed mutations ({mutations.length})
          </h4>
          {revision.proposal.rationale && (
            <p className="mt-1 text-xs text-slate-400">
              {revision.proposal.rationale}
            </p>
          )}
        </div>
        {!readOnly && !hasNone && (
          <button
            type="button"
            onClick={() =>
              setSelected(
                allSelected ? new Set<string>() : new Set(mutations.map((m) => m.id)),
              )
            }
            className="shrink-0 rounded-md border border-slate-700 px-2 py-1 text-[11px] font-medium text-slate-300 hover:border-slate-500"
          >
            {allSelected ? PROPOSAL_CLEAR : PROPOSAL_SELECT_ALL}
          </button>
        )}
      </header>

      {revision.parse_warnings && revision.parse_warnings.length > 0 && (
        <div className="rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-[11px] text-amber-200">
          <div className="mb-1 flex items-center gap-1.5 font-medium">
            <AlertTriangle className="h-3.5 w-3.5" />
            Parse warnings
          </div>
          <ul className="list-disc space-y-0.5 pl-4">
            {revision.parse_warnings.map((w, i) => (
              <li key={i}>{w}</li>
            ))}
          </ul>
        </div>
      )}

      {overlay && previewItems && (
        <div className="space-y-1">
          <p className="text-[11px] uppercase tracking-wider text-slate-500">
            Preview · dashed = added · faded = removed
          </p>
          <InitiativeDependencyGraph items={previewItems} overlay={overlay} />
        </div>
      )}

      {hasNone ? (
        <p className="rounded-md border border-slate-700/50 bg-slate-900/60 p-3 text-xs text-slate-400">
          Agent returned an empty mutation list. Ask for a revision if you
          expected changes.
        </p>
      ) : (
        <ul className="space-y-2">
          {mutations.map((m) => (
            <MutationCard
              key={m.id}
              mutation={m}
              checked={selected.has(m.id)}
              disabled={readOnly || isPending}
              applyOutcome={applyResult?.outcomes.find((o) => o.mutation_id === m.id) ?? null}
              onToggle={() => toggle(m.id)}
            />
          ))}
        </ul>
      )}

      {applyResult && (
        <div
          className={cn(
            "rounded-md border p-2 text-[11px]",
            applyResult.failed > 0
              ? "border-red-500/40 bg-red-500/10 text-red-200"
              : applyResult.skipped > 0
                ? "border-amber-500/30 bg-amber-500/10 text-amber-200"
                : "border-emerald-500/30 bg-emerald-500/10 text-emerald-200",
          )}
          data-testid={selectors.feedback.proposalApplySummary}
        >
          <strong>Applied:</strong> {applyResult.applied} ·{" "}
          <strong>Failed:</strong> {applyResult.failed} ·{" "}
          <strong>Skipped:</strong> {applyResult.skipped}
        </div>
      )}

      {error && (
        <p className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-300">
          {error}
        </p>
      )}

      {!readOnly && (
        <>
          <textarea
            placeholder={PROPOSAL_RATIONALE_PLACEHOLDER}
            value={rationale}
            onChange={(e) => setRationale(e.target.value)}
            disabled={isPending}
            rows={2}
            className="w-full resize-none rounded-md border border-slate-700 bg-slate-800/60 px-2 py-1.5 text-xs text-slate-200 placeholder-slate-500 outline-none focus:border-cyan-500/50"
          />

          <div className="flex flex-wrap justify-end gap-2">
            <button
              type="button"
              onClick={() => onDismiss(rationale.trim())}
              disabled={isPending}
              className="rounded-md border border-slate-700 px-3 py-1.5 text-xs font-medium text-slate-300 hover:border-slate-500 disabled:opacity-40"
              data-testid={selectors.feedback.proposalDismiss}
            >
              <CircleOff className="mr-1.5 inline h-3.5 w-3.5" />
              {PROPOSAL_DISMISS}
            </button>
            <button
              type="button"
              onClick={() => onReject(rationale.trim())}
              disabled={isPending}
              className="rounded-md border border-red-500/40 bg-red-500/10 px-3 py-1.5 text-xs font-medium text-red-200 hover:bg-red-500/20 disabled:opacity-40"
              data-testid={selectors.feedback.proposalReject}
            >
              <XCircle className="mr-1.5 inline h-3.5 w-3.5" />
              {PROPOSAL_REJECT}
            </button>
            <button
              type="button"
              onClick={() => onAccept(acceptedIds, rationale.trim())}
              disabled={isPending || acceptedIds.length === 0}
              className="rounded-md border border-emerald-500/40 bg-emerald-500/20 px-3 py-1.5 text-xs font-medium text-emerald-200 hover:bg-emerald-500/30 disabled:opacity-40"
              data-testid={selectors.feedback.proposalAccept}
            >
              <CheckCircle2 className="mr-1.5 inline h-3.5 w-3.5" />
              {acceptedIds.length === mutations.length
                ? `Accept all (${mutations.length})`
                : `Accept ${acceptedIds.length} of ${mutations.length}`}
            </button>
          </div>
        </>
      )}
    </section>
  );
});

// ---------------------------------------------------------------------------
// Per-mutation card
// ---------------------------------------------------------------------------

interface MutationCardProps {
  mutation: ProposalMutation;
  checked: boolean;
  disabled?: boolean;
  onToggle: () => void;
  applyOutcome: { applied: boolean; skipped?: boolean; error?: string } | null;
}

const OP_ICONS: Record<ProposalOp, typeof Plus> = {
  add_item: Plus,
  update_item: Edit3,
  change_status: GitBranch,
  change_priority: GitBranch,
  add_edge: ArrowRight,
  remove_edge: ArrowRight,
  move_initiative: MoveRight,
  archive_item: Trash2,
  interrupt_in_progress: Zap,
  split_item: GitBranch,
  merge_items: GitMerge,
};

const OP_LABELS: Record<ProposalOp, string> = {
  add_item: "Add item",
  update_item: "Update item",
  change_status: "Change status",
  change_priority: "Change priority",
  add_edge: "Add dependency edge",
  remove_edge: "Remove dependency edge",
  move_initiative: "Move to another initiative",
  archive_item: "Archive item",
  interrupt_in_progress: "Interrupt in-progress run",
  split_item: "Split item",
  merge_items: "Merge items",
};

function MutationCard({ mutation, checked, disabled, onToggle, applyOutcome }: MutationCardProps) {
  const Icon = OP_ICONS[mutation.op] ?? GitBranch;
  const label = OP_LABELS[mutation.op] ?? mutation.op;
  const destructive = mutation.op === "archive_item" || mutation.op === "interrupt_in_progress";

  // Outcome classification — applied | skipped | failed. Kept explicit
  // rather than deriving from `applied`/`error` at render time so the
  // badge and the card tint stay in sync.
  const outcomeKind = applyOutcome
    ? applyOutcome.applied
      ? "applied"
      : applyOutcome.skipped
        ? "skipped"
        : "failed"
    : null;

  return (
    <li
      className={cn(
        "rounded-lg border p-3 transition-colors",
        outcomeKind === "applied"
          ? "border-emerald-500/40 bg-emerald-500/10"
          : outcomeKind === "failed"
            ? "border-red-500/40 bg-red-500/10"
            : outcomeKind === "skipped"
              ? "border-amber-500/30 bg-amber-500/10 opacity-75"
              : destructive
                ? "border-amber-500/30 bg-amber-500/5"
                : "border-slate-700/60 bg-slate-950/50",
      )}
      data-testid={selectors.feedback.proposalMutation}
      data-mutation-id={mutation.id}
      data-mutation-op={mutation.op}
      data-outcome={outcomeKind ?? undefined}
    >
      <label className="flex cursor-pointer items-start gap-3">
        <input
          type="checkbox"
          checked={checked}
          onChange={onToggle}
          disabled={disabled || !!applyOutcome}
          className="mt-1 h-3.5 w-3.5 shrink-0 accent-cyan-500"
          data-testid={selectors.feedback.proposalMutationToggle}
        />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <Icon className={cn("h-3.5 w-3.5", destructive ? "text-amber-300" : "text-cyan-400")} />
            <span className="font-semibold text-slate-100">{label}</span>
            {mutation.target && (
              <code className="rounded bg-slate-800 px-1.5 py-0.5 text-[11px] text-slate-300">
                {mutation.target}
              </code>
            )}
            {outcomeKind && (
              <span
                className={cn(
                  "rounded-full px-2 py-0.5 text-[10px] font-medium",
                  outcomeKind === "applied" && "bg-emerald-500/20 text-emerald-200",
                  outcomeKind === "failed" && "bg-red-500/20 text-red-200",
                  outcomeKind === "skipped" && "bg-amber-500/20 text-amber-200",
                )}
              >
                {outcomeKind === "applied" && "Applied"}
                {outcomeKind === "failed" && "Failed"}
                {outcomeKind === "skipped" && "Skipped"}
              </span>
            )}
          </div>
          {mutation.rationale && (
            <p className="mt-1.5 text-[11px] leading-relaxed text-slate-400">
              {mutation.rationale}
            </p>
          )}
          <MutationSummary mutation={mutation} />
          {applyOutcome?.error && (
            <p className="mt-1.5 rounded border border-red-500/30 bg-red-500/10 px-2 py-1 text-[11px] text-red-200">
              {applyOutcome.error}
            </p>
          )}
        </div>
      </label>
    </li>
  );
}

function MutationSummary({ mutation }: { mutation: ProposalMutation }) {
  const bits: string[] = [];
  switch (mutation.op) {
    case "add_item":
      if (mutation.item) {
        bits.push(`${mutation.item.kind}/${mutation.item.name}`);
        if (mutation.item.title) bits.push(`"${mutation.item.title}"`);
        if (mutation.item.priority) bits.push(`P${mutation.item.priority}`);
        if (mutation.item.depends_on?.length) bits.push(`depends on ${mutation.item.depends_on.length} item(s)`);
      }
      break;
    case "update_item":
      if (mutation.patch) {
        const keys = Object.keys(mutation.patch);
        if (keys.length > 0) bits.push(`patch: ${keys.join(", ")}`);
      }
      break;
    case "change_status":
      if (mutation.status) bits.push(`→ ${mutation.status}`);
      break;
    case "change_priority":
      if (mutation.priority != null) bits.push(`→ P${mutation.priority}`);
      break;
    case "add_edge":
    case "remove_edge":
      if (mutation.from && mutation.to) bits.push(`${mutation.from} → ${mutation.to}`);
      break;
    case "move_initiative":
      bits.push(mutation.initiative ? `→ ${mutation.initiative}` : "→ (detach)");
      break;
    case "split_item":
      if (mutation.into) bits.push(`into ${mutation.into.length} item(s)`);
      break;
    case "merge_items":
      if (mutation.sources?.length && mutation.item) {
        bits.push(`${mutation.sources.join(" + ")} → ${mutation.item.kind}/${mutation.item.name}`);
      } else if (mutation.sources?.length) {
        bits.push(`merge ${mutation.sources.length} sources`);
      }
      break;
    default:
      break;
  }
  if (bits.length === 0) return null;
  return (
    <p className="mt-1 text-[11px] font-mono text-slate-500">{bits.join(" · ")}</p>
  );
}
