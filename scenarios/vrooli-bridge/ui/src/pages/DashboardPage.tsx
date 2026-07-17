import { useCallback, useState } from "react";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FleetPanel } from "../features/fleet/FleetPanel";
import { OnboardNodeForm } from "../features/fleet/OnboardNodeForm";
import { type OnboardingOp } from "../api/onboard";
import { PairNodeForm } from "../features/fleet/PairNodeForm";
import { HealthCard } from "../features/health/HealthCard";
import { RunHistory } from "../features/runs/RunHistory";
import { useTranslation } from "../i18n";

/**
 * Fleet-first home. The page leads with the fleet (the machines this control
 * plane runs work on) and the guided "Add a node" wizard, then the run-history
 * feed, then a demoted API health card. Manual code-based pairing is tucked into
 * a collapsed disclosure at the bottom for the rare operator who needs it — the
 * default path is Add a node.
 *
 * The fleet's "Add a node" call to action (its empty-state card, or the header
 * button once nodes exist) scrolls to and focuses the wizard, so a newcomer is
 * guided from "no machines" to a running node without hunting for the form.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const [retryTarget, setRetryTarget] = useState<OnboardingOp | null>(null);

  const handleAddNode = useCallback(() => {
    const section = document.getElementById("add-node");
    // scrollIntoView is unimplemented in jsdom; guard so tests never crash.
    if (section && typeof section.scrollIntoView === "function") {
      section.scrollIntoView({ behavior: "smooth", block: "start" });
    }
    document.getElementById("fleet-onboard-heading")?.focus();
  }, []);

  const handleRetryOnboarding = useCallback((op: OnboardingOp) => {
    setRetryTarget(op);
    requestAnimationFrame(() => document.getElementById("add-node")?.scrollIntoView({ behavior: "smooth", block: "start" }));
  }, []);

  return (
    <section
      data-testid={selectors.pages.dashboard}
      aria-labelledby="dashboard-heading"
      className="mx-auto flex w-full max-w-3xl flex-col gap-6"
    >
      <div className="flex flex-col gap-1">
        <h2 id="dashboard-heading" className="text-2xl font-semibold">
          {t(strings.pages.dashboard.title)}
        </h2>
        <p className="text-app-muted-foreground">{t(strings.pages.dashboard.description)}</p>
      </div>

      <FleetPanel onAddNode={handleAddNode} onRetryOnboarding={handleRetryOnboarding} />
      <OnboardNodeForm retryTarget={retryTarget} />
      <RunHistory />
      <HealthCard />

      <details className="rounded-panel border border-app-border bg-app-surface">
        <summary
          data-testid={selectors.fleet.pairing.disclosure}
          className="cursor-pointer list-none px-4 py-3 text-sm font-medium text-app-muted-foreground hover:text-app-foreground"
        >
          {t(strings.fleet.pairing.disclosureLabel)}
        </summary>
        <div className="flex flex-col gap-3 border-t border-app-border p-4">
          <p className="text-xs text-app-muted-foreground">{t(strings.fleet.pairing.disclosureNote)}</p>
          <PairNodeForm />
        </div>
      </details>
    </section>
  );
}
