import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { DomainMapView } from "../features/domains/DomainMapView";

export function TargetDomainsPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetDomains}
      aria-labelledby="target-domains-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-domains-heading" className="text-xl font-semibold">
          {t(strings.pages.targetDomains.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetDomains.description)}
        </p>
      </header>
      <DomainMapView scenario={scenario} />
    </section>
  );
}
