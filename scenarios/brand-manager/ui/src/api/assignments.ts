import { createClient } from "@connectrpc/connect";
import {
  AssignmentsService,
  type Assignment,
  type ListAssignmentsResponse,
  type ScenarioStatus,
} from "@vrooli/proto-types/brand-manager/v1/assignments/assignments_pb";

import { transport } from "./client";

export const assignmentsClient = createClient(AssignmentsService, transport);

/** listAssignments returns assignments ordered newest-applied first. */
export async function listAssignments(brandId = ""): Promise<Assignment[]> {
  const resp = await assignmentsClient.listAssignments({ brandId });
  return resp.assignments;
}

/** assignBrand links a brand to a scenario and returns the persisted assignment. */
export async function assignBrand(input: {
  brandId: string;
  scenarioName: string;
  elements?: string[];
}): Promise<Assignment> {
  const resp = await assignmentsClient.assignBrand({
    brandId: input.brandId,
    scenarioName: input.scenarioName,
    elements: input.elements ?? [],
  });
  if (!resp.assignment) {
    throw new Error("assign returned no assignment");
  }
  return resp.assignment;
}

/** getScenarioStatus returns a scenario's branding status. */
export async function getScenarioStatus(scenarioName: string): Promise<ScenarioStatus> {
  const resp = await assignmentsClient.getScenarioStatus({ scenarioName });
  if (!resp.status) {
    throw new Error("status returned no status");
  }
  return resp.status;
}

export type { Assignment, ListAssignmentsResponse, ScenarioStatus };
