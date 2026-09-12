import { useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Activity, FileText, Play } from "lucide-react";

import { Button } from "../../components/ui/button";
import { ErrorState, LoadingState } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { AuditOutcome, perfClient, type RunAuditResponse } from "../../api/perf";
import { severityChipClass } from "../fleet/severity";
import { useScenario } from "../perf/scenarioContextValue";
import { ScenarioPicker } from "../perf/ScenarioPicker";
import { TIER_LABEL_KEY, tierChipClass, tierKey } from "../perf/format";

/**
 * "Audit a scenario" workflow. Picks a scenario, shows its decided capture tier
 * and readiness findings (from ValidateReadiness), then runs a real perf audit
 * (RunAudit) and surfaces the produced trace + web-vitals artifacts plus the
 * audit outcome. The trace handle links straight into the trace analyzer.
 */
export function AuditWorkbench() {
  const { t } = useTranslation();
  const { scenario } = useScenario();
  const [lastAudit, setLastAudit] = useState<RunAuditResponse | null>(null);

  const readiness = useQuery({
    queryKey: ["readiness", scenario],
    queryFn: () => perfClient.validateReadiness({ scenario }),
  });

  const audit = useMutation({
    mutationFn: () => perfClient.runAudit({ scenario }),
    onSuccess: (res) => setLastAudit(res),
  });

  const findings = readiness.data?.assessment?.findings ?? [];

  return (
    <section
      data-testid={selectors.pages.audit}
      aria-labelledby="audit-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-3">
        <h2 id="audit-heading" className="text-2xl font-semibold">
          {t(strings.audit.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.audit.description)}</p>
        <div className="flex flex-wrap items-center gap-3">
          <ScenarioPicker />
          <Button
            data-testid={selectors.audit.runButton}
            onClick={() => audit.mutate()}
            disabled={audit.isPending}
          >
            <Play aria-hidden="true" className="me-1 h-4 w-4" />
            {audit.isPending ? t(strings.audit.running) : t(strings.audit.run)}
          </Button>
        </div>
      </header>

      {/* Tier panel */}
      <section
        data-testid={selectors.audit.tierPanel}
        aria-label={t(strings.audit.tier.title)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.audit.tier.title)}
        </h3>
        {readiness.isLoading && (
          <LoadingState
            className="mt-3"
            title={t(strings.audit.tier.loadingTitle)}
          />
        )}
        {readiness.error && (
          <ErrorState
            testId={selectors.audit.error}
            className="mt-3"
            title={t(strings.audit.tier.errorTitle)}
            message={errorMessage(readiness.error, t)}
            onRetry={() => void readiness.refetch()}
            retrying={readiness.isFetching}
          />
        )}
        {readiness.data && (
          <div className="mt-3 flex flex-wrap items-center gap-3 text-sm">
            <span
              data-testid={selectors.audit.tierBadge}
              className={[
                "rounded-control px-2 py-1 text-xs font-semibold uppercase",
                tierChipClass(tierKey(readiness.data.tier)),
              ].join(" ")}
            >
              {t(TIER_LABEL_KEY[tierKey(readiness.data.tier)])}
            </span>
            <span className="text-app-muted-foreground">
              {t(strings.audit.tier.framework)}:{" "}
              <span className="text-app-foreground">{readiness.data.uiFramework || "—"}</span>
            </span>
            <span className="text-app-muted-foreground">
              {t(strings.audit.tier.surfaces)}:{" "}
              <span className="text-app-foreground">
                {readiness.data.surfaces.join(", ") || "—"}
              </span>
            </span>
            {readiness.data.degradedReason && (
              <span className="text-app-warning">⚠ {readiness.data.degradedReason}</span>
            )}
          </div>
        )}
      </section>

      {/* Audit result */}
      {audit.error && (
        <p data-testid={selectors.audit.runError} className="text-app-danger">
          {errorMessage(audit.error, t)}
        </p>
      )}
      {lastAudit && <AuditResult result={lastAudit} />}

      {/* Readiness findings */}
      <section
        data-testid={selectors.audit.findings}
        aria-label={t(strings.audit.findings.title)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.audit.findings.title)}
        </h3>
        {findings.length === 0 ? (
          <p
            data-testid={selectors.audit.findingsEmpty}
            className="mt-3 text-app-muted-foreground"
          >
            {t(strings.audit.findings.empty)}
          </p>
        ) : (
          <ul className="mt-3 flex flex-col gap-2">
            {findings.map((f) => (
              <li
                key={`${f.code}:${f.location}`}
                data-testid={selectors.audit.findingRow({ code: f.code })}
                className="flex flex-col gap-1 rounded-control border border-app-border bg-app-surface-muted p-3 text-sm"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <span
                    className={[
                      "rounded-control px-1.5 py-0.5 text-xs font-semibold uppercase",
                      severityChipClass(f.severity),
                    ].join(" ")}
                  >
                    {f.severity}
                  </span>
                  <span className="font-medium text-app-foreground">{f.title || f.code}</span>
                  {f.autofixAvailable && (
                    <span className="rounded-control border border-app-info/40 bg-app-info/10 px-1.5 py-0.5 text-xs text-app-info">
                      {t(strings.audit.findings.autofixable)}
                    </span>
                  )}
                </div>
                {f.message && <p className="text-app-muted-foreground">{f.message}</p>}
                {f.location && (
                  <p className="font-mono text-xs text-app-muted-foreground">{f.location}</p>
                )}
              </li>
            ))}
          </ul>
        )}
        <p className="mt-3 text-xs text-app-muted-foreground">
          <Link to="/readiness" className="text-app-primary underline">
            {t(strings.audit.findings.readinessLink)}
          </Link>
        </p>
      </section>
    </section>
  );
}

