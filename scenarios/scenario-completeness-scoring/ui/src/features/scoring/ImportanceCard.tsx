import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { ImportanceSummary } from "../../api/scoring";

import { formatPoints } from "./format";

interface ImportanceCardProps {
  importance: ImportanceSummary;
}

export function ImportanceCard({ importance }: ImportanceCardProps) {
  const { t } = useTranslation();
  const signals = importance.signals;
  const coreText =
    signals && signals.distanceToCoreSeed >= 0 && signals.nearestCoreSeed !== ""
      ? t(strings.scoring.importance.coreDistance, {
          distance: signals.distanceToCoreSeed,
          seed: signals.nearestCoreSeed,
        })
      : t(strings.scoring.importance.coreUnknown);

  return (
    <section
      data-testid={selectors.scoring.importance.card}
      aria-label={t(strings.scoring.importance.title)}
      className="rounded-panel border border-app-border bg-app-card p-4 shadow-sm"
    >
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.scoring.importance.title)}
      </h3>
      <div className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
        <div>
          <p className="text-app-muted-foreground">{t(strings.scoring.importance.scoreLabel)}</p>
          <p data-testid={selectors.scoring.importance.score} className="text-2xl font-semibold">
            {formatPoints(importance.score)} / 1.0
          </p>
          {importance.systemRequired && (
            <p className="mt-1 text-xs font-medium text-amber-700 dark:text-amber-300">
              {t(strings.scoring.importance.systemRequired)}
            </p>
          )}
        </div>
        <dl data-testid={selectors.scoring.importance.signals} className="space-y-1">
          <div className="flex justify-between gap-3">
            <dt className="text-app-muted-foreground">{t(strings.scoring.importance.dependents)}</dt>
            <dd>
              {signals?.directReverseDependencyCount ?? 0} /{" "}
              {signals?.transitiveReverseDependencyCount ?? 0}
            </dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className="text-app-muted-foreground">{t(strings.scoring.importance.required)}</dt>
            <dd>{signals?.requiredReverseDependencyCount ?? 0}</dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className="text-app-muted-foreground">{t(strings.scoring.importance.core)}</dt>
            <dd className="text-right">{coreText}</dd>
          </div>
          <div className="flex justify-between gap-3">
            <dt className="text-app-muted-foreground">{t(strings.scoring.importance.recentActivity)}</dt>
            <dd>{signals?.recentActivityCount ?? 0}</dd>
          </div>
        </dl>
      </div>
      {importance.degraded.length > 0 && (
        <p className="mt-3 text-xs text-app-muted-foreground">
          {t(strings.scoring.importance.partial, { sources: importance.degraded.join("; ") })}
        </p>
      )}
    </section>
  );
}
