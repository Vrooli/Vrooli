import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import type { ChangeEvent, FormEvent, KeyboardEvent, MutableRefObject, RefObject, SyntheticEvent } from 'react';
import type {
  BridgeLogEvent,
  BridgeLogLevel,
  BridgeLogStreamState,
  BridgeNetworkEvent,
  BridgeNetworkStreamState,
} from '@vrooli/iframe-bridge';

import useIframeBridgeDiagnostics from '@/hooks/useIframeBridgeDiagnostics';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { useReportScreenshot } from '@/hooks/useReportScreenshot';
import { appService } from '@/services/api';
import type {
  ReportIssueConsoleLogEntry,
  ReportIssueNetworkEntry,
  ReportIssuePayload,
} from '@/services/api';
import { logger } from '@/services/logger';
import type { App, BridgeRuleReport } from '@/types';

import { toConsoleEntry, toNetworkEntry } from './reportFormatters';
import type {
  ReportAppLogStream,
  ReportConsoleEntry,
  ReportNetworkEntry,
} from './reportTypes';

const REPORT_APP_LOGS_MAX_LINES = 200;
const REPORT_CONSOLE_LOGS_MAX_LINES = 150;
const REPORT_NETWORK_MAX_EVENTS = 150;

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
  requestScreenshot: (options?: Record<string, unknown>) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string;
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
}

interface ReportResultState {
  issueId?: string;
  issueUrl?: string;
  message?: string;
}

