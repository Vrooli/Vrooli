import { describe, expect, it, vi } from "vitest";

const calls = vi.hoisted(() => ({
  list: vi.fn(),
  verify: vi.fn(),
}));

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:9999/api/v1",
  createScenarioConnectTransport: () => ({}),
}));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    listCapabilities: calls.list,
    getCapabilityStatus: vi.fn(),
    previewCapability: vi.fn(),
    applyCapability: vi.fn(),
    verifyCapability: calls.verify,
  }),
}));

import { CapabilityState, type ListCapabilitiesResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/capabilities/capabilities_pb";
import { fetchCapabilities, verifyCapability } from "./capabilities";

describe("capability client", () => {
  it("maps the canonical descriptor and evidence fields without a wire compatibility shim", async () => {
    calls.list.mockResolvedValueOnce({
      count: 1,
      capabilities: [{
        state: CapabilityState.NEEDS_OPERATOR_INPUT,
        candidates: [], missingInputs: ["destination"], evidence: [{
          kind: "filesystem", artifactIdentity: "disk-1", observedAt: undefined,
          verified: true, stage: "verified", owner: "backup", contextDigest: "ctx-v2",
          catalogRevision: "catalog-v3", configurationRevision: "config-v4", providerAdapterVersion: "adapter-v5",
        }],
        remediation: "choose a destination",
        descriptor: {
          version: "1", id: "backup", owner: "backup-provider", title: "Backup",
          description: "Prepare backups", risk: "low", prerequisites: [],
          inputs: [{ id: "destination", kind: "path", label: "Destination", description: "Folder", required: true,
            declinable: false, options: [], defaultValue: "", candidates: [], validation: "writable",
            constraints: { minLength: 1, maxLength: 120, minDuration: "", maxDuration: "" } }],
          policy: { requiresConfirmation: true, idempotent: true, retryable: true, protectedRoots: [], remediation: "review" },
          evidence: { kinds: ["filesystem"], stages: ["verified"], requiredFields: ["owner"], secretFree: true, freshness: "current" },
          remediation: "review", scope: "workspace", purpose: "backup", sensitivity: "low",
          disposition: "active", dispositionReason: "supported", referenceUrl: "https://example.invalid/backup",
          applicability: { platforms: ["linux"], environments: ["local"], targets: ["local"] },
          provenance: { requester: "operator", scope: "workspace", grantSource: "explicit", revocationLimit: "local" },
          lifecycle: { preview: true, apply: true, verify: true, revoke: false, recover: true, recovery: "retry" },
        },
      }],
    } as unknown as ListCapabilitiesResponse);

    await expect(fetchCapabilities("local")).resolves.toMatchObject({
      count: 1,
      capabilities: [{
        state: "needs_operator_input",
        descriptor: {
          scope: "workspace",
          disposition: "active",
          reference_url: "https://example.invalid/backup",
          inputs: [{ declinable: false, constraints: { min_length: 1, max_length: 120 } }],
          evidence: { stages: ["verified"] },
          lifecycle: { preview: true, apply: true, recover: true },
        },
        evidence: [{ stage: "verified", owner: "backup", context_digest: "ctx-v2", catalog_revision: "catalog-v3", configuration_revision: "config-v4", provider_adapter_version: "adapter-v5" }],
      }],
    });
  });

  it("sends a fully typed read-only verification request for the selected target", async () => {
    calls.verify.mockResolvedValueOnce({ capabilityId: "backup", outcome: "verified", evidence: [], retryable: true });

    await expect(verifyCapability("backup", "node-1")).resolves.toMatchObject({
      capability_id: "backup", outcome: "verified", evidence: [],
    });
    expect(calls.verify).toHaveBeenCalledWith({
      target: "node-1",
      verification: {
        capabilityId: "backup", targetId: "node-1", environment: "", accountIdentity: "",
        operation: "readiness-check", context: {}, effectClass: "read_only", maxOperations: 0,
        cleanupPolicy: "", timeoutSeconds: 30n,
      },
    });
  });
});
