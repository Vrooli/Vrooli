export interface PermissionReadResult { revision: string; valueJson: Uint8Array }
export interface PermissionRegistryClient {
  listPermissions(request: { owner: string }): Promise<{ permissions: Array<{ id: string }> }>;
  readPermission(request: { permissionId: string; owner: string }): Promise<PermissionReadResult>;
  writePermission(request: { permissionId: string; owner: string; expectedRevision: string; idempotencyKey: string; valueJson: Uint8Array }): Promise<{ receipt?: unknown }>;
}

const updateControlId = "onboarding/install-posture/update-control";
const sessionModeId = "onboarding/install-posture/session-mode";
export type UpdateControlMode = "observe" | "guard";
export type SessionMode = "personal" | "shared" | "hosted";
export type TrustPosture = "personal" | "shared" | "hosted";
export interface InstallPostureSnapshot {
  updateControl: UpdateControlMode;
  trustPosture: TrustPosture;
  sessionMode?: SessionMode;
  updateControlRevision: string;
  sessionModeRevision?: string;
}

function parseValue<T extends string>(json: Uint8Array, supported: readonly T[], field: string): T {
  const parsed = JSON.parse(new TextDecoder().decode(json)) as { value?: unknown; supported_values?: unknown };
  if (typeof parsed.value !== "string" || !supported.includes(parsed.value as T)) throw new Error(`Owner returned an unsupported ${field} value`);
  if (!Array.isArray(parsed.supported_values) || parsed.supported_values.some((item) => typeof item !== "string" || !supported.includes(item as T))) {
    throw new Error(`Owner returned an invalid ${field} vocabulary`);
  }
  return parsed.value as T;
}

export async function readInstallPostureFromRegistry(client: PermissionRegistryClient): Promise<InstallPostureSnapshot> {
  const [update, trust, sessionPermissions] = await Promise.all([
    client.readPermission({ permissionId: updateControlId, owner: "vrooli-onboarding" }),
    client.readPermission({ permissionId: "onboarding/install-posture/trust-posture", owner: "vrooli-onboarding" }),
    client.listPermissions({ owner: "api-core/operatorsession" }),
  ]);
  const sessionPermission = sessionPermissions.permissions.find((item) => item.id === sessionModeId);
  const session = sessionPermission
    ? await client.readPermission({ permissionId: sessionModeId, owner: "api-core/operatorsession" })
    : undefined;
  return {
    updateControl: parseValue<UpdateControlMode>(update.valueJson, ["observe", "guard"], "update-control"),
    trustPosture: parseValue<TrustPosture>(trust.valueJson, ["personal", "shared", "hosted"], "trust-posture"),
    updateControlRevision: update.revision,
    ...(session ? {
      sessionMode: parseValue<SessionMode>(session.valueJson, ["personal", "shared", "hosted"], "session-mode"),
      sessionModeRevision: session.revision,
    } : {}),
  };
}

export async function writeUpdateControlThroughRegistry(client: PermissionRegistryClient, value: UpdateControlMode, expectedRevision: string, idempotencyKey: string) {
  if (!expectedRevision.trim() || !idempotencyKey.trim()) throw new Error("Update-control writes require a revision and idempotency key");
  const response = await client.writePermission({
    permissionId: updateControlId,
    owner: "vrooli-onboarding",
    expectedRevision,
    idempotencyKey,
    valueJson: new TextEncoder().encode(JSON.stringify({ value })),
  });
  if (!response.receipt) throw new Error("Update-control write returned no durable receipt");
  return response.receipt;
}
