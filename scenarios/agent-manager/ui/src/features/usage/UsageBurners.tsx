import { useState } from "react";
import { Link } from "react-router-dom";
import { Activity } from "lucide-react";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1";
import { Card } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { Progress } from "@vrooli/react-component-library/Progress/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { Toggle } from "@vrooli/react-component-library/Toggle/1";
import type { UsageBurner, UsageCalibration, UsageWindowAttribution } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { DataState } from "../../components/feedback/DataState";
import { USAGE_GROUP_BY, useUsageBurners, type UsageGroupBy } from "./api";
import { burnerSourceLabel, formatPoints, formatServerText, formatShare, formatTokenCount, groupByLabel, windowKey, windowTitle } from "./format";

const GROUP_OPTIONS = USAGE_GROUP_BY.map((value) => ({ value, label: groupByLabel(value) }));

/** Run detail for a row: the row's run, or the largest run behind it. */
function burnerRunId(burner: UsageBurner, groupBy: string): string | undefined {
  return groupBy === "run" ? burner.key : burner.runIds[0];
}

function burnerMeta(burner: UsageBurner): string {
  const parts = [burnerSourceLabel(burner.source)];
  if (burner.model) parts.push(burner.model);
  else if (burner.runner) parts.push(burner.runner);
  if (burner.sessions > 1n) parts.push(`${burner.sessions} sessions`);
  if (burner.runs > 0n) parts.push(`${burner.runs} ${burner.runs === 1n ? "run" : "runs"}`);
  const modality = burner.modality;
  if (modality?.imagesIn) parts.push(`${modality.imagesIn} images in`);
  if (modality?.imagesOut) parts.push(`${modality.imagesOut} images out`);
  if (modality?.webSearches) parts.push(`${modality.webSearches} web searches`);
  return parts.join(" · ");
}


function BurnerRow({ burner, groupBy }: { burner: UsageBurner; groupBy: string }) {
  const runId = burnerRunId(burner, groupBy);
  const label = burner.label || burner.key;
  return (
    <li className="grid gap-1.5 border-b border-border py-3 last:border-b-0 sm:grid-cols-[minmax(0,2fr)_minmax(0,1.6fr)] sm:items-center sm:gap-4" data-testid="usage-burner">
      <div className="min-w-0">
        <div className="flex items-baseline justify-between gap-2">
          {runId ? (
            <Link to={`/runs/${encodeURIComponent(runId)}?tab=usage`} className="inline-flex min-h-11 min-w-0 items-center truncate font-medium text-primary underline-offset-2 hover:underline sm:min-h-8">{label}</Link>
          ) : (
            <span className="truncate font-medium">{label}</span>
          )}
          <span className="shrink-0 font-semibold tabular-nums sm:hidden">{formatShare(burner.share)}</span>
        </div>
        <p className="truncate text-xs text-muted-foreground">{burnerMeta(burner)}</p>
      </div>
      <div className="flex items-center gap-3">
        <Progress className="min-w-0 flex-1" size="sm" showValue={false} value={Math.max(0, burner.share * 100)} max={100} label={`${formatShare(burner.share)} of the window`} />
        <span className="shrink-0 text-sm font-semibold tabular-nums sm:min-w-28">
          {formatPoints(burner.points)}<span className="hidden sm:inline"> · {formatShare(burner.share)}</span>
        </span>
      </div>
    </li>
  );
}

