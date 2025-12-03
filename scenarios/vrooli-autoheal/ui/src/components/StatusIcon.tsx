// Status indicator icon component
// [REQ:UI-HEALTH-002]
import { CheckCircle, AlertTriangle, AlertCircle, Activity } from "lucide-react";
import type { HealthStatus } from "../lib/api";

interface StatusIconProps {
  status: HealthStatus;
  size?: number;
}

export function StatusIcon({ status, size = 20 }: StatusIconProps) {
  switch (status) {
    case "ok":
      return <CheckCircle className="text-emerald-500" size={size} />;
    case "warning":
      return <AlertTriangle className="text-amber-500" size={size} />;
    case "critical":
      return <AlertCircle className="text-red-500" size={size} />;
    default:
      return <Activity className="text-slate-400" size={size} />;
  }
}
