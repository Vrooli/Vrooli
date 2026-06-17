import type { MaturityAssessment } from "@vrooli/proto-types/common/v1/maturity_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";

/**
 * MaturitySummary renders the local-maturity verdict: the current level, the
 * next level, the finding codes blocking promotion, and the next level's exit
 * criteria (looked up from `assessment.local.levels`). Renders nothing when no
 * local assessment is present.
 */
export function MaturitySummary({ assessment }: { assessment?: MaturityAssessment }) {
  const { t } = useTranslation();
  const local = assessment?.local;
  if (!local) return null;

  const nextLevel = local.levels.find(
    (level) => `${level.id} ${level.name}` === local.nextLevel || level.id === local.nextLevel,
  );

  return (
    <Panel
      title={t(strings.validation.maturitySummaryTitle)}
      testId={selectors.validationWorkbench.maturitySummary}
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-wrap gap-4 text-sm">
          <div>
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.validation.currentLevelLabel)}
            </p>
            <p className="mt-1 font-medium">{local.currentLevel || t(strings.validation.unknown)}</p>
          </div>
          <div data-testid={selectors.validationWorkbench.nextLevel}>
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.validation.nextLevelLabel)}
            </p>
            <p className="mt-1 font-medium">
              {local.nextLevel || t(strings.validation.noNextLevel)}
            </p>
          </div>
        </div>

        <div>
          <p className="text-xs font-semibold uppercase text-app-muted-foreground">
            {t(strings.validation.nextLevelBlockers)}
          </p>
          {local.blockingFindingCodes.length === 0 ? (
            <p className="mt-1 text-sm text-app-muted-foreground">
              {t(strings.validation.noBlockers)}
            </p>
          ) : (
            <div className="mt-1 flex flex-wrap gap-2">
              {local.blockingFindingCodes.map((code) => (
                <Pill
                  key={code}
                  tone="border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300"
                >
                  {code}
                </Pill>
              ))}
            </div>
          )}
        </div>

        {nextLevel && nextLevel.exitCriteria.length > 0 && (
          <div>
            <p className="text-xs font-semibold uppercase text-app-muted-foreground">
              {t(strings.validation.exitCriteriaTitle, { level: local.nextLevel })}
            </p>
            <ul className="mt-1 list-inside list-disc text-sm text-app-muted-foreground">
              {nextLevel.exitCriteria.map((criterion) => (
                <li key={criterion}>{criterion}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </Panel>
  );
}
