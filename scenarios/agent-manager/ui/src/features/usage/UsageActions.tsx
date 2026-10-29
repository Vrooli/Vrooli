import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { Lightbulb } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageEvidenceRef, UsageLever, UsageRecommendationGroup } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatPoints } from "../../lib/format";
import { DataState } from "../../components/feedback/DataState";
import { evidenceText, formatSaving, formatServerText, formatShare, leverLabel, targetTitle } from "./format";

const BUTTON = "inline-flex min-h-11 items-center justify-center rounded-md border border-border px-3 text-sm font-medium hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:min-h-8";

/** A lever's action without the target it repeats: the card title already names the session. */
function leverText(lever: UsageLever, group: UsageRecommendationGroup): string {
  const label = group.targetLabel;
  const text = label ? lever.action.replace(` in ${label}`, "").replace(`${label} `, "").replace(label, "") : lever.action;
  return formatServerText(text.replace(/\s+/g, " ").replace(/ \(/, " (").trim());
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

function groupEvidence(group: UsageRecommendationGroup): UsageEvidenceRef[] {
  return group.levers.flatMap((lever) => lever.evidenceRefs).filter((ref, index, all) => all.findIndex((other) => other.kind === ref.kind && other.id === ref.id) === index);
}

function ActionCard({ group }: { group: UsageRecommendationGroup }) {
  const [showEvidence, setShowEvidence] = useState(false);
  const { shortId, title } = targetTitle(group.targetLabel || group.targetId);
  const evidence = groupEvidence(group);
  const firstRun = evidence.find((ref) => ref.kind === "run");
  const isFriction = group.crossCut;
  const nothingLeft = !isFriction && group.recoverablePoints <= 0;
  const runCount = evidence.filter((ref) => ref.kind === "run").length;
  return (
    <Card id={isFriction ? `friction-${group.targetId}` : undefined} className="flex scroll-mt-4 flex-col gap-3 bg-muted/20 p-3 sm:p-4" data-testid="usage-recommendation">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h4 className="break-words text-sm font-semibold sm:text-base">{title}</h4>
          <p className="font-mono text-xs text-muted-foreground">
            {isFriction ? "recurring friction" : `${group.targetKind} ${shortId || group.targetId.slice(0, 8)}`}{group.stillRunning && !isFriction ? " · still running" : ""}
          </p>
        </div>
        <div className="text-right">
          {nothingLeft ? <div className="text-sm font-semibold text-muted-foreground">Nothing left to save</div> : <div className="text-base font-bold tabular-nums text-success">≈ {formatSaving(isFriction ? group.combinedSavedPoints : group.recoverablePoints)}</div>}
          <div className="text-xs text-muted-foreground">{isFriction ? "also in the sessions above, not added to them" : nothingLeft ? "this target has stopped" : `recoverable before the reset${group.levers.length > 1 ? ", overlap removed" : ""}`}</div>
        </div>
      </div>
      <p className="flex flex-wrap gap-x-5 gap-y-1 text-xs text-muted-foreground">
        <span>Used so far <b className="tabular-nums text-foreground">{formatPoints(group.usedSoFarPoints)}</b></span>
        <span>At this rate <b className="tabular-nums text-foreground">≈ {formatPoints(group.atThisRatePoints)}</b>{group.atThisRatePoints > 100 ? " (more than the whole window)" : ""}</span>
        <span>{formatShare(group.shareOfWindowBurn)} of the window's burn</span>
      </p>
      <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
        {group.levers.map((lever, index) => (
          <li key={lever.rule} className="grid grid-cols-[auto_1fr_auto] items-start gap-x-2 text-sm">
            <span title={`${lever.confidence} confidence`}><StatusBadge tone="neutral">{leverLabel(lever.lever)}</StatusBadge></span>
            <span className="min-w-0 break-words">
              {leverText(lever, group)}
              {index > 0 ? <span className="text-muted-foreground"> · more after the larger {group.levers.length > 2 ? "levers" : "lever"}</span> : null}
            </span>
            <span className="font-semibold tabular-nums text-success">{nothingLeft ? "" : formatSaving(isFriction ? lever.marginalPoints : lever.recoverablePoints)}</span>
          </li>
        ))}
      </ul>
      {nothingLeft ? null : <p className="text-xs text-muted-foreground">Savings are what is still recoverable before the reset; points inside a description are what that activity used so far.</p>}
      <div className="flex flex-wrap gap-2">
        {isFriction ? <Link to="/usage?tab=friction" className={BUTTON}>See in Friction</Link> : null}
        {!isFriction && firstRun ? <Link to={`/runs/${encodeURIComponent(firstRun.id)}?tab=usage`} className={`${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`}>Open session</Link> : null}
        {evidence.length > 0 ? (
          <button type="button" aria-expanded={showEvidence} onClick={() => setShowEvidence((open) => !open)} className={BUTTON}>
            Show evidence{runCount > 0 ? ` (${runCount} ${runCount === 1 ? "run" : "runs"})` : ""}
          </button>
        ) : null}
      </div>
      {showEvidence ? (
        <div className="space-y-2 text-sm">
          {group.levers.map((lever) => <p key={lever.rule} className="text-muted-foreground">{formatServerText(lever.rationale)}</p>)}
          <ul className="m-0 max-h-72 list-none overflow-y-auto p-0">{evidence.map((ref) => <EvidenceRow key={`${ref.kind}/${ref.id}`} evidence={ref} />)}</ul>
        </div>
      ) : null}
    </Card>
  );
}

/** Cards for ranked groups. */
export function ActionGroups({ groups }: { groups: UsageRecommendationGroup[] }) {
  return (
    <div className="flex flex-col gap-3">
      {groups.map((group) => <ActionCard key={`${group.targetKind}/${group.targetId}`} group={group} />)}
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

/** How many actions are shown in full before the rest fold into "smaller actions". */
export const SHOWN_ACTIONS = 3;

interface UsageActionsProps {
  groups: UsageRecommendationGroup[];
  loading: boolean;
  error: unknown;
  onRetry: () => void;
  /** Groups the server folded away as near-zero savings. */
  foldedGroups: number;
  foldedPoints: number;
}

/** "Do this first": the largest actions for the selected window, the rest one click away. */
export function UsageActions({ groups, loading, error, onRetry, foldedGroups, foldedPoints }: UsageActionsProps) {
  const target = useLocation().hash.slice(1);
  const [showAll, setShowAll] = useState(false);
  const hidden = groups.slice(SHOWN_ACTIONS).some((group) => `friction-${group.targetId}` === target);
  useEffect(() => { if (target) document.getElementById(target)?.scrollIntoView({ block: "start" }); }, [target, groups.length, showAll]);
  const shown = groups.slice(0, SHOWN_ACTIONS);
  const rest = groups.slice(SHOWN_ACTIONS);
  const smaller = rest.length + foldedGroups;
  const smallerPoints = rest.reduce((sum, group) => sum + group.combinedSavedPoints, foldedPoints);
  return (
    <DataState
      compact
      subject="Recommendations"
      loading={loading}
      error={error}
      isEmpty={groups.length === 0}
      skeleton={<ActionsSkeleton />}
      onRetry={onRetry}
      empty={<EmptyState density="compact" icon={<Lightbulb aria-hidden="true" />} title="No actions for this window" description="Recommendations need provider readings and indexed usage for the window." />}
    >
      {groups.length === 0 ? <p className="pb-2 text-sm text-muted-foreground">No rule found a saving in this window.</p> : null}
      <ActionGroups groups={shown} />
      {smaller > 0 ? (
        <div className="mt-3 space-y-3">
          <p className="text-sm text-muted-foreground">
            + {smaller} smaller {smaller === 1 ? "action" : "actions"} for this window (≈ {formatSaving(smallerPoints)})
            {rest.length > 0 ? <button type="button" aria-expanded={showAll} onClick={() => setShowAll((open) => !open)} className="ml-2 inline-flex min-h-11 min-w-11 items-center justify-center text-primary hover:underline sm:min-h-8">{showAll ? "Hide" : "Show"}</button> : null}
          </p>
          {showAll || hidden ? <ActionGroups groups={rest} /> : null}
        </div>
      ) : null}
    </DataState>
  );
}
