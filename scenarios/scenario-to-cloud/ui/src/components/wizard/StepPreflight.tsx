import { useMemo, useState } from "react";
import { Shield, Play, CheckCircle2, AlertTriangle, XCircle, Server, Globe, Key, Network, HardDrive, Cpu, Wifi, Loader2, Zap, Trash2, Info, ChevronDown, ChevronUp, Copy, Check, Package, Activity, Database, Square, X } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import type { useDeployment } from "../../hooks/useDeployment";
import type { PreflightCheck, PreflightCheckStatus, DiskUsageResponse, DiskUsageEntry } from "../../lib/api";
import { stopPortServices, getDiskUsage, runDiskCleanup, stopScenarioProcesses, openFirewallPorts } from "../../lib/api";
import { DEFAULT_VPS_WORKDIR } from "../../lib/constants";

interface StepPreflightProps {
  deployment: ReturnType<typeof useDeployment>;
}

// Map check IDs to icons
const CHECK_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  dns_vps_host: Globe,
  dns_edge_apex: Globe,
  dns_edge_www: Globe,
  dns_do_origin: Globe,
  dns_og_worker_ready: Zap,
  dns_edge_ipv6: Globe,
  ssh_connect: Key,
  os_release: Server,
  ports_80_443: Network,
  firewall_inbound: Shield,
  outbound_network: Wifi,
  disk_free: HardDrive,
  ram_total: Cpu,
  // Bootstrap prerequisite checks
  cmd_curl: Package,
  cmd_git: Package,
  cmd_unzip: Package,
  cmd_tar: Package,
  cmd_jq: Package,
  apt_access: Package,
  // Stale process and credential checks
  stale_processes: Activity,
  secrets_read: Key,
  postgres_credentials: Database,
  redis_credentials: Database,
};

// Check definitions - used for preview and running states
export const CHECK_DEFINITIONS = [
  { id: "dns_vps_host", title: "Resolve VPS host", description: "Verify VPS hostname resolves to IP" },
  { id: "dns_edge_apex", title: "Apex domain", description: "Verify apex domain resolves to VPS or proxy" },
  { id: "dns_edge_www", title: "WWW domain", description: "Verify www domain resolves to VPS or proxy" },
  { id: "dns_do_origin", title: "Origin domain", description: "Verify do-origin points to VPS" },
  { id: "dns_og_worker_ready", title: "OG worker readiness", description: "Check proxy + origin routing for OG worker" },
  { id: "dns_edge_ipv6", title: "Edge IPv6 records", description: "Detect mismatched AAAA records" },
  { id: "ssh_connect", title: "SSH connectivity", description: "Test SSH connection to server" },
  { id: "os_release", title: "Ubuntu version", description: "Check Ubuntu version" },
  { id: "ports_80_443", title: "Ports 80/443 availability", description: "Verify ports are free for Caddy" },
  { id: "firewall_inbound", title: "Inbound firewall rules", description: "Verify firewall allows ports 80/443" },
  { id: "outbound_network", title: "Outbound network", description: "Test outbound HTTPS access" },
  { id: "disk_free", title: "Disk free space", description: "Ensure sufficient disk space" },
  { id: "ram_total", title: "RAM", description: "Check available RAM" },
  // Bootstrap prerequisite checks
  { id: "cmd_curl", title: "curl available", description: "Check curl is installed" },
  { id: "cmd_git", title: "git available", description: "Check git is installed" },
  { id: "cmd_unzip", title: "unzip available", description: "Check unzip is installed" },
  { id: "cmd_tar", title: "tar available", description: "Check tar is installed" },
  { id: "cmd_jq", title: "jq available", description: "Check jq is installed" },
  { id: "apt_access", title: "apt accessible", description: "Verify apt-get can be run" },
  // Stale process and credential checks
  { id: "stale_processes", title: "Stale process check", description: "Check for existing scenario processes" },
  { id: "secrets_read", title: "Secrets file", description: "Verify secrets.json is readable" },
  { id: "postgres_credentials", title: "PostgreSQL credentials", description: "Verify database credentials" },
];

