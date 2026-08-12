// CROSS-LANGUAGE COUPLING: Preset durations must match `presetDurations`
// in api/session_policy.go. Adding/removing a preset requires updating both.
export const POLICY_OPTIONS = [
    { mode: "never", label: "Never" },
    { mode: "preset", duration: "1h", label: "1 hour" },
    { mode: "preset", duration: "8h", label: "8 hours" },
    { mode: "preset", duration: "24h", label: "24 hours" },
];
/** Converts a policy mode + duration into a string key for form controls. */
export function policyKey(mode, duration) {
    return mode === "never" ? "never" : `${mode}:${duration ?? ""}`;
}
/**
 * Parses a select value into a policy payload.
 * Returns null for unknown/invalid values so callers can ignore safely.
 */
export function parsePolicySelection(value) {
    if (value === "never")
        return { mode: "never" };
    const [mode, duration] = value.split(":", 2);
    if (mode === "preset" && duration) {
        return { mode: "preset", duration };
    }
    return null;
}
