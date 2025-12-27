import { Shield, Play, CheckCircle2, AlertTriangle, XCircle, Server, Globe, Key, Network, HardDrive, Cpu, Wifi, Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { Alert } from "../ui/alert";
import type { useDeployment } from "../../hooks/useDeployment";
import type { PreflightCheck, PreflightCheckStatus } from "../../lib/api";

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
  { id: "os_release", title: "Ubuntu version", description: "Confirm Ubuntu 24.04 is running" },
  { id: "ports_80_443", title: "Ports 80/443 availability", description: "Verify ports are free for Caddy" },
  { id: "outbound_network", title: "Outbound network", description: "Test outbound HTTPS access" },
  { id: "disk_free", title: "Disk free space", description: "Ensure sufficient disk space (≥5 GiB)" },
  { id: "ram_total", title: "RAM", description: "Ensure sufficient RAM (≥1 GiB)" },
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
      return null; // Will render empty circle
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

interface CheckItemProps {
  id: string;
  title: string;
  description: string;
  state: CheckState;
  details?: string;
  hint?: string;
}

function CheckItem({ id, title, description, state, details, hint }: CheckItemProps) {
  const Icon = CHECK_ICONS[id] || Server;
  const StatusIcon = getStatusIcon(state);
  const colors = getStatusColor(state);

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
        <div className="flex-shrink-0">
          {StatusIcon ? (
            <StatusIcon className={`h-5 w-5 ${colors.text} ${state === "running" ? "animate-spin" : ""}`} />
          ) : (
            <div className="w-5 h-5 rounded-full border-2 border-slate-600" />
          )}
        </div>
      </div>
      {hint && state !== "pass" && state !== "pending" && state !== "running" && (
        <div className={`mt-2 ml-12 text-xs ${colors.text} bg-slate-900/50 rounded px-2 py-1`}>
          <span className="font-medium">Hint:</span> {hint}
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

  const failCount = preflightChecks?.filter(c => c.status === "fail").length ?? 0;
  const warnCount = preflightChecks?.filter(c => c.status === "warn").length ?? 0;

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
            />
          ))}
        </ul>
      </div>
    </div>
  );
}
