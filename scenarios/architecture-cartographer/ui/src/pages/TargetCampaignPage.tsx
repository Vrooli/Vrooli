import { Navigate } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useScenarioPath } from "../hooks/useScenarioPath";
import { CampaignWorkbench } from "../features/campaign/CampaignWorkbench";

export function TargetCampaignPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  if (scenario === null) return <Navigate to="/" replace />;

  return (
    <section
      data-testid={selectors.pages.targetCampaign}
      aria-labelledby="target-campaign-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-campaign-heading" className="text-xl font-semibold">
          {t(strings.pages.campaign.title)}
        </h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.pages.campaign.description)}</p>
      </header>
      <CampaignWorkbench scenario={scenario} />
    </section>
  );
}