export type CheckState = "pending" | "running" | PreflightCheckStatus;

interface PortBindingInfo {
  port: number;
  process?: string;
  pid?: number;
  service?: string;
}

function parsePortBindings(data?: Record<string, string>): PortBindingInfo[] {
  if (!data?.port_bindings) return [];
  try {
    const parsed = JSON.parse(data.port_bindings);
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map((binding) => ({
        port: Number(binding.port),
        process: typeof binding.process === "string" ? binding.process : undefined,
        pid: typeof binding.pid === "number" ? binding.pid : undefined,
        service: typeof binding.service === "string" ? binding.service : undefined,
      }))
      .filter((binding) => Number.isFinite(binding.port));
  } catch {
    return [];
  }
}

function formatPortBinding(binding: PortBindingInfo): string {
  const parts: string[] = [];
  parts.push(`port ${binding.port}`);
  if (binding.process) {
    parts.push(binding.process);
  }
  if (binding.pid) {
    parts.push(`pid ${binding.pid}`);
  }
  if (binding.service && binding.service !== binding.process) {
    parts.push(`service ${binding.service}`);
  }
  return parts.join(" - ");
}

function buildPortSelections(bindings: PortBindingInfo[]) {
  const services: Record<string, boolean> = {};
  const pids: Record<number, boolean> = {};

  bindings.forEach((binding) => {
    if (binding.service) {
      services[binding.service] = true;
      return;
    }
    if (binding.pid) {
      pids[binding.pid] = true;
    }
  });

  return { services, pids };
}

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

export interface PreflightDisplayCheck {
  id: string;
  title: string;
  description: string;
  state: CheckState;
  details?: string;
  hint?: string;
  data?: Record<string, string>;
}

