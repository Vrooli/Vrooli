import { describe, expect, it, vi } from "vitest";
import { readApprovedDependencySnapshot, type PermissionRegistryReader } from "./approvedDependenciesContract";

const value = {
  principal: "operator-1", source: ".vrooli/dependencies/approved-dependencies.json", source_revision: "abc123",
  source_digest: "sha256:0123456789abcdef", schema_version: "1", version: "2026-11-20", mode: "reviewed",
  guidance: "Use the dependency review workflow.",
  records: [{ ecosystem: "npm", package_name: "example", version_range: "^1.2.0", state: "approved", rationale: "Used by the app" }],
};

function registry(snapshot: unknown = value, revision = "approved-dependencies/v1#sha256:0123456789abcdef#abc123", canWrite = false, readRevision = revision): PermissionRegistryReader {
  return {
    listPermissions: vi.fn().mockResolvedValue({ permissions: [{ id: "onboarding/approved-dependencies", owner: "scenario-dependency-analyzer", schemaVersion: revision, canRead: true, canWrite }] }),
    readPermission: vi.fn().mockResolvedValue({ revision: readRevision, valueJson: new TextEncoder().encode(JSON.stringify(snapshot)) }),
  };
}

describe("approved dependency registry contract", () => {
  it("reads owner-bound metadata with the inventory revision", async () => {
    const client = registry();
    await expect(readApprovedDependencySnapshot(client)).resolves.toMatchObject({ source_digest: value.source_digest, records: value.records });
    expect(client.listPermissions).toHaveBeenCalledWith({ owner: "scenario-dependency-analyzer" });
    expect(client.readPermission).toHaveBeenCalledWith({ permissionId: "onboarding/approved-dependencies", owner: "scenario-dependency-analyzer" });
  });

  it("refuses writable descriptors and stale or incomplete owner data", async () => {
    await expect(readApprovedDependencySnapshot(registry(value, undefined, true))).rejects.toThrow(/read-only/);
    await expect(readApprovedDependencySnapshot(registry(value, "expected-revision", false, "old-revision"))).rejects.toThrow(/changed/);
    await expect(readApprovedDependencySnapshot(registry({ ...value, records: [{ ecosystem: "npm" }] }))).rejects.toThrow(/incomplete record/);
  });
});
