/**
 * Client-persisted display & accessibility preferences.
 *
 * These are UI-only prefs — they take effect entirely in the browser and never
 * touch the backend. Each is persisted to `localStorage` and applied to the
 * `<html>` element via an effect (mirroring `theme/ThemeProvider`), so the
 * whole app honors them and the choice survives a reload.
 *
 *   - fontScale     → `data-font-scale` on <html> (CSS scales the root font).
 *   - reducedMotion → `data-reduced-motion` on <html> (overrides the OS
 *                     `prefers-reduced-motion` gate the design already honors).
 *   - textDirection → `dir` on <html> ("auto" follows the active locale).
 *   - handedness    → `data-handedness` on <html> (mobile primary-action side).
 *
 * The store is a tiny `useState` + `localStorage` adapter exposed through a
 * context provider so a single effect owns the document writes and tests can
 * assert persist + apply in isolation.
 */
import { getCurrentLocale, getLocaleConfig, i18n } from "../../i18n";

export type FontScale = "small" | "default" | "large" | "xlarge";
export type ReducedMotion = "system" | "always" | "never";
export type TextDirection = "auto" | "ltr" | "rtl";
export type Handedness = "left" | "right";

export interface SettingsState {
  fontScale: FontScale;
  reducedMotion: ReducedMotion;
  textDirection: TextDirection;
  handedness: Handedness;
}

export const DEFAULT_SETTINGS: SettingsState = {
  fontScale: "default",
  reducedMotion: "system",
  textDirection: "auto",
  handedness: "right",
};

export const FONT_SCALES: readonly FontScale[] = ["small", "default", "large", "xlarge"];
export const REDUCED_MOTION_CHOICES: readonly ReducedMotion[] = ["system", "always", "never"];
export const TEXT_DIRECTION_CHOICES: readonly TextDirection[] = ["auto", "ltr", "rtl"];
export const HANDEDNESS_CHOICES: readonly Handedness[] = ["left", "right"];

export const SETTINGS_STORAGE_KEY = "vrooli.display-settings";

const FONT_SCALE_SET = new Set<string>(FONT_SCALES);
const REDUCED_MOTION_SET = new Set<string>(REDUCED_MOTION_CHOICES);
const TEXT_DIRECTION_SET = new Set<string>(TEXT_DIRECTION_CHOICES);
const HANDEDNESS_SET = new Set<string>(HANDEDNESS_CHOICES);

/**
 * Read persisted settings, tolerating a missing/partial/corrupt blob: any
 * unknown or absent field falls back to its default. External JSON, so every
 * access is guarded.
 */
export function readStoredSettings(): SettingsState {
  if (typeof window === "undefined") {
    return { ...DEFAULT_SETTINGS };
  }
  const raw = window.localStorage.getItem(SETTINGS_STORAGE_KEY);
  if (!raw) {
    return { ...DEFAULT_SETTINGS };
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return { ...DEFAULT_SETTINGS };
  }
  if (typeof parsed !== "object" || parsed === null) {
    return { ...DEFAULT_SETTINGS };
  }
  const record = parsed as Record<string, unknown>;
  return {
    fontScale: FONT_SCALE_SET.has(record.fontScale as string)
      ? (record.fontScale as FontScale)
      : DEFAULT_SETTINGS.fontScale,
    reducedMotion: REDUCED_MOTION_SET.has(record.reducedMotion as string)
      ? (record.reducedMotion as ReducedMotion)
      : DEFAULT_SETTINGS.reducedMotion,
    textDirection: TEXT_DIRECTION_SET.has(record.textDirection as string)
      ? (record.textDirection as TextDirection)
      : DEFAULT_SETTINGS.textDirection,
    handedness: HANDEDNESS_SET.has(record.handedness as string)
      ? (record.handedness as Handedness)
      : DEFAULT_SETTINGS.handedness,
  };
}

const writeStoredSettings = (settings: SettingsState): void => {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
};

/** Resolve the effective text direction: "auto" follows the active locale. */
export const resolveDirection = (choice: TextDirection): "ltr" | "rtl" => {
  if (choice === "ltr" || choice === "rtl") {
    return choice;
  }
  return getLocaleConfig(getCurrentLocale()).dir;
};

/**
 * Apply every setting to the document element. Idempotent and safe to call on
 * each change. "default"/"system"/"right" clear their attribute so the
 * baseline CSS owns the unset case.
 */
export function applySettings(settings: SettingsState): void {
  if (typeof document === "undefined") {
    return;
  }
  const root = document.documentElement;

  if (settings.fontScale === "default") {
    root.removeAttribute("data-font-scale");
  } else {
    root.setAttribute("data-font-scale", settings.fontScale);
  }

  if (settings.reducedMotion === "system") {
    root.removeAttribute("data-reduced-motion");
  } else {
    root.setAttribute("data-reduced-motion", settings.reducedMotion);
  }

  if (settings.handedness === "right") {
    root.removeAttribute("data-handedness");
  } else {
    root.setAttribute("data-handedness", settings.handedness);
  }

  root.dir = resolveDirection(settings.textDirection);
}

export { i18n, writeStoredSettings };
