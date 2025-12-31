import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { Activity, AlertCircle, Database, GitGraph, Search } from "lucide-react";
import { Button } from "./components/ui/button";
import { fetchHealth } from "./lib/api";
import { SearchPanel } from "./components/SearchPanel";
import { MetricsPanel } from "./components/MetricsPanel";
import { selectors } from "./consts/selectors";

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

function TabLink({
  route,
  activeRoute,
  label,
  icon,
  testId,
}: {
  route: Route;
  activeRoute: Route;
  label: string;
  icon: ReactNode;
  testId?: string;
}) {
  const isActive = route === activeRoute;
  return (
      <a
        href={routeToHash(route)}
        data-testid={testId}
      className={["ko-tab", isActive ? "ko-tab-active" : "ko-tab-inactive"].join(" ")}
      aria-current={isActive ? "page" : undefined}
    >
      {icon}
      {label}
    </a>
  );
}

function PageShell({ children }: { children: ReactNode }) {
  return <main className="p-6 ko-stack">{children}</main>;
}

function FeatureCardLink({
  route,
  title,
  description,
  icon,
  badge,
  testId,
}: {
  route: Route;
  title: string;
  description: string;
  icon: ReactNode;
  badge?: string;
  testId?: string;
}) {
  return (
      <a
        href={routeToHash(route)}
        data-testid={testId}
      className="ko-panel ko-panel-inset p-6 hover:bg-green-950/30 hover:border-green-500/60 transition-all cursor-pointer text-left block"
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="mb-3">{icon}</div>
          <h3 className="ko-text-lg font-semibold mb-2">{title}</h3>
          <p className="ko-text-sm ko-muted">{description}</p>
        </div>
        {badge && (
          <span className="ko-tag">
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
    <div className="min-h-screen bg-gradient-to-b from-black via-green-950/30 to-black text-green-200">
      {/* Matrix-style header */}
      <header className="border-b border-green-800/60 bg-black/80 backdrop-blur px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Database className="h-8 w-8 text-green-500" />
            <div>
              <h1 className="text-2xl font-bold tracking-tight" data-testid={selectors.header.title}>
                Knowledge Observatory
              </h1>
              <p className="ko-text-sm ko-subtle">Consciousness Monitor • Semantic Intelligence System</p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div
              className="ko-card flex items-center gap-2 px-3 py-1.5 font-mono"
              data-testid={selectors.header.statusBadge}
            >
              <Activity className={`h-4 w-4 ${isLoading ? "animate-pulse" : "opacity-80"}`} />
              <span className="ko-text-xs font-semibold uppercase tracking-wider text-green-200">
                {isLoading ? "Syncing..." : error ? "Offline" : "Online"}
              </span>
            </div>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <TabLink
              route="dashboard"
              activeRoute={route}
              label="Dashboard"
              icon={<Activity className="h-4 w-4" />}
              testId={selectors.nav.dashboard}
            />
            <TabLink
              route="search"
              activeRoute={route}
              label="Search"
              icon={<Search className="h-4 w-4" />}
              testId={selectors.nav.search}
            />
            <TabLink
              route="graph"
              activeRoute={route}
              label="Graph"
              icon={<GitGraph className="h-4 w-4" />}
              testId={selectors.nav.graph}
            />
            <TabLink
              route="metrics"
              activeRoute={route}
              label="Metrics"
              icon={<Database className="h-4 w-4" />}
              testId={selectors.nav.metrics}
            />
          </div>
          <div className="ko-meta" data-testid={selectors.header.pageTitle}>
            {pageTitle}
          </div>
        </div>
      </header>

      {route === "dashboard" && (
        <PageShell>
          <section className="ko-panel ko-section" data-testid={selectors.dashboard.quickActions}>
            <div className="flex items-center justify-between gap-4 flex-wrap">
              <div>
                <h2 className="ko-text-lg font-semibold">Start a Knowledge Check</h2>
                <p className="ko-text-sm ko-muted mt-1">
                  Jump into the workflows operators use most: search, assess, and explore.
                </p>
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  asChild
                  className="ko-button-primary"
                  data-testid={selectors.dashboard.quickSearch}
                >
                  <a href={routeToHash("search")}>Run a Search</a>
                </Button>
                <Button
                  asChild
                  className="ko-button-secondary"
                  data-testid={selectors.dashboard.quickMetrics}
                >
                  <a href={routeToHash("metrics")}>Review Metrics</a>
                </Button>
                <Button
                  asChild
                  className="ko-button-secondary"
                  data-testid={selectors.dashboard.quickGraph}
                >
                  <a href={routeToHash("graph")}>Explore Graph</a>
                </Button>
              </div>
            </div>
          </section>

          {/* API Health Status */}
          <section className="ko-panel ko-section" data-testid={selectors.dashboard.healthSection}>
            <div className="flex items-center gap-2 mb-4">
              <Activity className="h-5 w-5" />
              <h2 className="ko-text-lg font-semibold">System Health</h2>
            </div>

            {isLoading && (
              <div className="ko-stack-xs">
                <div className="ko-loading-bar"></div>
                <p className="ko-text-sm ko-muted">Querying knowledge base status...</p>
              </div>
            )}

            {error && (
              <div
                className="ko-alert ko-alert-danger"
                data-testid={selectors.dashboard.healthError}
              >
                <AlertCircle className="h-5 w-5 text-red-400 mt-0.5" />
                <div>
                  <p className="text-red-300 ko-alert-title">Connection Error</p>
                  <p className="ko-text-sm text-red-600 mt-1">
                    Unable to reach the API. Confirm the scenario is running, then refresh.
                  </p>
                </div>
              </div>
            )}

            {data && (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                <div className="ko-card p-4">
                  <p className="ko-meta">Status</p>
                  <p className="text-xl font-bold mt-1">{data.status}</p>
                </div>
                <div className="ko-card p-4">
                  <p className="ko-meta">Service</p>
                  <p className="text-xl font-bold mt-1">{data.service}</p>
                </div>
                <div className="ko-card p-4">
                  <p className="ko-meta">Last Update</p>
                  <p className="ko-text-sm font-semibold mt-1">{new Date(data.timestamp).toLocaleTimeString()}</p>
                </div>
              </div>
            )}

            <Button
              className="mt-4 ko-button-primary"
              onClick={() => refetch()}
              data-testid={selectors.dashboard.healthRefresh}
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
              testId={selectors.dashboard.featureSearch}
            />
            <FeatureCardLink
              route="graph"
              title="Knowledge Graph"
              description="Explore semantic relationships and concept connections"
              icon={<GitGraph className="h-8 w-8 text-green-500" />}
              badge="Preview"
              testId={selectors.dashboard.featureGraph}
            />
            <FeatureCardLink
              route="metrics"
              title="Quality Metrics"
              description="Monitor coherence, freshness, and redundancy scores"
              icon={<Database className="h-8 w-8 text-green-500" />}
              testId={selectors.dashboard.featureMetrics}
            />
          </section>

          {/* Quick Start Info */}
          <section className="ko-panel ko-section" data-testid={selectors.dashboard.cliSection}>
            <h2 className="ko-text-lg font-semibold mb-4">CLI Commands</h2>
            <div className="ko-stack-xs ko-text-sm">
              <div className="ko-card p-3">
                <code className="ko-code">knowledge-observatory search "your query"</code>
                <p className="ko-subtle ko-text-xs mt-1">Semantic search across knowledge base</p>
              </div>
              <div className="ko-card p-3">
                <code className="ko-code">knowledge-observatory health --watch</code>
                <p className="ko-subtle ko-text-xs mt-1">Real-time health monitoring</p>
              </div>
              <div className="ko-card p-3">
                <code className="ko-code">knowledge-observatory graph --center "concept"</code>
                <p className="ko-subtle ko-text-xs mt-1">Generate knowledge relationship graph</p>
              </div>
            </div>
          </section>
        </PageShell>
      )}

      {route === "search" && (
        <PageShell>
          <section className="ko-panel ko-section">
            <div className="flex items-center gap-2 mb-2">
              <Search className="h-5 w-5 text-green-500" />
              <h2 className="ko-text-lg font-semibold">Semantic Search</h2>
            </div>
            <p className="ko-text-sm ko-muted mb-4">
              Ask natural-language questions to locate related knowledge across all collections.
            </p>
            <SearchPanel />
          </section>
        </PageShell>
      )}

      {route === "metrics" && (
        <PageShell>
          <section className="ko-panel ko-section">
            <div className="flex items-center gap-2 mb-2">
              <Database className="h-5 w-5 text-green-500" />
              <h2 className="ko-text-lg font-semibold">Quality Metrics</h2>
            </div>
            <p className="ko-text-sm ko-muted mb-4">
              Track coherence, freshness, redundancy, and coverage to spot drift or gaps.
            </p>
            <MetricsPanel />
          </section>
        </PageShell>
      )}

      {route === "graph" && (
        <PageShell>
          <section className="ko-panel ko-section">
            <div className="flex items-center gap-2 mb-2">
              <GitGraph className="h-5 w-5 text-green-500" />
              <h2 className="ko-text-lg font-semibold">Knowledge Graph</h2>
            </div>
            <p className="ko-text-sm ko-muted mb-4">
              Visualize how concepts connect and where semantic clusters emerge.
            </p>
            <div
              className="ko-card text-center p-12"
              data-testid={selectors.graph.emptyState}
            >
              <GitGraph className="h-16 w-16 text-green-600 mx-auto mb-4" />
              <p className="ko-muted">Graph visualization UI is not implemented yet</p>
              <p className="ko-text-sm ko-subtle mt-2">
                This page is reserved for exploring semantic relationships once the graph API + UI are wired.
              </p>
            </div>
          </section>
        </PageShell>
      )}
    </div>
  );
}
