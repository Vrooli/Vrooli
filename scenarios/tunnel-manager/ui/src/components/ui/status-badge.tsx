import { CheckCircle2, XCircle, AlertTriangle, MinusCircle, CircleDot } from "lucide-react";
import { cn } from "../../lib/utils";
import type { StatusVariant } from "../../lib/utils";

const variantStyles: Record<StatusVariant, string> = {
  success: "bg-green-500/10 text-green-400",
  error: "bg-red-500/10 text-red-400",
  warning: "bg-yellow-500/10 text-yellow-400",
  neutral: "bg-slate-500/10 text-slate-300",
  info: "bg-blue-500/10 text-blue-400",
};

const variantIcons: Record<StatusVariant, React.ElementType> = {
  success: CheckCircle2,
  error: XCircle,
  warning: AlertTriangle,
  neutral: MinusCircle,
  info: CircleDot,
};

interface StatusBadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant: StatusVariant;
  label: string;
}

export function StatusBadge({ variant, label, className, ...rest }: StatusBadgeProps) {
  const Icon = variantIcons[variant];
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
        variantStyles[variant],
        className,
      )}
      {...rest}
    >
      <Icon className="h-3 w-3 shrink-0" aria-hidden="true" />
      {label}
    </span>
  );
}
