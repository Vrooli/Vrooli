import { useEffect, useCallback, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate, Outlet } from "react-router-dom";
import { Lightbulb, Package, Zap, Settings } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { applyTheme, cn, defaultQueryOptions, watchSystemTheme, type ResolvedTheme } from "../../lib";
import { settingsService } from "../../services";
import { useBacklogStore, useRecommendationsStore, useScenariosStore } from "../../stores";

interface TabConfig {
  id: string;
  label: string;
  icon: React.ElementType;
  path: string;
  testId: string;
  mobileTestId: string;
  /** Keyboard shortcut (1-4) for quick navigation */
  shortcut: string;
}

const tabs: TabConfig[] = [
  { id: "backlog", label: "Backlog", icon: Lightbulb, path: "/backlog", testId: selectors.tabs.backlog, mobileTestId: selectors.mobileTabs.backlog, shortcut: "1" },
  { id: "scenarios", label: "Scenarios", icon: Package, path: "/scenarios", testId: selectors.tabs.scenarios, mobileTestId: selectors.mobileTabs.scenarios, shortcut: "2" },
  { id: "recommendations", label: "Recommendations", icon: Zap, path: "/recommendations", testId: selectors.tabs.recommendations, mobileTestId: selectors.mobileTabs.recommendations, shortcut: "3" },
  { id: "settings", label: "Settings", icon: Settings, path: "/settings", testId: selectors.tabs.settings, mobileTestId: selectors.mobileTabs.settings, shortcut: "4" },
];

/**
 * MainLayout with keyboard navigation support.
 *
 * Experience Architecture (Phase 29):
 * - Keyboard shortcuts (1-4) for power users to quickly switch tabs
 * - Reduces mechanical friction for frequent navigators
 * - Only active when no input element is focused
 */
export function MainLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>("dark");
  const fetchBacklog = useBacklogStore((state) => state.fetchBacklog);
  const fetchScenarios = useScenariosStore((state) => state.fetchScenarios);
  const fetchRecommendations = useRecommendationsStore((state) => state.fetchRecommendations);

  const { data: settings } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsService.get(),
    ...defaultQueryOptions,
  });

  const activeTab = tabs.find(tab => location.pathname.startsWith(tab.path))?.id || "backlog";

  // Keyboard navigation handler for power users
  // Shortcuts: 1=Backlog, 2=Scenarios, 3=Recommendations, 4=Settings
  const handleKeyboardNav = useCallback((event: KeyboardEvent) => {
    // Don't intercept when user is typing in an input, textarea, or contenteditable
    const target = event.target as HTMLElement;
    if (
      target.tagName === "INPUT" ||
      target.tagName === "TEXTAREA" ||
      target.isContentEditable
    ) {
      return;
    }

    // Find matching tab by shortcut key
    const tab = tabs.find(t => t.shortcut === event.key);
    if (tab) {
      navigate(tab.path);
    }
  }, [navigate]);

  useEffect(() => {
    window.addEventListener("keydown", handleKeyboardNav);
    return () => window.removeEventListener("keydown", handleKeyboardNav);
  }, [handleKeyboardNav]);

  useEffect(() => {
    void fetchBacklog();
    void fetchScenarios();
    void fetchRecommendations();
  }, [fetchBacklog, fetchScenarios, fetchRecommendations]);

  useEffect(() => {
    const theme = settings?.theme ?? "dark";
    const resolved = applyTheme(theme);
    setResolvedTheme(resolved);

    if (theme === "system") {
      return watchSystemTheme((nextResolved) => {
        applyTheme("system");
        setResolvedTheme(nextResolved);
      });
    }

    return undefined;
  }, [settings?.theme]);

  const isLight = resolvedTheme === "light";

  return (
    <div
      className={cn(
        "min-h-screen",
        isLight ? "bg-slate-50 text-slate-900" : "bg-slate-950 text-slate-50"
      )}
      data-testid={selectors.layout.main}
    >
      {/* Desktop Header with Tabs */}
      <header
        className={cn(
          "hidden md:flex h-16 items-center justify-between border-b px-6",
          isLight ? "border-slate-200" : "border-white/10"
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
                    ? isLight
                      ? "bg-slate-200 text-cyan-600"
                      : "bg-slate-800 text-cyan-400"
                    : isLight
                      ? "text-slate-500 hover:text-slate-900 hover:bg-slate-200/60"
                      : "text-slate-300 hover:text-slate-100 hover:bg-slate-800/50"
                )}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
                <span className={cn("hidden lg:inline text-xs ml-1", isLight ? "text-slate-500" : "text-slate-400")}>
                  ({tab.shortcut})
                </span>
              </button>
            );
          })}
        </nav>
      </header>

      {/* Mobile Header */}
      <header className={cn("md:hidden flex h-14 items-center justify-center border-b px-4", isLight ? "border-slate-200" : "border-white/10")}>
        <h1 className="text-lg font-semibold">Swarm Manager</h1>
      </header>

      {/* Main Content */}
      <main className="pb-20 md:pb-6 p-6">
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      <nav
        className={cn(
          "md:hidden fixed bottom-0 left-0 right-0 h-16 border-t backdrop-blur",
          isLight ? "border-slate-200 bg-white/95" : "border-white/10 bg-slate-900/95"
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
                  isActive ? (isLight ? "text-cyan-600" : "text-cyan-400") : isLight ? "text-slate-500" : "text-slate-300"
                )}
              >
                <Icon className="h-5 w-5" />
                {tab.label}
              </button>
            );
          })}
        </div>
      </nav>
    </div>
  );
}
