import { useEffect } from "react";

/**
 * Centralised viewport-layout hook for the web-console PWA.
 *
 * This is the **single source of truth** for all viewport sizing. Every
 * height-related layout decision flows from the CSS custom properties this
 * hook sets. If you need to change how the app responds to keyboard
 * appearance, safe areas, or viewport resizing — this is the only file you
 * should need to touch.
 *
 * ## CSS custom properties set on `<html>`
 *
 * Layout is driven entirely through CSS so that no React re-render is
 * needed — re-rendering would cause focus-loss in xterm.
 *
 * | Variable            | Value                              | Purpose                                    |
 * |---------------------|------------------------------------|--------------------------------------------|
 * | `--wc-app-height`   | `visualViewport.height` (px)       | The actual visible viewport height,        |
 * |                     |                                    | excluding browser chrome and keyboard.     |
 * | `--wc-kb-height`    | keyboard height (px)               | How much of the viewport the keyboard      |
 * |                     |                                    | occupies. `0px` when keyboard is closed.   |
 * | `--wc-safe-bottom`  | `env(safe-area-inset-bottom)`      | Bottom safe-area inset for devices with    |
 * |                     | or `0px` when keyboard is open     | rounded corners / home indicators.         |
 * |                     |                                    | Automatically set to `0px` when the        |
 * |                     |                                    | keyboard is open (it covers the bottom).   |
 * | `--wc-safe-top`     | `env(safe-area-inset-top)`         | Top safe-area inset for devices with       |
 * |                     |                                    | notches / dynamic islands.                 |
 * | `--wc-safe-left`    | `env(safe-area-inset-left)`        | Left safe-area inset for landscape notches |
 * | `--wc-safe-right`   | `env(safe-area-inset-right)`       | Right safe-area inset for landscape edges. |
 *
 * ## Why NOT `100vh` or `h-screen`?
 *
 * On mobile browsers, `100vh` equals the *largest possible* viewport height
 * (with browser chrome fully hidden). This is taller than the actual visible
 * area, so content using `100vh` extends past the bottom of the screen. The
 * `visualViewport` API gives us the *real* visible height, which we expose
 * as `--wc-app-height`. Components use the Tailwind class `h-wc-app` (mapped
 * to `var(--wc-app-height, 100dvh)` in tailwind.config.ts) instead of
 * `h-screen`.
 *
 * The `100dvh` fallback handles the brief moment before this hook initialises
 * (e.g. loading/error states rendered before the Workspace mounts).
 *
 * ## How it works with the viewport meta tag
 *
 * The HTML `<meta name="viewport">` in index.html includes two critical
 * directives that this hook depends on:
 *
 * - **`interactive-widget=resizes-content`** — tells the browser to shrink
 *   the layout viewport when the virtual keyboard opens. This means
 *   `visualViewport.height` reflects the actual visible area, and we can
 *   use it directly as `--wc-app-height`.
 *
 * - **`viewport-fit=cover`** — extends the layout into device safe areas
 *   (notches, rounded corners, home indicator). Without this, the browser
 *   would inset the layout and `env(safe-area-inset-*)` would always be 0.
 *   Components at the screen edges use `--wc-safe-bottom` (which wraps
 *   `env(safe-area-inset-bottom)`) to add their own padding.
 *
 * ## Overscroll / pull-to-refresh prevention
 *
 * Two complementary mechanisms prevent unwanted page movement:
 *
 * 1. **`position: fixed; inset: 0; overscroll-behavior: none`** on `<body>`
 *    (set in styles.css) — prevents iOS Safari rubber-banding and
 *    pull-to-refresh gestures in standalone PWA mode.
 *
 * 2. **`window.scrollTo(0, 0)`** called on every viewport update — counteracts
 *    the browser's default behaviour of scrolling the page to keep a focused
 *    input visible. Our CSS layout already ensures the focused element is
 *    visible (the container shrinks via `--wc-app-height`), so the browser's
 *    scroll would just push content offscreen. Both mechanisms are required:
 *    `position: fixed` alone doesn't prevent `visualViewport.offsetTop` from
 *    shifting on iOS when an input receives focus.
 *
 * ## Why focusin/focusout polling is needed
 *
 * The `visualViewport` resize event is the primary signal, but some mobile
 * browsers don't fire it reliably for all focus targets. In particular,
 * focusing the MobileToolbar textarea (at the bottom of the layout) may not
 * trigger a resize event on certain devices/browsers, even though the virtual
 * keyboard opens and the viewport shrinks.
 *
 * To handle this, we also listen for `focusin`/`focusout` on the document and
 * poll the viewport dimensions several times during the ~300ms keyboard
 * open/close animation. This ensures the layout adjusts regardless of which
 * element receives focus.
 *
 * ## Consumers
 *
 * - **`Workspace.tsx`** — root container uses `h-wc-app` for its height.
 * - **`MobileToolbar.tsx`** — uses `pb-[var(--wc-safe-bottom)]` for bottom
 *   safe-area padding. Padding disappears when the keyboard opens.
 * - **`App.tsx`** — loading/error states use `h-wc-app` for full-height
 *   centering (the hook hasn't run yet, so the `100dvh` fallback kicks in).
 *
 * ## Test coverage
 *
 * See `useAppViewport.test.ts` for comprehensive tests covering initial var
 * setup, keyboard height calculation, resize/scroll events, focus polling,
 * safe-bottom toggling, scrollTo calls, cleanup, and graceful degradation
 * when `visualViewport` is unavailable.
 */
