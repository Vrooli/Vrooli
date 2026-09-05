import { Link, Navigate, useParams } from "react-router-dom";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { encodeScenarioPath, useScenarioPath } from "../hooks/useScenarioPath";
import { CampaignWorkbench } from "../features/campaign/CampaignWorkbench";

export function TargetCampaignDetailPage() {
  const { t } = useTranslation();
  const scenario = useScenarioPath();
  const params = useParams<{ campaignId?: string }>();
  const campaignId = params.campaignId ? decodeURIComponent(params.campaignId) : "";

  if (scenario === null) return <Navigate to="/" replace />;
  if (campaignId.length === 0) {
    return <Navigate to={`/targets/${encodeScenarioPath(scenario)}/campaign`} replace />;
  }

  return (
    <section
      data-testid={selectors.pages.targetCampaignDetail}
      aria-labelledby="target-campaign-detail-heading"
      className="flex flex-col gap-3"
    >
      <header className="flex flex-col gap-1">
        <h3 id="target-campaign-detail-heading" className="text-xl font-semibold">
          {t(strings.pages.campaign.title)}
        </h3>
        <Link
          to={`/targets/${encodeScenarioPath(scenario)}/campaign`}
          data-testid={selectors.features.campaign.detail.backLink}
          className="text-sm text-app-primary hover:underline"
        >
          {t(strings.pages.campaign.title)}
        </Link>
      </header>
      <CampaignWorkbench scenario={scenario} campaignId={campaignId} />
    </section>
  );
}
