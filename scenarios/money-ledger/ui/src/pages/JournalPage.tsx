import { useMemo, useState, type ChangeEvent, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Basis, type Posting } from "@vrooli/proto-types/money-ledger/v1/shared/ledger_types_pb";

import { fetchAccounts, fetchAdapters, fetchBooks, fetchPostings, ingestEvent, registerManualAdapter, reversePosting } from "../api/ledger";
import { DirtyStateGuard } from "@vrooli/react-component-library/DirtyStateGuard/1";
import { FormSection } from "@vrooli/react-component-library/FormSection/1";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Dialog } from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { DataTable } from "@vrooli/react-component-library/DataTable/1";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { formatCurrency, formatDate } from "../i18n/format";
import { useTranslation } from "../i18n";
import { configuredBookId } from "../api/ledger";
import { useSurfaceState } from "../hooks/useSurfaceState";

const today = () => new Date().toISOString().slice(0, 10);
const newExternalId = () => `manual-${Date.now()}-${Math.random().toString(36).slice(2)}`;

type TimestampLike = { seconds: bigint | number; nanos?: number };

const timestampToDate = (timestamp?: TimestampLike) => {
  if (!timestamp) return undefined;
  return new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000));
};

const basisName = (basis: Basis | number) => {
  const name = Basis[basis];
  return typeof name === "string" ? name : String(basis);
};

const formatTimestamp = (timestamp?: TimestampLike) => {
  const date = timestampToDate(timestamp);
  return date ? formatDate(date) : "—";
};

