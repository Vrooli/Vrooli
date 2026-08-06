/**
 * @vrooliComponentSource react-component-library:SidebarShell
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 7a17a37b-67a7-474c-a4f9-0d14bb5ed6c9
 * @vrooliComponentAppliedAt 2026-08-06T03:46:46Z
 * @vrooliComponentSourceSha256 7eafb128270a88b84ff62cf185419411fd63efbe4c9bbe18417ee677df0d450e
 * @vrooliComponentDriftHash f5141e71454b981b0f9a4cd5277fe82663814bfbcdc784bcc79cc7314df8b60c
 * @vrooliComponentTokenTranslation bg-app-primary/25->bg-app-primary/25,bg-app-surface->bg-app-surface,bg-app-surface-muted->bg-app-surface-muted,border-app-border->border-app-border,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import {
  type CSSProperties,
  type HTMLAttributes,
  type ReactNode,
  forwardRef,
  useEffect,
} from "react";
import { X } from "lucide-react";

export interface SidebarShellProps {
  children: ReactNode;
  mode?: "responsive" | "overlay" | "persistent";
  mobileOpen: boolean;
  /** Hide the persistent desktop rail without changing mobile drawer state. */
  desktopCollapsed?: boolean;
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

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

export const SidebarShell = forwardRef<HTMLDivElement, SidebarShellProps>(function SidebarShell(
  {
    children,
    mode = "responsive",
    mobileOpen,
    desktopCollapsed = false,
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

  useEffect(() => {
    if (!isDialogOpen) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onMobileClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isDialogOpen, onMobileClose]);

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
        : desktopCollapsed
          ? "hidden -translate-x-full md:!hidden"
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
          className={joinClasses(backdropClasses, backdropClassName)}
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
        aria-label={isDialogOpen ? mobileLabel : (desktopLabel ?? mobileLabel)}
        style={style}
        className={joinClasses(
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
        <div className={joinClasses("min-h-0 flex-1 overflow-auto", contentClassName)}>
          {children}
        </div>
        {resizeHandleProps && (
          <div
            data-testid="sidebar-shell-resize-handle"
            {...resizeHandleProps}
            className={joinClasses(resizeHandleClasses, resizeHandleProps.className)}
          />
        )}
      </div>
    </>
  );
});
