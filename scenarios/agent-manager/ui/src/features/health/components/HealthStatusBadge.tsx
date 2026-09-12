import { Badge } from "../../../components/ui/badge";
import type { HealthStatus } from "../api/types";

interface HealthStatusBadgeProps {
  status: HealthStatus;
}

export function HealthStatusBadge({ status }: HealthStatusBadgeProps) {
  switch (status) {
    case "ok":
      return <Badge variant="success">OK</Badge>;
    case "failed":
      return <Badge variant="destructive">Failed</Badge>;
    case "unknown":
    default:
      return <Badge variant="secondary">Unknown</Badge>;
  }
}
