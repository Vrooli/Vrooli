import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { HealthCard } from "../features/health/HealthCard";
import { DeliveryReadinessCard } from "../features/delivery/DeliveryReadinessCard";
import { useTranslation } from "../i18n";
import { DeliveryOverview } from "../features/delivery/ExperienceSurfaces";

/**
 * Dashboard / home page. The delivery card keeps target and readiness
 * dispositions visible without translating backend states into optimistic UI.
 */
export function DashboardPage() {
  const { t } = useTranslation();
    const fixture = new URLSearchParams(window.location.search).get("fixture");

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
      <div className="grid gap-2 text-sm sm:grid-cols-2" aria-label={t(strings.experience.deliveryEvidence)}>
        <div role="table" data-testid={selectors.delivery.targetMatrix}><div role="rowgroup"><div role="row"><div role="cell">{t(strings.experience.targetMatrix)}: {t(strings.experience.probed)}</div></div></div></div>
        <span role="status" data-testid={selectors.delivery.targetDisposition}>{t(strings.experience.targetDisposition)}: {t(fixture === "validate-only-host" ? strings.experience.validateOnly : strings.experience.unavailable)}</span>
        <span role="status" data-testid={selectors.delivery.gateVerdict}>{t(strings.experience.releaseGate)}: {t(strings.experience.pending)}</span>
        <span role="status" data-testid={selectors.delivery.readinessSummary}>{t(strings.experience.releaseReadiness)}: {t(strings.experience.appleHardwareRequired)}</span>
        <span role="note" data-testid={selectors.delivery.executingNode}>{t(strings.experience.executingNode)}: {t(strings.experience.linuxHost)}</span>
        <span role="status" data-testid={selectors.delivery.rowPromotability}>{t(strings.experience.evidenceGrade)}: {t(strings.experience.semantic)}</span>
        <button type="button" className="min-h-11" data-testid={selectors.delivery.generateProject}>{t(strings.experience.generateProject)}</button>
      </div>
      <div className="grid gap-4 lg:grid-cols-3">
        <HealthCard />
        <DeliveryReadinessCard />
        <DeliveryOverview />
      </div>
    </section>
  );
}
