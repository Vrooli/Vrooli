import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FleetView } from "../features/fleet/FleetView";
import { useTranslation } from "../i18n";

/**
 * Fleet page — the cross-scenario structure dashboard. `FleetService.ScanFleet`
 * grades every discovered scenario and this page renders the rollup: who fails
 * structure gating, which rules offend the most scenarios, and how the fleet
 * breaks down by detected profile.
 */
export function FleetPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.fleet}
      aria-labelledby="fleet-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="fleet-heading" className="text-2xl font-semibold">
        {t(strings.pages.fleet.title)}
      </h2>
      <p className="text-app-muted-foreground">{t(strings.pages.fleet.description)}</p>
      <FleetView />
    </section>
  );
}
