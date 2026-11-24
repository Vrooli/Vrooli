export interface CredentialGrantTarget {
  machineId: string;
  nodeId: string;
  enrollmentGeneration: number | bigint;
}

interface Descriptor {
  id: string;
  owner: string;
  principalId: string;
  scope: string;
  subjectKind: string;
  subjectId: string;
  targetBound: boolean;
  target?: CredentialGrantTarget;
  source: string;
  schemaVersion: string;
  canRead: boolean;
  canWrite: boolean;
}

export interface CredentialGrantAction {
  permissionId: string;
  principalId: string;
  operation: "create" | "revoke";
  subjectId: string;
  expectedRevision: string;
  target: { machineId: string; nodeId: string; enrollmentGeneration: number };
}

export interface CredentialGrantActionReceipt {
  id: string;
  permissionId: string;
  owner: string;
  actorId: string;
  target?: CredentialGrantTarget;
  expectedRevision: string;
  resultRevision: string;
  idempotencyKey: string;
  outcome: string;
  ownerState?: string;
}

export interface CredentialGrantActionRegistry {
  listPermissions(request: { owner: string }): Promise<{ permissions: Descriptor[] }>;
  writePermission(request: {
    permissionId: string;
    owner: string;
    target: CredentialGrantTarget;
    expectedRevision: string;
    idempotencyKey: string;
    valueJson: Uint8Array;
  }): Promise<{ receipt?: CredentialGrantActionReceipt }>;
}

const owner = "vrooli-bridge";
const actionPrefix = "bridge/credential-grant-action/";
const actionSource = "Bridge durable credential grant mutation receipt API";
const actionSchema = "bridge/credential-grant-mutation/v1";
const allowedClasses = new Set(["infrastructure", "per_install_generated", "user_prompt", "remote_fetch"]);
const allowedRetentions = new Set(["durable", "ephemeral"]);
const allowedOwnerStates = new Set(["pending", "dispatched", "retryable", "confirmed", "refused"]);

export async function listCredentialGrantActions(registry: CredentialGrantActionRegistry): Promise<CredentialGrantAction[]> {
  const inventory = await registry.listPermissions({ owner });
  const actions: CredentialGrantAction[] = [];
  for (const descriptor of inventory.permissions.filter((item) => item.id.startsWith(actionPrefix))) {
    const action = parseActionDescriptor(descriptor);
    actions.push(action);
  }
  return actions;
}

export async function writeCredentialGrantAction(
  registry: CredentialGrantActionRegistry,
  action: CredentialGrantAction,
  idempotencyKey: string,
  value: Record<string, unknown>,
): Promise<CredentialGrantActionReceipt> {
  if (!action || !action.permissionId.startsWith(actionPrefix) || !["create", "revoke"].includes(action.operation) ||
    !action.expectedRevision || !action.target.machineId || !action.target.nodeId || safeGeneration(action.target.enrollmentGeneration) === null ||
    !idempotencyKey.trim() || idempotencyKey !== idempotencyKey.trim()) {
    throw new Error("credential grant action request is invalid");
  }
  const normalized = normalizePayload(action.operation, value);
  const response = await registry.writePermission({
    permissionId: action.permissionId,
    owner,
    target: action.target,
    expectedRevision: action.expectedRevision,
    idempotencyKey,
    valueJson: new TextEncoder().encode(JSON.stringify(normalized)),
  });
  const receipt = response.receipt;
  if (!receipt || receipt.permissionId !== action.permissionId || receipt.owner !== owner || receipt.actorId !== action.principalId ||
    receipt.expectedRevision !== action.expectedRevision || receipt.resultRevision === "" || receipt.idempotencyKey !== idempotencyKey ||
    !receipt.id || receipt.outcome !== (action.operation === "create" ? "created" : "revoked") || !receipt.target ||
    !sameTarget(receipt.target, action.target) || !receipt.ownerState || !allowedOwnerStates.has(receipt.ownerState)) {
    throw new Error("Bridge returned a credential grant receipt with mismatched action binding");
  }
  return receipt;
}

