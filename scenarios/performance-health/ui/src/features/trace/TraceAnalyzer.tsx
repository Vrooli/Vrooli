import { useEffect, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { FileSearch, Info } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { EmptyState, ErrorState, LoadingState } from "../../components/ui/state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { perfClient, type AnalyzeTraceResponse } from "../../api/perf";
import { severityChipClass } from "../fleet/severity";
import { useScenario } from "../perf/scenarioContextValue";
import { ScenarioPicker } from "../perf/ScenarioPicker";
import { formatMs, formatMsFloat } from "../perf/format";

/**
 * "Open a trace" workflow. Takes a trace artifact handle (a filesystem path to a
 * captured `performance.json`, normally arriving from an audit run via the
 * `?artifact=` query param), calls AnalysisService.AnalyzeTrace, and renders the
 * web-vitals summary plus the located per-component commit table and findings.
 *
 * The artifact handle is shown verbatim so an operator can load the raw trace in
 * Chrome DevTools / `chrome://tracing`; the embedded viewer is the per-component
 * attribution table, which is the actionable distillation of the trace.
 */
export function TraceAnalyzer() {
  const { t } = useTranslation();
  const { scenario, setScenario } = useScenario();
  const [searchParams] = useSearchParams();
  const [artifact, setArtifact] = useState<string>(() => searchParams.get("artifact") ?? "");
  const [result, setResult] = useState<AnalyzeTraceResponse | null>(null);

  // Deep-link from an audit run: adopt the scenario + artifact from the URL.
  useEffect(() => {
    const qsScenario = searchParams.get("scenario");
    const qsArtifact = searchParams.get("artifact");
    if (qsScenario && qsScenario !== scenario) setScenario(qsScenario);
    if (qsArtifact) setArtifact(qsArtifact);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  const analyze = useMutation({
    mutationFn: () => perfClient.analyzeTrace({ scenario, traceArtifact: artifact.trim() }),
    onSuccess: (res) => setResult(res),
  });

  return (
    <section
      data-testid={selectors.pages.trace}
      aria-labelledby="trace-heading"
      className="flex flex-col gap-5"
    >
      <header className="flex flex-col gap-3">
        <h2 id="trace-heading" className="text-2xl font-semibold">
          {t(strings.trace.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.trace.description)}</p>
        <div className="flex flex-wrap items-end gap-3">
          <ScenarioPicker />
          <label className="flex flex-1 flex-col gap-1 text-sm">
            <span className="font-medium text-app-muted-foreground">
              {t(strings.trace.artifactLabel)}
            </span>
            <Input
              data-testid={selectors.trace.artifactInput}
              value={artifact}
              onChange={(e) => setArtifact(e.target.value)}
              placeholder={t(strings.trace.artifactPlaceholder)}
              className="min-w-[16rem] font-mono text-xs"
            />
          </label>
          <Button
            data-testid={selectors.trace.analyzeButton}
            onClick={() => analyze.mutate()}
            disabled={analyze.isPending || artifact.trim() === ""}
          >
            <FileSearch aria-hidden="true" className="me-1 h-4 w-4" />
            {analyze.isPending ? t(strings.common.loading) : t(strings.trace.analyze)}
          </Button>
        </div>
        <p className="flex items-start gap-2 text-xs text-app-muted-foreground">
          <Info aria-hidden="true" className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          {t(strings.trace.devtoolsHint)}
        </p>
      </header>

      {analyze.error && (
        <ErrorState
          testId={selectors.trace.error}
          title={t(strings.trace.errorTitle)}
          message={errorMessage(analyze.error, t)}
          onRetry={() => analyze.mutate()}
          retrying={analyze.isPending}
        />
      )}

      {analyze.isPending && <LoadingState title={t(strings.trace.loadingTitle)} />}

      {!result && !analyze.isPending && !analyze.error && (
        <EmptyState
          testId={selectors.trace.empty}
          icon={FileSearch}
          title={t(strings.trace.emptyTitle)}
          message={t(strings.trace.empty)}
        />
      )}

      {result && !analyze.isPending && <TraceResult result={result} />}
    </section>
  );
}

function TraceResult({ result }: { result: AnalyzeTraceResponse }) {
  const { t } = useTranslation();
  return (
    <>
      <dl
        data-testid={selectors.trace.vitals}
        className="grid grid-cols-2 gap-3 sm:grid-cols-4"
      >
        <Vital label={t(strings.trace.vital.lcp)} value={formatMs(result.lcpMs)} />
        <Vital label={t(strings.trace.vital.fcp)} value={formatMs(result.fcpMs)} />
        <Vital label={t(strings.trace.vital.longTask)} value={formatMs(result.longTaskMs)} />
        <Vital
          label={t(strings.trace.vital.components)}
          value={String(result.components.length)}
        />
      </dl>

      <section
        data-testid={selectors.trace.components}
        aria-label={t(strings.trace.componentsTitle)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.trace.componentsTitle)}
        </h3>
        {result.components.length === 0 ? (
          <p className="mt-3 text-app-muted-foreground">{t(strings.trace.componentsEmpty)}</p>
        ) : (
          <div className="mt-3 overflow-x-auto">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="text-xs uppercase tracking-wide text-app-muted-foreground">
                  <th scope="col" className="px-2 py-1 text-start font-medium">
                    {t(strings.trace.col.component)}
                  </th>
                  <th scope="col" className="px-2 py-1 text-end font-medium">
                    {t(strings.trace.col.commits)}
                  </th>
                  <th scope="col" className="px-2 py-1 text-end font-medium">
                    {t(strings.trace.col.avg)}
                  </th>
                  <th scope="col" className="px-2 py-1 text-end font-medium">
                    {t(strings.trace.col.max)}
                  </th>
                  <th scope="col" className="px-2 py-1 text-start font-medium">
                    {t(strings.trace.col.definition)}
                  </th>
                </tr>
              </thead>
              <tbody>
                {result.components.map((c) => (
                  <tr
                    key={c.component}
                    data-testid={selectors.trace.componentRow({ component: c.component })}
                    className="border-t border-app-border"
                  >
                    <td className="px-2 py-1.5 font-medium text-app-foreground">{c.component}</td>
                    <td className="px-2 py-1.5 text-end tabular-nums">{c.commitCount}</td>
                    <td className="px-2 py-1.5 text-end tabular-nums">{formatMsFloat(c.avgMs)}</td>
                    <td className="px-2 py-1.5 text-end tabular-nums">{formatMsFloat(c.maxMs)}</td>
                    <td className="px-2 py-1.5 font-mono text-xs text-app-muted-foreground">
                      {c.definition || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section
        data-testid={selectors.trace.findings}
        aria-label={t(strings.trace.findingsTitle)}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.trace.findingsTitle)}
        </h3>
        {result.findings.length === 0 ? (
          <p
            data-testid={selectors.trace.findingsEmpty}
            className="mt-3 text-app-muted-foreground"
          >
            {t(strings.trace.findingsEmpty)}
          </p>
        ) : (
          <ul className="mt-3 flex flex-col gap-2">
            {result.findings.map((f) => (
              <li
                key={`${f.code}:${f.component}`}
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
                  <span className="font-medium text-app-foreground">{f.component || f.code}</span>
                </div>
                {f.message && <p className="text-app-muted-foreground">{f.message}</p>}
                {f.definition && (
                  <p className="font-mono text-xs text-app-muted-foreground">{f.definition}</p>
                )}
                {f.evidence && (
                  <p className="font-mono text-xs text-app-muted-foreground">{f.evidence}</p>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  );
}

function Vital({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-panel border border-app-border bg-app-surface p-4">
      <dt className="text-xs uppercase tracking-wide text-app-muted-foreground">{label}</dt>
      <dd className="mt-2 text-2xl font-semibold tabular-nums">{value}</dd>
    </div>
  );
}
