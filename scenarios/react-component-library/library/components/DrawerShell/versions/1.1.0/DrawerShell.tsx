/**
 * @libraryId react-component-library:DrawerShell
 * @version 1.1.0
 * @status released
 * @deps {"react":"^18"}
 */
import { useCallback, useEffect, useId, useRef, type ReactNode } from "react";
import { useFocusTrap } from "../../../../hooks/useFocusTrap/versions/1.0.0/useFocusTrap";
import { layerManager } from "../../../../services/LayerManager/versions/1.0.0/LayerManager";
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

export function DrawerShell({
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
  const layerId = useRef(`drawer-shell-${titleId.replace(/:/g, "")}`);
  const close = useCallback(() => onClose(), [onClose]);

  useFocusTrap(open, panelRef);

  useEffect(() => {
    if (!open) return;
    if (typeof window === "undefined") return;
    const previousFocus = document.activeElement as HTMLElement | null;
    const removeLayer = layerManager.push({
      id: layerId.current,
      kind: "drawer",
      dismiss: close,
    });
    const onKeyDown = (event: KeyboardEvent) => {
      if (
        event.key === "Escape" &&
        layerManager.top()?.id === layerId.current
      ) {
        event.preventDefault();
        close();
      }
    };
    window.addEventListener("keydown", onKeyDown);
    closeButtonRef.current?.focus();
    return () => {
      removeLayer();
      window.removeEventListener("keydown", onKeyDown);
      previousFocus?.focus();
    };
  }, [close, open]);

  if (!open) return null;

  return (
    <>
      <style
        data-rcl-drawer-shell-styles
        dangerouslySetInnerHTML={{ __html: drawerShellStyles }}
      />
      <div data-rcl-drawer-shell-root>
        <button
          type="button"
          data-rcl-drawer-shell-backdrop
          aria-label="Dismiss drawer backdrop"
          onClick={close}
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
                onClick={close}
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
}

export default DrawerShell;
