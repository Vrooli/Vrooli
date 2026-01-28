import { useEffect, useCallback } from "react";
import { useLocation, useNavigate, Outlet } from "react-router-dom";
import { Lightbulb, Package, Zap, Settings } from "lucide-react";
import { selectors } from "../../consts/selectors";

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
  { id: "ideas", label: "Ideas", icon: Lightbulb, path: "/ideas", testId: selectors.tabs.ideas, mobileTestId: selectors.mobileTabs.ideas, shortcut: "1" },
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

  const activeTab = tabs.find(tab => location.pathname.startsWith(tab.path))?.id || "ideas";

  // Keyboard navigation handler for power users
  // Shortcuts: 1=Ideas, 2=Scenarios, 3=Recommendations, 4=Settings
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

  return (
    <div className="min-h-screen bg-slate-950 text-slate-50" data-testid={selectors.layout.main}>
      {/* Desktop Header with Tabs */}
      <header
        className="hidden md:flex h-16 items-center justify-between border-b border-white/10 px-6"
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
                className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? "bg-slate-800 text-cyan-400"
                    : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/50"
                }`}
              >
                <Icon className="h-4 w-4" />
                {tab.label}
                <span className="hidden lg:inline text-xs text-slate-500 ml-1">({tab.shortcut})</span>
              </button>
            );
          })}
        </nav>
      </header>

      {/* Mobile Header */}
      <header className="md:hidden flex h-14 items-center justify-center border-b border-white/10 px-4">
        <h1 className="text-lg font-semibold">Swarm Manager</h1>
      </header>

      {/* Main Content */}
      <main className="pb-20 md:pb-6 p-6">
        <Outlet />
      </main>

      {/* Mobile Bottom Navigation */}
      <nav
        className="md:hidden fixed bottom-0 left-0 right-0 h-16 border-t border-white/10 bg-slate-900/95 backdrop-blur"
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
                className={`flex flex-col items-center gap-1 px-3 py-2 text-xs font-medium transition-colors ${
                  isActive
                    ? "text-cyan-400"
                    : "text-slate-400"
                }`}
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
