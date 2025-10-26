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
  BridgeScreenshotMode,
  BridgeScreenshotOptions,
} from '@vrooli/iframe-bridge';

import useIframeBridgeDiagnostics from '@/hooks/useIframeBridgeDiagnostics';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { useReportScreenshot } from '@/hooks/useReportScreenshot';
import { appService, healthService } from '@/services/api';
import type {
  ReportIssueConsoleLogEntry,
  ReportIssueNetworkEntry,
  ReportIssuePayload,
  ReportIssueHealthCheckEntry,
  ReportIssueAppStatusSeverity,
  ScenarioIssueSummary,
  ReportIssueCapturePayload,
} from '@/services/api';
import { logger } from '@/services/logger';
import type { App, BridgeRuleReport } from '@/types';
import { useScenarioEngagementStore } from '@/state/scenarioEngagementStore';
import { useScenarioIssuesStore } from '@/state/scenarioIssuesStore';
import { useSnackPublisher } from '@/notifications/useSnackPublisher';
import type { SnackPublishOptions, SnackUpdateOptions } from '@/notifications/snackBus';
import {
  estimateReportPayloadSize,
  validatePayloadSize,
  compressBase64Image,
  formatBytes,
} from '@/utils/payloadSize';

import { toConsoleEntry, toNetworkEntry } from './reportFormatters';
import type {
  ReportAppLogStream,
  ReportConsoleEntry,
  ReportNetworkEntry,
  ReportHealthCheckEntry,
  ReportAppStatusSnapshot,
  ReportElementCapture,
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

interface ReportHealthChecksState {
  includeHealthChecks: boolean;
  setIncludeHealthChecks: (value: boolean) => void;
  entries: ReportHealthCheckEntry[];
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  formattedCapturedAt: string | null;
  total: number | null;
  fetch: () => void;
}

interface ReportAppStatusState {
  includeAppStatus: boolean;
  setIncludeAppStatus: (value: boolean) => void;
  snapshot: ReportAppStatusSnapshot | null;
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  formattedCapturedAt: string | null;
  fetch: () => void;
}

type ExistingIssuesStateInternal = {
  status: 'idle' | 'loading' | 'ready' | 'error';
  issues: ScenarioIssueSummary[];
  openCount: number;
  activeCount: number;
  totalCount: number;
  trackerUrl: string | null;
  lastFetched: string | null;
  stale: boolean;
  error: string | null;
  fromCache: boolean;
  appId: string | null;
};

interface ReportExistingIssuesState {
  status: ExistingIssuesStateInternal['status'];
  issues: ScenarioIssueSummary[];
  openCount: number;
  activeCount: number;
  totalCount: number;
  trackerUrl: string | null;
  lastFetched: string | null;
  stale: boolean;
  fromCache: boolean;
  error: string | null;
  refresh: () => void;
}

const createExistingIssuesState = (
  overrides: Partial<ExistingIssuesStateInternal> = {},
): ExistingIssuesStateInternal => ({
  status: 'idle',
  issues: [],
  openCount: 0,
  activeCount: 0,
  totalCount: 0,
  trackerUrl: null,
  lastFetched: null,
  stale: false,
  error: null,
  fromCache: false,
  appId: null,
  ...overrides,
});

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
    handleDismiss: () => void;
    handleReset: () => void;
  };
  logs: ReportLogsState;
  consoleLogs: ReportConsoleLogsState;
  network: ReportNetworkState;
  health: ReportHealthChecksState;
  status: ReportAppStatusState;
  existingIssues: ReportExistingIssuesState;
  diagnostics: ReportDiagnosticsState;
  screenshot: ReportScreenshotState;
  bridgeState: BridgePreviewState;
  reportSubmitting: boolean;
  reportError: string | null;
  reportIncludeScreenshot: boolean;
  diagnosticsSummaryIncluded: boolean;
  setIncludeDiagnosticsSummary: (value: boolean) => void;
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
  elementCaptures,
  onElementCapturesReset,
  onPrimaryCaptureDraftChange,
}: UseReportIssueStateParams): ReportIssueStateResult => {
  const [reportMessage, setReportMessage] = useState('');
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
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
  const [reportHealthChecks, setReportHealthChecks] = useState<ReportHealthCheckEntry[]>([]);
  const [reportHealthChecksLoading, setReportHealthChecksLoading] = useState(false);
  const [reportHealthChecksError, setReportHealthChecksError] = useState<string | null>(null);
  const [reportHealthChecksExpanded, setReportHealthChecksExpanded] = useState(false);
  const [reportHealthChecksFetchedAt, setReportHealthChecksFetchedAt] = useState<number | null>(null);
  const [reportHealthChecksTotal, setReportHealthChecksTotal] = useState<number | null>(null);
  const [reportIncludeHealthChecks, setReportIncludeHealthChecks] = useState(true);
  const [reportAppStatusSnapshot, setReportAppStatusSnapshot] = useState<ReportAppStatusSnapshot | null>(null);
  const [reportAppStatusLoading, setReportAppStatusLoading] = useState(false);
  const [reportAppStatusError, setReportAppStatusError] = useState<string | null>(null);
  const [reportAppStatusExpanded, setReportAppStatusExpanded] = useState(false);
  const [reportAppStatusFetchedAt, setReportAppStatusFetchedAt] = useState<number | null>(null);
  const [reportIncludeAppStatus, setReportIncludeAppStatus] = useState(true);
  const [reportIncludeDiagnostics, setReportIncludeDiagnostics] = useState(false);
  const [reportIncludeDiagnosticsManuallySet, setReportIncludeDiagnosticsManuallySet] = useState(false);
  const [existingIssuesState, setExistingIssuesState] = useState<ExistingIssuesStateInternal>(() => createExistingIssuesState());
  const markScenarioIssueCreated = useScenarioEngagementStore(state => state.markIssueCreated);
  const flagScenarioIssueReported = useScenarioIssuesStore(state => state.flagIssueReported);
  const snackPublisher = useSnackPublisher();

  const reportAppLogsFetchedForRef = useRef<string | null>(null);
  const reportConsoleLogsFetchedForRef = useRef<string | null>(null);
  const reportNetworkFetchedForRef = useRef<string | null>(null);
  const reportHealthChecksFetchedForRef = useRef<string | null>(null);
  const reportAppStatusFetchedForRef = useRef<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  const [shouldResetOnNextOpen, setShouldResetOnNextOpen] = useState(true);
  const lastResolvedAppIdRef = useRef<string | null>(null);
  const resolvedAppId = useMemo(() => {
    const candidate = (app?.id ?? appId ?? '').trim();
    return candidate === '' ? null : candidate;
  }, [app?.id, appId]);

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

  useEffect(() => {
    if (!onPrimaryCaptureDraftChange) {
      return;
    }

    const hasCaptureDraft = Boolean(
      reportIncludeScreenshot
      && canCaptureScreenshot
      && reportScreenshotData
      && !reportScreenshotLoading,
    );

    onPrimaryCaptureDraftChange(hasCaptureDraft);
  }, [
    onPrimaryCaptureDraftChange,
    reportIncludeScreenshot,
    canCaptureScreenshot,
    reportScreenshotData,
    reportScreenshotLoading,
  ]);

  const resetState = useCallback(() => {
    setReportSubmitting(false);
    setReportError(null);
    setReportMessage('');
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
    setReportHealthChecks([]);
    setReportHealthChecksLoading(false);
    setReportHealthChecksError(null);
    setReportHealthChecksExpanded(false);
    setReportHealthChecksFetchedAt(null);
    setReportHealthChecksTotal(null);
    setReportIncludeHealthChecks(true);
    setReportAppStatusSnapshot(null);
    setReportAppStatusLoading(false);
    setReportAppStatusError(null);
    setReportAppStatusExpanded(false);
    setReportAppStatusFetchedAt(null);
    setReportIncludeAppStatus(true);
    setReportIncludeDiagnostics(false);
    setReportIncludeDiagnosticsManuallySet(false);
    setExistingIssuesState(createExistingIssuesState());
    reportAppLogsFetchedForRef.current = null;
    reportConsoleLogsFetchedForRef.current = null;
    reportNetworkFetchedForRef.current = null;
    reportHealthChecksFetchedForRef.current = null;
    reportAppStatusFetchedForRef.current = null;
  }, []);

  const resolveReportLogIdentifier = useCallback(() => {
    const candidates = [app?.scenario_name, app?.id, appId]
      .map(value => (typeof value === 'string' ? value.trim() : ''))
      .filter(Boolean);

    if (candidates.length === 0) {
      return null;
    }

    return candidates[0];
  }, [app, appId]);

  const fetchExistingIssues = useCallback(async () => {
    const targetAppId = (app?.id ?? appId ?? '').trim();
    if (!targetAppId) {
      setExistingIssuesState(createExistingIssuesState());
      return;
    }

    setExistingIssuesState(prev => {
      const shouldReset = prev.appId !== targetAppId;
      if (shouldReset) {
        return createExistingIssuesState({
          status: 'loading',
          appId: targetAppId,
        });
      }

      return {
        ...prev,
        status: 'loading',
        error: null,
        appId: targetAppId,
      };
    });

    const summary = await appService.getScenarioIssues(targetAppId);
    if (!summary) {
      setExistingIssuesState(prev => {
        if (prev.issues.length > 0) {
          return {
            ...prev,
            status: 'ready',
            stale: true,
            error: 'Unable to refresh existing issues.',
            appId: targetAppId,
          };
        }

        return createExistingIssuesState({
          status: 'error',
          error: 'Unable to load existing issues.',
          appId: targetAppId,
        });
      });
      return;
    }

    const issues = Array.isArray(summary.issues) ? summary.issues : [];
    const openCount = typeof summary.open_count === 'number'
      ? summary.open_count
      : issues.reduce((count, issue) => (issue.status?.toLowerCase() === 'open' ? count + 1 : count), 0);
    const activeCount = typeof summary.active_count === 'number'
      ? summary.active_count
      : issues.reduce((count, issue) => (issue.status?.toLowerCase() === 'active' ? count + 1 : count), 0);
    const totalCount = typeof summary.total_count === 'number' ? summary.total_count : issues.length;

    setExistingIssuesState(createExistingIssuesState({
      status: 'ready',
      issues,
      openCount,
      activeCount,
      totalCount,
      trackerUrl: summary.tracker_url ?? null,
      lastFetched: summary.last_fetched ?? null,
      stale: Boolean(summary.stale),
      fromCache: Boolean(summary.from_cache),
      error: null,
      appId: targetAppId,
    }));
  }, [app?.id, appId]);

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

  const fetchReportHealthChecks = useCallback(async (options?: { force?: boolean }) => {
    if (reportHealthChecksLoading && !options?.force) {
      return;
    }

    const targetAppId = (app?.id ?? appId ?? '').trim();
    if (!targetAppId) {
      setReportHealthChecks([]);
      setReportHealthChecksTotal(null);
      setReportHealthChecksError('Preview app unavailable.');
      setReportHealthChecksFetchedAt(null);
      reportHealthChecksFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = targetAppId.toLowerCase();
    if (!options?.force && reportHealthChecksFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportHealthChecksLoading(true);
    setReportHealthChecksError(null);

    try {
      const result = await healthService.checkPreviewHealth(targetAppId);
      const entries = result.checks.map((entry, index): ReportHealthCheckEntry => {
        const rawId = typeof entry.id === 'string' ? entry.id.trim() : '';
        const id = rawId || `health-${index}`;
        const rawName = typeof entry.name === 'string' ? entry.name.trim() : '';
        const status: 'pass' | 'warn' | 'fail' = entry.status === 'pass'
          ? 'pass'
          : entry.status === 'warn'
            ? 'warn'
            : 'fail';
        const latency = typeof entry.latencyMs === 'number' && Number.isFinite(entry.latencyMs)
          ? Math.max(0, Math.round(entry.latencyMs))
          : null;

        return {
          id,
          name: rawName || id,
          status,
          endpoint: entry.endpoint ?? null,
          latencyMs: latency,
          message: entry.message ?? null,
          code: entry.code ?? null,
          response: entry.response ?? null,
        };
      });

      setReportHealthChecks(entries);
      setReportHealthChecksTotal(entries.length);

      let capturedAtMs: number | null = null;
      if (result.capturedAt) {
        const parsed = Date.parse(result.capturedAt);
        if (!Number.isNaN(parsed)) {
          capturedAtMs = parsed;
        }
      }
      const nextFetchedAt = entries.length > 0
        ? capturedAtMs ?? Date.now()
        : null;
      setReportHealthChecksFetchedAt(nextFetchedAt);
      reportHealthChecksFetchedForRef.current = normalizedIdentifier;

      const combinedErrors = result.errors.filter(message => typeof message === 'string' && message.trim().length > 0);
      const errorMessage = combinedErrors.length > 0 ? combinedErrors.join(' ') : null;

      if (entries.length === 0) {
        setReportHealthChecksError(errorMessage ?? 'No health checks available.');
      } else {
        setReportHealthChecksError(errorMessage ?? null);
      }
    } catch (error) {
      logger.warn('Failed to gather preview health checks for issue report', error);
      setReportHealthChecks([]);
      setReportHealthChecksTotal(null);
      setReportHealthChecksFetchedAt(null);
      reportHealthChecksFetchedForRef.current = null;
      const message = error instanceof Error && error.message
        ? error.message
        : 'Unable to gather health checks.';
      setReportHealthChecksError(message);
    } finally {
      setReportHealthChecksLoading(false);
    }
  }, [app?.id, appId, reportHealthChecksLoading]);

  const fetchReportAppStatus = useCallback(async (options?: { force?: boolean }) => {
    if (reportAppStatusLoading && !options?.force) {
      return;
    }

    const targetAppId = (app?.id ?? appId ?? '').trim();
    if (!targetAppId) {
      setReportAppStatusSnapshot(null);
      setReportAppStatusError('Preview app unavailable.');
      setReportAppStatusFetchedAt(null);
      reportAppStatusFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = targetAppId.toLowerCase();
    if (!options?.force && reportAppStatusFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportAppStatusLoading(true);
    setReportAppStatusError(null);

    try {
      const result = await appService.getAppStatusSnapshot(targetAppId);
      if (!result) {
        setReportAppStatusSnapshot(null);
        setReportAppStatusError('Unable to retrieve scenario status.');
        setReportAppStatusFetchedAt(null);
        reportAppStatusFetchedForRef.current = null;
        return;
      }

      const runtimeValue = typeof result.runtime === 'string' ? result.runtime.trim() : '';
      const snapshot: ReportAppStatusSnapshot = {
        appId: (result.appId ?? targetAppId).trim() || targetAppId,
        scenario: (result.scenario ?? targetAppId).trim() || targetAppId,
        statusLabel: (result.statusLabel ?? 'UNKNOWN').trim() || 'UNKNOWN',
        severity: (result.severity ?? 'warn') as ReportIssueAppStatusSeverity,
        runtime: runtimeValue || null,
        processCount: typeof result.processCount === 'number' ? result.processCount : null,
        details: Array.isArray(result.details)
          ? result.details.map(line => (typeof line === 'string' ? line : String(line))).filter(detail => detail.trim().length > 0)
          : [],
        capturedAt: result.capturedAt ?? null,
      };

      setReportAppStatusSnapshot(snapshot);
      const capturedAtMs = snapshot.capturedAt ? Date.parse(snapshot.capturedAt) : Number.NaN;
      const nextFetchedAt = Number.isNaN(capturedAtMs) ? Date.now() : capturedAtMs;
      setReportAppStatusFetchedAt(nextFetchedAt);
      setReportAppStatusError(null);
      reportAppStatusFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to gather scenario status for issue report', error);
      setReportAppStatusSnapshot(null);
      setReportAppStatusError('Unable to gather scenario status.');
      setReportAppStatusFetchedAt(null);
      reportAppStatusFetchedForRef.current = null;
    } finally {
      setReportAppStatusLoading(false);
    }
  }, [app?.id, appId, reportAppStatusLoading]);

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

  const handleRefreshReportHealthChecks = useCallback(() => {
    void fetchReportHealthChecks({ force: true });
  }, [fetchReportHealthChecks]);

  const handleRefreshReportAppStatus = useCallback(() => {
    void fetchReportAppStatus({ force: true });
  }, [fetchReportAppStatus]);

  const handleRefreshExistingIssues = useCallback(() => {
    void fetchExistingIssues();
  }, [fetchExistingIssues]);

  const handleReportMessageChange = useCallback((event: ChangeEvent<HTMLTextAreaElement>) => {
    setReportMessage(event.target.value);
  }, []);

  const handleIncludeDiagnosticsSummaryChange = useCallback((checked: boolean) => {
    setReportIncludeDiagnostics(checked);
    setReportIncludeDiagnosticsManuallySet(true);
  }, []);

  const handleReportMessageKeyDown = useCallback((event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Escape') {
      event.stopPropagation();
    }
  }, []);

  const handleDialogDismiss = useCallback(() => {
    setShouldResetOnNextOpen(false);
    onClose();
  }, [onClose]);

  const handleDialogReset = useCallback(() => {
    setShouldResetOnNextOpen(true);
    resetState();
    cleanupAfterDialogClose(canCaptureScreenshot);
    onElementCapturesReset();
    if (onPrimaryCaptureDraftChange) {
      onPrimaryCaptureDraftChange(false);
    }
    onClose();
  }, [
    canCaptureScreenshot,
    cleanupAfterDialogClose,
    onClose,
    onElementCapturesReset,
    onPrimaryCaptureDraftChange,
    resetState,
  ]);

  const handleSubmitReport = useCallback(async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmed = reportMessage.trim();
    const captureNotes = elementCaptures
      .map((capture, index) => {
        const note = capture.note.trim();
        if (!note) {
          return null;
        }

        const selectorLabel = capture.metadata.selector?.trim();
        const descriptiveLabel = capture.metadata.label?.trim();
        const tagLabel = capture.metadata.tagName ? `<${capture.metadata.tagName}>` : null;
        const label = selectorLabel || descriptiveLabel || tagLabel || `Element capture ${index + 1}`;
        return `- ${label}: ${note}`;
      })
      .filter((entry): entry is string => Boolean(entry));

    const hasPrimaryDescription = trimmed.length > 0;
    const includeDiagnosticsSummary = diagnosticsSummaryIncluded;
    const hasCaptureNotes = captureNotes.length > 0;

    if (!hasPrimaryDescription && !hasCaptureNotes && !includeDiagnosticsSummary) {
      setReportError('Add a description, include diagnostics, or add capture notes before sending.');
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
      const sections: string[] = [];
      if (hasPrimaryDescription) {
        sections.push(trimmed);
      }

      if (includeDiagnosticsSummary && diagnosticsDescription) {
        sections.push(['### Diagnostics Summary', '', diagnosticsDescription].join('\n'));
      }

      if (hasCaptureNotes) {
        sections.push(['### Element Capture Notes', '', ...captureNotes].join('\n'));
      }

      const finalMessage = sections.join('\n\n').trim();
      if (!finalMessage) {
        setReportError('Unable to prepare report contents.');
        return;
      }

      // Compress element captures if needed to reduce payload size
      const capturePayloads: ReportIssueCapturePayload[] = await Promise.all(
        elementCaptures.map(async (capture, index) => {
          const normalizedClasses = (capture.metadata.classes ?? []).filter(Boolean);
          const createdAtIso = Number.isFinite(capture.createdAt)
            ? new Date(capture.createdAt).toISOString()
            : null;

          // Compress if image is large (>500 KiB estimated)
          let imageData = capture.data;
          const estimatedSize = imageData.length * 0.75; // Rough estimate of decoded size
          if (estimatedSize > 512 * 1024) {
            try {
              logger.info(`Compressing large element capture ${index + 1} (estimated ${formatBytes(estimatedSize)})`);
              imageData = await compressBase64Image(imageData, 1600, 1200, 0.8);
            } catch (error) {
              logger.warn(`Failed to compress element capture ${index + 1}, using original`, error);
            }
          }

          return {
            id: capture.id || `element-${index + 1}`,
            type: 'element',
            width: capture.width,
            height: capture.height,
            data: imageData,
            note: capture.note.trim() || null,
            selector: capture.metadata.selector ?? null,
            tagName: capture.metadata.tagName ?? null,
            elementId: capture.metadata.elementId ?? null,
            classes: normalizedClasses.length > 0 ? normalizedClasses : null,
            label: capture.metadata.label ?? null,
            ariaDescription: capture.metadata.ariaDescription ?? null,
            title: capture.metadata.title ?? null,
            role: capture.metadata.role ?? null,
            text: capture.metadata.text ?? null,
            boundingBox: capture.metadata.boundingBox ?? null,
            clip: capture.clip ?? null,
            mode: capture.mode ?? null,
            filename: capture.filename ?? null,
            createdAt: createdAtIso,
          } satisfies ReportIssueCapturePayload;
        }),
      );

      if (includeScreenshot && reportScreenshotData) {
        // Compress primary screenshot if large
        let primaryScreenshotData = reportScreenshotData;
        const estimatedSize = reportScreenshotData.length * 0.75;
        if (estimatedSize > 512 * 1024) {
          try {
            logger.info(`Compressing primary screenshot (estimated ${formatBytes(estimatedSize)})`);
            primaryScreenshotData = await compressBase64Image(reportScreenshotData, 1920, 1080, 0.85);
          } catch (error) {
            logger.warn('Failed to compress primary screenshot, using original', error);
          }
        }

        capturePayloads.unshift({
          id: 'page-capture',
          type: 'page',
          width: reportScreenshotOriginalDimensions?.width ?? 0,
          height: reportScreenshotOriginalDimensions?.height ?? 0,
          data: primaryScreenshotData,
          note: null,
          selector: null,
          tagName: null,
          elementId: null,
          classes: null,
          label: 'Preview',
          ariaDescription: null,
          title: null,
          role: null,
          text: null,
          boundingBox: null,
          clip: reportScreenshotClip ?? null,
          mode: reportScreenshotClip ? 'clip' : 'full',
          filename: null,
          createdAt: new Date().toISOString(),
        } satisfies ReportIssueCapturePayload);
      }

      const payload: ReportIssuePayload = {
        message: finalMessage,
        primaryDescription: hasPrimaryDescription ? trimmed : null,
        includeDiagnosticsSummary,
        includeScreenshot,
        previewUrl: activePreviewUrl || null,
        appName: app?.name ?? null,
        scenarioName: app?.scenario_name ?? null,
        source: 'app-monitor',
        screenshotData: includeScreenshot ? reportScreenshotData ?? null : null,
      };

      if (capturePayloads.length > 0) {
        payload.captures = capturePayloads;
      }

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

      if (reportIncludeHealthChecks) {
        const healthEntries: ReportIssueHealthCheckEntry[] = reportHealthChecks.map((entry) => ({
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
        const totalHealthChecks = typeof reportHealthChecksTotal === 'number'
          ? reportHealthChecksTotal
          : healthEntries.length;
        if (totalHealthChecks > 0) {
          payload.healthChecksTotal = totalHealthChecks;
        }
        if (reportHealthChecksFetchedAt) {
          payload.healthChecksCapturedAt = new Date(reportHealthChecksFetchedAt).toISOString();
        }
      }

      if (reportIncludeAppStatus && reportAppStatusSnapshot && reportAppStatusSnapshot.details.length > 0) {
        payload.appStatusLines = reportAppStatusSnapshot.details;
        payload.appStatusLabel = reportAppStatusSnapshot.statusLabel;
        payload.appStatusSeverity = reportAppStatusSnapshot.severity;
        const capturedSource = reportAppStatusSnapshot.capturedAt
          ?? (reportAppStatusFetchedAt ? new Date(reportAppStatusFetchedAt).toISOString() : null);
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
      cleanupAfterDialogClose(canCaptureScreenshot);
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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- diagnosticsDescription and diagnosticsSummaryIncluded are computed memoized values defined later in the file
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
    reportIncludeHealthChecks,
    reportHealthChecks,
    reportHealthChecksFetchedAt,
    reportHealthChecksTotal,
    reportIncludeAppStatus,
    reportAppStatusSnapshot,
    reportAppStatusFetchedAt,
    elementCaptures,
    reportScreenshotClip,
    reportScreenshotOriginalDimensions,
    flagScenarioIssueReported,
    markScenarioIssueCreated,
    reportMessage,
    cleanupAfterDialogClose,
    isOpen,
    onClose,
    onElementCapturesReset,
    onPrimaryCaptureDraftChange,
    resetState,
    setShouldResetOnNextOpen,
    snackPublisher,
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
    diagnosticsWarnings,
  ]);

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

  useEffect(() => {
    if (!isOpen) {
      if (shouldResetOnNextOpen) {
        cleanupAfterDialogClose(canCaptureScreenshot);
      }
      return;
    }

    if (!shouldResetOnNextOpen) {
      return;
    }

    resetState();
    prepareForDialogOpen(canCaptureScreenshot);

    void fetchReportAppLogs({ force: true });
    void fetchReportConsoleLogs({ force: true });
    void fetchReportNetworkEvents({ force: true });
    void fetchReportHealthChecks({ force: true });
    void fetchReportAppStatus({ force: true });
    void refreshDiagnostics();
    setShouldResetOnNextOpen(false);
  }, [
    isOpen,
    shouldResetOnNextOpen,
    canCaptureScreenshot,
    cleanupAfterDialogClose,
    prepareForDialogOpen,
    fetchReportAppLogs,
    fetchReportConsoleLogs,
    fetchReportNetworkEvents,
    fetchReportHealthChecks,
    fetchReportAppStatus,
    refreshDiagnostics,
    resetState,
  ]);

  useEffect(() => {
    if (lastResolvedAppIdRef.current === resolvedAppId) {
      return;
    }
    lastResolvedAppIdRef.current = resolvedAppId;
    setShouldResetOnNextOpen(true);
  }, [resolvedAppId]);

  useEffect(() => {
    if (!isOpen) {
      return;
    }
    void fetchExistingIssues();
  }, [fetchExistingIssues, isOpen]);

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

  const formattedHealthChecksTime = useMemo(() => {
    if (!reportHealthChecksFetchedAt) {
      return null;
    }
    try {
      return new Date(reportHealthChecksFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportHealthChecksFetchedAt]);

  const formattedAppStatusTime = useMemo(() => {
    if (reportAppStatusSnapshot?.capturedAt) {
      try {
        return new Date(reportAppStatusSnapshot.capturedAt).toLocaleTimeString();
      } catch {
        // Fall back to fetched-at timestamp if parsing fails.
      }
    }
    if (!reportAppStatusFetchedAt) {
      return null;
    }
    try {
      return new Date(reportAppStatusFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportAppStatusFetchedAt, reportAppStatusSnapshot?.capturedAt]);

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
    health: {
      includeHealthChecks: reportIncludeHealthChecks,
      setIncludeHealthChecks: setReportIncludeHealthChecks,
      entries: reportHealthChecks,
      expanded: reportHealthChecksExpanded,
      toggleExpanded: () => setReportHealthChecksExpanded(prev => !prev),
      loading: reportHealthChecksLoading,
      error: reportHealthChecksError,
      formattedCapturedAt: formattedHealthChecksTime,
      total: reportHealthChecksTotal,
      fetch: handleRefreshReportHealthChecks,
    },
    status: {
      includeAppStatus: reportIncludeAppStatus,
      setIncludeAppStatus: setReportIncludeAppStatus,
      snapshot: reportAppStatusSnapshot,
      expanded: reportAppStatusExpanded,
      toggleExpanded: () => setReportAppStatusExpanded(prev => !prev),
      loading: reportAppStatusLoading,
      error: reportAppStatusError,
      formattedCapturedAt: formattedAppStatusTime,
      fetch: handleRefreshReportAppStatus,
    },
    existingIssues: {
      status: existingIssuesState.status,
      issues: existingIssuesState.issues,
      openCount: existingIssuesState.openCount,
      activeCount: existingIssuesState.activeCount,
      totalCount: existingIssuesState.totalCount,
      trackerUrl: existingIssuesState.trackerUrl,
      lastFetched: existingIssuesState.lastFetched,
      stale: existingIssuesState.stale,
      fromCache: existingIssuesState.fromCache,
      error: existingIssuesState.error,
      refresh: handleRefreshExistingIssues,
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
    reportSubmitting,
    reportError,
    reportIncludeScreenshot,
    diagnosticsSummaryIncluded,
    setIncludeDiagnosticsSummary: handleIncludeDiagnosticsSummaryChange,
    resetOnClose: resetState,
  };
};

export type {
  ReportIssueStateResult,
  ReportFormState,
  ReportLogsState,
  ReportConsoleLogsState,
  ReportNetworkState,
  ReportHealthChecksState,
  ReportAppStatusState,
  ReportDiagnosticsState,
  ReportScreenshotState,
  DiagnosticsViolation,
  BridgePreviewState,
  ReportExistingIssuesState,
};

export {
  REPORT_APP_LOGS_MAX_LINES,
  REPORT_CONSOLE_LOGS_MAX_LINES,
  REPORT_NETWORK_MAX_EVENTS,
};

export default useReportIssueState;
