import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { Holder } from "@vrooli/proto-types/token-economy/v1/access/access_pb";
import { useState } from "react";

import { minterClient, nextIdempotencyKey } from "../../api/tokenEconomy";
import { DataTable, type DataTableColumn } from "../../components/ui/data-table";
import { Input } from "../../components/ui/input";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsoleForm, ConsolePage, Field, RequestState } from "../console/ConsolePage";

const queryKey = ["token-economy", "minter", "holders"] as const;

export function HoldersPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [displayName, setDisplayName] = useState("");
  const [subject, setSubject] = useState("");
  const holders = useQuery({ queryKey, queryFn: () => minterClient.listHolders({}) });
  const createHolder = useMutation({
    mutationFn: () => minterClient.createHolder({ displayName, authenticatorSubject: subject, idempotencyKey: nextIdempotencyKey("holder") }),
    onSuccess: async () => { setDisplayName(""); setSubject(""); await queryClient.invalidateQueries({ queryKey }); },
  });
  const columns: Array<DataTableColumn<Holder>> = [
    { id: "name", header: t(strings.common.name), accessor: (row) => row.displayName, sortValue: (row) => row.displayName },
    { id: "subject", header: t(strings.holders.authenticatorSubject), accessor: (row) => <code>{row.authenticatorSubject}</code>, searchValue: (row) => row.authenticatorSubject },
    { id: "id", header: t(strings.common.identifier), accessor: (row) => <code>{row.id}</code>, searchValue: (row) => row.id },
  ];
  return (
    <ConsolePage testId="page-holders" title={t(strings.holders.title)} description={t(strings.holders.description)}>
      <ConsoleForm title={t(strings.holders.addTitle)} submitLabel={t(strings.holders.add)} submitTestId="holders-add" busy={createHolder.isPending} onSubmit={(event) => { event.preventDefault(); createHolder.mutate(); }}>
        <Field label={t(strings.common.name)}><Input data-testid="holders-display-name" required value={displayName} onChange={(event) => setDisplayName(event.target.value)} /></Field>
        <Field label={t(strings.holders.authenticatorSubject)}><Input data-testid="holders-identity-binding" required value={subject} onChange={(event) => setSubject(event.target.value)} /></Field>
      </ConsoleForm>
      <RequestState loading={holders.isLoading} error={holders.error ?? createHolder.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={holders.data?.holders ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.holders.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.holders.empty)} tableTestId="holders-table" />
    </ConsolePage>
  );
}
