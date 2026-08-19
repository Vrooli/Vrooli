/** @vrooliComponentSource react-component-library:StatusBadge */
import { useState } from "react";
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
import type { ComponentExperience } from "../../api/components";
import { EvidenceCarousel } from "../../../../library/components/EvidenceCarousel/versions/1.0.7/EvidenceCarousel";
import { OverlayCanvas } from "../../../../library/components/OverlayCanvas/versions/1.0.7/OverlayCanvas";

type VerdictTone = "success" | "danger" | "warning" | "neutral";

function tone(verdict: string): VerdictTone {
  return verdict === "passed"
    ? "success"
    : verdict === "failed"
      ? "danger"
      : verdict === "blocked" || verdict === "unmeasured"
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
        : verdict === "unmeasured"
          ? "Unmeasured"
          : "Inconclusive";
}

function StageRow({ result }: { result: ComponentTestResult }) {
  const isPassing = result.verdict === "passed";
  return (
    <li className="grid gap-space-2xs rounded-control border border-app-border bg-app-background p-space-xs sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-start">
      <span
        className={`mt-space-3xs flex h-control-compact w-control-compact items-center justify-center rounded-pill ${isPassing ? "bg-app-success/10 text-app-success" : "bg-app-danger/10 text-app-danger"}`}
        aria-hidden
      >
        {isPassing ? (
          <CheckCircle2 className="h-icon-sm w-icon-sm" />
        ) : (
          <CircleAlert className="h-icon-sm w-icon-sm" />
        )}
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
              className={`flex h-control-md w-control-md items-center justify-center rounded-control ${hasIssues ? "bg-app-warning/10 text-app-warning" : "bg-app-success/10 text-app-success"}`}
              aria-hidden
            >
              {hasIssues ? (
                <CircleAlert className="h-icon-md w-icon-md" />
              ) : (
                <ShieldCheck className="h-icon-md w-icon-md" />
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
        <section
          data-testid="tests-category-views"
          aria-label="Test categories"
          className="grid gap-space-xs md:grid-cols-2"
        >
          {(["integrity", "behavior", "experience", "cost"] as const).map((category) => {
            const results = report.results.filter((result) => {
              const stage = result.stage.toLowerCase();
              if (category === "integrity")
                return (
                  stage.includes("closure") ||
                  stage.includes("source") ||
                  stage.includes("contract")
                );
              if (category === "behavior")
                return stage.includes("declared") || stage.includes("behavior");
              if (category === "experience")
                return stage.includes("experience") || stage.includes("claim");
              return (
                stage.includes("performance") || stage.includes("cost") || stage.includes("console")
              );
            });
            const clean =
              results.length > 0 && results.every((result) => result.verdict === "passed");
            return (
              <section
                key={category}
                data-testid={`tests-category-${category}`}
                className="rounded-control border border-app-border bg-app-background p-space-xs"
              >
                <div className="flex items-center justify-between gap-space-2xs">
                  <h4 className="text-sm font-semibold capitalize">{category}</h4>
                  <StatusBadge
                    tone={
                      clean
                        ? "success"
                        : results.some((result) => result.verdict === "failed")
                          ? "danger"
                          : "warning"
                    }
                  >
                    {clean ? "Passed" : results.length ? "Needs review" : "Unmeasured"}
                  </StatusBadge>
                </div>
                {clean ? (
                  <p className="mt-space-2xs text-xs text-app-muted-foreground">
                    All recorded checks passed.
                  </p>
                ) : results.length ? (
                  <ul className="mt-space-2xs space-y-space-2xs">
                    {results.map((result, index) => (
                      <StageRow key={`${category}-${result.stage}-${index}`} result={result} />
                    ))}
                  </ul>
                ) : (
                  <p className="mt-space-2xs text-xs text-app-muted-foreground">
                    No capture is available for this category.
                  </p>
                )}
              </section>
            );
          })}
        </section>
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
              <FileCheck2 aria-hidden className="h-icon-sm w-icon-sm text-app-info" />
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
          <span className="block h-icon-sm w-field-compact rounded-pill bg-app-surface-muted" />
          <span className="block h-icon-xs w-panel-compact max-w-full rounded-pill bg-app-surface-muted" />
        </div>
        <span className="block h-icon-lg w-avatar-sm rounded-pill bg-app-surface-muted" />
      </div>
      <div className="mt-space-md space-y-space-2xs">
        <span className="block h-icon-xs w-avatar-md rounded-pill bg-app-surface-muted" />
        <span className="block h-control-2xl rounded-control bg-app-surface-muted" />
        <span className="block h-control-2xl rounded-control bg-app-surface-muted" />
      </div>
      <span className="sr-only">Loading test history…</span>
    </div>
  );
}

export function ComponentTestPanel({
  componentId,
  version,
  experience,
}: {
  componentId: string;
  version: string;
  experience?: ComponentExperience;
}) {
  const [search] = useSearchParams();
  const [includeClosure, setIncludeClosure] = useState(false);
  const failedClaims = (experience?.claims ?? []).filter((claim) =>
    (experience?.evidence ?? []).some(
      (item) => item.claimId === claim.id && item.verdict === "failed",
    ),
  );
  const [selectedClaimID, setSelectedClaimID] = useState("");
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
    mutationFn: () => runComponentTest({ componentId, version, includeClosure }),
    onSuccess: () =>
      void queryClient.invalidateQueries({ queryKey: ["component-tests", componentId, version] }),
  });
  const latest = run.data ?? selected.data ?? reports.data?.[0];
  const activeClaimID = selectedClaimID || failedClaims[0]?.id || "";
  const activeClaim = experience?.claims.find((claim) => claim.id === activeClaimID);
  const activeEvidence = experience?.evidence.find(
    (item) => item.claimId === activeClaimID && item.verdict === "failed",
  );
  const measurement = activeEvidence?.measurement;
  const overlaySubjects = (measurement?.subjects ?? []).map((subject) => ({
    id: subject.testId || subject.elementId,
    label: subject.value || subject.testId || subject.elementId,
    x: subject.bounds?.x ?? 0,
    y: subject.bounds?.y ?? 0,
    width: subject.bounds?.width ?? 0,
    height: subject.bounds?.height ?? 0,
  }));

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
        <label className="flex items-center gap-space-2xs text-xs text-app-muted-foreground">
          <input
            type="checkbox"
            checked={includeClosure}
            onChange={(event) => setIncludeClosure(event.target.checked)}
          />
          Include dependency closure
        </label>
        <Button size="sm" onClick={() => run.mutate()} disabled={run.isPending || !version}>
          <Play aria-hidden className="h-icon-sm w-icon-sm" />
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
      <section
        data-testid="claim-overlay-panel"
        aria-label="Claim overlay and evidence"
        className="space-y-space-xs rounded-panel border border-app-border bg-app-surface-muted p-space-sm"
      >
        <div className="flex flex-wrap items-center justify-between gap-space-xs">
          <div>
            <h3 className="text-sm font-semibold">Claim overlay</h3>
            <p className="text-xs text-app-muted-foreground">
              Select a failed claim to inspect its measured subjects.
            </p>
          </div>
          {failedClaims.length ? (
            <div role="tablist" aria-label="Failed claims" className="flex flex-wrap gap-space-2xs">
              {failedClaims.map((claim) => (
                <button
                  key={claim.id}
                  type="button"
                  role="tab"
                  aria-selected={claim.id === activeClaimID}
                  onClick={() => setSelectedClaimID(claim.id)}
                  className="rounded-control border border-app-border px-space-xs py-space-2xs text-xs focus:outline-none focus:ring-2 focus:ring-app-primary"
                >
                  {claim.id}
                </button>
              ))}
            </div>
          ) : null}
        </div>
        {activeClaim && activeEvidence && measurement ? (
          <>
            <p className="text-xs text-app-muted-foreground">{activeClaim.statement}</p>
            <dl className="grid grid-cols-3 gap-space-2xs text-xs">
              <div>
                <dt className="text-app-muted-foreground">Observed</dt>
                <dd className="font-semibold">{measurement.observed ?? "—"}</dd>
              </div>
              <div>
                <dt className="text-app-muted-foreground">Required</dt>
                <dd className="font-semibold">{measurement.required ?? "—"}</dd>
              </div>
              <div>
                <dt className="text-app-muted-foreground">Unit</dt>
                <dd className="font-semibold">{measurement.unit || "—"}</dd>
              </div>
            </dl>
            <OverlayCanvas
              subjects={overlaySubjects}
              message={`${measurement.metric || activeClaim.id} measured overlay`}
            />
          </>
        ) : (
          <p
            data-testid="claim-overlay-unmeasured"
            className="rounded-control border border-dashed border-app-border p-space-xs text-xs text-app-muted-foreground"
          >
            Unmeasured: no capture is available for the selected claim.
          </p>
        )}
        <EvidenceCarousel
          items={[
            "screenshot",
            "accessibility-tree",
            "computed-style",
            "layout-box",
            "console",
            "performance",
          ].map((kind) => ({
            id: kind,
            kind,
            status: activeEvidence?.captureRef ? ("available" as const) : ("missing" as const),
            reference: activeEvidence?.captureRef,
          }))}
        />
      </section>
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
                    className="h-icon-sm w-icon-sm shrink-0 text-app-muted-foreground"
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
