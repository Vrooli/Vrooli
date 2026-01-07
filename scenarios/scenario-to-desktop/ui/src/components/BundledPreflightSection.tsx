import { useEffect, useMemo, useState } from "react";
import { Braces, Copy, Download, LayoutList } from "lucide-react";
import type { BundlePreflightCheck, BundlePreflightJobStatusResponse, BundlePreflightLogTail, BundlePreflightResponse, BundlePreflightSecret, BundlePreflightStep } from "../lib/api";
import { fetchBundlePreflightHealth } from "../lib/api";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Label } from "./ui/label";

type PreflightIssueGuidance = {
  title: string;
  meaning: string;
  remediation: string[];
};

const PREFLIGHT_ISSUE_GUIDANCE: Record<string, PreflightIssueGuidance> = {
  manifest_invalid: {
    title: "Bundle manifest is invalid",
    meaning: "The manifest structure does not pass schema or platform validation.",
    remediation: [
      "Re-export the bundle from deployment-manager for the current OS/arch.",
      "Confirm the manifest points to assets that exist inside the bundle root."
    ]
  },
  binary_not_defined: {
    title: "Binary missing in manifest",
    meaning: "The service has no binary entry for this platform in bundle.json.",
    remediation: [
      "Ensure the service is built for this OS/arch, then re-export the bundle.",
      "Check deployment-manager export settings include service binaries."
    ]
  },
  binary_missing: {
    title: "Binary file not found",
    meaning: "The manifest references a binary path that doesn't exist in the bundle root.",
    remediation: [
      "Rebuild the service binary and re-export the bundle.",
      "Verify the binary path in bundle.json matches the staged file path."
    ]
  },
  binary_is_directory: {
    title: "Binary path is a directory",
    meaning: "The manifest points at a directory, but an executable file is required.",
    remediation: [
      "Update the manifest to point to the executable file.",
      "Re-export the bundle after rebuilding the service."
    ]
  },
  asset_missing: {
    title: "Asset file not found",
    meaning: "The manifest references an asset path that doesn't exist in the bundle root.",
    remediation: [
      "Rebuild the UI/assets and re-export the bundle.",
      "Confirm the asset path in bundle.json matches the staged file path."
    ]
  },
  asset_checksum_mismatch: {
    title: "Asset checksum mismatch",
    meaning: "The file exists but its checksum differs from the manifest entry.",
    remediation: [
      "Re-export the bundle so checksums match the staged files.",
      "Verify no post-export steps modify the bundle files."
    ]
  },
  asset_size_suspicious: {
    title: "Asset size is suspiciously small",
    meaning: "The asset is much smaller than expected, which often indicates a failed build.",
    remediation: [
      "Rebuild the asset and re-export the bundle.",
      "Verify the build output is complete (not a stub file)."
    ]
  },
  asset_size_warning: {
    title: "Asset size warning",
    meaning: "The asset is larger than expected but within the slack budget.",
    remediation: [
      "Re-export the bundle to update size metadata if this is expected.",
      "Review build artifacts for unexpected growth."
    ]
  }
};

type PreflightStepState = "pending" | "testing" | "pass" | "fail" | "warning" | "skipped";

type PreflightStepStatus = {
  state: PreflightStepState;
  label: string;
};

type HealthPeekState = {
  open: boolean;
  status: "idle" | "loading" | "ready" | "error";
  body?: string;
  error?: string;
  statusCode?: number;
  url?: string;
  contentType?: string;
  truncated?: boolean;
};

const PREFLIGHT_STEP_STYLES: Record<PreflightStepState, string> = {
  pending: "border-slate-800/70 bg-slate-950/70 text-slate-200",
  testing: "border-blue-800/70 bg-blue-950/60 text-blue-200",
  pass: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200",
  fail: "border-red-800/70 bg-red-950/60 text-red-200",
  warning: "border-amber-800/70 bg-amber-950/60 text-amber-200",
  skipped: "border-slate-800/70 bg-slate-950/60 text-slate-300"
};

const PREFLIGHT_STEP_CIRCLE_STYLES: Record<PreflightStepState, string> = {
  pending: "border-slate-800/70 bg-slate-950 text-slate-200",
  testing: "border-blue-800/70 bg-blue-950/60 text-blue-200",
  pass: "border-emerald-800/70 bg-emerald-950/40 text-emerald-200",
  fail: "border-red-800/70 bg-red-950/60 text-red-200",
  warning: "border-amber-800/70 bg-amber-950/60 text-amber-200",
  skipped: "border-slate-800/70 bg-slate-950/60 text-slate-300"
};

const PREFLIGHT_CHECK_STYLES: Record<BundlePreflightCheck["status"], string> = {
  pass: "border-emerald-800/70 text-emerald-200",
  fail: "border-red-800/70 text-red-200",
  warning: "border-amber-800/70 text-amber-200",
  skipped: "border-slate-800/70 text-slate-300"
};

const PREFLIGHT_CHECK_LABELS: Record<BundlePreflightCheck["status"], string> = {
  pass: "PASS",
  fail: "FAIL",
  warning: "WARN",
  skipped: "SKIP"
};

interface PreflightStepHeaderProps {
  index: number;
  title: string;
  status: PreflightStepStatus;
  subtitle?: string;
}

function PreflightStepHeader({ index, title, status, subtitle }: PreflightStepHeaderProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-start gap-3">
        <div className={`flex h-7 w-7 items-center justify-center rounded-full border text-xs font-semibold ${PREFLIGHT_STEP_CIRCLE_STYLES[status.state]}`}>
          {index}
        </div>
        <div>
          <p className="text-sm font-semibold text-slate-100">{title}</p>
          {subtitle && <p className="text-[11px] text-slate-400">{subtitle}</p>}
        </div>
      </div>
      <span className={`rounded-full border px-2 py-0.5 text-[11px] uppercase tracking-wide ${PREFLIGHT_STEP_STYLES[status.state]}`}>
        {status.label}
      </span>
    </div>
  );
}

