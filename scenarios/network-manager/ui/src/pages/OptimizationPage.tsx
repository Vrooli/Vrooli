import { useMutation } from "@tanstack/react-query";
import { useState } from "react";

import {
  approveOptimizationCandidate,
  createOptimizationRun,
  scoreOptimizationRun,
  type OptimizationRun,
} from "../api/network";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const panelClass = "rounded-panel border border-app-border bg-app-surface p-4";
const buttonClass = "rounded-control bg-app-primary px-3 py-2 text-sm font-medium text-app-primary-foreground";
const secondaryButtonClass = "rounded-control border border-app-border px-3 py-2 text-sm font-medium hover:bg-app-surface-muted";

export function OptimizationPage() {
  const { t } = useTranslation();
  const [run, setRun] = useState<OptimizationRun | undefined>();
  const createRun = useMutation({ mutationFn: createOptimizationRun, onSuccess: setRun });
  const scoreRun = useMutation({
    mutationFn: (runId: string) => scoreOptimizationRun(runId),
    onSuccess: setRun,
  });
  const approveCandidate = useMutation({
    mutationFn: ({ runId, candidateId }: { runId: string; candidateId: string }) =>
      approveOptimizationCandidate(runId, candidateId),
    onSuccess: setRun,
  });
  const topCandidate = run?.candidates.find((candidate) => candidate.approvalRequired) ?? run?.candidates[0];

  return (
    <section data-testid={selectors.pages.optimization} aria-labelledby="optimization-heading" className="flex flex-col gap-4">
      <div>
        <h2 id="optimization-heading" className="text-2xl font-semibold">
          {t(strings.pages.optimization.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.optimization.description)}</p>
      </div>

      <div className="flex flex-wrap gap-2">
        <button type="button" className={buttonClass} onClick={() => createRun.mutate()}>
          {t(strings.pages.optimization.start)}
        </button>
        <button
          type="button"
          className={secondaryButtonClass}
          disabled={!run}
          onClick={() => run && scoreRun.mutate(run.id)}
        >
          {t(strings.pages.optimization.score)}
        </button>
        <button
          type="button"
          className={secondaryButtonClass}
          disabled={!run || !topCandidate}
          onClick={() => run && topCandidate && approveCandidate.mutate({ runId: run.id, candidateId: topCandidate.id })}
        >
          {t(strings.pages.optimization.approve)}
        </button>
      </div>

      {!run && (
        <p data-testid={selectors.network.empty} className={panelClass}>
          {t(strings.pages.optimization.empty)}
        </p>
      )}

      {run && (
        <section data-testid={selectors.network.optimizationTimeline} className={panelClass}>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p className="text-lg font-semibold">{run.id}</p>
              <p className="text-sm text-app-muted-foreground">
                {t(strings.network.status)}: {run.status} · {run.scoringProfile}
              </p>
            </div>
            <p className="max-w-xl text-sm text-app-muted-foreground">{run.recommendation || t(strings.pages.optimization.comparison)}</p>
          </div>

          <div className="mt-5 grid gap-3 lg:grid-cols-3">
            {[
              strings.network.timeline.baseline,
              strings.network.timeline.candidate,
              strings.network.timeline.after,
            ].map((step) => (
              <div key={step} className="rounded-control border border-app-border bg-app-background p-3">
                <p className="text-sm font-semibold uppercase text-app-muted-foreground">{t(step)}</p>
                <p className="mt-2 text-sm">{t(strings.pages.optimization.comparison)}</p>
              </div>
            ))}
          </div>

          <div className="mt-5 grid gap-3">
            {run.candidates.map((candidate) => (
              <article key={candidate.id} className="rounded-control border border-app-border bg-app-background p-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <h3 className="font-semibold">{candidate.description}</h3>
                  <span className="text-sm text-app-muted-foreground">
                    {candidate.status} · {candidate.score.toFixed(2)}
                  </span>
                </div>
                <p className="mt-2 text-sm text-app-muted-foreground">
                  {t(strings.network.approvalRequired)}: {String(candidate.approvalRequired)}
                </p>
                <ul className="mt-2 list-disc space-y-1 ps-5 text-sm">
                  {(candidate.evidence.length > 0 ? candidate.evidence : [t(strings.network.none)]).map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </article>
            ))}
          </div>
        </section>
      )}
    </section>
  );
}
