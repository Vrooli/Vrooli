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
import { MaturitySummary } from "./components/MaturitySummary";
import { TestPlanTable } from "./components/TestPlanTable";
import { ExecutionResults } from "./components/ExecutionResults";
import { CoverageDashboard } from "./components/CoverageDashboard";
import { FindingsPanel } from "./components/FindingsPanel";
import { DiagnosticsPanel } from "./components/DiagnosticsPanel";
import { ImpactAndSkillsPanel } from "./components/ImpactAndSkillsPanel";
import { Metric } from "./components/shared";
import { normalize } from "./components/tone";

const DEFAULT_SCENARIO = "unit-health";

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

/**
 * ScenarioValidationWorkbench is Unit Health's dense operator surface: enter a
 * scenario, run `ValidateScenario`, and inspect the full verdict — local
 * maturity + next-level blockers, the discovered test plan, command execution
 * results, the coverage dashboard, architecture/quality findings, diagnostics,
 * and the global-impact / recommended-skills roll-up. Loading, idle, degraded,
 * running, complete, failed, and no-tests states are all handled.
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
    return [...data.findings].sort(
      (a, b) => severityWeight(a.severity) - severityWeight(b.severity),
    );
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

          {(data.runId || data.targetPath) && (
            <dl
              data-testid="validation-run-reference"
              className="flex flex-wrap gap-x-6 gap-y-1 rounded-panel border border-app-border bg-app-surface px-4 py-2 text-xs text-app-muted-foreground"
            >
              {data.runId && (
                <div className="flex gap-1">
                  <dt className="font-semibold">{t(strings.validation.runId)}</dt>
                  <dd className="font-mono">{data.runId}</dd>
                </div>
              )}
              {data.targetPath && (
                <div className="flex gap-1">
                  <dt className="font-semibold">{t(strings.validation.targetPath)}</dt>
                  <dd className="font-mono">{data.targetPath}</dd>
                </div>
              )}
            </dl>
          )}

          {data.artifacts.length > 0 && (
            <section
              data-testid="validation-artifacts"
              className="rounded-panel border border-app-border bg-app-surface px-4 py-2 text-xs"
            >
              <h3 className="mb-1 font-semibold text-app-muted-foreground">
                {t(strings.validation.artifactsTitle)}
              </h3>
              <dl className="flex flex-col gap-1">
                {data.artifacts.map((artifact) => (
                  <div key={`${artifact.kind}:${artifact.reference}`} className="flex flex-wrap gap-x-2">
                    <dt className="font-semibold text-app-muted-foreground">{artifact.label}</dt>
                    <dd className="font-mono text-app-muted-foreground">{artifact.reference}</dd>
                  </div>
                ))}
              </dl>
            </section>
          )}

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

          <MaturitySummary assessment={data.assessment} />
          <TestPlanTable workspaces={data.workspaces} plan={data.plan} />
          <ExecutionResults results={data.commandResults} />
          <CoverageDashboard coverage={data.coverage} />
          <FindingsPanel findings={findings} />
          <DiagnosticsPanel diagnostics={data.diagnostics} />
          <ImpactAndSkillsPanel assessment={data.assessment} nextSteps={data.nextSteps} />
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

export type { ValidateScenarioResponse };
export type WorkbenchChild = ReactNode;
