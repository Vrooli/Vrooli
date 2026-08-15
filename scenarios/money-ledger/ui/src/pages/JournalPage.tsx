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
      <Card data-testid={selectors.pages.eventTable} role="table">
        <CardHeader><CardTitle>{t(strings.pages.journal.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.journal.description)}</p>
          <div data-testid={selectors.pages.eventBasis} role="status" className="mt-4 rounded-md border p-3">{t(strings.pages.journal.eventBasis)}</div>
          <p data-testid={selectors.pages.eventSource} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.journal.eventSource)}</p>
          <p data-testid={selectors.pages.reversalLink} role="link" className="flex min-h-11 items-center text-sm text-app-muted-foreground">{t(strings.pages.journal.reversalLink)}</p>
          <p data-testid={selectors.pages.reversalReason} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.journal.reversalReason)}</p>
          <button type="button" data-testid={selectors.pages.reverseAction} className="mt-3 min-h-11 rounded-control border px-3 py-2">{t(strings.pages.journal.reversalLink)}</button>
          <p data-testid={selectors.pages.auditTrail} role="list" className="text-sm text-app-muted-foreground">{t(strings.pages.journal.auditTrail)}</p>
          <form data-testid={selectors.pages.manualEntry} aria-label={t(strings.pages.journal.manualEntry)} className="mt-3 rounded-md bg-app-surface-muted p-3">{t(strings.pages.journal.manualEntry)}</form>
          {postings.data?.postings.map((posting) => <p key={posting.id} className="mt-2 text-sm">{posting.event?.description || posting.id} · {posting.event?.amountMinor.toString()}</p>)}
          {empty && <p data-testid={selectors.pages.journalEmptyGuidance} role="note" className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.journal.emptyGuidance)}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
