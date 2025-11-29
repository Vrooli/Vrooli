import { CheckCircle, AlertTriangle, XCircle } from "lucide-react";
import { cn } from "../lib/utils";

type HealthStatus = "ok" | "degraded" | "failed";

interface HealthBadgeProps {
  status: HealthStatus;
  label?: string;
  className?: string;
}

// [REQ:SCS-HEALTH-001] Health status badges for collectors
export function HealthBadge({ status, label, className }: HealthBadgeProps) {
  const config = {
    ok: {
      icon: CheckCircle,
      color: "text-emerald-400",
      bg: "bg-emerald-400/10",
      text: "OK",
    },
    degraded: {
      icon: AlertTriangle,
      color: "text-amber-400",
      bg: "bg-amber-400/10",
      text: "Degraded",
    },
    failed: {
      icon: XCircle,
      color: "text-red-400",
      bg: "bg-red-400/10",
      text: "Failed",
    },
  };

  const { icon: Icon, color, bg, text } = config[status] || config.failed;

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium",
        color,
        bg,
        className
      )}
      data-testid={`health-badge-${status}`}
    >
      <Icon className="h-3 w-3" />
      {label || text}
    </span>
  );
}
