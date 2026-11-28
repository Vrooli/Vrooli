import { permissionRegistryClient as client } from "./permissionRegistry";

const permissionId = "onboarding/core-authority";
const owner = "vrooli-onboarding";

export interface CoreAuthority {
  seed: string[];
  trusted_base: string[];
  important?: { kind: "scenario" | "resource"; name: string }[];
  members?: { kind: "scenario" | "resource"; name: string }[];
}

// The receipt contains only the permission registry's durable mutation
// identity and revisions. Authority payloads and credentials are never echoed.
export interface CoreAuthorityReceipt {
  id: string;
  expectedRevision: string;
  resultRevision: string;
  idempotencyKey: string;
  outcome: string;
}

export async function readCoreAuthority(target = "local") {
  if (target !== "local") throw new Error("Core authority can only be changed on the local OperatorState target");
  const result = await client.readPermission({ permissionId, owner });
  let authority: CoreAuthority = { seed: [], trusted_base: [], important: [] };
  const valueJson = result.valueJson ?? new Uint8Array();
  if (valueJson.length > 0) {
    const decoded = JSON.parse(new TextDecoder().decode(valueJson)) as Partial<CoreAuthority>;
    authority = {
      seed: Array.isArray(decoded.seed) ? decoded.seed : [],
      trusted_base: Array.isArray(decoded.trusted_base) ? decoded.trusted_base : [],
      important: Array.isArray(decoded.important) ? decoded.important : [],
      members: Array.isArray(decoded.members) ? decoded.members : [],
    };
  }
  return { revision: result.revision, authority };
}

export async function writeCoreAuthority(
  authority: CoreAuthority,
  expectedRevision: string,
  idempotencyKey: string,
  target = "local",
): Promise<CoreAuthorityReceipt> {
  if (target !== "local") throw new Error("Core authority can only be changed on the local OperatorState target");
  if (!expectedRevision.trim() || !idempotencyKey.trim()) throw new Error("Core authority writes require a revision and idempotency key");
  const response = await client.writePermission({
    permissionId,
    owner,
    expectedRevision,
    idempotencyKey,
    valueJson: new TextEncoder().encode(JSON.stringify({ ...authority, important: authority.important ?? [], members: authority.members ?? [] })),
  });
  const receipt = response.receipt;
  if (!receipt) throw new Error("Core authority write returned no durable receipt");
  if (receipt.permissionId !== permissionId || receipt.owner !== owner || !receipt.actorId
    || receipt.expectedRevision !== expectedRevision || !receipt.resultRevision || receipt.resultRevision === expectedRevision
    || receipt.idempotencyKey !== idempotencyKey || receipt.outcome !== "updated"
    || receipt.target?.machineId || receipt.target?.nodeId || receipt.target?.enrollmentGeneration) {
    throw new Error("Core authority write returned a receipt bound to a different actor, target, revision, or request");
  }
  return {
    id: receipt.id,
    expectedRevision: receipt.expectedRevision,
    resultRevision: receipt.resultRevision,
    idempotencyKey: receipt.idempotencyKey,
    outcome: receipt.outcome,
  };
}
