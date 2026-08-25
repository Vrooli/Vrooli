import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { fetchNodes, fetchProposals, promoteNode, transition } from "../api/offers";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { Button } from "@vrooli/react-component-library/Button/1.2.0";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Textarea } from "@vrooli/react-component-library/Textarea/1.0.0";
import { DataTable } from "@vrooli/react-component-library/DataTable/1.2.0";
import { Status } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";

type TimestampLike = { seconds: bigint | number; nanos?: number };

const statusLabel = (status: unknown) => {
  if (typeof status === "number") {
    return ["STATUS_UNSPECIFIED", "IDEA", "CANDIDATE", "TRIGGER_MET", "ACTIVE", "SHIPPED", "RETIRED"][status] ?? "UNKNOWN";
  }
  return typeof status === "string" ? status.replace(/^STATUS_/, "") : "UNKNOWN";
};

const timestampLabel = (timestamp?: TimestampLike) => {
  if (!timestamp) return "—";
  return new Date(Number(timestamp.seconds) * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000)).toISOString();
};

function callerRole() {
  if (typeof window === "undefined") return "agent";
  return window.sessionStorage.getItem("vrooli.role") ?? "agent";
}

export function ProposalsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const role = callerRole();
  const operator = role === "operator";
  const proposals = useQuery({ queryKey: ["proposals"], queryFn: fetchProposals, retry: false });
  const nodes = useQuery({ queryKey: ["proposal-nodes"], queryFn: fetchNodes, retry: false });
  const [declineReasons, setDeclineReasons] = useState<Record<string, string>>({});
  const [message, setMessage] = useState("");
  const acceptMutation = useMutation({
    mutationFn: (nodeId: string) => promoteNode({ nodeId, actor: "operator", role: "operator" }),
    onSuccess: async () => {
      setMessage(t(strings.pages.proposals.acceptedNotice));
      await queryClient.invalidateQueries({ queryKey: ["proposals"] });
      await queryClient.invalidateQueries({ queryKey: ["proposal-nodes"] });
    },
  });
  const declineMutation = useMutation({
    mutationFn: ({ nodeId, reason }: { nodeId: string; reason: string }) => transition({ nodeId, status: Status.RETIRED, actor: `operator:decline:${reason}` }),
    onSuccess: async () => {
      setMessage(t(strings.pages.proposals.declineNotice));
      await queryClient.invalidateQueries({ queryKey: ["proposals"] });
      await queryClient.invalidateQueries({ queryKey: ["proposal-nodes"] });
    },
  });
  const nodesById = useMemo(() => new Map((nodes.data?.nodes ?? []).map((node) => [node.id, node])), [nodes.data?.nodes]);
  const surface = useSurfaceState({
    query: { isLoading: proposals.isLoading || nodes.isLoading, isFetching: proposals.isFetching || nodes.isFetching, isError: proposals.isError || nodes.isError, error: proposals.error ?? nodes.error },
    empty: Boolean(proposals.data && proposals.data.proposals.length === 0),
    mutation: { isPending: acceptMutation.isPending || declineMutation.isPending, isError: acceptMutation.isError || declineMutation.isError, error: acceptMutation.error ?? declineMutation.error },
  });
  const proposalRows = proposals.data?.proposals ?? [];
  const proposalColumns = [
    { id: "proposer", header: t(strings.pages.proposals.proposer), headerTestId: selectors.pages.proposalProposer, accessor: (proposal: typeof proposalRows[number]) => proposal.actor, searchValue: (proposal: typeof proposalRows[number]) => proposal.actor, className: "break-all" },
    { id: "requested-status", header: t(strings.pages.proposals.requestedStatus), headerTestId: selectors.pages.proposalRequestedStatus, accessor: (proposal: typeof proposalRows[number]) => statusLabel(proposal.requestedStatus), searchValue: (proposal: typeof proposalRows[number]) => statusLabel(proposal.requestedStatus), className: "break-all" },
    { id: "reason", header: t(strings.pages.proposals.reason), headerTestId: selectors.pages.proposalReason, accessor: (proposal: typeof proposalRows[number]) => proposal.reason || "—", searchValue: (proposal: typeof proposalRows[number]) => proposal.reason || "", className: "break-all" },
    { id: "evidence", header: t(strings.pages.proposals.evidence), headerTestId: selectors.pages.proposalEvidence, accessor: (proposal: typeof proposalRows[number]) => proposal.evidenceReference || "—", searchValue: (proposal: typeof proposalRows[number]) => proposal.evidenceReference || "", className: "break-all" },
    { id: "age", header: t(strings.pages.proposals.age), headerTestId: selectors.pages.proposalAge, accessor: (proposal: typeof proposalRows[number]) => timestampLabel(proposal.createdAt), searchValue: (proposal: typeof proposalRows[number]) => timestampLabel(proposal.createdAt), className: "break-all" },
    { id: "declines", header: t(strings.pages.proposals.declineHistory), headerTestId: selectors.pages.proposalDeclineHistory, accessor: (proposal: typeof proposalRows[number]) => proposal.declineHistory.length > 0 ? <ul className="space-y-1">{proposal.declineHistory.map((decline, index) => <li key={`${proposal.id}-decline-${index}`}>{decline.actor}: {decline.reason} ({timestampLabel(decline.createdAt)})</li>)}</ul> : <span>{t(strings.pages.proposals.noDeclines)}</span>, className: "break-all" },
    { id: "actions", header: t(strings.pages.proposals.actions), accessor: (proposal: typeof proposalRows[number]) => {
      const node = nodesById.get(proposal.nodeId);
      const reason = declineReasons[proposal.id] ?? "";
      return <div className="space-y-2">
        <p className="text-xs text-app-muted-foreground">{node?.name ?? proposal.nodeId}: {statusLabel(node?.status)} → {statusLabel(proposal.requestedStatus)}</p>
        <Button type="button" data-testid={selectors.pages.proposalAccept} size="sm" className="min-w-11 max-w-full overflow-hidden text-ellipsis whitespace-nowrap" aria-label={t(strings.pages.proposals.acceptAction)} disabled={!operator || acceptMutation.isPending || declineMutation.isPending} onClick={() => acceptMutation.mutate(proposal.nodeId)}>{t(strings.pages.proposals.acceptAction)}</Button>
        <label className="grid gap-1 text-xs" htmlFor={`proposal-decline-reason-${proposal.id}`}>
          <span>{t(strings.pages.proposals.declineReason)}</span>
          <Textarea id={`proposal-decline-reason-${proposal.id}`} data-testid={selectors.pages.proposalDeclineReason} className="min-w-11" value={reason} placeholder={t(strings.pages.proposals.declinePlaceholder)} onChange={(event) => setDeclineReasons({ ...declineReasons, [proposal.id]: event.target.value })} disabled={!operator || declineMutation.isPending} />
        </label>
        <Button type="button" data-testid={selectors.pages.proposalDecline} size="sm" variant="danger" className="min-w-11 max-w-full overflow-hidden text-ellipsis whitespace-nowrap" aria-label={t(strings.pages.proposals.declineAction)} disabled={!operator || !reason.trim() || acceptMutation.isPending || declineMutation.isPending} onClick={() => declineMutation.mutate({ nodeId: proposal.nodeId, reason: reason.trim() })}>{t(strings.pages.proposals.declineAction)}</Button>
      </div>;
    }, className: "break-all" },
  ];

  return (
    <ExperienceSurface surfaceId="proposals" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.proposals} aria-labelledby="proposals-heading" className="flex flex-col gap-4">
      <h2 id="proposals-heading" className="text-2xl font-semibold">{t(strings.pages.proposals.title)}</h2>
      <Card data-testid={selectors.pages.proposalList}>
        <CardHeader><CardTitle>{t(strings.pages.proposals.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <p className="text-app-muted-foreground">{t(strings.pages.proposals.description)}</p>
          <p data-testid={selectors.pages.proposalOperatorOnly} role="note" className="mt-2 text-sm text-app-muted-foreground">{operator ? t(strings.pages.proposals.operatorOnly) : t(strings.pages.proposals.roleUnavailable)}</p>
          <p data-testid={selectors.pages.proposalEffect} role="note" className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.proposals.effect, { current: "current status", target: "ACTIVE" })}</p>
          {proposals.data && proposals.data.proposals.length > 0 && <DataTable rows={proposalRows} columns={proposalColumns} getRowKey={(proposal) => proposal.id} getRowTestId={(proposal) => `proposal-row-${proposal.id}`} caption={t(strings.pages.proposals.tableCaption)} searchLabel={t(strings.pages.proposals.proposer)} searchPlaceholder={t(strings.pages.proposals.proposer)} emptyMessage={t(strings.pages.proposals.emptyGuidance)} tableTestId={selectors.pages.proposalTable} className="mt-4" />}
          {surface.state === "empty" && <p data-testid={selectors.pages.proposalsEmptyGuidance} className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.proposals.emptyGuidance)}</p>}
          {message && <p data-testid={selectors.pages.proposalChangeNotice} role="status" className="mt-3 text-sm text-app-success">{message}</p>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
