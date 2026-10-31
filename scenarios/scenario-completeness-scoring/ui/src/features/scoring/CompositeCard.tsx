import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { CompositeScore } from "../../api/scoring";
import { formatPoints } from "./format";

interface CompositeCardProps {
  composite: CompositeScore;
}

const GROUP_SELECTOR_IDS = ["quality", "coverage", "quantity", "ui"] as const;
type GroupSelectorId = (typeof GROUP_SELECTOR_IDS)[number];

const isKnownGroupId = (id: string): id is GroupSelectorId =>
  (GROUP_SELECTOR_IDS as readonly string[]).includes(id);

/**
 * Composite 0-100 score with classification and the per-group metric
 * breakdown. Mirrors the CLI report's COMPLETENESS SCORE section.
 */
export function CompositeCard({ composite }: CompositeCardProps) {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.scoring.composite.card}
      aria-label={t(strings.scoring.composite.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.scoring.composite.title)}
      </h3>
      <p data-testid={selectors.scoring.composite.score} className="mt-1 text-3xl font-semibold">
        {t(strings.scoring.composite.outOf, { score: composite.score })}
      </p>
      <p data-testid={selectors.scoring.composite.classification} className="text-sm text-app-muted-foreground">
        {composite.classificationLabel || composite.classification}
      </p>
      <div className="mt-3 space-y-3">
        {composite.groups.map((group) => (
          <section
            key={group.id}
            aria-label={group.label}
            {...(isKnownGroupId(group.id)
              ? { "data-testid": selectors.scoring.compositeGroup({ id: group.id }) }
              : {})}
          >
            <div className="flex items-baseline justify-between text-sm">
              <h4 className="font-medium">{group.label}</h4>
              <span className="text-app-muted-foreground">
                {t(strings.scoring.composite.groupPoints, {
                  points: formatPoints(group.score),
                  max: formatPoints(group.max),
                })}
              </span>
            </div>
            <ul className="mt-1 space-y-1 text-sm">
              {group.metrics.map((metric) => (
                <li key={metric.id} className="flex flex-wrap items-baseline justify-between gap-x-2 border-t border-app-border py-1">
                  <span>
                    {metric.label}
                    <span className="ms-2 text-xs text-app-muted-foreground">{metric.observed}</span>
                  </span>
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.scoring.composite.metricPoints, {
                      points: formatPoints(metric.points),
                      max: formatPoints(metric.maxPoints),
                    })}
                    {metric.threshold && <span className="ms-1">[{metric.threshold}]</span>}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </section>
  );
}
