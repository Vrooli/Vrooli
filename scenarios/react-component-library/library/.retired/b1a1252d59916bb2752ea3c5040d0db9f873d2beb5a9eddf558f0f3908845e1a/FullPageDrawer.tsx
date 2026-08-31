/**
 * @libraryId react-component-library:FullPageDrawer
 * @displayName FullPageDrawer
 * @description Shared overlay presentation for FullPageDrawer.
 * @version 1.0.3
 * @tags ["overlay","accessibility","responsive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import type { ReactNode, RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.1.1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { fullPageDrawerStyles } from "./styles";
export interface FullPageDrawerProps {
  open: boolean;
  onOpenChange?: (open: boolean) => void;
  onClose?: () => void;
  title: ReactNode;
  ariaLabel?: string;
  children: ReactNode;
  headerActions?: ReactNode;
  headerExtra?: ReactNode;
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
export function FullPageDrawer({
  open,
  onOpenChange,
  onClose,
  title,
  ariaLabel,
  children,
  headerActions,
  headerExtra,
  footer,
  closeLabel,
  avoidKeyboard = false,
  initialFocusRef,
  returnFocusRef,
  testId = "overlays.full-page-drawer",
  className,
  panelClassName,
  contentClassName,
  backdropClassName,
}: FullPageDrawerProps) {
  useLibraryStyleSheet("full-page-drawer-1.0.0", fullPageDrawerStyles);
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose?.();
    },
    modal: true,
    kind: "drawer",
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
      >
        <button
          type="button"
          data-testid={`${testId}.backdrop`}
          aria-label={closeLabel}
          className={classes(
            "rcl-full-page-drawer__backdrop",
            backdropClassName,
          )}
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
          className={classes("rcl-full-page-drawer__panel", panelClassName)}
        >
          <header>
            <div>
              <h2>{title}</h2>
              {headerExtra}
            </div>
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
            className={classes(
              "rcl-full-page-drawer__content",
              contentClassName,
            )}
          >
            {children}
          </div>
          {footer ? <footer>{footer}</footer> : null}
        </section>
      </div>
    </Portal>
  );
}
