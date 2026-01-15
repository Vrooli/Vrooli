/**
 * Hook for managing preflight validation via the pipeline.
 * Handles running preflight, polling for status, and extracting results.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  runPreflightPipeline,
  getPipelineStatus,
  extractPreflightResult,
  cancelPipeline,
  type BundlePreflightResponse,
  type BundlePreflightSecret,
  type PipelineStatus,
  type PipelineConfig
} from "../lib/api";

export interface UsePreflightSessionOptions {
  scenarioName: string;
  bundleManifestPath: string;
  isBundled: boolean;
  initialResult?: BundlePreflightResponse | null;
  initialError?: string | null;
  initialOverride?: boolean;
  initialSecrets?: Record<string, string>;
  /** Called when preflight completes successfully. Use to persist stage results. */
  onPreflightComplete?: (result: BundlePreflightResponse) => void;
}

export interface UsePreflightSessionResult {
  // State
  result: BundlePreflightResponse | null;
  error: string | null;
  pending: boolean;
  pipelineId: string | null;
  pipelineStatus: PipelineStatus | null;
  override: boolean;
  secrets: Record<string, string>;
  missingSecrets: BundlePreflightSecret[];

  // Status flags
  validationOk: boolean;
  readinessOk: boolean;
  secretsOk: boolean;
  preflightOk: boolean;

  // Actions
  setOverride: (override: boolean) => void;
  setSecret: (id: string, value: string) => void;
  setSecrets: (secrets: Record<string, string>) => void;
  runPreflight: (secretsOverride?: Record<string, string>, configOverride?: Partial<PipelineConfig>) => Promise<void>;
  cancelPreflight: () => Promise<void>;
  reset: () => void;
}

/**
 * Hook for managing preflight validation via the pipeline.
 * Uses the pipeline with stop_after_stage: "preflight" to run bundle and preflight stages.
 */
export function usePreflightSession({
  scenarioName,
  bundleManifestPath,
  isBundled,
  initialResult = null,
  initialError = null,
  initialOverride = false,
  initialSecrets = {},
  onPreflightComplete
}: UsePreflightSessionOptions): UsePreflightSessionResult {
  // Core state
  const [result, setResult] = useState<BundlePreflightResponse | null>(initialResult);
  const [error, setError] = useState<string | null>(initialError);
  const [pending, setPending] = useState(false);
  const [pipelineId, setPipelineId] = useState<string | null>(null);
  const [pipelineStatus, setPipelineStatus] = useState<PipelineStatus | null>(null);

  // Configuration state
  const [override, setOverride] = useState(initialOverride);
  const [secrets, setSecrets] = useState<Record<string, string>>(initialSecrets);

  // Computed values
  const missingSecrets = useMemo(() => {
    if (!result?.secrets) return [];
    return result.secrets.filter((secret) => secret.required && !secret.has_value);
  }, [result]);

  const validationOk = Boolean(result?.validation?.valid);
  const readinessOk = Boolean(result?.ready?.ready);
  const secretsOk = missingSecrets.length === 0;
  const preflightOk = Boolean(result) && validationOk && readinessOk && secretsOk;

  // Sync initial values
  useEffect(() => {
    setResult(initialResult ?? null);
    setError(initialError ?? null);
    setOverride(initialOverride);
    setSecrets(initialSecrets);
  }, [initialResult, initialError, initialOverride, initialSecrets]);

  // Reset when bundled mode changes or manifest path changes
  useEffect(() => {
    if (!isBundled) {
      setResult(null);
      setError(null);
      setOverride(false);
      setSecrets({});
      setPipelineId(null);
      setPipelineStatus(null);
    }
  }, [isBundled, bundleManifestPath]);

  // Poll for pipeline status when running
  useEffect(() => {
    if (!pipelineId || !pending) return;

    let cancelled = false;
    let timeoutId: number | undefined;

    const poll = async () => {
      try {
        const status = await getPipelineStatus(pipelineId, { verbose: true });
        if (cancelled) return;

        setPipelineStatus(status);

        // Check if pipeline is done
        if (status.status === "completed" || status.status === "failed" || status.status === "cancelled") {
          setPending(false);

          if (status.status === "completed") {
            const preflightResult = extractPreflightResult(status);
            if (preflightResult) {
              setResult(preflightResult);
              onPreflightComplete?.(preflightResult);
            }
          } else if (status.status === "failed") {
            // Extract error from failed stage
            const failedStage = Object.values(status.stages || {}).find(s => s?.status === "failed");
            setError(failedStage?.error || status.error || "Pipeline failed");
          } else if (status.status === "cancelled") {
            setError("Preflight was cancelled");
          }
          return;
        }

        // Continue polling
        timeoutId = window.setTimeout(poll, 2000);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : String(err));
        setPending(false);
      }
    };

    poll();

    return () => {
      cancelled = true;
      if (timeoutId) window.clearTimeout(timeoutId);
    };
  }, [pipelineId, pending, onPreflightComplete]);

  // Filter secrets for API calls
  const filterSecrets = useCallback((secretsInput: Record<string, string>) => {
    return Object.entries(secretsInput)
      .filter(([, value]) => value.trim())
      .reduce<Record<string, string>>((acc, [key, value]) => {
        acc[key] = value;
        return acc;
      }, {});
  }, []);

  // Run preflight validation via pipeline
  const runPreflight = useCallback(async (
    secretsOverride?: Record<string, string>,
    configOverride?: Partial<PipelineConfig>
  ) => {
    if (!scenarioName) {
      setError("Scenario name is required");
      return;
    }

    const manifestPath = bundleManifestPath.trim();
    if (!manifestPath && isBundled) {
      setError("Bundle manifest path is required for bundled mode");
      return;
    }

    setPending(true);
    setError(null);
    setResult(null);
    setPipelineStatus(null);
    setPipelineId(null);

    try {
      const filteredSecrets = filterSecrets(secretsOverride ?? secrets);

      const response = await runPreflightPipeline(scenarioName, {
        bundle_manifest_path: manifestPath || undefined,
        preflight_secrets: Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        ...configOverride
      });

      setOverride(false);
      setPipelineId(response.pipeline_id);
    } catch (err) {
      setResult(null);
      setError(err instanceof Error ? err.message : String(err));
      setPending(false);
    }
  }, [scenarioName, bundleManifestPath, isBundled, secrets, filterSecrets]);

  // Cancel running preflight
  const cancelPreflight = useCallback(async () => {
    if (!pipelineId) return;

    try {
      await cancelPipeline(pipelineId);
      // Status will be updated via polling
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [pipelineId]);

  // Reset all state
  const reset = useCallback(() => {
    setResult(null);
    setError(null);
    setPending(false);
    setPipelineId(null);
    setPipelineStatus(null);
    setOverride(false);
    setSecrets({});
  }, []);

  // Update a single secret
  const setSecret = useCallback((id: string, value: string) => {
    setSecrets((prev) => ({ ...prev, [id]: value }));
  }, []);

  const replaceSecrets = useCallback((nextSecrets: Record<string, string>) => {
    setSecrets(nextSecrets);
  }, []);

  return {
    // State
    result,
    error,
    pending,
    pipelineId,
    pipelineStatus,
    override,
    secrets,
    missingSecrets,

    // Status flags
    validationOk,
    readinessOk,
    secretsOk,
    preflightOk,

    // Actions
    setOverride,
    setSecret,
    setSecrets: replaceSecrets,
    runPreflight,
    cancelPreflight,
    reset
  };
}
