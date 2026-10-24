// Platform capabilities display component
// [REQ:UI-HEALTH-001] [REQ:WATCH-DETECT-001]
import { Server, CheckCircle, Circle } from "lucide-react";
import type { PlatformCapabilities } from "../../../lib/api";
import { useProtectionStatus } from "./SystemProtection";
import { Panel, PanelHeader, PanelTitle } from "../../../shared/ui/composites";

interface PlatformInfoProps {
  platform: PlatformCapabilities;
}

// Capability with protection status indicator
interface CapabilityInfo {
  name: string;
  available: boolean;
  monitored?: boolean;
  watchdogType?: string;
}

export function PlatformInfo({ platform }: PlatformInfoProps) {
  const { status: watchdogStatus } = useProtectionStatus();

  // Build capability list with monitoring status
  const capabilities: CapabilityInfo[] = [
    {
      name: "Docker",
      available: platform.hasDocker,
      monitored: platform.hasDocker, // Docker check is always registered when available
    },
    {
      name: "Systemd",
      available: platform.supportsSystemd,
      watchdogType: watchdogStatus?.watchdogType === "systemd" ? "systemd" : undefined,
      monitored: watchdogStatus?.watchdogInstalled && watchdogStatus?.watchdogType === "systemd",
    },
    {
      name: "Launchd",
      available: platform.supportsLaunchd,
      watchdogType: watchdogStatus?.watchdogType === "launchd" ? "launchd" : undefined,
      monitored: watchdogStatus?.watchdogInstalled && watchdogStatus?.watchdogType === "launchd",
    },
    {
      name: "RDP",
      available: platform.supportsRdp,
      monitored: platform.supportsRdp, // RDP check runs when available
    },
    {
      name: "Cloudflared",
      available: platform.supportsCloudflared,
      monitored: platform.supportsCloudflared, // Cloudflared check runs when available
    },
    {
      name: "WSL",
      available: platform.isWsl,
    },
  ].filter((cap) => cap.available);

  return (
    <Panel>
      <PanelHeader className="justify-start">
        <Server size={18} className="text-text-muted" />
        <PanelTitle>Platform</PanelTitle>
      </PanelHeader>
      <div className="space-y-2">
        <p className="text-sm">
          <span className="text-text-muted">OS:</span>{" "}
          <span className="capitalize">{platform.platform}</span>
          {platform.isHeadlessServer && <span className="ml-2 text-text-muted/80">(headless)</span>}
        </p>

        {/* Capabilities with protection indicators */}
        {capabilities.length > 0 && (
          <div className="space-y-1 mt-3">
            {capabilities.map((cap) => (
              <div key={cap.name} className="flex items-center justify-between text-xs">
                <span className="text-text-muted">{cap.name}</span>
                <span className="flex items-center gap-1.5">
                  {cap.monitored !== undefined && (
                    <>
                      {cap.monitored ? (
                        <CheckCircle size={12} className="text-accent-success" />
                      ) : (
                        <Circle size={12} className="text-text-muted/50" />
                      )}
                      <span className={cap.monitored ? "text-accent-success" : "text-text-muted/80"}>
                        {cap.monitored ? "Monitored" : "Available"}
                      </span>
                    </>
                  )}
                  {cap.monitored === undefined && (
                    <span className="text-text-muted/80">Available</span>
                  )}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </Panel>
  );
}
