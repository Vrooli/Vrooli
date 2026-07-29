/**
 * @vrooliComponentSource react-component-library:SidebarShell
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 37e07466-1e4f-487d-8b66-51602bdec4eb
 * @vrooliComponentAppliedAt 2026-07-09T04:31:19Z
 * @vrooliComponentSourceSha256 a191851f6b18d0195fdec6bfe430d59ef964fce7ca80b3b05e52d94b2760a036
 * @vrooliComponentDriftHash a191851f6b18d0195fdec6bfe430d59ef964fce7ca80b3b05e52d94b2760a036
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import {
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
  forwardRef,
} from "react";
import { X } from "lucide-react";
import { useEscapeDismiss } from "../../hooks/useEscapeDismiss";

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

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export const SidebarShell = forwardRef<HTMLDivElement, SidebarShellProps>(
  function SidebarShell(
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
    const isResponsive = mode === "responsive";
    const isOverlay = mode === "overlay";
    const isPersistent = mode === "persistent";
    const isDialogOpen = !isPersistent && mobileOpen;

    useEscapeDismiss(isDialogOpen, onMobileClose);

    const style: CSSProperties = width ? { width } : {};
    const backdropClasses = isResponsive
      ? "fixed inset-0 z-40 cursor-default md:hidden"
      : "fixed inset-0 z-40 cursor-default";
    const panelClasses = isPersistent
      ? "relative z-auto flex h-full w-auto shrink-0 flex-col border-r border-app-border bg-app-surface shadow-none"
      : isOverlay
        ? "fixed inset-y-0 left-0 z-50 h-dvh w-full max-w-none shrink-0 flex-col border-r border-app-border bg-app-surface shadow-xl transition-transform duration-200 ease-out"
        : "fixed inset-y-0 left-0 z-50 h-dvh w-full max-w-none shrink-0 flex-col border-r border-app-border bg-app-surface shadow-xl transition-transform duration-200 ease-out md:relative md:inset-auto md:z-auto md:flex md:h-full md:w-auto md:translate-x-0 md:shadow-none";
    const visibilityClasses = isPersistent
      ? "translate-x-0"
      : mobileOpen
        ? "flex translate-x-0"
        : isOverlay
          ? "hidden -translate-x-full"
          : "hidden -translate-x-full md:flex md:translate-x-0";
    const safeAreaClasses = isPersistent
      ? "p-0"
      : isOverlay
        ? "pt-safe pb-safe"
        : "pt-safe pb-safe md:p-0";
    const mobileHeaderClasses = isOverlay
      ? "flex items-center justify-between border-b border-app-border px-3 py-3"
      : "flex items-center justify-between border-b border-app-border px-3 py-3 md:hidden";
    const resizeHandleClasses = isPersistent
      ? "absolute right-[-6px] top-0 z-10 h-full w-3 cursor-col-resize bg-transparent transition-colors hover:bg-app-primary/25 focus-visible:bg-app-primary/25"
      : "absolute right-[-6px] top-0 z-10 hidden h-full w-3 cursor-col-resize bg-transparent transition-colors hover:bg-app-primary/25 focus-visible:bg-app-primary/25 md:block";

    return (
      <>
        {isDialogOpen && (
          <button
            type="button"
            data-testid="sidebar-shell-backdrop"
            aria-label={closeLabel}
            className={cn(backdropClasses, backdropClassName)}
            style={{ background: "color-mix(in srgb, var(--color-shell) 60%, transparent)" }}
            onClick={onMobileClose}
          />
        )}
        <div
          ref={ref}
          data-testid="sidebar-shell"
          data-mode={mode}
          role={isDialogOpen ? "dialog" : "complementary"}
          aria-modal={isDialogOpen ? "true" : undefined}
          aria-label={isDialogOpen ? mobileLabel : desktopLabel ?? mobileLabel}
          style={style}
          className={cn(
            panelClasses,
            visibilityClasses,
            safeAreaClasses,
            className,
            panelClassName,
          )}
        >
          {!isPersistent && (
            <div className={mobileHeaderClasses}>
              <div className="min-w-0">{mobileHeader}</div>
              <button
                type="button"
                data-testid="sidebar-shell-close"
                aria-label={closeLabel}
                onClick={onMobileClose}
                className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
              >
                <X aria-hidden className="h-5 w-5" />
              </button>
            </div>
          )}
          <div className={cn("min-h-0 flex-1 overflow-auto", contentClassName)}>
            {children}
          </div>
          {resizeHandleProps && (
            <div
              data-testid="sidebar-shell-resize-handle"
              {...resizeHandleProps}
              className={cn(resizeHandleClasses, resizeHandleProps.className)}
            />
          )}
        </div>
      </>
    );
  },
);
