// Application header with status badge and settings button

import { AlertCircle, Bot, CheckCircle2, Settings2, Wifi, WifiOff } from "lucide-react";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import type { HealthResponse } from "../../types";
import { HealthStatus } from "../../types";

type WsStatus = "connected" | "connecting" | "disconnected" | "error";

interface AppHeaderProps {
  health: HealthResponse | null | undefined;
  wsStatus: WsStatus;
  onStatusClick: () => void;
  onSettingsClick: () => void;
}

export function AppHeader({
  health,
  wsStatus,
  onStatusClick,
  onSettingsClick,
}: AppHeaderProps) {
  const isHealthy = health?.status === HealthStatus.HEALTHY;
  const healthLabel = health ? (isHealthy ? "Healthy" : "Degraded") : "Unknown";
  const wsLabel =
    wsStatus === "connected" ? "Live" : wsStatus === "connecting" ? "Connecting" : wsStatus === "error" ? "Error" : "Offline";

  const statusVariant =
    !isHealthy || wsStatus === "disconnected" || wsStatus === "error"
      ? "destructive"
      : wsStatus === "connecting"
      ? "secondary"
      : "success";

  const statusText = `${healthLabel} • ${wsLabel}`;

  return (
    <header className="flex items-center justify-between gap-4 border-b border-border/50 px-6 py-3 sm:px-10">
      <div className="flex items-center gap-3">
        <Bot className="h-6 w-6 text-primary" />
        <span className="text-lg font-semibold">Agent Manager</span>
        <Badge
          variant={statusVariant}
          className="gap-1 cursor-pointer"
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
          {isHealthy ? <CheckCircle2 className="h-3 w-3" /> : <AlertCircle className="h-3 w-3" />}
          {wsStatus === "connected" ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
          {statusText}
        </Badge>
      </div>
      <Button
        variant="ghost"
        size="sm"
        onClick={onSettingsClick}
        className="gap-2"
      >
        <Settings2 className="h-4 w-4" />
        Settings
      </Button>
    </header>
  );
}
