import { useRef, useEffect, useCallback, useState, createContext, useContext, type ReactNode } from "react";

interface PopoverContextValue {
  open: boolean;
  setOpen: (open: boolean) => void;
}

const PopoverCtx = createContext<PopoverContextValue | null>(null);

interface PopoverProps {
  children: ReactNode;
}

export function Popover({ children }: PopoverProps) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
      setOpen(false);
    }
  }, []);

  const handleEscape = useCallback((e: KeyboardEvent) => {
    if (e.key === "Escape") {
      setOpen(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
      document.addEventListener("keydown", handleEscape);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [open, handleClickOutside, handleEscape]);

  return (
    <PopoverCtx.Provider value={{ open, setOpen }}>
      <div className="relative inline-block" ref={containerRef}>
        {children}
      </div>
    </PopoverCtx.Provider>
  );
}

interface PopoverTriggerProps {
  children: ReactNode;
}

export function PopoverTrigger({ children }: PopoverTriggerProps) {
  const ctx = useContext(PopoverCtx);
  if (!ctx) return null;

  return (
    <div
      onClick={() => ctx.setOpen(!ctx.open)}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          ctx.setOpen(!ctx.open);
        }
      }}
    >
      {children}
    </div>
  );
}

interface PopoverContentProps {
  children: ReactNode;
  align?: "start" | "end";
  side?: "top" | "bottom";
  className?: string;
}

export function PopoverContent({ children, align = "end", side = "bottom", className = "" }: PopoverContentProps) {
  const ctx = useContext(PopoverCtx);
  if (!ctx || !ctx.open) return null;

  const alignClass = align === "start" ? "left-0" : "right-0";
  const sideClass = side === "top" ? "bottom-full mb-1" : "top-full mt-1";

  return (
    <div
      className={`absolute z-50 ${alignClass} ${sideClass} min-w-[280px] rounded-lg border border-slate-700 bg-slate-900 shadow-xl ${className}`}
      role="dialog"
    >
      {children}
    </div>
  );
}
