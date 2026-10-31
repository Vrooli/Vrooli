import { Progress } from "@vrooli/react-component-library/Progress/1";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import type { AgentUsage as Usage, AgentUsageWindow } from "./useAgentUsage";

const MAX_READING_AGE_MS = 6 * 60 * 60_000;

/** Only show a duration when there is exactly one valid provider window for it. */
function windowFor(windows: AgentUsageWindow[], minutes: number): AgentUsageWindow | undefined {
  const matches = windows.filter((window) => window.windowMinutes === minutes);
  if (matches.length !== 1) return undefined;
  const window = matches[0];
  return window && Number.isFinite(window.usedPercent) && window.usedPercent >= 0 && window.usedPercent <= 100 ? window : undefined;
}

export function AgentUsage({ usage, loading = false, agent, detailed = false }: { usage?: Usage | null; loading?: boolean; agent: string; detailed?: boolean }) {
  const { t, i18n } = useTranslation();
  if (loading && usage === undefined) return <span className="text-[11px] text-wc-text-muted">{t(strings.launcher.usageLoading)}</span>;
  if (!usage || usage.match === "unavailable") return <span className="text-[11px] text-wc-text-muted">{detailed && usage?.reason ? usage.reason : t(strings.launcher.usageUnavailable)}</span>;
  const observed = Date.parse(usage.observedAt);
  const stale = !Number.isFinite(observed) || observed > Date.now() + 60_000 || Date.now() - observed > MAX_READING_AGE_MS;
  const pools = Array.from(new Set(usage.windows.map((window) => window.pool))).sort();
  const fmtDate = (value: string) => {
    const time = Date.parse(value);
    return Number.isFinite(time) ? new Date(time).toLocaleString(i18n.resolvedLanguage) : "";
  };
  return <div className="w-full space-y-1.5" data-wc-rcl-theme data-testid={`launcher-usage-${usage.agentId}`}>
    <span className="block text-[11px] font-medium text-wc-text-muted">{usage.match === "likely" ? t(strings.launcher.usageEstimated) : t(strings.launcher.usageVerified)}{stale ? ` · ${t(strings.launcher.usageStale)}` : ""}</span>
    {pools.length > 1 && !detailed && <span className="block text-[11px] text-wc-text-muted">{t(strings.launcher.usageMultiplePools)}</span>}
    {(detailed ? pools : pools.length === 1 ? pools : []).map((pool) => {
      const inPool = usage.windows.filter((window) => window.pool === pool);
      const windows = [
        { label: t(strings.launcher.usageFiveHours), value: windowFor(inPool, 300) },
        { label: t(strings.launcher.usageWeekly), value: windowFor(inPool, 10080) },
      ];
      return <div key={pool} className="space-y-1">
        {detailed && pool && <span className="block text-xs font-medium text-wc-text-muted">{t(strings.launcher.usagePool, { pool })}</span>}
        {windows.map(({ label, value }) => {
      const resetTime = value?.resetsAt ? Date.parse(value.resetsAt) : NaN;
      const expired = Number.isFinite(resetTime) && resetTime <= Date.now();
      if (!value || stale || expired) return <span key={label} className="flex justify-between gap-2 text-[11px] text-wc-text-muted"><span>{label}</span><span>{t(expired || stale ? strings.launcher.usageStale : strings.launcher.usageNoReading)}</span></span>;
      const percent = new Intl.NumberFormat(i18n.resolvedLanguage, { maximumFractionDigits: 1 }).format(value.usedPercent);
      return <div key={label} className="space-y-0.5">
        <span className="flex justify-between gap-2 text-[11px] text-wc-text-muted"><span>{label}</span><span>{t(strings.launcher.usagePercent, { percent })}</span></span>
        <Progress value={value.usedPercent} size="sm" showValue={false} label={t(strings.launcher.usageBarLabel, { agent, window: label, percent })} style={{ minInlineSize: 0, width: "100%" }} />
        {detailed && value.resetsAt && fmtDate(value.resetsAt) && <span className="block text-[11px] text-wc-text-muted">{t(strings.launcher.usageResets, { time: fmtDate(value.resetsAt) })}</span>}
      </div>;
        })}
      </div>;
    })}
    {pools.length === 0 && <span className="block text-[11px] text-wc-text-muted">{t(strings.launcher.usageNoReading)}</span>}
    {detailed && <span className="block space-y-0.5 text-xs text-wc-text-muted">
      {usage.reason && <span className="block break-words">{usage.reason}</span>}
      {usage.source && <span className="block break-words">{t(strings.launcher.usageSource, { source: usage.source })}</span>}
      {fmtDate(usage.observedAt) && <span className="block">{t(strings.launcher.usageObserved, { time: fmtDate(usage.observedAt) })}</span>}
    </span>}
  </div>;
}
