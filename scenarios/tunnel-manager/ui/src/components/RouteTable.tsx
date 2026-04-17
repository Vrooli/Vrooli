import { useState, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Globe, ExternalLink, ChevronRight, Plus, Search } from "lucide-react";
import { fetchRoutes, type Route } from "../lib/api";
import { useSort } from "../lib/utils";
import { Button } from "./ui/button";
import { StatusBadge } from "./ui/status-badge";
import { SortHeader } from "./ui/sort-header";
import { RefreshButton } from "./ui/refresh-button";
import { QueryState } from "./ui/query-state";
import { EmptyState } from "./ui/empty-state";

type RouteSortField = "subdomain" | "scenario_name" | "local_port" | "enabled";

const compareRoutes = (a: Route, b: Route, field: RouteSortField): number => {
  switch (field) {
    case "subdomain": return a.subdomain.localeCompare(b.subdomain);
    case "scenario_name": return a.scenario_name.localeCompare(b.scenario_name);
    case "local_port": return a.local_port - b.local_port;
    case "enabled": return Number(b.enabled) - Number(a.enabled);
  }
};

function RouteCard({ route }: { route: Route }) {
  return (
    <Link
      to={`/routes/${route.id}`}
      className="flex items-center gap-3 rounded-lg bg-black/20 p-3 transition-colors hover:bg-white/5"
      data-testid={`route-row-${route.subdomain}`}
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium text-slate-200 truncate">{route.subdomain}</span>
          <StatusBadge
            variant={route.enabled ? "success" : "neutral"}
            label={route.enabled ? "enabled" : "disabled"}
          />
        </div>
        <div className="mt-1 flex items-center gap-3 text-xs text-slate-300">
          <span>{route.scenario_name}</span>
          <span className="font-mono">:{route.local_port}</span>
        </div>
      </div>
      <ChevronRight className="h-4 w-4 shrink-0 text-slate-500" aria-hidden="true" />
    </Link>
  );
}

export function RouteTable() {
  const { data: routes, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["routes"],
    queryFn: fetchRoutes,
    refetchInterval: 30000,
  });

  const [search, setSearch] = useState("");
  const { sorted, sortField, sortDir, toggleSort } = useSort(routes, "subdomain" as RouteSortField, compareRoutes);

  const filtered = useMemo(() => {
    if (!search.trim()) return sorted;
    const q = search.toLowerCase();
    return sorted.filter(
      (r) =>
        r.subdomain.toLowerCase().includes(q) ||
        r.scenario_name.toLowerCase().includes(q) ||
        String(r.local_port).includes(q),
    );
  }, [sorted, search]);

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4 sm:p-6" data-testid="route-manifest-panel">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Globe className="h-5 w-5 text-slate-300" aria-hidden="true" />
          <h2 className="text-lg font-semibold">Route Manifest</h2>
        </div>
        <RefreshButton onClick={() => refetch()} disabled={isFetching} aria-label="Refresh routes" />
      </div>

      <QueryState
        isLoading={isLoading}
        error={error}
        refetch={refetch}
        loadingLabel="Loading routes..."
        errorLabel="Failed to load routes"
        skeleton={
          <div className="space-y-2">
            {[1, 2, 3].map((i) => <div key={i} className="h-14 rounded-lg bg-white/5" />)}
          </div>
        }
      >
        {routes && routes.length === 0 && (
          <EmptyState
            icon={Globe}
            title="No routes configured"
            description="Routes map subdomains to local scenario ports. Add one to get started."
            className="mt-4"
          >
            <Link to="/routes" className="mt-3 inline-block" data-testid="routes-empty-cta">
              <Button size="sm">
                <Plus className="h-4 w-4 mr-2" />
                Add Route
              </Button>
            </Link>
          </EmptyState>
        )}

        {/* Search bar - shown when routes exist */}
        {routes && routes.length > 0 && (
          <div className="relative mt-4" data-testid="routes-search">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" aria-hidden="true" />
            <input
              type="search"
              placeholder="Filter by subdomain, scenario, or port..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-black/20 py-2 pl-9 pr-3 text-sm text-slate-200 placeholder-slate-500 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500/30 transition-colors"
              aria-label="Filter routes"
            />
          </div>
        )}

        {/* Result count */}
        {routes && routes.length > 0 && search.trim() && filtered.length > 0 && (
          <p className="mt-2 text-xs text-slate-500" role="status" aria-live="polite">{filtered.length} of {routes.length} routes</p>
        )}

        {/* No results from filter */}
        {routes && routes.length > 0 && filtered.length === 0 && search.trim() && (
          <div className="mt-3 rounded-lg border border-dashed border-white/10 p-4 text-center">
            <p className="text-sm text-slate-300">No routes match "{search}"</p>
          </div>
        )}

        {/* Mobile: card layout */}
        {filtered.length > 0 && (
          <div className="mt-3 flex flex-col gap-2 sm:hidden" role="list" aria-label="Route cards" data-testid="route-table">
            {filtered.map((route: Route) => (
              <RouteCard key={route.id} route={route} />
            ))}
          </div>
        )}

        {/* Desktop: table layout */}
        {filtered.length > 0 && (
          <div className="mt-3 hidden sm:block overflow-x-auto" data-testid="route-table">
            <table className="w-full text-sm">
              <caption className="sr-only">Configured tunnel routes</caption>
              <thead>
                <tr className="border-b border-white/10 text-left text-slate-300">
                  <th scope="col" className="pb-2 pr-4"><SortHeader field="subdomain" label="Subdomain" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                  <th scope="col" className="pb-2 pr-4"><SortHeader field="scenario_name" label="Scenario" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                  <th scope="col" className="pb-2 pr-4"><SortHeader field="local_port" label="Port" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                  <th scope="col" className="pb-2 pr-4"><SortHeader field="enabled" label="Status" current={sortField} dir={sortDir} onToggle={toggleSort} /></th>
                  <th scope="col" className="pb-2">Public URL</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((route: Route) => (
                  <tr key={route.id} className="border-b border-white/5" data-testid={`route-row-${route.subdomain}`}>
                    <td className="py-2 pr-4 font-medium text-slate-200">
                      <Link to={`/routes/${route.id}`} className="hover:text-blue-400 underline">
                        {route.subdomain}
                      </Link>
                    </td>
                    <td className="py-2 pr-4 text-slate-300">{route.scenario_name}</td>
                    <td className="py-2 pr-4 font-mono text-slate-300">{route.local_port}</td>
                    <td className="py-2 pr-4">
                      <StatusBadge
                        variant={route.enabled ? "success" : "neutral"}
                        label={route.enabled ? "enabled" : "disabled"}
                      />
                    </td>
                    <td className="py-2 text-slate-300 max-w-[200px] truncate" title={route.public_url || undefined}>
                      {route.public_url ? (
                        <a href={route.public_url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 hover:text-slate-200">
                          <span className="truncate">{route.public_url}</span>
                          <ExternalLink className="h-3 w-3 shrink-0" aria-hidden="true" />
                          <span className="sr-only">(opens in new tab)</span>
                        </a>
                      ) : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </QueryState>
    </div>
  );
}
