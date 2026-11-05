import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageCompareResponse, UsageCompareSide, UsageModePeriod } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatCount, formatPoints, MISSING } from "../../lib/format";
import { useModeBeforeAfter } from "./api";
import { formatAt } from "./format";
import { ENOUGH_ACCEPTED, formatWholePercent, periodName } from "./evidence";

const VERDICT = {
  improved: { tone: "success", text: "Cost per accepted outcome fell" },
  regressed: { tone: "danger", text: "Cost per accepted outcome rose" },
  unchanged: { tone: "neutral", text: "No clear change" },
  insufficient: { tone: "warning", text: "Not enough accepted outcomes to compare" },
} as const;

/** The `usage compare` command that reproduces a mode's before and after. */
export function compareCommand(beforeFrom: string, start: string, end: string): string {
  const side = (from: string, to: string) => `'{"window":{"custom":{"from":"${from}","to":"${to}"}}}'`;
  return `agent-manager usage compare --left-filter-json ${side(beforeFrom, start)} --right-filter-json ${side(start, end)}`;
}

function SideFigures({ title, side }: { title: string; side?: UsageCompareSide }) {
  const accepted = side ? Number(side.acceptedUnits) : 0;
  const enough = accepted >= ENOUGH_ACCEPTED;
  return (
    <div className="min-w-0 rounded-md border border-border p-3" data-testid="before-after-side">
      <div className="text-xs text-muted-foreground">{title}</div>
      <div className="text-lg font-bold tabular-nums">{side && enough ? `${formatPoints(side.pointsPerAccepted)} per accepted` : MISSING}</div>
      <div className="text-xs text-muted-foreground">{side ? `${formatCount(side.runs)} runs, ${formatCount(side.acceptedUnits)} accepted, ${formatCount(side.unknownUnits)} unknown` : "not loaded"}</div>
    </div>
  );
}

function Result({ data, command }: { data: UsageCompareResponse; command: string }) {
  const verdict = VERDICT[data.verdict as keyof typeof VERDICT] ?? VERDICT.insufficient;
  const comparable = data.verdict === "improved" || data.verdict === "regressed" || data.verdict === "unchanged";
  return (
    <div className="flex flex-col gap-2" data-testid="before-after-result">
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge tone={verdict.tone}>{verdict.text}</StatusBadge>
        {comparable ? <span className="text-sm tabular-nums">{data.pointsPerAcceptedChange > 0 ? "+" : ""}{formatWholePercent(data.pointsPerAcceptedChange)} per accepted outcome</span> : null}
      </div>
      <div className="grid gap-2 sm:grid-cols-2">
        <SideFigures title="Before (same length)" side={data.left} />
        <SideFigures title="During the mode" side={data.right} />
      </div>
      {data.caveats.map((caveat) => <p key={caveat} className="m-0 text-xs text-muted-foreground">{caveat}</p>)}
      <p className="m-0 text-xs text-muted-foreground">Observational: the work before and during may differ. Reproduce it with:</p>
      <code className="block overflow-x-auto whitespace-pre-wrap break-all rounded-md bg-muted p-2 text-xs" data-testid="before-after-command">{command}</code>
    </div>
  );
}

function ModeRow({ period, now }: { period: UsageModePeriod; now: string }) {
  const [open, setOpen] = useState(false);
  const end = period.endedAt || period.expiresAt || now;
  const start = Date.parse(period.startedAt);
  const length = Date.parse(end) - start;
  const before = Number.isFinite(length) && length > 0 ? new Date(start - length).toISOString().replace(".000Z", "Z") : "";
  const query = useModeBeforeAfter(period.id, period.startedAt, end, open);
  return (
    <li className="flex flex-col gap-2 border-t border-border pt-3 first:border-t-0 first:pt-0" data-testid="before-after-mode">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="min-w-0 text-sm"><span className="font-medium">{periodName(period)}</span> <span className="text-muted-foreground">{formatAt(period.startedAt)} to {period.endedAt ? formatAt(period.endedAt) : period.expiresAt ? `${formatAt(period.expiresAt)} (still on)` : "still on"}</span></div>
        {open ? null : <Button variant="secondary" className="min-h-11 sm:min-h-8" onClick={() => setOpen(true)} data-testid="before-after-open">Compare before and after</Button>}
      </div>
      {open && query.isLoading ? <p role="status" className="m-0 text-sm text-muted-foreground">Comparing…</p> : null}
      {open && query.error ? <p role="alert" className="m-0 text-sm text-destructive">Couldn't compare: {query.error.message}</p> : null}
      {open && query.data ? <Result data={query.data} command={compareCommand(before, new Date(start).toISOString().replace(".000Z", "Z"), new Date(Date.parse(end)).toISOString().replace(".000Z", "Z"))} /> : null}
    </li>
  );
}

/** Each recorded mode, newest first, with an on-demand before and after (the compare call is not cheap). */
export function ModeBeforeAfter({ periods, now }: { periods: readonly UsageModePeriod[]; now: string }) {
  if (periods.length === 0) return <p className="m-0 text-sm text-muted-foreground" data-testid="before-after-none">No usage mode ran in this period, so there is no before and after.</p>;
  return (
    <ul className="m-0 flex list-none flex-col gap-3 p-0" data-testid="before-after">
      {[...periods].reverse().slice(0, 5).map((period) => <ModeRow key={period.id} period={period} now={now} />)}
    </ul>
  );
}
