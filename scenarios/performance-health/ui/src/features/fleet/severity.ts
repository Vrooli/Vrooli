/**
 * Severity → Tailwind chip class mapping shared by the fleet, audit, and trace
 * surfaces. The backend reports severity as a lowercase free-text string
 * ("error" / "warning" / "info"); unknown values fall back to a muted style
 * rather than throwing so the UI never crashes on a severity the backend later
 * adds. Token-driven so chips track the active theme.
 */
const SEVERITY_CLASSES: Record<string, string> = {
  error: "border border-app-danger/40 bg-app-danger/10 text-app-danger",
  warning: "border border-app-warning/40 bg-app-warning/10 text-app-warning",
  info: "border border-app-info/40 bg-app-info/10 text-app-info",
};

const FALLBACK_CLASS =
  "border border-app-border bg-app-surface-muted text-app-muted-foreground";

export function severityChipClass(severity: string): string {
  return SEVERITY_CLASSES[severity.toLowerCase()] ?? FALLBACK_CLASS;
}
