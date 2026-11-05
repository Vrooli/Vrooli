import { createClient } from "@connectrpc/connect";
import { WorkspaceService, type AvailabilityException, type AvailabilityWindow, type PlanningProfile } from "@vrooli/proto-types/personal-planner/v1/workspace/workspace_pb";

import { transport } from "./client";

const workspaceClient = createClient(WorkspaceService, transport);

export async function fetchPlanningProfile(): Promise<PlanningProfile> {
  const response = await workspaceClient.getProfile({});
  if (!response.profile) throw new Error("The server returned no planning profile");
  return response.profile;
}

export async function updatePlanningProfile(profile: PlanningProfile, input: {
  timezone: string;
  weekStart: string;
  dailyCapacityMinutes: number;
  reserveMinutes: number;
  focusSessionMinutes: number;
}): Promise<PlanningProfile> {
  const response = await workspaceClient.updateProfile({ ...input, expectedRevision: profile.revision });
  if (!response.profile) throw new Error("The server returned no planning profile");
  return response.profile;
}

export async function fetchAvailability(): Promise<{ windows: AvailabilityWindow[]; exceptions: AvailabilityException[]; revision: bigint }> {
  const response = await workspaceClient.listAvailability({});
  return { windows: response.windows, exceptions: response.exceptions, revision: response.revision };
}

export async function replaceAvailability(input: {
  windows: AvailabilityWindow[];
  exceptions: AvailabilityException[];
  expectedRevision: bigint;
}): Promise<{ windows: AvailabilityWindow[]; exceptions: AvailabilityException[]; revision: bigint }> {
  const response = await workspaceClient.replaceAvailability(input);
  return { windows: response.windows, exceptions: response.exceptions, revision: response.revision };
}
