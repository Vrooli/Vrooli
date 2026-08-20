import { create } from "@bufbuild/protobuf";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApprovalPosture, CatalogEntrySchema, type CatalogEntry } from "@vrooli/proto-types/token-economy/v1/access/access_pb";
import { Gift, PauseCircle } from "lucide-react";
import { useState } from "react";

import { minterClient, nextIdempotencyKey } from "../../api/tokenEconomy";
import { Button } from "../../components/ui/button";
import { DataTable, type DataTableColumn } from "../../components/ui/data-table";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";
import { StatusBadge } from "../../components/ui/status-badge";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsoleForm, ConsolePage, Field, RequestState } from "../console/ConsolePage";

const queryKey = ["token-economy", "minter", "catalog"] as const;

export function CatalogPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [tokenTypeId, setTokenTypeId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [cost, setCost] = useState("1");
  const [posture, setPosture] = useState(String(ApprovalPosture.IMMEDIATE));
  const entries = useQuery({ queryKey, queryFn: () => minterClient.listCatalogEntries({ includeRetired: true }) });
  const refresh = () => queryClient.invalidateQueries({ queryKey });
  const createEntry = useMutation({ mutationFn: () => minterClient.createCatalogEntry({ entry: create(CatalogEntrySchema, { tokenTypeId, title, description, costAmount: BigInt(cost || "0"), approvalPosture: Number(posture) }), idempotencyKey: nextIdempotencyKey("catalog") }), onSuccess: refresh });
  const retireEntry = useMutation({ mutationFn: (id: string) => minterClient.retireCatalogEntry({ id, idempotencyKey: nextIdempotencyKey("catalog-retire") }), onSuccess: refresh });
  const columns: Array<DataTableColumn<CatalogEntry>> = [
    { id: "title", header: t(strings.common.name), accessor: (row) => <span className="inline-flex items-center gap-2"><Gift aria-hidden className="h-4 w-4" />{row.title}</span>, sortValue: (row) => row.title },
    { id: "cost", header: t(strings.catalog.cost), accessor: (row) => <span className="font-mono">{row.costAmount.toString()}</span>, sortValue: (row) => Number(row.costAmount) },
    { id: "approval", header: t(strings.catalog.approval), accessor: (row) => <StatusBadge tone={row.approvalPosture === ApprovalPosture.REQUIRES_APPROVAL ? "warning" : "success"}>{row.approvalPosture === ApprovalPosture.REQUIRES_APPROVAL ? t(strings.catalog.needsApproval) : t(strings.catalog.immediate)}</StatusBadge> },
    { id: "actions", header: t(strings.common.actions), accessor: (row) => row.retired ? <StatusBadge>{t(strings.common.retired)}</StatusBadge> : <Button data-testid={`catalog-retire-${row.id}`} size="sm" variant="danger" onClick={() => retireEntry.mutate(row.id)}><PauseCircle aria-hidden className="h-4 w-4" />{t(strings.common.retire)}</Button> },
  ];
  return (
    <ConsolePage testId="page-catalog" title={t(strings.catalog.title)} description={t(strings.catalog.description)}>
      <ConsoleForm title={t(strings.catalog.createTitle)} submitLabel={t(strings.catalog.create)} submitTestId="catalog-create" busy={createEntry.isPending} onSubmit={(event) => { event.preventDefault(); createEntry.mutate(); }}>
        <Field label={t(strings.catalog.tokenType)}><Input data-testid="catalog-token-type" required value={tokenTypeId} onChange={(event) => setTokenTypeId(event.target.value)} /></Field>
        <Field label={t(strings.common.name)}><Input data-testid="catalog-title" required value={title} onChange={(event) => setTitle(event.target.value)} /></Field>
        <Field label={t(strings.common.description)}><Input data-testid="catalog-description" required value={description} onChange={(event) => setDescription(event.target.value)} /></Field>
        <Field label={t(strings.catalog.cost)}><Input data-testid="catalog-cost" min="1" required type="number" value={cost} onChange={(event) => setCost(event.target.value)} /></Field>
        <Field label={t(strings.catalog.approval)}><Select data-testid="catalog-approval" value={posture} onChange={(event) => setPosture(event.target.value)} options={[{ value: String(ApprovalPosture.IMMEDIATE), label: t(strings.catalog.immediate) }, { value: String(ApprovalPosture.REQUIRES_APPROVAL), label: t(strings.catalog.needsApproval) }]} /></Field>
      </ConsoleForm>
      <RequestState loading={entries.isLoading} error={entries.error ?? createEntry.error ?? retireEntry.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={entries.data?.entries ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.catalog.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.catalog.empty)} tableTestId="catalog-table" />
    </ConsolePage>
  );
}
