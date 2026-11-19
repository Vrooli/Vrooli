export interface WorkbenchPreferences {
  density: "comfortable" | "compact";
  defaultDevice: "desktop" | "tablet" | "phone";
  reduceMotion: boolean;
  freshness: "24h" | "7d" | "30d";
  captureDesktop: boolean;
  capturePhone: boolean;
  runOnPublish: boolean;
}

export const WORKBENCH_PREFERENCES_KEY = "rcl.workbench.preferences.v1";

export const DEFAULT_WORKBENCH_PREFERENCES: WorkbenchPreferences = {
  density: "comfortable",
  defaultDevice: "desktop",
  reduceMotion: false,
  freshness: "24h",
  captureDesktop: true,
  capturePhone: true,
  runOnPublish: false,
};

export function readWorkbenchPreferences(): WorkbenchPreferences {
  if (typeof window === "undefined") return { ...DEFAULT_WORKBENCH_PREFERENCES };
  try {
    const raw = window.localStorage.getItem(WORKBENCH_PREFERENCES_KEY);
    if (!raw) return { ...DEFAULT_WORKBENCH_PREFERENCES };
    const parsed = JSON.parse(raw) as Partial<WorkbenchPreferences>;
    return { ...DEFAULT_WORKBENCH_PREFERENCES, ...parsed };
  } catch {
    return { ...DEFAULT_WORKBENCH_PREFERENCES };
  }
}

export function writeWorkbenchPreferences(preferences: WorkbenchPreferences): void {
  if (typeof window === "undefined") return;
  try {
    window.localStorage.setItem(WORKBENCH_PREFERENCES_KEY, JSON.stringify(preferences));
  } catch {
    /* local preference only */
  }
  applyWorkbenchPreferences(preferences);
}

export function applyWorkbenchPreferences(preferences: WorkbenchPreferences): void {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.wbDensity = preferences.density;
  document.documentElement.dataset.wbReduceMotion = String(preferences.reduceMotion);
}
