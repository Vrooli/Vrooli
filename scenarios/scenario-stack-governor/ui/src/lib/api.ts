import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

export type Health = { status: string; service: string; timestamp: string };

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) throw new Error(`API health check failed: ${res.status}`);
  return res.json() as Promise<Health>;
}

export type RuleDefinition = {
  id: string;
  title: string;
  summary: string;
  why_important: string;
  category: string;
  severity: "error" | "warn" | "info" | string;
  default_enabled: boolean;
  fixable: boolean;
};

export type RulesConfig = {
  version: string;
  enabled_rules: Record<string, boolean>;
};

export type RuleWithState = RuleDefinition & { enabled: boolean };

export async function fetchRules() {
  const url = buildApiUrl("/rules", { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  if (!res.ok) throw new Error(`Rules fetch failed: ${res.status}`);
  return res.json() as Promise<{ rules: RuleWithState[]; config: RulesConfig }>;
}

export async function fetchConfig() {
  const url = buildApiUrl("/config", { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  if (!res.ok) throw new Error(`Config fetch failed: ${res.status}`);
  return res.json() as Promise<RulesConfig>;
}

export async function putConfig(cfg: RulesConfig) {
  const url = buildApiUrl("/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cfg)
  });
  if (!res.ok) throw new Error(`Config update failed: ${res.status}`);
  return res.json() as Promise<RulesConfig>;
}

export type Evidence = { type: string; ref?: string; detail?: string };
export type Finding = {
  level: "error" | "warn" | "info" | string;
  message: string;
  evidence?: Evidence[];
  scenario_name?: string;
};
export type RuleResult = {
  rule_id: string;
  passed: boolean;
  started_at: string;
  finished_at: string;
  findings?: Finding[];
  error_count: number;
  warn_count: number;
};
export type RunResponse = { repo_root: string; results: RuleResult[] };

export type ScenariosResponse = { scenarios: string[] };

export async function fetchScenarios() {
  const url = buildApiUrl("/scenarios", { baseUrl: API_BASE });
  const res = await fetch(url, { headers: { "Content-Type": "application/json" }, cache: "no-store" });
  if (!res.ok) throw new Error(`Scenarios fetch failed: ${res.status}`);
  return res.json() as Promise<ScenariosResponse>;
}

export async function runRules(scenarioNames?: string[]) {
  const url = buildApiUrl("/run", { baseUrl: API_BASE });
  const body: Record<string, unknown> = {};
  if (scenarioNames && scenarioNames.length > 0) {
    body.scenario_names = scenarioNames;
  }
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`Run failed: ${res.status} ${body.slice(0, 200)}`);
  }
  return res.json() as Promise<RunResponse>;
}

export type FixChange = { type: string; detail: string };
export type FileDiff = { before: string; after: string };
export type FixResult = {
  scenario_name: string;
  rule_id: string;
  fixed: boolean;
  file_path: string;
  changes: FixChange[];
  error?: string;
  diff?: FileDiff;
};
export type FixRequest = {
  scenario_names: string[];
  rule_ids?: string[];
  dry_run?: boolean;
};
export type FixResponse = { repo_root: string; results: FixResult[] };

export async function fixRules(req: FixRequest) {
  const url = buildApiUrl("/fix", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req)
  });
  if (!res.ok) throw new Error(`Fix failed: ${res.status}`);
  return res.json() as Promise<FixResponse>;
}