interface ReportFormState {
  message: string;
  submitting: boolean;
  error: string | null;
  result: ReportResultState | null;
  handleMessageChange: (event: ChangeEvent<HTMLTextAreaElement>) => void;
  handleMessageKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  handleSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

interface ReportLogsState {
  includeAppLogs: boolean;
  setIncludeAppLogs: (value: boolean) => void;
  streams: ReportAppLogStream[];
  selections: Record<string, boolean>;
  toggleStream: (key: string, checked: boolean) => void;
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  truncated: boolean;
  formattedCapturedAt: string | null;
  logs: string[];
  total: number | null;
  fetch: () => void;
  selectedCount: number;
  totalStreamCount: number;
}

interface ReportConsoleLogsState {
  includeConsoleLogs: boolean;
  setIncludeConsoleLogs: (value: boolean) => void;
  entries: ReportConsoleEntry[];
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  truncated: boolean;
  formattedCapturedAt: string | null;
  total: number | null;
  fetch: () => void;
}

interface ReportNetworkState {
  includeNetworkRequests: boolean;
  setIncludeNetworkRequests: (value: boolean) => void;
  events: ReportNetworkEntry[];
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  truncated: boolean;
  formattedCapturedAt: string | null;
  total: number | null;
  fetch: () => void;
}

interface DiagnosticsViolation {
  title: string;
  description?: string;
  recommendation?: string;
  file_path?: string;
  line?: number;
  type: string;
}

interface ReportDiagnosticsState {
  diagnosticsState: 'loading' | 'error' | 'fail' | 'pass' | 'idle';
  diagnosticsLoading: boolean;
  diagnosticsWarning: string | null;
  diagnosticsError: string | null;
  diagnosticsViolations: DiagnosticsViolation[];
  diagnosticsScannedFileCount: number;
  diagnosticsCheckedAt: string | null;
  diagnosticsDescription: string;
  showDiagnosticsSetDescription: boolean;
  handleApplyDiagnosticsDescription: () => void;
  refreshDiagnostics: () => void;
  bridgeCompliance: BridgeComplianceResult | null;
  bridgeComplianceCheckedAt: string | null;
  bridgeComplianceFailures: string[];
  diagnosticsRuleResults: BridgeRuleReport[];
  diagnosticsWarnings: string[];
}

interface ReportScreenshotState {
  reportIncludeScreenshot: boolean;
  reportScreenshotData: string | null;
  reportScreenshotLoading: boolean;
  reportScreenshotError: string | null;
  reportScreenshotOriginalDimensions: { width: number; height: number } | null;
  reportSelectionRect: { x: number; y: number; width: number; height: number } | null;
  reportScreenshotClip: unknown;
  reportScreenshotInfo: string | null;
  reportScreenshotCountdown: number | null;
  assignScreenshotContainerRef: (node: HTMLDivElement | null) => void;
  reportScreenshotContainerHandlers: ReturnType<typeof useReportScreenshot>['reportScreenshotContainerHandlers'];
  selectionDimensionLabel: string | null;
  handleReportIncludeScreenshotChange: (event: ChangeEvent<HTMLInputElement>) => void;
  handleRetryScreenshotCapture: () => void;
  handleResetScreenshotSelection: () => void;
  handleDelayedScreenshotCapture: () => void;
  handleScreenshotImageLoad: (event: SyntheticEvent<HTMLImageElement>) => void;
}

interface ReportIssueStateResult {
  textareaRef: MutableRefObject<HTMLTextAreaElement | null>;
  form: ReportFormState;
  modal: {
    handleOverlayClick: () => void;
    handleDialogClose: () => void;
  };
  logs: ReportLogsState;
  consoleLogs: ReportConsoleLogsState;
  network: ReportNetworkState;
  diagnostics: ReportDiagnosticsState;
  screenshot: ReportScreenshotState;
  bridgeState: BridgePreviewState;
  reportResult: ReportResultState | null;
  reportSubmitting: boolean;
  reportError: string | null;
  reportIncludeScreenshot: boolean;
  resetOnClose: () => void;
}

const BRIDGE_FAILURE_LABELS: Record<string, string> = {
  HELLO: 'Preview never received the HELLO handshake from the iframe.',
  READY: 'Iframe bridge did not send the READY signal.',
  'SPA hooks': 'Single-page navigation hook did not respond to a test navigation.',
  'BACK/FWD': 'History navigation commands were not acknowledged by the iframe.',
  CHECK_FAILED: 'Runtime bridge check could not complete. Try refreshing the preview.',
  NO_IFRAME: 'Preview iframe was unavailable when the runtime check executed.',
};

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
}: UseReportIssueStateParams): ReportIssueStateResult => {
  const [reportMessage, setReportMessage] = useState('');
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const [reportResult, setReportResult] = useState<ReportResultState | null>(null);
  const [reportAppLogs, setReportAppLogs] = useState<string[]>([]);
  const [reportAppLogsTotal, setReportAppLogsTotal] = useState<number | null>(null);
  const [reportAppLogsLoading, setReportAppLogsLoading] = useState(false);
  const [reportAppLogsError, setReportAppLogsError] = useState<string | null>(null);
  const [reportAppLogsExpanded, setReportAppLogsExpanded] = useState(false);
  const [reportAppLogsFetchedAt, setReportAppLogsFetchedAt] = useState<number | null>(null);
  const [reportAppLogStreams, setReportAppLogStreams] = useState<ReportAppLogStream[]>([]);
  const [reportAppLogSelections, setReportAppLogSelections] = useState<Record<string, boolean>>({});
  const [reportIncludeAppLogs, setReportIncludeAppLogs] = useState(true);
  const [reportConsoleLogs, setReportConsoleLogs] = useState<ReportConsoleEntry[]>([]);
  const [reportConsoleLogsTotal, setReportConsoleLogsTotal] = useState<number | null>(null);
  const [reportConsoleLogsLoading, setReportConsoleLogsLoading] = useState(false);
  const [reportConsoleLogsError, setReportConsoleLogsError] = useState<string | null>(null);
  const [reportConsoleLogsExpanded, setReportConsoleLogsExpanded] = useState(false);
  const [reportConsoleLogsFetchedAt, setReportConsoleLogsFetchedAt] = useState<number | null>(null);
  const [reportIncludeConsoleLogs, setReportIncludeConsoleLogs] = useState(true);
  const [reportNetworkEvents, setReportNetworkEvents] = useState<ReportNetworkEntry[]>([]);
  const [reportNetworkTotal, setReportNetworkTotal] = useState<number | null>(null);
  const [reportNetworkLoading, setReportNetworkLoading] = useState(false);
  const [reportNetworkError, setReportNetworkError] = useState<string | null>(null);
  const [reportNetworkExpanded, setReportNetworkExpanded] = useState(false);
  const [reportNetworkFetchedAt, setReportNetworkFetchedAt] = useState<number | null>(null);
  const [reportIncludeNetworkRequests, setReportIncludeNetworkRequests] = useState(true);

  const reportAppLogsFetchedForRef = useRef<string | null>(null);
  const reportConsoleLogsFetchedForRef = useRef<string | null>(null);
  const reportNetworkFetchedForRef = useRef<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const {
    reportIncludeScreenshot,
    reportScreenshotData,
    reportScreenshotLoading,
    reportScreenshotError,
    reportScreenshotOriginalDimensions,
    reportSelectionRect,
    reportScreenshotClip,
    reportScreenshotInfo,
    reportScreenshotCountdown,
    reportScreenshotContainerRef,
    reportScreenshotContainerHandlers,
    selectionDimensionLabel,
    handleReportIncludeScreenshotChange,
    handleRetryScreenshotCapture,
    handleResetScreenshotSelection,
    handleDelayedScreenshotCapture,
    handleScreenshotImageLoad,
    prepareForDialogOpen,
    cleanupAfterDialogClose,
  } = useReportScreenshot({
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

  const assignScreenshotContainerRef = useCallback((node: HTMLDivElement | null) => {
    reportScreenshotContainerRef.current = node;
  }, [reportScreenshotContainerRef]);

  const resolveReportLogIdentifier = useCallback(() => {
    const candidates = [app?.scenario_name, app?.id, appId]
      .map(value => (typeof value === 'string' ? value.trim() : ''))
      .filter(Boolean);

    if (candidates.length === 0) {
      return null;
    }

    return candidates[0];
  }, [app, appId]);

  const fetchReportAppLogs = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveReportLogIdentifier();
    if (!identifier) {
      setReportAppLogs([]);
      setReportAppLogsTotal(null);
      setReportAppLogsError('Unable to determine which logs to include.');
      setReportAppLogsFetchedAt(null);
      reportAppLogsFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && reportAppLogsFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportAppLogsLoading(true);
    setReportAppLogsError(null);

    try {
      const result = await appService.getAppLogs(identifier, 'both');
      const rawLogs = Array.isArray(result.logs) ? result.logs : [];
      const trimmedLogs = rawLogs.slice(-REPORT_APP_LOGS_MAX_LINES);

      setReportAppLogs(trimmedLogs);
      setReportAppLogsTotal(rawLogs.length);
      const streams = Array.isArray(result.streams) ? result.streams : [];
      const normalizedStreams: ReportAppLogStream[] = streams.map((stream) => {
        const safeLines = Array.isArray(stream.lines) ? stream.lines : [];
        const trimmedLines = safeLines.slice(-REPORT_APP_LOGS_MAX_LINES);
        let label = stream.label || stream.key;
        if (!label) {
          label = stream.type === 'lifecycle' ? 'Lifecycle' : stream.step || 'Background';
        }
        if (stream.type === 'lifecycle') {
          label = 'Lifecycle';
        }
        return {
          key: stream.key,
          label,
          type: stream.type,
          lines: trimmedLines,
          total: safeLines.length,
          command: stream.command,
        };
      });

      setReportAppLogStreams(normalizedStreams);
      setReportAppLogSelections(previous => normalizedStreams.reduce<Record<string, boolean>>((acc, stream) => {
        const prior = previous?.[stream.key];
        acc[stream.key] = prior !== undefined ? prior : true;
        return acc;
      }, {}));
      setReportAppLogsFetchedAt(Date.now());
      reportAppLogsFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.error('Failed to load logs for issue report', error);
      setReportAppLogs([]);
      setReportAppLogsTotal(null);
      setReportAppLogStreams([]);
      setReportAppLogSelections({});
      setReportAppLogsError('Unable to load logs. Try again.');
      setReportAppLogsFetchedAt(null);
      reportAppLogsFetchedForRef.current = null;
    } finally {
      setReportAppLogsLoading(false);
    }
  }, [resolveReportLogIdentifier]);

  const fetchReportConsoleLogs = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveReportLogIdentifier();
    if (!identifier) {
      setReportConsoleLogs([]);
      setReportConsoleLogsTotal(null);
      setReportConsoleLogsError('Unable to determine which console logs to include.');
      setReportConsoleLogsFetchedAt(null);
      reportConsoleLogsFetchedForRef.current = null;
      return;
    }

    if (!bridgeState.isSupported || !bridgeState.caps.includes('logs')) {
      setReportConsoleLogs([]);
      setReportConsoleLogsTotal(null);
      setReportConsoleLogsError('Console log capture is not available for this preview.');
      setReportConsoleLogsFetchedAt(null);
      reportConsoleLogsFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && reportConsoleLogsFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportConsoleLogsLoading(true);
    setReportConsoleLogsError(null);

    try {
      if (logState && configureLogs && logState.enabled === false) {
        configureLogs({ enable: true });
      }

      let events: BridgeLogEvent[] = [];
      try {
        events = await requestLogBatch({ limit: REPORT_CONSOLE_LOGS_MAX_LINES });
      } catch (error) {
        logger.warn('Console log snapshot failed; using buffered events', error);
        events = getRecentLogs();
      }

      if (!Array.isArray(events)) {
        events = [];
      }

      const limited = events.slice(-REPORT_CONSOLE_LOGS_MAX_LINES);
      const entries = limited.map(toConsoleEntry);

      setReportConsoleLogs(entries);
      setReportConsoleLogsTotal(events.length);
      setReportConsoleLogsFetchedAt(Date.now());
      reportConsoleLogsFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to load console logs for issue report', error);
      setReportConsoleLogs([]);
      setReportConsoleLogsTotal(null);
      setReportConsoleLogsError('Unable to load console logs from the preview iframe.');
      setReportConsoleLogsFetchedAt(null);
      reportConsoleLogsFetchedForRef.current = null;
    } finally {
      setReportConsoleLogsLoading(false);
    }
  }, [
    bridgeState.caps,
    bridgeState.isSupported,
    configureLogs,
    getRecentLogs,
    logState,
    requestLogBatch,
    resolveReportLogIdentifier,
  ]);

  const fetchReportNetworkEvents = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveReportLogIdentifier();
    if (!identifier) {
      setReportNetworkEvents([]);
      setReportNetworkTotal(null);
      setReportNetworkError('Unable to determine which network events to include.');
      setReportNetworkFetchedAt(null);
      reportNetworkFetchedForRef.current = null;
      return;
    }

    if (!bridgeState.isSupported || !bridgeState.caps.includes('network')) {
      setReportNetworkEvents([]);
      setReportNetworkTotal(null);
      setReportNetworkError('Network capture is not available for this preview.');
      setReportNetworkFetchedAt(null);
      reportNetworkFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && reportNetworkFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportNetworkLoading(true);
    setReportNetworkError(null);

    try {
      if (networkState && configureNetwork && networkState.enabled === false) {
        configureNetwork({ enable: true });
      }

      let events: BridgeNetworkEvent[] = [];
      try {
        events = await requestNetworkBatch({ limit: REPORT_NETWORK_MAX_EVENTS });
      } catch (error) {
        logger.warn('Network request snapshot failed; using buffered events', error);
        events = getRecentNetworkEvents();
      }

      if (!Array.isArray(events)) {
        events = [];
      }

      const limited = events.slice(-REPORT_NETWORK_MAX_EVENTS);
      const entries = limited.map(toNetworkEntry);

      setReportNetworkEvents(entries);
      setReportNetworkTotal(events.length);
      setReportNetworkFetchedAt(Date.now());
      reportNetworkFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to load network events for issue report', error);
      setReportNetworkEvents([]);
      setReportNetworkTotal(null);
      setReportNetworkError('Unable to load network requests from the preview iframe.');
      setReportNetworkFetchedAt(null);
      reportNetworkFetchedForRef.current = null;
    } finally {
      setReportNetworkLoading(false);
    }
  }, [
    bridgeState.caps,
    bridgeState.isSupported,
    configureNetwork,
    getRecentNetworkEvents,
    networkState,
    requestNetworkBatch,
    resolveReportLogIdentifier,
  ]);

  const handleReportLogStreamToggle = useCallback((key: string, checked: boolean) => {
    setReportAppLogSelections(prev => ({
      ...prev,
      [key]: checked,
    }));
  }, []);

  const handleRefreshReportAppLogs = useCallback(() => {
    void fetchReportAppLogs({ force: true });
  }, [fetchReportAppLogs]);

  const handleRefreshReportConsoleLogs = useCallback(() => {
    void fetchReportConsoleLogs({ force: true });
  }, [fetchReportConsoleLogs]);

  const handleRefreshReportNetworkEvents = useCallback(() => {
    void fetchReportNetworkEvents({ force: true });
  }, [fetchReportNetworkEvents]);

  const handleReportMessageChange = useCallback((event: ChangeEvent<HTMLTextAreaElement>) => {
    setReportMessage(event.target.value);
  }, []);

  const handleReportMessageKeyDown = useCallback((event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Escape') {
      event.stopPropagation();
    }
  }, []);

  const handleOverlayClick = useCallback(() => {
    if (!reportSubmitting) {
      cleanupAfterDialogClose(canCaptureScreenshot);
      onClose();
    }
  }, [canCaptureScreenshot, cleanupAfterDialogClose, onClose, reportSubmitting]);

  const handleDialogClose = useCallback(() => {
    cleanupAfterDialogClose(canCaptureScreenshot);
    setReportSubmitting(false);
    setReportError(null);
    onClose();
  }, [canCaptureScreenshot, cleanupAfterDialogClose, onClose]);

  const handleSubmitReport = useCallback(async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmed = reportMessage.trim();
    if (!trimmed) {
      setReportError('Please add a short description of the issue.');
      return;
    }

    const targetAppId = app?.id ?? appId ?? '';
    if (!targetAppId) {
      setReportError('Unable to determine which application to report.');
      return;
    }

    const includeScreenshot = reportIncludeScreenshot && canCaptureScreenshot;
    if (includeScreenshot && !reportScreenshotData) {
      setReportError('Capture a screenshot before sending the report.');
      return;
    }

    setReportSubmitting(true);
    setReportError(null);

    try {
      const payload: ReportIssuePayload = {
        message: trimmed,
        includeScreenshot,
        previewUrl: activePreviewUrl || null,
        appName: app?.name ?? null,
        scenarioName: app?.scenario_name ?? null,
        source: 'app-monitor',
        screenshotData: includeScreenshot ? reportScreenshotData ?? null : null,
      };

      if (reportIncludeAppLogs) {
        const selectedStreams = reportAppLogStreams.filter(stream => reportAppLogSelections[stream.key] !== false);
        if (selectedStreams.length > 0) {
          const combinedLogs: string[] = [];

          selectedStreams.forEach((stream) => {
            if (combinedLogs.length >= REPORT_APP_LOGS_MAX_LINES) {
              return;
            }

            const contextLabel = stream.type === 'background'
              ? `Background: ${stream.label}`
              : 'Lifecycle';
            combinedLogs.push(`--- ${contextLabel} ---`);

            if (stream.type === 'background' && stream.command && combinedLogs.length < REPORT_APP_LOGS_MAX_LINES) {
              combinedLogs.push(`# ${stream.command}`);
            }

            const remaining = REPORT_APP_LOGS_MAX_LINES - combinedLogs.length;
            if (remaining > 0) {
              const tail = stream.lines.slice(-remaining);
              combinedLogs.push(...tail);
            }
          });

          payload.logs = combinedLogs;
          payload.logsTotal = selectedStreams.reduce((total, stream) => total + stream.total, 0);
          if (reportAppLogsFetchedAt) {
            payload.logsCapturedAt = new Date(reportAppLogsFetchedAt).toISOString();
          }
        } else if (reportAppLogs.length > 0) {
          payload.logs = reportAppLogs;
          payload.logsTotal = typeof reportAppLogsTotal === 'number' ? reportAppLogsTotal : reportAppLogs.length;
          if (reportAppLogsFetchedAt) {
            payload.logsCapturedAt = new Date(reportAppLogsFetchedAt).toISOString();
          }
        }
      }

      if (reportIncludeConsoleLogs && reportConsoleLogs.length > 0) {
        payload.consoleLogs = reportConsoleLogs.map(entry => entry.payload as ReportIssueConsoleLogEntry);
        payload.consoleLogsTotal = typeof reportConsoleLogsTotal === 'number'
          ? reportConsoleLogsTotal
          : reportConsoleLogs.length;
        if (reportConsoleLogsFetchedAt) {
          payload.consoleLogsCapturedAt = new Date(reportConsoleLogsFetchedAt).toISOString();
        }
      }

      if (reportIncludeNetworkRequests && reportNetworkEvents.length > 0) {
        payload.networkRequests = reportNetworkEvents.map(entry => entry.payload as ReportIssueNetworkEntry);
        payload.networkRequestsTotal = typeof reportNetworkTotal === 'number'
          ? reportNetworkTotal
          : reportNetworkEvents.length;
        if (reportNetworkFetchedAt) {
          payload.networkCapturedAt = new Date(reportNetworkFetchedAt).toISOString();
        }
      }

      const response = await appService.reportAppIssue(targetAppId, payload);

      const issueId = response.data?.issue_id;
      const issueUrl = response.data?.issue_url;
      setReportResult({
        issueId,
        issueUrl,
        message: response.message ?? 'Issue report sent successfully.',
      });
      setReportMessage('');
    } catch (error: unknown) {
      const fallbackMessage = (error as { message?: string })?.message ?? 'Failed to send issue report.';
      setReportError(fallbackMessage);
    } finally {
      setReportSubmitting(false);
    }
  }, [
    app?.id,
    app?.name,
    app?.scenario_name,
    activePreviewUrl,
    appId,
    canCaptureScreenshot,
    reportIncludeScreenshot,
    reportScreenshotData,
    reportIncludeAppLogs,
    reportAppLogStreams,
    reportAppLogSelections,
    reportAppLogsFetchedAt,
    reportAppLogs,
    reportAppLogsTotal,
    reportIncludeConsoleLogs,
    reportConsoleLogs,
    reportConsoleLogsFetchedAt,
    reportConsoleLogsTotal,
    reportIncludeNetworkRequests,
    reportNetworkEvents,
    reportNetworkFetchedAt,
    reportNetworkTotal,
    reportMessage,
  ]);

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

  const diagnosticsCheckedAt = useMemo(() => {
    if (!diagnosticsLastFetchedAt) {
      return null;
    }
    try {
      return new Date(diagnosticsLastFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [diagnosticsLastFetchedAt]);

  const diagnosticsViolations = useMemo(() => diagnosticsReport?.violations ?? [], [diagnosticsReport]);

  const diagnosticsRuleResults = useMemo(() => diagnosticsReport?.results ?? [], [diagnosticsReport]);

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

  const diagnosticsDescription = useMemo(() => {
    const runtimeFailures = (() => {
      if (!bridgeCompliance || bridgeCompliance.ok || !Array.isArray(bridgeCompliance.failures)) {
        return [] as string[];
      }
      const seen = new Set<string>();
      return bridgeCompliance.failures
        .map(code => BRIDGE_FAILURE_LABELS[code] ?? code)
        .filter(message => {
          const normalized = (message ?? '').toString().trim();
          if (!normalized || seen.has(normalized)) {
            return false;
          }
          seen.add(normalized);
          return true;
        });
    })();

    const hasDiagnosticsReport = Boolean(diagnosticsReport);
    const hasRuntimeIssues = runtimeFailures.length > 0;

    if (!hasDiagnosticsReport && !hasRuntimeIssues) {
      return '';
    }

    const hasRuleProblems = diagnosticsRuleResults.some((rule) => {
      const combinedWarnings = [rule.warning, ...(rule.warnings ?? [])]
        .map(entry => (entry ?? '').toString().trim())
        .filter(entry => entry.length > 0);
      return rule.violations.length > 0 || combinedWarnings.length > 0;
    });

    const hasScanFailure = diagnosticsScannedFileCount <= 0;
    const hasGlobalWarnings = diagnosticsWarnings.length > 0;

    if (!hasRuntimeIssues && !hasRuleProblems && !hasScanFailure && !hasGlobalWarnings) {
      return '';
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

    const formatTimestamp = (value: string | number | null | undefined) => {
      if (!value) {
        return null;
      }
      try {
        return new Date(value).toLocaleString();
      } catch {
        return null;
      }
    };

    const lines: string[] = [];
    lines.push(`Scenario-auditor diagnostics for ${scenarioSlug}`);

    if (diagnosticsReport?.checked_at) {
      const formatted = formatTimestamp(diagnosticsReport.checked_at);
      if (formatted) {
        lines.push(`Checked at ${formatted}.`);
      }
    }

    if (diagnosticsReport) {
      const scanMeta: string[] = [];
      if (diagnosticsScannedFileCount > 0) {
        scanMeta.push(`files scanned: ${diagnosticsScannedFileCount}`);
      } else {
        scanMeta.push('no files were scanned');
      }

      if (typeof diagnosticsReport.duration_ms === 'number' && Number.isFinite(diagnosticsReport.duration_ms)) {
        scanMeta.push(`duration: ${Math.max(0, Math.round(diagnosticsReport.duration_ms))} ms`);
      }

      if (scanMeta.length > 0) {
        lines.push(`Scan summary — ${scanMeta.join(', ')}.`);
      }

      if (diagnosticsWarnings.length > 0) {
        lines.push('Scenario-auditor warnings:');
        diagnosticsWarnings.forEach((warning) => {
          lines.push(`- ${warning}`);
        });
      }

      if (diagnosticsScannedFileCount <= 0) {
        lines.push('Scenario-auditor could not inspect the UI package. Ensure the scenario depends on @vrooli/iframe-bridge, exports a bootstrap entry, and includes UI/config files in rule targets.');
      }

      const ruleDetails: string[] = [];
      diagnosticsRuleResults.forEach((rule) => {
        const trimmedRuleId = (rule.rule_id ?? '').trim();
        const ruleNameCandidate = rule.name ?? trimmedRuleId;
        const trimmedRuleName = (ruleNameCandidate ?? '').toString().trim();
        const ruleName = trimmedRuleName.length > 0 ? trimmedRuleName : 'Diagnostics rule';

        const combinedWarnings = Array.from(new Set(
          [rule.warning, ...(rule.warnings ?? [])]
            .map(entry => (entry ?? '').toString().trim())
            .filter(entry => entry.length > 0),
        ));

        const hasRuleViolations = rule.violations.length > 0;
        const hasRuleWarnings = combinedWarnings.length > 0;

        if (!hasRuleViolations && !hasRuleWarnings && diagnosticsScannedFileCount > 0) {
          return;
        }

        const headerParts = [ruleName];
        if (trimmedRuleId) {
          headerParts.push(`(${trimmedRuleId})`);
        }

        const headerLine = headerParts.join(' ').trim();
        ruleDetails.push(`- ${headerLine}`);

        if (hasRuleViolations) {
          rule.violations.forEach((violation, violationIndex) => {
            const location = violation.file_path
              ? `${violation.file_path}${typeof violation.line === 'number' ? `:${violation.line}` : ''}`
              : null;
            const locationLabel = location ? ` (${location})` : '';
            const recommendation = (violation.recommendation ?? '').trim();
            const base = `${violationIndex + 1}. ${violation.title}${locationLabel} — ${violation.description || violation.type}`;
            ruleDetails.push(`    - ${base}`);
            if (recommendation) {
              ruleDetails.push(`      Recommendation: ${recommendation}`);
            }
          });
        }

        combinedWarnings.forEach((warning, warningIndex) => {
          ruleDetails.push(`    - Warning ${warningIndex + 1}: ${warning}`);
        });

        if (trimmedRuleId) {
          ruleDetails.push(`    - Re-run: scenario-auditor scan ${scenarioSlug} --rule ${trimmedRuleId} --wait --timeout 600`);
        }
      });

      if (ruleDetails.length > 0) {
        lines.push('Rule findings:');
        lines.push(...ruleDetails);
      }
    }

    if (hasRuntimeIssues) {
      lines.push('Runtime bridge diagnostics reported issues:');
      runtimeFailures.forEach((failure) => {
        lines.push(`- ${failure}`);
      });
      lines.push('Retest by reloading the preview in App Monitor and re-running diagnostics after the iframe bridge initializes.');
    }

    if (scenarioSlug) {
      lines.push(`Full scan: scenario-auditor scan ${scenarioSlug} --wait --timeout 600`);
    }

    return lines.join('\n');
  }, [
    app?.id,
    app?.scenario_name,
    appId,
    bridgeCompliance,
    diagnosticsReport,
    diagnosticsRuleResults,
    diagnosticsScannedFileCount,
    diagnosticsViolations,
    diagnosticsWarnings,
  ]);

  const handleApplyDiagnosticsDescription = useCallback(() => {
    if (!diagnosticsDescription) {
      return;
    }
    setReportMessage(previous => {
      const next = diagnosticsDescription;
      if (!next) {
        return previous;
      }

      const current = previous ?? '';
      const trimmedCurrent = current.trimEnd();
      if (!trimmedCurrent) {
        return next;
      }

      return `${trimmedCurrent}\n\n${next}`;
    });
    window.setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);
  }, [diagnosticsDescription]);

  const resetState = useCallback(() => {
    setReportSubmitting(false);
    setReportError(null);
    setReportResult(null);
    setReportAppLogs([]);
    setReportAppLogsTotal(null);
    setReportAppLogsError(null);
    setReportAppLogsExpanded(false);
    setReportAppLogsFetchedAt(null);
    setReportAppLogStreams([]);
    setReportAppLogSelections({});
    setReportIncludeAppLogs(true);
    setReportConsoleLogs([]);
    setReportConsoleLogsTotal(null);
    setReportConsoleLogsError(null);
    setReportConsoleLogsExpanded(false);
    setReportConsoleLogsFetchedAt(null);
    setReportIncludeConsoleLogs(true);
    setReportNetworkEvents([]);
    setReportNetworkTotal(null);
    setReportNetworkError(null);
    setReportNetworkExpanded(false);
    setReportNetworkFetchedAt(null);
    setReportIncludeNetworkRequests(true);
    reportAppLogsFetchedForRef.current = null;
    reportConsoleLogsFetchedForRef.current = null;
    reportNetworkFetchedForRef.current = null;
  }, []);

  useEffect(() => {
    if (!isOpen) {
      resetState();
      cleanupAfterDialogClose(canCaptureScreenshot);
      return;
    }

    setReportMessage('');
    setReportError(null);
    setReportResult(null);
    setReportAppLogs([]);
    setReportAppLogsTotal(null);
    setReportAppLogsError(null);
    setReportAppLogsExpanded(false);
    setReportAppLogsFetchedAt(null);
    setReportAppLogStreams([]);
    setReportAppLogSelections({});
    setReportIncludeAppLogs(true);
    setReportConsoleLogs([]);
    setReportConsoleLogsTotal(null);
    setReportConsoleLogsError(null);
    setReportConsoleLogsExpanded(false);
    setReportConsoleLogsFetchedAt(null);
    setReportIncludeConsoleLogs(true);
    setReportNetworkEvents([]);
    setReportNetworkTotal(null);
    setReportNetworkError(null);
    setReportNetworkExpanded(false);
    setReportNetworkFetchedAt(null);
    setReportIncludeNetworkRequests(true);
    reportAppLogsFetchedForRef.current = null;
    reportConsoleLogsFetchedForRef.current = null;
    reportNetworkFetchedForRef.current = null;
    prepareForDialogOpen(canCaptureScreenshot);

    void fetchReportAppLogs({ force: true });
    void fetchReportConsoleLogs({ force: true });
    void fetchReportNetworkEvents({ force: true });
    void refreshDiagnostics();
  }, [
    isOpen,
    canCaptureScreenshot,
    cleanupAfterDialogClose,
    prepareForDialogOpen,
    fetchReportAppLogs,
    fetchReportConsoleLogs,
    fetchReportNetworkEvents,
    refreshDiagnostics,
    resetState,
  ]);

  const logsTruncated = useMemo(() => (
    typeof reportAppLogsTotal === 'number' && reportAppLogsTotal > reportAppLogs.length
  ), [reportAppLogs.length, reportAppLogsTotal]);

  const formattedReportLogsTime = useMemo(() => {
    if (!reportAppLogsFetchedAt) {
      return null;
    }
    try {
      return new Date(reportAppLogsFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportAppLogsFetchedAt]);

  const consoleLogsTruncated = useMemo(() => (
    typeof reportConsoleLogsTotal === 'number' && reportConsoleLogsTotal > reportConsoleLogs.length
  ), [reportConsoleLogs.length, reportConsoleLogsTotal]);

  const formattedConsoleLogsTime = useMemo(() => {
    if (!reportConsoleLogsFetchedAt) {
      return null;
    }
    try {
      return new Date(reportConsoleLogsFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportConsoleLogsFetchedAt]);

  const networkEventsTruncated = useMemo(() => (
    typeof reportNetworkTotal === 'number' && reportNetworkTotal > reportNetworkEvents.length
  ), [reportNetworkEvents.length, reportNetworkTotal]);

  const selectedReportLogStreams = useMemo(() => (
    reportAppLogStreams.filter(stream => reportAppLogSelections[stream.key] !== false)
  ), [reportAppLogSelections, reportAppLogStreams]);

  const selectedReportLogCount = selectedReportLogStreams.length;
  const totalReportLogStreamCount = reportAppLogStreams.length;

  const formattedNetworkCapturedAt = useMemo(() => {
    if (!reportNetworkFetchedAt) {
      return null;
    }
    try {
      return new Date(reportNetworkFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportNetworkFetchedAt]);

  const bridgeComplianceCheckedAt = useMemo(() => {
    if (!bridgeCompliance) {
      return null;
    }

    try {
      return new Date(bridgeCompliance.checkedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [bridgeCompliance]);

  const bridgeComplianceFailures = useMemo(() => {
    if (!bridgeCompliance || bridgeCompliance.ok) {
      return [] as string[];
    }

    const normalized = Array.isArray(bridgeCompliance.failures)
      ? bridgeCompliance.failures
      : [];

    const seen = new Set<string>();
    return normalized
      .map(code => BRIDGE_FAILURE_LABELS[code] ?? code)
      .filter((message): message is string => {
        if (!message || seen.has(message)) {
          return false;
        }
        seen.add(message);
        return true;
      });
  }, [bridgeCompliance]);

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
    if (diagnosticsStatus === 'success' && diagnosticsEvaluation === 'pass') {
      return 'pass';
    }
    return 'idle';
  }, [diagnosticsLoading, diagnosticsStatus, diagnosticsEvaluation]);

  const showDiagnosticsSetDescription = Boolean(diagnosticsDescription);

  return {
    textareaRef,
    form: {
      message: reportMessage,
      submitting: reportSubmitting,
      error: reportError,
      result: reportResult,
      handleMessageChange: handleReportMessageChange,
      handleMessageKeyDown: handleReportMessageKeyDown,
      handleSubmit: handleSubmitReport,
    },
    modal: {
      handleOverlayClick,
      handleDialogClose,
    },
    logs: {
      includeAppLogs: reportIncludeAppLogs,
      setIncludeAppLogs: setReportIncludeAppLogs,
      streams: reportAppLogStreams,
      selections: reportAppLogSelections,
      toggleStream: handleReportLogStreamToggle,
      expanded: reportAppLogsExpanded,
      toggleExpanded: () => setReportAppLogsExpanded(prev => !prev),
      loading: reportAppLogsLoading,
      error: reportAppLogsError,
      truncated: logsTruncated,
      formattedCapturedAt: formattedReportLogsTime,
      logs: reportAppLogs,
      total: reportAppLogsTotal,
      fetch: handleRefreshReportAppLogs,
      selectedCount: selectedReportLogCount,
      totalStreamCount: totalReportLogStreamCount,
    },
    consoleLogs: {
      includeConsoleLogs: reportIncludeConsoleLogs,
      setIncludeConsoleLogs: setReportIncludeConsoleLogs,
      entries: reportConsoleLogs,
      expanded: reportConsoleLogsExpanded,
      toggleExpanded: () => setReportConsoleLogsExpanded(prev => !prev),
      loading: reportConsoleLogsLoading,
      error: reportConsoleLogsError,
      truncated: consoleLogsTruncated,
      formattedCapturedAt: formattedConsoleLogsTime,
      total: reportConsoleLogsTotal,
      fetch: handleRefreshReportConsoleLogs,
    },
    network: {
      includeNetworkRequests: reportIncludeNetworkRequests,
      setIncludeNetworkRequests: setReportIncludeNetworkRequests,
      events: reportNetworkEvents,
      expanded: reportNetworkExpanded,
      toggleExpanded: () => setReportNetworkExpanded(prev => !prev),
      loading: reportNetworkLoading,
      error: reportNetworkError,
      truncated: networkEventsTruncated,
      formattedCapturedAt: formattedNetworkCapturedAt,
      total: reportNetworkTotal,
      fetch: handleRefreshReportNetworkEvents,
    },
    diagnostics: {
      diagnosticsState,
      diagnosticsLoading,
      diagnosticsWarning: diagnosticsWarning ?? null,
      diagnosticsError: diagnosticsError ?? null,
      diagnosticsViolations,
      diagnosticsScannedFileCount,
      diagnosticsCheckedAt,
      diagnosticsDescription,
      showDiagnosticsSetDescription,
      handleApplyDiagnosticsDescription,
      refreshDiagnostics,
      bridgeCompliance,
      bridgeComplianceCheckedAt,
      bridgeComplianceFailures,
      diagnosticsRuleResults,
      diagnosticsWarnings,
    },
    screenshot: {
      reportIncludeScreenshot,
      reportScreenshotData,
      reportScreenshotLoading,
      reportScreenshotError,
      reportScreenshotOriginalDimensions,
      reportSelectionRect,
      reportScreenshotClip,
      reportScreenshotInfo,
      reportScreenshotCountdown,
      assignScreenshotContainerRef,
      reportScreenshotContainerHandlers,
      selectionDimensionLabel,
      handleReportIncludeScreenshotChange,
      handleRetryScreenshotCapture,
      handleResetScreenshotSelection,
      handleDelayedScreenshotCapture,
      handleScreenshotImageLoad,
    },
    bridgeState,
    reportResult,
    reportSubmitting,
    reportError,
    reportIncludeScreenshot,
    resetOnClose: resetState,
  };
};

export type {
  ReportIssueStateResult,
  ReportFormState,
  ReportLogsState,
  ReportConsoleLogsState,
  ReportNetworkState,
  ReportDiagnosticsState,
  ReportScreenshotState,
  DiagnosticsViolation,
  BridgePreviewState,
};

export {
  REPORT_APP_LOGS_MAX_LINES,
  REPORT_CONSOLE_LOGS_MAX_LINES,
  REPORT_NETWORK_MAX_EVENTS,
};

export default useReportIssueState;