export function buildChecksToDisplay(
  preflightChecks: PreflightCheck[] | null,
  isRunningPreflight: boolean,
): PreflightDisplayCheck[] {
  return CHECK_DEFINITIONS.map((def) => {
    const realCheck = preflightChecks?.find((check) => check.id === def.id);

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
}

export function buildReadOnlyChecks(state: CheckState): PreflightDisplayCheck[] {
  return CHECK_DEFINITIONS.map((def) => ({
    id: def.id,
    title: def.title,
    description: def.description,
    state,
  }));
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

interface PortStopModalProps {
  bindings: PortBindingInfo[];
  selections: { services: Record<string, boolean>; pids: Record<number, boolean> };
  loading: boolean;
  onToggleService: (service: string) => void;
  onTogglePID: (pid: number) => void;
  onConfirm: () => void;
  onClose: () => void;
}

function PortStopModal({
  bindings,
  selections,
  loading,
  onToggleService,
  onTogglePID,
  onConfirm,
  onClose,
}: PortStopModalProps) {
  const services = Array.from(
    new Set(bindings.map((binding) => binding.service).filter(Boolean))
  ) as string[];
  const pidLabels = new Map<number, string>();

  bindings.forEach((binding) => {
    if (!binding.pid || pidLabels.has(binding.pid)) {
      return;
    }
    const details: string[] = [];
    if (binding.process) {
      details.push(binding.process);
    }
    if (binding.port) {
      details.push(`port ${binding.port}`);
    }
    const suffix = details.length > 0 ? ` - ${details.join(", ")}` : "";
    pidLabels.set(binding.pid, `PID ${binding.pid}${suffix}`);
  });

  const selectedServices = Object.values(selections.services).some(Boolean);
  const selectedPIDs = Object.values(selections.pids).some(Boolean);
  const hasSelections = selectedServices || selectedPIDs;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={onClose}>
      <div className="bg-slate-800 rounded-lg p-6 max-w-lg w-full mx-4 max-h-[80vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold text-slate-200 flex items-center gap-2">
            <Network className="h-5 w-5" />
            Stop Services on Ports 80/443
          </h3>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-slate-200"
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          <div className="text-xs text-slate-400">
            Ports 80/443 must be free for Caddy to complete Let's Encrypt HTTP-01 challenges.
          </div>

          <div>
            <h4 className="text-sm font-medium text-slate-300 mb-2">Detected listeners</h4>
            <div className="bg-slate-900/50 rounded-lg p-3 space-y-1 text-xs text-slate-300">
              {bindings.map((binding, index) => (
                <div key={`${binding.port}-${binding.pid ?? index}`}>
                  {formatPortBinding(binding)}
                </div>
              ))}
            </div>
          </div>

          <div>
            <h4 className="text-sm font-medium text-slate-300 mb-2">Stop actions</h4>
            {services.length === 0 && pidLabels.size === 0 ? (
              <div className="text-xs text-slate-400">
                No actionable service or PID data was detected. Re-run preflight with SSH access that can read process info, or stop the listener manually.
              </div>
            ) : (
              <div className="space-y-2 text-xs text-slate-300">
                {services.map((service) => (
                  <label key={service} className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={selections.services[service] ?? false}
                      onChange={() => onToggleService(service)}
                      className="h-3.5 w-3.5 rounded border-slate-600 bg-slate-700 text-blue-500 focus:ring-blue-500"
                    />
                    <span>Stop service {service}</span>
                  </label>
                ))}
                {Array.from(pidLabels.entries()).map(([pid, label]) => (
                  <label key={pid} className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={selections.pids[pid] ?? false}
                      onChange={() => onTogglePID(pid)}
                      className="h-3.5 w-3.5 rounded border-slate-600 bg-slate-700 text-blue-500 focus:ring-blue-500"
                    />
                    <span>{label}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          <div className="text-xs text-slate-400">
            We try to stop systemd services first when possible, then fall back to terminating specific PIDs.
          </div>
        </div>

        <div className="mt-6 flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose} disabled={loading}>Cancel</Button>
          <Button onClick={onConfirm} disabled={!hasSelections || loading}>
            {loading ? (
              <Loader2 className="h-4 w-4 mr-1.5 animate-spin" />
            ) : (
              <Zap className="h-4 w-4 mr-1.5" />
            )}
            Stop Selected
          </Button>
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
  readOnly?: boolean;
}

function CheckItem({
  id,
  title,
  description,
  state,
  details,
  hint,
  data,
  onAction,
  actionLoading,
  readOnly,
}: CheckItemProps) {
  const [expanded, setExpanded] = useState(false);
  const Icon = CHECK_ICONS[id] || Server;
  const StatusIcon = getStatusIcon(state);
  const colors = getStatusColor(state);
  const portBindings = id === "ports_80_443" ? parsePortBindings(data) : [];

  // Determine if this check has actions available
  const hasPortsAction = id === "ports_80_443" && state === "fail" && portBindings.length > 0;
  const hasDiskAction = id === "disk_free" && (state === "fail" || state === "warn");
  const hasFirewallAction = id === "firewall_inbound" && state === "fail";
  const hasDNSInstructions =
    (id === "dns_edge_apex" || id === "dns_edge_www" || id === "dns_do_origin") &&
    state === "fail" &&
    hint &&
    hint.includes("\n");
  const hasStaleProcessesAction = id === "stale_processes" && state === "warn";

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
          {id === "ports_80_443" && portBindings.length > 0 && state === "fail" && (
            <div className="mt-2 text-xs text-slate-300">
              <div className="font-medium text-slate-400">Detected listeners</div>
              <div className="mt-1 space-y-1">
                {portBindings.map((binding, index) => (
                  <div key={`${binding.port}-${binding.pid ?? index}`} className="text-slate-300">
                    {formatPortBinding(binding)}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
        <div className="flex-shrink-0 flex items-center gap-2">
          {/* Action buttons */}
          {!readOnly && hasPortsAction && onAction && (
            <ActionButton
              onClick={() => onAction("stop_ports")}
              loading={actionLoading}
              icon={Zap}
            >
              Review & Stop
            </ActionButton>
          )}
          {!readOnly && hasDiskAction && onAction && (
            <ActionButton
              onClick={() => onAction("show_disk")}
              loading={actionLoading}
              icon={Info}
              variant="secondary"
            >
              Details
            </ActionButton>
          )}
          {!readOnly && hasFirewallAction && onAction && (
            <ActionButton
              onClick={() => onAction("open_firewall")}
              loading={actionLoading}
              icon={Shield}
            >
              Open 80/443
            </ActionButton>
          )}
          {!readOnly && hasStaleProcessesAction && onAction && (
            <>
              <ActionButton
                onClick={() => onAction("stop_scenario")}
                loading={actionLoading}
                icon={Square}
              >
                Stop Scenario
              </ActionButton>
              <ActionButton
                onClick={() => onAction("stop_all")}
                loading={actionLoading}
                icon={Square}
                variant="secondary"
              >
                Stop All
              </ActionButton>
            </>
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

export interface PreflightChecksPanelProps {
  checksToDisplay: PreflightDisplayCheck[];
  actionLoading?: string | null;
  onAction?: (checkId: string, action: string) => void;
  readOnly?: boolean;
}

export function PreflightChecksPanel({
  checksToDisplay,
  actionLoading,
  onAction,
  readOnly,
}: PreflightChecksPanelProps) {
  return (
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
            onAction={onAction ? (action) => onAction(check.id, action) : undefined}
            actionLoading={actionLoading?.startsWith(`${check.id}:`)}
            readOnly={readOnly}
          />
        ))}
      </ul>
    </div>
  );
}

export function StepPreflight({ deployment }: StepPreflightProps) {
  const {
    preflightPassed,
    preflightChecks,
    preflightError,
    isRunningPreflight,
    preflightOverride,
    setPreflightOverride,
    runPreflight,
    parsedManifest,
    sshKeyPath,
  } = deployment;

  // Get manifest from parsed result
  const manifest = parsedManifest.ok ? parsedManifest.value : null;

  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [diskUsage, setDiskUsage] = useState<DiskUsageResponse | null>(null);
  const [showDiskModal, setShowDiskModal] = useState(false);
  const [cleanupLoading, setCleanupLoading] = useState(false);
  const [showPortModal, setShowPortModal] = useState(false);
  const [portSelections, setPortSelections] = useState<{ services: Record<string, boolean>; pids: Record<number, boolean> }>({
    services: {},
    pids: {},
  });

  const failCount = preflightChecks?.filter(c => c.status === "fail").length ?? 0;
  const warnCount = preflightChecks?.filter(c => c.status === "warn").length ?? 0;
  const portBindings = useMemo(() => {
    const portsCheck = preflightChecks?.find((check) => check.id === "ports_80_443");
    return parsePortBindings(portsCheck?.data);
  }, [preflightChecks]);

  // Get SSH config from manifest (use sshKeyPath state as fallback for key_path)
  const getSSHConfig = () => {
    const vps = manifest?.target?.vps;
    return {
      host: vps?.host || "",
      port: vps?.port || 22,
      user: vps?.user || "root",
      key_path: vps?.key_path || sshKeyPath || "",
    };
  };

  const handleAction = async (checkId: string, action: string) => {
    setActionError(null);
    const sshConfig = getSSHConfig();
    if (!sshConfig.host || !sshConfig.key_path) {
      setActionError("Missing SSH configuration. Please configure VPS host and SSH key in the Manifest step.");
      return;
    }

    setActionLoading(`${checkId}:${action}`);

    try {
      if (action === "stop_ports") {
        setPortSelections(buildPortSelections(portBindings));
        setShowPortModal(true);
        return;
      } else if (action === "open_firewall") {
        const result = await openFirewallPorts({
          ...sshConfig,
          ports: [80, 443],
        });
        if (result.ok) {
          runPreflight();
        } else {
          setActionError(result.message || "Failed to update firewall rules.");
        }
        return;
      } else if (action === "show_disk") {
        const usage = await getDiskUsage(sshConfig);
        setDiskUsage(usage);
        setShowDiskModal(true);
      } else if (action === "stop_scenario" || action === "stop_all") {
        const vps = manifest?.target?.vps;
        const workdir = vps?.workdir || DEFAULT_VPS_WORKDIR;
        const scenarioId = action === "stop_scenario" ? manifest?.scenario?.id : undefined;
        const result = await stopScenarioProcesses({
          ...sshConfig,
          workdir,
          scenario_id: scenarioId,
        });
        if (result.ok) {
          // Re-run preflight to verify
          runPreflight();
        } else {
          // Server includes SSH details in the error message
          setActionError(result.message || "Failed to stop processes");
        }
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      setActionError(`Action failed: ${errorMessage}`);
    } finally {
      setActionLoading(null);
    }
  };

  const handlePortStop = async () => {
    const sshConfig = getSSHConfig();
    if (!sshConfig.host || !sshConfig.key_path) {
      setActionError("Missing SSH configuration. Please configure VPS host and SSH key in the Manifest step.");
      return;
    }

    const services = Object.entries(portSelections.services)
      .filter(([, selected]) => selected)
      .map(([service]) => service);
    const pids = Object.entries(portSelections.pids)
      .filter(([, selected]) => selected)
      .map(([pid]) => Number(pid));

    if (services.length === 0 && pids.length === 0) {
      setActionError("Select at least one service or PID to stop.");
      return;
    }

    setActionLoading("ports_80_443:stop_ports");
    try {
      const result = await stopPortServices({
        ...sshConfig,
        services,
        pids,
        prefer_service_stop: true,
      });
      if (result.ok) {
        setShowPortModal(false);
        runPreflight();
      } else {
        setActionError(result.message || "Failed to stop services on ports 80/443.");
      }
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      setActionError(`Action failed: ${errorMessage}`);
    } finally {
      setActionLoading(null);
    }
  };

  const togglePortService = (service: string) => {
    setPortSelections((prev) => ({
      ...prev,
      services: {
        ...prev.services,
        [service]: !prev.services[service],
      },
    }));
  };

  const togglePortPID = (pid: number) => {
    setPortSelections((prev) => ({
      ...prev,
      pids: {
        ...prev.pids,
        [pid]: !prev.pids[pid],
      },
    }));
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
  const checksToDisplay = buildChecksToDisplay(preflightChecks, isRunningPreflight);

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

      {/* Error from action buttons (stop processes, etc.) */}
      {actionError && (
        <Alert variant="error" title="Action Failed">
          {actionError}
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
      <PreflightChecksPanel
        checksToDisplay={checksToDisplay}
        actionLoading={actionLoading}
        onAction={handleAction}
      />

      {/* Override checkbox - shown when preflight fails but checks have run */}
      {preflightPassed === false && preflightChecks !== null && !isRunningPreflight && (
        <div className="mt-4 p-4 bg-slate-800/50 border border-amber-500/30 rounded-lg">
          <label className="flex items-start gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={preflightOverride}
              onChange={(e) => setPreflightOverride(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-slate-600 bg-slate-700 text-amber-500 focus:ring-amber-500"
            />
            <div>
              <span className="text-sm font-medium text-amber-400">
                Continue anyway despite failed checks
              </span>
              <p className="text-xs text-slate-400 mt-1">
                Some issues (like DNS not yet configured) can be resolved after deployment.
                Check this box if you understand the risks and want to proceed.
              </p>
            </div>
          </label>
        </div>
      )}

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

      {/* Port Stop Modal */}
      {showPortModal && (
        <PortStopModal
          bindings={portBindings}
          selections={portSelections}
          loading={actionLoading === "ports_80_443:stop_ports"}
          onToggleService={togglePortService}
          onTogglePID={togglePortPID}
          onConfirm={handlePortStop}
          onClose={() => setShowPortModal(false)}
        />
      )}
    </div>
  );
}
