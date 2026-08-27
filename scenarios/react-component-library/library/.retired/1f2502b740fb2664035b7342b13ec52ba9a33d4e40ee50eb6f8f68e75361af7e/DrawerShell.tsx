/**
 * @libraryId react-component-library:DrawerShell
 * @displayName DrawerShell
 * @description Ingested from web-console:ui/src/components/DrawerShell.tsx
 * @version 1.0.0
 * @tags ["overlay","drawer","layout"]
 * @originScenario web-console
 * @originPath ui/src/components/DrawerShell.tsx
 * @warning Ingested by React Component Library. Preserve this provenance header.
 */
import { useEffect, useId, useRef, type ReactNode } from "react";

import { useEscapeKey } from "./useEscapeKey";
import { useFocusTrap } from "./useFocusTrap";

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
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();

  useEscapeKey(open, onClose);
  useFocusTrap(open, panelRef);

  useEffect(() => {
    if (!open) return;
    const previousFocus = document.activeElement as HTMLElement | null;
    closeButtonRef.current?.focus();
    return () => previousFocus?.focus();
  }, [open]);

  if (!open) return null;

  const desktopSizeClasses =
    size === "compact"
      ? "md:inset-x-auto md:bottom-auto md:left-1/2 md:top-1/2 md:w-full md:max-w-md md:-translate-x-1/2 md:-translate-y-1/2 md:max-h-[80vh] md:rounded-2xl md:border"
      : "md:inset-x-8 md:bottom-8 md:top-8 md:rounded-2xl md:border";

  return (
    <div className="fixed inset-0 z-wc-drawer">
      <button
        type="button"
        className="absolute inset-0 bg-wc-backdrop"
        aria-label="Dismiss drawer backdrop"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        data-testid={panelTestId}
        className={
          "wc-stable-theme absolute inset-x-0 top-[max(1rem,var(--wc-safe-top,0px))] flex flex-col overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl " +
          desktopSizeClasses +
          (avoidKeyboard ? " bottom-[var(--wc-kb-height,0px)]" : " bottom-0")
        }
      >
        <div className="shrink-0 border-b border-wc-default px-4 py-3">
          <div className="flex items-center gap-3">
            <h2
              id={titleId}
              className="min-w-0 flex-1 truncate text-sm font-semibold text-wc-text-primary"
            >
              {title}
            </h2>
            {headerActions}
            <button
              ref={closeButtonRef}
              type="button"
              onClick={onClose}
              className="touch-target inline-flex shrink-0 items-center justify-center rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
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
