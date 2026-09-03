/**
 * @libraryId react-component-library:ResponsiveDialog
 * @displayName ResponsiveDialog
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
export const responsiveDialogStyles = `
[data-rcl-responsive-dialog] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); display: grid; align-items: end; pointer-events: none; }
[data-rcl-responsive-dialog][data-avoid-keyboard] { inset-block-end: var(--rcl-keyboard-inset, 0px); }

.rcl-responsive-dialog__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-responsive-dialog][data-state="closed"] .rcl-responsive-dialog__backdrop { opacity: 0; }

.rcl-responsive-dialog__panel { position: relative; display: flex; flex-direction: column; inline-size: 100%; max-block-size: calc(var(--rcl-viewport-height, 100dvh) - var(--rcl-safe-top, 0px) - var(--overlay-drawer-top-gap, 32px)); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-quick) var(--ease-standard); animation: rcl-responsive-dialog-enter-sheet var(--dur-moderate) var(--ease-enter); }
.rcl-responsive-dialog__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-responsive-dialog][data-state="closed"] .rcl-responsive-dialog__panel { transform: translateY(100%); animation: none; }
@keyframes rcl-responsive-dialog-enter-sheet { from { transform: translateY(100%); } }

.rcl-responsive-dialog__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-responsive-dialog__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-responsive-dialog__grabber > span { inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

.rcl-responsive-dialog__panel > header, .rcl-responsive-dialog__panel > footer { flex: 0 0 auto; display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); }
.rcl-responsive-dialog__panel > header { border-block-end: var(--border-hairline) solid var(--color-border); }
.rcl-responsive-dialog__panel > footer { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-end: calc(var(--space-sm) + var(--rcl-safe-bottom, 0px)); }
.rcl-responsive-dialog__panel > header > *:first-child { min-inline-size: 0; }
.rcl-responsive-dialog__panel > header > *:last-child { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3xs); }
.rcl-responsive-dialog__panel h2 { margin: 0; font: var(--text-heading); }

.rcl-responsive-dialog__subheader { flex: 0 0 auto; min-block-size: 0; border-block-end: var(--border-hairline) solid var(--color-border); }

.rcl-responsive-dialog__content { flex: 1 1 auto; min-block-size: 0; overflow: auto; overscroll-behavior: contain; padding-block-end: var(--rcl-safe-bottom, 0px); }
[data-rcl-responsive-dialog][data-content-padding="comfortable"] .rcl-responsive-dialog__content { padding: var(--space-md); padding-block-end: calc(var(--space-md) + var(--rcl-safe-bottom, 0px)); }
[data-rcl-responsive-dialog][data-has-footer] .rcl-responsive-dialog__content { padding-block-end: 0; }
[data-rcl-responsive-dialog][data-has-footer][data-content-padding="comfortable"] .rcl-responsive-dialog__content { padding-block-end: var(--space-md); }

.rcl-responsive-dialog__panel [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }

@media (min-width: 48rem) {
  [data-rcl-responsive-dialog] { place-items: center; padding: var(--space-md); }
  [data-rcl-responsive-dialog][data-avoid-keyboard] { inset-block-end: 0; }
  .rcl-responsive-dialog__panel { max-block-size: calc(var(--rcl-viewport-height, 100dvh) - (var(--space-lg) * 2)); border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); animation-name: rcl-responsive-dialog-enter-dialog; }
  [data-rcl-responsive-dialog][data-state="closed"] .rcl-responsive-dialog__panel { transform: translateY(var(--space-2xs)); opacity: 0; }
  [data-rcl-responsive-dialog][data-size="sm"] .rcl-responsive-dialog__panel { inline-size: var(--overlay-dialog-sm); }
  [data-rcl-responsive-dialog][data-size="md"] .rcl-responsive-dialog__panel { inline-size: var(--overlay-dialog-md); }
  [data-rcl-responsive-dialog][data-size="lg"] .rcl-responsive-dialog__panel { inline-size: var(--overlay-dialog-lg); }
  .rcl-responsive-dialog__panel > header, .rcl-responsive-dialog__panel > footer { padding: var(--space-md); }
  .rcl-responsive-dialog__panel > footer { padding-block-end: var(--space-md); }
  .rcl-responsive-dialog__content { padding-block-end: 0; }
  [data-rcl-responsive-dialog][data-content-padding="comfortable"] .rcl-responsive-dialog__content { padding: var(--space-md); }
}
@keyframes rcl-responsive-dialog-enter-dialog { from { transform: translateY(var(--space-2xs)); opacity: 0; } }
`;
/** The inline size the centered presentation takes at and above the medium breakpoint. */
export type ResponsiveDialogSize = "sm" | "md" | "lg";

