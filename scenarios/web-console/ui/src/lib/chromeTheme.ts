import { CHROME_PALETTE_TOKEN_NAMES, deriveChromePalette } from "./chromePalette";
import { isHexColor } from "./paneColor";

/**
 * Adaptive app-chrome color controller.
 *
 * Drives a derived palette plus two compatibility CSS custom properties on `<html>`:
 *   - `--wc-chrome-color` — the tint applied to the iOS notch / status-bar
 *     backing (the `body` background), the top app bar, the bottom mobile
 *     toolbar, and — in `sidebar` mode — the desktop sidebar surface.
 *   - `--wc-chrome-fg`    — a contrast-correct foreground (light vs dark) for
 *     text/icons rendered on the tinted chrome.
 *   - the semantic `--wc-*` design tokens consumed by child surfaces.
 * It also updates the `<meta name="theme-color">` tag (Android / PWA status bar).
 *
 * Everything here is **imperative on purpose**. The detected terminal
 * background changes frequently while a full-screen TUI repaints; routing that
 * through React state would re-render the heavy `TerminalPane` / xterm subtree
 * on every tick. Instead the focused pane's detector writes straight here, a
 * change-guard collapses no-op ticks (a busy TUI repaints constantly but its
 * dominant background rarely changes), and only a small CSS recalc on the
 * chrome regions results.
 *
 * Fallback chain: detected rendered bg → owner pane's configured theme
 * background → app default (vars removed, CSS falls back to the slate token).
 */

// Default meta theme-color when no tint is active. Matches the slate body
// background (rgb(--wc-surface-header) = #0f172a) shown behind the notch, so
// the OS status bar stays consistent with the un-tinted chrome.
const DEFAULT_THEME_COLOR = "#0f172a";

export interface ChromeConfig {
  /** Adaptive chrome is on AND the layout has a single focused pane. */
  enabled: boolean;
  /** sessionId of the pane that currently owns the chrome (the focused pane). */
  ownerSessionId: string | null;
  /** The owner pane's configured theme background (detection fallback). */
  fallbackColor: string | null;
}

let config: ChromeConfig = { enabled: false, ownerSessionId: null, fallbackColor: null };
const detected = new Map<string, string>();

// Change-guard state. `applied` holds the last color string written (or null
// for "default chrome"); `initialized` guards the very first apply so the
// default branch runs once on startup.
let appliedColor: string | null = null;
let initialized = false;
let metaEl: HTMLMetaElement | null = null;
// An active top-chrome banner outranks the terminal-derived tint for the OS
// status bar only. The banner is the most urgent thing on screen, so the notch
// should match it; the in-app chrome tokens stay terminal-derived so the rest
// of the UI does not flash when a banner appears. Owned by BannerRegion.
let bannerOverride: string | null = null;
let appliedMeta: string | null = null;

function resolveColor(): string | null {
  if (!config.enabled || !config.ownerSessionId) return null;
  const d = detected.get(config.ownerSessionId);
  if (d && isHexColor(d)) return d;
  if (config.fallbackColor && isHexColor(config.fallbackColor)) return config.fallbackColor;
  return null;
}

/** Resolve and write `<meta name="theme-color">`: banner ▸ chrome ▸ default. */
function applyMeta(chromeColor: string | null): void {
  if (typeof document === "undefined") return;
  const next = bannerOverride ?? chromeColor ?? DEFAULT_THEME_COLOR;
  if (next === appliedMeta) return;
  appliedMeta = next;
  if (!metaEl) {
    metaEl = document.querySelector('meta[name="theme-color"]');
  }
  metaEl?.setAttribute("content", next);
}

function apply(): void {
  if (typeof document === "undefined") return;
  const color = resolveColor();
  if (initialized && color === appliedColor) {
    applyMeta(color); // the banner override may still have moved
    return;           // change-guard: no-op tick for the token work
  }
  appliedColor = color;
  initialized = true;
  const root = document.documentElement.style;
  if (color) {
    const palette = deriveChromePalette(color);
    for (const [name, value] of Object.entries(palette)) {
      root.setProperty(name, value);
    }
    document.documentElement.dataset.wcAdaptiveChrome = "true";
    root.setProperty("--wc-chrome-color", color);
    root.setProperty("--wc-chrome-fg", `rgb(${palette["--wc-text-secondary"]})`);
  } else {
    for (const name of CHROME_PALETTE_TOKEN_NAMES) {
      root.removeProperty(name);
    }
    delete document.documentElement.dataset.wcAdaptiveChrome;
    root.removeProperty("--wc-chrome-color");
    root.removeProperty("--wc-chrome-fg");
  }
  applyMeta(color);
}

export const chromeTheme = {
  /**
   * Set the (low-frequency) owner/enable/fallback config. Called from an
   * effect in the layout shell when the display mode, focused pane, the
   * adaptive setting, or the owner's configured theme changes.
   */
  setConfig(next: ChromeConfig): void {
    config = next;
    apply();
  },
  /**
   * Report a pane's detected rendered background (or `null` to clear). Called
   * directly from the per-pane detector — never via React state. Only the
   * owner pane's detection moves the chrome.
   */
  setDetected(sessionId: string, color: string | null): void {
    if (color && isHexColor(color)) {
      if (detected.get(sessionId) === color) return; // identical — skip apply
      detected.set(sessionId, color);
    } else if (detected.has(sessionId)) {
      detected.delete(sessionId);
    } else {
      return;
    }
    if (sessionId === config.ownerSessionId) apply();
  },
  /**
   * Let the active top-chrome banner own the OS status-bar colour. Pass `null`
   * when no banner is showing. Only `<meta name="theme-color">` is affected —
   * the in-app chrome tokens stay terminal-derived.
   */
  setBannerOverride(color: string | null): void {
    const next = color && isHexColor(color) ? color : null;
    if (next === bannerOverride) return;
    bannerOverride = next;
    applyMeta(appliedColor);
  },
  /** Current applied chrome color, or `null` when the default chrome is shown. */
  getAppliedColor(): string | null {
    return appliedColor;
  },
  /** Test-only: reset all module state. */
  _reset(): void {
    config = { enabled: false, ownerSessionId: null, fallbackColor: null };
    detected.clear();
    appliedColor = null;
    initialized = false;
    metaEl = null;
    bannerOverride = null;
    appliedMeta = null;
  },
};
