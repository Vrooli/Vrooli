/**
 * @libraryId react-component-library:BottomSheet
 * @displayName BottomSheet
 * @description Shared overlay presentation for BottomSheet.
 * @version 1.1.3
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { Icon } from "@vrooli/react-component-library/Icon/1.1.3";
import { IconButton } from "@vrooli/react-component-library/IconButton/2.0.1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.3.3";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
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
  /** `comfortable` (the default) pads the scroll region; `none` hands it to the caller. */
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

const classes = (...values: Array<string | undefined>) => values.filter(Boolean).join(" ");

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
  contentPadding = "comfortable",
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.bottom-sheet",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: BottomSheetProps) {
  useLibraryStyleSheet("bottom-sheet-1.1.3", bottomSheetStyles);
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
        data-rcl-bottom-sheet
        data-avoid-keyboard={avoidKeyboard || undefined}
        data-content-padding={contentPadding}
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
          aria-label={ariaLabel ?? (typeof title === "string" ? title : closeLabel)}
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
            <div data-testid={`${testId}.subheader`} className="rcl-bottom-sheet__subheader">
              {subheader}
            </div>
          ) : null}
          <div className={classes("rcl-bottom-sheet__content", contentClassName)}>{children}</div>
          {footer ? <footer className="rcl-bottom-sheet__footer">{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
