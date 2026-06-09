import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { FleetView } from "../features/fleet/FleetView";
import { MeasuresCard } from "../features/measures/MeasuresCard";
import { useTranslation } from "../i18n";

/**
 * Fleet page — the measures-specific drill-down surface. The cross-scenario
 * coverage table plus a per-scenario domain breakdown answer "which scenarios
 * expose measures, at what tier/coverage". Per-scenario maturity rung is
 * automatic via the ecosystem-manager decision trace; this view is the
 * measures detail behind it.
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
      <MeasuresCard />
      <FleetView />
    </section>
  );
}
