// System Protection status component
// [REQ:WATCH-DETECT-001] [REQ:UI-HEALTH-001]
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Shield, CheckCircle, AlertTriangle, XCircle, ChevronDown, ChevronUp, ChevronRight, Copy, Check, RefreshCw } from "lucide-react";
import { fetchWatchdogStatus, fetchWatchdogTemplate, ProtectionLevel, WatchdogStatus } from "../lib/api";
import { ErrorDisplay } from "./ErrorDisplay";
import { selectors } from "../consts/selectors";

// Protection level colors and labels
const PROTECTION_CONFIG: Record<ProtectionLevel, { color: string; bgColor: string; label: string; icon: typeof CheckCircle }> = {
  full: {
    color: "text-emerald-400",
    bgColor: "bg-emerald-500/20",
    label: "Full Protection",
    icon: CheckCircle,
  },
  partial: {
    color: "text-amber-400",
    bgColor: "bg-amber-500/20",
    label: "Partial Protection",
    icon: AlertTriangle,
  },
  none: {
    color: "text-red-400",
    bgColor: "bg-red-500/20",
    label: "Not Protected",
    icon: XCircle,
  },
};

// Status indicator component
function StatusIndicator({ active, label }: { active: boolean; label: string }) {
  return (
    <div className="flex items-center justify-between py-1.5">
      <span className="text-sm text-slate-400">{label}</span>
      <span className={`flex items-center gap-1.5 text-sm ${active ? "text-emerald-400" : "text-slate-500"}`}>
        <span className={`w-2 h-2 rounded-full ${active ? "bg-emerald-400" : "bg-slate-600"}`} />
        {active ? "Active" : "Inactive"}
      </span>
    </div>
  );
}

