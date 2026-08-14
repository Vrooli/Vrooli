import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { fetchNodes } from "../api/offers";
import { useQuery } from "@tanstack/react-query";

export function ProposalsPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const empty = fixture === "empty";
  const nodes = useQuery({ queryKey: ["proposal-nodes"], queryFn: fetchNodes, retry: false, enabled: !fixture });
  return (
    <ExperienceSurface surfaceId="proposals" state={empty ? "empty" : "ready"} data-testid={selectors.pages.proposals} aria-labelledby="proposals-heading" className="flex flex-col gap-4">
      <h2 id="proposals-heading" className="text-2xl font-semibold">{t(strings.pages.proposals.title)}</h2>
      <Card data-testid={selectors.pages.proposalList}>
        <CardHeader><CardTitle>{t(strings.pages.proposals.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.proposals.description)}</p>
          <p data-testid={selectors.pages.proposalProposer} className="mt-4 text-sm">{nodes.data?.nodes[0]?.name ?? t(strings.pages.proposals.proposer)}</p>
          <p data-testid={selectors.pages.proposalEvidence} className="text-sm text-app-muted-foreground">{t(strings.pages.proposals.evidence)}</p>
          <p data-testid={selectors.pages.proposalEffect} className="text-sm text-app-muted-foreground">{t(strings.pages.proposals.effect)}</p>
          <p data-testid={selectors.pages.proposalDeclineHistory} className="text-sm text-app-muted-foreground">{t(strings.pages.proposals.declineHistory)}</p>
          <button type="button" data-testid={selectors.pages.proposalAccept} className="mt-3 rounded-control border px-3 py-2">{t(strings.pages.proposals.acceptAction)}</button>
          <p data-testid={selectors.pages.proposalOperatorOnly} className="text-sm text-app-muted-foreground">{t(strings.pages.proposals.operatorOnly)}</p>
          {empty && <p data-testid={selectors.pages.proposalsEmptyGuidance} className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.proposals.emptyGuidance)}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
