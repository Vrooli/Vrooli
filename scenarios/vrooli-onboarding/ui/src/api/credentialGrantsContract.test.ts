import { describe, expect, it, vi } from "vitest";
import { listCredentialGrantMetadata, type CredentialGrantRegistryReader } from "./credentialGrantsContract";

const descriptor = {
  id: "bridge/credential-grant/grant-1", owner: "vrooli-bridge", principalId: "operator-a", scope: "vrooli-bridge:read",
  subjectKind: "sealed-credential-grant", subjectId: "grant-1", targetBound: true,
  target: { machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 4n },
  source: "Bridge side-effect-free credential grant metadata API", schemaVersion: "bridge/credential-grant-metadata/v1#3:3:true:active", canRead: true, canWrite: false,
};
const value = {
  id: "grant-1", logical_id: "service/api", field: "token", class: "user_prompt", retention: "ephemeral",
  generation: 3, acked_generation: 3, status: "delivered", receipt_accepted: true, purge_accepted: false,
  receipt_at: "2026-11-20T08:00:00Z",
};

function reader(overrides: { descriptor?: object; target?: object; value?: object; revision?: string } = {}): CredentialGrantRegistryReader {
  return {
    listPermissions: vi.fn().mockResolvedValue({ permissions: [overrides.descriptor ?? descriptor] }),
    readPermission: vi.fn().mockResolvedValue({
      permissionId: descriptor.id, owner: "vrooli-bridge", target: overrides.target ?? descriptor.target, revision: overrides.revision ?? "3:3:true:active",
      valueJson: new TextEncoder().encode(JSON.stringify(overrides.value ?? value)),
    }),
  };
}

describe("sealed credential grant permission contract", () => {
  it("reads only allowlisted metadata bound to the registry target", async () => {
    const client = reader();
    await expect(listCredentialGrantMetadata(client)).resolves.toEqual([expect.objectContaining({
      id: "grant-1", status: "delivered", machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 4,
    })]);
    expect(client.readPermission).toHaveBeenCalledWith({ permissionId: descriptor.id, owner: "vrooli-bridge", target: descriptor.target });
  });

  it("keeps write-only action descriptors outside the read-only credential panel", async () => {
    const client = reader();
    vi.mocked(client.listPermissions).mockResolvedValue({ permissions: [descriptor, {
      ...descriptor,
      id: "bridge/credential-grant-action/revoke/Z3JhbnQtMQ",
      scope: "vrooli-bridge:write",
      subjectKind: "credential-grant-revoke",
      source: "Bridge durable credential grant mutation receipt API",
      schemaVersion: "bridge/credential-grant-mutation/v1/revoke#source=4;expected=3",
      canRead: false,
      canWrite: true,
    }] });
    await expect(listCredentialGrantMetadata(client)).resolves.toHaveLength(1);
    expect(client.readPermission).toHaveBeenCalledTimes(1);
  });

  it("rejects writable descriptors, target drift, and secret-bearing values", async () => {
    await expect(listCredentialGrantMetadata(reader({ descriptor: { ...descriptor, canWrite: true } }))).rejects.toThrow(/unsupported or writable/);
    await expect(listCredentialGrantMetadata(reader({ target: { ...descriptor.target, enrollmentGeneration: 5n } }))).rejects.toThrow(/target or source changed/);
    await expect(listCredentialGrantMetadata(reader({ revision: "4:4:true:active" }))).rejects.toThrow(/target or source changed/);
    await expect(listCredentialGrantMetadata(reader({ value: { ...value, secret_value: "private" } }))).rejects.toThrow(/unsupported credential grant metadata/);
  });
});
