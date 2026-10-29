import { useSearchParams } from "react-router-dom";
import { Card } from "@vrooli/react-component-library/Card/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { UsageWindowForecast } from "@vrooli/proto-types/agent-manager/v1/measures/measures_pb";
import { formatPoints, MISSING } from "../../lib/format";
import { useUsageForecast, useUsageRecommendations } from "./api";
import { formatAt, formatRate, usageStatus, windowKey, windowTitle } from "./format";
import { focusFigures, isAtRisk, pickWindow } from "./usageFocus";
import { SHOWN_ACTIONS, UsageActions } from "./UsageActions";
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

/** The page an alert opens: why this window matters, what it costs to fix, and whether the fix is enough. */
function FocusBand({ forecast, siblings, saving, shown }: { forecast: UsageWindowForecast; siblings: UsageWindowForecast[]; saving: number; shown: number }) {
  const figures = focusFigures(forecast, saving);
  const status = usageStatus(forecast.status);
  const atRisk = isAtRisk(forecast.status);
  return (
    <section aria-label="Selected window" data-testid="usage-focus" className={`space-y-3 rounded-lg border p-4 ${atRisk ? "border-destructive/40 bg-destructive/5" : "border-border bg-card"}`}>
      <div className="flex flex-wrap items-center gap-2">
        <StatusBadge tone={status.tone}>{status.short}</StatusBadge>
        <h3 className="min-w-0 text-base font-semibold">
          {windowTitle(forecast, siblings)}{atRisk && forecast.projectedExhaustionAt ? ` runs out around ${formatAt(forecast.projectedExhaustionAt)}` : `: ${status.label.toLowerCase()}`}
        </h3>
      </div>
      <div className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-3 lg:grid-cols-5">
        <Figure label="Used" value={`${Math.round(figures.used)}%`} />
        <Figure label={figures.burnMeasured ? "Burning now (24h)" : "Projected burn"} value={formatRate(figures.burn)} />
        <Figure label="Can afford until reset" value={figures.affordable > 0 ? `≤ ${formatRate(figures.affordable)}` : MISSING} />
        {figures.cutNeeded !== undefined ? <Figure label="Needs a cut of" value={`${Math.round(figures.cutNeeded * 100)}%`} tone="danger" /> : null}
        {atRisk && shown > 0 ? <Figure label={shown === 1 ? "The session action below saves" : `The ${shown} session actions below save`} value={`≈ ${formatPoints(figures.saving)}`} tone="success" /> : null}
      </div>
      {atRisk ? <p className="text-sm">{figures.consequence}</p> : null}
    </section>
  );
}

/** Usage › Overview: the selected window's focus band, the actions that fix it, and where its quota went. */
export function UsageOverview() {
  const [params, setParams] = useSearchParams();
  const forecast = useUsageForecast();
  const windows = forecast.data?.windows ?? [];
  const selected = pickWindow(windows, params.get("window") ?? "");
  const key = selected ? windowKey(selected) : "";
  const recommendations = useUsageRecommendations("", selected ? { provider: selected.provider, pool: selected.pool, window: selected.window } : undefined);
  // The query keeps the previous window's groups while the new one loads; only this window's groups are actions for it.
  const groups = (recommendations.data?.groups ?? []).filter((group) => windowKey(group) === key);
  // Friction plans cut across sessions (the same tokens), so only session and run plans add up.
  const counted = groups.slice(0, SHOWN_ACTIONS).filter((group) => !group.crossCut && group.recoverablePoints > 0);
  const select = (next: string) => setParams((current) => {
    const updated = new URLSearchParams(current);
    updated.set("window", next);
    return updated;
  }, { replace: true });
  return (
    <div className="flex flex-col gap-4">
      <UsageWindows windows={windows} loading={forecast.isLoading} error={forecast.error} onRetry={() => void forecast.refetch()} selected={key} onSelect={select} />
      {selected ? (
        <>
          <FocusBand forecast={selected} siblings={windows} saving={counted.reduce((sum, group) => sum + group.recoverablePoints, 0)} shown={counted.length} />
          <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
            <Card className="space-y-3 p-4 sm:p-5" data-testid="usage-section-recommendations">
              <div>
                <h3 className="text-base font-semibold">Do this first</h3>
                <p className="text-sm text-muted-foreground">for {windowTitle(selected, windows)} · grouped by what to change</p>
              </div>
              <UsageActions
                groups={groups}
                loading={recommendations.isLoading || recommendations.isPlaceholderData}
                error={recommendations.error}
                onRetry={() => void recommendations.refetch()}
                foldedGroups={Number(recommendations.data?.foldedGroups ?? 0n)}
                foldedPoints={recommendations.data?.foldedPoints ?? 0}
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
