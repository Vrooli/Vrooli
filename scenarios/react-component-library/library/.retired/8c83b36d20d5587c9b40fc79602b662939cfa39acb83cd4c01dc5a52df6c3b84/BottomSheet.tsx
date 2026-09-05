/**
 * @libraryId react-component-library:BottomSheet
 * @displayName BottomSheet
 * @description Shared overlay presentation for BottomSheet.
 * @version 1.0.4
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import {
  useState,
  type CSSProperties,
  type ReactNode,
  type RefObject,
} from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.1.1";
import { useSwipe } from "@vrooli/react-component-library/useSwipe/2.0.1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { bottomSheetStyles } from "./styles";
export interface BottomSheetProps {
  open: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title: ReactNode;
  ariaLabel?: string;
  children: ReactNode;
  headerActions?: ReactNode;
  footer?: ReactNode;
  closeLabel: string;
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
export function BottomSheet({
  open,
  onOpenChange,
  onClose,
  title,
  ariaLabel,
  children,
  headerActions,
  footer,
  closeLabel,
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.bottom-sheet",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: BottomSheetProps) {
  useLibraryStyleSheet("bottom-sheet-1.0.0", bottomSheetStyles);
  const [progress, setProgress] = useState(0);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: "sheet",
    initialFocusRef,
    returnFocusRef,
  });
  const swipe = useSwipe({
    direction: "down",
    threshold: 96,
    velocity: 0.5,
    onProgress: setProgress,
    onCommit: overlay.close,
    onCancel: () => setProgress(0),
  });
  if (!overlay.present) return null;
  return (
    <Portal>
      <div
        data-rcl-bottom-sheet
        data-avoid-keyboard={avoidKeyboard || undefined}
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
          ref={overlay.surfaceRef}
          data-testid={testId}
          role="dialog"
          aria-modal="true"
          aria-label={
            ariaLabel ?? (typeof title === "string" ? title : closeLabel)
          }
          className={classes("rcl-bottom-sheet__panel", panelClassName)}
          style={{ "--rcl-sheet-progress": progress } as CSSProperties}
        >
          <button
            type="button"
            className="rcl-bottom-sheet__handle"
            aria-label={closeLabel}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                overlay.close();
              }
            }}
            {...swipe}
          >
            <span aria-hidden />
          </button>
          <header className="rcl-bottom-sheet__header">
            <h2>{title}</h2>
            <div>
              {headerActions}
              <button
                type="button"
                data-testid={`${testId}.close`}
                aria-label={closeLabel}
                onClick={overlay.close}
              >
                ×
              </button>
            </div>
          </header>
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
