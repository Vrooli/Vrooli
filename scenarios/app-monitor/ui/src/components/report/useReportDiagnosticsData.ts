/**
 * Hook for managing diagnostics state in the report dialog
 */

import { useCallback, useEffect, useMemo, useState } from 'react';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import useIframeBridgeDiagnostics from '@/hooks/useIframeBridgeDiagnostics';
import type { App, BridgeRuleReport } from '@/types';
import { formatOptionalTimestamp } from './reportFormatters';
import { formatDiagnosticsDescription } from './reportDiagnosticsFormatter';
import {
  BRIDGE_FAILURE_LABELS,
  BRIDGE_CAPABILITY_DETAILS,
} from './reportConstants';

interface UseReportDiagnosticsDataParams {
  app: App | null;
  appId?: string;
  isOpen: boolean;
  bridgeCaps: string[];
  bridgeCompliance: BridgeComplianceResult | null;
}

interface DiagnosticsViolation {
  title: string;
  description?: string;
  recommendation?: string;
  file_path?: string;
  line?: number;
  type: string;
}

export interface ReportDiagnosticsDataState {
  diagnosticsState: 'loading' | 'error' | 'fail' | 'pass' | 'idle';
  diagnosticsLoading: boolean;
  diagnosticsWarning: string | null;
  diagnosticsError: string | null;
  diagnosticsViolations: DiagnosticsViolation[];
  diagnosticsScannedFileCount: number;
  diagnosticsCheckedAt: string | null;
  diagnosticsDescription: string;
  refreshDiagnostics: () => void;
  bridgeCompliance: BridgeComplianceResult | null;
  bridgeComplianceCheckedAt: string | null;
  bridgeComplianceFailures: string[];
  runtimeFailureEntries: Array<{ code: string; label: string; detail?: { title: string; recommendation: string } }>;
  diagnosticsRuleResults: BridgeRuleReport[];
  diagnosticsWarnings: string[];
  diagnosticsSummaryIncluded: boolean;
  setIncludeDiagnosticsSummary: (value: boolean) => void;
}

/**
 * Hook for managing diagnostics data and processing
 */
