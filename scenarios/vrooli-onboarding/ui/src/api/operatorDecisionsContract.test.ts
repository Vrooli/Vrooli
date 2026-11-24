import { describe, expect, it, vi } from "vitest";
import { listPendingOperatorDecisions, type DecisionPermissionRegistry } from "./operatorDecisionsContract";

const id = "onboarding/operator-decision/opaque-ref";
const revision = "3";
const schemaVersion = `notification-hub/pending-decision-reference/v1#${revision}`;
const descriptor = {
  id, owner: "notification-hub", principalId: "alice", scope: "notification-hub:pending-decisions:read",
  subjectKind: "pending-operator-decision", subjectId: "ask-1", source: "notification-hub/owner-pending-decision-projection/v1",
  schemaVersion, canRead: true, canWrite: false,
};
const reference = {
  decision_ref: "ask-1", owner_id: "alice", source: "treasury", source_event_type: "treasury.refund.decision_requested.v1",
  revision, state: "pending", created_at: "2026-11-20T10:00:00Z", updated_at: "2026-11-20T10:01:00Z",
  deadline: "2026-11-22T10:00:00Z", urgency: "high",
};

function registry(value: unknown = reference, inventoryDescriptor = descriptor, readRevision = revision): DecisionPermissionRegistry {
  return {
    listPermissions: vi.fn().mockResolvedValue({ permissions: [inventoryDescriptor] }),
    readPermission: vi.fn().mockResolvedValue({ permissionId: id, owner: "notification-hub", revision: readRevision, valueJson: new TextEncoder().encode(JSON.stringify(value)) }),
  };
}

describe("pending decision permission UI contract", () => {
  it("reads only principal-bound pending reference metadata through the registry", async () => {
    const client = registry();
    await expect(listPendingOperatorDecisions(client)).resolves.toEqual([{
      permissionId: id, revision, decisionRef: "ask-1", ownerId: "alice", source: "treasury",
      sourceEventType: "treasury.refund.decision_requested.v1", state: "pending", createdAt: reference.created_at,
      updatedAt: reference.updated_at, deadline: reference.deadline, urgency: "high",
    }]);
    expect(client.listPermissions).toHaveBeenCalledWith({ owner: "notification-hub" });
    expect(client.readPermission).toHaveBeenCalledWith({ permissionId: id, owner: "notification-hub" });
  });

  it("refuses writable, stale, cross-principal and private-data projections", async () => {
    await expect(listPendingOperatorDecisions(registry(reference, { ...descriptor, canWrite: true }))).rejects.toThrow(/unsupported or writable/);
    await expect(listPendingOperatorDecisions(registry(reference, descriptor, "2"))).rejects.toThrow(/changed/);
    await expect(listPendingOperatorDecisions(registry(reference, { ...descriptor, principalId: "bob" }))).rejects.toThrow(/unsupported pending-decision/);
    await expect(listPendingOperatorDecisions(registry({ ...reference, question: "private" }))).rejects.toThrow(/unsupported pending-decision/);
    await expect(listPendingOperatorDecisions(registry({ ...reference, state: "answered" }))).rejects.toThrow(/unsupported pending-decision/);
  });
});
