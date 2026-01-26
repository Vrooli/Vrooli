// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
export type Route = "dashboard" | "search" | "metrics" | "graph";

const ROUTE_HASHES: Record<Route, string> = {
  dashboard: "#/",
  search: "#/search",
  metrics: "#/metrics",
  graph: "#/graph",
};

const ROUTE_TITLES: Record<Route, string> = {
  dashboard: "Dashboard",
  search: "Semantic Search",
  metrics: "Quality Metrics",
  graph: "Knowledge Graph",
};

export function parseRouteFromHash(hash: string): Route {
  const value = hash.replace(/^#\/?/, "").trim().toLowerCase();
  if (!value) return "dashboard";
  if (value.startsWith("search")) return "search";
  if (value.startsWith("metrics")) return "metrics";
  if (value.startsWith("graph")) return "graph";
  return "dashboard";
}

export function routeToHash(route: Route): string {
  return ROUTE_HASHES[route];
}

export function getPageTitle(route: Route): string {
  return ROUTE_TITLES[route];
}
