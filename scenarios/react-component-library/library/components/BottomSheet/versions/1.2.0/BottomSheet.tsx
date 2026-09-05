/**
 * @libraryId react-component-library:BottomSheet
 * @displayName BottomSheet
 * @version 1.2.0
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
export const bottomSheetStyles = `
[data-rcl-bottom-sheet] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); display: grid; align-items: end; pointer-events: none; }
[data-rcl-bottom-sheet][data-avoid-keyboard] { inset-block-end: var(--rcl-keyboard-inset, 0px); }

.rcl-bottom-sheet__backdrop { position: absolute; inset: 0; margin: 0; padding: 0; border: 0; background: var(--color-scrim, rgb(15 23 42 / .52)); pointer-events: auto; opacity: 1; transition: opacity var(--dur-quick) var(--ease-standard); }
[data-rcl-bottom-sheet][data-state="closed"] .rcl-bottom-sheet__backdrop { opacity: 0; }

.rcl-bottom-sheet__panel { position: relative; display: flex; flex-direction: column; inline-size: 100%; max-block-size: calc(var(--rcl-viewport-height, 100dvh) - var(--rcl-safe-top, 0px) - var(--overlay-drawer-top-gap, 32px)); overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-block-end: 0; border-radius: var(--radius-sheet) var(--radius-sheet) 0 0; background: var(--color-surface-raised); color: var(--color-foreground); box-shadow: var(--elev-modal); pointer-events: auto; transition: transform var(--dur-quick) var(--ease-standard); animation: rcl-bottom-sheet-enter var(--dur-moderate) var(--ease-enter); }
.rcl-bottom-sheet__panel[data-dragging="true"] { transition: none; will-change: transform; }
[data-rcl-bottom-sheet][data-state="closed"] .rcl-bottom-sheet__panel { transform: translateY(100%); animation: none; }
@keyframes rcl-bottom-sheet-enter { from { transform: translateY(100%); } }

.rcl-bottom-sheet__grabber { position: absolute; z-index: 1; inset-block-start: 0; inset-inline-start: 50%; translate: -50% 0; inline-size: min(60%, 12rem); min-block-size: var(--tap-target-min); display: grid; justify-items: center; align-content: start; padding: var(--space-2xs) 0 0; margin: 0; border: 0; background: transparent; color: inherit; touch-action: none; cursor: grab; }
.rcl-bottom-sheet__grabber[data-rcl-overlay-dragging="true"] { cursor: grabbing; }
.rcl-bottom-sheet__grabber > span { inline-size: var(--overlay-grabber-inline, 2.25rem); block-size: var(--overlay-grabber-block, .25rem); border-radius: var(--radius-pill); background: var(--color-border-strong, var(--color-border)); }

.rcl-bottom-sheet__header, .rcl-bottom-sheet__footer { flex: 0 0 auto; display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); padding: var(--space-sm) var(--space-md); }
.rcl-bottom-sheet__header { border-block-end: var(--border-hairline) solid var(--color-border); }
.rcl-bottom-sheet__footer { border-block-start: var(--border-hairline) solid var(--color-border); padding-block-end: calc(var(--space-sm) + var(--rcl-safe-bottom, 0px)); }
.rcl-bottom-sheet__header > *:first-child { min-inline-size: 0; }
.rcl-bottom-sheet__header > *:last-child { display: flex; flex: 0 0 auto; align-items: center; gap: var(--space-3xs); }
.rcl-bottom-sheet__header h2 { margin: 0; font: var(--text-heading); }

.rcl-bottom-sheet__subheader { flex: 0 0 auto; min-block-size: 0; border-block-end: var(--border-hairline) solid var(--color-border); }

.rcl-bottom-sheet__content { flex: 1 1 auto; min-block-size: 0; overflow: auto; overscroll-behavior: contain; padding-block-end: var(--rcl-safe-bottom, 0px); }
[data-rcl-bottom-sheet][data-content-padding="comfortable"] .rcl-bottom-sheet__content { padding: var(--space-md); padding-block-end: calc(var(--space-md) + var(--rcl-safe-bottom, 0px)); }
[data-rcl-bottom-sheet][data-has-footer] .rcl-bottom-sheet__content { padding-block-end: 0; }
[data-rcl-bottom-sheet][data-has-footer][data-content-padding="comfortable"] .rcl-bottom-sheet__content { padding-block-end: var(--space-md); }

.rcl-bottom-sheet__panel [data-icon] { flex: 0 0 auto; inline-size: var(--icon-size-md); block-size: var(--icon-size-md); }

@media (min-width: 48rem) {
  [data-rcl-bottom-sheet] { padding: var(--space-xs); padding-block-end: max(var(--space-xs), var(--rcl-safe-bottom, 0px)); }
  [data-rcl-bottom-sheet][data-avoid-keyboard] { padding-block-end: var(--space-xs); }
  .rcl-bottom-sheet__panel { inline-size: min(100%, 42rem); margin-inline: auto; border-block-end: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-sheet); }
  [data-rcl-bottom-sheet][data-state="closed"] .rcl-bottom-sheet__panel { transform: translateY(calc(100% + var(--space-xs))); }
  .rcl-bottom-sheet__footer { padding-block-end: var(--space-sm); }
  .rcl-bottom-sheet__content { padding-block-end: 0; }
  [data-rcl-bottom-sheet][data-content-padding="comfortable"] .rcl-bottom-sheet__content { padding: var(--space-md); }
}
`;
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
    dismiss: { escape: true, backdrop: true, swipe: "bottom" },
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
