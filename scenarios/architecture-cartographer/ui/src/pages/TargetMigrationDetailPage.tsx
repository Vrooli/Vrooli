import { Link, Navigate, useParams } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { encodeScenarioPath, useScenarioPath } from "../hooks/useScenarioPath";
import { MigrationWorkbench } from "../features/migration/MigrationWorkbench";

export function TargetMigrationDetailPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  const params = useParams<{ migrationId?: string }>();
  const migrationId = params.migrationId ? decodeURIComponent(params.migrationId) : "";

  if (scenario === null) return <Navigate to="/" replace />;
  if (migrationId.length === 0) {
    return <Navigate to={`/targets/${encodeScenarioPath(scenario)}/migration`} replace />;
  }

  return (
    <section
      data-testid={selectors.pages.targetMigrationDetail}
      aria-labelledby="target-migration-detail-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-migration-detail-heading" className="text-xl font-semibold">
          {t(strings.pages.migration.title)}
        </h3>
        <Link
          to={`/targets/${encodeScenarioPath(scenario)}/migration`}
          data-testid={selectors.features.migration.detail.backLink}
          className="text-sm text-app-primary hover:underline"
        >
          {t(strings.pages.migration.title)}
        </Link>
      </header>
      <MigrationWorkbench scenario={scenario} migrationId={migrationId} />
    </section>
  );
}
