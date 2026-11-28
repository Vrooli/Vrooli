import { createClient } from "@connectrpc/connect";
import {
  ExportFormat,
  GraphService,
  type DescribeTechTreeResponse,
  type ExportTechTreeResponse,
} from "@vrooli/proto-types/tech-tree-designer/v1/graph/graph_pb";
import {
  CapabilityKind,
  OntologyService,
  type Capability,
  type CoverageSummary,
  type Fulfillment,
  type ListFocusResponse,
  type OverlayGraph,
} from "@vrooli/proto-types/tech-tree-designer/v1/ontology/ontology_pb";
import {
  PlanningService,
  type MaterializePlannedScenarioResponse,
  type PlannedProtoFile,
  type PlannedScenario,
  type ValidatePlannedScenarioResponse,
} from "@vrooli/proto-types/tech-tree-designer/v1/planning/planning_pb";

import { transport } from "./client";

export const graphClient = createClient(GraphService, transport);
export const ontologyClient = createClient(OntologyService, transport);
export const planningClient = createClient(PlanningService, transport);

export type GroupBy = "none" | "sector" | "tier";

export async function describeTechTree(groupBy: GroupBy): Promise<DescribeTechTreeResponse> {
  return graphClient.describeTechTree({
    scenarioFilter: [],
    limit: 0,
    stabilityFilter: "",
    groupBy: groupBy === "none" ? "" : groupBy,
  });
}

export async function exportTechTree(format: ExportFormat): Promise<ExportTechTreeResponse> {
  return graphClient.exportTechTree({
    format,
    scenarioFilter: [],
    stabilityFilter: "",
  });
}

export async function listPlannedScenarios(): Promise<PlannedScenario[]> {
  const response = await planningClient.listPlannedScenarios({ sector: "", tier: "" });
  return response.scenarios;
}

export async function createPlannedScenario(input: {
  slug: string;
  displayName: string;
  sector: string;
  tier: string;
}): Promise<PlannedScenario> {
  return planningClient.createPlannedScenario({
    ...input,
    targetStability: "experimental",
  });
}

export async function getPlannedScenario(slug: string): Promise<PlannedScenario> {
  return planningClient.getPlannedScenario({ slug });
}

export async function putPlannedProtoFile(input: {
  slug: string;
  path: string;
  text: string;
}): Promise<PlannedProtoFile> {
  return planningClient.putPlannedProtoFile(input);
}

export async function deletePlannedProtoFile(input: {
  slug: string;
  path: string;
}): Promise<boolean> {
  const response = await planningClient.deletePlannedProtoFile(input);
  return response.deleted;
}

export async function validatePlannedScenario(
  slug: string,
): Promise<ValidatePlannedScenarioResponse> {
  return planningClient.validatePlannedScenario({ slug });
}

export async function materializePlannedScenario(
  slug: string,
): Promise<MaterializePlannedScenarioResponse> {
  return planningClient.materializePlannedScenario({ slug });
}

export async function listCapabilities(): Promise<Capability[]> {
  const response = await ontologyClient.listCapabilities({
    parentId: "",
    kind: CapabilityKind.UNSPECIFIED,
    includeDescendants: true,
  });
  return response.capabilities;
}

export async function getOntologyCoverage(): Promise<CoverageSummary> {
  return ontologyClient.getCoverage({ includeSubtreeRollup: true });
}

export async function listOntologyFocus(): Promise<ListFocusResponse> {
  return ontologyClient.listFocus({ limit: 8 });
}

export async function listFulfillments(): Promise<Fulfillment[]> {
  const response = await ontologyClient.listFulfillments({ capabilityId: "", scenarioSlug: "" });
  return response.fulfillments;
}

export async function unlinkFulfillment(input: {
  capabilityId: string;
  scenarioSlug: string;
}): Promise<boolean> {
  const response = await ontologyClient.unlinkFulfillment(input);
  return response.deleted;
}

export async function describeOverlayGraph(): Promise<OverlayGraph | undefined> {
  const response = await ontologyClient.describeOverlayGraph({
    includeImplementation: true,
    includeOntology: true,
    includeFulfillment: true,
  });
  return response.graph;
}

export { ExportFormat };
export type {
  Capability,
  CoverageSummary,
  DescribeTechTreeResponse,
  ExportTechTreeResponse,
  Fulfillment,
  ListFocusResponse,
  OverlayGraph,
  PlannedProtoFile,
  PlannedScenario,
  ValidatePlannedScenarioResponse,
  MaterializePlannedScenarioResponse,
};
