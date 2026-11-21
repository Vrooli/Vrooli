/**
 * Severity → Tailwind chip class mapping for fleet rule conformance.
 *
 * The backend reports `worst_severity` as a lowercase free-text string
 * ("error" / "warning" / "info"). We map it to a token-driven chip class so a
 * fleet operator can rank rules by blast radius at a glance. Unknown values
 * fall back to the muted style rather than throwing — the UI must never
 * crash on an unexpected severity the backend later adds.
 */
export type SeverityClass = string;

const SEVERITY_CLASSES: Record<string, SeverityClass> = {
  error: "border border-app-danger/40 bg-app-danger/10 text-app-danger",
  warning: "border border-app-warning/40 bg-app-warning/10 text-app-warning",
  info: "border border-app-info/40 bg-app-info/10 text-app-info",
};

const FALLBACK_CLASS =
  "border border-app-border bg-app-surface-muted text-app-muted-foreground";

export function severityChipClass(severity: string): SeverityClass {
  return SEVERITY_CLASSES[severity.toLowerCase()] ?? FALLBACK_CLASS;
}
