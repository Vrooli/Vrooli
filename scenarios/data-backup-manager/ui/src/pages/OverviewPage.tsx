import { Link } from "react-router-dom";
import { ShieldCheck } from "lucide-react";

import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/EmptyState";
import { Button } from "../components/ui/button";
import { PostureBanner } from "../features/posture/PostureBanner";
import { StorageStrip } from "../features/overview/StorageStrip";
import { CoverageGrid } from "../features/overview/CoverageGrid";
import { CoverageBanner } from "../features/backup-coverage/CoverageBanner";
import { SuggestionsPanel } from "../features/discovery/SuggestionsPanel";
import { useDestinations } from "../hooks/useDestinations";
import { useTargets } from "../hooks/useTargets";
import { useDestinationSuggestions, useTargetSuggestions } from "../hooks/useSuggestions";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Overview — the operational landing surface. Posture banner first (is
 * everything protected, within cap, and verified?), then the storage strip and
 * the owner-grouped coverage grid. When nothing is set up yet, a single setup
 * call to action funnels the operator into destinations → plans.
 */
export function OverviewPage() {
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
    <section
      data-testid={selectors.pages.overview}
      aria-labelledby="overview-heading"
      className="flex flex-col gap-6"
    >
      <div id="overview-heading">
        <PageHeader title={t(strings.overview.title)} subtitle={t(strings.overview.subtitle)} />
      </div>

      <PostureBanner />

      {nothingConfigured ? (
        // Cold start: lead with discovered suggestions when we found any;
        // otherwise fall back to the manual setup call to action.
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
          <CoverageBanner />
          {hasSuggestions && <SuggestionsPanel />}
          <StorageStrip />
          <CoverageGrid />
        </>
      )}
    </section>
  );
}
