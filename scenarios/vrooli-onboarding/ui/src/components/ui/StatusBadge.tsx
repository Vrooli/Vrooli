import { cn } from "../../lib/utils";

interface StatusBadgeProps {
  children: React.ReactNode;
  tone?: "healthy" | "warning" | "muted";
  className?: string;
}

export function StatusBadge({ children, tone = "muted", className }: StatusBadgeProps) {
  return (
    <span className={cn(
      "rounded px-1.5 py-0.5 text-xs font-medium",
      tone === "healthy" && "bg-emerald-500/15 text-emerald-300",
      tone === "warning" && "bg-amber-400/15 text-amber-200",
      tone === "muted" && "bg-white/5 text-slate-300",
      className,
    )}>
      {children}
    </span>
  );
}
