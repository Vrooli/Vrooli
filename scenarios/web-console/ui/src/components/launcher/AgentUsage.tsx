/**
 * Usage rows, drawn from whatever windows the provider reports. `CardUsage` is
 * the compact form with reserved height; `UsageDetail` is Manage's full account
 * with absolute reset times.
 */
import { Progress } from "@vrooli/react-component-library/Progress/1";
import { Skeleton } from "@vrooli/react-component-library/Skeleton/1";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { formatAge, formatDateTime, formatPercent, formatShortDuration, formatTimeUntil } from "../../lib/localeFormat";
import type { AgentUsage as Usage, AgentUsageWindow } from "./useAgentUsage";
import { toneFor, usagePools, usageRows, windowIsStale, windowNames, type UsageRow, type UsageTone } from "./usageWindows";

const BAR_TONE: Record<UsageTone, "accent" | "warning" | "danger"> = { normal: "accent", hot: "warning", full: "danger" };
const VALUE_TEXT: Record<UsageTone, string> = { normal: "text-wc-text-primary", hot: "text-wc-warning", full: "text-wc-error-text" };
/** One row: a 16px line, 3px of air and a 4px bar. Fixed so an empty row weighs the same as a full one. */
const ROW_HEIGHT = "h-[27px]";

function useWindowText() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage;
  const names = (minutes: number) => windowNames(minutes, { week: t(strings.launcher.usageWindowWeek), weekly: t(strings.launcher.usageWindowWeekly) }, locale);
  return { t, locale, names };
}

/** The percentage bar. Its accessible name is the whole sentence, not the visible shorthand. */
function UsageBar({ percent, tone, label }: { percent: number; tone: UsageTone; label: string }) {
  return (
    <span data-wc-rcl-theme className="flex">
      <Progress value={percent} size="sm" tone={BAR_TONE[tone]} showValue={false} label={label} data-usage-tone={tone} style={{ minInlineSize: 0, width: "100%" }} />
    </span>
  );
}

/** `Week  87% · 4d`, with the full sentence as its accessible name. */
function CardRow({ row }: { row: UsageRow }) {
  const { t, locale, names } = useWindowText();
  const { window, stale, otherPools } = row;
  const name = names(window.windowMinutes);
  if (stale) {
    return (
      <span className={cn("flex items-start justify-between gap-1.5 whitespace-nowrap text-[11.5px] leading-4 text-wc-text-muted", ROW_HEIGHT)} data-usage-row="stale">
        <span>{name.short}</span>
        <span>{t(strings.launcher.usageStale)}</span>
      </span>
    );
  }
  const percent = formatPercent(window.usedPercent, locale);
  const remaining = window.resetsAt ? Date.parse(window.resetsAt) - Date.now() : NaN;
  const tone = toneFor(window.usedPercent);
  const sentence = [
    window.resetsAt && Number.isFinite(remaining)
      ? t(strings.launcher.usageRowResets, { window: name.long, percent, time: formatTimeUntil(window.resetsAt, locale) })
      : t(strings.launcher.usageRow, { window: name.long, percent }),
    otherPools > 0 ? t(strings.launcher.usageOtherPoolsLong, { count: otherPools }) : "",
  ].filter(Boolean).join(". ");
  return (
    <span data-usage-row="reading" className={cn("grid content-start gap-[3px]", ROW_HEIGHT)}>
      <span aria-hidden className="flex justify-between gap-1.5 whitespace-nowrap text-[11.5px] leading-4 tabular-nums text-wc-text-muted">
        <span>{name.short}{otherPools > 0 && <span className="ms-1 text-wc-text-faint">{t(strings.launcher.usageOtherPools, { count: otherPools })}</span>}</span>
        <span><b className={cn("font-semibold", VALUE_TEXT[tone])}>{t(strings.launcher.usagePercentShort, { percent })}</b>{Number.isFinite(remaining) && ` · ${formatShortDuration(remaining, locale)}`}</span>
      </span>
      <UsageBar percent={window.usedPercent} tone={tone} label={sentence} />
    </span>
  );
}

function BlankRows({ count, keyPrefix }: { count: number; keyPrefix: string }) {
  return Array.from({ length: count }, (_, index) => <span key={`${keyPrefix}-${String(index)}`} aria-hidden data-usage-row="blank" className={cn("block", ROW_HEIGHT)} />);
}

export interface CardUsageProps {
  usage?: Usage | null;
  loading: boolean;
  /** Rows this card reserves. The same for every card in the rail. */
  slots: number;
  /** Windows this harness says it will report, for placeholders. */
  declared: number;
}

