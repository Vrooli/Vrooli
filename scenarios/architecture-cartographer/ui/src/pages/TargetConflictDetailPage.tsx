import { Link, Navigate, useParams } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { encodeScenarioPath, useScenarioPath } from "../hooks/useScenarioPath";
import { ConflictWorkbench } from "../features/conflicts/ConflictWorkbench";

export function TargetConflictDetailPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  const params = useParams<{ conflictId?: string }>();
  const conflictId = params.conflictId ? decodeURIComponent(params.conflictId) : "";

  if (scenario === null) return <Navigate to="/" replace />;
  if (conflictId.length === 0) {
    return <Navigate to={`/targets/${encodeScenarioPath(scenario)}/conflicts`} replace />;
  }

  return (
    <section
      data-testid={selectors.pages.targetConflictDetail}
      aria-labelledby="target-conflict-detail-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-conflict-detail-heading" className="text-xl font-semibold">
          {t(strings.pages.conflictDetail.title)}
        </h3>
        <Link
          to={`/targets/${encodeScenarioPath(scenario)}/conflicts`}
          data-testid={selectors.features.conflicts.detail.backLink}
          className="text-sm text-app-primary hover:underline"
        >
          {t(strings.pages.conflictDetail.backToList)}
        </Link>
      </header>
      <ConflictWorkbench scenario={scenario} conflictId={conflictId} />
    </section>
  );
}
