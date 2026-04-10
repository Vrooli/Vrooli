import { CheckCircle, AlertCircle, Circle } from "lucide-react";
import type { HealthResponse } from "../lib/api";

interface HealthIndicatorProps {
  health?: HealthResponse;
  isHealthy: boolean;
}

export function HealthIndicator({ health, isHealthy }: HealthIndicatorProps) {
  return (
    <div
      className="flex items-center gap-2"
      data-testid="health-status"
      title={isHealthy ? "All systems healthy" : "System issues detected"}
    >
      {isHealthy ? (
        <CheckCircle className="h-4 w-4 text-emerald-500" />
      ) : health ? (
        <AlertCircle className="h-4 w-4 text-amber-500" />
      ) : (
        <Circle className="h-4 w-4 text-slate-600" />
      )}
    </div>
  );
}
