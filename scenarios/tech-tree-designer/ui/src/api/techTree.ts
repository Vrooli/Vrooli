import { createClient } from "@connectrpc/connect";
import {
  ExportFormat,
  GraphService,
  type DescribeTechTreeResponse,
  type ExportTechTreeResponse,
} from "@vrooli/proto-types/tech-tree-designer/v1/graph/graph_pb";
import {
  PlanningService,
  type MaterializePlannedScenarioResponse,
  type PlannedProtoFile,
  type PlannedScenario,
  type ValidatePlannedScenarioResponse,
} from "@vrooli/proto-types/tech-tree-designer/v1/planning/planning_pb";
import {
  RoadmapService,
  type ListMilestonesResponse,
  type ListSectorsResponse,
  type ProgressRollup,
} from "@vrooli/proto-types/tech-tree-designer/v1/roadmap/roadmap_pb";

import { transport } from "./client";

export const graphClient = createClient(GraphService, transport);
export const planningClient = createClient(PlanningService, transport);
export const roadmapClient = createClient(RoadmapService, transport);

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

export async function listSectors(): Promise<ListSectorsResponse> {
  return roadmapClient.listSectors({});
}

export async function listMilestones(): Promise<ListMilestonesResponse> {
  return roadmapClient.listMilestones({});
}

export async function getProgress(): Promise<ProgressRollup> {
  return roadmapClient.getProgress({ sector: "", tier: "" });
}

export { ExportFormat };
export type {
  DescribeTechTreeResponse,
  ExportTechTreeResponse,
  PlannedProtoFile,
  PlannedScenario,
  ValidatePlannedScenarioResponse,
  MaterializePlannedScenarioResponse,
  ListMilestonesResponse,
  ListSectorsResponse,
  ProgressRollup,
};
