// Application header with nav links, status badge and settings button

import { AlertCircle, CheckCircle2, Cog, Menu, Play, Wifi, WifiOff } from "lucide-react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import type { HealthResponse } from "../../types";
import { HealthStatus } from "../../types";
import type { NavSection } from "./SideNav";

type WsStatus = "connected" | "connecting" | "disconnected" | "error";

interface AppHeaderProps {
  health: HealthResponse | null | undefined;
  wsStatus: WsStatus;
  activeSection: NavSection;
  isMobile: boolean;
  onSectionChange: (section: NavSection) => void;
  onStatusClick: () => void;
  onSettingsClick: () => void;
  onQuickRunClick: () => void;
  onNavigationClick: () => void;
}

export function AppHeader({
  health,
  wsStatus,
  activeSection,
  isMobile,
  onSectionChange,
  onStatusClick,
  onSettingsClick,
  onQuickRunClick,
  onNavigationClick,
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
        <Button variant="ghost" size="icon" onClick={onNavigationClick} aria-label="Open navigation menu">
          <Menu className="h-5 w-5" />
        </Button>
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

      {/* Right: quick run and settings; primary navigation belongs in SideNav. */}
      <div className="flex items-center gap-2">
        {/* Quick Run button */}
        <Button
          variant="default"
          size="sm"
          onClick={onQuickRunClick}
          className="gap-2"
          aria-label="Quick Run"
        >
          <Play className="h-4 w-4" />
          <span className="hidden sm:inline">Quick Run</span>
        </Button>

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