export function useAppViewport(): void {
  useEffect(() => {
    const vv = window.visualViewport;
    if (!vv) return;

    const update = () => {
      const kbHeight = Math.max(0, window.innerHeight - vv.height - vv.offsetTop);
      const root = document.documentElement.style;
      root.setProperty("--wc-kb-height", `${kbHeight}px`);
      root.setProperty("--wc-app-height", `${vv.height}px`);
      root.setProperty("--wc-safe-top", "env(safe-area-inset-top)");
      root.setProperty("--wc-safe-left", "env(safe-area-inset-left)");
      root.setProperty("--wc-safe-right", "env(safe-area-inset-right)");
      // When the keyboard is open it covers the bottom edge, so the safe-area
      // inset is irrelevant. When closed, use the real device inset.
      root.setProperty("--wc-safe-bottom", kbHeight > 0 ? "0px" : "env(safe-area-inset-bottom)");
      // Prevent the browser from scrolling the page to keep the focused input
      // visible — our CSS layout already handles this, and the scroll would
      // push content offscreen.
      window.scrollTo(0, 0);
    };

    // Poll update() several times over the keyboard animation period.
    // This catches cases where visualViewport resize events don't fire
    // (or fire too early) for certain focus targets.
    let pollTimers: ReturnType<typeof setTimeout>[] = [];
    const pollDuringAnimation = () => {
      // Clear any pending polls from a previous focus change
      for (const t of pollTimers) clearTimeout(t);
      pollTimers = [];
      update();
      // Poll at 50ms intervals over 500ms to cover the keyboard animation
      for (const delay of [50, 100, 150, 200, 300, 500]) {
        pollTimers.push(setTimeout(update, delay));
      }
    };

    const isInputElement = (el: EventTarget | null): boolean => {
      if (!(el instanceof HTMLElement)) return false;
      const tag = el.tagName;
      return tag === "INPUT" || tag === "TEXTAREA" || el.isContentEditable;
    };

    const onFocusIn = (e: FocusEvent) => {
      if (isInputElement(e.target)) pollDuringAnimation();
    };
    const onFocusOut = (e: FocusEvent) => {
      if (isInputElement(e.target)) pollDuringAnimation();
    };

    vv.addEventListener("resize", update);
    vv.addEventListener("scroll", update);
    document.addEventListener("focusin", onFocusIn);
    document.addEventListener("focusout", onFocusOut);
    // Set initial values
    update();
    return () => {
      vv.removeEventListener("resize", update);
      vv.removeEventListener("scroll", update);
      document.removeEventListener("focusin", onFocusIn);
      document.removeEventListener("focusout", onFocusOut);
      for (const t of pollTimers) clearTimeout(t);
      document.documentElement.style.removeProperty("--wc-kb-height");
      document.documentElement.style.removeProperty("--wc-app-height");
      document.documentElement.style.removeProperty("--wc-safe-top");
      document.documentElement.style.removeProperty("--wc-safe-bottom");
      document.documentElement.style.removeProperty("--wc-safe-left");
      document.documentElement.style.removeProperty("--wc-safe-right");
    };
  }, []);
}
