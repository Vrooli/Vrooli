import { createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { AddFactRequestSchema, BoardService, CatalogService, CreateEdgeRequestSchema, CreateNodeRequestSchema, EdgeSchema, GatesService, DeclareTriggerRequestSchema, EvaluateRequestSchema, FactSchema, ListEdgesRequestSchema, ListNodesRequestSchema, ListProposalsRequestSchema, MergeNodesRequestSchema, NodeKind, PromoteRequestSchema, ProjectionRequestSchema, ReleaseLadderRequestSchema, ReleaseLadderService, PrerequisiteWalkRequestSchema, SetReleaseRankRequestSchema, SourceMode, Status, TransitionRequestSchema, TriggerClauseSchema, TriggerComposition, TriggerSchema, VerifyCatalogRequestSchema } from "@vrooli/proto-types/offer-desk/v1/offers/offers_pb";
import { transport } from "./client";

export const boardClient = createClient(BoardService, transport);
export const catalogClient = createClient(CatalogService, transport);
export const gatesClient = createClient(GatesService, transport);
export const releaseLadderClient = createClient(ReleaseLadderService, transport);
export function fetchBoard() { return boardClient.getBoard(create(ProjectionRequestSchema)); }
export function fetchNodes() { return catalogClient.listNodes(create(ListNodesRequestSchema)); }
export function fetchReleaseLadder(includeRetired = false) { return releaseLadderClient.getReleaseLadder(create(ReleaseLadderRequestSchema, { includeRetired })); }
export function fetchPrerequisites(streamNodeId: string) { return releaseLadderClient.getPrerequisites(create(PrerequisiteWalkRequestSchema, { streamNodeId })); }
export function setReleaseRank(input: { nodeId: string; releaseRank: number }) { return catalogClient.setReleaseRank(create(SetReleaseRankRequestSchema, { ...input, actor: "operator" })); }
export function fetchEdges() { return catalogClient.listEdges(create(ListEdgesRequestSchema)); }
export function fetchCatalogVerification(sourcePath = "docs/monetization") {
  return catalogClient.verifyCatalog(create(VerifyCatalogRequestSchema, { sourcePath, sourceMode: SourceMode.OPERATOR_SUPPLIED }));
}
export function fetchProposals() { return gatesClient.listProposals(create(ListProposalsRequestSchema)); }
export function mergeNodes(input: { survivingId: string; duplicateId: string; actor: string; dryRun: boolean }) {
  return catalogClient.mergeNodes(create(MergeNodesRequestSchema, input));
}
export function evaluateTriggers(dryRun = true) { return gatesClient.evaluate(create(EvaluateRequestSchema, { dryRun })); }

export function createNode(input: { kind: NodeKind; name: string; status: Status; triggerId?: string; actualAccountId?: string }) {
  return catalogClient.createNode(create(CreateNodeRequestSchema, { kind: input.kind, name: input.name, status: input.status, triggerId: input.triggerId ?? "", actualAccountId: input.actualAccountId ?? "" }));
}

export function transition(input: { nodeId: string; status: Status; actor: string }) {
  return catalogClient.transition(create(TransitionRequestSchema, input));
}

export function createEdge(input: { fromId: string; toId: string; kind: string; intendedPriceMinor?: bigint; currency?: string }) {
  return catalogClient.createEdge(create(CreateEdgeRequestSchema, { edge: create(EdgeSchema, { fromId: input.fromId, toId: input.toId, kind: input.kind, intendedPriceMinor: input.intendedPriceMinor ?? 0n, currency: input.currency ?? "" }) }));
}

export function declareTrigger(input: { id?: string; nodeId: string; factName: string; operator: string; threshold: number; clauses?: Array<{ factName: string; operator: string; threshold: number }>; composition?: TriggerComposition }) {
  return gatesClient.declareTrigger(create(DeclareTriggerRequestSchema, { trigger: create(TriggerSchema, { id: input.id ?? "", nodeId: input.nodeId, factName: input.factName, operator: input.operator, threshold: input.threshold, expression: "", clauses: (input.clauses ?? []).map((clause) => create(TriggerClauseSchema, clause)), composition: input.composition ?? TriggerComposition.ALL }) }));
}

export function addFact(input: { name: string; value: number; observedAt: Date; staleAfterDays: number; dimension?: string }) {
  return gatesClient.addFact(create(AddFactRequestSchema, { fact: create(FactSchema, { name: input.name, value: input.value, observedAt: timestampFromDate(input.observedAt), staleAfterDays: input.staleAfterDays, dimension: input.dimension ?? "" }) }));
}

export function promoteNode(input: { nodeId: string; actor: string; role: string }) {
  return gatesClient.promote(create(PromoteRequestSchema, input));
}
