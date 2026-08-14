import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fetchEdges, fetchNodes } from "../api/offers";

export function OffersPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const refused = fixture === "transition-refused" || fixture === "promotion-blocked";
  const nodes = useQuery({ queryKey: ["offer-nodes"], queryFn: fetchNodes, retry: false, enabled: !fixture });
  const edges = useQuery({ queryKey: ["offer-edges"], queryFn: fetchEdges, retry: false, enabled: !fixture });
  return (
    <ExperienceSurface surfaceId="offers" state={refused ? "error" : fixture === "empty" ? "empty" : "ready"} data-testid={selectors.pages.offers} aria-labelledby="offers-heading" className="flex flex-col gap-4">
      <h2 id="offers-heading" className="text-2xl font-semibold">{t(strings.pages.offers.title)}</h2>
      <Card data-testid={selectors.pages.offerGraph}>
        <CardHeader><CardTitle>{t(strings.pages.offers.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.offers.description)}</p>
          <div data-testid={selectors.pages.offerTable} className="mt-4 rounded-md border p-3">{t(strings.pages.offers.statusLabel)}</div>
          <p data-testid={selectors.pages.offerStatus} role="status" aria-label={t(strings.pages.offers.statusLabel)} className="text-sm">{t(strings.pages.offers.statusLabel)}</p>
          <p data-testid={selectors.pages.offerWaitingOn} role="note" aria-label={t(strings.pages.offers.waitingOn)} className="text-sm text-app-muted-foreground">{t(strings.pages.offers.waitingOn)}</p>
          <p data-testid={selectors.pages.offerLegalTransitions} role="note" aria-label={t(strings.pages.offers.legalTransitions)} className="text-sm text-app-muted-foreground">{t(strings.pages.offers.legalTransitions)}</p>
          <p data-testid={selectors.pages.offerRefusalReason} role="alert" className="text-sm text-app-danger">{refused ? t(strings.pages.offers.refusalReason) : ""}</p>
          <p data-testid={selectors.pages.offerRefusalRemedy} className="text-sm text-app-muted-foreground">{t(strings.pages.offers.refusalRemedy)}</p>
          <p data-testid={selectors.pages.offerAuditTrail} className="text-sm text-app-muted-foreground">{t(strings.pages.offers.auditTrail)}</p>
          <button type="button" data-testid={selectors.pages.offerPromote} className="mt-3 rounded-control border px-3 py-2">{t(strings.pages.offers.promoteAction)}</button>
          <p data-testid={selectors.pages.offerRoleRequirement} className="text-sm text-app-muted-foreground">{t(strings.pages.offers.roleRequirement)}</p>
          <p data-testid={selectors.pages.offerMembershipFinding} className="text-sm text-app-muted-foreground">{t(strings.pages.offers.membershipFinding)}</p>
          {nodes.data?.nodes.map((node) => <p key={node.id} className="text-sm">{node.name} · {node.status.toString()}</p>)}
          {edges.data?.edges.map((edge) => <p key={edge.id} className="text-sm text-app-muted-foreground">{edge.kind}</p>)}
          {fixture === "empty" && <p data-testid={selectors.pages.offersEmptyGuidance} className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.offers.emptyGuidance)}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
