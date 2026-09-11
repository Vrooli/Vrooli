import type { ViewportEnvironmentSnapshot } from "@vrooli/react-component-library/useViewportEnvironment/1";
import { readSafeAreaInsets } from "./safeArea";

/**
 * Where the app really ends.
 *
 * An installed iOS app (Add to Home Screen, `black-translucent` status bar)
 * can report a visual viewport shorter than the screen by exactly the status
 * bar's height. Everything sized to that viewport — the shell and every drawer
 * that avoids the keyboard — then stops that far above the bottom edge, and
 * each one still pads for the home indicator below it: two gutters, one of
 * them empty. Closing that one recognisable gap here, once, gives the shell and
 * every overlay the same bottom edge, and a single answer to whether the
 * home-indicator inset applies at all.
 *
 * The same iOS versions paint no `position: fixed` content below that short
 * viewport, so the caller also needs to know when the app has been extended
 * past it (`extended`): fixed roots are then positioned against the page
 * instead. See `components/AppViewportProvider.tsx` and `styles.css`.
 */

export interface ScreenFacts {
  /** An installed app, which owns the whole screen. */
  standalone: boolean;
  screenWidth: number;
  screenHeight: number;
  /** env(safe-area-inset-top), in px. */
  safeTop: number;
}

export interface CorrectedViewport {
  viewport: ViewportEnvironmentSnapshot;
  /**
   * Whether the app's bottom edge is the screen's, so the home-indicator inset
   * applies. False while the keyboard covers it, and when the app stops short
   * of the screen for a reason this module does not recognise.
   */
  reachesBottom: boolean;
  /** Whether this correction extended the app past the reported viewport. */
  extended: boolean;
}

/** Pixel slack for rounding between CSS px and the reported screen size. */
const SLACK_PX = 2;

export function correctViewport(viewport: ViewportEnvironmentSnapshot, screen: ScreenFacts): CorrectedViewport {
  if (viewport.keyboardVisible) return { viewport, reachesBottom: false, extended: false };
  // A browser tab's bottom edge belongs to the browser; trust what it reports.
  if (!screen.standalone || screen.screenWidth <= 0 || screen.screenHeight <= 0) {
    return { viewport, reachesBottom: true, extended: false };
  }

  // iOS reports the screen in portrait whatever the orientation.
  const landscape = viewport.visibleWidth > viewport.visibleHeight;
  const long = Math.max(screen.screenWidth, screen.screenHeight);
  const short = Math.min(screen.screenWidth, screen.screenHeight);
  const screenWidth = landscape ? long : short;
  const screenHeight = landscape ? short : long;
  // A windowed app (iPad split view, Stage Manager) does not own the screen.
  if (Math.abs(viewport.visibleWidth - screenWidth) > SLACK_PX) return { viewport, reachesBottom: true, extended: false };

  const gap = screenHeight - (viewport.offsetTop + viewport.visibleHeight);
  if (gap <= SLACK_PX) return { viewport, reachesBottom: true, extended: false };
  if (screen.safeTop > 0 && Math.abs(gap - screen.safeTop) <= SLACK_PX) {
    const height = screenHeight - viewport.offsetTop;
    return {
      viewport: { ...viewport, visibleHeight: height, layoutHeight: Math.max(viewport.layoutHeight, screenHeight) },
      reachesBottom: true,
      extended: true,
    };
  }
  return { viewport, reachesBottom: false, extended: false };
}

/** The platform facts {@link correctViewport} decides from. Impure: reads the window. */
export function readScreenFacts(): ScreenFacts {
  if (typeof window === "undefined") return { standalone: false, screenWidth: 0, screenHeight: 0, safeTop: 0 };
  const standalone = (navigator as Navigator & { standalone?: boolean }).standalone === true
    || (typeof window.matchMedia === "function" && window.matchMedia("(display-mode: standalone)").matches);
  if (!standalone) return { standalone, screenWidth: 0, screenHeight: 0, safeTop: 0 };
  return {
    standalone,
    screenWidth: window.screen.width,
    screenHeight: window.screen.height,
    safeTop: readSafeAreaInsets().top,
  };
}
