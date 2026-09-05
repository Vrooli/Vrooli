import { useQuery, useQueryClient } from "@tanstack/react-query";

import {
  getFrontier,
  listFacets,
  listPinProposals,
  resolvePinProposal,
} from "../../api/operator";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

export function OperatorReview() {
  const { t } = useTranslation();
  const client = useQueryClient();
  const facets = useQuery({ queryKey: ["operator", "facets"], queryFn: listFacets });
  const frontier = useQuery({ queryKey: ["operator", "frontier"], queryFn: getFrontier });
  const proposals = useQuery({ queryKey: ["operator", "pin-proposals"], queryFn: listPinProposals });
  const refresh = () => {
    void client.invalidateQueries({ queryKey: ["operator"] });
    void client.invalidateQueries({ queryKey: ["journal"] });
    void client.invalidateQueries({ queryKey: ["recall"] });
  };
  const error = facets.error ?? frontier.error ?? proposals.error;
  const frontierState: ExperienceSurfaceState = frontier.isLoading ? "loading" : frontier.error ? "error" : frontier.data?.nodes.length ? "ready" : "empty";
  const proposalState: ExperienceSurfaceState = proposals.isLoading ? "loading" : proposals.error ? "error" : proposals.data?.length ? "ready" : "empty";

  return (
    <section data-testid={selectors.pages.operator} aria-labelledby="operator-heading" className="flex flex-col gap-6">
      <div>
        <h2 id="operator-heading" className="text-2xl font-semibold">{t(strings.pages.operator.title)}</h2>
        <p className="text-app-muted-foreground">{t(strings.pages.operator.description)}</p>
      </div>
      {error && <p className="text-app-danger">{errorMessage(error, t)}</p>}
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.operator.frontierTitle)}</CardTitle></CardHeader>
        <CardContent>
          {frontier.data && <p className="mb-3 text-sm text-app-muted-foreground">{t(strings.pages.operator.frontierCount, { eligible: frontier.data.eligibleCount, target: frontier.data.target })}</p>}
          <ExperienceSurface surfaceId="frontier" state={frontierState} data-testid={selectors.operator.frontierList}>
          {frontier.data?.nodes.length === 0 && <EmptyState title={t(strings.pages.operator.frontierEmpty)} />}
          {frontier.data && frontier.data.nodes.length > 0 && <ul className="space-y-2">
            {frontier.data.nodes.map((node) => <li key={node.id} className="rounded-control border border-app-border p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="font-medium">{node.facetId || t(strings.pages.operator.unclassified)}</span>
                <span className="text-sm text-app-muted-foreground">{node.childIds.length > 0 ? t(strings.pages.operator.summary, { depth: node.depth, span: node.childIds.length }) : t(strings.pages.operator.leaf, { depth: node.depth })}</span>
              </div>
              <p className="mt-1 break-all text-xs text-app-muted-foreground">{node.entryId || node.id}</p>
            </li>)}
          </ul>}
          </ExperienceSurface>
        </CardContent>
      </Card>
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.operator.correctionTitle)}</CardTitle></CardHeader>
        <CardContent className="space-y-3">
          {facets.data?.length === 0 && <EmptyState title={t(strings.pages.operator.facetsEmpty)} />}
          {facets.data && facets.data.length > 0 && <p className="text-sm text-app-muted-foreground">{t(strings.pages.operator.facetCount, { count: facets.data.length })}</p>}
          <p className="text-sm">{t(strings.pages.operator.correctionHint)}</p>
          <Button size="sm" variant="secondary" onClick={refresh}>{t(strings.pages.operator.refresh)}</Button>
        </CardContent>
      </Card>
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.operator.proposalsTitle)}</CardTitle></CardHeader>
        <CardContent>
          <ExperienceSurface surfaceId="proposals" state={proposalState} data-testid={selectors.operator.proposalList}>
          {proposals.data?.length === 0 && <EmptyState title={t(strings.pages.operator.proposalsEmpty)} />}
          {proposals.data && proposals.data.length > 0 && <ul className="space-y-3">
            {proposals.data.map((proposal) => <li key={proposal.id} className="rounded-control border border-app-border p-3">
              <p>{proposal.rationale}</p>
              <p className="mt-1 text-xs text-app-muted-foreground">{proposal.entryIds.length} {t(strings.pages.operator.entries)}</p>
              <div className="mt-2 flex gap-2"><Button size="sm" onClick={() => void resolvePinProposal(proposal.id, true).then(refresh)}>{t(strings.pages.operator.accept)}</Button><Button size="sm" variant="secondary" onClick={() => void resolvePinProposal(proposal.id, false).then(refresh)}>{t(strings.pages.operator.reject)}</Button></div>
            </li>)}
          </ul>}
          </ExperienceSurface>
        </CardContent>
      </Card>
    </section>
  );
}
