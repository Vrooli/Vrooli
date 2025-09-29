import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { BridgeRuleReport } from '@/types';

type DiagnosticsStatus = 'idle' | 'loading' | 'success' | 'error';
type DiagnosticsEvaluation = 'pass' | 'fail' | 'unknown';

interface UseIframeBridgeDiagnosticsOptions {
  appId?: string | null;
  enabled?: boolean;
}

interface UseIframeBridgeDiagnosticsResult {
  status: DiagnosticsStatus;
  loading: boolean;
  report: BridgeRuleReport | null;
  error: string | null;
  warning: string | null;
  evaluation: DiagnosticsEvaluation;
  hasViolations: boolean;
  scannedFileCount: number;
  lastFetchedAt: number | null;
  refresh: () => Promise<void>;
}

const DEFAULT_ERROR_MESSAGE = 'Failed to load iframe bridge diagnostics.';

const useIframeBridgeDiagnostics = ({ appId, enabled = true }: UseIframeBridgeDiagnosticsOptions): UseIframeBridgeDiagnosticsResult => {
  const [status, setStatus] = useState<DiagnosticsStatus>('idle');
  const [report, setReport] = useState<BridgeRuleReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);
  const [lastFetchedAt, setLastFetchedAt] = useState<number | null>(null);
  const requestTokenRef = useRef<number | null>(null);

  const fetchDiagnostics = useCallback(async () => {
    if (!appId) {
      setReport(null);
      setWarning(null);
      setError(null);
      setLastFetchedAt(null);
      setStatus('idle');
      return;
    }

    const token = Date.now();
    requestTokenRef.current = token;
    setStatus('loading');
    setError(null);
    setWarning(null);

    try {
      const response = await appService.getIframeBridgeDiagnostics(appId);

      if (requestTokenRef.current !== token) {
        return;
      }

      if (!response?.success || !response.data) {
        const message = response?.error || response?.message || DEFAULT_ERROR_MESSAGE;
        setReport(null);
        setWarning(null);
        setError(message);
        setLastFetchedAt(null);
        setStatus('error');
        return;
      }

      setReport(response.data);
      setWarning(response.warning ?? null);
      setError(null);
      setLastFetchedAt(Date.now());
      setStatus('success');
    } catch (cause) {
      if (requestTokenRef.current !== token) {
        return;
      }

      logger.warn('Failed to fetch iframe bridge diagnostics', cause);
      const message = (cause as { message?: string })?.message || DEFAULT_ERROR_MESSAGE;
      setReport(null);
      setWarning(null);
      setError(message);
      setLastFetchedAt(null);
      setStatus('error');
    }
  }, [appId]);

  useEffect(() => {
    requestTokenRef.current = null;

    if (!enabled) {
      setStatus('idle');
      setReport(null);
      setWarning(null);
      setError(null);
      setLastFetchedAt(null);
      return;
    }

    void fetchDiagnostics();

    return () => {
      requestTokenRef.current = null;
    };
  }, [enabled, fetchDiagnostics]);

  const hasViolations = useMemo(() => Boolean(report?.violations?.length), [report]);
  const scannedFileCount = useMemo(() => report?.files_scanned ?? 0, [report]);

  const evaluation: DiagnosticsEvaluation = useMemo(() => {
    if (status !== 'success' || !report) {
      return 'unknown';
    }

    if (scannedFileCount <= 0) {
      return 'fail';
    }

    if (hasViolations) {
      return 'fail';
    }

    return 'pass';
  }, [hasViolations, report, scannedFileCount, status]);

  const refresh = useCallback(async () => {
    await fetchDiagnostics();
  }, [fetchDiagnostics]);

  return {
    status,
    loading: status === 'loading',
    report,
    error,
    warning,
    evaluation,
    hasViolations,
    scannedFileCount,
    lastFetchedAt,
    refresh,
  };
};

export default useIframeBridgeDiagnostics;
