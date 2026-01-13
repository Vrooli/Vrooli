/**
 * Hook for managing preflight session state and polling.
 * Handles starting, stopping, and refreshing preflight validations.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  startBundlePreflight,
  fetchBundlePreflightStatus,
  runBundlePreflight,
  type BundlePreflightResponse,
  type BundlePreflightJobStatusResponse,
  type BundlePreflightSecret
} from "../lib/api";

export interface UsePreflightSessionOptions {
  bundleManifestPath: string;
  isBundled: boolean;
  initialSessionId?: string | null;
  initialSessionExpiresAt?: string | null;
  initialResult?: BundlePreflightResponse | null;
  initialError?: string | null;
  initialOverride?: boolean;
  initialSecrets?: Record<string, string>;
  initialSessionTTL?: number;
  initialStartServices?: boolean;
  initialAutoRefresh?: boolean;
  /** Called when preflight completes successfully. Use to persist stage results. */
  onPreflightComplete?: (result: BundlePreflightResponse) => void;
}

export interface UsePreflightSessionResult {
  // State
  result: BundlePreflightResponse | null;
  error: string | null;
  pending: boolean;
  jobStatus: BundlePreflightJobStatusResponse | null;
  sessionId: string | null;
  sessionExpiresAt: string | null;
  sessionTTL: number;
  override: boolean;
  secrets: Record<string, string>;
  startServices: boolean;
  autoRefresh: boolean;
  missingSecrets: BundlePreflightSecret[];

  // Status flags
  validationOk: boolean;
  readinessOk: boolean;
  secretsOk: boolean;
  preflightOk: boolean;

  // Actions
  setSessionTTL: (ttl: number) => void;
  setOverride: (override: boolean) => void;
  setSecret: (id: string, value: string) => void;
  setSecrets: (secrets: Record<string, string>) => void;
  setAutoRefresh: (autoRefresh: boolean) => void;
  setStartServices: (startServices: boolean) => void;
  runPreflight: (secretsOverride?: Record<string, string>, manifestPathOverride?: string) => Promise<void>;
  refreshStatus: () => Promise<void>;
  stopSession: () => Promise<void>;
  reset: () => void;
}

const DEFAULT_SESSION_TTL = 120;

/**
 * Hook for managing preflight session lifecycle.
 */
