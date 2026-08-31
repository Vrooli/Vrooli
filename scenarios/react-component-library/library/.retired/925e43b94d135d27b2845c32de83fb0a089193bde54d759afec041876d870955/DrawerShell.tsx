/**
 * @libraryId react-component-library:DrawerShell
 * @displayName DrawerShell
 * @description Token-bound, self-contained drawer and modal shell.
 * @version 1.1.1
 * @tags ["overlay","drawer","layout","reviewed","accessibility"]
 * @deps {"react":"^18"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1.0.0";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";
import { drawerShellStyles } from "./styles";

export interface DrawerShellProps {
  open?: boolean;
  onClose?: () => void;
  closeAriaLabel?: string;
  title?: ReactNode;
  headerActions?: ReactNode;
  headerExtra?: ReactNode;
  panelTestId?: string;
  size?: "full" | "compact";
  avoidKeyboard?: boolean;
  children: ReactNode;
}

export const DrawerShell = withClassName(function DrawerShell({
  open = true,
  onClose = () => {},
  closeAriaLabel = "Close drawer",
  title = "Drawer",
  headerActions,
  headerExtra,
  panelTestId,
  size = "full",
  avoidKeyboard = false,
  children,
}: DrawerShellProps) {
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useFocusTrap(open, panelRef);
  useEscapeKey(open, onClose);

  useEffect(() => {
    if (!open) return;
    const previousFocus = document.activeElement as HTMLElement | null;
    closeButtonRef.current?.focus();
    return () => {
      previousFocus?.focus();
    };
  }, [open]);

  if (!open) return null;

  return (
    <>
      <StyleSheet name="drawershell-1-1-1-1" css={drawerShellStyles} />
      <div data-rcl-drawer-shell-root>
        <button
          type="button"
          data-rcl-drawer-shell-backdrop
          aria-label="Dismiss drawer backdrop"
          onClick={onClose}
        />
        <section
          ref={panelRef}
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          data-rcl-drawer-shell
          data-size={size}
          data-avoid-keyboard={avoidKeyboard ? "true" : "false"}
          data-testid={panelTestId}
        >
          <header data-rcl-drawer-shell-header>
            <div data-rcl-drawer-shell-title-row>
              <h2 id={titleId} data-rcl-drawer-shell-title>
                {title}
              </h2>
              {headerActions ? (
                <div data-rcl-drawer-shell-actions>{headerActions}</div>
              ) : null}
              <button
                ref={closeButtonRef}
                type="button"
                data-rcl-drawer-shell-close
                onClick={onClose}
                aria-label={closeAriaLabel}
              >
                <span aria-hidden="true">×</span>
              </button>
            </div>
            {headerExtra ? (
              <div data-rcl-drawer-shell-extra>{headerExtra}</div>
            ) : null}
          </header>
          <div data-rcl-drawer-shell-body>{children}</div>
        </section>
      </div>
    </>
  );
});

export default DrawerShell;
