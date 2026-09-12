import { createClient } from "@connectrpc/connect";
import { fromJson } from "@bufbuild/protobuf";
import { ValueSchema } from "@bufbuild/protobuf/wkt";
import { onboardingTransport } from "./base";
import {
  ProfileService,
  type EvaluateProfileResponse,
  type ListProfilesResponse,
} from "@vrooli/proto-types/vrooli-onboarding/v1/profiles/profiles_pb";

const client = createClient(ProfileService, onboardingTransport());

export function fetchProfiles(target = "local"): Promise<ListProfilesResponse> {
  return client.listProfiles({ target }) as unknown as Promise<ListProfilesResponse>;
}

export function evaluateProfile(
  profileId: string,
  answers: Record<string, unknown>,
  target = "local",
  manualDecisions: Record<string, boolean> = {},
): Promise<EvaluateProfileResponse> {
  const encodedAnswers = Object.fromEntries(
    Object.entries(answers).map(([key, value]) => [key, fromJson(ValueSchema, value as never)]),
  );
  return client.evaluateProfile({
    target,
    profileId,
    answers: encodedAnswers,
    manualDecisions,
  }) as unknown as Promise<EvaluateProfileResponse>;
}
