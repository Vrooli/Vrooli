import { useState } from "react";
import { Shield, Play, CheckCircle2, AlertTriangle, XCircle, Server, Globe, Key, Network, HardDrive, Cpu, Wifi, Loader2, Zap, Trash2, Info, ChevronDown, ChevronUp, Copy, Check } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import type { useDeployment } from "../../hooks/useDeployment";
import type { PreflightCheck, PreflightCheckStatus, DiskUsageResponse, DiskUsageEntry } from "../../lib/api";
import { stopPortServices, getDiskUsage, runDiskCleanup } from "../../lib/api";

interface StepPreflightProps {
  deployment: ReturnType<typeof useDeployment>;
}

// Map check IDs to icons
const CHECK_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  dns_vps_host: Globe,
  dns_edge_domain: Globe,
  dns_points_to_vps: Network,
  ssh_connect: Key,
  os_release: Server,
  ports_80_443: Network,
  outbound_network: Wifi,
  disk_free: HardDrive,
  ram_total: Cpu,
};

// Check definitions - used for preview and running states
const CHECK_DEFINITIONS = [
  { id: "dns_vps_host", title: "Resolve VPS host", description: "Verify VPS hostname resolves to IP" },
  { id: "dns_edge_domain", title: "Resolve edge domain", description: "Verify domain resolves to IP" },
  { id: "dns_points_to_vps", title: "DNS points to VPS", description: "Confirm domain points to VPS IP" },
  { id: "ssh_connect", title: "SSH connectivity", description: "Test SSH connection to server" },
  { id: "os_release", title: "Ubuntu version", description: "Check Ubuntu version" },
  { id: "ports_80_443", title: "Ports 80/443 availability", description: "Verify ports are free for Caddy" },
  { id: "outbound_network", title: "Outbound network", description: "Test outbound HTTPS access" },
  { id: "disk_free", title: "Disk free space", description: "Ensure sufficient disk space" },
  { id: "ram_total", title: "RAM", description: "Check available RAM" },
];

type CheckState = "pending" | "running" | PreflightCheckStatus;

function getStatusIcon(state: CheckState) {
  switch (state) {
    case "pass":
      return CheckCircle2;
    case "warn":
      return AlertTriangle;
    case "fail":
      return XCircle;
    case "running":
      return Loader2;
    case "pending":
      return null;
  }
}

function getStatusColor(state: CheckState) {
  switch (state) {
    case "pass":
      return {
        bg: "bg-emerald-500/20",
        text: "text-emerald-400",
        border: "border-emerald-500/30",
      };
    case "warn":
      return {
        bg: "bg-amber-500/20",
        text: "text-amber-400",
        border: "border-amber-500/30",
      };
    case "fail":
      return {
        bg: "bg-red-500/20",
        text: "text-red-400",
        border: "border-red-500/30",
      };
    case "running":
      return {
        bg: "bg-blue-500/20",
        text: "text-blue-400",
        border: "border-blue-500/30",
      };
    case "pending":
      return {
        bg: "bg-slate-700",
        text: "text-slate-400",
        border: "border-slate-700",
      };
  }
}

interface ActionButtonProps {
  onClick: () => void;
  loading?: boolean;
  icon: React.ComponentType<{ className?: string }>;
  children: React.ReactNode;
  variant?: "primary" | "secondary";
}

function ActionButton({ onClick, loading, icon: Icon, children, variant = "primary" }: ActionButtonProps) {
  const baseClasses = "inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-md transition-colors";
  const variantClasses = variant === "primary"
    ? "bg-blue-600 hover:bg-blue-500 text-white"
    : "bg-slate-700 hover:bg-slate-600 text-slate-200";

  return (
    <button
      onClick={onClick}
      disabled={loading}
      className={`${baseClasses} ${variantClasses} disabled:opacity-50 disabled:cursor-not-allowed`}
    >
      {loading ? (
        <Loader2 className="h-3 w-3 animate-spin" />
      ) : (
        <Icon className="h-3 w-3" />
      )}
      {children}
    </button>
  );
}

interface CopyButtonProps {
  text: string;
}

function CopyButton({ text }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for older browsers
      const textarea = document.createElement("textarea");
      textarea.value = text;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-1 px-2 py-0.5 text-xs bg-slate-700 hover:bg-slate-600 rounded transition-colors"
      title="Copy to clipboard"
    >
      {copied ? (
        <>
          <Check className="h-3 w-3 text-green-400" />
          <span className="text-green-400">Copied!</span>
        </>
      ) : (
        <>
          <Copy className="h-3 w-3" />
          <span>Copy</span>
        </>
      )}
    </button>
  );
}

