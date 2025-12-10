import { useQuery } from "@tanstack/react-query";
import { Shield, ShieldAlert, ShieldCheck, Info, AlertTriangle, ShieldOff } from "lucide-react";
import { fetchContainmentStatus, type ContainmentStatus } from "../../lib/api";
import { cn } from "../../lib/utils";
import { useState } from "react";

function getSecurityColor(level: number, max: number): string {
  const ratio = level / max;
  if (ratio >= 0.7) return "text-emerald-400 border-emerald-400/50 bg-emerald-400/10";
  if (ratio >= 0.4) return "text-amber-400 border-amber-400/50 bg-amber-400/10";
  return "text-rose-400 border-rose-400/50 bg-rose-400/10";
}

function getSecurityIcon(level: number, max: number) {
  const ratio = level / max;
  if (ratio >= 0.7) return ShieldCheck;
  if (ratio >= 0.4) return Shield;
  return ShieldAlert;
}

function getSecurityLabel(level: number, max: number): string {
  const ratio = level / max;
  if (ratio >= 0.7) return "Strong";
  if (ratio >= 0.4) return "Moderate";
  if (ratio > 0) return "Weak";
  return "None";
}

export interface ContainmentBadgeProps {
  /** If true, shows the full warning banner instead of just the badge */
  showWarningBanner?: boolean;
}

