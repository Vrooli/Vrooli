export interface ApprovedDependencyRecord {
  ecosystem: string;
  package_name: string;
  version_range: string;
  state: string;
  range_policy?: string;
  rationale?: string;
  approved_by?: string;
  approved_date?: string;
  last_reviewed?: string;
  review_expires?: string;
  license_notes?: string;
  security_notes?: string;
}

export interface ApprovedDependencySnapshot {
  principal: string;
  source: string;
  source_revision?: string;
  source_digest: string;
  schema_version: string;
  version: string;
  mode: string;
  guidance: string;
  records: ApprovedDependencyRecord[];
}

export interface PermissionRegistryReader {
  listPermissions(request: { owner: string }): Promise<{ permissions: Array<{ id: string; owner: string; schemaVersion: string; canRead: boolean; canWrite: boolean }> }>;
  readPermission(request: { permissionId: string; owner: string }): Promise<{ revision: string; valueJson: Uint8Array }>;
}

const permissionID = "onboarding/approved-dependencies";
const owner = "scenario-dependency-analyzer";

export async function readApprovedDependencySnapshot(registry: PermissionRegistryReader): Promise<ApprovedDependencySnapshot> {
  const inventory = await registry.listPermissions({ owner });
  const descriptor = inventory.permissions.find((item) => item.id === permissionID && item.owner === owner);
  if (!descriptor?.canRead || descriptor.canWrite) throw new Error("Approved dependency registry is not available as a read-only permission");
  const result = await registry.readPermission({ permissionId: permissionID, owner });
  if (!descriptor.schemaVersion || result.revision !== descriptor.schemaVersion) throw new Error("Approved dependency source changed; refresh permission inventory");
  const snapshot = JSON.parse(new TextDecoder().decode(result.valueJson)) as ApprovedDependencySnapshot;
  if (snapshot.schema_version !== "1" || !snapshot.source_digest || !snapshot.source || !snapshot.version || !snapshot.mode || !Array.isArray(snapshot.records)) {
    throw new Error("Approved dependency registry has an unsupported or incomplete version");
  }
  if (snapshot.records.some((record) => !record.ecosystem || !record.package_name || !record.version_range || !record.state)) {
    throw new Error("Approved dependency registry contains an incomplete record");
  }
  return snapshot;
}
