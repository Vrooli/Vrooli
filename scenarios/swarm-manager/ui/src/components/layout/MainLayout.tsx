import { useEffect, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate, Outlet } from "react-router-dom";
import { Lightbulb, Package, Zap, ScrollText, Settings } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { applyTheme, cn, defaultQueryOptions, watchSystemTheme } from "../../lib";
import { settingsService } from "../../services";
import { useBacklogStore, useScenariosStore } from "../../stores";

interface TabConfig {
  id: string;
  label: string;
  icon: React.ElementType;
  path: string;
  testId: string;
  mobileTestId: string;
  /** Keyboard shortcut (1-5) for quick navigation */
  shortcut: string;
}

const tabs: TabConfig[] = [
  { id: "backlog", label: "Backlog", icon: Lightbulb, path: "/backlog", testId: selectors.tabs.backlog, mobileTestId: selectors.mobileTabs.backlog, shortcut: "1" },
  { id: "scenarios", label: "Scenarios", icon: Package, path: "/scenarios", testId: selectors.tabs.scenarios, mobileTestId: selectors.mobileTabs.scenarios, shortcut: "2" },
  { id: "execution", label: "Execution", icon: Zap, path: "/execution", testId: selectors.tabs.execution, mobileTestId: selectors.mobileTabs.execution, shortcut: "3" },
  { id: "prompts", label: "Prompts", icon: ScrollText, path: "/prompts", testId: selectors.tabs.prompts, mobileTestId: selectors.mobileTabs.prompts, shortcut: "4" },
  { id: "settings", label: "Settings", icon: Settings, path: "/settings", testId: selectors.tabs.settings, mobileTestId: selectors.mobileTabs.settings, shortcut: "5" },
];

/**
 * MainLayout with keyboard navigation support.
 *
 * Experience Architecture (Phase 29):
 * - Keyboard shortcuts (1-5) for power users to quickly switch tabs
 * - Reduces mechanical friction for frequent navigators
 * - Only active when no input element is focused
 */
export function MainLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const fetchBacklog = useBacklogStore((state) => state.fetchBacklog);
  const fetchScenarios = useScenariosStore((state) => state.fetchScenarios);

  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  const activeTab = tabs.find(tab => location.pathname.startsWith(tab.path))?.id || "backlog";
  const isBacklogDetailsRoute = /^\/backlog\/[^/]+\/[^/]+/.test(location.pathname);
  const isScenarioDetailsRoute = /^\/scenarios\/[^/]+$/.test(location.pathname);
  const isImmersiveMobileRoute = isBacklogDetailsRoute || isScenarioDetailsRoute;

  // Keyboard navigation via centralized shortcut hook (see useKeyboardShortcuts)
  const handleTabNav = useCallback((key: string) => {
    const tab = tabs.find(t => t.shortcut === key);
    if (tab) {
      navigate(tab.path);
      return true;
    }
    return false;
  }, [navigate]);

  useKeyboardShortcuts({ onTabNav: handleTabNav });

  useEffect(() => {
    void fetchBacklog();
    void fetchScenarios();
  }, [fetchBacklog, fetchScenarios]);

  useEffect(() => {
    const theme = settings?.theme ?? "dark";
    applyTheme(theme);

    if (theme === "system") {
      return watchSystemTheme(() => {
        applyTheme("system");
      });
    }

    return undefined;
  }, [settings?.theme]);

  return (
    <div
      className={cn(
        "min-h-screen bg-slate-950 text-slate-50"
      )}
      data-testid={selectors.layout.main}
    >
      {/* Desktop Header with Tabs */}
      <header
        className={cn(
          "hidden md:flex h-16 items-center justify-between border-b border-slate-200/20 px-6"
        )}
        data-testid={selectors.layout.header}
      >
        <h1 className="text-xl font-semibold">Swarm Manager</h1>
        <nav className="flex items-center gap-1" data-testid={selectors.layout.desktopTabs}>
          {tabs.map((tab) => {
            const Icon = tab.icon;
            const isActive = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                onClick={() => navigate(tab.path)}
                data-testid={tab.testId}
                title={`${tab.label} (${tab.shortcut})`}
                className={cn(
                  "flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors",
                  isActive
                    ? "bg-slate-800/70 text-cyan-400"
                    : "text-slate-300 hover:text-slate-50 hover:bg-slate-800/50"
                )}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
                <span className="hidden lg:inline text-xs ml-1 text-slate-400">
                  ({tab.shortcut})
                </span>
              </button>
            );
          })}
        </nav>
      </header>

      {/* Mobile Header */}
      {!isImmersiveMobileRoute && (
        <header className="md:hidden flex h-14 items-center justify-center border-b border-slate-200/20 px-4">
          <h1 className="text-lg font-semibold">Swarm Manager</h1>
        </header>
      )}

      {/* Main Content */}
      <main className={cn(
        isImmersiveMobileRoute ? "p-0 md:p-6" : "p-6",
        isImmersiveMobileRoute ? "pb-0 md:pb-6" : "pb-20 md:pb-6"
      )}>
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      {!isImmersiveMobileRoute && (
        <nav
          className={cn(
            "md:hidden fixed bottom-0 left-0 right-0 h-16 border-t border-slate-200/20 bg-slate-950/95 backdrop-blur"
          )}
          data-testid={selectors.layout.mobileNav}
        >
          <div className="flex h-full items-center justify-around">
            {tabs.map((tab) => {
              const Icon = tab.icon;
              const isActive = activeTab === tab.id;
              return (
                <button
                  key={tab.id}
                  onClick={() => navigate(tab.path)}
                  data-testid={tab.mobileTestId}
                  className={cn(
                    "flex flex-col items-center gap-1 px-3 py-2 text-xs font-medium transition-colors",
                    isActive ? "text-cyan-400" : "text-slate-300"
                  )}
                >
                  <Icon className="h-5 w-5" />
                  {tab.label}
                </button>
              );
            })}
          </div>
        </nav>
      )}
    </div>
  );
}
