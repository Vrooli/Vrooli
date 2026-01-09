/**
 * Bundled preflight validation section.
 * Validates bundle manifests and staged assets by running the bundled runtime.
 *
 * This component orchestrates the preflight validation workflow:
 * 1. Load bundle + validate manifest structure
 * 2. Apply secrets required by services
 * 3. Start runtime control API
 * 4. Check service readiness
 * 5. Collect diagnostics (ports, logs, fingerprints)
 */

import { useEffect, useMemo, useState } from "react";
import { Braces, Copy, Download, LayoutList } from "lucide-react";
import type {
  BundlePreflightJobStatusResponse,
  BundlePreflightLogTail,
  BundlePreflightResponse,
  BundlePreflightSecret,
  BundlePreflightStep
} from "../lib/api";
import {
  JOB_STEP_STATE_LABELS,
  type PreflightStepState,
  type PreflightStepStatus
} from "../lib/preflight-constants";
import {
  detectLikelyRootMismatch,
  formatDuration,
  formatPortSummary,
  formatTimestamp,
  getBundleRootFromManifestPath,
  parseTimestamp
} from "../lib/preflight-utils";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import {
  CoverageMap,
  FingerprintsPanel,
  LogTailsPanel,
  MissingSecretsForm,
  PortSummaryPanel,
  PreflightCheckList,
  PreflightStepHeader,
  RuntimeInfoPanel,
  ServicesReadinessGrid,
  ValidationIssuesPanel,
  ValidationWarningsPanel
} from "./preflight";

// ============================================================================
// Props
// ============================================================================

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
  onStopSession: () => void;
}

// ============================================================================
// Status Resolution Helpers
// ============================================================================

function resolveJobStepStatus(
  jobStepById: Map<string, BundlePreflightStep>,
  stepId: string
): PreflightStepStatus | null {
  const step = jobStepById.get(stepId);
  if (!step) {
    return null;
  }
  const state = step.state === "running" ? "testing" : step.state;
  return {
    state: state as PreflightStepState,
    label: JOB_STEP_STATE_LABELS[step.state] || step.state
  };
}

function getValidationStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  validationValid?: boolean
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Testing" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (validationValid === true) {
    return { state: "pass", label: "Pass" };
  }
  if (validationValid === false) {
    return { state: "fail", label: "Fail" };
  }
  return { state: "warning", label: "Review" };
}

function getSecretsStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  missingSecretsCount: number
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Checking" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (missingSecretsCount > 0) {
    return { state: "warning", label: "Missing" };
  }
  return { state: "pass", label: "Ready" };
}

function getRuntimeStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasResult: boolean,
  hasRun: boolean
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Starting" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (hasResult) {
    return { state: "pass", label: "Running" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  return { state: "warning", label: "Unknown" };
}

function getServicesStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  ready?: boolean
): PreflightStepStatus {
  if (preflightPending) {
    return { state: "testing", label: "Starting" };
  }
  if (preflightError) {
    return { state: "fail", label: "Failed" };
  }
  if (!hasRun) {
    return { state: "pending", label: "Pending" };
  }
  if (ready === true) {
    return { state: "pass", label: "Ready" };
  }
  if (ready === false) {
    return { state: "warning", label: "Waiting (snapshot)" };
  }
  return { state: "warning", label: "Unknown" };
}

function getDiagnosticsStatus(
  preflightPending: boolean,
  preflightError: string | null,
  hasRun: boolean,
  diagnosticsAvailable: boolean
): PreflightStepStatus {
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
}

