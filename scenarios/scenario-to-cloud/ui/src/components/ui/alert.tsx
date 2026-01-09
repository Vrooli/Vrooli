import { cva, type VariantProps } from "class-variance-authority";
import { AlertCircle, CheckCircle2, Info, AlertTriangle, X } from "lucide-react";
import { cn } from "../../lib/utils";

const alertVariants = cva(
  "relative rounded-lg border p-4 flex gap-3",
  {
    variants: {
      variant: {
        default: "bg-slate-800/50 border-slate-700 text-slate-200",
        success: "bg-emerald-500/10 border-emerald-500/30 text-emerald-200",
        warning: "bg-amber-500/10 border-amber-500/30 text-amber-200",
        error: "bg-red-500/10 border-red-500/30 text-red-200",
        info: "bg-blue-500/10 border-blue-500/30 text-blue-200",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

const iconMap = {
  default: Info,
  success: CheckCircle2,
  warning: AlertTriangle,
  error: AlertCircle,
  info: Info,
};

export interface AlertProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof alertVariants> {
  title?: string;
  onDismiss?: () => void;
}

export function Alert({
  className,
  variant = "default",
  title,
  children,
  onDismiss,
  ...props
}: AlertProps) {
  const Icon = iconMap[variant ?? "default"];

  return (
    <div className={cn(alertVariants({ variant }), className)} role="alert" {...props}>
      <Icon className="h-5 w-5 flex-shrink-0 mt-0.5" />
      <div className="flex-1 min-w-0">
        {title && (
          <h5 className="font-medium mb-1">{title}</h5>
        )}
        <div className="text-sm opacity-90">{children}</div>
      </div>
      {onDismiss && (
        <button
          onClick={onDismiss}
          className="flex-shrink-0 p-1 rounded hover:bg-white/10 transition-colors"
          aria-label="Dismiss"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
