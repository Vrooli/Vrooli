import { describe, expect, it } from "vitest";
import type { ReadinessState } from "@vrooli/proto-types/device-control/v1/desktop/desktop_pb";
import { canMutateDesktop, desktopViewState } from "./desktopState";

describe("desktop state projection", () => {
  it("does not infer readiness from an absent or blocked host session", () => {
    expect(desktopViewState()).toBe("offline");
    expect(desktopViewState({ state: 2 as ReadinessState })).toBe("blocked");
    expect(desktopViewState({ state: 4 as ReadinessState })).toBe("blocked");
  });
  it("only enables mutations for a current controller lease", () => {
    expect(canMutateDesktop({ state: 6 as ReadinessState }, { connectionState: "connected", controller: false, leaseId: "l", leaseEpoch: 1n })).toBe(false);
    expect(canMutateDesktop({ state: 6 as ReadinessState }, { connectionState: "connected", controller: true, leaseId: "l", leaseEpoch: 1n })).toBe(true);
    expect(canMutateDesktop({ state: 6 as ReadinessState }, { connectionState: "revoked", controller: true, leaseId: "l", leaseEpoch: 2n })).toBe(false);
  });

  it("projects a peer transport drop as disconnected rather than ready", () => {
    expect(desktopViewState({ state: 6 as ReadinessState }, { connectionState: "disconnected", controller: true, leaseId: "l", leaseEpoch: 1n })).toBe("disconnected");
  });
});
