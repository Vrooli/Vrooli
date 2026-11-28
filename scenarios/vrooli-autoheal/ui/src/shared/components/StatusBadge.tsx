// Status badge component with color coding
// [REQ:UI-HEALTH-002]
import type { HealthStatus } from "../../lib/api";
import { Badge } from "../ui/primitives";

interface StatusBadgeProps {
  status: HealthStatus;
}

const tones: Record<HealthStatus, "success" | "warning" | "danger" | "neutral"> = {
  ok: "success",
  warning: "warning",
  critical: "danger",
  "not-applicable": "neutral",
};

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <Badge tone={tones[status] ?? "neutral"} className="text-sm">
      {status === "not-applicable" ? "N/A" : status.toUpperCase()}
    </Badge>
  );
}
