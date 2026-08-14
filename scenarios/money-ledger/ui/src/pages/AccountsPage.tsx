import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { configuredBookId, fetchAccounts, fetchBooks } from "../api/ledger";

export function AccountsPage() {
  const { t } = useTranslation();
  const fixture = new URLSearchParams(window.location.search).get("fixture");
  const empty = fixture === "empty";
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false, enabled: !fixture });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const accounts = useQuery({ queryKey: ["accounts", bookId], queryFn: () => fetchAccounts(bookId), retry: false, enabled: Boolean(bookId) && !fixture });
  return (
    <ExperienceSurface surfaceId="accounts" state={empty ? "empty" : "ready"} data-testid={selectors.pages.accounts} aria-labelledby="accounts-heading" className="flex flex-col gap-4">
      <h2 id="accounts-heading" className="text-2xl font-semibold">{t(strings.pages.accounts.title)}</h2>
      <Card data-testid={selectors.pages.bookList}>
        <CardHeader><CardTitle>{t(strings.pages.accounts.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.accounts.description)}</p>
          <div data-testid={selectors.pages.bookLabel} className="mt-4 rounded-md border p-3">{books.data?.books[0]?.name ?? t(strings.pages.accounts.title)}</div>
        </CardContent>
      </Card>
      <Card data-testid={selectors.pages.accountTable}>
        <CardContent className="pt-6">
          <p data-testid={selectors.pages.balanceBasis} className="text-sm text-app-muted-foreground">{t(strings.pages.accounts.balanceBasis)}</p>
          <p data-testid={selectors.pages.balanceGap} className="text-sm text-app-muted-foreground">{t(strings.pages.accounts.balanceGap)}</p>
          <p data-testid={selectors.pages.transferPair} className="text-sm text-app-muted-foreground">{t(strings.pages.accounts.transferPair)}</p>
          {accounts.data?.accounts.map((account) => <p key={account.id} className="mt-2 text-sm">{account.name} · {account.kind}</p>)}
          {empty && <p data-testid={selectors.pages.emptyGuidance} className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.accounts.emptyGuidance)}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
import { useQuery } from "@tanstack/react-query";
