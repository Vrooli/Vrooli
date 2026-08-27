/**
 * @libraryId react-component-library:SidebarShell
 * @displayName Sidebar Shell
 * @description Responsive sidebar parent with desktop resizing and mobile full-width safe-area drawer behavior.
 * @version 1.3.2
 * @tags ["layout","navigation","responsive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0","react-component-library:useEscapeKey":"^1.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { type CSSProperties, type HTMLAttributes, type ReactNode, forwardRef } from "react";
import { X } from "lucide-react";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";
import { sidebarShellStyles } from "./styles";

export interface SidebarShellProps {
  children: ReactNode;
  mode?: "responsive" | "overlay" | "persistent";
  mobileOpen: boolean;
  onMobileClose: () => void;
  mobileLabel: string;
  desktopLabel?: string;
  closeLabel: string;
  mobileHeader?: ReactNode;
  width?: number;
  resizeHandleProps?: HTMLAttributes<HTMLDivElement>;
  className?: string;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

export const SidebarShell = forwardRef<HTMLDivElement, SidebarShellProps>(function SidebarShell(
  {
    children,
    mode = "responsive",
    mobileOpen,
    onMobileClose,
    mobileLabel,
    desktopLabel,
    closeLabel,
    mobileHeader,
    width,
    resizeHandleProps,
    className,
    panelClassName,
    contentClassName,
    backdropClassName,
  },
  ref,
) {
  const isPersistent = mode === "persistent";
  const isDialogOpen = !isPersistent && mobileOpen;

  useEscapeKey(isDialogOpen, onMobileClose);

  const style: CSSProperties = width ? { width } : {};
  return (
    <>
      <StyleSheet name="sidebar-shell-1-3-2" css={sidebarShellStyles} />
      {isDialogOpen ? (
        <button
          type="button"
          data-testid="navigation.sidebar-backdrop"
          data-rcl-sidebar-backdrop=""
          data-mode={mode}
          aria-label={closeLabel}
          className={backdropClassName}
          onClick={onMobileClose}
        />
      ) : null}
      <div
        ref={ref}
        data-testid="sidebar-shell"
        data-rcl-sidebar-shell=""
        data-mode={mode}
        data-open={mobileOpen ? "true" : "false"}
        role={isDialogOpen ? "dialog" : "complementary"}
        aria-modal={isDialogOpen ? "true" : undefined}
        aria-label={isDialogOpen ? mobileLabel : (desktopLabel ?? mobileLabel)}
        style={style}
        className={cn(className, panelClassName)}
      >
        {!isPersistent && (
          <div className="rcl-sidebar-shell__header">
            <div className="rcl-sidebar-shell__header-content">{mobileHeader}</div>
            <button
              type="button"
              data-testid="navigation.sidebar-close"
              aria-label={closeLabel}
              onClick={onMobileClose}
              className="rcl-sidebar-shell__close"
            >
              <X aria-hidden className="rcl-sidebar-shell__icon" />
            </button>
          </div>
        )}
        <div className={cn("rcl-sidebar-shell__content", contentClassName)}>{children}</div>
        {resizeHandleProps && (
          <div
            data-testid="sidebar-shell-resize-handle"
            {...resizeHandleProps}
            className={cn("rcl-sidebar-shell__resize", resizeHandleProps.className)}
          />
        )}
      </div>
    </>
  );
});
