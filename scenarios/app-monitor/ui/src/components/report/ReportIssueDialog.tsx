import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChangeEvent, FormEvent, KeyboardEvent, MouseEvent } from 'react';
import clsx from 'clsx';
import {
  AlertTriangle,
  Bug,
  CheckCircle2,
  Eye,
  EyeOff,
  ExternalLink,
  Loader2,
  RefreshCw,
  X,
} from 'lucide-react';
import type {
  BridgeLogEvent,
  BridgeLogLevel,
  BridgeLogStreamState,
  BridgeNetworkEvent,
  BridgeNetworkStreamState,
} from '@vrooli/iframe-bridge';
import { appService } from '@/services/api';
import type { ReportIssueConsoleLogEntry, ReportIssueNetworkEntry, ReportIssuePayload } from '@/services/api';
import { logger } from '@/services/logger';
import type { App } from '@/types';
import { useReportScreenshot } from '@/hooks/useReportScreenshot';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import useIframeBridgeDiagnostics from '@/hooks/useIframeBridgeDiagnostics';

interface ReportConsoleEntry {
  display: string;
  severity: ConsoleSeverity;
  payload: ReportIssueConsoleLogEntry;
  timestamp: string;
  source: string;
  body: string;
}

interface ReportNetworkEntry {
  display: string;
  payload: ReportIssueNetworkEntry;
  timestamp: string;
  method: string;
  statusLabel: string;
  durationLabel: string | null;
  errorText: string | null;
}

interface ReportAppLogStream {
  key: string;
  label: string;
  type: 'lifecycle' | 'background';
  lines: string[];
  total: number;
  command?: string;
}

type ConsoleSeverity = 'error' | 'warn' | 'info' | 'log' | 'debug' | 'trace';

interface BridgePreviewState {
  isSupported: boolean;
  caps: string[];
}

