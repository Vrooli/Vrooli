import { useState } from "react";
import { Link } from "react-router-dom";
import { Bug } from "lucide-react";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Table } from "@vrooli/react-component-library/Table/1";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Toggle } from "@vrooli/react-component-library/Toggle/1";
import type { UsageFrictionProblem } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { formatCode, formatCount, formatDuration, formatPoints, formatTokens, subscriptionLabel } from "../../lib/format";
import { useFrictionActions, useUsageFriction } from "./api";
import { windowKey } from "./format";

const STATUSES = ["open", "reported", "dismissed"] as const;
type FrictionStatus = (typeof STATUSES)[number];

const BUTTON = "inline-flex whitespace-nowrap min-h-11 items-center justify-center rounded-md border border-border px-3 text-sm font-medium hover:bg-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50 sm:min-h-8";
const CELL = "max-md:!block max-md:!border-0 max-md:!px-0 max-md:!py-0.5";

function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="text-xl font-bold tabular-nums">{value}</div>
    </div>
  );
}

type Pending = "report" | "dismiss" | undefined;

/** The smallest saving the server ranks as its own action (usageattribution minPlanSaving). */
const MIN_PLAN_SAVING = 0.1;

/** The row's actions: one button each, confirmed inline (no dialog) and disabled while the call is out. */
function RowActions({ problem }: { problem: UsageFrictionProblem }) {
  const { report, dismiss } = useFrictionActions();
  const [confirming, setConfirming] = useState<Pending>();
  const busy = report.isPending || dismiss.isPending;
  const error = (report.error ?? dismiss.error) as Error | null;
  if (problem.status === "open" && problem.ownerKind === "agent" && !confirming) {
    return (
      <div className="flex gap-2">
        {/* A plan card exists only for a saving the actions list ranks (≥ 0.1 pts). */}
        {problem.estSavedPoints >= MIN_PLAN_SAVING ? (
          <Link to={`/usage?window=${encodeURIComponent(windowKey(problem))}#friction-${problem.fingerprint}`} className={`${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`}>Suggest a fix</Link>
        ) : null}
        <button type="button" onClick={() => setConfirming("dismiss")} className={BUTTON}>Dismiss</button>
      </div>
    );
  }
  if (problem.status === "reported") return <StatusBadge tone="success">Reported · {problem.bugId || "filed"}</StatusBadge>;
  if (problem.status === "dismissed") return <StatusBadge tone="neutral">Dismissed</StatusBadge>;
  const run = (action: "report" | "dismiss") => (action === "report" ? report : dismiss).mutate(problem.fingerprint, { onSettled: () => setConfirming(undefined) });
  if (confirming) {
    return (
      <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Confirm">
        <span className="text-sm">{confirming === "report" ? `File a bug with ${problem.owner || "its owner"}?` : "Dismiss this problem?"}</span>
        <button type="button" disabled={busy} onClick={() => run(confirming)} className={`${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`}>{busy ? "Working…" : "Confirm"}</button>
        <button type="button" disabled={busy} onClick={() => setConfirming(undefined)} className={BUTTON}>Cancel</button>
      </div>
    );
  }
  return (
    <div className="flex flex-col items-start gap-1 md:items-end">
      <div className="flex gap-2">
        <button type="button" onClick={() => setConfirming("report")} className={`${BUTTON} bg-primary text-primary-foreground hover:bg-primary/90`}>Report bug</button>
        <button type="button" onClick={() => setConfirming("dismiss")} className={BUTTON}>Dismiss</button>
      </div>
      {error ? <span role="alert" className="text-xs text-destructive">Could not complete: {error.message}</span> : null}
    </div>
  );
}

/** The description already says "Typically 13m 54s each time" until the server drops it: show the duration once. */
function showTypical(problem: UsageFrictionProblem): boolean {
  return problem.typicalWallClockMs >= 1000n && !/typically/i.test(problem.description);
}

/** A problem's cost is its redundant calls' own time (a stall's is its idle gap) and tokens, never the run's. */
function TimeLine({ problem }: { problem: UsageFrictionProblem }) {
  if (problem.wallClockMs <= 0n && problem.tokens <= 0n) return null;
  const time = problem.wallClockMs > 0n ? `${formatDuration(problem.wallClockMs)} ${problem.timeKind === "idle" ? "idle gap" : "redundant call time"}` : "";
  const tokens = problem.tokens > 0n ? `${formatTokens(problem.tokens)} tok` : "";
  return <div className="whitespace-nowrap text-muted-foreground" title={`${formatCount(problem.episodeTurns)} redundant calls`}>{[time, tokens].filter(Boolean).join(" · ")}</div>;
}

/** Column labels return inline on a phone, where the header row is hidden. */
function Label({ children }: { children: string }) {
  return <span className="mr-2 text-xs text-muted-foreground md:hidden">{children}</span>;
}

