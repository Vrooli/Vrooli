import { createClient } from "@connectrpc/connect";
import {
  FacetsService,
  type Facet,
  type PinProposal,
} from "@vrooli/proto-types/vrooli-memory/v1/facets/facets_pb";
import { ForestService, type Node } from "@vrooli/proto-types/vrooli-memory/v1/forest/forest_pb";

import { transport } from "./client";

const facetsClient = createClient(FacetsService, transport);
const forestClient = createClient(ForestService, transport);

export async function listFacets(): Promise<Facet[]> {
  const response = await facetsClient.listFacets({});
  return response.facets;
}

export async function setPin(entryId: string, pinned: boolean): Promise<void> {
  await facetsClient.setPin({ entryId, pinned });
}

export async function assignFacet(entryId: string, facetId: string): Promise<void> {
  await facetsClient.assignFacet({ entryId, facetId });
}

export async function listPinProposals(): Promise<PinProposal[]> {
  const response = await facetsClient.listPinProposals({});
  return response.proposals;
}

export async function resolvePinProposal(proposalId: string, accept: boolean): Promise<void> {
  await facetsClient.resolvePinProposal({ proposalId, accept });
}

export async function getFrontier(): Promise<Node[]> {
  const response = await forestClient.getFrontier({});
  return response.nodes;
}
