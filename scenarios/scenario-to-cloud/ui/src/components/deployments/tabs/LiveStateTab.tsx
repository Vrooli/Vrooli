import { useState } from "react";
import {
  RefreshCw,
  Loader2,
  AlertCircle,
  Activity,
  Network,
  HardDrive,
  Shield,
  Settings,
} from "lucide-react";
import { useLiveState, getTimeSince } from "../../../hooks/useLiveState";
import { ProcessCards } from "./ProcessCards";
import { PortTable } from "./PortTable";
import { SystemResources } from "./SystemResources";
import { CaddyStatus } from "./CaddyStatus";
import { VPSManagement } from "./VPSManagement";
import { cn } from "../../../lib/utils";

interface LiveStateTabProps {
  deploymentId: string;
  deploymentName?: string;
}

export function LiveStateTab({ deploymentId, deploymentName }: LiveStateTabProps) {
  const { data: liveState, isLoading, error, refetch, isFetching } = useLiveState(deploymentId);
  const [activeSection, setActiveSection] = useState<"processes" | "ports" | "system" | "caddy" | "management">("processes");

  const handleRefresh = () => {
    refetch();
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 rounded-lg border border-red-500/30 bg-red-500/10 text-red-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>Failed to load live state: {error.message}</span>
        </div>
      </div>
    );
  }

  if (!liveState?.ok) {
    return (
      <div className="p-4 rounded-lg border border-amber-500/30 bg-amber-500/10 text-amber-400">
        <div className="flex items-center gap-2">
          <AlertCircle className="h-5 w-5" />
          <span>{liveState?.error || "Unable to fetch live state from VPS"}</span>
        </div>
      </div>
    );
  }

  const sections = [
    { id: "processes" as const, label: "Processes", icon: Activity },
    { id: "ports" as const, label: "Network", icon: Network },
    { id: "system" as const, label: "System", icon: HardDrive },
    { id: "caddy" as const, label: "Edge/TLS", icon: Shield },
    { id: "management" as const, label: "VPS Management", icon: Settings },
  ];

  return (
    <div className="space-y-4">
      {/* Header with refresh and sync info */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 text-sm text-slate-400">
          <span>Last synced: {getTimeSince(liveState.timestamp)}</span>
          <span className="text-slate-600">|</span>
          <span>Took {liveState.sync_duration_ms}ms</span>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isFetching}
          className={cn(
            "flex items-center gap-2 px-3 py-1.5 rounded-lg border border-white/10",
            "hover:bg-white/5 transition-colors text-sm font-medium",
            isFetching && "opacity-50 cursor-not-allowed"
          )}
        >
          {isFetching ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Refresh
        </button>
      </div>

      {/* Section tabs */}
      <div className="flex gap-1 border-b border-white/10 pb-px">
        {sections.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveSection(id)}
            className={cn(
              "flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors rounded-t-lg",
              activeSection === id
                ? "bg-slate-800 text-white border-b-2 border-blue-500"
                : "text-slate-400 hover:text-white hover:bg-slate-800/50"
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Section content */}
      <div className="mt-4">
        {activeSection === "processes" && liveState.processes && (
          <ProcessCards
            deploymentId={deploymentId}
            processes={liveState.processes}
            expected={liveState.expected}
          />
        )}
        {activeSection === "ports" && liveState.ports && (
          <PortTable ports={liveState.ports} />
        )}
        {activeSection === "system" && liveState.system && (
          <SystemResources system={liveState.system} />
        )}
        {activeSection === "caddy" && liveState.caddy && (
          <CaddyStatus caddy={liveState.caddy} deploymentId={deploymentId} />
        )}
        {activeSection === "management" && (
          <VPSManagement
            deploymentId={deploymentId}
            deploymentName={deploymentName || deploymentId}
          />
        )}
      </div>
    </div>
  );
}
