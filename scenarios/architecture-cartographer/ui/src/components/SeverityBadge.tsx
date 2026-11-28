import { AlertOctagon, AlertTriangle, Circle, Info, ShieldAlert } from "lucide-react";

import { Badge } from "./ui/badge";
import { selectors } from "../consts/selectors";
import { cn } from "../lib/utils";

export type SeverityLevel = "info" | "low" | "medium" | "high" | "critical";

interface Style {
  variant: "info" | "default" | "warning" | "danger";
  Icon: React.ComponentType<{ className?: string }>;
}

const STYLE_BY_LEVEL: Record<SeverityLevel, Style> = {
  info: { variant: "info", Icon: Info },
  low: { variant: "default", Icon: Circle },
  medium: { variant: "warning", Icon: AlertTriangle },
  high: { variant: "danger", Icon: AlertOctagon },
  critical: { variant: "danger", Icon: ShieldAlert },
};

export interface SeverityBadgeProps {
  level: SeverityLevel;
  /** Pre-translated text label; must accompany the icon (no color-only). */
  label: string;
  className?: string;
}

export function SeverityBadge({ level, label, className }: SeverityBadgeProps) {
  const { variant, Icon } = STYLE_BY_LEVEL[level];
  return (
    <Badge
      variant={variant}
      data-testid={selectors.shared.severityBadge.root({ level })}
      className={cn("uppercase tracking-wide", className)}
      aria-label={label}
    >
      <Icon className="h-3 w-3" aria-hidden="true" />
      <span>{label}</span>
    </Badge>
  );
}
