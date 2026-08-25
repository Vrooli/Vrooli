import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { EarningSubmission } from "@vrooli/proto-types/token-economy/v1/earning/earning_pb";
import { BadgePlus } from "lucide-react";
import { useState } from "react";

import { earningClient, nextIdempotencyKey } from "../../api/tokenEconomy";
import { DataTable, type DataTableColumn } from "@vrooli/react-component-library/DataTable/1.2.0";
import { Input } from "../../components/ui/input";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1.1.0";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsoleForm, ConsolePage, Field, RequestState } from "../console/ConsolePage";

const queryKey = ["token-economy", "earning", "submissions"] as const;

export function EarningPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [holderId, setHolderId] = useState("");
  const [tokenTypeId, setTokenTypeId] = useState("");
  const [amount, setAmount] = useState("1");
  const [reason, setReason] = useState("");
  const submissions = useQuery({ queryKey, queryFn: () => earningClient.listEarnings({}) });
  const submitEarning = useMutation({ mutationFn: () => earningClient.submitEarning({ holderId, tokenTypeId, amountMinor: BigInt(amount || "0"), reason, dedupKey: nextIdempotencyKey("operator-earning") }), onSuccess: () => queryClient.invalidateQueries({ queryKey }) });
  const columns: Array<DataTableColumn<EarningSubmission>> = [
    { id: "holder", header: t(strings.earning.holder), accessor: (row) => <code>{row.holderId}</code>, searchValue: (row) => row.holderId },
    { id: "amount", header: t(strings.common.amount), accessor: (row) => <span className="font-mono">+{row.amountMinor.toString()}</span>, sortValue: (row) => Number(row.amountMinor) },
    { id: "reason", header: t(strings.earning.reason), accessor: (row) => row.reason, searchValue: (row) => row.reason },
    { id: "path", header: t(strings.earning.path), accessor: (row) => <StatusBadge tone="info"><BadgePlus aria-hidden className="mr-1 h-3 w-3" />{row.adapterIdentity || t(strings.earning.operatorAdapter)}</StatusBadge> },
    { id: "replay", header: t(strings.common.status), accessor: (row) => <StatusBadge tone={row.replayed ? "warning" : "success"}>{row.replayed ? t(strings.earning.replayed) : t(strings.earning.recorded)}</StatusBadge> },
  ];
  return (
    <ConsolePage testId="page-earning" title={t(strings.earning.title)} description={t(strings.earning.description)}>
      <ConsoleForm title={t(strings.earning.submitTitle)} submitLabel={t(strings.earning.submit)} submitTestId="earning-submit" busy={submitEarning.isPending} onSubmit={(event) => { event.preventDefault(); submitEarning.mutate(); }}>
        <Field label={t(strings.earning.holder)}><Input data-testid="earning-holder" required value={holderId} onChange={(event) => setHolderId(event.target.value)} /></Field>
        <Field label={t(strings.earning.tokenType)}><Input data-testid="earning-token-type" required value={tokenTypeId} onChange={(event) => setTokenTypeId(event.target.value)} /></Field>
        <Field label={t(strings.common.amount)}><Input data-testid="earning-amount" min="1" required type="number" value={amount} onChange={(event) => setAmount(event.target.value)} /></Field>
        <Field label={t(strings.earning.reason)}><Input data-testid="earning-reason" required value={reason} onChange={(event) => setReason(event.target.value)} /></Field>
      </ConsoleForm>
      <p className="text-sm text-app-muted-foreground">{t(strings.earning.sharedPathNote)}</p>
      <RequestState loading={submissions.isLoading} error={submissions.error ?? submitEarning.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={submissions.data?.submissions ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.earning.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.earning.empty)} tableTestId="earning-table" />
    </ConsolePage>
  );
}
