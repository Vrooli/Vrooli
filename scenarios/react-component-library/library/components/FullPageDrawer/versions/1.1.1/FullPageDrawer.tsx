/**
 * @libraryId react-component-library:FullPageDrawer
 * @displayName FullPageDrawer
 * @description Shared overlay presentation for FullPageDrawer.
 * @version 1.1.1
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { Icon } from "@vrooli/react-component-library/Icon/1.1.3";
import { IconButton } from "@vrooli/react-component-library/IconButton/2.0.1";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1.1.0";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.3.1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { fullPageDrawerStyles } from "./styles";

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
   * `comfortable` (the default) pads the scroll region. `none` hands the full
   * inline size to the caller, which then owns its own gutters. Scrolling
   * stays with the drawer either way; a caller that needs a fixed band above
   * the scroll region uses `subheader` rather than nesting a second scroller.
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
  contentPadding = "comfortable",
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.full-page-drawer",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: FullPageDrawerProps) {
  useLibraryStyleSheet("full-page-drawer-1.1.0", fullPageDrawerStyles);
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
    dismiss: { escape: true, backdrop: true, swipe: showGrabber ? "down" : false },
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
