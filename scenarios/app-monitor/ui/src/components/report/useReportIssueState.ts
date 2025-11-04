/**
 * Main hook for managing report issue dialog state
 * Orchestrates multiple data-fetching hooks and manages the submission workflow
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { ChangeEvent, FormEvent, KeyboardEvent, MutableRefObject, RefObject } from 'react';
import type {
  BridgeLogEvent,
  BridgeLogLevel,
  BridgeLogStreamState,
  BridgeNetworkEvent,
  BridgeNetworkStreamState,
  BridgeScreenshotMode,
  BridgeScreenshotOptions,
} from '@vrooli/iframe-bridge';

import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { useReportScreenshot } from '@/hooks/useReportScreenshot';
import { appService } from '@/services/api';
import type {
  ReportIssueConsoleLogEntry,
  ReportIssueNetworkEntry,
  ReportIssuePayload,
  ReportIssueHealthCheckEntry,
} from '@/services/api';
import { logger } from '@/services/logger';
import type { App } from '@/types';
import { useScenarioEngagementStore } from '@/state/scenarioEngagementStore';
import { useScenarioIssuesStore } from '@/state/scenarioIssuesStore';
import { useSnackPublisher } from '@/notifications/useSnackPublisher';
import type { SnackPublishOptions, SnackUpdateOptions } from '@/notifications/snackBus';
import {
  estimateReportPayloadSize,
  validatePayloadSize,
} from '@/utils/payloadSize';

import { compressElementCapturesIfNeeded, compressPrimaryScreenshotIfNeeded } from './reportCaptureCompression';
import {
  validateReportSubmission,
  buildReportMessage,
} from './reportSubmissionHelpers';
import type { ReportElementCapture } from './reportTypes';

// Import all the new data hooks
import { useReportLogsData, type ReportLogsDataState } from './useReportLogsData';
import { useReportConsoleLogsData, type ReportConsoleLogsDataState } from './useReportConsoleLogsData';
import { useReportNetworkData, type ReportNetworkDataState } from './useReportNetworkData';
import { useReportHealthData, type ReportHealthDataState } from './useReportHealthData';
import { useReportAppStatusData, type ReportAppStatusDataState } from './useReportAppStatusData';
import { useReportExistingIssues, type ReportExistingIssuesState } from './useReportExistingIssues';
import { useReportDiagnosticsData, type ReportDiagnosticsDataState } from './useReportDiagnosticsData';

interface BridgePreviewState {
  isSupported: boolean;
  caps: string[];
}

interface UseReportIssueStateParams {
  isOpen: boolean;
  onClose: () => void;
  appId?: string;
  app: App | null;
  activePreviewUrl: string | null;
  canCaptureScreenshot: boolean;
  previewContainerRef: RefObject<HTMLDivElement | null>;
  iframeRef: RefObject<HTMLIFrameElement | null>;
  isPreviewSameOrigin: boolean;
  bridgeSupportsScreenshot: boolean;
  requestScreenshot: (options?: BridgeScreenshotOptions) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string;
    mode?: BridgeScreenshotMode;
    clip?: { x: number; y: number; width: number; height: number };
  }>;
  bridgeState: BridgePreviewState;
  logState: BridgeLogStreamState | null;
  configureLogs: ((config: { enable?: boolean; streaming?: boolean; levels?: BridgeLogLevel[]; bufferSize?: number }) => boolean) | null;
  getRecentLogs: () => BridgeLogEvent[];
  requestLogBatch: (options?: { since?: number; afterSeq?: number; limit?: number }) => Promise<BridgeLogEvent[]>;
  networkState: BridgeNetworkStreamState | null;
  configureNetwork: ((config: { enable?: boolean; streaming?: boolean; bufferSize?: number }) => boolean) | null;
  getRecentNetworkEvents: () => BridgeNetworkEvent[];
  requestNetworkBatch: (options?: { since?: number; afterSeq?: number; limit?: number }) => Promise<BridgeNetworkEvent[]>;
  bridgeCompliance: BridgeComplianceResult | null;
  elementCaptures: ReportElementCapture[];
  onElementCaptureNoteChange: (captureId: string, note: string) => void;
  onElementCaptureRemove: (captureId: string) => void;
  onElementCapturesReset: () => void;
  onPrimaryCaptureDraftChange?: (hasCapture: boolean) => void;
}

interface ReportFormState {
  message: string;
  submitting: boolean;
  error: string | null;
  handleMessageChange: (event: ChangeEvent<HTMLTextAreaElement>) => void;
  handleMessageKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  handleSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

interface ReportIssueStateResult {
  textareaRef: MutableRefObject<HTMLTextAreaElement | null>;
  form: ReportFormState;
  modal: {
    handleDismiss: () => void;
    handleReset: () => void;
  };
  logs: ReportLogsDataState;
  consoleLogs: ReportConsoleLogsDataState;
  network: ReportNetworkDataState;
  health: ReportHealthDataState;
  status: ReportAppStatusDataState;
  existingIssues: ReportExistingIssuesState;
  diagnostics: ReportDiagnosticsDataState;
  screenshot: ReturnType<typeof useReportScreenshot>;
  bridgeState: BridgePreviewState;
  reportSubmitting: boolean;
  reportError: string | null;
  reportIncludeScreenshot: boolean;
  diagnosticsSummaryIncluded: boolean;
  setIncludeDiagnosticsSummary: (value: boolean) => void;
  resetOnClose: () => void;
}

const useReportIssueState = ({
  isOpen,
  onClose,
  appId,
  app,
  activePreviewUrl,
  canCaptureScreenshot,
  previewContainerRef,
  iframeRef,
  isPreviewSameOrigin,
  bridgeSupportsScreenshot,
  requestScreenshot,
  bridgeState,
  logState,
  configureLogs,
  getRecentLogs,
  requestLogBatch,
  networkState,
  configureNetwork,
  getRecentNetworkEvents,
  requestNetworkBatch,
  bridgeCompliance,
  elementCaptures,
  onElementCapturesReset,
  onPrimaryCaptureDraftChange,
}: UseReportIssueStateParams): ReportIssueStateResult => {
  // Form state
  const [reportMessage, setReportMessage] = useState('');
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  // Lifecycle management
  const [shouldResetOnNextOpen, setShouldResetOnNextOpen] = useState(true);
  const lastResolvedAppIdRef = useRef<string | null>(null);

  // Zustand stores
  const markScenarioIssueCreated = useScenarioEngagementStore(state => state.markIssueCreated);
  const flagScenarioIssueReported = useScenarioIssuesStore(state => state.flagIssueReported);
  const snackPublisher = useSnackPublisher();

  // Compose all data hooks
  const logs = useReportLogsData({ app, appId });

  const consoleLogs = useReportConsoleLogsData({
    app,
    appId,
    activePreviewUrl,
    bridgeSupported: bridgeState.isSupported,
    bridgeCaps: bridgeState.caps,
    logState,
    configureLogs,
    getRecentLogs,
    requestLogBatch,
  });

  const network = useReportNetworkData({
    app,
    appId,
    activePreviewUrl,
    bridgeSupported: bridgeState.isSupported,
    bridgeCaps: bridgeState.caps,
    networkState,
    configureNetwork,
    getRecentNetworkEvents,
    requestNetworkBatch,
  });

  const health = useReportHealthData({ app, appId });
  const status = useReportAppStatusData({ app, appId });
  const existingIssues = useReportExistingIssues({ app, appId, isOpen });

  const diagnostics = useReportDiagnosticsData({
    app,
    appId,
    isOpen,
    bridgeCaps: bridgeState.caps,
    bridgeCompliance,
  });

  const screenshot = useReportScreenshot({
    reportDialogOpen: isOpen,
    canCaptureScreenshot,
    activePreviewUrl: activePreviewUrl ?? '',
    previewContainerRef,
    iframeRef,
    isPreviewSameOrigin,
    bridgeSupportsScreenshot,
    requestScreenshot,
    logger,
  });

  // Resolved app ID for lifecycle tracking
  const resolvedAppId = useMemo(() => {
    const candidate = (app?.id ?? appId ?? '').trim();
    return candidate === '' ? null : candidate;
  }, [app?.id, appId]);

  // Track primary capture draft changes
  useEffect(() => {
    if (!onPrimaryCaptureDraftChange) {
      return;
    }

    const hasCaptureDraft = Boolean(
      screenshot.reportIncludeScreenshot
      && canCaptureScreenshot
      && screenshot.reportScreenshotData
      && !screenshot.reportScreenshotLoading,
    );

    onPrimaryCaptureDraftChange(hasCaptureDraft);
  }, [
    onPrimaryCaptureDraftChange,
    screenshot.reportIncludeScreenshot,
    canCaptureScreenshot,
    screenshot.reportScreenshotData,
    screenshot.reportScreenshotLoading,
  ]);

  // Reset all state
  const resetState = useCallback(() => {
    setReportSubmitting(false);
    setReportError(null);
    setReportMessage('');
    logs.reset();
    consoleLogs.reset();
    network.reset();
    health.reset();
    status.reset();
  }, [logs, consoleLogs, network, health, status]);

  // Form handlers
  const handleReportMessageChange = useCallback((event: ChangeEvent<HTMLTextAreaElement>) => {
    setReportMessage(event.target.value);
  }, []);

  const handleReportMessageKeyDown = useCallback((event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Escape') {
      event.stopPropagation();
    }
  }, []);

  // Modal handlers
  const handleDialogDismiss = useCallback(() => {
    setShouldResetOnNextOpen(false);
    onClose();
  }, [onClose]);

  const handleDialogReset = useCallback(() => {
    setShouldResetOnNextOpen(true);
    resetState();
    screenshot.cleanupAfterDialogClose(canCaptureScreenshot);
    onElementCapturesReset();
    if (onPrimaryCaptureDraftChange) {
      onPrimaryCaptureDraftChange(false);
    }
    onClose();
  }, [
    canCaptureScreenshot,
    screenshot,
    onClose,
    onElementCapturesReset,
    onPrimaryCaptureDraftChange,
    resetState,
  ]);

  // Submit handler - now with direct state access, no refs needed!
  const handleSubmitReport = useCallback(async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmed = reportMessage.trim();
    const includeDiagnosticsSummary = diagnostics.diagnosticsSummaryIncluded;
    const diagnosticsDescriptionSnapshot = diagnostics.diagnosticsDescription;
    const targetAppId = app?.id ?? appId ?? '';

    // Validate the submission
    const validation = validateReportSubmission(
      reportMessage,
      elementCaptures,
      includeDiagnosticsSummary,
      screenshot.reportIncludeScreenshot,
      screenshot.reportScreenshotData,
      canCaptureScreenshot,
      targetAppId,
    );

    if (!validation.valid) {
      setReportError(validation.error!);
      return;
    }

    const includeScreenshot = screenshot.reportIncludeScreenshot && canCaptureScreenshot;

    const issueSubject = [app?.name, app?.scenario_name, targetAppId]
      .map(value => (value ?? '').toString().trim())
      .find(value => value.length > 0)
      ?? targetAppId;

    const snackMetadata = { appId: targetAppId } as Record<string, unknown>;
    let progressSnackId: string | null = null;

    setReportSubmitting(true);
    setReportError(null);
    setShouldResetOnNextOpen(false);
    if (isOpen) {
      onClose();
    }

    try {
      // Build the final report message
      const finalMessage = buildReportMessage(
        trimmed,
        elementCaptures,
        diagnosticsDescriptionSnapshot,
        includeDiagnosticsSummary,
      );

      if (!finalMessage) {
        setReportError('Unable to prepare report contents.');
        return;
      }

      // Compress element captures if needed to reduce payload size
      const capturePayloads = await compressElementCapturesIfNeeded(elementCaptures);

      if (includeScreenshot && screenshot.reportScreenshotData) {
        const primaryCapture = await compressPrimaryScreenshotIfNeeded(
          screenshot.reportScreenshotData,
          screenshot.reportScreenshotOriginalDimensions,
          screenshot.reportScreenshotClip,
        );
        capturePayloads.unshift(primaryCapture);
      }

      // Build the base payload
      const hasPrimaryDescription = trimmed.length > 0;
      const payload: ReportIssuePayload = {
        message: finalMessage,
        primaryDescription: hasPrimaryDescription ? trimmed : null,
        includeDiagnosticsSummary,
        includeScreenshot,
        previewUrl: activePreviewUrl || null,
        appName: app?.name ?? null,
        scenarioName: app?.scenario_name ?? null,
        source: 'app-monitor',
        screenshotData: includeScreenshot ? screenshot.reportScreenshotData ?? null : null,
      };

      if (capturePayloads.length > 0) {
        payload.captures = capturePayloads;
      }

      // Add app logs to payload
      const appLogsPayload = logs.buildPayload();
      Object.assign(payload, appLogsPayload);

      if (consoleLogs.includeConsoleLogs && consoleLogs.entries.length > 0) {
        payload.consoleLogs = consoleLogs.entries.map(entry => entry.payload as ReportIssueConsoleLogEntry);
        payload.consoleLogsTotal = typeof consoleLogs.total === 'number'
          ? consoleLogs.total
          : consoleLogs.entries.length;
        if (consoleLogs.formattedCapturedAt) {
          const timestamp = new Date().toISOString(); // Use current time as fallback
          payload.consoleLogsCapturedAt = timestamp;
        }
      }

      if (network.includeNetworkRequests && network.events.length > 0) {
        payload.networkRequests = network.events.map(entry => entry.payload as ReportIssueNetworkEntry);
        payload.networkRequestsTotal = typeof network.total === 'number'
          ? network.total
          : network.events.length;
        if (network.formattedCapturedAt) {
          const timestamp = new Date().toISOString(); // Use current time as fallback
          payload.networkCapturedAt = timestamp;
        }
      }

      if (health.includeHealthChecks && health.entries.length > 0) {
        const healthEntries: ReportIssueHealthCheckEntry[] = health.entries.map((entry) => ({
          id: entry.id,
          name: entry.name,
          status: entry.status,
          endpoint: entry.endpoint ?? null,
          latencyMs: typeof entry.latencyMs === 'number' ? entry.latencyMs : null,
          message: entry.message ?? null,
          code: entry.code ?? null,
          response: entry.response ?? null,
        }));
        payload.healthChecks = healthEntries;
        const totalHealthChecks = typeof health.total === 'number'
          ? health.total
          : healthEntries.length;
        if (totalHealthChecks > 0) {
          payload.healthChecksTotal = totalHealthChecks;
        }
        if (health.formattedCapturedAt) {
          const timestamp = new Date().toISOString(); // Use current time as fallback
          payload.healthChecksCapturedAt = timestamp;
        }
      }

      if (status.includeAppStatus && status.snapshot && status.snapshot.details.length > 0) {
        payload.appStatusLines = status.snapshot.details;
        payload.appStatusLabel = status.snapshot.statusLabel;
        payload.appStatusSeverity = status.snapshot.severity;
        const capturedSource = status.snapshot.capturedAt ?? null;
        if (capturedSource) {
          payload.appStatusCapturedAt = capturedSource;
        }
      }

      // Validate payload size before submission to prevent 413 errors
      const sizeEstimate = estimateReportPayloadSize(payload);
      const sizeValidation = validatePayloadSize(sizeEstimate.total);

      if (!sizeValidation.ok) {
        setReportError(sizeValidation.message ?? 'Payload too large. Remove some captures or logs.');
        logger.error('Report payload exceeds size limit', {
          estimated: sizeValidation.estimatedSize,
          max: sizeValidation.maxSize,
          breakdown: sizeEstimate,
        });
        return;
      }

      // Log size warning if approaching limit
      if (sizeValidation.warning) {
        logger.warn('Report payload size approaching limit', {
          estimated: sizeValidation.estimatedSize,
          percentUsed: `${(sizeValidation.percentUsed * 100).toFixed(1)}%`,
          breakdown: sizeEstimate,
        });
      }

      progressSnackId = snackPublisher.publish({
        variant: 'loading',
        title: 'Reporting issue',
        message: `Creating issue for ${issueSubject}…`,
        autoDismiss: false,
        dismissible: false,
        metadata: snackMetadata,
      });

      const response = await appService.reportAppIssue(targetAppId, payload);

      const issueId = response.data?.issue_id;
      const issueUrl = response.data?.issue_url;
      const successMessage = issueId
        ? `Issue ${issueId} created for ${issueSubject}.`
        : `Issue created for ${issueSubject}.`;

      const successDescriptor: SnackPublishOptions = {
        variant: 'success',
        title: 'Issue reported',
        message: successMessage,
        dismissible: true,
        autoDismiss: true,
        durationMs: 7000,
        action: issueUrl
          ? {
              label: 'View details',
              handler: () => {
                window.open(issueUrl, '_blank', 'noopener');
              },
            }
          : undefined,
        metadata: { ...snackMetadata, issueId, issueUrl },
      };

      const successPatch: SnackUpdateOptions = {
        variant: successDescriptor.variant,
        title: successDescriptor.title,
        message: successDescriptor.message,
        dismissible: successDescriptor.dismissible,
        autoDismiss: successDescriptor.autoDismiss,
        durationMs: successDescriptor.durationMs,
        action: successDescriptor.action,
        metadata: successDescriptor.metadata,
      };

      if (progressSnackId) {
        snackPublisher.patch(progressSnackId, successPatch);
      } else {
        snackPublisher.publish(successDescriptor);
      }

      flagScenarioIssueReported(targetAppId);
      const engagementIdentifier = appId ?? app?.id ?? null;
      if (engagementIdentifier) {
        markScenarioIssueCreated(engagementIdentifier);
      }

      onElementCapturesReset();
      if (onPrimaryCaptureDraftChange) {
        onPrimaryCaptureDraftChange(false);
      }
      resetState();
      screenshot.cleanupAfterDialogClose(canCaptureScreenshot);
      setShouldResetOnNextOpen(true);
    } catch (error: unknown) {
      const fallbackMessage = (error as { message?: string })?.message ?? 'Failed to send issue report.';
      setReportError(fallbackMessage);
      const errorDescriptor: SnackPublishOptions = {
        variant: 'error',
        title: 'Issue report failed',
        message: issueSubject ? `${fallbackMessage} (${issueSubject})` : fallbackMessage,
        dismissible: true,
        autoDismiss: false,
        metadata: snackMetadata,
      };

      const errorPatch: SnackUpdateOptions = {
        variant: errorDescriptor.variant,
        title: errorDescriptor.title,
        message: errorDescriptor.message,
        dismissible: errorDescriptor.dismissible,
        autoDismiss: errorDescriptor.autoDismiss,
        durationMs: errorDescriptor.durationMs,
        action: errorDescriptor.action,
        metadata: errorDescriptor.metadata,
      };

      if (progressSnackId) {
        snackPublisher.patch(progressSnackId, errorPatch);
      } else {
        snackPublisher.publish(errorDescriptor);
      }
    } finally {
      setReportSubmitting(false);
    }
  }, [
    reportMessage,
    elementCaptures,
    diagnostics.diagnosticsSummaryIncluded,
    diagnostics.diagnosticsDescription,
    screenshot,
    canCaptureScreenshot,
    app,
    appId,
    activePreviewUrl,
    logs,
    consoleLogs,
    network,
    health,
    status,
    isOpen,
    onClose,
    flagScenarioIssueReported,
    markScenarioIssueCreated,
    onElementCapturesReset,
    onPrimaryCaptureDraftChange,
    resetState,
    snackPublisher,
  ]);

  // Lifecycle management - simplified!
  useEffect(() => {
    if (!isOpen) {
      if (shouldResetOnNextOpen) {
        screenshot.cleanupAfterDialogClose(canCaptureScreenshot);
      }
      return;
    }

    if (!shouldResetOnNextOpen) {
      return;
    }

    resetState();
    screenshot.prepareForDialogOpen(canCaptureScreenshot);

    void logs.fetch({ force: true });
    void consoleLogs.fetch({ force: true });
    void network.fetch({ force: true });
    void health.fetch({ force: true });
    void status.fetch({ force: true });
    void diagnostics.refreshDiagnostics();

    setShouldResetOnNextOpen(false);
  }, [
    isOpen,
    shouldResetOnNextOpen,
    canCaptureScreenshot,
    screenshot,
    logs,
    consoleLogs,
    network,
    health,
    status,
    diagnostics,
    resetState,
  ]);

  // Track app ID changes
  useEffect(() => {
    if (lastResolvedAppIdRef.current === resolvedAppId) {
      return;
    }
    lastResolvedAppIdRef.current = resolvedAppId;
    setShouldResetOnNextOpen(true);
  }, [resolvedAppId]);

  // Return same interface as before (backwards compatible)
  return {
    textareaRef,
    form: {
      message: reportMessage,
      submitting: reportSubmitting,
      error: reportError,
      handleMessageChange: handleReportMessageChange,
      handleMessageKeyDown: handleReportMessageKeyDown,
      handleSubmit: handleSubmitReport,
    },
    modal: {
      handleDismiss: handleDialogDismiss,
      handleReset: handleDialogReset,
    },
    logs,
    consoleLogs,
    network,
    health,
    status,
    existingIssues,
    diagnostics,
    screenshot,
    bridgeState,
    reportSubmitting,
    reportError,
    reportIncludeScreenshot: screenshot.reportIncludeScreenshot,
    diagnosticsSummaryIncluded: diagnostics.diagnosticsSummaryIncluded,
    setIncludeDiagnosticsSummary: diagnostics.setIncludeDiagnosticsSummary,
    resetOnClose: resetState,
  };
};

// Re-export data state types with shorter names for backward compatibility
export type ReportDiagnosticsState = ReportDiagnosticsDataState;
export type ReportConsoleLogsState = ReportConsoleLogsDataState;
export type ReportLogsState = ReportLogsDataState;
export type ReportNetworkState = ReportNetworkDataState;
export type ReportHealthChecksState = ReportHealthDataState;
export type ReportAppStatusState = ReportAppStatusDataState;
export type ReportScreenshotState = ReturnType<typeof useReportScreenshot>;

export type {
  ReportIssueStateResult,
  ReportFormState,
  BridgePreviewState,
};

export default useReportIssueState;
