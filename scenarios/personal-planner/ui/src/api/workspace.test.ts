import { afterEach, describe, expect, it, vi } from "vitest";

const client = vi.hoisted(() => ({ getProfile: vi.fn(), updateProfile: vi.fn(), listAvailability: vi.fn(), replaceAvailability: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => client }));

import { fetchAvailability, fetchPlanningProfile, replaceAvailability, updatePlanningProfile } from "./workspace";

afterEach(() => vi.clearAllMocks());

describe("workspace API transport", () => {
  it("reads and revision-protects the planning profile", async () => {
    const profile = { id: "default", revision: 2n, timezone: "UTC" } as never;
    client.getProfile.mockResolvedValue({ profile });
    client.updateProfile.mockResolvedValue({ profile });

    expect(await fetchPlanningProfile()).toBe(profile);
    expect(await updatePlanningProfile(profile, { timezone: "America/New_York", weekStart: "monday", dailyCapacityMinutes: 420, reserveMinutes: 60, focusSessionMinutes: 45 })).toBe(profile);
    expect(client.updateProfile).toHaveBeenCalledWith({ timezone: "America/New_York", weekStart: "monday", dailyCapacityMinutes: 420, reserveMinutes: 60, focusSessionMinutes: 45, expectedRevision: 2n });
  });

  it("rejects malformed profile responses", async () => {
    client.getProfile.mockResolvedValue({});
    await expect(fetchPlanningProfile()).rejects.toThrow("no planning profile");
    client.updateProfile.mockResolvedValue({});
    await expect(updatePlanningProfile({ revision: 1n } as never, { timezone: "UTC", weekStart: "monday", dailyCapacityMinutes: 480, reserveMinutes: 60, focusSessionMinutes: 45 })).rejects.toThrow("no planning profile");
  });

  it("reads and replaces availability with the workspace revision", async () => {
    const response = { windows: [], exceptions: [], revision: 4n } as never;
    client.listAvailability.mockResolvedValueOnce(response);
    client.replaceAvailability.mockResolvedValueOnce(response);
    expect(await fetchAvailability()).toEqual(response);
    expect(await replaceAvailability({ windows: [], exceptions: [], expectedRevision: 4n })).toEqual(response);
    expect(client.listAvailability).toHaveBeenCalledWith({});
    expect(client.replaceAvailability).toHaveBeenCalledWith({ windows: [], exceptions: [], expectedRevision: 4n });
  });
});
