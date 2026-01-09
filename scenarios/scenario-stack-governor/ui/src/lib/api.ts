import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

// Simple! Just specify if you want the /api/v1 suffix
const API_BASE = resolveApiBase({ appendSuffix: true });

export type Health = { status: string; service: string; timestamp: string };

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

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
export type Finding = { level: "error" | "warn" | "info" | string; message: string; evidence?: Evidence[] };
export type RuleResult = { rule_id: string; passed: boolean; started_at: string; finished_at: string; findings?: Finding[] };
export type RunResponse = { repo_root: string; results: RuleResult[] };

export async function runEnabledRules() {
  const url = buildApiUrl("/run", { baseUrl: API_BASE });
  const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" }, body: "{}" });
  if (!res.ok) throw new Error(`Run failed: ${res.status}`);
  return res.json() as Promise<RunResponse>;
}
