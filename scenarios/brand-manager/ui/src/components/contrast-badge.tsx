import { cn, formatContrastRatio } from "../lib/utils";

interface ContrastBadgeProps {
  ratio: number;
  passes: boolean;
  className?: string;
}

export function ContrastBadge({ ratio, passes, className }: ContrastBadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
        passes
          ? "bg-emerald-500/20 text-emerald-400"
          : "bg-red-500/20 text-red-400",
        className,
      )}
      data-testid="contrast-badge"
    >
      {formatContrastRatio(ratio)}
      {passes ? " ✓ AA" : " ✗ Fail"}
    </span>
  );
}
