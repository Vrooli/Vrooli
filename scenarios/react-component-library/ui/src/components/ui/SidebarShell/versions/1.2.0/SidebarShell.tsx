/**
 * @vrooliComponentSource react-component-library:SidebarShell
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption dd1e7ab7-681b-46b4-aad9-ebc580c0e9a7
 * @vrooliComponentAppliedAt 2026-08-12T11:40:33Z
 * @vrooliComponentSourceSha256 e911576710b48907a8b5bc87fcdc96a41dfb6af5875c606f0a7d66d94db16278
 * @vrooliComponentDriftHash c789f980eab887028c955eaf6073f53915bd0444947c0758c7038e41455d5c4c
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { type CSSProperties, type HTMLAttributes, type ReactNode, forwardRef } from "react";
import { X } from "lucide-react";
import { useEscapeKey } from "../../../../../hooks/useEscapeKey/versions/1.0.0/useEscapeKey";
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
      <style
        data-rcl-sidebar-shell-styles
        dangerouslySetInnerHTML={{ __html: sidebarShellStyles }}
      />
      {isDialogOpen ? (
        <button
          type="button"
          data-testid="sidebar-shell-backdrop"
          data-rcl-sidebar-backdrop
          data-mode={mode}
          aria-label={closeLabel}
          className={backdropClassName}
          onClick={onMobileClose}
        />
      ) : null}
      <div
        ref={ref}
        data-testid="sidebar-shell"
        data-rcl-sidebar-shell
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
              data-testid="sidebar-shell-close"
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
