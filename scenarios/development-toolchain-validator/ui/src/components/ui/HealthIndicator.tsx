import { CheckCircle, XCircle, AlertCircle } from "lucide-react";

// ─────────────────────────────────────────────────────────────────────────────
// Health Indicator Primitive
// [REQ:P0-001] Reference Scenario Registry - Health status display
// ─────────────────────────────────────────────────────────────────────────────
//
// A semantic status indicator for API health. Displays loading, connected,
// or disconnected states with appropriate colors and icons.
// ─────────────────────────────────────────────────────────────────────────────

export type HealthStatus = "loading" | "connected" | "disconnected";

interface HealthIndicatorProps {
  status: HealthStatus;
  isLoading?: boolean;
  testId?: string;
}

export function HealthIndicator({ status, isLoading, testId }: HealthIndicatorProps) {
  // If explicitly loading, override status
  const displayStatus = isLoading ? "loading" : status;

  const config = {
    loading: {
      icon: AlertCircle,
      text: "Checking...",
      colorClass: "text-amber-400"
    },
    connected: {
      icon: CheckCircle,
      text: "Connected",
      colorClass: "text-emerald-400"
    },
    disconnected: {
      icon: XCircle,
      text: "Disconnected",
      colorClass: "text-red-400"
    }
  } as const;

  const { icon: Icon, text, colorClass } = config[displayStatus];

  return (
    <div
      data-testid={testId}
      className="flex items-center gap-2 text-sm"
      role="status"
      aria-live="polite"
    >
      <Icon className={`h-4 w-4 ${colorClass}`} aria-hidden="true" />
      <span className={colorClass}>{text}</span>
    </div>
  );
}
