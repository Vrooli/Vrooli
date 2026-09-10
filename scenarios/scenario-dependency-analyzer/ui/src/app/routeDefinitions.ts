import { strings } from "../consts/strings";
import { Activity, Boxes, GitBranch, ShieldCheck, Waypoints, type LucideIcon } from "lucide-react";

export type AppRoute = "overview" | "graph" | "deployment" | "catalog" | "governance";

export interface RouteDefinition {
  readonly key: AppRoute;
  readonly path: string;
  readonly label: string;
  readonly icon: LucideIcon;
}

export const routeDefinitions: readonly RouteDefinition[] = [
  { key: "overview", path: "/", label: strings.layout.nav.orientation, icon: Activity },
  { key: "graph", path: "/graph", label: strings.layout.nav.graph, icon: GitBranch },
  { key: "deployment", path: "/deployment", label: strings.layout.nav.deployment, icon: Waypoints },
  { key: "catalog", path: "/catalog", label: strings.layout.nav.catalog, icon: Boxes },
  { key: "governance", path: "/governance", label: strings.layout.nav.governance, icon: ShieldCheck }
];

const routeKeys = new Set<AppRoute>(routeDefinitions.map((route) => route.key));

export const isRouteKey = (value: string | null): value is AppRoute =>
  value !== null && routeKeys.has(value as AppRoute);

export const routePath = (routeKey: AppRoute) =>
  routeDefinitions.find((route) => route.key === routeKey)?.path ?? "/";
