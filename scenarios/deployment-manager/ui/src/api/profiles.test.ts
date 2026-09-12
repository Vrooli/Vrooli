import { beforeEach, describe, expect, it, vi } from "vitest";

const clients = vi.hoisted(() => ({
  listProfiles: vi.fn(),
  createProfile: vi.fn(),
  getProfile: vi.fn(),
  updateProfile: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: () => clients,
}));

vi.mock("./transport", () => ({
  scenarioTransport: {},
}));

import { createProfile, getProfile, listProfiles, updateProfile } from "./profiles";

const profile = (overrides: Record<string, unknown> = {}) => ({
  id: "profile-1",
  name: "Production",
  scenario: "demo",
  tiers: [1, 2],
  version: 3,
  createdAt: { seconds: 10n, nanos: 500000000 },
  updatedAt: { seconds: 20n, nanos: 0 },
  swaps: {
    fields: {
      bool: { kind: { case: "boolValue", value: true } },
      double: { kind: { case: "doubleValue", value: 1.5 } },
      string: { kind: { case: "stringValue", value: "value" } },
      int: { kind: { case: "intValue", value: 7n } },
      null: { kind: { case: "nullValue", value: 0 } },
      list: { kind: { case: "listValue", value: { values: [{ kind: { case: "stringValue", value: "nested" } }] } } },
      object: { kind: { case: "objectValue", value: { fields: { child: { kind: { case: "stringValue", value: "value" } } } } } },
      unknown: { kind: { case: "unknown", value: 0 } },
    },
  },
  ...overrides,
});

describe("Profiles API adapter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("maps protobuf profile values and sends complete create/update requests", async () => {
    clients.listProfiles.mockResolvedValue({ profiles: [profile(), profile({ swaps: undefined, createdAt: undefined, updatedAt: undefined })] });
    const listed = await listProfiles();
    expect(listed[0]!).toMatchObject({ id: "profile-1", created_at: "1970-01-01T00:00:10.500Z" });
    expect(listed[0]!.swaps).toMatchObject({ bool: true, double: 1.5, string: "value", int: 7, null: null, list: ["nested"], object: { child: "value" } });
    expect(listed[0]!.swaps?.unknown).toBeUndefined();
    expect(listed[1]!.swaps).toBeUndefined();

    clients.createProfile.mockResolvedValue({ profile: { id: "new-profile", version: 4 } });
    await expect(createProfile({
      name: "New", scenario: "demo", tiers: [1],
      swaps: { null: null, bool: true, number: 2, text: "x", list: ["x"], object: { nested: false }, unsupported: undefined },
      secrets: {}, settings: {},
    })).resolves.toEqual({ id: "new-profile", version: 4 });
    expect(clients.createProfile).toHaveBeenCalledOnce();

    clients.updateProfile.mockResolvedValue({ profile: profile({ name: "Updated", createdAt: undefined, updatedAt: undefined }) });
    const updated = await updateProfile("profile-1", { name: "Updated", scenario: "demo", tiers: [2], swaps: { enabled: true }, secrets: {}, settings: {} });
    expect(updated.name).toBe("Updated");
    expect(updated.created_at).toBeUndefined();
  });

  it("reports missing profiles from create, get, and update responses", async () => {
    clients.createProfile.mockResolvedValue({});
    await expect(createProfile({ name: "x", scenario: "y", tiers: [] })).rejects.toThrow("returned no profile");
    clients.getProfile.mockResolvedValue({});
    await expect(getProfile("missing")).rejects.toThrow("Profile missing was not found");
    clients.updateProfile.mockResolvedValue({});
    await expect(updateProfile("missing", {})).rejects.toThrow("Profile missing was not found");
  });
});
