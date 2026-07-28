import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@tanstack/react-query";

import { listJournalEntries } from "../../api/journal";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../components/ui/card";
import { EmptyState } from "../../components/ui/empty-state";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { errorMessage } from "../../lib/errorMessage";

export function JournalTimeline() {
  const { t } = useTranslation();
  const query = useQuery({ queryKey: ["journal", "timeline"], queryFn: () => listJournalEntries() });
  return <Card data-testid={selectors.journal.surface} aria-label={t(strings.journal.title)}>
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
          <p className="mt-2 text-sm text-app-muted-foreground">{entry.facetId} · {entry.createdAt ? formatDate(timestampDate(entry.createdAt), { dateStyle: "medium", timeStyle: "short" }) : t(strings.journal.unknownTime)}</p>
        </li>)}
      </ol>}
    </CardContent>
  </Card>;
}
