import { useRef, type ReactNode } from "react";
import { X } from "lucide-react";

import { useEscapeKey } from "../hooks/useEscapeKey";
import { useFocusTrap } from "../hooks/useFocusTrap";

interface DrawerShellProps {
  /** Whether the drawer is mounted/visible. Returns null when false. */
  open: boolean;
  /** Close handler invoked by the backdrop, close button, and Escape key. */
  onClose: () => void;
  /** Accessible label for the close button. */
  closeAriaLabel: string;
  /** Primary header title (truncated single line). */
  title: ReactNode;
  /** Optional controls rendered in the title row, before the close button. */
  headerActions?: ReactNode;
  /** Optional content rendered below the title row (subtitle, badges, toolbars). */
  headerExtra?: ReactNode;
  /** Test id applied to the panel element. */
  panelTestId?: string;
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
 * DrawerShell is the shared full-page overlay used by message-contained
 * previews (file previews, Mermaid diagrams). It owns the backdrop, panel
 * sizing, safe-area handling, Escape-to-close, and the header chrome. It is a
 * pure UI contract and intentionally knows nothing about file previews,
 * preview ids, or diagram source.
 */
export function DrawerShell({
  open,
  onClose,
  closeAriaLabel,
  title,
  headerActions,
  headerExtra,
  panelTestId,
  avoidKeyboard = false,
  children,
}: DrawerShellProps) {
  useEscapeKey(open, onClose);

  // Keep keyboard focus inside the drawer while open (accessibility): Tab and
  // Shift+Tab cycle the panel's focusable controls rather than leaking to the
  // terminal/background behind the overlay.
  const panelRef = useRef<HTMLDivElement>(null);
  useFocusTrap(open, panelRef);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[80]">
      <div className="absolute inset-0 bg-wc-backdrop" onClick={onClose} />
      <div
        ref={panelRef}
        data-testid={panelTestId}
        className={
          "wc-stable-theme absolute inset-x-0 top-[max(1rem,var(--wc-safe-top,0px))] flex flex-col overflow-hidden rounded-t-[20px] border-t border-wc-default bg-wc-surface-raised shadow-2xl md:inset-x-8 md:bottom-8 md:top-8 md:rounded-2xl md:border " +
          // Lift the panel above the keyboard when requested; otherwise pin to
          // the viewport bottom. The md: breakpoint (desktop, no keyboard)
          // overrides both to a fixed inset.
          (avoidKeyboard ? "bottom-[var(--wc-kb-height,0px)]" : "bottom-0")
        }
      >
        <div className="shrink-0 border-b border-wc-default px-4 py-3">
          <div className="flex items-center gap-3">
            <h2 className="min-w-0 flex-1 truncate text-sm font-semibold text-wc-text-primary">{title}</h2>
            {headerActions}
            <button
              type="button"
              onClick={onClose}
              className="shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
              aria-label={closeAriaLabel}
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          {headerExtra}
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">{children}</div>
      </div>
    </div>
  );
}
