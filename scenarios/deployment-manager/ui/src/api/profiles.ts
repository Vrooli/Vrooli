import { createClient } from "@connectrpc/connect";
import { create, fromJson } from "@bufbuild/protobuf";
import { scenarioTransport } from "./transport";
import {
  CreateProfileRequestSchema,
  GetProfileRequestSchema,
  ListProfilesRequestSchema,
  ProfilesService,
  UpdateProfileRequestSchema,
} from "@vrooli/proto-types/deployment-manager/v1/profiles/profiles_pb";
import {
  JsonListSchema,
  JsonObjectSchema,
  JsonValueSchema,
} from "@vrooli/proto-types/common/v1/types_pb";
import type { JsonObject, JsonValue } from "@vrooli/proto-types/common/v1/types_pb";
import type { Profile } from "@vrooli/proto-types/deployment-manager/v1/profiles/profiles_pb";

const client = createClient(
  ProfilesService,
  scenarioTransport,
);

// This is a UI view model, not a wire contract. All requests and responses
// cross the boundary through the generated ProfilesService client above.
export type DeploymentProfile = {
  id: string;
  name: string;
  scenario: string;
  tiers: number[];
  swaps?: Record<string, unknown>;
  secrets?: Record<string, unknown>;
  settings?: Record<string, unknown>;
  version: number;
  created_at?: string;
  updated_at?: string;
};

type CreateProfileInput = {
  name: string;
  scenario: string;
  tiers: number[];
  swaps?: Record<string, unknown>;
  secrets?: Record<string, unknown>;
  settings?: Record<string, unknown>;
};

function jsonValue(value: unknown): JsonValue {
  if (value === null) return create(JsonValueSchema, { kind: { case: "nullValue", value: 0 } });
  if (typeof value === "boolean") return create(JsonValueSchema, { kind: { case: "boolValue", value } });
  if (typeof value === "string") return create(JsonValueSchema, { kind: { case: "stringValue", value } });
  if (typeof value === "number") return create(JsonValueSchema, { kind: { case: "doubleValue", value } });
  if (Array.isArray(value)) {
    return create(JsonValueSchema, {
      kind: { case: "listValue", value: create(JsonListSchema, { values: value.map(jsonValue) }) },
    });
  }
  if (typeof value === "object") {
    return create(JsonValueSchema, {
      kind: { case: "objectValue", value: jsonObjectRequired(value as Record<string, unknown>) },
    });
  }
  return create(JsonValueSchema, { kind: { case: "nullValue", value: 0 } });
}

function jsonObject(value: Record<string, unknown> | undefined): JsonObject | undefined {
  if (value === undefined) return undefined;
  return jsonObjectRequired(value);
}

function jsonObjectRequired(value: Record<string, unknown>): JsonObject {
  return create(JsonObjectSchema, {
    fields: Object.fromEntries(Object.entries(value).map(([key, entry]) => [key, jsonValue(entry)])),
  });
}

function jsonPlain(value: JsonValue | undefined): unknown {
  if (value === undefined) return undefined;
  switch (value.kind.case) {
    case "boolValue":
    case "doubleValue":
    case "stringValue":
      return value.kind.value;
    case "intValue":
      return Number(value.kind.value);
    case "nullValue":
      return null;
    case "objectValue":
      return jsonRecord(value.kind.value);
    case "listValue":
      return value.kind.value.values.map(jsonPlain);
    default:
      return undefined;
  }
}

function jsonRecord(value: Profile["swaps"]): Record<string, unknown> | undefined {
  if (value === undefined) return undefined;
  return Object.fromEntries(Object.entries(value.fields).map(([key, entry]) => [key, jsonPlain(entry)]));
}

function timestampString(value: Profile["createdAt"]): string | undefined {
  if (value === undefined) return undefined;
  return new Date(Number(value.seconds) * 1000 + Math.floor(value.nanos / 1_000_000)).toISOString();
}

function toViewModel(profile: Profile): DeploymentProfile {
  return {
    id: profile.id,
    name: profile.name,
    scenario: profile.scenario,
    tiers: [...profile.tiers],
    swaps: jsonRecord(profile.swaps),
    secrets: jsonRecord(profile.secrets),
    settings: jsonRecord(profile.settings),
    version: profile.version,
    created_at: timestampString(profile.createdAt),
    updated_at: timestampString(profile.updatedAt),
  };
}

export async function listProfiles(): Promise<DeploymentProfile[]> {
  const response = await client.listProfiles({
    ...fromJson(ListProfilesRequestSchema, {}),
    pageSize: 100,
  });
  return response.profiles.map(toViewModel);
}

export async function createProfile(input: CreateProfileInput): Promise<{ id: string; version: number }> {
  const response = await client.createProfile({
    ...fromJson(CreateProfileRequestSchema, {}),
    name: input.name,
    scenario: input.scenario,
    tiers: [...input.tiers],
    swaps: jsonObject(input.swaps),
    secrets: jsonObject(input.secrets),
    settings: jsonObject(input.settings),
  });
  if (response.profile === undefined) {
    throw new Error("ProfilesService.CreateProfile returned no profile");
  }
  return { id: response.profile.id, version: response.profile.version };
}

export async function getProfile(id: string): Promise<DeploymentProfile> {
  const response = await client.getProfile({
    ...fromJson(GetProfileRequestSchema, {}),
    profileId: id,
  });
  if (response.profile === undefined) {
    throw new Error(`Profile ${id} was not found`);
  }
  return toViewModel(response.profile);
}

export async function updateProfile(
  id: string,
  updates: Partial<DeploymentProfile>,
): Promise<DeploymentProfile> {
  const response = await client.updateProfile({
    ...fromJson(UpdateProfileRequestSchema, {}),
    profileId: id,
    ...(updates.name === undefined ? {} : { name: updates.name }),
    ...(updates.scenario === undefined ? {} : { scenario: updates.scenario }),
    ...(updates.tiers === undefined ? {} : { tiers: [...updates.tiers] }),
    swaps: jsonObject(updates.swaps),
    secrets: jsonObject(updates.secrets),
    settings: jsonObject(updates.settings),
  });
  if (response.profile === undefined) {
    throw new Error(`Profile ${id} was not found`);
  }
  return toViewModel(response.profile);
}
