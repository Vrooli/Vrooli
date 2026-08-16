import { useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { NodeKind, Status } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { FormSection } from "../components/FormSection";
import { DirtyStateGuard } from "../components/DirtyStateGuard";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Select } from "../components/ui/select";
import { DataTable } from "../components/ui/data-table";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { createEdge, createNode, fetchEdges, fetchNodes, promoteNode, transition } from "../api/offers";

const nodeKinds = [NodeKind.OFFER, NodeKind.VARIANT, NodeKind.CHANNEL, NodeKind.REVENUE_LINE, NodeKind.DELIVERABLE];
const statuses = [Status.IDEA, Status.CANDIDATE, Status.TRIGGER_MET, Status.ACTIVE, Status.SHIPPED, Status.RETIRED];
const edgeKinds = ["belongs_to", "sells_at", "requires", "feeds"];

const enumLabel = (value: number | string | undefined, values: Record<number, string>, fallback: string) => {
  if (typeof value === "number") return values[value] ?? fallback;
  return value || fallback;
};
const kindName = (value: number | string | undefined) => enumLabel(value, NodeKind, "NODE_KIND_UNSPECIFIED");
const statusName = (value: number | string | undefined) => enumLabel(value, Status, "STATUS_UNSPECIFIED");
const legalNextStatuses = (value: number | string | undefined) => {
  switch (statusName(value)) {
    case "IDEA": return [Status.CANDIDATE, Status.RETIRED];
    case "CANDIDATE": return [Status.TRIGGER_MET, Status.RETIRED];
    case "TRIGGER_MET": return [Status.ACTIVE, Status.RETIRED];
    case "ACTIVE": return [Status.SHIPPED, Status.RETIRED];
    case "SHIPPED": return [Status.RETIRED];
    default: return [];
  }
};

type NodeLike = { id: string; kind?: NodeKind | string; name: string; status: Status | string; triggerId?: string; actualAccountId?: string };
type EdgeLike = { id: string; fromId?: string; toId?: string; kind: string; intendedPriceMinor?: bigint; currency?: string };
type ProposalLike = { rationale?: string; reason?: string };

