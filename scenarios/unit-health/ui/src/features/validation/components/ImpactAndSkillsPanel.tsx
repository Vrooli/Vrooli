import type { MaturityAssessment } from "@vrooli/proto-types/common/v1/maturity_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Panel, Pill } from "./shared";

/**
 * ImpactAndSkillsPanel renders the global-impact grouping
 * (`assessment.findingsByGlobalImpact`), the recommended skill ids, and the
 * response-level next steps in one operator roll-up.
 */
export function ImpactAndSkillsPanel({
  assessment,
  nextSteps,
}: {
  assessment?: MaturityAssessment;
  nextSteps: string[];
}) {
  const { t } = useTranslation();
  const impacts = Object.entries(assessment?.findingsByGlobalImpact ?? {});
  const skills = assessment?.recommendedSkillIds ?? [];

  return (
    <div className="grid gap-3 lg:grid-cols-3">
      <Panel
        title={t(strings.validation.globalImpactTitle)}
        testId={selectors.validationWorkbench.globalImpact}
      >
        {impacts.length === 0 ? (
          <p
            data-testid={selectors.validationWorkbench.globalImpactEmpty}
            className="text-sm text-app-muted-foreground"
          >
            {t(strings.validation.globalImpactEmpty)}
          </p>
        ) : (
          <ul className="flex flex-col gap-1 text-sm">
            {impacts.map(([key, count]) => (
              <li
                key={key}
                data-testid={selectors.validationWorkbench.impactRow({ key })}
                className="flex items-center justify-between gap-2"
              >
                <span className="text-app-muted-foreground">{key}</span>
                <Pill tone="border-app-border bg-app-surface-muted text-app-foreground">
                  {count}
                </Pill>
              </li>
            ))}
          </ul>
        )}
      </Panel>

      <Panel
        title={t(strings.validation.recommendedSkillsTitle)}
        testId={selectors.validationWorkbench.recommendedSkills}
      >
        {skills.length === 0 ? (
          <p
            data-testid={selectors.validationWorkbench.recommendedSkillsEmpty}
            className="text-sm text-app-muted-foreground"
          >
            {t(strings.validation.recommendedSkillsEmpty)}
          </p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {skills.map((skill) => (
              <Pill
                key={skill}
                tone="border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300"
              >
                {skill}
              </Pill>
            ))}
          </div>
        )}
      </Panel>

      <Panel
        title={t(strings.validation.nextStepsTitle)}
        testId={selectors.validationWorkbench.nextSteps}
      >
        {nextSteps.length === 0 ? (
          <p className="text-sm text-app-muted-foreground">{t(strings.validation.noFindings)}</p>
        ) : (
          <ul className="list-inside list-disc text-sm text-app-muted-foreground">
            {nextSteps.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}
