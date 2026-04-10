import { useState, useRef, useEffect, useCallback, useLayoutEffect, type ReactNode } from "react";
import { createPortal } from "react-dom";

interface PopoverProps {
  trigger: ReactNode;
  children: ReactNode;
  /** Alignment relative to trigger */
  align?: "start" | "center" | "end";
  /** Direction the popover opens */
  direction?: "down" | "up";
}

const VIEWPORT_PADDING = 8;

export function Popover({ trigger, children, align = "center", direction = "down" }: PopoverProps) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [style, setStyle] = useState<React.CSSProperties>({ top: 0, left: 0, visibility: "hidden" });

  const close = useCallback(() => setOpen(false), []);

  const computePosition = useCallback(() => {
    if (!triggerRef.current || !popoverRef.current) return;
    const triggerRect = triggerRef.current.getBoundingClientRect();
    const popoverEl = popoverRef.current;
    const popoverW = popoverEl.offsetWidth;
    const popoverH = popoverEl.offsetHeight;
    const gap = 8;
    const vw = window.innerWidth;
    const vh = window.innerHeight;

    // Vertical: prefer requested direction, flip if not enough room
    let top: number;
    const spaceAbove = triggerRect.top - gap;
    const spaceBelow = vh - triggerRect.bottom - gap;

    if (direction === "up") {
      if (popoverH <= spaceAbove) {
        top = triggerRect.top - gap - popoverH;
      } else if (popoverH <= spaceBelow) {
        top = triggerRect.bottom + gap; // flip down
      } else {
        top = Math.max(VIEWPORT_PADDING, triggerRect.top - gap - popoverH);
      }
    } else {
      if (popoverH <= spaceBelow) {
        top = triggerRect.bottom + gap;
      } else if (popoverH <= spaceAbove) {
        top = triggerRect.top - gap - popoverH; // flip up
      } else {
        top = Math.max(VIEWPORT_PADDING, triggerRect.bottom + gap);
      }
    }

    // Clamp vertically so popover doesn't exceed viewport bottom
    if (top + popoverH > vh - VIEWPORT_PADDING) {
      top = vh - VIEWPORT_PADDING - popoverH;
    }
    if (top < VIEWPORT_PADDING) {
      top = VIEWPORT_PADDING;
    }

    // Horizontal: align to trigger, then clamp to viewport
    let left: number;
    if (align === "start") {
      left = triggerRect.left;
    } else if (align === "end") {
      left = triggerRect.right - popoverW;
    } else {
      left = triggerRect.left + triggerRect.width / 2 - popoverW / 2;
    }

    // Clamp horizontally
    if (left + popoverW > vw - VIEWPORT_PADDING) {
      left = vw - VIEWPORT_PADDING - popoverW;
    }
    if (left < VIEWPORT_PADDING) {
      left = VIEWPORT_PADDING;
    }

    // Set max height so content scrolls if popover still can't fit
    const maxH = vh - 2 * VIEWPORT_PADDING;

    setStyle({ top, left, maxHeight: maxH, visibility: "visible" });
  }, [align, direction]);

  // Recompute position on open and on scroll/resize
  useLayoutEffect(() => {
    if (!open) return;
    // Reset to hidden until we measure
    setStyle((prev) => ({ ...prev, visibility: "hidden" }));
    // Use rAF to ensure the popover is rendered and measurable
    const id = requestAnimationFrame(() => computePosition());
    return () => cancelAnimationFrame(id);
  }, [open, computePosition]);

  useEffect(() => {
    if (!open) return;

    const handleReposition = () => computePosition();
    window.addEventListener("resize", handleReposition);
    window.addEventListener("scroll", handleReposition, true);

    return () => {
      window.removeEventListener("resize", handleReposition);
      window.removeEventListener("scroll", handleReposition, true);
    };
  }, [open, computePosition]);

  // Close on outside click or escape
  useEffect(() => {
    if (!open) return;

    const handleClick = (e: MouseEvent) => {
      const target = e.target as Node;
      if (triggerRef.current?.contains(target)) return;
      if (popoverRef.current?.contains(target)) return;
      close();
    };
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };

    document.addEventListener("mousedown", handleClick);
    document.addEventListener("keydown", handleKey);
    return () => {
      document.removeEventListener("mousedown", handleClick);
      document.removeEventListener("keydown", handleKey);
    };
  }, [open, close]);

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center"
        aria-expanded={open}
      >
        {trigger}
      </button>
      {open &&
        createPortal(
          <div
            ref={popoverRef}
            className="fixed z-50 min-w-[200px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl overflow-y-auto"
            style={style}
            role="dialog"
          >
            {children}
          </div>,
          document.body
        )}
    </>
  );
}
