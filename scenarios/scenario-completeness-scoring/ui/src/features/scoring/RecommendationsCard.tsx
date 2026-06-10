import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { ActionPhase, Recommendation } from "../../api/scoring";
import { formatPoints, priorityKey } from "./format";

interface RecommendationsCardProps {
  recommendations: Recommendation[];
  actionPlan: ActionPhase[];
  /** Current composite score, used to project the post-plan score. */
  compositeScore: number;
}

/**
 * Prioritized recommendations (with point impact) and the phased action
 * plan, including the projected score after completing every phase.
 * Mirrors the CLI report's RECOMMENDATIONS + ACTION PLAN sections.
 */
export function RecommendationsCard({ recommendations, actionPlan, compositeScore }: RecommendationsCardProps) {
  const { t } = useTranslation();

  const projected = Math.round(
    actionPlan.reduce((total, phase) => total + phase.estimatedPoints, compositeScore),
  );

  return (
    <section
      data-testid={selectors.scoring.recommendations.card}
      aria-label={t(strings.scoring.recommendations.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.scoring.recommendations.title)}
      </h3>
      <ol className="mt-2 list-decimal space-y-1 ps-5 text-sm">
        {recommendations.map((recommendation, index) => {
          const key = priorityKey(recommendation.priority);
          return (
            <li key={`${recommendation.priority}-${index}`}>
              <span className="me-1 rounded bg-app-background px-1.5 py-0.5 text-xs font-medium uppercase">
                {key ? t(key) : recommendation.priority}
              </span>
              {recommendation.description}
              {recommendation.impactPoints > 0 && (
                <span className="ms-1 text-xs text-app-muted-foreground">
                  {t(strings.scoring.recommendations.impact, {
                    points: formatPoints(recommendation.impactPoints),
                  })}
                </span>
              )}
            </li>
          );
        })}
      </ol>
      {actionPlan.length > 0 && (
        <div data-testid={selectors.scoring.actionPlan.card} className="mt-4">
          <h4 className="text-sm font-semibold uppercase text-app-muted-foreground">
            {t(strings.scoring.actionPlan.title)}
          </h4>
          <ol className="mt-2 space-y-2 text-sm">
            {actionPlan.map((phase, index) => (
              <li key={`${phase.title}-${index}`}>
                <p className="font-medium">
                  {t(strings.scoring.actionPlan.phaseTitle, { index: index + 1, title: phase.title })}
                  <span className="ms-1 text-xs font-normal text-app-muted-foreground">
                    {t(strings.scoring.actionPlan.estimated, {
                      points: formatPoints(phase.estimatedPoints),
                    })}
                  </span>
                </p>
                <ul className="mt-1 list-disc space-y-0.5 ps-5">
                  {phase.actions.map((action) => (
                    <li key={action}>{action}</li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
          <p data-testid={selectors.scoring.actionPlan.projected} className="mt-2 text-sm text-app-muted-foreground">
            {t(strings.scoring.actionPlan.projected, { score: projected })}
          </p>
        </div>
      )}
    </section>
  );
}
