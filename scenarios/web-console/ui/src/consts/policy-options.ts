import type { PolicyMode } from "../lib/api";

// [REQ:P1-001b] Policy Configuration UI
// Shared policy option definitions used by SessionDrawer and SessionsPage.

export interface PolicyOption {
  mode: PolicyMode;
  duration?: string;
  label: string;
}

// CROSS-LANGUAGE COUPLING: Preset durations must match `presetDurations`
// in api/session_policy.go. Adding/removing a preset requires updating both.
export const POLICY_OPTIONS: PolicyOption[] = [
  { mode: "never", label: "Never" },
  { mode: "preset", duration: "1h", label: "1 hour" },
  { mode: "preset", duration: "8h", label: "8 hours" },
  { mode: "preset", duration: "24h", label: "24 hours" },
];

/** Converts a policy mode + duration into a string key for form controls. */
export function policyKey(mode: PolicyMode, duration?: string): string {
  return mode === "never" ? "never" : `${mode}:${duration ?? ""}`;
}