interface ReportIssueDialogProps {
  isOpen: boolean;
  onClose: () => void;
  appId?: string;
  app: App | null;
  activePreviewUrl: string | null;
  canCaptureScreenshot: boolean;
  previewContainerRef: React.RefObject<HTMLDivElement | null>;
  iframeRef: React.RefObject<HTMLIFrameElement | null>;
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

const REPORT_APP_LOGS_MAX_LINES = 200;
const REPORT_CONSOLE_LOGS_MAX_LINES = 150;
const REPORT_NETWORK_MAX_EVENTS = 150;
const MAX_CONSOLE_TEXT_LENGTH = 2000;
const MAX_NETWORK_URL_LENGTH = 2048;
const MAX_NETWORK_ERROR_LENGTH = 1500;
const MAX_NETWORK_REQUEST_ID_LENGTH = 128;
const REPORT_APP_LOGS_PANEL_ID = 'app-report-dialog-logs';
const REPORT_CONSOLE_LOGS_PANEL_ID = 'app-report-dialog-console';
const REPORT_NETWORK_PANEL_ID = 'app-report-dialog-network';

const normalizeConsoleLevel = (level: string | null | undefined): ConsoleSeverity => {
  const normalized = (level ?? '').toString().toLowerCase();
  if (['error', 'err', 'fatal', 'severe'].includes(normalized)) {
    return 'error';
  }
  if (['warn', 'warning'].includes(normalized)) {
    return 'warn';
  }
  if (['info', 'information', 'notice'].includes(normalized)) {
    return 'info';
  }
  if (['debug', 'verbose'].includes(normalized)) {
    return 'debug';
  }
  if (normalized === 'trace') {
    return 'trace';
  }
  return 'log';
};

const ReportIssueDialog = ({
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
}: ReportIssueDialogProps) => {
  const [reportMessage, setReportMessage] = useState('');
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const [reportResult, setReportResult] = useState<{ issueId?: string; issueUrl?: string; message?: string } | null>(null);
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

  const describeLogValue = useCallback((value: unknown): string => {
    if (value === null) {
      return 'null';
    }
    if (value === undefined) {
      return 'undefined';
    }
    if (typeof value === 'string') {
      return value;
    }
    if (typeof value === 'number' || typeof value === 'boolean') {
      return String(value);
    }
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }, []);

  const trimForPayload = useCallback((value: string, max: number): string => {
    if (typeof value !== 'string' || value.length === 0) {
      return value;
    }
    if (!Number.isFinite(max) || max <= 0 || value.length <= max) {
      return value;
    }
    return `${value.slice(0, max)}...`;
  }, []);

  const buildConsoleEventBody = useCallback((event: BridgeLogEvent): string => {
    const segments: string[] = [];
    if (event.message) {
      segments.push(event.message);
    }
    if (Array.isArray(event.args) && event.args.length > 0) {
      segments.push(event.args.map(describeLogValue).join(' '));
    }
    if (event.context && Object.keys(event.context).length > 0) {
      segments.push(describeLogValue(event.context));
    }
    const body = segments.filter(Boolean).join(' ');
    return body || '(no console output)';
  }, [describeLogValue]);

  const formatBridgeLogEvent = useCallback((event: BridgeLogEvent): string => {
    const timestamp = (() => {
      try {
        return new Date(event.ts).toLocaleTimeString();
      } catch {
        return String(event.ts);
      }
    })();
    const level = event.level.toUpperCase();
    const source = event.source;
    const body = buildConsoleEventBody(event);
    return `${timestamp} [${source}/${level}] ${body}`.trim();
  }, [buildConsoleEventBody]);

  const toConsoleEntry = useCallback((event: BridgeLogEvent): ReportConsoleEntry => {
    const timestamp = (() => {
      try {
        return new Date(event.ts).toLocaleTimeString();
      } catch {
        return String(event.ts);
      }
    })();
    const body = trimForPayload(buildConsoleEventBody(event), MAX_CONSOLE_TEXT_LENGTH);
    return {
      display: formatBridgeLogEvent(event),
      severity: normalizeConsoleLevel(event.level),
      payload: {
        ts: event.ts,
        level: event.level,
        source: event.source,
        text: body,
      },
      timestamp,
      source: event.source,
      body,
    };
  }, [buildConsoleEventBody, formatBridgeLogEvent, trimForPayload]);

  const formatBridgeNetworkEvent = useCallback((event: BridgeNetworkEvent): string => {
    const timestamp = (() => {
      try {
        return new Date(event.ts).toLocaleTimeString();
      } catch {
        return String(event.ts);
      }
    })();

    const method = event.method?.toUpperCase() ?? 'GET';
    const url = event.url ?? '(unknown URL)';
    const statusLabel = typeof event.status === 'number' ? ` [${event.status}]` : '';
    const durationLabel = typeof event.durationMs === 'number'
      ? ` (${Math.max(0, Math.round(event.durationMs))}ms)`
      : '';
    return `${timestamp} ${method} ${url}${statusLabel}${durationLabel}`;
  }, []);

  const toNetworkEntry = useCallback((event: BridgeNetworkEvent): ReportNetworkEntry => {
    const sanitizedURL = trimForPayload((event.url || '').trim() || '(unknown URL)', MAX_NETWORK_URL_LENGTH);
    const sanitizedError = trimForPayload(event.error ?? '', MAX_NETWORK_ERROR_LENGTH).trim();
    const sanitizedRequestId = trimForPayload(event.requestId ?? '', MAX_NETWORK_REQUEST_ID_LENGTH).trim();
    const normalizedDuration = typeof event.durationMs === 'number' && Number.isFinite(event.durationMs)
      ? Math.max(0, Math.round(event.durationMs))
      : undefined;
    const method = (event.method ?? 'GET').toUpperCase();
    const statusLabel = typeof event.status === 'number'
      ? `HTTP ${event.status}`
      : typeof event.ok === 'boolean'
        ? (event.ok ? 'OK' : 'Error')
        : '—';
    const durationLabel = typeof normalizedDuration === 'number' ? `${normalizedDuration} ms` : null;
    const timestamp = (() => {
      try {
        return new Date(event.ts).toLocaleTimeString();
      } catch {
        return String(event.ts);
      }
    })();

    return {
      display: formatBridgeNetworkEvent({
        ...event,
        method,
        url: sanitizedURL,
        error: sanitizedError,
        requestId: sanitizedRequestId,
        durationMs: normalizedDuration,
      }),
      payload: {
        ts: event.ts,
        kind: event.kind,
        method,
        url: sanitizedURL,
        status: typeof event.status === 'number' ? event.status : undefined,
        ok: typeof event.ok === 'boolean' ? event.ok : undefined,
        durationMs: normalizedDuration,
        requestId: sanitizedRequestId || undefined,
        error: sanitizedError || undefined,
      },
      timestamp,
      method,
      statusLabel,
      durationLabel,
      errorText: sanitizedError || null,
    };
  }, [formatBridgeNetworkEvent, trimForPayload]);

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
    toConsoleEntry,
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
    toNetworkEntry,
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
        payload.consoleLogs = reportConsoleLogs.map(entry => entry.payload);
        payload.consoleLogsTotal = typeof reportConsoleLogsTotal === 'number'
          ? reportConsoleLogsTotal
          : reportConsoleLogs.length;
        if (reportConsoleLogsFetchedAt) {
          payload.consoleLogsCapturedAt = new Date(reportConsoleLogsFetchedAt).toISOString();
        }
      }

