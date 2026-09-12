import { createClient } from "@connectrpc/connect";
import {
  FamiliesService,
  ReviewDecision,
  type FamilyEdge,
  type GetFrontierResponse,
  type PlanFamily,
} from "@vrooli/proto-types/plan-manager/v1/families/families_pb";

import { transport } from "./client";

export const familiesClient = createClient(FamiliesService, transport);

export async function listFamilies(): Promise<PlanFamily[]> {
  return (await familiesClient.listFamilies({ pageSize: 100 })).families;
}

export async function createFamily(input: { slug: string; outcome: string; sharedContext: string; maximumParallelPlans: number }): Promise<PlanFamily> {
  const response = await familiesClient.createFamily({
    slug: input.slug,
    outcome: input.outcome,
    sharedContext: input.sharedContext,
    policy: { maximumParallelPlans: input.maximumParallelPlans, unknownInteractionsSequential: true, requireReviewBeforeLaunch: true, validationPolicy: "" },
  });
  if (!response.family) throw new Error("server returned no family");
  return response.family;
}

export async function getFamily(familyId: string): Promise<PlanFamily> {
  const response = await familiesClient.getFamily({ familyId });
  if (!response.family) throw new Error("server returned no family");
  return response.family;
}

export function getFrontier(familyId: string): Promise<GetFrontierResponse> {
  return familiesClient.getFrontier({ familyId });
}

export async function proposeGraph(family: PlanFamily): Promise<PlanFamily> {
  const response = await familiesClient.proposeGraph({ familyId: family.familyId, expectedRevision: family.revision, explicitEdges: family.graph?.edges ?? [] });
  if (!response.family) throw new Error("server returned no family");
  return response.family;
}

export async function reviewGraph(input: { family: PlanFamily; decision: ReviewDecision; reviewer: string; rationale: string; correctedEdges: FamilyEdge[] }): Promise<PlanFamily> {
  if (!input.family.graph) throw new Error("family has no graph proposal");
  const response = await familiesClient.reviewGraph({
    familyId: input.family.familyId,
    expectedRevision: input.family.revision,
    graphRevision: input.family.graph.revision,
    decision: input.decision,
    reviewer: input.reviewer,
    rationale: input.rationale,
    correctedEdges: input.correctedEdges,
  });
  if (!response.family) throw new Error("server returned no family");
  return response.family;
}
