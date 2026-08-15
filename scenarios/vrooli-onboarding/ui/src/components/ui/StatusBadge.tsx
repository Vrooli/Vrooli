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
      tone === "healthy" && "bg-primary/15 text-primary",
      tone === "warning" && "bg-warning-surface text-warning",
      tone === "muted" && "bg-surface-muted text-muted",
      className,
    )}>
      {children}
    </span>
  );
}
