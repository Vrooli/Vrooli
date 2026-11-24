export interface StandingGrantMetadata {
  permissionId: string;
  id: string;
  subject: string;
  approver: string;
  operations: string[];
  targets: string[];
  state: "approved" | "revoked" | "expired";
  maxConcurrentRuns: number;
  maxRunBudgetUsd: number;
  approvedAt: string;
  expiresAt: string;
  revokedAt?: string;
}
export interface ScopeDeclaration { permissionId: string; scope: string; effect: string; runEligible: boolean; command: string }
export interface AgentManagerPermissionSnapshot { grants: StandingGrantMetadata[]; scopes: ScopeDeclaration[] }

export interface Descriptor {
  id: string; owner: string; principalId: string; scope: string; subjectKind: string; subjectId: string;
  source: string; schemaVersion: string; canRead: boolean; canWrite: boolean;
}
export interface PermissionRegistryReader {
  listPermissions(request: { owner: string }): Promise<{ permissions: Descriptor[] }>;
  readPermission(request: { permissionId: string; owner: string }): Promise<{ permissionId: string; owner: string; revision: string; valueJson: Uint8Array }>;
}

interface GrantJSON {
  id: string; subject: string; approver: string; operations: string[]; targets: string[]; state: string;
  max_concurrent_runs: number; max_run_budget_usd: number; approved_at: string; expires_at: string; revoked_at?: string;
}
interface ScopeJSON {
  scenario: string; scope: string; effect: string; run_eligible: boolean; permissions?: string[];
  command: string; service?: string; method?: string;
}

const owner = "agent-manager";
const prefix = "agent-manager/standing-grant/";
const source = "agent-manager/owner-standing-grant-projection/v1";
const readScope = "agent-manager:read";
const grantSchema = "agent-manager/standing-grant-metadata/v1#";
const scopeSchema = "agent-manager/scope-catalog/v1#";
const grantFields = new Set(["id", "subject", "approver", "operations", "targets", "state", "max_concurrent_runs", "max_run_budget_usd", "approved_at", "expires_at", "revoked_at"]);
const scopeFields = new Set(["scenario", "scope", "effect", "run_eligible", "permissions", "command", "service", "method"]);

export async function readStandingGrantProjection(registry: PermissionRegistryReader): Promise<AgentManagerPermissionSnapshot> {
  const inventory = await registry.listPermissions({ owner });
  const entries = inventory.permissions.filter((item) => item.owner === owner && item.id.startsWith(prefix));
  const grants: StandingGrantMetadata[] = [];
  const scopes: ScopeDeclaration[] = [];
  for (let start = 0; start < entries.length; start += 20) {
    const batch = entries.slice(start, start + 20);
    const rows = await Promise.all(batch.map(async (item) => {
      if (!item.canRead || item.canWrite || item.scope !== readScope || item.source !== source || !item.principalId) throw new Error("Agent Manager permission descriptor is unsupported or writable");
      const kind = item.subjectKind === "standing-grant" ? "grant" : item.subjectKind === "scope-declaration" ? "scope" : "unknown";
      const schemaPrefix = kind === "grant" ? grantSchema : scopeSchema;
      if (kind === "unknown" || !item.schemaVersion.startsWith(schemaPrefix) || !item.subjectId) throw new Error("Agent Manager permission descriptor has an unsupported version");
      const result = await registry.readPermission({ permissionId: item.id, owner });
      if (result.permissionId !== item.id || result.owner !== owner || result.revision !== item.schemaVersion.slice(schemaPrefix.length)) throw new Error("Agent Manager permission source changed; refresh inventory");
      const value = JSON.parse(new TextDecoder().decode(result.valueJson)) as GrantJSON & ScopeJSON;
      if (kind === "scope") {
        if (Object.keys(value).some((key) => !scopeFields.has(key)) || value.scenario !== owner || value.scope !== item.subjectId || !value.effect || typeof value.run_eligible !== "boolean" || !value.command || !["read", "write", "destructive"].includes(value.effect)) throw new Error("Agent Manager returned an unsupported scope declaration");
        return { kind, value: { permissionId: item.id, scope: value.scope, effect: value.effect, runEligible: value.run_eligible, command: value.command } as ScopeDeclaration };
      }
      const grant = value as GrantJSON;
      if (Object.keys(grant).some((key) => !grantFields.has(key)) || grant.id !== item.subjectId || !grant.subject || !grant.approver || !Array.isArray(grant.operations) || grant.operations.length === 0 || !Array.isArray(grant.targets) || grant.targets.length === 0 || !["approved", "revoked", "expired"].includes(grant.state) || !Number.isInteger(grant.max_concurrent_runs) || grant.max_concurrent_runs < 0 || !Number.isFinite(grant.max_run_budget_usd) || grant.max_run_budget_usd < 0 || !validDate(grant.approved_at) || !validDate(grant.expires_at) || (grant.revoked_at !== undefined && !validDate(grant.revoked_at))) throw new Error("Agent Manager returned unsupported standing-grant metadata");
      return { kind, value: {
        permissionId: item.id, id: grant.id, subject: grant.subject, approver: grant.approver, operations: grant.operations,
        targets: grant.targets, state: grant.state as StandingGrantMetadata["state"], maxConcurrentRuns: grant.max_concurrent_runs,
        maxRunBudgetUsd: grant.max_run_budget_usd, approvedAt: grant.approved_at, expiresAt: grant.expires_at, revokedAt: grant.revoked_at,
      } as StandingGrantMetadata };
    }));
    for (const row of rows) { if (row.kind === "scope") scopes.push(row.value as ScopeDeclaration); else grants.push(row.value as StandingGrantMetadata); }
  }
  return { grants, scopes };
}

function validDate(value: string): boolean { return Boolean(value && Number.isFinite(Date.parse(value))); }
