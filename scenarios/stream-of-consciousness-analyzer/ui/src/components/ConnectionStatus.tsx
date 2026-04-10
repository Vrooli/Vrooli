// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// Displays API connection state. Shows a banner when the API is unreachable
// so users understand why mutations fail rather than seeing silent errors.
import { useQuery } from "@tanstack/react-query";
import { WifiOff } from "lucide-react";
import { fetchHealth } from "../lib/api";
import { HEALTH_POLL_MS, HEALTH_DEGRADED_POLL_MS } from "../lib/config";

export function ConnectionStatus() {
  const { isError } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: (query) =>
      query.state.status === "error" ? HEALTH_DEGRADED_POLL_MS : HEALTH_POLL_MS,
    retry: 1,
  });

  if (!isError) return null;

  return (
    <div
      data-testid="connection-status"
      role="status"
      className="flex items-center gap-2 px-3 py-1.5 bg-yellow-900/30 border-b border-yellow-500/30 text-xs text-yellow-300"
    >
      <WifiOff className="h-3.5 w-3.5 shrink-0" />
      <span>API is unreachable. Changes may not save until the connection is restored.</span>
    </div>
  );
}
