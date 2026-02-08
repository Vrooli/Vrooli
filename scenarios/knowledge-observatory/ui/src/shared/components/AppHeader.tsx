// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { type ReactNode, useEffect, useState } from "react";
import { Activity, Database, FolderTree, GitGraph, Menu, Search, X } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { routeToHash, type Route } from "../controllers/routeController";
import { useIsMobile } from "../hooks/useViewportSize";

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
    route: "explorer",
    label: "Explorer",
    icon: <FolderTree className="h-4 w-4" />,
    testId: selectors.nav.explorer,
  },
  // Viewer is now integrated into Explorer - no separate tab needed
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
  const isMobile = useIsMobile();
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  useEffect(() => {
    setIsMobileMenuOpen(false);
  }, [route, isMobile]);

  useEffect(() => {
    if (!isMobileMenuOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setIsMobileMenuOpen(false);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [isMobileMenuOpen]);

  const closeMobileMenu = () => {
    setIsMobileMenuOpen(false);
  };

  return (
    <header className="ko-app-header">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <Database className="h-6 w-6 ko-icon shrink-0" />
          <div className="min-w-0">
            <h1
              className="truncate text-lg font-bold tracking-tight sm:text-xl"
              data-testid={selectors.header.title}
            >
              Knowledge Observatory
            </h1>
            <p className="ko-text-sm ko-subtle hidden md:block">
              Knowledge Health, Search, and Documentation Intelligence
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <div className="ko-status-badge" data-testid={selectors.header.statusBadge}>
            <Activity className={`h-3.5 w-3.5 ko-icon-muted ${statusPulse ? "animate-pulse" : "opacity-80"}`} />
            <span className="ko-text-xs ko-text-primary font-semibold uppercase tracking-wider">
              {isMobile ? statusLabel : `System ${statusLabel}`}
            </span>
          </div>

          <button
            type="button"
            className="ko-mobile-menu-trigger ko-focus-ring md:hidden"
            data-testid={selectors.header.mobileMenuButton}
            aria-label={isMobileMenuOpen ? "Close navigation menu" : "Open navigation menu"}
            aria-controls="ko-mobile-nav"
            aria-expanded={isMobileMenuOpen}
            onClick={() => setIsMobileMenuOpen((prev) => !prev)}
          >
            {isMobileMenuOpen ? <X className="h-4 w-4" /> : <Menu className="h-4 w-4" />}
          </button>
        </div>
      </div>

      <div className="mt-3 hidden items-center justify-between gap-3 md:flex">
        <div className="flex flex-wrap items-center gap-2.5">
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

      <div
        id="ko-mobile-nav"
        data-testid={selectors.header.mobileMenuPanel}
        className={[
          "ko-mobile-menu-panel md:hidden",
          isMobileMenuOpen ? "ko-mobile-menu-panel-open" : "ko-mobile-menu-panel-closed",
        ].join(" ")}
      >
        <nav aria-label="Mobile navigation" className="flex flex-col gap-2">
          {NAV_ITEMS.map((item) => (
            <a
              key={item.route}
              href={routeToHash(item.route)}
              data-testid={item.testId}
              className={[
                "ko-mobile-nav-link",
                item.route === route ? "ko-mobile-nav-link-active" : "ko-mobile-nav-link-inactive",
              ].join(" ")}
              aria-current={item.route === route ? "page" : undefined}
              onClick={closeMobileMenu}
            >
              {item.icon}
              {item.label}
            </a>
          ))}
        </nav>
        <div className="mt-2 ko-meta" data-testid={selectors.header.pageTitle}>
          {pageTitle}
        </div>
      </div>
    </header>
  );
}
