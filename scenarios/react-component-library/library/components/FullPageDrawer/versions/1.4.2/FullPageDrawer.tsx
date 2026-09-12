/**
 * @libraryId react-component-library:FullPageDrawer
 * @displayName FullPageDrawer
 * @description The drawer presentation occupying the full application viewport while retaining source context, navigation continuity, safe areas, and route-compatible dismissal.
 * @version 1.4.2
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useSwipeGesture } from "@vrooli/react-component-library/useSwipeGesture/2";
import { resolveGestureFeel } from "@vrooli/react-component-library/GestureTokens/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
export const fullPageDrawerStyles = `
[data-rcl-full-page-drawer] { position: fixed; inset: 0; z-index: var(--layer-modal); pointer-events: none; }
[data-rcl-full-page-drawer][data-avoid-keyboard] { inset: auto; inset-inline-start: var(--rcl-viewport-offset-left, 0px); inset-block-start: var(--rcl-viewport-offset-top, 0px); inline-size: var(--rcl-viewport-width, 100vw); block-size: var(--rcl-viewport-height, 100dvh); }

.rcl-full-page-drawer__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__backdrop { opacity: 0; }

.rcl-full-page-drawer__panel { position: absolute; inset-inline: 0; inset-block-start: calc(var(--rcl-safe-top, 0px) + var(--overlay-drawer-top-gap)); inset-block-end: 0; display: flex; flex-direction: column; min-block-size: 0; overflow: hidden; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: inset-block-start var(--dur-moderate) var(--ease-standard), transform var(--dur-moderate) var(--ease-standard); animation: rcl-full-page-drawer-enter var(--dur-moderate) var(--ease-enter); }
[data-rcl-full-page-drawer][data-expanded="true"] .rcl-full-page-drawer__panel { inset-block-start: var(--rcl-safe-top, 0px); }
.rcl-full-page-drawer__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__panel { transform: translateY(100%); animation: none; }
@keyframes rcl-full-page-drawer-enter { from { transform: translateY(100%); } }

.rcl-full-page-drawer__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-full-page-drawer__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-full-page-drawer__grabber > span { inline-size: var(--overlay-grabber-inline); block-size: var(--overlay-grabber-block); border-radius: var(--radius-pill); background: var(--color-border-strong); }

.rcl-full-page-drawer__panel > header, .rcl-full-page-drawer__panel > footer { flex: 0 0 auto; display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); }
.rcl-full-page-drawer__panel > header { border-block-end: var(--border-hairline) solid var(--color-border); }
.rcl-full-page-drawer__panel > footer { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-end: calc(var(--space-sm) + var(--rcl-safe-bottom, 0px)); }
.rcl-full-page-drawer__panel > header > *:first-child { min-inline-size: 0; }
.rcl-full-page-drawer__panel > header > *:last-child { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3xs); }
.rcl-full-page-drawer__panel h2 { margin: 0; font: var(--text-heading); }

.rcl-full-page-drawer__subheader { flex: 0 0 auto; min-block-size: 0; border-block-end: var(--border-hairline) solid var(--color-border); }

.rcl-full-page-drawer__content { flex: 1 1 auto; min-block-size: 0; overflow: auto; overscroll-behavior: contain; padding-block-end: var(--rcl-safe-bottom, 0px); }
[data-rcl-full-page-drawer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding: var(--space-md); padding-block-end: calc(var(--space-md) + var(--rcl-safe-bottom, 0px)); }
[data-rcl-full-page-drawer][data-has-footer] .rcl-full-page-drawer__content { padding-block-end: 0; }
[data-rcl-full-page-drawer][data-has-footer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding-block-end: var(--space-md); }

@media (min-width: 48rem) {
  .rcl-full-page-drawer__panel { inset: var(--space-md); border-radius: var(--radius-panel); animation-name: rcl-full-page-drawer-enter-inset; }
  [data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__panel { transform: translateY(var(--space-2xs)); opacity: 0; }
  .rcl-full-page-drawer__panel > header, .rcl-full-page-drawer__panel > footer { padding: var(--space-md); }
  .rcl-full-page-drawer__panel > footer { padding-block-end: var(--space-md); }
  .rcl-full-page-drawer__content { padding-block-end: 0; }
  [data-rcl-full-page-drawer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding: var(--space-md); }
}
@keyframes rcl-full-page-drawer-enter-inset { from { transform: translateY(var(--space-2xs)); opacity: 0; } }
/* The sheet owns the strip beneath it. An installed iOS 26 app paints nothing
   of its own below the viewport it reports (see useViewportEnvironment 1.1.0);
   that strip shows the page canvas, so while the sheet is open the canvas takes
   the sheet's surface and reads as the sheet continuing to the edge. WebKit
   paints it from the <html> background with the <body> background blended over
   it, so both are coloured: <html> alone is hidden by any opaque body (1.4.1). */
