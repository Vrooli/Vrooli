import { describe, expect, it, vi } from "vitest";
import { readStandingGrantProjection, type Descriptor, type PermissionRegistryReader } from "./standingGrantsContract";

const source = "agent-manager/owner-standing-grant-projection/v1";
const revision = "sha256:feedbeef";
const grantID = "agent-manager/standing-grant/grant-opaque";
const scopeID = "agent-manager/standing-grant/scope-opaque";
const grant = { id: "grant-opaque", subject: "worker", approver: "owner", operations: ["run.create"], targets: ["agent-manager:run/*"], state: "approved", max_concurrent_runs: 2, max_run_budget_usd: 5, approved_at: "2026-11-20T10:00:00Z", expires_at: "2026-12-20T10:00:00Z" };
const scope = { scenario: "agent-manager", scope: "agent-manager:read", effect: "read", run_eligible: true, permissions: ["run/read"], command: "permissions/list", service: "PermissionsService", method: "List" };

const descriptors: Descriptor[] = [
  { id: grantID, owner: "agent-manager", principalId: "alice", scope: "agent-manager:read", subjectKind: "standing-grant", subjectId: grant.id, source, schemaVersion: `agent-manager/standing-grant-metadata/v1#${revision}`, canRead: true, canWrite: false },
  { id: scopeID, owner: "agent-manager", principalId: "alice", scope: "agent-manager:read", subjectKind: "scope-declaration", subjectId: scope.scope, source, schemaVersion: `agent-manager/scope-catalog/v1#${revision}`, canRead: true, canWrite: false },
];

function registry(values: unknown[] = [grant, scope], inventory = descriptors, readRevision = revision): PermissionRegistryReader {
  return {
    listPermissions: vi.fn().mockResolvedValue({ permissions: inventory }),
    readPermission: vi.fn().mockImplementation(async ({ permissionId }) => ({
      permissionId, owner: "agent-manager", revision: readRevision,
      valueJson: new TextEncoder().encode(JSON.stringify(values[permissionId === grantID ? 0 : 1])),
    })),
  };
}

describe("Agent Manager permission UI contract", () => {
  it("keeps catalog declarations separate from metadata-only grant rows", async () => {
    await expect(readStandingGrantProjection(registry())).resolves.toEqual({
      grants: [{ permissionId: grantID, id: grant.id, subject: grant.subject, approver: grant.approver, operations: grant.operations, targets: grant.targets, state: "approved", maxConcurrentRuns: 2, maxRunBudgetUsd: 5, approvedAt: grant.approved_at, expiresAt: grant.expires_at, revokedAt: undefined }],
      scopes: [{ permissionId: scopeID, scope: scope.scope, effect: "read", runEligible: true, command: scope.command }],
    });
  });

  it("refuses writable, stale, unexpected and malformed owner projections", async () => {
    await expect(readStandingGrantProjection(registry([grant], [{ ...descriptors[0]!, canWrite: true }]))).rejects.toThrow(/unsupported or writable/);
    await expect(readStandingGrantProjection(registry([grant], descriptors, "stale"))).rejects.toThrow(/changed/);
    await expect(readStandingGrantProjection(registry([{ ...grant, private_note: "do not copy" }, scope]))).rejects.toThrow(/unsupported standing-grant/);
    await expect(readStandingGrantProjection(registry([{ ...grant, id: "different" }, scope]))).rejects.toThrow(/unsupported standing-grant/);
  });
});
