import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, Clock3, X } from "lucide-react";
import { useState } from "react";
import type { Redemption } from "@vrooli/proto-types/token-economy/v1/access/access_pb";

import { minterClient, nextIdempotencyKey } from "../../api/tokenEconomy";
import { Button } from "../../components/ui/button";
import { DataTable, type DataTableColumn } from "../../components/ui/data-table";
import { Input } from "../../components/ui/input";
import { StatusBadge } from "../../components/ui/status-badge";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { ConsolePage, RequestState } from "../console/ConsolePage";

const queryKey = ["token-economy", "minter", "pending-redemptions"] as const;

export function ApprovalsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [reasons, setReasons] = useState<Record<string, string>>({});
  const pending = useQuery({ queryKey, queryFn: () => minterClient.listPendingRedemptions({}) });
  const refresh = () => queryClient.invalidateQueries({ queryKey });
  const approve = useMutation({ mutationFn: (redemption: Redemption) => minterClient.approveRedemption({ redemptionId: redemption.id, reason: reasons[redemption.id] || t(strings.approvals.defaultApprovalReason), idempotencyKey: nextIdempotencyKey("approve") }), onSuccess: refresh });
  const deny = useMutation({ mutationFn: (redemption: Redemption) => minterClient.denyRedemption({ redemptionId: redemption.id, reason: reasons[redemption.id] || t(strings.approvals.defaultDenialReason), idempotencyKey: nextIdempotencyKey("deny") }), onSuccess: refresh });
  const columns: Array<DataTableColumn<Redemption>> = [
    { id: "holder", header: t(strings.approvals.holder), accessor: (row) => <code>{row.holderId}</code>, searchValue: (row) => row.holderId },
    { id: "entry", header: t(strings.approvals.reward), accessor: (row) => <code>{row.catalogEntryId}</code>, searchValue: (row) => row.catalogEntryId },
    { id: "amount", header: t(strings.common.amount), accessor: (row) => <span className="font-mono">{row.amount.toString()}</span>, sortValue: (row) => Number(row.amount) },
    { id: "status", header: t(strings.common.status), accessor: () => <StatusBadge tone="warning"><Clock3 aria-hidden className="mr-1 h-3 w-3" />{t(strings.common.pending)}</StatusBadge> },
    { id: "decision", header: t(strings.approvals.decision), accessor: (row) => <div className="grid min-w-56 gap-2"><Input data-testid={`approvals-reason-${row.id}`} aria-label={t(strings.approvals.reason)} placeholder={t(strings.approvals.reason)} value={reasons[row.id] ?? ""} onChange={(event) => setReasons((current) => ({ ...current, [row.id]: event.target.value }))} /><div className="flex gap-2"><Button data-testid={`approvals-approve-${row.id}`} size="sm" onClick={() => approve.mutate(row)}><Check aria-hidden className="h-4 w-4" />{t(strings.approvals.approve)}</Button><Button data-testid={`approvals-deny-${row.id}`} size="sm" variant="danger" onClick={() => deny.mutate(row)}><X aria-hidden className="h-4 w-4" />{t(strings.approvals.deny)}</Button></div></div> },
  ];
  return (
    <ConsolePage testId="page-approvals" title={t(strings.approvals.title)} description={t(strings.approvals.description)}>
      <RequestState loading={pending.isLoading} error={pending.error ?? approve.error ?? deny.error} loadingLabel={t(strings.common.loading)} errorLabel={t(strings.common.requestError)} />
      <DataTable rows={pending.data?.redemptions ?? []} columns={columns} getRowKey={(row) => row.id} caption={t(strings.approvals.tableCaption)} searchLabel={t(strings.common.search)} searchPlaceholder={t(strings.common.search)} emptyMessage={t(strings.approvals.empty)} tableTestId="approvals-table" />
    </ConsolePage>
  );
}
