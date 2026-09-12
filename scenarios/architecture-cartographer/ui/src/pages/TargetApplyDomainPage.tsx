import { Navigate, useParams } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { ApplyWorkspace } from "../features/apply/ApplyWorkspace";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { ErrorState } from "../components/ErrorState";

export function TargetApplyDomainPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  const params = useParams<{ domainKey: string }>();
  const domain = params.domainKey ? decodeURIComponent(params.domainKey) : "";

  if (scenario === null) return <Navigate to="/" replace />;
  if (domain.length === 0) return <Navigate to=".." replace />;

  return (
    <section
      data-testid={selectors.pages.targetApplyDomain}
      aria-labelledby="target-apply-domain-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-apply-domain-heading" className="text-xl font-semibold">
          {t(strings.pages.targetApply.title)} / <span className="font-mono">{domain}</span>
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetApply.description)}
        </p>
      </header>
      <ErrorBoundary
        fallback={
          <ErrorState title={t(strings.shared.error.title)} message="" />
        }
      >
        <ApplyWorkspace scenario={scenario} domain={domain} />
      </ErrorBoundary>
    </section>
  );
}
