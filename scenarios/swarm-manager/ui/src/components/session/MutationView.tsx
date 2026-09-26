/**
 * MutationView — renders one proposal mutation at full fidelity.
 *
 * Replaces a row that printed the op name, a `target` that is empty for every
 * creation op, and a rationale. An operator could not see the description they
 * were rewriting, the item they were creating, or the criteria they were
 * accepting. Each op now renders through the archetype that matches what it
 * does to an object; `mutation-archetypes.test.ts` fails when an op has no
 * archetype, so a new server op cannot ship blank.
 */
import { AlertTriangle, ArrowRight, FilePlus2, GitBranch, Merge, Minus, Plus, Trash2 } from "lucide-react";
import {
  archetypeFor,
  describePatch,
  headlineFor,
  isDestructiveOp,
  itemSpecRef,
  mutationSubject,
  type MutationBaseState,
  type PatchFieldChange,
} from "../../lib/mutation-archetypes";
import { buildLineDiff, diffStat } from "../../lib/word-diff";
import type { ProposalGoalMilestone, ProposalGoalSpec, ProposalItemSpec, ProposalMutation } from "../../types/proposal";

/** Long text collapses to this many characters until the operator expands it. */
const PREVIEW_CHARS = 260;
/** Acceptance criteria shown before the "show more" affordance. */
const CRITERIA_PREVIEW = 3;

export interface MutationViewProps {
  mutation: ProposalMutation;
  /** Current state of the target, when the caller could resolve it. */
  base?: MutationBaseState;
}

export function MutationView({ mutation, base }: MutationViewProps) {
  const subject = mutationSubject(mutation);
  const archetype = archetypeFor(mutation.op);
  const destructive = isDestructiveOp(mutation.op);

  return (
    <div className="flex flex-col gap-2" data-testid="mutation-view" data-op={mutation.op} data-archetype={archetype ?? "unknown"}>
      <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-sm">
        <span className={`font-medium ${destructive ? "text-rose-200" : "text-slate-100"}`}>{headlineFor(mutation.op)}</span>
        {destructive && <span className="rounded border border-rose-400/30 bg-rose-400/10 px-1.5 py-px text-[10px] uppercase tracking-wide text-rose-200">destructive</span>}
        {subject && <code className="min-w-0 break-all text-xs text-cyan-200">{subject}</code>}
      </div>

      <MutationBody mutation={mutation} base={base} />

      {mutation.rationale && (
        <p className="text-xs leading-5 text-slate-400">
          <span className="font-medium text-slate-300">Why: </span>{mutation.rationale}
        </p>
      )}
    </div>
  );
}

function MutationBody({ mutation, base }: MutationViewProps) {
  switch (archetypeFor(mutation.op)) {
    case "object_preview": return <ObjectPreview mutation={mutation} />;
    case "field_diff": return <FieldDiff mutation={mutation} base={base} />;
    case "scalar_transition": return <ScalarTransition mutation={mutation} base={base} />;
    case "edge_delta": return <EdgeDelta mutation={mutation} />;
    case "list_delta": return <ListDelta mutation={mutation} />;
    case "fan": return <FanOut mutation={mutation} />;
    case "destructive": return <Destructive mutation={mutation} base={base} />;
    case "scope_checklist": return <ScopeChecklist mutation={mutation} />;
    default:
      // Either an op this build predates, or a payload with no op at all. Say
      // so plainly rather than rendering an empty card that reads as
      // "no changes" — silence here would be indistinguishable from safety.
      return (
        <p className="rounded border border-amber-400/25 bg-amber-400/[0.07] p-2 text-xs text-amber-100">
          <AlertTriangle className="mr-1 inline h-3 w-3" aria-hidden />
          {mutation.op
            ? <>This build cannot display <code className="text-amber-50">{mutation.op}</code> in detail.</>
            : <>This mutation declares no operation, so its effect cannot be shown.</>}
          {" "}Review it with <code className="text-amber-50">swarm-manager proposals get</code> before applying.
        </p>
      );
  }
}

