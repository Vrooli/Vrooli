// Status dialog showing health and WebSocket connection details

import { AlertCircle, CheckCircle2, Wifi, WifiOff } from "lucide-react";
import { Badge } from "../ui/badge";
import { formatDuration } from "../../lib/format";
import { Card, CardContent } from "../ui/card";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import type { HealthResponse, JsonValue } from "../../types";
import { healthView } from "../../lib/healthState";
import { jsonValueToPlain } from "../../lib/utils";
import { formatStandardDateTime } from "../../lib/dateTime";

type WsStatus = "connected" | "connecting" | "disconnected" | "error";

interface StatusDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  health: HealthResponse | null | undefined;
  healthError: string | null | undefined;
  /** Why a reachable API reports itself degraded. */
  healthReason?: string;
  wsStatus: WsStatus;
}

export function StatusDialog({
  open,
  onOpenChange,
  health,
  healthError,
  healthReason,
  wsStatus,
}: StatusDialogProps) {
  const healthState = healthView(health, healthError, healthReason);
  const isHealthy = healthState.kind === "healthy";
  const wsLabel =
    wsStatus === "connected" ? "Live" : wsStatus === "connecting" ? "Connecting" : wsStatus === "error" ? "Error" : "Offline";

  const dependencyEntries = Object.entries(health?.dependencies ?? {});
  const metricEntries = Object.entries(health?.metrics ?? {});

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Status Details</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Badge variant={isHealthy ? "success" : healthState.kind === "degraded" ? "warning" : healthState.kind === "checking" ? "secondary" : "destructive"} className="gap-1">
              {isHealthy ? <CheckCircle2 className="h-3 w-3" /> : <AlertCircle className="h-3 w-3" />}
              {healthState.label}
            </Badge>
            <Badge
              variant={wsStatus === "connected" ? "success" : wsStatus === "connecting" ? "secondary" : "outline"}
              className="gap-1"
            >
              {wsStatus === "connected" ? <Wifi className="h-3 w-3" /> : <WifiOff className="h-3 w-3" />}
              {wsLabel}
            </Badge>
          </div>

          <div className="grid gap-3 text-sm">
            <div className="flex items-center justify-between text-muted-foreground">
              <span>Service</span>
              <span className="text-foreground">{health?.service || "Unknown"}</span>
            </div>
            <div className="flex items-center justify-between text-muted-foreground">
              <span>Readiness</span>
              <span className="text-foreground">{health?.readiness ? "Ready" : "Not ready"}</span>
            </div>
            <div className="flex items-center justify-between text-muted-foreground">
              <span>Version</span>
              <span className="text-foreground">{health?.version || "Unknown"}</span>
            </div>
            <div className="flex items-center justify-between text-muted-foreground">
              <span>Timestamp</span>
              <span className="text-foreground">{formatStandardDateTime(health?.timestamp)}</span>
            </div>
          </div>

          {dependencyEntries.length > 0 && (
            <DependenciesSection entries={dependencyEntries} />
          )}

          {metricEntries.length > 0 && (
            <MetricsSection entries={metricEntries} />
          )}

          {healthState.reason && (
            <Card className={`text-xs ${healthState.kind === "degraded" ? "border-warning/40 bg-warning/10" : "border-destructive/40 bg-destructive/10"}`}>
              <CardContent className="flex items-start gap-2 py-3">
                <AlertCircle className={`h-4 w-4 ${healthState.kind === "degraded" ? "text-warning" : "text-destructive"}`} aria-hidden="true" />
                <div>
                  <p className="font-semibold">{healthState.kind === "degraded" ? "Agent Manager is degraded" : "Agent Manager is unreachable"}</p>
                  <p className="text-muted-foreground">{healthState.reason}</p>
                </div>
              </CardContent>
            </Card>
          )}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function DependenciesSection({ entries }: { entries: [string, JsonValue][] }) {
  return (
    <div className="space-y-2">
      <p className="text-sm font-semibold">Dependencies</p>
      <div className="space-y-2 text-xs text-muted-foreground">
        {entries.map(([name, value]) => {
          const plain = jsonValueToPlain(value) as Record<string, unknown> | undefined;
          const status = plain?.status as string | undefined;
          const latency = plain?.latency_ms as number | undefined;
          const error = plain?.error as string | undefined;
          const storage = plain?.storage as string | undefined;
          return (
            <div key={name} className="flex flex-col gap-1 rounded-md border border-border p-2">
              <div className="flex items-center justify-between text-foreground">
                <span className="font-semibold">{name}</span>
                <span>{status || "unknown"}</span>
              </div>
              <div className="flex items-center justify-between">
                <span>Latency</span>
                <span>{formatDuration(latency)}</span>
              </div>
              {storage && (
                <div className="flex items-center justify-between">
                  <span>Storage</span>
                  <span>{storage}</span>
                </div>
              )}
              {error && (
                <div className="text-destructive">Error: {error}</div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function MetricsSection({ entries }: { entries: [string, JsonValue][] }) {
  return (
    <div className="space-y-2">
      <p className="text-sm font-semibold">Metrics</p>
      <div className="grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
        {entries.map(([name, value]) => (
          <div key={name} className="flex items-center justify-between rounded-md border border-border p-2">
            <span>{name}</span>
            <span className="text-foreground">{String(jsonValueToPlain(value) ?? "n/a")}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
