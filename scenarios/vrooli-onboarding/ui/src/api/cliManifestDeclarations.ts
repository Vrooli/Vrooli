import { permissionRegistryClient as client } from "./permissionRegistry";
const permissionId = "onboarding/cli-manifest-declarations";
const owner = "cli-core";

interface Declaration {
  manifest: string;
  declaring_owner: string;
  command: string;
  declared_effect: string;
  declared_run_eligible: boolean;
  declared_permissions: string[];
  process_spawn_declared: boolean;
  derived_scope: string;
  dispatch_eligible: boolean;
  authorization_source: string;
}

interface DeclarationSnapshot {
  schema_id: string;
  source_digest: string;
  authorization_note: string;
  declarations: Declaration[];
}

export async function listProcessSpawnDeclarations(): Promise<DeclarationSnapshot> {
  const inventory = await client.listPermissions({ owner });
  const descriptor = inventory.permissions.find((item) => item.id === permissionId && item.owner === owner);
  if (!descriptor?.canRead || descriptor.canWrite) throw new Error("CLI declaration source is not available as a read-only permission");
  const result = await client.readPermission({ permissionId, owner });
  if (result.revision !== descriptor.schemaVersion) throw new Error("CLI declaration source changed; refresh permission inventory");
  const snapshot = JSON.parse(new TextDecoder().decode(result.valueJson)) as DeclarationSnapshot;
  if (snapshot.schema_id !== "cli-manifest/v1" || !snapshot.source_digest || !snapshot.authorization_note || !Array.isArray(snapshot.declarations)) {
    throw new Error("CLI declaration source has an unsupported or incomplete version");
  }
  return { ...snapshot, declarations: snapshot.declarations.filter((item) => item.process_spawn_declared) };
}
