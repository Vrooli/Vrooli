// DOC: docs/reference/api-endpoints.md#documentation-deep-search
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  DeepSearchJob,
  DeepSearchRequest,
  fetchDeepSearchJob,
  startDeepSearch,
} from "../services/documentationApi";

const DEFAULT_SCOPE = "global";
const DEFAULT_MAX_RESULTS = 10;

const isActiveStatus = (status?: string) => status === "pending" || status === "running";

export function useDeepSearchController() {
  const [query, setQuery] = useState("");
  const [scope, setScope] = useState(DEFAULT_SCOPE);
  const [scenario, setScenario] = useState("");
  const [basePath, setBasePath] = useState("");
  const [maxResults, setMaxResults] = useState(DEFAULT_MAX_RESULTS);
  const [followRefs, setFollowRefs] = useState(true);
  const [timeoutSeconds, setTimeoutSeconds] = useState<number | undefined>(undefined);
  const [job, setJob] = useState<DeepSearchJob | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const pollingRef = useRef<number | null>(null);

  const isRunning = isActiveStatus(job?.status);
  const hasResults = Boolean(job?.results && job.results.length > 0);

  const clear = useCallback(() => {
    setJob(null);
    setError(null);
    setQuery("");
  }, []);

  const submit = useCallback(async () => {
    const trimmed = query.trim();
    if (!trimmed) return;
    const normalizedScope = scope.trim() || DEFAULT_SCOPE;
    if (normalizedScope === "scenario" && !scenario.trim()) {
      setError("Scenario name is required for scenario scope.");
      return;
    }
    if (normalizedScope === "path" && !basePath.trim()) {
      setError("Base path is required for path scope.");
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      const payload: DeepSearchRequest = {
        query: trimmed,
        scope: normalizedScope,
        scenario: scenario.trim() || undefined,
        base_path: basePath.trim() || undefined,
        max_results: maxResults > 0 ? maxResults : DEFAULT_MAX_RESULTS,
        follow_refs: followRefs,
        timeout_seconds: timeoutSeconds && timeoutSeconds > 0 ? timeoutSeconds : undefined,
      };
      const created = await startDeepSearch(payload);
      setJob(created);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setIsSubmitting(false);
    }
  }, [query, scope, scenario, basePath, maxResults, followRefs, timeoutSeconds]);

  const pollStatus = useCallback(
    async (jobId: string) => {
      try {
        const updated = await fetchDeepSearchJob(jobId);
        setJob(updated);
      } catch (err) {
        setError((err as Error).message);
      }
    },
    [setJob],
  );

  useEffect(() => {
    if (!job || !job.job_id || !isActiveStatus(job.status)) {
      if (pollingRef.current) {
        window.clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
      return;
    }
    if (pollingRef.current) return;
    pollingRef.current = window.setInterval(() => {
      pollStatus(job.job_id);
    }, 2000);
    return () => {
      if (pollingRef.current) {
        window.clearInterval(pollingRef.current);
        pollingRef.current = null;
      }
    };
  }, [job, pollStatus]);

  const viewModel = useMemo(() => {
    return {
      statusLabel: job?.status ?? "idle",
      progressLabel: job?.progress ?? "",
      errorMessage: job?.error ?? error ?? "",
      results: job?.results ?? [],
    };
  }, [job, error]);

  return {
    query,
    setQuery,
    scope,
    setScope,
    scenario,
    setScenario,
    basePath,
    setBasePath,
    maxResults,
    setMaxResults,
    followRefs,
    setFollowRefs,
    timeoutSeconds,
    setTimeoutSeconds,
    job,
    hasResults,
    isRunning,
    isSubmitting,
    error,
    submit,
    clear,
    viewModel,
  };
}
