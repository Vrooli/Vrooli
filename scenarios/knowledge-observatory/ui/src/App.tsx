import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, Database, GitGraph, Search } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchHealth } from "./lib/api";
import { SearchPanel } from "./components/SearchPanel";
import { MetricsPanel } from "./components/MetricsPanel";

type Route = "dashboard" | "search" | "metrics" | "graph";

function parseRouteFromHash(hash: string): Route {
  const value = hash.replace(/^#\/?/, "").trim().toLowerCase();
  if (!value) return "dashboard";
  if (value.startsWith("search")) return "search";
  if (value.startsWith("metrics")) return "metrics";
  if (value.startsWith("graph")) return "graph";
  return "dashboard";
}

function routeToHash(route: Route): string {
  switch (route) {
    case "dashboard":
      return "#/";
    case "search":
      return "#/search";
    case "metrics":
      return "#/metrics";
    case "graph":
      return "#/graph";
  }
}

function TabLink({ route, activeRoute, label, icon }: { route: Route; activeRoute: Route; label: string; icon: ReactNode }) {
  const isActive = route === activeRoute;
  return (
    <a
      href={routeToHash(route)}
      className={[
        "inline-flex items-center gap-2 px-3 py-2 rounded border transition-colors text-sm font-semibold",
        isActive
          ? "bg-green-950/40 border-green-700 text-green-200"
          : "bg-black/30 border-green-900/50 text-green-500 hover:bg-green-950/20 hover:border-green-700",
      ].join(" ")}
      aria-current={isActive ? "page" : undefined}
    >
      {icon}
      {label}
    </a>
  );
}

function PageShell({ children }: { children: ReactNode }) {
  return <main className="p-6 space-y-6">{children}</main>;
}

function FeatureCardLink({
  route,
  title,
  description,
  icon,
  badge,
}: {
  route: Route;
  title: string;
  description: string;
  icon: ReactNode;
  badge?: string;
}) {
  return (
    <a
      href={routeToHash(route)}
      className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg hover:bg-green-950/20 hover:border-green-700 transition-all cursor-pointer text-left block"
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="mb-3">{icon}</div>
          <h3 className="text-lg font-semibold mb-2">{title}</h3>
          <p className="text-sm text-green-600">{description}</p>
        </div>
        {badge && (
          <span className="text-xs text-green-500 border border-green-900/50 bg-black/40 px-2 py-1 rounded uppercase tracking-wider">
            {badge}
          </span>
        )}
      </div>
    </a>
  );
}

export default function App() {
  const [route, setRoute] = useState<Route>(() => parseRouteFromHash(window.location.hash));
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["health"],
    queryFn: fetchHealth,
    refetchInterval: 5000
  });

  useEffect(() => {
    const onHashChange = () => setRoute(parseRouteFromHash(window.location.hash));
    window.addEventListener("hashchange", onHashChange);
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const pageTitle = useMemo(() => {
    switch (route) {
      case "dashboard":
        return "Dashboard";
      case "search":
        return "Semantic Search";
      case "metrics":
        return "Quality Metrics";
      case "graph":
        return "Knowledge Graph";
    }
  }, [route]);

  return (
    <div className="min-h-screen bg-black text-green-400 font-mono">
      {/* Matrix-style header */}
      <header className="border-b border-green-900/50 bg-black/90 backdrop-blur px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Database className="h-8 w-8 text-green-500" />
            <div>
              <h1 className="text-2xl font-bold tracking-tight">Knowledge Observatory</h1>
              <p className="text-sm text-green-600">Consciousness Monitor • Semantic Intelligence System</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2 px-3 py-1.5 rounded bg-green-950/30 border border-green-900/50">
              <Activity className="h-4 w-4 animate-pulse" />
              <span className="text-xs font-semibold uppercase tracking-wider">
                {isLoading ? "Syncing..." : error ? "Offline" : "Online"}
              </span>
            </div>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <TabLink route="dashboard" activeRoute={route} label="Dashboard" icon={<Activity className="h-4 w-4" />} />
            <TabLink route="search" activeRoute={route} label="Search" icon={<Search className="h-4 w-4" />} />
            <TabLink route="graph" activeRoute={route} label="Graph" icon={<GitGraph className="h-4 w-4" />} />
            <TabLink route="metrics" activeRoute={route} label="Metrics" icon={<Database className="h-4 w-4" />} />
          </div>
          <div className="text-xs text-green-700 uppercase tracking-wider">{pageTitle}</div>
        </div>
      </header>

      {route === "dashboard" && (
        <PageShell>
          {/* API Health Status */}
          <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
            <div className="flex items-center gap-2 mb-4">
              <Activity className="h-5 w-5" />
              <h2 className="text-lg font-semibold">System Health</h2>
            </div>

            {isLoading && (
              <div className="space-y-2">
                <div className="h-2 bg-green-900/30 animate-pulse rounded"></div>
                <p className="text-sm text-green-600">Querying knowledge base status...</p>
              </div>
            )}

            {error && (
              <div className="p-4 border border-red-900/50 bg-red-950/20 rounded">
                <p className="text-red-400 font-semibold">⚠ Connection Error</p>
                <p className="text-sm text-red-600 mt-1">
                  Unable to establish connection to API. Verify scenario is running.
                </p>
              </div>
            )}

            {data && (
              <div className="grid grid-cols-3 gap-4">
                <div className="border border-green-900/50 bg-black/40 p-4 rounded">
                  <p className="text-xs text-green-600 uppercase tracking-wider">Status</p>
                  <p className="text-xl font-bold mt-1">{data.status}</p>
                </div>
                <div className="border border-green-900/50 bg-black/40 p-4 rounded">
                  <p className="text-xs text-green-600 uppercase tracking-wider">Service</p>
                  <p className="text-xl font-bold mt-1">{data.service}</p>
                </div>
                <div className="border border-green-900/50 bg-black/40 p-4 rounded">
                  <p className="text-xs text-green-600 uppercase tracking-wider">Last Update</p>
                  <p className="text-sm font-semibold mt-1">{new Date(data.timestamp).toLocaleTimeString()}</p>
                </div>
              </div>
            )}

            <Button
              className="mt-4 bg-green-900 hover:bg-green-800 text-green-100 border border-green-700"
              onClick={() => refetch()}
            >
              Refresh Status
            </Button>
          </section>

          {/* Feature Overview */}
          <section className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <FeatureCardLink
              route="search"
              title="Semantic Search"
              description="Query knowledge base using natural language across all collections"
              icon={<Search className="h-8 w-8 text-green-500" />}
            />
            <FeatureCardLink
              route="graph"
              title="Knowledge Graph"
              description="Explore semantic relationships and concept connections"
              icon={<GitGraph className="h-8 w-8 text-green-500" />}
              badge="Preview"
            />
            <FeatureCardLink
              route="metrics"
              title="Quality Metrics"
              description="Monitor coherence, freshness, and redundancy scores"
              icon={<Database className="h-8 w-8 text-green-500" />}
            />
          </section>

          {/* Quick Start Info */}
          <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
            <h2 className="text-lg font-semibold mb-4">CLI Commands</h2>
            <div className="space-y-2 text-sm">
              <div className="bg-black/40 p-3 rounded border border-green-900/30">
                <code className="text-green-300">knowledge-observatory search "your query"</code>
                <p className="text-green-600 text-xs mt-1">Semantic search across knowledge base</p>
              </div>
              <div className="bg-black/40 p-3 rounded border border-green-900/30">
                <code className="text-green-300">knowledge-observatory health --watch</code>
                <p className="text-green-600 text-xs mt-1">Real-time health monitoring</p>
              </div>
              <div className="bg-black/40 p-3 rounded border border-green-900/30">
                <code className="text-green-300">knowledge-observatory graph --center "concept"</code>
                <p className="text-green-600 text-xs mt-1">Generate knowledge relationship graph</p>
              </div>
            </div>
          </section>
        </PageShell>
      )}

      {route === "search" && (
        <PageShell>
          <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
            <div className="flex items-center gap-2 mb-4">
              <Search className="h-5 w-5 text-green-500" />
              <h2 className="text-lg font-semibold">Semantic Search</h2>
            </div>
            <SearchPanel />
          </section>
        </PageShell>
      )}

      {route === "metrics" && (
        <PageShell>
          <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
            <div className="flex items-center gap-2 mb-4">
              <Database className="h-5 w-5 text-green-500" />
              <h2 className="text-lg font-semibold">Quality Metrics</h2>
            </div>
            <MetricsPanel />
          </section>
        </PageShell>
      )}

      {route === "graph" && (
        <PageShell>
          <section className="border border-green-900/50 bg-green-950/10 p-6 rounded-lg">
            <div className="flex items-center gap-2 mb-4">
              <GitGraph className="h-5 w-5 text-green-500" />
              <h2 className="text-lg font-semibold">Knowledge Graph</h2>
            </div>
            <div className="text-center p-12 border border-green-900/30 bg-black/40 rounded">
              <GitGraph className="h-16 w-16 text-green-700 mx-auto mb-4" />
              <p className="text-green-600">Graph visualization UI is not implemented yet</p>
              <p className="text-sm text-green-700 mt-2">
                This page is reserved for exploring semantic relationships once the graph API + UI are wired.
              </p>
            </div>
          </section>
        </PageShell>
      )}
    </div>
  );
}
