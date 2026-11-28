export type ExperienceFreshnessState = "never_run" | "older_than_last_ui_change" | "evaluated_after_last_ui_change" | "ui_surface_unavailable" | "ui_change_time_unavailable";

export function experienceFreshness(state?: ExperienceFreshnessState): string {
  switch (state) {
    case "never_run": return "Never run";
    case "older_than_last_ui_change": return "Older than the last UI change";
    case "evaluated_after_last_ui_change": return "Evaluated after the last UI change";
    case "ui_surface_unavailable": return "UI surface unavailable";
    case "ui_change_time_unavailable": return "UI change time unavailable";
    default: return "Freshness unavailable";
  }
}

export function formatExperienceStatus(status?: string): string {
  if (!status) return "Unknown result";
  const words = status.trim().replace(/_/g, " ").toLowerCase();
  return words ? `${words.charAt(0).toUpperCase()}${words.slice(1)}` : "Unknown result";
}

export function experienceVerdictTone(freshness?: ExperienceFreshnessState, status?: string): "success" | "failure" | "warning" {
  if (freshness === "evaluated_after_last_ui_change" && status?.trim().toLowerCase() === "passed") return "success";
  if (status?.trim().toLowerCase() === "failed") return "failure";
  return "warning";
}
