import { useMemo, useState, type FormEvent } from "react";
import { useMutation } from "@tanstack/react-query";
import { CheckCircle2, FileDiff, Play, Search, XCircle } from "lucide-react";

import { previewFix, statusLabel, validateScenario, type ValidationReport } from "../api/validation";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

const DEFAULT_SCENARIO = "api-health";

export function ValidationPage() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState(DEFAULT_SCENARIO);
  const [path, setPath] = useState("");
  const [includeExecution, setIncludeExecution] = useState(false);
  const [lastReport, setLastReport] = useState<ValidationReport | null>(null);

  const validation = useMutation({
    mutationFn: () =>
      validateScenario({
        scenario,
        path,
        includeExecution,
      }),
    onSuccess: (report) => {
      setLastReport(report);
    },
  });

  const fixPreview = useMutation({
    mutationFn: () =>
      previewFix({
        scenario,
        path,
      }),
  });

  const report = validation.data ?? lastReport;
  const findings = report?.response.assessment?.findings ?? [];
  const capabilities = report?.response.assessment?.capabilities ?? [];
  const probe = report?.nativeDetail.target?.health_probe;
  const target = report?.nativeDetail.target;
  const summary = report?.nativeDetail.summary;
  const fixCandidates = fixPreview.data?.candidates ?? [];
  const fixMessages = fixPreview.data?.messages ?? [];

  const status = report ? statusLabel(report.response.status) : "idle";
  const statusTone = useMemo(() => {
    if (status === "passed") {
      return "text-emerald-600";
    }
    if (status === "failed" || status === "error") {
      return "text-red-600";
    }
    return "text-app-muted-foreground";
  }, [status]);

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    validation.mutate();
  };

  return (
    <section
      data-testid={selectors.pages.validation}
      aria-labelledby="validation-heading"
      className="flex flex-col gap-5"
    >
      <div>
        <h2 id="validation-heading" className="text-2xl font-semibold">
          {t(strings.pages.validation.title)}
        </h2>
        <p className="mt-1 text-sm text-app-muted-foreground">
          {t(strings.pages.validation.description)}
        </p>
      </div>

      <form
        data-testid={selectors.validation.form}
        onSubmit={submit}
        className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-4 lg:grid-cols-[minmax(12rem,1fr)_minmax(12rem,1fr)_auto_auto]"
      >
        <label className="flex flex-col gap-1 text-sm font-medium">
          {t(strings.pages.validation.scenarioLabel)}
          <Input
            value={scenario}
            onChange={(event) => setScenario(event.currentTarget.value)}
            data-testid={selectors.validation.scenarioInput}
            placeholder="api-health"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium">
          {t(strings.pages.validation.pathLabel)}
          <Input
            value={path}
            onChange={(event) => setPath(event.currentTarget.value)}
            data-testid={selectors.validation.pathInput}
            placeholder="/workspace/scenarios/api-health"
          />
        </label>
        <label className="flex items-center gap-2 self-end rounded-control border border-app-border px-3 py-2 text-sm">
          <input
            type="checkbox"
            checked={includeExecution}
            onChange={(event) => setIncludeExecution(event.currentTarget.checked)}
            data-testid={selectors.validation.executionToggle}
          />
          {t(strings.pages.validation.executionLabel)}
        </label>
        <div className="flex items-end gap-2">
          <Button type="submit" disabled={validation.isPending}>
            <Search className="h-4 w-4" aria-hidden="true" />
            {validation.isPending
              ? t(strings.pages.validation.validating)
              : t(strings.pages.validation.validate)}
          </Button>
          <Button
            type="button"
            variant="outline"
            disabled={fixPreview.isPending}
            onClick={() => fixPreview.mutate()}
            data-testid={selectors.validation.fixPreviewButton}
          >
            <FileDiff className="h-4 w-4" aria-hidden="true" />
            {t(strings.pages.validation.previewFixes)}
          </Button>
        </div>
      </form>

      {validation.error ? (
        <p data-testid={selectors.validation.error} className="text-sm text-red-600">
          {errorMessage(validation.error, t)}
        </p>
      ) : null}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(18rem,0.9fr)]">
        <article
          data-testid={selectors.validation.summary}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
                {t(strings.pages.validation.summaryTitle)}
              </h3>
              <p className={`mt-2 text-3xl font-semibold ${statusTone}`}>{status}</p>
            </div>
            {status === "passed" ? (
              <CheckCircle2 className="h-6 w-6 text-emerald-600" aria-hidden="true" />
            ) : status === "failed" || status === "error" ? (
              <XCircle className="h-6 w-6 text-red-600" aria-hidden="true" />
            ) : (
              <Play className="h-6 w-6 text-app-muted-foreground" aria-hidden="true" />
            )}
          </div>
          <dl className="mt-4 grid gap-3 sm:grid-cols-2">
            <Metric label={t(strings.pages.validation.targetLabel)} value={target?.scenario ?? report?.response.scenario ?? "—"} />
            <Metric label={t(strings.pages.validation.resolutionLabel)} value={target?.resolution ?? "—"} />
            <Metric label={t(strings.pages.validation.apiKindLabel)} value={target?.api_kind ?? "—"} />
            <Metric
              label={t(strings.pages.validation.findingsLabel)}
              value={`${summary?.errors ?? 0} / ${summary?.warnings ?? 0} / ${summary?.infos ?? 0}`}
            />
          </dl>
        </article>

        <article
          data-testid={selectors.validation.probe}
          className="rounded-panel border border-app-border bg-app-surface p-4"
        >
          <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.pages.validation.probeTitle)}
          </h3>
          {probe?.requested ? (
            <dl className="mt-4 grid gap-3">
              <Metric label={t(strings.pages.validation.probeUrlLabel)} value={probe.url ?? "—"} />
              <Metric label={t(strings.pages.validation.probeStatusLabel)} value={String(probe.status_code ?? "—")} />
              <Metric label={t(strings.pages.validation.probeSchemaLabel)} value={String(probe.schema_valid ?? false)} />
              <Metric label={t(strings.pages.validation.probeElapsedLabel)} value={`${probe.elapsed_millis ?? 0} ms`} />
              <Metric label={t(strings.pages.validation.probePayloadLabel)} value={probe.payload?.status ?? probe.failure_class ?? "—"} />
            </dl>
          ) : (
            <p className="mt-4 text-sm text-app-muted-foreground">
              {t(strings.pages.validation.probeSkipped)}
            </p>
          )}
        </article>
      </div>

      <article
        data-testid={selectors.validation.capabilities}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.validation.capabilitiesTitle)}
        </h3>
        <div className="mt-3 grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {capabilities.length > 0 ? capabilities.map((capability) => (
            <div key={capability.id} className="rounded-control border border-app-border p-3">
              <p className="text-sm font-semibold">{capability.label || capability.id}</p>
              <p className="mt-1 text-xs text-app-muted-foreground">
                {capability.currentLevel || "—"} → {capability.nextLevel || "—"}
              </p>
              {capability.blockingFindingCodes.length > 0 ? (
                <p className="mt-2 break-words text-xs text-app-muted-foreground">
                  {capability.blockingFindingCodes.join(", ")}
                </p>
              ) : null}
            </div>
          )) : (
            <p className="text-sm text-app-muted-foreground">
              {t(strings.pages.validation.noCapabilities)}
            </p>
          )}
        </div>
      </article>

      <article
        data-testid={selectors.validation.findings}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.validation.findingsTitle)}
        </h3>
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-xs uppercase text-app-muted-foreground">
              <tr>
                <th className="py-2 pr-4">{t(strings.pages.validation.severityColumn)}</th>
                <th className="py-2 pr-4">{t(strings.pages.validation.codeColumn)}</th>
                <th className="py-2 pr-4">{t(strings.pages.validation.messageColumn)}</th>
                <th className="py-2 pr-4">{t(strings.pages.validation.fixColumn)}</th>
              </tr>
            </thead>
            <tbody>
              {findings.length > 0 ? findings.map((finding) => (
                <tr key={`${finding.code}-${finding.location}`} className="border-t border-app-border">
                  <td className="py-2 pr-4">{finding.severity}</td>
                  <td className="py-2 pr-4 font-mono text-xs">{finding.code}</td>
                  <td className="py-2 pr-4">{finding.message || finding.title}</td>
                  <td className="py-2 pr-4">{finding.fixClass || "manual"}</td>
                </tr>
              )) : (
                <tr>
                  <td className="py-3 text-app-muted-foreground" colSpan={4}>
                    {t(strings.pages.validation.noFindings)}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </article>

      <article
        data-testid={selectors.validation.fixPreview}
        className="rounded-panel border border-app-border bg-app-surface p-4"
      >
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.pages.validation.fixPreviewTitle)}
        </h3>
        {fixPreview.error ? (
          <p className="mt-3 text-sm text-red-600">{errorMessage(fixPreview.error, t)}</p>
        ) : null}
        <div className="mt-3 flex flex-col gap-2">
          {fixCandidates.length > 0 ? fixCandidates.map((candidate) => (
            <div key={`${candidate.ruleId}-${candidate.filePath}`} className="rounded-control border border-app-border p-3">
              <p className="text-sm font-semibold">{candidate.ruleId}</p>
              <p className="mt-1 break-words font-mono text-xs text-app-muted-foreground">
                {candidate.filePath}
              </p>
              <p className="mt-2 text-sm">{candidate.description}</p>
            </div>
          )) : (
            <p className="text-sm text-app-muted-foreground">
              {fixMessages[0] ?? t(strings.pages.validation.noFixes)}
            </p>
          )}
        </div>
      </article>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="mt-1 break-words text-sm font-medium">{value}</dd>
    </div>
  );
}
