import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { configuredBookId, fetchBooks, fetchStatement } from "../api/ledger";

export function StatementsPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const empty = fixture === "empty";
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false, enabled: !fixture });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const statement = useQuery({ queryKey: ["statement", bookId], queryFn: () => fetchStatement(bookId), retry: false, enabled: Boolean(bookId) && !fixture });
  return (
    <ExperienceSurface surfaceId="statements" state={empty ? "empty" : "ready"} data-testid={selectors.pages.statements} aria-labelledby="statements-heading" className="flex flex-col gap-4">
      <h2 id="statements-heading" className="text-2xl font-semibold">{t(strings.pages.statements.title)}</h2>
      <Card data-testid={selectors.pages.statementFigures} role="table">
        <CardHeader><CardTitle>{t(strings.pages.statements.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.statements.description)}</p>
          <div data-testid={selectors.pages.periodSelector} role="form" className="mt-4 rounded-md border p-3">{t(strings.pages.statements.periodSelector)}</div>
          <p data-testid={selectors.pages.coverageNote} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.coverageNote)}</p>
          <p data-testid={selectors.pages.categoryBreakdown} role="table" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.categoryBreakdown)}</p>
          <p data-testid={selectors.pages.uncategorisedCount} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.uncategorisedCount)}</p>
          <p data-testid={selectors.pages.notTaxAdvice} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.statements.notTaxAdvice)}</p>
          {statement.data && <p className="text-sm tabular-nums">{statement.data.currency}: {statement.data.closingCashMinor.toString()}</p>}
          <button type="button" data-testid={selectors.pages.exportAction} className="mt-3 min-h-11 rounded-control border px-3 py-2">{t(strings.pages.statements.exportAction)}</button>
          <p data-testid={selectors.pages.statementsEmptyGuidance} role="note" className={empty ? "mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground" : "sr-only"}>{t(strings.pages.statements.emptyGuidance)}</p>
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