function PreflightCheckList({ checks }: { checks: BundlePreflightCheck[] }) {
  if (checks.length === 0) {
    return null;
  }

  return (
    <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-[11px] text-slate-200">
      <summary className="cursor-pointer text-xs font-semibold text-slate-100">
        Test cases ({checks.length})
      </summary>
      <ul className="mt-2 space-y-2">
        {checks.map((check) => {
          const listenURL = getListenURL(check.detail);
          return (
            <li key={check.id} className="rounded-md border border-slate-800/70 bg-slate-950/70 px-3 py-2 space-y-1">
            <div className="flex flex-wrap items-center gap-2">
              <span className={`rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-wide ${PREFLIGHT_CHECK_STYLES[check.status]}`}>
                {PREFLIGHT_CHECK_LABELS[check.status]}
              </span>
              <span className="text-slate-200">{check.name}</span>
            </div>
            {check.detail && (
              <p className="text-slate-400">
                <span>{check.detail}</span>
                {listenURL && (
                  <a
                    className="ml-2 inline-flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wide text-sky-300 hover:text-sky-200"
                    href={listenURL}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Open
                  </a>
                )}
              </p>
            )}
          </li>
          );
        })}
      </ul>
    </details>
  );
}
type CoverageCell = {
  label: string;
  active: boolean;
};

const COVERAGE_COLUMNS: CoverageCell[] = [
  { label: "Runtime", active: true },
  { label: "Validation", active: true },
  { label: "Secrets", active: true },
  { label: "Services", active: false },
  { label: "Electron UI", active: false }
];

const COVERAGE_ROWS = [
  {
    id: "preflight",
    label: "Preflight",
    note: "Runtime + services + logs",
    cells: COVERAGE_COLUMNS.map((cell) => ({
      ...cell,
      active: cell.label === "Electron UI" ? false : true
    }))
  },
  {
    id: "full",
    label: "Full app",
    note: "Runtime + services + UI",
    cells: COVERAGE_COLUMNS.map((cell) => ({ ...cell, active: true }))
  }
];

interface CoverageBadgeProps {
  label: string;
  active: boolean;
}

function CoverageBadge({ label, active }: CoverageBadgeProps) {
  return (
    <div className={`rounded-md border px-2 py-1 text-[10px] ${active ? "border-sky-800/70 bg-sky-950/40 text-sky-200" : "border-slate-800/70 bg-slate-950/60 text-slate-400"}`}>
      <div className="flex items-center justify-between gap-2">
        <span className="uppercase tracking-wide">{label}</span>
        <span className="text-[9px]">{active ? "On" : "Off"}</span>
      </div>
    </div>
  );
}

function formatDuration(ms: number) {
  if (!Number.isFinite(ms) || ms < 0) {
    return "n/a";
  }
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

function getListenURL(detail?: string) {
  if (!detail) {
    return null;
  }
  const match = detail.match(/listening on (\d+)/i);
  if (!match) {
    return null;
  }
  const port = Number.parseInt(match[1], 10);
  if (!Number.isFinite(port) || port <= 0) {
    return null;
  }
  return `http://localhost:${port}`;
}

function getServiceURL(
  serviceId: string,
  ports?: Record<string, Record<string, number>>,
  preferredPortName?: string
) {
  if (!ports) {
    return null;
  }
  const portMap = ports[serviceId];
  if (!portMap) {
    return null;
  }
  if (preferredPortName) {
    const port = portMap[preferredPortName];
    if (Number.isFinite(port) && port > 0) {
      return { url: `http://localhost:${port}`, port, portName: preferredPortName };
    }
    return null;
  }
  const preferred = ["ui", "api", "http"];
  let portName = preferred.find((name) => Number.isFinite(portMap[name]));
  if (!portName) {
    const names = Object.keys(portMap);
    portName = names.length > 0 ? names[0] : undefined;
  }
  if (!portName) {
    return null;
  }
  const port = portMap[portName];
  if (!Number.isFinite(port) || port <= 0) {
    return null;
  }
  return { url: `http://localhost:${port}`, port, portName };
}

type ManifestHealthConfig = {
  type?: string;
  path?: string;
  portName?: string;
};

function getManifestHealthConfig(manifest: unknown, serviceId: string): ManifestHealthConfig | null {
  if (!manifest || typeof manifest !== "object") {
    return null;
  }
  const services = (manifest as { services?: unknown }).services;
  if (!Array.isArray(services)) {
    return null;
  }
  const service = services.find((entry) => {
    if (!entry || typeof entry !== "object") {
      return false;
    }
    const id = (entry as { id?: unknown }).id;
    return typeof id === "string" && id === serviceId;
  }) as { health?: { type?: unknown; path?: unknown; port_name?: unknown } } | undefined;
  if (!service || !service.health || typeof service.health !== "object") {
    return null;
  }
  const type = typeof service.health.type === "string" ? service.health.type : undefined;
  const path = typeof service.health.path === "string" ? service.health.path : undefined;
  const portName = typeof service.health.port_name === "string" ? service.health.port_name : undefined;
  return { type, path, portName };
}

function normalizeHealthPath(path?: string) {
  const trimmed = path?.trim();
  if (!trimmed) {
    return null;
  }
  return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}

function parseTimestamp(value?: string) {
  if (!value) {
    return null;
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime()) || date.getFullYear() < 2000) {
    return null;
  }
  return date.getTime();
}

