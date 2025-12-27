import { useState, useRef, useEffect, useLayoutEffect } from "react";
import { createPortal } from "react-dom";
import { HelpCircle } from "lucide-react";
import { cn } from "../../lib/utils";

interface TooltipProps {
  content: React.ReactNode;
  children?: React.ReactNode;
  side?: "top" | "bottom" | "left" | "right";
  className?: string;
}

interface TooltipPosition {
  left: number;
  top: number;
}

export function Tooltip({ content, children, side = "top", className }: TooltipProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState<TooltipPosition | null>(null);
  const tooltipRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);

  // Use useLayoutEffect to calculate position synchronously before paint
  useLayoutEffect(() => {
    if (!isOpen || !tooltipRef.current || !triggerRef.current) {
      setPosition(null);
      return;
    }

    const tooltip = tooltipRef.current;
    const trigger = triggerRef.current;
    const triggerRect = trigger.getBoundingClientRect();
    const tooltipRect = tooltip.getBoundingClientRect();

    let left = 0;
    let top = 0;

    switch (side) {
      case "top":
        left = triggerRect.left + triggerRect.width / 2 - tooltipRect.width / 2;
        top = triggerRect.top - tooltipRect.height - 8;
        break;
      case "bottom":
        left = triggerRect.left + triggerRect.width / 2 - tooltipRect.width / 2;
        top = triggerRect.bottom + 8;
        break;
      case "left":
        left = triggerRect.left - tooltipRect.width - 8;
        top = triggerRect.top + triggerRect.height / 2 - tooltipRect.height / 2;
        break;
      case "right":
        left = triggerRect.right + 8;
        top = triggerRect.top + triggerRect.height / 2 - tooltipRect.height / 2;
        break;
    }

    // Keep tooltip within viewport
    const padding = 8;
    left = Math.max(padding, Math.min(left, window.innerWidth - tooltipRect.width - padding));
    top = Math.max(padding, Math.min(top, window.innerHeight - tooltipRect.height - padding));

    setPosition({ left, top });
  }, [isOpen, side]);

  // Reset position when closed
  useEffect(() => {
    if (!isOpen) {
      setPosition(null);
    }
  }, [isOpen]);

  const tooltipContent = isOpen ? (
    <div
      ref={tooltipRef}
      className="fixed z-50 max-w-xs px-3 py-2 text-sm bg-slate-800 border border-slate-700 rounded-lg shadow-xl text-slate-200 transition-opacity duration-100"
      style={{
        left: position?.left ?? -9999,
        top: position?.top ?? -9999,
        opacity: position ? 1 : 0,
        pointerEvents: position ? "auto" : "none",
      }}
      role="tooltip"
    >
      {content}
    </div>
  ) : null;

  return (
    <span className={cn("inline-flex", className)}>
      <button
        ref={triggerRef}
        type="button"
        onMouseEnter={() => setIsOpen(true)}
        onMouseLeave={() => setIsOpen(false)}
        onFocus={() => setIsOpen(true)}
        onBlur={() => setIsOpen(false)}
        className="inline-flex items-center justify-center text-slate-400 hover:text-slate-300 transition-colors"
        aria-label="More information"
      >
        {children || <HelpCircle className="h-4 w-4" />}
      </button>
      {createPortal(tooltipContent, document.body)}
    </span>
  );
}

interface HelpTooltipProps {
  content: React.ReactNode;
  className?: string;
}

export function HelpTooltip({ content, className }: HelpTooltipProps) {
  return (
    <Tooltip content={content} className={className}>
      <HelpCircle className="h-4 w-4" />
    </Tooltip>
  );
}
