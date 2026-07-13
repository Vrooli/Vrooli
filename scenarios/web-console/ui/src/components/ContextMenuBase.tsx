import { useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { useEscapeKey } from "../hooks/useEscapeKey";
import { useFloatingPosition } from "../hooks/useFloatingPosition";

interface ContextMenuBaseProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  onClose: () => void;
  children: ReactNode;
  /** Optional data-testid for the menu container. */
  "data-testid"?: string;
}

/** Shared item class for context menu buttons. */
export const contextMenuItemClass =
  "w-full flex items-center gap-2 text-start px-3 py-2 text-sm text-wc-text-primary hover:bg-white/10 transition-colors first:rounded-t-lg last:rounded-t-none last:rounded-b-lg";

/**
 * Reusable context menu wrapper providing:
 * - Fixed-position container with viewport clamping
 * - Backdrop overlay for click-to-dismiss
 * - Escape key dismissal
 * - Invisible-until-measured to prevent layout flash
 */
export default function ContextMenuBase({ position, onClose, children, "data-testid": testId }: ContextMenuBaseProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [menuSize, setMenuSize] = useState<{ width: number; height: number } | null>(null);
  const { clampPosition } = useFloatingPosition();

  // Measure menu after first render
  useLayoutEffect(() => {
    const el = menuRef.current;
    if (!el) return;
    setMenuSize({ width: el.offsetWidth, height: el.offsetHeight });
  }, []);

  // Dismiss on Escape
  useEscapeKey(true, onClose);

  // Compute clamped position (invisible until measured)
  const clamped = menuSize ? clampPosition(position.x, position.y, menuSize) : null;

  return (
    <>
      {/* Backdrop */}
      <div
        data-testid="ctx-backdrop"
        className="fixed inset-0 z-wc-menu-backdrop"
        onClick={onClose}
      />
      {/* Menu */}
      <div
        ref={menuRef}
        data-testid={testId}
        className="wc-stable-theme fixed z-wc-menu min-w-[140px] rounded-lg border border-wc-default bg-wc-surface-raised shadow-xl py-1"
        style={
          clamped
            ? { left: clamped.x, top: clamped.y }
            : { left: position.x, top: position.y, opacity: 0, pointerEvents: "none" as const }
        }
      >
        {children}
      </div>
    </>
  );
}