export function usePreflightSession({
  bundleManifestPath,
  isBundled,
  initialSessionId = null,
  initialSessionExpiresAt = null,
  initialResult = null,
  initialError = null,
  initialOverride = false,
  initialSecrets = {},
  initialSessionTTL = DEFAULT_SESSION_TTL,
  initialStartServices = true,
  initialAutoRefresh = true,
  onPreflightComplete
}: UsePreflightSessionOptions): UsePreflightSessionResult {
  // Core state
  const [result, setResult] = useState<BundlePreflightResponse | null>(initialResult);
  const [error, setError] = useState<string | null>(initialError);
  const [pending, setPending] = useState(false);
  const [jobId, setJobId] = useState<string | null>(null);
  const [jobStatus, setJobStatus] = useState<BundlePreflightJobStatusResponse | null>(null);

  // Session state
  const [sessionId, setSessionId] = useState<string | null>(initialSessionId);
  const [sessionExpiresAt, setSessionExpiresAt] = useState<string | null>(initialSessionExpiresAt);
  const [sessionTTL, setSessionTTL] = useState(initialSessionTTL);

  // Configuration state
  const [override, setOverride] = useState(initialOverride);
  const [secrets, setSecrets] = useState<Record<string, string>>(initialSecrets);
  const [startServices, setStartServices] = useState(initialStartServices);
  const [autoRefresh, setAutoRefresh] = useState(initialAutoRefresh);

  // Ref for interval callback to avoid stale closures
  const refreshStatusRef = useRef<(() => Promise<void>) | null>(null);
  // Track previous manifest path to detect genuine user changes vs initial load
  const prevManifestPathRef = useRef<string>(bundleManifestPath);
  // Track if we've received initial values from server (to avoid resetting on load)
  const hasReceivedInitialRef = useRef<boolean>(Boolean(initialResult));

  // Computed values
  const missingSecrets = useMemo(() => {
    if (!result?.secrets) return [];
    return result.secrets.filter((secret) => secret.required && !secret.has_value);
  }, [result]);

  const validationOk = Boolean(result?.validation?.valid);
  const readinessOk = Boolean(result?.ready?.ready);
  const secretsOk = missingSecrets.length === 0;
  const preflightOk = Boolean(result) && validationOk && readinessOk && secretsOk;

  useEffect(() => {
    setSessionId(initialSessionId ?? null);
    setSessionExpiresAt(initialSessionExpiresAt ?? null);
  }, [initialSessionId, initialSessionExpiresAt]);

  useEffect(() => {
    setResult(initialResult ?? null);
    setError(initialError ?? null);
    setOverride(initialOverride);
    setSecrets(initialSecrets);
    if (typeof initialSessionTTL === "number") {
      setSessionTTL(initialSessionTTL);
    }
    setStartServices(initialStartServices);
    setAutoRefresh(initialAutoRefresh);
    // Mark that we've received initial values from server
    if (initialResult) {
      hasReceivedInitialRef.current = true;
    }
  }, [
    initialResult,
    initialError,
    initialOverride,
    initialSecrets,
    initialSessionTTL,
    initialStartServices,
    initialAutoRefresh
  ]);

  // Reset state when bundleManifestPath changes or bundled mode is disabled
  useEffect(() => {
    if (!isBundled) {
      setResult(null);
      setError(null);
      setOverride(false);
      setSecrets({});
      prevManifestPathRef.current = bundleManifestPath;
      return;
    }

    const prevPath = prevManifestPathRef.current;
    prevManifestPathRef.current = bundleManifestPath;

    // Only reset if the manifest path genuinely changed (user changed it),
    // not on initial load from server where path goes from "" to loaded value
    const isGenuineChange = prevPath !== bundleManifestPath && prevPath !== "";

    // Also skip reset if we just received initial values from server
    if (!isGenuineChange && hasReceivedInitialRef.current) {
      hasReceivedInitialRef.current = false; // Reset flag after first load
      return;
    }

    if (isGenuineChange) {
      // Reset when manifest path actually changes to a different value
      setResult(null);
      setError(null);
      setOverride(false);
      setSecrets({});
    }
  }, [bundleManifestPath, isBundled]);

  // Poll for job status when job is running
  useEffect(() => {
    if (!jobId) return;

    let cancelled = false;
    let timeoutId: number | undefined;

    const poll = async () => {
      try {
        const status = await fetchBundlePreflightStatus({ job_id: jobId });
        if (cancelled) return;

        setJobStatus(status);
        setPending(status.status === "running");

        if (status.result) {
          setResult(status.result);
          setSessionId(status.result.session_id ?? null);
          setSessionExpiresAt(status.result.expires_at ?? null);
        }

        if (status.status === "failed") {
          setError(status.error || "Preflight failed.");
          setPending(false);
          return;
        }

        if (status.status === "completed") {
          setPending(false);
          // Notify parent when preflight completes successfully
          if (status.result && onPreflightComplete) {
            onPreflightComplete(status.result);
          }
          return;
        }

        timeoutId = window.setTimeout(poll, 1500);
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
  }, [jobId, onPreflightComplete]);

  // Auto-refresh status when session is active
  useEffect(() => {
    if (!autoRefresh || pending) return;

    const manifestPath = bundleManifestPath.trim();
    if (!manifestPath || !result || !sessionId) return;

    const interval = window.setInterval(() => {
      void refreshStatusRef.current?.();
    }, 10000);

    return () => window.clearInterval(interval);
  }, [autoRefresh, pending, bundleManifestPath, secrets, result, sessionId]);

  // Filter and format secrets for API calls
  const filterSecrets = useCallback((secretsInput: Record<string, string>) => {
    return Object.entries(secretsInput)
      .filter(([, value]) => value.trim())
      .reduce<Record<string, string>>((acc, [key, value]) => {
        acc[key] = value;
        return acc;
      }, {});
  }, []);

  // Run preflight validation
  const runPreflight = useCallback(async (
    secretsOverride?: Record<string, string>,
    manifestPathOverride?: string
  ) => {
    const manifestPath = (manifestPathOverride ?? bundleManifestPath).trim();
    if (!manifestPath) {
      setError("Provide bundle_manifest_path before running preflight.");
      return;
    }

    setPending(true);
    setError(null);
    setResult(null);
    setJobStatus(null);
    setJobId(null);
    setSessionId(null);
    setSessionExpiresAt(null);

    try {
      const timeoutSeconds = startServices ? 120 : 15;
      const filteredSecrets = filterSecrets(secretsOverride ?? secrets);

      const job = await startBundlePreflight({
        bundle_manifest_path: manifestPath,
        secrets: Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        start_services: startServices,
        timeout_seconds: timeoutSeconds,
        log_tail_lines: startServices ? 80 : undefined,
        session_ttl_seconds: startServices ? sessionTTL : undefined,
        session_id: sessionId ?? undefined
      });

      setOverride(false);
      setJobId(job.job_id);
    } catch (err) {
      setResult(null);
      setError(err instanceof Error ? err.message : String(err));
      setPending(false);
    }
  }, [bundleManifestPath, startServices, sessionTTL, sessionId, secrets, filterSecrets]);

  // Refresh preflight status without restarting
  const refreshStatusImpl = useCallback(async () => {
    if (!result || pending || !sessionId || jobStatus?.status === "running") return;

    const manifestPath = bundleManifestPath.trim();
    if (!manifestPath) return;

    setPending(true);

    try {
      const filteredSecrets = filterSecrets(secrets);
      const refreshResult = await runBundlePreflight({
        bundle_manifest_path: manifestPath,
        secrets: Object.keys(filteredSecrets).length > 0 ? filteredSecrets : undefined,
        start_services: startServices,
        timeout_seconds: 15,
        log_tail_lines: startServices ? 80 : undefined,
        status_only: true,
        session_id: sessionId,
        session_ttl_seconds: sessionTTL
      });

      setResult((prev) => {
        if (!prev) return refreshResult;

        const nextChecks = refreshResult.checks ?? prev.checks;
        const mergedChecks = prev.checks
          ? prev.checks.map((check) => {
              const updated = nextChecks?.find((item) => item.id === check.id);
              if (!updated) return check;
              if (updated.step === "services" || updated.step === "diagnostics") {
                return updated;
              }
              return check;
            })
          : nextChecks;

        return {
          ...prev,
          ready: refreshResult.ready ?? prev.ready,
          ports: refreshResult.ports ?? prev.ports,
          telemetry: refreshResult.telemetry ?? prev.telemetry,
          log_tails: refreshResult.log_tails ?? prev.log_tails,
          checks: mergedChecks ?? prev.checks
        };
      });

      setSessionExpiresAt(prev => refreshResult.expires_at ?? prev);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setPending(false);
    }
  }, [result, pending, sessionId, jobStatus, bundleManifestPath, secrets, startServices, sessionTTL, filterSecrets]);

  // Keep ref updated with latest callback
  useEffect(() => {
    refreshStatusRef.current = refreshStatusImpl;
  }, [refreshStatusImpl]);

  // Stop the preflight session
  const stopSession = useCallback(async () => {
    if (!sessionId || pending) return;

    const manifestPath = bundleManifestPath.trim();
    if (!manifestPath) return;

    setPending(true);

    try {
      await runBundlePreflight({
        bundle_manifest_path: manifestPath,
        status_only: true,
        session_id: sessionId,
        session_stop: true
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setPending(false);
      setSessionId(null);
      setSessionExpiresAt(null);
    }
  }, [sessionId, pending, bundleManifestPath]);

  // Reset all preflight state
  const reset = useCallback(() => {
    setResult(null);
    setError(null);
    setPending(false);
    setJobId(null);
    setJobStatus(null);
    setSessionId(null);
    setSessionExpiresAt(null);
    setOverride(false);
    setSecrets({});
    setStartServices(true);
    setAutoRefresh(true);
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
    jobStatus,
    sessionId,
    sessionExpiresAt,
    sessionTTL,
    override,
    secrets,
    startServices,
    autoRefresh,
    missingSecrets,

    // Status flags
    validationOk,
    readinessOk,
    secretsOk,
    preflightOk,

    // Actions
    setSessionTTL,
    setOverride,
    setSecret,
    setSecrets: replaceSecrets,
    setAutoRefresh,
    setStartServices,
    runPreflight,
    refreshStatus: refreshStatusImpl,
    stopSession,
    reset
  };
}
