import { describe, expect, it, vi } from "vitest";
import {
  listCredentialGrantActions,
  writeCredentialGrantAction,
  type CredentialGrantActionRegistry,
} from "./credentialGrantActionsContract";

const target = { machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 7n };
const targetKey = base64URL(`machine-1\0node-1\0${7}`);
const create = {
  id: `bridge/credential-grant-action/create/${targetKey}`, owner: "vrooli-bridge", principalId: "operator-a",
  scope: "vrooli-bridge:write", subjectKind: "credential-grant-create", subjectId: targetKey, targetBound: true,
  target, source: "Bridge durable credential grant mutation receipt API", schemaVersion: "bridge/credential-grant-mutation/v1/create#0", canRead: false, canWrite: true,
};
const revoke = {
  id: "bridge/credential-grant-action/revoke/Z3JhbnQtMQ", owner: "vrooli-bridge", principalId: "operator-a",
  scope: "vrooli-bridge:write", subjectKind: "credential-grant-revoke", subjectId: "grant-1", targetBound: true,
  target, source: "Bridge durable credential grant mutation receipt API", schemaVersion: "bridge/credential-grant-mutation/v1/revoke#source=4;expected=3", canRead: false, canWrite: true,
};

function registry(overrides: { permissions?: object[]; receipt?: object } = {}): CredentialGrantActionRegistry {
  return {
    listPermissions: vi.fn().mockResolvedValue({ permissions: overrides.permissions ?? [create, revoke] }),
    writePermission: vi.fn().mockResolvedValue({ receipt: overrides.receipt ?? {
      id: "receipt-1", permissionId: revoke.id, owner: "vrooli-bridge", actorId: "operator-a", target,
      expectedRevision: "3", resultRevision: "4", idempotencyKey: "request-1", outcome: "revoked", ownerState: "pending",
    } }),
  };
}

describe("write-only credential grant action contract", () => {
  it("lists create and revoke actions without reading metadata", async () => {
    const client = registry();
    await expect(listCredentialGrantActions(client)).resolves.toEqual([
      { permissionId: create.id, principalId: "operator-a", operation: "create", subjectId: targetKey, expectedRevision: "0", target: { machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 7 } },
      { permissionId: revoke.id, principalId: "operator-a", operation: "revoke", subjectId: "grant-1", expectedRevision: "3", target: { machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 7 } },
    ]);
    expect(client.listPermissions).toHaveBeenCalledWith({ owner: "vrooli-bridge" });
  });

  it("refuses readable, cross-owner, stale-schema, and malformed action descriptors", async () => {
    for (const invalid of [
      { ...create, canRead: true },
      { ...create, owner: "another-owner" },
      { ...create, target: { ...target, enrollmentGeneration: 8n } },
      { ...revoke, schemaVersion: "bridge/credential-grant-mutation/v1/revoke#source=4;expected=2" },
      { ...revoke, id: "bridge/credential-grant-action/revoke/not-base64!" },
    ]) {
      await expect(listCredentialGrantActions(registry({ permissions: [invalid] }))).rejects.toThrow();
    }
  });

  it("writes only the action payload and preserves target-bound pending receipt state", async () => {
    const client = registry({ receipt: {
      id: "receipt-1", permissionId: create.id, owner: "vrooli-bridge", actorId: "operator-a", target,
      expectedRevision: "0", resultRevision: "1", idempotencyKey: "request-1", outcome: "created", ownerState: "pending",
    } });
    const [action] = await listCredentialGrantActions(client);
    if (!action) throw new Error("create action missing from fixture");
    const receipt = await writeCredentialGrantAction(client, action, "request-1", { operation: "create", logical_id: "service/api", field: "token", class: "user_prompt", retention: "ephemeral", generation: 1 });
    expect(client.writePermission).toHaveBeenCalledWith(expect.objectContaining({
      permissionId: create.id, owner: "vrooli-bridge", target: { machineId: "machine-1", nodeId: "node-1", enrollmentGeneration: 7 },
      expectedRevision: "0", idempotencyKey: "request-1",
    }));
    const sent = vi.mocked(client.writePermission).mock.calls[0]?.[0];
    expect(JSON.parse(new TextDecoder().decode(sent?.valueJson))).toEqual({ operation: "create", logical_id: "service/api", field: "token", class: "user_prompt", retention: "ephemeral", generation: 1 });
    expect(receipt.ownerState).toBe("pending");
  });

  it("rejects secret, unknown, and owner-state-less results", async () => {
    const client = registry();
    const [createAction] = await listCredentialGrantActions(client);
    if (!createAction) throw new Error("create action missing from fixture");
    await expect(writeCredentialGrantAction(client, createAction, "request-1", { operation: "create", logical_id: "service/api", field: "token", class: "user_prompt", retention: "ephemeral", generation: 1, secret_value: "never" })).rejects.toThrow(/unsupported fields/);
    expect(client.writePermission).not.toHaveBeenCalled();
    client.writePermission = vi.fn().mockResolvedValue({ receipt: { id: "receipt-1", permissionId: create.id, owner: "vrooli-bridge", actorId: "operator-a", target, expectedRevision: "0", resultRevision: "1", idempotencyKey: "request-1", outcome: "created" } });
    await expect(writeCredentialGrantAction(client, createAction, "request-1", { operation: "create", logical_id: "service/api", field: "token", class: "user_prompt", retention: "ephemeral", generation: 1 })).rejects.toThrow(/mismatched action binding/);
  });

  it("rejects a receipt attributed to a different principal than the descriptor", async () => {
    const client = registry({ receipt: {
      id: "receipt-1", permissionId: create.id, owner: "vrooli-bridge", actorId: "operator-b", target,
      expectedRevision: "0", resultRevision: "1", idempotencyKey: "request-1", outcome: "created", ownerState: "pending",
    } });
    const [action] = await listCredentialGrantActions(client);
    if (!action) throw new Error("create action missing from fixture");
    await expect(writeCredentialGrantAction(client, action, "request-1", { operation: "create", logical_id: "service/api", field: "token", class: "user_prompt", retention: "ephemeral", generation: 1 })).rejects.toThrow(/mismatched action binding/);
  });
});

function base64URL(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}
