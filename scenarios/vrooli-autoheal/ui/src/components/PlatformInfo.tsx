// Platform capabilities display component
// [REQ:UI-HEALTH-001] [REQ:WATCH-DETECT-001]
import { Server, CheckCircle, Circle } from "lucide-react";
import type { PlatformCapabilities } from "../lib/api";
import { useProtectionStatus } from "./SystemProtection";

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
    <div className="rounded-xl border border-white/10 bg-white/5 p-4">
      <div className="flex items-center gap-2 mb-3">
        <Server size={18} className="text-slate-400" />
        <h3 className="font-medium text-slate-200">Platform</h3>
      </div>
      <div className="space-y-2">
        <p className="text-sm">
          <span className="text-slate-400">OS:</span>{" "}
          <span className="text-slate-200 capitalize">{platform.platform}</span>
          {platform.isHeadlessServer && <span className="text-slate-500 ml-2">(headless)</span>}
        </p>

        {/* Capabilities with protection indicators */}
        {capabilities.length > 0 && (
          <div className="space-y-1 mt-3">
            {capabilities.map((cap) => (
              <div key={cap.name} className="flex items-center justify-between text-xs">
                <span className="text-slate-400">{cap.name}</span>
                <span className="flex items-center gap-1.5">
                  {cap.monitored !== undefined && (
                    <>
                      {cap.monitored ? (
                        <CheckCircle size={12} className="text-emerald-400" />
                      ) : (
                        <Circle size={12} className="text-slate-600" />
                      )}
                      <span className={cap.monitored ? "text-emerald-400" : "text-slate-500"}>
                        {cap.monitored ? "Monitored" : "Available"}
                      </span>
                    </>
                  )}
                  {cap.monitored === undefined && (
                    <span className="text-slate-500">Available</span>
                  )}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