/** Padding applied to the dialog's own scroll region. */
export type ResponsiveDialogContentPadding = "comfortable" | "none";

/** Which affordance dismisses the dialog without leaving the surface. */
export type ResponsiveDialogAffordance = "auto" | "grabber" | "close";

/** Inputs to {@link ResponsiveDialog}. */
export interface ResponsiveDialogProps {
  open: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title: ReactNode;
  ariaLabel?: string;
  children: ReactNode;
  /** A full-bleed, non-scrolling band between the header and the scroll region. */
  subheader?: ReactNode;
  footer?: ReactNode;
  headerActions?: ReactNode;
  closeLabel: string;
  /** Accessible name for the grabber shown on the sheet presentation. Defaults to `closeLabel`. */
  grabberLabel?: string;
  /**
   * `auto` (the default) resolves to the grabber on the sheet presentation and
   * the close button on the centred one. A sheet that is dismissed by pushing
   * it back down does not also need a button that says so, and offering both
   * gives one surface three controls with the same accessible name.
   */
  dismissAffordance?: ResponsiveDialogAffordance;
  size?: ResponsiveDialogSize;
  /**
   * `none` (the default) hands the full content box to the caller, which then
   * owns its own gutters. This is the default because most content already
   * carries its own padding — a list with its own row insets, a split layout,
   * an editor sized to the box — and a surface gutter on top of that reads as
   * a double gutter, not as breathing room. `comfortable` opts back in to a
   * uniform pad for the plain-prose case. Scrolling stays with the dialog
   * either way; a caller that needs a fixed band above the scroll region uses
   * `subheader` rather than nesting a second scroller.
   */
  contentPadding?: ResponsiveDialogContentPadding;
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
 * A bounded form or decision: a bottom sheet below the medium breakpoint and a
 * centered, token-sized card at or above it, keeping one mounted child subtree
 * across the change.
 *
 * The sheet presentation is dismissed by dragging its grabber downward. Before
 * 1.1.0 the grabber was an `aria-hidden` bar with no handler behind it, so the
 * gesture the overlay contract advertised did not exist; it is now the shared
 * affordance from `useOverlaySurface`, reachable by pointer and by keyboard.
 */
export function ResponsiveDialog({
  open,
  onOpenChange,
  onClose,
  title,
  ariaLabel,
  children,
  subheader,
  footer,
  headerActions,
  closeLabel,
  grabberLabel,
  dismissAffordance = "auto",
  size = "md",
  contentPadding = "none",
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.responsive-dialog",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: ResponsiveDialogProps) {
  useLibraryStyleSheet("responsive-dialog-1.1.7", responsiveDialogStyles);
  const desktop = useBreakpoint("md");
  const showGrabber = dismissAffordance === "grabber" || (dismissAffordance === "auto" && !desktop);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: desktop ? "dialog" : "sheet",
    dismiss: { escape: true, backdrop: true, swipe: showGrabber ? "down" : false },
    initialFocusRef,
    returnFocusRef,
  });
  if (!overlay.present) return null;
  return (
    <Portal>
      <div
        data-rcl-responsive-dialog
        className={className}
        data-state={overlay.state}
        data-presentation={desktop ? "dialog" : "sheet"}
        data-size={size}
        data-content-padding={contentPadding}
        data-has-footer={footer ? "" : undefined}
        data-avoid-keyboard={avoidKeyboard || undefined}
      >
        <button
          type="button"
          data-testid={`${testId}.backdrop`}
          aria-label={closeLabel}
          className={classes("rcl-responsive-dialog__backdrop", backdropClassName)}
          {...overlay.backdropProps}
        />
        <section
          {...overlay.surfaceProps}
          data-testid={testId}
          role="dialog"
          aria-modal="true"
          aria-label={ariaLabel ?? (typeof title === "string" ? title : closeLabel)}
          className={classes("rcl-responsive-dialog__panel", panelClassName)}
        >
          {showGrabber ? (
            <button
              {...overlay.grabberProps}
              data-testid={`${testId}.grabber`}
              aria-label={grabberLabel ?? closeLabel}
              className="rcl-responsive-dialog__grabber"
            >
              <span aria-hidden />
            </button>
          ) : null}
          <header>
            <div>
              <h2>{title}</h2>
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
            <div data-testid={`${testId}.subheader`} className="rcl-responsive-dialog__subheader">
              {subheader}
            </div>
          ) : null}
          <div className={classes("rcl-responsive-dialog__content", contentClassName)}>
            {children}
          </div>
          {footer ? <footer>{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
