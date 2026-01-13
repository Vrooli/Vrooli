/**
 * Simple tooltip component using CSS-only approach for minimal bundle size.
 */

import * as React from "react";
import { cn } from "../../lib/utils";

interface TooltipProviderProps {
  children: React.ReactNode;
}

export function TooltipProvider({ children }: TooltipProviderProps) {
  return <>{children}</>;
}

interface TooltipProps {
  children: React.ReactNode;
}

export function Tooltip({ children }: TooltipProps) {
  return <div className="relative inline-block">{children}</div>;
}

interface TooltipTriggerProps {
  children: React.ReactNode;
  asChild?: boolean;
}

export function TooltipTrigger({ children, asChild }: TooltipTriggerProps) {
  return <div className="peer cursor-pointer">{children}</div>;
}

interface TooltipContentProps {
  children: React.ReactNode;
  side?: "top" | "bottom" | "left" | "right";
  className?: string;
}

export function TooltipContent({
  children,
  side = "top",
  className,
}: TooltipContentProps) {
  const positionClasses: Record<string, string> = {
    top: "bottom-full left-1/2 -translate-x-1/2 mb-2",
    bottom: "top-full left-1/2 -translate-x-1/2 mt-2",
    left: "right-full top-1/2 -translate-y-1/2 mr-2",
    right: "left-full top-1/2 -translate-y-1/2 ml-2",
  };

  return (
    <div
      className={cn(
        "pointer-events-none absolute z-50 hidden rounded-md border border-slate-700 bg-slate-800 px-3 py-1.5 text-xs text-slate-100 shadow-lg peer-hover:block",
        positionClasses[side],
        className
      )}
    >
      {children}
    </div>
  );
}