// ---------------------------------------------------------------------------
// Shared pieces
// ---------------------------------------------------------------------------

function Panel({ children }: { children: React.ReactNode }) {
  return <div className="rounded border border-slate-800 bg-slate-950/40 p-2.5">{children}</div>;
}

function MetaRow({ entries }: { entries: Array<[string, string | number | undefined]> }) {
  const present = entries.filter((entry): entry is [string, string | number] => entry[1] !== undefined && entry[1] !== "");
  if (present.length === 0) return null;
  return (
    <dl className="flex flex-wrap gap-x-4 gap-y-1 text-xs">
      {present.map(([label, value]) => (
        <div key={label} className="flex gap-1.5">
          <dt className="text-slate-500">{label}</dt>
          <dd className="text-slate-200 tabular-nums">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

function Tags({ tags }: { tags?: string[] }) {
  if (!tags?.length) return null;
  return (
    <div className="flex flex-wrap gap-1">
      {tags.map((tag) => <span key={tag} className="rounded bg-slate-800 px-1.5 py-px text-[10px] text-slate-300">{tag}</span>)}
    </div>
  );
}

/** Long text, clamped until expanded. Uses <details> so it works without JS state. */
function ExpandableText({ text, label = "Show full text" }: { text: string; label?: string }) {
  if (!text) return null;
  if (text.length <= PREVIEW_CHARS) return <p className="whitespace-pre-wrap text-xs leading-5 text-slate-300">{text}</p>;
  return (
    <details className="group">
      <summary className="cursor-pointer list-none">
        <span className="whitespace-pre-wrap text-xs leading-5 text-slate-300 group-open:hidden">{text.slice(0, PREVIEW_CHARS).trimEnd()}… </span>
        <span className="text-xs text-cyan-300 group-open:hidden">{label}</span>
        <span className="hidden text-xs text-cyan-300 group-open:inline">Collapse</span>
      </summary>
      <p className="mt-1 whitespace-pre-wrap text-xs leading-5 text-slate-300">{text}</p>
    </details>
  );
}

function CriteriaList({ title, entries }: { title: string; entries?: string[] }) {
  if (!entries?.length) return null;
  const head = entries.slice(0, CRITERIA_PREVIEW);
  const rest = entries.slice(CRITERIA_PREVIEW);
  return (
    <div className="rounded border border-slate-800 bg-slate-900/40 p-2">
      <p className="text-[11px] font-medium text-slate-300">{title} ({entries.length})</p>
      <ol className="mt-1 flex flex-col gap-1">
        {head.map((entry, index) => (
          <li key={entry} className="flex gap-1.5 text-[11px] leading-5 text-slate-400">
            <span className="shrink-0 tabular-nums text-slate-600">{index + 1}</span><span className="min-w-0">{entry}</span>
          </li>
        ))}
      </ol>
      {rest.length > 0 && (
        <details className="mt-1">
          <summary className="cursor-pointer list-none text-[11px] text-cyan-300">Show {rest.length} more</summary>
          <ol className="mt-1 flex flex-col gap-1">
            {rest.map((entry, index) => (
              <li key={entry} className="flex gap-1.5 text-[11px] leading-5 text-slate-400">
                <span className="shrink-0 tabular-nums text-slate-600">{CRITERIA_PREVIEW + index + 1}</span><span className="min-w-0">{entry}</span>
              </li>
            ))}
          </ol>
        </details>
      )}
    </div>
  );
}

/** `old → new`. An empty destination reads as a word, never as a blank. */
function Transition({ before, after, emptyAfterLabel = "cleared" }: { before?: string; after: string; emptyAfterLabel?: string }) {
  return (
    <div className="flex flex-wrap items-center gap-1.5 text-xs">
      {before !== undefined
        ? <span className="rounded bg-rose-400/15 px-1.5 py-0.5 font-mono text-rose-200">{before || "unset"}</span>
        : <span className="text-slate-500">current value unavailable</span>}
      <ArrowRight className="h-3 w-3 shrink-0 text-slate-600" aria-hidden />
      <span className="rounded bg-emerald-400/15 px-1.5 py-0.5 font-mono text-emerald-200">{after || emptyAfterLabel}</span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Archetype: object preview
// ---------------------------------------------------------------------------

/**
 * `suppressRef` avoids printing the same reference twice: the card header
 * already shows the mutation's subject, which for a single-object preview is
 * this very item. Children of a split still show theirs, because the header
 * carries the source ref instead.
 */
function ItemSpecCard({ spec, suppressRef = false }: { spec: ProposalItemSpec; suppressRef?: boolean }) {
  return (
    <Panel>
      <div className="flex flex-col gap-2">
        <p className="text-sm font-medium leading-5 text-slate-100">{spec.title || "Untitled item"}</p>
        {!suppressRef && <code className="break-all text-xs text-cyan-200">{itemSpecRef(spec)}</code>}
        <MetaRow entries={[["priority", spec.priority], ["effort", spec.effort], ["milestone", spec.milestone]]} />
        <Tags tags={spec.tags} />
        {spec.description && <div className="border-t border-slate-800 pt-2"><ExpandableText text={spec.description} /></div>}
        {spec.note && <p className="text-[11px] italic text-slate-400">Note: {spec.note}</p>}
        <CriteriaList title="Acceptance criteria" entries={spec.acceptance_allow} />
        <CriteriaList title="Must not" entries={spec.acceptance_deny} />
        {spec.depends_on?.length ? <p className="text-[11px] text-slate-400">Depends on: {spec.depends_on.join(", ")}</p> : null}
      </div>
    </Panel>
  );
}

function MilestoneCard({ milestone }: { milestone: ProposalGoalMilestone }) {
  return (
    <Panel>
      <div className="flex flex-col gap-2">
        <p className="text-sm font-medium leading-5 text-slate-100">{milestone.title || milestone.name}</p>
        <code className="break-all text-xs text-cyan-200">{milestone.name}</code>
        {milestone.description && <ExpandableText text={milestone.description} />}
        <CriteriaList title="Acceptance criteria" entries={milestone.acceptance_criteria} />
        {milestone.items?.length ? <p className="text-[11px] text-slate-400">{milestone.items.length} member item(s): {milestone.items.join(", ")}</p> : null}
        {milestone.depends_on?.length ? <p className="text-[11px] text-slate-400">Depends on: {milestone.depends_on.join(", ")}</p> : null}
      </div>
    </Panel>
  );
}

function GoalCard({ goal }: { goal: ProposalGoalSpec }) {
  return (
    <Panel>
      <div className="flex flex-col gap-2">
        <p className="text-sm font-medium leading-5 text-slate-100">{goal.title || goal.name}</p>
        <code className="break-all text-xs text-cyan-200">{goal.name}</code>
        <MetaRow entries={[["priority", goal.priority], ["milestones", goal.milestones?.length]]} />
        {goal.description && <ExpandableText text={goal.description} />}
        {goal.targets?.length ? <p className="text-[11px] text-slate-400">Targets: {goal.targets.join(", ")}</p> : null}
        {goal.milestones?.length ? (
          <div className="flex flex-col gap-1.5 border-t border-slate-800 pt-2">
            {goal.milestones.map((milestone) => <MilestoneCard key={milestone.name} milestone={milestone} />)}
          </div>
        ) : null}
      </div>
    </Panel>
  );
}

function ObjectPreview({ mutation }: { mutation: ProposalMutation }) {
  if (mutation.item) return <ItemSpecCard spec={mutation.item} suppressRef />;
  if (mutation.goal) return <GoalCard goal={mutation.goal} />;
  if (mutation.goal_milestone) return <MilestoneCard milestone={mutation.goal_milestone} />;
  return <EmptyPayload op={mutation.op} />;
}

function EmptyPayload({ op }: { op: string }) {
  return (
    <p className="rounded border border-amber-400/25 bg-amber-400/[0.07] p-2 text-xs text-amber-100">
      <AlertTriangle className="mr-1 inline h-3 w-3" aria-hidden />
      This <code className="text-amber-50">{op}</code> carries no payload. Applying it may do nothing.
    </p>
  );
}

// ---------------------------------------------------------------------------
// Archetype: field diff
// ---------------------------------------------------------------------------

function ProseDiff({ before, after }: { before: string; after: string }) {
  const rows = buildLineDiff(before, after);
  return (
    <div className="overflow-hidden rounded border border-slate-800 font-mono text-[11px] leading-5">
      {rows.map((row, rowIndex) => (
        <div
          key={rowIndex}
          className={`grid grid-cols-[1rem_1fr] gap-1 px-1.5 py-0.5 ${
            row.kind === "delete" ? "bg-rose-500/10" : row.kind === "insert" ? "bg-emerald-500/10" : ""
          }`}
        >
          <span aria-hidden className={row.kind === "delete" ? "text-rose-300" : row.kind === "insert" ? "text-emerald-300" : "text-slate-700"}>
            {row.kind === "delete" ? "−" : row.kind === "insert" ? "+" : " "}
          </span>
          <span className={`min-w-0 break-words ${row.kind === "context" ? "text-slate-500" : "text-slate-200"}`}>
            {row.segments.map((segment, segmentIndex) => (
              <span
                key={segmentIndex}
                className={segment.kind === "delete" ? "rounded-sm bg-rose-400/25" : segment.kind === "insert" ? "rounded-sm bg-emerald-400/25" : ""}
              >
                {segment.text}
              </span>
            ))}
          </span>
        </div>
      ))}
    </div>
  );
}

function FieldChangeRow({ change }: { change: PatchFieldChange }) {
  const stat = change.before !== undefined ? diffStat(change.before, change.after) : undefined;
  return (
    <div className="flex flex-col gap-1">
      <div className="flex flex-wrap items-baseline justify-between gap-x-2">
        <span className="font-mono text-[11px] font-semibold text-slate-200">{change.label}</span>
        {stat && (
          <span className="font-mono text-[11px] tabular-nums">
            <span className="text-rose-300">−{stat.removed.toLocaleString()}</span>{" "}
            <span className="text-emerald-300">+{stat.added.toLocaleString()}</span>
          </span>
        )}
      </div>
      {change.cleared && <p className="text-[11px] text-amber-200">Cleared — this field is emptied, not left alone.</p>}
      {change.presentation === "prose" && change.before !== undefined
        ? <ProseDiff before={change.before} after={change.after} />
        : change.before !== undefined
          ? <Transition before={change.before} after={change.after} />
          : (
            <div className="flex flex-col gap-1">
              <p className="text-[11px] text-slate-500">New value</p>
              <div className="rounded border border-emerald-400/20 bg-emerald-400/[0.06] p-1.5">
                <ExpandableText text={change.after || "(empty)"} />
              </div>
            </div>
          )}
    </div>
  );
}

function FieldDiff({ mutation, base }: MutationViewProps) {
  if (mutation.op === "update_milestone" && mutation.goal_milestone) {
    return <MilestoneCard milestone={mutation.goal_milestone} />;
  }
  const summary = describePatch(mutation.patch, base?.patch);
  if (summary.changed.length === 0) return <EmptyPayload op={mutation.op} />;

  return (
    <Panel>
      <div className="flex flex-col gap-3">
        <p className="text-[11px] text-slate-400">
          {summary.changed.length} field{summary.changed.length === 1 ? "" : "s"} change
          {base?.patch ? "" : " · current values unavailable, showing incoming only"}
        </p>
        {summary.changed.map((change) => <FieldChangeRow key={change.field} change={change} />)}
        {summary.unchanged.length > 0 && (
          <p className="border-t border-slate-800 pt-2 text-[11px] text-slate-500">
            Unchanged: {summary.unchanged.join(" · ")}
          </p>
        )}
      </div>
    </Panel>
  );
}

// ---------------------------------------------------------------------------
// Archetype: scalar transition
// ---------------------------------------------------------------------------

function ScalarTransition({ mutation, base }: MutationViewProps) {
  if (mutation.op === "change_status") {
    return <Transition before={base?.status} after={mutation.status ?? ""} emptyAfterLabel="unset" />;
  }
  if (mutation.op === "change_priority") {
    return <Transition before={base?.patch?.priority?.toString()} after={mutation.priority?.toString() ?? ""} emptyAfterLabel="unset" />;
  }
  // move_milestone: an empty destination detaches the item, which must read as
  // an action rather than a blank chip.
  return <Transition before={base?.milestone} after={mutation.milestone ?? ""} emptyAfterLabel="detached from milestone" />;
}

// ---------------------------------------------------------------------------
// Archetype: edge delta
// ---------------------------------------------------------------------------

function EdgeDelta({ mutation }: { mutation: ProposalMutation }) {
  const from = mutation.from || mutation.target || "";
  const to = mutation.to || "";
  const removing = mutation.op === "remove_edge";
  return (
    <Panel>
      <div className="flex flex-wrap items-center gap-1.5 text-xs">
        <GitBranch className={`h-3.5 w-3.5 shrink-0 ${removing ? "text-rose-300" : "text-emerald-300"}`} aria-hidden />
        <code className="break-all text-cyan-200">{from || "(unset)"}</code>
        <span className={removing ? "text-rose-300 line-through" : "text-slate-400"}>depends on</span>
        <code className="break-all text-cyan-200">{to || "(unset)"}</code>
      </div>
      {(!from || !to) && <p className="mt-1.5 text-[11px] text-amber-200">One endpoint is missing; this edge may not apply.</p>}
    </Panel>
  );
}

// ---------------------------------------------------------------------------
// Archetype: list delta
// ---------------------------------------------------------------------------

function ListDelta({ mutation }: { mutation: ProposalMutation }) {
  const removing = mutation.op === "unassign_milestone_items" || mutation.op === "remove_goal_target";
  const entries = mutation.items ?? mutation.targets ?? [];
  if (entries.length === 0) return <EmptyPayload op={mutation.op} />;
  const Icon = removing ? Minus : Plus;
  return (
    <Panel>
      <p className="text-[11px] text-slate-400">{removing ? "Removing" : "Adding"} {entries.length} entr{entries.length === 1 ? "y" : "ies"}</p>
      <ul className="mt-1 flex flex-col gap-0.5">
        {entries.map((entry) => (
          <li key={entry} className="flex items-baseline gap-1.5 text-xs">
            <Icon className={`h-3 w-3 shrink-0 self-center ${removing ? "text-rose-300" : "text-emerald-300"}`} aria-hidden />
            <code className="break-all text-cyan-200">{entry}</code>
          </li>
        ))}
      </ul>
    </Panel>
  );
}

// ---------------------------------------------------------------------------
// Archetype: fan out / fan in
// ---------------------------------------------------------------------------

function FanOut({ mutation }: { mutation: ProposalMutation }) {
  const splitting = mutation.op === "split_item";
  const sources = splitting ? [mutation.target ?? ""].filter(Boolean) : (mutation.sources ?? []);
  const children = splitting ? (mutation.into ?? []) : (mutation.item ? [mutation.item] : []);

  return (
    <div className="flex flex-col gap-2">
      <div className="rounded border border-rose-400/25 bg-rose-400/[0.06] p-2">
        <p className="flex items-center gap-1.5 text-[11px] font-medium text-rose-100">
          {splitting ? <GitBranch className="h-3 w-3" aria-hidden /> : <Merge className="h-3 w-3" aria-hidden />}
          {sources.length} source item{sources.length === 1 ? "" : "s"} will be archived
        </p>
        <ul className="mt-1 flex flex-col gap-0.5">
          {sources.map((source) => <li key={source}><code className="break-all text-[11px] text-rose-200">{source}</code></li>)}
        </ul>
        {!splitting && <p className="mt-1 text-[11px] text-rose-200/80">External edges are retargeted to the merged item; edges between sources are dropped.</p>}
      </div>
      <p className="flex items-center gap-1.5 text-[11px] text-slate-400">
        <FilePlus2 className="h-3 w-3 shrink-0 text-emerald-300" aria-hidden />
        {children.length} item{children.length === 1 ? "" : "s"} created
      </p>
      {children.length === 0
        ? <EmptyPayload op={mutation.op} />
        : children.map((child) => (
          // A merge's header already names the merged item; a split's names
          // the source, so its children keep their own refs.
          <ItemSpecCard key={itemSpecRef(child)} spec={child} suppressRef={!splitting} />
        ))}
      {splitting && <p className="text-[11px] text-amber-200">Dependents of the source are not retargeted automatically.</p>}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Archetype: destructive
// ---------------------------------------------------------------------------

const REVERSAL_HINT: Record<string, string> = {
  archive_item: "Reversible with the unarchive endpoint.",
  archive_milestone: "Reversible with the unarchive endpoint.",
  interrupt_in_progress: "Not reversible — the run is cancelled and must be restarted.",
  recreate_item: "The replacement preserves lineage; the original stays archived.",
  recreate_milestone: "Member items move to the successor milestone.",
};

function Destructive({ mutation, base }: MutationViewProps) {
  return (
    <div className="rounded border border-rose-400/25 bg-rose-400/[0.06] p-2.5">
      <p className="flex items-start gap-1.5 text-xs text-rose-100">
        <Trash2 className="mt-0.5 h-3.5 w-3.5 shrink-0" aria-hidden />
        <span>
          {base?.title ? <span className="font-medium">{base.title}</span> : <code className="break-all">{mutationSubject(mutation) || "target"}</code>}
          {mutation.detach_open ? " — open member items will be detached first." : ""}
        </span>
      </p>
      {mutation.op === "archive_milestone" && !mutation.detach_open && (
        <p className="mt-1.5 text-[11px] text-amber-200">Archiving fails if the milestone still has active member items.</p>
      )}
      <p className="mt-1.5 text-[11px] text-slate-400">{REVERSAL_HINT[mutation.op] ?? "Review carefully — this removes state."}</p>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Archetype: scope checklist
// ---------------------------------------------------------------------------

const SCOPE_LABELS: Record<string, string> = {
  review: "Review rounds and their evidence",
  handoff_executions: "Handoff execution history",
  plan_unbind: "Binding to the plan of record",
};

function ScopeChecklist({ mutation }: { mutation: ProposalMutation }) {
  const scopes = mutation.reset_scope ?? [];
  if (scopes.length === 0) return <EmptyPayload op={mutation.op} />;
  return (
    <Panel>
      <p className="text-[11px] text-slate-400">Removes {scopes.length} derived artifact group{scopes.length === 1 ? "" : "s"}. The item specification is kept.</p>
      <ul className="mt-1 flex flex-col gap-0.5">
        {scopes.map((scope) => (
          <li key={scope} className="flex items-baseline gap-1.5 text-xs text-slate-200">
            <Trash2 className="h-3 w-3 shrink-0 self-center text-rose-300" aria-hidden />
            {SCOPE_LABELS[scope] ?? scope}
          </li>
        ))}
      </ul>
    </Panel>
  );
}
