export interface PendingDecisionReference {
  permissionId: string;
  revision: string;
  decisionRef: string;
  ownerId: string;
  source: string;
  sourceEventType: string;
  state: "pending";
  createdAt: string;
  updatedAt: string;
  deadline: string;
  urgency: string;
}

export interface DecisionPermissionDescriptor {
  id: string;
  owner: string;
  principalId: string;
  scope: string;
  subjectKind: string;
  subjectId: string;
  source: string;
  schemaVersion: string;
  canRead: boolean;
  canWrite: boolean;
}

export interface DecisionPermissionRegistry {
  listPermissions(request: { owner: string }): Promise<{ permissions: DecisionPermissionDescriptor[] }>;
  readPermission(request: { permissionId: string; owner: string }): Promise<{ permissionId: string; owner: string; revision: string; valueJson: Uint8Array }>;
}

interface PendingDecisionReferenceJSON {
  decision_ref: string;
  owner_id: string;
  source: string;
  source_event_type: string;
  revision: string;
  state: string;
  created_at: string;
  updated_at: string;
  deadline?: string;
  urgency?: string;
}

const permissionPrefix = "onboarding/operator-decision/";
const owner = "notification-hub";
const schemaVersion = "notification-hub/pending-decision-reference/v1#";
const scope = "notification-hub:pending-decisions:read";
const source = "notification-hub/owner-pending-decision-projection/v1";
const subjectKind = "pending-operator-decision";
const allowedFields = new Set(["decision_ref", "owner_id", "source", "source_event_type", "revision", "state", "created_at", "updated_at", "deadline", "urgency"]);

export async function listPendingOperatorDecisions(registry: DecisionPermissionRegistry): Promise<PendingDecisionReference[]> {
  const inventory = await registry.listPermissions({ owner });
  const matches = inventory.permissions.filter((item) => item.owner === owner && item.id.startsWith(permissionPrefix));
  const decisions: PendingDecisionReference[] = [];
  for (let start = 0; start < matches.length; start += 20) {
    const batch = matches.slice(start, start + 20);
    decisions.push(...await Promise.all(batch.map(async (item) => {
      if (!item.canRead || item.canWrite || item.scope !== scope || item.subjectKind !== subjectKind || item.source !== source || !item.schemaVersion.startsWith(schemaVersion) || !item.subjectId) {
        throw new Error("Notification Hub decision descriptor is unsupported or writable");
      }
      const result = await registry.readPermission({ permissionId: item.id, owner });
      if (result.permissionId !== item.id || result.owner !== owner || result.revision !== item.schemaVersion.slice(schemaVersion.length)) {
        throw new Error("Notification Hub decision changed; refresh the permission inventory");
      }
      const parsed = JSON.parse(new TextDecoder().decode(result.valueJson)) as PendingDecisionReferenceJSON;
      if (Object.keys(parsed).some((key) => !allowedFields.has(key)) || parsed.decision_ref !== item.subjectId || !parsed.owner_id || item.principalId !== parsed.owner_id || !parsed.source || !parsed.source_event_type.endsWith(".decision_requested.v1") || parsed.revision !== result.revision || parsed.state !== "pending" || !parsed.created_at || !parsed.updated_at || (parsed.urgency && !["low", "normal", "high", "critical"].includes(parsed.urgency))) {
        throw new Error("Notification Hub returned an unsupported pending-decision projection");
      }
      return {
        permissionId: item.id,
        revision: parsed.revision,
        decisionRef: parsed.decision_ref,
        ownerId: parsed.owner_id,
        source: parsed.source,
        sourceEventType: parsed.source_event_type,
        state: "pending" as const,
        createdAt: parsed.created_at,
        updatedAt: parsed.updated_at,
        deadline: parsed.deadline ?? "",
        urgency: parsed.urgency ?? "",
      };
    })));
  }
  return decisions;
}
