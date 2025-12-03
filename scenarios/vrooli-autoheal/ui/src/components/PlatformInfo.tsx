// Platform capabilities display component
// [REQ:UI-HEALTH-001]
import { Server } from "lucide-react";
import type { PlatformCapabilities } from "../lib/api";

interface PlatformInfoProps {
  platform: PlatformCapabilities;
}

export function PlatformInfo({ platform }: PlatformInfoProps) {
  const capabilities: string[] = [
    platform.hasDocker && "Docker",
    platform.supportsSystemd && "Systemd",
    platform.supportsLaunchd && "Launchd",
    platform.supportsRdp && "RDP",
    platform.supportsCloudflared && "Cloudflared",
    platform.isWsl && "WSL",
  ].filter((cap): cap is string => typeof cap === "string");

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
        {capabilities.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-2">
            {capabilities.map((cap) => (
              <span key={cap} className="px-2 py-0.5 text-xs rounded bg-slate-800 text-slate-300">
                {cap}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
