import { Fragment, useState } from "react";
import { Link } from "react-router-dom";
import { Lightbulb } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageAction, UsageActionEffect, UsageEvidenceRef } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatCount, formatPoints, formatTokens } from "../../lib/format";
import { DataState } from "../../components/feedback/DataState";
import { FrictionDetail } from "./UsageFriction";
import { evidenceText, formatServerText } from "./format";
import { runnerForProvider } from "./modes";

const BUTTON = "inline-flex min-h-11 items-center justify-center rounded-md border border-border px-3 text-sm font-medium hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-8";

const SOURCE_LABELS: Record<string, string> = { lever: "Quota", tool: "Tool", opportunity: "Opportunity", intervention: "Operator" };

/** The score on its group's scale (a quota lever's share of its window's points, else √(share of its kind of harm × share of runs touched)); a range is unproven. */
export function scoreText(action: UsageAction): string {
  if (action.effect?.basis === "range") return "unproven";
  return `${(action.score * 100).toFixed(1)}%`;
}

/** The effect in the unit of its harm: "161 calls · observed", "≈ 3.2 pts · would have saved". */
export function effectText(effect: UsageActionEffect | undefined): string {
  if (!effect) return "";
  switch (effect.unit) {
    case "points":
      return effect.basis === "range" ? `0 to ${formatPoints(effect.upper)} · unproven` : `≈ ${formatPoints(effect.value)} · ${effect.basis}`;
    case "tokens":
      return `${formatTokens(BigInt(Math.round(effect.value)))} tokens re-read · ${effect.basis}`;
    case "ms":
      return `${formatCount(Math.round(effect.value / 60000))} min · ${effect.basis}`;
    default:
      return `${formatCount(Math.round(effect.value))} ${effect.unit} · ${effect.basis}`;
  }
}

/** One evidence row: its kind, then text that tells it apart from its siblings (label and short id). */
function EvidenceRow({ evidence }: { evidence: UsageEvidenceRef }) {
  const text = evidenceText(evidence);
  return (
    <li className="flex min-h-11 min-w-0 items-center gap-2 border-b border-border last:border-b-0 sm:min-h-8" data-testid="usage-evidence">
      <span className="w-16 shrink-0 text-xs text-muted-foreground">{evidence.kind}</span>
      {evidence.kind === "run" ? (
        <Link className="flex min-h-11 min-w-0 flex-1 items-center truncate text-primary underline-offset-2 hover:underline sm:min-h-8" to={`/runs/${encodeURIComponent(evidence.id)}?tab=usage`}>{text}</Link>
      ) : (
        <span className="min-w-0 flex-1 truncate" title={evidence.id}>{text}</span>
      )}
    </li>
  );
}

function ActionCard({ action, onStartMode }: { action: UsageAction; onStartMode?: (action: UsageAction) => void }) {
  const [open, setOpen] = useState(false);
  const firstRun = action.evidenceRefs.find((ref) => ref.kind === "run");
  return (
    <Card className="flex flex-col gap-2 bg-muted/20 p-3 sm:p-4" data-testid="usage-recommendation">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h4 className="break-words text-sm font-semibold sm:text-base">{action.title}</h4>
          <p className="text-xs text-muted-foreground">{`${action.ownerKind}: ${action.owner}`}</p>
        </div>
        <div className="text-right">
          <div className={`text-base font-bold tabular-nums ${action.effect?.basis === "range" ? "text-muted-foreground" : "text-success"}`}>{scoreText(action)}</div>
          <div className="text-xs text-muted-foreground">{effectText(action.effect)}</div>
          <div className="max-w-64 text-xs text-muted-foreground" data-testid="usage-action-share">{action.scoreBasis}</div>
        </div>
      </div>
      <div className="flex flex-wrap gap-1.5">
        <StatusBadge tone="neutral">{SOURCE_LABELS[action.source] ?? action.source}</StatusBadge>
        <StatusBadge tone="neutral">{action.kind}</StatusBadge>
        <span title="confidence"><StatusBadge tone={action.confidence === "high" ? "success" : "neutral"}>{action.confidence}</StatusBadge></span>
      </div>
      <p className="break-words text-sm">{formatServerText(action.action)}</p>
      <FrictionDetail rootCause={action.rootCause} subs={action.subs} guesses={action.guesses} />
      {action.recoverablePoints > 0 ? <p className="text-xs text-muted-foreground">≈ {formatPoints(action.recoverablePoints)} of it still recoverable before the window resets</p> : null}
      <div className="flex flex-wrap gap-2">
        {onStartMode && action.source === "lever" && runnerForProvider(action.provider) ? (
          <button type="button" onClick={() => onStartMode(action)} className={`${BUTTON} border-primary text-primary`} data-testid="usage-action-start-mode">Start a mode</button>
        ) : null}
        {firstRun ? <Link to={`/runs/${encodeURIComponent(firstRun.id)}?tab=usage`} className={BUTTON}>Open a run</Link> : null}
        <button type="button" aria-expanded={open} onClick={() => setOpen((value) => !value)} className={BUTTON}>
          {open ? "Hide evidence" : "Show evidence"}
        </button>
      </div>
      {open ? (
        <div className="space-y-2 text-sm" data-testid="usage-action-evidence">
          <p className="text-muted-foreground">{formatServerText(action.rationale)}</p>
          {action.drilldown.length > 0 ? (
            <ul className="m-0 list-none space-y-1 p-0">
              {action.drilldown.map((drill) => <li key={`${drill.kind}/${drill.id}`} className="break-words text-xs"><b>{drill.label}</b> · {drill.detail}</li>)}
            </ul>
          ) : null}
          {action.evidenceRefs.length > 0 ? <ul className="m-0 max-h-72 list-none overflow-y-auto p-0">{action.evidenceRefs.map((ref) => <EvidenceRow key={`${ref.kind}/${ref.id}`} evidence={ref} />)}</ul> : null}
          <p className="break-words text-xs text-muted-foreground">Verify: {action.verify}</p>
        </div>
      ) : null}
    </Card>
  );
}

