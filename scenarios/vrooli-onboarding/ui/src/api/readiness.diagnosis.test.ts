import { beforeEach, expect, it, vi } from "vitest";

const rpc = vi.hoisted(() => ({ getReadiness: vi.fn() }));
vi.mock("@connectrpc/connect", () => ({ createClient: () => rpc }));
vi.mock("./base", () => ({ onboardingTransport: () => ({}) }));

import { fetchReadiness } from "./readiness";

describe("credential diagnosis projection", () => {
  beforeEach(() => vi.clearAllMocks());

  it("reads the JSON Struct root directly and preserves provider capabilities", async () => {
    rpc.getReadiness.mockResolvedValue({
      status: 1,
      scenarios: [], resources: [], credentials: [{
        resource: "mail", logicalId: "mail", field: "api_key", label: "Mail key",
        required: true, status: 1, legacyStatus: "configured", consumerRefs: ["Mail sender"],
        tiers: ["local"], provenance: [], evidenceStatus: "verified", evidenceDetail: "Owner exercised",
        evidenceNextAction: "", evidenceCredentialVersion: "credential-v2", providerState: "available", providerDetail: "Healthy",
      }], hosts: [], integrations: [],
      checkedAt: undefined,
      credentialDiagnosis: { provider: { platform: "linux", adapter: "native", backend: "native keyring", condition: "available", available: true, writable: true, explanation: "Owner is ready.", write_condition: "writes healthy" } },
      recovery: undefined,
      blockers: [], degraded: [], degradedDigest: "", degradedAcknowledged: false,
    });

    await expect(fetchReadiness("node-1")).resolves.toMatchObject({
      credential_diagnosis: {
        provider: { platform: "linux", adapter: "native", backend: "native keyring", condition: "available", available: true, writable: true, explanation: "Owner is ready.", write_condition: "writes healthy" },
      },
      credentials: [{ tiers: ["local"], evidence_status: "verified", evidence_detail: "Owner exercised", evidence_credential_version: "credential-v2", provider_state: "available", provider_detail: "Healthy" }],
    });
  });

  it("rejects malformed provider values instead of passing unknown fields into the UI", async () => {
    rpc.getReadiness.mockResolvedValue({
      status: 1,
      scenarios: [], resources: [], credentials: [], hosts: [], integrations: [],
      checkedAt: undefined,
      credentialDiagnosis: { provider: "unavailable" },
      recovery: undefined,
      blockers: [], degraded: [], degradedDigest: "", degradedAcknowledged: false,
    });

    await expect(fetchReadiness("local")).resolves.toMatchObject({ credential_diagnosis: undefined });
  });
});
