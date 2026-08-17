/** @vrooliComponentSource react-component-library:StatusBadge */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CheckCircle2,
  ChevronRight,
  CircleAlert,
  FileCheck2,
  Play,
  ShieldCheck,
} from "lucide-react";
import { Link, useSearchParams } from "react-router-dom";

import {
  getComponentTestReport,
  listComponentTestReports,
  runComponentTest,
  type ComponentTestReport,
  type ComponentTestResult,
} from "../../api/componentTests";
import { Button } from "../../components/Button";
import { EmptyState } from "../../components/EmptyState";
import { StatusBadge } from "../../components/StatusBadge";
import { assetSearchForTab } from "../../routes";

type VerdictTone = "success" | "danger" | "warning" | "neutral";

function tone(verdict: string): VerdictTone {
  return verdict === "passed"
    ? "success"
    : verdict === "failed"
      ? "danger"
      : verdict === "blocked"
        ? "warning"
        : "neutral";
}

function verdictLabel(verdict: string) {
  return verdict === "passed"
    ? "Passed"
    : verdict === "failed"
      ? "Needs attention"
      : verdict === "blocked"
        ? "Blocked"
        : "Inconclusive";
}

function StageRow({ result }: { result: ComponentTestResult }) {
  const isPassing = result.verdict === "passed";
  return (
    <li className="grid gap-space-2xs rounded-control border border-app-border bg-app-background p-space-xs sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-start">
      <span
        className={`mt-space-3xs flex h-7 w-7 items-center justify-center rounded-pill ${isPassing ? "bg-app-success/10 text-app-success" : "bg-app-danger/10 text-app-danger"}`}
        aria-hidden
      >
        {isPassing ? <CheckCircle2 className="h-4 w-4" /> : <CircleAlert className="h-4 w-4" />}
      </span>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-space-2xs">
          <h5 className="font-medium capitalize">{result.stage.replace(/_/g, " ")}</h5>
          <span className="font-mono text-xs text-app-muted-foreground">
            {result.assetLibraryId}@{result.version}
          </span>
        </div>
        {result.message && (
          <p className="mt-space-3xs text-xs text-app-muted-foreground">{result.message}</p>
        )}
        {result.remediation && (
          <p className="mt-space-2xs rounded-control border border-app-warning/30 bg-app-warning/10 px-space-2xs py-space-2xs text-xs text-app-foreground">
            <span className="font-medium text-app-warning">Recommended next step:</span>{" "}
            {result.remediation}
          </p>
        )}
      </div>
      <StatusBadge tone={tone(result.verdict)}>{verdictLabel(result.verdict)}</StatusBadge>
    </li>
  );
}

