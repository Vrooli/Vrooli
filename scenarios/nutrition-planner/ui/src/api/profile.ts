import { Code, ConnectError, createClient } from "@connectrpc/connect";
import { create } from "@bufbuild/protobuf";
import { KitchenDeviceSchema, ProfileService, type Profile } from "@vrooli/proto-types/nutrition-planner/v1/profile/profile_pb";
import { transport } from "./client";

const client = createClient(ProfileService, transport);

export async function getProfile(workspaceId: string): Promise<Profile | undefined> {
  const response = await client.getProfile({ workspaceId });
  return response.profile;
}

export async function saveKitchenDevices(workspaceId: string, devices: Array<{ id: string; type: string; count: number; capacity: string; capabilities: string[] }>): Promise<Profile> {
  const current = await getProfile(workspaceId);
  if (!current) throw new Error("Complete profile setup before saving physical devices.");
  const response = await client.applyProfile({
    workspaceId, preset: current.preset, excludedGroups: current.excludedGroups, allergies: current.allergies,
    appliances: current.appliances, cookware: current.cookware, tools: current.tools,
    costWeight: current.costWeight, effortWeight: current.effortWeight, varietyWeight: current.varietyWeight,
    kitchenDevices: devices.map((device) => create(KitchenDeviceSchema, device)), replaceKitchenDevices: true,
  });
  if (!response.profile) throw new Error("The API returned no saved kitchen devices.");
  return response.profile;
}

export async function saveProfileDraft(workspaceId: string, draftJson: string): Promise<Profile> {
  const response = await client.saveProfileDraft({ workspaceId, draftJson });
  if (!response.profile) throw new Error("The API returned no saved onboarding draft.");
  return response.profile;
}

export async function applyProfile(input: { workspaceId: string; preset: string; excludedGroups: string[]; allergies: string[]; appliances: string[]; cookware?: string[]; tools?: string[]; costWeight: number; effortWeight: number; varietyWeight: number }): Promise<{ profile: Profile; matchingMeals: bigint; needsReviewMeals: bigint; excludedMeals: bigint }> {
  let existing: Profile | undefined;
  if (input.cookware === undefined || input.tools === undefined) {
    try {
      existing = await getProfile(input.workspaceId);
    } catch (reason: unknown) {
      if (!(reason instanceof ConnectError) || reason.code !== Code.NotFound) throw reason;
    }
  }
  const response = await client.applyProfile({
    ...input,
    cookware: input.cookware ?? existing?.cookware ?? [],
    tools: input.tools ?? existing?.tools ?? [],
  });
  if (!response.profile) throw new Error("The API returned no applied profile.");
  return { profile: response.profile, matchingMeals: response.matchingMeals, needsReviewMeals: response.needsReviewMeals, excludedMeals: response.excludedMeals };
}