function formatTimestamp(value?: string) {
  const ts = parseTimestamp(value);
  if (!ts) {
    return "";
  }
  return new Date(ts).toLocaleTimeString();
}

function countLines(text?: string) {
  if (!text) {
    return 0;
  }
  const lines = text.split(/\r?\n/);
  if (lines.length === 0) {
    return 0;
  }
  const last = lines[lines.length - 1];
  return last === "" ? lines.length - 1 : lines.length;
}

interface BundledPreflightSectionProps {
  bundleManifestPath: string;
  bundleManifest?: unknown;
  preflightResult: BundlePreflightResponse | null;
  preflightPending: boolean;
  preflightError: string | null;
  preflightJobStatus?: BundlePreflightJobStatusResponse | null;
  missingSecrets: BundlePreflightSecret[];
  secretInputs: Record<string, string>;
  preflightOk: boolean;
  preflightOverride: boolean;
  preflightSessionTTL: number;
  preflightSessionId?: string | null;
  preflightSessionExpiresAt?: string | null;
  preflightLogTails?: BundlePreflightLogTail[];
  onOverrideChange: (value: boolean) => void;
  onSessionTTLChange: (value: number) => void;
  onSecretChange: (id: string, value: string) => void;
  onRun: (secretsOverride?: Record<string, string>) => void;
}