export function useReportDiagnosticsData({
  app,
  appId,
  isOpen,
  bridgeCaps,
  bridgeCompliance,
}: UseReportDiagnosticsDataParams): ReportDiagnosticsDataState {
  const [reportIncludeDiagnostics, setReportIncludeDiagnostics] = useState(false);
  const [reportIncludeDiagnosticsManuallySet, setReportIncludeDiagnosticsManuallySet] = useState(false);

  const {
    status: diagnosticsStatus,
    loading: diagnosticsLoading,
    report: diagnosticsReport,
    error: diagnosticsError,
    warning: diagnosticsWarning,
    evaluation: diagnosticsEvaluation,
    scannedFileCount: diagnosticsScannedFileCount,
    lastFetchedAt: diagnosticsLastFetchedAt,
    refresh: refreshDiagnostics,
  } = useIframeBridgeDiagnostics({
    appId,
    enabled: isOpen && Boolean(appId),
  });

  const diagnosticsCheckedAt = useMemo(
    () => formatOptionalTimestamp(diagnosticsLastFetchedAt),
    [diagnosticsLastFetchedAt],
  );

  const runtimeFailureCodes = useMemo(() => {
    const codes = new Set<string>();

    if (bridgeCompliance && Array.isArray(bridgeCompliance.failures)) {
      bridgeCompliance.failures.forEach(rawCode => {
        const code = (rawCode ?? '').toString().trim();
        if (code) {
          codes.add(code);
        }
      });
    }

    if (!bridgeCaps.includes('logs')) {
      codes.add('CAP_LOGS');
    }
    if (!bridgeCaps.includes('network')) {
      codes.add('CAP_NETWORK');
    }

    return Array.from(codes);
  }, [bridgeCompliance, bridgeCaps]);

  const runtimeFailureEntries = useMemo(() => runtimeFailureCodes.map(code => ({
    code,
    label: BRIDGE_FAILURE_LABELS[code] ?? code,
    detail: BRIDGE_CAPABILITY_DETAILS[code],
  })), [runtimeFailureCodes]);

  const bridgeComplianceFailures = useMemo(
    () => runtimeFailureEntries.map(entry => entry.label),
    [runtimeFailureEntries],
  );

  const diagnosticsViolations = useMemo(() => diagnosticsReport?.violations ?? [], [diagnosticsReport]);

  const baseDiagnosticsRuleResults = useMemo(() => diagnosticsReport?.results ?? [], [diagnosticsReport]);

  const runtimeDiagnosticsRule = useMemo<BridgeRuleReport | null>(() => {
    if (runtimeFailureEntries.length === 0) {
      return null;
    }

    const scenarioSlug = (() => {
      const candidates = [
        diagnosticsReport?.scenario,
        app?.scenario_name,
        app?.id,
        appId,
      ];
      const match = candidates
        .map(value => (value ?? '').toString().trim())
        .find(value => value.length > 0);
      return match ?? 'scenario';
    })();

    const checkedAtIso = (() => {
      if (bridgeCompliance?.checkedAt) {
        try {
          return new Date(bridgeCompliance.checkedAt).toISOString();
        } catch {
          // fall through
        }
      }
      return new Date().toISOString();
    })();

    const violations = runtimeFailureEntries.map((entry, index) => ({
      type: 'runtime_bridge_capability',
      title: entry.detail?.title ?? entry.label,
      description: entry.label,
      file_path: 'runtime-bridge',
      line: index + 1,
      recommendation: entry.detail?.recommendation ?? 'Restart the scenario preview so the iframe bridge reinitializes with capture capabilities enabled.',
      severity: 'high',
      standard: 'runtime',
    }));

    return {
      rule_id: 'runtime_bridge_capabilities',
      name: 'Runtime bridge capabilities',
      scenario: scenarioSlug,
      files_scanned: 0,
      duration_ms: 0,
      warning: undefined,
      warnings: [],
      targets: ['runtime'],
      violations,
      checked_at: checkedAtIso,
    } satisfies BridgeRuleReport;
  }, [app?.id, app?.scenario_name, appId, bridgeCompliance?.checkedAt, diagnosticsReport?.scenario, runtimeFailureEntries]);

  const diagnosticsRuleResults = useMemo(() => (
    runtimeDiagnosticsRule ? [...baseDiagnosticsRuleResults, runtimeDiagnosticsRule] : baseDiagnosticsRuleResults
  ), [baseDiagnosticsRuleResults, runtimeDiagnosticsRule]);

  const diagnosticsWarnings = useMemo(() => {
    if (!diagnosticsReport) {
      return [] as string[];
    }

    const collected: string[] = [];
    if (Array.isArray(diagnosticsReport.warnings)) {
      for (const entry of diagnosticsReport.warnings) {
        const trimmed = typeof entry === 'string' ? entry.trim() : '';
        if (trimmed) {
          collected.push(trimmed);
        }
      }
    }

    if (!collected.length && diagnosticsReport.warning) {
      const fallback = diagnosticsReport.warning.trim();
      if (fallback) {
        collected.push(fallback);
      }
    }

    return collected;
  }, [diagnosticsReport]);

  const diagnosticsDescription = useMemo(
    () => formatDiagnosticsDescription({
      diagnosticsReport,
      runtimeFailureEntries,
      diagnosticsRuleResults,
      diagnosticsScannedFileCount,
      diagnosticsWarnings,
      app,
      appId,
    }),
    [
      diagnosticsReport,
      runtimeFailureEntries,
      diagnosticsRuleResults,
      diagnosticsScannedFileCount,
      diagnosticsWarnings,
      app,
      appId,
    ],
  );

  useEffect(() => {
    const hasSummary = diagnosticsDescription.trim().length > 0;
    if (!hasSummary) {
      setReportIncludeDiagnostics(false);
      setReportIncludeDiagnosticsManuallySet(false);
      return;
    }

    if (!reportIncludeDiagnosticsManuallySet) {
      setReportIncludeDiagnostics(true);
    }
  }, [diagnosticsDescription, reportIncludeDiagnosticsManuallySet]);

  const diagnosticsSummaryIncluded = useMemo(() => (
    reportIncludeDiagnostics && diagnosticsDescription.trim().length > 0
  ), [reportIncludeDiagnostics, diagnosticsDescription]);

  const handleIncludeDiagnosticsSummaryChange = useCallback((checked: boolean) => {
    setReportIncludeDiagnostics(checked);
    setReportIncludeDiagnosticsManuallySet(true);
  }, []);

  const bridgeComplianceCheckedAt = useMemo(
    () => formatOptionalTimestamp(bridgeCompliance?.checkedAt),
    [bridgeCompliance?.checkedAt],
  );

  const diagnosticsState = useMemo(() => {
    if (diagnosticsLoading) {
      return 'loading';
    }
    if (diagnosticsStatus === 'error') {
      return 'error';
    }
    if (diagnosticsStatus === 'success' && diagnosticsEvaluation === 'fail') {
      return 'fail';
    }
    if (runtimeFailureEntries.length > 0) {
      return 'fail';
    }
    if (diagnosticsStatus === 'success' && diagnosticsEvaluation === 'pass') {
      return 'pass';
    }
    return 'idle';
  }, [diagnosticsEvaluation, diagnosticsLoading, diagnosticsStatus, runtimeFailureEntries.length]);

  return {
    diagnosticsState,
    diagnosticsLoading,
    diagnosticsWarning: diagnosticsWarning ?? null,
    diagnosticsError: diagnosticsError ?? null,
    diagnosticsViolations,
    diagnosticsScannedFileCount,
    diagnosticsCheckedAt,
    diagnosticsDescription,
    refreshDiagnostics,
    bridgeCompliance,
    bridgeComplianceCheckedAt,
    bridgeComplianceFailures,
    runtimeFailureEntries,
    diagnosticsRuleResults,
    diagnosticsWarnings,
    diagnosticsSummaryIncluded,
    setIncludeDiagnosticsSummary: handleIncludeDiagnosticsSummaryChange,
  };
}
