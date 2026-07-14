import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FleetPanel } from "../features/fleet/FleetPanel";
import { OnboardNodeForm } from "../features/fleet/OnboardNodeForm";
import { PairNodeForm } from "../features/fleet/PairNodeForm";
import { HealthCard } from "../features/health/HealthCard";
import { RunHistory } from "../features/runs/RunHistory";
import { useTranslation } from "../i18n";

/**
 * Dashboard / home page — the fleet control-plane operator surface. It composes
 * the API health card, the pairing form (mint a code for a manual bootstrap),
 * the one-shot onboarding form (drive a raw SSH host to ONLINE with live step
 * states), the fleet panel (presence + OS/arch/version/health + live per-node
 * job status + revoke), and the run-history feed so the full operator flow —
 * onboard or pair a node, watch dispatch land, follow a run to completion — is
 * reachable without leaving the page.
 */
export function DashboardPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="dashboard-heading" className="text-2xl font-semibold">
        {t(strings.pages.dashboard.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      <div className="grid gap-4 md:grid-cols-2">
        <HealthCard />
        <PairNodeForm />
        <OnboardNodeForm />
        <FleetPanel />
        <RunHistory />
      </div>
    </section>
  );
}
