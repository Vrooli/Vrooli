// DOC: docs/concepts/GLOSSARY.md#core-terms
// DOC: docs/reference/configuration.md
import type { PolicyMode } from "../api/sessions";

// [REQ:P1-001b] Policy Configuration UI
// Shared policy option definitions used by the unified settings surface.

export interface PolicyOption {
  mode: PolicyMode;
  duration?: string;
  label: string;
}

export interface ParsedPolicySelection {
  mode: PolicyMode;
  duration?: string;
}

// CROSS-LANGUAGE COUPLING: Preset durations must match `presetDurations`
// in api/internal/policy/policy.go, and idle durations must lie within its
// idle bounds (15m to 168h). Adding/removing an option requires checking both.
export const POLICY_OPTIONS: PolicyOption[] = [
  { mode: "never", label: "Never" },
  { mode: "preset", duration: "1h", label: "1 hour" },
  { mode: "preset", duration: "8h", label: "8 hours" },
  { mode: "preset", duration: "24h", label: "24 hours" },
  // Archived once the agent has done no work for this long with no goal or
  // turn running; reopen brings the conversation back.
  { mode: "idle", duration: "6h", label: "When idle 6 hours" },
  { mode: "idle", duration: "24h", label: "When idle 24 hours" },
];

/** Converts a policy mode + duration into a string key for form controls. */
export function policyKey(mode: PolicyMode, duration?: string): string {
  return mode === "never" ? "never" : `${mode}:${duration ?? ""}`;
}

/**
 * Parses a select value into a policy payload.
 * Returns null for unknown/invalid values so callers can ignore safely.
 */
export function parsePolicySelection(value: string): ParsedPolicySelection | null {
  if (value === "never") return { mode: "never" };
  const [mode, duration] = value.split(":", 2);
  if ((mode === "preset" || mode === "idle") && duration) {
    return { mode, duration };
  }
  return null;
}
