/**
 * @libraryId react-component-library:useViewportEnvironment
 * @displayName useViewportEnvironment
 * @description A singleton browser viewport environment with coalesced visual-viewport measurements, focus-qualified keyboard geometry, SSR fallbacks, and an explicit host override.
 * @version 1.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:useViewportEnvironment
 * @vrooliComponentSourceSlot hooks.use-viewport-environment */
import {
  createContext,
  useContext,
  useMemo,
  useSyncExternalStore,
  type CSSProperties,
  type ReactNode,
} from "react";

/**
 * 1.1.0 — `reachesScreenBottom`, and the bottom inset every overlay reads.
 *
 * `--rcl-safe-bottom` was `env(safe-area-inset-bottom)` unconditionally, so
 * every drawer reserved the home indicator's height even when nothing of the
 * app sat over the home indicator:
 *
 *   - with the virtual keyboard up, which covers that edge;
 *   - in an installed iOS 26 app with a translucent status bar, which reports
 *     a viewport shorter than the screen by exactly the status bar's height and
 *     paints no fixed-position content below it. The strip under the app is
 *     not the app's to use, so reserving the inset inside it only stacked a
 *     second gap on the first.
 *
 * The environment now answers once whether the app's bottom edge is the
 * screen's, and `useViewportEnvironmentStyle` hands every overlay root the
 * matching `--rcl-safe-bottom`. The strip itself is left alone: stretching the
 * app over it needs positioning that iOS then lets the page scroll, which
 * fights every scroll-lock and pan correction.
 */
export interface ViewportEnvironmentSnapshot {
  layoutWidth: number;
  layoutHeight: number;
  visibleWidth: number;
  visibleHeight: number;
  offsetLeft: number;
  offsetTop: number;
  scale: number;
  keyboardInset: number;
  keyboardVisible: boolean;
  /**
   * Whether the visible area ends at the physical screen's bottom edge, so
   * the home-indicator inset applies to whatever sits there.
   */
  reachesScreenBottom: boolean;
}

export type ViewportEnvironmentStyle = CSSProperties & {
  "--rcl-viewport-width": string;
  "--rcl-viewport-height": string;
  "--rcl-viewport-offset-left": string;
  "--rcl-viewport-offset-top": string;
  "--rcl-keyboard-inset": string;
  "--rcl-safe-bottom": string;
};

const serverSnapshot: ViewportEnvironmentSnapshot = Object.freeze({
  layoutWidth: 0,
  layoutHeight: 0,
  visibleWidth: 0,
  visibleHeight: 0,
  offsetLeft: 0,
  offsetTop: 0,
  scale: 1,
  keyboardInset: 0,
  keyboardVisible: false,
  reachesScreenBottom: true,
});

const keyboardMinimumPixels = 96;
const keyboardMinimumRatio = 0.15;
const stableDeltaPixels = 2;
const SAFE_BOTTOM = "env(safe-area-inset-bottom, 0px)";

function isEditable(element: Element | null) {
  if (!(element instanceof HTMLElement)) return false;
  if (element.isContentEditable) return true;
  return element.matches(
    "input:not([type='button']):not([type='checkbox']):not([type='radio']):not([type='range']):not([type='submit']):not([type='reset']), textarea, select",
  );
}

function isInstalledApp() {
  const standalone = (navigator as Navigator & { standalone?: boolean }).standalone === true;
  return (
    standalone ||
    (typeof window.matchMedia === "function" &&
      window.matchMedia("(display-mode: standalone)").matches)
  );
}

/** env(safe-area-inset-top) in px, read through a probe because CSS alone exposes it. */
function safeAreaTop(): number {
  if (!document.body) return 0;
  const probe = document.createElement("div");
  probe.style.cssText =
    "position:fixed;top:0;left:0;visibility:hidden;pointer-events:none;padding-top:env(safe-area-inset-top, 0px)";
  document.body.append(probe);
  const value = Number.parseFloat(getComputedStyle(probe).paddingTop);
  probe.remove();
  return Number.isFinite(value) ? value : 0;
}

/**
 * An installed app that owns the whole screen, whose visible area stops short
 * of the screen's bottom by exactly the status-bar inset: the iOS 26 shape.
 * Anything else — a browser tab, a windowed app, an unexplained gap — is
 * trusted as reported.
 */
function stopsShortOfScreen(
  visibleWidth: number,
  visibleHeight: number,
  offsetTop: number,
): boolean {
  if (!isInstalledApp()) return false;
  const { width, height } = window.screen;
  if (width <= 0 || height <= 0) return false;
  // iOS reports the screen in portrait whatever the orientation.
  const landscape = visibleWidth > visibleHeight;
  const screenWidth = landscape ? Math.max(width, height) : Math.min(width, height);
  const screenHeight = landscape ? Math.min(width, height) : Math.max(width, height);
  if (Math.abs(visibleWidth - screenWidth) > stableDeltaPixels) return false;
  const gap = screenHeight - (offsetTop + visibleHeight);
  if (gap <= stableDeltaPixels) return false;
  const top = safeAreaTop();
  return top > 0 && Math.abs(gap - top) <= stableDeltaPixels;
}

function snapshotsEqual(a: ViewportEnvironmentSnapshot, b: ViewportEnvironmentSnapshot) {
  return (Object.keys(a) as Array<keyof ViewportEnvironmentSnapshot>).every(
    (key) => a[key] === b[key],
  );
}

