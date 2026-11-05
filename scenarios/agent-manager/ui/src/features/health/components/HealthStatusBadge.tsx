import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { HealthStatus } from "../api/types";

interface HealthStatusBadgeProps {
  status: HealthStatus;
}

export function HealthStatusBadge({ status }: HealthStatusBadgeProps) {
  switch (status) {
    case "ok":
      return <StatusBadge tone="success">OK</StatusBadge>;
    case "failed":
      return <StatusBadge tone="danger">Failed</StatusBadge>;
    case "unknown":
    default:
      return <StatusBadge tone="neutral">Unknown</StatusBadge>;
  }
}
