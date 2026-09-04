/**
 * @libraryId react-component-library:FullPageDrawer
 * @displayName FullPageDrawer
 * @version 1.2.0
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
export const fullPageDrawerStyles = `
[data-rcl-full-page-drawer] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); pointer-events: none; }

.rcl-full-page-drawer__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__backdrop { opacity: 0; }

.rcl-full-page-drawer__panel { position: absolute; inset-inline: 0; inset-block-start: calc(var(--rcl-safe-top, 0px) + var(--overlay-drawer-top-gap, 32px)); inset-block-end: 0; display: flex; flex-direction: column; min-block-size: 0; overflow: hidden; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-moderate) var(--ease-standard); animation: rcl-full-page-drawer-enter var(--dur-moderate) var(--ease-enter); }
.rcl-full-page-drawer__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-full-page-drawer][data-state="closed"] .rcl-full-page-drawer__panel { transform: translateY(100%); animation: none; }
[data-rcl-full-page-drawer][data-avoid-keyboard] .rcl-full-page-drawer__panel { inset-block-end: var(--rcl-keyboard-inset, 0px); }
@keyframes rcl-full-page-drawer-enter { from { transform: translateY(100%); } }

.rcl-full-page-drawer__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-full-page-drawer__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-full-page-drawer__grabber > span { inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

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
  [data-rcl-full-page-drawer][data-avoid-keyboard] .rcl-full-page-drawer__panel { inset-block-end: max(var(--space-md), var(--rcl-keyboard-inset, 0px)); }
  .rcl-full-page-drawer__panel > header, .rcl-full-page-drawer__panel > footer { padding: var(--space-md); }
  .rcl-full-page-drawer__panel > footer { padding-block-end: var(--space-md); }
  .rcl-full-page-drawer__content { padding-block-end: 0; }
  [data-rcl-full-page-drawer][data-content-padding="comfortable"] .rcl-full-page-drawer__content { padding: var(--space-md); }
}
@keyframes rcl-full-page-drawer-enter-inset { from { transform: translateY(var(--space-2xs)); opacity: 0; } }
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
  useLibraryStyleSheet("full-page-drawer-1.1.7", fullPageDrawerStyles);
  const desktop = useBreakpoint("md");
  const showGrabber = dismissAffordance === "grabber" || (dismissAffordance === "auto" && !desktop);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: "drawer",
    dismiss: { escape: true, backdrop: true, swipe: showGrabber ? "bottom" : false },
    initialFocusRef,
    returnFocusRef,
  });
  if (!overlay.present) return null;
  return (
    <Portal>
      <div
        data-rcl-full-page-drawer
        className={className}
        data-state={overlay.state}
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
              {...overlay.grabberProps}
              data-testid={`${testId}.grabber`}
              aria-label={grabberLabel ?? closeLabel}
              className="rcl-full-page-drawer__grabber"
            >
              <span aria-hidden />
            </button>
          ) : null}
          <header>
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
            <div data-testid={`${testId}.subheader`} className="rcl-full-page-drawer__subheader">
              {subheader}
            </div>
          ) : null}
          <div className={classes("rcl-full-page-drawer__content", contentClassName)}>
            {children}
          </div>
          {footer ? <footer>{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
