import { chromeTheme as libraryChrome } from "@vrooli/react-component-library/ChromeTheme";
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
 * The OS status bar is NOT owned here. That is two platform mechanisms that
 * must always agree — `<meta name="theme-color">` on Android, a painted
 * safe-area strip on iOS black-translucent — and the library's `ChromeTheme`
 * service owns both. This controller registers the terminal-derived tint as
 * that service's *base*, so anything more urgent (a banner) outranks it without
 * either side knowing about the other.
 *
 * What stays here is the part that is genuinely web-console's: detecting a
 * terminal's rendered background and deriving a full semantic palette from it.
 * No other scenario has a terminal to detect, which is why this never belonged
 * in the library and why only the resolved colour crosses the boundary.
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

function resolveColor(): string | null {
  if (!config.enabled || !config.ownerSessionId) return null;
  const d = detected.get(config.ownerSessionId);
  if (d && isHexColor(d)) return d;
  if (config.fallbackColor && isHexColor(config.fallbackColor)) return config.fallbackColor;
  return null;
}

/**
 * Publish the resting chrome to the library service.
 *
 * Only the base is set here. A banner contributes at a higher priority through
 * `BannerRegion`, and the service resolves between them — which is what stops
 * the two from fighting over the notch, and what keeps a dismissed banner from
 * leaving it tinted.
 */
function publishStatusBar(chromeColor: string | null): void {
  libraryChrome.setBase({ statusColor: chromeColor ?? DEFAULT_THEME_COLOR });
}

function apply(): void {
  if (typeof document === "undefined") return;
  const color = resolveColor();
  if (initialized && color === appliedColor) {
    publishStatusBar(color); // the resolved base may still have moved
    return;                  // change-guard: no-op tick for the token work
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
  publishStatusBar(color);
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
    libraryChrome._reset();
  },
};
