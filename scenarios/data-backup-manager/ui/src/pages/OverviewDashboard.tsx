import { Link } from "react-router-dom";
import { ShieldCheck } from "lucide-react";

import { EmptyState } from "../components/EmptyState";
import { Button } from "../components/ui/button";
import { PostureBanner } from "../features/posture/PostureBanner";
import { StorageStrip } from "../features/overview/StorageStrip";
import { MetricsStrip } from "../features/overview/MetricsStrip";
import { CoverageGrid } from "../features/overview/CoverageGrid";
import { CoverageBanner } from "../features/backup-coverage/CoverageBanner";
import { SuggestionsPanel } from "../features/discovery/SuggestionsPanel";
import { useDestinations } from "../hooks/useDestinations";
import { useTargets } from "../hooks/useTargets";
import { useDestinationSuggestions, useTargetSuggestions } from "../hooks/useSuggestions";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/** Data-backed overview panels, loaded after the stable shell and heading. */
export function OverviewDashboard() {
  const { t } = useTranslation();
  const destinations = useDestinations();
  const targets = useTargets();
  const targetSuggestions = useTargetSuggestions();
  const destinationSuggestions = useDestinationSuggestions();

  const nothingConfigured =
    !destinations.isLoading &&
    !targets.isLoading &&
    (destinations.data?.length ?? 0) === 0 &&
    (targets.data?.length ?? 0) === 0;

  const hasSuggestions =
    (targetSuggestions.data?.length ?? 0) + (destinationSuggestions.data?.length ?? 0) > 0;

  return (
    <>
      <PostureBanner />
      {nothingConfigured ? (
        hasSuggestions ? (
          <SuggestionsPanel onboarding />
        ) : (
          <EmptyState
            icon={ShieldCheck}
            title={t(strings.overview.setupTitle)}
            description={t(strings.overview.setupBody)}
            data-testid={selectors.overview.setupCta}
            action={
              <div className="flex flex-wrap justify-center gap-2">
                <Button asChild>
                  <Link to="/destinations">{t(strings.overview.setupCtaDestinations)}</Link>
                </Button>
                <Button asChild variant="outline">
                  <Link to="/plans">{t(strings.overview.setupCtaPlans)}</Link>
                </Button>
              </div>
            }
          />
        )
      ) : (
        <>
          <CoverageBanner showComplete reserveSpace />
          <StorageStrip />
          <MetricsStrip />
          <CoverageGrid maxRows={6} />
        </>
      )}
    </>
  );
}

export default OverviewDashboard;