export function ContainmentBadge({ showWarningBanner = false }: ContainmentBadgeProps) {
  const [showDetails, setShowDetails] = useState(false);

  const { data: status, isLoading, error } = useQuery({
    queryKey: ["containment-status"],
    queryFn: fetchContainmentStatus,
    staleTime: 60000, // Cache for 1 minute
  });

  if (isLoading) {
    return (
      <div className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-xs text-slate-400">
        Checking containment...
      </div>
    );
  }

  if (error || !status) {
    return (
      <div className="rounded-lg border border-rose-400/30 bg-rose-400/5 px-3 py-2 text-xs text-rose-300">
        Failed to check containment status
      </div>
    );
  }

  const colorClass = getSecurityColor(status.securityLevel, status.maxSecurityLevel);
  const Icon = getSecurityIcon(status.securityLevel, status.maxSecurityLevel);
  const securityLabel = getSecurityLabel(status.securityLevel, status.maxSecurityLevel);
  const hasWarnings = status.warnings && status.warnings.length > 0;
  const isNoContainment = status.activeProvider === "none" || status.securityLevel === 0;

  // Warning banner for no containment scenario
  if (showWarningBanner && isNoContainment) {
    return (
      <div className="rounded-xl border-2 border-rose-500/50 bg-rose-950/30 p-4">
        <div className="flex items-start gap-3">
          <ShieldOff className="h-6 w-6 text-rose-400 flex-shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0">
            <h4 className="font-semibold text-rose-100 flex items-center gap-2">
              No OS-Level Containment
              <span className="rounded-full bg-rose-500/20 border border-rose-500/50 px-2 py-0.5 text-[10px] font-medium">
                REDUCED SECURITY
              </span>
            </h4>
            <p className="mt-1 text-sm text-rose-200/80">
              Agents will run without Docker isolation. While tool-level restrictions (directory scoping, command allowlists)
              are still enforced, OS-level containment provides an additional security layer that is currently unavailable.
            </p>
            <div className="mt-3 text-xs text-rose-300/70 space-y-1">
              <p className="flex items-center gap-2">
                <span className="text-emerald-400">✓</span>
                <span>Directory scoping: Active (agents can only access scenario files)</span>
              </p>
              <p className="flex items-center gap-2">
                <span className="text-emerald-400">✓</span>
                <span>Command allowlist: Active (only permitted commands can run)</span>
              </p>
              <p className="flex items-center gap-2">
                <span className="text-rose-400">✗</span>
                <span>Process isolation: Unavailable (install Docker for this)</span>
              </p>
              <p className="flex items-center gap-2">
                <span className="text-rose-400">✗</span>
                <span>Memory/CPU limits: Unavailable (install Docker for this)</span>
              </p>
            </div>
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="mt-3 text-xs text-rose-300 hover:text-white underline"
            >
              {showDetails ? "Hide details" : "Learn more"}
            </button>
          </div>
        </div>

        {showDetails && (
          <div className="mt-4 pt-4 border-t border-rose-500/30 text-xs text-rose-200/70 space-y-2">
            <p>
              <strong>Why Docker containment matters:</strong> Docker provides OS-level isolation by running agents
              in containers with restricted capabilities, no network access, and resource limits. This prevents
              agents from affecting your system even if they attempt to escape their tool restrictions.
            </p>
            <p>
              <strong>To enable containment:</strong> Install Docker and ensure it's running. Test-genie will
              automatically detect and use it for agent execution.
            </p>
          </div>
        )}
      </div>
    );
  }

  return (
    <div className="relative">
      <button
        onClick={() => setShowDetails(!showDetails)}
        className={cn(
          "flex items-center gap-2 rounded-lg border px-3 py-2 text-xs transition-colors",
          colorClass,
          "hover:bg-opacity-20 cursor-pointer"
        )}
      >
        <Icon className="h-4 w-4" />
        <span className="font-medium">
          {isNoContainment
            ? "No Containment"
            : `${status.activeProvider.charAt(0).toUpperCase() + status.activeProvider.slice(1)} Containment`
          }
        </span>
        <span className={cn(
          "rounded-full px-1.5 py-0.5 text-[10px] font-medium",
          status.securityLevel >= 7 ? "bg-emerald-400/20 text-emerald-300" :
          status.securityLevel >= 4 ? "bg-amber-400/20 text-amber-300" :
          "bg-rose-400/20 text-rose-300"
        )}>
          {securityLabel}
        </span>
        {hasWarnings && <AlertTriangle className="h-3 w-3 text-amber-400" />}
        {isNoContainment && <ShieldOff className="h-3 w-3 text-rose-400" />}
      </button>

      {showDetails && (
        <div className="absolute top-full left-0 mt-2 z-50 w-80 rounded-xl border border-white/10 bg-slate-900 p-4 shadow-xl">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-semibold text-white flex items-center gap-2">
              <Shield className="h-4 w-4" />
              Containment Status
            </h4>
            <button
              onClick={() => setShowDetails(false)}
              className="text-slate-400 hover:text-white text-xs"
            >
              Close
            </button>
          </div>

          {/* Security Level Bar */}
          <div className="mb-4">
            <div className="flex justify-between text-xs text-slate-400 mb-1">
              <span>Security Level</span>
              <span>{status.securityLevel}/{status.maxSecurityLevel}</span>
            </div>
            <div className="h-2 rounded-full bg-white/10 overflow-hidden">
              <div
                className={cn(
                  "h-full rounded-full transition-all",
                  status.securityLevel >= 7 ? "bg-emerald-400" :
                  status.securityLevel >= 4 ? "bg-amber-400" : "bg-rose-400"
                )}
                style={{ width: `${(status.securityLevel / status.maxSecurityLevel) * 100}%` }}
              />
            </div>
          </div>

          {/* Active Provider */}
          <div className="mb-3 p-2 rounded-lg border border-white/10 bg-white/5">
            <span className="text-xs text-slate-400">Active Provider:</span>
            <p className="text-sm text-white font-medium">
              {status.providers.find(p => p.type === status.activeProvider)?.name || status.activeProvider}
            </p>
          </div>

          {/* Warnings */}
          {hasWarnings && (
            <div className="mb-3 space-y-2">
              {status.warnings.map((warning, idx) => (
                <div
                  key={idx}
                  className="flex items-start gap-2 p-2 rounded-lg border border-amber-400/30 bg-amber-400/5 text-xs text-amber-200"
                >
                  <AlertTriangle className="h-3 w-3 flex-shrink-0 mt-0.5" />
                  <span>{warning}</span>
                </div>
              ))}
            </div>
          )}

          {/* Available Providers */}
          <div className="space-y-2">
            <span className="text-xs text-slate-400">Available Providers:</span>
            {status.providers.map((provider) => (
              <div
                key={provider.type}
                className={cn(
                  "p-2 rounded-lg border text-xs",
                  status.activeProvider === provider.type
                    ? "border-cyan-400/50 bg-cyan-400/10"
                    : "border-white/10 bg-white/5"
                )}
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium text-white">{provider.name}</span>
                  <span className={cn(
                    "px-1.5 py-0.5 rounded text-[10px]",
                    provider.securityLevel >= 7 ? "bg-emerald-400/20 text-emerald-300" :
                    provider.securityLevel >= 4 ? "bg-amber-400/20 text-amber-300" :
                    "bg-rose-400/20 text-rose-300"
                  )}>
                    Level {provider.securityLevel}
                  </span>
                </div>
                <p className="text-slate-400 mt-1">{provider.description}</p>
              </div>
            ))}
          </div>

          {/* Info note */}
          <div className="mt-3 flex items-start gap-2 text-[10px] text-slate-500">
            <Info className="h-3 w-3 flex-shrink-0 mt-0.5" />
            <span>{status.note}</span>
          </div>
        </div>
      )}
    </div>
  );
}
