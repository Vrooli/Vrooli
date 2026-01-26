// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ReactNode } from "react";
import { Activity, Database, GitGraph, Search } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { routeToHash, type Route } from "../controllers/routeController";

const NAV_ITEMS: Array<{ route: Route; label: string; icon: ReactNode; testId: string }> = [
  {
    route: "dashboard",
    label: "Dashboard",
    icon: <Activity className="h-4 w-4" />,
    testId: selectors.nav.dashboard,
  },
  {
    route: "search",
    label: "Search",
    icon: <Search className="h-4 w-4" />,
    testId: selectors.nav.search,
  },
  {
    route: "graph",
    label: "Graph",
    icon: <GitGraph className="h-4 w-4" />,
    testId: selectors.nav.graph,
  },
  {
    route: "metrics",
    label: "Metrics",
    icon: <Database className="h-4 w-4" />,
    testId: selectors.nav.metrics,
  },
];

export type AppHeaderProps = {
  route: Route;
  pageTitle: string;
  statusPulse: boolean;
  statusLabel: string;
};

type TabLinkProps = {
  route: Route;
  activeRoute: Route;
  label: string;
  icon: ReactNode;
  testId?: string;
};

function TabLink({ route, activeRoute, label, icon, testId }: TabLinkProps) {
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

export function AppHeader({ route, pageTitle, statusPulse, statusLabel }: AppHeaderProps) {
  return (
    <header className="ko-app-header">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Database className="h-8 w-8 ko-icon" />
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
            <Activity
              className={`h-4 w-4 ko-icon-muted ${statusPulse ? "animate-pulse" : "opacity-80"}`}
            />
            <span className="ko-text-xs ko-text-primary font-semibold uppercase tracking-wider">
              {statusLabel}
            </span>
          </div>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          {NAV_ITEMS.map((item) => (
            <TabLink
              key={item.route}
              route={item.route}
              activeRoute={route}
              label={item.label}
              icon={item.icon}
              testId={item.testId}
            />
          ))}
        </div>
        <div className="ko-meta" data-testid={selectors.header.pageTitle}>
          {pageTitle}
        </div>
      </div>
    </header>
  );
}
