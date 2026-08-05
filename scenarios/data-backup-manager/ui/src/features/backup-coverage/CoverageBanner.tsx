import { useState } from "react";
import { AlertTriangle, KeyRound, ShieldCheck } from "lucide-react";

import { Button } from "../../components/ui/button";
import { StatusChip } from "../../components/ui/status-chip";
import { useAcceptDefaultTargets, useCoverageReport } from "../../hooks/useCoverageReport";
import { sourceKindSlug } from "../../lib/status";
import { SOURCE_KIND_STRINGS } from "../../consts/statusStrings";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * CoverageBanner — the first-real-backup readiness surface. It reads the live
 * coverage report and, when non-sensitive recommended targets remain
 * unregistered, leads with a prominent "register recommended" action. Sensitive
 * (credential/token) suggestions are shown separately and only registered on a
 * deliberate, explicit click.
 *
 * `detailed` renders the full recommended + sensitive lists (Targets page);
 * the compact form (Overview / Plans) shows just the call to action. When
 * coverage is complete the compact form renders nothing unless `showComplete`.
 */
export function CoverageBanner({
  detailed = false,
  showComplete = false,
  reserveSpace = false,
}: {
  detailed?: boolean;
  showComplete?: boolean;
  reserveSpace?: boolean;
}) {
  const { t } = useTranslation();
  const report = useCoverageReport();
  const accept = useAcceptDefaultTargets();
  const [mode, setMode] = useState<"recommended" | "sensitive" | null>(null);

  const data = report.data;
  if (!data || !data.summary) {
    return reserveSpace ? (
      <section
        data-testid={selectors.coverage.banner}
        aria-busy="true"
        aria-labelledby="coverage-heading"
        className="flex min-h-36 flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
      >
        <div id="coverage-heading" className="flex flex-col gap-1">
          <h2 className="text-sm font-semibold text-app-foreground">
            {t(strings.coverage.title)}
          </h2>
          <p className="text-sm text-app-muted-foreground">
            {t(strings.coverage.liveDescription)}
          </p>
        </div>
      </section>
    ) : null;
  }
  const summary = data.summary;
  const recommended = data.recommendedTargets;
  const sensitive = data.sensitiveTargets;

  const complete = summary.defaultCoverageComplete;
  if (complete && !summary.hasSensitiveUnreviewed && !showComplete && !detailed) {
    return null;
  }

  const registerRecommended = () => {
    setMode("recommended");
    accept.mutate({ includeSensitive: false, dryRun: false }, { onSettled: () => setMode(null) });
  };
  const registerSensitive = () => {
    setMode("sensitive");
    accept.mutate({ includeSensitive: true, dryRun: false }, { onSettled: () => setMode(null) });
  };

  return (
    <section
      data-testid={selectors.coverage.banner}
      aria-labelledby="coverage-heading"
      className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div id="coverage-heading" className="flex items-start gap-3">
        {complete ? (
          <ShieldCheck aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-success" />
        ) : (
          <AlertTriangle aria-hidden="true" className="mt-0.5 h-5 w-5 shrink-0 text-app-warning" />
        )}
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="text-sm font-semibold text-app-foreground">{t(strings.coverage.title)}</h2>
          <p className="text-sm text-app-muted-foreground">
            {t(strings.coverage.liveDescription)}
          </p>
          <p className="max-w-2xl text-xs text-app-muted-foreground">
            {complete ? t(strings.coverage.complete) : t(strings.coverage.incompleteTitle)}
          </p>
          <p className="text-xs text-app-muted-foreground">
            {t(strings.coverage.summary, {
              registered: summary.registeredCount,
              recommended: summary.recommendedCount,
              sensitive: summary.sensitiveCount,
              planned: summary.plannedCount,
              verified: summary.verifiedCount,
            })}
          </p>
        </div>
      </div>

      {/* Recommended (non-sensitive) — the default action. */}
      {recommended.length > 0 && (
        <div className="flex flex-col gap-2">
          {detailed && (
            <p className="max-w-2xl text-xs text-app-foreground">
              {t(strings.coverage.incompleteBody, { count: recommended.length })}
            </p>
          )}
          {detailed && (
            <ul data-testid={selectors.coverage.recommendedList} className="flex flex-col gap-2">
              {recommended.map((s) => (
                <li
                  key={s.id}
                  className="flex flex-wrap items-center gap-2 rounded-panel border border-app-border bg-app-surface-muted p-2"
                >
                  <span className="truncate text-sm font-medium text-app-foreground">
                    {s.owner}/{s.name}
                  </span>
                  <StatusChip tone="info" labelKey={SOURCE_KIND_STRINGS[sourceKindSlug(s.sourceKind)]} />
                  <span className="truncate font-mono text-xs text-app-muted-foreground">{s.locator}</span>
                </li>
              ))}
            </ul>
          )}
          <div>
            <Button
              size="sm"
              data-testid={selectors.coverage.registerRecommended}
              disabled={accept.isPending}
              onClick={registerRecommended}
            >
              {mode === "recommended"
                ? t(strings.coverage.registering)
                : t(strings.coverage.registerRecommended, { count: recommended.length })}
            </Button>
          </div>
        </div>
      )}

      {/* Sensitive — review-only, deliberate opt-in. */}
      {sensitive.length > 0 && (
        <div className="flex flex-col gap-2 rounded-panel border border-app-warning/40 bg-app-warning/10 p-3">
          <div className="flex items-start gap-2">
            <KeyRound aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-app-warning" />
            <div className="flex flex-col gap-1">
              <p className="text-sm font-medium text-app-foreground">{t(strings.coverage.sensitiveTitle)}</p>
              {detailed && (
                <p className="text-xs text-app-muted-foreground">
                  {t(strings.coverage.sensitiveBody, { count: sensitive.length })}
                </p>
              )}
            </div>
          </div>
          {detailed && (
            <ul data-testid={selectors.coverage.sensitiveList} className="flex flex-col gap-2">
              {sensitive.map((s) => (
                <li
                  key={s.id}
                  className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-2"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="truncate text-sm font-medium text-app-foreground">
                      {s.owner}/{s.name}
                    </span>
                    <StatusChip tone="warning" labelKey={strings.discovery.sensitive} />
                  </div>
                  <span className="truncate font-mono text-xs text-app-muted-foreground">{s.locator}</span>
                  {s.warning && <p className="text-xs text-app-warning">{s.warning}</p>}
                </li>
              ))}
            </ul>
          )}
          <div>
            <Button
              size="sm"
              variant="outline"
              data-testid={selectors.coverage.registerSensitive}
              disabled={accept.isPending}
              onClick={registerSensitive}
            >
              {mode === "sensitive"
                ? t(strings.coverage.registering)
                : t(strings.coverage.registerSensitive, { count: sensitive.length })}
            </Button>
          </div>
        </div>
      )}
    </section>
  );
}
