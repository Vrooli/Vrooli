import { useState, type ReactNode } from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "../../lib/utils";

export interface CollapsibleProps {
  title: string;
  badge?: ReactNode;
  defaultOpen?: boolean;
  className?: string;
  children: ReactNode;
}

export function Collapsible({
  title,
  badge,
  defaultOpen = false,
  className,
  children,
}: CollapsibleProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div
      className={cn(
        "rounded-lg border border-white/10 bg-white/5",
        className
      )}
    >
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between p-4 text-left hover:bg-white/5 transition-colors rounded-t-lg"
      >
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-slate-300">{title}</span>
          {badge}
        </div>
        <ChevronDown
          className={cn(
            "h-4 w-4 text-slate-500 transition-transform duration-200",
            isOpen && "rotate-180"
          )}
        />
      </button>
      {isOpen && (
        <div className="px-4 pb-4 pt-0 border-t border-white/5">
          {children}
        </div>
      )}
    </div>
  );
}

// Badge component for collapsible headers
export interface StatusBadgeProps {
  status: "success" | "warning" | "error" | "info" | "neutral";
  children: ReactNode;
}

const statusStyles = {
  success: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
  warning: "bg-amber-500/20 text-amber-400 border-amber-500/30",
  error: "bg-red-500/20 text-red-400 border-red-500/30",
  info: "bg-blue-500/20 text-blue-400 border-blue-500/30",
  neutral: "bg-slate-500/20 text-slate-400 border-slate-500/30",
};

export function StatusBadge({ status, children }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border",
        statusStyles[status]
      )}
    >
      {children}
    </span>
  );
}
