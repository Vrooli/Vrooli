export const normalize = (value: string) => value.trim().toLowerCase();

export const shortPath = (path: string) => path || "unknown";

export const statusToneClass = (status: string) => {
  switch (normalize(status)) {
    case "passed":
    case "ok":
      return "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
    case "failed":
    case "error":
    case "below":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    case "degraded":
    case "warning":
      return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    default:
      return "border-app-border bg-app-surface-muted text-app-muted-foreground";
  }
};

export const severityToneClass = (severity: string) => {
  switch (normalize(severity)) {
    case "error":
      return "border-red-500/40 bg-red-500/10 text-red-700 dark:text-red-300";
    case "warning":
      return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    default:
      return "border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300";
  }
};
