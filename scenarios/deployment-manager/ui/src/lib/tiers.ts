import type { DependencyAnalysisResponse, DeploymentProfile } from "./api";

// ============================================================================
// Tier Configuration
// ============================================================================

export type TierKey = "local" | "desktop" | "mobile" | "saas" | "enterprise";

export interface TierOption {
  id: number;
  key: TierKey;
  label: string;
  description: string;
}

export const TIER_OPTIONS: TierOption[] = [
  { id: 1, key: "local", label: "Tier 1 · Local/Dev", description: "Reference stack, best for iteration" },
  { id: 2, key: "desktop", label: "Tier 2 · Desktop", description: "Windows/macOS/Linux bundles" },
  { id: 3, key: "mobile", label: "Tier 3 · Mobile", description: "iOS/Android packages" },
  { id: 4, key: "saas", label: "Tier 4 · SaaS/Cloud", description: "Managed or self-hosted cloud" },
  { id: 5, key: "enterprise", label: "Tier 5 · Appliance", description: "Dedicated hardware deployments" },
];

export const TIER_NAMES: Record<string, string> = {
  local: "Local/Dev",
  desktop: "Desktop",
  mobile: "Mobile",
  saas: "SaaS/Cloud",
  enterprise: "Enterprise",
};

export const TIER_KEY_BY_ID: Record<string, TierKey> = {
  "1": "local",
  "2": "desktop",
  "3": "mobile",
  "4": "saas",
  "5": "enterprise",
};

export function getTierByKey(key: TierKey): TierOption | undefined {
  return TIER_OPTIONS.find((tier) => tier.key === key);
}

export function getTierById(id: number): TierOption | undefined {
  return TIER_OPTIONS.find((tier) => tier.id === id);
}

export function getFirstMatchingTierKey(tierIds: number[]): TierKey {
  const tier = TIER_OPTIONS.find((t) => tierIds.includes(t.id));
  return tier?.key ?? "desktop";
}

// ============================================================================
// Fitness Score Utilities
// ============================================================================

export function getFitnessColor(score: number): string {
  if (score >= 75) return "text-green-400";
  if (score >= 50) return "text-yellow-400";
  if (score > 0) return "text-orange-400";
  return "text-red-400";
}

// ============================================================================
// Scenario Options
// ============================================================================

const DEFAULT_SCENARIOS = ["picker-wheel", "system-monitor", "browser-automation-studio"];

export function buildScenarioOptions(
  profiles: DeploymentProfile[] | undefined,
  currentInput: string,
  maxResults = 12
): string[] {
  const existing = Array.from(new Set((profiles ?? []).map((p) => p.scenario).filter(Boolean)));
  const combined = Array.from(new Set([...existing, ...DEFAULT_SCENARIOS]));
  const typed = currentInput.trim().toLowerCase();
  const filtered = combined.filter((name) =>
    typed ? name.toLowerCase().includes(typed) : true
  );
  if (typed && !combined.some((name) => name.toLowerCase() === typed)) {
    filtered.unshift(currentInput);
  }
  return filtered.slice(0, maxResults);
}

// ============================================================================
// Type Guards
// ============================================================================

export function isTierKey(value: string): value is TierKey {
  return ["local", "desktop", "mobile", "saas", "enterprise"].includes(value);
}
