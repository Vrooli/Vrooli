/**
 * @libraryId react-component-library:SidebarShell
 * @displayName Sidebar Shell
 * @version 1.3.1
 * @tags ["layout","navigation","responsive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0","react-component-library:useEscapeKey":"^1.0.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { type CSSProperties, type HTMLAttributes, type ReactNode, forwardRef } from "react";
import { X } from "lucide-react";
import { useEscapeKey } from "../../../../hooks/useEscapeKey/versions/1.0.0/useEscapeKey";
export const sidebarShellStyles = `
[data-rcl-sidebar-shell] { min-block-size: 0; min-inline-size: 0; display: flex; flex-direction: column; border-inline-end: var(--border-hairline) solid var(--color-border); background: var(--color-surface); color: var(--color-foreground); }
[data-rcl-sidebar-shell][data-mode="persistent"] { position: relative; z-index: auto; block-size: 100%; flex-shrink: 0; box-shadow: var(--elev-flat); }
[data-rcl-sidebar-shell][data-mode="overlay"], [data-rcl-sidebar-shell][data-mode="responsive"] { position: fixed; inset-block: 0; inset-inline-start: 0; z-index: var(--layer-modal); block-size: 100dvh; inline-size: 100%; max-inline-size: none; padding-block: env(safe-area-inset-top) env(safe-area-inset-bottom); box-shadow: var(--elev-modal); transform: translateX(0); transition: transform var(--dur-quick) var(--ease-standard), visibility var(--dur-quick) var(--ease-standard); }
[data-rcl-sidebar-shell][data-mode="overlay"][data-open="false"], [data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"] { visibility: hidden; transform: translateX(-100%); }
[data-rcl-sidebar-shell][data-mode="responsive"] { display: flex; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-xs); min-block-size: var(--tap-target-min); border-block-end: var(--border-hairline) solid var(--color-border); padding-inline: var(--space-xs); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__header-content { min-inline-size: 0; overflow-wrap: anywhere; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__close { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); flex-shrink: 0; border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__close:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__icon { inline-size: var(--space-sm); block-size: var(--space-sm); }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__content { min-block-size: 0; min-inline-size: 0; flex: 1; overflow: auto; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__resize { position: absolute; inset-block: 0; inset-inline-end: calc(var(--space-3xs) * -1); z-index: var(--layer-sticky); inline-size: var(--space-xs); border: 0; background: transparent; cursor: col-resize; }
[data-rcl-sidebar-shell] .rcl-sidebar-shell__resize:hover, [data-rcl-sidebar-shell] .rcl-sidebar-shell__resize:focus-visible { background: color-mix(in srgb, var(--color-primary) 25%, transparent); }
[data-rcl-sidebar-backdrop] { position: fixed; inset: 0; z-index: calc(var(--layer-modal) - 1); border: 0; background: color-mix(in srgb, var(--color-shell) 60%, transparent); cursor: default; }
[data-rcl-sidebar-shell] :focus-visible, [data-rcl-sidebar-backdrop]:focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (min-width: 768px) { [data-rcl-sidebar-shell][data-mode="responsive"] { position: relative; inset: auto; z-index: auto; block-size: 100%; inline-size: auto; padding-block: 0; box-shadow: var(--elev-flat); transform: none; visibility: visible; } [data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"] { transform: none; visibility: visible; } [data-rcl-sidebar-shell][data-mode="responsive"] .rcl-sidebar-shell__header { display: none; } [data-rcl-sidebar-shell][data-mode="overlay"] { inline-size: min(22rem, 100vw); max-inline-size: 22rem; } [data-rcl-sidebar-backdrop][data-mode="responsive"] { display: none; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-sidebar-shell] *, [data-rcl-sidebar-shell] *::before, [data-rcl-sidebar-shell] *::after, [data-rcl-sidebar-backdrop] { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-sidebar-shell] { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-sidebar-backdrop] { background: Canvas; opacity: .8; } }
`;
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
        data-rcl-sidebar-shell-styles=""
        dangerouslySetInnerHTML={{ __html: sidebarShellStyles }}
      />
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
