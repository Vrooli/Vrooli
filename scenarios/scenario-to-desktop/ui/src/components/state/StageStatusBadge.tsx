/**
 * Badge component showing pipeline stage status.
 * Displays valid/stale/invalid/none status with appropriate styling.
 */

import { Badge } from "../ui/badge";
import { CheckCircle, AlertTriangle, XCircle, Circle } from "lucide-react";
import { cn } from "../../lib/utils";
import type { StageStatus } from "../../lib/api";

export type StageStatusType = "valid" | "stale" | "invalid" | "none";

interface StageStatusBadgeProps {
  status: StageStatusType;
  stageName?: string;
  showLabel?: boolean;
  size?: "sm" | "md";
  className?: string;
}

const statusConfig: Record<
  StageStatusType,
  {
    label: string;
    variant: "success" | "warning" | "error" | "secondary";
    Icon: typeof CheckCircle;
  }
> = {
  valid: {
    label: "Valid",
    variant: "success",
    Icon: CheckCircle,
  },
  stale: {
    label: "Stale",
    variant: "warning",
    Icon: AlertTriangle,
  },
  invalid: {
    label: "Failed",
    variant: "error",
    Icon: XCircle,
  },
  none: {
    label: "Pending",
    variant: "secondary",
    Icon: Circle,
  },
};

export function StageStatusBadge({
  status,
  stageName,
  showLabel = true,
  size = "md",
  className,
}: StageStatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.none;
  const { label, variant, Icon } = config;

  const iconSize = size === "sm" ? "h-3 w-3" : "h-3.5 w-3.5";
  const textSize = size === "sm" ? "text-[10px]" : "text-xs";

  return (
    <Badge
      variant={variant}
      className={cn("gap-1.5", textSize, className)}
    >
      <Icon className={iconSize} />
      {showLabel && <span>{stageName ? `${stageName}: ${label}` : label}</span>}
    </Badge>
  );
}

interface StageStatusFromApiProps {
  stageStatus: StageStatus | undefined;
  stageName?: string;
  showLabel?: boolean;
  size?: "sm" | "md";
  className?: string;
}

/**
 * Convenience wrapper that accepts StageStatus from the API.
 */
export function StageStatusBadgeFromApi({
  stageStatus,
  stageName,
  showLabel = true,
  size = "md",
  className,
}: StageStatusFromApiProps) {
  const status: StageStatusType = (stageStatus?.status as StageStatusType) || "none";

  return (
    <StageStatusBadge
      status={status}
      stageName={stageName}
      showLabel={showLabel}
      size={size}
      className={className}
    />
  );
}