@media not all and (min-width: 48rem) {
  html:has([data-rcl-full-page-drawer][data-state="open"]),
  html:has([data-rcl-full-page-drawer][data-state="open"]) body { background-color: var(--color-surface-raised); }
}
.rcl-full-page-drawer__panel [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }
`;
/** Which affordance dismisses the drawer without leaving the surface. */
export type FullPageDrawerAffordance = "auto" | "grabber" | "close";

/** Padding applied to the drawer's own scroll region. */
export type FullPageDrawerContentPadding = "comfortable" | "none";

/** Inputs to {@link FullPageDrawer}. */
export interface FullPageDrawerProps {
  open: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title: ReactNode;
  ariaLabel?: string;
  children: ReactNode;
  headerActions?: ReactNode;
  headerExtra?: ReactNode;
  /**
   * A full-bleed band between the header and the scroll region: tabs, a
   * filter row, a search field. It does not scroll and it is not padded, so a
   * tab strip reaches both edges instead of sitting inside the content gutter.
   */
  subheader?: ReactNode;
  footer?: ReactNode;
  closeLabel: string;
  /**
   * Accessible name for the drag affordance. Defaults to `closeLabel`, which
   * is accurate — activating it dismisses the drawer — but a consumer that
   * wants to name the gesture may override it.
   */
  grabberLabel?: string;
  /**
   * `auto` (the default) resolves to the grabber below the medium breakpoint
   * and the close button at or above it: a sheet that rises from the bottom
   * edge is dismissed by pushing it back down, while an inset card is
   * dismissed by a button. `grabber` and `close` pin the choice.
   */
  dismissAffordance?: FullPageDrawerAffordance;
  /** Allow the mobile sheet to rise to the top edge with an upward grabber swipe. */
  expandable?: boolean;
  /**
   * Called when the sheet rises to the top edge and when it settles back, so
   * a caller can spend the room it gains — a header that becomes a search
   * field, a toolbar that moves up into the header. Not called on open: the
   * sheet always opens settled.
   */
  onExpandedChange?: (expanded: boolean) => void;
  /**
   * `none` (the default) hands the full content box to the caller, which then
   * owns its own gutters. This is the default because most content already
   * carries its own padding — a list with its own row insets, a split layout,
   * an editor sized to the box — and a surface gutter on top of that reads as
   * a double gutter, not as breathing room. `comfortable` opts back in to a
   * uniform pad for the plain-prose case. Scrolling stays with the drawer
   * either way; a caller that needs a fixed band above the scroll region uses
   * `subheader` rather than nesting a second scroller.
   */
  contentPadding?: FullPageDrawerContentPadding;
  avoidKeyboard?: boolean;
  initialFocusRef?: RefObject<HTMLElement | null>;
  returnFocusRef?: RefObject<HTMLElement | null>;
  testId?: string;
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
}

const classes = (...values: Array<string | undefined>) => values.filter(Boolean).join(" ");

/**
 * The page-scale overlay: a task that temporarily replaces the workspace while
 * keeping the route and source context intact.
 *
 * Below the medium breakpoint it is a sheet flush with the bottom edge, opened
 * by rising from it and dismissed by a drag on its grabber. The surface is
 * deliberately *not* inset from the bottom by the safe area — a surface that
 * floats above the edge it slid in from reads as a rendering fault, and the
 * inset is only ever right when the library and the host agree about where the
 * viewport ends. The safe area and the keyboard are cleared as padding inside
 * the scroll region and the footer, through the host viewport contract
 * (`--rcl-safe-*`, `--rcl-keyboard-inset`) that `BaseStyles` publishes.
 *
 * At and above the medium breakpoint it is an inset card with a close button.
 */
export function FullPageDrawer({
  open,
  onOpenChange,
  onClose,
  title,
  ariaLabel,
  children,
  headerActions,
  headerExtra,
  subheader,
  footer,
  closeLabel,
  grabberLabel,
  dismissAffordance = "auto",
  expandable = true,
  onExpandedChange,
  contentPadding = "none",
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.full-page-drawer",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: FullPageDrawerProps) {
  useLibraryStyleSheet("react-component-library:FullPageDrawer", "1.3.0", fullPageDrawerStyles);
  const desktop = useBreakpoint("md");
  const showGrabber = dismissAffordance === "grabber" || (dismissAffordance === "auto" && !desktop);
  const [expanded, setExpanded] = useState(false);
  const gestureFeel = resolveGestureFeel();
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: "drawer",
    dismiss: { escape: true, backdrop: true, swipe: showGrabber && !expanded ? "bottom" : false },
    initialFocusRef,
    returnFocusRef,
  });
  const expandSwipe = useSwipeGesture({
    direction: "top",
    stages: [gestureFeel.dismissThreshold],
    releaseMode: "commit",
    disabled: !showGrabber || !expandable || expanded,
    onRelease: (release) => {
      if (release.outcome === "commit") setExpanded(true);
    },
  });
  const collapseSwipe = useSwipeGesture({
    direction: "bottom",
    stages: [gestureFeel.axisSlop * 2, gestureFeel.dismissThreshold],
    releaseMode: "commit",
    disabled: !showGrabber || !expandable || !expanded,
    onMove: ({ offset }) => {
      const surface = overlay.surfaceRef.current;
      if (!surface) return;
      surface.style.transform = offset > 0 ? `translateY(${String(offset)}px)` : "";
      if (offset > 0) surface.setAttribute("data-dragging", "true");
    },
    onRelease: (release) => {
      const surface = overlay.surfaceRef.current;
      if (surface) {
        surface.style.transform = "";
        surface.removeAttribute("data-dragging");
      }
      if (release.outcome !== "commit") return;
      if (release.stage >= 2) {
        overlay.close();
        return;
      }
      setExpanded(false);
    },
  });
  useEffect(() => {
    if (!open) setExpanded(false);
  }, [open]);
  // Report changes only: the ref holds the last value told to the caller, so
  // the first render (always settled) and a re-render that did not move the
  // sheet say nothing.
  const onExpandedChangeRef = useRef(onExpandedChange);
  onExpandedChangeRef.current = onExpandedChange;
  const reportedExpandedRef = useRef(false);
  useEffect(() => {
    if (reportedExpandedRef.current === expanded) return;
    reportedExpandedRef.current = expanded;
    onExpandedChangeRef.current?.(expanded);
  }, [expanded]);
  if (!overlay.present) return null;
  return (
    <Portal>
      <div
        {...overlay.rootProps}
        data-rcl-full-page-drawer
        className={className}
        data-state={overlay.state}
        data-expanded={expanded ? "true" : "false"}
        data-avoid-keyboard={avoidKeyboard || undefined}
        data-content-padding={contentPadding}
        data-has-footer={footer ? "" : undefined}
      >
        <button
          type="button"
          data-testid={`${testId}.backdrop`}
          aria-label={closeLabel}
          className={classes("rcl-full-page-drawer__backdrop", backdropClassName)}
          {...overlay.backdropProps}
        />
        <section
          {...overlay.surfaceProps}
          data-testid={testId}
          role="dialog"
          aria-modal="true"
          aria-label={ariaLabel ?? (typeof title === "string" ? title : closeLabel)}
          className={classes("rcl-full-page-drawer__panel", panelClassName)}
        >
          {showGrabber ? (
            <button
              key="grabber"
              {...overlay.grabberProps}
              onPointerDown={(event) => {
                if (expanded) collapseSwipe.onPointerDown(event);
                else expandSwipe.onPointerDown(event);
                overlay.grabberProps.onPointerDown?.(event);
              }}
              data-testid={`${testId}.grabber`}
              aria-label={grabberLabel ?? closeLabel}
              className="rcl-full-page-drawer__grabber"
            >
              <span aria-hidden />
            </button>
          ) : null}
          <header key="header">
            <div>
              <h2>{title}</h2>
              {headerExtra}
            </div>
            <div>
              {headerActions}
              {showGrabber ? null : (
                <IconButton
                  data-testid={`${testId}.close`}
                  aria-label={closeLabel}
                  size="lg"
                  onClick={overlay.close}
                >
                  <Icon name="close" />
                </IconButton>
              )}
            </div>
          </header>
          {subheader ? (
            <div
              key="subheader"
              data-testid={`${testId}.subheader`}
              className="rcl-full-page-drawer__subheader"
            >
              {subheader}
            </div>
          ) : null}
          <div key="content" className={classes("rcl-full-page-drawer__content", contentClassName)}>
            {children}
          </div>
          {footer ? <footer key="footer">{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
