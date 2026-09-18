import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { ProfileSchema } from "@vrooli/proto-types/nutrition-planner/v1/profile/profile_pb";

const getProfile = vi.hoisted(() => vi.fn());
const saveProfileDraft = vi.hoisted(() => vi.fn());
const applyProfile = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ getProfile, saveProfileDraft, applyProfile }) }));
vi.mock("./client", () => ({ transport: {} }));

import { applyProfile as apply, getProfile as get, saveProfileDraft as save } from "./profile";

describe("profile API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("reads the current profile", async () => { const profile = create(ProfileSchema, { workspaceId: "w1" }); getProfile.mockResolvedValue({ profile }); await expect(get("w1")).resolves.toBe(profile); });
  it("saves a draft and rejects a hollow response", async () => { const profile = create(ProfileSchema, { workspaceId: "w1" }); saveProfileDraft.mockResolvedValue({ profile }); await expect(save("w1", "{}" )).resolves.toBe(profile); saveProfileDraft.mockResolvedValue({}); await expect(save("w1", "{}" )).rejects.toThrow("no saved onboarding draft"); });
  it("applies a profile and rejects a hollow response", async () => { const profile = create(ProfileSchema, { workspaceId: "w1" }); applyProfile.mockResolvedValue({ profile }); const input = { workspaceId: "w1", preset: "vegan", excludedGroups: [], allergies: [], appliances: [], costWeight: .34, effortWeight: .33, varietyWeight: .33 }; await expect(apply(input)).resolves.toBe(profile); applyProfile.mockResolvedValue({}); await expect(apply(input)).rejects.toThrow("no applied profile"); });
});