function Report({ report }: { report: ComponentTestReport }) {
  const passed = report.results.filter((result) => result.verdict === "passed").length;
  const attention = report.results.length - passed;
  const hasIssues = attention > 0;
  return (
    <article
      data-testid="component-test-report"
      className="overflow-hidden rounded-panel border border-app-border bg-app-surface shadow-sm"
    >
      <header
        className={`border-b p-space-sm ${hasIssues ? "border-app-warning/30 bg-app-warning/5" : "border-app-success/30 bg-app-success/5"}`}
      >
        <div className="flex flex-wrap items-start justify-between gap-space-xs">
          <div className="flex items-start gap-space-xs">
            <span
              className={`flex h-10 w-10 items-center justify-center rounded-control ${hasIssues ? "bg-app-warning/10 text-app-warning" : "bg-app-success/10 text-app-success"}`}
              aria-hidden
            >
              {hasIssues ? (
                <CircleAlert className="h-5 w-5" />
              ) : (
                <ShieldCheck className="h-5 w-5" />
              )}
            </span>
            <div>
              <p className="text-sm font-semibold">{verdictLabel(report.verdict)}</p>
              <p className="mt-space-3xs text-xs text-app-muted-foreground">
                Declared behavior for {report.rootLibraryId || "this component"}
                {report.rootVersion ? `@${report.rootVersion}` : ""}
                {report.includeClosure ? " and its dependency closure" : ""}.
              </p>
            </div>
          </div>
          <StatusBadge tone={tone(report.verdict)}>{report.verdict}</StatusBadge>
        </div>
        <div className="mt-space-xs flex flex-wrap gap-space-2xs text-xs">
          <span className="rounded-pill bg-app-surface px-space-xs py-space-3xs text-app-muted-foreground">
            <strong className="text-app-foreground">{passed}</strong> checks passed
          </span>
          {hasIssues && (
            <span className="rounded-pill bg-app-surface px-space-xs py-space-3xs text-app-muted-foreground">
              <strong className="text-app-foreground">{attention}</strong> need attention
            </span>
          )}
          <span className="rounded-pill bg-app-surface px-space-xs py-space-3xs font-mono text-app-muted-foreground">
            {report.id}
          </span>
        </div>
      </header>
      <div className="space-y-space-sm p-space-sm">
        <section aria-labelledby="test-results-heading">
          <div className="mb-space-2xs flex items-center justify-between">
            <h4 id="test-results-heading" className="text-sm font-semibold">
              Results
            </h4>
            <span className="text-xs text-app-muted-foreground">
              {report.results.length} stage{report.results.length === 1 ? "" : "s"}
            </span>
          </div>
          <ul className="space-y-space-2xs">
            {report.results.map((result, index) => (
              <StageRow key={`${result.stage}-${result.assetLibraryId}-${index}`} result={result} />
            ))}
          </ul>
        </section>
        {(report.artifacts?.length ?? 0) > 0 && (
          <section
            aria-labelledby="test-evidence-heading"
            className="rounded-control border border-app-border bg-app-background p-space-xs"
          >
            <div className="flex items-center gap-space-2xs">
              <FileCheck2 aria-hidden className="h-4 w-4 text-app-info" />
              <h4 id="test-evidence-heading" className="text-sm font-semibold">
                Evidence
              </h4>
            </div>
            <p className="mt-space-3xs text-xs text-app-muted-foreground">
              Durable inputs captured with this run.
            </p>
            <ul className="mt-space-2xs space-y-space-3xs">
              {report.artifacts?.map((artifact) => (
                <li
                  key={artifact.reference}
                  className="break-all font-mono text-xs text-app-muted-foreground"
                >
                  {artifact.kind} · {artifact.reference}
                </li>
              ))}
            </ul>
          </section>
        )}
      </div>
    </article>
  );
}

function TestHistorySkeleton() {
  return (
    <div
      data-testid="component-test-history-skeleton"
      role="status"
      aria-live="polite"
      aria-label="Loading test history"
      className="animate-pulse rounded-panel border border-app-border bg-app-surface p-space-sm"
    >
      <div className="flex items-start justify-between gap-space-sm">
        <div className="space-y-space-2xs">
          <span className="block h-4 w-28 rounded-pill bg-app-surface-muted" />
          <span className="block h-3 w-64 max-w-full rounded-pill bg-app-surface-muted" />
        </div>
        <span className="block h-6 w-16 rounded-pill bg-app-surface-muted" />
      </div>
      <div className="mt-space-md space-y-space-2xs">
        <span className="block h-3 w-20 rounded-pill bg-app-surface-muted" />
        <span className="block h-14 rounded-control bg-app-surface-muted" />
        <span className="block h-14 rounded-control bg-app-surface-muted" />
      </div>
      <span className="sr-only">Loading test history…</span>
    </div>
  );
}

