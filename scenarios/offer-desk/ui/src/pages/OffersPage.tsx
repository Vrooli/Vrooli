import { useMemo, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { NodeKind, Status } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";

import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { ExperienceSurface } from "../components/experience/ExperienceSurface";
import { FormSection } from "@vrooli/react-component-library/FormSection/1";
import { DirtyStateGuard } from "@vrooli/react-component-library/DirtyStateGuard/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "../components/ui/input";
import { Select } from "@vrooli/react-component-library/Select/1";
import { DataTable } from "@vrooli/react-component-library/DataTable/1";
import { useSurfaceState } from "../hooks/useSurfaceState";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { createEdge, createNode, fetchBoard, fetchCatalogVerification, fetchEdges, fetchNodes, mergeNodes, promoteNode, transition } from "../api/offers";

const nodeKinds = [NodeKind.OFFER, NodeKind.VARIANT, NodeKind.CHANNEL, NodeKind.REVENUE_LINE, NodeKind.DELIVERABLE];
const statuses = [Status.IDEA, Status.CANDIDATE, Status.TRIGGER_MET, Status.PROPOSED, Status.ACTIVE, Status.SHIPPED, Status.RETIRED];
const edgeKinds = ["belongs_to", "sells_at", "requires", "feeds"];
const edgeLabel = (kind: string, t: ReturnType<typeof useTranslation>["t"]) => ({
  belongs_to: t(strings.pages.offers.edgeBelongsTo),
  sells_at: t(strings.pages.offers.edgeSellsAt),
  requires: t(strings.pages.offers.edgeRequires),
  feeds: t(strings.pages.offers.edgeFeeds),
}[kind] ?? kind);

const enumLabel = (value: number | string | undefined, values: Record<number, string>, fallback: string) => {
  if (typeof value === "number") return values[value] ?? fallback;
  return value || fallback;
};
const kindName = (value: number | string | undefined) => enumLabel(value, NodeKind, "NODE_KIND_UNSPECIFIED");
const statusName = (value: number | string | undefined) => enumLabel(value, Status, "STATUS_UNSPECIFIED");
const localizedRankReason = (reason: string | undefined, t: ReturnType<typeof useTranslation>["t"]) => {
  if (!reason) return t(strings.pages.dashboard.rankReasonMissing);
  if (reason === "status not set") return t(strings.pages.dashboard.rankReasonStatusNotSet);
  if (reason === "captured, not planned against") return t(strings.pages.dashboard.rankReasonIdea);
  if (reason === "blocked: trigger not met") return t(strings.pages.dashboard.rankReasonCandidate);
  if (reason === "trigger fired") return t(strings.pages.dashboard.rankReasonTriggerMet);
  if (reason === "awaiting operator decision") return t(strings.pages.dashboard.rankReasonProposed);
  if (reason === "active and earning nothing") return t(strings.pages.dashboard.rankReasonActiveEarningNothing);
  if (reason === "active and earning") return t(strings.pages.dashboard.rankReasonActiveEarning);
  if (reason === "shipped and earning nothing") return t(strings.pages.dashboard.rankReasonShippedEarningNothing);
  if (reason === "shipped and earning") return t(strings.pages.dashboard.rankReasonShippedEarning);
  if (reason === "retired") return t(strings.pages.dashboard.rankReasonRetired);
  const unknown = /^(active|shipped); earnings unknown — (.+) unavailable$/.exec(reason);
  if (unknown) return t(unknown[1] === "active" ? strings.pages.dashboard.rankReasonActiveUnknown : strings.pages.dashboard.rankReasonShippedUnknown, { source: unknown[2] });
  const status = /^unknown status: (.+)$/.exec(reason);
  return status ? t(strings.pages.dashboard.rankReasonUnknown, { status: status[1] }) : reason;
};
const legalNextStatuses = (value: number | string | undefined) => {
  switch (statusName(value)) {
    case "IDEA": return [Status.CANDIDATE, Status.RETIRED];
    case "CANDIDATE": return [Status.TRIGGER_MET, Status.RETIRED];
    case "TRIGGER_MET": return [Status.ACTIVE, Status.RETIRED];
    case "PROPOSED": return [Status.ACTIVE, Status.RETIRED];
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
  const board = useQuery({ queryKey: ["offer-board"], queryFn: fetchBoard, retry: false });
  const verification = useQuery({ queryKey: ["offer-catalog-verification", "docs/monetization"], queryFn: () => fetchCatalogVerification(), retry: false });
  const nodeRows = useMemo(() => (nodes.data?.nodes ?? []) as NodeLike[], [nodes.data?.nodes]);
  const edgeRows = useMemo(() => (edges.data?.edges ?? []) as EdgeLike[], [edges.data?.edges]);
  const refused = nodes.isError || edges.isError;
  const rankReasons = useMemo(() => new Map((board.data?.entries ?? []).map((entry) => [entry.nodeId, entry.rankReason])), [board.data?.entries]);
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
  const [mergeForm, setMergeForm] = useState({ survivingId: "", duplicateId: "" });
  const [mergePreview, setMergePreview] = useState<{ movedEdges: number; movedTriggers: number; movedEvaluations: number; movedProposals: number; movedFindings: number; collapsedEdgeIds: string[] }>();
  const [mergeResult, setMergeResult] = useState("");
  const [groupedView, setGroupedView] = useState(true);

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
  const mergePreviewMutation = useMutation({
    mutationFn: () => mergeNodes({ survivingId: mergeForm.survivingId, duplicateId: mergeForm.duplicateId, actor: "operator", dryRun: true }),
    onSuccess: (response) => { setMergePreview(response); setMergeResult(""); setChangeError(false); },
    onError: () => { setMergePreview(undefined); setChangeError(true); },
  });
  const mergeApplyMutation = useMutation({
    mutationFn: () => mergeNodes({ survivingId: mergeForm.survivingId, duplicateId: mergeForm.duplicateId, actor: "operator", dryRun: false }),
    onSuccess: async () => { await refreshCatalog(); setMergePreview(undefined); setMergeResult(t(strings.pages.offers.mergeResult)); setMergeForm({ survivingId: "", duplicateId: "" }); },
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
  const groupedNodes = useMemo(() => {
    const groups = new Map<string, NodeLike[]>();
    for (const node of nodeRows) {
      const group = kindName(node.kind);
      groups.set(group, [...(groups.get(group) ?? []), node]);
    }
    return [...groups.entries()].sort(([left], [right]) => left.localeCompare(right));
  }, [nodeRows]);
  const edgeCounts = useMemo(() => {
    const counts = new Map<string, Record<string, number>>();
    for (const edge of edgeRows) {
      for (const nodeId of [edge.fromId, edge.toId]) {
        if (!nodeId) continue;
        const row = counts.get(nodeId) ?? {};
        row[edge.kind] = (row[edge.kind] ?? 0) + 1;
        counts.set(nodeId, row);
      }
    }
    return counts;
  }, [edgeRows]);
  const danglingMemberships = edgeRows.filter((edge) => edge.kind.toLowerCase().includes("belongs") && (!nodeIds.has(edge.fromId ?? "") || !nodeIds.has(edge.toId ?? "")));
  const selectedNode = nodeRows[0];
  const nodeColumns = [
    { id: "name", header: t(strings.pages.offers.offerLabel), accessor: (node: NodeLike) => node.name, searchValue: (node: NodeLike) => node.name, className: "break-words" },
    { id: "kind", header: t(strings.pages.offers.nodeKindLabel), accessor: (node: NodeLike) => kindName(node.kind), searchValue: (node: NodeLike) => kindName(node.kind), className: "break-words" },
    { id: "status", header: t(strings.pages.offers.statusLabel), accessor: (node: NodeLike) => statusName(node.status), searchValue: (node: NodeLike) => statusName(node.status), className: "break-words" },
    { id: "rank-reason", header: t(strings.pages.dashboard.rankReason), accessor: (node: NodeLike) => localizedRankReason(rankReasons.get(node.id), t), searchValue: (node: NodeLike) => rankReasons.get(node.id) || "", className: "break-words" },
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
  const submitMergePreview = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setMessage("");
    const survivor = nodeRows.find((node) => node.id === mergeForm.survivingId);
    const duplicate = nodeRows.find((node) => node.id === mergeForm.duplicateId);
    if (!survivor || !duplicate || survivor.id === duplicate.id || kindName(survivor.kind) !== kindName(duplicate.kind)) { setChangeError(true); return; }
    setChangeError(false);
    mergePreviewMutation.mutate();
  };

  return (
    <ExperienceSurface surfaceId="offers" state={surface.state} statusMessage={surface.reason} data-testid={selectors.pages.offers} aria-labelledby="offers-heading" className="flex flex-col gap-4">
      <h2 id="offers-heading" className="text-2xl font-semibold">{t(strings.pages.offers.title)}</h2>
      <Card className="min-w-0 max-w-full overflow-hidden">
        <CardHeader><CardTitle>{t(strings.pages.offers.cardTitle)}</CardTitle></CardHeader>
        <CardContent>
          <img
            data-testid={selectors.pages.offerGraph}
            src="data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='320' height='80' viewBox='0 0 320 80'%3E%3Cpath d='M40 40h100m40 0h100' stroke='%236b7280' stroke-width='2'/%3E%3Ccircle cx='40' cy='40' r='18' fill='%233b82f6'/%3E%3Ccircle cx='160' cy='40' r='18' fill='%2322c55e'/%3E%3Ccircle cx='280' cy='40' r='18' fill='%23f59e0b'/%3E%3C/svg%3E"
            alt={t(strings.pages.offers.description)}
            className="h-20 w-full rounded-md border object-contain p-2"
          />
          <p className="text-app-muted-foreground">{t(strings.pages.offers.description)}</p>
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
            <div><h3 className="font-semibold">{t(strings.pages.offers.groupedViewTitle)}</h3><p className="text-sm text-app-muted-foreground">{t(strings.pages.offers.groupedViewDescription)} {t(strings.pages.offers.groupedKindCount, { count: groupedNodes.length })} · {t(strings.pages.offers.groupedRelationshipCount, { count: edgeRows.length })}</p></div>
            <Button type="button" variant="secondary" data-testid={selectors.pages.catalogViewToggle} aria-pressed={groupedView} onClick={() => setGroupedView((current) => !current)}>{groupedView ? t(strings.pages.offers.catalogViewFlat) : t(strings.pages.offers.catalogViewGrouped)}</Button>
          </div>
          {verification.data?.duplicateIdentities.length ? <p data-testid={selectors.pages.catalogDuplicateBanner} role="alert" className="mt-3 rounded-md border border-amber-500 bg-amber-50 p-3 text-sm text-amber-900">{t(strings.pages.offers.duplicateIdentityBanner, { count: verification.data.duplicateIdentities.length })}</p> : null}
          {verification.isError && <p role="note" className="mt-3 rounded-md border border-dashed p-3 text-sm text-app-muted-foreground">{t(strings.pages.offers.catalogVerificationUnavailable)}</p>}
          {groupedView ? <div data-testid={selectors.pages.catalogGroupedView} role="region" aria-label={t(strings.pages.offers.groupedViewTitle)} className="mt-3 grid w-full min-w-0 max-w-full gap-4 overflow-x-auto">
            {!groupedNodes.length && <p role="note" className="rounded-md border border-dashed p-3 text-app-muted-foreground">{t(strings.pages.offers.groupedEmptyGuidance)}</p>}
            {groupedNodes.map(([kind, group], groupIndex) => <section data-testid={selectors.pages.catalogKindGroup} key={kind} role="group" aria-label={kind} className="w-full min-w-0 max-w-full overflow-hidden rounded-md border p-3">
              <div className="mb-3 flex flex-wrap items-baseline justify-between gap-2"><h4 className="font-semibold">{kind}</h4><span className="text-sm text-app-muted-foreground">{t(strings.pages.offers.groupedNodeCount, { count: group.length })}</span></div>
              <div className="min-w-0 max-w-full overflow-x-auto">
                <table data-testid={groupIndex === 0 ? selectors.pages.offerTable : selectors.pages.catalogNodeEdgeCounts} className="w-full table-fixed border-collapse text-sm"><caption className="sr-only">{kind}</caption><thead><tr className="border-b text-left"><th className="break-words p-2">{t(strings.pages.offers.offerLabel)}</th>{edgeKinds.map((edgeKind) => <th className="break-words p-2" key={edgeKind}>{edgeLabel(edgeKind, t)}</th>)}<th className="w-12 break-words p-2">{t(strings.pages.offers.transitionAction)}</th></tr></thead><tbody>{group.map((node) => { const counts = edgeCounts.get(node.id) ?? {}; return <tr key={node.id} className="border-b"><th scope="row" className="break-words p-2 text-left font-normal">{node.name}</th>{edgeKinds.map((edgeKind) => <td className="break-words p-2 tabular-nums" key={edgeKind}>{counts[edgeKind] ?? 0}</td>)}<td className="w-12 break-words p-2"><Button type="button" data-testid={selectors.pages.offerPromote} size="sm" className="min-h-11 min-w-11 whitespace-normal break-words" disabled={promotionPending} onClick={() => void handlePromotion(node.id)}>{t(strings.pages.offers.promoteAction)}</Button></td></tr>; })}</tbody></table>
              </div>
              {group.every((node) => !edgeRows.some((edge) => edge.fromId === node.id || edge.toId === node.id)) && <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.offers.groupedNoEdges)}</p>}
            </section>)}
          </div> : <DataTable rows={nodeRows} columns={nodeColumns} getRowKey={(node) => node.id} caption={t(strings.pages.offers.statusLabel)} searchLabel={t(strings.pages.offers.offerLabel)} searchPlaceholder={t(strings.pages.offers.offerLabel)} emptyMessage={t(strings.pages.offers.emptyGuidance)} tableTestId={selectors.pages.offerTable} className="mt-4" />}
          <p data-testid={selectors.pages.offerStatus} role="status" aria-label={t(strings.pages.offers.statusLabel)} className="text-sm">{selectedNode ? `${selectedNode.name}: ${statusName(selectedNode.status)}` : t(strings.pages.offers.emptyGuidance)}</p>
          <p data-testid={selectors.pages.offerRankReason} role="status" aria-label={t(strings.pages.dashboard.rankReason)} className="break-words text-sm">{selectedNode ? localizedRankReason(rankReasons.get(selectedNode.id), t) : t(strings.pages.dashboard.rankReasonMissing)}</p>
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
      <Card>
        <CardHeader><CardTitle>{t(strings.pages.offers.mergeTitle)}</CardTitle></CardHeader>
        <CardContent>
          <form data-testid={selectors.pages.offerMergeForm} aria-label={t(strings.pages.offers.mergeTitle)} className="grid gap-3" onSubmit={submitMergePreview}>
            <label className="grid gap-1" htmlFor="offer-merge-survivor"><span>{t(strings.pages.offers.mergeSurvivorLabel)}</span><Select id="offer-merge-survivor" value={mergeForm.survivingId} onChange={(event) => { setMergeForm({ ...mergeForm, survivingId: event.target.value }); setMergePreview(undefined); }} options={nodeRows.map((node) => ({ value: node.id, label: `${node.name} · ${kindName(node.kind)}` }))} placeholder={t(strings.pages.offers.mergeSurvivorLabel)} /></label>
            <label className="grid gap-1" htmlFor="offer-merge-duplicate"><span>{t(strings.pages.offers.mergeDuplicateLabel)}</span><Select id="offer-merge-duplicate" value={mergeForm.duplicateId} onChange={(event) => { setMergeForm({ ...mergeForm, duplicateId: event.target.value }); setMergePreview(undefined); }} options={nodeRows.map((node) => ({ value: node.id, label: `${node.name} · ${kindName(node.kind)}` }))} placeholder={t(strings.pages.offers.mergeDuplicateLabel)} /></label>
            <Button type="submit" disabled={mergePreviewMutation.isPending || !mergeForm.survivingId || !mergeForm.duplicateId}>{t(strings.pages.offers.mergePreviewAction)}</Button>
          </form>
          <div data-testid={selectors.pages.offerMergeSummary} role="status" className="mt-3 grid gap-2 rounded-md border p-3 text-sm">
            {mergePreview ? <><p>{t(strings.pages.offers.mergeDryRunNotice)}</p><p>{t(strings.pages.offers.mergePreview, { duplicate: nodeRows.find((node) => node.id === mergeForm.duplicateId)?.name ?? mergeForm.duplicateId, survivor: nodeRows.find((node) => node.id === mergeForm.survivingId)?.name ?? mergeForm.survivingId, edges: mergePreview.movedEdges, triggers: mergePreview.movedTriggers, evaluations: mergePreview.movedEvaluations, proposals: mergePreview.movedProposals, findings: mergePreview.movedFindings, collapsed: mergePreview.collapsedEdgeIds.length })}</p></> : <p>{t(strings.pages.offers.mergePreviewAction)}</p>}
            <div className="flex flex-wrap gap-2"><Button type="button" disabled={mergeApplyMutation.isPending || !mergePreview} onClick={() => mergeApplyMutation.mutate()}>{t(strings.pages.offers.mergeApplyAction)}</Button><Button type="button" variant="secondary" disabled={!mergePreview} onClick={() => setMergePreview(undefined)}>{t(strings.pages.offers.mergeCancelAction)}</Button></div>
          </div>
          {mergeResult && <p role="status" className="mt-2 text-sm text-app-success">{mergeResult}</p>}
        </CardContent>
      </Card>
      {changeError && <p role="alert" className="text-sm text-app-danger">{t(strings.pages.offers.requestError)}</p>}
      {message && <p data-testid={selectors.pages.offerChangeNotice} role="status" className="text-sm text-app-success">{message}</p>}
    </ExperienceSurface>
  );
}
