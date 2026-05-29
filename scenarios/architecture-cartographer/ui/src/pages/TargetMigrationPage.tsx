import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { MigrationWorkbench } from "../features/migration/MigrationWorkbench";

export function TargetMigrationPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetMigration}
      aria-labelledby="target-migration-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-migration-heading" className="text-xl font-semibold">
          {t(strings.pages.migration.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.migration.description)}</p>
      </header>
      <MigrationWorkbench scenario={scenario} />
    </section>
  );
}
