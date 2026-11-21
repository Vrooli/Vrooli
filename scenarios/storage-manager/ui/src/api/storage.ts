import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export type StorageOwner = {
  kind: string;
  id: string;
  manifest_path: string;
  storage_declared?: boolean;
  storage_entries?: StorageEntry[];
};

export type StorageEntry = {
  name: string;
  kind: string;
  rung?: string;
  path?: { value?: string; by_os?: Record<string, string> } | string;
  budget?: { max_age?: string; max_bytes?: string; rationale?: string };
};

export type InventoryFinding = {
  code: string;
  severity: string;
  owner_kind?: string;
  owner_id?: string;
  manifest_path?: string;
  message: string;
};

export type StorageInventory = {
  repo_root: string;
  owners: StorageOwner[];
  findings?: InventoryFinding[];
};

export type CensusEntry = {
  owner: string;
  kind?: string;
  name: string;
  path: string;
  bytes: number;
  declared: boolean;
};

export type CensusFinding = {
  code: string;
  severity: string;
  owner?: string;
  kind?: string;
  path?: string;
  message: string;
};

export type CensusReport = {
  snapshot_id?: string;
  observed_at?: string;
  root: string;
  measured_bytes: number;
  attributed_bytes: number;
  unattributed_bytes: number | null;
  unreadable_bytes?: number;
  closed: boolean;
  accounting_identity: boolean;
  confidence: string;
  scan_coverage?: { measured_bytes: number; device_used_bytes?: number; device_total_bytes?: number; complete: boolean };
  growth_slope_bytes_per_hour?: number;
  owner_counts?: Record<string, number>;
  unreadable_paths?: string[];
  findings?: CensusFinding[];
  entries: CensusEntry[];
  framework_roots?: string[];
};

export type AdoptionKind = { total: number; storage_declared: number; with_storage: number; with_budget: number };
export type AdoptionSuggestion = {
  kind: string;
  owner: string;
  manifest_path: string;
  priority: string;
  reason: string;
  observed_bytes?: number;
  measurement_complete: boolean;
};
export type AdoptionReport = {
  total_owners: number;
  findings: number;
  by_kind: Record<string, AdoptionKind>;
  suggestions?: AdoptionSuggestion[];
};

export type InfraHealthReport = {
  owner_count: number;
  owners_with_declared_ceiling: number;
  declared_ceiling_coverage: number;
  measured_bytes_under_enforced_ceiling: number;
  enforced_ceiling_coverage: number;
  snapshot_count: number;
  confidence: string;
  growth_slope_bytes_per_hour?: number;
  latest_snapshot?: CensusReport;
};

export type PlacementOwner = {
  kind: string;
  owner: string;
  entry: string;
  rung: string;
  path?: string;
  applicable: boolean;
  error?: string;
};
export type PlacementView = {
  platform: string;
  owners: PlacementOwner[];
  lever_warnings?: Array<{ key: string; message: string }>;
  lever_error?: string;
};

export type RetentionOwner = {
  kind: string;
  id: string;
  manifest_path: string;
  budgets?: Array<{ name: string; target_kind: string; max_age?: string; max_bytes?: number; rationale?: string }>;
  enforcement_state?: string;
  last_enforcement_time?: string | null;
  findings?: Array<{ code: string; budget: string; observed_bytes: number; max_bytes: number; message: string }>;
  error?: string;
};
export type RetentionInventory = { owners: RetentionOwner[]; findings?: InventoryFinding[] };
export type PlacementAudit = {
  id: string;
  plan_id: string;
  entry: string;
  source: string;
  destination: string;
  status: string;
  source_preserved: boolean;
  message?: string;
};

async function getJson<T>(path: string): Promise<T> {
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    method: "GET",
    cache: "no-store",
  });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as T;
}

export const fetchInventory = () => getJson<StorageInventory>("/api/v1/storage/inventory");
export const fetchCensusHistory = () => getJson<CensusReport[]>("/api/v1/census/history?limit=12");
export const fetchAdoption = () => getJson<AdoptionReport>("/api/v1/adoption?measure=true&limit=12");
export const fetchInfraHealth = () => getJson<InfraHealthReport>("/api/v1/infra-health/storage");
export const fetchPlacement = () => getJson<PlacementView>("/api/v1/placement");
export const fetchRetentionOwners = () => getJson<RetentionInventory>("/api/v1/retention/owners");
export const fetchPlacementAudit = () => getJson<PlacementAudit[]>("/api/v1/placement/audit");
