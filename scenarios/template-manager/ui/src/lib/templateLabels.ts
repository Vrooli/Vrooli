/**
 * Domain enum → label + tone mappers shared by the dashboard overview and the
 * drill-down detail views. Centralised so the meaning of a `TemplateKind`,
 * `ValidationMode`, run status, or debt severity never drifts between the card
 * that links to a detail view and the detail view itself.
 *
 * These map the *numeric* proto enum values (and the free-form status/severity
 * strings the engine persists) rather than importing the enum objects, because
 * the dashboard already receives plain numbers over the wire.
 */
export type Tone = "neutral" | "success" | "warning" | "danger" | "info";

export function kindLabel(kind: number): string {
  switch (kind) {
    case 1:
      return "scenario";
    case 2:
      return "design";
    case 3:
      return "resource";
    default:
      return "template";
  }
}

export function modeLabel(mode: number): string {
  switch (mode) {
    case 1:
      return "shallow";
    case 2:
      return "deep";
    case 3:
      return "drift";
    default:
      return "validation";
  }
}

/** Tone for a validation run / phase status string. */
export function runStatusTone(status: string): Tone {
  switch (status) {
    case "passed":
    case "ok":
    case "green":
      return "success";
    case "failed":
    case "error":
    case "red":
      return "danger";
    case "running":
    case "pending":
      return "info";
    default:
      return "warning";
  }
}

/** Tone for a debt entry status string. */
export function debtStatusTone(status: string): Tone {
  switch (status) {
    case "open":
      return "danger";
    case "resolved":
    case "closed":
      return "success";
    case "acknowledged":
    case "tracked":
      return "warning";
    default:
      return "neutral";
  }
}

/** Tone for a debt severity string. */
export function severityTone(severity: string): Tone {
  switch (severity) {
    case "critical":
    case "high":
      return "danger";
    case "medium":
      return "warning";
    case "low":
      return "info";
    default:
      return "neutral";
  }
}

/** Tone for a drift snapshot given how many items drifted. */
export function driftTone(driftCount: number): Tone {
  return driftCount > 0 ? "warning" : "success";
}
