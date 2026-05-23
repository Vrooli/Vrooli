import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { ManifestView } from "../features/manifest/ManifestView";

export function TargetManifestPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetManifest}
      aria-labelledby="target-manifest-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-manifest-heading" className="text-xl font-semibold">
          {t(strings.pages.targetManifest.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">
          {t(strings.pages.targetManifest.description)}
        </p>
      </header>
      <ManifestView scenario={scenario} />
    </section>
  );
}
