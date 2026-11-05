/**
 * Hook for Preflight section state management.
 * Extracts business logic from PreflightSection.tsx for testability.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import type { PipelineConfig } from "../lib/api";
import type { PreflightSecret } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import { writeToClipboard, triggerBlobDownload } from "../lib/browser";
import {
  detectLikelyRootMismatch,
  formatDuration,
  formatPortSummary,
  getBundleRootFromManifestPath,
  parseTimestamp,
} from "../lib/preflight-utils";
import {
  presentPreflight,
  type PreflightPresentation,
  type RuntimePresentation,
  type ServiceReadinessPresentation,
} from "../lib/preflightPresentation";
import {
  usePipelineStore,
  selectIsRunning,
  selectPreflightOk,
  selectMissingSecrets,
} from "../store";
import { type PreflightStepStatus } from "../services/preflight.service";
import {
  buildJobStepMap,
  resolveAllStepStatuses,
} from "../controllers/preflightController";
import { filterNonEmptySecrets } from "../controllers/pipelineController";

// ============================================================================
// Types
// ============================================================================

export interface UsePreflightSectionProps {
  scenarioName: string;
  bundleManifestPath?: string;
  bundleManifest?: unknown;
  isBundled?: boolean;
}

export interface PreflightStepStatuses {
  validation: PreflightStepStatus;
  secrets: PreflightStepStatus;
  runtime: PreflightStepStatus;
  services: PreflightStepStatus;
  diagnostics: PreflightStepStatus;
}

export interface UsePreflightSectionReturn {
  // View state
  viewMode: "summary" | "json";
  setViewMode: (mode: "summary" | "json") => void;
  copyStatus: "idle" | "copied" | "error";

  // Step statuses
  stepStatuses: PreflightStepStatuses;

  // Preflight result data (nullable since preflightResult can be null)
  validation: PreflightPresentation["validation"];
  readiness: PreflightPresentation["ready"];
  ports: PreflightPresentation["ports"] | undefined;
  telemetry: PreflightPresentation["telemetry"];
  runtimeInfo: RuntimePresentation | undefined;
  fingerprints: unknown[];
  logTails: unknown[] | undefined;
  checks: unknown[];
  preflightErrors: string[];

  // Computed values
  bundleRootPreview: string;
  portSummary: string | null;
  snapshotLabel: string;
  snapshotAge: string;
  likelyRootMismatch: boolean;
  diagnosticsAvailable: boolean;
  hasRun: boolean;
  readinessDetails: Array<[string, ServiceReadinessPresentation]>;
  snapshotTs: number;
  tick: number;

  // Filtered checks by step
  validationChecks: unknown[];
  secretChecks: unknown[];
  runtimeChecks: unknown[];
  serviceChecks: unknown[];
  diagnosticsChecks: unknown[];

  // Pipeline store state
  preflightResult: ReturnType<
    typeof usePipelineStore.getState
  >["preflightResult"];
  preflightSecrets: Record<string, string>;
  preflightOverride: boolean;
  preflightError: string | null;
  preflightPending: boolean;
  preflightOk: boolean;
  missingSecrets: PreflightSecret[];

  // Job step details
  resolveJobStepDetail: (stepId: string) => string | undefined;

  // JSON export
  preflightPayload: unknown;

  // Actions
  runPreflight: (
    secretsOverride?: Record<string, string>,
    configOverride?: Partial<PipelineConfig>,
  ) => Promise<void>;
  cancelPipeline: () => Promise<void>;
  setPreflightSecret: (id: string, value: string) => void;
  setPreflightOverride: (override: boolean) => void;
  copyJson: () => Promise<void>;
  downloadJson: () => void;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function usePreflightSection(
  props: UsePreflightSectionProps,
): UsePreflightSectionReturn {
  const { scenarioName, bundleManifestPath = "", isBundled = true } = props;

  // ========== Local State ==========
  const [viewMode, setViewMode] = useState<"summary" | "json">("summary");
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied" | "error">(
    "idle",
  );
  const [tick, setTick] = useState(() => Date.now());

  // ========== Pipeline Store State ==========
  const preflightResult = usePipelineStore((s) => s.preflightResult);
  const preflightSecrets = usePipelineStore((s) => s.preflightSecrets);
  const preflightOverride = usePipelineStore((s) => s.preflightOverride);
  const pipelineStatus = usePipelineStore((s) => s.pipelineStatus);
  const preflightError = usePipelineStore((s) => s.errorInfo?.message ?? null);
  const preflightPending = usePipelineStore(selectIsRunning);
  const preflightOk = usePipelineStore(selectPreflightOk);
  const missingSecrets = usePipelineStore(selectMissingSecrets);

  // Pipeline store actions
  const runPreflightStage = usePipelineStore((s) => s.runPreflightStage);
  const cancelPipelineAction = usePipelineStore((s) => s.cancelPipeline);
  const setPreflightSecretAction = usePipelineStore(
    (s) => s.setPreflightSecret,
  );
  const setPreflightOverrideAction = usePipelineStore(
    (s) => s.setPreflightOverride,
  );

  // ========== Derived State ==========

  // Map pipeline stages to preflight steps using controller function
  const jobStepById = useMemo(
    () => buildJobStepMap(pipelineStatus),
    [pipelineStatus],
  );

  // Extract result components
  const preflight = useMemo(
    () => (preflightResult ? presentPreflight(preflightResult) : undefined),
    [preflightResult],
  );
  const validation = preflight?.validation;
  const readiness = preflight?.ready;
  const ports = preflight?.ports;
  const telemetry = preflight?.telemetry;
  const runtimeInfo = preflight?.runtime;
  const fingerprints = preflight?.fingerprints ?? [];
  const logTails = preflight?.logTails;
  const checks = preflight?.checks ?? [];
  const preflightErrors = preflight?.errors ?? [];

  // Computed values
  const bundleRootPreview = getBundleRootFromManifestPath(bundleManifestPath);
  const readinessDetails = readiness?.details
    ? Object.entries(readiness.details)
    : [];
  const portSummary = formatPortSummary(ports);

  const latestServiceUpdate = readinessDetails.reduce((latest, [, status]) => {
    const ts = parseTimestamp(status.updated_at);
    return ts ? Math.max(latest, ts) : latest;
  }, 0);
  const snapshotTs =
    parseTimestamp(readiness?.snapshot_at) || latestServiceUpdate || 0;
  const snapshotLabel = snapshotTs
    ? new Date(snapshotTs).toLocaleTimeString()
    : "";
  const snapshotAge = snapshotTs
    ? formatDuration(Math.max(0, tick - snapshotTs))
    : "";

  const likelyRootMismatch = detectLikelyRootMismatch(
    validation?.valid,
    validation?.missing_assets.length ?? 0,
    validation?.missing_binaries.length ?? 0,
    bundleManifestPath,
  );

  const diagnosticsAvailable = Boolean(
    portSummary || telemetry?.path || (logTails && logTails.length > 0),
  );
  const hasRun = Boolean(preflightResult || preflightError || pipelineStatus);

  // Filter checks by step
  const validationChecks = checks.filter(
    (check) => check.step === "validation",
  );
  const secretChecks = checks.filter((check) => check.step === "secrets");
  const runtimeChecks = checks.filter((check) => check.step === "runtime");
  const serviceChecks = checks.filter((check) => check.step === "services");
  const diagnosticsChecks = checks.filter(
    (check) => check.step === "diagnostics",
  );

  // Resolve step statuses using controller function
  const stepStatuses: PreflightStepStatuses = useMemo(
    () =>
      resolveAllStepStatuses(
        jobStepById,
        preflightPending,
        preflightError,
        hasRun,
        preflight ?? null,
        diagnosticsAvailable,
        missingSecrets.length,
      ),
    [
      jobStepById,
      preflightPending,
      preflightError,
      hasRun,
      preflight,
      diagnosticsAvailable,
      missingSecrets.length,
    ],
  );

  const resolveJobStepDetail = useCallback(
    (stepId: string) => jobStepById.get(stepId)?.detail,
    [jobStepById],
  );

  // JSON export payload
  const preflightPayload = useMemo(
    () => ({
      bundle_manifest_path: bundleManifestPath,
      start_services: true,
      result: preflightResult,
      error: preflightError || undefined,
      missing_secrets: missingSecrets,
    }),
    [bundleManifestPath, preflightResult, preflightError, missingSecrets],
  );

  // ========== Effects ==========

  // Tick for snapshot age display
  useEffect(() => {
    if (!preflightResult) return;
    const interval = window.setInterval(() => {
      setTick(Date.now());
    }, 1000);
    return () => {
      window.clearInterval(interval);
    };
  }, [preflightResult]);

  // ========== Actions ==========

  const runPreflight = useCallback(
    async (
      secretsOverride?: Record<string, string>,
      configOverride?: Partial<PipelineConfig>,
    ) => {
      if (!scenarioName) return;
      const manifestPath = bundleManifestPath.trim();
      if (!manifestPath && isBundled) return;

      const filteredSecrets = filterNonEmptySecrets(
        secretsOverride ?? preflightSecrets,
      );

      setPreflightOverrideAction(false);

      await runPreflightStage({
        bundleManifestPath: manifestPath || undefined,
        preflightSecrets:
          Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        ...configOverride,
      });
    },
    [
      scenarioName,
      bundleManifestPath,
      isBundled,
      preflightSecrets,
      runPreflightStage,
      setPreflightOverrideAction,
    ],
  );

  const copyJson = useCallback(async () => {
    if (!preflightResult && !preflightError) return;
    const result = await writeToClipboard(
      JSON.stringify(preflightPayload, null, 2),
    );
    if (result.success) {
      setCopyStatus("copied");
      setTimeout(() => {
        setCopyStatus("idle");
      }, 1500);
    } else {
      console.warn("Failed to copy preflight JSON", result.error);
      setCopyStatus("error");
      setTimeout(() => {
        setCopyStatus("idle");
      }, 2000);
    }
  }, [preflightResult, preflightError, preflightPayload]);

  const downloadJson = useCallback(() => {
    const blob = new Blob([JSON.stringify(preflightPayload, null, 2)], {
      type: "application/json",
    });
    triggerBlobDownload(blob, "preflight.json");
  }, [preflightPayload]);

  // ========== Return ==========

  return {
    // View state
    viewMode,
    setViewMode,
    copyStatus,

    // Step statuses
    stepStatuses,

    // Preflight result data
    validation,
    readiness,
    ports,
    telemetry,
    runtimeInfo,
    fingerprints,
    logTails,
    checks,
    preflightErrors,

    // Computed values
    bundleRootPreview,
    portSummary,
    snapshotLabel,
    snapshotAge,
    likelyRootMismatch,
    diagnosticsAvailable,
    hasRun,
    readinessDetails,
    snapshotTs,
    tick,

    // Filtered checks
    validationChecks,
    secretChecks,
    runtimeChecks,
    serviceChecks,
    diagnosticsChecks,

    // Pipeline store state
    preflightResult,
    preflightSecrets,
    preflightOverride,
    preflightError,
    preflightPending,
    preflightOk,
    missingSecrets,

    // Job step details
    resolveJobStepDetail,

    // JSON export
    preflightPayload,

    // Actions
    runPreflight,
    cancelPipeline: cancelPipelineAction,
    setPreflightSecret: setPreflightSecretAction,
    setPreflightOverride: setPreflightOverrideAction,
    copyJson,
    downloadJson,
  };
}
