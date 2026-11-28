import { Link } from "react-router-dom";
import { ShieldCheck } from "lucide-react";

import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
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
          <div data-testid={selectors.overview.setupCta}>
            <EmptyState
              icon={<ShieldCheck size={32} />}
              title={t(strings.overview.setupTitle)}
              description={t(strings.overview.setupBody)}
              action={
                <div className="flex flex-wrap justify-center gap-2">
                  <Link
                    to="/destinations"
                    className="inline-flex items-center justify-center rounded-control text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-60 bg-app-primary text-app-primary-foreground hover:brightness-95 h-11 px-5"
                  >
                    {t(strings.overview.setupCtaDestinations)}
                  </Link>
                  <Link
                    to="/plans"
                    className="inline-flex items-center justify-center rounded-control text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-60 border border-app-border text-app-foreground hover:bg-app-surface-muted h-11 px-5"
                  >
                    {t(strings.overview.setupCtaPlans)}
                  </Link>
                </div>
              }
            />
          </div>
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
