// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
// [REQ:P2-001] Provider resilience status display
import { useQuery } from "@tanstack/react-query";
import { Activity, AlertTriangle } from "lucide-react";
import { listProviders } from "../lib/api";
import { PROVIDER_POLL_MS } from "../lib/config";

export function ProviderStatus() {
  const { data: providers = [], isLoading, isError } = useQuery({
    queryKey: ["providers"],
    queryFn: listProviders,
    refetchInterval: PROVIDER_POLL_MS,
    retry: 2,
  });

  if (isLoading) return null;

  if (isError) {
    return (
      <div data-testid="provider-status" className="flex items-center gap-1.5 px-2 py-1 text-xs">
        <AlertTriangle className="h-3 w-3 text-red-400" />
        <span className="text-red-400">API offline</span>
      </div>
    );
  }

  const active = providers.find((p) => p.active);

  return (
    <div data-testid="provider-status" className="flex items-center gap-1.5 px-2 py-1 text-xs">
      <Activity className="h-3 w-3" />
      <span className={active ? "text-green-400" : "text-yellow-500"}>
        {active ? active.name : "No LLM"}
      </span>
      {providers.length > 1 && (
        <span className="text-slate-600">
          (+{providers.length - 1} fallback)
        </span>
      )}
    </div>
  );
}
