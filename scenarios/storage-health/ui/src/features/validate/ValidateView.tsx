import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { CheckCircle2, Wrench } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState, ErrorState, LoadingState, Skeleton } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import {
  storageClient,
  ValidationStatus,
  type FixResponse,
  type ValidateScenarioResponse,
} from "../../api/storage";

const STATUS_LABEL: Record<ValidationStatus, (typeof strings.validate.status)[keyof typeof strings.validate.status]> = {
  [ValidationStatus.UNSPECIFIED]: strings.validate.status.unspecified,
  [ValidationStatus.PASSED]: strings.validate.status.passed,
  [ValidationStatus.FAILED]: strings.validate.status.failed,
  [ValidationStatus.DEGRADED]: strings.validate.status.degraded,
  [ValidationStatus.ERROR]: strings.validate.status.error,
  [ValidationStatus.SKIPPED]: strings.validate.status.skipped,
};

const isOk = (status: ValidationStatus) =>
  status === ValidationStatus.PASSED || status === ValidationStatus.SKIPPED;

type Severity = "ERROR" | "WARNING" | "INFO";
const SEVERITY_RANK: Record<Severity, number> = { ERROR: 0, WARNING: 1, INFO: 2 };
const SEVERITY_LABEL: Record<Severity, (typeof strings.validate.severity)[keyof typeof strings.validate.severity]> = {
  ERROR: strings.validate.severity.error,
  WARNING: strings.validate.severity.warning,
  INFO: strings.validate.severity.info,
};

const normalizeSeverity = (raw: string): Severity => {
  if (raw.endsWith("ERROR")) return "ERROR";
  if (raw.endsWith("WARNING")) return "WARNING";
  return "INFO";
};

const severityCount = (counts: Record<string, number>, sev: string): number =>
  (counts[`SEVERITY_${sev}`] ?? 0) + (counts[`FINDING_SEVERITY_${sev}`] ?? 0);

/**
 * Per-scenario validation workflow. Honors `?scenario=` (pre-fills and runs),
 * renders the status header + findings (severity-sorted, remediation on errors),
 * and drives the Preview → Apply autofix flow, re-validating after a write.
 */