/** The group the API ranks an action in: quota levers and friction actions score on different scales, so each group is headed with its scale. */
function actionGroup(action: UsageAction): string {
  if (action.ownerKind === "session") return "Live sessions · quota scale";
  return action.source === "lever" ? "Quota levers · share of the window's points" : "Friction actions · √(harm share × runs touched)";
}

/** Cards for ranked actions, headed per scoring group as the API orders them. */
export function ActionList({ actions, onStartMode }: { actions: UsageAction[]; onStartMode?: (action: UsageAction) => void }) {
  return (
    <div className="flex flex-col gap-3">
      {actions.map((action, index) => {
        const previous = actions[index - 1];
        return (
          <Fragment key={`${action.source}/${action.kind}/${action.owner}/${action.fingerprints.join(",")}`}>
            {!previous || actionGroup(previous) !== actionGroup(action) ? <h4 className="text-xs font-medium text-muted-foreground" data-testid="usage-action-group">{actionGroup(action)}</h4> : null}
            <ActionCard action={action} onStartMode={onStartMode} />
          </Fragment>
        );
      })}
    </div>
  );
}

function ActionsSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {Array.from({ length: 2 }, (_, index) => <Skeleton key={index} className="h-40 rounded-lg" />)}
    </div>
  );
}

/** How many actions are shown before the rest fold behind "Show". */
const SHOWN_ACTIONS = 5;

interface UsageActionsProps {
  actions: UsageAction[];
  loading: boolean;
  error: unknown;
  onRetry: () => void;
  /** Actions the server counted under the materiality floor: their score is under materiality. */
  below: number;
  materiality: number;
  /** What each source could see, and how effects are scored. */
  notes: string[];
  /** Quota-lever cards offer to start a usage mode when this is given. */
  onStartMode?: (action: UsageAction) => void;
}

/** "Do this first": the ranked owner-level actions, the rest one click away, with the coverage that bounds the list. */
export function UsageActions({ actions, loading, error, onRetry, below, materiality, notes, onStartMode }: UsageActionsProps) {
  const [showAll, setShowAll] = useState(false);
  const rest = actions.slice(SHOWN_ACTIONS);
  return (
    <DataState
      compact
      subject="Recommendations"
      loading={loading}
      error={error}
      isEmpty={actions.length === 0 && notes.length === 0}
      skeleton={<ActionsSkeleton />}
      onRetry={onRetry}
      empty={<EmptyState density="compact" icon={<Lightbulb aria-hidden="true" />} title="No actions" description="Recommendations need provider readings, indexed usage and projected tool calls." />}
    >
      {actions.length === 0 ? <p className="pb-2 text-sm text-muted-foreground">No action is above the materiality floor; the coverage below says what was seen.</p> : null}
      <ActionList actions={actions.slice(0, SHOWN_ACTIONS)} onStartMode={onStartMode} />
      {rest.length > 0 || below > 0 ? (
        <div className="mt-3 space-y-3">
          <p className="text-sm text-muted-foreground">
            {rest.length > 0 ? `+ ${rest.length} more ${rest.length === 1 ? "action" : "actions"}` : ""}
            {rest.length > 0 && below > 0 ? " · " : ""}
            {below > 0 ? `${below} scoring under ${Math.round(materiality * 100)}%` : ""}
            {rest.length > 0 ? <button type="button" aria-expanded={showAll} onClick={() => setShowAll((value) => !value)} className="ml-2 inline-flex min-h-11 min-w-11 items-center justify-center text-primary hover:underline sm:min-h-8">{showAll ? "Hide" : "Show"}</button> : null}
          </p>
          {showAll ? <ActionList actions={rest} onStartMode={onStartMode} /> : null}
        </div>
      ) : null}
      {notes.length > 0 ? (
        <details className="mt-3 text-xs text-muted-foreground" data-testid="usage-recommendation-notes">
          <summary className="inline-flex min-h-11 cursor-pointer items-center sm:min-h-8">Coverage and scoring</summary>
          <ul className="m-0 list-disc space-y-1 pl-4">{notes.map((note) => <li key={note}>{formatServerText(note)}</li>)}</ul>
        </details>
      ) : null}
    </DataState>
  );
}
