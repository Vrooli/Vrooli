import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";

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
  const containerRef = useRef<HTMLDivElement>(null);

  const close = useCallback(() => setOpen(false), []);

  // Close on outside click
  useEffect(() => {
    if (!open) return;

    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        close();
      }
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

  const alignClass =
    align === "start"
      ? "left-0"
      : align === "end"
        ? "right-0"
        : "left-1/2 -translate-x-1/2";

  return (
    <div ref={containerRef} className="relative inline-flex">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="inline-flex items-center"
        aria-expanded={open}
      >
        {trigger}
      </button>
      {open && (
        <div
          className={`absolute ${direction === "up" ? "bottom-full mb-2" : "top-full mt-2"} z-50 min-w-[200px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl ${alignClass}`}
          role="dialog"
        >
          {children}
        </div>
      )}
    </div>
  );
}
