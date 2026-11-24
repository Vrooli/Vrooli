export interface CredentialGrantMetadata {
  permissionId: string;
  id: string;
  logicalId: string;
  field: string;
  grantClass: string;
  retention: string;
  generation: number;
  ackedGeneration: number;
  status: "awaiting_receipt" | "delivered" | "revoked";
  grantedAt: string;
  revokedAt: string;
  receiptAt: string;
  receiptAccepted: boolean;
  purgeState: string;
  purgeReceiptAt: string;
  purgeAccepted: boolean;
  machineId: string;
  nodeId: string;
  enrollmentGeneration: number;
}

interface Target {
  machineId: string;
  nodeId: string;
  enrollmentGeneration: number | bigint;
}

interface Descriptor {
  id: string; owner: string; principalId: string; scope: string; subjectKind: string; subjectId: string;
  targetBound: boolean; target?: Target; source: string; schemaVersion: string; canRead: boolean; canWrite: boolean;
}

export interface CredentialGrantRegistryReader {
  listPermissions(request: { owner: string }): Promise<{ permissions: Descriptor[] }>;
  readPermission(request: { permissionId: string; owner: string; target: Target }): Promise<{
    permissionId: string; owner: string; target?: Target; revision: string; valueJson: Uint8Array;
  }>;
}

interface GrantJSON {
  id: string; logical_id: string; field: string; class: string; retention: string;
  generation: number; acked_generation: number; status: string; granted_at?: string; revoked_at?: string;
  receipt_at?: string; receipt_accepted: boolean; purge_state?: string; purge_receipt_at?: string; purge_accepted: boolean;
}

const owner = "vrooli-bridge";
const prefix = "bridge/credential-grant/";
const scope = "vrooli-bridge:read";
const subjectKind = "sealed-credential-grant";
const source = "Bridge side-effect-free credential grant metadata API";
const schemaVersion = "bridge/credential-grant-metadata/v1";
const allowedFields = new Set([
  "id", "logical_id", "field", "class", "retention", "generation", "acked_generation", "status", "granted_at",
  "revoked_at", "receipt_at", "receipt_accepted", "purge_state", "purge_receipt_at", "purge_accepted",
]);

export async function listCredentialGrantMetadata(registry: CredentialGrantRegistryReader): Promise<CredentialGrantMetadata[]> {
  const inventory = await registry.listPermissions({ owner });
  const descriptors = inventory.permissions.filter((item) => item.owner === owner && item.id.startsWith(prefix));
  const result: CredentialGrantMetadata[] = [];
  for (const descriptor of descriptors) {
    const target = descriptor.target;
    const generation = target ? safeGeneration(target.enrollmentGeneration) : null;
    if (!descriptor.canRead || descriptor.canWrite || descriptor.scope !== scope || descriptor.subjectKind !== subjectKind ||
      descriptor.source !== source || !descriptor.schemaVersion.startsWith(`${schemaVersion}#`) || !descriptor.principalId ||
      descriptor.id !== `${prefix}${descriptor.subjectId}` || !descriptor.targetBound || !target ||
      !target.machineId || !target.nodeId || generation === null || generation <= 0) {
      throw new Error("Bridge credential grant descriptor is unsupported or writable");
    }
    const read = await registry.readPermission({ permissionId: descriptor.id, owner, target });
    const expectedRevision = descriptor.schemaVersion.slice(schemaVersion.length + 1);
    if (read.permissionId !== descriptor.id || read.owner !== owner || !read.revision || read.revision !== expectedRevision ||
      !read.target || read.target.machineId !== target.machineId || read.target.nodeId !== target.nodeId ||
      safeGeneration(read.target.enrollmentGeneration) !== generation) {
      throw new Error("Bridge credential grant target or source changed; refresh the permission inventory");
    }
    const grant = JSON.parse(new TextDecoder().decode(read.valueJson)) as GrantJSON;
    if (Object.keys(grant).some((key) => !allowedFields.has(key)) || grant.id !== descriptor.subjectId || !grant.logical_id ||
      !grant.field || !["infrastructure", "per_install_generated", "user_prompt", "remote_fetch"].includes(grant.class) ||
      !["durable", "ephemeral"].includes(grant.retention) || !Number.isInteger(grant.generation) || grant.generation < 1 ||
      !Number.isInteger(grant.acked_generation) || grant.acked_generation < 0 || grant.acked_generation > grant.generation ||
      !["awaiting_receipt", "delivered", "revoked"].includes(grant.status) || typeof grant.receipt_accepted !== "boolean" ||
      typeof grant.purge_accepted !== "boolean" || !validOptionalDate(grant.granted_at) || !validOptionalDate(grant.revoked_at) ||
      !validOptionalDate(grant.receipt_at) || !validOptionalDate(grant.purge_receipt_at)) {
      throw new Error("Bridge returned unsupported credential grant metadata");
    }
    result.push({
      permissionId: descriptor.id, id: grant.id, logicalId: grant.logical_id, field: grant.field, grantClass: grant.class,
      retention: grant.retention, generation: grant.generation, ackedGeneration: grant.acked_generation,
      status: grant.status as CredentialGrantMetadata["status"], grantedAt: grant.granted_at ?? "", revokedAt: grant.revoked_at ?? "",
      receiptAt: grant.receipt_at ?? "", receiptAccepted: grant.receipt_accepted, purgeState: grant.purge_state ?? "",
      purgeReceiptAt: grant.purge_receipt_at ?? "", purgeAccepted: grant.purge_accepted,
      machineId: target.machineId, nodeId: target.nodeId, enrollmentGeneration: generation,
    });
  }
  return result;
}

function safeGeneration(value: number | bigint): number | null {
  const converted = typeof value === "bigint" ? Number(value) : value;
  return Number.isSafeInteger(converted) ? converted : null;
}

function validOptionalDate(value: string | undefined): boolean {
  return value === undefined || (Boolean(value) && Number.isFinite(Date.parse(value)));
}
