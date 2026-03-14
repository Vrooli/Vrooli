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

export function Popover({ trigger, children, align = "center", direction = "down" }: PopoverProps) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState<{ top: number; left: number }>({ top: 0, left: 0 });

  const close = useCallback(() => setOpen(false), []);

  const computePosition = useCallback(() => {
    if (!triggerRef.current) return;
    const rect = triggerRef.current.getBoundingClientRect();
    const gap = 8;

    let top: number;
    if (direction === "up") {
      top = rect.top - gap;
    } else {
      top = rect.bottom + gap;
    }

    let left: number;
    if (align === "start") {
      left = rect.left;
    } else if (align === "end") {
      left = rect.right;
    } else {
      left = rect.left + rect.width / 2;
    }

    setPosition({ top, left });
  }, [align, direction]);

  // Recompute position on open and on scroll/resize
  useLayoutEffect(() => {
    if (!open) return;
    computePosition();
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

  const alignTransform =
    align === "start"
      ? "translateX(0)"
      : align === "end"
        ? "translateX(-100%)"
        : "translateX(-50%)";

  const directionTransform = direction === "up" ? "translateY(-100%)" : "";

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
            className="fixed z-50 min-w-[200px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl"
            style={{
              top: position.top,
              left: position.left,
              transform: `${alignTransform} ${directionTransform}`.trim(),
            }}
            role="dialog"
          >
            {children}
          </div>,
          document.body
        )}
    </>
  );
}
