import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { listJournalEntries } from "../../api/journal";
import { assignFacet, listFacets, setPin } from "../../api/operator";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { ExperienceSurface, type ExperienceSurfaceState } from "../../components/experience/ExperienceSurface";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { errorMessage } from "../../lib/errorMessage";
import { Select } from "@vrooli/react-component-library/Select/1";

export function JournalTimeline() {
  const { t } = useTranslation();
  const query = useQuery({ queryKey: ["journal", "timeline"], queryFn: () => listJournalEntries() });
  const facets = useQuery({ queryKey: ["operator", "facets"], queryFn: listFacets });
  const [pinned, setPinned] = useState<Set<string>>(new Set());
  const client = useQueryClient();
  const correction = useMutation({
    mutationFn: async ({ entryId, facetId, pin }: { entryId: string; facetId?: string; pin?: boolean }) => {
      if (facetId) await assignFacet(entryId, facetId);
      if (pin !== undefined) await setPin(entryId, pin);
    },
    onSuccess: (_data, variables) => {
      if (variables.pin !== undefined) {
        setPinned((current) => {
          const next = new Set(current);
          if (variables.pin) next.add(variables.entryId);
          else next.delete(variables.entryId);
          return next;
        });
      }
      void client.invalidateQueries({ queryKey: ["journal"] });
      void client.invalidateQueries({ queryKey: ["recall"] });
      void client.invalidateQueries({ queryKey: ["operator"] });
    },
  });
  const experienceState: ExperienceSurfaceState = query.isLoading ? "loading" : query.error ? "error" : query.data?.length === 0 ? "empty" : "ready";
  return <ExperienceSurface surfaceId="timeline" state={experienceState} data-testid={selectors.journal.surface} aria-label={t(strings.journal.title)}>
    <Card>
    <CardHeader className="flex-row items-center justify-between gap-3">
      <CardTitle>{t(strings.journal.title)}</CardTitle>
      <Button size="sm" variant="secondary" onClick={() => void query.refetch()} disabled={query.isFetching}>{t(strings.journal.retry)}</Button>
    </CardHeader>
    <CardContent>
      {query.isLoading && <p data-testid={selectors.journal.loading}>{t(strings.journal.loading)}</p>}
      {query.error && <p data-testid={selectors.journal.error} className="text-app-danger">{errorMessage(query.error, t)}</p>}
      {query.data?.length === 0 && <div data-testid={selectors.journal.empty}><EmptyState title={t(strings.journal.empty)} /></div>}
      {query.data && query.data.length > 0 && <ol data-testid={selectors.journal.list} className="space-y-3">
        {query.data.map((entry) => <li key={entry.id} className="rounded-control border border-app-border p-3">
          <p className="whitespace-pre-wrap">{entry.body}</p>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-app-muted-foreground">
            <span>{entry.facetId} · {entry.createdAt ? formatDate(timestampDate(entry.createdAt), { dateStyle: "medium", timeStyle: "short" }) : t(strings.journal.unknownTime)}</span>
            <Select aria-label={t(strings.journal.facetCorrection)} value="" options={facets.data?.map((facet) => ({ value: facet.id, label: facet.label })) ?? []} placeholder={t(strings.journal.changeFacet)} onChange={(event) => correction.mutate({ entryId: entry.id, facetId: event.target.value })} disabled={correction.isPending || !facets.data?.length} className="min-h-8 w-auto py-1 text-xs" />
            <Button size="sm" variant="secondary" onClick={() => correction.mutate({ entryId: entry.id, pin: !pinned.has(entry.id) })} disabled={correction.isPending}>{pinned.has(entry.id) ? t(strings.journal.unpin) : t(strings.journal.pin)}</Button>
          </div>
          {correction.error && <p className="mt-2 text-sm text-app-danger">{errorMessage(correction.error, t)}</p>}
        </li>)}
      </ol>}
    </CardContent>
    </Card>
  </ExperienceSurface>;
}
