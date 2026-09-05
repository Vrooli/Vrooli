import { strings } from "../consts/strings";

export type AppRoute = "overview" | "graph" | "deployment" | "catalog" | "governance";

export interface RouteDefinition {
  readonly key: AppRoute;
  readonly path: string;
  readonly label: string;
}

export const routeDefinitions: readonly RouteDefinition[] = [
  { key: "overview", path: "/", label: strings.layout.nav.orientation },
  { key: "graph", path: "/graph", label: strings.layout.nav.graph },
  { key: "deployment", path: "/deployment", label: strings.layout.nav.deployment },
  { key: "catalog", path: "/catalog", label: strings.layout.nav.catalog },
  { key: "governance", path: "/governance", label: strings.layout.nav.governance }
];

const routeKeys = new Set<AppRoute>(routeDefinitions.map((route) => route.key));

export const isRouteKey = (value: string | null): value is AppRoute =>
  value !== null && routeKeys.has(value as AppRoute);

export const routePath = (routeKey: AppRoute) =>
  routeDefinitions.find((route) => route.key === routeKey)?.path ?? "/";
