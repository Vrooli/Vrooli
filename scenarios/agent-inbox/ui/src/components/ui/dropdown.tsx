import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";
import { cn } from "../../lib/utils";

interface DropdownProps {
  trigger: ReactNode;
  children: ReactNode;
  align?: "left" | "right";
  className?: string;
}

export function Dropdown({ trigger, children, align = "left", className }: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const handleClickOutside = useCallback((event: MouseEvent) => {
    if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
      setIsOpen(false);
    }
  }, []);

  const handleEscape = useCallback((event: KeyboardEvent) => {
    if (event.key === "Escape") {
      setIsOpen(false);
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      document.addEventListener("mousedown", handleClickOutside);
      document.addEventListener("keydown", handleEscape);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [isOpen, handleClickOutside, handleEscape]);

  return (
    <div ref={dropdownRef} className="relative">
      <div onClick={() => setIsOpen(!isOpen)}>{trigger}</div>
      {isOpen && (
        <div
          className={cn(
            "absolute z-50 mt-1 min-w-[180px] bg-slate-900 border border-white/10 rounded-lg shadow-xl overflow-hidden animate-in fade-in-0 zoom-in-95 duration-100",
            align === "right" ? "right-0" : "left-0",
            className
          )}
          data-testid="dropdown-menu"
        >
          <div onClick={() => setIsOpen(false)}>{children}</div>
        </div>
      )}
    </div>
  );
}

interface DropdownItemProps {
  children: ReactNode;
  onClick?: () => void;
  disabled?: boolean;
  destructive?: boolean;
  className?: string;
}

export function DropdownItem({ children, onClick, disabled, destructive, className }: DropdownItemProps) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "w-full flex items-center gap-2 px-3 py-2 text-sm text-left transition-colors",
        "hover:bg-white/5 focus:bg-white/5 focus:outline-none",
        "disabled:opacity-50 disabled:cursor-not-allowed",
        destructive ? "text-red-400 hover:text-red-300" : "text-slate-300 hover:text-white",
        className
      )}
    >
      {children}
    </button>
  );
}

export function DropdownSeparator() {
  return <div className="h-px bg-white/10 my-1" />;
}