      if (reportIncludeNetworkRequests && reportNetworkEvents.length > 0) {
        payload.networkRequests = reportNetworkEvents.map(entry => entry.payload);
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

  const diagnosticsViolations = diagnosticsReport?.violations ?? [];
  const diagnosticsDescription = useMemo(() => {
    if (!diagnosticsReport) {
      return '';
    }

    if (diagnosticsScannedFileCount <= 0) {
      return [
        'Scenario auditor did not scan any UI files for iframe bridge compliance.',
        'Ensure the scenario UI package depends on @vrooli/iframe-bridge and exports a bootstrap entry point.',
      ].join('\n');
    }

    if (!diagnosticsViolations.length) {
      return '';
    }

    const header = `Scenario auditor detected ${diagnosticsViolations.length} iframe bridge issue${diagnosticsViolations.length === 1 ? '' : 's'}:`;
    const items = diagnosticsViolations.map((violation, index) => {
      const recommendation = violation.recommendation ? ` Recommendation: ${violation.recommendation}` : '';
      const location = violation.file_path ? ` (${violation.file_path}:${violation.line ?? 1})` : '';
      const prefix = `${index + 1}. ${violation.title}${location}`;
      return `${prefix} — ${violation.description || violation.type}.${recommendation}`.trim();
    });
    return [header, ...items].join('\n');
  }, [diagnosticsReport, diagnosticsScannedFileCount, diagnosticsViolations]);

  const handleApplyDiagnosticsDescription = useCallback(() => {
    if (!diagnosticsDescription) {
      return;
    }
    setReportMessage(diagnosticsDescription);
    window.setTimeout(() => {
      textareaRef.current?.focus();
    }, 0);
  }, [diagnosticsDescription]);

  useEffect(() => {
    if (!isOpen) {
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

  const showDiagnosticsSetDescription = diagnosticsState === 'fail' && diagnosticsViolations.length > 0 && Boolean(diagnosticsDescription);

  if (!isOpen) {
    return null;
  }

  return (
    <div
      className="report-dialog__overlay"
      role="presentation"
      onClick={handleOverlayClick}
    >
      <div
        className="report-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="app-report-dialog-title"
        onClick={(event: MouseEvent<HTMLDivElement>) => event.stopPropagation()}
      >
        <div className="report-dialog__header">
          <h2 id="app-report-dialog-title" className="report-dialog__title">
            <Bug aria-hidden size={20} />
            <span>Report an Issue</span>
          </h2>
          <button
            type="button"
            className="report-dialog__close"
            onClick={handleDialogClose}
            disabled={reportSubmitting}
            aria-label="Close report dialog"
          >
            <X aria-hidden size={16} />
          </button>
        </div>

        {reportResult ? (
          <div className="report-dialog__state">
            <p className="report-dialog__success">
              {reportResult.message ?? 'Issue report sent successfully.'}
            </p>
            {reportResult.issueId && (
              <p className="report-dialog__success-id">
                Tracking ID: <span>{reportResult.issueId}</span>
              </p>
            )}
            {reportResult.issueUrl && (
              <div className="report-dialog__success-link">
                <button
                  type="button"
                  className="report-dialog__button"
                  onClick={() => window.open(reportResult.issueUrl, '_blank', 'noopener,noreferrer')}
                >
                  <ExternalLink aria-hidden size={16} />
                  <span>Open in Issue Tracker</span>
                </button>
              </div>
            )}
            <div className="report-dialog__actions">
              <button
                type="button"
                className="report-dialog__button report-dialog__button--primary"
                onClick={handleDialogClose}
              >
                Close
              </button>
            </div>
          </div>
        ) : (
          <form className="report-dialog__form" onSubmit={handleSubmitReport}>
            <div className="report-dialog__layout">
              <div className="report-dialog__lane report-dialog__lane--primary">
                <label htmlFor="app-report-message" className="report-dialog__label">
                  Describe the issue
                </label>
                <textarea
                  ref={textareaRef}
                  id="app-report-message"
                  className="report-dialog__textarea"
                  value={reportMessage}
                  onChange={handleReportMessageChange}
                  onKeyDown={handleReportMessageKeyDown}
                  rows={6}
                  placeholder="What are you seeing? Include steps to reproduce if possible."
                  disabled={reportSubmitting}
                  required
                />

                <div
                  className={clsx(
                    'report-dialog__bridge',
                    diagnosticsState === 'fail' && 'report-dialog__bridge--warning',
                    diagnosticsState === 'pass' && 'report-dialog__bridge--ok',
                    diagnosticsState === 'error' && 'report-dialog__bridge--warning',
                  )}
                >
                  <div className="report-dialog__bridge-head">
                    {diagnosticsState === 'loading' && <Loader2 aria-hidden size={18} className="spinning" />}
                    {diagnosticsState === 'pass' && <CheckCircle2 aria-hidden size={18} />}
                    {diagnosticsState === 'fail' && <AlertTriangle aria-hidden size={18} />}
                    {diagnosticsState === 'error' && <AlertTriangle aria-hidden size={18} />}
                    <span>
                      {diagnosticsState === 'loading' && 'Checking iframe bridge setup…'}
                      {diagnosticsState === 'pass' && 'Iframe bridge diagnostics passed'}
                      {diagnosticsState === 'fail' && 'Iframe bridge diagnostics reported issues'}
                      {diagnosticsState === 'error' && 'Unable to run iframe bridge diagnostics'}
                      {diagnosticsState === 'idle' && 'Iframe bridge diagnostics not run'}
                    </span>
                    <button
                      type="button"
                      className="report-dialog__bridge-refresh"
                      onClick={() => refreshDiagnostics()}
                      disabled={diagnosticsLoading}
                      aria-label="Refresh iframe bridge diagnostics"
                    >
                      {diagnosticsLoading ? (
                        <Loader2 aria-hidden size={14} className="spinning" />
                      ) : (
                        <RefreshCw aria-hidden size={14} />
                      )}
                    </button>
                  </div>
                  {diagnosticsCheckedAt && !diagnosticsLoading && (
                    <p className="report-dialog__bridge-meta">Checked at {diagnosticsCheckedAt}</p>
                  )}

                  {diagnosticsState === 'pass' && diagnosticsWarning && (
                    <p className="report-dialog__bridge-note">{diagnosticsWarning}</p>
                  )}

                  {diagnosticsState === 'pass' && diagnosticsScannedFileCount > 0 && (
                    <p className="report-dialog__bridge-note">
                      Scanned {diagnosticsScannedFileCount} file{diagnosticsScannedFileCount === 1 ? '' : 's'} and found no violations.
                    </p>
                  )}

                  {diagnosticsState === 'fail' && diagnosticsScannedFileCount <= 0 && (
                    <p className="report-dialog__bridge-note">
                      Scenario auditor did not scan any matching UI files. Confirm your scenario UI imports the shared iframe bridge and is included in the rule targets.
                    </p>
                  )}

                  {diagnosticsState === 'fail' && diagnosticsViolations.length > 0 && (
                    <ul className="report-dialog__bridge-list">
                      {diagnosticsViolations.map((violation) => (
                        <li key={`${violation.file_path}:${violation.line}:${violation.title}`}>
                          <strong>{violation.title}</strong>
                          {violation.file_path && (
                            <span>
                              {' '}
                              ({violation.file_path}
                              {typeof violation.line === 'number' ? `:${violation.line}` : ''})
                            </span>
                          )}
                          {violation.description && <div>{violation.description}</div>}
                          {violation.recommendation && (
                            <div className="report-dialog__bridge-recommendation">{violation.recommendation}</div>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}

                  {diagnosticsState === 'error' && diagnosticsError && (
                    <div className="report-dialog__bridge-note">
                      <p>{diagnosticsError}</p>
                      <button
                        type="button"
                        className="report-dialog__button report-dialog__button--ghost"
                        onClick={() => refreshDiagnostics()}
                        disabled={diagnosticsLoading}
                      >
                        Retry
                      </button>
                    </div>
                  )}

                  {showDiagnosticsSetDescription && (
                    <div className="report-dialog__bridge-actions">
                      <button
                        type="button"
                        className="report-dialog__button report-dialog__button--ghost"
                        onClick={handleApplyDiagnosticsDescription}
                      >
                        Use details in description
                      </button>
                    </div>
                  )}

                  {bridgeCompliance && bridgeCompliance.ok && (
                    <div className="report-dialog__bridge-inline">
                      <span>Runtime bridge check passed</span>
                    </div>
                  )}
                </div>

                <div className="report-dialog__logs">
                  <section className="report-dialog__logs-section">
                    <div className="report-dialog__logs-header">
                      <label
                        className={clsx(
                          'report-dialog__logs-include',
                          !reportIncludeAppLogs && 'report-dialog__logs-include--off',
                        )}
                      >
                        <input
                          type="checkbox"
                          checked={reportIncludeAppLogs}
                          onChange={(event) => setReportIncludeAppLogs(event.target.checked)}
                          aria-label="Include app logs in report"
                        />
                        <span className="report-dialog__logs-title">App logs</span>
                      </label>
                      <button
                        type="button"
                        className="report-dialog__logs-toggle"
                        onClick={() => setReportAppLogsExpanded(prev => !prev)}
                        aria-expanded={reportAppLogsExpanded}
                        aria-controls={REPORT_APP_LOGS_PANEL_ID}
                        aria-label={reportAppLogsExpanded ? 'Hide app logs' : 'Show app logs'}
                      >
                        {reportAppLogsExpanded ? (
                          <EyeOff aria-hidden size={18} />
                        ) : (
                          <Eye aria-hidden size={18} />
                        )}
                      </button>
                    </div>
                    {reportAppLogStreams.length > 0 && (
                      <div className="report-dialog__logs-streams">
                        {reportAppLogStreams.map(stream => (
                          <label key={stream.key} className="report-dialog__logs-stream">
                            <input
                              type="checkbox"
                              checked={reportAppLogSelections[stream.key] !== false}
                              onChange={(event) => handleReportLogStreamToggle(stream.key, event.target.checked)}
                              disabled={!reportIncludeAppLogs}
                            />
                            <span>{stream.label}</span>
                            <span className="report-dialog__logs-count">{stream.total}</span>
                          </label>
                        ))}
                        <div className="report-dialog__logs-streams-meta">
                          {reportIncludeAppLogs ? (
                            <span>{`${selectedReportLogCount} of ${totalReportLogStreamCount} log streams selected`}</span>
                          ) : (
                            <span>Enable app logs to include these streams.</span>
                          )}
                          {reportIncludeAppLogs && selectedReportLogCount === 0 && (
                            <span className="report-dialog__logs-streams-warning">No log streams selected</span>
                          )}
                        </div>
                      </div>
                    )}
                    <div
                      id={REPORT_APP_LOGS_PANEL_ID}
                      className="report-dialog__logs-panel"
                      style={reportAppLogsExpanded ? undefined : { display: 'none' }}
                      aria-hidden={!reportAppLogsExpanded}
                    >
                      <div className="report-dialog__logs-meta">
                        <span>
                          {reportAppLogsLoading
                            ? 'Loading logs…'
                            : reportAppLogs.length > 0
                              ? `Showing last ${reportAppLogs.length}${logsTruncated ? ` of ${reportAppLogsTotal}` : ''} lines${formattedReportLogsTime ? ` (captured ${formattedReportLogsTime})` : ''}.`
                              : reportAppLogsError
                                ? 'Logs unavailable.'
                                : 'No logs captured yet.'}
                        </span>
                        <button
                          type="button"
                          className="report-dialog__logs-refresh"
                          onClick={handleRefreshReportAppLogs}
                          disabled={reportAppLogsLoading}
                        >
                          {reportAppLogsLoading ? (
                            <Loader2 aria-hidden size={14} className="spinning" />
                          ) : (
                            <RefreshCw aria-hidden size={14} />
                          )}
                          <span>{reportAppLogsLoading ? 'Loading' : 'Refresh'}</span>
                        </button>
                      </div>
                      {reportAppLogsLoading ? (
                        <div className="report-dialog__logs-loading">
                          <Loader2 aria-hidden size={18} className="spinning" />
                          <span>Fetching logs…</span>
                        </div>
                      ) : reportAppLogsError ? (
                        <div className="report-dialog__logs-message">
                          <p>{reportAppLogsError}</p>
                          <button
                            type="button"
                            className="report-dialog__button report-dialog__button--ghost"
                            onClick={handleRefreshReportAppLogs}
                            disabled={reportAppLogsLoading}
                          >
                            Retry
                          </button>
                        </div>
                      ) : reportAppLogs.length === 0 ? (
                        <p className="report-dialog__logs-empty">No logs available for this scenario.</p>
                      ) : (
                        <pre className="report-dialog__logs-content">{reportAppLogs.join('\n')}</pre>
                      )}
                    </div>
                  </section>

                  <section className="report-dialog__logs-section">
                    <div className="report-dialog__logs-header">
                      <label
                        className={clsx(
                          'report-dialog__logs-include',
                          !reportIncludeConsoleLogs && 'report-dialog__logs-include--off',
                          !bridgeState.caps.includes('logs') && 'report-dialog__logs-include--disabled',
                        )}
                      >
                        <input
                          type="checkbox"
                          checked={reportIncludeConsoleLogs}
                          onChange={(event) => setReportIncludeConsoleLogs(event.target.checked)}
                          aria-label="Include console logs in report"
                          disabled={!bridgeState.caps.includes('logs')}
                        />
                        <span className="report-dialog__logs-title">Console logs</span>
                      </label>
                      <button
                        type="button"
                        className="report-dialog__logs-toggle"
                        onClick={() => setReportConsoleLogsExpanded(prev => !prev)}
                        aria-expanded={reportConsoleLogsExpanded}
                        aria-controls={REPORT_CONSOLE_LOGS_PANEL_ID}
                        disabled={!bridgeState.caps.includes('logs')}
                        aria-label={reportConsoleLogsExpanded ? 'Hide console logs' : 'Show console logs'}
                      >
                        {reportConsoleLogsExpanded ? (
                          <EyeOff aria-hidden size={18} />
                        ) : (
                          <Eye aria-hidden size={18} />
                        )}
                      </button>
                    </div>
                    <div
                      id={REPORT_CONSOLE_LOGS_PANEL_ID}
                      className="report-dialog__logs-panel"
                      style={reportConsoleLogsExpanded ? undefined : { display: 'none' }}
                      aria-hidden={!reportConsoleLogsExpanded}
                    >
                      <div className="report-dialog__logs-meta">
                        <span>
                          {reportConsoleLogsLoading
                            ? 'Loading console output…'
                            : reportConsoleLogs.length > 0
                              ? `Showing last ${reportConsoleLogs.length}${consoleLogsTruncated ? ` of ${reportConsoleLogsTotal}` : ''} events${formattedConsoleLogsTime ? ` (captured ${formattedConsoleLogsTime})` : ''}.`
                              : reportConsoleLogsError
                                ? 'Console logs unavailable.'
                                : 'No console output captured yet.'}
                        </span>
                        <button
                          type="button"
                          className="report-dialog__logs-refresh"
                          onClick={handleRefreshReportConsoleLogs}
                          disabled={reportConsoleLogsLoading || !bridgeState.caps.includes('logs')}
                        >
                          {reportConsoleLogsLoading ? (
                            <Loader2 aria-hidden size={14} className="spinning" />
                          ) : (
                            <RefreshCw aria-hidden size={14} />
                          )}
                          <span>{reportConsoleLogsLoading ? 'Loading' : 'Refresh'}</span>
                        </button>
                      </div>
                      {reportConsoleLogsLoading ? (
                        <div className="report-dialog__logs-loading">
                          <Loader2 aria-hidden size={18} className="spinning" />
                          <span>Fetching console output…</span>
                        </div>
                      ) : reportConsoleLogsError ? (
                        <div className="report-dialog__logs-message">
                          <p>{reportConsoleLogsError}</p>
                          <button
                            type="button"
                            className="report-dialog__button report-dialog__button--ghost"
                            onClick={handleRefreshReportConsoleLogs}
                            disabled={reportConsoleLogsLoading}
                          >
                            Retry
                          </button>
                        </div>
                      ) : reportConsoleLogs.length === 0 ? (
                        <p className="report-dialog__logs-empty">No console output captured yet.</p>
                      ) : (
                        <div className="report-dialog__logs-content report-dialog__logs-content--console">
                          {reportConsoleLogs.map((entry, index) => (
                            <div
                              key={`${entry.payload.ts}-${index}`}
                              className={clsx(
                                'report-dialog__console-line',
                                `report-dialog__console-line--${entry.severity}`,
                              )}
                            >
                              <div className="report-dialog__console-meta">
                                <span className="report-dialog__console-timestamp">{entry.timestamp}</span>
                                <span className="report-dialog__console-source">{entry.source}</span>
                                <span className="report-dialog__console-level">{entry.payload.level.toUpperCase()}</span>
                              </div>
                              <div className="report-dialog__console-body">{entry.body}</div>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </section>

                  <section className="report-dialog__logs-section">
                    <div className="report-dialog__logs-header">
                      <label
                        className={clsx(
                          'report-dialog__logs-include',
                          !reportIncludeNetworkRequests && 'report-dialog__logs-include--off',
                          !bridgeState.caps.includes('network') && 'report-dialog__logs-include--disabled',
                        )}
                      >
                        <input
                          type="checkbox"
                          checked={reportIncludeNetworkRequests}
                          onChange={(event) => setReportIncludeNetworkRequests(event.target.checked)}
                          aria-label="Include network activity in report"
                          disabled={!bridgeState.caps.includes('network')}
                        />
                        <span className="report-dialog__logs-title">Network requests</span>
                      </label>
                      <button
                        type="button"
                        className="report-dialog__logs-toggle"
                        onClick={() => setReportNetworkExpanded(prev => !prev)}
                        aria-expanded={reportNetworkExpanded}
                        aria-controls={REPORT_NETWORK_PANEL_ID}
                        disabled={!bridgeState.caps.includes('network')}
                        aria-label={reportNetworkExpanded ? 'Hide network requests' : 'Show network requests'}
                      >
                        {reportNetworkExpanded ? (
                          <EyeOff aria-hidden size={18} />
                        ) : (
                          <Eye aria-hidden size={18} />
                        )}
                      </button>
                    </div>
                    <div
                      id={REPORT_NETWORK_PANEL_ID}
                      className="report-dialog__logs-panel"
                      style={reportNetworkExpanded ? undefined : { display: 'none' }}
                      aria-hidden={!reportNetworkExpanded}
                    >
                      <div className="report-dialog__logs-meta">
                        <span>
                          {reportNetworkLoading
                            ? 'Loading requests…'
                            : reportNetworkEvents.length > 0
                              ? `Showing last ${reportNetworkEvents.length}${networkEventsTruncated ? ` of ${reportNetworkTotal}` : ''} requests${formattedNetworkCapturedAt ? ` (captured ${formattedNetworkCapturedAt})` : ''}.`
                              : reportNetworkError
                                ? 'Network requests unavailable.'
                                : 'No network requests captured yet.'}
                        </span>
                        <button
                          type="button"
                          className="report-dialog__logs-refresh"
                          onClick={handleRefreshReportNetworkEvents}
                          disabled={reportNetworkLoading || !bridgeState.caps.includes('network')}
                        >
                          {reportNetworkLoading ? (
                            <Loader2 aria-hidden size={14} className="spinning" />
                          ) : (
                            <RefreshCw aria-hidden size={14} />
                          )}
                          <span>{reportNetworkLoading ? 'Loading' : 'Refresh'}</span>
                        </button>
                      </div>
                      {reportNetworkLoading ? (
                        <div className="report-dialog__logs-loading">
                          <Loader2 aria-hidden size={18} className="spinning" />
                          <span>Fetching requests…</span>
                        </div>
                      ) : reportNetworkError ? (
                        <div className="report-dialog__logs-message">
                          <p>{reportNetworkError}</p>
                          <button
                            type="button"
                            className="report-dialog__button report-dialog__button--ghost"
                            onClick={handleRefreshReportNetworkEvents}
                            disabled={reportNetworkLoading}
                          >
                            Retry
                          </button>
                        </div>
                      ) : reportNetworkEvents.length === 0 ? (
                        <p className="report-dialog__logs-empty">No network requests captured.</p>
                      ) : (
                        <div className="report-dialog__logs-content report-dialog__logs-content--network">
                          {reportNetworkEvents.map((entry, index) => {
                            const statusClass = entry.payload.status && entry.payload.status >= 400
                              ? 'report-dialog__network-status--error'
                              : entry.payload.ok === false
                                ? 'report-dialog__network-status--error'
                                : 'report-dialog__network-status--ok';
                            return (
                              <div className="report-dialog__network-entry" key={`${entry.payload.ts}-${index}`}>
                                <div className="report-dialog__network-row">
                                  <span className="report-dialog__network-timestamp">{entry.timestamp}</span>
                                  <span className="report-dialog__network-method">{entry.method}</span>
                                  <span className={clsx('report-dialog__network-status', statusClass)}>{entry.statusLabel}</span>
                                  {entry.durationLabel && (
                                    <span className="report-dialog__network-duration">{entry.durationLabel}</span>
                                  )}
                                  {entry.payload.requestId && (
                                    <span className="report-dialog__network-request">{entry.payload.requestId}</span>
                                  )}
                                </div>
                                <div className="report-dialog__network-url" title={entry.payload.url}>{entry.payload.url}</div>
                                {entry.errorText && (
                                  <div className="report-dialog__network-error">{entry.errorText}</div>
                                )}
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  </section>
                </div>
              </div>

              <div className="report-dialog__lane report-dialog__lane--secondary">
                <label className="report-dialog__checkbox">
                  <input
                    type="checkbox"
                    checked={reportIncludeScreenshot && canCaptureScreenshot}
                    onChange={handleReportIncludeScreenshotChange}
                    disabled={!canCaptureScreenshot || reportSubmitting}
                  />
                  <span>Include screenshot of the current preview</span>
                </label>
                {!canCaptureScreenshot && (
                  <p className="report-dialog__hint">Load the preview to capture a screenshot.</p>
                )}
                {reportIncludeScreenshot && canCaptureScreenshot && (
                  <div className="report-dialog__preview" aria-live="polite">
                    {reportScreenshotLoading && (
                      <div className="report-dialog__preview-loading">
                        <Loader2 aria-hidden size={18} className="spinning" />
                        <span>Capturing screenshot…</span>
                      </div>
                    )}

                    {!reportScreenshotLoading && reportScreenshotError && (
                      <div className="report-dialog__preview-error">
                        <p>{reportScreenshotError}</p>
                        <button
                          type="button"
                          className="report-dialog__button report-dialog__button--ghost"
                          onClick={handleRetryScreenshotCapture}
                          disabled={reportScreenshotLoading}
                        >
                          Retry capture
                        </button>
                      </div>
                    )}

                    {!reportScreenshotLoading && !reportScreenshotError && reportScreenshotInfo && (
                      <p className="report-dialog__preview-info">{reportScreenshotInfo}</p>
                    )}

                    {!reportScreenshotLoading && !reportScreenshotError && reportScreenshotData && (
                      <>
                        <div
                          className={clsx(
                            'report-dialog__preview-image',
                            'report-dialog__preview-image--selectable',
                          )}
                          ref={assignScreenshotContainerRef}
                          {...reportScreenshotContainerHandlers}
                        >
                          <img
                            src={`data:image/png;base64,${reportScreenshotData}`}
                            alt="Preview screenshot"
                            loading="lazy"
                            draggable={false}
                            onLoad={handleScreenshotImageLoad}
                          />
                          {reportSelectionRect && (
                            <div
                              className="report-dialog__selection"
                              style={{
                                left: `${reportSelectionRect.x}px`,
                                top: `${reportSelectionRect.y}px`,
                                width: `${reportSelectionRect.width}px`,
                                height: `${reportSelectionRect.height}px`,
                              }}
                            >
                              {selectionDimensionLabel && (
                                <span className="report-dialog__selection-label">{selectionDimensionLabel}</span>
                              )}
                            </div>
                          )}
                        </div>
                        <div className="report-dialog__preview-actions">
                          <div className="report-dialog__preview-buttons">
                            <button
                              type="button"
                              className="report-dialog__button report-dialog__button--ghost"
                              onClick={handleDelayedScreenshotCapture}
                              disabled={reportScreenshotLoading}
                            >
                              {reportScreenshotCountdown !== null
                                ? `Cancel timer (${reportScreenshotCountdown}s)`
                                : 'Capture in 5s'}
                            </button>
                            <button
                              type="button"
                              className="report-dialog__button report-dialog__button--ghost"
                              onClick={handleRetryScreenshotCapture}
                              disabled={reportScreenshotLoading}
                            >
                              {reportScreenshotLoading ? 'Capturing…' : 'Capture now'}
                            </button>
                            <button
                              type="button"
                              className="report-dialog__button report-dialog__button--ghost"
                              onClick={handleResetScreenshotSelection}
                              disabled={reportScreenshotLoading || !reportScreenshotClip}
                            >
                              Reset area
                            </button>
                          </div>
                          <p className="report-dialog__preview-hint">
                            {reportScreenshotClip === null
                              ? 'Drag on the screenshot to focus on the area that needs attention. Use the timer to stage transient UI before the snap.'
                              : 'Crop saved. Drag again to fine-tune or reset to capture a different view.'}
                          </p>
                        </div>
                        {reportScreenshotOriginalDimensions && (
                          <p className="report-dialog__preview-meta">
                            {`Original size: ${reportScreenshotOriginalDimensions.width}×${reportScreenshotOriginalDimensions.height}`}
                          </p>
                        )}
                      </>
                    )}

                    {!reportScreenshotLoading && !reportScreenshotError && !reportScreenshotData && (
                      <div className="report-dialog__preview-actions">
                        <div className="report-dialog__preview-buttons">
                          <button
                            type="button"
                            className="report-dialog__button report-dialog__button--ghost"
                            onClick={handleDelayedScreenshotCapture}
                            disabled={reportScreenshotLoading}
                          >
                            {reportScreenshotCountdown !== null
                              ? `Cancel timer (${reportScreenshotCountdown}s)`
                              : 'Capture in 5s'}
                          </button>
                          <button
                            type="button"
                            className="report-dialog__button report-dialog__button--ghost"
                            onClick={handleRetryScreenshotCapture}
                            disabled={reportScreenshotLoading}
                          >
                            {reportScreenshotLoading ? 'Capturing…' : 'Capture now'}
                          </button>
                        </div>
                        <p className="report-dialog__preview-hint">
                          Ready to capture the current preview. Use the delay to open menus or tooltips before the screenshot fires.
                        </p>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>

            {reportError && (
              <p className="report-dialog__error" role="alert">
                {reportError}
              </p>
            )}

            <div className="report-dialog__actions">
              <button
                type="button"
                className="report-dialog__button"
                onClick={handleDialogClose}
                disabled={reportSubmitting}
              >
                Cancel
              </button>
              <button
                type="submit"
                className="report-dialog__button report-dialog__button--primary"
                disabled={
                  reportSubmitting || (
                    reportIncludeScreenshot && canCaptureScreenshot && (reportScreenshotLoading || !reportScreenshotData)
                  )
                }
              >
                {reportSubmitting ? (
                  <>
                    <Loader2 aria-hidden size={16} className="spinning" />
                    Sending…
                  </>
                ) : (
                  'Send Report'
                )}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
};

export default ReportIssueDialog;
