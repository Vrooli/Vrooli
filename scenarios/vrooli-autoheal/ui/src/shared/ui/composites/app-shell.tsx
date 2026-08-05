import type { ReactNode } from "react";
import { BookOpen, Clock, LayoutDashboard, Loader2, Play, Settings, Shield, ShieldAlert, TrendingUp } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import { cn } from "../../../lib/utils";
import type { TabType } from "../../../hooks/useActiveTab";
import { Badge, Button } from "../primitives";
import { TabTrigger } from "./tab-trigger";

interface AppShellProps {
  activeTab: TabType;
  children: ReactNode;
  isTickRunning: boolean;
  onOpenSettings: () => void;
  onRunTick: () => void;
  onTabChange: (tab: TabType) => void;
}

const navItems: Array<{
  tab: TabType;
  label: string;
  icon: typeof LayoutDashboard;
  selector: string;
}> = [
  { tab: "dashboard", label: "Dashboard", icon: LayoutDashboard, selector: selectors.tabs.dashboard },
  { tab: "trends", label: "Trends", icon: TrendingUp, selector: selectors.tabs.trends },
  { tab: "timeline", label: "Timeline", icon: Clock, selector: selectors.tabs.timeline },
  { tab: "incidents", label: "Incidents", icon: ShieldAlert, selector: selectors.tabs.incidents },
  { tab: "docs", label: "Docs", icon: BookOpen, selector: selectors.tabs.docs },
];

export function AppShell({
  activeTab,
  children,
  isTickRunning,
  onOpenSettings,
  onRunTick,
  onTabChange,
}: AppShellProps) {
  return (
    <div
      className="min-h-full overflow-x-hidden bg-surface-base text-text-primary"
      data-testid={selectors.dashboard}
    >
      <header
        className="sticky top-0 z-30 border-b border-border-default/70 bg-surface-elevated/95 shadow-panel backdrop-blur"
        data-testid={selectors.shell.header}
      >
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-2 px-3 py-2.5 sm:gap-3 sm:px-4 sm:py-3">
          <div className="flex min-w-0 items-center gap-2 sm:gap-3">
            <Shield className="h-5 w-5 shrink-0 text-accent-primary sm:h-6 sm:w-6" />
            <div className="min-w-0">
              <h1 className="truncate text-base font-semibold leading-tight sm:text-xl">Vrooli Autoheal</h1>
              <p className="hidden text-xs text-text-muted sm:block">Self-healing infrastructure supervisor</p>
            </div>
          </div>

          <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
            <Button
              variant="outline"
              size="icon"
              onClick={onOpenSettings}
              data-testid={selectors.settingsButton}
              title="Open settings"
              aria-label="Open settings"
            >
              <Settings className="h-4 w-4" />
            </Button>

            <Button
              size="icon"
              onClick={onRunTick}
              disabled={isTickRunning}
              data-testid={selectors.runTickButton}
              title={isTickRunning ? "Health check cycle is running" : "Run health check cycle"}
              aria-label={isTickRunning ? "Health check cycle is running" : "Run health check cycle"}
              className="sm:hidden"
            >
              {isTickRunning ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
            </Button>

            <Button
              size="sm"
              onClick={onRunTick}
              disabled={isTickRunning}
              data-testid={selectors.runTickButtonDesktop}
              className="hidden px-3 sm:inline-flex"
              aria-label="Run health check cycle"
            >
              {isTickRunning ? <Loader2 className="h-4 w-4 animate-spin sm:mr-2" /> : <Play className="h-4 w-4 sm:mr-2" />}
              Run Tick
            </Button>
            {isTickRunning ? (
              <Badge tone="info" className="hidden sm:inline-flex">
                Tick Running
              </Badge>
            ) : null}
          </div>
        </div>

        <div className="mx-auto max-w-6xl px-3 sm:px-4">
          <nav
            aria-label="Autoheal sections"
            className={cn(
              "flex min-w-0 gap-1 overflow-x-auto overscroll-x-contain",
              "[-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
            )}
            data-testid={selectors.shell.nav}
          >
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <TabTrigger
                  key={item.tab}
                  onClick={() => onTabChange(item.tab)}
                  active={activeTab === item.tab}
                  data-testid={item.selector}
                  className="min-h-10 shrink-0 px-3 sm:px-4"
                  aria-current={activeTab === item.tab ? "page" : undefined}
                >
                  <Icon size={16} />
                  {item.label}
                </TabTrigger>
              );
            })}
          </nav>
        </div>
      </header>

      <main
        className="mx-auto min-w-0 max-w-6xl px-3 py-3 sm:px-4 sm:py-5"
        data-testid={selectors.shell.content}
      >
        {children}
      </main>
    </div>
  );
}