export function ValidateView() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const initial = searchParams.get("scenario") ?? "";

  const [scenarioInput, setScenarioInput] = useState(initial);
  const [submitted, setSubmitted] = useState(initial);
  const [previewing, setPreviewing] = useState(false);
  const [fixResult, setFixResult] = useState<FixResponse | null>(null);
  const [fixMessage, setFixMessage] = useState<string | null>(null);

  const query = useQuery<ValidateScenarioResponse>({
    queryKey: ["validate", submitted],
    queryFn: () => storageClient.validateScenario({ scenario: submitted }),
    enabled: submitted.length > 0,
  });

  // Re-run when an external deep-link changes the scenario (e.g. a fleet row).
  useEffect(() => {
    const next = searchParams.get("scenario") ?? "";
    if (next && next !== submitted) {
      setScenarioInput(next);
      setSubmitted(next);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const run = (scenario: string) => {
    const slug = scenario.trim();
    if (!slug) return;
    setSubmitted(slug);
    setFixResult(null);
    setFixMessage(null);
    const params = new URLSearchParams(searchParams);
    params.set("scenario", slug);
    setSearchParams(params, { replace: true });
  };

  const previewMutation = useMutation({
    mutationFn: () => storageClient.previewFix({ scenario: submitted }),
    onSuccess: (res) => {
      setPreviewing(true);
      setFixResult(res);
      setFixMessage(null);
    },
  });

  const applyMutation = useMutation({
    mutationFn: () => storageClient.applyFix({ scenario: submitted }),
    onSuccess: (res) => {
      setPreviewing(false);
      setFixResult(res);
      if (res.candidates.length === 0) {
        setFixMessage(t(strings.validate.fix.noop));
      } else {
        setFixMessage(t(strings.validate.fix.applied));
        void query.refetch();
      }
    },
  });

  const data = query.data;
  const assessment = data?.assessment;
  const findings = [...(assessment?.findings ?? [])].sort(
    (a, b) => SEVERITY_RANK[normalizeSeverity(a.severity)] - SEVERITY_RANK[normalizeSeverity(b.severity)],
  );
  const counts = assessment?.findingsBySeverity ?? {};
  const status = data?.status ?? ValidationStatus.UNSPECIFIED;

  return (
    <section
      data-testid={selectors.pages.validate}
      aria-labelledby="validate-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-1">
        <h2 id="validate-heading" className="text-2xl font-semibold">
          {t(strings.validate.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.validate.description)}</p>
      </header>

      <form
        className="flex flex-wrap items-end gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          run(scenarioInput);
        }}
      >
        <label className="flex flex-col gap-1">
          <span className="text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.validate.input.label)}
          </span>
          <Input
            data-testid={selectors.validate.input}
            value={scenarioInput}
            onChange={(e) => setScenarioInput(e.target.value)}
            placeholder={t(strings.validate.input.placeholder)}
            aria-label={t(strings.validate.input.label)}
            className="w-64"
          />
        </label>
        <Button data-testid={selectors.validate.runButton} type="submit" disabled={!scenarioInput.trim()}>
          {t(strings.validate.input.run)}
        </Button>
      </form>

      {submitted.length === 0 && (
        <EmptyState
          testId={selectors.validate.prompt}
          title={t(strings.validate.prompt.title)}
          message={t(strings.validate.prompt.message)}
        />
      )}

      {submitted.length > 0 && query.isLoading && (
        <LoadingState
          testId={selectors.validate.loading}
          title={t(strings.validate.loadingTitle)}
          skeleton={
            <div className="flex flex-col gap-3">
              <Skeleton className="h-16 w-full" />
              <Skeleton className="h-24 w-full" />
            </div>
          }
        />
      )}

      {query.error && (
        <ErrorState
          testId={selectors.validate.error}
          title={t(strings.validate.errorTitle)}
          message={errorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
          retrying={query.isFetching}
        />
      )}

      {data && !query.error && (
        <>
          <div
            data-testid={selectors.validate.resultHeader}
            className="flex flex-wrap items-center gap-3 rounded-panel border border-app-border bg-app-surface p-4"
          >
            <span
              data-testid={selectors.validate.statusPill}
              className={[
                "rounded-control px-2 py-0.5 text-sm font-semibold",
                isOk(status)
                  ? "bg-app-surface-muted text-app-foreground"
                  : "bg-app-danger/10 text-app-danger",
              ].join(" ")}
            >
              {t(STATUS_LABEL[status])}
            </span>
            <span className="text-sm text-app-muted-foreground">{data.scenario}</span>
            <span className="ms-auto flex flex-wrap gap-3 text-sm">
              <Count label={t(strings.validate.counts.errors)} value={severityCount(counts, "ERROR")} alert />
              <Count label={t(strings.validate.counts.warnings)} value={severityCount(counts, "WARNING")} />
              <Count label={t(strings.validate.counts.infos)} value={severityCount(counts, "INFO")} />
            </span>
          </div>

          {findings.length === 0 ? (
            <EmptyState
              testId={selectors.validate.clean}
              title={t(strings.validate.clean.title)}
              message={t(strings.validate.clean.message)}
              icon={CheckCircle2}
            />
          ) : (
            <>
            <h3 className="text-sm font-medium text-app-muted-foreground">
              {t(strings.validate.findings.title)}
            </h3>
            <ul data-testid={selectors.validate.findingsList} className="flex flex-col gap-3">
              {findings.map((f, i) => {
                const sev = normalizeSeverity(f.severity);
                return (
                  <li
                    key={`${f.code}-${i}`}
                    data-testid={selectors.validate.finding({ index: i })}
                    className={[
                      "rounded-panel border p-4",
                      sev === "ERROR"
                        ? "border-app-danger/40 bg-app-danger/5"
                        : "border-app-border bg-app-surface",
                    ].join(" ")}
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={[
                          "rounded-control px-1.5 py-0.5 text-xs font-semibold uppercase",
                          sev === "ERROR" ? "bg-app-danger/10 text-app-danger" : "bg-app-surface-muted text-app-foreground",
                        ].join(" ")}
                      >
                        {t(SEVERITY_LABEL[sev])}
                      </span>
                      <span className="text-xs text-app-muted-foreground">{f.code}</span>
                      <span className="font-medium text-app-foreground">{f.title}</span>
                      {f.autofixAvailable && (
                        <Button
                          data-testid={selectors.validate.autofix({ index: i })}
                          variant="outline"
                          size="sm"
                          className="ms-auto"
                          onClick={() => previewMutation.mutate()}
                          disabled={previewMutation.isPending}
                        >
                          <Wrench aria-hidden="true" className="me-1 h-4 w-4" />
                          {t(strings.validate.findings.autofix)}
                        </Button>
                      )}
                    </div>
                    {f.location && (
                      <p className="mt-1 text-xs text-app-muted-foreground">{f.location}</p>
                    )}
                    {(sev === "ERROR" || f.remediation) && f.remediation && (
                      <p className="mt-2 text-sm text-app-foreground">
                        <span className="font-medium">{t(strings.validate.findings.remediationLabel)} </span>
                        {f.remediation}
                      </p>
                    )}
                  </li>
                );
              })}
            </ul>
            </>
          )}

          {findings.length > 0 && (
            <div className="flex flex-wrap items-center gap-2">
              <Button
                data-testid={selectors.validate.previewButton}
                variant="outline"
                size="sm"
                onClick={() => previewMutation.mutate()}
                disabled={previewMutation.isPending}
              >
                {previewMutation.isPending ? t(strings.validate.fix.previewing) : t(strings.validate.fix.preview)}
              </Button>
              {previewing && (
                <Button
                  data-testid={selectors.validate.applyButton}
                  size="sm"
                  onClick={() => {
                    if (window.confirm(t(strings.validate.fix.confirm))) {
                      applyMutation.mutate();
                    }
                  }}
                  disabled={applyMutation.isPending}
                >
                  {applyMutation.isPending ? t(strings.validate.fix.applying) : t(strings.validate.fix.apply)}
                </Button>
              )}
            </div>
          )}

          {fixMessage && (
            <p data-testid={selectors.validate.fixMessage} className="text-sm text-app-foreground" role="status">
              {fixMessage}
            </p>
          )}

          {previewing && fixResult && fixResult.candidates.length > 0 && (
            <section
              data-testid={selectors.validate.candidates}
              aria-label={t(strings.validate.fix.candidatesTitle)}
              className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
            >
              <h3 className="text-sm font-medium text-app-muted-foreground">
                {t(strings.validate.fix.candidatesTitle)}
              </h3>
              {fixResult.candidates.map((c, i) => (
                <div key={`${c.ruleId}-${i}`} className="flex flex-col gap-1 text-sm">
                  <span className="font-medium text-app-foreground">{c.filePath}</span>
                  <span className="text-app-muted-foreground">{c.description}</span>
                  {c.after && (
                    <pre className="overflow-x-auto rounded-control bg-app-surface-muted p-2 text-xs">
                      {c.after}
                    </pre>
                  )}
                </div>
              ))}
            </section>
          )}
        </>
      )}
    </section>
  );
}

function Count({ label, value, alert }: { label: string; value: number; alert?: boolean }) {
  return (
    <span className="flex items-center gap-1">
      <span className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</span>
      <span
        className={[
          "font-semibold tabular-nums",
          alert && value > 0 ? "text-app-danger" : "text-app-foreground",
        ].join(" ")}
      >
        {value}
      </span>
    </span>
  );
}
