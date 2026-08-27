/**
 * @libraryId react-component-library:ResponsiveDialog
 * @displayName ResponsiveDialog
 * @description Shared overlay presentation for ResponsiveDialog.
 * @version 1.0.4
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { useBreakpoint } from "@vrooli/react-component-library/useMediaQuery/1.1.0";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.1.1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { responsiveDialogStyles } from "./styles";
export type ResponsiveDialogSize = "sm" | "md" | "lg";
export interface ResponsiveDialogProps {
  open: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title: ReactNode;
  ariaLabel?: string;
  children: ReactNode;
  footer?: ReactNode;
  headerActions?: ReactNode;
  closeLabel: string;
  size?: ResponsiveDialogSize;
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
export function ResponsiveDialog({
  open,
  onOpenChange,
  onClose,
  title,
  ariaLabel,
  children,
  footer,
  headerActions,
  closeLabel,
  size = "md",
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.responsive-dialog",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: ResponsiveDialogProps) {
  useLibraryStyleSheet("responsive-dialog-1.0.0", responsiveDialogStyles);
  const desktop = useBreakpoint("md");
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: desktop ? "dialog" : "sheet",
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
          ref={overlay.surfaceRef}
          data-testid={testId}
          role="dialog"
          aria-modal="true"
          aria-label={ariaLabel ?? (typeof title === "string" ? title : closeLabel)}
          className={classes("rcl-responsive-dialog__panel", panelClassName)}
        >
          {!desktop ? <div className="rcl-responsive-dialog__handle" aria-hidden /> : null}
          <header>
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
          <div className={classes("rcl-responsive-dialog__content", contentClassName)}>
            {children}
          </div>
          {footer ? <footer>{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
