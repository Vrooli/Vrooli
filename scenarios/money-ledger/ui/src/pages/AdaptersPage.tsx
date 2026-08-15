import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fetchAdapters } from "../api/ledger";

export function AdaptersPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const unavailable = fixture === "adapter-unavailable" || fixture === "credentials-missing";
  const adapters = useQuery({ queryKey: ["adapters"], queryFn: fetchAdapters, retry: false, enabled: !fixture });
  return (
    <ExperienceSurface surfaceId="adapters" state={unavailable ? "partial" : fixture === "empty" ? "empty" : "ready"} data-testid={selectors.pages.adapters} aria-labelledby="adapters-heading" className="flex flex-col gap-4">
      <h2 id="adapters-heading" className="text-2xl font-semibold">{t(strings.pages.adapters.title)}</h2>
      <Card data-testid={selectors.pages.adapterList} role="list">
        <CardHeader><CardTitle>{t(strings.pages.adapters.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.adapters.description)}</p>
          <div data-testid={selectors.pages.manualAdapterEntry} role="listitem" className="mt-4 rounded-md border p-3">{t(strings.pages.adapters.manualAdapter)}</div>
          <p data-testid={selectors.pages.adapterAvailability} role="status" className="mt-3">{unavailable ? t(strings.pages.adapters.unavailable) : t(strings.pages.adapters.available)}</p>
          <p data-testid={selectors.pages.failureReason} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.failureReason)}</p>
          <p data-testid={selectors.pages.lastSuccessAge} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.lastSuccessAge)}</p>
          <p data-testid={selectors.pages.missingImpact} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.missingImpact)}</p>
          <p data-testid={selectors.pages.credentialGap} role="alert" className="text-sm text-app-muted-foreground">{t(strings.pages.adapters.credentialGap)}</p>
          {adapters.data?.adapters.map((adapter) => <p key={adapter.id} className="text-sm">{adapter.name} · {adapter.enabled ? t(strings.pages.adapters.available) : t(strings.pages.adapters.unavailable)}</p>)}
          <p data-testid={selectors.pages.adaptersEmptyGuidance} role="note" className={fixture === "empty" ? "mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground" : "sr-only"}>{t(strings.pages.adapters.emptyGuidance)}</p>
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
