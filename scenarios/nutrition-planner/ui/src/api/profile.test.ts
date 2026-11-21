import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { ProfileSchema } from "@vrooli/proto-types/nutrition-planner/v1/profile/profile_pb";

const getProfile = vi.hoisted(() => vi.fn());
const saveProfileDraft = vi.hoisted(() => vi.fn());
const applyProfile = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ Code: { NotFound: 5 }, ConnectError: class extends Error { code: number; constructor(message: string, code: number) { super(message); this.code = code; } }, createClient: () => ({ getProfile, saveProfileDraft, applyProfile }) }));
vi.mock("./client", () => ({ transport: {} }));

import { applyProfile as apply, getProfile as get, saveProfileDraft as save } from "./profile";

describe("profile API", () => {
  beforeEach(() => { vi.clearAllMocks(); getProfile.mockResolvedValue({}); });
  it("reads the current profile", async () => { const profile = create(ProfileSchema, { workspaceId: "w1" }); getProfile.mockResolvedValue({ profile }); await expect(get("w1")).resolves.toBe(profile); });
  it("saves a draft and rejects a hollow response", async () => { const profile = create(ProfileSchema, { workspaceId: "w1" }); saveProfileDraft.mockResolvedValue({ profile }); await expect(save("w1", "{}" )).resolves.toBe(profile); saveProfileDraft.mockResolvedValue({}); await expect(save("w1", "{}" )).rejects.toThrow("no saved onboarding draft"); });
  it("applies a profile and preserves the eligibility counts", async () => { const profile = create(ProfileSchema, { workspaceId: "w1" }); applyProfile.mockResolvedValue({ profile, matchingMeals: 3n, needsReviewMeals: 2n, excludedMeals: 4n }); const input = { workspaceId: "w1", preset: "vegan", excludedGroups: [], allergies: [], appliances: [], costWeight: .34, effortWeight: .33, varietyWeight: .33 }; await expect(apply(input)).resolves.toEqual({ profile, matchingMeals: 3n, needsReviewMeals: 2n, excludedMeals: 4n }); expect(applyProfile).toHaveBeenLastCalledWith({ ...input, cookware: [], tools: [] }); applyProfile.mockResolvedValue({}); await expect(apply(input)).rejects.toThrow("no applied profile"); });
  it("preserves saved categories when older profile callers omit them", async () => {
    const current = create(ProfileSchema, { workspaceId: "w1", cookware: ["saucepan"], tools: ["spatula"] });
    getProfile.mockResolvedValue({ profile: current });
    applyProfile.mockResolvedValue({ profile: current, matchingMeals: 0n, needsReviewMeals: 0n, excludedMeals: 0n });
    const input = { workspaceId: "w1", preset: "vegan", excludedGroups: [], allergies: [], appliances: [], costWeight: .34, effortWeight: .33, varietyWeight: .33 };
    await apply(input);
    expect(applyProfile).toHaveBeenCalledWith({ ...input, cookware: ["saucepan"], tools: ["spatula"] });
  });
  it("allows a first profile apply when no saved profile exists", async () => {
    getProfile.mockRejectedValue(new ConnectError("missing", Code.NotFound));
    const profile = create(ProfileSchema, { workspaceId: "w1" });
    applyProfile.mockResolvedValue({ profile, matchingMeals: 0n, needsReviewMeals: 0n, excludedMeals: 0n });
    const input = { workspaceId: "w1", preset: "vegan", excludedGroups: [], allergies: [], appliances: [], costWeight: .34, effortWeight: .33, varietyWeight: .33 };
    await apply(input);
    expect(applyProfile).toHaveBeenCalledWith({ ...input, cookware: [], tools: [] });
  });
  it("replaces device records through profile apply without losing category selections", async () => {
    const profile = create(ProfileSchema, { workspaceId: "w1", preset: "vegan", appliances: ["stove"], cookware: ["saucepan"], tools: ["spatula"], kitchenDevices: [] });
    getProfile.mockResolvedValue({ profile });
    applyProfile.mockResolvedValue({ profile });
    const devices = [{ id: "range-main", type: "range", count: 1, capacity: "5 burners", capabilities: ["cooktop", "oven"] }];
    await expect((await import("./profile")).saveKitchenDevices("w1", devices)).resolves.toBe(profile);
    expect(applyProfile).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", preset: "vegan", appliances: ["stove"], cookware: ["saucepan"], tools: ["spatula"], replaceKitchenDevices: true, kitchenDevices: [expect.objectContaining(devices[0])] }));
    applyProfile.mockResolvedValue({});
    await expect((await import("./profile")).saveKitchenDevices("w1", [])).rejects.toThrow("no saved kitchen devices");
  });
});
