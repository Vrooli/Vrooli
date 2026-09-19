import { useLayoutEffect, type RefObject } from "react";
import type { FollowerFrame } from "./useFollowerPresentation";

/**
 * Duration of the follower reflow, shared by the terminal host and the device
 * frame drawn around it.
 *
 * Both surfaces must animate on the same property set for the same time or the
 * bezel and its contents visibly slide apart. They previously carried separate
 * literals, and had already drifted: one transitioned `transform` and the other
 * did not.
 *
 * Reduced motion is honoured globally in `styles.css`, whose `!important`
 * duration override outranks these inline declarations.
 */
export const FOLLOWER_TRANSITION_MS = 240;

/**
 * Paint order inside a follower pane. The enclosure is opaque, so it must sit
 * *below* the terminal surface: painting it above is what hid the terminal
 * behind the device. Values come from the z-scale in `tailwind.config.ts`.
 */
export const FOLLOWER_ENCLOSURE_Z = 1;
export const FOLLOWER_SCREEN_Z = 2;

export const FOLLOWER_TRANSITION = [
  "left",
  "top",
  "width",
  "height",
  "transform",
].map((property) => `${property} ${String(FOLLOWER_TRANSITION_MS)}ms ease`).join(", ");

/** The xterm surface this hook drives. */
interface LayoutTerminal {
  options: { fontSize?: number };
  cols: number;
  rows: number;
  resize: (cols: number, rows: number) => void;
}

/**
 * Apply a computed follower presentation to the live xterm surface.
 *
 * This is the imperative half of follower presentation; `useFollowerPresentation`
 * is the pure half that decides the geometry. They are separate functions but
 * one concern, and keeping them in one place gives `terminal.options.fontSize`
 * a single owner on the follower path — it previously had two, one here and one
 * in the lifecycle hook, with no stated protocol between them.
 */
export function useFollowerViewportLayout(options: {
  frame: FollowerFrame | null;
  terminal: LayoutTerminal | null;
  hostRef: RefObject<HTMLDivElement | null>;
  /**
   * Wrapper positioned over the device's screen opening. It carries the
   * rounded clip, so the terminal is bounded by the screen rather than
   * painting over the bezel, and it lifts the terminal above the enclosure
   * drawn behind it.
   */
  screenRef: RefObject<HTMLDivElement | null>;
  /** Font size to restore when this pane is not following. */
  paneFontSize: number;
  /** Re-fit the terminal to its container, used on the leader path. */
  refit: () => void;
  /** Fit xterm to the host box after the follower styles are applied. */
  fitToHost: () => void;
}): void {
  const { frame, terminal, hostRef, screenRef, paneFontSize, refit, fitToHost } = options;
  useLayoutEffect(() => {
    const host = hostRef.current;
    const screen = screenRef.current;
    if (!host || !terminal) return;
    if (!frame) {
      if (screen) {
        screen.style.position = "";
        screen.style.left = "";
        screen.style.top = "";
        screen.style.width = "";
        screen.style.height = "";
        screen.style.borderRadius = "";
        screen.style.overflow = "";
        screen.style.transition = "";
        screen.style.zIndex = "";
      }
      host.style.position = "";
      host.style.left = "";
      host.style.top = "";
      host.style.width = "";
      host.style.height = "";
      host.style.transformOrigin = "";
      host.style.transform = "";
      host.style.transition = "";
      terminal.options.fontSize = paneFontSize;
      refit();
      return;
    }
    const { screenRect, apertureRect } = frame;
    if (screen) {
      // The clip box is the screen opening, not the grid: the grid is
      // letterboxed inside it and the leftover is the device's own screen.
      screen.style.position = "absolute";
      screen.style.left = `${String(apertureRect.x)}px`;
      screen.style.top = `${String(apertureRect.y)}px`;
      screen.style.width = `${String(apertureRect.width)}px`;
      screen.style.height = `${String(apertureRect.height)}px`;
      screen.style.borderRadius = `${String(apertureRect.radius)}px`;
      screen.style.overflow = "hidden";
      screen.style.transition = FOLLOWER_TRANSITION;
      screen.style.zIndex = String(FOLLOWER_SCREEN_Z);
    }
    // The host is positioned relative to the clip box when one is present, so
    // its offsets are aperture-relative rather than pane-relative.
    const originX = screen ? apertureRect.x : 0;
    const originY = screen ? apertureRect.y : 0;
    host.style.position = "absolute";
    host.style.left = `${String(screenRect.x - originX)}px`;
    host.style.top = `${String(screenRect.y - originY)}px`;
    host.style.width = `${String(screenRect.width / screenRect.scale)}px`;
    host.style.height = `${String(screenRect.height / screenRect.scale)}px`;
    host.style.transformOrigin = "top left";
    host.style.transform = screenRect.scale < 1 ? `scale(${String(screenRect.scale)})` : "";
    host.style.transition = FOLLOWER_TRANSITION;
    terminal.options.fontSize = screenRect.fontSize;
    fitToHost();
    // The follower mirrors the leader's grid exactly; fitting to the host is
    // only how the font lands, not how the dimensions are chosen.
    if (terminal.cols !== frame.cols || terminal.rows !== frame.rows) {
      terminal.resize(frame.cols, frame.rows);
    }
  }, [frame, hostRef, screenRef, paneFontSize, refit, fitToHost, terminal]);
}
