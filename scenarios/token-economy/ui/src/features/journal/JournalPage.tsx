import { useMutation, useQuery } from "@tanstack/react-query";
import { Download, ShieldCheck, ShieldQuestion } from "lucide-react";
import { useState } from "react";
import { VerificationStatus, type Event } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { minterClient } from "../../api/tokenEconomy";
import { Button } from "@vrooli/react-component-library/Button/2";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1";
import { Input } from "../../components/ui/input";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsolePage, Field, RequestState } from "../console/ConsolePage";

export function JournalPage() {
  const { t } = useTranslation();
  const [holderId, setHolderId] = useState("");
  const [tokenTypeId, setTokenTypeId] = useState("");
  const events = useQuery({ queryKey: ["token-economy", "minter", "journal", holderId, tokenTypeId], queryFn: () => minterClient.listJournalEvents({ holderId, tokenTypeId }) });
  const exportJournal = useMutation({ mutationFn: () => minterClient.exportJournal({ holderId, tokenTypeId }), onSuccess: (result) => {
    const blob = new Blob([JSON.stringify(result.events, (_key: string, value: unknown): unknown => typeof value === "bigint" ? value.toString() : value, 2)], { type: "application/json" });
    const href = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = href;
    anchor.download = "token-economy-journal.json";
    anchor.click();
    URL.revokeObjectURL(href);
  } });
  const columns: Array<DataTableColumn<Event>> = [
    { id: "kind", header: t(strings.journal.kind), accessor: (row) => <StatusBadge tone={row.amount < 0n ? "warning" : "success"}>{row.kind}</StatusBadge>, sortValue: (row) => row.kind },
    { id: "amount", header: t(strings.common.amount), accessor: (row) => <span className="font-mono">{row.amount > 0n ? "+" : ""}{row.amount.toString()}</span>, sortValue: (row) => Number(row.amount) },
    { id: "reason", header: t(strings.journal.reason), accessor: (row) => row.reason, searchValue: (row) => row.reason },
    { id: "actor", header: t(strings.journal.actor), accessor: (row) => <span className="inline-flex items-center gap-2">{row.actorVerificationStatus === VerificationStatus.VERIFIED ? <ShieldCheck aria-hidden className="h-4 w-4 text-app-success" /> : <ShieldQuestion aria-hidden className="h-4 w-4 text-app-warning" />}<span data-testid={`journal-provenance-${row.id}`}>{row.actorIdentity || t(strings.journal.noActor)} · {row.actorVerificationStatus === VerificationStatus.VERIFIED ? t(strings.journal.verified) : t(strings.journal.unverified)}</span></span> },
    { id: "id", header: t(strings.common.identifier), accessor: (row) => <code>{row.id}</code>, searchValue: (row) => row.id },
  ];
  return (
    <ConsolePage testId="page-journal" title={t(strings.journal.title)} description={t(strings.journal.description)}>
      <div className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-4 md:grid-cols-[1fr_1fr_auto]">
        <Field label={t(strings.journal.holderFilter)}><Input data-testid="journal-holder-filter" value={holderId} onChange={(event) => setHolderId(event.target.value)} /></Field>
        <Field label={t(strings.journal.tokenFilter)}><Input data-testid="journal-token-filter" value={tokenTypeId} onChange={(event) => setTokenTypeId(event.target.value)} /></Field>
        <div className="flex items-end"><Button data-testid="journal-export" variant="secondary" disabled={exportJournal.isPending} onClick={() => exportJournal.mutate()}><Download aria-hidden className="h-4 w-4" />{t(strings.journal.export)}</Button></div>
      </div>
      <RequestState loading={events.isLoading} error={events.error ?? exportJournal.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={events.data?.events ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.journal.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.journal.empty)} tableTestId="journal-event-table" />
    </ConsolePage>
  );
}
