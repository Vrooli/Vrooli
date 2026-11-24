import { describe, expect, it } from "vitest";
import { hostMutationCapability, onboardingTargetCapability, requireHostMutationCapability } from "./hostCapabilities";

describe("Onboarding host mutation capability", () => {
  it("allows local owner writes and rejects all declared mutations on remote targets", () => {
    for (const operation of ["tool", "safeguard", "notification-recipient"] as const) {
      expect(hostMutationCapability("local", operation)).toEqual({ supported: true });
      expect(hostMutationCapability("", operation).supported).toBe(false);
      expect(hostMutationCapability("   ", operation).supported).toBe(false);
      const remote = hostMutationCapability("node-1", operation);
      expect(remote.supported).toBe(false);
      expect(remote.reason).toMatch(/only be changed|only be changed by/);
      expect(() => requireHostMutationCapability("node-1", operation)).toThrow(remote.reason);
      expect(() => requireHostMutationCapability("local", operation)).not.toThrow();
      expect(() => requireHostMutationCapability("", operation)).toThrow(/Choose the local host or an available remote node/);
    }
  });

  it("describes the supported target classes and their owner boundary", () => {
    expect(onboardingTargetCapability("local")).toMatchObject({ kind: "local", profileSession: "owner-scoped", hostRequirements: "read-write", hostPermissionWrites: true });
    const targets = [{ id: "node-1", kind: "NODE_KIND_AGENT" }, { id: "ssh-1", kind: "NODE_KIND_SSH" }, { id: "phone-1", kind: "NODE_KIND_ATTACHED" }];
    expect(onboardingTargetCapability("node-1", targets)).toMatchObject({ kind: "remote-node", profileSession: "owner-scoped", hostRequirements: "read-only", hostPermissionWrites: false });
    expect(onboardingTargetCapability("ssh-1", targets)).toMatchObject({ kind: "remote-node", profileSession: "owner-scoped" });
    expect(onboardingTargetCapability("phone-1", targets)).toMatchObject({ kind: "unsupported", reason: expect.stringContaining("not a registered host") });
    expect(onboardingTargetCapability("unknown", targets)).toMatchObject({ kind: "unsupported", reason: expect.stringContaining("not a registered host") });
    expect(onboardingTargetCapability(" ")).toMatchObject({ kind: "unsupported", profileSession: "unsupported", hostRequirements: "unsupported", hostPermissionWrites: false });
  });
});
