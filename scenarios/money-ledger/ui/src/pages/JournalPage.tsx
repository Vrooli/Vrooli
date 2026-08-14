import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { configuredBookId, fetchBooks, fetchPostings } from "../api/ledger";

export function JournalPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const empty = fixture === "empty";
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false, enabled: !fixture });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const postings = useQuery({ queryKey: ["postings", bookId], queryFn: () => fetchPostings(bookId), retry: false, enabled: Boolean(bookId) && !fixture });
  return (
    <ExperienceSurface surfaceId="journal" state={empty ? "empty" : "ready"} data-testid={selectors.pages.journal} aria-labelledby="journal-heading" className="flex flex-col gap-4">
      <h2 id="journal-heading" className="text-2xl font-semibold">{t(strings.pages.journal.title)}</h2>
      <Card data-testid={selectors.pages.eventTable}>
        <CardHeader><CardTitle>{t(strings.pages.journal.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.journal.description)}</p>
          <div data-testid={selectors.pages.eventBasis} className="mt-4 rounded-md border p-3">{t(strings.pages.journal.eventBasis)}</div>
          <p data-testid={selectors.pages.eventSource} className="text-sm text-app-muted-foreground">{t(strings.pages.journal.eventSource)}</p>
          <p data-testid={selectors.pages.reversalLink} className="text-sm text-app-muted-foreground">{t(strings.pages.journal.reversalLink)}</p>
          <p data-testid={selectors.pages.reversalReason} className="text-sm text-app-muted-foreground">{t(strings.pages.journal.reversalReason)}</p>
          <p data-testid={selectors.pages.auditTrail} className="text-sm text-app-muted-foreground">{t(strings.pages.journal.auditTrail)}</p>
          <div data-testid={selectors.pages.manualEntry} className="mt-3 rounded-md bg-app-surface-muted p-3">{t(strings.pages.journal.manualEntry)}</div>
          {postings.data?.postings.map((posting) => <p key={posting.id} className="mt-2 text-sm">{posting.event?.description || posting.id} · {posting.event?.amountMinor.toString()}</p>)}
          {empty && <p data-testid={selectors.pages.emptyGuidance} className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.journal.emptyGuidance)}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