export function ComponentTestPanel({
  componentId,
  version,
}: {
  componentId: string;
  version: string;
}) {
  const [search] = useSearchParams();
  const reportID = search.get("testReport") || "";
  const queryClient = useQueryClient();
  const reports = useQuery({
    queryKey: ["component-tests", componentId, version],
    queryFn: () => listComponentTestReports({ componentId, version }),
    enabled: Boolean(componentId && version),
  });
  const selected = useQuery({
    queryKey: ["component-test-report", reportID],
    queryFn: () => getComponentTestReport(reportID),
    enabled: Boolean(reportID),
  });
  const run = useMutation({
    mutationFn: () => runComponentTest({ componentId, version, includeClosure: true }),
    onSuccess: () =>
      void queryClient.invalidateQueries({ queryKey: ["component-tests", componentId, version] }),
  });
  const latest = run.data ?? selected.data ?? reports.data?.[0];

  return (
    <section
      data-testid="component-test-panel"
      className="space-y-space-sm text-sm text-app-foreground"
      aria-label="Component tests"
    >
      <header className="flex flex-wrap items-start justify-between gap-space-xs rounded-panel border border-app-border bg-app-surface-muted p-space-sm">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-app-primary">
            Quality evidence
          </p>
          <h3 className="mt-space-3xs text-lg font-semibold">Component tests</h3>
          <p className="mt-space-3xs max-w-xl text-xs text-app-muted-foreground">
            Validate declared behavior for version {version || "unselected"}, including the pinned
            dependency closure. Each run is retained as reviewable evidence.
          </p>
        </div>
        <Button size="sm" onClick={() => run.mutate()} disabled={run.isPending || !version}>
          <Play aria-hidden className="h-4 w-4" />
          {run.isPending ? "Running checks…" : "Run component tests"}
        </Button>
      </header>
      {run.isError && (
        <p
          role="alert"
          className="rounded-control border border-app-danger/30 bg-app-danger/10 p-space-xs text-xs text-app-danger"
        >
          {run.error instanceof Error
            ? run.error.message
            : "The component test could not be started."}
        </p>
      )}
      {selected.isError && (
        <p
          role="alert"
          className="rounded-control border border-app-warning/30 bg-app-warning/10 p-space-xs text-xs text-app-warning"
        >
          The requested report is unavailable. Choose a report from history or run the current
          contract.
        </p>
      )}
      {reports.isError && (
        <p
          role="alert"
          className="rounded-control border border-app-danger/30 bg-app-danger/10 p-space-xs text-xs text-app-danger"
        >
          Test history could not be loaded. Retry this page; any new test result remains available
          here.
        </p>
      )}
      {reports.isLoading ? (
        <TestHistorySkeleton />
      ) : latest ? (
        <Report report={latest} />
      ) : reports.isError ? null : (
        <EmptyState
          className="border border-dashed border-app-border bg-app-surface-muted p-space-md text-xs"
          title="No component test evidence yet"
          description="Run the declared contract to create a durable, shareable result."
        />
      )}
      {reports.data && reports.data.length > 1 && (
        <section aria-labelledby="test-history-heading">
          <h4 id="test-history-heading" className="text-sm font-semibold">
            Run history
          </h4>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            Open a prior run without losing the current component context.
          </p>
          <ul className="mt-space-2xs space-y-space-3xs">
            {reports.data.slice(1).map((report) => (
              <li key={report.id}>
                <Link
                  to={assetSearchForTab("tests", report.id)}
                  className="flex items-center justify-between gap-space-xs rounded-control border border-app-border bg-app-surface px-space-xs py-space-2xs text-sm transition hover:bg-app-surface-muted focus:outline-none focus:ring-2 focus:ring-app-primary"
                  aria-label={`Open component test report ${report.id}`}
                >
                  <span className="flex min-w-0 items-center gap-space-2xs">
                    <StatusBadge tone={tone(report.verdict)}>
                      {verdictLabel(report.verdict)}
                    </StatusBadge>
                    <span className="truncate font-mono text-xs text-app-muted-foreground">
                      {report.id}
                    </span>
                  </span>
                  <ChevronRight
                    aria-hidden
                    className="h-4 w-4 shrink-0 text-app-muted-foreground"
                  />
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </section>
  );
}
