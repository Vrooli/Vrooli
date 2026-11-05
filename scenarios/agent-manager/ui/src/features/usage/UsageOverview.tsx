import { useState } from "react";
import { useSearchParams } from "react-router-dom";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card } from "@vrooli/react-component-library/Card/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageAction, UsageWindowForecast, UsageWindowOutcome } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatPoints, MISSING } from "../../lib/format";
import { useUsageForecast, useUsageModes, useUsageRecommendations } from "./api";
import { formatAt, formatRate, usageStatus, windowKey, windowTitle } from "./format";
import { consequence, isAtRisk, pickWindow } from "./usageFocus";
import { UsageActions } from "./UsageActions";
import { effectOfAction, runnerForProvider, type EffectRange, type ModeRunner } from "./modes";
import { UsageModeBanners } from "./UsageModeBanner";
import { UsageModeSheet } from "./UsageModeSheet";
import { UsageBurners } from "./UsageBurners";
import { UsageWindows, WindowDetail } from "./UsageWindows";

function Figure({ label, value, tone }: { label: string; value: string; tone?: "danger" | "success" }) {
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`text-lg font-bold tabular-nums ${tone === "danger" ? "text-destructive" : tone === "success" ? "text-success" : ""}`}>{value}</div>
    </div>
  );
}

/** The page an alert opens: why this window matters, what it costs to fix, and whether the fix is enough; every projected figure is the API's outcome. */
function FocusBand({ forecast, siblings, outcome }: { forecast: UsageWindowForecast; siblings: UsageWindowForecast[]; outcome?: UsageWindowOutcome }) {
  const status = usageStatus(forecast.status);
  const atRisk = isAtRisk(forecast.status);
  const measured = forecast.burn.find((rate) => rate.horizon === "24h" && rate.available);
  const levers = outcome?.levers ?? 0;
  return (
    <section aria-label="Selected window" data-testid="usage-focus" className={`space-y-3 rounded-lg border p-4 ${atRisk ? "border-destructive/40 bg-destructive/5" : "border-border bg-card"}`}>
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge tone={status.tone}>{status.short}</StatusBadge>
        <h3 className="min-w-0 text-base font-semibold">
          {windowTitle(forecast, siblings)}{atRisk && forecast.projectedExhaustionAt ? ` runs out around ${formatAt(forecast.projectedExhaustionAt)}` : `: ${status.label.toLowerCase()}`}
        </h3>
      </div>
      <div className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3 lg:grid-cols-5">
        <Figure label="Used" value={`${Math.round(forecast.usedPercent)}%`} />
        <Figure label={measured ? "Burning now (24h)" : "Projected burn"} value={formatRate(measured?.percentPerHour ?? forecast.centralPercentPerHour)} />
        <Figure label="Can afford until reset" value={forecast.lastingPercentPerHour > 0 ? `≤ ${formatRate(forecast.lastingPercentPerHour)}` : MISSING} />
        {outcome?.cutNeeded !== undefined ? <Figure label="Needs a cut of" value={`${Math.round(outcome.cutNeeded * 100)}%`} tone="danger" /> : null}
        {atRisk && levers > 0 ? <Figure label={levers === 1 ? "The quota action below can still save" : `The ${levers} quota actions below can still save`} value={`≈ ${formatPoints(outcome?.recoverablePoints ?? 0)}`} tone="success" /> : null}
      </div>
      {atRisk ? <p className="text-sm">{consequence(forecast, outcome)}</p> : null}
    </section>
  );
}

/** Usage › Overview: the selected window's focus band, the actions that fix it, and where its quota went. */
export function UsageOverview() {
  const [params, setParams] = useSearchParams();
  const forecast = useUsageForecast();
  const modes = useUsageModes();
  const [sheet, setSheet] = useState<{ runner: ModeRunner | ""; windowName: string; windowMinutes?: number; effect?: EffectRange } | null>(null);
  const windows = forecast.data?.windows ?? [];
  const selected = pickWindow(windows, params.get("window") ?? "");
  const key = selected ? windowKey(selected) : "";
  const recommendations = useUsageRecommendations("", selected ? { provider: selected.provider, pool: selected.pool, window: selected.window } : undefined);
  // The query keeps the previous window's list while the new one loads: quota levers of another window are not actions for this one.
  const actions = (recommendations.data?.actions ?? []).filter((action) => action.source !== "lever" || windowKey(action) === key);
  const notes = [...(recommendations.data?.coverage ?? []), ...(recommendations.data?.scoring ?? [])];
  const select = (next: string) => setParams((current) => {
    const updated = new URLSearchParams(current);
    updated.set("window", next);
    return updated;
  }, { replace: true });
  const startFromAction = (action: UsageAction) => setSheet({ runner: runnerForProvider(action.provider), windowName: windowTitle(action, windows), windowMinutes: Number.parseInt(action.window, 10) || undefined, effect: effectOfAction(action) });
  const startFromWindow = () => selected && setSheet({
    runner: runnerForProvider(selected.provider),
    windowName: windowTitle(selected, windows),
    windowMinutes: Number(selected.windowMinutes),
    effect: recommendations.data?.outcome && recommendations.data.outcome.recoverablePoints > 0 ? { highPoints: recommendations.data.outcome.recoverablePoints } : undefined,
  });
  return (
    <div className="flex flex-col gap-4">
      <UsageModeBanners modes={modes.data} windows={windows} />
      {selected && (modes.data?.modes.length ?? 0) === 0 ? (
        <div className="flex justify-end"><Button variant="secondary" onClick={startFromWindow} data-testid="mode-open-sheet">Save usage for a while</Button></div>
      ) : null}
      {sheet ? (
        <UsageModeSheet
          open
          onClose={() => setSheet(null)}
          runner={sheet.runner}
          windowName={sheet.windowName}
          windowMinutes={sheet.windowMinutes}
          effect={sheet.effect}
          windows={windows}
          reach={modes.data?.reach ?? []}
          reachNotes={modes.data?.reachNotes ?? []}
        />
      ) : null}
      <UsageWindows windows={windows} loading={forecast.isLoading} error={forecast.error} onRetry={() => void forecast.refetch()} selected={key} onSelect={select} />
      {selected ? (
        <>
          <FocusBand forecast={selected} siblings={windows} outcome={recommendations.isPlaceholderData ? undefined : recommendations.data?.outcome} />
          <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
            <Card className="space-y-3 p-4 sm:p-5" data-testid="usage-section-recommendations">
              <div>
                <h3 className="text-base font-semibold">Do this first</h3>
                <p className="text-sm text-muted-foreground">quota levers for {windowTitle(selected, windows)}, tool friction, opportunities and operator corrections, ranked by score: a quota lever by its share of the window's points, anything else by √(share of its kind of harm × share of runs it touches)</p>
              </div>
              <UsageActions
                actions={actions}
                loading={recommendations.isLoading || recommendations.isPlaceholderData}
                error={recommendations.error}
                onRetry={() => void recommendations.refetch()}
                below={Number(recommendations.data?.belowMateriality ?? 0n)}
                materiality={recommendations.data?.materialityScore ?? 0}
                notes={notes}
                onStartMode={startFromAction}
              />
            </Card>
            <Card className="p-4 sm:p-5" data-testid="usage-section-burners"><UsageBurners selected={key} rules={recommendations.data?.rules ?? []} /></Card>
          </div>
          <WindowDetail forecast={selected} siblings={windows} />
        </>
      ) : null}
    </div>
  );
}
