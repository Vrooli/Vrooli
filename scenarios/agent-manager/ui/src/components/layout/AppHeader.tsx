// Application header with nav links, status badge and settings button

import {
  Activity,
  AlertCircle,
  BarChart3,
  Bot,
  CheckCircle2,
  ClipboardList,
  Cog,
  Play,
  Search,
  Settings2,
  Wifi,
  WifiOff,
} from "lucide-react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import type { HealthResponse } from "../../types";
import { HealthStatus } from "../../types";
import type { NavSection } from "./MobileNav";

type WsStatus = "connected" | "connecting" | "disconnected" | "error";

interface AppHeaderProps {
  health: HealthResponse | null | undefined;
  wsStatus: WsStatus;
  activeSection: NavSection;
  isMobile: boolean;
  onSectionChange: (section: NavSection) => void;
  onStatusClick: () => void;
  onSettingsClick: () => void;
}

const navItems: Array<{
  id: NavSection;
  label: string;
  icon: typeof Activity;
}> = [
  { id: "dashboard", label: "Dashboard", icon: Activity },
  { id: "profiles", label: "Profiles", icon: Settings2 },
  { id: "tasks", label: "Tasks", icon: ClipboardList },
  { id: "runs", label: "Runs", icon: Play },
  { id: "investigations", label: "Investigations", icon: Search },
  { id: "stats", label: "Stats", icon: BarChart3 },
];

export function AppHeader({
  health,
  wsStatus,
  activeSection,
  isMobile,
  onSectionChange,
  onStatusClick,
  onSettingsClick,
}: AppHeaderProps) {
  const isHealthy = health?.status === HealthStatus.HEALTHY;
  const healthLabel = health ? (isHealthy ? "Healthy" : "Degraded") : "Unknown";
  const wsLabel =
    wsStatus === "connected"
      ? "Live"
      : wsStatus === "connecting"
        ? "Connecting"
        : wsStatus === "error"
          ? "Error"
          : "Offline";

  const statusVariant =
    !isHealthy || wsStatus === "disconnected" || wsStatus === "error"
      ? "destructive"
      : wsStatus === "connecting"
        ? "secondary"
        : "success";

  const statusText = `${healthLabel} • ${wsLabel}`;

  return (
    <header className="sticky top-0 z-30 flex items-center justify-between gap-4 border-b border-border bg-background/95 backdrop-blur-sm px-4 py-2 sm:px-6 lg:px-10">
      {/* Left: Logo and status */}
      <div className="flex items-center gap-3 shrink-0">
        <Bot className="h-6 w-6 text-primary" />
        <span className="text-lg font-semibold hidden sm:inline">Agent Manager</span>
        <Badge
          variant={statusVariant}
          className="gap-1 cursor-pointer text-xs"
          onClick={onStatusClick}
          role="button"
          tabIndex={0}
          aria-label="Open status details"
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              onStatusClick();
            }
          }}
        >
          {isHealthy ? (
            <CheckCircle2 className="h-3 w-3" />
          ) : (
            <AlertCircle className="h-3 w-3" />
          )}
          {wsStatus === "connected" ? (
            <Wifi className="h-3 w-3" />
          ) : (
            <WifiOff className="h-3 w-3" />
          )}
          <span className="hidden md:inline">{statusText}</span>
        </Badge>
      </div>

      {/* Right: Navigation + Settings */}
      <div className="flex items-center gap-2">
        {/* Navigation - hidden on mobile */}
        {!isMobile && (
          <nav className="flex items-center gap-1">
            {navItems.map((item) => {
              const isActive = activeSection === item.id;
              const Icon = item.icon;

              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => onSectionChange(item.id)}
                  className={`flex items-center gap-2 px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                    isActive
                      ? "bg-primary/10 text-primary"
                      : "text-muted-foreground hover:text-foreground hover:bg-muted/50"
                  }`}
                  aria-current={isActive ? "page" : undefined}
                >
                  <Icon className="h-4 w-4" />
                  {/* Show label on larger screens, hide when narrower to avoid overflow */}
                  <span className="hidden min-[1120px]:inline">{item.label}</span>
                </button>
              );
            })}
          </nav>
        )}

        {/* Settings icon button */}
        <Button
          variant="ghost"
          size="icon"
          onClick={onSettingsClick}
          aria-label="Settings"
        >
          <Cog className="h-5 w-5" />
        </Button>
      </div>
    </header>
  );
}
