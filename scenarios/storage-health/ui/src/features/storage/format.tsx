import { Lock, Unlock } from "lucide-react";

import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Shared presentation helpers for storage-health surfaces. Status is always
 * conveyed by icon + text (never color alone) so the isolation badge and the
 * fitness meter stay axe-clean.
 */

/**
 * Isolation status badge. The single most important safety signal in this app:
 * 🔒 isolation-ready (ok) vs 🔓 ISOLATION-UNREADY (alert). Both icon and text
 * change so the meaning never depends on color.
 */
export function IsolationBadge({ ready }: { ready: boolean }) {
  const { t } = useTranslation();
  if (ready) {
    return (
      <span
        className="inline-flex items-center gap-1 rounded-control border border-app-border bg-app-surface-muted px-1.5 py-0.5 text-xs font-medium text-app-foreground"
        title={t(strings.isolation.readyLabel)}
      >
        <Lock aria-hidden="true" className="h-3 w-3" />
        {t(strings.isolation.ready)}
      </span>
    );
  }
  return (
    <span
      className="inline-flex items-center gap-1 rounded-control border border-app-danger/40 bg-app-danger/10 px-1.5 py-0.5 text-xs font-semibold uppercase text-app-danger"
      title={t(strings.isolation.unreadyLabel)}
    >
      <Unlock aria-hidden="true" className="h-3 w-3" />
      {t(strings.isolation.unready)}
    </span>
  );
}

/** Small chip list of storage engines for a scenario row. */
export function EngineChips({ engines }: { engines: string[] }) {
  if (engines.length === 0) return <span className="text-app-muted-foreground">—</span>;
  return (
    <span className="flex flex-wrap gap-1">
      {engines.map((engine) => (
        <span
          key={engine}
          className="rounded-control bg-app-surface-muted px-1.5 py-0.5 text-xs text-app-foreground"
        >
          {engine}
        </span>
      ))}
    </span>
  );
}

/**
 * A labelled 0–1 fitness meter. Renders the numeric value (e.g. "0.80") plus a
 * progress bar so the score is legible without relying on color.
 */
export function FitnessMeter({ score, label }: { score: number; label: string }) {
  const clamped = Math.max(0, Math.min(1, score));
  const pct = Math.round(clamped * 100);
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</span>
      <div
        role="meter"
        aria-valuemin={0}
        aria-valuemax={1}
        aria-valuenow={clamped}
        aria-label={label}
        className="h-2 w-24 overflow-hidden rounded-control bg-app-surface-muted"
      >
        <div className="h-full bg-app-primary" style={{ width: `${pct}%` }} />
      </div>
      <span className="text-sm font-semibold tabular-nums text-app-foreground">
        {clamped.toFixed(2)}
      </span>
    </div>
  );
}

