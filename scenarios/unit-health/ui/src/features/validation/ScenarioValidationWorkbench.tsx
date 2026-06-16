import { useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useMutation } from "@tanstack/react-query";
import {
  CheckCircle2,
  FileWarning,
  Layers,
  PlayCircle,
  ShieldCheck,
} from "lucide-react";

import { validationClient } from "../../api/validation";
import type { ValidateScenarioResponse } from "../../api/validation";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

const DEFAULT_SCENARIO = "unit-health";

const normalize = (value: string) => value.trim().toLowerCase();

const severityWeight = (severity: string) => {
  switch (normalize(severity)) {
    case "error":
      return 0;
    case "warning":
      return 1;
    default:
      return 2;
  }
};

const statusTone = (status: string) => {
  switch (normalize(status)) {
    case "passed":
      return "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
    case "failed":
    case "error":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    case "degraded":
      return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    default:
      return "border-app-border bg-app-surface-muted text-app-muted-foreground";
  }
};

const severityTone = (severity: string) => {
  switch (normalize(severity)) {
    case "error":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    case "warning":
      return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    default:
      return "border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300";
  }
};

const shortPath = (path: string) => path || "unknown";

/**
 * ScenarioValidationWorkbench is Unit Health's operator surface: enter a
 * scenario, run `ValidateScenario`, and read back the maturity verdict plus
 * normalized findings. This is the focused first cut — the dense test-plan,
 * coverage, and diagnostics dashboards are deferred to a later phase. It
 * mirrors the structure of quality-health's audit workbench (form + metrics +
 * findings list, with loading/empty/error states and i18n strings).
 */