function ProblemRow({ problem }: { problem: UsageFrictionProblem }) {
  return (
    <tr data-testid="usage-friction-row" className="max-md:block max-md:border-b max-md:border-border max-md:py-2">
      <td className={CELL}>
        <b className="break-words">{problem.description || formatCode(problem.pattern)}</b>
        <div className="text-xs text-muted-foreground">{formatCode(problem.pattern)}{showTypical(problem) ? ` · typically ${formatDuration(problem.typicalWallClockMs)} each time` : ""}</div>
      </td>
      <td className={CELL}><Label>Owner</Label>{problem.ownerKind === "agent" ? problem.owner : <code className="break-words text-sm">{problem.owner}</code>}</td>
      <td className={`${CELL} whitespace-nowrap tabular-nums md:text-right`}><Label>Seen</Label>{formatCount(problem.episodes)}× · {formatCount(problem.runs)} {problem.runs === 1n ? "run" : "runs"}</td>
      <td className={`${CELL} tabular-nums md:text-right`}>
        <Label>Cost</Label>{problem.points > 0 ? <div className="whitespace-nowrap">≈ {formatPoints(problem.points)}</div> : null}
        <TimeLine problem={problem} />
      </td>
      <td className={`${CELL} md:text-right`}><RowActions problem={problem} /></td>
    </tr>
  );
}

/** Usage › Friction: recurring problems in plain words, each with an owner, a cost, and Report bug or Dismiss. */
export function UsageFriction() {
  const [status, setStatus] = useState<FrictionStatus>("open");
  const friction = useUsageFriction();
  const all = friction.data?.problems ?? [];
  const rows = all.filter((problem) => problem.status === status);
  // Totals cover open problems only, and points stay per subscription window:
  // a Claude point and a Codex point are not the same amount.
  const open = all.filter((problem) => problem.status === "open");
  const pointsByWindow = new Map<string, number>();
  for (const problem of open) {
    const label = subscriptionLabel(problem.provider, problem.window);
    pointsByWindow.set(label, (pointsByWindow.get(label) ?? 0) + problem.points);
  }
  const points = [...pointsByWindow].map(([label, value]) => `≈ ${formatPoints(value)} ${label}`).join(" · ") || `≈ ${formatPoints(0)}`;
  const wall = open.reduce((sum, problem) => (problem.timeKind === "redundant" ? sum + problem.wallClockMs : sum), 0n);
  const total = (friction.data?.open ?? 0n) + (friction.data?.reported ?? 0n) + (friction.data?.dismissed ?? 0n);
  return (
    <div className="flex flex-col gap-4" data-testid="usage-friction">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="grid grid-cols-2 gap-x-8 gap-y-3 sm:grid-cols-4">
          <Kpi label="Open friction this week" value={points} />
          <Kpi label="Redundant call time" value={formatDuration(wall)} />
          <Kpi label="Recurring problems" value={formatCount(total)} />
          <Kpi label="Reported to owners" value={formatCount(friction.data?.reported)} />
        </div>
        <ButtonGroup label="Status" collapse="never" className="flex-wrap">
          {STATUSES.map((value) => <Toggle key={value} size="lg" pressed={status === value} onPressedChange={() => setStatus(value)}>{formatCode(value)}</Toggle>)}
        </ButtonGroup>
      </div>
      {friction.data && friction.data.runsAnalysed < friction.data.runsTotal ? (
        <p role="status" className="text-sm text-muted-foreground" data-testid="usage-friction-progress">Re-analysing {formatCount(friction.data.runsAnalysed)} of {formatCount(friction.data.runsTotal)} runs; the numbers below fill in as it goes.</p>
      ) : null}
      <Card className="p-0">
        <DataState
          subject="Friction"
          loading={friction.isLoading}
          error={friction.error}
          isEmpty={rows.length === 0}
          onRetry={() => void friction.refetch()}
          empty={<EmptyState icon={<Bug aria-hidden="true" />} title={`No ${status} problems`} description="Recurring friction appears here once the same problem has cost several runs time or quota." />}
        >
          <Table>
            <table>
              <thead className="max-md:sr-only">
                <tr><th scope="col">What keeps going wrong</th><th scope="col">Owner</th><th scope="col" className="md:text-right">Seen</th><th scope="col" className="md:text-right">Cost</th><th scope="col"><span className="sr-only">Actions</span></th></tr>
              </thead>
              <tbody>{rows.map((problem) => <ProblemRow key={problem.fingerprint} problem={problem} />)}</tbody>
            </table>
          </Table>
        </DataState>
      </Card>
      <p className="text-sm text-muted-foreground">Report bug files the problem to a scenario or command owner through the report-bug path, with the evidence attached, and marks the row Reported. Problems owned by agent behaviour link to their recommendation instead. The fingerprint is detail, not the headline.</p>
    </div>
  );
}
