import { describe, expect, it, vi } from "vitest";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ConsentWeight,
  DeploymentTier,
  SafetyPolicySchema,
  type SafetyPolicy,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

const mocks = vi.hoisted(() => ({
  client: { getPolicy: vi.fn() },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

import { getPolicy, highConsentOps, needsConsent } from "./safety";

const makePolicy = (
  overrides: MessageInitShape<typeof SafetyPolicySchema> = {},
): SafetyPolicy =>
  create(SafetyPolicySchema, {
    tier: DeploymentTier.PUBLIC,
    requireConsent: true,
    forceNsfwScan: true,
    requireProvenance: true,
    rateLimitPerMin: 30,
    summary: "Public tier",
    opWeights: [
      { operation: "text_to_image", weight: ConsentWeight.NONE },
      { operation: "edit_instruct", weight: ConsentWeight.HIGH },
      { operation: "inpaint", weight: ConsentWeight.HIGH },
    ],
    ...overrides,
  });

describe("api/safety", () => {
  it("getPolicy proxies the SafetyService client", async () => {
    const policy = makePolicy();
    mocks.client.getPolicy.mockResolvedValueOnce(policy);

    const out = await getPolicy();

    expect(out).toBe(policy);
    expect(mocks.client.getPolicy).toHaveBeenCalledWith({});
  });

  it("highConsentOps returns only the HIGH-weight operations", () => {
    const ops = highConsentOps(makePolicy());
    expect(ops.has("edit_instruct")).toBe(true);
    expect(ops.has("inpaint")).toBe(true);
    expect(ops.has("text_to_image")).toBe(false);
  });

  it("highConsentOps is empty for a null policy", () => {
    expect(highConsentOps(null).size).toBe(0);
    expect(highConsentOps(undefined).size).toBe(0);
  });

  it("needsConsent is true for a high-weight op when the public policy requires consent", () => {
    expect(needsConsent(makePolicy(), "edit_instruct")).toBe(true);
  });

  it("needsConsent is false for a none-weight op even on the public tier", () => {
    expect(needsConsent(makePolicy(), "text_to_image")).toBe(false);
  });

  it("needsConsent is false on the local tier (requireConsent off) for any op", () => {
    const local = makePolicy({ tier: DeploymentTier.LOCAL, requireConsent: false });
    expect(needsConsent(local, "edit_instruct")).toBe(false);
  });

  it("needsConsent is false for a missing policy", () => {
    expect(needsConsent(null, "edit_instruct")).toBe(false);
  });
});
