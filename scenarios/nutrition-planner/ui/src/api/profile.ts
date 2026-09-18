import { createClient } from "@connectrpc/connect";
import { ProfileService, type Profile } from "@vrooli/proto-types/nutrition-planner/v1/profile/profile_pb";
import { transport } from "./client";

const client = createClient(ProfileService, transport);

export async function getProfile(workspaceId: string): Promise<Profile | undefined> {
  const response = await client.getProfile({ workspaceId });
  return response.profile;
}

export async function saveProfileDraft(workspaceId: string, draftJson: string): Promise<Profile> {
  const response = await client.saveProfileDraft({ workspaceId, draftJson });
  if (!response.profile) throw new Error("The API returned no saved onboarding draft.");
  return response.profile;
}

export async function applyProfile(input: { workspaceId: string; preset: string; excludedGroups: string[]; allergies: string[]; appliances: string[]; costWeight: number; effortWeight: number; varietyWeight: number }): Promise<{ profile: Profile; matchingMeals: bigint }> {
  const response = await client.applyProfile(input);
  if (!response.profile) throw new Error("The API returned no applied profile.");
  return { profile: response.profile, matchingMeals: response.matchingMeals };
}
