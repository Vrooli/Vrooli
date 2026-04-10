import { ArrowRight, Shield } from "lucide-react";
import { Button } from "./ui/button";
import type { Health } from "../lib/api";

export function HealthStatus({
  data,
  isLoading,
  error,
  onRefresh
}: {
  data?: Health;
  isLoading: boolean;
  error: Error | null;
  onRefresh: () => void;
}) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/5 p-5 backdrop-blur">
      <div className="flex items-center gap-2 text-slate-200">
        <Shield className="h-4 w-4" />
        <p className="text-sm font-medium">API Health</p>
      </div>
      {isLoading && <p className="mt-2 text-sm text-slate-300">Checking...</p>}
      {error && (
        <p className="mt-2 text-sm text-red-300">
          Unable to reach the API. Start the scenario with `vrooli scenario start scenario-stack-governor`.
        </p>
      )}
      {data && (
        <p className="mt-2 text-sm text-slate-200">
          <span className="text-slate-400">Status:</span> {data.status}
        </p>
      )}
      <Button className="mt-3" variant="outline" size="sm" onClick={onRefresh}>
        Refresh <ArrowRight className="ml-2 h-4 w-4" />
      </Button>
    </div>
  );
}