function parseActionDescriptor(descriptor: Descriptor): CredentialGrantAction {
  const target = descriptor.target;
  const generation = target ? safeGeneration(target.enrollmentGeneration) : null;
  if (descriptor.owner !== owner || !descriptor.principalId || descriptor.canRead || !descriptor.canWrite ||
    descriptor.scope !== "vrooli-bridge:write" || descriptor.source !== actionSource || !descriptor.targetBound ||
    !target || !target.machineId || !target.nodeId || generation === null || generation <= 0) {
    throw new Error("Bridge credential action descriptor is unsupported or readable");
  }
  const suffix = descriptor.id.slice(actionPrefix.length);
  const slash = suffix.indexOf("/");
  if (slash <= 0 || slash === suffix.length - 1) throw new Error("Bridge credential action descriptor ID is malformed");
  const operation = suffix.slice(0, slash);
  const encodedSubject = suffix.slice(slash + 1);
  let expectedRevision: string;
  if (operation === "create") {
    if (descriptor.subjectKind !== "credential-grant-create" || descriptor.subjectId !== encodeTargetKey(target.machineId, target.nodeId, generation) ||
      encodedSubject !== descriptor.subjectId || descriptor.schemaVersion !== `${actionSchema}/create#0`) {
      throw new Error("Bridge credential create action is not bound to its target and source revision");
    }
    expectedRevision = "0";
  } else if (operation === "revoke") {
    const grantId = decodeBase64URL(encodedSubject);
    const match = descriptor.schemaVersion.match(/^bridge\/credential-grant-mutation\/v1\/revoke#source=([1-9]\d*);expected=([1-9]\d*)$/);
    const sourceText = match?.[1];
    const expectedText = match?.[2];
    if (descriptor.subjectKind !== "credential-grant-revoke" || grantId !== descriptor.subjectId || !sourceText || !expectedText) {
      throw new Error("Bridge credential revoke action has an invalid grant or source revision binding");
    }
    const sourceRevision = BigInt(sourceText);
    const expected = BigInt(expectedText);
    if (sourceRevision > 9223372036854775807n || expected > 9223372036854775807n ||
      (sourceRevision !== expected && sourceRevision !== expected + 1n)) {
      throw new Error("Bridge credential revoke action has an invalid grant or source revision binding");
    }
    expectedRevision = expected.toString();
  } else {
    throw new Error("Bridge credential action operation is unsupported");
  }
  return {
    permissionId: descriptor.id,
    principalId: descriptor.principalId,
    operation,
    subjectId: descriptor.subjectId,
    expectedRevision,
    target: { machineId: target.machineId, nodeId: target.nodeId, enrollmentGeneration: generation },
  };
}

function normalizePayload(operation: "create" | "revoke", value: Record<string, unknown>): Record<string, unknown> {
  const allowed = operation === "create"
    ? ["operation", "logical_id", "field", "class", "retention", "generation"]
    : ["operation"];
  if (Object.keys(value).some((key) => !allowed.includes(key)) || value.operation !== operation) {
    throw new Error("credential grant action payload contains unsupported fields");
  }
  if (operation === "revoke") return { operation };
  if (typeof value.logical_id !== "string" || !value.logical_id.trim() || value.logical_id !== value.logical_id.trim() ||
    typeof value.field !== "string" || !value.field.trim() || value.field !== value.field.trim() ||
    typeof value.class !== "string" || !allowedClasses.has(value.class) ||
    typeof value.retention !== "string" || !allowedRetentions.has(value.retention) ||
    typeof value.generation !== "number" || !Number.isSafeInteger(value.generation) || value.generation < 1) {
    throw new Error("credential grant create payload is invalid");
  }
  return { operation, logical_id: value.logical_id, field: value.field, class: value.class, retention: value.retention, generation: value.generation };
}

function safeGeneration(value: number | bigint): number | null {
  const converted = typeof value === "bigint" ? Number(value) : value;
  return Number.isSafeInteger(converted) ? converted : null;
}

function sameTarget(left: CredentialGrantTarget, right: CredentialGrantTarget): boolean {
  return left.machineId === right.machineId && left.nodeId === right.nodeId && safeGeneration(left.enrollmentGeneration) === safeGeneration(right.enrollmentGeneration);
}

function encodeTargetKey(machineId: string, nodeId: string, generation: number): string {
  return encodeBase64URL(`${machineId}\0${nodeId}\0${generation}`);
}

function encodeBase64URL(value: string): string {
  const bytes = new TextEncoder().encode(value);
  let binary = "";
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
}

function decodeBase64URL(value: string): string {
  try {
    const base64 = value.replaceAll("-", "+").replaceAll("_", "/");
    const binary = atob(base64 + "=".repeat((4 - base64.length % 4) % 4));
    return new TextDecoder("utf-8", { fatal: true }).decode(Uint8Array.from(binary, (byte) => byte.charCodeAt(0)));
  } catch {
    return "";
  }
}