function CalibrationDisclosure({ calibration }: { calibration: UsageCalibration }) {
  return (
    <details className="rounded-md border border-border px-3 py-2 text-sm">
      <summary className="flex min-h-8 cursor-pointer items-center font-medium">How this is estimated</summary>
      <div className="mt-2 space-y-2 text-muted-foreground">
        <p>
          Each time slice's observed increase is split across the sessions active in it by model-weighted tokens, then calibrated against the provider's readings
          ({calibration.method === "fitted" ? `fitted on ${calibration.slices} slices, R² ${calibration.rSquared.toFixed(2)}` : calibration.method === "scaled" ? "one scale on prior model ratios: too few slices to fit" : "no usage to calibrate"}).
          {calibration.offDiskPointsPerHour > 0 ? ` About ${calibration.offDiskPointsPerHour.toFixed(2)} pts/h is not explained by usage on this machine.` : ""}
        </p>
        {calibration.note ? <p>{formatServerText(calibration.note)}</p> : null}
        {calibration.weights.length > 0 ? (
          <ul className="space-y-1 font-mono text-xs">
            {calibration.weights.map((weight) => (
              <li key={`${weight.runner}/${weight.model}`}>
                {weight.runner} {weight.model}: {formatTokenCount(Math.round(weight.cachedInputTokensPerPoint))} cached-input tokens per point · input ×{weight.inputWeight.toFixed(1)} · output ×{weight.outputWeight.toFixed(1)} ({weight.source})
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </details>
  );
}

function WindowAttribution({ attribution, siblings }: { attribution: UsageWindowAttribution; siblings: UsageWindowAttribution[] }) {
  const groupBy = attribution.groupBy || "session";
  const pending = Number(attribution.pendingTokens?.total ?? 0n);
  return (
    <div className="space-y-3" data-testid="usage-attribution">
      <p className="text-sm text-muted-foreground">
        {windowTitle(attribution, siblings)}: share of the {formatPoints(attribution.observedIncreasePoints).replace("≈ ", "")} used this window, estimated from model-weighted tokens and calibrated to provider readings.
      </p>
      <ul className="m-0 list-none p-0">
        {attribution.burners.map((burner) => <BurnerRow key={burner.key} burner={burner} groupBy={groupBy} />)}
        {attribution.otherBurners > 0n ? (
          <li className="flex items-center gap-3 border-b border-border py-3 text-sm text-muted-foreground" data-testid="usage-burner-other">
            <span className="min-w-0 flex-1">Everything else ({attribution.otherBurners.toString()} more)</span>
            <span className="shrink-0 font-semibold tabular-nums">{formatPoints(attribution.otherPoints)}</span>
          </li>
        ) : null}
        <li className="flex items-center gap-3 py-3 text-sm text-muted-foreground" data-testid="usage-burner-unattributed">
          <span className="min-w-0 flex-1">Unattributed / off-disk (other machines, provider apps, uncaptured transcripts)</span>
          <span className="shrink-0 font-semibold tabular-nums">{formatPoints(attribution.unattributedPoints)}</span>
        </li>
      </ul>
      {attribution.meteredLines.length > 0 ? (
        <div className="space-y-1 text-sm" data-testid="usage-metered-lines">
          <p className="font-medium">Metered outside the token counters (not in the points above)</p>
          <ul className="m-0 list-none space-y-1 p-0 text-muted-foreground">
            {attribution.meteredLines.map((line) => (
              <li key={line.kind}>
                {groupByLabel(line.kind.replace(/_/g, "-"))}: {line.count.toString()} · {line.state === "known" ? line.amount || "charge recorded" : line.state === "estimated" ? `estimated ${line.amount}`.trim() : "charge unknown"}
                {line.note ? ` — ${line.note}` : ""}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {pending > 0 ? <p className="text-xs text-muted-foreground">{formatTokenCount(pending)} tokens were used after the newest reading and are not in any percent yet.</p> : null}
      {attribution.calibration ? <CalibrationDisclosure calibration={attribution.calibration} /> : null}
    </div>
  );
}

/** Shaped like burner rows: label and meta on the left, bar and points on the right. */
function BurnersSkeleton({ rows = 4 }: { rows?: number }) {
  return (
    <div className="space-y-3">
      <Skeleton className="h-4 w-3/4" />
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="grid gap-1.5 border-b border-border py-3 last:border-b-0 sm:grid-cols-[minmax(0,2fr)_minmax(0,1.6fr)] sm:items-center sm:gap-4">
          <div className="space-y-1.5">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-3 w-1/2" />
          </div>
          <Skeleton className="h-2 w-full" />
        </div>
      ))}
    </div>
  );
}

/** "What's using it": UsageBurners for every current window, with a group-by switch over the eight dimensions. */
export function UsageBurners() {
  const [groupBy, setGroupBy] = useState<UsageGroupBy>("session");
  const burners = useUsageBurners(groupBy);
  const windows = (burners.data?.windows ?? []).filter((window) => window.observedIncreasePoints > 0 || window.burners.length > 0);
  const select = (value: string) => setGroupBy(USAGE_GROUP_BY.includes(value as UsageGroupBy) ? (value as UsageGroupBy) : "session");
  return (
    <Card className={`space-y-3 p-4 transition-opacity sm:p-5 ${burners.isPlaceholderData ? "opacity-60" : ""}`} aria-busy={burners.isPlaceholderData}>
      <div className="flex flex-wrap items-center justify-end gap-2">
        <ButtonGroup label="Group by" collapse="never" className="hidden flex-wrap md:flex">
          {GROUP_OPTIONS.map((option) => (
            <Toggle key={option.value} size="sm" pressed={groupBy === option.value} onPressedChange={() => select(option.value)}>{option.label}</Toggle>
          ))}
        </ButtonGroup>
        <label className="flex items-center gap-2 text-sm text-muted-foreground md:hidden">
          By
          <Select aria-label="Group by" value={groupBy} options={GROUP_OPTIONS} onValueChange={select} />
        </label>
      </div>
      <DataState
        compact
        subject="Attribution"
        loading={burners.isLoading}
        error={burners.error}
        isEmpty={windows.length === 0}
        skeleton={<BurnersSkeleton />}
        onRetry={() => void burners.refetch()}
        empty={<EmptyState density="compact" icon={<Activity aria-hidden="true" />} title="Nothing to attribute yet" description="Attribution starts when a window has provider readings to split across sessions." />}
      >
        {windows.map((window) => <WindowAttribution key={windowKey(window)} attribution={window} siblings={windows} />)}
      </DataState>
    </Card>
  );
}
