import { useQuery } from "@tanstack/react-query";

import { convergenceClient } from "../../api/convergence";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { eligibilityLabel, tierLabel } from "../labels";

/**
 * ConvergenceBoard renders per-template four-lens fitness and gold-star
 * generated-golden health/eligibility (OT-P1-002). Numbers + candidates only;
 * tiering is advisory.
 */
export function ConvergenceBoard() {
  const { t } = useTranslation();
  const { data, isLoading, error } = useQuery({
    queryKey: ["convergence", "status"],
    queryFn: () => convergenceClient.getConvergenceStatus({}),
  });

  if (isLoading) {
    return (
      <p data-testid={selectors.convergence.loading} className="text-app-muted-foreground">
        {t(strings.common.loading)}
      </p>
    );
  }
  if (error) {
    return (
      <p data-testid={selectors.convergence.error} className="text-red-400">
        {t(strings.common.error)}
      </p>
    );
  }

  const templates = data?.templates ?? [];
  const references = data?.references ?? [];
  if (templates.length === 0 && references.length === 0) {
    return (
      <p data-testid={selectors.convergence.empty} className="text-app-muted-foreground">
        {t(strings.pages.convergence.empty)}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <section aria-labelledby="conv-templates-heading" className="flex flex-col gap-3">
        <h3 id="conv-templates-heading" className="text-lg font-semibold">
          {t(strings.pages.convergence.templatesHeading)}
        </h3>
        <p
          data-testid={selectors.convergence.methodology}
          className="text-xs italic text-app-muted-foreground"
        >
          {t(strings.pages.convergence.methodologyNote)}
        </p>
        <ul className="flex flex-col gap-2">
          {templates.map((tf) => (
            <li
              key={tf.template}
              data-testid={selectors.convergence.template}
              className="rounded-panel border border-app-border bg-app-surface p-3"
            >
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">{tf.template}</span>
                <span className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                  {tierLabel(tf.tier)}
                </span>
              </div>
              <p className="mt-1 text-xs text-app-muted-foreground">
                {t(strings.pages.convergence.lensLabel, {
                  cost: tf.perReplicaCost,
                  drift: tf.driftSurfaceCount,
                  contracts: tf.commentOnlyContractCount,
                  edits: tf.coordinatedEditCount,
                })}
              </p>
            </li>
          ))}
        </ul>
      </section>

      <section aria-labelledby="conv-references-heading" className="flex flex-col gap-3">
        <h3 id="conv-references-heading" className="text-lg font-semibold">
          {t(strings.pages.convergence.referencesHeading)}
        </h3>
        <ul className="flex flex-col gap-2">
          {references.map((rh) => (
            <li
              key={rh.scenario}
              data-testid={selectors.convergence.reference}
              className="rounded-panel border border-app-border bg-app-surface p-3"
            >
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium">{rh.scenario}</span>
                <span className="rounded bg-app-border px-2 py-0.5 text-xs font-mono">
                  {eligibilityLabel(rh.eligibility)}
                </span>
                {rh.cleanOnAllTools && (
                  <span className="text-xs text-green-500">
                    {t(strings.pages.convergence.cleanBadge)}
                  </span>
                )}
              </div>
              <p className="mt-1 text-xs text-app-muted-foreground">
                {t(strings.pages.convergence.referenceLabel, {
                  days: rh.stabilityDays,
                  breadth: rh.breadth,
                })}
              </p>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
