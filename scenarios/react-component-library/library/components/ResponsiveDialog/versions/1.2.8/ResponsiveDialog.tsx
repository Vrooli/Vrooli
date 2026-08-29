/**
 * @libraryId react-component-library:ResponsiveDialog
 * @displayName ResponsiveDialog
 * @description Shared overlay presentation for ResponsiveDialog.
 * @version 1.2.8
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { Icon } from "@vrooli/react-component-library/Icon/1.1.3";
import { IconButton } from "@vrooli/react-component-library/IconButton/3.1.1";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1.1.0";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.3.11";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { responsiveDialogStyles } from "./styles";

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
        {...overlay.rootProps}
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
              key="grabber"
              {...overlay.grabberProps}
              data-testid={`${testId}.grabber`}
              aria-label={grabberLabel ?? closeLabel}
              className="rcl-responsive-dialog__grabber"
            >
              <span aria-hidden />
            </button>
          ) : null}
          <header key="header">
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
            <div
              key="subheader"
              data-testid={`${testId}.subheader`}
              className="rcl-responsive-dialog__subheader"
            >
              {subheader}
            </div>
          ) : null}
          <div
            key="content"
            className={classes("rcl-responsive-dialog__content", contentClassName)}
          >
            {children}
          </div>
          {footer ? <footer key="footer">{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