interface DiskUsageModalProps {
  usage: DiskUsageResponse | null;
  loading: boolean;
  onClose: () => void;
  onCleanup: (actions: string[]) => void;
  cleanupLoading: boolean;
}

function DiskUsageModal({ usage, loading, onClose, onCleanup, cleanupLoading }: DiskUsageModalProps) {
  if (!usage && !loading) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-slate-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[80vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <h3 className="text-lg font-semibold text-slate-200 mb-4 flex items-center gap-2">
          <HardDrive className="h-5 w-5" />
          Disk Usage Details
        </h3>

        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-blue-400" />
          </div>
        ) : usage ? (
          <div className="space-y-4">
            {/* Summary */}
            <div className="grid grid-cols-3 gap-4">
              <div className="bg-slate-700/50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-slate-200">{usage.free_space}</div>
                <div className="text-xs text-slate-400">Free</div>
              </div>
              <div className="bg-slate-700/50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-slate-200">{usage.total_space}</div>
                <div className="text-xs text-slate-400">Total</div>
              </div>
              <div className="bg-slate-700/50 rounded-lg p-3 text-center">
                <div className="text-2xl font-bold text-slate-200">{usage.used_percent}%</div>
                <div className="text-xs text-slate-400">Used</div>
              </div>
            </div>

            {/* Largest directories */}
            {usage.largest_dirs && usage.largest_dirs.length > 0 && (
              <div>
                <h4 className="text-sm font-medium text-slate-300 mb-2">Largest Directories</h4>
                <div className="bg-slate-900/50 rounded-lg p-3 space-y-1">
                  {usage.largest_dirs.map((dir: DiskUsageEntry, i: number) => (
                    <div key={i} className="flex justify-between text-sm">
                      <span className="text-slate-400 font-mono truncate">{dir.path}</span>
                      <span className="text-slate-300 ml-2">{dir.size}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Cleanup actions */}
            <div>
              <h4 className="text-sm font-medium text-slate-300 mb-2">Quick Cleanup</h4>
              <div className="flex flex-wrap gap-2">
                <ActionButton
                  onClick={() => onCleanup(["apt_clean"])}
                  loading={cleanupLoading}
                  icon={Trash2}
                  variant="secondary"
                >
                  Clean apt cache
                </ActionButton>
                <ActionButton
                  onClick={() => onCleanup(["journal_vacuum"])}
                  loading={cleanupLoading}
                  icon={Trash2}
                  variant="secondary"
                >
                  Vacuum journals
                </ActionButton>
                <ActionButton
                  onClick={() => onCleanup(["docker_prune"])}
                  loading={cleanupLoading}
                  icon={Trash2}
                  variant="secondary"
                >
                  Prune Docker
                </ActionButton>
                <ActionButton
                  onClick={() => onCleanup(["apt_clean", "journal_vacuum", "docker_prune", "tmp_clean"])}
                  loading={cleanupLoading}
                  icon={Zap}
                >
                  Run All
                </ActionButton>
              </div>
            </div>
          </div>
        ) : null}

        <div className="mt-6 flex justify-end">
          <Button variant="ghost" onClick={onClose}>Close</Button>
        </div>
      </div>
    </div>
  );
}

interface CheckItemProps {
  id: string;
  title: string;
  description: string;
  state: CheckState;
  details?: string;
  hint?: string;
  data?: Record<string, string>;
  onAction?: (action: string) => void;
  actionLoading?: boolean;
}

function CheckItem({ id, title, description, state, details, hint, data, onAction, actionLoading }: CheckItemProps) {
  const [expanded, setExpanded] = useState(false);
  const Icon = CHECK_ICONS[id] || Server;
  const StatusIcon = getStatusIcon(state);
  const colors = getStatusColor(state);

  // Determine if this check has actions available
  const hasPortsAction = id === "ports_80_443" && state === "fail" && data?.processes;
  const hasDiskAction = id === "disk_free" && (state === "fail" || state === "warn");
  const hasDNSInstructions = id === "dns_points_to_vps" && state === "fail" && hint && hint.includes("\n");

  // Check if hint is multi-line (for expandable display)
  const isMultiLineHint = hint && hint.includes("\n");

  return (
    <li className={`p-3 rounded-lg bg-slate-800/50 border ${colors.border}`}>
      <div className="flex items-center gap-4">
        <div className="flex-shrink-0">
          <div className={`w-8 h-8 rounded-full flex items-center justify-center ${colors.bg}`}>
            <Icon className={`h-4 w-4 ${colors.text}`} />
          </div>
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-slate-200">{title}</p>
          <p className="text-xs text-slate-400 mt-0.5">
            {details || description}
          </p>
        </div>
        <div className="flex-shrink-0 flex items-center gap-2">
          {/* Action buttons */}
          {hasPortsAction && onAction && (
            <ActionButton
              onClick={() => onAction("stop_ports")}
              loading={actionLoading}
              icon={Zap}
            >
              Free Ports
            </ActionButton>
          )}
          {hasDiskAction && onAction && (
            <ActionButton
              onClick={() => onAction("show_disk")}
              loading={actionLoading}
              icon={Info}
              variant="secondary"
            >
              Details
            </ActionButton>
          )}
          {StatusIcon ? (
            <StatusIcon className={`h-5 w-5 ${colors.text} ${state === "running" ? "animate-spin" : ""}`} />
          ) : (
            <div className="w-5 h-5 rounded-full border-2 border-slate-600" />
          )}
        </div>
      </div>

      {/* Hint section */}
      {hint && state !== "pass" && state !== "pending" && state !== "running" && (
        <div className="mt-2 ml-12">
          {isMultiLineHint ? (
            <div className={`text-xs ${colors.text} bg-slate-900/50 rounded px-3 py-2`}>
              <button
                onClick={() => setExpanded(!expanded)}
                className="flex items-center gap-1 font-medium hover:underline"
              >
                {expanded ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                {expanded ? "Hide instructions" : "Show instructions"}
              </button>
              {expanded && (
                <div className="mt-2 space-y-2">
                  {hint.split("\n\n").map((section, i) => (
                    <div key={i} className="text-slate-300">
                      {section.split("\n").map((line, j) => {
                        // Check if line looks like a command
                        if (line.startsWith("- ") && line.includes(":")) {
                          const [provider, ...rest] = line.substring(2).split(":");
                          return (
                            <div key={j} className="ml-2">
                              <span className="font-medium">{provider}:</span>
                              <span className="text-slate-400">{rest.join(":")}</span>
                            </div>
                          );
                        }
                        return <div key={j}>{line}</div>;
                      })}
                    </div>
                  ))}
                  {hasDNSInstructions && data?.vps_ips && (
                    <div className="mt-2 flex items-center gap-2">
                      <span className="text-slate-400">Target IP:</span>
                      <code className="bg-slate-800 px-2 py-0.5 rounded text-slate-200">{data.vps_ips.split(",")[0]}</code>
                      <CopyButton text={data.vps_ips.split(",")[0]} />
                    </div>
                  )}
                </div>
              )}
            </div>
          ) : (
            <div className={`text-xs ${colors.text} bg-slate-900/50 rounded px-2 py-1`}>
              <span className="font-medium">Hint:</span> {hint}
            </div>
          )}
        </div>
      )}
    </li>
  );
}

export function StepPreflight({ deployment }: StepPreflightProps) {
  const {
    preflightPassed,
    preflightChecks,
    preflightError,
    isRunningPreflight,
    runPreflight,
    parsedManifest,
  } = deployment;

  // Get manifest from parsed result
  const manifest = parsedManifest.ok ? parsedManifest.value : null;

  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [diskUsage, setDiskUsage] = useState<DiskUsageResponse | null>(null);
  const [showDiskModal, setShowDiskModal] = useState(false);
  const [cleanupLoading, setCleanupLoading] = useState(false);

  const failCount = preflightChecks?.filter(c => c.status === "fail").length ?? 0;
  const warnCount = preflightChecks?.filter(c => c.status === "warn").length ?? 0;

  // Get SSH config from manifest
  const getSSHConfig = () => {
    const vps = manifest?.target?.vps;
    return {
      host: vps?.host || "",
      port: vps?.port || 22,
      user: vps?.user || "root",
      key_path: vps?.key_path || "",
    };
  };

  const handleAction = async (checkId: string, action: string) => {
    const sshConfig = getSSHConfig();
    if (!sshConfig.host || !sshConfig.key_path) {
      console.error("Missing SSH configuration");
      return;
    }

    setActionLoading(`${checkId}:${action}`);

    try {
      if (action === "stop_ports") {
        const result = await stopPortServices(sshConfig);
        if (result.ok) {
          // Re-run preflight to verify
          runPreflight();
        }
      } else if (action === "show_disk") {
        const usage = await getDiskUsage(sshConfig);
        setDiskUsage(usage);
        setShowDiskModal(true);
      }
    } catch (error) {
      console.error(`Action ${action} failed:`, error);
    } finally {
      setActionLoading(null);
    }
  };

  const handleCleanup = async (actions: string[]) => {
    const sshConfig = getSSHConfig();
    if (!sshConfig.host || !sshConfig.key_path) return;

    setCleanupLoading(true);
    try {
      await runDiskCleanup({ ...sshConfig, actions });
      // Refresh disk usage
      const usage = await getDiskUsage(sshConfig);
      setDiskUsage(usage);
      // Re-run preflight
      runPreflight();
    } catch (error) {
      console.error("Cleanup failed:", error);
    } finally {
      setCleanupLoading(false);
    }
  };

  // Build the display list: merge definitions with real results
  const checksToDisplay = CHECK_DEFINITIONS.map((def) => {
    const realCheck = preflightChecks?.find(c => c.id === def.id);

    let state: CheckState;
    if (realCheck) {
      state = realCheck.status;
    } else if (isRunningPreflight) {
      state = "running";
    } else {
      state = "pending";
    }

    return {
      id: def.id,
      title: realCheck?.title || def.title,
      description: def.description,
      state,
      details: realCheck?.details,
      hint: realCheck?.hint,
      data: realCheck?.data,
    };
  });

  return (
    <div className="space-y-6">
      {/* Info Banner */}
      <Alert variant="info" title="VPS Preflight Checks">
        Before deployment, we verify your target server is properly configured and accessible.
      </Alert>

      {/* Run Preflight Button */}
      <div className="flex items-center gap-3">
        <Button
          onClick={runPreflight}
          disabled={isRunningPreflight || !parsedManifest.ok}
        >
          {isRunningPreflight ? (
            <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
          ) : (
            <Play className="h-4 w-4 mr-1.5" />
          )}
          {isRunningPreflight ? "Running Checks..." : "Run Preflight Checks"}
        </Button>
      </div>

      {/* Error from API call itself */}
      {preflightError && (
        <Alert variant="error" title="Preflight Failed">
          {preflightError}
        </Alert>
      )}

      {/* Success - all passed */}
      {preflightPassed === true && warnCount === 0 && !isRunningPreflight && (
        <Alert variant="success" title="Preflight Passed">
          <div className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            All checks passed. Your server is ready for deployment.
          </div>
        </Alert>
      )}

      {/* Partial Success - passed with warnings */}
      {preflightPassed === true && warnCount > 0 && !isRunningPreflight && (
        <Alert variant="warning" title="Preflight Passed with Warnings">
          <div className="flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            {warnCount} warning{warnCount !== 1 ? "s" : ""} detected. Review before proceeding.
          </div>
        </Alert>
      )}

      {/* Failed - has failures */}
      {preflightPassed === false && failCount > 0 && !isRunningPreflight && !preflightError && (
        <Alert variant="error" title="Preflight Failed">
          <div className="flex items-center gap-2">
            <XCircle className="h-4 w-4" />
            {failCount} check{failCount !== 1 ? "s" : ""} failed. Address the issues below before deployment.
          </div>
        </Alert>
      )}

      {/* Preflight Checks List - Always visible */}
      <div className="space-y-3">
        <h4 className="text-sm font-medium text-slate-300">
          Preflight Checks ({checksToDisplay.length})
        </h4>
        <ul className="space-y-2">
          {checksToDisplay.map((check) => (
            <CheckItem
              key={check.id}
              id={check.id}
              title={check.title}
              description={check.description}
              state={check.state}
              details={check.details}
              hint={check.hint}
              data={check.data}
              onAction={(action) => handleAction(check.id, action)}
              actionLoading={actionLoading?.startsWith(`${check.id}:`)}
            />
          ))}
        </ul>
      </div>

      {/* Disk Usage Modal */}
      {showDiskModal && (
        <DiskUsageModal
          usage={diskUsage}
          loading={actionLoading === "disk_free:show_disk"}
          onClose={() => setShowDiskModal(false)}
          onCleanup={handleCleanup}
          cleanupLoading={cleanupLoading}
        />
      )}
    </div>
  );
}
