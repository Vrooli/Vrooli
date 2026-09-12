import { cva, type VariantProps } from "class-variance-authority";
import {
  AlertTriangle,
  CheckCircle2,
  CircleDashed,
  Loader2,
  XCircle,
  type LucideIcon,
} from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

const pillVariants = cva(
  "inline-flex items-center gap-1.5 rounded-pill px-2.5 py-1 text-xs font-medium",
  {
    variants: {
      status: {
        ok: "bg-app-success/15 text-app-success",
        warn: "bg-app-warning/15 text-app-warning",
        error: "bg-app-danger/15 text-app-danger",
        info: "bg-app-info/15 text-app-info",
        pending: "bg-app-surface-muted text-app-muted-foreground",
        running: "bg-app-info/15 text-app-info",
      },
    },
    defaultVariants: { status: "info" },
  },
);

const DEFAULT_ICONS: Record<NonNullable<VariantProps<typeof pillVariants>["status"]>, LucideIcon> = {
  ok: CheckCircle2,
  warn: AlertTriangle,
  error: XCircle,
  info: CircleDashed,
  pending: CircleDashed,
  running: Loader2,
};

export interface StatusPillProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof pillVariants> {
  label: string;
  icon?: LucideIcon;
}

export const StatusPill = React.forwardRef<HTMLSpanElement, StatusPillProps>(function StatusPill(
  { className, status = "info", label, icon, ...props },
  ref,
) {
  const Icon = icon ?? DEFAULT_ICONS[status ?? "info"];
  const spin = status === "running" && !icon;
  return (
    <span ref={ref} className={cn(pillVariants({ status, className }))} {...props}>
      <Icon aria-hidden className={cn("h-3.5 w-3.5", spin && "animate-spin")} />
      <span>{label}</span>
    </span>
  );
});
