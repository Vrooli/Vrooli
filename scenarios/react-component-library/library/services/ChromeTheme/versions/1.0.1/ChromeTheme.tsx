/**
 * @libraryId react-component-library:ChromeTheme
 * @displayName ChromeTheme
 * @description Scoped runtime service for predictable ChromeTheme behavior.
 * @version 1.0.1
 * @tags ["runtime","state"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

/** @vrooliComponentSource services.chrome-theme */
import { useEffect, useId, type CSSProperties } from "react";

/**
 * "The colour of the status bar" is two different mechanisms, and a surface
 * that sets only one is correct on one platform and stale on the other:
 *
 *   - `<meta name="theme-color">` is read by Android Chrome and installed
 *     PWAs. The OS paints it, and it composites the value itself — alpha there
 *     resolves against black, so this channel must be opaque.
 *   - On iOS standalone (`apple-mobile-web-app-status-bar-style:
 *     black-translucent`) the web view extends *under* the notch and
 *     `theme-color` is ignored entirely. The only thing colouring the status
 *     bar is the element the app paints into the top safe area, which is what
 *     `StatusBarFill` is for. That channel may carry alpha, and usually should,
 *     so the strip and the surface directly beneath it resolve identically.
 *
 * This service owns both, resolves them from one value, and writes them
 * together — so the two can never drift.
 */
export interface ChromeContribution {
  /** Opaque colour for the OS status bar. Alpha here renders as black. */
  readonly statusColor: string;
  /** Colour painted into the safe-area strip. Defaults to `statusColor`. */
  readonly fillColor?: string;
}

interface Contributor {
  readonly key: string;
  readonly priority: number;
  readonly chrome: ChromeContribution;
}

export interface ResolvedChrome {
  readonly statusColor: string;
  readonly fillColor: string;
  /** Key of the contributor that won, or `null` when the base is showing. */
  readonly source: string | null;
}

type Listener = (resolved: ResolvedChrome | null) => void;

const contributors = new Map<string, Contributor>();
const listeners = new Set<Listener>();
let base: ChromeContribution | null = null;
let applied: ResolvedChrome | null = null;
let metaEl: HTMLMetaElement | null = null;

function fillOf(chrome: ChromeContribution): string {
  return chrome.fillColor ?? chrome.statusColor;
}

/**
 * Highest priority wins; ties break on key so the result is stable across
 * renders rather than depending on registration order. Falling through to
 * `base` is what makes an app's resting chrome (a brand colour, a
 * terminal-derived tint) the floor rather than a special case.
 */
function resolve(): ResolvedChrome | null {
  let winner: Contributor | null = null;
  for (const entry of contributors.values()) {
    if (
      !winner ||
      entry.priority > winner.priority ||
      (entry.priority === winner.priority && entry.key < winner.key)
    ) {
      winner = entry;
    }
  }
  if (winner) {
    return {
      statusColor: winner.chrome.statusColor,
      fillColor: fillOf(winner.chrome),
      source: winner.key,
    };
  }
  if (base) return { statusColor: base.statusColor, fillColor: fillOf(base), source: null };
  return null;
}

function apply(): void {
  const next = resolve();
  if (
    next?.statusColor === applied?.statusColor &&
    next?.fillColor === applied?.fillColor &&
    next?.source === applied?.source
  ) {
    return; // change-guard: a contributor moved but the resolved chrome did not
  }
  applied = next;

  if (typeof document !== "undefined") {
    const root = document.documentElement;
    if (next) {
      if (!metaEl) {
        metaEl = document.querySelector('meta[name="theme-color"]');
        if (!metaEl) {
          metaEl = document.createElement("meta");
          metaEl.setAttribute("name", "theme-color");
          document.head.appendChild(metaEl);
        }
      }
      metaEl.setAttribute("content", next.statusColor);
      root.style.setProperty("--rcl-status-fill", next.fillColor);
      root.dataset.rclStatusFill = next.source ?? "base";
    } else {
      root.style.removeProperty("--rcl-status-fill");
      delete root.dataset.rclStatusFill;
    }
  }

  for (const listener of listeners) listener(next);
}

