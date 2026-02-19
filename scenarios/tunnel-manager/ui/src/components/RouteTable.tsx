import { useQuery } from "@tanstack/react-query";
import { Globe, RefreshCw } from "lucide-react";
import { fetchRoutes, type Route } from "../lib/api";
import { Button } from "./ui/button";

export function RouteTable() {
  const { data: routes, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["routes"],
    queryFn: fetchRoutes,
    refetchInterval: 30000,
  });

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Globe className="h-5 w-5 text-slate-400" />
          <h2 className="text-lg font-semibold">Route Manifest</h2>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isFetching}>
          <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
        </Button>
      </div>

      {isLoading && <p className="mt-4 text-slate-400">Loading routes...</p>}
      {error && <p className="mt-4 text-red-400">Failed to load routes</p>}

      {routes && routes.length === 0 && (
        <p className="mt-4 text-slate-400">No routes configured.</p>
      )}

      {routes && routes.length > 0 && (
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-white/10 text-left text-slate-400">
                <th className="pb-2 pr-4">Subdomain</th>
                <th className="pb-2 pr-4">Scenario</th>
                <th className="pb-2 pr-4">Port</th>
                <th className="pb-2 pr-4">Status</th>
                <th className="pb-2">Public URL</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((route: Route) => (
                <tr key={route.id} className="border-b border-white/5">
                  <td className="py-2 pr-4 font-medium text-slate-200">{route.subdomain}</td>
                  <td className="py-2 pr-4 text-slate-300">{route.scenario_name}</td>
                  <td className="py-2 pr-4 font-mono text-slate-300">{route.local_port}</td>
                  <td className="py-2 pr-4">
                    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${
                      route.enabled
                        ? "bg-green-500/10 text-green-400"
                        : "bg-slate-500/10 text-slate-400"
                    }`}>
                      {route.enabled ? "enabled" : "disabled"}
                    </span>
                  </td>
                  <td className="py-2 text-slate-400">
                    {route.public_url ? (
                      <a href={route.public_url} target="_blank" rel="noopener noreferrer" className="hover:text-slate-200 underline">
                        {route.public_url}
                      </a>
                    ) : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
