import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { evaluateTriggers } from "../api/offers";

export function TriggersPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const parseError = fixture === "parse-error";
  const evaluation = useQuery({ queryKey: ["trigger-evaluation"], queryFn: () => evaluateTriggers(true), retry: false, enabled: !fixture });
  return (
    <ExperienceSurface surfaceId="triggers" state={parseError ? "error" : fixture === "empty" ? "empty" : "ready"} data-testid={selectors.pages.triggers} aria-labelledby="triggers-heading" className="flex flex-col gap-4">
      <h2 id="triggers-heading" className="text-2xl font-semibold">{t(strings.pages.triggers.title)}</h2>
      <Card data-testid={selectors.pages.triggerEditor}>
        <CardHeader><CardTitle>{t(strings.pages.triggers.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.triggers.description)}</p>
          <div data-testid={selectors.pages.triggerParseStatus} className="mt-4 rounded-md border p-3">{parseError ? t(strings.pages.triggers.parseError) : t(strings.pages.triggers.parseReady)}</div>
          <p data-testid={selectors.pages.triggerParseError} role="alert" className="text-sm text-app-danger">{parseError ? t(strings.pages.triggers.parseErrorDetail) : ""}</p>
          <button type="button" data-testid={selectors.pages.triggerDryRun} className="mt-3 rounded-control border px-3 py-2">{t(strings.pages.triggers.dryRunAction)}</button>
          <p data-testid={selectors.pages.triggerDryRunVerdict} className="text-sm">{t(strings.pages.triggers.dryRunVerdict)}</p>
          <p data-testid={selectors.pages.triggerFactTrace} role="note" aria-label={t(strings.pages.triggers.factTrace)} className="text-sm text-app-muted-foreground">{t(strings.pages.triggers.factTrace)}</p>
          <p data-testid={selectors.pages.triggerMissingFact} role="note" aria-label={t(strings.pages.triggers.missingFact)} className="text-sm text-app-muted-foreground">{t(strings.pages.triggers.missingFact)}</p>
          <p data-testid={selectors.pages.factRegistry} role="note" aria-label={t(strings.pages.triggers.factRegistry)} className="text-sm text-app-muted-foreground">{t(strings.pages.triggers.factRegistry)}</p>
          <p data-testid={selectors.pages.evaluationFreshness} role="note" aria-label={t(strings.pages.triggers.evaluationFreshness)} className="text-sm text-app-muted-foreground">{t(strings.pages.triggers.evaluationFreshness)}</p>
          <p data-testid={selectors.pages.evaluationStalledAlert} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.triggers.stalledAlert)}</p>
          {evaluation.data?.evaluations.map((item) => <p key={item.id} className="text-sm">{item.factName} · {item.verdict.toString()}</p>)}
          {fixture === "empty" && <p data-testid={selectors.pages.triggersEmptyGuidance} className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.triggers.emptyGuidance)}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