export function OffersPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const nodes = useQuery({ queryKey: ["offer-nodes"], queryFn: fetchNodes, retry: false });
  const edges = useQuery({ queryKey: ["offer-edges"], queryFn: fetchEdges, retry: false });
  const nodeRows = useMemo(() => (nodes.data?.nodes ?? []) as NodeLike[], [nodes.data?.nodes]);
  const edgeRows = (edges.data?.edges ?? []) as EdgeLike[];
  const refused = nodes.isError || edges.isError;
  const surface = useSurfaceState({
    query: { isLoading: nodes.isLoading || edges.isLoading, isFetching: nodes.isFetching || edges.isFetching, isError: refused, error: nodes.error || edges.error },
    empty: Boolean(nodes.data && nodes.data.nodes.length === 0),
  });

  const [nodeForm, setNodeForm] = useState({ kind: String(NodeKind.OFFER), name: "", status: String(Status.IDEA), triggerId: "", actualAccountId: "" });
  const [transitionForm, setTransitionForm] = useState({ nodeId: "", status: "" });
  const [edgeForm, setEdgeForm] = useState({ fromId: "", toId: "", kind: edgeKinds[0] ?? "belongs_to", intendedPriceMinor: "", currency: "" });
  const [message, setMessage] = useState("");
  const [changeError, setChangeError] = useState(false);
  const [promotionProposal, setPromotionProposal] = useState<ProposalLike>();
  const [promotionPending, setPromotionPending] = useState(false);

  const refreshCatalog = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["offer-nodes"] }),
      queryClient.invalidateQueries({ queryKey: ["offer-edges"] }),
      queryClient.invalidateQueries({ queryKey: ["offer-board"] }),
    ]);
    setChangeError(false);
    setMessage(t(strings.pages.offers.savedNotice));
  };
  const createNodeMutation = useMutation({
    mutationFn: () => createNode({ kind: Number(nodeForm.kind), name: nodeForm.name.trim(), status: Number(nodeForm.status), triggerId: nodeForm.triggerId.trim(), actualAccountId: nodeForm.actualAccountId.trim() }),
    onSuccess: async () => { await refreshCatalog(); setNodeForm({ kind: String(NodeKind.OFFER), name: "", status: String(Status.IDEA), triggerId: "", actualAccountId: "" }); },
    onError: () => setChangeError(true),
  });
  const transitionMutation = useMutation({
    mutationFn: () => transition({ nodeId: transitionForm.nodeId, status: Number(transitionForm.status), actor: "operator" }),
    onSuccess: async () => { await refreshCatalog(); setTransitionForm({ nodeId: transitionForm.nodeId, status: "" }); },
    onError: () => setChangeError(true),
  });
  const createEdgeMutation = useMutation({
    mutationFn: () => createEdge({ fromId: edgeForm.fromId.trim(), toId: edgeForm.toId.trim(), kind: edgeForm.kind, intendedPriceMinor: edgeForm.intendedPriceMinor ? BigInt(edgeForm.intendedPriceMinor) : 0n, currency: edgeForm.currency.trim() }),
    onSuccess: async () => { await refreshCatalog(); setEdgeForm({ fromId: "", toId: "", kind: edgeKinds[0] ?? "belongs_to", intendedPriceMinor: "", currency: "" }); },
    onError: () => setChangeError(true),
  });
  const handlePromotion = async (nodeId: string) => {
    setPromotionPending(true);
    setChangeError(false);
    try {
      const response = await promoteNode({ nodeId, actor: "agent", role: "agent" });
      setPromotionProposal(response.proposal);
    } catch {
      setChangeError(true);
    } finally {
      setPromotionPending(false);
    }
  };

  const selectedTransitionNode = nodeRows.find((node) => node.id === transitionForm.nodeId);
  const nextStatuses = legalNextStatuses(selectedTransitionNode?.status);
  const nodeIds = useMemo(() => new Set(nodeRows.map((node) => node.id)), [nodeRows]);
  const danglingMemberships = edgeRows.filter((edge) => edge.kind.toLowerCase().includes("belongs") && (!nodeIds.has(edge.fromId ?? "") || !nodeIds.has(edge.toId ?? "")));
  const selectedNode = nodeRows[0];
  const nodeColumns = [
    { id: "name", header: t(strings.pages.offers.offerLabel), accessor: (node: NodeLike) => node.name, searchValue: (node: NodeLike) => node.name, className: "break-words" },
    { id: "kind", header: t(strings.pages.offers.nodeKindLabel), accessor: (node: NodeLike) => kindName(node.kind), searchValue: (node: NodeLike) => kindName(node.kind), className: "break-words" },
    { id: "status", header: t(strings.pages.offers.statusLabel), accessor: (node: NodeLike) => statusName(node.status), searchValue: (node: NodeLike) => statusName(node.status), className: "break-words" },
    { id: "trigger", header: t(strings.pages.offers.nodeTriggerLabel), accessor: (node: NodeLike) => node.triggerId || "—", searchValue: (node: NodeLike) => node.triggerId || "", className: "break-words" },
    { id: "account", header: t(strings.pages.offers.actualAccountLabel), accessor: (node: NodeLike) => node.actualAccountId || "—", searchValue: (node: NodeLike) => node.actualAccountId || "", className: "break-words" },
    { id: "action", header: t(strings.pages.offers.transitionAction), accessor: (node: NodeLike) => <Button type="button" data-testid={selectors.pages.offerPromote} size="sm" className="w-full min-w-11 whitespace-normal break-words text-center" disabled={promotionPending} onClick={() => void handlePromotion(node.id)}>{t(strings.pages.offers.promoteAction)}</Button>, className: "min-w-11 break-words" },
  ];

  const submitNode = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    if (!nodeForm.name.trim()) { setChangeError(true); return; }
    setChangeError(false);
    createNodeMutation.mutate();
  };
  const submitTransition = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    if (!transitionForm.nodeId || !transitionForm.status) { setChangeError(true); return; }
    setChangeError(false);
    transitionMutation.mutate();
  };
  const submitEdge = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    if (!edgeForm.fromId.trim() || !edgeForm.toId.trim() || !edgeForm.kind) { setChangeError(true); return; }
    try { if (edgeForm.intendedPriceMinor) BigInt(edgeForm.intendedPriceMinor); } catch { setChangeError(true); return; }
    setChangeError(false);
    createEdgeMutation.mutate();
  };

  return (
    <ExperienceSurface surfaceId="offers" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.offers} aria-labelledby="offers-heading" className="flex flex-col gap-4">
      <h2 id="offers-heading" className="text-2xl font-semibold">{t(strings.pages.offers.title)}</h2>
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.offers.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <img
            data-testid={selectors.pages.offerGraph}
            src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='320' height='80' viewBox='0 0 320 80'%3E%3Cpath d='M40 40h100m40 0h100' stroke='%236b7280' stroke-width='2'/%3E%3Ccircle cx='40' cy='40' r='18' fill='%233b82f6'/%3E%3Ccircle cx='160' cy='40' r='18' fill='%2322c55e'/%3E%3Ccircle cx='280' cy='40' r='18' fill='%23f59e0b'/%3E%3C/svg%3E"
            alt={t(strings.pages.offers.description)}
            className="h-20 w-full rounded-md border object-contain p-2"
          />
          <p className="text-app-muted-foreground">{t(strings.pages.offers.description)}</p>
          <DataTable rows={nodeRows} columns={nodeColumns} getRowKey={(node) => node.id} caption={t(strings.pages.offers.statusLabel)} searchLabel={t(strings.pages.offers.offerLabel)} searchPlaceholder={t(strings.pages.offers.offerLabel)} emptyMessage={t(strings.pages.offers.emptyGuidance)} tableTestId={selectors.pages.offerTable} className="mt-4" />
          <p data-testid={selectors.pages.offerStatus} role="status" aria-label={t(strings.pages.offers.statusLabel)} className="text-sm">{selectedNode ? `${selectedNode.name}: ${statusName(selectedNode.status)}` : t(strings.pages.offers.emptyGuidance)}</p>
          <p data-testid={selectors.pages.offerWaitingOn} role="note" aria-label={t(strings.pages.offers.waitingOn)} className="text-sm text-app-muted-foreground">{selectedNode?.status === Status.CANDIDATE || statusName(selectedNode?.status) === "CANDIDATE" ? t(strings.pages.offers.candidateRequiresTrigger) : t(strings.pages.offers.waitingOn)}</p>
          <ul data-testid={selectors.pages.offerLegalTransitions} aria-label={t(strings.pages.offers.legalTransitions)} className="text-sm text-app-muted-foreground"><li>{selectedNode ? legalNextStatuses(selectedNode.status).map((status) => statusName(status)).join(", ") || statusName(selectedNode.status) : t(strings.pages.offers.legalTransitions)}</li></ul>
          <p data-testid={selectors.pages.offerRefusalReason} role="alert" className="text-sm text-app-danger">{refused || changeError ? t(strings.pages.offers.refusalReason) : ""}</p>
          <p data-testid={selectors.pages.offerRefusalRemedy} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.offers.refusalRemedy)}</p>
          <ul data-testid={selectors.pages.offerAuditTrail} className="text-sm text-app-muted-foreground"><li>{t(strings.pages.offers.auditTrail)}</li></ul>
          {promotionProposal && <p role="status" className="mt-2 text-sm">{promotionProposal.rationale ?? promotionProposal.reason}</p>}
          <p data-testid={selectors.pages.offerRoleRequirement} role="note" className="text-sm text-app-muted-foreground">{t(strings.pages.offers.roleRequirement)}</p>
          <p data-testid={selectors.pages.offerMembershipFinding} role="note" className="text-sm text-app-muted-foreground">{danglingMemberships.length ? `${t(strings.pages.offers.findingLabel)}: ${t(strings.pages.offers.membershipFinding)} (${danglingMemberships.length})` : ""}</p>
          {edgeRows.map((edge) => <p key={edge.id} className="text-sm text-app-muted-foreground">{edge.fromId || "?"} → {edge.toId || "?"} · {edge.kind}</p>)}
          {surface.state === "empty" && <p data-testid={selectors.pages.offersEmptyGuidance} role="note" className="mt-3 rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.offers.emptyGuidance)}</p>}
        </CardContent>
      </Card>

      <div className="grid gap-4 lg:grid-cols-3">
        <DirtyStateGuard isDirty={Boolean(nodeForm.name || nodeForm.triggerId || nodeForm.actualAccountId)} protectUnload title={t(strings.pages.offers.createNodeTitle)} description={t(strings.pages.offers.description)}>
          <FormSection title={t(strings.pages.offers.createNodeTitle)}>
            <form data-testid={selectors.pages.offerCreateNode} aria-label={t(strings.pages.offers.createNodeTitle)} className="grid gap-3" onSubmit={submitNode}>
              <label className="grid gap-1" htmlFor="offer-node-kind"><span>{t(strings.pages.offers.nodeKindLabel)}</span><Select id="offer-node-kind" value={nodeForm.kind} onChange={(event) => setNodeForm({ ...nodeForm, kind: event.target.value })} options={nodeKinds.map((kind) => ({ value: String(kind), label: kindName(kind) }))} /></label>
              <label className="grid gap-1" htmlFor="offer-node-name"><span>{t(strings.pages.offers.nodeNameLabel)}</span><Input id="offer-node-name" value={nodeForm.name} onChange={(event) => setNodeForm({ ...nodeForm, name: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="offer-node-status"><span>{t(strings.pages.offers.nodeStatusLabel)}</span><Select id="offer-node-status" value={nodeForm.status} onChange={(event) => setNodeForm({ ...nodeForm, status: event.target.value })} options={statuses.map((status) => ({ value: String(status), label: statusName(status) }))} /></label>
              <label className="grid gap-1" htmlFor="offer-node-trigger"><span>{t(strings.pages.offers.nodeTriggerLabel)}</span><Input id="offer-node-trigger" value={nodeForm.triggerId} onChange={(event) => setNodeForm({ ...nodeForm, triggerId: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="offer-node-account"><span>{t(strings.pages.offers.actualAccountLabel)}</span><Input id="offer-node-account" value={nodeForm.actualAccountId} onChange={(event) => setNodeForm({ ...nodeForm, actualAccountId: event.target.value })} /></label>
              <Button type="submit" disabled={createNodeMutation.isPending}>{t(strings.pages.offers.createNodeAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
        <DirtyStateGuard isDirty={Boolean(transitionForm.nodeId || transitionForm.status)} protectUnload title={t(strings.pages.offers.transitionAction)} description={t(strings.pages.offers.refusalRemedy)}>
          <FormSection title={t(strings.pages.offers.transitionAction)}>
            <form data-testid={selectors.pages.offerTransition} aria-label={t(strings.pages.offers.transitionAction)} className="grid gap-3" onSubmit={submitTransition}>
              <label className="grid gap-1" htmlFor="offer-transition-node"><span>{t(strings.pages.offers.offerLabel)}</span><Select id="offer-transition-node" value={transitionForm.nodeId} onChange={(event) => setTransitionForm({ nodeId: event.target.value, status: "" })} options={nodeRows.map((node) => ({ value: node.id, label: node.name }))} placeholder={t(strings.pages.offers.offerLabel)} /></label>
              <label className="grid gap-1" htmlFor="offer-transition-status"><span>{t(strings.pages.offers.transitionTargetLabel)}</span><Select id="offer-transition-status" value={transitionForm.status} onChange={(event) => setTransitionForm({ ...transitionForm, status: event.target.value })} options={nextStatuses.map((status) => ({ value: String(status), label: statusName(status) }))} placeholder={t(strings.pages.offers.transitionTargetLabel)} /></label>
              {selectedTransitionNode?.status === Status.CANDIDATE || statusName(selectedTransitionNode?.status) === "CANDIDATE" ? <p className="text-sm text-app-muted-foreground">{t(strings.pages.offers.candidateRequiresTrigger)}</p> : null}
              <Button type="submit" disabled={transitionMutation.isPending || !nextStatuses.length}>{t(strings.pages.offers.transitionAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
        <DirtyStateGuard isDirty={Boolean(edgeForm.fromId || edgeForm.toId || edgeForm.intendedPriceMinor || edgeForm.currency)} protectUnload title={t(strings.pages.offers.edgeTitle)} description={t(strings.pages.offers.description)}>
          <FormSection title={t(strings.pages.offers.edgeTitle)}>
            <form data-testid={selectors.pages.offerCreateEdge} aria-label={t(strings.pages.offers.edgeTitle)} className="grid gap-3" onSubmit={submitEdge}>
              <label className="grid gap-1" htmlFor="offer-edge-from"><span>{t(strings.pages.offers.edgeFromLabel)}</span><Select id="offer-edge-from" value={edgeForm.fromId} onChange={(event) => setEdgeForm({ ...edgeForm, fromId: event.target.value })} options={nodeRows.map((node) => ({ value: node.id, label: node.name }))} placeholder={t(strings.pages.offers.edgeFromLabel)} /></label>
              <label className="grid gap-1" htmlFor="offer-edge-to"><span>{t(strings.pages.offers.edgeToLabel)}</span><Select id="offer-edge-to" value={edgeForm.toId} onChange={(event) => setEdgeForm({ ...edgeForm, toId: event.target.value })} options={nodeRows.map((node) => ({ value: node.id, label: node.name }))} placeholder={t(strings.pages.offers.edgeToLabel)} /></label>
              <label className="grid gap-1" htmlFor="offer-edge-kind"><span>{t(strings.pages.offers.edgeKindLabel)}</span><Select id="offer-edge-kind" value={edgeForm.kind} onChange={(event) => setEdgeForm({ ...edgeForm, kind: event.target.value })} options={edgeKinds.map((kind) => ({ value: kind, label: kind }))} /></label>
              <label className="grid gap-1" htmlFor="offer-edge-price"><span>{t(strings.pages.offers.intendedPriceLabel)}</span><Input id="offer-edge-price" type="number" min="0" step="1" value={edgeForm.intendedPriceMinor} onChange={(event) => setEdgeForm({ ...edgeForm, intendedPriceMinor: event.target.value })} /></label>
              <label className="grid gap-1" htmlFor="offer-edge-currency"><span>{t(strings.pages.offers.currencyLabel)}</span><Input id="offer-edge-currency" value={edgeForm.currency} onChange={(event) => setEdgeForm({ ...edgeForm, currency: event.target.value })} /></label>
              <Button type="submit" disabled={createEdgeMutation.isPending}>{t(strings.pages.offers.createEdgeAction)}</Button>
            </form>
          </FormSection>
        </DirtyStateGuard>
      </div>
      {changeError && <p role="alert" className="text-sm text-app-danger">{t(strings.pages.offers.requestError)}</p>}
      {message && <p data-testid={selectors.pages.offerChangeNotice} role="status" className="text-sm text-app-success">{message}</p>}
    </ExperienceSurface>
  );
}
