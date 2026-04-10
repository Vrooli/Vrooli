// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
export type Route = "dashboard" | "search" | "explorer" | "viewer" | "metrics" | "graph" | "collection";

const ROUTE_HASHES: Record<Route, string> = {
  dashboard: "#/",
  search: "#/search",
  explorer: "#/explorer",
  viewer: "#/viewer",
  metrics: "#/metrics",
  graph: "#/graph",
  collection: "#/collections",
};

const ROUTE_TITLES: Record<Route, string> = {
  dashboard: "Dashboard",
  search: "Unified Search",
  explorer: "Scenario Explorer",
  viewer: "Document Viewer",
  metrics: "Quality Metrics",
  graph: "Knowledge Graph",
  collection: "Collection Details",
};

export function parseRouteFromHash(hash: string): Route {
  const value = hash.replace(/^#\/?/, "").trim().toLowerCase();
  if (!value) return "dashboard";
  if (value.startsWith("search")) return "search";
  if (value.startsWith("explorer")) return "explorer";
  if (value.startsWith("viewer")) return "viewer";
  if (value.startsWith("metrics")) return "metrics";
  if (value.startsWith("graph")) return "graph";
  if (value.startsWith("collections/")) return "collection";
  return "dashboard";
}

export function parseCollectionNameFromHash(hash: string): string {
  const value = hash.replace(/^#\/?/, "").trim();
  if (!value.toLowerCase().startsWith("collections/")) {
    return "";
  }
  const encoded = value.slice("collections/".length).trim();
  if (!encoded) {
    return "";
  }
  try {
    return decodeURIComponent(encoded);
  } catch {
    return encoded;
  }
}

export function routeToHash(route: Route): string {
  return ROUTE_HASHES[route];
}

export function collectionToHash(collectionName: string): string {
  return `#/collections/${encodeURIComponent(collectionName.trim())}`;
}

export function getPageTitle(route: Route): string {
  return ROUTE_TITLES[route];
}
