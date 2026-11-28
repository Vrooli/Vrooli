import { useState } from "react";
import { AlertTriangle, KeyRound, ShieldCheck } from "lucide-react";

import { Button } from "@vrooli/react-component-library/Button/2.2.14";
import { StatusChip } from "../../components/ui/status-chip";
import { useAcceptDefaultTargets, useCoverageReport, useRequestSensitiveReview } from "../../hooks/useCoverageReport";
import { sourceKindSlug } from "../../lib/status";
import { SOURCE_KIND_STRINGS } from "../../consts/statusStrings";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { cn } from "../../lib/utils";

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
  const reviewSensitive = useRequestSensitiveReview();
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
  const reacquirable = data.reacquirableTargets;
  const registeredReacquirable = data.registeredTargets.filter((target) => target.reacquirable);

  const complete = summary.defaultCoverageComplete;
  if (complete && !summary.hasSensitiveUnreviewed && !showComplete && !detailed) {
    return null;
  }

  const registerRecommended = () => {
    setMode("recommended");
    accept.mutate({ includeSensitive: false, dryRun: false }, { onSettled: () => setMode(null) });
  };
  const requestSensitiveDecision = () => {
    setMode("sensitive");
    reviewSensitive.mutate(undefined, { onSettled: () => setMode(null) });
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
          <p className="text-xs text-app-muted-foreground">
            {t(strings.coverage.independenceSummary, {
              // The server classifies each target once by distinct storage
              // device. Re-deriving it here from destination counts used to
              // mis-bucket a target whose second destination had failed but
              // whose first copy was still good.
              protected: summary.protectedTargetCount,
              single: summary.singleCopyTargetCount,
              none: summary.unprotectedTargetCount,
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
              shape="square"
            >
              {mode === "recommended"
                ? t(strings.coverage.registering)
                : t(strings.coverage.registerRecommended, { count: recommended.length })}
            </Button>
          </div>
        </div>
      )}

      {/* Sensitive — review-only, deliberate opt-in. */}
      {detailed && reacquirable.length > 0 && (
        <div className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface-muted p-3">
          <div>
            <p className="text-sm font-medium text-app-foreground">{t(strings.coverage.reacquirableTitle, { count: reacquirable.length })}</p>
            <p className="text-xs text-app-muted-foreground">{t(strings.coverage.reacquirableBody)}</p>
          </div>
          <ul data-testid={selectors.coverage.reacquirableList} className="flex flex-col gap-2">
            {reacquirable.map((s) => (
              <li key={s.id} className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-2">
                <span className="text-sm font-medium text-app-foreground">{s.owner}/{s.name}</span>
                <span className="truncate font-mono text-xs text-app-muted-foreground">{s.locator}</span>
                {(s.budgetMaxBytes || s.budgetMaxAge || s.retention) && (
                  <span className="text-xs text-app-muted-foreground">
                    {[s.budgetMaxBytes && `budget ≤ ${s.budgetMaxBytes}`, s.budgetMaxAge && `age ≤ ${s.budgetMaxAge}`, s.retention && `retention: ${s.retention}`].filter(Boolean).join(" · ")}
                  </span>
                )}
                {s.reacquisitionCommand && (
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.coverage.reacquisitionCommand)} <code>{s.reacquisitionCommand}</code>
                  </span>
                )}
                {s.reacquisitionEvidence && (
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.coverage.reacquisitionEvidence)} <code>{s.reacquisitionEvidence}</code>
                  </span>
                )}
                {s.reacquisitionNeedsOperator && s.reacquisitionAuthReason && (
                  <span className="text-xs text-app-warning">
                    {t(strings.coverage.reacquisitionAuthorization, { reason: s.reacquisitionAuthReason })}
                  </span>
                )}
                {(!s.reacquisitionCommand || !s.reacquisitionEvidence) && (
                  <span className="text-xs text-app-warning">
                    {t(strings.coverage.reacquisitionProofIncomplete)}
                  </span>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {detailed && registeredReacquirable.length > 0 && (
        <div className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface-muted p-3">
          <div>
            <p className="text-sm font-medium text-app-foreground">
              {t(strings.coverage.registeredReacquirableTitle, { count: registeredReacquirable.length })}
            </p>
            <p className="text-xs text-app-muted-foreground">
              {t(strings.coverage.registeredReacquirableBody)}
            </p>
          </div>
          <ul data-testid={selectors.coverage.registeredReacquirableList} className="flex flex-col gap-2">
            {registeredReacquirable.map((target) => (
              <li key={target.id} className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-2">
                <span className="text-sm font-medium text-app-foreground">{target.owner}/{target.name}</span>
                {target.reacquisitionCommand && (
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.coverage.reacquisitionCommand)} <code>{target.reacquisitionCommand}</code>
                  </span>
                )}
                {target.reacquisitionEvidence && (
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.coverage.reacquisitionEvidence)} <code>{target.reacquisitionEvidence}</code>
                  </span>
                )}
                {target.reacquisitionNeedsOperator && target.reacquisitionAuthReason && (
                  <span className="text-xs text-app-warning">
                    {t(strings.coverage.reacquisitionAuthorization, { reason: target.reacquisitionAuthReason })}
                  </span>
                )}
              </li>
            ))}
          </ul>
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
              variant="ghost"
              data-testid={selectors.coverage.requestSensitiveReview}
              disabled={reviewSensitive.isPending}
              onClick={requestSensitiveDecision}
              shape="square"
              className={cn("border border-app-border text-app-foreground hover:bg-app-surface-muted")}
            >
              {mode === "sensitive"
                ? t(strings.coverage.requestingSensitiveReview)
                : t(strings.coverage.requestSensitiveReview, { count: sensitive.length })}
            </Button>
            {reviewSensitive.data && (
              <p role="status" className="text-xs text-app-muted-foreground">
                {t(strings.coverage.sensitiveReviewPending, {
                  count: reviewSensitive.data.candidateCount,
                  id: reviewSensitive.data.askId,
                })}
              </p>
            )}
            {reviewSensitive.isError && (
              <p role="alert" className="text-xs text-app-warning">
                {t(strings.coverage.sensitiveReviewFailed)}
              </p>
            )}
          </div>
        </div>
      )}
    </section>
  );
}