/** A card's usage block. Its height is `slots` rows in every state. */
export function CardUsage({ usage, loading, slots, declared }: CardUsageProps) {
  const { t } = useTranslation();
  const rows = usageRows(usage);
  let content;
  let used;
  if (rows.length > 0) {
    content = rows.map((row) => <CardRow key={row.window.windowMinutes} row={row} />);
    used = rows.length;
  } else if (loading && usage === undefined && declared > 0) {
    used = Math.min(declared, slots);
    content = Array.from({ length: used }, (_, index) => (
      <span key={index} aria-hidden data-usage-row="placeholder" className={cn("grid content-start", ROW_HEIGHT)}>
        <Skeleton style={{ blockSize: 9, marginBlockStart: 3 }} />
        <Skeleton style={{ blockSize: 4, marginBlockStart: 4 }} />
      </span>
    ));
  } else {
    used = 1;
    content = (
      <span data-usage-row="message" className={cn("block text-[11.5px] leading-4 text-wc-text-faint", ROW_HEIGHT)}>
        {t(declared > 0 ? strings.launcher.usageUnavailable : strings.launcher.usageNotReported)}
      </span>
    );
  }
  return <span className="grid content-start gap-[7px]" data-testid="launcher-card-usage">{content}<BlankRows count={Math.max(0, slots - used)} keyPrefix="blank" /></span>;
}

function DetailRow({ window, usage }: { window: AgentUsageWindow; usage: Usage }) {
  const { t, locale, names } = useWindowText();
  const name = names(window.windowMinutes);
  if (windowIsStale(window, usage)) {
    return <p className="flex justify-between gap-2 text-[13px] text-wc-text-muted" data-usage-row="stale"><span>{name.long}</span><span>{t(strings.launcher.usageStale)}</span></p>;
  }
  const percent = formatPercent(window.usedPercent, locale);
  const tone = toneFor(window.usedPercent);
  const resets = window.resetsAt ? formatDateTime(window.resetsAt, locale) : "";
  return (
    <div className="grid gap-1">
      <p aria-hidden className="flex justify-between gap-2 text-[13px] tabular-nums text-wc-text-muted"><span>{name.long}</span><b className={cn("font-semibold", VALUE_TEXT[tone])}>{t(strings.launcher.usagePercent, { percent })}</b></p>
      <UsageBar percent={window.usedPercent} tone={tone} label={t(strings.launcher.usageRow, { window: name.long, percent })} />
      {resets && <p className="text-xs text-wc-text-muted">{t(strings.launcher.usageResets, { time: resets })}</p>}
    </div>
  );
}

/** Manage's usage section: every pool, absolute resets and one source sentence. */
export function UsageDetail({ usage, loading, declared, agentLabel }: { usage?: Usage | null; loading: boolean; declared: number; agentLabel: string }) {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage;
  if (usage === undefined && loading) return <p role="status" className="text-sm text-wc-text-muted">{t(strings.launcher.usageLoading)}</p>;
  if (!usage || usage.match === "unavailable") {
    if (declared === 0 && !usage?.reason) return <p className="text-sm text-wc-text-muted">{t(strings.launcher.usageNotReportedFor, { name: agentLabel })}</p>;
    return <p className="break-words text-sm text-wc-text-muted">{usage?.reason || t(strings.launcher.usageUnavailable)}</p>;
  }
  const pools = usagePools(usage);
  const age = formatAge(usage.observedAt, locale);
  return (
    <div className="grid gap-3" data-testid={`launcher-usage-${usage.agentId}`}>
      {pools.length === 0 && <p className="text-sm text-wc-text-muted">{t(strings.launcher.usageNoReading)}</p>}
      {pools.map(({ pool, windows }) => (
        <div key={pool} className="grid gap-2.5">
          {pools.length > 1 && pool && <p className="text-xs font-medium text-wc-text-secondary">{t(strings.launcher.usagePool, { pool })}</p>}
          {windows.map((window) => <DetailRow key={`${pool}-${String(window.windowMinutes)}`} window={window} usage={usage} />)}
        </div>
      ))}
      <p data-testid="launcher-usage-source" className="break-words text-xs leading-relaxed text-wc-text-muted">
        {t(usage.match === "likely" ? strings.launcher.usageSourceLikely : strings.launcher.usageSourceVerified, { source: usage.source || t(strings.launcher.usageSourceUnknown), age: age || t(strings.launcher.versionUnknown) })}
      </p>
    </div>
  );
}
