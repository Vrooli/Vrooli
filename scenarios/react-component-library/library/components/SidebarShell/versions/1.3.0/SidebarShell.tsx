/**
 * @libraryId react-component-library:SidebarShell
 * @displayName Sidebar Shell
 * @description Responsive sidebar parent with desktop resizing and mobile full-width safe-area drawer behavior.
 * @version 1.3.0
 * @tags ["layout","navigation","responsive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0","react-component-library:useEscapeKey":"^1.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { cn } from "../../../../foundations/ClassMerge/versions/1.0.0/ClassMerge";
import { type CSSProperties, type HTMLAttributes, type ReactNode, forwardRef } from "react";
import { X } from "lucide-react";
import { useEscapeKey } from "../../../../hooks/useEscapeKey/versions/1.0.0/useEscapeKey";
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
      <style
        data-rcl-sidebar-shell-styles=""
        dangerouslySetInnerHTML={{ __html: sidebarShellStyles }}
      />
      {isDialogOpen ? (
        <button
          type="button"
          data-testid="navigation.sidebar"
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
              data-testid="navigation.sidebar"
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
