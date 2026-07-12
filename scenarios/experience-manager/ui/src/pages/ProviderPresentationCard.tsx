import type { SharedPhasePresentation } from "../api/experience";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

export function ProviderPresentationCard({
  presentation,
  loading,
  unavailable,
}: {
  presentation?: SharedPhasePresentation;
  loading: boolean;
  unavailable: boolean;
}) {
  const { t } = useTranslation();
  if (loading) {
    return <section aria-live="polite" className="rounded-panel border border-app-border bg-app-surface p-4">{t(strings.experience.findings.providerLoading)}</section>;
  }
  if (unavailable || !presentation) {
    return <section role="status" className="rounded-panel border border-app-border bg-app-surface p-4">{t(strings.experience.findings.providerUnavailable)}</section>;
  }
  if (presentation.contractVersion !== "v1") {
    return <section role="status" className="rounded-panel border border-app-border bg-app-surface p-4">{t(strings.experience.findings.providerHistorical, { version: presentation.contractVersion || "unknown" })}</section>;
  }
  return (
    <section aria-label={t(strings.experience.findings.providerMaturity)} className="rounded-panel border border-app-border bg-app-surface p-4">
      <h3 className="font-semibold">{t(strings.experience.findings.providerMaturity)}</h3>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.experience.findings.providerContract, { version: presentation.contractVersion })}</p>
      {presentation.northStar ? <p className="mt-3 text-sm"><strong>{t(strings.experience.findings.providerNorthStar)}</strong> {presentation.northStar}</p> : null}
      {presentation.nextAction ? <p className="mt-2 text-sm"><strong>{t(strings.experience.findings.providerNextAction)}</strong> {presentation.nextAction}</p> : null}
      <ul className="mt-3 grid gap-2" aria-label={t(strings.experience.findings.providerCapabilities)}>
        {(presentation.capabilities ?? []).map((capability) => (
          <li key={capability.id} className="rounded-control bg-app-surface-muted p-3 text-sm">
            <p className="font-medium">{capability.label || capability.id}</p>
            <p className="text-app-muted-foreground">{capability.currentLevelLabel || capability.currentLevel}{capability.nextLevel ? ` → ${capability.nextLevel}` : ""}</p>
            {capability.nextUnlock ? <p className="mt-1"><strong>{t(strings.experience.findings.providerNextAction)}</strong> {capability.nextUnlock}</p> : null}
          </li>
        ))}
      </ul>
      {(presentation.documentationTopics ?? []).length > 0 ? <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.experience.findings.providerDocumentation, { topics: (presentation.documentationTopics ?? []).join(" · ") })}</p> : null}
    </section>
  );
}
