/**
 * @libraryId react-component-library:BottomSheet
 * @displayName BottomSheet
 * @description Shared overlay presentation for BottomSheet.
 * @version 1.2.7
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { bottomSheetStyles } from "./styles";

/** Padding applied to the sheet's own scroll region. */
export type BottomSheetContentPadding = "comfortable" | "none";

/** Inputs to {@link BottomSheet}. */
export interface BottomSheetProps {
  open: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title: ReactNode;
  ariaLabel?: string;
  children: ReactNode;
  headerActions?: ReactNode;
  /** A full-bleed, non-scrolling band between the header and the scroll region. */
  subheader?: ReactNode;
  footer?: ReactNode;
  closeLabel: string;
  /** Accessible name for the grabber. Defaults to `closeLabel`. */
  grabberLabel?: string;
  /** Also render a close button beside the header actions. */
  showCloseButton?: boolean;
  /**
   * `none` (the default) hands the full content box to the caller, which then
   * owns its own gutters. This is the default because most content already
   * carries its own padding — a list with its own row insets, a split layout,
   * an editor sized to the box — and a surface gutter on top of that reads as
   * a double gutter, not as breathing room. `comfortable` opts back in to a
   * uniform pad for the plain-prose case. Scrolling stays with the sheet
   * either way; a caller that needs a fixed band above the scroll region uses
   * `subheader` rather than nesting a second scroller.
   */
  contentPadding?: BottomSheetContentPadding;
  avoidKeyboard?: boolean;
  initialFocusRef?: RefObject<HTMLElement | null>;
  returnFocusRef?: RefObject<HTMLElement | null>;
  testId?: string;
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
}

const classes = (...values: Array<string | undefined>) =>
  values.filter(Boolean).join(" ");

/**
 * A sheet anchored to the bottom edge, sized to its content up to the usable
 * viewport, and dismissed by dragging its grabber downward.
 *
 * Its height and bottom inset come from the `BaseStyles` host viewport
 * contract rather than `100dvh` and `env(safe-area-inset-bottom)`, so an
 * application that manages its own scrolling or keyboard handling gets a sheet
 * that agrees with the rest of its chrome.
 */
export function BottomSheet({
  open,
  onOpenChange,
  onClose,
  title,
  ariaLabel,
  children,
  headerActions,
  subheader,
  footer,
  closeLabel,
  grabberLabel,
  showCloseButton = false,
  contentPadding = "none",
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.bottom-sheet",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: BottomSheetProps) {
  useLibraryStyleSheet("bottom-sheet-1.1.7", bottomSheetStyles);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: "sheet",
    dismiss: { escape: true, backdrop: true, swipe: "down" },
    initialFocusRef,
    returnFocusRef,
  });
  if (!overlay.present) return null;
  return (
    <Portal>
      <div
        {...overlay.rootProps}
        data-rcl-bottom-sheet
        data-avoid-keyboard={avoidKeyboard || undefined}
        data-content-padding={contentPadding}
        data-has-footer={footer ? "" : undefined}
        className={className}
        data-state={overlay.state}
      >
        <button
          type="button"
          data-testid={`${testId}.backdrop`}
          aria-label={closeLabel}
          className={classes("rcl-bottom-sheet__backdrop", backdropClassName)}
          {...overlay.backdropProps}
        />
        <section
          {...overlay.surfaceProps}
          data-testid={testId}
          role="dialog"
          aria-modal="true"
          aria-label={
            ariaLabel ?? (typeof title === "string" ? title : closeLabel)
          }
          className={classes("rcl-bottom-sheet__panel", panelClassName)}
        >
          <button
            {...overlay.grabberProps}
            data-testid={`${testId}.grabber`}
            aria-label={grabberLabel ?? closeLabel}
            className="rcl-bottom-sheet__grabber"
          >
            <span aria-hidden />
          </button>
          <header className="rcl-bottom-sheet__header">
            <div>
              <h2>{title}</h2>
            </div>
            <div>
              {headerActions}
              {showCloseButton ? (
                <IconButton
                  data-testid={`${testId}.close`}
                  aria-label={closeLabel}
                  size="lg"
                  onClick={overlay.close}
                >
                  <Icon name="close" />
                </IconButton>
              ) : null}
            </div>
          </header>
          {subheader ? (
            <div
              data-testid={`${testId}.subheader`}
              className="rcl-bottom-sheet__subheader"
            >
              {subheader}
            </div>
          ) : null}
          <div
            className={classes("rcl-bottom-sheet__content", contentClassName)}
          >
            {children}
          </div>
          {footer ? (
            <footer className="rcl-bottom-sheet__footer">{footer}</footer>
          ) : null}
        </section>
      </div>
    </Portal>
  );
}