export const chromeTheme = {
  /**
   * The resting appearance, shown whenever no contributor outranks it. Pass
   * `null` to fall back to whatever the page already had.
   */
  setBase(chrome: ChromeContribution | null): void {
    base = chrome;
    apply();
  },
  /**
   * Claim the status bar under `key` until released. Passing `null` for
   * `chrome` releases instead, so a caller can drive this straight from its
   * own "is anything showing?" state without branching.
   *
   * `priority` decides who wins when several claim at once — the same ladder
   * the surfaces themselves are ordered by, so the notch matches the most
   * urgent thing on screen rather than the most recent.
   */
  contribute(key: string, chrome: ChromeContribution | null, priority = 0): void {
    if (!chrome) {
      if (!contributors.delete(key)) return;
    } else {
      const existing = contributors.get(key);
      if (
        existing &&
        existing.priority === priority &&
        existing.chrome.statusColor === chrome.statusColor &&
        fillOf(existing.chrome) === fillOf(chrome)
      ) {
        return;
      }
      contributors.set(key, { key, priority, chrome });
    }
    apply();
  },
  /** Drop a contribution. Safe to call for a key that never contributed. */
  release(key: string): void {
    if (contributors.delete(key)) apply();
  },
  /** The chrome currently painted, or `null` when nothing has claimed it. */
  current(): ResolvedChrome | null {
    return applied;
  },
  /** Observe resolved changes — for hosts mirroring chrome into native shells. */
  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
  /** Test-only: drop all state and un-paint. */
  _reset(): void {
    contributors.clear();
    listeners.clear();
    base = null;
    applied = null;
    metaEl = null;
    if (typeof document !== "undefined") {
      document.documentElement.style.removeProperty("--rcl-status-fill");
      delete document.documentElement.dataset.rclStatusFill;
    }
  },
};

/**
 * Declare a status-bar contribution for as long as this component is mounted
 * and `chrome` is non-null.
 *
 * Prefer this over calling `chromeTheme` directly: the release on unmount is
 * the part that is easy to forget, and a leaked contribution keeps the notch
 * tinted for a surface that is gone.
 */
export function useChromeContribution(
  chrome: ChromeContribution | null,
  options?: { readonly priority?: number; readonly key?: string },
): void {
  const generatedKey = useId();
  const key = options?.key ?? generatedKey;
  const priority = options?.priority ?? 0;
  const statusColor = chrome?.statusColor ?? null;
  const fillColor = chrome ? fillOf(chrome) : null;

  useEffect(() => {
    chromeTheme.contribute(
      key,
      statusColor ? { statusColor, fillColor: fillColor ?? statusColor } : null,
      priority,
    );
    return () => {
      chromeTheme.release(key);
    };
  }, [key, priority, statusColor, fillColor]);
}

const styles = `
[data-rcl-status-fill-strip] { flex: 0 0 auto; block-size: var(--rcl-safe-top, 0px); background-color: var(--rcl-status-fill, transparent); transition: background-color var(--dur-quick, 120ms) ease; }
@media (prefers-reduced-motion: reduce) { [data-rcl-status-fill-strip] { transition: none; } }
`;

export interface StatusBarFillProps {
  readonly className?: string;
  readonly style?: CSSProperties;
  readonly testId?: string;
}

/**
 * The paintable top safe area.
 *
 * Content must not be placed inside it — this is the region the OS draws its
 * own status bar over. Render it as the first child of the app frame, above
 * whatever chrome comes next, and the notch inherits the resolved colour.
 *
 * It ships with the service on purpose: the service cannot tint a strip that
 * the app never rendered, and an app that renders the strip without the
 * service has a permanently transparent notch. They are one contract.
 */
export function StatusBarFill({ className, style, testId }: StatusBarFillProps) {
  return (
    <>
      <StyleSheet name="chrome-theme-1-0-0-1" css={styles} />
      <div
        data-rcl-status-fill-strip
        data-testid={testId}
        aria-hidden="true"
        className={className}
        style={style}
      />
    </>
  );
}