/**
 * Static map of outcome key → typed translation key. Explicit `strings.*`
 * accessors (not computed) so the no-unused-keys lint rule sees each key used.
 */
const OUTCOME_LABEL_KEY = {
  captured: strings.audit.outcome.captured,
  skipped: strings.audit.outcome.skipped,
  failed: strings.audit.outcome.failed,
  unavailable: strings.audit.outcome.unavailable,
  unknown: strings.audit.outcome.unknown,
} as const;

function AuditResult({ result }: { result: RunAuditResponse }) {
  const { t } = useTranslation();
  const outcomeKey: keyof typeof OUTCOME_LABEL_KEY =
    result.outcome === AuditOutcome.CAPTURED
      ? "captured"
      : result.outcome === AuditOutcome.SKIPPED
        ? "skipped"
        : result.outcome === AuditOutcome.UNAVAILABLE
          ? "unavailable"
          : result.outcome === AuditOutcome.FAILED
            ? "failed"
            : "unknown";
  // UNAVAILABLE (no browser / BAS down) is a degraded environment, not a pass —
  // warning styling, distinct from a clean N/A skip and from a hard failure.
  const outcomeClass =
    outcomeKey === "captured"
      ? "border border-app-success/40 bg-app-success/10 text-app-success"
      : outcomeKey === "skipped" || outcomeKey === "unavailable"
        ? "border border-app-warning/40 bg-app-warning/10 text-app-warning"
        : "border border-app-danger/40 bg-app-danger/10 text-app-danger";

  return (
    <section
      data-testid={selectors.audit.result}
      aria-label={t(strings.audit.result.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center gap-3">
        <Activity aria-hidden="true" className="h-4 w-4 text-app-muted-foreground" />
        <h3 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.audit.result.title)}
        </h3>
        <span
          data-testid={selectors.audit.outcomeBadge}
          className={["rounded-control px-2 py-0.5 text-xs font-semibold uppercase", outcomeClass].join(
            " ",
          )}
        >
          {t(OUTCOME_LABEL_KEY[outcomeKey])}
        </span>
      </div>
      {result.reason && <p className="mt-2 text-sm text-app-warning">{result.reason}</p>}
      <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
        {result.traceArtifact && (
          <div>
            <dt className="text-xs uppercase text-app-muted-foreground">
              {t(strings.audit.result.trace)}
            </dt>
            <dd className="mt-1 flex items-center gap-2">
              <FileText aria-hidden="true" className="h-3.5 w-3.5 text-app-muted-foreground" />
              <Link
                data-testid={selectors.audit.analyzeTraceLink}
                to={`/trace?scenario=${encodeURIComponent(result.scenario)}&artifact=${encodeURIComponent(result.traceArtifact)}`}
                className="break-all font-mono text-xs text-app-primary underline"
              >
                {result.traceArtifact}
              </Link>
            </dd>
          </div>
        )}
        {result.webVitalsArtifact && (
          <div>
            <dt className="text-xs uppercase text-app-muted-foreground">
              {t(strings.audit.result.webVitals)}
            </dt>
            <dd className="mt-1 break-all font-mono text-xs text-app-foreground">
              {result.webVitalsArtifact}
            </dd>
          </div>
        )}
      </dl>
    </section>
  );
}