export function JournalPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const books = useQuery({ queryKey: ["books"], queryFn: fetchBooks, retry: false });
  const bookId = configuredBookId() || books.data?.books[0]?.id || "";
  const accounts = useQuery({ queryKey: ["accounts", bookId], queryFn: () => fetchAccounts(bookId), retry: false, enabled: Boolean(bookId) });
  const postings = useQuery({ queryKey: ["postings", bookId], queryFn: () => fetchPostings(bookId), retry: false, enabled: Boolean(bookId) });
  const adapters = useQuery({ queryKey: ["adapters"], queryFn: fetchAdapters, retry: false });
  const [entryExternalId] = useState(newExternalId);
  const [eventForm, setEventForm] = useState({ date: today(), accountId: "", amountMinor: "", currency: "USD", description: "", externalId: entryExternalId });
  const [eventValidationError, setEventValidationError] = useState(false);
  const [eventMessage, setEventMessage] = useState("");
  const [selectedPosting, setSelectedPosting] = useState<Posting | null>(null);
  const [reversalReason, setReversalReason] = useState("");
  const [reversalValidationError, setReversalValidationError] = useState(false);
  const [reversalMessage, setReversalMessage] = useState("");

  const invalidateLedger = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["postings"] }),
      queryClient.invalidateQueries({ queryKey: ["accounts"] }),
      queryClient.invalidateQueries({ queryKey: ["all-accounts"] }),
      queryClient.invalidateQueries({ queryKey: ["position"] }),
      queryClient.invalidateQueries({ queryKey: ["statement"] }),
    ]);
  };

  const eventMutation = useMutation({
    mutationFn: async (input: { externalId: string; accountId: string; bookId: string; amountMinor: bigint; currency: string; occurredAt: Date; description: string }) => {
      const manualAdapter = adapters.data?.adapters.find((adapter) => adapter.id === "manual");
      if (manualAdapter && !manualAdapter.enabled) throw new Error("manual adapter is disabled");
      if (!manualAdapter) await registerManualAdapter();
      return ingestEvent(input);
    },
    onSuccess: async (response) => {
      await invalidateLedger();
      await queryClient.invalidateQueries({ queryKey: ["adapters"] });
      setEventValidationError(false);
      setEventMessage(response.duplicate ? t(strings.pages.journal.duplicateNotice) : `${t(strings.pages.journal.savedNotice)} · ${response.posting?.id ?? response.receipt?.id ?? ""}`);
    },
    onError: () => setEventValidationError(true),
  });

  const reversalMutation = useMutation({
    mutationFn: (input: { postingId: string; reason: string }) => reversePosting(input.postingId, input.reason),
    onSuccess: async (response) => {
      await invalidateLedger();
      setSelectedPosting(null);
      setReversalReason("");
      setReversalValidationError(false);
      setReversalMessage(`${t(strings.pages.journal.savedNotice)} · ${response.posting?.id ?? ""}`);
    },
    onError: () => setReversalValidationError(true),
  });

  const surface = useSurfaceState({
    query: { isLoading: books.isLoading || accounts.isLoading || postings.isLoading, isFetching: books.isFetching || accounts.isFetching || postings.isFetching, isError: books.isError || accounts.isError || postings.isError, error: books.error || accounts.error || postings.error },
    empty: Boolean(postings.data && postings.data.postings.length === 0),
  });
  const accountNames = useMemo(() => new Map((accounts.data?.accounts ?? []).map((account) => [account.id, account.name])), [accounts.data?.accounts]);
  const reversalByOriginal = useMemo(() => new Map((postings.data?.postings ?? []).filter((posting) => posting.reversalOf).map((posting) => [posting.reversalOf, posting.id])), [postings.data?.postings]);
  const postingRows = postings.data?.postings ?? [];
  const postingColumns = [
    { id: "date", header: t(strings.pages.journal.postingDate), accessor: (posting: typeof postingRows[number]) => formatTimestamp(posting.event?.occurredAt), searchValue: (posting: typeof postingRows[number]) => formatTimestamp(posting.event?.occurredAt), className: "break-words" },
    { id: "account", header: t(strings.pages.journal.postingAccount), accessor: (posting: typeof postingRows[number]) => posting.event ? accountNames.get(posting.event.accountId) ?? posting.event.accountId : posting.id, searchValue: (posting: typeof postingRows[number]) => posting.event ? accountNames.get(posting.event.accountId) ?? posting.event.accountId : posting.id, className: "break-words" },
    { id: "description", header: t(strings.pages.journal.descriptionLabel), accessor: (posting: typeof postingRows[number]) => posting.event?.description || "—", className: "break-words" },
    { id: "amount", header: t(strings.pages.journal.amountLabel), accessor: (posting: typeof postingRows[number]) => posting.event ? formatCurrency(Number(posting.event.amountMinor) / 100, posting.event.currency) : "—", className: "break-words tabular-nums" },
    { id: "currency", header: t(strings.pages.journal.postingCurrency), accessor: (posting: typeof postingRows[number]) => posting.event?.currency || "—", className: "break-words" },
    { id: "basis", header: t(strings.pages.journal.postingBasis), accessor: (posting: typeof postingRows[number]) => posting.event ? basisName(posting.event.basis) : "—", className: "break-words" },
    { id: "source", header: t(strings.pages.journal.sourceLabel), accessor: (posting: typeof postingRows[number]) => posting.event?.adapterId || "—", className: "break-words" },
    { id: "reversal", header: t(strings.pages.journal.reversalLink), accessor: (posting: typeof postingRows[number]) => { const reversalId = posting.reversalOf || reversalByOriginal.get(posting.id); const canReverse = !posting.reversalOf && !reversalByOriginal.has(posting.id); return reversalId ? <span>{t(strings.pages.journal.linkedReversal)}: {reversalId}</span> : canReverse ? <Button type="button" size="sm" variant="secondary" data-testid={`journal-reverse-${posting.id}`} onClick={() => { setSelectedPosting(posting); setReversalReason(""); setReversalValidationError(false); }}>{t(strings.pages.journal.reverseAction)}</Button> : "—"; }, className: "break-words" },
  ];
  const submitEvent = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setEventMessage("");
    let amount: bigint;
    try {
      amount = BigInt(eventForm.amountMinor);
    } catch {
      amount = 0n;
    }
    if (!bookId || !eventForm.accountId || !eventForm.date || !eventForm.externalId.trim() || !eventForm.currency.trim() || amount === 0n) {
      setEventValidationError(true);
      return;
    }
    setEventValidationError(false);
    eventMutation.mutate({
      externalId: eventForm.externalId.trim(),
      accountId: eventForm.accountId,
      bookId,
      amountMinor: amount,
      currency: eventForm.currency.trim().toUpperCase(),
      occurredAt: new Date(`${eventForm.date}T00:00:00Z`),
      description: eventForm.description.trim(),
    });
  };

  const submitReversal = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!selectedPosting || !reversalReason.trim()) {
      setReversalValidationError(true);
      return;
    }
    setReversalValidationError(false);
    reversalMutation.mutate({ postingId: selectedPosting.id, reason: reversalReason.trim() });
  };

  return (
    <ExperienceSurface surfaceId="journal" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.journal} aria-labelledby="journal-heading" className="flex flex-col gap-4">
      <h2 id="journal-heading" className="text-2xl font-semibold">{t(strings.pages.journal.title)}</h2>
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.journal.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.journal.description)}</p>
          <div data-testid={selectors.pages.eventBasis} role="status" className="mt-4 rounded-md border p-3">{t(strings.pages.journal.eventBasis)}</div>
          <p data-testid={selectors.pages.eventSource} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.journal.eventSource)}</p>
          <p className="mt-2 text-sm text-app-muted-foreground"><a href="#journal-table" data-testid={selectors.pages.reversalLink} className="inline-flex min-h-11 items-center">{t(strings.pages.journal.reversalLink)}</a></p>
          <p data-testid={selectors.pages.reversalReason} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.journal.reversalReason)}</p>
          {reversalMessage && <p role="status" className="mt-2 text-sm text-app-success">{reversalMessage}</p>}

          <div aria-label={t(strings.pages.journal.manualEntry)} className="mt-4">
            <DirtyStateGuard isDirty={Boolean(eventForm.accountId || eventForm.amountMinor || eventForm.description || eventForm.externalId !== entryExternalId)} protectUnload title={t(strings.pages.journal.manualEntry)} description={t(strings.pages.journal.eventBasis)}>
              <FormSection title={t(strings.pages.journal.manualEntry)} description={t(strings.pages.journal.eventSource)}>
                <form data-testid={selectors.pages.manualEntry} aria-label={t(strings.pages.journal.manualEntry)} className="grid gap-3" onSubmit={submitEvent}>
                  <label className="grid gap-1" htmlFor="journal-date"><span>{t(strings.pages.journal.dateLabel)}</span><Input id="journal-date" type="date" value={eventForm.date} onChange={(event) => setEventForm({ ...eventForm, date: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="journal-account"><span>{t(strings.pages.journal.accountLabel)}</span><Select id="journal-account" value={eventForm.accountId} onChange={(event: ChangeEvent<HTMLSelectElement>) => setEventForm({ ...eventForm, accountId: event.target.value })} options={(accounts.data?.accounts ?? []).map((account) => ({ value: account.id, label: `${account.name} · ${account.kind}` }))} placeholder={t(strings.pages.journal.accountLabel)} /></label>
                  <label className="grid gap-1" htmlFor="journal-amount"><span>{t(strings.pages.journal.signedAmountLabel)}</span><Input id="journal-amount" type="number" step="1" value={eventForm.amountMinor} onChange={(event) => setEventForm({ ...eventForm, amountMinor: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="journal-currency"><span>{t(strings.pages.journal.currencyLabel)}</span><Input id="journal-currency" value={eventForm.currency} onChange={(event) => setEventForm({ ...eventForm, currency: event.target.value })} maxLength={3} /></label>
                  <label className="grid gap-1" htmlFor="journal-description"><span>{t(strings.pages.journal.descriptionLabel)}</span><Textarea id="journal-description" value={eventForm.description} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setEventForm({ ...eventForm, description: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="journal-external-id"><span>{t(strings.pages.journal.externalIdLabel)}</span><Input id="journal-external-id" value={eventForm.externalId} onChange={(event) => setEventForm({ ...eventForm, externalId: event.target.value })} /></label>
                  <label className="grid gap-1" htmlFor="journal-basis"><span>{t(strings.pages.journal.basisLabel)}</span><Input id="journal-basis" value={basisName(Basis.OPERATOR_ASSERTED)} readOnly /></label>
                  {eventValidationError && <p role="alert" className="text-sm text-app-danger">{eventMutation.isError ? t(strings.pages.journal.requestError) : t(strings.pages.journal.validationError)}</p>}
                  {eventMessage && <p role="status" className="text-sm text-app-success">{eventMessage}</p>}
                  <Button type="submit" disabled={eventMutation.isPending || !bookId}>{t(strings.pages.journal.submitEntry)}</Button>
                </form>
              </FormSection>
            </DirtyStateGuard>
          </div>

          <ul data-testid={selectors.pages.auditTrail} className="mt-4 grid gap-2 text-sm text-app-muted-foreground">
            <li>{(postings.data?.postings ?? []).some((posting) => posting.audit.length > 0) ? t(strings.pages.journal.auditTrail) : t(strings.pages.journal.noAudit)}</li>
            {(postings.data?.postings ?? []).flatMap((posting) => posting.audit.map((audit) => (
              <li key={audit.id} className="rounded-md border p-2"><span className="font-medium">{audit.actor}</span> · {formatTimestamp(audit.createdAt)} · {audit.reason} · {audit.priorValue || "—"}</li>
            )))}
          </ul>

          <DataTable rows={postingRows} columns={postingColumns} getRowKey={(posting) => posting.id} caption={t(strings.pages.journal.cardTitle)} searchLabel={t(strings.pages.journal.postingAccount)} searchPlaceholder={t(strings.pages.journal.postingAccount)} emptyMessage={t(strings.pages.journal.emptyGuidance)} tableTestId={selectors.pages.eventTable} className="mt-4" />
          <p data-testid={selectors.pages.journalEmptyGuidance} role="note" className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.journal.emptyGuidance)}</p>
          <p className="mt-3 text-sm text-app-muted-foreground">{t(strings.pages.journal.reversalReason)}</p>
        </CardContent>
      </Card>

      <Dialog open={Boolean(selectedPosting)} title={t(strings.pages.journal.reversalDialogTitle)} description={t(strings.pages.journal.reversalDialogDescription)} onClose={() => setSelectedPosting(null)} closeLabel={t(strings.pages.journal.cancel)} footer={<div className="flex justify-end gap-2"><Button type="button" variant="secondary" onClick={() => setSelectedPosting(null)}>{t(strings.pages.journal.cancel)}</Button><Button type="submit" form="reverse-posting-form" disabled={reversalMutation.isPending}>{t(strings.pages.journal.confirmReversal)}</Button></div>}>
        <form id="reverse-posting-form" className="grid gap-3" onSubmit={submitReversal}>
          <p className="text-sm text-app-muted-foreground">{selectedPosting?.id}</p>
          <label className="grid gap-1" htmlFor="reversal-reason"><span>{t(strings.pages.journal.reversalReasonLabel)}</span><Textarea id="reversal-reason" value={reversalReason} onChange={(event: ChangeEvent<HTMLTextAreaElement>) => setReversalReason(event.target.value)} /></label>
          {reversalValidationError && <p role="alert" className="text-sm text-app-danger">{reversalMutation.isError ? t(strings.pages.journal.requestError) : t(strings.pages.journal.validationError)}</p>}
        </form>
      </Dialog>
    </ExperienceSurface>
  );
}
