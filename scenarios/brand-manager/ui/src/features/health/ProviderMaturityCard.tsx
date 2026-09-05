import { useQuery } from "@tanstack/react-query";

import { fetchBrandingValidation, sharedPresentationFromResponse } from "../../api/validation";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

// ProviderMaturityCard renders only the canonical PhasePresentation supplied by
// Brand Manager's ScenarioValidationService. Branding-specific reports stay on
// their domain surfaces; this card does not re-order or re-score their data.
export function ProviderMaturityCard() {
  const { t } = useTranslation();
  const query = useQuery({
    queryKey: ["brand-manager-provider-presentation"],
    queryFn: fetchBrandingValidation,
    staleTime: 60_000,
  });
  const presentation = sharedPresentationFromResponse(query.data);

  return (
    <section className="rounded-panel border border-app-border bg-app-surface p-4" aria-label={t(strings.health.providerMaturity)}>
      <h3 className="font-semibold">{t(strings.health.providerMaturity)}</h3>
      {query.isLoading ? <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.health.providerLoading)}</p> : null}
      {query.isError || (!query.isLoading && !presentation) ? <p role="status" className="mt-2 text-sm text-app-muted-foreground">{t(strings.health.providerUnavailable)}</p> : null}
      {presentation && presentation.contractVersion !== "v1" ? <p role="status" className="mt-2 text-sm text-app-muted-foreground">{t(strings.health.providerHistorical, { version: presentation.contractVersion || "unknown" })}</p> : null}
      {presentation?.contractVersion === "v1" ? (
        <div className="mt-2 text-sm">
          <p className="text-app-muted-foreground">{t(strings.health.providerContract, { version: presentation.contractVersion })}</p>
          {presentation.northStar ? <p className="mt-2"><strong>{t(strings.health.providerNorthStar)}</strong> {presentation.northStar}</p> : null}
          {presentation.nextAction ? <p className="mt-2"><strong>{t(strings.health.providerNextAction)}</strong> {presentation.nextAction}</p> : null}
          <ul className="mt-3 grid gap-2" aria-label={t(strings.health.providerCapabilities)}>
            {(presentation.capabilities ?? []).map((capability) => (
              <li key={capability.id} className="rounded-control bg-app-surface-muted p-3">
                <p className="font-medium">{capability.label || capability.id}</p>
                <p className="text-app-muted-foreground">{capability.currentLevelLabel || capability.currentLevel}{capability.nextLevel ? ` → ${capability.nextLevel}` : ""}</p>
                {capability.nextUnlock ? <p className="mt-1"><strong>{t(strings.health.providerNextAction)}</strong> {capability.nextUnlock}</p> : null}
              </li>
            ))}
          </ul>
          {(presentation.documentationTopics ?? []).length > 0 ? <p className="mt-3 text-xs text-app-muted-foreground">{t(strings.health.providerDocumentation, { topics: (presentation.documentationTopics ?? []).join(" · ") })}</p> : null}
        </div>
      ) : null}
    </section>
  );
}