// Installation instructions modal/accordion
function InstallationGuide({ onClose }: { onClose: () => void }) {
  const [copiedOneLiner, setCopiedOneLiner] = useState(false);
  const [copiedTemplate, setCopiedTemplate] = useState(false);
  const [showTemplate, setShowTemplate] = useState(false);

  const { data, isLoading, error } = useQuery({
    queryKey: ["watchdog-template"],
    queryFn: fetchWatchdogTemplate,
  });

  const handleCopyOneLiner = async () => {
    if (data?.oneLiner) {
      await navigator.clipboard.writeText(data.oneLiner);
      setCopiedOneLiner(true);
      setTimeout(() => setCopiedOneLiner(false), 2000);
    }
  };

  const handleCopyTemplate = async () => {
    if (data?.template) {
      await navigator.clipboard.writeText(data.template);
      setCopiedTemplate(true);
      setTimeout(() => setCopiedTemplate(false), 2000);
    }
  };

  if (isLoading) {
    return (
      <div className="mt-3 pt-3 border-t border-white/10">
        <p className="text-sm text-slate-500">Loading installation guide...</p>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="mt-3 pt-3 border-t border-white/10">
        <ErrorDisplay error={error} compact />
      </div>
    );
  }

  return (
    <div className="mt-3 pt-3 border-t border-white/10 space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-medium text-slate-200">Quick Install</h4>
        <button
          onClick={onClose}
          className="text-xs text-slate-500 hover:text-slate-300"
        >
          Close
        </button>
      </div>

      {/* One-liner install command */}
      {data.oneLiner && (
        <div className="space-y-1">
          <p className="text-xs text-slate-400">
            Run this command in your terminal (requires <code className="text-blue-400">jq</code>):
          </p>
          <div className="relative group">
            <pre className="text-xs bg-slate-900 rounded p-2 pr-10 overflow-x-auto text-emerald-400 font-mono whitespace-pre-wrap break-all">
              {data.oneLiner}
            </pre>
            <button
              onClick={handleCopyOneLiner}
              className="absolute top-2 right-2 p-1.5 rounded bg-slate-800 hover:bg-slate-700 transition-colors"
              title="Copy command"
            >
              {copiedOneLiner ? (
                <Check size={14} className="text-emerald-400" />
              ) : (
                <Copy size={14} className="text-slate-400" />
              )}
            </button>
          </div>
        </div>
      )}

      {/* Manual instructions toggle */}
      <button
        onClick={() => setShowTemplate(!showTemplate)}
        className="text-xs text-slate-500 hover:text-slate-300 flex items-center gap-1"
      >
        {showTemplate ? "Hide" : "Show"} manual instructions
        <ChevronRight size={12} className={`transition-transform ${showTemplate ? "rotate-90" : ""}`} />
      </button>

      {/* Manual instructions (collapsed by default) */}
      {showTemplate && (
        <div className="space-y-2 pl-2 border-l-2 border-slate-700">
          <div className="text-xs text-slate-400 space-y-1">
            {data.instructions.split("\n").map((line, i) => (
              <p key={i}>{line}</p>
            ))}
          </div>

          {/* Template with copy button */}
          <div className="relative">
            <pre className="text-xs bg-slate-900 rounded p-2 overflow-x-auto max-h-32 text-slate-300 font-mono">
              {data.template.slice(0, 500)}{data.template.length > 500 ? "..." : ""}
            </pre>
            <button
              onClick={handleCopyTemplate}
              className="absolute top-2 right-2 p-1 rounded bg-slate-800 hover:bg-slate-700 transition-colors"
              title="Copy template"
            >
              {copiedTemplate ? (
                <Check size={14} className="text-emerald-400" />
              ) : (
                <Copy size={14} className="text-slate-400" />
              )}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

interface SystemProtectionProps {
  compact?: boolean;
}

export function SystemProtection({ compact = false }: SystemProtectionProps) {
  const [showGuide, setShowGuide] = useState(false);

  const { data, isLoading, error, refetch, isFetching } = useQuery({
    queryKey: ["watchdog"],
    queryFn: () => fetchWatchdogStatus(false),
    staleTime: 60000, // Cache for 1 minute
    refetchInterval: 120000, // Refresh every 2 minutes
  });

  const handleRefresh = () => {
    refetch();
  };

  if (isLoading) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-4" data-testid={selectors.systemProtection}>
        <div className="flex items-center gap-2 mb-3">
          <Shield size={18} className="text-blue-400" />
          <h3 className="font-medium text-sm">System Protection</h3>
        </div>
        <p className="text-sm text-slate-500">Loading...</p>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 p-4" data-testid={selectors.systemProtection}>
        <div className="flex items-center gap-2 mb-3">
          <Shield size={18} className="text-blue-400" />
          <h3 className="font-medium text-sm">System Protection</h3>
        </div>
        <ErrorDisplay error={error} onRetry={() => refetch()} compact />
      </div>
    );
  }

  const config = PROTECTION_CONFIG[data.protectionLevel];
  const Icon = config.icon;

  // Compact mode for header
  if (compact) {
    return (
      <div
        className={`flex items-center gap-1.5 px-2 py-1 rounded ${config.bgColor}`}
        title={config.label}
        data-testid={selectors.systemProtectionCompact}
      >
        <Icon size={14} className={config.color} />
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-white/10 bg-white/5 p-4" data-testid={selectors.systemProtection}>
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Shield size={18} className="text-blue-400" />
          <h3 className="font-medium text-sm">System Protection</h3>
        </div>
        <button
          onClick={handleRefresh}
          disabled={isFetching}
          className="p-1 rounded hover:bg-white/10 transition-colors"
          title="Refresh status"
        >
          <RefreshCw size={14} className={`text-slate-400 ${isFetching ? "animate-spin" : ""}`} />
        </button>
      </div>

      {/* Protection Level Badge */}
      <div className={`flex items-center gap-2 p-2 rounded-lg ${config.bgColor} mb-3`}>
        <Icon size={18} className={config.color} />
        <span className={`text-sm font-medium ${config.color}`}>{config.label}</span>
      </div>

      {/* Status Details */}
      <div className="space-y-0.5">
        <StatusIndicator active={data.loopRunning} label="Autoheal Loop" />
        <StatusIndicator active={data.watchdogInstalled} label="OS Watchdog" />
        <StatusIndicator active={data.bootProtectionActive} label="Boot Recovery" />
      </div>

      {/* Watchdog Type Info */}
      {data.watchdogType && (
        <p className="text-xs text-slate-500 mt-2">
          Type: <span className="text-slate-400">{data.watchdogType}</span>
          {data.servicePath && (
            <span className="block truncate" title={data.servicePath}>
              Path: {data.servicePath}
            </span>
          )}
        </p>
      )}

      {/* Error Display */}
      {data.lastError && (
        <p className="text-xs text-amber-400 mt-2">{data.lastError}</p>
      )}

      {/* Install Button / Guide Toggle */}
      {!data.watchdogInstalled && data.canInstall && (
        <button
          onClick={() => setShowGuide(!showGuide)}
          className="mt-3 w-full flex items-center justify-between px-3 py-2 text-sm bg-blue-500/20 hover:bg-blue-500/30 text-blue-400 rounded-lg transition-colors"
        >
          <span>Set up OS Watchdog</span>
          {showGuide ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>
      )}

      {/* Installation Guide */}
      {showGuide && <InstallationGuide onClose={() => setShowGuide(false)} />}

      {/* Cannot Install Warning */}
      {!data.watchdogInstalled && !data.canInstall && (
        <p className="text-xs text-slate-500 mt-3">
          OS watchdog not available on this platform
        </p>
      )}
    </div>
  );
}

// Export a hook for getting protection status elsewhere
export function useProtectionStatus(): { status: WatchdogStatus | undefined; isLoading: boolean } {
  const { data, isLoading } = useQuery({
    queryKey: ["watchdog"],
    queryFn: () => fetchWatchdogStatus(false),
    staleTime: 60000,
  });

  return { status: data, isLoading };
}