// ============================================================================
// Component
// ============================================================================

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
  onRun,
  onStopSession
}: BundledPreflightSectionProps) {
  const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">("idle");
  const [tick, setTick] = useState(() => Date.now());

  // ============================================================================
  // Derived State
  // ============================================================================

  const jobStepById = useMemo(() => {
    const steps = preflightJobStatus?.steps ?? [];
    const map = new Map<string, BundlePreflightStep>();
    steps.forEach((step) => map.set(step.id, step));
    return map;
  }, [preflightJobStatus?.steps]);

  const validation = preflightResult?.validation;
  const readiness = preflightResult?.ready;
  const ports = preflightResult?.ports;
  const telemetry = preflightResult?.telemetry;
  const runtimeInfo = preflightResult?.runtime;
  const fingerprints = preflightResult?.service_fingerprints ?? [];
  const logTails = preflightLogTails ?? preflightResult?.log_tails;
  const checks = preflightResult?.checks ?? [];
  const preflightErrors = preflightResult?.errors ?? [];

  const bundleRootPreview = getBundleRootFromManifestPath(bundleManifestPath);
  const readinessDetails = readiness?.details ? Object.entries(readiness.details) : [];
  const portSummary = formatPortSummary(ports);

  const latestServiceUpdate = readinessDetails.reduce((latest, [, status]) => {
    const ts = parseTimestamp(status.updated_at);
    return ts ? Math.max(latest, ts) : latest;
  }, 0);
  const snapshotTs = parseTimestamp(readiness?.snapshot_at) || latestServiceUpdate || 0;
  const snapshotLabel = snapshotTs ? new Date(snapshotTs).toLocaleTimeString() : "";
  const snapshotAge = snapshotTs ? formatDuration(Math.max(0, tick - snapshotTs)) : "";

  const likelyRootMismatch = detectLikelyRootMismatch(
    validation?.valid,
    validation?.missing_assets?.length ?? 0,
    validation?.missing_binaries?.length ?? 0,
    bundleManifestPath
  );

  const diagnosticsAvailable = Boolean(
    portSummary || telemetry?.path || (logTails && logTails.length > 0)
  );

  const hasRun = Boolean(preflightResult || preflightError || preflightJobStatus);

  // Filter checks by step
  const validationChecks = checks.filter((check) => check.step === "validation");
  const secretChecks = checks.filter((check) => check.step === "secrets");
  const runtimeChecks = checks.filter((check) => check.step === "runtime");
  const serviceChecks = checks.filter((check) => check.step === "services");
  const diagnosticsChecks = checks.filter((check) => check.step === "diagnostics");

  // Resolve step statuses
  const resolvedValidationStatus = resolveJobStepStatus(jobStepById, "validation")
    ?? getValidationStatus(preflightPending, preflightError, hasRun, validation?.valid);
  const resolvedSecretsStatus = resolveJobStepStatus(jobStepById, "secrets")
    ?? getSecretsStatus(preflightPending, preflightError, hasRun, missingSecrets.length);
  const resolvedRuntimeStatus = resolveJobStepStatus(jobStepById, "runtime")
    ?? getRuntimeStatus(preflightPending, preflightError, Boolean(preflightResult), hasRun);
  const resolvedServicesStatus = resolveJobStepStatus(jobStepById, "services")
    ?? getServicesStatus(preflightPending, preflightError, hasRun, readiness?.ready);
  const resolvedDiagnosticsStatus = resolveJobStepStatus(jobStepById, "diagnostics")
    ?? getDiagnosticsStatus(preflightPending, preflightError, hasRun, diagnosticsAvailable);

  const resolveJobStepDetail = (stepId: string) => jobStepById.get(stepId)?.detail;

  // JSON export payload
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

  // ============================================================================
  // Effects
  // ============================================================================

  useEffect(() => {
    if (!preflightResult) {
      return;
    }
    const interval = window.setInterval(() => setTick(Date.now()), 1000);
    return () => window.clearInterval(interval);
  }, [preflightResult]);

  // ============================================================================
  // Handlers
  // ============================================================================

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

  const downloadJson = () => {
    const blob = new Blob([JSON.stringify(preflightPayload, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "preflight.json";
    link.click();
    URL.revokeObjectURL(url);
  };

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/40 p-4 space-y-3">
      {/* Header */}
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
          {preflightSessionId && (
            <Button
              type="button"
              size="sm"
              variant="ghost"
              onClick={onStopSession}
              disabled={preflightPending}
            >
              Stop session
            </Button>
          )}
        </div>
      </div>

      {/* Session TTL control */}
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
              if (!Number.isNaN(next)) {
                onSessionTTLChange(Math.min(900, Math.max(30, next)));
              }
            }}
            className="h-7 w-20 text-xs"
          />
        </div>
      </div>

      {/* Warnings */}
      {!bundleManifestPath.trim() && (
        <p className="text-xs text-amber-200">Add a bundle_manifest_path to enable preflight checks.</p>
      )}

      {preflightError && (
        <div className="rounded-md border border-red-800 bg-red-950/40 p-3 text-sm text-red-200">
          {preflightError}
        </div>
      )}

      {/* Summary View */}
      {viewMode === "summary" && (
        <div className="space-y-4">
          {/* Bundle Context */}
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
            {preflightErrors.length > 0 && (
              <div className="rounded-md border border-amber-800/60 bg-amber-950/20 p-2 text-[11px] text-amber-100">
                <p className="font-semibold text-amber-100">Preflight warnings</p>
                <ul className="mt-1 space-y-1">
                  {preflightErrors.map((err, idx) => (
                    <li key={`preflight-error-${idx}`}>{err}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          {/* Runtime Info */}
          {runtimeInfo && <RuntimeInfoPanel runtimeInfo={runtimeInfo} />}

          {/* Steps */}
          <div className="space-y-3">
            {/* Step 1: Validation */}
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
                <p className="text-[11px] text-slate-300">{resolveJobStepDetail("validation")}</p>
              )}
              {validation?.valid && !preflightError && (
                <p className="text-[11px] text-slate-300">No validation issues detected.</p>
              )}
              {preflightError && !validation && (
                <p className="text-[11px] text-red-200">
                  Validation did not complete. Review the error above and re-run preflight.
                </p>
              )}
              {validation && !validation.valid && <ValidationIssuesPanel validation={validation} />}
              {validation?.warnings && <ValidationWarningsPanel warnings={validation.warnings} />}
              <PreflightCheckList checks={validationChecks} />
            </div>

            {/* Step 2: Secrets */}
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={2}
                title="Apply secrets"
                subtitle="Required secrets must be present for readiness"
                status={resolvedSecretsStatus}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">Run preflight to detect required secrets.</p>
              )}
              {resolveJobStepDetail("secrets") && (
                <p className="text-[11px] text-slate-300">{resolveJobStepDetail("secrets")}</p>
              )}
              {preflightError && (
                <p className="text-[11px] text-red-200">Secrets were not checked because preflight failed.</p>
              )}
              {hasRun && !preflightError && missingSecrets.length === 0 && (
                <p className="text-[11px] text-slate-300">All required secrets are present for this run.</p>
              )}
              <MissingSecretsForm
                missingSecrets={missingSecrets}
                secretInputs={secretInputs}
                preflightPending={preflightPending}
                onSecretChange={onSecretChange}
                onApplySecrets={(secrets) => onRun(secrets)}
              />
              <PreflightCheckList checks={secretChecks} />
            </div>

            {/* Step 3: Runtime */}
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
                <p className="text-[11px] text-slate-300">{resolveJobStepDetail("runtime")}</p>
              )}
              {!preflightPending && preflightResult && (
                <p className="text-[11px] text-slate-300">
                  Control API is responding. Runtime supervisor initialized.
                </p>
              )}
              {!preflightPending && !preflightResult && !preflightError && (
                <p className="text-[11px] text-slate-400">Run preflight to boot the runtime supervisor.</p>
              )}
              {preflightError && (
                <p className="text-[11px] text-red-200">Control API failed to start. Review the error above.</p>
              )}
              <PreflightCheckList checks={runtimeChecks} />
            </div>

            {/* Step 4: Services */}
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
                  <ServicesReadinessGrid
                    readinessDetails={readinessDetails}
                    ports={ports}
                    bundleManifest={bundleManifest}
                    preflightSessionId={preflightSessionId}
                    snapshotTs={snapshotTs}
                    tick={tick}
                  />
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
                <p className="text-[11px] text-slate-300">{resolveJobStepDetail("services")}</p>
              )}
              <PreflightCheckList checks={serviceChecks} />
            </div>

            {/* Step 5: Diagnostics */}
            <div className="rounded-md border border-slate-800 bg-slate-950/40 p-3 space-y-3">
              <PreflightStepHeader
                index={5}
                title="Diagnostics"
                subtitle="Ports, telemetry, and optional log tails"
                status={resolvedDiagnosticsStatus}
              />
              {!hasRun && (
                <p className="text-[11px] text-slate-400">Run preflight to collect diagnostics.</p>
              )}
              {resolveJobStepDetail("diagnostics") && (
                <p className="text-[11px] text-slate-300">{resolveJobStepDetail("diagnostics")}</p>
              )}
              {hasRun && !diagnosticsAvailable && (
                <p className="text-[11px] text-slate-400">No diagnostics reported yet.</p>
              )}
              <PortSummaryPanel portSummary={portSummary} telemetryPath={telemetry?.path} />
              {logTails && <LogTailsPanel logTails={logTails} />}
              {fingerprints.length > 0 && <FingerprintsPanel fingerprints={fingerprints} />}
              <PreflightCheckList checks={diagnosticsChecks} />
            </div>
          </div>

          {/* Coverage Map */}
          <CoverageMap />

          {/* Override */}
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

      {/* JSON View */}
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
                onClick={downloadJson}
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