class BrowserViewportEnvironment {
  private snapshot = serverSnapshot;
  private readonly listeners = new Set<() => void>();
  private frame = 0;
  private candidateInset = 0;
  private candidateSamples = 0;
  private keyboardEstablished = false;

  getSnapshot = () => this.snapshot;
  getServerSnapshot = () => serverSnapshot;

  subscribe = (listener: () => void) => {
    this.listeners.add(listener);
    if (this.listeners.size === 1) this.start();
    return () => {
      this.listeners.delete(listener);
      if (this.listeners.size === 0) this.stop();
    };
  };

  private start() {
    if (typeof window === "undefined" || typeof document === "undefined") return;
    window.addEventListener("resize", this.schedule, { passive: true });
    document.addEventListener("focusin", this.schedule, true);
    document.addEventListener("focusout", this.schedule, true);
    window.visualViewport?.addEventListener("resize", this.schedule, { passive: true });
    window.visualViewport?.addEventListener("scroll", this.schedule, { passive: true });
    this.measure();
  }

  private stop() {
    if (typeof window === "undefined" || typeof document === "undefined") return;
    window.removeEventListener("resize", this.schedule);
    document.removeEventListener("focusin", this.schedule, true);
    document.removeEventListener("focusout", this.schedule, true);
    window.visualViewport?.removeEventListener("resize", this.schedule);
    window.visualViewport?.removeEventListener("scroll", this.schedule);
    if (this.frame) window.cancelAnimationFrame(this.frame);
    this.frame = 0;
    this.candidateInset = 0;
    this.candidateSamples = 0;
    this.keyboardEstablished = false;
    this.snapshot = serverSnapshot;
  }

  private schedule = () => {
    if (this.frame || typeof window === "undefined") return;
    this.frame = window.requestAnimationFrame(() => {
      this.frame = 0;
      this.measure();
    });
  };

  private measure = () => {
    if (typeof window === "undefined" || typeof document === "undefined") return;
    const visual = window.visualViewport;
    const layoutWidth = Math.max(0, window.innerWidth);
    const layoutHeight = Math.max(0, window.innerHeight);
    const visibleWidth = Math.max(0, visual?.width ?? layoutWidth);
    const visibleHeight = Math.max(0, visual?.height ?? layoutHeight);
    const offsetLeft = Math.max(0, visual?.offsetLeft ?? 0);
    const offsetTop = Math.max(0, visual?.offsetTop ?? 0);
    const scale = visual?.scale && visual.scale > 0 ? visual.scale : 1;
    const rawInset = Math.max(0, layoutHeight - visibleHeight - offsetTop);
    const minimumInset = Math.max(keyboardMinimumPixels, layoutHeight * keyboardMinimumRatio);
    const eligible = isEditable(document.activeElement) && Math.abs(scale - 1) < 0.01;

    if (!eligible || rawInset < minimumInset) {
      this.keyboardEstablished = false;
      this.candidateInset = 0;
      this.candidateSamples = 0;
    } else if (!this.keyboardEstablished) {
      if (Math.abs(rawInset - this.candidateInset) <= stableDeltaPixels) this.candidateSamples += 1;
      else {
        this.candidateInset = rawInset;
        this.candidateSamples = 1;
      }
      this.keyboardEstablished = this.candidateSamples >= 2;
      if (!this.keyboardEstablished) this.schedule();
    }

    const keyboardVisible = this.keyboardEstablished;
    const next: ViewportEnvironmentSnapshot = {
      layoutWidth,
      layoutHeight,
      visibleWidth,
      visibleHeight,
      offsetLeft,
      offsetTop,
      scale,
      keyboardInset: keyboardVisible ? rawInset : 0,
      keyboardVisible,
      reachesScreenBottom:
        !keyboardVisible && !stopsShortOfScreen(visibleWidth, visibleHeight, offsetTop),
    };
    if (snapshotsEqual(this.snapshot, next)) return;
    this.snapshot = next;
    this.listeners.forEach((listener) => listener());
  };
}

const browserEnvironment = new BrowserViewportEnvironment();
const overrideContext = createContext<ViewportEnvironmentSnapshot | null>(null);

export interface ViewportEnvironmentProviderProps {
  value: ViewportEnvironmentSnapshot;
  children: ReactNode;
}

export function ViewportEnvironmentProvider({ value, children }: ViewportEnvironmentProviderProps) {
  return <overrideContext.Provider value={value}>{children}</overrideContext.Provider>;
}

export function useViewportEnvironment() {
  const override = useContext(overrideContext);
  const browser = useSyncExternalStore(
    browserEnvironment.subscribe,
    browserEnvironment.getSnapshot,
    browserEnvironment.getServerSnapshot,
  );
  return override ?? browser;
}

export function useViewportEnvironmentStyle(): ViewportEnvironmentStyle {
  const viewport = useViewportEnvironment();
  return useMemo(
    () => ({
      "--rcl-viewport-width": `${viewport.visibleWidth}px`,
      "--rcl-viewport-height": `${viewport.visibleHeight}px`,
      "--rcl-viewport-offset-left": `${viewport.offsetLeft}px`,
      "--rcl-viewport-offset-top": `${viewport.offsetTop}px`,
      "--rcl-keyboard-inset": `${viewport.keyboardInset}px`,
      "--rcl-safe-bottom": viewport.reachesScreenBottom ? SAFE_BOTTOM : "0px",
    }),
    [
      viewport.keyboardInset,
      viewport.offsetLeft,
      viewport.offsetTop,
      viewport.reachesScreenBottom,
      viewport.visibleHeight,
      viewport.visibleWidth,
    ],
  );
}
