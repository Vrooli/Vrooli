/**
 * @libraryId react-component-library:DrawerShell
 * @displayName DrawerShell
 * @description Ingested from web-console:ui/src/components/DrawerShell.tsx
 * @version 0.1.0-draft.1
 * @tags ["overlay","drawer","layout"]
 * @status draft
 * @category overlays
 * @originScenario web-console
 * @originPath ui/src/components/DrawerShell.tsx
 * @warning Ingested by React Component Library. Preserve this provenance header.
 */
import { useEffect, useId, useRef, type ReactNode } from "react";

interface DrawerShellProps {
  /** Whether the drawer is mounted/visible. Returns null when false. */
  open?: boolean;
  /** Close handler invoked by the backdrop, close button, and Escape key. */
  onClose?: () => void;
  /** Accessible label for the close button. */
  closeAriaLabel?: string;
  /** Primary header title (truncated single line). */
  title?: ReactNode;
  /** Optional controls rendered in the title row, before the close button. */
  headerActions?: ReactNode;
  /** Optional content rendered below the title row (subtitle, badges, toolbars). */
  headerExtra?: ReactNode;
  /** Test id applied to the panel element. */
  panelTestId?: string;
  /**
   * Panel sizing. 'full' (default) is the full-page inset card on desktop;
   * 'compact' keeps the identical mobile bottom sheet but renders a centered
   * auto-height max-w-md card on desktop, so small panels (appearance, AI
   * prompt) can adopt the drawer without becoming full-page.
   */
  size?: "full" | "compact";
  /**
   * When true, the panel bottom is lifted above the on-screen keyboard by
   * anchoring it to `--wc-kb-height` (set by useAppViewport) instead of the
   * viewport bottom. Opt-in because only drawers with a focused input (the
   * composer) need it — file/Mermaid previews have no input and leave it off
   * (where it would be a `0px` no-op anyway). This reuses the exact keyboard
   * signal the MobileToolbar layout rides on, so the drawer's own action row
   * sits above the keyboard on iOS (where `position: fixed` does not shrink
   * for the keyboard).
   */
  avoidKeyboard?: boolean;
  /** Drawer body. */
  children: ReactNode;
}

/**
 * DrawerShell is the shared modal/drawer surface for web-console: full-page
 * previews, the composer, settings, the launcher, and small compact panels.
 * It owns the backdrop, panel sizing, safe-area handling, dialog semantics
 * (role=dialog, aria-modal, labelled title), Escape-to-close, focus trapping,
 * and the header chrome. It is a pure UI contract and intentionally knows
 * nothing about its consumers' domains.
 */
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
  const titleId = useId();

  useEffect(() => {
    if (!open) return;
    const previousFocus = document.activeElement as HTMLElement | null;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKeyDown);
    closeButtonRef.current?.focus();
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      previousFocus?.focus();
    };
  }, [onClose, open]);

  if (!open) return null;

  const desktopSizeClasses =
    size === "compact"
      ? "md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-full md:max-w-md md:-translate-x-1/2 md:-translate-y-1/2 md:max-h-[80vh] md:rounded-panel md:border"
      : "md:inset-x-8 md:bottom-8 md:top-8 md:rounded-panel md:border";

  return (
    <div className="fixed inset-0 z-50">
      <button
        type="button"
        className="absolute inset-0 cursor-default bg-app-background/70"
        aria-label="Close drawer"
        onClick={onClose}
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        data-testid={panelTestId}
        className={
          "absolute inset-x-0 top-4 flex flex-col overflow-hidden rounded-t-panel border-t border-app-border bg-app-surface shadow-2xl " +
          desktopSizeClasses +
          (avoidKeyboard
            ? " bottom-[var(--rcl-keyboard-offset,0px)]"
            : " bottom-0")
        }
      >
        <div className="shrink-0 border-b border-app-border px-4 py-3">
          <div className="flex items-center gap-3">
            <h2
              id={titleId}
              className="min-w-0 flex-1 truncate text-sm font-semibold text-app-foreground"
            >
              {title}
            </h2>
            {headerActions}
            <button
              ref={closeButtonRef}
              type="button"
              onClick={onClose}
              className="shrink-0 rounded-pill p-1.5 text-app-muted-foreground transition hover:bg-app-surface-muted hover:text-app-foreground"
              aria-label={closeAriaLabel}
            >
              <span aria-hidden>×</span>
            </button>
          </div>
          {headerExtra}
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">{children}</div>
      </div>
    </div>
  );
}

export default DrawerShell;
