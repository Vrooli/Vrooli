import { useMemo, useState, type ChangeEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { archiveBook, createAccount, createBook, fetchAccounts, fetchBooks, fetchPostings, transfer } from "../api/ledger";
import { DirtyStateGuard } from "@vrooli/react-component-library/DirtyStateGuard/1";
import { FormSection } from "@vrooli/react-component-library/FormSection/1";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { DataTable } from "@vrooli/react-component-library/DataTable/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { formatCurrency } from "../i18n/format";
import { useTranslation } from "../i18n";
import { configuredBookId } from "../api/ledger";
import { useSurfaceState } from "../hooks/useSurfaceState";

const today = () => new Date().toISOString().slice(0, 10);

export function AccountsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false });
  const [selectedBookId, setSelectedBookId] = useState(configuredBookId());
  const bookId = selectedBookId || books.data?.books[0]?.id || "";
  const accounts = useQuery({ queryKey: ["accounts", bookId], queryFn: () => fetchAccounts(bookId), retry: false, enabled: Boolean(bookId) });
  const postings = useQuery({ queryKey: ["postings", bookId], queryFn: () => fetchPostings(bookId), retry: false, enabled: Boolean(bookId) });
  const bookIds = useMemo(() => books.data?.books.map((book) => book.id) ?? [], [books.data?.books]);
  const allAccounts = useQuery({
    queryKey: ["all-accounts", bookIds],
    queryFn: async () => (await Promise.all(bookIds.map((id) => fetchAccounts(id)))).flatMap((response) => response.accounts),
    retry: false,
    enabled: bookIds.length > 0,
  });

  const [bookForm, setBookForm] = useState({ name: "", currency: "USD" });
  const [accountForm, setAccountForm] = useState({ name: "", kind: "ASSET" });
  const [transferForm, setTransferForm] = useState({ fromAccountId: "", toAccountId: "", amountMinor: "", currency: "USD", description: "", date: today() });
  const [bookMessage, setBookMessage] = useState("");
  const [accountMessage, setAccountMessage] = useState("");
  const [transferMessage, setTransferMessage] = useState("");
  const [bookError, setBookError] = useState(false);
  const [accountError, setAccountError] = useState(false);
  const [transferError, setTransferError] = useState(false);

  const invalidateLedger = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["books"] }),
      queryClient.invalidateQueries({ queryKey: ["accounts"] }),
      queryClient.invalidateQueries({ queryKey: ["all-accounts"] }),
      queryClient.invalidateQueries({ queryKey: ["postings"] }),
      queryClient.invalidateQueries({ queryKey: ["position"] }),
    ]);
  };

  const createBookMutation = useMutation({
    mutationFn: (input: { name: string; currency: string }) => createBook(input.name, input.currency),
    onSuccess: async (response) => {
      await invalidateLedger();
      if (response.book?.id) setSelectedBookId(response.book.id);
      setBookForm({ name: "", currency: "USD" });
      setBookError(false);
      setBookMessage(t(strings.pages.accounts.savedNotice));
    },
    onError: () => setBookError(true),
  });

  const createAccountMutation = useMutation({
    mutationFn: (input: { bookId: string; name: string; kind: string }) => createAccount(input.bookId, input.name, input.kind),
    onSuccess: async () => {
      await invalidateLedger();
      setAccountForm({ name: "", kind: "ASSET" });
      setAccountError(false);
      setAccountMessage(t(strings.pages.accounts.savedNotice));
    },
    onError: () => setAccountError(true),
  });

  const archiveBookMutation = useMutation({
    mutationFn: (id: string) => archiveBook(id),
    onSuccess: async () => {
      await invalidateLedger();
      setSelectedBookId("");
    },
  });

  const transferMutation = useMutation({
    mutationFn: (input: { fromAccountId: string; toAccountId: string; amountMinor: bigint; currency: string; externalId: string; description: string; occurredAt: Date }) => transfer(input),
    onSuccess: async () => {
      await invalidateLedger();
      setTransferForm((current) => ({ ...current, amountMinor: "", description: "" }));
      setTransferError(false);
      setTransferMessage(t(strings.pages.accounts.savedNotice));
    },
    onError: () => setTransferError(true),
  });

  const surface = useSurfaceState({
    query: {
      isLoading: books.isLoading || accounts.isLoading || postings.isLoading,
      isFetching: books.isFetching || accounts.isFetching || postings.isFetching,
      isError: books.isError || accounts.isError || postings.isError,
      error: books.error || accounts.error || postings.error,
    },
    empty: Boolean(accounts.data && accounts.data.accounts.length === 0),
  });
  const selectedBook = books.data?.books.find((book) => book.id === bookId);
  const balances = new Map<string, bigint>();
  postings.data?.postings.forEach((posting) => {
    const accountId = posting.event?.accountId;
    if (!accountId || !posting.event) return;
    balances.set(accountId, (balances.get(accountId) ?? 0n) + posting.event.amountMinor);
  });
  const accountOptions = (allAccounts.data ?? []).map((account) => ({ value: account.id, label: `${account.name} · ${account.kind}` }));
  const accountRows = accounts.data?.accounts ?? [];
  const accountColumns = [
    { id: "account", header: t(strings.pages.accounts.accountLabel), accessor: (account: typeof accountRows[number]) => account.name, searchValue: (account: typeof accountRows[number]) => account.name, className: "break-words" },
    { id: "kind", header: t(strings.pages.accounts.kindLabel), accessor: (account: typeof accountRows[number]) => account.kind, searchValue: (account: typeof accountRows[number]) => account.kind, className: "break-words" },
    { id: "balance", header: t(strings.pages.accounts.balanceLabel), accessor: (account: typeof accountRows[number]) => { const balance = balances.get(account.id) ?? 0n; return !postings.isSuccess || !selectedBook ? t(strings.pages.accounts.balanceUnavailable) : formatCurrency(Number(balance) / 100, selectedBook.currency); }, className: "break-words tabular-nums" },
    { id: "basis", header: t(strings.pages.accounts.basisLabel), accessor: () => postings.isSuccess ? t(strings.pages.accounts.balanceBasis) : t(strings.pages.accounts.balanceUnavailable), className: "break-words" },
  ];

  const submitBook = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setBookMessage("");
    if (!bookForm.name.trim() || !bookForm.currency.trim()) {
      setBookError(true);
      return;
    }
    setBookError(false);
    createBookMutation.mutate({ name: bookForm.name.trim(), currency: bookForm.currency.trim().toUpperCase() });
  };
  const submitAccount = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setAccountMessage("");
    if (!bookId || !accountForm.name.trim() || !accountForm.kind.trim()) {
      setAccountError(true);
      return;
    }
    setAccountError(false);
    createAccountMutation.mutate({ bookId, name: accountForm.name.trim(), kind: accountForm.kind.trim().toUpperCase() });
  };
  const submitTransfer = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setTransferMessage("");
    let amount: bigint;
    try {
      amount = BigInt(transferForm.amountMinor);
    } catch {
      amount = 0n;
    }
    if (!transferForm.fromAccountId || !transferForm.toAccountId || transferForm.fromAccountId === transferForm.toAccountId || amount <= 0n || !transferForm.currency.trim() || !transferForm.date) {
      setTransferError(true);
      return;
    }
    setTransferError(false);
    transferMutation.mutate({
      fromAccountId: transferForm.fromAccountId,
      toAccountId: transferForm.toAccountId,
      amountMinor: amount,
      currency: transferForm.currency.trim().toUpperCase(),
      externalId: `transfer-${Date.now()}`,
      description: transferForm.description.trim(),
      occurredAt: new Date(`${transferForm.date}T00:00:00Z`),
    });
  };

  return (
    <ExperienceSurface surfaceId="accounts" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.accounts} aria-labelledby="accounts-heading" className="flex flex-col gap-4">
      <h2 id="accounts-heading" className="text-2xl font-semibold">{t(strings.pages.accounts.title)}</h2>
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.accounts.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.accounts.description)}</p>
          <p data-testid={selectors.pages.bookLabel} role="status" className="mt-2 text-sm text-app-muted-foreground">
            {selectedBook ? `${t(strings.pages.accounts.selectBook)}: ${selectedBook.name} · ${selectedBook.currency}` : t(strings.pages.accounts.selectBook)}
          </p>
          <ul data-testid={selectors.pages.bookList} aria-label={t(strings.pages.accounts.cardTitle)} className="mt-4 grid gap-2 sm:grid-cols-2">
            {books.data?.books.map((book) => (
              <li key={book.id} className="rounded-md border p-3">
                <Button type="button" variant="ghost" className="min-h-11 w-full justify-start text-left" aria-pressed={book.id === bookId} onClick={() => setSelectedBookId(book.id)}>
                  <span className="font-medium">{book.name}</span>
                  <span className="mt-1 block text-sm text-app-muted-foreground">{book.currency} · {book.id}</span>
                </Button>
              </li>
            ))}
          </ul>
          {selectedBook && <div className="mt-3 flex flex-wrap items-center gap-2">
            <Button type="button" data-testid="book-archive-control" variant="secondary" disabled={archiveBookMutation.isPending} onClick={() => archiveBookMutation.mutate(selectedBook.id)}>
              {t(strings.pages.accounts.archiveBookAction)}
            </Button>
            <span data-testid="archived-book-notice" role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.accounts.archivedBookNotice)}</span>
          </div>}
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-2">
        <DirtyStateGuard isDirty={Boolean(bookForm.name || bookForm.currency !== "USD")} protectUnload title={t(strings.pages.accounts.createBookTitle)} description={t(strings.pages.accounts.description)}>
          <FormSection title={t(strings.pages.accounts.createBookTitle)}>
            <form className="grid gap-3" onSubmit={submitBook}>
              <label className="grid gap-1" htmlFor="book-name"><span>{t(strings.pages.accounts.bookNameLabel)}</span><Input id="book-name" value={bookForm.name} onChange={(event) => setBookForm({ ...bookForm, name: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="book-currency"><span>{t(strings.pages.accounts.currencyLabel)}</span><Input id="book-currency" value={bookForm.currency} onChange={(event) => setBookForm({ ...bookForm, currency: event.target.value })} maxLength={3} /></label>
              {bookError && <p role="alert" className="text-sm text-app-danger">{createBookMutation.isError ? t(strings.pages.accounts.requestError) : t(strings.pages.accounts.validationError)}</p>}
              {bookMessage && <p role="status" className="text-sm text-app-success">{bookMessage}</p>}
              <Button type="submit" disabled={createBookMutation.isPending}>{t(strings.pages.accounts.createBookAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>

        <DirtyStateGuard isDirty={Boolean(accountForm.name || accountForm.kind !== "ASSET")} protectUnload title={t(strings.pages.accounts.createAccountTitle)} description={t(strings.pages.accounts.description)}>
          <FormSection title={t(strings.pages.accounts.createAccountTitle)}>
            <form className="grid gap-3" onSubmit={submitAccount}>
              <label className="grid gap-1" htmlFor="account-book"><span>{t(strings.pages.accounts.selectBook)}</span><Select id="account-book" value={bookId} onChange={(event: ChangeEvent<HTMLSelectElement>) => setSelectedBookId(event.target.value)} options={(books.data?.books ?? []).map((book) => ({ value: book.id, label: `${book.name} · ${book.currency}` }))} placeholder={t(strings.pages.accounts.selectBook)} /></label>
              <label className="grid gap-1" htmlFor="account-name"><span>{t(strings.pages.accounts.accountNameLabel)}</span><Input id="account-name" value={accountForm.name} onChange={(event) => setAccountForm({ ...accountForm, name: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="account-kind"><span>{t(strings.pages.accounts.accountKindLabel)}</span><Input id="account-kind" value={accountForm.kind} onChange={(event) => setAccountForm({ ...accountForm, kind: event.target.value.toUpperCase() })} aria-describedby="account-kind-vocabulary" /></label>
              <span id="account-kind-vocabulary" data-testid="account-kind-vocabulary" role="group" className="text-xs text-app-muted-foreground">{t(strings.pages.accounts.acceptedKinds)}</span>
              {accountError && <p role="alert" className="text-sm text-app-danger">{createAccountMutation.isError ? t(strings.pages.accounts.requestError) : t(strings.pages.accounts.validationError)}</p>}
              {accountMessage && <p role="status" className="text-sm text-app-success">{accountMessage}</p>}
              <Button type="submit" disabled={createAccountMutation.isPending || !bookId}>{t(strings.pages.accounts.createAccountAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
      </div>

      <Card>
        <CardContent className="pt-6">
          <p data-testid={selectors.pages.balanceBasis} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.accounts.balanceBasis)}</p>
          <p data-testid={selectors.pages.balanceGap} role="status" className="text-sm text-app-muted-foreground">{t(strings.pages.accounts.balanceGap)}</p>
          <Button type="button" variant="ghost" data-testid={selectors.pages.transferPair} className="mt-3 text-left" onClick={() => document.getElementById("transfer-form")?.scrollIntoView({ behavior: "smooth" })}>{t(strings.pages.accounts.transferPair)}</Button>
          <DataTable rows={accountRows} columns={accountColumns} getRowKey={(account) => account.id} caption={t(strings.pages.accounts.cardTitle)} searchLabel={t(strings.pages.accounts.accountLabel)} searchPlaceholder={t(strings.pages.accounts.accountLabel)} emptyMessage={t(strings.pages.accounts.emptyGuidance)} tableTestId={selectors.pages.accountTable} className="mt-4" />
          <p data-testid={selectors.pages.accountsEmptyGuidance} role="note" className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.accounts.emptyGuidance)}</p>
        </CardContent>
      </Card>

      <DirtyStateGuard isDirty={Boolean(transferForm.fromAccountId || transferForm.toAccountId || transferForm.amountMinor || transferForm.description)} protectUnload title={t(strings.pages.accounts.transferTitle)} description={t(strings.pages.accounts.transferPair)}>
        <div id="transfer-form">
          <FormSection title={t(strings.pages.accounts.transferTitle)}>
            <form className="grid gap-3" onSubmit={submitTransfer}>
              <label className="grid gap-1" htmlFor="transfer-from"><span>{t(strings.pages.accounts.fromAccountLabel)}</span><Select id="transfer-from" value={transferForm.fromAccountId} onChange={(event: ChangeEvent<HTMLSelectElement>) => setTransferForm({ ...transferForm, fromAccountId: event.target.value })} options={accountOptions} placeholder={t(strings.pages.accounts.fromAccountLabel)} /></label>
              <label className="grid gap-1" htmlFor="transfer-to"><span>{t(strings.pages.accounts.toAccountLabel)}</span><Select id="transfer-to" value={transferForm.toAccountId} onChange={(event: ChangeEvent<HTMLSelectElement>) => setTransferForm({ ...transferForm, toAccountId: event.target.value })} options={accountOptions} placeholder={t(strings.pages.accounts.toAccountLabel)} /></label>
              <label className="grid gap-1" htmlFor="transfer-amount"><span>{t(strings.pages.accounts.transferAmountLabel)}</span><Input id="transfer-amount" type="number" min="1" step="1" value={transferForm.amountMinor} onChange={(event) => setTransferForm({ ...transferForm, amountMinor: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="transfer-currency"><span>{t(strings.pages.accounts.currencyLabel)}</span><Input id="transfer-currency" value={transferForm.currency} onChange={(event) => setTransferForm({ ...transferForm, currency: event.target.value })} maxLength={3} /></label>
              <label className="grid gap-1" htmlFor="transfer-date"><span>{t(strings.pages.journal.dateLabel)}</span><Input id="transfer-date" type="date" value={transferForm.date} onChange={(event) => setTransferForm({ ...transferForm, date: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="transfer-description"><span>{t(strings.pages.accounts.transferDescriptionLabel)}</span><Textarea id="transfer-description" value={transferForm.description} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setTransferForm({ ...transferForm, description: event.target.value })} /></label>
              {transferError && <p role="alert" className="text-sm text-app-danger">{transferMutation.isError ? t(strings.pages.accounts.requestError) : t(strings.pages.accounts.validationError)}</p>}
              {transferMessage && <p role="status" className="text-sm text-app-success">{transferMessage}</p>}
              <Button type="submit" disabled={transferMutation.isPending}>{t(strings.pages.accounts.transferAction)}</Button>
            </form>
          </FormSection>
        </div>
      </DirtyStateGuard>
    </ExperienceSurface>
  );
}
