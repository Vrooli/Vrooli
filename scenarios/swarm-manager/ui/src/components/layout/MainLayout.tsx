import { useEffect, useCallback, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate, Outlet } from "react-router-dom";
import { Activity, Lightbulb, Package, ScrollText, Settings, Square, X, Zap } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { useKeyboardShortcuts } from "../../hooks/useKeyboardShortcuts";
import { applyTheme, cn, defaultQueryOptions, formatRelativeTime, watchSystemTheme } from "../../lib";
import { settingsService } from "../../services";
import { useAgentActivitiesStore, useBacklogStore, useScenariosStore } from "../../stores";

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
  const agentActivities = useAgentActivitiesStore((state) => state.activities);
  const stopRun = useAgentActivitiesStore((state) => state.stopRun);
  const refreshActivities = useAgentActivitiesStore((state) => state.refreshActivities);
  const [showAgentsDropdown, setShowAgentsDropdown] = useState(false);

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
    void refreshActivities(true);
    const timer = window.setInterval(() => {
      void refreshActivities(true);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [refreshActivities]);

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

  const sortedActiveActivities = useMemo(() => {
    return [...agentActivities].sort(
      (a, b) => new Date(b.requestedAt).getTime() - new Date(a.requestedAt).getTime()
    );
  }, [agentActivities]);

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
        <div className="flex items-center gap-3">
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
          <div className="relative">
            <button
              type="button"
              className="flex items-center gap-2 rounded-lg border border-slate-700/80 bg-slate-900/45 px-3 py-2 text-sm text-slate-100 hover:bg-slate-800/70"
              onClick={() => setShowAgentsDropdown((prev) => !prev)}
              data-testid={selectors.layout.agentsToggle}
            >
              <Activity className="h-4 w-4 text-cyan-300" />
              Agents running
              <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-200">
                {sortedActiveActivities.length}
              </span>
            </button>
            {showAgentsDropdown && (
              <>
                <button
                  type="button"
                  className="fixed inset-0 z-40 cursor-default bg-transparent"
                  aria-label="Close agents dropdown"
                  onClick={() => setShowAgentsDropdown(false)}
                />
                <div
                  className="absolute right-0 top-12 z-50 w-[360px] rounded-lg border border-slate-700/80 bg-slate-950 shadow-xl"
                  data-testid={selectors.layout.agentsDropdown}
                >
                  <div className="flex items-center justify-between border-b border-slate-800 px-3 py-2">
                    <div>
                      <p className="text-sm font-semibold text-slate-100">Agents running</p>
                      <p className="text-xs text-slate-400">{sortedActiveActivities.length} active activity item(s)</p>
                    </div>
                    <button
                      type="button"
                      className="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
                      onClick={() => setShowAgentsDropdown(false)}
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                  <div className="max-h-80 overflow-y-auto p-2">
                    {sortedActiveActivities.length === 0 ? (
                      <p className="rounded-lg border border-slate-800 bg-slate-900/40 px-3 py-6 text-center text-sm text-slate-400">
                        No agents are currently running.
                      </p>
                    ) : (
                      <div className="space-y-2">
                        {sortedActiveActivities.map((activity) => (
                          <div key={activity.activityId} className="rounded-lg border border-slate-800 bg-slate-900/45 p-3">
                            <div className="flex items-start justify-between gap-2">
                              <div className="min-w-0">
                                <p className="truncate text-sm font-medium text-slate-100">
                                  {activity.ownerTitle ?? `${activity.ownerType}/${activity.ownerName}`}
                                </p>
                                {activity.runId && (
                                  <p className="font-mono text-xs text-cyan-300">{activity.runId}</p>
                                )}
                              </div>
                              <span className="rounded-full bg-cyan-500/15 px-2 py-0.5 text-[11px] text-cyan-200">
                                {activity.status.replace("_", " ")}
                              </span>
                            </div>
                            <p className="mt-1 text-xs text-slate-400">
                              {activity.purpose.replace("_", " ")} • Requested {formatRelativeTime(activity.requestedAt)}
                            </p>
                            <div className="mt-2 flex items-center gap-2">
                              {activity.ownerType === "backlog" && activity.ownerKind && activity.ownerName && (
                                <button
                                  type="button"
                                  className="rounded border border-slate-700/80 bg-slate-900/60 px-2 py-1 text-xs text-slate-200 hover:bg-slate-800/70"
                                  onClick={() => {
                                    navigate(`/backlog/${activity.ownerKind}/${activity.ownerName}`);
                                    setShowAgentsDropdown(false);
                                  }}
                                >
                                  Open
                                </button>
                              )}
                              {activity.runId && (
                                <button
                                  type="button"
                                  className="rounded border border-red-500/40 bg-red-500/10 px-2 py-1 text-xs text-red-200 hover:bg-red-500/20"
                                  onClick={() => void stopRun(activity.runId ?? "")}
                                  disabled={activity.isStopping}
                                >
                                  <Square className="mr-1 inline h-3 w-3" />
                                  {activity.isStopping ? "Stopping..." : "Stop"}
                                </button>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      </header>

      {/* Mobile header removed — bottom nav already shows current page */}

      {/* Main Content */}
      <main className={cn(
        isImmersiveMobileRoute ? "p-0 md:p-6" : "pt-0 px-6 md:pt-6",
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