export function ScenarioValidationWorkbench() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState(DEFAULT_SCENARIO);

  const validation = useMutation({
    mutationFn: (target: string) =>
      validationClient.validateScenario({
        scenario: target,
        includeExecution: true,
        useCache: true,
      }),
  });

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const next = scenario.trim();
    if (!next) return;
    validation.mutate(next);
  };

  const data = validation.data;

  const findings = useMemo(() => {
    if (!data) return [];
    return [...data.findings].sort((a, b) => severityWeight(a.severity) - severityWeight(b.severity));
  }, [data]);

  return (
    <section
      data-testid={selectors.validationWorkbench.root}
      aria-labelledby="validation-workbench-heading"
      className="flex flex-col gap-5"
    >
      <div className="flex flex-col gap-3 border-b border-app-border pb-4 lg:flex-row lg:items-end lg:justify-between">
        <div className="max-w-3xl">
          <p
            data-testid={selectors.app.eyebrow}
            className="text-xs font-semibold uppercase text-app-muted-foreground"
          >
            {t(strings.app.eyebrow)}
          </p>
          <h2 id="validation-workbench-heading" className="mt-1 text-2xl font-semibold">
            {t(strings.validation.title)}
          </h2>
          <p
            data-testid={selectors.app.description}
            className="mt-2 text-sm text-app-muted-foreground"
          >
            {t(strings.app.description)}
          </p>
        </div>
        <form onSubmit={handleSubmit} className="flex flex-wrap items-end gap-2">
          <label className="flex min-w-64 flex-col gap-1 text-sm">
            <span className="text-app-muted-foreground">{t(strings.validation.scenarioLabel)}</span>
            <Input
              data-testid={selectors.validationWorkbench.scenarioInput}
              value={scenario}
              onChange={(event) => setScenario(event.target.value)}
              aria-label={t(strings.validation.scenarioLabel)}
              placeholder={t(strings.validation.scenarioPlaceholder)}
            />
          </label>
          <Button
            data-testid={selectors.validationWorkbench.runButton}
            type="submit"
            disabled={validation.isPending}
          >
            <PlayCircle aria-hidden="true" className="h-4 w-4" />
            {validation.isPending ? t(strings.validation.running) : t(strings.validation.run)}
          </Button>
        </form>
      </div>

      {validation.isPending && (
        <div
          data-testid={selectors.validationWorkbench.loading}
          className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground"
        >
          {t(strings.validation.loading)}
        </div>
      )}

      {validation.error && (
        <div
          data-testid={selectors.validationWorkbench.error}
          className="rounded-panel border border-red-500/40 bg-red-500/10 p-4 text-sm text-red-700 dark:text-red-300"
        >
          {errorMessage(validation.error, t)}
        </div>
      )}

      {data && (
        <>
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
            <Metric
              testId={selectors.validationWorkbench.status}
              label={t(strings.validation.status)}
              value={data.status || t(strings.validation.unknown)}
              tone={statusTone(data.status)}
              icon={<ShieldCheck aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.validationWorkbench.maturity}
              label={t(strings.validation.maturity)}
              value={
                data.maturity
                  ? `R${data.maturity.rung} ${data.maturity.label}`
                  : t(strings.validation.unknown)
              }
              icon={<CheckCircle2 aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.validationWorkbench.counts}
              label={t(strings.validation.findings)}
              value={t(strings.validation.countSummary, {
                errors: data.counts?.errors ?? 0,
                warnings: data.counts?.warnings ?? 0,
                infos: data.counts?.infos ?? 0,
              })}
              icon={<FileWarning aria-hidden="true" className="h-4 w-4" />}
            />
            <Metric
              testId={selectors.validationWorkbench.surfaces}
              label={t(strings.validation.surfaces)}
              value={String(data.counts?.surfaces ?? data.surfaces.length)}
              icon={<Layers aria-hidden="true" className="h-4 w-4" />}
            />
          </div>

          {data.summary && <p className="text-sm text-app-muted-foreground">{data.summary}</p>}

          {data.maturity?.rationale && (
            <p
              data-testid={selectors.validationWorkbench.rationale}
              className="text-sm text-app-muted-foreground"
            >
              {data.maturity.rationale}
            </p>
          )}

          {data.degradedReason && (
            <div
              data-testid={selectors.validationWorkbench.degraded}
              className="rounded-panel border border-amber-500/40 bg-amber-500/10 p-4 text-sm text-amber-700 dark:text-amber-300"
            >
              {data.degradedReason}
            </div>
          )}

          <Panel title={t(strings.validation.findingsTitle)} testId={selectors.validationWorkbench.findings}>
            {findings.length === 0 ? (
              <p
                data-testid={selectors.validationWorkbench.empty}
                className="text-sm text-app-muted-foreground"
              >
                {t(strings.validation.noFindings)}
              </p>
            ) : (
              <div className="flex flex-col gap-2">
                {findings.map((finding) => (
                  <article
                    key={finding.id}
                    data-testid={selectors.validationWorkbench.findingRow({ id: finding.id })}
                    className="rounded-control border border-app-border bg-app-surface p-3"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={`rounded-control border px-2 py-0.5 text-xs ${severityTone(finding.severity)}`}
                      >
                        {finding.severity}
                      </span>
                      <span className="text-xs font-medium text-app-muted-foreground">{finding.code}</span>
                      <span className="text-xs text-app-muted-foreground">{shortPath(finding.filePath)}</span>
                    </div>
                    <p className="mt-2 text-sm font-medium">{finding.message}</p>
                    {finding.evidence && (
                      <p className="mt-1 line-clamp-2 text-xs text-app-muted-foreground">{finding.evidence}</p>
                    )}
                  </article>
                ))}
              </div>
            )}
          </Panel>
        </>
      )}

      {!data && !validation.isPending && !validation.error && (
        <p
          data-testid={selectors.validationWorkbench.idle}
          className="rounded-panel border border-app-border bg-app-surface p-4 text-sm text-app-muted-foreground"
        >
          {t(strings.validation.idle)}
        </p>
      )}
    </section>
  );
}

function Metric({
  label,
  value,
  testId,
  icon,
  tone = "border-app-border bg-app-surface text-app-foreground",
}: {
  label: string;
  value: string;
  testId: string;
  icon: ReactNode;
  tone?: string;
}) {
  return (
    <div data-testid={testId} className={`rounded-panel border p-4 ${tone}`}>
      <div className="flex items-center justify-between gap-3">
        <p className="text-xs font-semibold uppercase">{label}</p>
        {icon}
      </div>
      <p className="mt-3 text-xl font-semibold">{value}</p>
    </div>
  );
}

function Panel({ title, testId, children }: { title: string; testId: string; children: ReactNode }) {
  return (
    <section data-testid={testId} className="rounded-panel border border-app-border bg-app-surface p-4">
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">{title}</h3>
      <div className="mt-3">{children}</div>
    </section>
  );
}

export type { ValidateScenarioResponse };
