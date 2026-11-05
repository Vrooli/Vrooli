import { describe, expect, it, vi } from "vitest";
import { expose, exposureClient, listExposures, listLeases, LeaseStatus } from "./exposure";

describe("exposure API helpers", () => {
  it("unwraps exposures", async () => {
    const exposures = [{ scenario: "alpha" }];
    const spy = vi.spyOn(exposureClient, "listExposures").mockResolvedValueOnce({ exposures } as never);
    await expect(listExposures()).resolves.toBe(exposures);
    expect(spy).toHaveBeenCalledWith({});
  });
  it("forwards lease status", async () => {
    const leases = [{ scenario: "alpha" }];
    const spy = vi.spyOn(exposureClient, "listLeases").mockResolvedValueOnce({ leases } as never);
    await expect(listLeases(LeaseStatus.ACTIVE)).resolves.toBe(leases);
    expect(spy).toHaveBeenCalledWith({ status: LeaseStatus.ACTIVE });
  });
  it("uses safe defaults when granting exposure", async () => {
    const lease = { scenario: "alpha" };
    const spy = vi.spyOn(exposureClient, "expose").mockResolvedValueOnce({ lease, publicUrl: "https://alpha.example" } as never);
    await expect(expose("alpha")).resolves.toEqual({ lease, publicUrl: "https://alpha.example" });
    expect(spy).toHaveBeenCalledWith({ scenario: "alpha", ttlSeconds: 0n, requestedBy: "" });
  });
});