export function BundledPreflightSection({
  bundleManifestPath,
  bundleManifest,
  preflightResult,
  preflightPending,
  preflightError,
  preflightJobStatus,
  missingSecrets,
  secretInputs,
  preflightOk,
  preflightOverride,
  preflightSessionTTL,
  preflightSessionId,
  preflightSessionExpiresAt,
  preflightLogTails,
  onOverrideChange,
  onSessionTTLChange,
  onSecretChange,
  onRun
}: BundledPreflightSectionProps) {
  const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">("idle");
  const [logCopyStatus, setLogCopyStatus] = useState<Record<string, "idle" | "copied" | "error">>({});
  const [healthPeek, setHealthPeek] = useState<Record<string, HealthPeekState>>({});
  const [tick, setTick] = useState(() => Date.now());
  const jobSteps = preflightJobStatus?.steps ?? [];
  const jobStepById = useMemo(() => {
    const map = new Map<string, BundlePreflightStep>();
    jobSteps.forEach((step) => {
      map.set(step.id, step);
    });
    return map;
  }, [jobSteps]);
  const validation = preflightResult?.validation;
  const readiness = preflightResult?.ready;
  const ports = preflightResult?.ports;
  const telemetry = preflightResult?.telemetry;
  const logTails = preflightLogTails ?? preflightResult?.log_tails;
  const checks = preflightResult?.checks ?? [];
  const bundleRootPreview = bundleManifestPath.trim()
    ? bundleManifestPath.trim().replace(/[/\\][^/\\]+$/, "")
    : "";
  const readinessDetails = readiness?.details ? Object.entries(readiness.details) : [];
  const latestServiceUpdate = readinessDetails.reduce((latest, [, status]) => {
    const ts = parseTimestamp(status.updated_at);
    if (!ts) {
      return latest;
    }
    return Math.max(latest, ts);
  }, 0);
  const snapshotTs = parseTimestamp(readiness?.snapshot_at) || latestServiceUpdate || 0;
  const snapshotLabel = snapshotTs ? new Date(snapshotTs).toLocaleTimeString() : "";
  const snapshotAge = snapshotTs ? formatDuration(Math.max(0, tick - snapshotTs)) : "";
  const hasMissingArtifacts = Boolean(
    validation
      && !validation.valid
      && ((validation.missing_assets?.length ?? 0) > 0 || (validation.missing_binaries?.length ?? 0) > 0)
  );
  const likelyRootMismatch = Boolean(
    hasMissingArtifacts
      && bundleManifestPath.trim()
      && !bundleManifestPath.includes("scenario-to-desktop/data/staging")
  );
  const preflightPayload = useMemo(
    () => ({
      bundle_manifest_path: bundleManifestPath,
      start_services: true,
      result: preflightResult,
      error: preflightError || undefined,
      missing_secrets: missingSecrets
    }),
    [bundleManifestPath, preflightResult, preflightError, missingSecrets]
  );
  const hasRun = Boolean(preflightResult || preflightError || preflightJobStatus);

  useEffect(() => {
    if (!preflightResult) {
      return;
    }
    const interval = window.setInterval(() => setTick(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, [preflightResult]);

  const toggleHealthPeek = async (serviceId: string, healthSupported: boolean) => {
    if (!healthSupported) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { open: true, status: "error", error: "Health proxy only supports http checks with a configured health path." }
      }));
      return;
    }
    if (!preflightSessionId) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { open: true, status: "error", error: "Preflight session missing. Re-run preflight with a session TTL to fetch health responses." }
      }));
      return;
    }
    const current = healthPeek[serviceId];
    if (current?.open) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { ...current, open: false }
      }));
      return;
    }
    if (current?.status === "ready" && current.body) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { ...current, open: true }
      }));
      return;
    }
    setHealthPeek((prev) => ({
      ...prev,
      [serviceId]: { open: true, status: "loading" }
    }));
    try {
      const response = await fetchBundlePreflightHealth({
        session_id: preflightSessionId,
        service_id: serviceId
      });
      const raw = response.body ?? "";
      let body = raw.trim();
      try {
        const json = JSON.parse(raw);
        body = JSON.stringify(json, null, 2);
      } catch {
        // Leave body as raw text when not JSON.
      }
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: {
          open: true,
          status: "ready",
          body,
          statusCode: response.status_code,
          url: response.url,
          contentType: response.content_type,
          truncated: response.truncated
        }
      }));
    } catch (err) {
      setHealthPeek((prev) => ({
        ...prev,
        [serviceId]: { open: true, status: "error", error: err instanceof Error ? err.message : String(err) }
      }));
    }
  };
  const portSummary = ports
    ? Object.entries(ports)
        .map(([svc, portMap]) => {
          const pairs = Object.entries(portMap)
            .map(([name, port]) => `${name}:${port}`)
            .join(", ");
          return `${svc}(${pairs})`;
        })
        .join(" · ")
    : "";
  const diagnosticsAvailable = Boolean(
    portSummary || telemetry?.path || (logTails && logTails.length > 0)
  );
  const validationChecks = checks.filter((check) => check.step === "validation");
  const secretChecks = checks.filter((check) => check.step === "secrets");
  const runtimeChecks = checks.filter((check) => check.step === "runtime");
  const serviceChecks = checks.filter((check) => check.step === "services");
  const diagnosticsChecks = checks.filter((check) => check.step === "diagnostics");

  const resolveJobStepStatus = (stepId: string): PreflightStepStatus | null => {
    const step = jobStepById.get(stepId);
    if (!step) {
      return null;
    }
    const state = step.state === "running" ? "testing" : step.state;
    const labelMap: Record<BundlePreflightStep["state"], string> = {
      pending: "Pending",
      running: "Running",
      pass: "Pass",
      fail: "Fail",
      warning: "Warning",
      skipped: "Skipped"
    };
    return {
      state: state as PreflightStepState,
      label: labelMap[step.state]
    };
  };
  const resolveJobStepDetail = (stepId: string) => jobStepById.get(stepId)?.detail;

  const stepValidationStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Testing" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (validation?.valid === true) {
      return { state: "pass", label: "Pass" };
    }
    if (validation?.valid === false) {
      return { state: "fail", label: "Fail" };
    }
    return { state: "warning", label: "Review" };
  })();
  const resolvedValidationStatus = resolveJobStepStatus("validation") ?? stepValidationStatus;

  const stepSecretsStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Checking" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (missingSecrets.length > 0) {
      return { state: "warning", label: "Missing" };
    }
    return { state: "pass", label: "Ready" };
  })();
  const resolvedSecretsStatus = resolveJobStepStatus("secrets") ?? stepSecretsStatus;

  const stepRuntimeStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Starting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (preflightResult) {
      return { state: "pass", label: "Running" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    return { state: "warning", label: "Unknown" };
  })();
  const resolvedRuntimeStatus = resolveJobStepStatus("runtime") ?? stepRuntimeStatus;

  const stepServicesStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Starting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (readiness?.ready === true) {
      return { state: "pass", label: "Ready" };
    }
    if (readiness?.ready === false) {
      return { state: "warning", label: "Waiting (snapshot)" };
    }
    return { state: "warning", label: "Unknown" };
  })();
  const resolvedServicesStatus = resolveJobStepStatus("services") ?? stepServicesStatus;

  const stepDiagnosticsStatus: PreflightStepStatus = (() => {
    if (preflightPending) {
      return { state: "testing", label: "Collecting" };
    }
    if (preflightError) {
      return { state: "fail", label: "Failed" };
    }
    if (!hasRun) {
      return { state: "pending", label: "Pending" };
    }
    if (diagnosticsAvailable) {
      return { state: "pass", label: "Available" };
    }
    return { state: "warning", label: "Empty" };
  })();
  const resolvedDiagnosticsStatus = resolveJobStepStatus("diagnostics") ?? stepDiagnosticsStatus;

  const copyJson = async () => {
    if (!preflightResult && !preflightError) {
      return;
    }
    try {
      await navigator.clipboard.writeText(JSON.stringify(preflightPayload, null, 2));
      setCopyStatus("copied");
      setTimeout(() => setCopyStatus("idle"), 1500);
    } catch (error) {
      console.warn("Failed to copy preflight JSON", error);
      setCopyStatus("error");
      setTimeout(() => setCopyStatus("idle"), 2000);
    }
  };

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4 space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-100">Preflight validation</p>
          <p className="text-xs text-slate-400">
            Validates the bundle manifest + staged assets by running the bundled runtime (no desktop wrapper needed).
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex items-center gap-1 rounded-full border border-slate-800/70 bg-slate-950/60 p-1 text-[11px]">
            <Button
              type="button"
              size="sm"
              variant={viewMode === "summary" ? "default" : "ghost"}
              onClick={() => setViewMode("summary")}
              aria-label="Show preflight summary"
              className="h-10 w-10 p-0"
            >
              <LayoutList className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              size="sm"
              variant={viewMode === "json" ? "default" : "ghost"}
              onClick={() => setViewMode("json")}
              aria-label="Show preflight JSON"
              className="h-10 w-10 p-0"
            >
              <Braces className="h-4 w-4" />
            </Button>
          </div>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => onRun()}
            disabled={preflightPending || !bundleManifestPath.trim()}
            className="gap-2"
          >
            {preflightPending ? "Running..." : "Run preflight"}
          </Button>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-3 text-xs text-slate-400">
        <div className="flex items-center gap-2">
          <Label htmlFor="preflight-session-ttl" className="text-[11px] text-slate-400">
            Session TTL (s)
          </Label>
          <Input
            id="preflight-session-ttl"
            type="number"
            min={30}
            max={900}
            value={preflightSessionTTL}
            onChange={(e) => {
              const next = Number(e.target.value);
              if (Number.isNaN(next)) {
                return;
              }
              onSessionTTLChange(Math.min(900, Math.max(30, next)));
            }}
            className="h-7 w-20 text-xs"
          />
        </div>
      </div>

      {!bundleManifestPath.trim() && (
        <p className="text-xs text-amber-200">Add a bundle_manifest_path to enable preflight checks.</p>
      )}

      {preflightError && (
        <div className="rounded-md border border-red-800 bg-red-950/40 p-3 text-sm text-red-200">
          {preflightError}
        </div>
      )}

      {viewMode === "summary" && (
        <div className="space-y-4">
          <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-200 space-y-2">
            <p className="font-semibold text-slate-100">Bundle context</p>
            <div className="space-y-1 text-[11px] text-slate-300">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-slate-400">Manifest</span>
                <span className="text-slate-200">{bundleManifestPath.trim() || "Not set"}</span>
              </div>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-slate-400">Bundle root</span>
                <span className="text-slate-200">{bundleRootPreview || "Unknown"}</span>
              </div>
            </div>
            {likelyRootMismatch && (
              <p className="text-[11px] text-amber-200">
                Missing artifacts detected. If your bundle assets live elsewhere, re-export the bundle so the manifest
                and staged files sit in the same directory.
              </p>
            )}
          </div>

          <div className="space-y-3">
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={1}
                title="Load bundle + validate"
                subtitle="Manifest structure and staged binaries/assets"
                status={resolvedValidationStatus}
              />
              <p className="text-[11px] text-slate-400">
                Confirms the manifest is valid and staged files exist with matching checksums.
              </p>
              {resolveJobStepDetail("validation") && (
                <p className="text-[11px] text-slate-300">
                  {resolveJobStepDetail("validation")}
                </p>
              )}
              {validation?.valid && !preflightError && (
                <p className="text-[11px] text-slate-300">No validation issues detected.</p>
              )}
              {preflightError && !validation && (
                <p className="text-[11px] text-red-200">
                  Validation did not complete. Review the error above and re-run preflight.
                </p>
              )}
              {validation && !validation.valid && (
                <details className="rounded-md border border-red-900/50 bg-red-950/20 p-3 text-[11px] text-red-200" open>
                  <summary className="cursor-pointer text-xs font-semibold text-red-100">Validation issues</summary>
                  <div className="mt-2 space-y-3">
                    {validation.errors && validation.errors.length > 0 && (
                      <div className="space-y-2">
                        {validation.errors.map((err, idx) => {
                          const guidance = PREFLIGHT_ISSUE_GUIDANCE[err.code];
                          return (
                            <div key={`${err.code}-${idx}`} className="rounded-md border border-red-900/50 bg-red-950/40 p-2 space-y-1">
                              <p className="font-semibold text-red-100">
                                {guidance?.title || err.code}
                              </p>
                              <p>{err.message}</p>
                              {(err.service || err.path) && (
                                <p className="text-[11px] text-red-200/80">
                                  {err.service ? `Service: ${err.service}` : ""}{err.service && err.path ? " · " : ""}{err.path ? `Path: ${err.path}` : ""}
                                </p>
                              )}
                              {guidance && (
                                <>
                                  <p className="text-[11px] text-red-200/80">{guidance.meaning}</p>
                                  <ul className="space-y-1 text-[11px] text-red-100">
                                    {guidance.remediation.map((step, stepIdx) => (
                                      <li key={`${err.code}-remedy-${stepIdx}`}>• {step}</li>
                                    ))}
                                  </ul>
                                </>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                    {validation.missing_binaries && validation.missing_binaries.length > 0 && (
                      <div className="space-y-1 text-[11px] text-red-100">
                        <p className="font-semibold text-red-100">Missing binaries</p>
                        <ul className="space-y-1">
                          {validation.missing_binaries.map((item, idx) => (
                            <li key={`${item.service_id}-${item.path}-${idx}`}>
                              {item.service_id}: {item.path} ({item.platform})
                            </li>
                          ))}
                        </ul>
                        <p className="text-red-200/80">
                          Rebuild the service binaries and re-export the bundle to update manifest paths.
                        </p>
                      </div>
                    )}
                    {validation.missing_assets && validation.missing_assets.length > 0 && (
                      <div className="space-y-1 text-[11px] text-red-100">
                        <p className="font-semibold text-red-100">Missing assets</p>
                        <ul className="space-y-1">
                          {validation.missing_assets.map((item, idx) => (
                            <li key={`${item.service_id}-${item.path}-${idx}`}>
                              {item.service_id}: {item.path}
                            </li>
                          ))}
                        </ul>
                        <p className="text-red-200/80">
                          Rebuild UI/assets and re-export the bundle so the staged paths exist.
                        </p>
                      </div>
                    )}
                    {validation.invalid_checksums && validation.invalid_checksums.length > 0 && (
                      <div className="space-y-1 text-[11px] text-red-100">
                        <p className="font-semibold text-red-100">Invalid checksums</p>
                        <ul className="space-y-1">
                          {validation.invalid_checksums.map((item, idx) => (
                            <li key={`${item.service_id}-${item.path}-${idx}`}>
                              {item.service_id}: {item.path}
                            </li>
                          ))}
                        </ul>
                        <p className="text-red-200/80">
                          Re-export the bundle after rebuilding assets so checksums match.
                        </p>
                      </div>
                    )}
                  </div>
                </details>
              )}
              {validation?.warnings && validation.warnings.length > 0 && (
                <details className="rounded-md border border-amber-800/50 bg-amber-950/20 p-3 text-[11px] text-amber-200">
                  <summary className="cursor-pointer text-xs font-semibold">Warnings</summary>
                  <ul className="mt-2 space-y-1">
                    {validation.warnings.map((warn, idx) => (
                      <li key={`${warn.code}-${idx}`}>{warn.message}</li>
                    ))}
                  </ul>
                </details>
              )}
              <PreflightCheckList checks={validationChecks} />
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={2}
                title="Apply secrets"
                subtitle="Required secrets must be present for readiness"
                status={resolvedSecretsStatus}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to detect required secrets.
                </p>
              )}
              {resolveJobStepDetail("secrets") && (
                <p className="text-[11px] text-slate-300">
                  {resolveJobStepDetail("secrets")}
                </p>
              )}
              {preflightError && (
                <p className="text-[11px] text-red-200">
                  Secrets were not checked because preflight failed.
                </p>
              )}
              {hasRun && !preflightError && missingSecrets.length === 0 && (
                <p className="text-[11px] text-slate-300">All required secrets are present for this run.</p>
              )}
              {missingSecrets.length > 0 && (
                <div className="rounded-md border border-amber-800/60 bg-amber-950/20 p-3 text-xs text-amber-100 space-y-3">
                  <p className="font-semibold text-amber-100">Missing required secrets</p>
                  <p className="text-amber-100/80">
                    Enter temporary values to validate readiness. Values are used only for this preflight run.
                  </p>
                  <div className="space-y-2">
                    {missingSecrets.map((secret) => (
                      <div key={secret.id} className="space-y-1">
                        <Label htmlFor={`preflight-${secret.id}`}>
                          {secret.prompt?.label || secret.id}
                        </Label>
                        <Input
                          id={`preflight-${secret.id}`}
                          type="password"
                          value={secretInputs[secret.id] || ""}
                          onChange={(e) => onSecretChange(secret.id, e.target.value)}
                          placeholder={secret.prompt?.hint || "Enter value"}
                        />
                        {secret.class && (
                          <p className="text-[11px] text-amber-200/80">
                            {secret.class.replace(/_/g, " ")}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => onRun(secretInputs)}
                    disabled={preflightPending}
                  >
                    Apply secrets and re-run
                  </Button>
                </div>
              )}
              <PreflightCheckList checks={secretChecks} />
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={3}
                title="Start runtime control API"
                subtitle="IPC auth token + control API readiness"
                status={resolvedRuntimeStatus}
              />
              {preflightPending && (
                <p className="text-[11px] text-slate-400">
                  Starting the runtime supervisor and waiting for the control API.
                </p>
              )}
              {resolveJobStepDetail("runtime") && (
                <p className="text-[11px] text-slate-300">
                  {resolveJobStepDetail("runtime")}
                </p>
              )}
              {!preflightPending && preflightResult && (
                <p className="text-[11px] text-slate-300">
                  Control API is responding. Runtime supervisor initialized.
                </p>
              )}
              {!preflightPending && !preflightResult && !preflightError && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to boot the runtime supervisor.
                </p>
              )}
              {preflightError && (
                <p className="text-[11px] text-red-200">
                  Control API failed to start. Review the error above.
                </p>
              )}
              <PreflightCheckList checks={runtimeChecks} />
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={4}
                title="Services ready"
                subtitle="Optional service startup + readiness checks"
                status={resolvedServicesStatus}
              />
              {readiness && readinessDetails.length > 0 && (
                <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200" open={!readiness.ready}>
                  <summary className="cursor-pointer text-xs font-semibold text-slate-100">
                    Readiness details
                  </summary>
                  <p className="mt-1 text-[11px] text-slate-400">
                    Snapshot {snapshotLabel ? `at ${snapshotLabel}` : "captured during this preflight run"}
                    {snapshotAge ? ` (${snapshotAge} ago)` : ""}.
                    {typeof readiness?.waited_seconds === "number" && readiness.waited_seconds > 0
                      ? ` Waited ${readiness.waited_seconds}s before capturing status.`
                      : ""}
                    Auto-refresh updates this snapshot every 10s while the session is active.
                  </p>
                  {(preflightSessionId || preflightSessionExpiresAt) && (
                    <p className="mt-1 text-[11px] text-slate-500">
                      Session {preflightSessionId ? preflightSessionId.slice(0, 8) : "active"}
                      {preflightSessionExpiresAt ? ` · expires ${formatTimestamp(preflightSessionExpiresAt)}` : ""}
                    </p>
                  )}
                  <div className="mt-2 grid gap-2 sm:grid-cols-2">
                    {readinessDetails.map(([serviceId, status]) => {
                      const referenceTs = snapshotTs || tick;
                      const updatedAt = parseTimestamp(status.updated_at);
                      const startedAt = parseTimestamp(status.started_at);
                      const readyAt = parseTimestamp(status.ready_at);
                      const isSkipped = Boolean(status.skipped);
                      const isUpdatedAhead = updatedAt ? updatedAt > tick + 5000 : false;
                      const statusAge = updatedAt && !isUpdatedAhead ? formatDuration(Math.max(0, referenceTs - updatedAt)) : "";
                      const startedAge = startedAt ? formatDuration(Math.max(0, referenceTs - startedAt)) : "";
                      const readyAge = readyAt ? formatDuration(Math.max(0, referenceTs - readyAt)) : "";
                      const statusLabel = isSkipped ? "Skipped" : status.ready ? "Ready" : "Waiting";
                      const statusClass = isSkipped ? "text-slate-300" : status.ready ? "text-emerald-200" : "text-amber-200";
                      const healthConfig = getManifestHealthConfig(bundleManifest, serviceId);
                      const healthType = healthConfig?.type?.toLowerCase();
                      const healthPath = normalizeHealthPath(healthConfig?.path);
                      const healthPortName = healthConfig?.portName;
                      const portInfo = getServiceURL(serviceId, ports);
                      const healthPortInfo = healthPortName
                        ? getServiceURL(serviceId, ports, healthPortName)
                        : null;
                      const healthSupported = healthType === "http" && Boolean(healthPath) && Boolean(healthPortInfo);
                      const listenURL = portInfo?.url ?? getListenURL(status.message);
                      const healthURL = healthPortInfo?.url && healthPath
                        ? `${healthPortInfo.url}${healthPath}`
                        : null;
                      const peekState = healthPeek[serviceId];
                      return (
                        <div key={serviceId} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-1">
                          <p className="text-[11px] uppercase tracking-wide text-slate-400">{serviceId}</p>
                          <p className={`text-xs ${statusClass}`}>
                            {statusLabel}
                          </p>
                          {status.message && (
                            <p className="text-[11px] text-slate-300">
                              <span>{status.message}</span>
                            </p>
                          )}
                          {listenURL && (
                            <div className="flex flex-wrap items-center gap-2 text-[10px]">
                              <a
                                className="inline-flex items-center gap-1 rounded-sm border border-sky-900/60 bg-sky-950/40 px-2 py-0.5 font-semibold uppercase tracking-wide text-sky-200 hover:text-sky-100"
                                href={listenURL}
                                target="_blank"
                                rel="noreferrer"
                              >
                                Open
                              </a>
                              {healthURL && (
                                <a
                                  className="inline-flex items-center gap-1 rounded-sm border border-slate-800/60 bg-slate-950/60 px-2 py-0.5 font-semibold uppercase tracking-wide text-slate-200 hover:text-slate-100"
                                  href={healthURL}
                                  target="_blank"
                                  rel="noreferrer"
                                >
                                  Health
                                </a>
                              )}
                              {healthSupported && (
                                <button
                                  type="button"
                                  className="inline-flex items-center gap-1 rounded-sm border border-slate-800/60 bg-slate-950/60 px-2 py-0.5 font-semibold uppercase tracking-wide text-slate-200 hover:text-slate-100"
                                  onClick={() => toggleHealthPeek(serviceId, healthSupported)}
                                >
                                  {peekState?.open ? "Hide" : "View"} response
                                </button>
                              )}
                              <span className="text-slate-400">{listenURL}</span>
                              {portInfo && (
                                <span className="text-slate-400">
                                  {portInfo.portName}:{portInfo.port}
                                </span>
                              )}
                            </div>
                          )}
                          {!healthSupported && (
                            <p className="text-[10px] text-slate-500">
                              Health check not configured for http proxy inspection.
                            </p>
                          )}
                          {peekState?.open && (
                            <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-2 text-[10px] text-slate-200">
                              {peekState.status === "loading" && (
                                <p className="text-slate-400">Loading health response...</p>
                              )}
                              {peekState.status === "error" && (
                                <p className="text-red-200">
                                  Failed to fetch health response via the runtime proxy. Open the Health link to inspect directly.
                                  {peekState.error ? ` (${peekState.error})` : ""}
                                </p>
                              )}
                              {peekState.status === "ready" && (
                                <div className="space-y-1">
                                  <p className="text-[10px] text-slate-400">
                                    {peekState.statusCode ? `HTTP ${peekState.statusCode}` : "HTTP response"}
                                    {peekState.url ? ` · ${peekState.url}` : ""}
                                    {peekState.truncated ? " · truncated" : ""}
                                  </p>
                                  {peekState.body && (
                                    <pre className="whitespace-pre-wrap break-words">{peekState.body}</pre>
                                  )}
                                </div>
                              )}
                            </div>
                          )}
                          {status.message === "pending start" && !isSkipped && (
                            <p className="text-[11px] text-slate-500">
                              Not launched yet; waiting on dependencies or secrets.
                            </p>
                          )}
                          {status.ready && !isSkipped && readyAge && (
                            <p className="text-[11px] text-emerald-200/80">Ready for {readyAge}</p>
                          )}
                          {!status.ready && !isSkipped && startedAge && (
                            <p className="text-[11px] text-amber-200/80">Starting for {startedAge}</p>
                          )}
                          {!status.ready && !isSkipped && !startedAge && statusAge && (
                            <p className="text-[11px] text-slate-400">Status set {statusAge} ago</p>
                          )}
                          {status.updated_at && isUpdatedAhead && (
                            <p className="text-[11px] text-amber-200/80">
                              Last update {formatTimestamp(status.updated_at)} (ahead of local clock)
                            </p>
                          )}
                          {status.updated_at && !isUpdatedAhead && formatTimestamp(status.updated_at) && (
                            <p className="text-[11px] text-slate-500">
                              Last update {formatTimestamp(status.updated_at)}
                            </p>
                          )}
                          {typeof status.exit_code === "number" && (
                            <p className="text-[11px] text-slate-400">Exit code: {status.exit_code}</p>
                          )}
                        </div>
                      );
                    })}
                  </div>
                  {!readiness.ready && (
                    <p className="mt-2 text-[11px] text-slate-300">
                      Readiness is a snapshot from preflight. Services shut down after this run, so re-run to refresh status or inspect log tails for why a service is waiting.
                    </p>
                  )}
                </details>
              )}
              {(!readiness || readinessDetails.length === 0) && (
                <p className="text-[11px] text-slate-400">
                  No service readiness details yet. Re-run preflight or verify the bundle defines services for this target.
                </p>
              )}
              {resolveJobStepDetail("services") && (
                <p className="text-[11px] text-slate-300">
                  {resolveJobStepDetail("services")}
                </p>
              )}
              <PreflightCheckList checks={serviceChecks} />
            </div>

            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={5}
                title="Diagnostics"
                subtitle="Ports, telemetry, and optional log tails"
                status={resolvedDiagnosticsStatus}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">
                  Run preflight to collect diagnostics.
                </p>
              )}
              {resolveJobStepDetail("diagnostics") && (
                <p className="text-[11px] text-slate-300">
                  {resolveJobStepDetail("diagnostics")}
                </p>
              )}
              {hasRun && !diagnosticsAvailable && (
                <p className="text-[11px] text-slate-400">
                  No diagnostics reported yet.
                </p>
              )}
              {(portSummary || telemetry?.path) && (
                <div className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200 space-y-2">
                  {portSummary && <p>Ports: {portSummary}</p>}
                  {telemetry?.path && <p>Telemetry: {telemetry.path}</p>}
                </div>
              )}
              {logTails && logTails.length > 0 && (
                <details className="rounded-md border border-slate-800/70 bg-slate-950/70 p-3 text-xs text-slate-200">
                  <summary className="cursor-pointer text-xs font-semibold text-slate-100">
                    Service log tails
                  </summary>
                  <div className="mt-2 space-y-2">
                    {logTails.map((tail, idx) => {
                      const copyKey = `${tail.service_id}-${idx}`;
                      const lineCount = countLines(tail.content);
                      const copyState = logCopyStatus[copyKey] || "idle";
                      return (
                        <div key={copyKey} className="rounded-md border border-slate-800/70 bg-slate-950/80 p-2 space-y-2">
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <p className="text-[11px] uppercase tracking-wide text-slate-400">
                              {tail.service_id} · {lineCount} of {tail.lines} lines
                            </p>
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              className="h-7 px-2 text-[10px]"
                              onClick={async () => {
                                try {
                                  await navigator.clipboard.writeText(tail.content || "");
                                  setLogCopyStatus((prev) => ({ ...prev, [copyKey]: "copied" }));
                                  window.setTimeout(() => {
                                    setLogCopyStatus((prev) => ({ ...prev, [copyKey]: "idle" }));
                                  }, 1500);
                                } catch (error) {
                                  console.warn("Failed to copy log tail", error);
                                  setLogCopyStatus((prev) => ({ ...prev, [copyKey]: "error" }));
                                }
                              }}
                            >
                              <Copy className="h-3 w-3" />
                              <span className="ml-1">
                                {copyState === "copied" ? "Copied" : copyState === "error" ? "Failed" : "Copy"}
                              </span>
                            </Button>
                          </div>
                          {tail.error ? (
                            <p className="text-amber-200">{tail.error}</p>
                          ) : (
                            <pre className="max-h-40 overflow-auto whitespace-pre-wrap text-[11px] text-slate-200">
                              {tail.content || "No log output yet."}
                            </pre>
                          )}
                        </div>
                      );
                    })}
                  </div>
                </details>
              )}
              <PreflightCheckList checks={diagnosticsChecks} />
            </div>
          </div>

          <details className="rounded-md border border-slate-800 bg-slate-950/40 p-3 text-xs text-slate-200">
            <summary className="cursor-pointer text-xs font-semibold text-slate-100">Coverage map</summary>
            <p className="mt-2 text-[11px] text-slate-400">
              Preflight starts the runtime supervisor, validates bundle files, applies secrets, starts services, and
              captures readiness plus log tails. The Electron UI is not started during preflight.
            </p>
            <div className="mt-3 space-y-2">
              {COVERAGE_ROWS.map((row) => {
                const isActive = row.id === "preflight";
                return (
                  <div
                    key={row.id}
                    className={`rounded-md border px-3 py-2 ${isActive ? "border-slate-600/70 bg-slate-900/60" : "border-slate-800/70 bg-slate-950/60"}`}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2 text-[11px] text-slate-400">
                      <span className="text-slate-200">{row.label}</span>
                      <span>{row.note}</span>
                    </div>
                    <div className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
                      {row.cells.map((cell) => (
                        <CoverageBadge key={`${row.id}-${cell.label}`} label={cell.label} active={cell.active} />
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          </details>

          {!preflightOk && (
            <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-amber-800 bg-amber-950/20 p-3 text-xs text-amber-100">
              <span>Override preflight and allow generation anyway.</span>
              <Checkbox
                checked={preflightOverride}
                onChange={(e) => onOverrideChange(e.target.checked)}
                label="Override"
              />
            </div>
          )}
        </div>
      )}

      {viewMode === "json" && (
        <div className="rounded-md border border-slate-800 bg-slate-950/60 p-3 space-y-2">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-xs font-semibold text-slate-200">Preflight JSON</p>
            <div className="flex items-center gap-2">
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={copyJson}
                disabled={!preflightResult && !preflightError}
                aria-label="Copy preflight JSON"
                className="h-10 w-10 p-0"
              >
                <Copy className="h-4 w-4" />
              </Button>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  const blob = new Blob([JSON.stringify(preflightPayload, null, 2)], { type: "application/json" });
                  const url = URL.createObjectURL(blob);
                  const link = document.createElement("a");
                  link.href = url;
                  link.download = "preflight.json";
                  link.click();
                  URL.revokeObjectURL(url);
                }}
                disabled={!preflightResult && !preflightError}
                aria-label="Download preflight JSON"
                className="h-10 w-10 p-0"
              >
                <Download className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <pre className="max-h-96 overflow-auto whitespace-pre-wrap rounded-md border border-slate-800/70 bg-slate-950/80 p-3 text-[11px] text-slate-200">
            {JSON.stringify(preflightPayload, null, 2)}
          </pre>
          {copyStatus !== "idle" && (
            <p className="text-[11px] text-slate-400">
              {copyStatus === "copied" ? "Copied to clipboard." : "Copy failed."}
            </p>
          )}
          <p className="text-[11px] text-slate-400">
            Use this view to share the full preflight snapshot with an agent or teammate.
          </p>
        </div>
      )}
    </div>
  );
}
