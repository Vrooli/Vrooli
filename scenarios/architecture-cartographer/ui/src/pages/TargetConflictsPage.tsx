import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { ConflictWorkbench } from "../features/conflicts/ConflictWorkbench";

export function TargetConflictsPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetConflicts}
      aria-labelledby="target-conflicts-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-conflicts-heading" className="text-xl font-semibold">
          {t(strings.pages.conflicts.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.conflicts.description)}
        </p>
      </header>
      <ConflictWorkbench scenario={scenario} />
    </section>
  );
}
